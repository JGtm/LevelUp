// Package duckdb — tactical_repo_ownership.go : L'OUVRABILITE D'UN MATCH POUR UN JOUEUR
// (ADR 0029, lot M1 — detail d'une cellule, Tactique S.1).
//
// # POURQUOI CETTE VERIFICATION EST SEPAREE DES TROIS LECTURES
//
// KillPositions, KillEvents et MortsAvecContexte filtrent DEJA leur univers par
// `mp.xuid = ?` (cf. tactical_repo_univers.go) : tout ce qu'elles rendent est donc
// OUVRABLE par construction. Cette methode-ci verifie autre chose — le PERIMETRE
// DEMANDE lui-meme (`scope.MatchIDs`, le corps de la requete), AVANT toute lecture
// d'occupation. Un client qui poserait un match_id d'un autre joueur dans ce
// corps (bug, cache perime, appel hostile) doit voir ce match COMPTE dans
// `matchs_non_ouvrables`, jamais mele aux contributions — meme garde que la
// Couche B de l'ADR (`MatchViewRepo.IsParticipant` / `Q17bIsParticipant`), en
// version BATCH puisque le detail d'une cellule verifie tout le perimetre d'un
// coup plutot qu'un match a la fois.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// QTacticalMatchsOuvrablesTemplate : %s = la date de debut canonique (StartTimeCanonicalSQL,
// jamais le littéral recopié — garde-rail archlint/no_raw_start_time_literal_test.go), %s =
// les points d'interrogation du IN.
//
// L'EXISTS EST LA MEME NATURE DE GARDE QUE Q17bIsParticipant (queries_match.go) : la
// participation du joueur au match, sans condition de carte ni de perimetre — ce n'est PAS
// une seconde definition de l'univers tactique, c'est une verification INDEPENDANTE de ce
// que le client a demande.
//
// campaignExclusionToken TERMINE LE WHERE (prefixe `Q`, garde-rail structurel
// campaign_exclusion_guard_test.go) : sans lui, un match de Campagne Halo 5 (masque
// partout ailleurs cote lecture) serait declare ouvrable ici. Resolu au call site par
// resolveCampaignExclusion, qui connait le titre du joueur (no-op pour Infinite).
const QTacticalMatchsOuvrablesTemplate = `
SELECT mr.match_id, %s AS start_time_utc
FROM match_registry mr
WHERE mr.match_id IN (%s)
  AND EXISTS (SELECT 1 FROM match_participants mp WHERE mp.match_id = mr.match_id AND mp.xuid = ?)` +
	campaignExclusionToken

// MatchsOuvrables verifie, pour `matchIDs`, lesquels `playerXUID` a REELLEMENT joues.
//
// UN MATCH ABSENT DU RESULTAT N'EST PAS OUVRABLE : ni erreur (les matchs etrangers sont un
// etat NOMINAL du perimetre — un client peut toujours en poser un par accident), ni entree a
// zero-value ambigue (une date zero se confondrait avec « inconnue » plutot que « refusee »).
func (r *TacticalRepo) MatchsOuvrables(
	ctx context.Context, playerXUID string, matchIDs []string,
) (map[string]time.Time, error) {
	out := make(map[string]time.Time, len(matchIDs))
	if playerXUID == "" || len(matchIDs) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, tacticalReadTimeout)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "TacticalRepo.MatchsOuvrables: shared reader", "err", err)
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	// DEDOUBLONNE : un match_id repete dans le perimetre ne doit pas gonfler le IN, ni
	// entrainer plusieurs lignes pour la meme cle dans la map de sortie.
	uniques := dedupliquer(matchIDs)
	query := resolveCampaignExclusion(
		fmt.Sprintf(QTacticalMatchsOuvrablesTemplate, StartTimeCanonicalSQL("mr"), Placeholders(len(uniques))),
		r.pdb.TitleSlug, "mr")
	args := append(ToAnySlice(uniques), playerXUID)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, r.degrader(ctx, "MatchsOuvrables", err)
	}
	err = scanRows(ctx, rows, "TacticalRepo.MatchsOuvrables", func(sc rowScanner) error {
		var matchID string
		var start sql.NullTime
		if err := sc.Scan(&matchID, &start); err != nil {
			return err
		}
		if start.Valid {
			out[matchID] = start.Time
		} else {
			// Un match ouvrable dont la date est inconnue reste OUVRABLE : la nullite ne
			// porte que sur le tri des contributions (elle finit en fin de liste), jamais
			// sur le verdict de participation.
			out[matchID] = time.Time{}
		}
		return nil
	})
	return out, err
}

// dedupliquer rend `ids` sans doublon, ordre d'entree conserve.
func dedupliquer(ids []string) []string {
	vus := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if vus[id] {
			continue
		}
		vus[id] = true
		out = append(out, id)
	}
	return out
}
