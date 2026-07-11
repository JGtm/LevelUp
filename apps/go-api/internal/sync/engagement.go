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

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
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
) (int, []matchIntensityUpdate, error) {
	if !engagementColumnsAvailable(ctx, playerDB) {
		slog.WarnContext(ctx, "engagement: colonnes manquantes, migration Phase 2 a appliquer",
			"xuid", xuid)
		return 0, nil, nil
	}

	matches, err := loadMatchesForEngagement(ctx, sharedDB, xuid)
	if err != nil {
		return 0, nil, fmt.Errorf("batchComputeEngagementScores: load matches: %w", err)
	}
	if len(matches) == 0 {
		return 0, nil, nil
	}

	existing := loadExistingEngagementScores(ctx, playerDB)
	historyByMode := make(map[string][]domain.HistoricalEngagementBrut)
	// Entrees de l'attendu ancre lobby par mode (coef lobby global + bins),
	// chargees une fois par mode. Le compute persiste le residu (history du
	// percentile) : il DOIT utiliser le meme modele que le serving live, sinon
	// l'historique melange deux univers. Cold-start au 1er passe (tables vides),
	// resolu en universe bin apres recompute + 2e passe (cf. plan Phase 4).
	expectedByMode := make(map[string]expectedInputs)
	updated := 0
	var intensities []matchIntensityUpdate
	now := time.Now().UTC()

	// Phase 3 du refactor ART : seul le batch path est conservé (le legacy
	// row-by-row déclenchait le bug ART). Accumulation updates en RAM +
	// flush via PostSyncEnrichmentPersister (1 single UPDATE multi-row
	// multi-col).
	hasPaces := pacesColumnsAvailable(ctx, playerDB)
	var pendingUpdates []persist.EnrichmentMultiColumnUpdate

	for _, m := range matches {
		if !force && existing[m.MatchID] {
			continue
		}
		if m.IsPvE {
			// PvE non couvert v1 (cf doc reflexion §3.4 perimetre)
			continue
		}

		modeCategory := normalizeModeCategoryFromFlags(m.IsRanked)

		events, err := loadEventsForMatch(ctx, sharedDB, m.MatchID)
		if err != nil {
			slog.DebugContext(ctx, "engagement: events load failed",
				"match_id", m.MatchID, "err", err)
			continue
		}
		if len(events) == 0 {
			continue
		}

		teamXUIDs, err := loadTeamXUIDs(ctx, sharedDB, m.MatchID, m.TargetTeamID, xuid)
		if err != nil {
			slog.DebugContext(ctx, "engagement: team xuids load failed",
				"match_id", m.MatchID, "err", err)
			continue
		}

		playerEvents, teamEvents, lobbyEvents := partitionMatchEvents(events, xuid, teamXUIDs)

		history, ok := historyByMode[modeCategory]
		if !ok {
			h, herr := loadHistoryForCategory(ctx, playerDB, modeCategory, m.MatchID)
			if herr != nil {
				// Lot B (audit #6) : history indisponible par erreur DB → skip ce
				// match (score reste NULL, re-tentable via force) au lieu de
				// persister un engagement_score dérivé d'une baseline vide erronée.
				slog.ErrorContext(ctx, "engagement: skip match — history indisponible (pas de score faux)", "match_id", m.MatchID, "err", herr)
				observability.IncCounterT(ctxkeys.TitleSlug(ctx), "engagement_history_load_error_total")
				continue
			}
			history = h
			historyByMode[modeCategory] = history
		}

		// highlight_events.time_ms est relatif au debut du match (0 a durationMS),
		// pas un epoch UTC. On normalise donc les bornes a [0, duration] pour
		// rester dans le meme repere que les events.
		exp, ok := expectedByMode[modeCategory]
		if !ok {
			exp = loadExpectedInputsForMode(ctx, playerDB, xuid, modeCategory)
			expectedByMode[modeCategory] = exp
		}

		durationMS := m.EndTimeMS - m.StartTimeMS
		input := temporal.EngagementScoreInput{
			PlayerEvents:       playerEvents,
			TeamEvents:         teamEvents,
			LobbyEvents:        lobbyEvents,
			NTeam:              m.NTeam,
			NHumansLobby:       m.NHumansLobby,
			XUID:               xuid,
			MatchStartMS:       0,
			MatchEndMS:         durationMS,
			History:            history,
			CoefLobbyShare:     exp.coefLobby,
			HasGlobalLobbyCoef: exp.hasGlobal,
			ResponseBins:       exp.bins,
			PersonalScore:      m.PersonalScore,
			Kills:              m.Kills,
			Assists:            m.Assists,
			Mode:               modeCategory,
			IsTeamMode:         m.IsTeamMode,
		}

		result, err := temporal.ComputeEngagementScore(input)
		if err != nil {
			slog.DebugContext(ctx, "engagement: compute skip",
				"match_id", m.MatchID, "err", err)
			continue
		}

		pendingUpdates = append(pendingUpdates,
			buildEngagementUpdate(m.MatchID, modeCategory, result, now, hasPaces))
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "engagement_score_computed_total")

		// match_intensity (shared.match_registry) : accumulé et flushé par le
		// CALLER (burst court étape 1 contention — plus d'écriture shared inline
		// pendant le compute ; le sharedDB reçu peut être un lecteur RO).
		if result.MatchIntensity > 0 {
			intensities = append(intensities, matchIntensityUpdate{
				matchID: m.MatchID, intensity: result.MatchIntensity,
			})
		}

		// Met a jour l'historique en memoire pour les matchs suivants
		// (preserve la coherence : les matchs futurs voient le residu courant).
		historyByMode[modeCategory] = append(history, domain.HistoricalEngagementBrut{
			MatchID: m.MatchID,
			Brut:    result.ResidualBrut,
		})
		updated++
	}

	// Flush des updates accumulés en 1 single UPDATE multi-row.
	if len(pendingUpdates) > 0 {
		p := persist.NewPostSyncEnrichmentPersister(playerDB)
		if _, err := p.BatchUpdateMulti(ctx, pendingUpdates); err != nil {
			slog.ErrorContext(ctx, "engagement: BatchUpdateMulti échoué",
				"batch_size", len(pendingUpdates), "err", err)
			observability.IncCounterT(ctxkeys.TitleSlug(ctx), "engagement_persist_error_total")
			return 0, nil, fmt.Errorf("engagement batch flush: %w", err)
		}
	}

	return updated, intensities, nil
}

