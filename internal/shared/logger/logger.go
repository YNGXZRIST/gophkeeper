package logger

import (
	"fmt"
	"gophkeeper/internal/shared/errors/labelerrors"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	ModeProduction  = "production"
	ModeDevelopment = "development"
	ModeDefault     = ModeDevelopment

	dirPerm os.FileMode = 0o750

	DefaultLogDir = "./logs"

	// Rotation settings for production log files.
	logMaxSizeMB  = 50   // rotate once a file exceeds this size
	logMaxBackups = 5    // keep this many rotated files
	logMaxAgeDays = 30   // delete rotated files older than this
	logCompress   = true // gzip rotated files
)

// Config defines how the logger is built for the server.
type Config struct {
	// Mode selects the logging profile: ModeProduction or ModeDevelopment.
	Mode string
	// Dir is the directory where log files are stored.
	Dir string
	// Prefix identifies the component in log file names.
	Prefix  string
	Console bool
}

// Initialize builds a ready-to-use logger for the configured run mode.
// Development mode logs human-readable output to the console; production mode
// records structured logs to files, keeping errors separate from informational
// records, and mirrors them to stdout only when Config.Console is set. The
// returned logger should be flushed with Sync before the program exits.
func Initialize(c *Config) (*zap.Logger, error) {
	var (
		log *zap.Logger
		err error
	)

	switch c.Mode {
	case ModeProduction:
		log, err = newProduction(c)
	case ModeDevelopment:
		log, err = newDevelopment()
	default:
		err = fmt.Errorf("invalid mode: %s", c.Mode)
	}
	if err != nil {
		return nil, labelerrors.NewLabelError("LOGGER", err)
	}

	return log, nil
}

// newDevelopment builds a logger for local development.
func newDevelopment() (*zap.Logger, error) {
	log, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("could not create development logger: %w", err)
	}
	return log, nil
}

// newProduction builds a logger for production, routing informational records
// and errors to separate files, optionally mirroring them to stdout.
func newProduction(c *Config) (*zap.Logger, error) {
	if err := os.MkdirAll(c.Dir, dirPerm); err != nil {
		return nil, fmt.Errorf("could not create log directory %q: %w", c.Dir, err)
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encCfg)
	infoLevel := zap.LevelEnablerFunc(func(l zapcore.Level) bool { return l < zapcore.ErrorLevel })
	errLevel := zap.LevelEnablerFunc(func(l zapcore.Level) bool { return l >= zapcore.ErrorLevel })
	cores := []zapcore.Core{
		zapcore.NewCore(encoder, zapcore.AddSync(newLogWriter(c, "info")), infoLevel),
		zapcore.NewCore(encoder, zapcore.AddSync(newLogWriter(c, "errors")), errLevel),
	}
	if c.Console {
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.InfoLevel))
	}

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}

// newLogWriter builds a rotating log writer for the given record kind, bounding
// log files by size and age so they do not grow without limit.
func newLogWriter(c *Config, kind string) *lumberjack.Logger {
	name := kind + ".log"
	if c.Prefix != "" {
		name = c.Prefix + "_" + name
	}

	return &lumberjack.Logger{
		Filename:   filepath.Join(c.Dir, name),
		MaxSize:    logMaxSizeMB,
		MaxBackups: logMaxBackups,
		MaxAge:     logMaxAgeDays,
		Compress:   logCompress,
	}
}
