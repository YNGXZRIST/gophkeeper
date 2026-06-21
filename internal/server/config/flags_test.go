package config

import (
	"os"
	"path/filepath"
	"testing"

	"gophkeeper/internal/server/transport"
	"gophkeeper/internal/shared/logger"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("CONFIG", "")
	t.Setenv("TRANSPORT", "")
	t.Setenv("ADDRESS", "")
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("APP_MODE", "")
	t.Setenv("LOG_DIR", "")
	t.Setenv("JWT_SECRET", "jwt")
	t.Setenv("REFRESH_SECRET", "refresh")

	got, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Transport != DefaultTransport {
		t.Errorf("Transport = %q, want %q", got.Transport, DefaultTransport)
	}
	if got.Address != DefaultAddress {
		t.Errorf("Address = %q, want %q", got.Address, DefaultAddress)
	}
	if got.AppMode != DefaultAppMode {
		t.Errorf("AppMode = %q, want %q", got.AppMode, DefaultAppMode)
	}
	if got.LogDir != logger.DefaultLogDir {
		t.Errorf("LogDir = %q, want %q", got.LogDir, logger.DefaultLogDir)
	}
}

func TestLoadFlagsOverrideEnv(t *testing.T) {
	t.Setenv("CONFIG", "")
	t.Setenv("TRANSPORT", "grpc")
	t.Setenv("ADDRESS", ":1111")
	t.Setenv("APP_MODE", logger.ModeProduction)
	t.Setenv("JWT_SECRET", "jwt")
	t.Setenv("REFRESH_SECRET", "refresh")

	args := []string{
		"-t", transport.HTTP,
		"-a", ":2222",
		"-d", "postgres://x",
		"-m", logger.ModeDevelopment,
		"-l", "/tmp/logs",
	}
	got, err := Load(args)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Transport != transport.HTTP {
		t.Errorf("Transport = %q, want %q", got.Transport, transport.HTTP)
	}
	if got.Address != ":2222" {
		t.Errorf("Address = %q, want :2222", got.Address)
	}
	if got.DSN != "postgres://x" {
		t.Errorf("DSN = %q, want postgres://x", got.DSN)
	}
	if got.AppMode != logger.ModeDevelopment {
		t.Errorf("AppMode = %q, want %q", got.AppMode, logger.ModeDevelopment)
	}
	if got.LogDir != "/tmp/logs" {
		t.Errorf("LogDir = %q, want /tmp/logs", got.LogDir)
	}
}

func TestLoadEnvOverridesNothingSet(t *testing.T) {
	t.Setenv("CONFIG", "")
	t.Setenv("TRANSPORT", logger.ModeProduction)
	t.Setenv("JWT_SECRET", "jwt")
	t.Setenv("REFRESH_SECRET", "refresh")

	_, err := Load(nil)
	if err == nil {
		t.Fatal("Load() error = nil, want invalid transport error")
	}
}

func TestLoadFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	content := `{
		"transport": "grpc",
		"address": ":9999",
		"database_dsn": "dsn-from-file",
		"app_mode": "production",
		"log_dir": "/var/log/app",
		"jwt_secret": "jwt-file",
		"refresh_secret": "refresh-file"
	}`
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	clearConfigEnv(t)

	got, err := Load([]string{"-config", cfgPath})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Address != ":9999" {
		t.Errorf("Address = %q, want :9999", got.Address)
	}
	if got.DSN != "dsn-from-file" {
		t.Errorf("DSN = %q, want dsn-from-file", got.DSN)
	}
	if got.JWTSecret != "jwt-file" {
		t.Errorf("JWTSecret = %q, want jwt-file", got.JWTSecret)
	}
	if got.ConfigFilePath != cfgPath {
		t.Errorf("ConfigFilePath = %q, want %q", got.ConfigFilePath, cfgPath)
	}
}

func TestLoadConfigViaCONFIGEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	content := `{"jwt_secret":"j","refresh_secret":"r"}`
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	clearConfigEnv(t)
	t.Setenv("CONFIG", cfgPath)

	got, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.JWTSecret != "j" || got.RefreshSecret != "r" {
		t.Errorf("secrets = (%q,%q), want (j,r)", got.JWTSecret, got.RefreshSecret)
	}
}

