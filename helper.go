package gopi

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/teejays/goku-util/errutil"
	"github.com/teejays/goku-util/httputil"
	"github.com/teejays/goku-util/log"
	"github.com/teejays/goku-util/panics"

	"github.com/teejays/gopi/json"
	"github.com/teejays/gopi/validator"
)

// GetQueryParamInt extracts the param value with given name out of the URL query
func GetQueryParamInt(r *http.Request, name string, defaultVal int) (int, error) {
	err := r.ParseForm()
	if err != nil {
		return defaultVal, err
	}
	values, exist := r.Form[name]
	log.DebugNoCtx("URL values", "param", name, "value", values)
	if !exist {
		return defaultVal, nil
	}
	if len(values) > 1 {
		return defaultVal, fmt.Errorf("multiple URL form values found for %s", name)
	}

	val, err := strconv.Atoi(values[0])
	if err != nil {
		return defaultVal, fmt.Errorf("error parsing %s value to an int: %v", name, err)
	}
	return val, nil
}

// GetMuxParamInt extracts the param with given name out of the route path
func GetMuxParamInt(r *http.Request, name string) (int64, error) {

	var vars = mux.Vars(r)

	log.DebugNoCtx("MUX vars", "value", vars)
	valStr := vars[name]
	if strings.TrimSpace(valStr) == "" {
		return -1, fmt.Errorf("could not find var %s in the route", name)
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return -1, fmt.Errorf("could not convert var %s to an int64: %v", name, err)
	}

	return int64(val), nil
}

// GetMuxParamStr extracts the param with given name out of the route path
func GetMuxParamStr(r *http.Request, name string) (string, error) {

	var vars = mux.Vars(r)
	log.DebugNoCtx("MUX vars", "value", vars)
	valStr := vars[name]
	if strings.TrimSpace(valStr) == "" {
		return "", fmt.Errorf("var '%s' is not in the route", name)
	}

	return valStr, nil
}

type StandardResponse struct {
	StatusCode int
	Data       interface{}
	Error      interface{}
}

func WriteStandardResponse(w http.ResponseWriter, v interface{}) {
	var resp = StandardResponse{
		StatusCode: http.StatusOK,
		Data:       v,
		Error:      nil,
	}
	writeResponse(w, http.StatusOK, resp)
}

// WriteResponse is a helper function to help write HTTP response
func WriteResponse(w http.ResponseWriter, code int, v interface{}) {
	writeResponse(w, code, v)
}

func writeResponse(w http.ResponseWriter, code int, v interface{}) {
	w.WriteHeader(code)
	log.DebugNoCtx("api: writeResponse", "kind", reflect.ValueOf(v).Kind(), "content", v)

	if v == nil {
		return
	}

	// Json marshal the resp
	data, err := json.Marshal(v)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Write the response
	_, err = w.Write(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
}

// WriteError is a helper function to help write HTTP response
func WriteError(w http.ResponseWriter, code int, err error) {
	writeError(w, code, err)
}

func writeError(w http.ResponseWriter, code int, err error) {

	var errMessage string

	// For Internal errors passed, use a generic message
	if code == http.StatusInternalServerError {
		errMessage = ErrMessageGeneric
	}

	log.ErrorNoCtx("Writing error to http response", "error", err)

	// If it a goku error?
	if gErr, ok := errutil.AsGokuError(err); ok {
		errMessage = gErr.GetExternalMsg()
		if code < 1 {
			code = gErr.GetHTTPStatus()
		}
	}

	if errMessage == "" {
		errMessage = err.Error()

	}

	// Still no code? Use InternalServerError
	if code < 1 {
		code = http.StatusInternalServerError
	}

	resp := StandardResponse{
		StatusCode: code,
		Data:       nil,
		Error:      errMessage,
	}

	w.WriteHeader(code)
	data, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Sprintf("Failed to json.Unmarshal an error for http response: %v", err))
	}
	_, err = w.Write(data)
	if err != nil {
		panic(fmt.Sprintf("Failed to write error to the http response: %v", err))
	}
}

// UnmarshalJSONFromRequest takes in a pointer to an object and populates
// it by reading the content body of the HTTP request, and unmarshaling the
// body into the variable v.
func UnmarshalJSONFromRequest(r *http.Request, v interface{}) error {
	// Read the HTTP request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// api.WriteError(w, http.StatusBadRequest, err, false, nil)
		return err
	}
	defer r.Body.Close()

	if len(body) < 1 {
		// api.WriteError(w, http.StatusBadRequest, api.ErrEmptyBody, false, nil)
		return ErrEmptyBody
	}

	log.DebugNoCtx("api: Unmarshaling to JSON", "body", string(body))

	// Unmarshal JSON into Go type
	err = json.Unmarshal(body, &v)
	if err != nil {
		log.ErrorNoCtx("api: Unmarshaling to JSON", "error", err)
		return ErrInvalidJSON
	}

	err = validator.Validate(v)
	if err != nil {
		return err
	}

	return nil
}

func HandlerWrapper[ReqT any, RespT any](httpMethod string, fn func(context.Context, ReqT) (RespT, error)) http.HandlerFunc {

	switch httpMethod {
	case http.MethodGet:
		return GetGenericGetHandler(fn)
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return GetGenericPostPutPatchHandler(fn)
	default:
		// Todo: Implement other method types like DELETE?
	}

	panics.P("HTTP Method type [%s] not implemented by routes.HandlerWrapper().", httpMethod)
	return nil
}

func GetGenericGetHandler[ReqT, RespT any](fn func(context.Context, ReqT) (RespT, error)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		log.Debug(ctx, "[HTTP Handler] Starting...")

		// Get the req data from URL
		reqParam, ok := r.URL.Query()["req"]
		if !ok || len(reqParam) < 1 {
			WriteError(w, http.StatusBadRequest, fmt.Errorf("URL param 'req' is required"))
			return
		}
		if len(reqParam) > 1 {
			WriteError(w, http.StatusBadRequest, fmt.Errorf("multiple URL params with name 'req' found"))
			return
		}

		var req ReqT
		err := json.Unmarshal([]byte(reqParam[0]), &req)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err)
			return
		}

		// Call the method
		resp, err := fn(ctx, req)
		if err != nil {
			WriteError(w, 0, err)
			return
		}

		WriteStandardResponse(w, resp)
		return
	}
}

func GetGenericPostPutPatchHandler[ReqT, RespT any](fn func(context.Context, ReqT) (RespT, error)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		log.Debug(ctx, "[HTTP Handler] Starting...")

		// Get the req from HTTP body
		var req ReqT
		err := httputil.UnmarshalJSONFromRequest(r, &req)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err)
			return
		}

		// Call the method
		resp, err := fn(r.Context(), req)
		if err != nil {
			WriteError(w, 0, err)
			return
		}

		WriteStandardResponse(w, resp)

	}
}
