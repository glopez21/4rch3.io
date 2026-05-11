package middleware

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/4rch3/portal/internal/auth"
	"github.com/4rch3/portal/internal/models"
)

type contextKey string

const (
	UserKey      contextKey = "user"
	ClaimsKey    contextKey = "claims"
	IsLANKey     contextKey = "is_lan"
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
	authService  *auth.Service
	lanSubnets   []*net.IPNet
	trustedProxies []*net.IPNet
}

func New(authService *auth.Service, lanCIDRs, trustedCIDRs []string) (*Middleware, error) {
	m := &Middleware{authService: authService}

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
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := net.ParseIP(strings.TrimSpace(parts[0]))
		if ip != nil {
			for _, trusted := range m.trustedProxies {
				if trusted.Contains(ip) {
					return strings.TrimSpace(parts[0])
				}
			}
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
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
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		if cookie, err := r.Cookie("4rch3_session"); err == nil {
			token = cookie.Value
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
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(token, "apikey:") {
			apiKey := strings.TrimPrefix(token, "apikey:")
			user, err := m.authService.ValidateAPIKey(apiKey)
			if err == nil && user != nil {
				ctx := context.WithValue(r.Context(), UserKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		claims, err := m.authService.ValidateJWT(token)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		user, err := m.authService.GetUserByID(claims.UserID)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

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
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.status, time.Since(start))
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
