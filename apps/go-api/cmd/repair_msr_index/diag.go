//go:build cgo

package main

// diag.go — diagnostic et réparation des index ART de `match_skill_rank`.
//
// Principe du diagnostic, repris de cmd/repair_psa_index/diag.go (même famille de
// défaut, même doctrine) : pour un axe indexé, deux comptages qui DOIVENT être
// égaux sont comparés —
//   - référence par SCAN FORCÉ : la clé de regroupement est une EXPRESSION
//     (`col || ''`, `CAST(col AS VARCHAR)`) qu'aucun index ART ne peut servir ;
//   - mesure par LOOKUP INDEXÉ (`WHERE col = ?`), que le planner sert par l'index.
//
// Tout écart = index désynchronisé de la table (bug DuckDB #23645). Les clés NULL
// sont exclues : `col = NULL` ne matche jamais et produirait un faux écart ; leur
// nombre est reporté.
//
// DDL DE RÉPARATION : jamais recopiée. Elle est CAPTURÉE dans la base
// (`duckdb_indexes().sql`) avant le DROP, puis rejouée — c'est par construction
// l'index que les migrations ont posé (autorité :
// games/halo_infinite/migrations/steps_player_match_skill_rank.go). Une DDL
// recopiée dans un outil dérive en silence dès que la migration évolue.

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// maxSampledKeys borne le nombre de clés sondées par axe. L'axe du triplet porte
// ~1 clé par ligne (des milliers) ; au-delà de cette borne le diagnostic reste
// représentatif et le rapport dit explicitement qu'il a été tronqué.
const maxSampledKeys = 5000

// axis décrit un axe de vérification.
type axis struct {
	name string
	// keyExprs : expressions de regroupement, SCAN FORCÉ (jamais servi par un ART).
	keyExprs []string
	// lookupWhere : prédicat de lookup, colonnes NUES (l'index peut servir).
	lookupWhere string
	// indexes : index susceptibles de servir ce lookup → à reconstruire si écart.
	indexes []string
}

// msrAxes — les trois axes indexés de match_skill_rank (un par index posé par
// steps_player_match_skill_rank.go).
var msrAxes = []axis{
	{
		name:        "playlist_group (idx_msr_playlist)",
		keyExprs:    []string{"playlist_group || ''"},
		lookupWhere: "playlist_group = ?",
		indexes:     []string{"idx_msr_playlist"},
	},
	{
		name:        "rating_type (idx_msr_rating_type)",
		keyExprs:    []string{"rating_type || ''"},
		lookupWhere: "rating_type = ?",
		indexes:     []string{"idx_msr_rating_type"},
	},
	{
		// L'index porte (match_id, rating_type, written_at). Le lookup sonde le
		// triplet entier : c'est le seul prédicat que cet index sert pleinement.
		name:        "match_id+rating_type+written_at (idx_msr_match_lookup)",
		keyExprs:    []string{"match_id || ''", "rating_type || ''", "CAST(written_at AS VARCHAR)"},
		lookupWhere: "match_id = ? AND rating_type = ? AND written_at = CAST(? AS TIMESTAMP)",
		indexes:     []string{"idx_msr_match_lookup"},
	},
}

// divergence — une clé dont le lookup indexé ne rend pas le compte du scan.
type divergence struct {
	key     []string
	scanned int
	indexed int
}

// axisReport — résultat du diagnostic d'un axe.
type axisReport struct {
	axis        string
	keys        int
	nullKeys    int
	truncated   bool // le nombre de clés distinctes dépassait maxSampledKeys
	scannedRows int
	indexedRows int
	divergences []divergence
}

func (r axisReport) ok() bool { return len(r.divergences) == 0 }

// keyCount — une clé distincte et son compte de référence (par scan).
type keyCount struct {
	key   []string
	count int
}

// scanReference établit, par SCAN FORCÉ, le compte de référence par clé.
func scanReference(ctx context.Context, db *sql.DB, a axis, rep *axisReport) ([]keyCount, error) {
	exprs := strings.Join(a.keyExprs, ", ")
	groupSQL := fmt.Sprintf(
		`SELECT %s, COUNT(*) FROM match_skill_rank GROUP BY %s ORDER BY %s`, exprs, exprs, exprs)
	rows, err := db.QueryContext(ctx, groupSQL)
	if err != nil {
		return nil, fmt.Errorf("scan de référence (%s): %w", a.name, err)
	}
	defer rows.Close()

	var refs []keyCount
	for rows.Next() {
		vals := make([]sql.NullString, len(a.keyExprs))
		dest := make([]any, 0, len(a.keyExprs)+1)
		for i := range vals {
			dest = append(dest, &vals[i])
		}
		var n int
		dest = append(dest, &n)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan de référence (%s): %w", a.name, err)
		}
		key := make([]string, 0, len(vals))
		nullKey := false
		for _, v := range vals {
			if !v.Valid {
				nullKey = true
				break
			}
			key = append(key, v.String)
		}
		if nullKey {
			rep.nullKeys++
			continue
		}
		if len(refs) >= maxSampledKeys {
			rep.truncated = true
			continue
		}
		refs = append(refs, keyCount{key: key, count: n})
		rep.scannedRows += n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("itération du scan (%s): %w", a.name, err)
	}
	return refs, nil
}

