package news

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"bytes"
	"io"
	"regexp"
	"net/http"
	"net/url"
	"strings"
	"time"

	htmlpkg "golang.org/x/net/html"
)

const (
	maxFeedBytes      = 2 * 1024 * 1024
	maxItemsPerSource = 500
)

type Item struct {
	ID          int64  `json:"id"`
	SourceID    int64  `json:"source_id"`
	SourceName  string `json:"source_name"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Author      string `json:"author,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	Summary     string `json:"summary,omitempty"`
	URL         string `json:"url"`
	GUID        string `json:"guid"`
	ImageURL    string `json:"image_url,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type rssDocument struct {
	Channel struct {
		Title string `xml:"title"`
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string        `xml:"title"`
	Author      string        `xml:"author"`
	Creator     string        `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Link        string        `xml:"link"`
	GUID        string        `xml:"guid"`
	PubDate     string        `xml:"pubDate"`
	Description string        `xml:"description"`
	ContentEncoded string    `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	Enclosure   rssEnclosure  `xml:"enclosure"`
	Media       []rssMedia    `xml:"http://search.yahoo.com/mrss/ content"`
	Thumbnail   []rssMedia    `xml:"http://search.yahoo.com/mrss/ thumbnail"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type rssMedia struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Medium string `xml:"medium,attr"`
}

type atomDocument struct {
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	ID        string     `xml:"id"`
	Author    atomAuthor `xml:"author"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
	Links     []atomLink `xml:"link"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

func (s Service) RefreshDueSources(ctx context.Context, now time.Time) (int, error) {
	sources, err := s.ListSources()
	if err != nil {
		return 0, err
	}
	refreshed := 0
	for _, source := range sources {
		if !source.Enabled || !sourceDue(source, now) {
			continue
		}
		if _, err := s.FetchSource(ctx, source.ID); err != nil {
			continue
		}
		refreshed++
	}
	return refreshed, nil
}

func sourceDue(source Source, now time.Time) bool {
	if source.LastFetchedAt == "" {
		return true
	}
	last, err := time.Parse(time.RFC3339, source.LastFetchedAt)
	if err != nil {
		return true
	}
	return !last.Add(time.Duration(source.RefreshIntervalMinutes) * time.Minute).After(now)
}

func (s Service) FetchSource(ctx context.Context, id int64) (int, error) {
	source, err := s.GetSource(id)
	if err != nil {
		return 0, err
	}
	items, err := fetchItems(ctx, s.client(), source.URL)
	nowString := time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		_, _ = s.DB.Exec(`UPDATE news_sources SET last_fetched_at=?,last_error=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, nowString, err.Error(), id)
		return 0, err
	}

	inserted := 0
	for _, item := range items {
		if item.Title == "" || item.URL == "" {
			continue
		}
		if item.GUID == "" {
			item.GUID = stableGUID(item.URL, item.Title)
		}
		result, err := s.DB.Exec(`INSERT OR IGNORE INTO news_items
			(source_id,title,author,published_at,summary,url,guid,image_url,article_html)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			source.ID, item.Title, item.Author, nullableTime(item.PublishedAt),
			item.Summary, item.URL, item.GUID, item.ImageURL, item.ArticleHTML)
		if err != nil {
			return inserted, fmt.Errorf("store news item: %w", err)
		}
		if n, _ := result.RowsAffected(); n > 0 {
			inserted++
		}
		if item.ImageURL != "" {
			_, _ = s.DB.Exec(`UPDATE news_items SET image_url=? WHERE source_id=? AND guid=? AND (image_url IS NULL OR image_url='')`,
				item.ImageURL, source.ID, item.GUID)
		}
	}

	if _, err := s.DB.Exec(`DELETE FROM news_items
		WHERE source_id=? AND id NOT IN (
			SELECT id FROM news_items WHERE source_id=? ORDER BY CASE WHEN published_at IS NULL OR published_at='' THEN 1 ELSE 0 END, datetime(published_at) DESC, id DESC LIMIT ?
		)`, source.ID, source.ID, maxItemsPerSource); err != nil {
		return inserted, fmt.Errorf("apply news retention: %w", err)
	}
	if _, err := s.DB.Exec(`UPDATE news_sources SET last_fetched_at=?,last_success_at=?,last_error='',updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		nowString, nowString, id); err != nil {
		return inserted, fmt.Errorf("update news source status: %w", err)
	}
	return inserted, nil
}

func (s Service) ListItems(limit int, category string, sourceID int64) ([]Item, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	category = strings.TrimSpace(category)

	query := `SELECT i.id,i.source_id,s.name,s.category,i.title,i.author,
		COALESCE(i.published_at,''),i.summary,i.url,i.guid,i.image_url,COALESCE(i.article_html,''),i.created_at
		FROM news_items i JOIN news_sources s ON s.id=i.source_id
		WHERE s.enabled=1`
	args := []any{}
	if category != "" {
		query += ` AND s.category=?`
		args = append(args, category)
	}
	if sourceID > 0 {
		query += ` AND s.id=?`
		args = append(args, sourceID)
	}
	query += ` ORDER BY CASE WHEN i.published_at IS NULL OR i.published_at='' THEN 1 ELSE 0 END, datetime(i.published_at) DESC,i.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Item, 0)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID,&item.SourceID,&item.SourceName,&item.Category,
			&item.Title,&item.Author,&item.PublishedAt,&item.Summary,&item.URL,&item.GUID,
			&item.ImageURL,&item.ArticleHTML,&item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}


