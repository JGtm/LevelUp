// Package duckdb — match_view_repo_scoreboard.go : scoreboard + objective
// score de la vue Match. Découpé de match_view_repo.go (god-file split,
// refactor 2026-05-27).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/timing"
)

// GetMatchScoreboard retourne les stats de tous les joueurs (Q12).
// Exécutée sur SharedReader (ADR 0016) — Q12 lit medals_earned + weapon_kills
// + match_participants (shared-only) ; les noms viennent de l'annuaire du match en portée base
// (lot A du plan perf « lectures par périmètre », plus de jointure sur la vue des noms).
//
// Les stats d'objectifs sont chargées SÉPARÉMENT (Q12bObjectiveStats) et
// best-effort : leur panne (vue match_objective_stats_latest non migrée...) ne doit JAMAIS
// faire échouer le scoreboard (G1, incident prod 25/07) mais n'est plus TUE : les lignes sont
// rendues avec une *domain.SectionDegradedError (le service dit `section_unavailable`).
func (r *MatchViewRepo) GetMatchScoreboard(ctx context.Context, matchID string) ([]domain.ScoreboardRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard: shared reader: %w", err)
	}
	defer release()

	stop := timing.FromContext(ctx).Section("match_scoreboard")
	results, err := r.lireTableauDeScore(ctx, sharedDB, matchID)
	stop()
	if err != nil {
		return nil, err
	}
	stop = timing.FromContext(ctx).Section("match_scoreboard_annuaire")
	err = nommerLignesPorteeBase(ctx, sharedDB, []string{matchID}, results, accesLigne[domain.ScoreboardRaw]{
		xuid:   func(s domain.ScoreboardRaw) string { return s.XUID },
		match:  func(domain.ScoreboardRaw) string { return matchID },
		nommer: func(s *domain.ScoreboardRaw, gt string) { s.Gamertag = gt },
	})
	stop()
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard: %w", err)
	}

	// Objectifs : section DÉGRADABLE indépendamment (le tableau reste servi) — mais sa panne est
	// DITE : l'erreur de section est renvoyée avec les lignes (domain.SectionDegradedError).
	var degraded []error
	objByXUID, objErr := loadMatchObjectiveStats(ctx, sharedDB, matchID)
	if objErr != nil {
		degraded = append(degraded, objErr)
	}
	for i := range results {
		if obj, ok := objByXUID[results[i].XUID]; ok {
			results[i].Obj = obj
		}
	}
	// Statistiques d'ASSAUT : SECONDE source, dégradable à part elle aussi, et gatée par la
	// capability `film.bomb_stats` (câblée au wiring — jamais un slug). Un titre qui ne la
	// déclare pas ne paie même pas la requête. Elles s'AJOUTENT au bloc d'objectif du joueur :
	// les deux jeux de colonnes sont disjoints, et un match d'Assaut n'a de toute façon aucun
	// bloc API à écraser.
	if r.bombStats {
		bombByXUID, bombErr := loadMatchBombStats(ctx, sharedDB, matchID)
		if bombErr != nil {
			degraded = append(degraded, bombErr)
		}
		for i := range results {
			if b, ok := bombByXUID[results[i].XUID]; ok {
				fusionnerStatsBombe(&results[i].Obj, b)
			}
		}
	}

	r.attachTopWeapons(ctx, matchID, results)
	r.attachTopWeaponLabels(ctx, results)
	if len(degraded) > 0 {
		return results, &domain.SectionDegradedError{Section: "scoreboard_objectives", Err: errors.Join(degraded...)}
	}
	return results, nil
}

