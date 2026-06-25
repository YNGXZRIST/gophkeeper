package logger

import (
	"context"
	"fmt"
	"gophkeeper/internal/shared/errors/labelerrors"
	"io"
	"log/slog"
	"os"
	"path/filepath"

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

	// levelCeiling sits above every slog level, bounding the open-ended upper
	// half of the error file's range.
	levelCeiling = slog.Level(1 << 30)
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
// records, and mirrors them to stdout only when Config.Console is set.
func Initialize(c *Config) (*slog.Logger, error) {
	var (
		log *slog.Logger
		err error
	)

	switch c.Mode {
	case ModeProduction:
		log, err = newProduction(c)
	case ModeDevelopment:
		log = newDevelopment()
	default:
		err = fmt.Errorf("invalid mode: %s", c.Mode)
	}
	if err != nil {
		return nil, labelerrors.NewLabelError("LOGGER", err)
	}

	return log, nil
}

// newDevelopment builds a logger for local development that writes
// human-readable records to stderr down to the debug level.
func newDevelopment() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))
}

// newProduction builds a logger for production, routing informational records
// and errors to separate files, optionally mirroring them to stdout.
func newProduction(c *Config) (*slog.Logger, error) {
	if err := os.MkdirAll(c.Dir, dirPerm); err != nil {
		return nil, fmt.Errorf("could not create log directory %q: %w", c.Dir, err)
	}

	opts := &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true}
	handlers := []slog.Handler{
		newRangeHandler(newLogWriter(c, "info"), opts, slog.LevelDebug, slog.LevelError),
		newRangeHandler(newLogWriter(c, "errors"), opts, slog.LevelError, levelCeiling),
	}
	if c.Console {
		handlers = append(handlers, slog.NewJSONHandler(os.Stdout, opts))
	}

	return slog.New(newFanoutHandler(handlers...)), nil
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

// rangeHandler enables an underlying handler only for records whose level falls
// in [min, max); it lets a single logger send informational records and errors
// to different writers.
type rangeHandler struct {
	slog.Handler
	min, max slog.Level
}

func newRangeHandler(w io.Writer, opts *slog.HandlerOptions, min, max slog.Level) rangeHandler {
	return rangeHandler{Handler: slog.NewJSONHandler(w, opts), min: min, max: max}
}

func (h rangeHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.min && l < h.max
}

func (h rangeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return rangeHandler{Handler: h.Handler.WithAttrs(attrs), min: h.min, max: h.max}
}

func (h rangeHandler) WithGroup(name string) slog.Handler {
	return rangeHandler{Handler: h.Handler.WithGroup(name), min: h.min, max: h.max}
}

// fanoutHandler dispatches each record to every child handler that enables it,
// so one logger can write to several destinations at once.
type fanoutHandler struct {
	handlers []slog.Handler
}

func newFanoutHandler(handlers ...slog.Handler) fanoutHandler {
	return fanoutHandler{handlers: handlers}
}

func (h fanoutHandler) Enabled(ctx context.Context, l slog.Level) bool {
	for _, sub := range h.handlers {
		if sub.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

func (h fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, sub := range h.handlers {
		if !sub.Enabled(ctx, r.Level) {
			continue
		}
		if err := sub.Handle(ctx, r.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (h fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(h.handlers))
	for i, sub := range h.handlers {
		next[i] = sub.WithAttrs(attrs)
	}
	return fanoutHandler{handlers: next}
}

func (h fanoutHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(h.handlers))
	for i, sub := range h.handlers {
		next[i] = sub.WithGroup(name)
	}
	return fanoutHandler{handlers: next}
}
