package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/4rch3/portal/internal/auth"
	"github.com/4rch3/portal/internal/config"
	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/discovery"
	"github.com/4rch3/portal/internal/handlers"
	"github.com/4rch3/portal/internal/metrics"
	"github.com/4rch3/portal/internal/middleware"
	"github.com/4rch3/portal/internal/models"
	"github.com/4rch3/portal/internal/monitor"
	"github.com/4rch3/portal/internal/proxy"
	"golang.org/x/time/rate"
)

func main() {
	bootstrap := flag.Bool("bootstrap", false, "create admin user from env and exit")
	resetPassword := flag.Bool("reset-password", false, "reset a user's password from env and exit")
	flag.Parse()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	overrideFromEnv(cfg)

	if err := cfg.Validate(); err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}

	setupLogging(cfg.Logging)

	if err := os.MkdirAll("data", 0755); err != nil {
		slog.Error("failed to create data dir", "error", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	authSvc := auth.New(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, cfg.Auth.BCryptCost, cfg.Auth.SessionCookieName)

	svc := auth.NewService(authSvc, db)

	if *bootstrap {
		handleBoostrap(svc)
		return
	}

	if *resetPassword {
		handleResetPassword(svc, authSvc)
		return
	}

	tryBootstrapFromEnv(svc)

	mw, err := middleware.New(svc, cfg.Server.LANSubnets, cfg.Server.TrustedProxies, cfg.Auth.SessionCookieName)
	if err != nil {
		slog.Error("failed to create middleware", "error", err)
		os.Exit(1)
	}

	proxySvc := proxy.New(db)

	authHandler := handlers.NewAuthHandler(db, svc, cfg.Auth.SessionCookieName, cfg.Auth.SessionExpiry)
	grafanaDomain := domainFromURL(cfg.Server.ExternalURL)
	dashHandler := handlers.NewDashboardHandler(db, grafanaDomain)
	adminHandler := handlers.NewAdminHandler(db, svc)

	authLimiter := middleware.NewRateLimiter(rate.Limit(cfg.Auth.RateLimitPerMin)/60, 1)
	apiLimiter := middleware.NewRateLimiter(rate.Limit(cfg.Auth.APIRateLimitPerMin)/60, 5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n, err := db.DeleteExpiredSessions()
				if err != nil {
					slog.Warn("session cleanup failed", "error", err)
				} else if n > 0 {
					slog.Debug("cleaned expired sessions", "count", n)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				services, err := db.ListServices()
				if err == nil {
					var active float64
					for _, svc := range services {
						if svc.Enabled {
							active++
						}
					}
					metrics.ActiveServices.Set(active)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	if cfg.Discovery.DockerSocket != "" {
		dockerDisc, err := discovery.NewDockerDiscovery(
			db,
			cfg.Discovery.DockerSocket,
			cfg.Discovery.LabelPrefix,
			cfg.Discovery.DefaultNetwork,
			cfg.Discovery.ScanInterval,
		)
		if err != nil {
			slog.Warn("failed to create docker discovery", "error", err)
		} else {
			if err := dockerDisc.TestConnection(); err != nil {
				slog.Warn("docker socket not available", "error", err)
			} else {
				go dockerDisc.Start(ctx)
				defer dockerDisc.Close()
			}
		}
	}

	mon := monitor.New(db, cfg.Monitor.CheckInterval, cfg.Monitor.Timeout, cfg.Monitor.RetryCount)
	go mon.Start(ctx)

	mux := http.NewServeMux()

	mux.Handle("/login", authLimiter.Middleware(http.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("/logout", authHandler.Logout)
	mux.HandleFunc("/api/auth/me", authHandler.Me)

	mux.Handle("/api/services/status", apiLimiter.Middleware(mw.RequireAuth(http.HandlerFunc(dashHandler.ServiceStatus))))
	mux.Handle("/admin", mw.RequireAuth(mw.RequireRole("admin")(http.HandlerFunc(adminHandler.Dashboard))))
	mux.Handle("/admin/users", mw.RequireAuth(mw.RequireRole("admin")(http.HandlerFunc(adminHandler.Users))))

	mux.Handle("/svc/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxySvc.Handler().ServeHTTP(w, r)
	}))

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"4rch3-portal"}`))
	})

	mux.Handle("/metrics", metrics.Handler())

	webhookHandler := handlers.NewWebhookHandler(db)
	mux.HandleFunc("/api/webhook/deploy", webhookHandler.Deploy)
	mux.HandleFunc("/api/deploys", webhookHandler.RecentDeploys)

	mux.Handle("/api/deploy-panel", mw.RequireAuth(http.HandlerFunc(dashHandler.DeployPanel)))

	mux.Handle("/", http.HandlerFunc(dashHandler.Index))

	var finalHandler http.Handler = mux

	finalHandler = metrics.Instrument(finalHandler)
	finalHandler = middleware.CORSMiddleware(cfg.CORS)(finalHandler)
	finalHandler = mw.Authenticate(finalHandler)
	finalHandler = mw.LANDetect(finalHandler)
	finalHandler = middleware.SecurityHeaders(finalHandler)
	finalHandler = middleware.Logging(finalHandler)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      finalHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("portal starting", "addr", addr)
		slog.Info("mode", "mode", func() string {
			if len(cfg.Server.ExternalURL) > 0 {
				return "public (" + cfg.Server.ExternalURL + ")"
			}
			return "local-only"
		}())

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server exited")
}

func setupLogging(cfg config.LoggingConfig) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var h slog.Handler
	switch cfg.Format {
	case "json":
		h = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	default:
		h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(h))
}

func overrideFromEnv(cfg *config.Config) {
	if v := os.Getenv("PORTAL_JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("PORTAL_DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("PORTAL_EXTERNAL_URL"); v != "" {
		cfg.Server.ExternalURL = v
	}
	if v := os.Getenv("PORTAL_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Server.Port)
	}
}

func tryBootstrapFromEnv(svc *auth.Service) {
	username := os.Getenv("PORTAL_ADMIN_USERNAME")
	password := os.Getenv("PORTAL_ADMIN_PASSWORD")
	email := os.Getenv("PORTAL_ADMIN_EMAIL")

	if username == "" || password == "" {
		return
	}

	users, err := svc.ListUsers()
	if err != nil {
		slog.Warn("could not check for existing users", "error", err)
		return
	}

	for _, u := range users {
		if u.Role == "admin" {
			slog.Info("admin user already exists, skipping bootstrap")
			return
		}
	}

	if email == "" {
		email = username + "@4rch3.io"
	}

	if _, err := svc.CreateUser(username, email, password, "admin"); err != nil {
		slog.Warn("failed to bootstrap admin user", "error", err)
		return
	}

	slog.Info("bootstrapped admin user", "username", username)
}

func handleBoostrap(svc *auth.Service) {
	username := os.Getenv("PORTAL_ADMIN_USERNAME")
	password := os.Getenv("PORTAL_ADMIN_PASSWORD")
	email := os.Getenv("PORTAL_ADMIN_EMAIL")

	if username == "" || password == "" {
		log.Fatal("PORTAL_ADMIN_USERNAME and PORTAL_ADMIN_PASSWORD must be set")
	}
	if email == "" {
		email = username + "@4rch3.io"
	}

	user, err := svc.CreateUser(username, email, password, "admin")
	if err != nil {
		log.Fatalf("failed to create admin user: %v", err)
	}

	fmt.Printf("admin user created:\n")
	fmt.Printf("  username: %s\n", user.Username)
	fmt.Printf("  email:    %s\n", user.Email)
	fmt.Printf("  api key:  %s\n", user.APIKey)
}

func handleResetPassword(svc *auth.Service, authSvc *auth.Auth) {
	username := os.Getenv("PORTAL_ADMIN_USERNAME")
	password := os.Getenv("PORTAL_ADMIN_PASSWORD")

	if username == "" || password == "" {
		log.Fatal("PORTAL_ADMIN_USERNAME and PORTAL_ADMIN_PASSWORD must be set")
	}

	users, err := svc.ListUsers()
	if err != nil {
		log.Fatalf("failed to list users: %v", err)
	}

	var target *models.User
	for _, u := range users {
		if u.Username == username {
			target = u
			break
		}
	}

	if target == nil {
		log.Fatalf("user %q not found", username)
	}

	hash, err := authSvc.HashPassword(password)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	if err := svc.UpdatePassword(target.ID, hash); err != nil {
		log.Fatalf("failed to update password: %v", err)
	}

	fmt.Printf("password reset for user %q (id=%d)\n", target.Username, target.ID)
}

func domainFromURL(raw string) string {
	if raw == "" {
		return "homelab.local"
	}
	// strip scheme
	domain := raw
	for _, prefix := range []string{"https://", "http://"} {
		if len(domain) > len(prefix) && domain[:len(prefix)] == prefix {
			domain = domain[len(prefix):]
			break
		}
	}
	// strip path
	for i := 0; i < len(domain); i++ {
		if domain[i] == '/' || domain[i] == ':' {
			domain = domain[:i]
			break
		}
	}
	return domain
}
