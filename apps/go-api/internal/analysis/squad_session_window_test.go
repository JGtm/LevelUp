package analysis

import "testing"

const day = int64(86400)

// times génère n timestamps Unix espacés de gapDays jours, croissants.
func times(n int, gapDays int64) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = int64(i) * gapDays * day
	}
	return out
}

func TestSquadSessionWindowKeep_BelowFloorKeepsAll(t *testing.T) {
	cfg := DefaultSquadSessionWindow()
	// n <= MinSessions → on garde tout.
	for n := 0; n <= cfg.MinSessions; n++ {
		if got := SquadSessionWindowKeep(times(n, 1), cfg); got != n {
			t.Errorf("n=%d: expected keep=%d, got %d", n, n, got)
		}
	}
}

func TestSquadSessionWindowKeep_HardcoreClampsToMinDays(t *testing.T) {
	cfg := DefaultSquadSessionWindow()
	// 30 sessions à 1 j d'écart : 12×1 = 12 j < MinDays(14) → horizon 14 j.
	// cutoff = jour 29 − 14 = jour 15 → sessions 15..29 = 15.
	if got := SquadSessionWindowKeep(times(30, 1), cfg); got != 15 {
		t.Errorf("hardcore: expected 15, got %d", got)
	}
}

func TestSquadSessionWindowKeep_OccasionalClampsToMaxDays(t *testing.T) {
	cfg := DefaultSquadSessionWindow()
	// 30 sessions à 10 j d'écart : 12×10 = 120 j = MaxDays → horizon 120 j.
	// last = jour 290, cutoff = jour 170 → sessions i où 10i ≥ 170 → i=17..29 = 13.
	if got := SquadSessionWindowKeep(times(30, 10), cfg); got != 13 {
		t.Errorf("occasional: expected 13, got %d", got)
	}
}

func TestSquadSessionWindowKeep_CapsAtMaxSessions(t *testing.T) {
	cfg := DefaultSquadSessionWindow()
	// 50 sessions le même jour (écart médian 0 → fallback 1 j, horizon 14 j) :
	// toutes ≥ cutoff → 50 → plafonné à MaxSessions(20).
	all := make([]int64, 50) // tous à 0
	if got := SquadSessionWindowKeep(all, cfg); got != cfg.MaxSessions {
		t.Errorf("dense: expected %d, got %d", cfg.MaxSessions, got)
	}
}

func TestSquadSessionWindowKeep_FloorsAtMinSessions(t *testing.T) {
	cfg := DefaultSquadSessionWindow()
	// 10 sessions très espacées (1000 j) : horizon plafonné à MaxDays(120) →
	// seule la dernière tombe dans la fenêtre → relevé au plancher MinSessions(6).
	if got := SquadSessionWindowKeep(times(10, 1000), cfg); got != cfg.MinSessions {
		t.Errorf("sparse: expected floor %d, got %d", cfg.MinSessions, got)
	}
}
