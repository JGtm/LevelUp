package observability

import (
	"log/slog"
	"testing"
	"time"
)

// TestObs_TitledParityAndRouting — ORACLE DOUBLE MT-05 (PMT-10 Expand) :
//   - (a) PARITÉ : le titre par défaut (halo_infinite) et le titre vide (legacy)
//     collapsent vers la MÊME dimension nue → byte-identique au pré-seam.
//   - (b) ROUTING : un titre non-défaut (synthetic_title_b) obtient des
//     clés/buckets DISTINCTS et ne pollue pas la vue Halo.
func TestObs_TitledParityAndRouting(t *testing.T) {
	// ── expvar : obsKey + agrégation ────────────────────────────────────────
	if obsKey("", "svc") != "svc" || obsKey("halo_infinite", "svc") != "svc" {
		t.Error("obsKey : défaut/vide doit donner la clé NUE (parité Halo)")
	}
	if obsKey("synthetic_title_b", "svc") != "synthetic_title_b.svc" {
		t.Errorf("obsKey : titre non-défaut doit préfixer, got %q", obsKey("synthetic_title_b", "svc"))
	}

	Reset()
	RecordDurationMS("pmt10_dur", 10)                   // legacy → nu
	RecordDurationMST("halo_infinite", "pmt10_dur", 30) // défaut → MÊME clé nue
	if c, _, _, _ := LoadDurationStats("pmt10_dur"); c != 2 {
		t.Errorf("parité : legacy + halo_infinite doivent agréger (clé nue), count=%d want 2", c)
	}
	RecordDurationMST("synthetic_title_b", "pmt10_dur", 99) // routé distinct
	if c, _, _, _ := LoadDurationStats("pmt10_dur"); c != 2 {
		t.Errorf("routing : synthetic ne doit PAS polluer la clé nue, count=%d want 2", c)
	}
	if c, _, _, m := LoadDurationStatsT("synthetic_title_b", "pmt10_dur"); c != 1 || m != 99 {
		t.Errorf("routing : bucket synthetic count=%d max=%d want 1/99", c, m)
	}
	Reset()

	// ── error_collector ─────────────────────────────────────────────────────
	ec := newErrorCollector(16)
	mk := func() slog.Record { return slog.NewRecord(time.Now(), slog.LevelError, "mod: boom", 0) }
	ec.recordT(mk(), "")                  // nu
	ec.recordT(mk(), "halo_infinite")     // défaut → MÊME bucket nu
	ec.recordT(mk(), "synthetic_title_b") // distinct
	var bare, synth *ErrorBucket
	esnap := ec.snapshot()
	for i := range esnap {
		switch esnap[i].Title {
		case "":
			bare = &esnap[i]
		case "synthetic_title_b":
			synth = &esnap[i]
		}
	}
	if bare == nil || bare.Count != 2 {
		t.Errorf("error parité : bucket nu (Title=\"\") doit compter 2 (legacy+halo), got %+v", bare)
	}
	if synth == nil || synth.Count != 1 {
		t.Errorf("error routing : bucket synthetic_title_b distinct count=1, got %+v", synth)
	}

	// ── player_api_collector ────────────────────────────────────────────────
	pc := newPlayerAPICollector(16)
	pc.record("match_history", "JG", 10, false)                      // nu
	pc.recordT("halo_infinite", "match_history", "JG", 20, false)    // défaut → MÊME
	pc.recordT("synthetic_title_b", "match_history", "JG", 99, true) // distinct
	var pbare, psynth *PlayerAPIStat
	psnap := pc.snapshot()
	for i := range psnap {
		switch psnap[i].Title {
		case "":
			pbare = &psnap[i]
		case "synthetic_title_b":
			psynth = &psnap[i]
		}
	}
	if pbare == nil || pbare.Count != 2 {
		t.Errorf("player parité : stat nu doit compter 2, got %+v", pbare)
	}
	if psynth == nil || psynth.Count != 1 || psynth.Errors != 1 {
		t.Errorf("player routing : stat synthetic distinct count=1 errors=1, got %+v", psynth)
	}
}
