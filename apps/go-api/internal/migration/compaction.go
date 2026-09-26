package migration

// compaction.go — LA COMPACTION DES PASSES SUPERSÉDÉES (plan perf du 2026-09-26, étape C).
//
// Chaque redécodage d'un film AJOUTE une passe complète aux tables du film (écriture INSERT-only,
// ADR 0019 / 0026) ; la vue `_latest` ne sert que la dernière. Sur la base locale du 2026-09-26,
// environ 90 % des lignes de ces tables étaient des passes mortes, et toute lecture qui évalue une
// de ces vues les parcourt. La compaction les retire — SANS DELETE (bug DuckDB ART #23645) : la
// table est RECONSTRUITE à côté avec son DDL exact, remplie des seules lignes que la vue retient,
// puis échangée dans une transaction (cœur commun table_swap.go).
//
// Ce qui est garanti, et vérifié AVANT le COMMIT de chaque table (sinon rollback, base intacte) :
//   - la table neuve porte exactement les lignes de la règle du registre (garde de cardinalité) ;
//   - le DDL de la table (colonnes, défauts, clé primaire, NOT NULL) et ses index sont identiques
//     au caractère près (`duckdb_tables().sql`, `duckdb_indexes().sql`) ;
//   - la vue `_latest` rend le même résultat (empreinte : nombre de lignes + somme des hash de
//     ligne) ;
//   - les `id` gardés sont conservés et la séquence n'est pas touchée : le prochain `id` continue
//     après le plus grand jamais tiré, aucun n'est réutilisé.
//
// C'est une opération d'ENTRETIEN (DC.6) : jamais au boot ni après un sync. Elle exige l'écrivain
// exclusif (serveur arrêté) et se joue après une campagne de redécodage (`levelup compact-passes`).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// compactSuffix : suffixe de la table de construction d'une compaction.
const compactSuffix = "__compact"

// Statuts d'une table dans le rapport de compaction.
const (
	CompactionAbsente      = "absente"       // table ou vue absente de cette base
	CompactionDejaCompacte = "deja-compacte" // aucune ligne hors de la passe retenue
	CompactionACompacter   = "a-compacter"   // --dry-run : ce que la compaction retirerait
	CompactionCompactee    = "compactee"
)

// CompactionTable : le rapport d'une table.
type CompactionTable struct {
	Table    string
	Status   string
	Raw      int64 // lignes brutes avant
	Latest   int64 // lignes servies par la vue `_latest`
	Kept     int64 // lignes de la passe retenue (>= Latest : la passe entière)
	After    int64 // lignes brutes après (== Kept une fois compactée)
	Duration time.Duration
}

// lecteurSQL : ce que *sql.DB et *sql.Tx ont en commun pour lire.
type lecteurSQL interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// CompactSupersededPasses compacte chaque table du registre (ou, en `dryRun`, mesure seulement
// ce qu'elle retirerait, sans aucune écriture). Idempotente : une seconde passe rend
// `deja-compacte` partout. Chaque table est échangée dans SA transaction : une erreur arrête la
// commande, les tables déjà compactées le restent (chacune est complète et vérifiée).
func CompactSupersededPasses(ctx context.Context, db *sql.DB, dryRun bool) ([]CompactionTable, error) {
	out := make([]CompactionTable, 0, len(tablesCompactables))
	compactees := 0
	for _, c := range tablesCompactables {
		r, err := compacterTable(ctx, db, c, dryRun)
		if err != nil {
			return out, err
		}
		if r.Status == CompactionCompactee {
			compactees++
		}
		out = append(out, r)
	}
	if compactees > 0 {
		if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
			return out, fmt.Errorf("compaction: checkpoint final: %w", err)
		}
	}
	return out, nil
}

