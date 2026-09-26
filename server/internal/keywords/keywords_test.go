package keywords

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestKeywordRoundTrip(t *testing.T) {
	db, err := sql.Open("sqlite", "file:keywordtest?mode=memory&cache=shared")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE service_keywords (keyword TEXT PRIMARY KEY, display_name TEXT NOT NULL, description TEXT NOT NULL);
		CREATE TABLE service_keyword_targets (id INTEGER PRIMARY KEY AUTOINCREMENT, keyword TEXT NOT NULL, target_type TEXT NOT NULL, target_id INTEGER NOT NULL, label TEXT NOT NULL);`)
	if err != nil { t.Fatal(err) }
	s := NewService(db)
	want := Keyword{Keyword:"retro", DisplayName:"Retro Computing", Description:"Retro community", Targets:[]Target{{Type:"chat",ID:7,Label:"Retro Chat"},{Type:"event",ID:12,Label:"Retro Night"}}}
	if err := s.Upsert(want); err != nil { t.Fatal(err) }
	got, err := s.Resolve(" RETRO ")
	if err != nil { t.Fatal(err) }
	if got.Keyword != "RETRO" || len(got.Targets) != 2 || got.Targets[1].ID != 12 { t.Fatalf("unexpected result: %#v", got) }
}
