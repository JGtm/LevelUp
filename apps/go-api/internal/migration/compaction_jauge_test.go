//go:build cgo

package migration

// compaction_jauge_test.go — la jauge « passes par match » (compaction_jauge.go) : mesure en
// lecture seule, rapport 1 exactement après compaction, table absente, table hors registre.

import (
	"context"
	"testing"
)

// TestJaugePasses_AvantEtApresCompaction : 6 lignes brutes dont 3 gardées → rapport 2, au-delà
// du seuil ; la mesure n'écrit rien ; après compaction le rapport vaut 1, sous le seuil.
func TestJaugePasses_AvantEtApresCompaction(t *testing.T) {
	db := openTmpDB(t)
	seedBombStats(t, db)
	ctx := context.Background()

	j, presente, err := MesurerPasses(ctx, db, "match_bomb_stats")
	if err != nil || !presente {
		t.Fatalf("MesurerPasses: presente=%v err=%v", presente, err)
	}
	if j.Brutes != 6 || j.Gardees != 3 || j.Rapport() != 2 || !j.AuDessusDuSeuil() {
		t.Fatalf("jauge avant = %+v (rapport %.2f), attendu 6/3, rapport 2, au-delà du seuil", j, j.Rapport())
	}
	if got := countRows(t, db, "match_bomb_stats"); got != 6 {
		t.Fatalf("la mesure a écrit : %d lignes brutes, attendu 6", got)
	}

	if _, err := CompactSupersededPasses(ctx, db, false); err != nil {
		t.Fatalf("compaction: %v", err)
	}
	j, _, err = MesurerPasses(ctx, db, "match_bomb_stats")
	if err != nil {
		t.Fatalf("MesurerPasses après: %v", err)
	}
	if j.Brutes != 3 || j.Gardees != 3 || j.Rapport() != 1 || j.AuDessusDuSeuil() {
		t.Fatalf("jauge après = %+v, attendu 3/3, rapport 1, sous le seuil", j)
	}
}

// TestJaugePasses_TableAbsenteEtHorsRegistre : une table du registre absente de la base n'est pas
// une erreur (presente=false) ; une table hors registre en est une.
func TestJaugePasses_TableAbsenteEtHorsRegistre(t *testing.T) {
	db := openTmpDB(t)
	ctx := context.Background()
	if _, presente, err := MesurerPasses(ctx, db, "match_kill_events"); err != nil || presente {
		t.Fatalf("table absente : presente=%v err=%v, attendu false, nil", presente, err)
	}
	if _, _, err := MesurerPasses(ctx, db, "match_registry"); err == nil {
		t.Fatal("table hors registre : erreur attendue")
	}
}

// TestJaugePasses_Rapport : une table vide vaut 1 ; le seuil est strict.
func TestJaugePasses_Rapport(t *testing.T) {
	cas := []struct {
		j        JaugePasses
		rapport  float64
		auDessus bool
	}{
		{JaugePasses{}, 1, false},
		{JaugePasses{Brutes: 3, Gardees: 2}, 1.5, false},
		{JaugePasses{Brutes: 151, Gardees: 100}, 1.51, true},
	}
	for _, c := range cas {
		if c.j.Rapport() != c.rapport || c.j.AuDessusDuSeuil() != c.auDessus {
			t.Errorf("%+v : rapport %.2f au-dessus %v, attendu %.2f %v",
				c.j, c.j.Rapport(), c.j.AuDessusDuSeuil(), c.rapport, c.auDessus)
		}
	}
}
