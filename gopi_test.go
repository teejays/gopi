package gopi_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teejays/gopi"
)

type SampleReq struct {
	Ping                string
	RequestErrorWithMsg string
}
type SampleResp struct {
	Pong string
}

func SampleEndpoint(ctx context.Context, req SampleReq) (SampleResp, error) {
	if req.RequestErrorWithMsg != "" {
		return SampleResp{}, fmt.Errorf("%s", req.RequestErrorWithMsg)
	}
	return SampleResp{Pong: req.Ping}, nil
}

func TestNewServer(t *testing.T) {

	tests := []struct {
		name       string
		routes     []gopi.Route
		middleware gopi.MiddlewareFuncs
		wantErr    error
	}{
		{
			name:    "No routes",
			routes:  []gopi.Route{},
			wantErr: fmt.Errorf("no routes provided"),
		},
		{
			name: "Single route, good, GET ",
			routes: []gopi.Route{
				{
					Method:      http.MethodGet,
					Path:        "foo",
					HandlerFunc: gopi.HandlerWrapper(http.MethodGet, SampleEndpoint),
				},
			},
			wantErr: nil,
		},
		{
			name: "Single route, good, POST ",
			routes: []gopi.Route{
				{
					Method:      http.MethodPost,
					Path:        "foo",
					HandlerFunc: gopi.HandlerWrapper(http.MethodGet, SampleEndpoint),
				},
			},
			wantErr: nil,
		},
		{
			name: "Single route, bad, no http method",
			routes: []gopi.Route{
				{
					Method:      "",
					Path:        "foo",
					HandlerFunc: gopi.HandlerWrapper(http.MethodGet, SampleEndpoint),
				},
			},
			wantErr: fmt.Errorf("no http method"),
		},
		{
			name: "Single route, bad, no path",
			routes: []gopi.Route{
				{
					Method:      http.MethodGet,
					Path:        "",
					HandlerFunc: gopi.HandlerWrapper(http.MethodGet, SampleEndpoint),
				},
			},
			wantErr: fmt.Errorf("no path"),
		},
		{
			name: "Single route, bad, no handler",
			routes: []gopi.Route{
				{
					Method:      http.MethodGet,
					Path:        "foo",
					HandlerFunc: nil,
				},
			},
			wantErr: fmt.Errorf("no HandlerFunc"),
		},
		{
			name: "Multiple routes, good",
			routes: []gopi.Route{
				{
					Method:      http.MethodGet,
					Path:        "foo",
					HandlerFunc: gopi.HandlerWrapper(http.MethodGet, SampleEndpoint),
				},
				{
					Method:      http.MethodPost,
					Path:        "foo",
					HandlerFunc: gopi.HandlerWrapper(http.MethodGet, SampleEndpoint),
				},
			},
			wantErr: nil,
		},
		{
			name: "Multiple routes, bad, conflicting routes",
			routes: []gopi.Route{
				{
					Method:      http.MethodGet,
					Path:        "foo",
					HandlerFunc: gopi.HandlerWrapper(http.MethodGet, SampleEndpoint),
				},
				{
					Method:      http.MethodPost,
					Path:        "foo",
					HandlerFunc: gopi.HandlerWrapper(http.MethodGet, SampleEndpoint),
				},
				{
					Method:      http.MethodPost,
					Path:        "foo",
					HandlerFunc: gopi.HandlerWrapper(http.MethodGet, SampleEndpoint),
				},
			},
			wantErr: fmt.Errorf("multiple routes"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			_, err := gopi.NewServer(ctx, tt.routes, tt.middleware)
			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			}
		})
	}
}
