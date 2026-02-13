package worker

import (
	"testing"
)

func TestDiscoverLinks_Comprehensive(t *testing.T) {
	type args struct {
		sourceID     string
		host         string
		links        []string
		currentDepth int
		maxDepth     int
		exclusions   []string
	}
	tests := []struct {
		name string
		args args
		want []string // We'll just check the URLs for simplicity in this table
	}{
		{
			name: "Basic Positive",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links:    []string{"https://example.com/foo", "https://example.com/bar"},
				maxDepth: 5,
			},
			want: []string{"https://example.com/foo", "https://example.com/bar"},
		},
		{
			name: "Max Depth Reached",
			args: args{
				sourceID:     "src1",
				host:         "example.com",
				links:        []string{"https://example.com/foo"},
				currentDepth: 5,
				maxDepth:     5,
			},
			want: nil, // Should return empty/nil
		},
		{
			name: "External Host Ignored",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links:    []string{"https://google.com", "https://other.com/foo"},
				maxDepth: 5,
			},
			want: nil,
		},
		{
			name: "Subdomain Mismatch (Strict Host)",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links:    []string{"https://api.example.com/foo"},
				maxDepth: 5,
			},
			want: nil, // Current logic checks linkU.Host == host
		},
		{
			name: "Fragment Stripping",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links:    []string{"https://example.com/foo#section1", "https://example.com/bar#top"},
				maxDepth: 5,
			},
			want: []string{"https://example.com/foo", "https://example.com/bar"},
		},
		{
			name: "Exclusion Pattern",
			args: args{
				sourceID:   "src1",
				host:       "example.com",
				links:      []string{"https://example.com/valid", "https://example.com/exclude/me"},
				exclusions: []string{".*exclude.*"},
				maxDepth:   5,
			},
			want: []string{"https://example.com/valid"},
		},
		{
			name: "Deduplication Exact",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links:    []string{"https://example.com/foo", "https://example.com/foo"},
				maxDepth: 5,
			},
			want: []string{"https://example.com/foo"},
		},
		{
			name: "Deduplication via Normalization",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links:    []string{"https://example.com/foo", "https://example.com/foo#frag"},
				maxDepth: 5,
			},
			want: []string{"https://example.com/foo"},
		},
		{
			name: "Non-HTTP Schemes Ignored",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links: []string{
					"mailto:user@example.com",
					"tel:1234567890",
					"javascript:alert(1)",
					"ftp://example.com/file", // Host matches, but logic doesn't filter scheme! Wait, let's verify logic.
				},
				maxDepth: 5,
			},
			want: nil, // If logic only checks Host, FTP might pass if host matches!
		},
		{
			name: "Malformed URLs",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links:    []string{"://bad-url", "   ", ""},
				maxDepth: 5,
			},
			want: nil,
		},
		{
			name: "Unicode Characters",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links:    []string{"https://example.com/café", "https://example.com/über"},
				maxDepth: 5,
			},
			want: []string{"https://example.com/caf%C3%A9", "https://example.com/%C3%BCber"},
		},
		{
			name: "Query Parameters Preserved",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links:    []string{"https://example.com/search?q=foo"},
				maxDepth: 5,
			},
			want: []string{"https://example.com/search?q=foo"},
		},
		{
			name: "Port Mismatch",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links:    []string{"https://example.com:8080/foo"}, // Host parsed as example.com:8080
				maxDepth: 5,
			},
			want: nil, // "example.com:8080" != "example.com"
		},
		{
			name: "Escaped Spaces",
			args: args{
				sourceID: "src1",
				host:     "example.com",
				links:    []string{"https://example.com/foo%20bar"},
				maxDepth: 5,
			},
			want: []string{"https://example.com/foo%20bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DiscoverLinks(
				tt.args.sourceID,
				tt.args.host,
				tt.args.links,
				tt.args.currentDepth,
				tt.args.maxDepth,
				tt.args.exclusions,
			)

			if len(got) != len(tt.want) {
				t.Fatalf("DiscoverLinks() got %d items, want %d. Got: %+v", len(got), len(tt.want), got)
			}

			// Verify each item
			for i, wantURL := range tt.want {
				if got[i].URL != wantURL {
					t.Errorf("DiscoverLinks()[%d].URL = %v, want %v", i, got[i].URL, wantURL)
				}
				// Basic sanity checks for other fields
				if got[i].SourceID != tt.args.sourceID {
					t.Errorf("DiscoverLinks()[%d].SourceID = %v, want %v", i, got[i].SourceID, tt.args.sourceID)
				}
				if got[i].Status != "pending" {
					t.Errorf("DiscoverLinks()[%d].Status = %v, want pending", i, got[i].Status)
				}
				expectedDepth := tt.args.currentDepth + 1
				if got[i].Depth != expectedDepth {
					t.Errorf("DiscoverLinks()[%d].Depth = %v, want %v", i, got[i].Depth, expectedDepth)
				}
			}
		})
	}
}
