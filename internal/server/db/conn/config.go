package conn

// Config stores database DSN and retry settings.
type Config struct {
	DSN        string
	MaxRetries int
}

// MaxRetries is default retry limit for DB operations.
const MaxRetries int = 3

// NewCfg creates DB configuration from DSN string.
func NewCfg(dsn string) *Config {
	return &Config{
		DSN:        dsn,
		MaxRetries: MaxRetries,
	}
}
