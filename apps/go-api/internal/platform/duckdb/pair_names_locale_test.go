package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
)

// PairNamesIn : le nom d'une paire suit la langue du contexte ; une paire sans traduction, une langue de
// repli ou l'absence de metadata laissent le nom du registre.
func TestPairNamesIn_SuitLaLangueDuContexte(t *testing.T) {
	meta := metaDesVariantes(t)
	for _, q := range []string{
		`CREATE TABLE mode_name_tr (mode_en VARCHAR, lang VARCHAR, name VARCHAR)`,
		`INSERT INTO mode_name_tr VALUES ('Slayer','fr','Assassin')`,
	} {
		if _, err := meta.Exec(context.Background(), q); err != nil {
			t.Fatalf("seed %q: %v", q, err)
		}
	}
	refs := []PairRef{{Name: "Arena:Slayer on Bazaar"}, {Name: "Arena:Mystery on Bazaar"}, {}}

	fr := PairNamesIn(ctxkeys.WithLocale(context.Background(), "fr"), meta, refs)
	if fr[0] != "Assassin" || fr[1] != "Arena:Mystery on Bazaar" || fr[2] != "" {
		t.Errorf("fr : %q", fr)
	}
	en := PairNamesIn(ctxkeys.WithLocale(context.Background(), "en"), meta, refs)
	if en[0] != "Arena:Slayer on Bazaar" || en[1] != "Arena:Mystery on Bazaar" {
		t.Errorf("en : %q", en)
	}
	sans := PairNamesIn(ctxkeys.WithLocale(context.Background(), "fr"), nil, refs)
	if sans[0] != "Arena:Slayer on Bazaar" {
		t.Errorf("sans metadata : %q", sans)
	}
}
