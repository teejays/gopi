package gopi

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/teejays/clog"
	"github.com/teejays/goku-util/errutil"

	"github.com/teejays/gopi/json"
	"github.com/teejays/gopi/validator"
)

// GetQueryParamInt extracts the param value with given name  out of the URL query
func GetQueryParamInt(r *http.Request, name string, defaultVal int) (int, error) {
	err := r.ParseForm()
	if err != nil {
		return defaultVal, err
	}
	values, exist := r.Form[name]
	clog.Debugf("URL values for %s: %+v", name, values)
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
	clog.Debugf("MUX vars are: %+v", vars)
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
	clog.Debugf("MUX vars are: %+v", vars)
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
	clog.Debugf("api: writeResponse: content kind: %v; content:\n%+v", reflect.ValueOf(v).Kind(), v)

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

	clog.Error(err.Error())

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

	clog.Debugf("api: Unmarshaling to JSON: body:\n%+v", string(body))

	// Unmarshal JSON into Go type
	err = json.Unmarshal(body, &v)
	if err != nil {
		// api.WriteError(w, http.StatusBadRequest, err, true, api.ErrInvalidJSON)
		clog.Errorf("api: Error unmarshaling JSON: %v", err)
		return ErrInvalidJSON
	}

	err = validator.Validate(v)
	if err != nil {
		return err
	}

	return nil
}
