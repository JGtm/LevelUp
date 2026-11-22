package historystate_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"levelup/go-api/internal/sync/historystate"
	"levelup/go-api/internal/sync/syncmeta"
)

// Sans marque-page rangé : pas de marque-page, la passe repart du début.
func TestLoadCursor_Absent(t *testing.T) {
	if _, ok := historystate.LoadCursor(context.Background(), openMemDB(t, true)); ok {
		t.Fatal("base neuve : aucun marque-page attendu")
	}
	if _, ok := historystate.LoadCursor(context.Background(), nil); ok {
		t.Fatal("sans base : aucun marque-page attendu")
	}
}

// Aller-retour : ce qui est rangé est relu à l'identique, une seule ligne, sans toucher à la clé de fin.
func TestSaveCursor_AllerRetour(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	want := historystate.Cursor{Offset: 400, LastMatchID: "m-400"}

	if !historystate.SaveCursor(ctx, db, want) {
		t.Fatal("SaveCursor doit rendre vrai")
	}
	got, ok := historystate.LoadCursor(ctx, db)
	if !ok || got != want {
		t.Fatalf("LoadCursor = (%+v, %v), attendu (%+v, true)", got, ok, want)
	}
	// Une mise à jour remplace la valeur (pas de seconde ligne).
	want = historystate.Cursor{Offset: 600, LastMatchID: "m-600"}
	if !historystate.SaveCursor(ctx, db, want) {
		t.Fatal("second SaveCursor : attendu vrai")
	}
	if got, ok := historystate.LoadCursor(ctx, db); !ok || got != want {
		t.Fatalf("après mise à jour : (%+v, %v), attendu (%+v, true)", got, ok, want)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sync_meta WHERE key = ?`, historystate.CursorKey).Scan(&n); err != nil || n != 1 {
		t.Errorf("lignes du marque-page = %d (err %v), attendu 1", n, err)
	}
	if historystate.Complete(ctx, db) {
		t.Error("ranger un marque-page ne doit pas marquer l'historique complet")
	}
	if historystate.CursorKey != "history_catchup_cursor" {
		t.Errorf("CursorKey = %q : le nom de la clé est un contrat de données", historystate.CursorKey)
	}
}

// Une position invalide n'est jamais rangée et laisse l'ancien marque-page en place.
func TestSaveCursor_PositionInvalide_Refusee(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	good := historystate.Cursor{Offset: 200, LastMatchID: "m-200"}
	historystate.SaveCursor(ctx, db, good)

	for name, c := range map[string]historystate.Cursor{
		"décalage nul":      {Offset: 0, LastMatchID: "x"},
		"décalage < 0":      {Offset: -3, LastMatchID: "x"},
		"identifiant vide":  {Offset: 10, LastMatchID: ""},
		"identifiant blanc": {Offset: 10, LastMatchID: "  "},
	} {
		if historystate.SaveCursor(ctx, db, c) {
			t.Errorf("%s : SaveCursor doit rendre faux", name)
		}
	}
	if got, ok := historystate.LoadCursor(ctx, db); !ok || got != good {
		t.Errorf("l'ancien marque-page doit rester : (%+v, %v)", got, ok)
	}
}

// Écriture impossible (table absente, base absente) : faux, sans panic.
func TestSaveCursor_EcritureEnEchec_RendFaux(t *testing.T) {
	c := historystate.Cursor{Offset: 5, LastMatchID: "m"}
	if historystate.SaveCursor(context.Background(), openMemDB(t, false), c) {
		t.Error("table absente : SaveCursor doit rendre faux")
	}
	if historystate.SaveCursor(context.Background(), nil, c) {
		t.Error("sans base : SaveCursor doit rendre faux")
	}
}

// Parseur tolérant : une valeur illisible ou incohérente vaut « pas de marque-page ».
func TestLoadCursor_ValeurIllisible_PasDeMarquePage(t *testing.T) {
	for name, raw := range map[string]string{
		"vide":               "",
		"blanc":              "   ",
		"pas du JSON":        "400:m-400",
		"JSON tronqué":       `{"offset":400,"last_match_`,
		"type inattendu":     `{"offset":"quatre cents","last_match_id":"m"}`,
		"décalage nul":       `{"offset":0,"last_match_id":"m"}`,
		"décalage négatif":   `{"offset":-1,"last_match_id":"m"}`,
		"identifiant absent": `{"offset":400}`,
		"tableau":            `[1,2]`,
	} {
		t.Run(name, func(t *testing.T) {
			db := openMemDB(t, true)
			ctx := context.Background()
			if err := syncmeta.Set(ctx, db, historystate.CursorKey, raw); err != nil {
				t.Fatal(err)
			}
			if c, ok := historystate.LoadCursor(ctx, db); ok {
				t.Errorf("valeur %q : marque-page %+v rendu, attendu aucun", raw, c)
			}
		})
	}
}

// Lecture en échec (table absente) : pas de marque-page.
func TestLoadCursor_LectureEnEchec(t *testing.T) {
	if _, ok := historystate.LoadCursor(context.Background(), openMemDB(t, false)); ok {
		t.Error("table absente : aucun marque-page attendu")
	}
}

// ClearCursor efface sans supprimer la ligne ; sans marque-page il n'écrit rien.
func TestClearCursor(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()

	if !historystate.ClearCursor(ctx, db) {
		t.Fatal("sans marque-page : ClearCursor doit rendre vrai")
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sync_meta WHERE key = ?`, historystate.CursorKey).Scan(&n); err != nil || n != 0 {
		t.Errorf("sans marque-page, ClearCursor ne doit rien écrire : lignes = %d (err %v)", n, err)
	}

	historystate.SaveCursor(ctx, db, historystate.Cursor{Offset: 3, LastMatchID: "m"})
	if !historystate.ClearCursor(ctx, db) {
		t.Fatal("ClearCursor doit rendre vrai")
	}
	if _, ok := historystate.LoadCursor(ctx, db); ok {
		t.Error("après ClearCursor : aucun marque-page attendu")
	}
	// Un nouveau marque-page se range après un effacement (la ligne vide est réutilisée).
	if !historystate.SaveCursor(ctx, db, historystate.Cursor{Offset: 9, LastMatchID: "n"}) {
		t.Fatal("SaveCursor après ClearCursor : attendu vrai")
	}
	if got, ok := historystate.LoadCursor(ctx, db); !ok || got.Offset != 9 {
		t.Errorf("relecture après réécriture : (%+v, %v)", got, ok)
	}

	if historystate.ClearCursor(ctx, openMemDB(t, false)) {
		t.Error("table absente : ClearCursor doit rendre faux")
	}
	if historystate.ClearCursor(ctx, nil) {
		t.Error("sans base : ClearCursor doit rendre faux")
	}
}

// MarkComplete efface le marque-page : fin rangée = plus de position à reprendre.
func TestMarkComplete_EffaceLeMarquePage(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	historystate.SaveCursor(ctx, db, historystate.Cursor{Offset: 1200, LastMatchID: "m-1200"})

	if !historystate.MarkComplete(ctx, db, "J900004") {
		t.Fatal("MarkComplete doit rendre vrai")
	}
	if _, ok := historystate.LoadCursor(ctx, db); ok {
		t.Error("après MarkComplete : le marque-page doit être effacé")
	}
	if !historystate.Complete(ctx, db) {
		t.Error("après MarkComplete : historique complet")
	}
}

// Les échecs de rattrapage se rangent à côté du marque-page : l'un ne fait jamais perdre l'autre.
func TestSaveFailures_ConserveLeMarquePage_Et_Reciproquement(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	fails := []historystate.Failure{{MatchID: "a", Attempts: 1}, {MatchID: "b", Attempts: 2}}

	if !historystate.SaveFailures(ctx, db, fails) {
		t.Fatal("SaveFailures doit rendre vrai")
	}
	if _, ok := historystate.LoadCursor(ctx, db); ok {
		t.Error("des échecs seuls ne font pas un marque-page")
	}
	if got := historystate.LoadCatchUp(ctx, db); got.Cursor != nil || len(got.Failures) != 2 || got.Failures[1] != fails[1] {
		t.Fatalf("LoadCatchUp = %+v, want deux échecs sans marque-page", got)
	}

	cur := historystate.Cursor{Offset: 200, LastMatchID: "m-200"}
	if !historystate.SaveCursor(ctx, db, cur) {
		t.Fatal("SaveCursor")
	}
	got := historystate.LoadCatchUp(ctx, db)
	if got.Cursor == nil || *got.Cursor != cur || len(got.Failures) != 2 {
		t.Fatalf("après SaveCursor : %+v, want marque-page et les deux échecs conservés", got)
	}

	if !historystate.SaveFailures(ctx, db, []historystate.Failure{{MatchID: "b", Attempts: 3}}) {
		t.Fatal("SaveFailures (mise à jour)")
	}
	got = historystate.LoadCatchUp(ctx, db)
	if got.Cursor == nil || *got.Cursor != cur || len(got.Failures) != 1 || got.Failures[0].Attempts != 3 {
		t.Fatalf("après mise à jour des échecs : %+v", got)
	}

	// Liste vide : les échecs s'effacent, le marque-page reste.
	historystate.SaveFailures(ctx, db, nil)
	got = historystate.LoadCatchUp(ctx, db)
	if got.Cursor == nil || len(got.Failures) != 0 {
		t.Fatalf("après liste vide : %+v, want le marque-page seul", got)
	}
}

// Sans marque-page ni échec rangés, une liste vide ne laisse qu'une valeur vide.
func TestSaveFailures_ListeVideSansMarquePage_ValeurVide(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	historystate.SaveFailures(ctx, db, []historystate.Failure{{MatchID: "a", Attempts: 1}})
	historystate.SaveFailures(ctx, db, nil)
	if v, ok, err := syncmeta.Get(ctx, db, historystate.CursorKey); err != nil || !ok || v != "" {
		t.Errorf("valeur = (%q, %v, %v), want une valeur vide", v, ok, err)
	}
	if got := historystate.LoadCatchUp(ctx, db); got.Cursor != nil || len(got.Failures) != 0 {
		t.Errorf("LoadCatchUp = %+v, want vide", got)
	}
}

// Borne : au plus MaxFailures entrées, les plus anciennes sortent.
func TestSaveFailures_BorneLesPlusAnciennesSortent(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	n := historystate.MaxFailures + 25
	in := make([]historystate.Failure, n)
	for i := range in {
		in[i] = historystate.Failure{MatchID: fmt.Sprintf("m%03d", i), Attempts: 1}
	}
	if !historystate.SaveFailures(ctx, db, in) {
		t.Fatal("SaveFailures")
	}
	got := historystate.LoadCatchUp(ctx, db).Failures
	if len(got) != historystate.MaxFailures {
		t.Fatalf("%d échecs rangés, want %d", len(got), historystate.MaxFailures)
	}
	if got[0] != in[25] || got[len(got)-1] != in[n-1] {
		t.Errorf("entrées gardées : %v ... %v, want les %d plus récentes (de %v à %v)", got[0], got[len(got)-1], historystate.MaxFailures, in[25], in[n-1])
	}
}

// Une entrée d'échec sans identifiant ou sans tentative est écartée à la lecture et à l'écriture.
func TestFailures_EntreesInvalidesEcartees(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	if err := syncmeta.Set(ctx, db, historystate.CursorKey,
		`{"failed":[{"id":"","attempts":2},{"id":"a","attempts":0},{"id":"b","attempts":2}]}`); err != nil {
		t.Fatal(err)
	}
	got := historystate.LoadCatchUp(ctx, db).Failures
	if len(got) != 1 || got[0].MatchID != "b" {
		t.Errorf("Failures = %v, want seulement b", got)
	}
	historystate.SaveFailures(ctx, db, []historystate.Failure{{MatchID: "", Attempts: 1}, {MatchID: "c", Attempts: 0}, {MatchID: "d", Attempts: 1}})
	got = historystate.LoadCatchUp(ctx, db).Failures
	if len(got) != 1 || got[0].MatchID != "d" {
		t.Errorf("après écriture : %v, want seulement d", got)
	}
}

// ClearCursor et MarkComplete effacent aussi les échecs.
func TestClearCursor_EtMarkComplete_EffacentLesEchecs(t *testing.T) {
	ctx := context.Background()
	for name, clear := range map[string]func(db *sql.DB){
		"ClearCursor":  func(db *sql.DB) { historystate.ClearCursor(ctx, db) },
		"MarkComplete": func(db *sql.DB) { historystate.MarkComplete(ctx, db, "alice") },
	} {
		db := openMemDB(t, true)
		historystate.SaveCursor(ctx, db, historystate.Cursor{Offset: 5, LastMatchID: "m"})
		historystate.SaveFailures(ctx, db, []historystate.Failure{{MatchID: "a", Attempts: 2}})
		clear(db)
		if got := historystate.LoadCatchUp(ctx, db); got.Cursor != nil || len(got.Failures) != 0 {
			t.Errorf("%s : état restant %+v, want vide", name, got)
		}
	}
}
