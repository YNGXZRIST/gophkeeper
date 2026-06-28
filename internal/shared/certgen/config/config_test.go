package config

import (
	"reflect"
	"testing"
)

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.CertPath != DefaultCertPath {
		t.Errorf("CertPath = %q, want %q", cfg.CertPath, DefaultCertPath)
	}
	if cfg.KeyPath != DefaultKeyPath {
		t.Errorf("KeyPath = %q, want %q", cfg.KeyPath, DefaultKeyPath)
	}
	if cfg.EmbedPath != DefaultEmbedPath {
		t.Errorf("EmbedPath = %q, want %q", cfg.EmbedPath, DefaultEmbedPath)
	}
	if cfg.Force {
		t.Errorf("Force = true, want false by default")
	}
	wantHosts := []string{"localhost", "127.0.0.1", "::1"}
	if !reflect.DeepEqual(cfg.Hosts, wantHosts) {
		t.Errorf("Hosts = %v, want %v", cfg.Hosts, wantHosts)
	}
}

func TestParseOverrides(t *testing.T) {
	args := []string{
		"-cert", "/tmp/c.crt",
		"-key", "/tmp/c.key",
		"-embed", "/tmp/e.crt",
		"-host", "a.example,192.168.0.1",
		"-force",
	}
	cfg, err := Parse(args)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.CertPath != "/tmp/c.crt" {
		t.Errorf("CertPath = %q", cfg.CertPath)
	}
	if cfg.KeyPath != "/tmp/c.key" {
		t.Errorf("KeyPath = %q", cfg.KeyPath)
	}
	if cfg.EmbedPath != "/tmp/e.crt" {
		t.Errorf("EmbedPath = %q", cfg.EmbedPath)
	}
	if !cfg.Force {
		t.Errorf("Force = false, want true")
	}
	wantHosts := []string{"a.example", "192.168.0.1"}
	if !reflect.DeepEqual(cfg.Hosts, wantHosts) {
		t.Errorf("Hosts = %v, want %v", cfg.Hosts, wantHosts)
	}
}

func TestParseHostTrimAndSplit(t *testing.T) {
	cfg, err := Parse([]string{"-host", "a, b ,,c"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	wantHosts := []string{"a", "b", "c"}
	if !reflect.DeepEqual(cfg.Hosts, wantHosts) {
		t.Errorf("Hosts = %v, want %v", cfg.Hosts, wantHosts)
	}
}

func TestParseUnknownFlag(t *testing.T) {
	if _, err := Parse([]string{"-nope"}); err == nil {
		t.Fatalf("Parse: expected error for unknown flag")
	}
}
