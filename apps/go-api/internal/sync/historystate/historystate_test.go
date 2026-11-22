package historystate_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/historystate"
	"levelup/go-api/internal/sync/syncmeta"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openMemDB ouvre un DuckDB en mémoire (une seule connexion : chaque connexion ouvrirait sinon
// sa propre base) ; withTable crée la table sync_meta d'une base joueur.
func openMemDB(t *testing.T, withTable bool) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if withTable {
		if _, err := db.Exec(`CREATE TABLE sync_meta (
			key VARCHAR PRIMARY KEY, value VARCHAR, updated_at TIMESTAMP)`); err != nil {
			t.Fatalf("create sync_meta: %v", err)
		}
	}
	return db
}

// Clé absente : l'historique n'a jamais été parcouru jusqu'au bout, la passe doit rattraper.
func TestComplete_CleAbsente_Incomplet(t *testing.T) {
	db := openMemDB(t, true)
	if mustComplete(t, context.Background(), db) {
		t.Fatal("base neuve : l'historique ne doit pas être réputé complet")
	}
}

// Clé rangée : complet ; la clé garde son nom historique (les joueurs déjà marqués le restent).
func TestMarkComplete_PuisComplete(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()

	if !historystate.MarkComplete(ctx, db, "J900004") {
		t.Fatal("MarkComplete doit rendre vrai quand la clé est rangée")
	}
	if !mustComplete(t, ctx, db) {
		t.Fatal("après MarkComplete, l'historique doit être réputé complet")
	}
	if historystate.Key != "history_complete_at" {
		t.Errorf("Key = %q : le nom de la clé est un contrat de données", historystate.Key)
	}
	if v, ok, err := syncmeta.Get(ctx, db, historystate.Key); err != nil || !ok || v == "" {
		t.Errorf("syncmeta.Get = (%q, %v, %v), attendu un horodatage", v, ok, err)
	}
	// Un second rangement met la clé à jour : une seule ligne.
	if !historystate.MarkComplete(ctx, db, "J900004") {
		t.Fatal("second MarkComplete : attendu vrai")
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sync_meta WHERE key = ?`, historystate.Key).Scan(&n); err != nil || n != 1 {
		t.Errorf("lignes de la clé = %d (err %v), attendu 1", n, err)
	}
}

// mustComplete lit l'état de l'historique et échoue sur une erreur de lecture.
func mustComplete(t *testing.T, ctx context.Context, db *sql.DB) bool {
	t.Helper()
	complete, err := historystate.Complete(ctx, db)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	return complete
}

// Lecture en échec (base sans table sync_meta, ou absente) : l'erreur est rendue et l'état n'est
// jamais « complet » (D-25 : l'appelant ne rattrape pas, n'écrit rien et garde le joueur en attente).
func TestComplete_LectureEnEchec_ErreurJamaisComplet(t *testing.T) {
	ctx := context.Background()
	if complete, err := historystate.Complete(ctx, openMemDB(t, false)); err == nil || complete {
		t.Errorf("lecture en échec : (%v, %v), attendu (faux, erreur)", complete, err)
	}
	complete, err := historystate.Complete(ctx, nil)
	if !errors.Is(err, historystate.ErrNoPlayerDB) || complete {
		t.Errorf("sans base : (%v, %v), attendu (faux, ErrNoPlayerDB)", complete, err)
	}
	// Poignée fermée (« sql: database is closed », constaté au rejeu du journal) : même règle.
	closed := openMemDB(t, true)
	_ = closed.Close()
	if complete, err := historystate.Complete(ctx, closed); err == nil || complete {
		t.Errorf("poignée fermée : (%v, %v), attendu (faux, erreur)", complete, err)
	}
}

// Écriture impossible : rendu faux (la passe reste « en attente »), sans panic.
func TestMarkComplete_EcritureEnEchec_RendFaux(t *testing.T) {
	ctx := context.Background()
	if historystate.MarkComplete(ctx, openMemDB(t, false), "J900004") {
		t.Error("table absente : MarkComplete doit rendre faux")
	}
	if historystate.MarkComplete(ctx, nil, "J900004") {
		t.Error("sans base : MarkComplete doit rendre faux")
	}
}

// Le budget par passe est le plafond par défaut d'une passe, sans valeur propre.
func TestPassBudget_EstLePlafondParDefaut(t *testing.T) {
	if got, want := historystate.PassBudget(), domain.DefaultSyncOptions().MaxMatches; got != want || got <= 0 {
		t.Errorf("PassBudget = %d, attendu %d (> 0)", got, want)
	}
	if historystate.MaxDepth <= historystate.PassBudget() {
		t.Errorf("MaxDepth = %d doit dépasser le budget d'une passe", historystate.MaxDepth)
	}
}
