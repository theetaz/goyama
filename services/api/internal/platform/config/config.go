package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config is loaded from environment variables via github.com/caarlos0/env.
type Config struct {
	Port         int      `env:"PORT" envDefault:"8080"`
	Env          string   `env:"APP_ENV" envDefault:"dev"`
	LogLevel     string   `env:"LOG_LEVEL" envDefault:"info"`
	CorsOrigins  []string `env:"CORS_ORIGINS" envDefault:"http://localhost:5173,http://localhost:5174" envSeparator:","`
	CorpusPath   string   `env:"CORPUS_PATH" envDefault:"../../corpus/releases/v0.0.1-drafts"`
	DatabaseURL  string   `env:"DATABASE_URL"` // optional for dev; JSONL corpus used when empty
	ReadTimeout  int      `env:"HTTP_READ_TIMEOUT_SEC" envDefault:"10"`
	WriteTimeout int      `env:"HTTP_WRITE_TIMEOUT_SEC" envDefault:"15"`
}

// Load reads config from the environment and returns a populated struct.
func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("parse env: %w", err)
	}
	return cfg, nil
}
