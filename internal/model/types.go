package model

import "time"

type IngestBatchRequest struct {
	WorkspaceSlug string            `json:"workspace_slug"`
	Marketplace   string            `json:"marketplace"`
	Items         []IngestBatchItem `json:"items"`
}

type IngestBatchItem struct {
	Capture ExtensionCapture `json:"capture"`
}

type ExtensionCapture struct {
	ID             string         `json:"id"`
	CapturedAt     time.Time      `json:"capturedAt"`
	Marketplace    string         `json:"marketplace"`
	ShopKey        string         `json:"shopKey"`
	PageContext    PageContext    `json:"pageContext"`
	RawCapture     RawCapture     `json:"rawCapture"`
	ParsedSummary  ParsedSummary  `json:"parsedSummary"`
	ReplayTemplate ReplayTemplate `json:"replayTemplate"`
}

type PageContext struct {
	PageURL      string `json:"pageUrl"`
	PageTitle    string `json:"pageTitle"`
	ShopKey      string `json:"shopKey"`
	ShopName     string `json:"shopName"`
	PageType     string `json:"pageType"`
	CapturedFrom string `json:"capturedFrom"`
	Marketplace  string `json:"marketplace"`
}

type RawCapture struct {
	URL             string         `json:"url"`
	Method          string         `json:"method"`
	RequestHeaders  map[string]any `json:"requestHeaders"`
	ResponseHeaders map[string]any `json:"responseHeaders"`
	RequestBody     string         `json:"requestBody"`
	ResponseBody    string         `json:"responseBody"`
	Status          int            `json:"status"`
	SourceChannel   string         `json:"sourceChannel"`
	OperationName   string         `json:"operationName"`
	EntityType      string         `json:"entityType"`
}

type ParsedSummary struct {
	OperationName   string         `json:"operationName"`
	EntityType      string         `json:"entityType"`
	RequestJSON     map[string]any `json:"requestJson"`
	ResponseJSON    map[string]any `json:"responseJson"`
	LeafFields      map[string]any `json:"leafFields"`
	ImportantFields map[string]any `json:"importantFields"`
	PageContext     PageContext    `json:"pageContext"`
}

type ReplayTemplate struct {
	Fingerprint   string         `json:"fingerprint"`
	ShopKey       string         `json:"shopKey"`
	URL           string         `json:"url"`
	Method        string         `json:"method"`
	Headers       map[string]any `json:"headers"`
	Body          string         `json:"body"`
	OperationName string         `json:"operationName"`
	CreatedAt     time.Time      `json:"createdAt"`
}

type IngestBatchResponse struct {
	Accepted   int                     `json:"accepted"`
	Recomputed bool                    `json:"recomputed"`
	Dashboards map[string]ShopEnvelope `json:"dashboards"`
	JobID      string                  `json:"job_id,omitempty"`
}

type ShopEnvelope struct {
	Marketplace string         `json:"marketplace"`
	ShopKey     string         `json:"shop_key"`
	GeneratedAt time.Time      `json:"generated_at"`
	Source      string         `json:"source"`
	Tabs        DashboardTabs  `json:"tabs"`
}

type DashboardTabs struct {
	Profile     map[string]any `json:"profile"`
	Reputation  map[string]any `json:"reputation"`
	Catalog     map[string]any `json:"catalog"`
	Performance map[string]any `json:"performance"`
	Promotions  map[string]any `json:"promotions"`
	Operations  map[string]any `json:"operations"`
	Products    []any          `json:"products"`
	Insights    []any          `json:"insights"`
	Debug       map[string]any `json:"debug"`
}

type ShopDashboardResponse struct {
	Dashboard ShopEnvelope `json:"dashboard"`
}

type RefreshAnalyticsRequest struct {
	WorkspaceSlug string `json:"workspace_slug"`
	Marketplace   string `json:"marketplace"`
	ShopKey       string `json:"shop_key"`
	Reason        string `json:"reason"`
}

type ProductEnvelope struct {
	ID                string         `json:"id,omitempty"`
	ProductIDExternal string         `json:"product_id_external,omitempty"`
	ProductKey        string         `json:"product_key,omitempty"`
	Name              string         `json:"name,omitempty"`
	URL               string         `json:"url,omitempty"`
	CapturedAt        *time.Time     `json:"captured_at,omitempty"`
	Profile           map[string]any `json:"profile"`
	Pricing           map[string]any `json:"pricing"`
	Performance       map[string]any `json:"performance"`
	Inventory         map[string]any `json:"inventory"`
	Promotion         map[string]any `json:"promotion"`
	Review            map[string]any `json:"review"`
	Debug             map[string]any `json:"debug,omitempty"`
}

type ProductsResponse struct {
	Products []ProductEnvelope `json:"products"`
}

type JobEnvelope struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Marketplace    string     `json:"marketplace"`
	ShopKey        string     `json:"shop_key"`
	Status         string     `json:"status"`
	ItemsTotal     *int64     `json:"items_total,omitempty"`
	ItemsProcessed *int64     `json:"items_processed,omitempty"`
	Reason         *string    `json:"reason,omitempty"`
	ErrorMessage   *string    `json:"error_message,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type JobResponse struct {
	Job JobEnvelope `json:"job"`
}

func EmptyDashboardTabs() DashboardTabs {
	return DashboardTabs{
		Profile:     map[string]any{},
		Reputation:  map[string]any{},
		Catalog:     map[string]any{},
		Performance: map[string]any{},
		Promotions:  map[string]any{},
		Operations:  map[string]any{},
		Products:    []any{},
		Insights:    []any{},
		Debug:       map[string]any{},
	}
}
