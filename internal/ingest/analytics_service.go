package ingest

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"bukadata/internal/analytics"
	"bukadata/internal/httpx"
	"bukadata/internal/repository"
)

type AnalyticsService struct {
	repo *repository.Store
}

func NewAnalyticsService(repo *repository.Store) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

func (s *AnalyticsService) RecomputeShop(ctx context.Context, tx *sql.Tx, workspaceID, marketplace, shopKey, reason string) error {
	shopID, err := s.repo.FindShopID(ctx, tx, workspaceID, marketplace, shopKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return httpx.NewError(http.StatusNotFound, "shop_not_found", "shop not found")
		}
		return err
	}
	candidates, err := s.repo.ListParsedCandidates(ctx, tx, workspaceID, marketplace, shopKey, analytics.ParsedCaptureLimit)
	if err != nil {
		return err
	}
	snapshot := analytics.BuildShopSnapshot(marketplace, shopKey, reason, candidates, time.Now().UTC())
	if err := s.repo.ReplaceLatestShopSnapshot(ctx, tx, repository.NewShopSnapshotRecord(workspaceID, shopID, snapshot)); err != nil {
		return err
	}
	if err := s.repo.ClearLatestProductSnapshotsForShop(ctx, tx, workspaceID, shopID); err != nil {
		return err
	}
	for _, product := range snapshot.Products {
		productRecord := repository.NewProductRecord(workspaceID, shopID, marketplace, product)
		productID, err := s.repo.UpsertProduct(ctx, tx, productRecord)
		if err != nil {
			return err
		}
		if err := s.repo.ReplaceLatestProductSnapshot(ctx, tx, repository.NewProductSnapshotRecord(workspaceID, shopID, productID, snapshot.CapturedAt, product)); err != nil {
			return err
		}
	}
	return nil
}
