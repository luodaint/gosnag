package dbqueryanalysis

import "testing"

func TestNormalizeSQL(t *testing.T) {
	got := normalizeSQL("SELECT *\nFROM reservs WHERE id_restaurant = 6587 AND date = '2026-09-10';")
	want := "SELECT * FROM reservs WHERE id_restaurant = ? AND date = ?"
	if got != want {
		t.Fatalf("unexpected normalized sql: got %q want %q", got, want)
	}
}

func TestExtractQueriesAndAnalyze(t *testing.T) {
	breadcrumbs := []Breadcrumb{
		{Category: "db.query", Message: "SELECT * FROM reservs WHERE id = 1", Data: map[string]any{"duration_ms": 1.2}},
		{Category: "db.query", Message: "SELECT * FROM reservs WHERE id = 2", Data: map[string]any{"duration_ms": 1.4}},
		{Category: "db.query", Message: "SELECT * FROM reservs WHERE id = 3", Data: map[string]any{"duration_ms": 1.6}},
		{Category: "db.query", Message: "SELECT * FROM reservs WHERE id = 4", Data: map[string]any{"duration_ms": 1.8}},
	}

	items, warnings := extractQueries(breadcrumbs)
	if len(items) != 4 {
		t.Fatalf("expected 4 queries, got %d", len(items))
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}

	res := analyzeQueries(items)
	if res.Summary.QueryCount != 4 {
		t.Fatalf("unexpected query count: %d", res.Summary.QueryCount)
	}
	if len(res.Queries) != 1 {
		t.Fatalf("expected 1 grouped query, got %d", len(res.Queries))
	}
	if !res.Queries[0].SuspectedNPlusOne {
		t.Fatalf("expected N+1 candidate")
	}
}
