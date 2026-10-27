package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	GoroutinePoolSize     int
	MaxWaitingConnections int
	BufferSize            int
	BindedPort            string
	MetricsAddr           string
	LogLevel              string
}

// Default returns the built-in configuration.
func Default() Config {
	return Config{
		GoroutinePoolSize:     100000,
		MaxWaitingConnections: 10000,
		BufferSize:            4 * 1024, // 4 KB
		BindedPort:            ":6379",
		MetricsAddr:           ":2112",
		LogLevel:              "info",
	}
}

// Load builds the configuration with precedence defaults < environment variables < flags.
// args are the command-line arguments without the program name.
func Load(args []string) (Config, error) {
	cfg := Default()

	if err := applyEnv(&cfg); err != nil {
		return Config{}, err
	}

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.IntVar(&cfg.GoroutinePoolSize, "pool-size", cfg.GoroutinePoolSize, "number of worker goroutines (REDIS_POOL_SIZE)")
	fs.IntVar(&cfg.MaxWaitingConnections, "max-waiting-conns", cfg.MaxWaitingConnections, "connections queued before Accept blocks (REDIS_MAX_WAITING_CONNS)")
	fs.IntVar(&cfg.BufferSize, "buffer-size", cfg.BufferSize, "per-connection read buffer in bytes (REDIS_BUFFER_SIZE)")
	fs.StringVar(&cfg.BindedPort, "port", cfg.BindedPort, "listen address, e.g. :6379 (REDIS_PORT)")
	fs.StringVar(&cfg.MetricsAddr, "metrics-addr", cfg.MetricsAddr, "address for /metrics and /healthz (REDIS_METRICS_ADDR)")
	fs.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "debug, info, warn or error (REDIS_LOG_LEVEL)")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func applyEnv(cfg *Config) error {
	ints := []struct {
		key string
		dst *int
	}{
		{"REDIS_POOL_SIZE", &cfg.GoroutinePoolSize},
		{"REDIS_MAX_WAITING_CONNS", &cfg.MaxWaitingConnections},
		{"REDIS_BUFFER_SIZE", &cfg.BufferSize},
	}

	for _, e := range ints {
		v, ok := os.LookupEnv(e.key)
		if !ok || v == "" {
			continue
		}

		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid value for %s=%q: %w", e.key, v, err)
		}

		*e.dst = n
	}

	strs := []struct {
		key string
		dst *string
	}{
		{"REDIS_PORT", &cfg.BindedPort},
		{"REDIS_METRICS_ADDR", &cfg.MetricsAddr},
		{"REDIS_LOG_LEVEL", &cfg.LogLevel},
	}

	for _, e := range strs {
		if v, ok := os.LookupEnv(e.key); ok && v != "" {
			*e.dst = v
		}
	}

	return nil
}
