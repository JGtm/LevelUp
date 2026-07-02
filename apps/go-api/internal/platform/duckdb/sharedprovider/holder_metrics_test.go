package sharedprovider

// Tests unitaires PURS (sans tag integration, sans DB) du collecteur par-détenteur.

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/ctxkeys"
)

// TestDBWriterLabel_RoundTripAndDefault : le label posé via ctxkeys est lu tel
// quel ; un ctx nu retombe sur le défaut "unlabeled" (call-site non instrumenté).
func TestDBWriterLabel_RoundTripAndDefault(t *testing.T) {
	ctx := ctxkeys.WithDBWriterLabel(context.Background(), "sync_v2_postsync")
	if got := ctxkeys.DBWriterLabel(ctx); got != "sync_v2_postsync" {
		t.Errorf("DBWriterLabel = %q, want sync_v2_postsync", got)
	}
	if got := ctxkeys.DBWriterLabel(context.Background()); got != ctxkeys.DBWriterLabelUnlabeled {
		t.Errorf("DBWriterLabel(ctx nu) = %q, want %q", got, ctxkeys.DBWriterLabelUnlabeled)
	}
	// Label vide = no-op (on ne remplace pas un label existant par du vide).
	ctx2 := ctxkeys.WithDBWriterLabel(ctx, "")
	if got := ctxkeys.DBWriterLabel(ctx2); got != "sync_v2_postsync" {
		t.Errorf("label vide ne doit pas écraser, got %q", got)
	}
}

// TestRecordHolderWindow_PerLabelStats : accumulation count/total/max par label,
// avg dérivé à la lecture, tri TotalMs décroissant dans holderSnapshot.
func TestRecordHolderWindow_PerLabelStats(t *testing.T) {
	// Labels uniques au test : le collecteur est process-wide.
	recordHolderWindow("t_stats_a", 100)
	recordHolderWindow("t_stats_a", 300)
	recordHolderWindow("t_stats_b", 50)
	recordHolderWatchdog("t_stats_a")

	var a, b *HolderWindowStat
	for i, h := range holderSnapshot() {
		hh := h
		switch h.Label {
		case "t_stats_a":
			a = &hh
		case "t_stats_b":
			b = &hh
		}
		// Tri décroissant sur TotalMs.
		if i > 0 && holderSnapshot()[i-1].TotalMs < h.TotalMs {
			t.Errorf("holderSnapshot non trié TotalMs desc à l'index %d", i)
		}
	}
	if a == nil || b == nil {
		t.Fatal("labels de test absents du snapshot")
	}
	if a.Count != 2 || a.TotalMs != 400 || a.AvgMs != 200 || a.MaxMs != 300 || a.WatchdogFired != 1 {
		t.Errorf("t_stats_a = %+v, want count=2 total=400 avg=200 max=300 watchdog=1", a)
	}
	if b.Count != 1 || b.TotalMs != 50 || b.MaxMs != 50 {
		t.Errorf("t_stats_b = %+v, want count=1 total=50 max=50", b)
	}
}

// TestHolderCollector_CapEvictsLeastActive : au cap, l'entrée la moins active
// est évincée (garde ADR 0009 contre un label dynamique fuité).
func TestHolderCollector_CapEvictsLeastActive(t *testing.T) {
	c := &holderCollector{entries: make(map[string]*holderStats)}
	// Remplir jusqu'au cap, avec des counts croissants (label_0 = le moins actif).
	for i := 0; i < maxHolderLabels; i++ {
		e := c.entryLocked(fmt.Sprintf("cap_label_%d", i))
		e.count = int64(i + 1)
	}
	if len(c.entries) != maxHolderLabels {
		t.Fatalf("attendu %d entrées, got %d", maxHolderLabels, len(c.entries))
	}
	// Une entrée de plus : cap_label_0 (count=1) doit être évincé.
	c.entryLocked("cap_label_new")
	if len(c.entries) != maxHolderLabels {
		t.Errorf("le cap doit être maintenu à %d, got %d", maxHolderLabels, len(c.entries))
	}
	if _, ok := c.entries["cap_label_0"]; ok {
		t.Error("cap_label_0 (le moins actif) aurait dû être évincé")
	}
	if _, ok := c.entries["cap_label_new"]; !ok {
		t.Error("cap_label_new devrait être présent après éviction")
	}
}
