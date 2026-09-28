package news

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchItemsRSS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Example News</title>
<item><title> First story </title><link>/story/1</link><guid>abc-1</guid><pubDate>Mon, 21 Sep 2026 12:00:00 GMT</pubDate><description>Hello &amp; welcome</description></item>
</channel></rss>`))
	}))
	defer server.Close()

	items, err := fetchItems(context.Background(), server.Client(), server.URL+"/feed.xml")
	if err != nil { t.Fatal(err) }
	if len(items) != 1 { t.Fatalf("got %d items, want 1", len(items)) }
	if items[0].Title != "First story" || items[0].GUID != "abc-1" { t.Fatalf("unexpected item: %+v", items[0]) }
	if items[0].URL != server.URL+"/story/1" { t.Fatalf("unexpected URL: %q", items[0].URL) }
	if !strings.Contains(items[0].Summary, "Hello & welcome") { t.Fatalf("unexpected summary: %q", items[0].Summary) }
	if items[0].PublishedAt != "2026-09-21T12:00:00Z" { t.Fatalf("unexpected published time: %q", items[0].PublishedAt) }
}

func TestFetchItemsAtom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom"><title>Example</title>
<entry><title>Atom story</title><id>tag:example,2026:1</id>
<author><name>A. Writer</name></author><published>2026-09-22T14:30:00Z</published>
<link rel="alternate" href="https://example.com/story/1"/><summary>A summary</summary></entry>
</feed>`))
	}))
	defer server.Close()

	items, err := fetchItems(context.Background(), server.Client(), server.URL+"/feed.xml")
	if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].Author != "A. Writer" || items[0].URL != "https://example.com/story/1" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestSourceDue(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	source := Source{RefreshIntervalMinutes: 30, LastFetchedAt: now.Add(-31 * time.Minute).Format(time.RFC3339)}
	if !sourceDue(source, now) { t.Fatal("source should be due") }
	source.LastFetchedAt = now.Add(-29 * time.Minute).Format(time.RFC3339)
	if sourceDue(source, now) { t.Fatal("source should not be due") }
	source.LastFetchedAt = ""
	if !sourceDue(source, now) { t.Fatal("never-fetched source should be due") }
}
