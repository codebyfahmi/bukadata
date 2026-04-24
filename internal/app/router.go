package app

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"bukadata/internal/auth"
	"bukadata/internal/config"
	"bukadata/internal/httpx"
	"bukadata/internal/observability"
	"bukadata/internal/service"
	"bukadata/internal/storage"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	once    sync.Once
	handler http.Handler
)

func New() http.Handler {
	once.Do(func() {
		cfg := config.MustLoad()
		db, err := sql.Open("pgx", cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("open database: %v", err)
		}
		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetMaxIdleConns(5)
		db.SetMaxOpenConns(10)
		authClient := auth.NewSupabaseClient(cfg.SupabaseURL, cfg.SupabaseAnonKey, cfg.SupabaseServiceRoleKey)
		rawStorage := storage.NewSupabaseRawPayloadStorage(cfg)
		svc := service.New(cfg, db, authClient, rawStorage)
		handler = observability.RequestLogging(routes(svc))
	})
	return handler
}

func routes(svc *service.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	mux.Handle("/v1/extension/ingest/captures/batch", svc.Authenticated(http.HandlerFunc(svc.HandleIngestBatch)))
	mux.Handle("/v1/extension/dashboard/shop", svc.Authenticated(http.HandlerFunc(svc.HandleGetShopDashboard)))
	mux.Handle("/v1/extension/analytics/refresh", svc.Authenticated(http.HandlerFunc(svc.HandleRefreshAnalytics)))
	mux.Handle("/v1/extension/products", svc.Authenticated(http.HandlerFunc(svc.HandleGetProducts)))
	mux.Handle("/v1/extension/jobs/", svc.Authenticated(http.HandlerFunc(svc.HandleGetJob)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"name": "bukadata-marketplace-intelligence-api"})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/") || r.URL.Path == "/healthz" {
			mux.ServeHTTP(w, r)
			return
		}
		httpx.WriteError(w, http.StatusNotFound, "route_not_found", "route not found")
	})
}
