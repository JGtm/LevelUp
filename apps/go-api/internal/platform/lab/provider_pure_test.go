// Package lab — tests unitaires des helpers PURS du diagnostic d'instance
// (zéro DB, zéro filesystem). Les chemins DB (listAllMedalEntries) sont
// integration-only et restent non couverts en unit par construction.
package lab

import (
	"errors"
	"testing"
)

func TestIsMissingRelationError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"catalog missing", errors.New("Catalog Error: Table with name waypoint_medals_raw does not exist"), true},
		{"other error", errors.New("IO Error: disk full"), false},
		{"catalog sans does-not-exist", errors.New("catalog error: ambiguous"), false},
	}
	for _, tc := range cases {
		if got := isMissingRelationError(tc.err); got != tc.want {
			t.Errorf("%s: isMissingRelationError = %v, want %v", tc.name, got, tc.want)
		}
	}
}
