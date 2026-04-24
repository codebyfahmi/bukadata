package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"bukadata/internal/analytics"
	"bukadata/internal/model"
)

func (s *Store) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, &sql.TxOptions{})
}

func (s *Store) UpsertUserAndGetWorkspace(ctx context.Context, tx *sql.Tx, authUserID, email, fullName, workspaceSlug string) (Workspace, error) {
	var userID string
	err := tx.QueryRowContext(ctx, `
		insert into users (id, auth_user_id, email, full_name, created_at, updated_at)
		values (gen_random_uuid(), $1, $2, $3, now(), now())
		on conflict (auth_user_id)
		do update set email = excluded.email, full_name = excluded.full_name, updated_at = now()
		returning id
	`, authUserID, nullableStringValue(email), nullableStringValue(fullName)).Scan(&userID)
	if err != nil {
		return Workspace{}, err
	}

	var workspace Workspace
	err = tx.QueryRowContext(ctx, `
		select w.id, w.slug, w.name
		from workspaces w
		join workspace_members wm on wm.workspace_id = w.id
		where w.slug = $1 and wm.user_id = $2
		limit 1
	`, workspaceSlug, userID).Scan(&workspace.ID, &workspace.Slug, &workspace.Name)
	if err != nil {
		return Workspace{}, err
	}
	return workspace, nil
}

func (s *Store) InsertIngestJob(ctx context.Context, tx *sql.Tx, id, workspaceID, marketplace string, itemsTotal int) error {
	_, err := tx.ExecContext(ctx, `
		insert into ingest_jobs (id, workspace_id, marketplace_code, shop_key, status, items_total, items_processed, created_at, updated_at)
		values ($1, $2, $3, '', 'running', $4, 0, now(), now())
	`, id, workspaceID, marketplace, itemsTotal)
	return err
}

func (s *Store) UpdateIngestJobProgress(ctx context.Context, tx *sql.Tx, id string, processed int) error {
	_, err := tx.ExecContext(ctx, `update ingest_jobs set items_processed = $2, updated_at = now() where id = $1`, id, processed)
	return err
}

func (s *Store) CompleteIngestJob(ctx context.Context, tx *sql.Tx, id string) error {
	_, err := tx.ExecContext(ctx, `update ingest_jobs set status = 'completed', items_processed = items_total, updated_at = now() where id = $1`, id)
	return err
}

func (s *Store) FailIngestJob(ctx context.Context, tx *sql.Tx, id string, message string) error {
	_, err := tx.ExecContext(ctx, `update ingest_jobs set status = 'failed', error_message = $2, updated_at = now() where id = $1`, id, nullableStringValue(message))
	return err
}

func (s *Store) InsertAnalyticsJob(ctx context.Context, tx *sql.Tx, id, workspaceID, marketplace, shopKey, reason string) error {
	_, err := tx.ExecContext(ctx, `
		insert into analytics_jobs (id, workspace_id, marketplace_code, shop_key, status, reason, created_at, updated_at)
		values ($1, $2, $3, $4, 'running', $5, now(), now())
	`, id, workspaceID, marketplace, shopKey, nullableStringValue(reason))
	return err
}

func (s *Store) CompleteAnalyticsJob(ctx context.Context, tx *sql.Tx, id string) error {
	_, err := tx.ExecContext(ctx, `update analytics_jobs set status = 'completed', updated_at = now() where id = $1`, id)
	return err
}

func (s *Store) FailAnalyticsJob(ctx context.Context, tx *sql.Tx, id, message string) error {
	_, err := tx.ExecContext(ctx, `update analytics_jobs set status = 'failed', error_message = $2, updated_at = now() where id = $1`, id, nullableStringValue(message))
	return err
}

func (s *Store) EnsureShop(ctx context.Context, tx *sql.Tx, workspaceID string, capture model.ExtensionCapture) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
		insert into shops (
			id, workspace_id, marketplace_code, shop_key, shop_id_external, slug, url, name, status, location, shop_type, created_at, updated_at
		)
		values (gen_random_uuid(), $1, $2, $3, null, $4, $5, $6, null, null, null, now(), now())
		on conflict (workspace_id, marketplace_code, shop_key)
		do update set slug = excluded.slug, url = excluded.url, name = coalesce(nullif(excluded.name, ''), shops.name), updated_at = now()
		returning id
	`, workspaceID, capture.Marketplace, capture.ShopKey, capture.ShopKey, capture.PageContext.PageURL, capture.PageContext.ShopName).Scan(&id)
	return id, err
}

func (s *Store) FindDuplicateRawCaptureID(ctx context.Context, tx *sql.Tx, workspaceID, marketplace, shopKey, requestHash, responseHash string, capturedAt time.Time) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
		select id from raw_captures
		where workspace_id = $1 and marketplace_code = $2 and shop_key = $3 and request_hash = $4 and response_hash = $5 and captured_at = $6
		limit 1
	`, workspaceID, marketplace, shopKey, requestHash, responseHash, capturedAt).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

