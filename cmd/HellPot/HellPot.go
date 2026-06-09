package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/t3chn0m4g3/hellpot/internal/config"
	"github.com/t3chn0m4g3/hellpot/internal/extra"
	hellhttp "github.com/t3chn0m4g3/hellpot/internal/http"
)

var version string // set by linker

type appServer interface {
	Serve() error
	Shutdown() error
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	config.Version = normalizedVersion(version, config.Version)

	opts, err := config.ParseArgs(args, stdout, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	config.NoColor = opts.NoColor

	if opts.ShowVersion {
		_, _ = fmt.Fprintln(stdout, config.Version)
		return 0
	}

	if opts.BannerOnly {
		extra.BannerTo(stdout)
		return 0
	}

	if opts.GenConfig {
		path, err := config.GenerateConfig("./config.toml", opts)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "Default config written to %s\n", path)
		return 0
	}

	if err = config.Init(opts); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	var log zerolog.Logger

	if config.DockerLogging {
		config.CurrentLogFile = "/dev/stdout"
		config.NoColor = true
		log, err = config.StartLogger(false, stdout)
	} else {
		log, err = config.StartLogger(true)
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	if !config.DockerLogging {
		extra.BannerTo(stdout)
	}

	log.Info().Str("caller", "config").Str("file", config.Filename).Msg(config.Filename)
	log.Info().Str("caller", "logger").Msg(config.CurrentLogFile)
	log.Debug().Str("caller", "logger").Msg("debug enabled")
	log.Trace().Str("caller", "logger").Msg("trace enabled")

	srv, err := hellhttp.NewServer()
	if err != nil {
		log.Error().Err(err).Msg("HTTP setup error")
		return 1
	}

	return serveUntilSignal(srv, log)
}

func serveUntilSignal(srv appServer, log zerolog.Logger) int {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stopChan)

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Serve()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			log.Error().Err(err).Msg("HTTP error")
			return 1
		}
		return 0
	case sig := <-stopChan:
		log.Warn().Str("signal", sig.String()).Msg("Shutting down server...")
		if err := srv.Shutdown(); err != nil {
			log.Error().Err(err).Msg("HTTP shutdown error")
			return 1
		}
		if err := <-errChan; err != nil {
			log.Debug().Err(err).Msg("HTTP server stopped after shutdown")
		}
		return 0
	}
}

func normalizedVersion(linkerVersion, fallback string) string {
	v := strings.TrimSpace(linkerVersion)
	if v == "" {
		return fallback
	}

	v = strings.TrimPrefix(v, "refs/tags/")
	v = strings.TrimPrefix(v, "refs/heads/")
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return fallback
	}

	return v
}
