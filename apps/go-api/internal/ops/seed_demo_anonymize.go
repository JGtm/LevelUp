// Package ops — seed_demo_anonymize.go : anonymisation de la démo PENDANT la copie des tables.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// ─── ANONYMISATION À LA COPIE ─────────────────────────────────────────────────────────────
//
// Les identités réelles (xuid, gamertag) sont remappées vers le roster démo PENDANT la copie
// de chaque table : `CREATE TABLE <t> AS SELECT * REPLACE (<remap> AS <col>) FROM src.<t>`.
// La ligne d'origine n'entre donc jamais dans la base démo, et aucune table n'est réécrite
// après coup.
//
// POURQUOI PAS UN UPDATE APRÈS LA COPIE (la forme d'avant le 2026-10-09). Plusieurs tables
// extraites sont APPEND-ONLY (kill_positions, match_kill_events, match_objective_stats,
// match_csrs…) : ADR 0019/0026 n'y admettent que des INSERT. L'UPDATE interpolé
// (`fmt.Sprintf("UPDATE %s …")`) échappait en outre aux garde-rails, qui scannent des
// littéraux ; TestNoMutationOnAppendOnlyTablesInOps interdit désormais cette forme dans
// internal/ops.
//
// UNE SEULE LISTE : les colonnes d'identité sont déclarées par table dans sharedTablesWhere
// et playerTablesWhere (champ `identity`). Une colonne déclarée mais absente de la table
// source (schéma d'un autre titre ou plus ancien) n'est simplement pas remappée.

// demoXUIDMapTable : la table de correspondance posée dans la base démo le temps de la copie.
const demoXUIDMapTable = "_demo_xuid_map"

// demoGamertagMapTable : la correspondance PAR NOM (gamertag réel → gamertag démo), dérivée de
// la précédente et des alias de la source, pour les colonnes qui portent un nom sans xuid.
const demoGamertagMapTable = "_demo_gamertag_map"

// xuidRemap : une identité réelle et son identité démo. toGamertag vide = gamertag conservé.
type xuidRemap struct {
	from, toXUID, toGamertag string
}

// rosterRemaps : les correspondances du roster démo (toutes les identités du corpus).
func rosterRemaps(roster []demoRosterEntry) []xuidRemap {
	out := make([]xuidRemap, 0, len(roster))
	for _, e := range roster {
		out = append(out, xuidRemap{from: e.SourceXUID, toXUID: e.DemoXUID, toGamertag: e.DemoGamertag})
	}
	return out
}

// installXUIDMap crée la table de correspondance dans la base démo (valeurs LIÉES). Une
// identité réelle listée deux fois garde sa première correspondance : la sous-requête de
// remappage doit rendre au plus une ligne.
func installXUIDMap(ctx context.Context, dst *sql.DB, remaps []xuidRemap) error {
	if _, err := dst.ExecContext(ctx, `CREATE OR REPLACE TABLE `+demoXUIDMapTable+
		` (old_xuid VARCHAR, new_xuid VARCHAR, new_gamertag VARCHAR)`); err != nil {
		return fmt.Errorf("table de correspondance démo: %w", err)
	}
	seen := make(map[string]bool, len(remaps))
	for _, m := range remaps {
		if m.from == "" || seen[m.from] {
			continue
		}
		seen[m.from] = true
		var gt any
		if m.toGamertag != "" {
			gt = m.toGamertag
		}
		if _, err := dst.ExecContext(ctx, `INSERT INTO `+demoXUIDMapTable+` VALUES (?, ?, ?)`,
			m.from, m.toXUID, gt); err != nil {
			return fmt.Errorf("correspondance démo %s: %w", m.from, err)
		}
	}
	return nil
}

