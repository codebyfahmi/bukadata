package service

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"bukadata/internal/analytics"
	"bukadata/internal/auth"
	"bukadata/internal/config"
	"bukadata/internal/dashboard"
	"bukadata/internal/httpx"
	"bukadata/internal/ingest"
	"bukadata/internal/jobs"
	"bukadata/internal/model"
	"bukadata/internal/repository"
	"bukadata/internal/storage"
	"bukadata/internal/workspace"

	"github.com/google/uuid"
)

type Service struct {
	cfg              config.Config
	repo             *repository.Store
	authClient       *auth.SupabaseClient
	rawStorage       storage.RawPayloadStorage
	ingestService    *ingest.Service
	analyticsService *ingest.AnalyticsService
	dashboardService *dashboard.Service
	jobsService      *jobs.Service
}

func New(cfg config.Config, db *sql.DB, authClient *auth.SupabaseClient, rawStorage storage.RawPayloadStorage) *Service {
	repo := repository.New(db)
	analyticsService := ingest.NewAnalyticsService(repo)
	dashboardService := dashboard.New(repo)
	return &Service{
		cfg:              cfg,
		repo:             repo,
		authClient:       authClient,
		rawStorage:       rawStorage,
		ingestService:    ingest.New(repo, rawStorage, analyticsService, dashboardService),
		analyticsService: analyticsService,
		dashboardService: dashboardService,
		jobsService:      jobs.New(repo),
	}
}

func (s *Service) Authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := auth.ParseBearerToken(r.Header.Get("Authorization"))
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, "missing_bearer_token", "missing or invalid authorization header")
			return
		}
		principal, err := s.authClient.ValidateAccessToken(r.Context(), token)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_token", "unable to validate access token")
			return
		}
		ctx := context.WithValue(r.Context(), principalContextKey, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Email == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_fields", "email and password are required")
		return
	}
	token, err := s.authClient.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "email atau password salah")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"expires_in":    token.ExpiresIn,
	})
}

func (s *Service) HandleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.RefreshToken == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_fields", "refresh_token is required")
		return
	}
	token, err := s.authClient.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh token tidak valid atau sudah kedaluwarsa")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"expires_in":    token.ExpiresIn,
	})
}

func (s *Service) HandleIngestBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req model.IngestBatchRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	workspaceSlug, err := workspace.ResolveSlug(r, req.WorkspaceSlug)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	req.WorkspaceSlug = workspaceSlug
	principal, _ := principalFromContext(r.Context()).(*auth.Principal)
	resp, err := s.IngestBatch(r.Context(), principal, req)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (s *Service) HandleGetShopDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	workspaceSlug, err := workspace.ResolveSlug(r, r.URL.Query().Get("workspace_slug"))
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	principal, _ := principalFromContext(r.Context()).(*auth.Principal)
	dashboard, err := s.GetShopDashboard(r.Context(), principal, workspaceSlug, r.URL.Query().Get("marketplace"), r.URL.Query().Get("shop_key"))
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, model.ShopDashboardResponse{Dashboard: dashboard})
}

func (s *Service) HandleRefreshAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req model.RefreshAnalyticsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	workspaceSlug, err := workspace.ResolveSlug(r, req.WorkspaceSlug)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	req.WorkspaceSlug = workspaceSlug
	principal, _ := principalFromContext(r.Context()).(*auth.Principal)
	dashboard, jobID, err := s.RefreshAnalytics(r.Context(), principal, req)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"job_id": jobID, "dashboard": dashboard})
}

func (s *Service) HandleGetProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	workspaceSlug, err := workspace.ResolveSlug(r, r.URL.Query().Get("workspace_slug"))
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	principal, _ := principalFromContext(r.Context()).(*auth.Principal)
	products, err := s.GetProducts(r.Context(), principal, workspaceSlug, r.URL.Query().Get("marketplace"), r.URL.Query().Get("shop_key"))
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, model.ProductsResponse{Products: products})
}

