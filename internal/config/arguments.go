package config

import (
	"flag"
	"io"
)

// CLIOptions contains flags parsed from the command line before configuration is loaded.
type CLIOptions struct {
	ConfigPath  string
	Debug       bool
	Trace       bool
	NoColor     bool
	BannerOnly  bool
	GenConfig   bool
	ShowVersion bool
}

// ParseArgs parses HellPot's CLI flags without touching configuration files.
func ParseArgs(args []string, stdout, stderr io.Writer) (CLIOptions, error) {
	var opts CLIOptions
	var help bool

	fs := flag.NewFlagSet(Title, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		PrintUsage(stdout)
	}

	fs.StringVar(&opts.ConfigPath, "config", "", "Specify config file")
	fs.StringVar(&opts.ConfigPath, "c", "", "Specify config file")
	fs.BoolVar(&opts.Debug, "debug", false, "enable debug logging")
	fs.BoolVar(&opts.Debug, "v", false, "enable debug logging")
	fs.BoolVar(&opts.Trace, "trace", false, "enable trace logging")
	fs.BoolVar(&opts.Trace, "vv", false, "enable trace logging")
	fs.BoolVar(&opts.NoColor, "nocolor", false, "disable color and banner")
	fs.BoolVar(&opts.BannerOnly, "banner", false, "show banner + version and exit")
	fs.BoolVar(&opts.GenConfig, "genconfig", false, "write default config to ./config.toml and exit")
	fs.BoolVar(&opts.ShowVersion, "version", false, "show version and exit")
	fs.BoolVar(&help, "help", false, "show this help and exit")
	fs.BoolVar(&help, "h", false, "show this help and exit")

	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if help {
		PrintUsage(stdout)
		return opts, flag.ErrHelp
	}
	return opts, nil
}
