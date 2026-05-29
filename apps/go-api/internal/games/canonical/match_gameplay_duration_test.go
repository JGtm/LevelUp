package canonical

import "testing"

func iptr(v int) *int       { return &v }
func i64ptr(v int64) *int64 { return &v }

func TestMatchSummary_GameplayDurationSeconds(t *testing.T) {
	cases := []struct {
		name string
		dur  *int
		t0   *int64
		want *int // nil = attendu nil
	}{
		{"nil duration → nil", nil, i64ptr(28000), nil},
		{"duration sans T0 → durée brute", iptr(447), nil, iptr(447)},
		{"duration − T0 (countdown retranché)", iptr(447), i64ptr(28000), iptr(419)},
		{"T0 > duration → clamp 0", iptr(20), i64ptr(28000), iptr(0)},
		{"T0 nul explicite → durée brute", iptr(600), i64ptr(0), iptr(600)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MatchSummary{DurationSeconds: tc.dur, T0Ms: tc.t0}.GameplayDurationSeconds()
			switch {
			case tc.want == nil && got != nil:
				t.Errorf("want nil, got %d", *got)
			case tc.want != nil && got == nil:
				t.Errorf("want %d, got nil", *tc.want)
			case tc.want != nil && got != nil && *got != *tc.want:
				t.Errorf("want %d, got %d", *tc.want, *got)
			}
		})
	}
}
