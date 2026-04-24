package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                 string
	AppBaseURL             string
	DatabaseURL            string
	SupabaseURL            string
	SupabaseAnonKey        string
	SupabaseServiceRoleKey string
	SupabaseJWTSecret      string
	RawCaptureBucket       string
	LogLevel               string
	RawInlineMaxBytes      int
	ServerReadTimeout      time.Duration
	ServerWriteTimeout     time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		AppBaseURL:             getEnv("APP_BASE_URL", "http://localhost:3000"),
		DatabaseURL:            strings.TrimSpace(os.Getenv("DATABASE_URL")),
		SupabaseURL:            strings.TrimSpace(os.Getenv("SUPABASE_URL")),
		SupabaseAnonKey:        strings.TrimSpace(os.Getenv("SUPABASE_ANON_KEY")),
		SupabaseServiceRoleKey: strings.TrimSpace(os.Getenv("SUPABASE_SERVICE_ROLE_KEY")),
		SupabaseJWTSecret:      getEnv("SUPABASE_JWT_SECRET", ""),
		RawCaptureBucket:       getEnv("RAW_CAPTURE_BUCKET", ""),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("missing required env: DATABASE_URL")
	}
	if cfg.SupabaseURL == "" {
		return Config{}, fmt.Errorf("missing required env: SUPABASE_URL")
	}
	if cfg.SupabaseAnonKey == "" {
		return Config{}, fmt.Errorf("missing required env: SUPABASE_ANON_KEY")
	}
	if cfg.SupabaseServiceRoleKey == "" {
		return Config{}, fmt.Errorf("missing required env: SUPABASE_SERVICE_ROLE_KEY")
	}

	var err error
	if cfg.RawInlineMaxBytes, err = getEnvInt("RAW_CAPTURE_INLINE_MAX_BYTES", 65536); err != nil {
		return Config{}, fmt.Errorf("invalid RAW_CAPTURE_INLINE_MAX_BYTES: %w", err)
	}
	if cfg.ServerReadTimeout, err = getEnvSeconds("SERVER_READ_TIMEOUT", 15); err != nil {
		return Config{}, fmt.Errorf("invalid SERVER_READ_TIMEOUT: %w", err)
	}
	if cfg.ServerWriteTimeout, err = getEnvSeconds("SERVER_WRITE_TIMEOUT", 15); err != nil {
		return Config{}, fmt.Errorf("invalid SERVER_WRITE_TIMEOUT: %w", err)
	}

	return cfg, nil
}

func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	return cfg
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func getEnvSeconds(key string, fallback int) (time.Duration, error) {
	value, err := getEnvInt(key, fallback)
	if err != nil {
		return 0, err
	}
	return time.Duration(value) * time.Second, nil
}
