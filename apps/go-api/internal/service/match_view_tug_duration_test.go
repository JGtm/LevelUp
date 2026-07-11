package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// TestTugDurationMS couvre A3 (résidus H5) : la durée servant à binner la
// Dominance privilégie playable_duration_seconds (Infinite, inchangé) et tombe
// en fallback sur la durée de gameplay dérivée (duration_seconds − T0) quand
// playable est NULL (Halo 5, qui n'écrit jamais ce champ). Sans fallback la
// Dominance est vide sur tout Halo 5.
func TestTugDurationMS(t *testing.T) {
	t.Parallel()

	ptrI64 := func(v int64) *int64 { return &v }
	ptrF64 := func(v float64) *float64 { return &v }

	cases := []struct {
		name string
		meta *domain.MatchMetaRaw
		want int64
	}{
		{
			name: "playable présent (Infinite) → prioritaire",
			meta: &domain.MatchMetaRaw{
				PlayableDurationSeconds: ptrI64(300),
				DurationSeconds:         ptrF64(360),
				T0Ms:                    ptrI64(15000),
			},
			want: 300 * 1000,
		},
		{
			name: "playable NULL (Halo 5) → fallback duration − T0",
			meta: &domain.MatchMetaRaw{
				PlayableDurationSeconds: nil,
				DurationSeconds:         ptrF64(600),
				T0Ms:                    ptrI64(28000),
			},
			want: (600 - 28) * 1000,
		},
		{
			name: "playable NULL, pas de T0 → duration brute",
			meta: &domain.MatchMetaRaw{
				PlayableDurationSeconds: nil,
				DurationSeconds:         ptrF64(480),
			},
			want: 480 * 1000,
		},
		{
			name: "playable NULL et duration NULL → 0 (pas de dominance fabriquée)",
			meta: &domain.MatchMetaRaw{},
			want: 0,
		},
		{
			name: "meta nil → 0",
			meta: nil,
			want: 0,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tugDurationMS(tc.meta); got != tc.want {
				t.Errorf("tugDurationMS = %d, want %d", got, tc.want)
			}
		})
	}
}
