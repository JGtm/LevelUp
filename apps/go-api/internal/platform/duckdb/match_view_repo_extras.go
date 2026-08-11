// Package duckdb — match_view_repo_extras.go : sections secondaires de la
// vue Match (events highlight, KV pairs, encounters, encounter stats, media,
// expected stats, history for avg, player assists model). Découpé de
// match_view_repo.go (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// GetMatchEvents retourne les events highlight du match (Q21).
// Exécutée sur SharedReader (ADR 0016, shared-only).
func (r *MatchViewRepo) GetMatchEvents(ctx context.Context, matchID string) ([]domain.EventRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, nil
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, Q21MatchEventsWithXUID, matchID)
	if err != nil {
		// La table peut être absente sur certains matchs → retourner vide
		return nil, nil
	}
	defer rows.Close()

	var results []domain.EventRaw
	for rows.Next() {
		var e domain.EventRaw
		if err := rows.Scan(&e.EventType, &e.TimeMS, &e.XUID, &e.Gamertag); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchEvents scan: %w", err)
		}
		results = append(results, e)
	}
	return results, rows.Err()
}

// GetMatchKillSources retourne la source de dégât de chaque mort du match (Q21b).
// Exécutée sur SharedReader (ADR 0016, shared-only).
//
// Dégradation gracieuse assumée à DEUX endroits : reader indisponible, ou table absente
// d'une base non migrée. Dans les deux cas on rend une tranche vide, jamais une erreur —
// le kill feed s'affiche alors sans arme, ce qui est son état d'avant ce lot. L'échec est
// loggé pour qu'il ne soit pas SILENCIEUX (règle : jamais d'erreur avalée).
func (r *MatchViewRepo) GetMatchKillSources(ctx context.Context, matchID string) ([]domain.KillSourceRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "match_view: kill sources indisponibles (shared reader)",
			"match_id", matchID, "err", err)
		return nil, nil
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, Q21bKillSources, matchID)
	if err != nil {
		slog.WarnContext(ctx, "match_view: kill sources indisponibles (Q21b)",
			"match_id", matchID, "err", err)
		return nil, nil
	}
	defer rows.Close()

	var results []domain.KillSourceRaw
	for rows.Next() {
		var (
			ks  domain.KillSourceRaw
			tag uint32
		)
		if err := rows.Scan(&ks.XUID, &ks.TimeMS, &tag); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchKillSources scan: %w", err)
		}
		ks.SourceTag = tag
		results = append(results, ks)
	}
	return results, rows.Err()
}

// GetMatchKVPairs retourne les paires killer→victim du match (Q20).
// Exécutée sur SharedReader (ADR 0016, shared-only).
func (r *MatchViewRepo) GetMatchKVPairs(ctx context.Context, matchID string) ([]domain.KVPairRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, nil
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, Q20KVPairs, matchID)
	if err != nil {
		// Q20 lit la TABLE `killer_victim_pairs` directement depuis le 2026-08-02 (la vue
		// v_killer_victim_full a été supprimée avec la bascule J4 ; elle ne fait plus partie du
		// schéma). La table peut manquer sur une DB non migrée → duels vides, jamais d'erreur.
		return nil, nil
	}
	defer rows.Close()

	var results []domain.KVPairRaw
	for rows.Next() {
		var kv domain.KVPairRaw
		if err := rows.Scan(
			&kv.KillerXUID,
			&kv.KillerGT,
			&kv.VictimXUID,
			&kv.VictimGT,
			&kv.KillCount,
			&kv.TimeMS,
		); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchKVPairs scan: %w", err)
		}
		results = append(results, kv)
	}
	return results, rows.Err()
}

// GetMatchEncounters retourne l'historique de rencontres avec les participants (Q23).
// Exécutée sur SharedReader (ADR 0016) — Q23 lit match_participants +
// v_gamertag_lookup (shared-only).
func (r *MatchViewRepo) GetMatchEncounters(ctx context.Context, matchID, myXUID string) ([]domain.EncounterRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounters: shared reader: %w", err)
	}
	defer release()

	// Masquage Campagne (Halo 5) : count_together agrège l'historique du joueur
	// (me.xuid) → forme by-match-id sur me.match_id. No-op Infinite. Item backlog H1.
	q := resolveCampaignExclusionByMatchID(Q23MatchEncounters, r.pdb.TitleSlug, "me.match_id")
	rows, err := sharedDB.QueryContext(ctx, q,
		matchID, myXUID, // this_match WHERE
		matchID, myXUID, // my_team WHERE
		myXUID, // me.xuid = ?
	)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounters: %w", err)
	}
	defer rows.Close()

	var results []domain.EncounterRaw
	for rows.Next() {
		var enc domain.EncounterRaw
		if err := rows.Scan(&enc.XUID, &enc.Gamertag, &enc.IsBot, &enc.CountTogether, &enc.IsAlly); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounters scan: %w", err)
		}
		results = append(results, enc)
	}
	return results, rows.Err()
}

