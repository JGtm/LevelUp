package replayartifacts

// usage_frags.go — LE FIL DES MORTS DU RÉSUMÉ D'USAGE : les frags du lot, lus en base AVANT tout
// segment d'écriture (même règle que les niveaux d'armes et les véhicules, cf. [Deriver]), pour
// les frags sans bouclier (`replay.AttachShieldlessKills`, citation « Presque, ça ne compte pas »).
//
// UNE LECTURE EN ÉCHEC N'ARRÊTE PAS LE RÉSUMÉ : ses autres grandeurs ne dépendent pas de la base.
// Les frags du lot restent alors « non mesurés » (colonne NULL), et c'est journalisé en ERROR ;
// `levelup backfill-usage-summary` les rejoue.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// LireFragsDuFilDesMorts lit, pour un lot de matchs, les frags de `match_kill_events_latest`. Rend
// une entrée par match demandé : un match sans aucune ligne est « non lu » (Read faux), jamais zéro.
//
// EXPORTEE pour le backfill, qui doit lire les frags comme le fil de l'eau.
func LireFragsDuFilDesMorts(ctx context.Context, db *sql.DB, ids []string) (map[string]replay.ShieldlessKillsInput, error) {
	out := make(map[string]replay.ShieldlessKillsInput, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := db.QueryContext(ctx, `SELECT match_id, time_ms, COALESCE(feed_killer_xuid, ''), COALESCE(victim_xuid, '')
		FROM match_kill_events_latest
		WHERE match_id IN (?`+strings.Repeat(",?", len(ids)-1)+`)
		ORDER BY match_id, time_ms`, args...)
	if err != nil {
		return nil, fmt.Errorf("resume d'usage: lecture du fil des morts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, killer, victim string
		var timeMS int
		if err := rows.Scan(&id, &timeMS, &killer, &victim); err != nil {
			return nil, fmt.Errorf("resume d'usage: scan du fil des morts: %w", err)
		}
		in := out[id]
		in.Read = true
		in.Kills = append(in.Kills, replay.ShieldlessKillRef{Killer: killer, Victim: victim, TimeMS: timeMS})
		out[id] = in
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resume d'usage: lignes du fil des morts: %w", err)
	}
	return out, nil
}

// preparerFragsDuResume pose sur chaque artefact lu les frags de son match. Best-effort : sans
// segment de lecture ou sur une lecture en échec, les frags restent non lus.
func preparerFragsDuResume(ctx context.Context, d Deps, lus []artefactLu) {
	if len(lus) == 0 {
		return
	}
	if d.WithRead == nil {
		slog.WarnContext(ctx, "post-sync: résumé d'usage — aucun segment de lecture câblé, frags sans bouclier non mesurés",
			"titleSlug", d.TitleSlug, "matchs", len(lus))
		return
	}
	var frags map[string]replay.ShieldlessKillsInput
	var lecture error
	ouvert := false
	d.WithRead(ctx, "fil des morts du résumé d'usage", func(sharedDB *sql.DB) {
		ouvert = true
		frags, lecture = LireFragsDuFilDesMorts(ctx, sharedDB, matchIDsDuLot(lus))
	})
	if !ouvert || lecture != nil {
		slog.ErrorContext(ctx, "post-sync: résumé d'usage — fil des morts illisible, frags sans bouclier non mesurés "+
			"(backfill-usage-summary les rejouera)",
			"titleSlug", d.TitleSlug, "matchs", len(lus), "segment_ouvert", ouvert, "err", lecture)
		return
	}
	for i := range lus {
		lus[i].frags = frags[lus[i].matchID]
	}
}