// buildEngagementUpdate construit une EnrichmentMultiColumnUpdate pour 1 match.
// hasPaces=true => 8 colonnes (full migration), false => 4 colonnes (fallback
// pre-migration paces). Le set de colonnes est homogène sur tous les matchs
// d'un même cycle (toutes les rows ont le même schema).
func buildEngagementUpdate(matchID, modeCategory string, result domain.EngagementScoreResult, now time.Time, hasPaces bool) persist.EnrichmentMultiColumnUpdate {
	var scoreArg any
	if result.EngagementScore != nil {
		scoreArg = *result.EngagementScore
	}
	fields := map[string]any{
		"engagement_score":            scoreArg,
		"engagement_score_brut":       result.ResidualBrut,
		"engagement_score_confidence": result.Confidence,
		"mode_category":               modeCategory,
	}
	if hasPaces {
		fields["engagement_pace_player"] = result.MeanPaceJoueur
		fields["engagement_pace_team"] = result.MeanPaceTeam
		fields["engagement_pace_lobby"] = result.MeanPaceLobby
		fields["engagement_player_activity"] = result.PlayerActivity
	}
	_ = now // updated_at géré par le persister
	return persist.EnrichmentMultiColumnUpdate{
		MatchID: matchID,
		Fields:  fields,
	}
}

// expectedInputs regroupe les entrees de l'attendu ancre lobby pour un mode :
// coef lobby global (fallback), flag de disponibilite, et bins de reponse.
type expectedInputs struct {
	coefLobby float64
	hasGlobal bool
	bins      *domain.EngagementResponseBins
}

