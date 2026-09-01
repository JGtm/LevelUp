// Package duckdb — match_view_repo_scoreboard.go : scoreboard + objective
// score de la vue Match. Découpé de match_view_repo.go (god-file split,
// refactor 2026-05-27).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// GetMatchScoreboard retourne les stats de tous les joueurs (Q12).
// Exécutée sur SharedReader (ADR 0016) — Q12 lit medals_earned + weapon_kills
// + match_participants + v_gamertag_lookup (shared-only).
//
// Les stats d'objectifs sont chargées SÉPARÉMENT (Q12bObjectiveStats) et
// best-effort : leur absence (vue match_objective_stats_latest non migrée) ne
// doit JAMAIS faire échouer le scoreboard (G1, incident prod 25/07).
func (r *MatchViewRepo) GetMatchScoreboard(ctx context.Context, matchID string) ([]domain.ScoreboardRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard: shared reader: %w", err)
	}
	defer release()

	// Q12 utilise 2 fois match_id : CTE des medailles, puis WHERE. Set perfect-kill
	// resolu pour le titre du joueur (HINF byte-identique).
	q := resolvePerfectKillClause(Q12MatchScoreboard, "medal_name_id", pdbTitleSlug(r.pdb))
	rows, err := sharedDB.QueryContext(ctx, q, matchID, matchID)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Objectifs : section DÉGRADABLE indépendamment (map vide si la vue manque).
	objByXUID := loadMatchObjectiveStats(ctx, sharedDB, matchID)
	for i := range results {
		if obj, ok := objByXUID[results[i].XUID]; ok {
			results[i].Obj = obj
		}
	}

	r.attachTopWeapons(ctx, matchID, results)
	r.attachTopWeaponLabels(ctx, results)
	return results, nil
}

// scanScoreboardRow scanne une ligne de Q12 et sanitize les flottants.
// Extrait de GetMatchScoreboard (seuil 80 lignes/fonction).
func scanScoreboardRow(rows *sql.Rows) (domain.ScoreboardRaw, error) {
	var s domain.ScoreboardRaw
	// top_weapon_id est UBIGINT côté DuckDB → scanner en *uint64 pour
	// éviter l'overflow int64 sur les hash de filmshell (bit63=1).
	var topWeaponU *uint64
	if err := rows.Scan(
		&s.XUID,
		&s.Gamertag,
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
	); err != nil {
		return domain.ScoreboardRaw{}, err
	}
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
// (Q12bObjectiveStats), en BEST-EFFORT.
//
// Dégradation explicite (G1, 2026-07-26) : si la vue match_objective_stats_latest
// est absente (DB non migrée → Catalog Error) ou si la requête échoue, on logge un
// WARN structuré et on retourne une map VIDE. Le scoreboard reste servi entier ;
// seule la section objectifs manque. Jamais d'erreur avalée en silence : le WARN
// précède toujours la dégradation (CLAUDE.md règle n°3).
func loadMatchObjectiveStats(ctx context.Context, db *sql.DB, matchID string) map[string]domain.ObjectiveRaw {
	out := map[string]domain.ObjectiveRaw{}
	rows, err := db.QueryContext(ctx, Q12bObjectiveStats, matchID)
	if err != nil {
		slog.WarnContext(ctx, "match scoreboard: stats objectifs indisponibles, section dégradée",
			"match_id", matchID, "query", "Q12bObjectiveStats", "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		xuid, obj, scanErr := scanObjectiveRow(rows)
		if scanErr != nil {
			slog.WarnContext(ctx, "match scoreboard: scan stats objectifs échoué, ligne ignorée",
				"match_id", matchID, "err", scanErr)
			continue
		}
		out[xuid] = obj
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "match scoreboard: itération stats objectifs interrompue, section dégradée",
			"match_id", matchID, "err", err)
	}
	return out
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
// Dégradation silencieuse à 0 si la table est absente ou ligne vide.
func (r *MatchViewRepo) GetMatchObjectiveScore(ctx context.Context, xuid, matchID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var tableCount int
	tblRows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'personal_score_awards'
	`)
	if err != nil {
		return 0, nil
	}
	scanErr := tblRows.Scan(&tableCount)
	_ = tblRows.Close()
	if scanErr != nil || tableCount == 0 {
		return 0, nil
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
		return 0, nil
	}
	defer totalRows.Close()
	if err := totalRows.Scan(&total); err != nil {
		return 0, nil
	}
	return total, nil
}