// diagnoseAxis compare, clé par clé, le comptage par scan et le comptage indexé.
func diagnoseAxis(ctx context.Context, db *sql.DB, a axis) (axisReport, error) {
	rep := axisReport{axis: a.name}

	refs, err := scanReference(ctx, db, a, &rep)
	if err != nil {
		return rep, err
	}
	rep.keys = len(refs)

	stmt, err := db.PrepareContext(ctx, `SELECT COUNT(*) FROM match_skill_rank WHERE `+a.lookupWhere)
	if err != nil {
		return rep, fmt.Errorf("préparation du lookup (%s): %w", a.name, err)
	}
	defer stmt.Close()

	for _, ref := range refs {
		args := make([]any, 0, len(ref.key))
		for _, k := range ref.key {
			args = append(args, k)
		}
		var indexed int
		if err := stmt.QueryRowContext(ctx, args...).Scan(&indexed); err != nil {
			return rep, fmt.Errorf("lookup indexé (%s, clé %v): %w", a.name, ref.key, err)
		}
		rep.indexedRows += indexed
		if indexed != ref.count {
			rep.divergences = append(rep.divergences, divergence{
				key: ref.key, scanned: ref.count, indexed: indexed,
			})
		}
	}
	return rep, nil
}

// diagnoseAll passe les trois axes et retourne les rapports dans l'ordre.
func diagnoseAll(ctx context.Context, db *sql.DB) ([]axisReport, error) {
	reports := make([]axisReport, 0, len(msrAxes))
	for _, a := range msrAxes {
		rep, err := diagnoseAxis(ctx, db, a)
		if err != nil {
			return reports, err
		}
		reports = append(reports, rep)
	}
	return reports, nil
}

// indexesToRebuild — union (ordonnée) des index des axes en écart.
func indexesToRebuild(reports []axisReport) []string {
	seen := map[string]bool{}
	for i, rep := range reports {
		if rep.ok() || i >= len(msrAxes) {
			continue
		}
		for _, name := range msrAxes[i].indexes {
			seen[name] = true
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// captureIndexDDL relève, dans la base, la DDL des index de match_skill_rank.
// C'est l'autorité : l'instruction rejouée est celle que les migrations ont posée.
func captureIndexDDL(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT index_name, sql FROM duckdb_indexes()
		 WHERE table_name = 'match_skill_rank' AND sql IS NOT NULL ORDER BY index_name`)
	if err != nil {
		return nil, fmt.Errorf("duckdb_indexes(): %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, ddl string
		if err := rows.Scan(&name, &ddl); err != nil {
			return nil, fmt.Errorf("duckdb_indexes() scan: %w", err)
		}
		out[name] = ddl
	}
	return out, rows.Err()
}

// repairIndexes reconstruit les index nommés : DROP puis CREATE avec la DDL
// capturée. Aucune ligne de données n'est touchée — DDL d'index uniquement,
// jamais de DELETE ni d'UPDATE (ce serait le vecteur ART lui-même).
func repairIndexes(ctx context.Context, db *sql.DB, names []string) error {
	ddls, err := captureIndexDDL(ctx, db)
	if err != nil {
		return err
	}
	for _, name := range names {
		ddl, ok := ddls[name]
		if !ok {
			return fmt.Errorf("DDL introuvable pour l'index %q dans duckdb_indexes() — "+
				"index absent de cette base ? autorité : "+
				"games/halo_infinite/migrations/steps_player_match_skill_rank.go", name)
		}
		if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS `+name); err != nil {
			return fmt.Errorf("DROP INDEX %s: %w", name, err)
		}
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("CREATE INDEX %s (%s): %w", name, ddl, err)
		}
	}
	// CHECKPOINT : rend la DDL durable dans le fichier plutôt que dans le seul WAL.
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("CHECKPOINT après réparation: %w", err)
	}
	return nil
}

// existingIndexes liste les index présents sur match_skill_rank.
func existingIndexes(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT index_name FROM duckdb_indexes() WHERE table_name = 'match_skill_rank' ORDER BY index_name`)
	if err != nil {
		return nil, fmt.Errorf("duckdb_indexes(): %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("duckdb_indexes() scan: %w", err)
		}
		out = append(out, name)
	}
	return out, rows.Err()
}