// GetMatchEncounterStats retourne les stats riches par encounter (Q23b,
// chunk MV4.C'). Permet d'attribuer les badges narratifs ally_plus et
// tough_enemy.
//
// Tolérance : si killer_victim_pairs ou xuid_aliases sont absents, retourne
// les rows partielles. Si la table principale (match_participants) est
// absente, retourne nil + warning.
func (r *MatchViewRepo) GetMatchEncounterStats(ctx context.Context, matchID, myXUID string) ([]domain.EncounterStatsRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// Q23b lit match_participants + match_registry + killer_victim_pairs +
	// v_gamertag_lookup (shared-only) — via SharedReader (ADR 0016).
	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounterStats: shared reader: %w", err)
	}
	defer release()

	// PMT-5 : exprs win/loss title-aware (fallback "eh.me_outcome = 2/3" byte-identique
	// Halo). Ordre des %s du template : win, loss, win, loss.
	winExpr := outcomeSQLEq(ctx, "eh.me_outcome", canonical.OutcomeWin, "eh.me_outcome = 2")
	lossExpr := outcomeSQLEq(ctx, "eh.me_outcome", canonical.OutcomeLoss, "eh.me_outcome = 3")
	// Masquage Campagne (Halo 5) : my_history agrège tout l'historique du joueur
	// → forme by-match-id (résolue AVANT Sprintf, sans placeholder). No-op Infinite.
	tpl := resolveCampaignExclusionByMatchID(Q23bMatchEncounterStats, r.pdb.TitleSlug, "match_id")
	encounterStatsSQL := fmt.Sprintf(tpl, winExpr, lossExpr, winExpr, lossExpr)
	rows, err := sharedDB.QueryContext(ctx, encounterStatsSQL,
		matchID, myXUID, // this_match
		matchID, myXUID, // my_team
		myXUID,         // my_history
		myXUID, myXUID, // kv_stats CASE (kills_dealt, deaths_suffered)
		myXUID, myXUID, // kv_stats JOIN ON
	)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounterStats: %w", err)
	}
	defer rows.Close()

	var out []domain.EncounterStatsRaw
	for rows.Next() {
		var (
			s           domain.EncounterStatsRaw
			firstSeenAt sql.NullTime
			lastSeenAt  sql.NullTime
		)
		if err := rows.Scan(
			&s.XUID,
			&s.AllyCount,
			&s.EnemyCount,
			&s.WinsAsAlly,
			&s.LossesAsAlly,
			&s.WinsVsEnemy,
			&s.LossesVsEnemy,
			&s.KillsDealt,
			&s.DeathsSuffered,
			&firstSeenAt,
			&lastSeenAt,
		); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounterStats scan: %w", err)
		}
		if firstSeenAt.Valid {
			s.FirstSeen = firstSeenAt.Time
		}
		if lastSeenAt.Valid {
			s.LastSeenAt = lastSeenAt.Time
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetMatchMedia retourne les médias associés au match (Q24).
// Utilise shared_social DB. Cross-joueur : tous les auteurs sont retournés.
// Le champ Liked est celui DU VIEWER (r.viewerSlug), pas un état global.
func (r *MatchViewRepo) GetMatchMedia(ctx context.Context, matchID string) ([]domain.MediaAssocRaw, error) {
	if r.pdb.SharedSocial == nil {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.SharedSocial.Query(ctx, Q24MatchMedia, r.viewer(), matchID)
	if err != nil {
		// Absence de la table ou DB non configurée → résultat vide sans erreur
		// bloquante. On loggue tout de même : un échec récurrent ici masquerait
		// silencieusement un bug d'index (cf. incident 2026-05-07 Q24).
		slog.WarnContext(ctx, "match_view: Q24 query échouée — médias indisponibles",
			"match_id", matchID, "err", err)
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var results []domain.MediaAssocRaw
	for rows.Next() {
		var m domain.MediaAssocRaw
		var captureTime *time.Time
		if err := rows.Scan(
			&m.FileID,
			&m.FileName,
			&m.FilePath,
			&m.Kind,
			&m.ThumbnailPath,
			&captureTime,
			&m.Liked,
		); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchMedia scan: %w", err)
		}
		if captureTime != nil {
			s := captureTime.Format(time.RFC3339)
			m.CaptureTime = &s
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// GetMatchExpectedStats retourne les stats attendues pour ce match (Q26).
// Exécutée sur SharedReader (ADR 0016) — Q26 lit match_participants (shared-only).
func (r *MatchViewRepo) GetMatchExpectedStats(ctx context.Context, matchID, xuid string) (*domain.ExpectedStatsRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer release()

	var row domain.ExpectedStatsRaw
	err = sharedDB.QueryRowContext(ctx, Q26MatchExpectedStats, matchID, xuid).Scan(
		&row.KillsExpected,
		&row.DeathsExpected,
		&row.KillsStddev,
		&row.DeathsStddev,
	)
	if err != nil {
		// Colonnes absentes ou match introuvable → nil sans erreur
		return nil, nil //nolint:nilerr
	}
	return &row, nil
}

// GetHistoryForAvg retourne les 50 derniers matchs du joueur (Q29) pour le
// calcul des moyennes historiques K/D/A + spree/headshots/perfect.
// Exécutée sur SharedReader (ADR 0016) — Q29 lit match_participants +
// match_registry + medals_earned (shared-only).
func (r *MatchViewRepo) GetHistoryForAvg(ctx context.Context, xuid string) ([]domain.MatchHistAvgRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "GetHistoryForAvg shared reader failed", "err", err)
		return nil, nil //nolint:nilerr
	}
	defer release()

	// Set perfect-kill résolu pour le titre du joueur (HINF byte-identique).
	q := resolvePerfectKillClause(Q29HistoryForAvg, "m.medal_name_id", pdbTitleSlug(r.pdb))
	q = resolveCampaignExclusion(q, pdbTitleSlug(r.pdb), "r")
	rows, err := sharedDB.QueryContext(ctx, q, xuid, xuid)
	if err != nil {
		slog.WarnContext(ctx, "GetHistoryForAvg query failed", "err", err)
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var results []domain.MatchHistAvgRow
	for rows.Next() {
		var row domain.MatchHistAvgRow
		if err := rows.Scan(
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.HeadshotKills,
			&row.MaxKillingSpree,
			&row.PerfectKills,
			&row.PairName,
			&row.IsFirefight,
			&row.IsRanked,
			&row.DurationSeconds,
		); err != nil {
			return nil, fmt.Errorf("GetHistoryForAvg scan: %w", err)
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// GetHistoryForAvgBulk retourne l'historique Q29 (50 derniers matchs) pour PLUSIEURS
// xuids en une seule requête (J3). Remplace la boucle ~8× GetHistoryForAvg du builder
// Match View (moyennes K/D locales des amis du scoreboard). Résultat groupé par xuid ;
// sémantique par joueur identique à GetHistoryForAvg. Best-effort : reader/requête en
// échec → map nil (le builder dégrade sans expected K/D, comme la voie unitaire).
func (r *MatchViewRepo) GetHistoryForAvgBulk(ctx context.Context, xuids []string) (map[string][]domain.MatchHistAvgRow, error) {
	if len(xuids) == 0 {
		return map[string][]domain.MatchHistAvgRow{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "GetHistoryForAvgBulk shared reader failed", "err", err)
		return nil, nil //nolint:nilerr
	}
	defer release()

	ph := strings.TrimRight(strings.Repeat("?,", len(xuids)), ",")
	q := resolvePerfectKillClause(
		fmt.Sprintf(Q29HistoryForAvgBulkTpl, ph, ph), "m.medal_name_id", pdbTitleSlug(r.pdb))
	q = resolveCampaignExclusion(q, pdbTitleSlug(r.pdb), "r")
	// La liste xuid alimente les DEUX clauses IN (recent puis perfect), dans l'ordre.
	args := make([]any, 0, len(xuids)*2)
	for _, x := range xuids {
		args = append(args, x)
	}
	for _, x := range xuids {
		args = append(args, x)
	}
	rows, err := sharedDB.QueryContext(ctx, q, args...)
	if err != nil {
		slog.WarnContext(ctx, "GetHistoryForAvgBulk query failed", "err", err)
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string][]domain.MatchHistAvgRow, len(xuids))
	for rows.Next() {
		var xuid string
		var row domain.MatchHistAvgRow
		if err := rows.Scan(
			&xuid,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.HeadshotKills,
			&row.MaxKillingSpree,
			&row.PerfectKills,
			&row.PairName,
			&row.IsFirefight,
			&row.IsRanked,
			&row.DurationSeconds,
		); err != nil {
			return nil, fmt.Errorf("GetHistoryForAvgBulk scan: %w", err)
		}
		out[xuid] = append(out[xuid], row)
	}
	return out, rows.Err()
}

// GetPlayerAssistsModel retourne les coefs OLS expected_assists pour un mode.
// Retourne nil si la table est absente ou si le mode n'a pas assez de données.
func (r *MatchViewRepo) GetPlayerAssistsModel(ctx context.Context, gameVariantName string) (*domain.PlayerAssistsModel, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const q = `
		SELECT game_variant_name,
		       coef_intercept, coef_kills, coef_deaths,
		       coef_damage_dealt, coef_damage_taken, coef_mmr_delta,
		       r2, n_samples
		FROM player_assists_model
		WHERE game_variant_name = ?
		LIMIT 1
	`
	var m domain.PlayerAssistsModel
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, q, gameVariantName)
	if err != nil {
		return nil, nil //nolint:nilerr — table absente, mode inconnu ou Reopen concurrent : dégradation gracieuse
	}
	defer rows.Close()
	if err := rows.Scan(
		&m.GameVariantName,
		&m.Intercept, &m.CoefKills, &m.CoefDeaths,
		&m.CoefDamageDealt, &m.CoefDamageTaken, &m.CoefMMRDelta,
		&m.R2, &m.N,
	); err != nil {
		return nil, nil //nolint:nilerr
	}
	return &m, nil
}
