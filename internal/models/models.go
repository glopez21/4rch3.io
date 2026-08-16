package models

import "time"

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleUser     Role = "user"
	RoleReadOnly Role = "readonly"
	RoleService  Role = "service"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	MFAEnabled   bool      `json:"mfa_enabled"`
	MFASecret    string    `json:"-"`
	APIKey       string    `json:"api_key,omitempty"`
	Active       bool      `json:"active"`
	LastLogin    time.Time `json:"last_login"`
	LastIP       string    `json:"last_ip"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Session struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
}

type Service struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	InternalURL string    `json:"internal_url"`
	Icon        string    `json:"icon"`
	Category    string    `json:"category"`
	Group       string    `json:"group"`
	HealthPath  string    `json:"health_path"`
	Type        string    `json:"type"`
	DockerID    string    `json:"docker_id,omitempty"`
	DockerName  string    `json:"docker_name,omitempty"`
	Enabled     bool      `json:"enabled"`
	Public      bool      `json:"public"`
	Status      string    `json:"status"`
	Latency     int64     `json:"latency"`
	LastChecked time.Time `json:"last_checked"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type HealthCheck struct {
	ID        int64     `json:"id"`
	ServiceID int64     `json:"service_id"`
	Status    string    `json:"status"`
	Latency   int64     `json:"latency"`
	Code      int       `json:"code"`
	Error     string    `json:"error,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

type AuditLog struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Detail    string    `json:"detail"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

type InviteCode struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	CreatedBy int64     `json:"created_by"`
	MaxUses   int       `json:"max_uses"`
	UseCount  int       `json:"use_count"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
