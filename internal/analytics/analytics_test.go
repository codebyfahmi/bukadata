package analytics

import (
	"testing"
	"time"
)

func TestSourceScorePrefersGraphQLAndHTTP200(t *testing.T) {
	a := ParsedCandidate{RequestURL: "https://example.com/rest", StatusCode: 200, OperationName: "GetShop", ImportantFields: map[string]any{"a": 1}}
	b := ParsedCandidate{RequestURL: "https://example.com/graphql", StatusCode: 200, OperationName: "GetShop", ImportantFields: map[string]any{"a": 1}}
	if SourceScore(b) <= SourceScore(a) {
		t.Fatalf("expected graphql candidate score to be higher")
	}
}

func TestNormalizeEntityToTab(t *testing.T) {
	cases := map[string]string{
		"shop":         "profile",
		"profile":      "profile",
		"reputation":   "reputation",
		"catalog":      "catalog",
		"performance":  "performance",
		"promotion":    "promotions",
		"operations":   "operations",
		"product_list": "products",
	}
	for input, expected := range cases {
		if got := NormalizeEntityToTab(input, nil); got != expected {
			t.Fatalf("NormalizeEntityToTab(%q) = %q want %q", input, got, expected)
		}
	}
}

func TestProductIdentityFallback(t *testing.T) {
	p := ProductRecord{Profile: map[string]any{"name": "Sample"}}
	if ProductIdentity(p) == "" {
		t.Fatalf("expected product identity fallback hash")
	}
}

func TestCollectProductRecords(t *testing.T) {
	candidates := []ParsedCandidate{{
		ID: "1", EntityType: "products", CapturedAt: time.Now().UTC(), ImportantFields: map[string]any{
			"items": []any{map[string]any{"productId": "123", "name": "Alpha", "pricing": map[string]any{"price": 10}}},
		},
	}}
	products := CollectProductRecords(ElectedCapture{}, candidates)
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if products[0].ProductIDExternal != "123" {
		t.Fatalf("unexpected product identity: %#v", products[0])
	}
}

func TestEmptyDashboardTabsStable(t *testing.T) {
	d := EmptyDashboard("tokopedia", "shop")
	if d.Tabs.Products == nil || d.Tabs.Insights == nil || d.Tabs.Profile == nil || d.Tabs.Debug == nil {
		t.Fatalf("expected stable non-nil dashboard tabs")
	}
}
