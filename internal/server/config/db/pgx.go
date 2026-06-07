// Package db contains database connection configuration.
package db

// Config stores database DSN and retry settings.
type Config struct {
	DNS        string
	MaxRetries int
}

// MaxRetries is default retry limit for DB operations.
const MaxRetries int = 3

// NewCfg creates DB configuration from DSN string.
func NewCfg(dns string) *Config {
	return &Config{
		DNS:        dns,
		MaxRetries: MaxRetries,
	}
}
