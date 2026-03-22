package middleware

import "testing"

func TestNegotiateEncoding(t *testing.T) {
	tests := []struct {
		accept string
		want   string
	}{
		{"", ""},
		{"gzip", "gzip"},
		{"deflate", "deflate"},
		{"zstd", "zstd"},
		{"gzip, deflate", "gzip"},
		{"gzip, zstd", "zstd"},
		{"br, gzip, deflate", "gzip"},
		{"identity", ""},
	}
	for _, tt := range tests {
		got := negotiateEncoding(tt.accept)
		if got != tt.want {
			t.Errorf("negotiateEncoding(%q) = %q, want %q", tt.accept, got, tt.want)
		}
	}
}

func TestIsCompressible(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"text/html; charset=utf-8", true},
		{"application/json", true},
		{"application/activity+json", true},
		{"text/calendar", true},
		{"image/png", false},
		{"application/octet-stream", false},
		{"image/webp", false},
	}
	for _, tt := range tests {
		got := isCompressible(tt.ct)
		if got != tt.want {
			t.Errorf("isCompressible(%q) = %v, want %v", tt.ct, got, tt.want)
		}
	}
}