func (s *Store) InsertRawCapture(ctx context.Context, tx *sql.Tx, record RawCaptureRecord) error {
	_, err := tx.ExecContext(ctx, `
		insert into raw_captures (
			id, workspace_id, marketplace_code, shop_key, page_url, page_type, source_channel,
			request_url, http_method, status_code, operation_name, entity_type,
			request_headers_json, response_headers_json, request_body_text, response_body_text,
			request_body_storage_path, response_body_storage_path, request_hash, response_hash, captured_at, created_at
		)
		values (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13::jsonb, $14::jsonb, $15, $16,
			$17, $18, $19, $20, $21, now()
		)
	`, record.ID, record.WorkspaceID, record.MarketplaceCode, record.ShopKey, record.PageURL, record.PageType, record.SourceChannel,
		record.RequestURL, upperOrDefault(record.HTTPMethod, "POST"), record.StatusCode, record.OperationName, record.EntityType,
		string(record.RequestHeadersJSON), string(record.ResponseHeadersJSON), record.RequestBodyText, record.ResponseBodyText,
		record.RequestBodyStoragePath, record.ResponseBodyStoragePath, record.RequestHash, record.ResponseHash, record.CapturedAt)
	return err
}

func (s *Store) InsertParsedCapture(ctx context.Context, tx *sql.Tx, record ParsedCaptureRecord) error {
	_, err := tx.ExecContext(ctx, `
		insert into parsed_captures (
			id, workspace_id, raw_capture_id, marketplace_code, shop_key, page_url, page_type,
			operation_name, entity_type, leaf_fields_json, important_fields_json, captured_at, created_at
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, $12, now())
	`, record.ID, record.WorkspaceID, record.RawCaptureID, record.MarketplaceCode, record.ShopKey, record.PageURL, record.PageType,
		record.OperationName, record.EntityType, string(record.LeafFieldsJSON), string(record.ImportantFieldsJSON), record.CapturedAt)
	return err
}

func (s *Store) UpsertSourceTemplate(ctx context.Context, tx *sql.Tx, record SourceTemplateRecord) error {
	_, err := tx.ExecContext(ctx, `
		insert into source_templates (
			id, workspace_id, marketplace_code, shop_key, template_type, fingerprint, request_url, http_method,
			headers_json, body_text, operation_name, is_active, last_seen_at, created_at, updated_at
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, true, $12, now(), now())
		on conflict (workspace_id, marketplace_code, shop_key, fingerprint)
		do update set
			template_type = excluded.template_type,
			request_url = excluded.request_url,
			http_method = excluded.http_method,
			headers_json = excluded.headers_json,
			body_text = excluded.body_text,
			operation_name = excluded.operation_name,
			is_active = true,
			last_seen_at = excluded.last_seen_at,
			updated_at = now()
	`, record.ID, record.WorkspaceID, record.MarketplaceCode, record.ShopKey, record.TemplateType, record.Fingerprint, record.RequestURL, upperOrDefault(record.HTTPMethod, "POST"), string(record.HeadersJSON), record.BodyText, record.OperationName, record.LastSeenAt)
	return err
}

func (s *Store) FindShopID(ctx context.Context, tx *sql.Tx, workspaceID, marketplace, shopKey string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `select id from shops where workspace_id = $1 and marketplace_code = $2 and shop_key = $3 limit 1`, workspaceID, marketplace, shopKey).Scan(&id)
	return id, err
}

