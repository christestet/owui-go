package pipelines

import "testing"

func TestNormalizeRegistration_Fallbacks(t *testing.T) {
	raw := map[string]any{
		"pipeline_id": "reg-a",
		"base_url":    "http://pipelines-a:9099",
	}
	reg := normalizeRegistration(raw, 3)
	if reg.RegistrationID != "reg-a" {
		t.Fatalf("expected registration_id reg-a, got %q", reg.RegistrationID)
	}
	if reg.URL != "http://pipelines-a:9099" {
		t.Fatalf("expected URL fallback from base_url, got %q", reg.URL)
	}
	if reg.URLIdx != 3 {
		t.Fatalf("expected fallback urlIdx=3, got %d", reg.URLIdx)
	}
}

func TestNormalizePipe_Fallbacks(t *testing.T) {
	reg := Registration{RegistrationID: "reg-a", URL: "http://x", URLIdx: 2}
	raw := map[string]any{
		"pipeline_id": "rag_ingest",
		"title":       "RAG Ingest",
	}
	pipe := normalizePipe(raw, reg)
	if pipe.PipeID != "rag_ingest" {
		t.Fatalf("expected pipe_id fallback from pipeline_id, got %q", pipe.PipeID)
	}
	if pipe.Name != "RAG Ingest" {
		t.Fatalf("expected name fallback from title, got %q", pipe.Name)
	}
	if pipe.RegistrationID != "reg-a" || pipe.URLIdx != 2 {
		t.Fatalf("expected parent context copied, got reg=%q idx=%d", pipe.RegistrationID, pipe.URLIdx)
	}
}

func TestResolvePipe_Ambiguous(t *testing.T) {
	inv := &Inventory{
		PipeIndex: map[string][]Pipe{
			"rag_ingest": {
				{PipeID: "rag_ingest", URLIdx: 0},
				{PipeID: "rag_ingest", URLIdx: 1},
			},
		},
	}
	_, err := resolvePipe(inv, "rag_ingest", nil)
	if err == nil {
		t.Fatal("expected ambiguity error")
	}
	if err.Error() != "pipe_id \"rag_ingest\" is ambiguous across urlIdx [0,1]; pass --url-idx" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolvePipe_WithURLIdx(t *testing.T) {
	idx := 1
	inv := &Inventory{
		PipeIndex: map[string][]Pipe{
			"rag_ingest": {
				{PipeID: "rag_ingest", URLIdx: 0},
				{PipeID: "rag_ingest", URLIdx: 1},
			},
		},
	}
	pipe, err := resolvePipe(inv, "rag_ingest", &idx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pipe.URLIdx != 1 {
		t.Fatalf("expected urlIdx=1, got %d", pipe.URLIdx)
	}
}

func TestNextFreeURLIdx(t *testing.T) {
	inv := &Inventory{
		Registrations: []Registration{{URLIdx: 0}, {URLIdx: 3}, {URLIdx: 2}},
	}
	if got := nextFreeURLIdx(inv); got != 4 {
		t.Fatalf("expected next free idx 4, got %d", got)
	}
}

func TestFilterInventory(t *testing.T) {
	inv := &Inventory{
		Registrations: []Registration{{RegistrationID: "pipeline-reg-01", URL: "http://a", URLIdx: 0}},
		Pipes:         []Pipe{{PipeID: "rag_ingest", Name: "RAG Ingest", RegistrationID: "pipeline-reg-01", URLIdx: 0}},
		PipeIndex:     map[string][]Pipe{"rag_ingest": {{PipeID: "rag_ingest", URLIdx: 0}}},
		RegIndex:      map[string][]Registration{"pipeline-reg-01": {{RegistrationID: "pipeline-reg-01", URLIdx: 0}}},
	}
	filtered := filterInventory(inv, "rag")
	if len(filtered.Pipes) != 1 {
		t.Fatalf("expected 1 pipe after filter, got %d", len(filtered.Pipes))
	}
	if len(filtered.Registrations) != 0 {
		t.Fatalf("expected 0 registrations after filter on pipe name, got %d", len(filtered.Registrations))
	}
}