func sanitizeArticleHTML(value, base string) string {
	value = strings.TrimSpace(value)
	if value == "" { return "" }
	value = replaceYouTubeEmbeds(value)
	fragments, err := htmlpkg.ParseFragment(strings.NewReader(value), &htmlpkg.Node{Type: htmlpkg.ElementNode, Data: "div"})
	if err != nil { return "" }
	root := &htmlpkg.Node{Type: htmlpkg.ElementNode, Data: "div"}
	for _, fragment := range fragments { root.AppendChild(fragment) }
	sanitizeHTMLChildren(root, base)
	var out bytes.Buffer
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if err := htmlpkg.Render(&out, child); err != nil { return "" }
	}
	return strings.TrimSpace(out.String())
}

var allowedArticleTags = map[string]bool{
	"p": true, "br": true, "strong": true, "em": true, "b": true, "i": true,
	"u": true, "h1": true, "h2": true, "h3": true, "h4": true,
	"ul": true, "ol": true, "li": true, "blockquote": true, "a": true,
	"img": true, "figure": true, "figcaption": true, "hr": true,
	"div": true, "span": true,
}

var blockedArticleTags = map[string]bool{
	"script": true, "style": true, "noscript": true, "iframe": true,
	"object": true, "embed": true, "form": true, "canvas": true,
}

func sanitizeHTMLChildren(parent *htmlpkg.Node, base string) {
	for child := parent.FirstChild; child != nil; {
		next := child.NextSibling
		if child.Type == htmlpkg.ElementNode {
			tag := strings.ToLower(child.Data)
			if blockedArticleTags[tag] {
				parent.RemoveChild(child)
				child = next
				continue
			}
			if !allowedArticleTags[tag] {
				for grand := child.FirstChild; grand != nil; {
					grandNext := grand.NextSibling
					parent.InsertBefore(grand, child)
					grand = grandNext
				}
				parent.RemoveChild(child)
				child = next
				continue
			}
			sanitizeArticleAttributes(child, base)
			if child.Parent != nil { sanitizeHTMLChildren(child, base) }
		}
		child = next
	}
}

