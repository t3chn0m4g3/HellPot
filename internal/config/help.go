package config

import (
	"fmt"
	"io"
)

// PrintUsage writes CLI help to the provided writer.
func PrintUsage(w io.Writer) {
	_, _ = fmt.Fprintf(w, `%s v[%s]

Usage:
  %s [options]

Options:
  -c, --config <file>   Specify config file
  -v, --debug           Enable debug logging
  -vv, --trace          Enable trace logging
      --nocolor         Disable color and banner
      --banner          Show banner + version and exit
      --genconfig       Write default config to ./config.toml and exit
      --version         Show version and exit
  -h, --help            Show this help and exit
`, Title, Version, Title)
}