// installGamertagMap dérive la correspondance par nom : les gamertags que la source associe
// (xuid_aliases, match_participants) à un xuid du roster. Une table source absente (player
// DB) ne contribue rien.
func installGamertagMap(ctx context.Context, dst *sql.DB) error {
	if _, err := dst.ExecContext(ctx, `CREATE OR REPLACE TABLE `+demoGamertagMapTable+
		` (old_gamertag VARCHAR, new_gamertag VARCHAR)`); err != nil {
		return fmt.Errorf("table de correspondance des noms démo: %w", err)
	}
	for _, src := range []string{tableXUIDAliases, tableMatchParticipants} {
		cols, err := sourceColumns(ctx, dst, src)
		if err != nil {
			return err
		}
		if !cols[colXUID] || !cols[colGamertag] {
			continue
		}
		if _, err := dst.ExecContext(ctx, `INSERT INTO `+demoGamertagMapTable+`
			SELECT DISTINCT s.gamertag, m.new_gamertag FROM src.`+src+` s
			JOIN `+demoXUIDMapTable+` m ON m.old_xuid = s.xuid
			WHERE s.gamertag IS NOT NULL AND s.gamertag <> '' AND m.new_gamertag IS NOT NULL`); err != nil {
			return fmt.Errorf("correspondance des noms démo (%s): %w", src, err)
		}
	}
	return nil
}

// dropXUIDMap retire les tables de correspondance : elles portent les identités RÉELLES et ne
// doivent pas rester dans une base publiée.
func dropXUIDMap(ctx context.Context, dst *sql.DB) error {
	for _, t := range []string{demoXUIDMapTable, demoGamertagMapTable} {
		if _, err := dst.ExecContext(ctx, `DROP TABLE IF EXISTS `+t); err != nil {
			return fmt.Errorf("retrait de la table de correspondance démo %s: %w", t, err)
		}
	}
	return nil
}

// sourceColumns rend les colonnes de src.<table> ; vide quand la table source n'existe pas.
func sourceColumns(ctx context.Context, dst *sql.DB, table string) (map[string]bool, error) {
	rows, err := dst.QueryContext(ctx,
		`SELECT column_name FROM duckdb_columns() WHERE database_name = 'src' AND table_name = ?`, table)
	if err != nil {
		return nil, fmt.Errorf("colonnes de src.%s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	cols := map[string]bool{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("colonnes de src.%s (scan): %w", table, err)
		}
		cols[c] = true
	}
	return cols, rows.Err()
}

// anonymizedSelect rend le SELECT de copie d'une table : colonnes techniques exclues pour
// une table append-only reconstruite par migration, colonnes d'identité remappées par les
// tables de correspondance, filtre `where` (déjà instancié) inchangé.
func anonymizedSelect(t extractTable, cols map[string]bool, where string) string {
	var replace []string
	for _, pair := range t.identity {
		xuidCol, gtCol := pair[0], pair[1]
		if !cols[xuidCol] {
			continue
		}
		replace = append(replace, fmt.Sprintf(`COALESCE(%s, t.%s) AS %s`, xuidLookup("new_xuid", "t."+xuidCol), xuidCol, xuidCol))
		if gtCol != "" && cols[gtCol] {
			replace = append(replace, fmt.Sprintf(`COALESCE(%s, t.%s) AS %s`, xuidLookup("new_gamertag", "t."+xuidCol), gtCol, gtCol))
		}
	}
	for _, c := range t.gamertagOnly {
		if cols[c] {
			// Inconnu de la correspondance → NULL : un nom réel hors roster ne traverse pas.
			replace = append(replace, fmt.Sprintf(`(SELECT any_value(g.new_gamertag) FROM %s g WHERE g.old_gamertag = t.%s) AS %s`,
				demoGamertagMapTable, c, c))
		}
	}
	for _, c := range t.xuidLists {
		if cols[c] {
			replace = append(replace, xuidListRemap(c))
		}
	}
	for _, c := range t.jsonIdentity {
		if cols[c] {
			replace = append(replace, jsonIdentityRemap(c))
		}
	}
	proj := extractSelectExpr(t.appendOnly)
	if len(replace) > 0 {
		proj += " REPLACE (" + strings.Join(replace, ", ") + ")"
	}
	return fmt.Sprintf(`SELECT %s FROM src.%s t WHERE %s`, proj, t.name, where)
}

// xuidLookup : la sous-requête qui rend le champ `field` de la correspondance pour le xuid
// réel `expr` (NULL quand il n'y figure pas).
func xuidLookup(field, expr string) string {
	return fmt.Sprintf(`(SELECT any_value(m.%s) FROM %s m WHERE m.old_xuid = %s)`, field, demoXUIDMapTable, expr)
}

