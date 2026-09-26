//go:build cgo

package migration

// compaction_rewrite_test.go — DC.5 sur le DuckDB embarqué (1.5.x) : la recopie d'une base par
// COPY FROM DATABASE préserve-t-elle ce que la réécriture de fichier doit préserver ? Tables,
// vues (dont une vue sur vue), index, macro, séquences AVEC leur valeur courante (utilisée,
// jamais utilisée, START non trivial), comptes et empreintes des vues du registre — chaque
// fichier relu SEUL.

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func ouvrirFichier(t *testing.T, chemin string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", chemin)
	if err != nil {
		t.Fatalf("open %s: %v", chemin, err)
	}
	return db
}

func TestCopierBaseVers_PreserveCatalogueSequencesEtVues(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	src := ouvrirFichier(t, filepath.Join(dir, "src.duckdb"))
	seedBombStats(t, src)
	for _, q := range []string{
		`CREATE SEQUENCE s_jamais_tiree`,
		`CREATE SEQUENCE s_depart START 10`,
		`SELECT nextval('s_depart')`,
		`CREATE VIEW v_sur_vue AS SELECT match_id, count(*) AS n FROM match_bomb_stats_latest GROUP BY 1`,
		`CREATE MACRO plus_un(a) AS a + 1`,
	} {
		mustExec(t, src, q)
	}
	if _, err := CompactSupersededPasses(ctx, src, false); err != nil {
		t.Fatalf("compaction: %v", err)
	}
	avant, err := LireInventaire(ctx, src)
	if err != nil {
		t.Fatalf("inventaire source: %v", err)
	}
	var maxID int64
	if err := src.QueryRow(`SELECT MAX(id) FROM match_bomb_stats`).Scan(&maxID); err != nil {
		t.Fatal(err)
	}
	cible := filepath.Join(dir, "neuf.duckdb")
	if err := CopierBaseVers(ctx, src, cible); err != nil {
		t.Fatalf("copie: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Fatal(err)
	}

	neuf := ouvrirFichier(t, cible)
	defer neuf.Close() //nolint:errcheck
	apres, err := LireInventaire(ctx, neuf)
	if err != nil {
		t.Fatalf("inventaire du fichier neuf: %v", err)
	}
	if ecarts := avant.Ecarts(apres); len(ecarts) > 0 {
		t.Fatalf("la réécriture ne préserve pas la base :\n%s", strings.Join(ecarts, "\n"))
	}
	if len(apres.Vues) != 1 || apres.Comptes["match_bomb_stats"] != 3 {
		t.Fatalf("inventaire inattendu : %+v", apres)
	}

	// Les séquences continuent, y compris celle qui n'a jamais tiré et celle à START 10.
	var id, jamais, depart int64
	if err := neuf.QueryRow(`INSERT INTO match_bomb_stats (match_id, xuid) VALUES ('m9', 'x9')
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("insert dans le fichier neuf: %v", err)
	}
	if err := neuf.QueryRow(`SELECT nextval('s_jamais_tiree'), nextval('s_depart')`).Scan(&jamais, &depart); err != nil {
		t.Fatal(err)
	}
	if id != maxID+1 || jamais != 1 || depart != 11 {
		t.Fatalf("séquences après réécriture : id=%d (attendu %d), jamais=%d (1), depart=%d (11)",
			id, maxID+1, jamais, depart)
	}
	var plus int
	if err := neuf.QueryRow(`SELECT plus_un(1)`).Scan(&plus); err != nil || plus != 2 {
		t.Fatalf("macro perdue : %d, %v", plus, err)
	}
}

// TestInventaire_EcartsDetectes : le témoin — un inventaire qui diffère d'une ligne, d'un objet
// ou d'une empreinte n'est pas déclaré identique.
func TestInventaire_EcartsDetectes(t *testing.T) {
	a := Inventaire{Objets: []string{"table t x"}, Comptes: map[string]int64{"t": 3},
		Vues: map[string]EmpreinteVue{"t_latest": {Lignes: 3, Somme: "9"}}}
	b := Inventaire{Objets: []string{"table t y"}, Comptes: map[string]int64{"t": 2},
		Vues: map[string]EmpreinteVue{"t_latest": {Lignes: 3, Somme: "8"}}}
	if n := len(a.Ecarts(b)); n != 3 {
		t.Fatalf("%d écarts, attendu 3 : %v", n, a.Ecarts(b))
	}
	if n := len(a.Ecarts(a)); n != 0 {
		t.Fatalf("inventaire différent de lui-même : %v", a.Ecarts(a))
	}
}
