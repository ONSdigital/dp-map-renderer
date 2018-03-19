package config

import (
	"time"

	"strings"

	"github.com/ONSdigital/go-ns/log"
	"github.com/kelseyhightower/envconfig"
)

// Config is the configuration for this service
type Config struct {
	BindAddr           string        `envconfig:"BIND_ADDR"`
	CORSAllowedOrigins string        `envconfig:"CORS_ALLOWED_ORIGINS"`
	ShutdownTimeout    time.Duration `envconfig:"SHUTDOWN_TIMEOUT"`
	SVG2PNGExecutable  string        `envconfig:"SVG_2_PNG_EXECUTABLE"`
	SVG2PNGArgLine     string        `envconfig:"SVG_2_PNG_ARG_LINE"`
	SVG2PNGArguments   []string
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
		SVG2PNGExecutable:  "rsvg-convert",
		SVG2PNGArgLine:     "<SVG>|-o|<PNG>",
	}

	cfg.SVG2PNGArguments = strings.Split(cfg.SVG2PNGArgLine, "|")

	return cfg, envconfig.Process("", cfg)
}

// Log writes all config properties to log.Debug
func (cfg *Config) Log() {
	log.Debug("Configuration", log.Data{
		"BindAddr":           cfg.BindAddr,
		"CORSAllowedOrigins": cfg.CORSAllowedOrigins,
		"ShutdownTimeout":    cfg.ShutdownTimeout,
		"SVG2PNGExecutable":  cfg.SVG2PNGExecutable,
		"SVG2PNGArgLine":     cfg.SVG2PNGArgLine,
		"SVG2PNGArguments":   cfg.SVG2PNGArguments,
	})

}
