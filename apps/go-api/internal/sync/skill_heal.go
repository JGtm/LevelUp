// Package sync — skill_heal.go : self-healing pour les matchs où team_mmr
// est NULL en DB (matchs syncés avant que GetMatchSkill ne soit câblé dans
// processMatch, ou avec un échec transitoire de l'endpoint skill).
//
// healSkillForMissingMatches détecte les matchs du joueur dont team_mmr est
// NULL dans shared.match_participants, appelle GetMatchSkill par batch, et
// UPSERT les colonnes skill via InsertParticipants (qui préserve les colonnes
// non-nulles existantes via COALESCE).
//
// Coût : 0 appel API si tout est déjà rempli. ~N appels API au premier passage
// post-déploiement, puis 0 ensuite (idempotent).
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// healSkillForMissingMatches récupère les match_id du joueur où team_mmr est
// NULL, fetch GetMatchSkill pour chacun, et UPSERT les rows. Best-effort : ne
// renvoie jamais d'erreur, juste un compteur (matches_healed, error).
func healSkillForMissingMatches(
	ctx context.Context,
	sharedDB *sql.DB,
	client HaloClient,
	xuid string,
	maxMatches int,
) (int, error) {
	// 1. Lister les matchs où ce joueur a une donnée skill manquante :
	// team_mmr NULL OR deaths_stddev NULL (= ancien binaire qui ne lisait
	// pas l'API skill ou n'extrayait pas Deaths.StdDev).
	const limitQuery = `
		SELECT match_id
		FROM match_participants
		WHERE xuid = ? AND (team_mmr IS NULL OR deaths_stddev IS NULL)
		ORDER BY match_id ASC
		LIMIT ?`
	if maxMatches <= 0 {
		maxMatches = 200
	}
	rows, err := sharedDB.QueryContext(ctx, limitQuery, xuid, maxMatches)
	if err != nil {
		return 0, fmt.Errorf("healSkill query: %w", err)
	}
	var matchIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("healSkill scan: %w", err)
		}
		matchIDs = append(matchIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("healSkill rows: %w", err)
	}
	if len(matchIDs) == 0 {
		return 0, nil
	}

	healed := 0
	for _, matchID := range matchIDs {
		// 2. Lister les XUIDs humains de ce match.
		humanXUIDs, err := loadHumanXUIDsForMatch(ctx, sharedDB, matchID)
		if err != nil {
			slog.WarnContext(ctx, "healSkill: load xuids échoué",
				"match_id", matchID, "err", err)
			continue
		}
		if len(humanXUIDs) == 0 {
			continue // bots-only ou match custom
		}

		// 3. GetMatchSkill.
		skill, err := client.GetMatchSkill(ctx, matchID, humanXUIDs)
		if err != nil {
			slog.WarnContext(ctx, "healSkill: GetMatchSkill échoué",
				"match_id", matchID, "err", err)
			continue
		}
		if len(skill) == 0 {
			continue // skill absent (custom/local)
		}

		// 4. Construire ParticipantRow minimaux et UPSERT (COALESCE conserve
		// les autres colonnes : kills, deaths, etc.).
		updates := make([]ParticipantRow, 0, len(skill))
		for _, sd := range skill {
			updates = append(updates, ParticipantRow{
				MatchID:        matchID,
				XUID:           sd.XUID,
				TeamMMR:        sd.TeamMMR,
				EnemyMMR:       sd.EnemyMMR,
				KillsExpected:  sd.KillsExpected,
				DeathsExpected: sd.DeathsExpected,
				KillsStddev:    sd.KillsStdDev,
			})
		}
		if err := InsertParticipants(sharedDB, updates); err != nil {
			slog.WarnContext(ctx, "healSkill: upsert échoué",
				"match_id", matchID, "err", err)
			continue
		}
		healed++
	}
	return healed, nil
}

// loadHumanXUIDsForMatch retourne les XUIDs humains (non-bots) participants
// au match. Utilisé pour passer la liste correcte à GetMatchSkill.
func loadHumanXUIDsForMatch(ctx context.Context, sharedDB *sql.DB, matchID string) ([]string, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT xuid FROM match_participants
		WHERE match_id = ? AND xuid NOT LIKE 'bid(%'
	`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var xuids []string
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			return nil, err
		}
		if x != "" {
			xuids = append(xuids, x)
		}
	}
	return xuids, rows.Err()
}
