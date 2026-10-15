package migration

// highlight_phantoms.go — RETIRER DE `highlight_events` LES LIGNES FANTOMES DU LECTEUR DES TEMPS FORTS.
//
// Avant la correction « un marqueur de fin = un evenement » (grammar.Rev du 2026-10-15), le
// lecteur des temps forts pouvait decoder un bloc DEUX FOIS, sous deux XUID : un faux motif de XUID
// dans les octets de la fiche d un joueur precedait le vrai de quelques milliers de bits. Les deux
// lignes sont dans `highlight_events` (86 matchs au parc du 2026-10-15, un seul joueur touche).
//
// UNE LIGNE FANTOME est une ligne dont le XUID n est PAS un participant du match et qui a un
// jumeau (meme match, type, instant, type_hint, raw_json) porte par un XUID qui, lui, l est. La
// regle ne devine rien : un fantome n existe que face a son vrai. Un match sans participants
// charges n en produit aucune (rien n est retire dans le doute).
//
// SANS DELETE : `highlight_events` porte des index ART (PK, idx_highlight_match,
// idx_events_match_type) ; un DELETE y est le declencheur du bug DuckDB ART #23645 (ADR 0019). La
// table est reconstruite a cote, sans les fantomes, puis echangee (swapKeepingRows, ADR 0026) ;
// les `id` gardes et la sequence ne bougent pas. Idempotent : sans fantome, rien n est ecrit.
// Operation d ENTRETIEN, serveur arrete (`levelup repair-highlight-phantoms`).

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
)

// phantomRowsSQL rend `(id, match_id)` de chaque ligne fantome.
const phantomRowsSQL = `
	WITH h AS (
		SELECT he.id, he.match_id, he.event_type, he.time_ms, he.type_hint,
		       COALESCE(he.raw_json, '') AS rj, (p.xuid IS NOT NULL) AS est_participant
		FROM highlight_events he
		LEFT JOIN match_participants p ON p.match_id = he.match_id AND p.xuid = he.xuid
	), g AS (
		SELECT match_id, event_type, time_ms, type_hint, rj FROM h
		GROUP BY ALL HAVING bool_or(est_participant) AND bool_or(NOT est_participant)
	)
	SELECT h.id, h.match_id FROM h
	JOIN g USING (match_id, event_type, time_ms, type_hint, rj)
	WHERE NOT h.est_participant`

// HighlightPhantomReport : ce que la detection a trouve (et, hors dry-run, retire).
type HighlightPhantomReport struct {
	Rows     int
	MatchIDs []string
	Purged   bool
}

// DetectHighlightPhantoms mesure les lignes fantomes sans rien ecrire.
func DetectHighlightPhantoms(ctx context.Context, q lecteurSQL) (HighlightPhantomReport, error) {
	rows, err := q.QueryContext(ctx, phantomRowsSQL)
	if err != nil {
		return HighlightPhantomReport{}, fmt.Errorf("highlight_events fantomes: detection: %w", err)
	}
	defer rows.Close() //nolint:errcheck // lecture seule, l erreur d iteration est lue par rows.Err
	var rep HighlightPhantomReport
	vus := map[string]bool{}
	for rows.Next() {
		var id int64
		var m string
		if err := rows.Scan(&id, &m); err != nil {
			return HighlightPhantomReport{}, fmt.Errorf("highlight_events fantomes: scan: %w", err)
		}
		rep.Rows++
		if !vus[m] {
			vus[m] = true
			rep.MatchIDs = append(rep.MatchIDs, m)
		}
	}
	if err := rows.Err(); err != nil {
		return HighlightPhantomReport{}, fmt.Errorf("highlight_events fantomes: lignes: %w", err)
	}
	sort.Strings(rep.MatchIDs)
	return rep, nil
}

// PurgeHighlightPhantoms retire les lignes fantomes par reconstruction de la table. Sans fantome
// ou en `dryRun`, rien n est ecrit.
func PurgeHighlightPhantoms(ctx context.Context, db *sql.DB, dryRun bool) (HighlightPhantomReport, error) {
	rep, err := DetectHighlightPhantoms(ctx, db)
	if err != nil || rep.Rows == 0 || dryRun {
		return rep, err
	}
	var total int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM highlight_events`).Scan(&total); err != nil {
		return rep, fmt.Errorf("highlight_events fantomes: compte: %w", err)
	}
	keep := fmt.Sprintf(`WHERE id NOT IN (SELECT id FROM (%s))`, phantomRowsSQL)
	n, err := swapKeepingRows(ctx, db, "highlight_events", "", keep)
	if err != nil {
		return rep, err
	}
	// La garde de cardinalite du swap prouve Rebuilt == Expected ; ici, que Expected est bien
	// « tout sauf les fantomes ».
	if n.Expected != total-int64(rep.Rows) {
		return rep, fmt.Errorf("highlight_events fantomes: %d lignes gardees pour %d - %d fantomes",
			n.Expected, total, rep.Rows)
	}
	rep.Purged = true
	return rep, nil
}
