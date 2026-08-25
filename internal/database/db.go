package database

import (
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/4rch3/portal/internal/models"
)

//go:embed schema.sql
var schemaFS embed.FS

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Migrate() error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}

	// migrations for new columns
	db.Exec(`ALTER TABLE services ADD COLUMN public_url TEXT DEFAULT ''`)
	db.Exec(`ALTER TABLE services ADD COLUMN scheme TEXT DEFAULT 'http'`)

	return nil
}

func (db *DB) CreateUser(u *models.User) error {
	_, err := db.Exec(
		`INSERT INTO users (username, email, password_hash, role, api_key)
		 VALUES (?, ?, ?, ?, ?)`,
		u.Username, u.Email, u.PasswordHash, u.Role, u.APIKey,
	)
	return err
}

func (db *DB) GetUserByID(id int64) (*models.User, error) {
	u := &models.User{}
	var lastLogin, lastIP, mfaSecret, apiKey, createdAt, updatedAt sql.NullString
	err := db.QueryRow(
		`SELECT id, username, email, password_hash, role, mfa_enabled, mfa_secret,
		        api_key, active, last_login, last_ip, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role,
		&u.MFAEnabled, &mfaSecret, &apiKey, &u.Active,
		&lastLogin, &lastIP, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	u.MFASecret = mfaSecret.String
	u.APIKey = apiKey.String
	if lastLogin.Valid {
		u.LastLogin, _ = time.Parse("2006-01-02 15:04:05", lastLogin.String)
	}
	if lastIP.Valid {
		u.LastIP = lastIP.String
	}
	if createdAt.Valid {
		u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
	}
	if updatedAt.Valid {
		u.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
	}
	return u, nil
}

func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	u := &models.User{}
	var lastLogin, lastIP, mfaSecret, apiKey, createdAt, updatedAt sql.NullString
	err := db.QueryRow(
		`SELECT id, username, email, password_hash, role, mfa_enabled, mfa_secret,
		        api_key, active, last_login, last_ip, created_at, updated_at
		 FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role,
		&u.MFAEnabled, &mfaSecret, &apiKey, &u.Active,
		&lastLogin, &lastIP, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	u.MFASecret = mfaSecret.String
	u.APIKey = apiKey.String
	if lastLogin.Valid {
		u.LastLogin, _ = time.Parse("2006-01-02 15:04:05", lastLogin.String)
	}
	if lastIP.Valid {
		u.LastIP = lastIP.String
	}
	if createdAt.Valid {
		u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
	}
	if updatedAt.Valid {
		u.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
	}
	return u, nil
}

func (db *DB) GetUserByAPIKey(key string) (*models.User, error) {
	u := &models.User{}
	var lastLogin, lastIP, mfaSecret, apiKey, createdAt, updatedAt sql.NullString
	err := db.QueryRow(
		`SELECT id, username, email, password_hash, role, mfa_enabled, mfa_secret,
		        api_key, active, last_login, last_ip, created_at, updated_at
		 FROM users WHERE api_key = ? AND active = 1`, key,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role,
		&u.MFAEnabled, &mfaSecret, &apiKey, &u.Active,
		&lastLogin, &lastIP, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	u.MFASecret = mfaSecret.String
	u.APIKey = apiKey.String
	if lastLogin.Valid {
		u.LastLogin, _ = time.Parse("2006-01-02 15:04:05", lastLogin.String)
	}
	if lastIP.Valid {
		u.LastIP = lastIP.String
	}
	if createdAt.Valid {
		u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
	}
	if updatedAt.Valid {
		u.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
	}
	return u, nil
}

func (db *DB) ListUsers() ([]*models.User, error) {
	rows, err := db.Query(
		`SELECT id, username, email, role, active, last_login, last_ip, created_at, updated_at
		 FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		var lastLogin, lastIP, createdAt, updatedAt sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Active,
			&lastLogin, &lastIP, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if lastLogin.Valid {
			u.LastLogin, _ = time.Parse("2006-01-02 15:04:05", lastLogin.String)
		}
		if lastIP.Valid {
			u.LastIP = lastIP.String
		}
		if createdAt.Valid {
			u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
		}
		if updatedAt.Valid {
			u.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
		}
		users = append(users, u)
	}
	return users, nil
}

func (db *DB) UpdateUserLogin(id int64, ip string) error {
	_, err := db.Exec(
		`UPDATE users SET last_login = datetime('now'), last_ip = ? WHERE id = ?`,
		ip, id,
	)
	return err
}

func (db *DB) UpdateUserRole(id int64, role models.Role) error {
	_, err := db.Exec(`UPDATE users SET role = ?, updated_at = datetime('now') WHERE id = ?`, role, id)
	return err
}

func (db *DB) UpdatePassword(id int64, hash string) error {
	_, err := db.Exec(`UPDATE users SET password_hash = ?, updated_at = datetime('now') WHERE id = ?`, hash, id)
	return err
}

func (db *DB) DeleteUser(id int64) error {
	_, err := db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func (db *DB) CreateSession(s *models.Session) error {
	_, err := db.Exec(
		`INSERT INTO sessions (id, user_id, token, expires_at, ip, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.Token, s.ExpiresAt.Format("2006-01-02 15:04:05"), s.IP, s.UserAgent,
	)
	return err
}

func (db *DB) GetSession(token string) (*models.Session, error) {
	s := &models.Session{}
	var expiresAt, createdAt string
	err := db.QueryRow(
		`SELECT id, user_id, token, expires_at, created_at, ip, user_agent
		 FROM sessions WHERE token = ?`, token,
	).Scan(&s.ID, &s.UserID, &s.Token, &expiresAt, &createdAt, &s.IP, &s.UserAgent)
	if err != nil {
		return nil, err
	}
	s.ExpiresAt, _ = time.Parse("2006-01-02 15:04:05", expiresAt)
	s.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return s, nil
}

func (db *DB) DeleteSession(token string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func (db *DB) DeleteUserSessions(userID int64) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func (db *DB) DeleteExpiredSessions() (int64, error) {
	res, err := db.Exec(`DELETE FROM sessions WHERE expires_at <= datetime('now')`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DB) CreateService(svc *models.Service) (int64, error) {
	res, err := db.Exec(
	 `INSERT INTO services (name, display_name, description, url, public_url, internal_url, scheme, icon,
		 category, group_name, health_path, type, docker_id, docker_name, enabled, public)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET
		 display_name=excluded.display_name, description=excluded.description, url=excluded.url,
		 public_url=excluded.public_url, internal_url=excluded.internal_url, scheme=excluded.scheme,
		 icon=excluded.icon, category=excluded.category,
		 group_name=excluded.group_name, health_path=excluded.health_path,
		 docker_id=excluded.docker_id, docker_name=excluded.docker_name,
		 updated_at=datetime('now')`,
		svc.Name, svc.DisplayName, svc.Description, svc.URL, svc.PublicURL, svc.InternalURL, svc.Scheme, svc.Icon,
		svc.Category, svc.Group, svc.HealthPath, svc.Type, svc.DockerID, svc.DockerName,
		svc.Enabled, svc.Public,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) ListServices() ([]*models.Service, error) {
	rows, err := db.Query(
		`SELECT id, name, display_name, description, url, public_url, internal_url, scheme, icon,
		 category, group_name, health_path, type, docker_id, docker_name,
		 enabled, public, status, latency, last_checked, created_at, updated_at
		 FROM services ORDER BY category, display_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []*models.Service
	for rows.Next() {
		s := &models.Service{}
		var lastChecked, createdAt, updatedAt sql.NullString
		if err := rows.Scan(&s.ID, &s.Name, &s.DisplayName, &s.Description,
			&s.URL, &s.PublicURL, &s.InternalURL, &s.Scheme, &s.Icon, &s.Category, &s.Group,
			&s.HealthPath, &s.Type, &s.DockerID, &s.DockerName,
			&s.Enabled, &s.Public, &s.Status, &s.Latency,
			&lastChecked, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if lastChecked.Valid {
			s.LastChecked, _ = time.Parse("2006-01-02 15:04:05", lastChecked.String)
		}
		if createdAt.Valid {
			s.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
		}
		if updatedAt.Valid {
			s.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
		}
		services = append(services, s)
	}
	return services, nil
}

func (db *DB) GetService(id int64) (*models.Service, error) {
	s := &models.Service{}
	var lastChecked, createdAt, updatedAt sql.NullString
	err := db.QueryRow(
		`SELECT id, name, display_name, description, url, public_url, internal_url, scheme, icon,
		 category, group_name, health_path, type, docker_id, docker_name,
		 enabled, public, status, latency, last_checked, created_at, updated_at
		 FROM services WHERE id = ?`, id,
	).Scan(&s.ID, &s.Name, &s.DisplayName, &s.Description,
		&s.URL, &s.PublicURL, &s.InternalURL, &s.Scheme, &s.Icon, &s.Category, &s.Group,
		&s.HealthPath, &s.Type, &s.DockerID, &s.DockerName,
		&s.Enabled, &s.Public, &s.Status, &s.Latency,
		&lastChecked, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if lastChecked.Valid {
		s.LastChecked, _ = time.Parse("2006-01-02 15:04:05", lastChecked.String)
	}
	if createdAt.Valid {
		s.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
	}
	if updatedAt.Valid {
		s.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
	}
	return s, nil
}

func (db *DB) UpdateServiceStatus(id int64, status string, latency int64) error {
	_, err := db.Exec(
		`UPDATE services SET status = ?, latency = ?, last_checked = datetime('now'),
		 updated_at = datetime('now') WHERE id = ?`,
		status, latency, id,
	)
	return err
}

func (db *DB) DeleteService(id int64) error {
	_, err := db.Exec(`DELETE FROM services WHERE id = ?`, id)
	return err
}

func (db *DB) RecordHealthCheck(h *models.HealthCheck) error {
	_, err := db.Exec(
		`INSERT INTO health_checks (service_id, status, latency, code, error)
		 VALUES (?, ?, ?, ?, ?)`,
		h.ServiceID, h.Status, h.Latency, h.Code, h.Error,
	)
	return err
}

func (db *DB) GetRecentHealthChecks(serviceID int64, limit int) ([]*models.HealthCheck, error) {
	rows, err := db.Query(
		`SELECT id, service_id, status, latency, code, error, checked_at
		 FROM health_checks WHERE service_id = ?
		 ORDER BY checked_at DESC LIMIT ?`, serviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []*models.HealthCheck
	for rows.Next() {
		h := &models.HealthCheck{}
		var checkedAt string
		if err := rows.Scan(&h.ID, &h.ServiceID, &h.Status, &h.Latency,
			&h.Code, &h.Error, &checkedAt); err != nil {
			return nil, err
		}
		h.CheckedAt, _ = time.Parse("2006-01-02 15:04:05", checkedAt)
		checks = append(checks, h)
	}
	return checks, nil
}

func (db *DB) LogAudit(a *models.AuditLog) error {
	_, err := db.Exec(
		`INSERT INTO audit_logs (user_id, action, resource, detail, ip, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		a.UserID, a.Action, a.Resource, a.Detail, a.IP, a.UserAgent,
	)
	return err
}

func (db *DB) GetAuditLogs(limit int) ([]*models.AuditLog, error) {
	rows, err := db.Query(
		`SELECT id, user_id, action, resource, detail, ip, user_agent, created_at
		 FROM audit_logs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		l := &models.AuditLog{}
		var createdAt string
		if err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.Resource,
			&l.Detail, &l.IP, &l.UserAgent, &createdAt); err != nil {
			return nil, err
		}
		l.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		logs = append(logs, l)
	}
	return logs, nil
}

func (db *DB) GetDeployEvents(limit int) ([]*models.AuditLog, error) {
	rows, err := db.Query(
		`SELECT id, user_id, action, resource, detail, ip, user_agent, created_at
		 FROM audit_logs WHERE action = 'deploy' ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		l := &models.AuditLog{}
		var createdAt string
		if err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.Resource,
			&l.Detail, &l.IP, &l.UserAgent, &createdAt); err != nil {
			return nil, err
		}
		l.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		logs = append(logs, l)
	}
	return logs, nil
}

func (db *DB) CreateInviteCode(code *models.InviteCode) error {
	_, err := db.Exec(
		`INSERT INTO invite_codes (code, created_by, max_uses, expires_at)
		 VALUES (?, ?, ?, ?)`,
		code.Code, code.CreatedBy, code.MaxUses, code.ExpiresAt.Format("2006-01-02 15:04:05"),
	)
	return err
}

func (db *DB) UseInviteCode(code string) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var id, maxUses, useCount int
	var expiresAt string
	err = tx.QueryRow(
		`SELECT id, max_uses, use_count, expires_at FROM invite_codes WHERE code = ?`, code,
	).Scan(&id, &maxUses, &useCount, &expiresAt)
	if err != nil {
		return false, err
	}

	expires, _ := time.Parse("2006-01-02 15:04:05", expiresAt)
	if time.Now().After(expires) {
		return false, fmt.Errorf("invite code expired")
	}

	if useCount >= maxUses {
		return false, fmt.Errorf("invite code max uses reached")
	}

	_, err = tx.Exec(`UPDATE invite_codes SET use_count = use_count + 1 WHERE id = ?`, id)
	if err != nil {
		return false, err
	}

	return true, tx.Commit()
}
