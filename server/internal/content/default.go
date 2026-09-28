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
			"icon": "/\\\\_/\\\\\n( o.o )\\n > ^ <",
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
			map[string]any{"title": "Games", "description": "Online games — coming soon", "icon": "🎮", "destination": map[string]any{"type": "service", "service": "games"}},
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
