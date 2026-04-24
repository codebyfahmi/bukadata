package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"bukadata/internal/model"
)

const ParsedCaptureLimit = 200

type ParsedCandidate struct {
	ID              string
	EntityType      string
	OperationName   string
	PageURL         string
	PageType        string
	CapturedAt      time.Time
	StatusCode      int
	RequestURL      string
	ImportantFields map[string]any
	LeafFields      map[string]any
}

type ElectedCapture struct {
	Candidate ParsedCandidate
	Score     int
}

type ProductRecord struct {
	ProductIDExternal string
	ProductKey        string
	Name              string
	URL               string
	Profile           map[string]any
	Pricing           map[string]any
	Performance       map[string]any
	Inventory         map[string]any
	Promotion         map[string]any
	Review            map[string]any
	Debug             map[string]any
}

type ShopSnapshotData struct {
	CapturedAt     time.Time
	Profile        map[string]any
	Reputation     map[string]any
	Catalog        map[string]any
	Performance    map[string]any
	Promotions     map[string]any
	Operations     map[string]any
	Products       []ProductRecord
	ProductsTab    []any
	Insights       []any
	Debug          map[string]any
	SourceType     string
	SourceCaptureID *string
}

func BuildShopSnapshot(marketplace, shopKey, reason string, candidates []ParsedCandidate, now time.Time) ShopSnapshotData {
	selected := ElectCandidates(candidates)
	products := CollectProductRecords(selected["products"], candidates)

	profile := MapOrEmpty(selected["profile"])
	reputation := MapOrEmpty(selected["reputation"])
	catalog := MapOrEmpty(selected["catalog"])
	performance := MapOrEmpty(selected["performance"])
	promotions := MapOrEmpty(selected["promotions"])
	operations := MapOrEmpty(selected["operations"])
	insights := BuildInsights(profile, reputation, catalog, performance, products)

	productsTab := make([]any, 0, len(products))
	for _, p := range products {
		productsTab = append(productsTab, p.ToEnvelope())
	}

	debug := map[string]any{
		"marketplace":      marketplace,
		"shop_key":         shopKey,
		"reason":           reason,
		"selected_sources": BuildSelectedSourceDebug(selected),
		"parsed_count":     len(candidates),
		"generated_at":     now,
	}

	return ShopSnapshotData{
		CapturedAt:      now,
		Profile:         profile,
		Reputation:      reputation,
		Catalog:         catalog,
		Performance:     performance,
		Promotions:      promotions,
		Operations:      operations,
		Products:        products,
		ProductsTab:     productsTab,
		Insights:        insights,
		Debug:           debug,
		SourceType:      "server_recompute",
		SourceCaptureID: nil,
	}
}

func ElectCandidates(candidates []ParsedCandidate) map[string]ElectedCapture {
	result := map[string]ElectedCapture{}
	for _, candidate := range candidates {
		tab := NormalizeEntityToTab(candidate.EntityType, candidate.ImportantFields)
		score := SourceScore(candidate)
		current, ok := result[tab]
		if !ok || score > current.Score || (score == current.Score && candidate.CapturedAt.After(current.Candidate.CapturedAt)) {
			result[tab] = ElectedCapture{Candidate: candidate, Score: score}
		}
	}
	return result
}

func SourceScore(candidate ParsedCandidate) int {
	score := 0
	if strings.Contains(strings.ToLower(candidate.RequestURL), "graphql") {
		score += 30
	}
	if candidate.StatusCode == http.StatusOK {
		score += 20
	}
	if strings.TrimSpace(candidate.OperationName) != "" {
		score += 10
	}
	score += MapSize(candidate.ImportantFields) * 2
	score += MapSize(candidate.LeafFields)
	return score
}

func NormalizeEntityToTab(entityType string, importantFields map[string]any) string {
	normalized := strings.ToLower(strings.TrimSpace(entityType))
	switch normalized {
	case "profile", "shop":
		return "profile"
	case "reputation":
		return "reputation"
	case "catalog":
		return "catalog"
	case "performance":
		return "performance"
	case "promotion", "promotions":
		return "promotions"
	case "operation", "operations":
		return "operations"
	case "product", "products", "product_list":
		return "products"
	}
	if _, ok := importantFields["products"]; ok {
		return "products"
	}
	return "profile"
}

