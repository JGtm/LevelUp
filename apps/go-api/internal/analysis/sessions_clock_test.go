package analysis

import (
	"testing"
	"time"
)

// Tests deterministes pour IsSessionPotentiallyActiveAt utilisant un now
// injecte (P3.7 / gap test IV — auparavant le test dependait de l'heure
// reelle).
func TestIsSessionPotentiallyActiveAt(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 4, 29, 14, 0, 0, 0, loc) // Mercredi 14:00 UTC

	cases := []struct {
		name          string
		lastMatchTime time.Time
		cutoffHour    int
		want          bool
	}{
		{
			"meme jour -> active",
			time.Date(2026, 4, 29, 9, 0, 0, 0, loc),
			6,
			true,
		},
		{
			"hier 23h, cutoff=6h, now 14h -> non (now.Hour 14 >= cutoff 6)",
			time.Date(2026, 4, 28, 23, 0, 0, 0, loc),
			6,
			false,
		},
		{
			"hier 23h, cutoff=18h, now 14h -> oui (14 < 18)",
			time.Date(2026, 4, 28, 23, 0, 0, 0, loc),
			18,
			true,
		},
		{
			"avant-hier -> non",
			time.Date(2026, 4, 27, 23, 0, 0, 0, loc),
			18,
			false,
		},
		{
			"semaine derniere -> non",
			time.Date(2026, 4, 22, 14, 0, 0, 0, loc),
			18,
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsSessionPotentiallyActiveAt(now, tc.lastMatchTime, tc.cutoffHour)
			if got != tc.want {
				t.Errorf("IsSessionPotentiallyActiveAt(%v, %v, %d) = %v, want %v",
					now, tc.lastMatchTime, tc.cutoffHour, got, tc.want)
			}
		})
	}
}
