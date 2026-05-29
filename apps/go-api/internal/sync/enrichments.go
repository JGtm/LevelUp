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

// Seuils pour considérer un bot teammate comme polluant les stats du joueur.
// Hybride : plancher absolu + ratio match. Validé 2026-05-27 (cf. thought_log).
// Un bot anecdotique (humain qui quitte 10s avant la fin, etc.) ne déclenche
// plus le flag — évite la pollution des "pires performances" et de la pill UI.
//
// botLateJoinIgnoreRatio : si un bot est `joined_in_progress = TRUE` et que
// son time_played représente moins de cette fraction du match, on l'ignore
// dans la somme polluante. Sémantique : "remplaçant tardif sans impact
// significatif". Les bots `present_at_beginning = TRUE` sont toujours
// comptés intégralement (déséquilibre dès le début).
const (
	botPresenceMinSeconds  = 30
	botPresenceMinRatio    = 0.15
	botLateJoinIgnoreRatio = 0.30
)

// computeAndPersistHadBotTeammate calcule had_bot_teammate selon un seuil
// hybride : un bot coéquipier compte UNIQUEMENT si la SOMME de son time_played
// dans le match dépasse botPresenceMinSeconds (30s) ET botPresenceMinRatio
// (15%) de duration_seconds. Sinon il est considéré comme anecdotique.
//
// Raffinement late-join (mini-Phase 0.5 colonnes ParticipationInfo) : un bot
// `joined_in_progress = TRUE` dont le time_played représente moins de
// botLateJoinIgnoreRatio (30%) du match est ignoré dans la somme. Sémantique :
// remplaçant tardif sans impact significatif. Les bots `present_at_beginning
// = TRUE` sont toujours comptés (déséquilibre dès le début).
//
// La fonction écrit dans player_match_enrichment.had_bot_teammate :
//   - TRUE pour les matchs dont la présence bot dépasse les seuils
//   - FALSE pour les matchs avec bot mais sous les seuils (réversion correcte
//     des flags TRUE laissés par l'ancien algo binaire pré-2026-05-27)
//
// Idempotent : recalcule pour TOUS les matchs (cheap : SQL pur, pas d'API).
// Best-effort : skip silencieux si schema absent.
func computeAndPersistHadBotTeammate(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (int, error) {
	// Étape 1a : matchs avec au moins un bot teammate (sans seuil).
	anyBotIDs, err := queryAnyBotMatchIDs(ctx, sharedDB, xuid)
	if err != nil {
		return 0, fmt.Errorf("had_bot_teammate any-bot query: %w", err)
	}

	// Étape 1b : sous-ensemble qui passe les seuils (présence significative).
	sigBotIDs, err := querySignificantBotMatchIDs(ctx, sharedDB, xuid)
	if err != nil {
		return 0, fmt.Errorf("had_bot_teammate significant query: %w", err)
	}

	// Différence : matchs avec bot SOUS le seuil → repasser à FALSE si
	// l'ancien algo binaire les avait marqués TRUE.
	sigSet := make(map[string]struct{}, len(sigBotIDs))
	for _, id := range sigBotIDs {
		sigSet[id] = struct{}{}
	}
	var nonSigBotIDs []string
	for _, id := range anyBotIDs {
		if _, ok := sigSet[id]; !ok {
			nonSigBotIDs = append(nonSigBotIDs, id)
		}
	}

	total := 0
	// Étape 2 : SET TRUE pour les matchs significatifs (idempotent).
	n, err := setHadBotFlag(ctx, playerDB, sigBotIDs, true)
	if err != nil {
		return 0, fmt.Errorf("had_bot_teammate set TRUE: %w", err)
	}
	total += n

	// Étape 3 : SET FALSE pour les matchs avec bot mais sous le seuil
	// (réversion des anciens flags TRUE de l'algo binaire).
	n, err = setHadBotFlag(ctx, playerDB, nonSigBotIDs, false)
	if err != nil {
		return 0, fmt.Errorf("had_bot_teammate set FALSE: %w", err)
	}
	total += n

	return total, nil
}

// queryAnyBotMatchIDs retourne les match_ids où le joueur a au moins un
// teammate bot (sans condition de durée). Sert à identifier l'ensemble à
// re-évaluer (incluant les flags TRUE de l'ancien algo à potentiellement reset).
func queryAnyBotMatchIDs(ctx context.Context, sharedDB *sql.DB, xuid string) ([]string, error) {
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
		return nil, err
	}
	defer rows.Close()
	return scanMatchIDs(rows)
}

