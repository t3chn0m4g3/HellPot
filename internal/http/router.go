package http

import (
	"bufio"
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/fasthttp/router"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"

	"github.com/t3chn0m4g3/hellpot/internal/config"
	"github.com/t3chn0m4g3/hellpot/internal/heffalump"
)

var log *zerolog.Logger

func getRealRemote(ctx *fasthttp.RequestCtx) string {
	xrealip := string(ctx.Request.Header.Peek(config.HeaderName))
	if len(xrealip) > 0 {
		return xrealip
	}
	return ctx.RemoteIP().String()
}

func hellPot(ctx *fasthttp.RequestCtx) {
	path, pok := ctx.UserValue("path").(string)
	if len(path) < 1 || !pok {
		path = "/"
	}

	remoteAddr := getRealRemote(ctx)

	slog := log.With().
		Str("USERAGENT", string(ctx.UserAgent())).
		Str("REMOTE_ADDR", remoteAddr).
		Interface("URL", string(ctx.RequestURI())).Logger()

	for _, denied := range config.UseragentBlacklistMatchers {
		if strings.Contains(string(ctx.UserAgent()), denied) {
			slog.Trace().Msg("Ignoring useragent")
			ctx.Error("Not found", fasthttp.StatusNotFound)
			return
		}
	}

	if config.Trace {
		slog = slog.With().Str("caller", path).Logger()
	}

	slog.Info().Msg("NEW")

	s := time.Now()
	var n int64

	ctx.SetBodyStreamWriter(func(bw *bufio.Writer) {
		var err error
		var wn int64

		for {
			wn, err = heffalump.DefaultHeffalump.WriteHell(bw)
			n += wn
			if err != nil {
				slog.Trace().Err(err).Msg("END_ON_ERR")
				break
			}
		}

		slog.Info().
			Int64("BYTES", n).
			Dur("DURATION", time.Since(s)).
			Msg("FINISH")
	})

}

func getSrv(r *router.Router) *fasthttp.Server {
	if !config.RestrictConcurrency {
		config.MaxWorkers = fasthttp.DefaultConcurrency
	}

	log = config.GetLogger()

	return &fasthttp.Server{
		// User defined server name
		// Likely not useful if behind a reverse proxy without additional configuration of the proxy server.
		Name: config.FakeServerName,

		/*
			from fasthttp docs: "By default request read timeout is unlimited."
			My thinking here is avoiding some sort of weird oversized GET query just in case.
		*/
		ReadTimeout:        5 * time.Second,
		MaxRequestBodySize: 1 * 1024 * 1024,

		// Help curb abuse of HellPot (we've always needed this badly)
		MaxConnsPerIP:      10,
		MaxRequestsPerConn: 2,
		Concurrency:        config.MaxWorkers,

		// only accept GET requests
		GetOnly: true,

		// we don't care if a request ends up being handled by a different handler (in fact it probably will)
		KeepHijackedConns: true,

		CloseOnShutdown: true,

		// No need to keepalive, our response is a sort of keep-alive ;)
		DisableKeepalive: true,

		Handler: r.Handler,
		Logger:  log,
	}
}

// Server wraps HellPot's fasthttp server and listener configuration.
type Server struct {
	srv            *fasthttp.Server
	address        string
	useUnixSocket  bool
	unixSocketPath string
}

// NewServer builds HellPot's HTTP server and request router.
func NewServer() (*Server, error) {
	log = config.GetLogger()

	r := router.New()

	if config.MakeRobots && !config.CatchAll {
		r.GET("/robots.txt", robotsTXT)
	}

	if !config.CatchAll {
		for _, p := range config.Paths {
			route := routePath(p)
			log.Trace().Str("caller", "router").Str("route", route).Msg("Add route")
			r.GET(route, hellPot)
		}
	} else {
		log.Trace().Msg("Catch-All mode enabled...")
		r.GET("/{path:*}", hellPot)
	}

	srv := getSrv(r)

	//goland:noinspection GoBoolExpressions
	if !config.UseUnixSocket || runtime.GOOS == "windows" {
		return &Server{
			srv:     srv,
			address: net.JoinHostPort(config.HTTPBind, config.HTTPPort),
		}, nil
	}

	if len(config.UnixSocketPath) < 1 {
		return nil, fmt.Errorf("unix_socket_path configuration directive appears to be empty")
	}

	return &Server{
		srv:            srv,
		useUnixSocket:  true,
		unixSocketPath: config.UnixSocketPath,
	}, nil
}

// Serve starts the configured listener.
func (s *Server) Serve() error {
	if s.useUnixSocket {
		log.Info().Str("caller", s.unixSocketPath).Msg("Listening and serving HTTP...")
		return listenOnUnixSocket(s.unixSocketPath, s.srv)
	}

	log.Info().Str("caller", s.address).Msg("Listening and serving HTTP...")
	return s.srv.ListenAndServe(s.address)
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown() error {
	return s.srv.Shutdown()
}

// Serve starts our HTTP server and request router.
func Serve() error {
	srv, err := NewServer()
	if err != nil {
		return err
	}
	return srv.Serve()
}

func routePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if strings.HasPrefix(p, "/") {
		return p
	}
	return "/" + p
}
