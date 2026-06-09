package http

import (
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/t3chn0m4g3/hellpot/internal/config"
)

func robotsTXT(ctx *fasthttp.RequestCtx) {
	slog := log.With().
		Str("USERAGENT", string(ctx.UserAgent())).
		Str("REMOTE_ADDR", getRealRemote(ctx)).
		Interface("URL", string(ctx.RequestURI())).Logger()
	paths := &strings.Builder{}
	paths.WriteString("User-agent: *\r\n")
	for _, p := range config.Paths {
		paths.WriteString("Disallow: ")
		paths.WriteString(routePath(p))
		paths.WriteString("\r\n")
	}
	paths.WriteString("\r\n")

	slog.Debug().
		Strs("PATHS", config.Paths).
		Msg("SERVE_ROBOTS")

	ctx.SetContentType("text/plain; charset=utf-8")
	if _, err := ctx.WriteString(paths.String()); err != nil {
		slog.Error().Err(err).Msg("SERVE_ROBOTS_ERROR")
	}
}
