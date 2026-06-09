# HellPot

HellPot is an HTTP honeypot for clients that ignore `robots.txt`, probe common
application paths, or otherwise wander where they were told not to go. When a
client reaches a configured trap route, HellPot responds with an endless stream
of generated HTML-like text.

The generated stream comes from an internal adaptation of
[Heffalump](https://github.com/carlmjohnson/heffalump), a Markov-chain text
generator. HellPot keeps that code under `internal/heffalump` because it is an
implementation detail of the honeypot, not a public library API.

Under the hood HellPot uses:

- [fasthttp](https://github.com/valyala/fasthttp) for HTTP serving.
- [koanf](https://github.com/knadh/koanf) for TOML and environment-based configuration.
- [zerolog](https://github.com/rs/zerolog) for structured JSON logging.
- A bundled Markov source based on Nietzsche's *The Birth of Tragedy*.

## Safety Notes

HellPot is intentionally annoying to bad clients. That is the entire point.
Deploy it thoughtfully.

- Prefer running HellPot behind a reverse proxy and route only known trap paths
  to it.
- Do not put normal website traffic behind HellPot unless you really want those
  users to receive an endless response.
- Use `http.router.catchall = true` only for dedicated trap hosts or deliberate
  fallback/error routing.
- Tune `performance.restrict_concurrency`, `performance.max_workers`, and your
  reverse proxy rate limits for the hardware you actually have.
- Logs can grow quickly on noisy hosts. Use log rotation or Docker stdout logging
  with an external log collector.

## Requirements

- Go 1.26.4 or newer for building from source.
- Docker or another OCI-compatible builder if you want container images.
- Linux, Windows, macOS, and FreeBSD are supported build targets.
- Unix socket serving is supported on Linux, macOS, and FreeBSD. Windows falls
  back to TCP serving.

## Installation

### Release Binary

Download a release binary from:

```text
https://github.com/t3chn0m4g3/hellpot/releases/latest
```

Then run it directly:

```shell
./HellPot --help
./HellPot --genconfig
./HellPot -c config.toml
```

### From Source

```shell
git clone https://github.com/t3chn0m4g3/hellpot
cd hellpot
make
./HellPot --help
```

### Docker

Build a local image:

```shell
docker build --build-arg VERSION=0.60 -t hellpot:0.60 .
```

Run with the bundled Docker config:

```shell
docker run --rm -p 8080:8080 hellpot:0.60
```

Run with Docker Compose:

```shell
docker compose up --build
```

Run with your own config:

```shell
docker run --rm \
  -p 8080:8080 \
  -v "$PWD/config.toml:/config:ro" \
  hellpot:0.60
```

The Docker image runs from `scratch` as a numeric non-root user and uses
`docker_config.toml` by default. That file enables `logger.docker_logging`, so
logs go to stdout instead of a file under `/logs`.

Published image names used by the release workflow:

```text
t3chn0m4g3/hellpot
ghcr.io/t3chn0m4g3/hellpot
```

## Building From Source

The Makefile provides the usual local workflow:

```shell
make deps     # go mod tidy -v
make check    # go vet ./... and go test ./...
make build    # build ./cmd/HellPot into ./HellPot
make run      # go run ./cmd/HellPot
```

The default project version is stored in `VERSION`. `make build` embeds that
value in the binary unless you override it:

```shell
make build
VERSION=0.60 make build
```

Equivalent raw Go build:

```shell
go build -trimpath -o HellPot ./cmd/HellPot
```

Build with a version embedded in the binary:

```shell
go build -trimpath \
  -ldflags "-s -w -X main.version=0.60" \
  -o HellPot \
  ./cmd/HellPot
```

The release workflow passes the tag or ref name through the same linker flag.

## CLI Reference

```shell
HellPot [options]

Options:
  -c, --config <file>   Specify config file
  -v, --debug           Enable debug logging
  -vv, --trace          Enable trace logging
      --nocolor         Disable color and banner
      --banner          Show banner + version and exit
      --genconfig       Write default config to ./config.toml and exit
      --version         Show version and exit
  -h, --help            Show this help and exit
```

Examples:

```shell
./HellPot --version
./HellPot --genconfig
./HellPot -c config.toml
./HellPot -c config.toml -v
./HellPot -c config.toml -vv --nocolor
```

## Configuration

HellPot uses TOML configuration. Generate a starter file:

```shell
./HellPot --genconfig
```

The generated file is commented and includes every currently supported option.

Run with an explicit file:

```shell
./HellPot -c config.toml
```

When no `-c` or `--config` is provided, HellPot looks for configuration in this
order:

1. `/etc/HellPot/config.toml` on non-Windows systems.
2. The user config directory, usually `$XDG_CONFIG_HOME/HellPot/config.toml` or
   `$HOME/.config/HellPot/config.toml`.
3. `./config.toml` only if the user config directory is unavailable.

If the chosen default config file is missing, HellPot writes a default config to
that location and then loads it. Explicit `-c` paths must already exist.

### Environment Overrides

Every config key can be overridden with an environment variable prefixed with
`HELLPOT_`.

Rules:

- The prefix is stripped.
- The name is lowercased.
- Single underscores become dots.
- Double underscores become literal underscores inside a key segment.

Examples:

```shell
HELLPOT_HTTP_BIND__ADDR="0.0.0.0" ./HellPot
HELLPOT_HTTP_BIND__PORT="8081" ./HellPot
HELLPOT_LOGGER_DEBUG="false" ./HellPot
HELLPOT_HTTP_ROUTER_CATCHALL="true" ./HellPot
```

## Example Config

```toml
[deception]
  server_name = "nginx"

[http]
  bind_addr = "127.0.0.1"
  bind_port = "8080"
  real_ip_header = "X-Real-IP"
  uagent_string_blacklist = ["Cloudflare-Traffic-Manager", "curl"]
  use_unix_socket = false
  unix_socket_path = "/var/run/hellpot"
  unix_socket_permissions = "0666"

  [http.router]
    catchall = false
    makerobots = true
    paths = ["wp-login.php", "wp-login"]

[logger]
  debug = true
  trace = false
  nocolor = false
  use_date_filename = true
  docker_logging = false
  console_time_format = "3:04PM"

[performance]
  restrict_concurrency = false
  max_workers = 256
```

## Config Reference

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `deception.server_name` | string | `nginx` | Value used for the HTTP `Server` header. Reverse proxies may hide or replace it. |
| `http.bind_addr` | string | `127.0.0.1` | TCP bind address. Use `0.0.0.0` for container or public network binds. |
| `http.bind_port` | string | `8080` | TCP bind port. Kept as a string for compatibility with existing configs. |
| `http.real_ip_header` | string | `X-Real-IP` | Header used for logging the original client IP behind a reverse proxy. |
| `http.uagent_string_blacklist` | list | `["Cloudflare-Traffic-Manager"]` | Case-sensitive user-agent substrings that receive `404 Not found` instead of the endless stream. |
| `http.use_unix_socket` | bool | `false` | Enables Unix socket serving on supported platforms. Overrides TCP serving. |
| `http.unix_socket_path` | string | `/var/run/hellpot` | Unix socket path when `http.use_unix_socket` is enabled. |
| `http.unix_socket_permissions` | string | `0666` | Octal permissions applied to the Unix socket after binding. |
| `http.router.catchall` | bool | `false` | When true, all GET paths are trap paths and `robots.txt` is not generated by HellPot. |
| `http.router.makerobots` | bool | `true` | When true and catchall is false, HellPot serves `/robots.txt`. |
| `http.router.paths` | list | `["wp-login.php", "wp-login"]` | Trap paths used for route registration and `robots.txt` disallow entries. |
| `logger.debug` | bool | `true` | Enables debug-level logging. Can also be forced with `-v` or `--debug`. |
| `logger.trace` | bool | `false` | Enables trace-level logging. Can also be forced with `-vv` or `--trace`. |
| `logger.directory` | string | empty | Directory for JSON log files. Empty falls back to `$HOME/.local/share/HellPot/logs`. |
| `logger.nocolor` | bool | `false` | Disables color and the decorative banner. Defaults to true on Windows. |
| `logger.use_date_filename` | bool | `true` | Adds the current date/time to generated log filenames. |
| `logger.docker_logging` | bool | `false` | Sends logs to stdout and disables file logging. The bundled Docker config sets this to true. |
| `logger.console_time_format` | string | `3:04PM` | Time format passed to Go's `time.Format` for console output. |
| `performance.restrict_concurrency` | bool | `false` | When false, fasthttp uses its default concurrency. |
| `performance.max_workers` | int | `256` | Server concurrency when `performance.restrict_concurrency` is true. |

## Running HellPot

### Direct TCP

```toml
[http]
bind_addr = "127.0.0.1"
bind_port = "8080"
use_unix_socket = false
```

```shell
./HellPot -c config.toml
```

### Path-Based Trap Mode

Path-based mode serves only configured trap paths and optionally `/robots.txt`.

```toml
[http.router]
catchall = false
makerobots = true
paths = ["wp-login.php", "wp-login", "admin"]
```

Requests to `/wp-login.php`, `/wp-login`, and `/admin` trigger the endless
stream. `/robots.txt` advertises those paths as disallowed.

### Catchall Mode

Catchall mode traps every GET path:

```toml
[http.router]
catchall = true
```

Use this for a dedicated honeypot vhost, error-document backend, or intentionally
isolated trap service. In catchall mode HellPot does not register its own
`/robots.txt` handler.

### Unix Socket Mode

Unix socket mode is useful when the reverse proxy and HellPot run on the same
host:

```toml
[http]
use_unix_socket = true
unix_socket_path = "/run/hellpot.sock"
unix_socket_permissions = "0660"
```

HellPot unlinks an existing socket at that path before listening.

## Reverse Proxy Examples

### nginx, TCP Backend

```nginx
location = /robots.txt {
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_pass http://127.0.0.1:8080$request_uri;
}

location = /wp-login.php {
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_pass http://127.0.0.1:8080$request_uri;
}

location = /wp-login {
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_pass http://127.0.0.1:8080$request_uri;
}
```

### nginx, Unix Socket Backend

```nginx
location = /wp-login.php {
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_pass http://unix:/run/hellpot.sock:$request_uri;
}
```

### Apache ErrorDocument Backend

This pattern sends missing paths to HellPot while normal existing files continue
to be served by Apache.

```apache
<VirtualHost *:80>
    ServerName example.com

    ErrorDocument 404 "/content/404"

    <Directory "/var/www/html/.well-known/">
        ErrorDocument 404 default
    </Directory>

    ProxyPreserveHost On
    ProxyPass        "/content/" "http://127.0.0.1:8080/"
    ProxyPassReverse "/content/" "http://127.0.0.1:8080/"

    <Location "/content/">
        RequestHeader set X-Real-IP "%{REMOTE_ADDR}s"
        SetOutputFilter RATE_LIMIT
        SetEnv rate-limit 5
    </Location>
</VirtualHost>
```

## Docker Details

The Dockerfile uses a multi-stage build:

1. `golang:1.26.4` builds and runs `go vet` plus `go test`.
2. `scratch` runs the statically linked compiled binary as UID/GID `65532`.

Build:

```shell
docker build --build-arg VERSION=0.60 -t hellpot:0.60 .
```

Run with port mapping:

```shell
docker run --rm -p 8080:8080 hellpot:0.60
```

Run with a mounted config:

```shell
docker run --rm \
  -p 8080:8080 \
  -v "$PWD/docker_config.toml:/config:ro" \
  hellpot:0.60
```

Run with Docker Compose:

```shell
docker compose up --build
```

The Compose file builds the local Dockerfile, publishes `${HELLPOT_PORT:-8080}`
to container port `8080`, mounts `./docker_config.toml` read-only at `/config`,
and applies a read-only filesystem plus basic Linux capability hardening.

The image entrypoint is:

```text
/app -c /config
```

The default `docker_config.toml` binds `0.0.0.0:8080`, enables catchall mode, and
sends logs to stdout.

## Logging

Default non-Docker behavior:

- Pretty console logs are written to stdout.
- JSON log files are written under `logger.directory`.
- If `logger.directory` is empty, HellPot uses `$HOME/.local/share/HellPot/logs`.

Docker behavior:

- Set `logger.docker_logging = true`.
- HellPot writes structured logs to stdout.
- File logging is disabled.
- `--nocolor` or `logger.nocolor = true` suppresses color/banner output.

Use `-v` or `--debug` for debug logs, and `-vv` or `--trace` for trace logs.

## Development

Common checks:

```shell
go test ./...
go test -race ./...
go vet ./...
go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
docker build --build-arg VERSION=test -t hellpot:test .
```

Local build and run:

```shell
make build
./HellPot --version
./HellPot --help
```

## Project Layout

```text
cmd/HellPot/          CLI entrypoint and application orchestration
internal/config/      CLI flags, TOML loading, env overrides, logger setup
internal/http/        fasthttp server, routes, robots.txt, socket listeners
internal/heffalump/   internal Markov stream generator adapted from Heffalump
internal/extra/       banner rendering and presentation helpers
docker_config.toml    default container configuration
Dockerfile            multi-stage Go 1.26.4 scratch image build
Makefile              local development shortcuts
```

`internal/heffalump` is actively used by `internal/http`: trap responses call
`DefaultHeffalump.WriteHell` to produce the endless stream. It is internalized so
external projects do not rely on it as a public API.

## Attribution

HellPot's stream generator is based on
[Heffalump](https://github.com/carlmjohnson/heffalump) by Carl Johnson. The
original Heffalump MIT license is retained in `internal/heffalump/LICENSE`.

## Related Suffering

- https://github.com/ginger51011/pandoras_pot
  - A HellPot-inspired HTTP honeypot written in Rust.
