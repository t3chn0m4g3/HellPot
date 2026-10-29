//go:build linux || darwin || freebsd

package http

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/valyala/fasthttp"

	"github.com/telekom-security/hellpot/internal/config"
)

func listenOnUnixSocket(addr string, srv *fasthttp.Server) error {
	var err error
	var unixAddr *net.UnixAddr
	var unixListener *net.UnixListener
	unixAddr, err = net.ResolveUnixAddr("unix", addr)
	if err != nil {
		return err
	}
	// Before we set socket permissions, we want to make sure only the user HellPot is running under
	// has permission to the socket.
	oldmask := syscall.Umask(0o077)
	unixListener, err = net.ListenUnix("unix", unixAddr)
	syscall.Umask(oldmask)
	if err != nil {
		return err
	}
	if err = os.Chmod(
		unixAddr.Name,
		os.FileMode(config.UnixSocketPermissions),
	); err != nil {
		if closeErr := unixListener.Close(); closeErr != nil {
			return fmt.Errorf("set Unix socket permissions: %w (close listener: %v)", err, closeErr)
		}
		return fmt.Errorf("set Unix socket permissions: %w", err)
	}

	return srv.Serve(unixListener)
}
