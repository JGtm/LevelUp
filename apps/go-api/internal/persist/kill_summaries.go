package persist

// kill_summaries.go — L'ÉCRITURE DES RÉSUMÉS PAR MATCH DU JOURNAL DES MORTS (ADR 0039).
//
// Une passe de `match_kill_events` porte dans SA transaction, avec SON horodatage, une
// génération de résumés : une tête (`match_kill_heads`, S0), les frags par paire et source
// (`match_kill_pairs`, S1), les appuis par paire (`match_assist_pairs`, S2). Le calcul est la
// fonction pure `analysis/killsummary` ; ce fichier porte l'adaptateur des morts EN MÉMOIRE et
// LA fonction d'insertion, [insertKillSummaryRows], que tous les écrivains empruntent :
// [insertKillEventRows] (chaque passe écrite), et la relecture d'une passe en base
// (kill_summaries_relecture.go : rattrapage, seed démo synthétique).
//
// ANTI-ART (ADR 0019/0026/0030) : INSERT purs, aucun DELETE / UPDATE / ON CONFLICT. Remplacer
// une génération consiste à en écrire une nouvelle ; les vues `_latest` (la tête décide) ne
// servent que la dernière.
//
// COUPLAGE ASSUMÉ : une erreur base pendant l'écriture des résumés annule la transaction de la
// passe — dans le sync primaire, celle du match entier, comme une erreur du journal. Le journal
// et ses résumés n'existent jamais l'un sans l'autre pour une passe écrite en ligne.

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis/killsummary"
)

// killSummaryWrite : une génération de résumés à écrire.
type killSummaryWrite struct {
	matchID string
	// srcPass : le `decode_pass` de la passe résumée.
	srcPass string
	// writtenAt : l'horodatage de la passe résumée. La tête est ordonnée comme la passe : une
	// tête plus récente résume toujours une passe au moins aussi récente.
	writtenAt time.Time
	summary   killsummary.Summary
}

// summaryOfPass : la génération de résumés d'une passe qu'on écrit, après les normalisations
// de [killEventArgs] (texte vide = NULL, SourceTag 0 = source non mesurée).
func summaryOfPass(in KillSourceBatch, pass string, now time.Time) killSummaryWrite {
	return killSummaryWrite{
		matchID: in.MatchID, srcPass: pass, writtenAt: now,
		summary: killsummary.Summarize(in.Publishable, killSummaryRowsOf(in.Deaths)),
	}
}

// killSummaryRowsOf : l'adaptateur des morts EN MÉMOIRE vers le type neutre. Les champs vides
// gardent leur sens de NULL (killEventArgs les écrit NULL) ; la victime gamertag, la portée et
// l'instant n'entrent dans aucun résumé.
func killSummaryRowsOf(deaths []KillEventInsert) []killsummary.Row {
	out := make([]killsummary.Row, len(deaths))
	for i := range deaths {
		d := &deaths[i]
		out[i] = killsummary.Row{
			VictimXUID: d.VictimXUID,
			KillerXUID: d.FeedKillerXUID, KillerGamertag: d.FeedKillerGamertag,
			AssistXUID: d.AssistXUID, AssistGamertag: d.AssistGamertag, AssistKnown: d.AssistKnown,
			SourceTag:       d.SourceTag,
			KillerDamagePct: d.KillerDamagePct, AssistDamagePct: d.AssistDamagePct,
		}
	}
	return out
}

