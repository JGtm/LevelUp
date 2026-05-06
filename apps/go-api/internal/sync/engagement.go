// Package sync — engagement.go : calcul batch du score d'engagement par match.
//
// Reference plan : .ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md §3 (Sync/Backfill).
//
// Pipeline :
//   - Selection des matchs PvP du joueur sans engagement_score (ou tous si force)
//   - Pour chaque match :
//   - Load events depuis shared.highlight_events (joueur + autres humains)
//   - Load metadata match_registry (start_time, end_time, mode flags)
//   - Determine NTeam / NHumansLobby via match_participants
//   - Compute via temporal.ComputeEngagementScore (cold start coefs = 1.0)
//   - Persist engagement_score / engagement_score_brut / confidence dans
//     player_match_enrichment + mode_category
//
// Dependances :
//   - Necessite que MBitEvents soit set (highlight_events charges)
//   - Necessite que la migration Phase 2 ait ete appliquee (colonnes
//     engagement_score* presentes). Sinon skip silencieux avec warning.
//
// Cold start : si aucun coef stocke pour le joueur sur la categorie de mode,
// utilise 1.0 / 1.0 comme defauts neutres ("fait sa part"). Les coefficients
// seront raffines par recompute en Phase 3.b (recompute coefs par categorie).
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability"
)

// engagementMatchRow regroupe les inputs necessaires pour calculer le score
// d'un match donne.
type engagementMatchRow struct {
	MatchID       string
	StartTimeMS   int64
	EndTimeMS     int64
	IsRanked      bool
	IsPvE         bool
	TargetTeamID  int  // team du joueur cible
	NTeam         int  // taille de l'equipe alliee humains (joueur cible inclus)
	NHumansLobby  int  // taille du lobby humains
	IsTeamMode    bool // false si NTeam == 1 (FFA-like)
	PersonalScore int
	Kills         int
	Assists       int
}

// batchComputeEngagementScores calcule les engagement_score manquants pour le
// joueur. Retourne le nombre de matchs mis a jour.
//
// Si force=true, recalcule pour tous les matchs (pas seulement les manquants).
//
// Skip silencieux si la migration Phase 2 n'est pas appliquee (colonne
// engagement_score absente). Detection via information_schema.
func batchComputeEngagementScores(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	force bool,
) (int, error) {
	if !engagementColumnsAvailable(playerDB) {
		slog.WarnContext(ctx, "engagement: colonnes manquantes, migration Phase 2 a appliquer",
			"xuid", xuid)
		return 0, nil
	}

	matches, err := loadMatchesForEngagement(sharedDB, xuid)
	if err != nil {
		return 0, fmt.Errorf("batchComputeEngagementScores: load matches: %w", err)
	}
	if len(matches) == 0 {
		return 0, nil
	}

	existing := loadExistingEngagementScores(playerDB)
	historyByMode := make(map[string][]domain.HistoricalEngagementBrut)
	updated := 0
	now := time.Now().UTC()

	for _, m := range matches {
		if !force && existing[m.MatchID] {
			continue
		}
		if m.IsPvE {
			// PvE non couvert v1 (cf doc reflexion §3.4 perimetre)
			continue
		}

		modeCategory := normalizeModeCategoryFromFlags(m.IsRanked)

		events, err := loadEventsForMatch(sharedDB, m.MatchID)
		if err != nil {
			slog.DebugContext(ctx, "engagement: events load failed",
				"match_id", m.MatchID, "err", err)
			continue
		}
		if len(events) == 0 {
			continue
		}

		teamXUIDs, err := loadTeamXUIDs(sharedDB, m.MatchID, m.TargetTeamID, xuid)
		if err != nil {
			slog.DebugContext(ctx, "engagement: team xuids load failed",
				"match_id", m.MatchID, "err", err)
			continue
		}

		playerEvents, teamEvents, lobbyEvents := partitionMatchEvents(events, xuid, teamXUIDs)

		history, ok := historyByMode[modeCategory]
		if !ok {
			history = loadHistoryForCategory(playerDB, modeCategory, m.MatchID)
			historyByMode[modeCategory] = history
		}

		// highlight_events.time_ms est relatif au debut du match (0 a durationMS),
		// pas un epoch UTC. On normalise donc les bornes a [0, duration] pour
		// rester dans le meme repere que les events.
		durationMS := m.EndTimeMS - m.StartTimeMS
		input := temporal.EngagementScoreInput{
			PlayerEvents:   playerEvents,
			TeamEvents:     teamEvents,
			LobbyEvents:    lobbyEvents,
			NTeam:          m.NTeam,
			NHumansLobby:   m.NHumansLobby,
			XUID:           xuid,
			MatchStartMS:   0,
			MatchEndMS:     durationMS,
			History:        history,
			CoefTeamShare:  1.0, // cold start neutre
			CoefLobbyShare: 1.0,
			PersonalScore:  m.PersonalScore,
			Kills:          m.Kills,
			Assists:        m.Assists,
			Mode:           modeCategory,
			IsTeamMode:     m.IsTeamMode,
		}

		result, err := temporal.ComputeEngagementScore(input)
		if err != nil {
			slog.DebugContext(ctx, "engagement: compute skip",
				"match_id", m.MatchID, "err", err)
			continue
		}

		if err := persistEngagementScore(playerDB, m.MatchID, modeCategory, result, now); err != nil {
			slog.ErrorContext(ctx, "engagement: persist failed",
				"match_id", m.MatchID, "err", err)
			observability.IncCounter("engagement_persist_error_total")
			continue
		}
		observability.IncCounter("engagement_score_computed_total")

		// Persist match_intensity dans shared.match_registry (best-effort).
		if result.MatchIntensity > 0 {
			_ = persistMatchIntensity(sharedDB, m.MatchID, result.MatchIntensity)
		}

		// Met a jour l'historique en memoire pour les matchs suivants
		// (preserve la coherence : les matchs futurs voient le residu courant).
		historyByMode[modeCategory] = append(history, domain.HistoricalEngagementBrut{
			MatchID: m.MatchID,
			Brut:    result.ResidualBrut,
		})
		updated++
	}

	return updated, nil
}

