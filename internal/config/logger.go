package config

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var (
	// CurrentLogFile is used for accessing the location of the currently used log file across packages.
	CurrentLogFile string
	logFile        io.Writer
	logDir         string
	logger         zerolog.Logger
)

func prepLogDir() error {
	logDir = snek.String("logger.directory")
	if logDir == "" {
		logDir = filepath.Join(home, ".local", "share", Title, "logs")
	}
	return os.MkdirAll(logDir, 0o750)
}

// StartLogger instantiates an instance of our zerolog loggger so we can hook it in our main package.
// While this does return a logger, it should not be used for additional retrievals of the logger. Use GetLogger().
func StartLogger(pretty bool, targets ...io.Writer) (zerolog.Logger, error) {
	logFileName := "HellPot"

	if snek.Bool("logger.use_date_filename") {
		tn := strings.ReplaceAll(time.Now().Format(time.RFC822), " ", "_")
		tn = strings.ReplaceAll(tn, ":", "-")
		logFileName = logFileName + "_" + tn
	}

	var err error

	switch {
	case len(targets) > 0:
		logFile = io.MultiWriter(targets...)
	default:
		if err = prepLogDir(); err != nil {
			return zerolog.Logger{}, fmt.Errorf("cannot create log directory %q: %w", logDir, err)
		}
		CurrentLogFile = path.Join(logDir, logFileName+".log")
		// #nosec G304 -- logDir is an explicit configuration value; the file name is generated here.
		logFile, err = os.OpenFile(CurrentLogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			return zerolog.Logger{}, fmt.Errorf("cannot create log file %q: %w", CurrentLogFile, err)
		}
	}

	var logWriter = logFile

	if pretty {
		logWriter = zerolog.MultiLevelWriter(zerolog.ConsoleWriter{TimeFormat: ConsoleTimeFormat, NoColor: NoColor, Out: os.Stdout}, logFile)
	}

	logger = zerolog.New(logWriter).With().Timestamp().Logger()
	return logger, nil
}

// GetLogger retrieves our global logger object.
func GetLogger() *zerolog.Logger {
	// future logic here
	return &logger
}
