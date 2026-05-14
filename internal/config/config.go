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
	Host        string        `yaml:"host"`
	Port        int           `yaml:"port"`
	ExternalURL string        `yaml:"external_url"`
	ReadTimeout time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	TrustedProxies []string  `yaml:"trusted_proxies"`
	LANSubnets   []string    `yaml:"lan_subnets"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type AuthConfig struct {
	JWTSecret         string        `yaml:"jwt_secret"`
	JWTExpiry         time.Duration `yaml:"jwt_expiry"`
	SessionExpiry     time.Duration `yaml:"session_expiry"`
	BCryptCost        int           `yaml:"bcrypt_cost"`
	MFARequired       bool          `yaml:"mfa_required"`
	RateLimitPerMin   int           `yaml:"rate_limit_per_min"`
	APIRateLimitPerMin int          `yaml:"api_rate_limit_per_min"`
	AllowRegistration bool          `yaml:"allow_registration"`
	InviteOnly        bool          `yaml:"invite_only"`
	SessionCookieName string        `yaml:"session_cookie_name"`
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
	CheckInterval   time.Duration `yaml:"check_interval"`
	Timeout         time.Duration `yaml:"timeout"`
	RetryCount      int           `yaml:"retry_count"`
	StatusPagePublic bool         `yaml:"status_page_public"`
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
			Host:          "0.0.0.0",
			Port:          8081,
			ExternalURL:   "https://4rch3.io",
			ReadTimeout:   10 * time.Second,
			WriteTimeout:  30 * time.Second,
			TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
			LANSubnets:    []string{"192.168.0.0/16", "10.0.0.0/8"},
		},
		Database: DatabaseConfig{
			Path: "data/portal.db",
		},
		Auth: AuthConfig{
			JWTSecret:          "change-me-in-production",
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
			Enabled: true,
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
