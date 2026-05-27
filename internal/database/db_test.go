package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/4rch3/portal/internal/models"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	return db
}

func testUser() *models.User {
	return &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         "user",
		APIKey:       "4rch3_test-api-key",
		Active:       true,
	}
}

func TestOpenAndMigrate(t *testing.T) {
	db := openTestDB(t)
	if db == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestCreateAndGetUserByID(t *testing.T) {
	db := openTestDB(t)
	u := testUser()
	if err := db.CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	created, err := db.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}

	got, err := db.GetUserByID(created.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if got.Username != "testuser" {
		t.Errorf("expected 'testuser', got '%s'", got.Username)
	}
	if got.Email != "test@example.com" {
		t.Errorf("expected 'test@example.com', got '%s'", got.Email)
	}
	if got.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", got.Role)
	}
	if !got.Active {
		t.Error("expected user to be active")
	}
}

func TestGetUserByUsername(t *testing.T) {
	db := openTestDB(t)
	u := testUser()
	db.CreateUser(u)

	got, err := db.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	if got.Username != "testuser" {
		t.Errorf("expected 'testuser', got '%s'", got.Username)
	}

	_, err = db.GetUserByUsername("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestGetUserByAPIKey(t *testing.T) {
	db := openTestDB(t)
	u := testUser()
	if err := db.CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	got, err := db.GetUserByAPIKey("4rch3_test-api-key")
	if err != nil {
		t.Fatalf("GetUserByAPIKey failed: %v", err)
	}
	if got.Username != "testuser" {
		t.Errorf("expected 'testuser', got '%s'", got.Username)
	}

	_, err = db.GetUserByAPIKey("invalid")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestListUsers(t *testing.T) {
	db := openTestDB(t)
	users := []*models.User{
		{Username: "alice", Email: "alice@test.com", PasswordHash: "hash1", Role: "admin", APIKey: "key1", Active: true},
		{Username: "bob", Email: "bob@test.com", PasswordHash: "hash2", Role: "user", APIKey: "key2", Active: true},
	}
	for _, u := range users {
		if err := db.CreateUser(u); err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
	}

	listed, err := db.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(listed) != 2 {
		t.Errorf("expected 2 users, got %d", len(listed))
	}
}

func TestUpdateUserLogin(t *testing.T) {
	db := openTestDB(t)
	u := testUser()
	db.CreateUser(u)
	created, _ := db.GetUserByUsername("testuser")

	if err := db.UpdateUserLogin(created.ID, "10.0.0.1"); err != nil {
		t.Fatalf("UpdateUserLogin failed: %v", err)
	}

	got, _ := db.GetUserByID(created.ID)
	if got.LastIP != "10.0.0.1" {
		t.Errorf("expected IP '10.0.0.1', got '%s'", got.LastIP)
	}
}

func TestUpdateUserRole(t *testing.T) {
	db := openTestDB(t)
	u := testUser()
	db.CreateUser(u)
	created, _ := db.GetUserByUsername("testuser")

	if err := db.UpdateUserRole(created.ID, "admin"); err != nil {
		t.Fatalf("UpdateUserRole failed: %v", err)
	}

	got, _ := db.GetUserByID(created.ID)
	if got.Role != "admin" {
		t.Errorf("expected role 'admin', got '%s'", got.Role)
	}
}

func TestUpdatePassword(t *testing.T) {
	db := openTestDB(t)
	u := testUser()
	db.CreateUser(u)
	created, _ := db.GetUserByUsername("testuser")

	if err := db.UpdatePassword(created.ID, "$2a$10$newhash"); err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}

	got, _ := db.GetUserByID(created.ID)
	if got.PasswordHash != "$2a$10$newhash" {
		t.Errorf("expected new hash, got '%s'", got.PasswordHash)
	}
}

func TestDeleteUser(t *testing.T) {
	db := openTestDB(t)
	u := testUser()
	db.CreateUser(u)
	created, _ := db.GetUserByUsername("testuser")

	if err := db.DeleteUser(created.ID); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	_, err := db.GetUserByID(created.ID)
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

func createTestUser(t *testing.T, db *DB) *models.User {
	t.Helper()
	u := testUser()
	if err := db.CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	created, err := db.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	return created
}

func TestCreateAndGetSession(t *testing.T) {
	db := openTestDB(t)
	u := createTestUser(t, db)

	s := &models.Session{
		ID:        "sess-1",
		UserID:    u.ID,
		Token:     "session-token-abc",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IP:        "10.0.0.1",
		UserAgent: "test-agent",
	}
	if err := db.CreateSession(s); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	got, err := db.GetSession("session-token-abc")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if got.ID != "sess-1" {
		t.Errorf("expected ID 'sess-1', got '%s'", got.ID)
	}
	if got.UserID != u.ID {
		t.Errorf("expected UserID %d, got %d", u.ID, got.UserID)
	}
	if got.IP != "10.0.0.1" {
		t.Errorf("expected IP '10.0.0.1', got '%s'", got.IP)
	}
}

func TestDeleteSession(t *testing.T) {
	db := openTestDB(t)
	u := createTestUser(t, db)

	s := &models.Session{
		ID:        "sess-2",
		UserID:    u.ID,
		Token:     "token-to-delete",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	db.CreateSession(s)

	if err := db.DeleteSession("token-to-delete"); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	_, err := db.GetSession("token-to-delete")
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

func TestDeleteUserSessions(t *testing.T) {
	db := openTestDB(t)
	u := createTestUser(t, db)

	for i := 0; i < 3; i++ {
		s := &models.Session{
			ID:        fmt.Sprintf("sess-%d", i),
			UserID:    u.ID,
			Token:     fmt.Sprintf("token-%d", i),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		db.CreateSession(s)
	}

	if err := db.DeleteUserSessions(u.ID); err != nil {
		t.Fatalf("DeleteUserSessions failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := db.GetSession(fmt.Sprintf("token-%d", i))
		if err == nil {
			t.Errorf("expected session %d to be deleted", i)
		}
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	db := openTestDB(t)
	u := createTestUser(t, db)

	s1 := &models.Session{
		ID: "expired", UserID: u.ID, Token: "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	db.CreateSession(s1)

	s2 := &models.Session{
		ID: "valid", UserID: u.ID, Token: "valid-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	db.CreateSession(s2)

	n, err := db.DeleteExpiredSessions()
	if err != nil {
		t.Fatalf("DeleteExpiredSessions failed: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 deleted, got %d", n)
	}

	_, err = db.GetSession("expired-token")
	if err == nil {
		t.Fatal("expected expired session to be deleted")
	}

	_, err = db.GetSession("valid-token")
	if err != nil {
		t.Fatalf("expected valid session to remain: %v", err)
	}
}

func TestCreateAndListServices(t *testing.T) {
	db := openTestDB(t)

	svc := &models.Service{
		Name:        "test-svc",
		DisplayName: "Test Service",
		Description: "A test",
		URL:         "https://test.local",
		InternalURL: "http://test:8080",
		Icon:        "🔬",
		Category:    "monitoring",
		Group:       "observability",
		HealthPath:  "/health",
		Type:        "docker",
		DockerID:    "abc123",
		DockerName:  "test_container",
		Enabled:     true,
		Public:      false,
	}

	id, err := db.CreateService(svc)
	if err != nil {
		t.Fatalf("CreateService failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	services, err := db.ListServices()
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0].Name != "test-svc" {
		t.Errorf("expected 'test-svc', got '%s'", services[0].Name)
	}
}

func TestGetService(t *testing.T) {
	db := openTestDB(t)
	svc := &models.Service{
		Name: "get-test", DisplayName: "Get Test",
		URL: "https://get.test", Type: "external",
	}
	id, err := db.CreateService(svc)
	if err != nil {
		t.Fatalf("CreateService failed: %v", err)
	}

	got, err := db.GetService(id)
	if err != nil {
		t.Fatalf("GetService failed: %v", err)
	}
	if got.Name != "get-test" {
		t.Errorf("expected 'get-test', got '%s'", got.Name)
	}
}

func TestUpdateServiceStatus(t *testing.T) {
	db := openTestDB(t)
	svc := &models.Service{
		Name: "status-test", DisplayName: "Status Test",
		URL: "https://status.test", Type: "external",
	}
	id, _ := db.CreateService(svc)

	if err := db.UpdateServiceStatus(id, "online", 42); err != nil {
		t.Fatalf("UpdateServiceStatus failed: %v", err)
	}

	got, _ := db.GetService(id)
	if got.Status != "online" {
		t.Errorf("expected status 'online', got '%s'", got.Status)
	}
	if got.Latency != 42 {
		t.Errorf("expected latency 42, got %d", got.Latency)
	}
}

func TestRecordAndGetHealthChecks(t *testing.T) {
	db := openTestDB(t)
	svc := &models.Service{
		Name: "health-test", DisplayName: "Health Test",
		URL: "https://health.test", Type: "external",
	}
	id, _ := db.CreateService(svc)

	for i := 0; i < 3; i++ {
		h := &models.HealthCheck{
			ServiceID: id,
			Status:    "online",
			Latency:   int64(10 + i),
			Code:      200,
		}
		if err := db.RecordHealthCheck(h); err != nil {
			t.Fatalf("RecordHealthCheck failed: %v", err)
		}
	}

	checks, err := db.GetRecentHealthChecks(id, 10)
	if err != nil {
		t.Fatalf("GetRecentHealthChecks failed: %v", err)
	}
	if len(checks) != 3 {
		t.Errorf("expected 3 checks, got %d", len(checks))
	}
}

func TestAuditLogs(t *testing.T) {
	db := openTestDB(t)
	u := createTestUser(t, db)

	a := &models.AuditLog{
		UserID:    u.ID,
		Action:    "login",
		Resource:  "session",
		Detail:    "User logged in",
		IP:        "10.0.0.1",
		UserAgent: "curl",
	}
	if err := db.LogAudit(a); err != nil {
		t.Fatalf("LogAudit failed: %v", err)
	}

	logs, err := db.GetAuditLogs(10)
	if err != nil {
		t.Fatalf("GetAuditLogs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Action != "login" {
		t.Errorf("expected 'login', got '%s'", logs[0].Action)
	}
}

func TestInviteCodes(t *testing.T) {
	db := openTestDB(t)
	u := createTestUser(t, db)

	code := &models.InviteCode{
		Code:      "welcome2026",
		CreatedBy: u.ID,
		MaxUses:   5,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := db.CreateInviteCode(code); err != nil {
		t.Fatalf("CreateInviteCode failed: %v", err)
	}

	ok, err := db.UseInviteCode("welcome2026")
	if err != nil {
		t.Fatalf("UseInviteCode failed: %v", err)
	}
	if !ok {
		t.Fatal("expected invite code to be valid")
	}

	_, err = db.UseInviteCode("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent code")
	}
}

func TestDeleteService(t *testing.T) {
	db := openTestDB(t)
	svc := &models.Service{
		Name: "delete-me", DisplayName: "Delete Me",
		URL: "https://delete.me", Type: "external",
	}
	id, _ := db.CreateService(svc)

	if err := db.DeleteService(id); err != nil {
		t.Fatalf("DeleteService failed: %v", err)
	}

	_, err := db.GetService(id)
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}
