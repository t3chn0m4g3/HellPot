//go:build windows
// +build windows

package http

import (
	"errors"

	"github.com/valyala/fasthttp"
)

func listenOnUnixSocket(addr string, srv *fasthttp.Server) error {
	return errors.New("unix sockets are not supported on Windows")
}