// lireTableauDeScore lit Q12 (lignes sans nom).
func (r *MatchViewRepo) lireTableauDeScore(ctx context.Context, db *sql.DB, matchID string) ([]domain.ScoreboardRaw, error) {
	// Q12 utilise 2 fois match_id : CTE des medailles, puis WHERE. Set perfect-kill
	// resolu pour le titre du joueur (HINF byte-identique).
	q := resolvePerfectKillClause(Q12MatchScoreboard, "medal_name_id", pdbTitleSlug(r.pdb))
	rows, err := db.QueryContext(ctx, q, matchID, matchID)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard: %w", err)
	}
	defer rows.Close()

	var results []domain.ScoreboardRaw
	for rows.Next() {
		s, scanErr := scanScoreboardRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard scan: %w", scanErr)
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// scanScoreboardRow scanne une ligne de Q12 et sanitize les flottants.
// Extrait de GetMatchScoreboard (seuil 80 lignes/fonction).
func scanScoreboardRow(rows *sql.Rows) (domain.ScoreboardRaw, error) {
	var s domain.ScoreboardRaw
	// top_weapon_id est UBIGINT côté DuckDB → scanner en *uint64 pour
	// éviter l'overflow int64 sur les hash de filmshell (bit63=1).
	var topWeaponU *uint64
	// Participation : nullable au DDL (matchs d'avant les colonnes) — sql.Null*, jamais
	// des types nus (la leçon Q20 du 2026-09-02 : une seule ligne NULL casse tout le scan).
	var joined, left sql.NullBool
	var firstJoined, lastLeave sql.NullTime
	if err := rows.Scan(
		&s.XUID,
		&s.IsBot,
		&s.TeamID,
		&s.RankInTeam,
		&s.OutcomeCode,
		&s.PersonalScore,
		&s.Kills,
		&s.Deaths,
		&s.Assists,
		&s.KDA,
		&s.Accuracy,
		&s.TimePlayed,
		&s.TeamMMR,
		&s.EnemyMMR,
		&s.ShotsFired,
		&s.ShotsHit,
		&s.DamageDealt,
		&s.DamageTaken,
		&s.AvgLifeSeconds,
		&s.HeadshotKills,
		&s.MaxKillingSpree,
		&s.GrenadeKills,
		&s.MeleeKills,
		&s.PowerWeaponKills,
		&s.AssassinationKills,
		&s.GroundPoundKills,
		&s.ShoulderBashKills,
		&s.PerfectKills,
		&topWeaponU,
		&s.KillsExpected,
		&s.DeathsExpected,
		&s.KillsStdDev,
		&s.DeathsStdDev,
		&joined,
		&left,
		&firstJoined,
		&lastLeave,
	); err != nil {
		return domain.ScoreboardRaw{}, err
	}
	applyParticipation(&s, joined, left, firstJoined, lastLeave)
	if topWeaponU != nil {
		v := int64(*topWeaponU) //nolint:gosec
		s.TopWeaponID = &v
	}
	// Sanitize : NaN/Inf venant de la DB (ex. 0/0 dans l'API Halo) cause un
	// échec silencieux de json.Encode → corps HTTP vide. On remplace par nil.
	s.PersonalScore = sanitizeF64(s.PersonalScore)
	s.KDA = sanitizeF64(s.KDA)
	s.Accuracy = sanitizeF64(s.Accuracy)
	s.TimePlayed = sanitizeF64(s.TimePlayed)
	s.TeamMMR = sanitizeF64(s.TeamMMR)
	s.EnemyMMR = sanitizeF64(s.EnemyMMR)
	s.DamageDealt = sanitizeF64(s.DamageDealt)
	s.DamageTaken = sanitizeF64(s.DamageTaken)
	s.AvgLifeSeconds = sanitizeF64(s.AvgLifeSeconds)
	s.KillsExpected = sanitizeF64(s.KillsExpected)
	s.DeathsExpected = sanitizeF64(s.DeathsExpected)
	s.KillsStdDev = sanitizeF64(s.KillsStdDev)
	s.DeathsStdDev = sanitizeF64(s.DeathsStdDev)
	return s, nil
}

// applyParticipation pose les champs de participation (nullable) sur la ligne scannée.
// Les instants sortent en UTC — la base est saine depuis la correction TZ du 2026-05-29,
// et normaliser ici garantit qu'un RFC3339 servi au client ne portera jamais un fuseau.
func applyParticipation(s *domain.ScoreboardRaw, joined, left sql.NullBool,
	firstJoined, lastLeave sql.NullTime) {
	if joined.Valid {
		s.JoinedInProgress = &joined.Bool
	}
	if left.Valid {
		s.LeftInProgress = &left.Bool
	}
	if firstJoined.Valid {
		t := firstJoined.Time.UTC()
		s.FirstJoinedTime = &t
	}
	if lastLeave.Valid {
		t := lastLeave.Time.UTC()
		s.LastLeaveTime = &t
	}
}

// attachTopWeaponLabels résout les libellés d'armes top-weapon du scoreboard.
func (r *MatchViewRepo) attachTopWeaponLabels(ctx context.Context, results []domain.ScoreboardRaw) {
	var topWeaponIDs []int64
	for _, s := range results {
		if s.TopWeaponID != nil {
			topWeaponIDs = append(topWeaponIDs, *s.TopWeaponID)
		}
	}
	if len(topWeaponIDs) == 0 {
		return
	}
	labels := r.lookupWeaponLabels(ctx, topWeaponIDs)
	for i := range results {
		if results[i].TopWeaponID != nil {
			results[i].TopWeaponLabel = labels[*results[i].TopWeaponID]
		}
	}
}

// loadMatchObjectiveStats charge les stats d'objectifs d'un match par xuid
// (Q12bObjectiveStats). La section est DÉGRADABLE (le tableau des scores reste servi) mais sa
// panne est DITE : la map rendue porte les lignes lues, et l'erreur non nil signale une
// requête en échec (vue absente d'une DB non migrée comprise), une ligne illisible ou une
// itération interrompue. Chaque cas est journalisé en ERREUR avant d'être renvoyé.
func loadMatchObjectiveStats(ctx context.Context, db *sql.DB, matchID string) (map[string]domain.ObjectiveRaw, error) {
	out := map[string]domain.ObjectiveRaw{}
	rows, err := db.QueryContext(ctx, Q12bObjectiveStats, matchID)
	if err != nil {
		slog.Log(ctx, observability.LevelOnCancellation(err, slog.LevelError), "match scoreboard: stats objectifs indisponibles, section dégradée",
			"match_id", matchID, "query", "Q12bObjectiveStats", "err", err)
		return out, fmt.Errorf("stats objectifs Q12b: %w", err)
	}
	defer rows.Close()
	var failures []error
	for rows.Next() {
		xuid, obj, scanErr := scanObjectiveRow(rows)
		if scanErr != nil {
			slog.ErrorContext(ctx, "match scoreboard: scan stats objectifs échoué, ligne ignorée",
				"match_id", matchID, "err", scanErr)
			failures = append(failures, fmt.Errorf("scan stats objectifs: %w", scanErr))
			continue
		}
		out[xuid] = obj
	}
	if err := rows.Err(); err != nil {
		slog.Log(ctx, observability.LevelOnCancellation(err, slog.LevelError), "match scoreboard: itération stats objectifs interrompue, section dégradée",
			"match_id", matchID, "err", err)
		failures = append(failures, fmt.Errorf("itération stats objectifs: %w", err))
	}
	return out, errors.Join(failures...)
}

// scanObjectiveRow scanne une ligne de Q12bObjectiveStats (xuid + 41 compteurs).
func scanObjectiveRow(rows *sql.Rows) (string, domain.ObjectiveRaw, error) {
	var (
		xuid string
		o    domain.ObjectiveRaw
	)
	if err := rows.Scan(
		&xuid,
		&o.FlagCaptures,
		&o.FlagCaptureAssists,
		&o.FlagGrabs,
		&o.FlagSecures,
		&o.FlagSteals,
		&o.FlagReturns,
		&o.FlagCarriersKilled,
		&o.FlagReturnersKilled,
		&o.KillsAsFlagCarrier,
		&o.KillsAsFlagReturner,
		&o.TimeAsFlagCarrierSeconds,
		&o.ZoneCaptures,
		&o.ZoneSecures,
		&o.ZoneOffensiveKills,
		&o.ZoneDefensiveKills,
		&o.ZoneScoringTicks,
		&o.TimeInZonesSeconds,
		&o.KillsAsSkullCarrier,
		&o.SkullCarriersKilled,
		&o.SkullGrabs,
		&o.SkullScoringTicks,
		&o.TimeAsSkullCarrierSeconds,
		&o.LongestTimeAsSkullCarrierSeconds,
		&o.KillsAsPowerSeedCarrier,
		&o.PowerSeedCarriersKilled,
		&o.PowerSeedsDeposited,
		&o.PowerSeedsStolen,
		&o.TimeAsPowerSeedCarrierSeconds,
		&o.TimeAsPowerSeedDriverSeconds,
		&o.ExtractionConversionsCompleted,
		&o.ExtractionConversionsDenied,
		&o.ExtractionInitiationsCompleted,
		&o.ExtractionInitiationsDenied,
		&o.SuccessfulExtractions,
		&o.KillsAsVip,
		&o.VipKills,
		&o.VipAssists,
		&o.TimesSelectedAsVip,
		&o.MaxKillingSpreeAsVip,
		&o.TimeAsVipSeconds,
		&o.LongestTimeAsVipSeconds,
	); err != nil {
		return "", domain.ObjectiveRaw{}, err
	}
	return xuid, o, nil
}

// GetMatchObjectiveScore retourne la somme award_score des awards de
// catégorie 'objective' pour (xuid, matchID). Mirror du squad
// LoadObjectiveScores (cf. squad_v2_adapter.go) appliqué à un seul match.
//
// ABSENCE ≠ PANNE : la table `personal_score_awards` du joueur peut ne pas exister (titre ou
// joueur sans awards) ; l'absence est CONSTATÉE par une sonde du catalogue et vaut 0. Toute
// défaillance de la sonde ou de la lecture remonte en erreur.
func (r *MatchViewRepo) GetMatchObjectiveScore(ctx context.Context, xuid, matchID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var tableCount int
	tblRows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'personal_score_awards'
	`)
	if err != nil {
		return 0, fmt.Errorf("MatchViewRepo.GetMatchObjectiveScore: sonde personal_score_awards: %w", err)
	}
	scanErr := tblRows.Scan(&tableCount)
	_ = tblRows.Close()
	if scanErr != nil {
		return 0, fmt.Errorf("MatchViewRepo.GetMatchObjectiveScore: sonde personal_score_awards: %w", scanErr)
	}
	if tableCount == 0 {
		return 0, nil // absence constatée : pas d'awards pour ce joueur / ce titre
	}

	var total int
	totalRows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT COALESCE(SUM(award_score), 0)::INTEGER
		FROM personal_score_awards_latest
		WHERE award_category = 'objective'
		  AND xuid = ?
		  AND match_id = ?
	`, xuid, matchID)
	if err != nil {
		return 0, fmt.Errorf("MatchViewRepo.GetMatchObjectiveScore: %w", err)
	}
	defer totalRows.Close()
	if err := totalRows.Scan(&total); err != nil {
		return 0, fmt.Errorf("MatchViewRepo.GetMatchObjectiveScore scan: %w", err)
	}
	return total, nil
}

