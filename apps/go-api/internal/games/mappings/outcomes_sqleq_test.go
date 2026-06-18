package mappings

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// haloLikeSet construit un set d'issues aux codes Halo (win=2, loss=3, tie=1,
// dnf=4) pour prouver la parité byte-identique des expressions SQL.
func haloLikeSet() *OutcomeMappingSet {
	return NewOutcomeMappingSet("halo_infinite", 1, map[string]OutcomeMapping{
		"win":  {Key: "win", RawCode: 2},
		"loss": {Key: "loss", RawCode: 3},
		"tie":  {Key: "tie", RawCode: 1},
		"dnf":  {Key: "dnf", RawCode: 4},
	})
}

func TestSQLEqExpr_HaloByteIdentical(t *testing.T) {
	s := haloLikeSet()
	cases := []struct {
		o    canonical.Outcome
		want string
	}{
		{canonical.OutcomeWin, "outcome = 2"},
		{canonical.OutcomeLoss, "outcome = 3"},
		{canonical.OutcomeTie, "outcome = 1"},
		{canonical.OutcomeDNF, "outcome = 4"},
	}
	for _, c := range cases {
		got, ok := s.SQLEqExpr("outcome", c.o)
		if !ok || got != c.want {
			t.Errorf("SQLEqExpr(%s) = %q,%v ; want %q,true (parité littéral legacy)", c.o, got, ok, c.want)
		}
	}
	// SQLIsWinExpr délègue → même résultat que SQLEqExpr(win).
	if w, ok := s.SQLIsWinExpr("outcome"); !ok || w != "outcome = 2" {
		t.Errorf("SQLIsWinExpr = %q,%v ; want outcome = 2,true", w, ok)
	}
}

func TestSQLEqExpr_RoutesDistinctCodes(t *testing.T) {
	// Un 2e titre aux codes distincts route VRAIMENT (preuve non cosmétique).
	s := NewOutcomeMappingSet("synthetic_x", 1, map[string]OutcomeMapping{
		"win":  {Key: "win", RawCode: 5},
		"loss": {Key: "loss", RawCode: 6},
	})
	if got, ok := s.SQLEqExpr("outcome", canonical.OutcomeWin); !ok || got != "outcome = 5" {
		t.Errorf("routing win = %q,%v ; want outcome = 5", got, ok)
	}
	if got, ok := s.SQLEqExpr("outcome", canonical.OutcomeLoss); !ok || got != "outcome = 6" {
		t.Errorf("routing loss = %q,%v ; want outcome = 6", got, ok)
	}
}

func TestSQLEqExpr_DegradesWhenRawCodeAbsent(t *testing.T) {
	// Titre sans raw_code (cf. synthetic_title_b/outcomes.toml) → ok=false → le
	// caller retombe sur son littéral legacy (jamais le code d'un autre titre).
	s := NewOutcomeMappingSet("no_codes", 1, map[string]OutcomeMapping{
		"win": {Key: "win"}, // RawCode 0 = absent
	})
	if _, ok := s.SQLEqExpr("outcome", canonical.OutcomeWin); ok {
		t.Error("SQLEqExpr devrait dégrader (ok=false) quand raw_code est absent")
	}
}
