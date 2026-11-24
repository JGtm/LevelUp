package migration

// highlight_events_dedupe.go — RETIRER DE `highlight_events` LES LIGNES ÉCRITES DEUX FOIS.
//
// Deux familles de doublons, mesurées sur le parc du 2026-11-24 (172 matchs, 15 186 éliminations en
// trop, plus les morts, médailles et temps forts « mode » des mêmes matchs) :
//
//  1. FORME SANS CODE (145 matchs, synchronisés les 25 et 26 mai 2026) : une première passe a écrit
//     chaque fait sans `type_hint` ni `raw_json` (l'ancien état du flux principal), une seconde l'a
//     écrit avec son code. Une ligne SANS CODE qui a un jumeau AVEC code (même match, type, instant,
//     joueur) est le reste de la première passe : elle part. Une ligne sans code sans jumeau reste :
//     c'est la seule trace du fait.
//  2. RECOPIE EN BLOC (27 matchs, février à mai 2026) : un rejeu a réinséré tout le match. Le
//     match porte alors CHAQUE ligne en nombre pair, et au moins une élimination ; il garde la moitié
//     de chaque lot de lignes identiques (la première moitié par `id`). Un match qui n'a que quelques
//     répétitions légitimes (deux médailles identiques au même instant) porte aussi des lignes en
//     exemplaire unique : il n'est pas touché.
//
// Ce n'est PAS un dédoublonnage par clé : deux faits identiques d'un même instant existent pour de
// bon (224 groupes de médailles identiques sur 133 matchs, dont 214 portent un code sans `raw_json`, et
// 28 groupes de temps forts « mode » sur la base du poste le 2026-11-24 ; sur Halo 5, des milliers). La
// règle ne retire que ce qu'une des deux passes explique.
//
// SANS DELETE : `highlight_events` porte des index ART (PK, idx_highlight_match,
// idx_events_match_type) ; un DELETE y est le déclencheur du bug DuckDB ART #23645 (ADR 0019). La
// table est reconstruite à côté, sans ces lignes, puis échangée (swapKeepingRows, ADR 0026) ; les
// `id` gardés et la séquence ne bougent pas. Idempotent : sans doublon, rien n'est écrit.
// Borné à Halo Infinite par le step qui l'appelle (`OnlyTitles`) : la règle n'a été mesurée que là.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// highlightEventsDoomedSQL rend `(id, why)` de chaque ligne à retirer : `bare` (forme sans code
// doublée par sa jumelle avec code) ou `recopy` (seconde moitié d'un match recopié en bloc).
const highlightEventsDoomedSQL = `
	WITH h AS (
		SELECT id, match_id, event_type, COALESCE(time_ms, -1) AS t, COALESCE(xuid, '') AS x,
		       type_hint, raw_json, (type_hint IS NULL AND raw_json IS NULL) AS bare
		FROM highlight_events
	), twin AS (
		SELECT DISTINCT match_id, event_type, t, x FROM h WHERE NOT bare
	), bare_with_twin AS (
		SELECT b.id FROM h b
		JOIN twin w ON w.match_id = b.match_id AND w.event_type = b.event_type AND w.t = b.t AND w.x = b.x
		WHERE b.bare
	), r AS (
		SELECT * FROM h WHERE id NOT IN (SELECT id FROM bare_with_twin)
	), k AS (
		SELECT match_id, event_type, t, x, type_hint, raw_json, count(*) AS c FROM r GROUP BY ALL
	), recopied AS (
		SELECT match_id FROM k GROUP BY match_id
		HAVING bool_and(c % 2 = 0) AND bool_or(event_type = 'kill')
	), ranked AS (
		SELECT id,
		       row_number() OVER (PARTITION BY match_id, event_type, t, x, type_hint, raw_json ORDER BY id) AS rn,
		       count(*) OVER (PARTITION BY match_id, event_type, t, x, type_hint, raw_json) AS c
		FROM r WHERE match_id IN (SELECT match_id FROM recopied)
	)
	SELECT id, 'bare' AS why FROM bare_with_twin
	UNION ALL
	SELECT id, 'recopy' AS why FROM ranked WHERE rn > c / 2`

// HighlightEventsDedupeReport : ce que la détection a trouvé (et, hors dry-run, retiré).
type HighlightEventsDedupeReport struct {
	Before   int64
	Bare     int64 // lignes sans code doublées par leur jumelle avec code
	Recopied int64 // lignes de la seconde moitié d'un match recopié en bloc
	Matches  int64
	After    int64
}

// Removed : nombre total de lignes à retirer.
func (r HighlightEventsDedupeReport) Removed() int64 { return r.Bare + r.Recopied }

// DetectHighlightEventsDuplicates mesure les lignes à retirer sans rien écrire.
func DetectHighlightEventsDuplicates(ctx context.Context, q lecteurSQL) (HighlightEventsDedupeReport, error) {
	var rep HighlightEventsDedupeReport
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM highlight_events`).Scan(&rep.Before); err != nil {
		return rep, fmt.Errorf("highlight_events doublons: compte: %w", err)
	}
	err := q.QueryRowContext(ctx, `
		WITH doomed AS (`+highlightEventsDoomedSQL+`)
		SELECT COUNT(*) FILTER (WHERE d.why = 'bare'), COUNT(*) FILTER (WHERE d.why = 'recopy'),
		       COUNT(DISTINCT h.match_id)
		FROM doomed d JOIN highlight_events h USING (id)`).Scan(&rep.Bare, &rep.Recopied, &rep.Matches)
	if err != nil {
		return rep, fmt.Errorf("highlight_events doublons: détection: %w", err)
	}
	rep.After = rep.Before - rep.Removed()
	return rep, nil
}

// ApplyHighlightEventsDedupe retire les lignes écrites deux fois par reconstruction de la table.
// Sans table ou sans doublon, rien n'est écrit. Journalise les comptes avant et après.
func ApplyHighlightEventsDedupe(db *sql.DB) error {
	ctx := bootCtx()
	if err := recoverOrphanTable(ctx, db, "highlight_events", purgeSuffix); err != nil {
		return err
	}
	if ok, err := tableExists(db, "highlight_events"); err != nil || !ok {
		return err
	}
	rep, err := DetectHighlightEventsDuplicates(ctx, db)
	if err != nil || rep.Removed() == 0 {
		return err
	}
	keep := fmt.Sprintf(`WHERE id NOT IN (SELECT id FROM (%s))`, highlightEventsDoomedSQL)
	n, err := swapKeepingRows(ctx, db, "highlight_events", "", keep)
	if err != nil {
		return err
	}
	// La garde de cardinalité du swap prouve Rebuilt == Expected ; ici, que Expected est bien
	// « tout sauf les lignes détectées ».
	if n.Rebuilt != rep.After {
		return fmt.Errorf("highlight_events doublons: %d lignes gardées pour %d - %d retirées",
			n.Rebuilt, rep.Before, rep.Removed())
	}
	slog.InfoContext(ctx, "migration: highlight_events dédoublonné (swap, sans DELETE)",
		"before", rep.Before, "after", n.Rebuilt, "removed_bare_form", rep.Bare,
		"removed_recopied", rep.Recopied, "matches", rep.Matches)
	return nil
}
