package sync

import (
	"errors"
	"testing"

	"levelup/go-api/internal/analysis"
)

// ── isNotFoundErr ────────────────────────────────────────────────────────────

func TestIsNotFoundErr_Nil(t *testing.T) {
	if isNotFoundErr(nil) {
		t.Fatal("expected false for nil")
	}
}

func TestIsNotFoundErr_404(t *testing.T) {
	if !isNotFoundErr(errors.New("HTTP 404 Not Found")) {
		t.Fatal("expected true for HTTP 404")
	}
}

func TestIsNotFoundErr_410(t *testing.T) {
	if !isNotFoundErr(errors.New("HTTP 410 Gone")) {
		t.Fatal("expected true for HTTP 410")
	}
}

func TestIsNotFoundErr_RessourceAbsente(t *testing.T) {
	if !isNotFoundErr(errors.New("ressource absente")) {
		t.Fatal("expected true for ressource absente")
	}
}

func TestIsNotFoundErr_OtherError(t *testing.T) {
	if isNotFoundErr(errors.New("connection refused")) {
		t.Fatal("expected false for other error")
	}
}

// ── attributionsToRows ───────────────────────────────────────────────────────

func TestAttributionsToRows_FiltersByXUID(t *testing.T) {
	wid := uint64(42)
	attrs := []analysis.KillAttribution{
		{XUID: "target", TimeMS: 100, WeaponID: &wid, Confidence: "high"},
		{XUID: "other", TimeMS: 200, WeaponID: &wid, Confidence: "low"},
		{XUID: "target", TimeMS: 300, WeaponID: &wid, Confidence: "medium"},
	}
	rows := attributionsToRows(attrs, "target")
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows for target, got %d", len(rows))
	}
	if rows[0].TimeMS != 100 {
		t.Fatalf("expected TimeMS=100, got %d", rows[0].TimeMS)
	}
	if rows[1].TimeMS != 300 {
		t.Fatalf("expected TimeMS=300, got %d", rows[1].TimeMS)
	}
}

func TestAttributionsToRows_Empty(t *testing.T) {
	rows := attributionsToRows(nil, "x")
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}