func (s *Service) HandleGetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	workspaceSlug, err := workspace.ResolveSlug(r, r.URL.Query().Get("workspace_slug"))
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	principal, _ := principalFromContext(r.Context()).(*auth.Principal)
	job, err := s.GetJob(r.Context(), principal, workspaceSlug, strings.TrimPrefix(r.URL.Path, "/v1/extension/jobs/"))
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, model.JobResponse{Job: job})
}

func (s *Service) IngestBatch(ctx context.Context, principal *auth.Principal, req model.IngestBatchRequest) (*model.IngestBatchResponse, error) {
	if principal == nil {
		return nil, NewError(http.StatusUnauthorized, "unauthorized", "missing principal")
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil { return nil, err }
	defer tx.Rollback()
	workspaceRef, err := s.repo.UpsertUserAndGetWorkspace(ctx, tx, principal.AuthUserID, principal.Email, principal.FullName, req.WorkspaceSlug)
	if err != nil {
		return nil, translateRepoError(err, "workspace_access_denied", "user is not a member of the workspace")
	}
	jobID := uuid.NewString()
	if err := s.repo.InsertIngestJob(ctx, tx, jobID, workspaceRef.ID, req.Marketplace, len(req.Items)); err != nil {
		return nil, err
	}
	resp, err := s.ingestService.Run(ctx, tx, workspaceRef, req, jobID)
	if err != nil {
		_ = s.repo.FailIngestJob(ctx, tx, jobID, err.Error())
		return nil, err
	}
	if err := s.repo.CompleteIngestJob(ctx, tx, jobID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil { return nil, err }
	return resp, nil
}

func (s *Service) GetShopDashboard(ctx context.Context, principal *auth.Principal, workspaceSlug, marketplace, shopKey string) (model.ShopEnvelope, error) {
	if principal == nil {
		return model.ShopEnvelope{}, NewError(http.StatusUnauthorized, "unauthorized", "missing principal")
	}
	if workspaceSlug == "" || marketplace == "" || shopKey == "" {
		return model.ShopEnvelope{}, NewError(http.StatusBadRequest, "missing_query_params", "workspace_slug, marketplace, and shop_key are required")
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil { return model.ShopEnvelope{}, err }
	defer tx.Rollback()
	workspaceRef, err := s.repo.UpsertUserAndGetWorkspace(ctx, tx, principal.AuthUserID, principal.Email, principal.FullName, workspaceSlug)
	if err != nil { return model.ShopEnvelope{}, translateRepoError(err, "workspace_access_denied", "user is not a member of the workspace") }
	dash, err := s.dashboardService.GetLatest(ctx, tx, workspaceRef.ID, marketplace, shopKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s.dashboardService.Empty(marketplace, shopKey), nil
		}
		return model.ShopEnvelope{}, err
	}
	if err := tx.Commit(); err != nil { return model.ShopEnvelope{}, err }
	return dash, nil
}

func (s *Service) RefreshAnalytics(ctx context.Context, principal *auth.Principal, req model.RefreshAnalyticsRequest) (model.ShopEnvelope, string, error) {
	if principal == nil {
		return model.ShopEnvelope{}, "", NewError(http.StatusUnauthorized, "unauthorized", "missing principal")
	}
	if req.WorkspaceSlug == "" || req.Marketplace == "" || req.ShopKey == "" {
		return model.ShopEnvelope{}, "", NewError(http.StatusBadRequest, "missing_fields", "workspace_slug, marketplace, and shop_key are required")
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil { return model.ShopEnvelope{}, "", err }
	defer tx.Rollback()
	workspaceRef, err := s.repo.UpsertUserAndGetWorkspace(ctx, tx, principal.AuthUserID, principal.Email, principal.FullName, req.WorkspaceSlug)
	if err != nil { return model.ShopEnvelope{}, "", translateRepoError(err, "workspace_access_denied", "user is not a member of the workspace") }
	jobID := uuid.NewString()
	if err := s.repo.InsertAnalyticsJob(ctx, tx, jobID, workspaceRef.ID, req.Marketplace, req.ShopKey, req.Reason); err != nil {
		return model.ShopEnvelope{}, "", err
	}
	if err := s.analyticsService.RecomputeShop(ctx, tx, workspaceRef.ID, req.Marketplace, req.ShopKey, chooseReason(req.Reason)); err != nil {
		_ = s.repo.FailAnalyticsJob(ctx, tx, jobID, err.Error())
		return model.ShopEnvelope{}, "", err
	}
	if err := s.repo.CompleteAnalyticsJob(ctx, tx, jobID); err != nil {
		return model.ShopEnvelope{}, "", err
	}
	dash, err := s.dashboardService.GetLatest(ctx, tx, workspaceRef.ID, req.Marketplace, req.ShopKey)
	if err != nil { return model.ShopEnvelope{}, "", err }
	if err := tx.Commit(); err != nil { return model.ShopEnvelope{}, "", err }
	return dash, jobID, nil
}

func (s *Service) GetProducts(ctx context.Context, principal *auth.Principal, workspaceSlug, marketplace, shopKey string) ([]model.ProductEnvelope, error) {
	if principal == nil {
		return nil, NewError(http.StatusUnauthorized, "unauthorized", "missing principal")
	}
	if workspaceSlug == "" || marketplace == "" || shopKey == "" {
		return nil, NewError(http.StatusBadRequest, "missing_query_params", "workspace_slug, marketplace, and shop_key are required")
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil { return nil, err }
	defer tx.Rollback()
	workspaceRef, err := s.repo.UpsertUserAndGetWorkspace(ctx, tx, principal.AuthUserID, principal.Email, principal.FullName, workspaceSlug)
	if err != nil { return nil, translateRepoError(err, "workspace_access_denied", "user is not a member of the workspace") }
	rows, err := s.repo.ListProducts(ctx, tx, workspaceRef.ID, marketplace, shopKey)
	if err != nil { return nil, err }
	products := make([]model.ProductEnvelope, 0, len(rows))
	for _, row := range rows {
		products = append(products, model.ProductEnvelope{
			ID:                row.ID,
			ProductIDExternal: row.ProductIDExternal,
			ProductKey:        row.ProductKey,
			Name:              row.Name,
			URL:               row.URL,
			CapturedAt:        row.CapturedAt,
			Profile:           decodeMap(row.ProfileJSON),
			Pricing:           decodeMap(row.PricingJSON),
			Performance:       decodeMap(row.PerformanceJSON),
			Inventory:         decodeMap(row.InventoryJSON),
			Promotion:         decodeMap(row.PromotionJSON),
			Review:            decodeMap(row.ReviewJSON),
			Debug:             decodeMap(row.DebugJSON),
		})
	}
	if err := tx.Commit(); err != nil { return nil, err }
	return products, nil
}

func (s *Service) GetJob(ctx context.Context, principal *auth.Principal, workspaceSlug, jobID string) (model.JobEnvelope, error) {
	if principal == nil {
		return model.JobEnvelope{}, NewError(http.StatusUnauthorized, "unauthorized", "missing principal")
	}
	if workspaceSlug == "" || jobID == "" {
		return model.JobEnvelope{}, NewError(http.StatusBadRequest, "missing_fields", "workspace_slug and job_id are required")
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil { return model.JobEnvelope{}, err }
	defer tx.Rollback()
	workspaceRef, err := s.repo.UpsertUserAndGetWorkspace(ctx, tx, principal.AuthUserID, principal.Email, principal.FullName, workspaceSlug)
	if err != nil { return model.JobEnvelope{}, translateRepoError(err, "workspace_access_denied", "user is not a member of the workspace") }
	job, err := s.jobsService.Get(ctx, tx, workspaceRef.ID, jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.JobEnvelope{}, NewError(http.StatusNotFound, "job_not_found", "job not found")
		}
		return model.JobEnvelope{}, err
	}
	if err := tx.Commit(); err != nil { return model.JobEnvelope{}, err }
	return job, nil
}

func translateRepoError(err error, code, message string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return NewError(http.StatusForbidden, code, message)
	}
	return err
}

func decodeMap(raw []byte) map[string]any {
	return analytics.DecodeMap(raw)
}

func chooseReason(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return "manual_refresh"
	}
	return reason
}
