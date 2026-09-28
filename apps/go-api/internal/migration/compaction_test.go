//go:build cgo

package migration

// compaction_test.go — la MÉCANIQUE de la compaction (compaction.go) et du cœur commun
// (table_swap.go), sur la table réelle `match_bomb_stats` (DDL et vue de sa migration) : garde de
// cardinalité, vérification avant COMMIT, orphelin, refus d'une vue dont la règle a changé,
// dry-run, table absente. Le bout-à-bout sur le schéma partagé COMPLET (les 14 tables, les
// séquences, les index) vit dans internal/games/halo_infinite/migrations/compaction_e2e_test.go.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// seedBombStats : la table réelle et 6 lignes — (m1,x1) en 3 versions, (m1,x2) en 1, (m2,x1) en
// 2 : 3 lignes servies par la vue, 3 versions mortes.
func seedBombStats(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := applyMatchBombStats(db); err != nil {
		t.Fatalf("applyMatchBombStats: %v", err)
	}
	lignes := []struct {
		match, xuid string
		arms, sec   int
	}{
		{"m1", "x1", 1, 1}, {"m1", "x1", 2, 2}, {"m1", "x2", 7, 3},
		{"m2", "x1", 4, 4}, {"m1", "x1", 3, 5}, {"m2", "x1", 5, 6},
	}
	for _, l := range lignes {
		if _, err := db.Exec(`INSERT INTO match_bomb_stats (match_id, xuid, bomb_arms, written_at)
			VALUES (?, ?, ?, TIMESTAMP '2026-01-01 00:00:00' + to_seconds(?))`,
			l.match, l.xuid, l.arms, l.sec); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

// lireVueOrdonnee : la lecture complète et ordonnée d'une relation, ligne par ligne.
func lireVueOrdonnee(t *testing.T, db *sql.DB, rel string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT * FROM ` + rel + ` ORDER BY id`)
	if err != nil {
		t.Fatalf("select %s: %v", rel, err)
	}
	defer rows.Close() //nolint:errcheck
	cols, _ := rows.Columns()
	var out []string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("scan %s: %v", rel, err)
		}
		out = append(out, fmt.Sprint(vals...))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows %s: %v", rel, err)
	}
	return out
}

func rapportDe(t *testing.T, rs []CompactionTable, table string) CompactionTable {
	t.Helper()
	for _, r := range rs {
		if r.Table == table {
			return r
		}
	}
	t.Fatalf("table %s absente du rapport", table)
	return CompactionTable{}
}

// TestCompaction_BombStats_VueIdentiqueEtTablesAbsentes : la table réelle est compactée à ses
// seules lignes servies, la vue rend le même résultat à l'octet, et les douze autres tables du
// registre, absentes de cette base, sont sautées sans erreur.
func TestCompaction_BombStats_VueIdentiqueEtTablesAbsentes(t *testing.T) {
	db := openTmpDB(t)
	seedBombStats(t, db)
	avant := lireVueOrdonnee(t, db, "match_bomb_stats_latest")

	rs, err := CompactSupersededPasses(context.Background(), db, false)
	if err != nil {
		t.Fatalf("compaction: %v", err)
	}
	r := rapportDe(t, rs, "match_bomb_stats")
	if r.Status != CompactionCompactee || r.Raw != 6 || r.Kept != 3 || r.After != 3 || r.Latest != 3 {
		t.Fatalf("rapport = %+v, attendu compactee 6 -> 3", r)
	}
	if got := countRows(t, db, "match_bomb_stats"); got != 3 {
		t.Fatalf("brutes après = %d, attendu 3", got)
	}
	if apres := lireVueOrdonnee(t, db, "match_bomb_stats_latest"); strings.Join(apres, "\n") != strings.Join(avant, "\n") {
		t.Fatalf("vue changée :\navant %v\naprès %v", avant, apres)
	}
	for _, x := range rs {
		if x.Table != "match_bomb_stats" && x.Status != CompactionAbsente {
			t.Fatalf("%s : statut %q, attendu absente", x.Table, x.Status)
		}
	}
}

// TestCompaction_DryRunNEcritRien : --dry-run mesure et n'écrit rien.
func TestCompaction_DryRunNEcritRien(t *testing.T) {
	db := openTmpDB(t)
	seedBombStats(t, db)
	rs, err := CompactSupersededPasses(context.Background(), db, true)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	r := rapportDe(t, rs, "match_bomb_stats")
	if r.Status != CompactionACompacter || r.Raw != 6 || r.Kept != 3 || r.After != 3 {
		t.Fatalf("rapport dry-run = %+v", r)
	}
	if got := countRows(t, db, "match_bomb_stats"); got != 6 {
		t.Fatalf("dry-run a écrit : %d lignes, attendu 6", got)
	}
}

// TestCompaction_RegleDeVueChangee_Refus : une vue dont la règle ne correspond plus au registre
// fait REFUSER la table (erreur), sans rien écrire.
func TestCompaction_RegleDeVueChangee_Refus(t *testing.T) {
	db := openTmpDB(t)
	seedBombStats(t, db)
	mustExec(t, db, `CREATE OR REPLACE VIEW match_bomb_stats_latest AS SELECT * FROM match_bomb_stats
		QUALIFY row_number() OVER (PARTITION BY match_id ORDER BY written_at DESC, id DESC) = 1`)
	_, err := CompactSupersededPasses(context.Background(), db, false)
	if err == nil || !strings.Contains(err.Error(), "ne porte plus la règle") {
		t.Fatalf("attendu un refus de la table, got %v", err)
	}
	if got := countRows(t, db, "match_bomb_stats"); got != 6 {
		t.Fatalf("table touchée malgré le refus : %d lignes", got)
	}
}

// TestSwapTableTx_GardeDeCardinalite_Rollback : une table neuve qui n'a pas le compte attendu
// fait abandonner l'échange AVANT le DROP — table d'origine intacte, rien de committé.
func TestSwapTableTx_GardeDeCardinalite_Rollback(t *testing.T) {
	db := openTmpDB(t)
	seedBombStats(t, db)
	_, err := swapTableTx(context.Background(), db, tableSwap{
		Table: "match_bomb_stats", Suffix: compactSuffix,
		Expected: `SELECT 99`,
		Build:    []string{`CREATE TABLE match_bomb_stats__compact AS SELECT * FROM match_bomb_stats`},
	})
	if err == nil || !strings.Contains(err.Error(), "abandonné") {
		t.Fatalf("attendu l'abandon de la garde, got %v", err)
	}
	if got := countRows(t, db, "match_bomb_stats"); got != 6 {
		t.Fatalf("lignes après rollback = %d, attendu 6", got)
	}
	if has, _ := tableExists(db, "match_bomb_stats__compact"); has {
		t.Fatal("table de construction committée malgré le rollback")
	}
}

// TestSwapTableTx_EchecApresLeDrop_Rollback : le « crash en cours de swap » simulé — une erreur
// APRÈS le DROP et le RENAME (vérification finale) annule tout : table, lignes, index intacts.
func TestSwapTableTx_EchecApresLeDrop_Rollback(t *testing.T) {
	db := openTmpDB(t)
	seedBombStats(t, db)
	ctx := context.Background()
	ddlAvant, _ := ddlDeTable(ctx, db, "match_bomb_stats")
	indexAvant, _ := ddlDesIndex(ctx, db, "match_bomb_stats")
	creer, err := ddlDeConstruction(ddlAvant, "match_bomb_stats")
	if err != nil {
		t.Fatal(err)
	}
	panne := errors.New("panne simulée entre le RENAME et le COMMIT")
	_, err = swapTableTx(ctx, db, tableSwap{
		Table: "match_bomb_stats", Suffix: compactSuffix,
		Expected: `SELECT COUNT(*) FROM match_bomb_stats`,
		Build: []string{creer,
			`INSERT INTO match_bomb_stats__compact SELECT * FROM match_bomb_stats`},
		PostRename: indexAvant,
		Verify:     func(context.Context, *sql.Tx) error { return panne },
	})
	if !errors.Is(err, panne) {
		t.Fatalf("attendu la panne simulée, got %v", err)
	}
	if got := countRows(t, db, "match_bomb_stats"); got != 6 {
		t.Fatalf("lignes après rollback = %d, attendu 6", got)
	}
	ddlApres, _ := ddlDeTable(ctx, db, "match_bomb_stats")
	indexApres, _ := ddlDesIndex(ctx, db, "match_bomb_stats")
	if ddlApres != ddlAvant || strings.Join(indexApres, ";") != strings.Join(indexAvant, ";") {
		t.Fatalf("schéma changé malgré le rollback :\n%s %v\n%s %v", ddlAvant, indexAvant, ddlApres, indexApres)
	}
}

// TestCompaction_OrphelinRefuse : l'état qu'un swap non transactionnel laisserait (table absente,
// `__compact` présente, SANS les index secondaires que seul le PostRename pose) n'est pas
// « réparé » : renommer l'orpheline perdrait `idx_match_bomb_stats_match` pour de bon. Refus, état
// laissé tel quel pour que l'exploitant remette la sauvegarde. Si une version de la commande
// récupère quand même, elle doit rendre les index d'origine (garde de l'ancienne récupération).
func TestCompaction_OrphelinRefuse(t *testing.T) {
	db := openTmpDB(t)
	seedBombStats(t, db)
	ctx := context.Background()
	indexAvant, _ := ddlDesIndex(ctx, db, "match_bomb_stats")
	if len(indexAvant) == 0 {
		t.Fatal("fixture : match_bomb_stats doit porter un index secondaire")
	}
	ddl, _ := ddlDeTable(ctx, db, "match_bomb_stats")
	creer, err := ddlDeConstruction(ddl, "match_bomb_stats")
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, creer)
	mustExec(t, db, `INSERT INTO match_bomb_stats__compact SELECT * FROM match_bomb_stats`)
	mustExec(t, db, `DROP TABLE match_bomb_stats`)

	for _, dryRun := range []bool{true, false} {
		_, err := CompactSupersededPasses(ctx, db, dryRun)
		if err == nil {
			if index, _ := ddlDesIndex(ctx, db, "match_bomb_stats"); strings.Join(index, ";") != strings.Join(indexAvant, ";") {
				t.Fatalf("orpheline récupérée SANS ses index : avant %v, après %v", indexAvant, index)
			}
		}
		if err == nil || !strings.Contains(err.Error(), "orpheline") {
			t.Fatalf("dry-run=%v : attendu un refus de l'orpheline, got %v", dryRun, err)
		}
	}
	if has, _ := tableExists(db, "match_bomb_stats__compact"); !has {
		t.Fatal("l'orpheline a été touchée malgré le refus")
	}
	if has, _ := tableExists(db, "match_bomb_stats"); has {
		t.Fatal("une table match_bomb_stats est apparue malgré le refus")
	}
}

// TestCompaction_VueQuiRetientPlusQueLaRegle_RollbackIntegral : une vue qui porte la signature du
// registre mais retient PLUS de lignes que la règle (`… = 1 OR bomb_arms = 1` garde une version
// ancienne) — la compaction lui retirerait une ligne servie. La vérification avant COMMIT doit le
// voir et tout annuler : lignes, DDL et index intacts, aucune table de construction.
func TestCompaction_VueQuiRetientPlusQueLaRegle_RollbackIntegral(t *testing.T) {
	db := openTmpDB(t)
	seedBombStats(t, db)
	ctx := context.Background()
	mustExec(t, db, `CREATE OR REPLACE VIEW match_bomb_stats_latest AS SELECT * FROM match_bomb_stats
		QUALIFY row_number() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1
		OR bomb_arms = 1`)
	vueAvant := lireVueOrdonnee(t, db, "match_bomb_stats_latest")
	ddlAvant, _ := ddlDeTable(ctx, db, "match_bomb_stats")
	indexAvant, _ := ddlDesIndex(ctx, db, "match_bomb_stats")

	_, err := CompactSupersededPasses(ctx, db, false)
	if err == nil || !strings.Contains(err.Error(), "ne rend plus le même résultat") {
		t.Fatalf("attendu le refus de la vérification avant COMMIT, got %v", err)
	}
	if got := countRows(t, db, "match_bomb_stats"); got != 6 {
		t.Fatalf("lignes après rollback = %d, attendu 6", got)
	}
	ddl, _ := ddlDeTable(ctx, db, "match_bomb_stats")
	index, _ := ddlDesIndex(ctx, db, "match_bomb_stats")
	if ddl != ddlAvant || strings.Join(index, ";") != strings.Join(indexAvant, ";") {
		t.Fatalf("schéma changé malgré le rollback :\n%s %v\n%s %v", ddlAvant, indexAvant, ddl, index)
	}
	if vue := lireVueOrdonnee(t, db, "match_bomb_stats_latest"); strings.Join(vue, "\n") != strings.Join(vueAvant, "\n") {
		t.Fatalf("vue changée malgré le rollback")
	}
	if has, _ := tableExists(db, "match_bomb_stats__compact"); has {
		t.Fatal("table de construction committée malgré le rollback")
	}
}

// TestVerifierApresEchange_EcartsDeDDLEtDIndex : la vérification lit le VRAI catalogue et refuse
// un DDL ou une liste d'index qui ne sont plus ceux d'avant (ce que rendrait une construction qui
// perd un défaut ou un index) ; elle accepte le schéma identique.
func TestVerifierApresEchange_EcartsDeDDLEtDIndex(t *testing.T) {
	db := openTmpDB(t)
	seedBombStats(t, db)
	ctx := context.Background()
	var c compactable
	for _, x := range tablesCompactables {
		if x.Table == "match_bomb_stats" {
			c = x
		}
	}
	ddl, _ := ddlDeTable(ctx, db, c.Table)
	index, _ := ddlDesIndex(ctx, db, c.Table)
	vue, err := empreinteDeVue(ctx, db, c.View)
	if err != nil {
		t.Fatal(err)
	}
	cas := []struct {
		nom, attendu string
		avant        schemaAvant
	}{
		{"identique", "", schemaAvant{ddl: ddl, index: index, vue: vue}},
		{"DDL", "DDL de", schemaAvant{ddl: strings.Replace(ddl, "NOT NULL", "", 1), index: index, vue: vue}},
		{"index", "index de", schemaAvant{ddl: ddl, index: append(append([]string{}, index...),
			"CREATE INDEX idx_disparu ON match_bomb_stats(xuid);"), vue: vue}},
	}
	for _, k := range cas {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		err = verifierApresEchange(ctx, tx, c, k.avant)
		_ = tx.Rollback() // lecture seule
		if k.attendu == "" && err != nil {
			t.Fatalf("%s : refus inattendu : %v", k.nom, err)
		}
		if k.attendu != "" && (err == nil || !strings.Contains(err.Error(), k.attendu)) {
			t.Fatalf("%s : attendu un refus %q, got %v", k.nom, k.attendu, err)
		}
	}
}

// TestDDLDeConstruction : le DDL est repris au caractère près, seul le nom change ; une forme
// inconnue est refusée.
func TestDDLDeConstruction(t *testing.T) {
	got, err := ddlDeConstruction(`CREATE TABLE t(id BIGINT PRIMARY KEY, t_x VARCHAR);`, "t")
	if err != nil || got != `CREATE TABLE t__compact(id BIGINT PRIMARY KEY, t_x VARCHAR);` {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := ddlDeConstruction(`CREATE TABLE main.t(id BIGINT);`, "t"); err == nil {
		t.Fatal("forme inattendue acceptée")
	}
}

// TestRegistre_SignaturesSurLeSQLRenduParDuckDB : le SQL des vues TEL QUE DuckDB 1.5 le rend
// (relevé sur la copie de la base locale du 2026-09-26, parenthèses et alias compris) porte la
// règle du registre — et une règle modifiée ne la porte plus.
func TestRegistre_SignaturesSurLeSQLRenduParDuckDB(t *testing.T) {
	rendus := map[string]string{
		"kill_positions":         `CREATE VIEW kill_positions_latest AS SELECT * FROM kill_positions AS p QUALIFY ((p.decode_pass = first_value(p.decode_pass) OVER (PARTITION BY p.match_id ORDER BY p.written_at DESC, p.id DESC)) AND (row_number() OVER (PARTITION BY p.match_id, p.decode_pass, p.killer_xuid, p.time_ms ORDER BY p.written_at DESC, p.id DESC) = 1));`,
		"match_kill_events":      `CREATE VIEW match_kill_events_latest AS SELECT e.*, CASE  WHEN ((e.killer_damage_pct IS NULL)) THEN (NULL) ELSE ((99 - CAST(e.killer_damage_pct AS SMALLINT)) - CAST(e.assist_damage_pct AS SMALLINT)) END AS damage_pct_residual FROM match_kill_events AS e QUALIFY (e.decode_pass = first_value(e.decode_pass) OVER (PARTITION BY e.match_id ORDER BY e.written_at DESC, e.id DESC));`,
		"match_usage_players":    `CREATE VIEW match_usage_players_latest AS SELECT p.* FROM match_usage_players AS p INNER JOIN match_usage_films_latest AS f ON (((f.match_id = p.match_id) AND (f.summary_pass = p.summary_pass)));`,
		"match_player_positions": `CREATE VIEW match_player_positions_latest AS SELECT p.* FROM match_player_positions AS p QUALIFY (p.positions_pass = first_value(p.positions_pass) OVER (PARTITION BY p.match_id ORDER BY p.written_at DESC, p.id DESC));`,
		"match_bomb_stats":       `CREATE VIEW match_bomb_stats_latest AS SELECT * FROM match_bomb_stats QUALIFY (row_number() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1);`,
	}
	for _, c := range tablesCompactables {
		rendu, ok := rendus[c.Table]
		if !ok {
			continue
		}
		if !c.regleDeLaVueTenue(rendu) {
			t.Errorf("%s : la signature %q n'est pas trouvée dans le SQL rendu", c.Table, c.signature)
		}
		modifie := strings.Replace(rendu, "DESC)", "ASC)", 1)
		if modifie == rendu { // la jointure des joueurs : une autre colonne de passe
			modifie = strings.Replace(rendu, "f.summary_pass = p.summary_pass", "f.summary_rev = p.summary_rev", 1)
		}
		if modifie == rendu || c.regleDeLaVueTenue(modifie) {
			t.Errorf("%s : une règle modifiée passe encore la signature", c.Table)
		}
	}
}

// reMutationDeLigne : les écritures qui toucheraient des lignes EXISTANTES d'une table (le
// déclencheur du bug ART #23645). La compaction n'a droit qu'à CREATE / INSERT dans une table
// neuve / DROP / RENAME.
var reMutationDeLigne = regexp.MustCompile(
	`(?i)\bDELETE\s+FROM\b|\bUPDATE\s+\S+\s+SET\b|\bON\s+CONFLICT\b|\bINSERT\s+OR\b|\bTRUNCATE\b`)

// TestCompaction_AucuneMutationDeLigne — GARDE-RAIL ANTI-ART de la compaction. Les scans de
// internal/sync (no_art_patterns_test.go, append_only_state_guard_test.go) excluent
// `internal/migration` par construction (migrations one-shot, mono-processus) : la compaction, qui
// y vit et touche les tables les plus grosses de la base, est gardée ICI.
func TestCompaction_AucuneMutationDeLigne(t *testing.T) {
	for _, f := range []string{"compaction.go", "compaction_registry.go", "table_swap.go", "compaction_rewrite.go"} {
		src, err := os.ReadFile(f)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatalf("lecture %s: %v", f, err)
		}
		if m := reMutationDeLigne.FindString(stripComments(string(src))); m != "" {
			t.Errorf("%s : mutation de ligne %q — interdit (bug ART #23645) : reconstruire par "+
				"CREATE + INSERT dans une table neuve + swap (table_swap.go)", f, m)
		}
	}
	if reMutationDeLigne.FindString("x := `DELETE FROM match_kill_events WHERE 1`") == "" {
		t.Fatal("témoin : le motif ne détecte plus un DELETE")
	}
}

var (
	reCommentaireBloc  = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reCommentaireLigne = regexp.MustCompile(`(?m)//.*$`)
)

func stripComments(src string) string {
	return reCommentaireLigne.ReplaceAllString(reCommentaireBloc.ReplaceAllString(src, ""), "")
}
