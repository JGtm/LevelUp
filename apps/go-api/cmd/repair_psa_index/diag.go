package main

// diag.go — diagnostic et réparation des index ART de personal_score_awards.
//
// Principe du diagnostic : pour un axe donné (une colonne ou un tuple indexé),
// on compare deux comptages qui DOIVENT être égaux :
//   - le comptage de référence, obtenu par SCAN FORCÉ (la clé de regroupement est
//     une expression — `col || ''` / `CAST(col AS VARCHAR)` — qu'aucun index ART
//     ne peut servir) ;
//   - le comptage par LOOKUP INDEXÉ (`WHERE col = ?`), qui passe par l'index.
// Tout écart = index désynchronisé de la table (bug DuckDB ART #23046).
//
// Les clés NULL sont exclues du diagnostic : elles ne sont pas interrogeables par
// égalité (`col = NULL` ne matche jamais) et produiraient un faux écart. Leur
// nombre est reporté pour transparence.

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// psaIndexDDL — DDL des index, RECOPIÉE À L'IDENTIQUE de leur autorité :
// internal/migration/steps_player_append_only_personal_score_awards.go (PostSwap)
// et internal/migration/steps_player_schema_authority.go. Une réparation doit
// recréer exactement l'index que les migrations posent, sinon les DB divergent.
var psaIndexDDL = map[string]string{
	"idx_psa_match":    `CREATE INDEX IF NOT EXISTS idx_psa_match     ON personal_score_awards(match_id)`,
	"idx_psa_category": `CREATE INDEX IF NOT EXISTS idx_psa_category  ON personal_score_awards(award_category)`,
	"idx_psa_gen":      `CREATE INDEX IF NOT EXISTS idx_psa_gen       ON personal_score_awards(match_id, xuid, generation_id)`,
}

// axis décrit un axe de vérification.
type axis struct {
	name string
	// keyExprs : expressions de regroupement, SCAN FORCÉ (jamais servi par un ART).
	keyExprs []string
	// lookupWhere : prédicat de lookup, colonnes NUES (l'index peut servir).
	// Le paramètre, lui, peut être casté.
	lookupWhere string
	// indexes : index susceptibles de servir ce lookup → à reconstruire si l'axe diverge.
	indexes []string
}

// psaAxes — les trois axes indexés de personal_score_awards.
var psaAxes = []axis{
	{
		name:        "match_id (idx_psa_match)",
		keyExprs:    []string{"match_id || ''"},
		lookupWhere: "match_id = ?",
		// En théorie le lookup sur match_id pourrait être servi par idx_psa_match
		// OU par le préfixe d'idx_psa_gen. Mesure sur les 4 DB réelles (2026-08-27) :
		// cet axe accuse un déficit alors que l'axe du triplet rend le compte EXACT
		// sur les mêmes bases — le planner sert donc bien ce prédicat par
		// idx_psa_match, et chaque index a son axe dédié. On ne reconstruit que
		// l'index effectivement sondé (aucune écriture superflue sur données réelles).
		indexes: []string{"idx_psa_match"},
	},
	{
		name:        "award_category (idx_psa_category)",
		keyExprs:    []string{"award_category || ''"},
		lookupWhere: "award_category = ?",
		indexes:     []string{"idx_psa_category"},
	},
	{
		name:        "match_id+xuid+generation_id (idx_psa_gen)",
		keyExprs:    []string{"match_id || ''", "xuid || ''", "CAST(generation_id AS VARCHAR)"},
		lookupWhere: "match_id = ? AND xuid = ? AND generation_id = CAST(? AS BIGINT)",
		indexes:     []string{"idx_psa_gen"},
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
	scannedRows int
	indexedRows int
	divergences []divergence
}

func (r axisReport) ok() bool { return len(r.divergences) == 0 }

// diagnoseAxis compare, clé par clé, le comptage par scan et le comptage indexé.
func diagnoseAxis(ctx context.Context, db *sql.DB, a axis) (axisReport, error) {
	rep := axisReport{axis: a.name}

	groupSQL := fmt.Sprintf(
		`SELECT %s, COUNT(*) FROM personal_score_awards GROUP BY %s ORDER BY %s`,
		strings.Join(a.keyExprs, ", "),
		strings.Join(a.keyExprs, ", "),
		strings.Join(a.keyExprs, ", "))
	rows, err := db.QueryContext(ctx, groupSQL)
	if err != nil {
		return rep, fmt.Errorf("scan de référence (%s): %w", a.name, err)
	}

	type keyCount struct {
		key   []string
		count int
	}
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
			rows.Close()
			return rep, fmt.Errorf("scan de référence (%s): %w", a.name, err)
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
		refs = append(refs, keyCount{key: key, count: n})
		rep.scannedRows += n
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return rep, fmt.Errorf("itération du scan (%s): %w", a.name, err)
	}
	rows.Close()
	rep.keys = len(refs)

	stmt, err := db.PrepareContext(ctx,
		`SELECT COUNT(*) FROM personal_score_awards WHERE `+a.lookupWhere)
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
	reports := make([]axisReport, 0, len(psaAxes))
	for _, a := range psaAxes {
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
		if rep.ok() {
			continue
		}
		for _, name := range psaAxes[i].indexes {
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

// repairIndexes reconstruit les index nommés : DROP puis CREATE à l'identique.
// Aucune ligne de données n'est touchée — DDL d'index uniquement.
func repairIndexes(ctx context.Context, db *sql.DB, names []string) error {
	for _, name := range names {
		ddl, ok := psaIndexDDL[name]
		if !ok {
			return fmt.Errorf("index inconnu %q (pas de DDL de référence)", name)
		}
		if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS `+name); err != nil {
			return fmt.Errorf("DROP INDEX %s: %w", name, err)
		}
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("CREATE INDEX %s: %w", name, err)
		}
	}
	// CHECKPOINT : rend la DDL durable dans le fichier plutôt que dans le seul WAL.
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("CHECKPOINT après réparation: %w", err)
	}
	return nil
}

// existingIndexes liste les index présents sur personal_score_awards.
func existingIndexes(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT index_name FROM duckdb_indexes() WHERE table_name = 'personal_score_awards' ORDER BY index_name`)
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
