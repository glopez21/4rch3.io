package auth

import (
	"testing"
	"time"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/models"
)

func testAuth(t *testing.T) *Auth {
	return New("test-secret-that-is-at-least-32-bytes-long!", 1*time.Hour, 4, "4rch3_session")
}

func TestHashAndVerifyPassword(t *testing.T) {
	a := testAuth(t)
	hash, err := a.HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if !a.VerifyPassword(hash, "hunter2") {
		t.Error("VerifyPassword should return true for correct password")
	}
	if a.VerifyPassword(hash, "wrong") {
		t.Error("VerifyPassword should return false for wrong password")
	}
}

func TestGenerateAndValidateJWT(t *testing.T) {
	a := testAuth(t)
	user := &models.User{ID: 1, Username: "admin", Role: "admin"}

	token, err := a.GenerateJWT(user)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := a.ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}
	if claims.UserID != 1 {
		t.Errorf("expected UserID 1, got %d", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Errorf("expected Username 'admin', got '%s'", claims.Username)
	}
	if claims.Role != "admin" {
		t.Errorf("expected Role 'admin', got '%s'", claims.Role)
	}
}

func TestValidateInvalidJWT(t *testing.T) {
	a := testAuth(t)
	_, err := a.ValidateJWT("invalid.token.here")
	if err == nil {
		t.Fatal("expected error for invalid JWT")
	}
}

func TestValidateExpiredJWT(t *testing.T) {
	a := New("test-secret-that-is-at-least-32-bytes-long!", -1*time.Hour, 4, "4rch3_session")
	user := &models.User{ID: 1, Username: "test", Role: "user"}
	token, err := a.GenerateJWT(user)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	_, err = a.ValidateJWT(token)
	if err == nil {
		t.Fatal("expected error for expired JWT")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}
	if len(key) < 10 {
		t.Errorf("expected long key, got %d chars", len(key))
	}
	if key[:6] != "4rch3_" {
		t.Errorf("expected '4rch3_' prefix, got '%s'", key[:6])
	}
}

func TestGenerateSessionToken(t *testing.T) {
	t1, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken failed: %v", err)
	}
	t2, _ := GenerateSessionToken()
	if t1 == "" {
		t.Fatal("expected non-empty token")
	}
	if t1 == t2 {
		t.Fatal("expected unique tokens")
	}
}

func TestGenerateInviteCode(t *testing.T) {
	code, err := GenerateInviteCode()
	if err != nil {
		t.Fatalf("GenerateInviteCode failed: %v", err)
	}
	if code == "" {
		t.Fatal("expected non-empty code")
	}
}

func TestSecureCompare(t *testing.T) {
	if !SecureCompare("hello", "hello") {
		t.Error("expected equal strings to match")
	}
	if SecureCompare("hello", "world") {
		t.Error("expected different strings to not match")
	}
	if SecureCompare("", "non-empty") {
		t.Error("expected empty vs non-empty to not match")
	}
}

func TestAuthServiceCreateUser(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	a := testAuth(t)
	svc := NewService(a, db)

	user, err := svc.CreateUser("testuser", "test@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", user.Username)
	}
	if user.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", user.Role)
	}
	if user.APIKey == "" {
		t.Error("expected non-empty API key")
	}
}

func TestAuthServiceAuthenticate(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	a := testAuth(t)
	svc := NewService(a, db)

	svc.CreateUser("alice", "alice@example.com", "securepass", "admin")

	user, err := svc.Authenticate("alice", "securepass")
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("expected 'alice', got '%s'", user.Username)
	}

	_, err = svc.Authenticate("alice", "wrongpass")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}

	_, err = svc.Authenticate("nonexistent", "anything")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestAuthServiceValidateAPIKey(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	a := testAuth(t)
	svc := NewService(a, db)

	created, err := svc.CreateUser("apiuser", "api@example.com", "testpass", "service")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	found, err := svc.ValidateAPIKey(created.APIKey)
	if err != nil {
		t.Fatalf("ValidateAPIKey failed: %v", err)
	}
	if found.Username != "apiuser" {
		t.Errorf("expected 'apiuser', got '%s'", found.Username)
	}

	_, err = svc.ValidateAPIKey("invalid-key")
	if err == nil {
		t.Fatal("expected error for invalid API key")
	}
}