// loadExpectedInputsForMode charge le coef lobby global + les bins de reponse
// pour (xuid, mode) depuis la player DB. Best-effort : tables absentes ou pas de
// row → cold-start (coef 1.0, hasGlobal=false, bins nil). hasGlobal=true implique
// >= MinMatchesForCoef samples (le recompute ne persiste la row qu'a ce seuil).
//
// Miroir de service.loadExpectedInputs (chemin serving), pour que le residu
// persiste par le compute soit dans le meme univers que le residu servi live.
func loadExpectedInputsForMode(ctx context.Context, playerDB *sql.DB, xuid, modeCategory string) expectedInputs {
	out := expectedInputs{coefLobby: 1.0}
	if coefficientsTableAvailable(ctx, playerDB) {
		var coefLobby float64
		err := playerDB.QueryRowContext(ctx,
			`SELECT coef_lobby_share FROM engagement_coefficients WHERE xuid = ? AND mode_category = ?`,
			xuid, modeCategory).Scan(&coefLobby)
		if err == nil {
			out.coefLobby = coefLobby
			out.hasGlobal = true
		}
	}
	if responseBinsTableAvailable(ctx, playerDB) {
		out.bins = loadResponseBinsRows(ctx, playerDB, xuid, modeCategory)
	}
	return out
}

// loadResponseBinsRows lit les bins persistes pour (xuid, mode). nil si aucun.
func loadResponseBinsRows(ctx context.Context, playerDB *sql.DB, xuid, modeCategory string) *domain.EngagementResponseBins {
	rows, err := playerDB.QueryContext(ctx,
		`SELECT intensity_bin, lower_bound, upper_bound, coef_lobby, n_matches
		 FROM engagement_response_bins WHERE xuid = ? AND mode_category = ? ORDER BY lower_bound`,
		xuid, modeCategory)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var bins []domain.EngagementIntensityBin
	for rows.Next() {
		var b domain.EngagementIntensityBin
		if scanErr := rows.Scan(&b.Bin, &b.LowerBound, &b.UpperBound, &b.CoefLobby, &b.NMatches); scanErr == nil {
			bins = append(bins, b)
		}
	}
	if err := rows.Err(); err != nil || len(bins) == 0 {
		return nil
	}
	return &domain.EngagementResponseBins{XUID: xuid, ModeCategory: modeCategory, Bins: bins}
}

