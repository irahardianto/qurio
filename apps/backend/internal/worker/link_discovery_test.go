package worker

import (
	"testing"
)

func TestDiscoverLinks(t *testing.T) {
	links := []string{
		"https://example.com/page1",
		"https://example.com/page2#frag",
		"https://other.com/page3",
		"https://example.com/exclude",
	}
	exclusions := []string{".*exclude.*"}
	
	pages := DiscoverLinks("src1", "example.com", links, 0, 2, exclusions)
	
	if len(pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(pages))
	}
	if pages[0].URL != "https://example.com/page1" {
		t.Errorf("expected page1, got %s", pages[0].URL)
	}
    // The second page should be page2 normalized (no frag)
    if pages[1].URL != "https://example.com/page2" {
        t.Errorf("expected page2, got %s", pages[1].URL)
    }
}
