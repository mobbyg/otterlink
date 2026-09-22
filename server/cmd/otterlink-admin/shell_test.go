package main

import (
	"reflect"
	"testing"
)

func TestSplitPipelineRespectsQuotes(t *testing.T) {
	got, err := splitPipeline(\`users list | where display_name="A|B" | select username\`)
	if err != nil { t.Fatal(err) }
	want := []string{"users list", \`where display_name="A|B"\`, "select username"}
	if !reflect.DeepEqual(got, want) { t.Fatalf("got %#v want %#v", got, want) }
}

func TestTokenizeQuotes(t *testing.T) {
	got, err := tokenize(\`chat create "My Community"\`)
	if err != nil { t.Fatal(err) }
	want := []string{"chat", "create", "My Community"}
	if !reflect.DeepEqual(got, want) { t.Fatalf("got %#v want %#v", got, want) }
}

func TestPipeWhere(t *testing.T) {
	in := Records{{"username":"alice","status":"active"},{"username":"bob","status":"disabled"}}
	got, err := pipeWhere([]string{"status=active"}, in)
	if err != nil { t.Fatal(err) }
	if len(got) != 1 || got[0]["username"] != "alice" { t.Fatalf("unexpected result: %#v", got) }
}

func TestPipeSelectAndCount(t *testing.T) {
	in := Records{{"username":"alice","role":"admin"},{"username":"bob","role":"user"}}
	got, err := pipeSelect([]string{"username"}, in)
	if err != nil { t.Fatal(err) }
	if len(got) != 2 || got[0]["role"] != nil || got[0]["username"] != "alice" { t.Fatalf("unexpected select: %#v", got) }
	count := Records{{"count":len(got)}}
	if count[0]["count"] != 2 { t.Fatal("unexpected count") }
}

func TestPipeLimit(t *testing.T) {
	in := Records{{"n":1},{"n":2},{"n":3}}
	got, err := pipeLimit([]string{"2"}, in, true)
	if err != nil { t.Fatal(err) }
	if len(got) != 2 || got[1]["n"] != 2 { t.Fatalf("unexpected head: %#v", got) }
	got, err = pipeLimit([]string{"2"}, in, false)
	if err != nil { t.Fatal(err) }
	if len(got) != 2 || got[0]["n"] != 2 { t.Fatalf("unexpected tail: %#v", got) }
}
