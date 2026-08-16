package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Auth      AuthConfig      `yaml:"auth"`
	Discovery DiscoveryConfig `yaml:"discovery"`
	Monitor   MonitorConfig   `yaml:"monitor"`
	CORS      CORSConfig      `yaml:"cors"`
	Logging   LoggingConfig   `yaml:"logging"`
}

type ServerConfig struct {
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
	ExternalURL    string        `yaml:"external_url"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	TrustedProxies []string      `yaml:"trusted_proxies"`
	LANSubnets     []string      `yaml:"lan_subnets"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type AuthConfig struct {
	JWTSecret          string        `yaml:"jwt_secret"`
	JWTExpiry          time.Duration `yaml:"jwt_expiry"`
	SessionExpiry      time.Duration `yaml:"session_expiry"`
	BCryptCost         int           `yaml:"bcrypt_cost"`
	MFARequired        bool          `yaml:"mfa_required"`
	RateLimitPerMin    int           `yaml:"rate_limit_per_min"`
	APIRateLimitPerMin int           `yaml:"api_rate_limit_per_min"`
	AllowRegistration  bool          `yaml:"allow_registration"`
	InviteOnly         bool          `yaml:"invite_only"`
	SessionCookieName  string        `yaml:"session_cookie_name"`
}

type DiscoveryConfig struct {
	DockerSocket    string        `yaml:"docker_socket"`
	ScanInterval    time.Duration `yaml:"scan_interval"`
	DefaultNetwork  string        `yaml:"default_network"`
	LabelPrefix     string        `yaml:"label_prefix"`
	ProxmoxEndpoint string        `yaml:"proxmox_endpoint"`
	ProxmoxTokenID  string        `yaml:"proxmox_token_id"`
	ProxmoxSecret   string        `yaml:"proxmox_secret"`
}

type MonitorConfig struct {
	CheckInterval    time.Duration `yaml:"check_interval"`
	Timeout          time.Duration `yaml:"timeout"`
	RetryCount       int           `yaml:"retry_count"`
	StatusPagePublic bool          `yaml:"status_page_public"`
}

type CORSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:           "0.0.0.0",
			Port:           8081,
			ExternalURL:    "https://4rch3.io",
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   30 * time.Second,
			TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
			LANSubnets:     []string{"192.168.0.0/16", "10.0.0.0/8"},
		},
		Database: DatabaseConfig{
			Path: "data/portal.db",
		},
		Auth: AuthConfig{
			JWTSecret:          "MUST-BE-SET-VIA-PORTAL_JWT_SECRET-ENV-VAR",
			JWTExpiry:          24 * time.Hour,
			SessionExpiry:      30 * 24 * time.Hour,
			BCryptCost:         12,
			RateLimitPerMin:    10,
			APIRateLimitPerMin: 60,
			AllowRegistration:  false,
			InviteOnly:         true,
			SessionCookieName:  "4rch3_session",
		},
		Discovery: DiscoveryConfig{
			DockerSocket:   "unix:///var/run/docker.sock",
			ScanInterval:   30 * time.Second,
			DefaultNetwork: "devnet",
			LabelPrefix:    "4rch3.io",
		},
		Monitor: MonitorConfig{
			CheckInterval:    30 * time.Second,
			Timeout:          5 * time.Second,
			RetryCount:       2,
			StatusPagePublic: false,
		},
		CORS: CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"https://4rch3.io", "http://homelab.local"},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret must be set")
	}
	if c.Auth.JWTExpiry <= 0 {
		return fmt.Errorf("auth.jwt_expiry must be positive, got %v", c.Auth.JWTExpiry)
	}
	if c.Auth.SessionExpiry <= 0 {
		return fmt.Errorf("auth.session_expiry must be positive, got %v", c.Auth.SessionExpiry)
	}
	if c.Auth.BCryptCost < 4 || c.Auth.BCryptCost > 31 {
		return fmt.Errorf("auth.bcrypt_cost must be between 4 and 31, got %d", c.Auth.BCryptCost)
	}
	if c.Auth.RateLimitPerMin < 1 {
		return fmt.Errorf("auth.rate_limit_per_min must be at least 1, got %d", c.Auth.RateLimitPerMin)
	}
	if c.Auth.APIRateLimitPerMin < 1 {
		return fmt.Errorf("auth.api_rate_limit_per_min must be at least 1, got %d", c.Auth.APIRateLimitPerMin)
	}
	if c.Monitor.CheckInterval <= 0 {
		return fmt.Errorf("monitor.check_interval must be positive, got %v", c.Monitor.CheckInterval)
	}
	if c.Monitor.Timeout <= 0 {
		return fmt.Errorf("monitor.timeout must be positive, got %v", c.Monitor.Timeout)
	}
	if c.Discovery.ScanInterval <= 0 {
		return fmt.Errorf("discovery.scan_interval must be positive, got %v", c.Discovery.ScanInterval)
	}
	return nil
}