// insertKillSummaryRows écrit UNE génération de résumés dans la transaction FOURNIE, sans la
// committer : une tête, ses paires, ses appuis, sous un `summary_pass` neuf.
func insertKillSummaryRows(ctx context.Context, tx *sql.Tx, w killSummaryWrite) error {
	gen, err := newDecodePassID()
	if err != nil {
		return fmt.Errorf("persist: summary_pass %s: %w", w.matchID, err)
	}
	cols := summaryColumns{matchID: w.matchID, gen: gen, rev: w.summary.Rev, src: w.srcPass, at: w.writtenAt}
	h := w.summary.Head
	if _, err := tx.ExecContext(ctx, insertKillHeadSQL, cols.args(
		h.Publishable, h.Rows, h.AssistKnown, h.KillerXUID, h.SourceAndKiller)...); err != nil {
		return fmt.Errorf("persist: INSERT match_kill_heads %s: %w", w.matchID, err)
	}
	if err := insertKillPairRows(ctx, tx, cols, w.summary.Pairs); err != nil {
		return err
	}
	return insertAssistPairRows(ctx, tx, cols, w.summary.Assists)
}

// summaryColumns : les colonnes communes aux trois familles, dans l'ordre des INSERT.
type summaryColumns struct {
	matchID, gen, rev, src string
	at                     time.Time
}

func (c summaryColumns) args(rest ...any) []any {
	return append([]any{c.matchID, c.gen, c.rev, c.src, c.at}, rest...)
}

func insertKillPairRows(ctx context.Context, tx *sql.Tx, cols summaryColumns, pairs []killsummary.Pair) error {
	if len(pairs) == 0 {
		return nil
	}
	stmt, err := tx.PrepareContext(ctx, insertKillPairSQL)
	if err != nil {
		return fmt.Errorf("persist: prepare match_kill_pairs %s: %w", cols.matchID, err)
	}
	defer func() { _ = stmt.Close() }()
	for _, p := range pairs {
		if _, err := stmt.ExecContext(ctx, cols.args(
			nullIfEmpty(p.KillerXUID), nullIfEmpty(p.VictimXUID),
			nullUnless(p.SourceTag != 0, int64(p.SourceTag)), p.Kills)...); err != nil {
			return fmt.Errorf("persist: INSERT match_kill_pairs %s: %w", cols.matchID, err)
		}
	}
	return nil
}

func insertAssistPairRows(ctx context.Context, tx *sql.Tx, cols summaryColumns, groups []killsummary.AssistPair) error {
	if len(groups) == 0 {
		return nil
	}
	stmt, err := tx.PrepareContext(ctx, insertAssistPairSQL)
	if err != nil {
		return fmt.Errorf("persist: prepare match_assist_pairs %s: %w", cols.matchID, err)
	}
	defer func() { _ = stmt.Close() }()
	for _, g := range groups {
		if _, err := stmt.ExecContext(ctx, cols.args(
			nullIfEmpty(g.AssistXUID), nullIfEmpty(g.AssistGamertag),
			nullIfEmpty(g.KillerXUID), nullIfEmpty(g.KillerGamertag), nullIfEmpty(g.VictimXUID),
			g.N, g.Stolen, g.PctSum, g.PctCount, g.Low, g.Mid, g.High)...); err != nil {
			return fmt.Errorf("persist: INSERT match_assist_pairs %s: %w", cols.matchID, err)
		}
	}
	return nil
}

const insertKillHeadSQL = `
	INSERT INTO match_kill_heads
		(match_id, summary_pass, summary_rev, src_decode_pass, written_at,
		 publishable, n_rows, n_assist_known, n_killer_xuid, n_src_killer)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

const insertKillPairSQL = `
	INSERT INTO match_kill_pairs
		(match_id, summary_pass, summary_rev, src_decode_pass, written_at,
		 feed_killer_xuid, victim_xuid, source_tag, kills)
	VALUES (?, ?, ?, ?, ?, ?, ?, CAST(? AS UINTEGER), ?)`

const insertAssistPairSQL = `
	INSERT INTO match_assist_pairs
		(match_id, summary_pass, summary_rev, src_decode_pass, written_at,
		 assist_xuid, assist_gamertag, feed_killer_xuid, feed_killer_gamertag, victim_xuid,
		 n, n_stolen, sum_assist_pct, n_assist_pct, n_low, n_mid, n_high)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
