package web

import "testing"

func TestNativeNewsSlugPattern(t *testing.T) {
	valid := []string{"welcome", "update-2026", "a_1", "0-starts-with-number"}
	for _, slug := range valid {
		if !nativeNewsSlugPattern.MatchString(slug) {
			t.Errorf("expected slug %q to be valid", slug)
		}
	}
	invalid := []string{"", "-starts-with-hyphen", "_starts-with-underscore", "Uppercase", "contains space", "a/route", "a-very-long-" + string(make([]byte, 70))}
	for _, slug := range invalid {
		if nativeNewsSlugPattern.MatchString(slug) {
			t.Errorf("expected slug %q to be invalid", slug)
		}
	}
}

func TestContainsNativeArticleDestination(t *testing.T) {
	content := map[string]any{
		"elements": []any{
			map[string]any{
				"type": "button",
				"destination": map[string]any{
					"type": "native-news",
					"slug": "welcome-to-otterlink",
				},
			},
		},
	}
	if !containsNativeArticleDestination(content, "welcome-to-otterlink") {
		t.Fatal("expected nested Home destination to be detected")
	}
	if containsNativeArticleDestination(content, "other-article") {
		t.Fatal("did not expect unrelated slug to be detected")
	}
	if containsNativeArticleDestination(map[string]any{
		"destination": map[string]any{"type": "service", "service": "news"},
	}, "welcome-to-otterlink") {
		t.Fatal("service destinations must not count as native article links")
	}
}