// querySignificantBotMatchIDs filtre par seuil hybride. Un match est
// significatif ssi la SOMME des bot_time_played (avec les late-join
// inférieurs à botLateJoinIgnoreRatio comptés à 0) atteint à la fois
// botPresenceMinSeconds (plancher absolu) ET botPresenceMinRatio (ratio match).
//
// Le CASE dans le SUM écarte les bots `joined_in_progress=TRUE` dont le
// time_played représente moins de botLateJoinIgnoreRatio du match —
// typiquement un humain qui quitte près de la fin et est remplacé par un bot
// pour quelques secondes (cf. cas Madina/615b3ebc, bot 6s sur 390s).
func querySignificantBotMatchIDs(ctx context.Context, sharedDB *sql.DB, xuid string) ([]string, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		WITH bot_presence AS (
		    SELECT mp_self.match_id,
		           SUM(
		               CASE
		                   WHEN COALESCE(mp_bot.joined_in_progress, FALSE) = TRUE
		                        AND COALESCE(r.duration_seconds, 0) > 0
		                        AND CAST(COALESCE(mp_bot.time_played_seconds, 0) AS DOUBLE)
		                            / r.duration_seconds < ?
		                   THEN 0
		                   ELSE COALESCE(mp_bot.time_played_seconds, 0)
		               END
		           ) AS total_bot_seconds,
		           MAX(COALESCE(r.duration_seconds, 0)) AS match_duration
		    FROM match_participants mp_self
		    JOIN match_participants mp_bot
		      ON mp_self.match_id = mp_bot.match_id
		     AND mp_self.team_id = mp_bot.team_id
		     AND mp_bot.xuid <> mp_self.xuid
		    JOIN match_registry r
		      ON r.match_id = mp_self.match_id
		    WHERE mp_self.xuid = ?
		      AND mp_bot.xuid LIKE 'bid(%'
		    GROUP BY mp_self.match_id
		)
		SELECT match_id
		FROM bot_presence
		WHERE total_bot_seconds >= ?
		  AND match_duration > 0
		  AND CAST(total_bot_seconds AS DOUBLE) / match_duration >= ?
	`, botLateJoinIgnoreRatio, xuid, botPresenceMinSeconds, botPresenceMinRatio)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMatchIDs(rows)
}

func scanMatchIDs(rows *sql.Rows) ([]string, error) {
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// setHadBotFlag UPDATE player_match_enrichment.had_bot_teammate pour les
// match_ids donnés. Filtre "current != value" pour garder l'idempotence
// (un 2e run consécutif renvoie 0 row affectée).
func setHadBotFlag(ctx context.Context, playerDB *sql.DB, matchIDs []string, value bool) (int, error) {
	if len(matchIDs) == 0 {
		return 0, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(matchIDs)), ",")
	currentClause := "COALESCE(had_bot_teammate, FALSE) = FALSE"
	if !value {
		currentClause = "COALESCE(had_bot_teammate, FALSE) = TRUE"
	}
	q := fmt.Sprintf(`
		UPDATE player_match_enrichment
		SET had_bot_teammate = ?
		WHERE match_id IN (%s)
		  AND %s
	`, placeholders, currentClause)
	args := make([]any, 0, 1+len(matchIDs))
	args = append(args, value)
	for _, id := range matchIDs {
		args = append(args, id)
	}
	res, err := playerDB.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
