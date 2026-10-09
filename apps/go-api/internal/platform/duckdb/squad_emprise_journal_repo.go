// Package duckdb — squad_emprise_journal_repo.go : les frags PAR ARME du journal des morts du film,
// lus par l'Emprise pour compter les frags aux armes spéciales sous la définition de leurs prises
// (analysis/squademprise/special_frags.go).
//
// UNE LECTURE BORNÉE PAR LA LISTE DES MATCHS, liée en une constante `VARCHAR[]` que DuckDB pousse
// sous la fenêtre de la vue `match_kill_events_latest` (ADR 0026 : la vue seulement ; ADR 0036 I2).
//
//	porte      `publishable` : la passe de décodage est fiable ligne à ligne. Un frag compté par
//	           ARME et par TUEUR est une lecture ligne à ligne ; un match dont la passe ne l'est pas
//	           n'est pas « lu » et l'Emprise retombe sur la feuille de match.
//	tueur      `feed_killer_xuid`, le tueur du kill-feed, et `victim_xuid` : le calcul en tire le camp
//	           du tueur, écarte suicides et trahisons, et range un tueur bot (xuid NULL) d'après sa
//	           victime (analysis/squademprise/special_frags.go).
//	arme       `source_tag` traduit en clé de registre par le classificateur du titre ; une source
//	           sans clé (mêlée, grenade, environnement) n'est la frappe d'aucune arme de socle.
//
// Sans classificateur (titre sans `film.kill_source`) : aucun match lu. Vue absente :
// games.ErrCapabilityNotSupported.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/games"
)

// qSquadEmpriseJournal : les lignes publiables des matchs demandés (%s : le prédicat de liste), par
// (match, tueur, victime, source). Une ligne sans source dit que la passe est lue sans être comptée.
const qSquadEmpriseJournal = `
SELECT k.match_id, k.feed_killer_xuid, k.victim_xuid, k.source_tag, COUNT(*)::INTEGER AS frags
FROM ` + KillEventsCanonicalTable + ` k
WHERE %s AND k.publishable
GROUP BY k.match_id, k.feed_killer_xuid, k.victim_xuid, k.source_tag
ORDER BY k.match_id, k.feed_killer_xuid, k.victim_xuid, k.source_tag`

// LoadJournalWeaponKills rend les frags par (match, tueur, victime, clé d'arme) du journal des
// morts, et les matchs dont le journal est publiable. Liste vide ou sans classificateur : lecture
// vide.
func (r *SquadEmpriseRepo) LoadJournalWeaponKills(ctx context.Context, matchIDs []string) (squademprise.JournalRead, error) {
	out := squademprise.JournalRead{Read: map[string]bool{}}
	if len(matchIDs) == 0 {
		return out, nil
	}
	if r.classifier == nil {
		slog.DebugContext(ctx, "SquadEmpriseRepo: aucun classificateur de source de degat : journal non lu",
			"matchs", len(matchIDs))
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, squadEmpriseQueryTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("SquadEmpriseRepo: shared reader: %w", err)
	}
	defer release()

	par, arg := clauseListeMatchs("k.match_id", matchIDs)
	rows, err := db.QueryContext(ctx, fmt.Sprintf(qSquadEmpriseJournal, par), arg)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "SquadEmpriseRepo: journal des morts absent",
				"table", KillEventsCanonicalTable, "err", err)
			return out, games.ErrCapabilityNotSupported
		}
		return out, fmt.Errorf("SquadEmpriseRepo: journal query: %w", err)
	}
	defer rows.Close()
	sansCle := 0
	for rows.Next() {
		var (
			match  string
			killer sql.NullString
			victim sql.NullString
			tag    sql.NullInt64
			n      int
		)
		if err := rows.Scan(&match, &killer, &victim, &tag, &n); err != nil {
			return out, fmt.Errorf("SquadEmpriseRepo: journal scan: %w", err)
		}
		out.Read[match] = true
		// Sans source, ou tueur et victime tous deux bots : ni arme ni camp à lire.
		if !tag.Valid || (!killer.Valid && !victim.Valid) {
			continue
		}
		key, ok := r.classifier.KillSourceRegistryKey(uint32(tag.Int64))
		if !ok {
			sansCle += n
			continue
		}
		out.Rows = append(out.Rows, squademprise.JournalKillRow{
			MatchID: match, XUID: killer.String, VictimXUID: victim.String, WeaponKey: key, Kills: n,
		})
	}
	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("SquadEmpriseRepo: journal rows: %w", err)
	}
	slog.DebugContext(ctx, "SquadEmpriseRepo: journal des morts lu",
		"matchs", len(matchIDs), "matchs_publiables", len(out.Read), "lignes", len(out.Rows), "frags_sans_cle", sansCle)
	return out, nil
}