// engagementColumnsAvailable verifie la presence de la colonne engagement_score
// sur player_match_enrichment.
func engagementColumnsAvailable(ctx context.Context, playerDB *sql.DB) bool {
	var count int
	err := playerDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'player_match_enrichment'
		  AND column_name = 'engagement_score'
	`).Scan(&count)
	return err == nil && count > 0
}

// loadMatchesForEngagement charge les matchs PvP du joueur avec metadata.
//
// Filtre les matchs où le bit MBitEvents est set ET highlight_events est
// vide — c'est l'indicateur "events fetch tenté, aucune donnée disponible"
// (film 404 CDN ou match jamais joué assez longtemps). Pour ces matchs,
// engagement_score reste NULL definitivement — pas la peine de re-tenter
// à chaque sync. Les matchs où MBitEvents n'est pas set restent inclus :
// events_heal peut encore les charger.
//
// Note : MarkEventsLoaded() synchronise events_loaded (boolean legacy) et
// le bit MBitEvents (bitmask) ; on utilise le bit pour cohérence avec le
// reste du projet (skill, weapon_kills, pve, etc. utilisent tous des bits).
func loadMatchesForEngagement(ctx context.Context, sharedDB *sql.DB, xuid string) ([]engagementMatchRow, error) {
	q := fmt.Sprintf(`
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
		  AND (
		    (COALESCE(mr.backfill_completed, 0) & %d) = 0
		    OR EXISTS (SELECT 1 FROM highlight_events he WHERE he.match_id = mr.match_id)
		  )
		ORDER BY mr.start_time ASC
	`, MBitEvents)
	rows, err := sharedDB.QueryContext(ctx, q, xuid)
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
		nTeam, nLobby := loadTeamSizes(ctx, sharedDB, out[i].MatchID, out[i].TargetTeamID)
		out[i].NTeam = nTeam
		out[i].NHumansLobby = nLobby
		out[i].IsTeamMode = nTeam > 1 // FFA si NTeam=1
	}
	return out, rows.Err()
}

// loadTeamSizes compte les humains de l'equipe alliee et du lobby pour un match.
func loadTeamSizes(ctx context.Context, sharedDB *sql.DB, matchID string, teamID int) (nTeam, nLobby int) {
	var q = `
		SELECT
			SUM(CASE WHEN team_id = ? AND ` + analysis.SQLIsNotBotCol("xuid") + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN ` + analysis.SQLIsNotBotCol("xuid") + ` THEN 1 ELSE 0 END)
		FROM match_participants
		WHERE match_id = ?
	`
	var team, lobby sql.NullInt64
	_ = sharedDB.QueryRowContext(ctx, q, teamID, matchID).Scan(&team, &lobby)
	return int(team.Int64), int(lobby.Int64)
}

// loadEventsForMatch charge les events highlight_events pour un match.
//
// Title-agnostic : selon le titre, highlight_events ne porte PAS forcément les
// kills. Halo Infinite y stocke kill/death/medal/mode ; Halo 5 n'y stocke QUE
// des médailles, les kills horodatés vivant dans killer_victim_pairs. Le code
// aval (engagement_score.go) checke aussi EventAssist, EventFinisher, etc. —
// branches inertes si le titre ne les produit pas (jamais déclenchées).
//
// Quand highlight_events ne contient aucun kill/death, on synthétise les events
// kill/death depuis killer_victim_pairs (kills horodatés, time_ms relatif au
// match) et on les fusionne aux médailles existantes, triés par TimeMS — sinon
// la courbe d'engagement mesurerait la cadence des médailles, pas des kills, et
// PlayerActivity sous-compterait les morts (0 death). killer_victim_pairs est
// dérivé du film comme highlight_events, mais les DEUX peuvent coexister avec
// des contenus disjoints (médailles d'un côté, kills de l'autre).
//
// Si les deux tables sont vides (ex. film 404 historique), retourne nil sans
// erreur ; l'engagement_score restera NULL pour ce match.
func loadEventsForMatch(ctx context.Context, sharedDB *sql.DB, matchID string) ([]canonical.HighlightEvent, error) {
	const q = `
		SELECT match_id, event_type, COALESCE(time_ms, 0), COALESCE(xuid, '')
		FROM highlight_events
		WHERE match_id = ?
		ORDER BY time_ms ASC
	`
	rows, err := sharedDB.QueryContext(ctx, q, matchID)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !analysis.HasCanonicalKillOrDeath(out) {
		synth, serr := loadSyntheticKillEventsForMatch(ctx, sharedDB, matchID)
		if serr == nil && len(synth) > 0 {
			out = analysis.MergeAndSortCanonicalEvents(out, synth)
		}
	}
	return out, nil
}

// loadSyntheticKillEventsForMatch charge killer_victim_pairs et synthétise des
// events canoniques kill/death via le helper partagé
// analysis.SynthesizeKillEventsFromKVPairs (source unique de la règle). Best-effort.
func loadSyntheticKillEventsForMatch(ctx context.Context, sharedDB *sql.DB, matchID string) ([]canonical.HighlightEvent, error) {
	const q = `
		SELECT COALESCE(killer_xuid, ''), COALESCE(victim_xuid, ''), COALESCE(time_ms, 0)
		FROM killer_victim_pairs
		WHERE match_id = ?
		ORDER BY time_ms ASC
	`
	rows, err := sharedDB.QueryContext(ctx, q, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	inputs := make([]analysis.KVSyntheticInput, 0)
	for rows.Next() {
		var in analysis.KVSyntheticInput
		if err := rows.Scan(&in.KillerXUID, &in.VictimXUID, &in.TimeMS); err != nil {
			continue
		}
		inputs = append(inputs, in)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return analysis.SynthesizeKillEventsFromKVPairs(inputs, matchID), nil
}

// loadTeamXUIDs charge les XUIDs des coequipiers humains (joueur cible exclu).
func loadTeamXUIDs(ctx context.Context, sharedDB *sql.DB, matchID string, teamID int, targetXUID string) (map[string]bool, error) {
	var q = `
		SELECT xuid FROM match_participants
		WHERE match_id = ?
		  AND team_id = ?
		  AND ` + analysis.SQLIsNotBotCol("xuid") + `
		  AND xuid <> ?
	`
	rows, err := sharedDB.QueryContext(ctx, q, matchID, teamID, targetXUID)
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
func loadHistoryForCategory(ctx context.Context, playerDB *sql.DB, modeCategory, excludeMatchID string) ([]domain.HistoricalEngagementBrut, error) {
	const q = `
		SELECT match_id, engagement_score_brut
		FROM player_match_enrichment_latest
		WHERE mode_category = ?
		  AND engagement_score_brut IS NOT NULL
		  AND match_id <> ?
		ORDER BY match_id DESC
		LIMIT 200
	`
	rows, err := playerDB.QueryContext(ctx, q, modeCategory, excludeMatchID)
	if err != nil {
		// Lot B (audit #6) : ne plus avaler l'erreur — une history vide PAR ERREUR
		// (et non par absence réelle) fausserait la baseline, donc le score persisté.
		slog.ErrorContext(ctx, "engagement: chargement history catégorie échoué", "mode_category", modeCategory, "err", err)
		return nil, err
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
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "engagement: itération history échouée", "mode_category", modeCategory, "err", err)
		return out, err
	}
	return out, nil
}

// loadExistingEngagementScores retourne le set des match_id dont l'engagement a
// déjà été TENTÉ (présence d'une row stage='engagement'), pour skip en non-force.
//
// Append-only #23046 — IDEMPOTENCE : on lit la présence du stage, PAS
// engagement_score IS NOT NULL. Un match insufficient_history (score NULL LÉGITIME
// et PERMANENT — les ~10 premiers matchs d'une catégorie n'auront jamais assez
// d'historique) serait sinon ré-INSÉRÉ à CHAQUE post-sync → croissance non bornée
// sur la table la plus écrite. Lecture physique (la vue n'expose pas `stage`) :
// marqueur d'idempotence writer-side, comme l'ancre stage='live' du persister.
// Le re-score à maturité de l'historique passe par le chemin force=true.
func loadExistingEngagementScores(ctx context.Context, playerDB *sql.DB) map[string]bool {
	rows, err := playerDB.QueryContext(ctx, `
		SELECT match_id FROM player_match_enrichment WHERE stage = 'engagement'
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

// pacesColumnsAvailable verifie la presence de la colonne engagement_pace_team
// (et donc du jeu complet de 4 colonnes paces ajoutees ensemble par migration).
func pacesColumnsAvailable(ctx context.Context, playerDB *sql.DB) bool {
	var count int
	err := playerDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'player_match_enrichment'
		  AND column_name = 'engagement_pace_team'
	`).Scan(&count)
	return err == nil && count > 0
}

// matchIntensityUpdate : écriture shared différée (match_registry.match_intensity),
// accumulée pendant le compute engagement et flushée par le caller en burst court
// (étape 1 contention — le compute ne tient plus le writer shared).
type matchIntensityUpdate struct {
	matchID   string
	intensity float64
}

// persistMatchIntensity met a jour shared.match_registry.match_intensity.
func persistMatchIntensity(ctx context.Context, sharedDB *sql.DB, matchID string, intensity float64) error {
	_, err := sharedDB.ExecContext(ctx, `
		UPDATE match_registry SET match_intensity = ?
		WHERE match_id = ?
	`, intensity, matchID)
	return err
}

// persistMatchIntensities flushe un lot d'intensités accumulées (best-effort,
// même sémantique que l'ancien write inline : les erreurs sont ignorées une à
// une). SQL inchangé — seul le MOMENT de l'écriture change (burst caller).
func persistMatchIntensities(ctx context.Context, sharedDB *sql.DB, ups []matchIntensityUpdate) {
	for _, u := range ups {
		_ = persistMatchIntensity(ctx, sharedDB, u.matchID, u.intensity)
	}
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