func TestLoadConfigFileMissing(t *testing.T) {
	clearConfigEnv(t)
	_, err := Load([]string{"-c=/nonexistent/path/config.json"})
	if err == nil {
		t.Fatal("Load() error = nil, want open error")
	}
}

func TestLoadConfigFileInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(cfgPath, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	clearConfigEnv(t)
	_, err := Load([]string{"-c", cfgPath})
	if err == nil {
		t.Fatal("Load() error = nil, want decode error")
	}
}

func TestLoadInvalidFlag(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("JWT_SECRET", "jwt")
	t.Setenv("REFRESH_SECRET", "refresh")
	_, err := Load([]string{"-unknown"})
	if err == nil {
		t.Fatal("Load() error = nil, want flag parse error")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		flags   Flags
		wantErr bool
	}{
		{
			name:  "valid grpc development",
			flags: Flags{Transport: transport.GRPC, AppMode: logger.ModeDevelopment, JWTSecret: "j", RefreshSecret: "r"},
		},
		{
			name:  "valid http production",
			flags: Flags{Transport: transport.HTTP, AppMode: logger.ModeProduction, JWTSecret: "j", RefreshSecret: "r"},
		},
		{
			name:    "invalid transport",
			flags:   Flags{Transport: "ftp", AppMode: logger.ModeProduction, JWTSecret: "j", RefreshSecret: "r"},
			wantErr: true,
		},
		{
			name:    "invalid app mode",
			flags:   Flags{Transport: transport.GRPC, AppMode: "staging", JWTSecret: "j", RefreshSecret: "r"},
			wantErr: true,
		},
		{
			name:    "missing jwt secret",
			flags:   Flags{Transport: transport.GRPC, AppMode: logger.ModeProduction, RefreshSecret: "r"},
			wantErr: true,
		},
		{
			name:    "missing refresh secret",
			flags:   Flags{Transport: transport.GRPC, AppMode: logger.ModeProduction, JWTSecret: "j"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(&tt.flags)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigPath(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  string
		want string
	}{
		{name: "long flag space", args: []string{"-config", "a.json"}, want: "a.json"},
		{name: "short flag space", args: []string{"-c", "b.json"}, want: "b.json"},
		{name: "double dash space", args: []string{"--config", "c.json"}, want: "c.json"},
		{name: "short equals", args: []string{"-c=d.json"}, want: "d.json"},
		{name: "long equals", args: []string{"-config=e.json"}, want: "e.json"},
		{name: "double dash equals", args: []string{"--config=f.json"}, want: "f.json"},
		{name: "flag at end no value", args: []string{"-config"}, want: ""},
		{name: "fallback to env", args: nil, env: "g.json", want: "g.json"},
		{name: "none", args: nil, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CONFIG", tt.env)
			if got := configPath(tt.args); got != tt.want {
				t.Fatalf("configPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOverride(t *testing.T) {
	dst := &Flags{Transport: "old", Address: "old"}
	src := &Flags{
		Transport:     "new",
		Address:       "new-addr",
		DSN:           "new-dsn",
		AppMode:       "new-mode",
		LogDir:        "new-log",
		JWTSecret:     "new-jwt",
		RefreshSecret: "new-refresh",
	}
	override(dst, src)
	if dst.Transport != "new" || dst.Address != "new-addr" || dst.DSN != "new-dsn" ||
		dst.AppMode != "new-mode" || dst.LogDir != "new-log" ||
		dst.JWTSecret != "new-jwt" || dst.RefreshSecret != "new-refresh" {
		t.Fatalf("override() did not apply all fields: %+v", dst)
	}

	dst2 := &Flags{Transport: "keep"}
	override(dst2, &Flags{})
	if dst2.Transport != "keep" {
		t.Fatalf("override() with empty src changed value: %+v", dst2)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"CONFIG", "TRANSPORT", "ADDRESS", "DATABASE_DSN", "APP_MODE", "LOG_DIR", "JWT_SECRET", "REFRESH_SECRET"} {
		t.Setenv(k, "")
	}
}