// xuidListRemap : une liste « xuid,xuid,… » réécrite en xuid démo, TRIÉE (une signature reste
// comparable d'un match à l'autre) ; un xuid hors roster est retiré. Vide et NULL sont
// conservés tels quels (ils ne disent pas la même chose).
func xuidListRemap(c string) string {
	return fmt.Sprintf(`CASE WHEN t.%[1]s IS NULL OR t.%[1]s = '' THEN t.%[1]s ELSE COALESCE((
		SELECT string_agg(m.new_xuid, ',' ORDER BY m.new_xuid) FROM (SELECT trim(unnest(string_split(t.%[1]s, ','))) AS x) u
		JOIN %[2]s m ON m.old_xuid = u.x), '') END AS %[1]s`, c, demoXUIDMapTable)
}

// jsonIdentityRemap : un document JSON {"xuid": …, "gamertag": …} réécrit avec l'identité démo
// de son xuid. Un document illisible, ou dont le xuid n'est pas au roster, garde sa forme —
// le contrôle des valeurs de fin de seed (verifyDemoAnonymization) le signalerait.
func jsonIdentityRemap(c string) string {
	return fmt.Sprintf(`CASE WHEN json_valid(t.%[1]s) THEN COALESCE((
		SELECT any_value(CAST(json_merge_patch(CAST(t.%[1]s AS JSON),
			json_object('xuid', TRY_CAST(m.new_xuid AS BIGINT), 'gamertag', m.new_gamertag)) AS VARCHAR))
		FROM %[2]s m WHERE m.old_xuid = json_extract_string(t.%[1]s, '$.xuid')), t.%[1]s)
		ELSE t.%[1]s END AS %[1]s`, c, demoXUIDMapTable)
}

// copyAnonymizedTables copie chaque table de `tables` de src vers la base démo, anonymisée,
// et rend le nombre de lignes par table. Une table source absente est journalisée et
// comptée à 0. Une copie en échec interrompt tout, sauf si `tolerant` (player DB : une base
// legacy peut porter une table sans les colonnes techniques attendues) — elle est alors
// journalisée et comptée à 0.
func copyAnonymizedTables(ctx context.Context, dst *sql.DB, tables []extractTable, idsLit string,
	remaps []xuidRemap, tolerant bool) (counts map[string]int, err error) {
	// Les tables de correspondance portent les identités RÉELLES : retirées sur TOUS les
	// chemins de sortie, copie en échec comprise.
	defer func() {
		if derr := dropXUIDMap(ctx, dst); derr != nil {
			slog.ErrorContext(ctx, "seed-demo: tables de correspondance non retirées", "err", derr)
			if err == nil {
				err = derr
			}
		}
	}()
	if err := installXUIDMap(ctx, dst, remaps); err != nil {
		return nil, err
	}
	if err := installGamertagMap(ctx, dst); err != nil {
		return nil, err
	}
	counts = make(map[string]int, len(tables))
	for _, t := range tables {
		cols, err := sourceColumns(ctx, dst, t.name)
		if err != nil {
			return counts, err
		}
		if len(cols) == 0 {
			slog.WarnContext(ctx, "seed-demo: table source absente, ignorée", "table", t.name)
			counts[t.name] = 0
			continue
		}
		where := t.where
		if strings.Contains(where, "%s") {
			where = fmt.Sprintf(where, idsLit)
		}
		if _, err := dst.ExecContext(ctx, `CREATE TABLE `+t.name+` AS `+anonymizedSelect(t, cols, where)); err != nil {
			if !tolerant {
				return counts, fmt.Errorf("copie anonymisée de %s: %w", t.name, err)
			}
			slog.WarnContext(ctx, "seed-demo: copie de table en échec, ignorée", "table", t.name, "err", err)
			counts[t.name] = 0
			continue
		}
		var n int
		if err := dst.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+t.name).Scan(&n); err != nil {
			return counts, fmt.Errorf("count %s: %w", t.name, err)
		}
		counts[t.name] = n
	}
	return counts, nil
}
