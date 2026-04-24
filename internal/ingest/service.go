package ingest

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"strings"
	"time"

	"bukadata/internal/analytics"
	"bukadata/internal/model"
	"bukadata/internal/repository"
	"bukadata/internal/service"
)

type AnalyticsWriter interface {
	RecomputeShop(ctx context.Context, tx *sql.Tx, workspaceID, marketplace, shopKey, reason string) error
}

type DashboardReader interface {
	GetLatest(ctx context.Context, tx *sql.Tx, workspaceID, marketplace, shopKey string) (model.ShopEnvelope, error)
}

type RawPayloadStorage interface {
	Store(ctx context.Context, objectPath string, body string) (inlineText *string, storagePath *string, err error)
}

type Service struct {
	repo       *repository.Store
	storage    RawPayloadStorage
	analytics  AnalyticsWriter
	dashboard  DashboardReader
}

func New(repo *repository.Store, storage RawPayloadStorage, analytics AnalyticsWriter, dashboard DashboardReader) *Service {
	return &Service{repo: repo, storage: storage, analytics: analytics, dashboard: dashboard}
}

func (s *Service) Run(ctx context.Context, tx *sql.Tx, workspace repository.Workspace, req model.IngestBatchRequest, jobID string) (*model.IngestBatchResponse, error) {
	if req.Marketplace == "" {
		return nil, service.NewError(http.StatusBadRequest, "missing_marketplace", "marketplace is required")
	}
	if !isSupportedMarketplace(req.Marketplace) {
		return nil, service.NewError(http.StatusBadRequest, "invalid_marketplace", "marketplace is not supported")
	}
	if len(req.Items) == 0 {
		return nil, service.NewError(http.StatusBadRequest, "missing_items", "items must not be empty")
	}

	affectedShops := map[string]struct{}{}
	for index, item := range req.Items {
		capture, err := normalizeCapture(req.Marketplace, item.Capture)
		if err != nil {
			return nil, err
		}
		if !isSupportedMarketplace(capture.Marketplace) {
			return nil, service.NewError(http.StatusBadRequest, "invalid_marketplace", "marketplace is not supported")
		}
		if _, err := s.repo.EnsureShop(ctx, tx, workspace.ID, capture); err != nil {
			return nil, err
		}
		requestHash := analytics.SHA256Hex(capture.RawCapture.RequestBody)
		responseHash := analytics.SHA256Hex(capture.RawCapture.ResponseBody)
		rawID, err := s.repo.FindDuplicateRawCaptureID(ctx, tx, workspace.ID, capture.Marketplace, capture.ShopKey, requestHash, responseHash, capture.CapturedAt)
		if err != nil {
			return nil, err
		}
		if rawID == "" {
			requestInline, requestStoragePath, err := s.storage.Store(ctx, buildStorageObjectPath(workspace.Slug, capture.ShopKey, "request", requestHash), capture.RawCapture.RequestBody)
			if err != nil { return nil, err }
			responseInline, responseStoragePath, err := s.storage.Store(ctx, buildStorageObjectPath(workspace.Slug, capture.ShopKey, "response", responseHash), capture.RawCapture.ResponseBody)
			if err != nil { return nil, err }
			rawRecord := repository.NewRawCaptureRecord(workspace.ID, capture, requestHash, responseHash, requestInline, responseInline, requestStoragePath, responseStoragePath)
			rawID = rawRecord.ID
			if err := s.repo.InsertRawCapture(ctx, tx, rawRecord); err != nil { return nil, err }
		}
		if err := s.repo.InsertParsedCapture(ctx, tx, repository.NewParsedCaptureRecord(workspace.ID, rawID, capture)); err != nil {
			return nil, err
		}
		if strings.TrimSpace(capture.ReplayTemplate.Fingerprint) != "" {
			if err := s.repo.UpsertSourceTemplate(ctx, tx, repository.NewSourceTemplateRecord(workspace.ID, capture)); err != nil {
				return nil, err
			}
		}
		affectedShops[capture.ShopKey] = struct{}{}
		if err := s.repo.UpdateIngestJobProgress(ctx, tx, jobID, index+1); err != nil {
			return nil, err
		}
	}

	shopKeys := make([]string, 0, len(affectedShops))
	for shopKey := range affectedShops {
		shopKeys = append(shopKeys, shopKey)
	}
	sort.Strings(shopKeys)
	dashboards := make(map[string]model.ShopEnvelope, len(shopKeys))
	for _, shopKey := range shopKeys {
		if err := s.analytics.RecomputeShop(ctx, tx, workspace.ID, req.Marketplace, shopKey, "ingest"); err != nil {
			return nil, err
		}
		dashboard, err := s.dashboard.GetLatest(ctx, tx, workspace.ID, req.Marketplace, shopKey)
		if err != nil {
			return nil, err
		}
		dashboards[shopKey] = dashboard
	}

	return &model.IngestBatchResponse{Accepted: len(req.Items), Recomputed: true, Dashboards: dashboards, JobID: jobID}, nil
}

func normalizeCapture(defaultMarketplace string, capture model.ExtensionCapture) (model.ExtensionCapture, error) {
	if capture.CapturedAt.IsZero() {
		capture.CapturedAt = time.Now().UTC()
	}
	if capture.Marketplace == "" {
		capture.Marketplace = defaultMarketplace
	}
	if capture.ShopKey == "" {
		capture.ShopKey = capture.PageContext.ShopKey
	}
	if capture.PageContext.PageURL == "" {
		capture.PageContext.PageURL = capture.ParsedSummary.PageContext.PageURL
	}
	if capture.PageContext.ShopKey == "" {
		capture.PageContext.ShopKey = capture.ShopKey
	}
	if capture.Marketplace == "" || capture.ShopKey == "" || capture.PageContext.PageURL == "" {
		return capture, service.NewError(http.StatusBadRequest, "invalid_capture", "each capture must include capturedAt, marketplace, shopKey, and pageContext.pageUrl")
	}
	if capture.RawCapture.URL == "" || capture.ParsedSummary.EntityType == "" && capture.RawCapture.EntityType == "" {
		return capture, service.NewError(http.StatusBadRequest, "invalid_capture", "rawCapture and parsedSummary are required")
	}
	return capture, nil
}

func isSupportedMarketplace(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "tokopedia", "shopee":
		return true
	default:
		return false
	}
}

func buildStorageObjectPath(workspaceSlug, shopKey, kind, hash string) string {
	return "raw-captures/" + workspaceSlug + "/" + shopKey + "/" + kind + "-" + hash + ".txt"
}
