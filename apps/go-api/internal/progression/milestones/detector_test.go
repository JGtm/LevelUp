package milestones

import (
	"context"
	"testing"
	"time"
)

// detector_test.go — tests unitaires du détecteur de milestones.
// Fakes en mémoire pour isoler la logique.

// ─── Fakes ──────────────────────────────────────────────────────────────────

type fakeCatalog struct {
	entries map[string][]CatalogEntry // key = titleSlug
}

func newFakeCatalog() *fakeCatalog { return &fakeCatalog{entries: map[string][]CatalogEntry{}} }

func (r *fakeCatalog) Upsert(_ context.Context, e CatalogEntry) error {
	r.entries[e.TitleSlug] = append(r.entries[e.TitleSlug], e)
	return nil
}
func (r *fakeCatalog) ListByTitle(_ context.Context, titleSlug string) ([]CatalogEntry, error) {
	return r.entries[titleSlug], nil
}

type fakeEarned struct {
	rows []Earned
}

func newFakeEarned() *fakeEarned { return &fakeEarned{} }

func (r *fakeEarned) IsEarned(_ context.Context, userID, titleSlug, milestoneID string) (bool, error) {
	for _, e := range r.rows {
		if e.UserID == userID && e.TitleSlug == titleSlug && e.MilestoneID == milestoneID {
			return true, nil
		}
	}
	return false, nil
}
func (r *fakeEarned) Append(_ context.Context, e Earned) error {
	r.rows = append(r.rows, e)
	return nil
}
func (r *fakeEarned) ListByUser(_ context.Context, userID, titleSlug string) ([]Earned, error) {
	var out []Earned
	for _, e := range r.rows {
		if e.UserID == userID && e.TitleSlug == titleSlug {
			out = append(out, e)
		}
	}
	return out, nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func seedCatalog(c *fakeCatalog, titleSlug string, entries ...CatalogEntry) {
	for _, e := range entries {
		e.TitleSlug = titleSlug
		_ = c.Upsert(context.Background(), e)
	}
}

func findResult(results []DetectionResult, milestoneID string) (DetectionResult, bool) {
	for _, r := range results {
		if r.Milestone.ID == milestoneID {
			return r, true
		}
	}
	return DetectionResult{}, false
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestDetect_Unlock_ThresholdReached(t *testing.T) {
	cat := newFakeCatalog()
	earned := newFakeEarned()
	d := NewDetector(cat, earned)

	seedCatalog(cat, "halo_infinite",
		CatalogEntry{ID: "matches.100", Metric: "matches_played", Threshold: 100, TitleEN: "Centurion", TitleFR: "Centurion"},
	)

	results, err := d.Detect(context.Background(), DetectInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: time.Now(),
		Stats: PlayerStats{Metrics: map[string]float64{"matches_played": 105}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	r, ok := findResult(results, "matches.100")
	if !ok {
		t.Fatal("missing result for matches.100")
	}
	if !r.Earned {
		t.Errorf("expected Earned=true (value 105 >= threshold 100)")
	}
	if r.AlreadyHad {
		t.Errorf("AlreadyHad should be false on first unlock")
	}
	if len(earned.rows) != 1 {
		t.Errorf("EarnedRepo should have 1 row, got %d", len(earned.rows))
	}
}

func TestDetect_AlreadyEarned_NoDuplicate(t *testing.T) {
	cat := newFakeCatalog()
	earned := newFakeEarned()
	d := NewDetector(cat, earned)

	seedCatalog(cat, "halo_infinite",
		CatalogEntry{ID: "wins.50", Metric: "wins", Threshold: 50, TitleEN: "Winner", TitleFR: "Vainqueur"},
	)
	// Seed earned row.
	_ = earned.Append(context.Background(), Earned{
		UserID: "u1", TitleSlug: "halo_infinite", MilestoneID: "wins.50", EarnedAt: time.Now(),
	})

	results, err := d.Detect(context.Background(), DetectInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: time.Now(),
		Stats: PlayerStats{Metrics: map[string]float64{"wins": 200}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	r, _ := findResult(results, "wins.50")
	if !r.AlreadyHad {
		t.Errorf("expected AlreadyHad=true")
	}
	if r.Earned {
		t.Errorf("expected Earned=false (already had it)")
	}
	if len(earned.rows) != 1 {
		t.Errorf("EarnedRepo should still have 1 row (no duplicate), got %d", len(earned.rows))
	}
}

// DP14 : la bande near-miss passe de 90 % à 98 % du seuil. À 95 % il reste des
// semaines (1 000 kills sur 20 000), à 98 % le dénouement est imminent.
func TestDetect_NearMiss_Within2Percent(t *testing.T) {
	cat := newFakeCatalog()
	earned := newFakeEarned()
	d := NewDetector(cat, earned)

	seedCatalog(cat, "halo_infinite",
		CatalogEntry{ID: "kills.1000", Metric: "kills", Threshold: 1000, TitleEN: "Killer", TitleFR: "Tueur"},
	)

	detect := func(kills float64) DetectionResult {
		results, err := d.Detect(context.Background(), DetectInput{
			UserID: "u1", TitleSlug: "halo_infinite", Now: time.Now(),
			Stats: PlayerStats{Metrics: map[string]float64{"kills": kills}},
		})
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		r, _ := findResult(results, "kills.1000")
		return r
	}

	// 950 kills = 95% → PAS near-miss (DP14 : encore trop loin).
	if r := detect(950); r.NearMiss || r.Earned {
		t.Errorf("950 (95%%) : attendu aucun signal, got Earned=%v NearMiss=%v", r.Earned, r.NearMiss)
	}
	// 985 kills = 98.5% → near-miss (dans la bande [98%, 100%)).
	if r := detect(985); !r.NearMiss || r.Earned {
		t.Errorf("985 (98.5%%) : attendu NearMiss=true, got Earned=%v NearMiss=%v", r.Earned, r.NearMiss)
	}
	if len(earned.rows) != 0 {
		t.Errorf("EarnedRepo doit rester vide sur near-miss only")
	}
	// 1000 kills = 100% → earned, pas near-miss.
	if r := detect(1000); !r.Earned || r.NearMiss {
		t.Errorf("1000 (100%%) : attendu Earned=true, got Earned=%v NearMiss=%v", r.Earned, r.NearMiss)
	}
}

func TestDetect_FarFromThreshold_NoSignal(t *testing.T) {
	cat := newFakeCatalog()
	earned := newFakeEarned()
	d := NewDetector(cat, earned)

	seedCatalog(cat, "halo_infinite",
		CatalogEntry{ID: "kills.10000", Metric: "kills", Threshold: 10000, TitleEN: "Legend", TitleFR: "Légende"},
	)

	results, err := d.Detect(context.Background(), DetectInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: time.Now(),
		Stats: PlayerStats{Metrics: map[string]float64{"kills": 3500}}, // 35%, loin
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	r, _ := findResult(results, "kills.10000")
	if r.Earned || r.NearMiss {
		t.Errorf("expected no signal (Earned=%v, NearMiss=%v) at 35%% progress", r.Earned, r.NearMiss)
	}
	if r.Progress < 0.34 || r.Progress > 0.36 {
		t.Errorf("Progress = %.3f, want ~0.35", r.Progress)
	}
}

func TestDetect_MultipleEntries_OnePass(t *testing.T) {
	// Plusieurs paliers sur la même métrique : seuls les paliers atteints
	// doivent être Earned, le reste reste candidat.
	cat := newFakeCatalog()
	earned := newFakeEarned()
	d := NewDetector(cat, earned)

	seedCatalog(cat, "halo_infinite",
		CatalogEntry{ID: "matches.100", Metric: "matches_played", Threshold: 100, TitleEN: "Centurion", TitleFR: "Centurion"},
		CatalogEntry{ID: "matches.500", Metric: "matches_played", Threshold: 500, TitleEN: "Veteran", TitleFR: "Vétéran"},
		CatalogEntry{ID: "matches.1000", Metric: "matches_played", Threshold: 1000, TitleEN: "Elite", TitleFR: "Élite"},
	)

	results, err := d.Detect(context.Background(), DetectInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: time.Now(),
		Stats: PlayerStats{Metrics: map[string]float64{"matches_played": 540}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}

	r100, _ := findResult(results, "matches.100")
	r500, _ := findResult(results, "matches.500")
	r1000, _ := findResult(results, "matches.1000")
	if !r100.Earned {
		t.Errorf("matches.100 should be Earned (540 >= 100)")
	}
	if !r500.Earned {
		t.Errorf("matches.500 should be Earned (540 >= 500)")
	}
	if r1000.Earned {
		t.Errorf("matches.1000 should not be Earned (540 < 1000)")
	}
	if !r1000.NearMiss && r1000.Progress < 0.9 {
		// 540/1000 = 0.54 → ni earned ni near-miss
	}
	if len(earned.rows) != 2 {
		t.Errorf("EarnedRepo should have 2 rows (100 + 500), got %d", len(earned.rows))
	}
}
