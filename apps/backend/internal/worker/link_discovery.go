package worker

import (
	"net/url"
	"regexp"
)

func DiscoverLinks(sourceID, host string, links []string, currentDepth, maxDepth int, exclusions []string) []PageDTO {
	if currentDepth >= maxDepth {
		return nil
	}

	var newPages []PageDTO
	seen := make(map[string]bool)

	for _, link := range links {
		// 1. External Check
		linkU, err := url.Parse(link)
		if err != nil || linkU.Host != host {
			continue
		}

		// Normalize: Strip Fragment
		linkU.Fragment = ""
		normalizedLink := linkU.String()

		// 2. Exclusion Check
		excluded := false
		for _, ex := range exclusions {
			if matched, _ := regexp.MatchString(ex, normalizedLink); matched {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		if seen[normalizedLink] {
			continue
		}
		seen[normalizedLink] = true

		newPages = append(newPages, PageDTO{
			SourceID: sourceID,
			URL:      normalizedLink,
			Status:   "pending",
			Depth:    currentDepth + 1,
		})
	}
	return newPages
}
