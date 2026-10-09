package replayverite

import (
	"errors"
	"testing"
)

// TestLireOracleEtRegistre_VideOuIllisibleEstRefuse : un oracle ou un registre vide ne juge rien,
// et le lire comme « aucun ecart » serait faux.
func TestLireOracleEtRegistre_VideOuIllisibleEstRefuse(t *testing.T) {
	if _, err := LireOracle([]byte(`{"matchId":"m"}`)); !errors.Is(err, ErrOracleVide) {
		t.Errorf("oracle vide : err = %v, veut ErrOracleVide", err)
	}
	if _, err := LireOracle([]byte(`{`)); err == nil {
		t.Error("oracle illisible accepte")
	}
	o, err := LireOracle([]byte(`{"matchId":"m","players":[{"xuid":"111","personalScore":300}]}`))
	if err != nil || *o.Players[0].PersonalScore != 300 {
		t.Errorf("oracle = %+v, err = %v", o, err)
	}
	if _, err := LireRegistre([]byte(`{}`)); err == nil {
		t.Error("registre vide accepte")
	}
	reg, err := LireRegistre([]byte(`{"repli_a":true,"repli_b":false}`))
	if err != nil || !reg["repli_a"] || reg["repli_b"] {
		t.Errorf("registre = %v, err = %v", reg, err)
	}
}