func (s *Store) ListParsedCandidates(ctx context.Context, tx *sql.Tx, workspaceID, marketplace, shopKey string, limit int) ([]analytics.ParsedCandidate, error) {
	rows, err := tx.QueryContext(ctx, `
		select pc.id, coalesce(pc.entity_type, ''), coalesce(pc.operation_name, ''), coalesce(pc.page_url, ''), coalesce(pc.page_type, ''),
		       pc.captured_at, coalesce(rc.status_code, 0), coalesce(rc.request_url, ''),
		       coalesce(pc.important_fields_json, '{}'::jsonb), coalesce(pc.leaf_fields_json, '{}'::jsonb)
		from parsed_captures pc
		left join raw_captures rc on rc.id = pc.raw_capture_id
		where pc.workspace_id = $1 and pc.marketplace_code = $2 and pc.shop_key = $3
		order by pc.captured_at desc
		limit $4
	`, workspaceID, marketplace, shopKey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []analytics.ParsedCandidate
	for rows.Next() {
		var item analytics.ParsedCandidate
		var importantRaw, leafRaw []byte
		if err := rows.Scan(&item.ID, &item.EntityType, &item.OperationName, &item.PageURL, &item.PageType, &item.CapturedAt, &item.StatusCode, &item.RequestURL, &importantRaw, &leafRaw); err != nil {
			return nil, err
		}
		item.ImportantFields = analytics.DecodeMap(importantRaw)
		item.LeafFields = analytics.DecodeMap(leafRaw)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ReplaceLatestShopSnapshot(ctx context.Context, tx *sql.Tx, record ShopSnapshotRecord) error {
	if _, err := tx.ExecContext(ctx, `update shop_snapshots set is_latest = false where workspace_id = $1 and shop_id = $2 and is_latest = true`, record.WorkspaceID, record.ShopID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		insert into shop_snapshots (
			id, workspace_id, shop_id, captured_at, source_type, source_capture_id,
			profile_json, reputation_json, catalog_json, performance_json, promotions_json,
			operations_json, products_json, insights_json, debug_json, is_latest, created_at
		)
		values ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, $12::jsonb, $13::jsonb, $14::jsonb, $15::jsonb, true, now())
	`, record.ID, record.WorkspaceID, record.ShopID, record.CapturedAt, record.SourceType, record.SourceCaptureID,
		string(record.ProfileJSON), string(record.ReputationJSON), string(record.CatalogJSON), string(record.PerformanceJSON), string(record.PromotionsJSON), string(record.OperationsJSON), string(record.ProductsJSON), string(record.InsightsJSON), string(record.DebugJSON))
	return err
}

func (s *Store) ClearLatestProductSnapshotsForShop(ctx context.Context, tx *sql.Tx, workspaceID, shopID string) error {
	_, err := tx.ExecContext(ctx, `
		update product_snapshots
		set is_latest = false
		where workspace_id = $1 and shop_id = $2 and is_latest = true
	`, workspaceID, shopID)
	return err
}

func (s *Store) UpsertProduct(ctx context.Context, tx *sql.Tx, record ProductRecord) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
		insert into products (
			id, workspace_id, shop_id, marketplace_code, product_key, product_id_external, product_identity, url, name, created_at, updated_at
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())
		on conflict (workspace_id, marketplace_code, shop_id, product_identity)
		do update set
			product_key = coalesce(excluded.product_key, products.product_key),
			product_id_external = coalesce(excluded.product_id_external, products.product_id_external),
			url = coalesce(nullif(excluded.url, ''), products.url),
			name = coalesce(nullif(excluded.name, ''), products.name),
			updated_at = now()
		returning id
	`, record.ID, record.WorkspaceID, record.ShopID, record.MarketplaceCode, record.ProductKey, record.ProductIDExternal, record.ProductIdentity, record.URL, record.Name).Scan(&id)
	return id, err
}

func (s *Store) ReplaceLatestProductSnapshot(ctx context.Context, tx *sql.Tx, record ProductSnapshotRecord) error {
	if _, err := tx.ExecContext(ctx, `update product_snapshots set is_latest = false where workspace_id = $1 and product_id = $2 and is_latest = true`, record.WorkspaceID, record.ProductID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		insert into product_snapshots (
			id, workspace_id, shop_id, product_id, captured_at, source_type, source_capture_id,
			profile_json, pricing_json, performance_json, inventory_json, promotion_json, review_json, debug_json, is_latest, created_at
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, $12::jsonb, $13::jsonb, $14::jsonb, true, now())
	`, record.ID, record.WorkspaceID, record.ShopID, record.ProductID, record.CapturedAt, record.SourceType, record.SourceCaptureID, string(record.ProfileJSON), string(record.PricingJSON), string(record.PerformanceJSON), string(record.InventoryJSON), string(record.PromotionJSON), string(record.ReviewJSON), string(record.DebugJSON))
	return err
}

func (s *Store) GetLatestDashboard(ctx context.Context, tx *sql.Tx, workspaceID, marketplace, shopKey string) (ShopDashboardView, error) {
	var view ShopDashboardView
	err := tx.QueryRowContext(ctx, `
		select ss.captured_at,
		       coalesce(ss.profile_json, '{}'::jsonb),
		       coalesce(ss.reputation_json, '{}'::jsonb),
		       coalesce(ss.catalog_json, '{}'::jsonb),
		       coalesce(ss.performance_json, '{}'::jsonb),
		       coalesce(ss.promotions_json, '{}'::jsonb),
		       coalesce(ss.operations_json, '{}'::jsonb),
		       coalesce(ss.products_json, '[]'::jsonb),
		       coalesce(ss.insights_json, '[]'::jsonb),
		       coalesce(ss.debug_json, '{}'::jsonb)
		from shop_snapshots ss
		join shops s on s.id = ss.shop_id
		where ss.workspace_id = $1 and s.marketplace_code = $2 and s.shop_key = $3 and ss.is_latest = true
		order by ss.captured_at desc
		limit 1
	`, workspaceID, marketplace, shopKey).Scan(&view.CapturedAt, &view.ProfileJSON, &view.ReputationJSON, &view.CatalogJSON, &view.PerformanceJSON, &view.PromotionsJSON, &view.OperationsJSON, &view.ProductsJSON, &view.InsightsJSON, &view.DebugJSON)
	return view, err
}

func (s *Store) ListProducts(ctx context.Context, tx *sql.Tx, workspaceID, marketplace, shopKey string) ([]ProductSnapshotView, error) {
	rows, err := tx.QueryContext(ctx, `
		select p.id, coalesce(p.product_id_external, ''), coalesce(p.product_key, ''), coalesce(p.name, ''), coalesce(p.url, ''),
		       ps.captured_at,
		       coalesce(ps.profile_json, '{}'::jsonb),
		       coalesce(ps.pricing_json, '{}'::jsonb),
		       coalesce(ps.performance_json, '{}'::jsonb),
		       coalesce(ps.inventory_json, '{}'::jsonb),
		       coalesce(ps.promotion_json, '{}'::jsonb),
		       coalesce(ps.review_json, '{}'::jsonb),
		       coalesce(ps.debug_json, '{}'::jsonb)
		from products p
		join shops s on s.id = p.shop_id
		left join product_snapshots ps on ps.product_id = p.id and ps.is_latest = true
		where p.workspace_id = $1 and p.marketplace_code = $2 and s.shop_key = $3
		order by p.updated_at desc, p.created_at desc
	`, workspaceID, marketplace, shopKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []ProductSnapshotView
	for rows.Next() {
		var item ProductSnapshotView
		var capturedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.ProductIDExternal, &item.ProductKey, &item.Name, &item.URL, &capturedAt, &item.ProfileJSON, &item.PricingJSON, &item.PerformanceJSON, &item.InventoryJSON, &item.PromotionJSON, &item.ReviewJSON, &item.DebugJSON); err != nil {
			return nil, err
		}
		item.CapturedAt = NewNullableTime(capturedAt)
		products = append(products, item)
	}
	return products, rows.Err()
}

func (s *Store) GetJob(ctx context.Context, tx *sql.Tx, workspaceID, jobID string) (JobRecord, error) {
	var (
		rec JobRecord
		itemsTotal sql.NullInt64
		itemsProcessed sql.NullInt64
		reason sql.NullString
		errorMessage sql.NullString
	)
	err := tx.QueryRowContext(ctx, `
		select id, 'ingest' as job_type, marketplace_code, shop_key, status, items_total, items_processed, null::text as reason, error_message, created_at, updated_at
		from ingest_jobs where workspace_id = $1 and id = $2
		union all
		select id, 'analytics' as job_type, marketplace_code, shop_key, status, null::bigint, null::bigint, reason, error_message, created_at, updated_at
		from analytics_jobs where workspace_id = $1 and id = $2
		limit 1
	`, workspaceID, jobID).Scan(&rec.ID, &rec.Type, &rec.Marketplace, &rec.ShopKey, &rec.Status, &itemsTotal, &itemsProcessed, &reason, &errorMessage, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		return JobRecord{}, err
	}
	rec.ItemsTotal = PtrInt64(itemsTotal)
	rec.ItemsProcessed = PtrInt64(itemsProcessed)
	rec.Reason = PtrNullString(reason)
	rec.ErrorMessage = PtrNullString(errorMessage)
	return rec, nil
}

func nullableStringValue(value string) any {
	if value == "" { return nil }
	return value
}

func upperOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return strings.ToUpper(fmt.Sprintf("%s", value))
}
