// Package config carrega configuração de variáveis de ambiente.
// Falha cedo se algo crítico estiver faltando — preferível a defaults silenciosos.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Env             string
	Bind            string
	DBPath          string
	DataDir         string
	ThumbsDir       string
	LogsDir         string
	SecretsDir      string
	SambaStateDir   string
	AdminUsername   string
	AdminPassword   string
	JWTTTL          time.Duration
	RefreshTTL      time.Duration
	AllowedOrigin   string
	LogLevel        string
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:           getEnv("PINAS_ENV", "production"),
		Bind:          getEnv("PINAS_BIND", "0.0.0.0:8080"),
		DBPath:        getEnv("PINAS_DB_PATH", "/var/lib/pinas/db/pinas.db"),
		DataDir:       getEnv("PINAS_DATA_DIR", "/var/lib/pinas/data"),
		ThumbsDir:     getEnv("PINAS_THUMBS_DIR", "/var/lib/pinas/thumbs"),
		LogsDir:       getEnv("PINAS_LOGS_DIR", "/var/lib/pinas/logs"),
		SecretsDir:    getEnv("PINAS_SECRETS_DIR", "/var/lib/pinas/secrets"),
		SambaStateDir: getEnv("PINAS_SAMBA_STATE_DIR", "/var/lib/pinas/samba"),
		AdminUsername: getEnv("PINAS_ADMIN_USERNAME", "admin"),
		AdminPassword: os.Getenv("PINAS_ADMIN_PASSWORD"),
		AllowedOrigin: getEnv("PINAS_ALLOWED_ORIGIN", "https://pinas.local"),
		LogLevel:      getEnv("PINAS_LOG_LEVEL", "info"),
	}

	jwtTTL, err := time.ParseDuration(getEnv("PINAS_JWT_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("PINAS_JWT_TTL invalid: %w", err)
	}
	cfg.JWTTTL = jwtTTL

	refTTL, err := time.ParseDuration(getEnv("PINAS_REFRESH_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("PINAS_REFRESH_TTL invalid: %w", err)
	}
	cfg.RefreshTTL = refTTL

	if cfg.AdminPassword == "" {
		// Aceitamos vazio só fora de produção — em prod deve estar setado.
		if cfg.Env == "production" {
			return nil, fmt.Errorf("PINAS_ADMIN_PASSWORD obrigatório em produção")
		}
	}
	if len(cfg.AdminPassword) > 0 && len(cfg.AdminPassword) < 8 {
		return nil, fmt.Errorf("PINAS_ADMIN_PASSWORD muito curto (mínimo 8 caracteres)")
	}

	return cfg, nil
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Env, "production")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
