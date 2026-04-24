package workspace

import (
	"net/http/httptest"
	"testing"
)

func TestResolveSlugPrefersHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/?workspace_slug=query-workspace", nil)
	r.Header.Set(HeaderWorkspaceSlug, "header-workspace")
	slug, err := ResolveSlug(r, "header-workspace")
	if err != nil {
		t.Fatalf("ResolveSlug() error = %v", err)
	}
	if slug != "header-workspace" {
		t.Fatalf("expected header workspace slug, got %q", slug)
	}
}

func TestResolveSlugMismatch(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set(HeaderWorkspaceSlug, "header-workspace")
	if _, err := ResolveSlug(r, "body-workspace"); err == nil {
		t.Fatalf("expected mismatch error")
	}
}
