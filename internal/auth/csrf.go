package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (a *Auth) GenerateCSRFToken(sessionToken string) (string, error) {
	nonce, err := generateRandomBytes(16)
	if err != nil {
		return "", err
	}
	ts := time.Now().Unix()
	data := fmt.Sprintf("%s:%d:%s", base64.RawURLEncoding.EncodeToString(nonce), ts, sessionToken)
	mac := hmac.New(sha256.New, []byte(a.jwtSecret))
	mac.Write([]byte(data))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(data)) + "." + sig, nil
}

func (a *Auth) ValidateCSRFToken(token, sessionToken string, maxAge time.Duration) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}

	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(a.jwtSecret))
	mac.Write(data)
	expected := mac.Sum(nil)

	if !hmac.Equal(sig, expected) {
		return false
	}

	dataStr := string(data)
	pieces := strings.SplitN(dataStr, ":", 3)
	if len(pieces) != 3 {
		return false
	}

	ts, err := strconv.ParseInt(pieces[1], 10, 64)
	if err != nil {
		return false
	}

	if pieces[2] != sessionToken {
		return false
	}

	if time.Since(time.Unix(ts, 0)) > maxAge {
		return false
	}

	return true
}
