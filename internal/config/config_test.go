package config

import (
	"os"
	"testing"
)

func TestLoadRequiresMandatoryEnv(t *testing.T) {
	keys := []string{"DATABASE_URL", "SUPABASE_URL", "SUPABASE_ANON_KEY", "SUPABASE_SERVICE_ROLE_KEY"}
	for _, key := range keys {
		_ = os.Unsetenv(key)
	}
	if _, err := Load(); err == nil {
		t.Fatalf("expected error when required env vars are missing")
	}
}

func TestLoadParsesTimeouts(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("SUPABASE_URL", "https://example.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "anon")
	t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "service")
	t.Setenv("SERVER_READ_TIMEOUT", "12")
	t.Setenv("SERVER_WRITE_TIMEOUT", "20")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ServerReadTimeout.Seconds() != 12 || cfg.ServerWriteTimeout.Seconds() != 20 {
		t.Fatalf("unexpected timeout parsing: %#v", cfg)
	}
}