func (p ProductRecord) ToEnvelope() model.ProductEnvelope {
	return model.ProductEnvelope{
		ProductIDExternal: p.ProductIDExternal,
		ProductKey:        p.ProductKey,
		Name:              p.Name,
		URL:               p.URL,
		Profile:           ensureMap(p.Profile),
		Pricing:           ensureMap(p.Pricing),
		Performance:       ensureMap(p.Performance),
		Inventory:         ensureMap(p.Inventory),
		Promotion:         ensureMap(p.Promotion),
		Review:            ensureMap(p.Review),
		Debug:             ensureMap(p.Debug),
	}
}

func CollectProductRecords(productElection ElectedCapture, candidates []ParsedCandidate) []ProductRecord {
	records := map[string]ProductRecord{}
	var sources []ParsedCandidate
	if productElection.Candidate.ID != "" {
		sources = append(sources, productElection.Candidate)
	}
	for _, candidate := range candidates {
		if NormalizeEntityToTab(candidate.EntityType, candidate.ImportantFields) == "products" {
			sources = append(sources, candidate)
		}
	}

	for _, candidate := range sources {
		for _, object := range ExtractProductObjects(candidate.ImportantFields) {
			record := ProductRecord{
				ProductIDExternal: FirstString(object, "product_id_external", "productId", "product_id", "id"),
				ProductKey:        FirstString(object, "product_key", "slug", "key"),
				Name:              FirstString(object, "name", "title"),
				URL:               FirstString(object, "url", "link"),
				Profile:           object,
				Pricing:           PickMap(object, "pricing", "price", "prices"),
				Performance:       PickMap(object, "performance", "stats", "analytics"),
				Inventory:         PickMap(object, "inventory", "stock"),
				Promotion:         PickMap(object, "promotion", "promotions"),
				Review:            PickMap(object, "review", "reviews", "rating"),
				Debug: map[string]any{
					"source_entity_type": candidate.EntityType,
					"captured_at":        candidate.CapturedAt,
				},
			}
			key := FirstNonEmpty(record.ProductIDExternal, record.ProductKey, record.URL, record.Name)
			if key == "" {
				continue
			}
			records[key] = MergeProductRecord(records[key], record)
		}
	}

	list := make([]ProductRecord, 0, len(records))
	for _, record := range records {
		list = append(list, record)
	}
	sort.Slice(list, func(i, j int) bool {
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})
	return list
}

func BuildInsights(profile, reputation, catalog, performance map[string]any, products []ProductRecord) []any {
	insights := []any{map[string]any{
		"code":     "product_count",
		"label":    "Detected products",
		"value":    len(products),
		"severity": "info",
	}}
	if followers := FirstNumeric(profile, "followers", "follower_count"); followers > 0 {
		insights = append(insights, map[string]any{"code": "followers", "label": "Follower base", "value": followers, "severity": "info"})
	}
	if rating := FirstNumeric(reputation, "rating", "shop_rating"); rating > 0 {
		insights = append(insights, map[string]any{"code": "shop_rating", "label": "Shop rating", "value": rating, "severity": "info"})
	}
	if activeProducts := FirstNumeric(catalog, "total_products", "product_count"); activeProducts > 0 {
		insights = append(insights, map[string]any{"code": "catalog_size", "label": "Catalog size", "value": activeProducts, "severity": "info"})
	}
	if sales := FirstNumeric(performance, "sales", "total_sales"); sales > 0 {
		insights = append(insights, map[string]any{"code": "sales_signal", "label": "Sales signal", "value": sales, "severity": "info"})
	}
	return insights
}

func BuildSelectedSourceDebug(selected map[string]ElectedCapture) map[string]any {
	debug := map[string]any{}
	for tab, entry := range selected {
		debug[tab] = map[string]any{
			"capture_id":     entry.Candidate.ID,
			"entity_type":    entry.Candidate.EntityType,
			"operation_name": entry.Candidate.OperationName,
			"captured_at":    entry.Candidate.CapturedAt,
			"score":          entry.Score,
			"request_url":    entry.Candidate.RequestURL,
		}
	}
	return debug
}