// compacterTable traite une table : réparation d'un orphelin, contrôle de la règle de sa vue,
// mesures, puis échange (sauf dry-run ou table déjà compacte).
func compacterTable(ctx context.Context, db *sql.DB, c compactable, dryRun bool) (CompactionTable, error) {
	debut := time.Now()
	r := CompactionTable{Table: c.Table}
	if !dryRun {
		if err := recoverOrphanTable(ctx, db, c.Table, compactSuffix); err != nil {
			return r, err
		}
	}
	sqlVue, present, err := lireTableEtVue(ctx, db, c)
	if err != nil {
		return r, err
	}
	if !present {
		r.Status = CompactionAbsente
		slog.InfoContext(ctx, "compaction: table ou vue absente, sautée", "table", c.Table)
		return r, nil
	}
	if !c.regleDeLaVueTenue(sqlVue) {
		return r, fmt.Errorf("compaction %s: la vue %s ne porte plus la règle du registre "+
			"(compaction_registry.go) — table refusée, rien n'est écrit", c.Table, c.View)
	}
	if err := mesurer(ctx, db, c, &r); err != nil {
		return r, err
	}
	avant, err := empreinteDeVue(ctx, db, c.View)
	if err != nil {
		return r, err
	}
	r.Latest = avant.Lignes
	switch {
	case r.Kept == r.Raw:
		r.Status, r.After = CompactionDejaCompacte, r.Raw
	case dryRun:
		r.Status, r.After = CompactionACompacter, r.Kept
	default:
		if err := echangerTable(ctx, db, c, avant); err != nil {
			return r, err
		}
		if err := compter(ctx, db, "SELECT COUNT(*) FROM "+c.Table, &r.After); err != nil {
			return r, fmt.Errorf("compaction %s: compte après: %w", c.Table, err)
		}
		r.Status = CompactionCompactee
	}
	r.Duration = time.Since(debut)
	slog.InfoContext(ctx, "compaction: table traitée", "table", c.Table, "status", r.Status,
		"brutes_avant", r.Raw, "latest", r.Latest, "gardees", r.Kept, "brutes_apres", r.After,
		"duration", r.Duration.Round(time.Millisecond).String())
	return r, nil
}

// lireTableEtVue rend le SQL de la vue, et si la table ET la vue existent.
func lireTableEtVue(ctx context.Context, db *sql.DB, c compactable) (string, bool, error) {
	hasTable, err := tableExists(db, c.Table)
	if err != nil || !hasTable {
		return "", false, wrapCompaction(c.Table, "table", err)
	}
	var sqlVue string
	err = db.QueryRowContext(ctx, `SELECT sql FROM duckdb_views()
		WHERE schema_name = 'main' AND view_name = ?`, c.View).Scan(&sqlVue)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, wrapCompaction(c.Table, "vue", err)
	}
	return sqlVue, true, nil
}

func wrapCompaction(table, quoi string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("compaction %s: lecture du catalogue (%s): %w", table, quoi, err)
}

// mesurer lit les lignes brutes et les lignes de la passe retenue.
func mesurer(ctx context.Context, db *sql.DB, c compactable, r *CompactionTable) error {
	if err := compter(ctx, db, "SELECT COUNT(*) FROM "+c.Table, &r.Raw); err != nil {
		return fmt.Errorf("compaction %s: compte brut: %w", c.Table, err)
	}
	if err := compter(ctx, db, requeteGardees(c), &r.Kept); err != nil {
		return fmt.Errorf("compaction %s: compte des lignes gardées: %w", c.Table, err)
	}
	return nil
}

func requeteGardees(c compactable) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM (SELECT * FROM %s %s)", c.Table, c.keep)
}

func compter(ctx context.Context, q lecteurSQL, requete string, n *int64) error {
	return q.QueryRowContext(ctx, requete).Scan(n)
}

// EmpreinteVue : ce qu'une vue rend, résumé — nombre de lignes et somme des hash de ligne (toutes
// colonnes). Indépendante de l'ordre, sensible à toute ligne ajoutée, perdue ou modifiée.
type EmpreinteVue struct {
	Lignes int64
	Somme  string
}

