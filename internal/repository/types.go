package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"bukadata/internal/analytics"
	"bukadata/internal/model"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *sql.DB { return s.db }

type Workspace struct {
	ID   string
	Slug string
	Name string
}

type JobRecord struct {
	ID             string
	Type           string
	Marketplace    string
	ShopKey        string
	Status         string
	ItemsTotal     *int64
	ItemsProcessed *int64
	Reason         *string
	ErrorMessage   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RawCaptureRecord struct {
	ID                    string
	WorkspaceID           string
	MarketplaceCode       string
	ShopKey               string
	PageURL               string
	PageType              string
	SourceChannel         string
	RequestURL            string
	HTTPMethod            string
	StatusCode            int
	OperationName         *string
	EntityType            *string
	RequestHeadersJSON    []byte
	ResponseHeadersJSON   []byte
	RequestBodyText       *string
	ResponseBodyText      *string
	RequestBodyStoragePath *string
	ResponseBodyStoragePath *string
	RequestHash           string
	ResponseHash          string
	CapturedAt            time.Time
}

type ParsedCaptureRecord struct {
	ID                 string
	WorkspaceID        string
	RawCaptureID       *string
	MarketplaceCode    string
	ShopKey            string
	PageURL            string
	PageType           string
	OperationName      *string
	EntityType         string
	LeafFieldsJSON     []byte
	ImportantFieldsJSON []byte
	CapturedAt         time.Time
}

type SourceTemplateRecord struct {
	ID              string
	WorkspaceID     string
	MarketplaceCode string
	ShopKey         string
	TemplateType    string
	Fingerprint     string
	RequestURL      string
	HTTPMethod      string
	HeadersJSON     []byte
	BodyText        string
	OperationName   *string
	LastSeenAt      time.Time
}

type ShopSnapshotRecord struct {
	ID             string
	WorkspaceID    string
	ShopID         string
	CapturedAt     time.Time
	SourceType     string
	SourceCaptureID *string
	ProfileJSON    []byte
	ReputationJSON []byte
	CatalogJSON    []byte
	PerformanceJSON []byte
	PromotionsJSON []byte
	OperationsJSON []byte
	ProductsJSON   []byte
	InsightsJSON   []byte
	DebugJSON      []byte
}

type ProductRecord struct {
	ID                string
	WorkspaceID       string
	ShopID            string
	MarketplaceCode   string
	ProductKey        *string
	ProductIDExternal *string
	ProductIdentity   string
	URL               *string
	Name              *string
}

type ProductSnapshotRecord struct {
	ID             string
	WorkspaceID    string
	ShopID         string
	ProductID      string
	CapturedAt     time.Time
	SourceType     string
	SourceCaptureID *string
	ProfileJSON    []byte
	PricingJSON    []byte
	PerformanceJSON []byte
	InventoryJSON  []byte
	PromotionJSON  []byte
	ReviewJSON     []byte
	DebugJSON      []byte
}

type ProductSnapshotView struct {
	ID                string
	ProductIDExternal string
	ProductKey        string
	Name              string
	URL               string
	CapturedAt        *time.Time
	ProfileJSON       []byte
	PricingJSON       []byte
	PerformanceJSON   []byte
	InventoryJSON     []byte
	PromotionJSON     []byte
	ReviewJSON        []byte
	DebugJSON         []byte
}

type ShopDashboardView struct {
	CapturedAt       time.Time
	ProfileJSON      []byte
	ReputationJSON   []byte
	CatalogJSON      []byte
	PerformanceJSON  []byte
	PromotionsJSON   []byte
	OperationsJSON   []byte
	ProductsJSON     []byte
	InsightsJSON     []byte
	DebugJSON        []byte
}

type NullableTime struct {
	Time  time.Time
	Valid bool
}

func NewNullableTime(nt sql.NullTime) *time.Time {
	if !nt.Valid { return nil }
	return &nt.Time
}

func PtrString(value string) *string {
	if value == "" { return nil }
	return &value
}

func PtrInt64(value sql.NullInt64) *int64 {
	if !value.Valid { return nil }
	return &value.Int64
}

func PtrNullString(value sql.NullString) *string {
	if !value.Valid { return nil }
	return &value.String
}

func JSONBytes(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func NewRawCaptureRecord(workspaceID string, capture model.ExtensionCapture, requestHash, responseHash string, requestInline, responseInline, requestStoragePath, responseStoragePath *string) RawCaptureRecord {
	requestHeaders := JSONBytes(capture.RawCapture.RequestHeaders)
	responseHeaders := JSONBytes(capture.RawCapture.ResponseHeaders)
	return RawCaptureRecord{
		ID:                     uuid.NewString(),
		WorkspaceID:            workspaceID,
		MarketplaceCode:        capture.Marketplace,
		ShopKey:                capture.ShopKey,
		PageURL:                capture.PageContext.PageURL,
		PageType:               capture.PageContext.PageType,
		SourceChannel:          capture.RawCapture.SourceChannel,
		RequestURL:             capture.RawCapture.URL,
		HTTPMethod:             capture.RawCapture.Method,
		StatusCode:             capture.RawCapture.Status,
		OperationName:          PtrString(capture.RawCapture.OperationName),
		EntityType:             PtrString(capture.RawCapture.EntityType),
		RequestHeadersJSON:     requestHeaders,
		ResponseHeadersJSON:    responseHeaders,
		RequestBodyText:        requestInline,
		ResponseBodyText:       responseInline,
		RequestBodyStoragePath: requestStoragePath,
		ResponseBodyStoragePath: responseStoragePath,
		RequestHash:            requestHash,
		ResponseHash:           responseHash,
		CapturedAt:             capture.CapturedAt,
	}
}

func NewParsedCaptureRecord(workspaceID, rawCaptureID string, capture model.ExtensionCapture) ParsedCaptureRecord {
	leaf := JSONBytes(capture.ParsedSummary.LeafFields)
	important := JSONBytes(capture.ParsedSummary.ImportantFields)
	return ParsedCaptureRecord{
		ID:                  uuid.NewString(),
		WorkspaceID:         workspaceID,
		RawCaptureID:        PtrString(rawCaptureID),
		MarketplaceCode:     capture.Marketplace,
		ShopKey:             capture.ShopKey,
		PageURL:             capture.PageContext.PageURL,
		PageType:            capture.PageContext.PageType,
		OperationName:       PtrString(firstNonEmpty(capture.ParsedSummary.OperationName, capture.RawCapture.OperationName)),
		EntityType:          firstNonEmpty(capture.ParsedSummary.EntityType, capture.RawCapture.EntityType, "shop"),
		LeafFieldsJSON:      leaf,
		ImportantFieldsJSON: important,
		CapturedAt:          capture.CapturedAt,
	}
}

func NewSourceTemplateRecord(workspaceID string, capture model.ExtensionCapture) SourceTemplateRecord {
	return SourceTemplateRecord{
		ID:              uuid.NewString(),
		WorkspaceID:     workspaceID,
		MarketplaceCode: capture.Marketplace,
		ShopKey:         capture.ShopKey,
		TemplateType:    "replay",
		Fingerprint:     capture.ReplayTemplate.Fingerprint,
		RequestURL:      capture.ReplayTemplate.URL,
		HTTPMethod:      capture.ReplayTemplate.Method,
		HeadersJSON:     JSONBytes(capture.ReplayTemplate.Headers),
		BodyText:        capture.ReplayTemplate.Body,
		OperationName:   PtrString(capture.ReplayTemplate.OperationName),
		LastSeenAt:      nonZeroTime(capture.ReplayTemplate.CreatedAt, capture.CapturedAt),
	}
}

func NewShopSnapshotRecord(workspaceID, shopID string, data analytics.ShopSnapshotData) ShopSnapshotRecord {
	return ShopSnapshotRecord{
		ID:              uuid.NewString(),
		WorkspaceID:     workspaceID,
		ShopID:          shopID,
		CapturedAt:      data.CapturedAt,
		SourceType:      data.SourceType,
		SourceCaptureID: data.SourceCaptureID,
		ProfileJSON:     JSONBytes(data.Profile),
		ReputationJSON:  JSONBytes(data.Reputation),
		CatalogJSON:     JSONBytes(data.Catalog),
		PerformanceJSON: JSONBytes(data.Performance),
		PromotionsJSON:  JSONBytes(data.Promotions),
		OperationsJSON:  JSONBytes(data.Operations),
		ProductsJSON:    JSONBytes(data.ProductsTab),
		InsightsJSON:    JSONBytes(data.Insights),
		DebugJSON:       JSONBytes(data.Debug),
	}
}

func NewProductRecord(workspaceID, shopID, marketplace string, product analytics.ProductRecord) ProductRecord {
	return ProductRecord{
		ID:                uuid.NewString(),
		WorkspaceID:       workspaceID,
		ShopID:            shopID,
		MarketplaceCode:   marketplace,
		ProductKey:        PtrString(product.ProductKey),
		ProductIDExternal: PtrString(product.ProductIDExternal),
		ProductIdentity:   analytics.ProductIdentity(product),
		URL:               PtrString(product.URL),
		Name:              PtrString(product.Name),
	}
}

func NewProductSnapshotRecord(workspaceID, shopID, productID string, capturedAt time.Time, product analytics.ProductRecord) ProductSnapshotRecord {
	return ProductSnapshotRecord{
		ID:              uuid.NewString(),
		WorkspaceID:     workspaceID,
		ShopID:          shopID,
		ProductID:       productID,
		CapturedAt:      capturedAt,
		SourceType:      "server_recompute",
		ProfileJSON:     JSONBytes(product.Profile),
		PricingJSON:     JSONBytes(product.Pricing),
		PerformanceJSON: JSONBytes(product.Performance),
		InventoryJSON:   JSONBytes(product.Inventory),
		PromotionJSON:   JSONBytes(product.Promotion),
		ReviewJSON:      JSONBytes(product.Review),
		DebugJSON:       JSONBytes(product.Debug),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func nonZeroTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}
