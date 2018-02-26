package config

import (
	"time"

	"github.com/ONSdigital/go-ns/log"
	"github.com/kelseyhightower/envconfig"
)

// Config is the configuration for this service
type Config struct {
	BindAddr           string        `envconfig:"BIND_ADDR"`
	CORSAllowedOrigins string        `envconfig:"CORS_ALLOWED_ORIGINS"`
	ShutdownTimeout    time.Duration `envconfig:"SHUTDOWN_TIMEOUT"`
}

var cfg *Config

// Get configures the application and returns the configuration
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		BindAddr:           ":23500",
		CORSAllowedOrigins: "*",
		ShutdownTimeout:    5 * time.Second,
	}

	return cfg, envconfig.Process("", cfg)
}

// Log writes all config properties to log.Debug
func (cfg *Config) Log() {
	log.Debug("Configuration", log.Data{
		"BindAddr":           cfg.BindAddr,
		"CORSAllowedOrigins": cfg.CORSAllowedOrigins,
		"ShutdownTimeout":    cfg.ShutdownTimeout,
	})

}
