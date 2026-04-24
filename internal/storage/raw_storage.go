package storage

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"bukadata/internal/config"
)

type RawPayloadStorage interface {
	Store(ctx context.Context, objectPath string, body string) (inlineText *string, storagePath *string, err error)
}

type SupabaseRawPayloadStorage struct {
	cfg        config.Config
	httpClient *http.Client
}

func NewSupabaseRawPayloadStorage(cfg config.Config) *SupabaseRawPayloadStorage {
	return &SupabaseRawPayloadStorage{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (s *SupabaseRawPayloadStorage) Store(ctx context.Context, objectPath string, body string) (*string, *string, error) {
	if len(body) <= s.cfg.RawInlineMaxBytes || s.cfg.RawCaptureBucket == "" {
		copy := body
		return &copy, nil, nil
	}

	cleanPath := strings.TrimLeft(path.Clean(objectPath), "/")
	endpoint := strings.TrimRight(s.cfg.SupabaseURL, "/") + "/storage/v1/object/" + url.PathEscape(s.cfg.RawCaptureBucket) + "/" + cleanPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(body))
	if err != nil {
		return nil, nil, fmt.Errorf("build storage request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.cfg.SupabaseServiceRoleKey)
	req.Header.Set("apikey", s.cfg.SupabaseServiceRoleKey)
	req.Header.Set("x-upsert", "true")
	req.Header.Set("Content-Type", mime.TypeByExtension(".txt"))
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("upload raw payload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("upload raw payload failed with status %d", resp.StatusCode)
	}

	return nil, &cleanPath, nil
}
