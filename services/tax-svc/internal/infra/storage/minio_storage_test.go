package storage

import (
	"testing"
)

func TestParsePublicBaseURL_AllowsHostOnly(t *testing.T) {
	got, err := parsePublicBaseURL("http://localhost:9000")
	if err != nil {
		t.Fatalf("parsePublicBaseURL: %v", err)
	}
	if got == nil {
		t.Fatal("expected parsed URL, got nil")
	}
	if got.Host != "localhost:9000" {
		t.Fatalf("unexpected host: %q", got.Host)
	}
}

func TestParsePublicBaseURL_RejectsPathPrefix(t *testing.T) {
	_, err := parsePublicBaseURL("https://cdn.example.com/s3")
	if err == nil {
		t.Fatal("expected error for URL path prefix")
	}
}