func sanitizeArticleAttributes(node *htmlpkg.Node, base string) {
	attrs := make([]htmlpkg.Attribute, 0, 2)
	for _, attr := range node.Attr {
		switch strings.ToLower(attr.Key) {
		case "src":
			if strings.EqualFold(node.Data, "img") {
				if value := normalizeURL(attr.Val, base); value != "" {
					attrs = append(attrs, htmlpkg.Attribute{Key: "src", Val: value})
				}
			}
		case "href":
			if strings.EqualFold(node.Data, "a") {
				if value := normalizeURL(attr.Val, base); value != "" {
					attrs = append(attrs, htmlpkg.Attribute{Key: "href", Val: value})
				}
			}
		case "alt":
			if strings.EqualFold(node.Data, "img") {
				attrs = append(attrs, htmlpkg.Attribute{Key: "alt", Val: attr.Val})
			}
		}
	}
	if strings.EqualFold(node.Data, "img") {
		hasSource := false
		for _, attr := range attrs {
			if attr.Key == "src" { hasSource = true; break }
		}
		if !hasSource && node.Parent != nil {
			node.Parent.RemoveChild(node)
			return
		}
	}
	node.Attr = attrs
}

var youtubeIframePattern = regexp.MustCompile("(?is)<iframe[^>]+src=[\\\"']([^\\\"']+)[\\\"'][^>]*>.*?</iframe>")

func replaceYouTubeEmbeds(value string) string {
	return youtubeIframePattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := youtubeIframePattern.FindStringSubmatch(match)
		if len(parts) != 2 { return "" }
		videoURL := normalizeYouTubeURL(parts[1])
		if videoURL == "" { return "" }
		return "<p><a href=\"" + html.EscapeString(videoURL) + "\">▶ Watch on YouTube</a></p>"
	})
}

func normalizeYouTubeURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(html.UnescapeString(raw)))
	if err != nil { return "" }
	host := strings.ToLower(u.Hostname())
	if host == "youtu.be" {
		id := strings.Trim(strings.TrimSpace(u.Path), "/")
		if id != "" { return "https://www.youtube.com/watch?v=" + url.QueryEscape(id) }
	}
	if host == "youtube.com" || host == "www.youtube.com" || host == "m.youtube.com" {
		if strings.HasPrefix(u.Path, "/embed/") {
			id := strings.Trim(strings.TrimPrefix(u.Path, "/embed/"), "/")
			if id != "" { return "https://www.youtube.com/watch?v=" + url.QueryEscape(id) }
		}
		if id := u.Query().Get("v"); id != "" {
			return "https://www.youtube.com/watch?v=" + url.QueryEscape(id)
		}
	}
	return ""
}

func extractImageURL(value, base string) string {
	value = html.UnescapeString(value)
	lower := strings.ToLower(value)
	for {
		start := strings.Index(lower, "<img")
		if start < 0 {
			return ""
		}
		tagEnd := strings.Index(value[start:], ">")
		if tagEnd < 0 {
			return ""
		}
		tagEnd += start
		tag := value[start : tagEnd+1]
		lowerTag := strings.ToLower(tag)
		src := strings.Index(lowerTag, "src=")
		if src >= 0 {
			rest := strings.TrimSpace(tag[src+4:])
			if len(rest) > 1 && (rest[0] == '\'' || rest[0] == '"') {
				quote := rest[0]
				if end := strings.IndexByte(rest[1:], quote); end >= 0 {
					return normalizeURL(rest[1:end+1], base)
				}
			}
		}
		value = value[tagEnd+1:]
		lower = strings.ToLower(value)
	}
}

