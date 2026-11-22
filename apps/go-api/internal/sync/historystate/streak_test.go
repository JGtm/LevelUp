package historystate_test

import (
	"context"
	"testing"

	"levelup/go-api/internal/sync/historystate"
	"levelup/go-api/internal/sync/syncmeta"
)

func TestStreak_AllerRetourEtAutreSujet(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	if got := historystate.LoadStreak(ctx, db, "m-50"); got.Attempts != 0 || got.Subject != "m-50" {
		t.Fatalf("base neuve : %+v, attendu un compte vide", got)
	}
	want := historystate.Streak{Subject: "m-50", Attempts: 2, Skipped: 1, LastError: "GetPlayerMatches: 500"}
	if !historystate.SaveStreak(ctx, db, want) {
		t.Fatal("SaveStreak doit rendre vrai")
	}
	if got := historystate.LoadStreak(ctx, db, "m-50"); got != want {
		t.Fatalf("LoadStreak = %+v, attendu %+v", got, want)
	}
	// Autre sujet : le point a bougé, le compte repart de zéro.
	if got := historystate.LoadStreak(ctx, db, "m-75"); got.Attempts != 0 || got.Skipped != 0 {
		t.Errorf("autre sujet : %+v, attendu un compte vide", got)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sync_meta WHERE key = ?`, historystate.StreakKey).Scan(&n); err != nil || n != 1 {
		t.Errorf("lignes du compte = %d (err %v), attendu 1", n, err)
	}
	if historystate.StreakKey != "history_failure_streak" {
		t.Errorf("StreakKey = %q : le nom de la clé est un contrat de données", historystate.StreakKey)
	}
	if historystate.Complete(ctx, db) {
		t.Error("ranger un compte d'échecs ne doit pas marquer l'historique complet")
	}
}

func TestClearStreak(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	if !historystate.ClearStreak(ctx, db) {
		t.Fatal("sans compte rangé : ClearStreak doit rendre vrai sans rien écrire")
	}
	historystate.SaveStreak(ctx, db, historystate.Streak{Subject: "m", Attempts: 3})
	if !historystate.ClearStreak(ctx, db) {
		t.Fatal("ClearStreak doit rendre vrai")
	}
	if got := historystate.LoadStreak(ctx, db, "m"); got.Attempts != 0 {
		t.Errorf("après effacement : %+v, attendu un compte vide", got)
	}
}

func TestLoadStreak_ValeurIllisibleOuEnEchec(t *testing.T) {
	ctx := context.Background()
	for name, raw := range map[string]string{"vide": "", "pas du JSON": "3", "négatif": `{"subject":"m","attempts":-2}`} {
		db := openMemDB(t, true)
		if err := syncmeta.Set(ctx, db, historystate.StreakKey, raw); err != nil {
			t.Fatal(err)
		}
		if got := historystate.LoadStreak(ctx, db, "m"); got.Attempts != 0 {
			t.Errorf("%s : %+v, attendu un compte vide", name, got)
		}
	}
	if got := historystate.LoadStreak(ctx, openMemDB(t, false), "m"); got.Attempts != 0 {
		t.Errorf("table absente : %+v, attendu un compte vide", got)
	}
	if historystate.SaveStreak(ctx, openMemDB(t, false), historystate.Streak{Subject: "m"}) || historystate.SaveStreak(ctx, nil, historystate.Streak{Subject: "m"}) {
		t.Error("écriture impossible : SaveStreak doit rendre faux")
	}
}