// engagementColumnsAvailable verifie la presence de la colonne engagement_score
// sur player_match_enrichment.
func engagementColumnsAvailable(playerDB *sql.DB) bool {
	var count int
	err := playerDB.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'player_match_enrichment'
		  AND column_name = 'engagement_score'
	`).Scan(&count)
	return err == nil && count > 0
}

// loadMatchesForEngagement charge les matchs PvP du joueur avec metadata.
func loadMatchesForEngagement(sharedDB *sql.DB, xuid string) ([]engagementMatchRow, error) {
	const q = `
		SELECT
			mr.match_id,
			COALESCE(EPOCH_MS(mr.start_time_utc), EPOCH_MS(mr.start_time)),
			COALESCE(EPOCH_MS(mr.end_time_utc), EPOCH_MS(mr.end_time)),
			COALESCE(mr.is_ranked, FALSE),
			COALESCE(mr.is_firefight, FALSE),
			COALESCE(mp.team_id, 0),
			COALESCE(mp.personal_score, 0),
			COALESCE(mp.kills, 0),
			COALESCE(mp.assists, 0)
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND mr.start_time IS NOT NULL
		  AND mr.end_time IS NOT NULL
		ORDER BY mr.start_time ASC
	`
	rows, err := sharedDB.Query(q, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []engagementMatchRow
	for rows.Next() {
		var m engagementMatchRow
		if err := rows.Scan(
			&m.MatchID,
			&m.StartTimeMS,
			&m.EndTimeMS,
			&m.IsRanked,
			&m.IsPvE,
			&m.TargetTeamID,
			&m.PersonalScore,
			&m.Kills,
			&m.Assists,
		); err != nil {
			continue
		}
		out = append(out, m)
	}

	// Pour chaque match, charger NTeam et NHumansLobby (1 query par match — pas optimal
	// mais simple pour Phase 3 ; pourrait etre joint dans la query principale en Phase 3.b).
	for i := range out {
		nTeam, nLobby := loadTeamSizes(sharedDB, out[i].MatchID, out[i].TargetTeamID)
		out[i].NTeam = nTeam
		out[i].NHumansLobby = nLobby
		out[i].IsTeamMode = nTeam > 1 // FFA si NTeam=1
	}
	return out, rows.Err()
}

// loadTeamSizes compte les humains de l'equipe alliee et du lobby pour un match.
func loadTeamSizes(sharedDB *sql.DB, matchID string, teamID int) (nTeam, nLobby int) {
	const q = `
		SELECT
			SUM(CASE WHEN team_id = ? AND xuid NOT LIKE 'bid(%' THEN 1 ELSE 0 END),
			SUM(CASE WHEN xuid NOT LIKE 'bid(%' THEN 1 ELSE 0 END)
		FROM match_participants
		WHERE match_id = ?
	`
	var team, lobby sql.NullInt64
	_ = sharedDB.QueryRow(q, teamID, matchID).Scan(&team, &lobby)
	return int(team.Int64), int(lobby.Int64)
}

// loadEventsForMatch charge les events highlight_events pour un match.
func loadEventsForMatch(sharedDB *sql.DB, matchID string) ([]canonical.HighlightEvent, error) {
	const q = `
		SELECT match_id, event_type, COALESCE(time_ms, 0), COALESCE(xuid, '')
		FROM highlight_events
		WHERE match_id = ?
		ORDER BY time_ms ASC
	`
	rows, err := sharedDB.Query(q, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []canonical.HighlightEvent
	for rows.Next() {
		var ev canonical.HighlightEvent
		if err := rows.Scan(&ev.MatchID, &ev.EventType, &ev.TimeMS, &ev.XUID); err != nil {
			continue
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// loadTeamXUIDs charge les XUIDs des coequipiers humains (joueur cible exclu).
func loadTeamXUIDs(sharedDB *sql.DB, matchID string, teamID int, targetXUID string) (map[string]bool, error) {
	const q = `
		SELECT xuid FROM match_participants
		WHERE match_id = ?
		  AND team_id = ?
		  AND xuid NOT LIKE 'bid(%'
		  AND xuid <> ?
	`
	rows, err := sharedDB.Query(q, matchID, teamID, targetXUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teammates := make(map[string]bool)
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			continue
		}
		teammates[x] = true
	}
	return teammates, rows.Err()
}

// partitionMatchEvents separe les events en player / team / lobby selon le
// XUID acteur. PartitionMatchEvents est plus fin que partitionEvents (service)
// car il dispose des TeamXUIDs explicites.
func partitionMatchEvents(
	all []canonical.HighlightEvent,
	targetXUID string,
	teamXUIDs map[string]bool,
) (player, team, lobby []canonical.HighlightEvent) {
	player = make([]canonical.HighlightEvent, 0)
	team = make([]canonical.HighlightEvent, 0)
	lobby = all
	for _, e := range all {
		actor := eventActor(e)
		switch {
		case actor == targetXUID:
			player = append(player, e)
		case teamXUIDs[actor]:
			team = append(team, e)
		}
	}
	return player, team, lobby
}

// eventActor retourne le XUID acteur d'un event en utilisant le champ legacy
// XUID (la table shared.highlight_events n'a qu'une colonne xuid, pas de
// KillerXUID/VictimXUID/PlayerXUID).
func eventActor(e canonical.HighlightEvent) string {
	if e.XUID != "" {
		return e.XUID
	}
	return ""
}

// loadHistoryForCategory charge les residus historiques du joueur sur une
// categorie de mode, en excluant le match courant.
func loadHistoryForCategory(playerDB *sql.DB, modeCategory, excludeMatchID string) []domain.HistoricalEngagementBrut {
	const q = `
		SELECT match_id, engagement_score_brut
		FROM player_match_enrichment
		WHERE mode_category = ?
		  AND engagement_score_brut IS NOT NULL
		  AND match_id <> ?
		ORDER BY match_id DESC
		LIMIT 200
	`
	rows, err := playerDB.Query(q, modeCategory, excludeMatchID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []domain.HistoricalEngagementBrut
	for rows.Next() {
		var h domain.HistoricalEngagementBrut
		if err := rows.Scan(&h.MatchID, &h.Brut); err != nil {
			continue
		}
		out = append(out, h)
	}
	return out
}

// loadExistingEngagementScores retourne un set de match_id qui ont deja un
// score persiste. Permet de skip rapidement sans force.
func loadExistingEngagementScores(playerDB *sql.DB) map[string]bool {
	rows, err := playerDB.Query(`
		SELECT match_id FROM player_match_enrichment
		WHERE engagement_score IS NOT NULL
	`)
	if err != nil {
		return map[string]bool{}
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			out[id] = true
		}
	}
	return out
}

// persistEngagementScore UPSERT le score + paces dans player_match_enrichment.
//
// Si les colonnes engagement_pace_* sont absentes (migration recompute coefs
// non appliquee), retombe sur la version 5-col (sans paces) pour rester
// compatible avec les bases anterieures.
func persistEngagementScore(
	playerDB *sql.DB,
	matchID, modeCategory string,
	result domain.EngagementScoreResult,
	now time.Time,
) error {
	var scoreArg any
	if result.EngagementScore != nil {
		scoreArg = *result.EngagementScore
	}
	if pacesColumnsAvailable(playerDB) {
		_, err := playerDB.Exec(`
			INSERT INTO player_match_enrichment (
				match_id, engagement_score, engagement_score_brut,
				engagement_score_confidence, mode_category,
				engagement_pace_player, engagement_pace_team, engagement_pace_lobby,
				engagement_player_activity, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (match_id) DO UPDATE SET
				engagement_score = EXCLUDED.engagement_score,
				engagement_score_brut = EXCLUDED.engagement_score_brut,
				engagement_score_confidence = EXCLUDED.engagement_score_confidence,
				mode_category = EXCLUDED.mode_category,
				engagement_pace_player = EXCLUDED.engagement_pace_player,
				engagement_pace_team = EXCLUDED.engagement_pace_team,
				engagement_pace_lobby = EXCLUDED.engagement_pace_lobby,
				engagement_player_activity = EXCLUDED.engagement_player_activity,
				updated_at = EXCLUDED.updated_at
		`, matchID, scoreArg, result.ResidualBrut, result.Confidence, modeCategory,
			result.MeanPaceJoueur, result.MeanPaceTeam, result.MeanPaceLobby,
			result.PlayerActivity, now)
		return err
	}
	// Fallback : pre-migration recompute coefs (5 colonnes).
	_, err := playerDB.Exec(`
		INSERT INTO player_match_enrichment (
			match_id, engagement_score, engagement_score_brut,
			engagement_score_confidence, mode_category, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (match_id) DO UPDATE SET
			engagement_score = EXCLUDED.engagement_score,
			engagement_score_brut = EXCLUDED.engagement_score_brut,
			engagement_score_confidence = EXCLUDED.engagement_score_confidence,
			mode_category = EXCLUDED.mode_category,
			updated_at = EXCLUDED.updated_at
	`, matchID, scoreArg, result.ResidualBrut, result.Confidence, modeCategory, now)
	return err
}

// pacesColumnsAvailable verifie la presence de la colonne engagement_pace_team
// (et donc du jeu complet de 4 colonnes paces ajoutees ensemble par migration).
func pacesColumnsAvailable(playerDB *sql.DB) bool {
	var count int
	err := playerDB.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'player_match_enrichment'
		  AND column_name = 'engagement_pace_team'
	`).Scan(&count)
	return err == nil && count > 0
}

// persistMatchIntensity met a jour shared.match_registry.match_intensity.
func persistMatchIntensity(sharedDB *sql.DB, matchID string, intensity float64) error {
	_, err := sharedDB.Exec(`
		UPDATE match_registry SET match_intensity = ?
		WHERE match_id = ?
	`, intensity, matchID)
	return err
}

// normalizeModeCategoryFromFlags retourne la categorie de mode normalisee
// depuis les flags is_ranked / is_pve. PvE est filtre en amont (cf v1 perimetre).
func normalizeModeCategoryFromFlags(isRanked bool) string {
	if isRanked {
		return "PvP_ranked"
	}
	return "PvP_unranked"
}

// (Le recompute des coefficients vit dans engagement_recompute.go pour
// respecter la limite 500L par fichier — cf. arch-rules § Modularité.)
