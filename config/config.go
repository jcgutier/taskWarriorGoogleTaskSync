package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	GoogleCredentialsPath       string `json:"googleCredentialsPath"`
	GoogleTokenPath             string `json:"googleTokenPath"`
	GoogleTaskListFilter        string `json:"googleTaskListFilter"`
	SyncIntervalSeconds         int    `json:"syncIntervalSeconds"`
	DryRun                      bool   `json:"dryRun"`
	MetricsListenAddress        string `json:"metricsListenAddress"`
	BackoffMaxAttempts          int    `json:"backoffMaxAttempts"`
	BackoffInitialDelaySeconds  int    `json:"backoffInitialDelaySeconds"`
}

func LoadConfig() (*Config, error) {
	cfg := defaultConfig()
	configPath := "config/config.json"
	if envPath, ok := os.LookupEnv("CONFIG_FILE_PATH"); ok && envPath != "" {
		configPath = envPath
	}

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
		}
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		GoogleCredentialsPath:      "",
		GoogleTokenPath:            "token.json",
		GoogleTaskListFilter:       "Mis tareas",
		SyncIntervalSeconds:        300,
		DryRun:                     false,
		MetricsListenAddress:       ":9090",
		BackoffMaxAttempts:         5,
		BackoffInitialDelaySeconds: 1,
	}
}

func applyEnvOverrides(cfg *Config) {
	if val, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); ok && val != "" {
		cfg.GoogleCredentialsPath = val
	}
	if val, ok := os.LookupEnv("GOOGLE_TASKS_TOKEN_PATH"); ok && val != "" {
		cfg.GoogleTokenPath = val
	}
	if val, ok := os.LookupEnv("GOOGLE_TASK_LIST_FILTER"); ok && val != "" {
		cfg.GoogleTaskListFilter = val
	}
	if val, ok := os.LookupEnv("SYNC_INTERVAL_SECONDS"); ok && val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			cfg.SyncIntervalSeconds = parsed
		}
	}
	if val, ok := os.LookupEnv("DRY_RUN"); ok && val != "" {
		if parsed, err := strconv.ParseBool(strings.TrimSpace(val)); err == nil {
			cfg.DryRun = parsed
		}
	}
	if val, ok := os.LookupEnv("METRICS_LISTEN_ADDRESS"); ok && val != "" {
		cfg.MetricsListenAddress = val
	}
	if val, ok := os.LookupEnv("BACKOFF_MAX_ATTEMPTS"); ok && val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			cfg.BackoffMaxAttempts = parsed
		}
	}
	if val, ok := os.LookupEnv("BACKOFF_INITIAL_DELAY_SECONDS"); ok && val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			cfg.BackoffInitialDelaySeconds = parsed
		}
	}
}
