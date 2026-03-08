package config

import (
	"fmt"
	"log/slog"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppEnv            string `env:"APP_ENV" envDefault:"development"`
	Port              string `env:"PORT" envDefault:"8090"`
	DatabaseURL       string `env:"DATABASE_URL" envDefault:"postgres://luminor:luminor@localhost:5442/luminor?sslmode=disable"`
	SessionKey        string `env:"SESSION_KEY" envDefault:"change-me-in-production-32bytes!"`
	CSRFKey           string `env:"CSRF_KEY" envDefault:"change-me-in-production-32bytes!"`
	BaseURL           string `env:"BASE_URL" envDefault:"http://localhost:8090"`
	RAGDatabaseURL    string `env:"RAG_DATABASE_URL" envDefault:"postgres://luminor:luminor@localhost:5443/luminor_rag?sslmode=disable"`
	LocalInferenceURL string `env:"LOCAL_INFERENCE_URL" envDefault:"http://local-inference:11434"`
	EmbedModel        string `env:"EMBED_MODEL" envDefault:"nomic-embed-text"`
	ChatModel         string `env:"CHAT_MODEL" envDefault:"llama3.1:8b"`
}

func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func (c Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func Load() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	slog.Info("config loaded",
		"app_env", cfg.AppEnv,
		"port", cfg.Port,
		"base_url", cfg.BaseURL,
	)

	return cfg, nil
}
