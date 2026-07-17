package relations

import (
	"testing"
	"time"
)

func TestIsRevived(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	ptr := func(d time.Duration) *time.Time { v := now.Add(-d); return &v }
	day := 24 * time.Hour

	cases := []struct {
		name         string
		encounters30 int
		prevSeen     *time.Time
		want         bool
	}{
		{"jamais vu avant la fenetre (prevSeen nil)", 3, nil, false},
		{"aucune rencontre recente", 0, ptr(200 * day), false},
		{"gap 89 j (sous le seuil)", 2, ptr(89 * day), false},
		{"gap 90 j (au seuil, inclus)", 1, ptr(90 * day), true},
		{"gap 200 j (bien au-dela)", 4, ptr(200 * day), true},
		{"recent mais prevSeen recent (relation continue)", 5, ptr(10 * day), false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsRevived(tc.encounters30, tc.prevSeen, now); got != tc.want {
				t.Fatalf("IsRevived(%d, %v) = %v, want %v", tc.encounters30, tc.prevSeen, got, tc.want)
			}
		})
	}
}
