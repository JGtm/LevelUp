package analysis

import "testing"

func TestLongestRun(t *testing.T) {
	isTrue := func(b bool) bool { return b }

	cases := []struct {
		name     string
		items    []bool
		wantLen  int
		wantStrt int
	}{
		{"vide", nil, 0, 0},
		{"aucun match", []bool{false, false}, 0, 0},
		{"tout match", []bool{true, true, true}, 3, 0},
		{"run au milieu", []bool{false, true, true, false, true}, 2, 1},
		{"premier gagne en cas d'égalité", []bool{true, true, false, true, true}, 2, 0},
		{"run final plus long", []bool{true, false, true, true, true}, 3, 2},
		{"singleton", []bool{false, true, false}, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotLen, gotStart := LongestRun(tc.items, isTrue)
			if gotLen != tc.wantLen || gotStart != tc.wantStrt {
				t.Fatalf("LongestRun(%v) = (%d, %d) ; want (%d, %d)",
					tc.items, gotLen, gotStart, tc.wantLen, tc.wantStrt)
			}
		})
	}
}
