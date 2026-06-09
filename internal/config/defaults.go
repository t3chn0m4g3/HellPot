package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var configSections = []string{"logger", "http", "performance", "deception"}

var defOpts = map[string]map[string]interface{}{
	"logger": {
		"debug":               true,
		"trace":               false,
		"directory":           "",
		"nocolor":             false,
		"use_date_filename":   true,
		"docker_logging":      false,
		"console_time_format": time.Kitchen,
	},
	"http": {
		"use_unix_socket":         false,
		"unix_socket_path":        "/var/run/hellpot",
		"unix_socket_permissions": "0666",
		"bind_addr":               "127.0.0.1",
		"bind_port":               "8080",
		"real_ip_header":          "X-Real-IP",

		"router": map[string]interface{}{
			"catchall":   false,
			"makerobots": true,
			"paths": []string{
				"wp-login.php",
				"wp-login",
			},
		},
		"uagent_string_blacklist": []string{
			"Cloudflare-Traffic-Manager",
		},
	},
	"performance": {
		"restrict_concurrency": false,
		"max_workers":          256,
	},
	"deception": {
		"server_name": "nginx",
	},
}

const defaultConfigTemplate = `[deception]
  # Used as the HTTP "Server" response header.
  # Reverse proxies may hide or replace this header.
  server_name = "nginx"

[http]
  # TCP listener address and port.
  # Use 127.0.0.1 when HellPot is only reached through a local reverse proxy.
  # Use 0.0.0.0 for Docker or direct public binds.
  bind_addr = "127.0.0.1"
  bind_port = "8080"

  # Header containing the original client IP in reverse proxy deployments.
  real_ip_header = "X-Real-IP"

  # Case-sensitive user-agent substrings that should not be trapped.
  # Matching clients receive "Not found" for every request.
  uagent_string_blacklist = ["Cloudflare-Traffic-Manager"]

  # Unix socket listener.
  # When use_unix_socket is true, the Unix socket listener overrides TCP.
  # This is ignored on Windows, where HellPot always uses TCP.
  use_unix_socket = false
  unix_socket_path = "/var/run/hellpot"
  unix_socket_permissions = "0666"

  [http.router]
    # When true, every GET request path is trapped.
    # Catchall mode disables HellPot's robots.txt handler.
    catchall = false

    # When true and catchall is false, HellPot serves /robots.txt.
    makerobots = true

    # Trap paths and robots.txt Disallow entries.
    # Entries may be written with or without a leading slash.
    # Only used when catchall is false.
    paths = [
      "wp-login.php",
      "wp-login"
    ]

[logger]
  # Verbose logging. Same effect as -v or --debug.
  debug = true

  # Extra verbose logging. Same effect as -vv or --trace.
  trace = false

  # Directory for JSON log files.
  # Empty means $HOME/.local/share/HellPot/logs.
  directory = ""

  # Disable color and the decorative banner.
  # This defaults to true on Windows-generated configs.
  nocolor = {{NOCOLOR}}

  # Include the current date/time in generated log file names.
  use_date_filename = true

  # Send structured logs to stdout and disable file logging.
  # This should usually be true in Docker and false for normal host installs.
  docker_logging = false

  # Go time format used by pretty console logs.
  # See https://pkg.go.dev/time#pkg-constants for common formats.
  console_time_format = "3:04PM"

[performance]
  # Restrict fasthttp concurrency to max_workers.
  restrict_concurrency = false

  # Only used when restrict_concurrency is true.
  max_workers = 256
`

func defaultConfigTOML() []byte {
	noColor := "false"
	if runtime.GOOS == "windows" {
		noColor = "true"
	}
	return []byte(strings.ReplaceAll(defaultConfigTemplate, "{{NOCOLOR}}", noColor))
}

func writeDefaultConfig(path string) (string, error) {
	if err := os.WriteFile(path, defaultConfigTOML(), 0o600); err != nil {
		return "", err
	}

	pathAbs, absErr := filepath.Abs(path)
	if absErr == nil && pathAbs != "" {
		path = pathAbs
	}

	return path, nil
}

func setDefaults() error {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		defOpts["logger"]["nocolor"] = true
	}
	for _, def := range configSections {
		for key, val := range defOpts[def] {
			if _, ok := val.(map[string]interface{}); !ok {
				if err := snek.Set(def+"."+key, val); err != nil {
					return err
				}
				continue
			}
			for k, v := range val.(map[string]interface{}) {
				if err := snek.Set(def+"."+key+"."+k, v); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
