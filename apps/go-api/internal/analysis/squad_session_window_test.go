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
	// 30 sessions à 1 j d'écart : 18×1 = 18 j < MinDays(21) → horizon 21 j.
	// cutoff = jour 29 − 21 = jour 8 → sessions 8..29 = 22.
	if got := SquadSessionWindowKeep(times(30, 1), cfg); got != 22 {
		t.Errorf("hardcore: expected 22, got %d", got)
	}
}

func TestSquadSessionWindowKeep_OccasionalClampsToMaxDays(t *testing.T) {
	cfg := DefaultSquadSessionWindow()
	// 30 sessions à 10 j d'écart : 18×10 = 180 j = MaxDays → horizon 180 j.
	// last = jour 290, cutoff = jour 110 → sessions i où 10i ≥ 110 → i=11..29 = 19.
	if got := SquadSessionWindowKeep(times(30, 10), cfg); got != 19 {
		t.Errorf("occasional: expected 19, got %d", got)
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
