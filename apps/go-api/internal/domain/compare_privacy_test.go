package domain

import "testing"

// ── CompareRequest.Validate ──────────────────────────────────────────────────

func TestCompareRequest_Validate_Empty(t *testing.T) {
	r := CompareRequest{}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty target_gamertag")
	}
}

func TestCompareRequest_Validate_Valid(t *testing.T) {
	r := CompareRequest{TargetGamertag: "Player2"}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCompareRequest_Validate_WithFilters(t *testing.T) {
	r := CompareRequest{
		TargetGamertag: "Player2",
		Filters: FilterContextInput{
			FilterMode: "period",
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// ── NewPrivacyWarning ────────────────────────────────────────────────────────

func TestNewPrivacyWarning_None(t *testing.T) {
	w := NewPrivacyWarning(MatchPrivacyInfo{})
	if w.Level != "none" {
		t.Fatalf("expected 'none', got %q", w.Level)
	}
}

func TestNewPrivacyWarning_Full(t *testing.T) {
	w := NewPrivacyWarning(MatchPrivacyInfo{IsPrivate: true})
	if w.Level != "full" {
		t.Fatalf("expected 'full', got %q", w.Level)
	}
	if w.Message == "" {
		t.Fatal("expected non-empty message for full privacy")
	}
}

func TestNewPrivacyWarning_Partial(t *testing.T) {
	w := NewPrivacyWarning(MatchPrivacyInfo{IsPartial: true})
	if w.Level != "partial" {
		t.Fatalf("expected 'partial', got %q", w.Level)
	}
	if w.Message == "" {
		t.Fatal("expected non-empty message for partial privacy")
	}
}

func TestNewPrivacyWarning_PrivateTakesPrecedence(t *testing.T) {
	w := NewPrivacyWarning(MatchPrivacyInfo{IsPrivate: true, IsPartial: true})
	if w.Level != "full" {
		t.Fatalf("expected 'full' when both private+partial, got %q", w.Level)
	}
}
