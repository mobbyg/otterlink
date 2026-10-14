package news

import "testing"

func TestValidateSource(t *testing.T) {
	valid := []struct{name, rawURL, category string; refresh int}{
		{"Example", "https://example.com/feed.xml", "Local", 30},
	}
	for _, tc := range valid {
		if err := validateSource(tc.name, tc.rawURL, tc.category, tc.refresh); err != nil {
			t.Fatalf("valid source rejected: %v", err)
		}
	}
	invalid := []struct{name, rawURL, category string; refresh int}{
		{"", "https://example.com/feed.xml", "Local", 30},
		{"Example", "ftp://example.com/feed.xml", "Local", 30},
		{"Example", "https://example.com/feed.xml", "", 30},
		{"Example", "https://example.com/feed.xml", "Local", 4},
	}
	for _, tc := range invalid {
		if err := validateSource(tc.name, tc.rawURL, tc.category, tc.refresh); err == nil {
			t.Fatalf("invalid source accepted: %+v", tc)
		}
	}
}
