package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"gophkeeper/internal/server/logger"
	"gophkeeper/internal/server/transport"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Defaults applied when a value is not provided by any source.
const (
	DefaultTransport = "grpc"
	DefaultAddress   = ":8080"
	DefaultAppMode   = logger.ModeDefault
)

// Flags holds server configuration assembled from config file, env and CLI flags.
type Flags struct {
	Transport      string `json:"transport"     env:"TRANSPORT"`
	Address        string `json:"address"       env:"ADDRESS"`
	DSN            string `json:"database_dsn"  env:"DATABASE_DSN"`
	AppMode        string `json:"app_mode"      env:"APP_MODE"`
	LogDir         string `json:"log_dir"       env:"LOG_DIR"`
	JWTSecret      string `json:"jwt_secret"     env:"JWT_SECRET"`
	RefreshSecret  string `json:"refresh_secret" env:"REFRESH_SECRET"`
	ConfigFilePath string `json:"-"`
}

// Load assembles configuration from all sources with ascending priority:
//
//	defaults (lowest) -> JSON config -> environment -> CLI flags (highest)
//
// args is the program arguments without the binary name (usually os.Args[1:]).
func Load(args []string) (*Flags, error) {
	opt := new(Flags)

	// 1. JSON config — lowest priority. Path comes from -config/-c or CONFIG env.
	if path := configPath(args); path != "" {
		opt.ConfigFilePath = path
		if err := opt.parseConfig(path); err != nil {
			return nil, fmt.Errorf("config: %w", err)
		}
	}

	// 2. Environment — overrides config for variables that are set.
	if err := opt.parseEnv(); err != nil {
		return nil, fmt.Errorf("env: %w", err)
	}

	// 3. CLI flags — highest priority, override everything that was passed.
	if err := opt.parseArgs(args); err != nil {
		return nil, fmt.Errorf("args: %w", err)
	}

	// 4. Defaults — fill whatever is still empty.
	applyDefaults(opt)

	if err := validate(opt); err != nil {
		return nil, err
	}

	return opt, nil
}

// parseConfig applies values from the JSON config file, the lowest-priority source.
func (opt *Flags) parseConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var cfg Flags
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return fmt.Errorf("decoding config file: %w", err)
	}

	override(opt, &cfg)
	return nil
}

// parseEnv applies values from environment variables, overriding the config file.
func (opt *Flags) parseEnv() error {
	cfg := new(Flags)
	if err := env.Parse(cfg); err != nil {
		return fmt.Errorf("parsing env: %w", err)
	}
	override(opt, cfg)
	return nil
}

// parseArgs applies values from CLI flags, the highest-priority source; only
// flags the user actually passed take effect.
func (opt *Flags) parseArgs(args []string) error {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)

	var (
		transport string
		address   string
		dsn       string
		mode      string
		logDir    string
		cfgPath   string
	)

	fs.StringVar(&transport, "t", "", "transport: grpc or http")
	fs.StringVar(&address, "a", "", "listen address (host:port)")
	fs.StringVar(&dsn, "d", "", "database DSN")
	fs.StringVar(&mode, "m", "", "app mode: development or production")
	fs.StringVar(&logDir, "l", "", "log directory")
	fs.StringVar(&cfgPath, "config", "", "config file path")
	fs.StringVar(&cfgPath, "c", "", "config file path (shorthand)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { visited[f.Name] = true })

	if visited["t"] {
		opt.Transport = transport
	}
	if visited["a"] {
		opt.Address = address
	}
	if visited["d"] {
		opt.DSN = dsn
	}
	if visited["m"] {
		opt.AppMode = mode
	}
	if visited["l"] {
		opt.LogDir = logDir
	}
	if visited["config"] || visited["c"] {
		opt.ConfigFilePath = cfgPath
	}

	return nil
}

// override applies the set fields of src onto dst, so a later source wins over an earlier one.
func override(dst, src *Flags) {
	if src.Transport != "" {
		dst.Transport = src.Transport
	}
	if src.Address != "" {
		dst.Address = src.Address
	}
	if src.DSN != "" {
		dst.DSN = src.DSN
	}
	if src.AppMode != "" {
		dst.AppMode = src.AppMode
	}
	if src.LogDir != "" {
		dst.LogDir = src.LogDir
	}
	if src.JWTSecret != "" {
		dst.JWTSecret = src.JWTSecret
	}
	if src.RefreshSecret != "" {
		dst.RefreshSecret = src.RefreshSecret
	}
}

// applyDefaults sets defaults for fields left empty by every source.
func applyDefaults(opt *Flags) {
	if opt.Transport == "" {
		opt.Transport = DefaultTransport
	}
	if opt.Address == "" {
		opt.Address = DefaultAddress
	}
	if opt.AppMode == "" {
		opt.AppMode = DefaultAppMode
	}
	if opt.LogDir == "" {
		opt.LogDir = logger.DefaultLogDir
	}
}

// validate checks invariants that cannot be defaulted.
func validate(opt *Flags) error {
	switch opt.Transport {
	case transport.GRPC, transport.HTTP:
	default:
		return fmt.Errorf("invalid transport %q: want grpc or http", opt.Transport)
	}
	switch opt.AppMode {
	case logger.ModeDevelopment, logger.ModeProduction:
	default:
		return fmt.Errorf("invalid app mode %q: want development or production", opt.AppMode)
	}
	if opt.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required: set it in the config file or JWT_SECRET env")
	}
	if opt.RefreshSecret == "" {
		return fmt.Errorf("refresh_secret is required: set it in the config file or REFRESH_SECRET env")
	}
	return nil
}

// configPath resolves the config file path from the -config/-c flag or the
// CONFIG environment variable, returning empty when none is set.
func configPath(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		for _, p := range []string{"-c=", "-config=", "--config="} {
			if v, ok := strings.CutPrefix(a, p); ok {
				return v
			}
		}
		if a == "-c" || a == "-config" || a == "--config" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return os.Getenv("CONFIG")
}
