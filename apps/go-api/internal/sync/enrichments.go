// Package sync — enrichments.go : helpers de calcul/dérivation pour les
// colonnes qui ne viennent pas directement de l'API mais sont calculées
// depuis les données déjà chargées.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// ensureGamertagForSelf garantit que le ParticipantRow correspondant au xuid
// du joueur synchronisé porte un gamertag — si l'API renvoie Gamertag/PlayerName
// vide pour le joueur appelant, on tombe sur le gamertag connu du SyncEngine.
func ensureGamertagForSelf(participants []ParticipantRow, selfXUID, selfGamertag string) {
	if selfXUID == "" || selfGamertag == "" {
		return
	}
	for i := range participants {
		if participants[i].XUID == selfXUID && (participants[i].Gamertag == nil || *participants[i].Gamertag == "") {
			gt := selfGamertag
			participants[i].Gamertag = &gt
			return
		}
	}
}

// computeAndPersistHadBotTeammate calcule `had_bot_teammate` pour les matchs
// du joueur (xuid) et l'écrit dans player_match_enrichment. Vrai si au moins
// un coéquipier (même team_id, xuid != self) a un xuid commençant par `bid(`.
//
// Idempotent : recalcule pour TOUS les matchs (cheap : SQL pur, pas d'API).
// Best-effort : skip silencieux si schema absent.
func computeAndPersistHadBotTeammate(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (int, error) {
	// Étape 1 : trouver les match_ids où le joueur a au moins un teammate bot.
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT DISTINCT mp_self.match_id
		FROM match_participants mp_self
		JOIN match_participants mp_other
		  ON mp_self.match_id = mp_other.match_id
		 AND mp_self.team_id = mp_other.team_id
		 AND mp_other.xuid <> mp_self.xuid
		WHERE mp_self.xuid = ?
		  AND mp_other.xuid LIKE 'bid(%'
	`, xuid)
	if err != nil {
		return 0, fmt.Errorf("had_bot_teammate query: %w", err)
	}
	defer rows.Close()

	var matchIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		matchIDs = append(matchIDs, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(matchIDs) == 0 {
		return 0, nil
	}

	// Étape 2 : UPDATE player_match_enrichment.had_bot_teammate = TRUE.
	// On utilise IN (?,?,?) avec placeholders pour éviter une boucle.
	placeholders := strings.TrimRight(strings.Repeat("?,", len(matchIDs)), ",")
	args := make([]any, 0, len(matchIDs))
	for _, id := range matchIDs {
		args = append(args, id)
	}
	q := fmt.Sprintf(`
		UPDATE player_match_enrichment
		SET had_bot_teammate = TRUE
		WHERE match_id IN (%s)
		  AND COALESCE(had_bot_teammate, FALSE) = FALSE
	`, placeholders)
	res, err := playerDB.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("had_bot_teammate update: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