// loadMatchBombStats charge les STATISTIQUES D'ASSAUT d'un match par xuid.
//
// SECONDE REQUÊTE, SECONDE TABLE, ET C'EST UNE DÉCISION DE SCHÉMA. Elles ne vivent pas dans
// `match_objective_stats` mais dans `match_bomb_stats` (décision 1 du plan d'Assaut) : la vue
// `match_objective_stats_latest` ne garde qu'UNE ligne par (match_id, xuid), et deux
// producteurs — le sync API et le film — s'y écraseraient l'un l'autre. Un re-sync API
// effacerait les stats de bombe de la vue.
//
// MÊME CONTRAT QUE `loadMatchObjectiveStats` (lignes lues + erreur de section, journalisée en
// ERREUR), à UNE distinction explicite près : la vue `match_bomb_stats_latest` ABSENTE du
// lecteur n'est pas une panne de requête. Le lecteur SNAPSHOT ne l'exporte pas (cf.
// ops/snapshot_export.go : sharedSnapshotTables / sharedSnapshotMatchKeyedRaw) : sans cette
// sonde, chaque vue d'un match d'un titre à statistiques d'Assaut servie du snapshot passerait
// pour en panne. L'absence est constatée (relationExists), journalisée en WARN, et rend une
// map vide sans erreur ; toute autre défaillance remonte.
func loadMatchBombStats(ctx context.Context, db *sql.DB, matchID string) (map[string]domain.ObjectiveRaw, error) {
	out := map[string]domain.ObjectiveRaw{}
	present, err := relationExists(ctx, db, "match_bomb_stats_latest")
	if err != nil {
		slog.Log(ctx, observability.LevelOnCancellation(err, slog.LevelError), "match scoreboard: sonde des stats d'Assaut en échec, colonnes absentes",
			"match_id", matchID, "err", err)
		return out, err
	}
	if !present {
		slog.WarnContext(ctx, "match scoreboard: stats d'Assaut absentes de ce lecteur (snapshot ?), colonnes absentes",
			"match_id", matchID, "relation", "match_bomb_stats_latest")
		return out, nil
	}
	rows, err := db.QueryContext(ctx, Q12cBombStats, matchID)
	if err != nil {
		slog.Log(ctx, observability.LevelOnCancellation(err, slog.LevelError), "match scoreboard: stats d'Assaut indisponibles, colonnes absentes",
			"match_id", matchID, "query", "Q12cBombStats", "err", err)
		return out, fmt.Errorf("stats d'Assaut Q12c: %w", err)
	}
	defer rows.Close()
	var failures []error
	for rows.Next() {
		var xuid string
		var o domain.ObjectiveRaw
		if err := rows.Scan(&xuid, &o.BombDetonations, &o.BombArms, &o.BombGrabs,
			&o.TimeAsBombCarrierSeconds, &o.BombCarriersKilled); err != nil {
			slog.ErrorContext(ctx, "match scoreboard: scan stats d'Assaut échoué, ligne ignorée",
				"match_id", matchID, "err", err)
			failures = append(failures, fmt.Errorf("scan stats d'Assaut: %w", err))
			continue
		}
		out[xuid] = o
	}
	if err := rows.Err(); err != nil {
		slog.Log(ctx, observability.LevelOnCancellation(err, slog.LevelError), "match scoreboard: itération stats d'Assaut interrompue, colonnes absentes",
			"match_id", matchID, "err", err)
		failures = append(failures, fmt.Errorf("itération stats d'Assaut: %w", err))
	}
	return out, errors.Join(failures...)
}

// fusionnerStatsBombe recopie les cinq mesures d'Assaut sur la ligne d'objectif du joueur. Elles
// s'AJOUTENT au bloc lu dans `match_objective_stats_latest` sans jamais l'écraser : les deux
// jeux de colonnes sont disjoints, et un match d'Assaut n'a de toute façon aucun bloc API.
func fusionnerStatsBombe(dst *domain.ObjectiveRaw, src domain.ObjectiveRaw) {
	dst.BombDetonations = src.BombDetonations
	dst.BombArms = src.BombArms
	dst.BombGrabs = src.BombGrabs
	dst.TimeAsBombCarrierSeconds = src.TimeAsBombCarrierSeconds
	dst.BombCarriersKilled = src.BombCarriersKilled
}
