package auth

import "testing"

func TestParseBearerToken(t *testing.T) {
	token, ok := ParseBearerToken("Bearer abc123")
	if !ok || token != "abc123" {
		t.Fatalf("expected valid bearer token")
	}
	if _, ok := ParseBearerToken(""); ok {
		t.Fatalf("expected empty header to fail")
	}
	if _, ok := ParseBearerToken("Basic abc123"); ok {
		t.Fatalf("expected non-bearer header to fail")
	}
}