func empreinteDeVue(ctx context.Context, q lecteurSQL, vue string) (EmpreinteVue, error) {
	var e EmpreinteVue
	err := q.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*),
		CAST(COALESCE(SUM(CAST(hash(v) AS HUGEINT)), 0) AS VARCHAR) FROM %s v`, vue)).
		Scan(&e.Lignes, &e.Somme)
	if err != nil {
		return e, fmt.Errorf("compaction: empreinte de %s: %w", vue, err)
	}
	return e, nil
}

// echangerTable reconstruit la table à côté (DDL exact), la remplit des lignes gardées et
// l'échange ; le DDL, les index et l'empreinte de la vue sont vérifiés avant le COMMIT.
func echangerTable(ctx context.Context, db *sql.DB, c compactable, avant EmpreinteVue) error {
	ddl, err := ddlDeTable(ctx, db, c.Table)
	if err != nil {
		return err
	}
	creer, err := ddlDeConstruction(ddl, c.Table)
	if err != nil {
		return err
	}
	index, err := ddlDesIndex(ctx, db, c.Table)
	if err != nil {
		return err
	}
	_, err = swapTableTx(ctx, db, tableSwap{
		Table:    c.Table,
		Suffix:   compactSuffix,
		Expected: requeteGardees(c),
		Build: []string{creer, fmt.Sprintf(`INSERT INTO %s%s SELECT * FROM %s %s`,
			c.Table, compactSuffix, c.Table, c.keep)},
		PostRename: index,
		Verify: func(ctx context.Context, tx *sql.Tx) error {
			return verifierApresEchange(ctx, tx, c, schemaAvant{ddl: ddl, index: index, vue: avant})
		},
	})
	if err != nil {
		return fmt.Errorf("compaction: %w", err)
	}
	return nil
}

// schemaAvant : ce que l'échange doit laisser identique.
type schemaAvant struct {
	ddl   string
	index []string
	vue   EmpreinteVue
}

func verifierApresEchange(ctx context.Context, tx *sql.Tx, c compactable, avant schemaAvant) error {
	ddl, err := ddlDeTable(ctx, tx, c.Table)
	if err != nil {
		return err
	}
	if ddl != avant.ddl {
		return fmt.Errorf("DDL de %s changé :\navant %s\naprès %s", c.Table, avant.ddl, ddl)
	}
	index, err := ddlDesIndex(ctx, tx, c.Table)
	if err != nil {
		return err
	}
	if strings.Join(index, "\n") != strings.Join(avant.index, "\n") {
		return fmt.Errorf("index de %s changés : avant %v, après %v", c.Table, avant.index, index)
	}
	apres, err := empreinteDeVue(ctx, tx, c.View)
	if err != nil {
		return err
	}
	if apres != avant.vue {
		return fmt.Errorf("la vue %s ne rend plus le même résultat : avant %+v, après %+v",
			c.View, avant.vue, apres)
	}
	return nil
}

// ddlDeTable rend le `CREATE TABLE` que DuckDB tient pour la table (colonnes, défauts,
// contraintes) — la source du DDL de la table neuve et de la vérification.
func ddlDeTable(ctx context.Context, q lecteurSQL, table string) (string, error) {
	var ddl string
	if err := q.QueryRowContext(ctx, `SELECT sql FROM duckdb_tables()
		WHERE schema_name = 'main' AND table_name = ?`, table).Scan(&ddl); err != nil {
		return "", fmt.Errorf("compaction %s: DDL de la table: %w", table, err)
	}
	return ddl, nil
}

// ddlDeConstruction réécrit le `CREATE TABLE <table>(` en `CREATE TABLE <table>__compact(` : le
// reste du DDL est repris au caractère près. Refus si le préfixe n'est pas celui attendu (une
// forme de DDL inconnue ne se devine pas).
func ddlDeConstruction(ddl, table string) (string, error) {
	prefixe := "CREATE TABLE " + table + "("
	if !strings.HasPrefix(ddl, prefixe) {
		return "", fmt.Errorf("compaction %s: DDL de forme inattendue (préfixe %q absent) : %s",
			table, prefixe, ddl)
	}
	return "CREATE TABLE " + table + compactSuffix + "(" + ddl[len(prefixe):], nil
}

// ddlDesIndex rend les `CREATE INDEX` de la table, triés par nom d'index.
func ddlDesIndex(ctx context.Context, q lecteurSQL, table string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT sql FROM duckdb_indexes()
		WHERE schema_name = 'main' AND table_name = ? ORDER BY index_name`, table)
	if err != nil {
		return nil, fmt.Errorf("compaction %s: index: %w", table, err)
	}
	defer rows.Close() //nolint:errcheck // lecture seule, l'erreur d'itération est lue par rows.Err
	var out []string
	for rows.Next() {
		var s sql.NullString
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("compaction %s: index: %w", table, err)
		}
		if !s.Valid || s.String == "" {
			return nil, fmt.Errorf("compaction %s: un index sans DDL, recréation impossible", table)
		}
		out = append(out, s.String)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("compaction %s: index: %w", table, err)
	}
	return out, nil
}
