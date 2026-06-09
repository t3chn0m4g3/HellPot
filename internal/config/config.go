package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/rs/zerolog"

	"github.com/knadh/koanf/parsers/toml"
	viper "github.com/knadh/koanf/v2"
)

// generic vars
var (
	noColorForce = false
	forceDebug   = false
	forceTrace   = false
	home         string
	snek         = viper.New(".")
)

func init() {
	home, _ = os.UserHomeDir()
	if home == "" {
		home = os.Getenv("HOME")
	}
	if home == "" {
		println("WARNING: could not determine home directory")
	}
}

// exported generic vars
var (
	// Trace is the value of our trace (extra verbose)  on/off toggle as per the current configuration.
	Trace bool
	// Debug is the value of our debug (verbose) on/off toggle as per the current configuration.
	Debug bool
	// Filename returns the current location of our toml config file.
	Filename string
)

func writeConfig() (string, error) {
	prefConfigLocation, _ := os.UserConfigDir()
	if prefConfigLocation != "" {
		prefConfigLocation = filepath.Join(prefConfigLocation, Title)
	}

	if prefConfigLocation == "" {
		home, _ = os.UserHomeDir()
		prefConfigLocation = filepath.Join(home, ".config", Title)
	}

	if _, err := os.Stat(prefConfigLocation); os.IsNotExist(err) {
		if err = os.MkdirAll(prefConfigLocation, 0o750); err != nil && !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("error writing new config: %w", err)
		}
	}

	Filename = filepath.Join(prefConfigLocation, "config.toml")

	written, err := writeDefaultConfig(Filename)
	if err != nil {
		return "", fmt.Errorf("error writing new config: %w", err)
	}
	Filename = written

	return Filename, nil
}

func resetState(opts CLIOptions) {
	snek = viper.New(".")
	noColorForce = opts.NoColor
	forceDebug = opts.Debug
	forceTrace = opts.Trace

	BannerOnly = opts.BannerOnly
	GenConfig = opts.GenConfig
	NoColor = false
	DockerLogging = false
	MakeRobots = false
	CatchAll = false
	ConsoleTimeFormat = ""
	HTTPBind = ""
	HTTPPort = ""
	HeaderName = ""
	Paths = nil
	UseUnixSocket = false
	UnixSocketPath = ""
	UnixSocketPermissions = 0
	UseragentBlacklistMatchers = nil
	RestrictConcurrency = false
	MaxWorkers = 0
	FakeServerName = ""
	Trace = false
	Debug = false
	Filename = ""

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// GenerateConfig writes the default configuration file to path.
func GenerateConfig(path string, opts CLIOptions) (string, error) {
	resetState(opts)
	if err := setDefaults(); err != nil {
		return "", err
	}
	return writeDefaultConfig(path)
}

// Init initializes the configuration engine and exported configuration values.
func Init(opts CLIOptions) error {
	resetState(opts)

	if err := setDefaults(); err != nil {
		return err
	}

	if opts.ConfigPath != "" {
		if err := loadCustomConfig(opts.ConfigPath); err != nil {
			return err
		}
		return associateExportedVariables()
	}

	chosen := ""
	exists := false

	uconf, _ := os.UserConfigDir()

	switch runtime.GOOS {
	case "windows":
		//
	default:
		if _, err := os.Stat(filepath.Join("/etc/", Title, "config.toml")); err == nil {
			chosen = filepath.Join("/etc/", Title, "config.toml")
			exists = true
		}
	}

	if chosen == "" && uconf == "" && home != "" {
		uconf = filepath.Join(home, ".config")
	}

	if chosen == "" && uconf != "" {
		_ = os.MkdirAll(filepath.Join(uconf, Title), 0750)
		chosen = filepath.Join(uconf, Title, "config.toml")
		if _, err := os.Stat(chosen); err == nil {
			exists = true
		}
	}

	if chosen == "" {
		pwd, _ := os.Getwd()
		if _, err := os.Stat("./config.toml"); err == nil {
			chosen = "./config.toml"
			exists = true
		} else {
			if _, err := os.Stat(filepath.Join(pwd, "config.toml")); err == nil {
				chosen = filepath.Join(pwd, "config.toml")
				exists = true
			}
		}
	}

	if chosen == "" || !exists {
		var err error
		chosen, err = writeConfig()
		if err != nil {
			return err
		}
	}

	Filename = chosen

	if err := snek.Load(file.Provider(chosen), toml.Parser()); err != nil {
		return fmt.Errorf("failed to load config file %q: %w", chosen, err)
	}

	/*	snek.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
		snek.SetEnvPrefix(Title)
		snek.AutomaticEnv()
	*/
	return associateExportedVariables()
}

func loadCustomConfig(path string) error {
	Filename, _ = filepath.Abs(path)

	if err := snek.Load(file.Provider(Filename), toml.Parser()); err != nil {
		return fmt.Errorf("failed to load specified config file %q: %w", Filename, err)
	}

	return nil
}

func processOpts() {
	// string options and their exported variables
	stringOpt := map[string]*string{
		"http.bind_addr":             &HTTPBind,
		"http.bind_port":             &HTTPPort,
		"http.real_ip_header":        &HeaderName,
		"logger.directory":           &logDir,
		"logger.console_time_format": &ConsoleTimeFormat,
		"deception.server_name":      &FakeServerName,
	}
	// string slice options and their exported variables
	strSliceOpt := map[string]*[]string{
		"http.router.paths":            &Paths,
		"http.uagent_string_blacklist": &UseragentBlacklistMatchers,
	}
	// bool options and their exported variables
	boolOpt := map[string]*bool{
		"performance.restrict_concurrency": &RestrictConcurrency,
		"http.use_unix_socket":             &UseUnixSocket,
		"logger.debug":                     &Debug,
		"logger.trace":                     &Trace,
		"logger.nocolor":                   &NoColor,
		"logger.docker_logging":            &DockerLogging,
		"http.router.makerobots":           &MakeRobots,
		"http.router.catchall":             &CatchAll,
	}
	// integer options and their exported variables
	intOpt := map[string]*int{
		"performance.max_workers": &MaxWorkers,
	}

	for key, opt := range stringOpt {
		*opt = snek.String(key)
	}
	for key, opt := range strSliceOpt {
		*opt = snek.Strings(key)
	}
	for key, opt := range boolOpt {
		*opt = snek.Bool(key)
	}
	for key, opt := range intOpt {
		*opt = snek.Int(key)
	}
}

func associateExportedVariables() error {
	_ = snek.Load(env.Provider("HELLPOT_", ".", func(s string) string {
		s = strings.TrimPrefix(s, "HELLPOT_")
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "__", " ")
		s = strings.ReplaceAll(s, "_", ".")
		s = strings.ReplaceAll(s, " ", "_")
		return s
	}), nil)

	processOpts()

	if noColorForce {
		NoColor = true
	}

	if UseUnixSocket {
		UnixSocketPath = snek.String("http.unix_socket_path")
		parsedPermissions, err := strconv.ParseUint(snek.String("http.unix_socket_permissions"), 8, 32)
		if err != nil {
			return fmt.Errorf("invalid http.unix_socket_permissions: %w", err)
		}
		UnixSocketPermissions = uint32(parsedPermissions)
	}

	// We set exported variables here so that it tracks when accessed from other packages.

	if Debug || forceDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		Debug = true
	}
	if Trace || forceTrace {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
		Trace = true
	}

	return nil
}
