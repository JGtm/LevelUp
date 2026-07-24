package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/games/mappings"
)

// twoErasInfinite reproduit les éras Halo Infinite : ×1 avant le 2025-11-18 (borne
// ouverte à gauche), ×2 depuis (borne ouverte à droite).
func twoErasInfinite() []mappings.CareerXPEra {
	cut := time.Date(2025, 11, 18, 0, 0, 0, 0, time.UTC)
	return []mappings.CareerXPEra{
		{From: time.Time{}, To: cut, Multiplier: 1.0},
		{From: cut, To: time.Time{}, Multiplier: 2.0},
	}
}

func TestEstimateCareerXP_EraBoundaries(t *testing.T) {
	t.Parallel()
	eras := twoErasInfinite()
	const ps = 1500

	cases := []struct {
		name    string
		endedAt time.Time
		want    int
	}{
		{"veille 2025-11-17 23:59:59 UTC → ×1", time.Date(2025, 11, 17, 23, 59, 59, 0, time.UTC), 1500},
		{"jour J 2025-11-18 00:00:00 UTC → ×2 (From inclusif)", time.Date(2025, 11, 18, 0, 0, 0, 0, time.UTC), 3000},
		{"jour J +1s → ×2", time.Date(2025, 11, 18, 0, 0, 1, 0, time.UTC), 3000},
		{"lendemain 2025-11-19 → ×2", time.Date(2025, 11, 19, 12, 0, 0, 0, time.UTC), 3000},
		{"bien avant → ×1", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1500},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := EstimateCareerXP(ps, c.endedAt, eras); got != c.want {
				t.Errorf("EstimateCareerXP(%d, %s) = %d, want %d", ps, c.endedAt.Format(time.RFC3339), got, c.want)
			}
		})
	}
}

func TestEstimateCareerXP_DegradesToZero(t *testing.T) {
	t.Parallel()
	eras := twoErasInfinite()
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// Score 0 → 0.
	if got := EstimateCareerXP(0, now, eras); got != 0 {
		t.Errorf("score 0 = %d, want 0", got)
	}
	// Éras vides → 0 (aucun multiplicateur connu), jamais de panic.
	if got := EstimateCareerXP(1500, now, nil); got != 0 {
		t.Errorf("éras nil = %d, want 0", got)
	}
	if got := EstimateCareerXP(1500, now, []mappings.CareerXPEra{}); got != 0 {
		t.Errorf("éras vides = %d, want 0", got)
	}
	// Trou de couverture : une seule éra fermée des deux côtés, instant hors bornes → 0.
	bounded := []mappings.CareerXPEra{{
		From:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:         time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		Multiplier: 2.0,
	}}
	if got := EstimateCareerXP(1500, now, bounded); got != 0 {
		t.Errorf("hors couverture = %d, want 0", got)
	}
}

func TestEstimateCareerXP_Rounding(t *testing.T) {
	t.Parallel()
	// Multiplicateur non entier (futur raffinement playlist, ex. BTB ×1,8) : arrondi
	// à l'entier le plus proche. 1,8 × 1005 = 1809.0 → 1809.
	eras := []mappings.CareerXPEra{{From: time.Time{}, To: time.Time{}, Multiplier: 1.8}}
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := EstimateCareerXP(1005, at, eras); got != 1809 {
		t.Errorf("1.8 × 1005 = %d, want 1809", got)
	}
	// 1,8 × 1006 = 1810.8 → 1811 (arrondi supérieur).
	if got := EstimateCareerXP(1006, at, eras); got != 1811 {
		t.Errorf("1.8 × 1006 = %d, want 1811", got)
	}
}
