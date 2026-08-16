package middleware

import (
	"bufio"
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/4rch3/portal/internal/auth"
	"github.com/4rch3/portal/internal/models"
)

type contextKey string

const (
	UserKey   contextKey = "user"
	ClaimsKey contextKey = "claims"
	IsLANKey  contextKey = "is_lan"
)

func GetUser(r *http.Request) *models.User {
	user, _ := r.Context().Value(UserKey).(*models.User)
	return user
}

func GetClaims(r *http.Request) *auth.Claims {
	claims, _ := r.Context().Value(ClaimsKey).(*auth.Claims)
	return claims
}

func IsLAN(r *http.Request) bool {
	lan, _ := r.Context().Value(IsLANKey).(bool)
	return lan
}

type Middleware struct {
	authService    *auth.Service
	lanSubnets     []*net.IPNet
	trustedProxies []*net.IPNet
	sessionCookie  string
}

func New(authService *auth.Service, lanCIDRs, trustedCIDRs []string, sessionCookie string) (*Middleware, error) {
	m := &Middleware{authService: authService, sessionCookie: sessionCookie}

	for _, cidr := range lanCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		m.lanSubnets = append(m.lanSubnets, network)
	}

	for _, cidr := range trustedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		m.trustedProxies = append(m.trustedProxies, network)
	}

	return m, nil
}

func (m *Middleware) realIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remoteIP := net.ParseIP(host)
	if remoteIP == nil {
		return host
	}

	isTrusted := false
	for _, trusted := range m.trustedProxies {
		if trusted.Contains(remoteIP) {
			isTrusted = true
			break
		}
	}

	if !isTrusted {
		return host
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(parts[i]))
			if ip == nil {
				continue
			}
			isProxy := false
			for _, trusted := range m.trustedProxies {
				if trusted.Contains(ip) {
					isProxy = true
					break
				}
			}
			if !isProxy {
				return ip.String()
			}
		}
		for _, part := range parts {
			if ip := net.ParseIP(strings.TrimSpace(part)); ip != nil {
				return ip.String()
			}
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip := net.ParseIP(xri); ip != nil {
			return ip.String()
		}
	}

	return host
}

func (m *Middleware) isLANRequest(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, subnet := range m.lanSubnets {
		if subnet.Contains(parsed) {
			return true
		}
	}
	return false
}

func (m *Middleware) LANDetect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := m.realIP(r)
		ctx := context.WithValue(r.Context(), IsLANKey, m.isLANRequest(ip))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + msg + `"}`))
}

func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user == nil {
			if strings.HasPrefix(r.Header.Get("Accept"), "application/json") ||
				r.Header.Get("HX-Request") != "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) RequireRole(roles ...models.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r)
			if user == nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSONError(w, http.StatusForbidden, "forbidden")
		})
	}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		var fromCookie bool

		if cookie, err := r.Cookie(m.sessionCookie); err == nil {
			token = cookie.Value
			fromCookie = true
		}

		if token == "" {
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
				token = "apikey:" + apiKey
			}
		}

		if token == "" {
			slog.Debug("authenticate: no token found", "path", r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(token, "apikey:") {
			apiKey := strings.TrimPrefix(token, "apikey:")
			user, err := m.authService.ValidateAPIKey(apiKey)
			if err == nil && user != nil {
				slog.Debug("authenticate: api key valid", "username", user.Username)
				ctx := context.WithValue(r.Context(), UserKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			slog.Debug("authenticate: api key invalid", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		if fromCookie {
			session, err := m.authService.GetSession(token)
			if err != nil || session == nil {
				slog.Debug("authenticate: session not found", "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if time.Now().After(session.ExpiresAt) {
				slog.Debug("authenticate: session expired", "expires_at", session.ExpiresAt)
				m.authService.DeleteSession(token)
				next.ServeHTTP(w, r)
				return
			}
			user, err := m.authService.GetUserByID(session.UserID)
			if err != nil || user == nil {
				slog.Debug("authenticate: user not found", "user_id", session.UserID, "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if !user.Active {
				slog.Debug("authenticate: user inactive", "username", user.Username)
				m.authService.DeleteSession(token)
				next.ServeHTTP(w, r)
				return
			}
			slog.Debug("authenticate: session valid", "username", user.Username)
			ctx := context.WithValue(r.Context(), UserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		claims, err := m.authService.ValidateJWT(token)
		if err != nil {
			slog.Debug("authenticate: jwt invalid", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		user, err := m.authService.GetUserByID(claims.UserID)
		if err != nil {
			slog.Debug("authenticate: jwt user not found", "user_id", claims.UserID, "error", err)
			next.ServeHTTP(w, r)
			return
		}

		if !user.Active {
			slog.Debug("authenticate: jwt user inactive", "username", user.Username)
			next.ServeHTTP(w, r)
			return
		}

		slog.Debug("authenticate: jwt valid", "username", user.Username)

		ctx := context.WithValue(r.Context(), UserKey, user)
		ctx = context.WithValue(ctx, ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(wrapped, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration", time.Since(start),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}