func fetchItems(ctx context.Context, client *http.Client, rawURL string) ([]Item, error) {
	body, err := downloadFeed(ctx, client, rawURL)
	if err != nil {
		return nil, err
	}

	var rss rssDocument
	if err := xml.Unmarshal(body, &rss); err == nil && len(rss.Channel.Items) > 0 {
		items := make([]Item, 0, len(rss.Channel.Items))
		for _, entry := range rss.Channel.Items {
			author := strings.TrimSpace(entry.Author)
			if author == "" {
				author = strings.TrimSpace(entry.Creator)
			}
			articleSource := entry.ContentEncoded
			if articleSource == "" {
				articleSource = entry.Description
			}
			articleHTML := sanitizeArticleHTML(articleSource, rawURL)
			imageURL := extractImageURL(articleSource, rawURL)
			for _, media := range entry.Thumbnail {
				if media.URL != "" {
					imageURL = media.URL
					break
				}
			}
			if imageURL == "" {
				for _, media := range entry.Media {
					if media.URL != "" {
						imageURL = media.URL
						break
					}
				}
			}
			if imageURL == "" && strings.HasPrefix(strings.ToLower(entry.Enclosure.Type), "image/") {
				imageURL = entry.Enclosure.URL
			}
			items = append(items, Item{
				Title: cleanText(entry.Title),
				Author: cleanText(author),
				PublishedAt: normalizePublished(entry.PubDate),
				Summary: cleanText(entry.Description),
				URL: normalizeURL(entry.Link, rawURL),
				GUID: cleanText(entry.GUID),
				ImageURL: normalizeURL(imageURL, rawURL),
				ArticleHTML: articleHTML,
			})
		}
		return items, nil
	}

	var atom atomDocument
	if err := xml.Unmarshal(body, &atom); err != nil {
		return nil, fmt.Errorf("invalid RSS/Atom XML: %w", err)
	}
	if len(atom.Entries) == 0 {
		return nil, errors.New("feed contains no RSS items or Atom entries")
	}
	items := make([]Item, 0, len(atom.Entries))
	for _, entry := range atom.Entries {
		link := ""
		for _, candidate := range entry.Links {
			if candidate.Rel == "" || candidate.Rel == "alternate" {
				link = candidate.Href
				break
			}
		}
		published := entry.Published
		if published == "" {
			published = entry.Updated
		}
		summary := entry.Summary
		if summary == "" {
			summary = entry.Content
		}
		articleHTML := sanitizeArticleHTML(entry.Content, rawURL)
		if articleHTML == "" {
			articleHTML = sanitizeArticleHTML(summary, rawURL)
		}
		imageURL := extractImageURL(entry.Content, rawURL)
		if imageURL == "" {
			imageURL = extractImageURL(summary, rawURL)
		}
		items = append(items, Item{
			Title: cleanText(entry.Title),
			Author: cleanText(entry.Author.Name),
			PublishedAt: normalizePublished(published),
			Summary: cleanText(summary),
			URL: normalizeURL(link, rawURL),
			GUID: cleanText(entry.ID),
			ImageURL: imageURL,
			ArticleHTML: articleHTML,
		})
	}
	return items, nil
}

func downloadFeed(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OtterLink-News/1.0")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml;q=0.9, */*;q=0.1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("feed request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("feed returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read feed: %w", err)
	}
	if len(body) > maxFeedBytes {
		return nil, errors.New("feed exceeds 2 MiB limit")
	}
	return body, nil
}

func normalizePublished(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	layouts := []string{
		time.RFC3339, time.RFC3339Nano, time.RFC1123Z, time.RFC1123,
		time.RFC822Z, time.RFC822, time.RFC850,
		"Mon, 2 Jan 2006 15:04:05 MST",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	return value
}

func normalizeURL(raw, base string) string {
	raw = strings.TrimSpace(html.UnescapeString(raw))
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return ""
		}
		return parsed.String()
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	resolved := baseURL.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	return resolved.String()
}

func cleanText(value string) string {
	value = html.UnescapeString(strings.TrimSpace(value))
	return strings.Join(strings.Fields(value), " ")
}

func nullableTime(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func stableGUID(rawURL, title string) string {
	sum := sha256.Sum256([]byte(rawURL + "\x00" + title))
	return hex.EncodeToString(sum[:])
}
