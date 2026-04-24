package jobs

import (
	"context"
	"database/sql"

	"bukadata/internal/model"
	"bukadata/internal/repository"
)

type Service struct {
	repo *repository.Store
}

func New(repo *repository.Store) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, tx *sql.Tx, workspaceID, jobID string) (model.JobEnvelope, error) {
	rec, err := s.repo.GetJob(ctx, tx, workspaceID, jobID)
	if err != nil {
		return model.JobEnvelope{}, err
	}
	return model.JobEnvelope{
		ID:             rec.ID,
		Type:           rec.Type,
		Marketplace:    rec.Marketplace,
		ShopKey:        rec.ShopKey,
		Status:         rec.Status,
		ItemsTotal:     rec.ItemsTotal,
		ItemsProcessed: rec.ItemsProcessed,
		Reason:         rec.Reason,
		ErrorMessage:   rec.ErrorMessage,
		CreatedAt:      rec.CreatedAt,
		UpdatedAt:      rec.UpdatedAt,
	}, nil
}
