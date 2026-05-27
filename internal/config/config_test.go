package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Server.Port != 8081 {
		t.Errorf("expected port 8081, got %d", cfg.Server.Port)
	}
	if cfg.Auth.JWTExpiry != 24*time.Hour {
		t.Errorf("expected JWTExpiry 24h, got %v", cfg.Auth.JWTExpiry)
	}
	if !cfg.CORS.Enabled {
		t.Error("expected CORS enabled by default")
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
server:
  port: 9090
auth:
  jwt_secret: "test-secret"
cors:
  enabled: false
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Auth.JWTSecret != "test-secret" {
		t.Errorf("expected JWTSecret 'test-secret', got '%s'", cfg.Auth.JWTSecret)
	}
	if cfg.CORS.Enabled {
		t.Error("expected CORS disabled")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load should return default config for missing file: %v", err)
	}
	if cfg.Server.Port != 8081 {
		t.Errorf("expected default port 8081, got %d", cfg.Server.Port)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(": invalid yaml {"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestConfigValidateDefault(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret-that-is-long-enough-for-hmac"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected default config to be valid: %v", err)
	}
}

func TestConfigValidateBadPort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Port = 99999
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestConfigValidateDefaultJWTSecret(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.JWTSecret = "change-me-in-production"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for default JWT secret")
	}
}

func TestConfigValidateEmptyJWTSecret(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.JWTSecret = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty JWT secret")
	}
}

func TestConfigValidateBadBCryptCost(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.BCryptCost = 1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for low bcrypt cost")
	}
}

func TestConfigValidateNegativeTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Monitor.Timeout = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative timeout")
	}
}
