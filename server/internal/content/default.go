package content

import (
	"database/sql"
	"encoding/json"
)

func EnsureDefaultHome(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM content_screens WHERE slug='home'`).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return nil
	}

	content := map[string]any{
		"hero": map[string]any{
			"title": "Welcome to Otter Link",
			"body": "Your online world for modern and retro computers. See what's new, meet other Otters, and explore the services available to you.",
			"icon": "/\\_/\\\n( o.o )\n > ^ <",
		},
		"announcements": []any{
			map[string]any{
				"title": "Welcome to the new Otter Link desktop",
				"body": "The Qt 6 client now opens services as movable windows, giving Otter Link a classic online-service feel.",
				"action_label": "Learn more",
			},
			map[string]any{
				"title": "People & Buddy Presence",
				"body": "See your buddies, organize them into groups, and watch their online status change while you're connected.",
				"action_label": "Open People",
				"destination": map[string]any{"type": "service", "service": "people"},
			},
			map[string]any{
				"title": "Community Chat",
				"body": "Talk with the Otter Link community in the shared chat service.",
				"action_label": "Open Chat",
				"destination": map[string]any{"type": "service", "service": "chat"},
			},
		},
		"services": []any{
			map[string]any{"title": "People", "description": "Buddies and online users", "icon": "👥", "destination": map[string]any{"type": "service", "service": "people"}},
			map[string]any{"title": "Community Chat", "description": "Talk with everyone online", "icon": "💬", "destination": map[string]any{"type": "service", "service": "chat"}},
			map[string]any{"title": "Mail", "description": "Private messages — coming soon", "icon": "✉", "destination": map[string]any{"type": "service", "service": "mail"}},
			map[string]any{"title": "Boards", "description": "Community discussions — coming soon", "icon": "▤", "destination": map[string]any{"type": "service", "service": "boards"}},
			map[string]any{"title": "News", "description": "Updates and announcements", "icon": "📰", "destination": map[string]any{"type": "service", "service": "news"}},
			map[string]any{"title": "Games", "description": "Interactive Fiction and classic text adventures", "icon": "🎮", "destination": map[string]any{"type": "service", "service": "games"}},
		},
		"footer": "Otter Link is actively being built. More services are on the way.",
	}
	encoded, err := json.Marshal(content)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO content_screens
		(slug, title, published, priority, version, content_json)
		VALUES ('home', 'Welcome to Otter Link', 1, 0, 1, ?)`, string(encoded))
	return err
}

func EnsureDefaultGames(db *sql.DB) error {
	if err := ensureGamesHomeTileDescription(db); err != nil {
		return err
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM content_screens WHERE slug='games'").Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return nil
	}

	content := map[string]any{
		"hero": map[string]any{
			"title": "OtterLink Games",
			"body": "Explore interactive fiction and classic text adventures. Browse the catalog to see which stories are available on this server.",
			"icon": "🎮",
		},
		"announcements": []any{
			map[string]any{
				"title": "Welcome to Games",
				"body": "The catalog and this home screen are managed by OtterLink administrators. Story files are uploaded or installed by the server operator, and only games the server is permitted to host should be published.",
				"action_label": "Browse Catalog",
				"destination": map[string]any{"type": "service", "service": "games-catalog"},
			},
		},
		"services": []any{
			map[string]any{
				"title": "Interactive Fiction Catalog",
				"description": "Browse available stories and start a private game session.",
				"icon": "📚",
				"destination": map[string]any{"type": "service", "service": "games-catalog"},
			},
		},
		"footer": "Game sessions are private to your account. Your saves are stored separately from other players.",
	}
	encoded, err := json.Marshal(content)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO content_screens (slug,title,published,priority,version,content_json) VALUES ('games','OtterLink Games',1,0,1,?)", string(encoded))
	return err
}

func ensureGamesHomeTileDescription(db *sql.DB) error {
	var raw string
	if err := db.QueryRow("SELECT content_json FROM content_screens WHERE slug='home'").Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	var content map[string]any
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		return nil
	}
	services, ok := content["services"].([]any)
	if !ok {
		return nil
	}
	changed := false
	for _, value := range services {
		item, ok := value.(map[string]any)
		if !ok || item["title"] != "Games" || item["description"] != "Online games — coming soon" {
			continue
		}
		item["description"] = "Interactive Fiction and classic text adventures"
		changed = true
	}
	if !changed {
		return nil
	}
	encoded, err := json.Marshal(content)
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE content_screens SET content_json=?,version=version+1,updated_at=CURRENT_TIMESTAMP WHERE slug='home'", string(encoded))
	return err
}
