package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"

	"bukadata/internal/analytics"
	"bukadata/internal/model"
	"bukadata/internal/repository"
)

type Service struct {
	repo *repository.Store
}

func New(repo *repository.Store) *Service {
	return &Service{repo: repo}
}

func (s *Service) Empty(marketplace, shopKey string) model.ShopEnvelope {
	return analytics.EmptyDashboard(marketplace, shopKey)
}

func (s *Service) GetLatest(ctx context.Context, tx *sql.Tx, workspaceID, marketplace, shopKey string) (model.ShopEnvelope, error) {
	view, err := s.repo.GetLatestDashboard(ctx, tx, workspaceID, marketplace, shopKey)
	if err != nil {
		return model.ShopEnvelope{}, err
	}
	var products []any
	_ = json.Unmarshal(view.ProductsJSON, &products)
	var insights []any
	_ = json.Unmarshal(view.InsightsJSON, &insights)
	return model.ShopEnvelope{
		Marketplace: marketplace,
		ShopKey:     shopKey,
		GeneratedAt: view.CapturedAt,
		Source:      "server",
		Tabs: model.DashboardTabs{
			Profile:     analytics.DecodeMap(view.ProfileJSON),
			Reputation:  analytics.DecodeMap(view.ReputationJSON),
			Catalog:     analytics.DecodeMap(view.CatalogJSON),
			Performance: analytics.DecodeMap(view.PerformanceJSON),
			Promotions:  analytics.DecodeMap(view.PromotionsJSON),
			Operations:  analytics.DecodeMap(view.OperationsJSON),
			Products:    products,
			Insights:    insights,
			Debug:       analytics.DecodeMap(view.DebugJSON),
		},
	}, nil
}

func IsNotFound(err error) bool {
	return err == sql.ErrNoRows
}
