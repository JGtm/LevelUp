package historystate_test

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/sync/historystate"
	"levelup/go-api/internal/sync/syncmeta"
)

func TestSkipSet_AllerRetourSansDoublon(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	if got := historystate.LoadSkipSet(ctx, db); len(got) != 0 {
		t.Fatalf("base neuve : %v, attendu un ensemble vide", got)
	}
	if !historystate.AddSkips(ctx, db, []historystate.Skip{
		{MatchID: "m-1", Reason: historystate.SkipNotFound},
		{MatchID: "m-2", Reason: historystate.SkipAbandoned},
		{MatchID: "m-1", Reason: historystate.SkipAbandoned},
		{MatchID: " ", Reason: historystate.SkipNotFound},
	}) {
		t.Fatal("AddSkips doit rendre vrai")
	}
	if got := historystate.LoadSkipSet(ctx, db); len(got) != 2 || !got["m-1"] || !got["m-2"] {
		t.Fatalf("LoadSkipSet = %v, attendu m-1 et m-2", got)
	}
	// Deuxième écriture : s'ajoute à la suite, un doublon ne change rien.
	historystate.AddSkips(ctx, db, []historystate.Skip{{MatchID: "m-3", Reason: historystate.SkipNotFound}, {MatchID: "m-2"}})
	if got := historystate.LoadSkipSet(ctx, db); len(got) != 3 || !got["m-3"] {
		t.Fatalf("après ajout : %v, attendu 3 matchs", got)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sync_meta WHERE key = ?`, historystate.SkipSetKey).Scan(&n); err != nil || n != 1 {
		t.Errorf("lignes de l'ensemble = %d (err %v), attendu 1", n, err)
	}
	if historystate.SkipSetKey != "history_skip_set" {
		t.Errorf("SkipSetKey = %q : le nom de la clé est un contrat de données", historystate.SkipSetKey)
	}
}

// Rien de neuf à ranger : aucune écriture (la valeur, donc son horodatage, ne bouge pas).
func TestAddSkips_RienDeNeuf_NEcritRien(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	if !historystate.AddSkips(ctx, db, nil) {
		t.Fatal("liste vide : AddSkips doit rendre vrai")
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sync_meta`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("liste vide : %d lignes écrites (err %v), attendu 0", n, err)
	}
	historystate.AddSkips(ctx, db, []historystate.Skip{{MatchID: "m-1", Reason: historystate.SkipNotFound}})
	before, _, _ := syncmeta.Get(ctx, db, historystate.SkipSetKey)
	if !historystate.AddSkips(ctx, db, []historystate.Skip{{MatchID: "m-1", Reason: historystate.SkipNotFound}}) {
		t.Fatal("doublon : AddSkips doit rendre vrai")
	}
	after, _, _ := syncmeta.Get(ctx, db, historystate.SkipSetKey)
	if before != after {
		t.Errorf("un doublon a réécrit la valeur : %q -> %q", before, after)
	}
}

// Borne de 2 000 entrées : les plus anciennes sortent, les plus récentes restent.
func TestAddSkips_BorneDe2000_LesPlusAnciennesSortent(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	if historystate.MaxSkipSet != 2000 {
		t.Fatalf("MaxSkipSet = %d, attendu 2000", historystate.MaxSkipSet)
	}
	var first []historystate.Skip
	for i := 0; i < historystate.MaxSkipSet; i++ {
		first = append(first, historystate.Skip{MatchID: fmt.Sprintf("m-%04d", i), Reason: historystate.SkipNotFound})
	}
	if !historystate.AddSkips(ctx, db, first) {
		t.Fatal("AddSkips doit rendre vrai")
	}
	if got := historystate.LoadSkipSet(ctx, db); len(got) != historystate.MaxSkipSet {
		t.Fatalf("à la borne : %d entrées, attendu %d", len(got), historystate.MaxSkipSet)
	}
	historystate.AddSkips(ctx, db, []historystate.Skip{
		{MatchID: "neuf-1", Reason: historystate.SkipAbandoned}, {MatchID: "neuf-2", Reason: historystate.SkipAbandoned},
	})
	got := historystate.LoadSkipSet(ctx, db)
	if len(got) != historystate.MaxSkipSet {
		t.Fatalf("au-delà de la borne : %d entrées, attendu %d", len(got), historystate.MaxSkipSet)
	}
	if got["m-0000"] || got["m-0001"] {
		t.Error("les deux plus anciennes entrées doivent être sorties")
	}
	if !got["m-0002"] || !got["neuf-1"] || !got["neuf-2"] {
		t.Error("les entrées récentes doivent rester")
	}
}

// La fin d'historique efface le marque-page, pas l'ensemble des matchs à ne plus demander.
func TestSkipSet_MarkCompleteNEffacePasLEnsemble(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	historystate.AddSkips(ctx, db, []historystate.Skip{{MatchID: "m-1", Reason: historystate.SkipNotFound}})
	historystate.SaveCursor(ctx, db, historystate.Cursor{Offset: 10, LastMatchID: "m-10"})
	if !historystate.MarkComplete(ctx, db, "gt") {
		t.Fatal("MarkComplete doit rendre vrai")
	}
	if _, ok := historystate.LoadCursor(ctx, db); ok {
		t.Error("le marque-page doit être effacé")
	}
	if got := historystate.LoadSkipSet(ctx, db); !got["m-1"] {
		t.Errorf("après la fin d'historique : %v, m-1 doit rester à ne plus demander", got)
	}
}

func TestSkipSet_ValeurIllisibleEnEchecEtBaseAbsente(t *testing.T) {
	ctx := context.Background()
	for name, raw := range map[string]string{"vide": "", "pas du JSON": "3", "forme inconnue": `{"skipped":"x"}`} {
		db := openMemDB(t, true)
		if err := syncmeta.Set(ctx, db, historystate.SkipSetKey, raw); err != nil {
			t.Fatal(err)
		}
		if got := historystate.LoadSkipSet(ctx, db); len(got) != 0 {
			t.Errorf("%s : %v, attendu un ensemble vide", name, got)
		}
		// Une valeur illisible est remplacée à la prochaine écriture.
		if !historystate.AddSkips(ctx, db, []historystate.Skip{{MatchID: "m-9", Reason: historystate.SkipNotFound}}) {
			t.Errorf("%s : AddSkips doit remplacer la valeur", name)
		}
		if got := historystate.LoadSkipSet(ctx, db); !got["m-9"] || len(got) != 1 {
			t.Errorf("%s : après écriture %v, attendu m-9 seul", name, got)
		}
	}
	// Lecture en échec (table absente) : rien n'est écrit, l'ensemble rangé ne peut pas être écrasé.
	horsTable := openMemDB(t, false)
	if got := historystate.LoadSkipSet(ctx, horsTable); len(got) != 0 {
		t.Errorf("table absente : %v, attendu un ensemble vide", got)
	}
	if historystate.AddSkips(ctx, horsTable, []historystate.Skip{{MatchID: "m"}}) {
		t.Error("lecture en échec : AddSkips doit rendre faux")
	}
	if historystate.AddSkips(ctx, nil, []historystate.Skip{{MatchID: "m"}}) {
		t.Error("base absente : AddSkips doit rendre faux")
	}
	if historystate.LoadSkipSet(ctx, nil) != nil {
		t.Error("base absente : LoadSkipSet doit rendre nil")
	}
}
