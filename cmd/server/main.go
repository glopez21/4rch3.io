package main

import (
	"context"
	"fmt"
	"log"
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
	"github.com/4rch3/portal/internal/middleware"
	"github.com/4rch3/portal/internal/monitor"
	"github.com/4rch3/portal/internal/proxy"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	authSvc := auth.New(auth.AuthConfig{
		JWTSecret:         cfg.Auth.JWTSecret,
		JWTExpiry:         cfg.Auth.JWTExpiry,
		BCryptCost:        cfg.Auth.BCryptCost,
		SessionCookieName: cfg.Auth.SessionCookieName,
	})

	svc := auth.NewService(authSvc, db)

	mw, err := middleware.New(svc, cfg.Server.LANSubnets, cfg.Server.TrustedProxies)
	if err != nil {
		log.Fatalf("failed to create middleware: %v", err)
	}

	proxySvc := proxy.New(db)

	authHandler := handlers.NewAuthHandler(db, svc, auth.AuthConfig{
		JWTSecret:         cfg.Auth.JWTSecret,
		JWTExpiry:         cfg.Auth.JWTExpiry,
		BCryptCost:        cfg.Auth.BCryptCost,
		SessionCookieName: cfg.Auth.SessionCookieName,
	})
	dashHandler := handlers.NewDashboardHandler(db)
	adminHandler := handlers.NewAdminHandler(db, svc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.Discovery.DockerSocket != "" {
		dockerDisc, err := discovery.NewDockerDiscovery(
			db,
			cfg.Discovery.DockerSocket,
			cfg.Discovery.LabelPrefix,
			cfg.Discovery.DefaultNetwork,
			cfg.Discovery.ScanInterval,
		)
		if err != nil {
			log.Printf("warning: failed to create docker discovery: %v", err)
		} else {
			go dockerDisc.Start(ctx)
			defer dockerDisc.Close()
		}
	}

	mon := monitor.New(db, cfg.Monitor.CheckInterval, cfg.Monitor.Timeout, cfg.Monitor.RetryCount)
	go mon.Start(ctx)

	mux := http.NewServeMux()

	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/logout", authHandler.Logout)
	mux.HandleFunc("/api/auth/me", authHandler.Me)

	mux.Handle("/api/services/status", mw.RequireAuth(http.HandlerFunc(dashHandler.ServiceStatus)))
	mux.Handle("/admin", mw.RequireAuth(mw.RequireRole("admin")(http.HandlerFunc(adminHandler.Dashboard))))
	mux.Handle("/admin/users", mw.RequireAuth(mw.RequireRole("admin")(http.HandlerFunc(adminHandler.Users))))

	mux.Handle("/svc/", mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxySvc.Handler().ServeHTTP(w, r)
	})))

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"4rch3-portal"}`))
	})

	mux.Handle("/", mw.Authenticate(http.HandlerFunc(dashHandler.Index)))

	var finalHandler http.Handler = mux

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
		log.Printf("4rch3.io portal starting on %s", addr)
		log.Printf("mode: %s", func() string {
			if len(cfg.Server.ExternalURL) > 0 {
				return "public (" + cfg.Server.ExternalURL + ")"
			}
			return "local-only"
		}())

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}
