package main

import (
	"fmt"
	"net/http"

	"github.com/teejays/gopi"
	"golang.org/x/net/context"
)

func main() {
	ctx := context.Background()
	if err := mainCtx(ctx); err != nil {
		panic(fmt.Sprintf("%s", err))
	}
}

func mainCtx(ctx context.Context) error {

	routes := []gopi.Route{
		{
			Method:       http.MethodGet,
			Path:         "/ping",
			Version:      1,
			HandlerFunc:  gopi.HandlerWrapper(http.MethodGet, PingHandler),
			Authenticate: false,
		},
	}

	mws := gopi.MiddlewareFuncs{}
	s, err := gopi.NewServer(ctx, routes, mws)
	if err != nil {
		return err
	}

	err = s.StartServer(ctx, "127.0.0.1", 8080)
	if err != nil {
		return err
	}

	return nil
}

type PingReq struct {
	Msg           string
	RequestErrMsg string
}

type PingResp struct {
	Msg string
}

func PingHandler(ctx context.Context, req PingReq) (PingResp, error) {
	if req.RequestErrMsg != "" {
		return PingResp{}, fmt.Errorf("%s", req.RequestErrMsg)
	}
	return PingResp{Msg: fmt.Sprintf("You said: %s", req.Msg)}, nil
}
