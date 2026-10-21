package persist

// kill_summaries_relecture.go — LES RÉSUMÉS D'UNE PASSE RELUE EN BASE (ADR 0039, décision 5).
//
// Second adaptateur de la fonction pure `analysis/killsummary` : les lignes de la passe
// COURANTE d'un match, lues par `match_kill_events_latest` (ADR 0026 : jamais la table), avec
// les mêmes normalisations que l'adaptateur en mémoire — un NULL se lit comme le texte vide ou
// le SourceTag 0 que [killEventArgs] écrit NULL. Le test de parité des deux adaptateurs les
// tient égaux.
//
// Ce chemin sert les passes écrites hors de [insertKillEventRows] (migrations historiques, seed
// démo synthétique, passes antérieures aux résumés) et la réécriture d'une tête périmée après
// une montée de `killsummary.Rev`. Il écrit par la même fonction d'insertion,
// [insertKillSummaryRows], dans sa propre transaction.

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis/killsummary"
)

// latestKillPassQuery : les colonnes que les résumés lisent, sur la passe courante du match.
const latestKillPassQuery = `
	SELECT decode_pass, written_at, publishable,
	       victim_xuid, feed_killer_xuid, feed_killer_gamertag,
	       assist_xuid, assist_gamertag, assist_known,
	       source_tag, killer_damage_pct, assist_damage_pct
	FROM match_kill_events_latest
	WHERE match_id = ?`

// latestKillPass : la passe courante d'un match, sous la forme neutre des résumés.
type latestKillPass struct {
	decodePass  string
	writtenAt   time.Time
	publishable bool
	rows        []killsummary.Row
}

// readLatestKillPass relit la passe courante du match. `ok` est faux quand le match n'a aucune
// ligne au journal.
//
// `decode_pass`, `written_at` et `publishable` sont des attributs de la PASSE : la vue ne rend
// qu'une passe, qui porte la même valeur sur toutes ses lignes ; la dernière lue gagne.
func readLatestKillPass(ctx context.Context, q rowQuerier, matchID string) (latestKillPass, bool, error) {
	rows, err := q.QueryContext(ctx, latestKillPassQuery, matchID)
	if err != nil {
		return latestKillPass{}, false, fmt.Errorf("persist: relecture passe %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()
	var out latestKillPass
	for rows.Next() {
		r, err := scanLatestKillRow(rows, &out)
		if err != nil {
			return latestKillPass{}, false, fmt.Errorf("persist: relecture passe %s (scan): %w", matchID, err)
		}
		out.rows = append(out.rows, r)
	}
	if err := rows.Err(); err != nil {
		return latestKillPass{}, false, fmt.Errorf("persist: relecture passe %s (rows): %w", matchID, err)
	}
	return out, len(out.rows) > 0, nil
}

// scanLatestKillRow lit une ligne : les attributs de passe dans `p`, la mort en retour. Toute
// colonne nullable passe par un `sql.Null*` : un NULL scanné dans un type nu deviendrait une
// valeur mesurée.
func scanLatestKillRow(rows *sql.Rows, p *latestKillPass) (killsummary.Row, error) {
	var (
		r                               killsummary.Row
		victimX, killerX, killerGT      sql.NullString
		assistX, assistGT               sql.NullString
		sourceTag, killerPct, assistPct sql.NullInt64
	)
	if err := rows.Scan(&p.decodePass, &p.writtenAt, &p.publishable,
		&victimX, &killerX, &killerGT, &assistX, &assistGT, &r.AssistKnown,
		&sourceTag, &killerPct, &assistPct); err != nil {
		return r, err
	}
	r.VictimXUID, r.KillerXUID, r.KillerGamertag = victimX.String, killerX.String, killerGT.String
	r.AssistXUID, r.AssistGamertag = assistX.String, assistGT.String
	r.SourceTag = uint32(sourceTag.Int64)
	r.KillerDamagePct, r.AssistDamagePct = pctFromNull(killerPct), pctFromNull(assistPct)
	return r, nil
}

// PersistLatestKillPassSummaries écrit, dans sa propre transaction, une génération de résumés
// pour la passe COURANTE du match relue en base, à l'horodatage de cette passe (la tête reste
// ordonnée comme la passe qu'elle résume). Rend faux, sans rien écrire, pour un match sans
// ligne au journal.
//
// PRÉ-REQUIS : le caller tient l'écrivain de shared_matches_v2.duckdb (ADR 0013).
func PersistLatestKillPassSummaries(ctx context.Context, db txBeginner, matchID string) (bool, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("persist: BeginTx résumés %s: %w", matchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op après Commit

	pass, ok, err := readLatestKillPass(ctx, tx, matchID)
	if err != nil || !ok {
		return false, err
	}
	if err := insertKillSummaryRows(ctx, tx, killSummaryWrite{
		matchID: matchID, srcPass: pass.decodePass, writtenAt: pass.writtenAt,
		summary: killsummary.Summarize(pass.publishable, pass.rows),
	}); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("persist: Commit résumés %s: %w", matchID, err)
	}
	return true, nil
}