func ExtractProductObjects(payload map[string]any) []map[string]any {
	var records []map[string]any
	var walk func(any)
	walk = func(node any) {
		switch value := node.(type) {
		case []any:
			for _, item := range value {
				walk(item)
			}
		case map[string]any:
			if LooksLikeProduct(value) {
				records = append(records, value)
			}
			for _, nested := range value {
				walk(nested)
			}
		}
	}
	walk(payload)
	return records
}

func LooksLikeProduct(m map[string]any) bool {
	return FirstString(m, "product_id_external", "productId", "product_id", "slug", "url", "name", "title") != ""
}

func MergeProductRecord(base, incoming ProductRecord) ProductRecord {
	if base.ProductIDExternal == "" {
		base.ProductIDExternal = incoming.ProductIDExternal
	}
	if base.ProductKey == "" {
		base.ProductKey = incoming.ProductKey
	}
	if base.Name == "" {
		base.Name = incoming.Name
	}
	if base.URL == "" {
		base.URL = incoming.URL
	}
	base.Profile = MergeMaps(base.Profile, incoming.Profile)
	base.Pricing = MergeMaps(base.Pricing, incoming.Pricing)
	base.Performance = MergeMaps(base.Performance, incoming.Performance)
	base.Inventory = MergeMaps(base.Inventory, incoming.Inventory)
	base.Promotion = MergeMaps(base.Promotion, incoming.Promotion)
	base.Review = MergeMaps(base.Review, incoming.Review)
	base.Debug = MergeMaps(base.Debug, incoming.Debug)
	return base
}

func MergeMaps(a, b map[string]any) map[string]any {
	if a == nil {
		a = map[string]any{}
	}
	for key, value := range b {
		if _, exists := a[key]; !exists {
			a[key] = value
		}
	}
	return a
}

func PickMap(m map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if child, ok := m[key].(map[string]any); ok {
			return child
		}
	}
	result := map[string]any{}
	for _, key := range keys {
		if value, ok := m[key]; ok {
			result[key] = value
		}
	}
	return result
}

func MapOrEmpty(entry ElectedCapture) map[string]any {
	if entry.Candidate.ID == "" {
		return map[string]any{}
	}
	if len(entry.Candidate.ImportantFields) > 0 {
		return ensureMap(entry.Candidate.ImportantFields)
	}
	if len(entry.Candidate.LeafFields) > 0 {
		return ensureMap(entry.Candidate.LeafFields)
	}
	return map[string]any{}
}

func EmptyDashboard(marketplace, shopKey string) model.ShopEnvelope {
	return model.ShopEnvelope{
		Marketplace: marketplace,
		ShopKey:     shopKey,
		GeneratedAt: time.Now().UTC(),
		Source:      "server",
		Tabs:        model.EmptyDashboardTabs(),
	}
}

func ProductIdentity(p ProductRecord) string {
	if identity := FirstNonEmpty(p.ProductIDExternal, p.ProductKey, p.URL, p.Name); identity != "" {
		return identity
	}
	raw, _ := json.Marshal(p.Profile)
	return SHA256Hex(string(raw))
}

func DecodeMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func MapSize(m map[string]any) int {
	size := 0
	var walk func(any)
	walk = func(node any) {
		switch value := node.(type) {
		case map[string]any:
			for _, child := range value {
				size++
				walk(child)
			}
		case []any:
			for _, child := range value {
				size++
				walk(child)
			}
		}
	}
	walk(m)
	return size
}

func SHA256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func FirstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := m[key]; ok {
			switch value := raw.(type) {
			case string:
				if strings.TrimSpace(value) != "" {
					return value
				}
			case float64:
				if value != 0 {
					return fmt.Sprintf("%.0f", value)
				}
			case int:
				if value != 0 {
					return fmt.Sprintf("%d", value)
				}
			case int64:
				if value != 0 {
					return fmt.Sprintf("%d", value)
				}
			}
		}
	}
	return ""
}

func FirstNumeric(m map[string]any, keys ...string) float64 {
	for _, key := range keys {
		if raw, ok := m[key]; ok {
			switch value := raw.(type) {
			case float64:
				return value
			case int:
				return float64(value)
			case int64:
				return float64(value)
			}
		}
	}
	return 0
}

func ensureMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
