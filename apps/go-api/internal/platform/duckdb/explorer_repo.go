// Package duckdb — ExplorerRepo : matchs communs entre deux joueurs.
//
// Port Go de apps/api/app/routers/explorer.py + services/explorer.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// ExplorerRepo implémente port.ExplorerRepository sur DuckDB.
type ExplorerRepo struct {
	pdb  *PlayerDB
	xuid string
}

// NewExplorerRepo crée un ExplorerRepo.
func NewExplorerRepo(pdb *PlayerDB, xuid string) *ExplorerRepo {
	return &ExplorerRepo{pdb: pdb, xuid: xuid}
}

// GetCommonMatches retourne les matchs communs entre xuid1 et xuid2 (max 100).
// Q19 retourne 10 colonnes : match_id, start_time, map_ui, mode_ui,
// player1_team_id, player2_team_id, player1_outcome, player1_kills, player1_deaths, player1_kda.
// Exécutée sur SharedReader (ADR 0016, shared-only).
func (r *ExplorerRepo) GetCommonMatches(ctx context.Context, xuid1, xuid2 string) ([]domain.CommonMatchRaw, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetCommonMatches: shared reader: %w", err)
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, Q19CommonMatches, xuid1, xuid2)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetCommonMatches: query: %w", err)
	}
	defer rows.Close()

	var results []domain.CommonMatchRaw
	for rows.Next() {
		var m domain.CommonMatchRaw
		if err := rows.Scan(
			&m.MatchID,
			&m.StartTime,
			&m.MapUI,
			&m.ModeUI,
			&m.Player1TeamID,
			&m.Player2TeamID,
			&m.Player1Outcome,
			&m.Player1Kills,
			&m.Player1Deaths,
			&m.Player1KDA,
		); err != nil {
			return nil, fmt.Errorf("ExplorerRepo.GetCommonMatches: scan: %w", err)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetCommonMatches: rows: %w", err)
	}
	return results, nil
}

// GetKillerVictimBetween retourne les kills croisés agrégés entre xuid1 et xuid2 (Q19b).
// Exécutée sur SharedReader (ADR 0016, shared-only).
func (r *ExplorerRepo) GetKillerVictimBetween(ctx context.Context, xuid1, xuid2 string) (domain.KillerVictimAggregate, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return domain.KillerVictimAggregate{}, fmt.Errorf("ExplorerRepo.GetKillerVictimBetween: shared reader: %w", err)
	}
	defer release()
	row := sharedDB.QueryRowContext(ctx, Q19bKillerVictimBetween, xuid1, xuid2, xuid2, xuid1)
	var agg domain.KillerVictimAggregate
	if err := row.Scan(&agg.KillsDealt, &agg.DeathsSuffered); err != nil {
		return domain.KillerVictimAggregate{}, fmt.Errorf("ExplorerRepo.GetKillerVictimBetween: %w", err)
	}
	return agg, nil
}

// GetParticipantStatsForMatches agrège les stats brutes du joueur cible
// (xuid) sur la liste de matchs fournie. Lecture sur shared.match_participants.
//
// Win/loss/draw sont dérivés du champ `outcome` (1=tie, 2=win, 3=loss,
// 4=DNF). Les DNF ne comptent ni en wins ni en losses ni en draws — convention
// alignée sur le reste du produit (cf. compare_repo, match_view).
//
// Retourne nil si matchIDs est vide.
func (r *ExplorerRepo) GetParticipantStatsForMatches(
	ctx context.Context, xuid string, matchIDs []string,
) (*domain.ParticipantStatsAggregate, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(matchIDs)), ",")
	// PMT-5 / MT-06 : agrégats wins/losses/draws title-aware via le resolver d'issues
	// (fallback littéral Halo byte-identique si non câblé / titre sans raw_code).
	winExpr := outcomeSQLEq(ctx, "outcome", canonical.OutcomeWin, "outcome = 2")
	lossExpr := outcomeSQLEq(ctx, "outcome", canonical.OutcomeLoss, "outcome = 3")
	drawExpr := outcomeSQLEq(ctx, "outcome", canonical.OutcomeTie, "outcome = 1")
	q := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(kills), 0)               AS kills,
			COALESCE(SUM(deaths), 0)              AS deaths,
			COALESCE(SUM(assists), 0)             AS assists,
			COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS wins,
			COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS losses,
			COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS draws,
			COALESCE(SUM(shots_fired), 0)         AS shots_fired,
			COALESCE(SUM(shots_hit), 0)           AS shots_hit,
			COALESCE(SUM(damage_dealt), 0.0)      AS damage_dealt,
			COALESCE(SUM(damage_taken), 0.0)      AS damage_taken,
			COALESCE(SUM(headshot_kills), 0)      AS headshot_kills,
			COALESCE(SUM(melee_kills), 0)         AS melee_kills,
			COALESCE(SUM(power_weapon_kills), 0)  AS power_weapon_kills,
			COALESCE(SUM(grenade_kills), 0)       AS grenade_kills,
			COALESCE(SUM(time_played_seconds), 0) AS time_played_seconds,
			COALESCE(SUM(personal_score), 0)      AS personal_score
		FROM match_participants
		WHERE xuid = ? AND match_id IN (%s)
	`, winExpr, lossExpr, drawExpr, placeholders)

	args := make([]any, 0, 1+len(matchIDs))
	args = append(args, xuid)
	for _, mid := range matchIDs {
		args = append(args, mid)
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetParticipantStatsForMatches: shared reader: %w", err)
	}
	defer release()

	row := db.QueryRowContext(ctx, q, args...)
	var agg domain.ParticipantStatsAggregate
	err = row.Scan(
		&agg.Kills, &agg.Deaths, &agg.Assists,
		&agg.Wins, &agg.Losses, &agg.Draws,
		&agg.ShotsFired, &agg.ShotsHit,
		&agg.DamageDealt, &agg.DamageTaken,
		&agg.HeadshotKills, &agg.MeleeKills,
		&agg.PowerWeaponKills, &agg.GrenadeKills,
		&agg.TimePlayedSeconds, &agg.PersonalScore,
	)
	if err == sql.ErrNoRows {
		// Aucune ligne : le joueur n'a aucun participants row sur ces matchs.
		// On retourne un agrégat zéro (sampleSize sera 0 côté service).
		return &domain.ParticipantStatsAggregate{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetParticipantStatsForMatches: scan: %w", err)
	}
	return &agg, nil
}

// GetMedalCountsForMatches retourne le total d'occurrences (SUM(count)) et le
// nombre de types distincts de médailles gagnées par le joueur sur la liste
// de matchs fournie. Lecture sur shared.medals_earned.
//
// Retourne nil si matchIDs est vide.
func (r *ExplorerRepo) GetMedalCountsForMatches(
	ctx context.Context, xuid string, matchIDs []string,
) (*domain.MedalCountsAggregate, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(matchIDs)), ",")
	// Frags parfaits = set de médailles « frag parfait » du titre du joueur
	// (source unique analysis.PerfectKillMedalIDs ; HINF = {1512363953},
	// même approche que Q12MatchScoreboard / Q30 queries_squad.go).
	perfectClause := perfectKillMedalInClause("medal_name_id", pdbTitleSlug(r.pdb))
	q := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(count), 0)                  AS total,
			COALESCE(COUNT(DISTINCT medal_name_id), 0) AS unique_count,
			COALESCE(SUM(CASE WHEN %s THEN count ELSE 0 END), 0) AS perfect_kills
		FROM medals_earned
		WHERE xuid = ? AND match_id IN (%s)
	`, perfectClause, placeholders)

	args := make([]any, 0, 1+len(matchIDs))
	args = append(args, xuid)
	for _, mid := range matchIDs {
		args = append(args, mid)
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetMedalCountsForMatches: shared reader: %w", err)
	}
	defer release()

	row := db.QueryRowContext(ctx, q, args...)
	var agg domain.MedalCountsAggregate
	if err := row.Scan(&agg.Total, &agg.Unique, &agg.PerfectKills); err != nil {
		if err == sql.ErrNoRows {
			return &domain.MedalCountsAggregate{}, nil
		}
		return nil, fmt.Errorf("ExplorerRepo.GetMedalCountsForMatches: scan: %w", err)
	}
	return &agg, nil
}

// GetMatchStartTimesForXUID retourne les start_time (UTC) de tous les matchs du
// joueur dans shared.match_participants (join match_registry). Pattern timezone
// canonique (via StartTimeCanonicalSQL). Sert au
// bucketing "matchs par saison" côté service.
func (r *ExplorerRepo) GetMatchStartTimesForXUID(ctx context.Context, xuid string) ([]time.Time, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, nil
	}
	q := `
		SELECT ` + StartTimeCanonicalSQL("reg") + ` AS start_time
		FROM match_participants p
		JOIN match_registry reg ON reg.match_id = p.match_id
		WHERE p.xuid = ?`

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetMatchStartTimesForXUID: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, xuid)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetMatchStartTimesForXUID: query: %w", err)
	}
	defer rows.Close()

	var out []time.Time
	for rows.Next() {
		var ts time.Time
		if err := rows.Scan(&ts); err != nil {
			return nil, fmt.Errorf("ExplorerRepo.GetMatchStartTimesForXUID: scan: %w", err)
		}
		out = append(out, ts)
	}
	return out, rows.Err()
}

// GetTargetRecentMatches retourne les `limit` derniers matchs PvP (firefight
// exclu via is_firefight) du joueur, pour les graphes "profil de combat"
// (Q19cTargetRecentMatches). Lecture SharedReader (ADR 0016, shared-only).
func (r *ExplorerRepo) GetTargetRecentMatches(
	ctx context.Context, xuid string, limit int,
) ([]domain.ExplorerTargetRecentMatch, error) {
	if strings.TrimSpace(xuid) == "" || limit <= 0 {
		return nil, nil
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetTargetRecentMatches: shared reader: %w", err)
	}
	defer release()

	// Set perfect-kill résolu pour le titre du joueur (HINF byte-identique).
	q := resolvePerfectKillClause(Q19cTargetRecentMatches, "medal_name_id", pdbTitleSlug(r.pdb))
	rows, err := db.QueryContext(ctx, q, xuid, xuid, limit)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetTargetRecentMatches: query: %w", err)
	}
	defer rows.Close()

	var out []domain.ExplorerTargetRecentMatch
	for rows.Next() {
		m, scanErr := scanTargetRecentMatch(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("ExplorerRepo.GetTargetRecentMatches: scan: %w", scanErr)
		}
		out = append(out, m)
	}
	if rerr := rows.Err(); rerr != nil {
		return nil, rerr
	}
	// pair_name_fr est NULL en base → ResolveModeUI ne produit que de l'EN. On
	// applique la traduction FR canonique (metadata.mode_name_tr), même source
	// que l'historique de matchs / la home, pour le donut "Répartition des modes".
	r.translateModeUIsFR(ctx, out)
	return out, nil
}

// TranslateModeUIsFR expose translateModeUIsFR (port.ExplorerRepository) pour que
// l'ExplorerService homogénéise la source LIVE du profil de combat avec la locale.
func (r *ExplorerRepo) TranslateModeUIsFR(ctx context.Context, rows []domain.ExplorerTargetRecentMatch) {
	r.translateModeUIsFR(ctx, rows)
}

// translateModeUIsFR remplace en place les libellés de mode EN normalisés
// (ex. "CTF") par leur traduction FR depuis metadata.mode_name_tr (lang='fr').
// Best-effort : silencieux si Metadata absent / table absente / erreur — l'EN
// est alors conservé. Clé = sous-mode EN normalisé (même convention que career).
func (r *ExplorerRepo) translateModeUIsFR(ctx context.Context, rows []domain.ExplorerTargetRecentMatch) {
	if len(rows) == 0 || r.pdb == nil {
		return
	}
	// Source LIVE : ModeUI est souvent vide (le MatchInfo brut de l'API n'expose pas
	// de PublicName, seulement des AssetId). On résout d'abord le mode depuis l'AssetId
	// du pair via shared.match_registry, AVANT la traduction FR ci-dessous.
	r.resolveModeUIsFromPairID(ctx, rows)
	if r.pdb.Metadata == nil {
		return
	}
	modeSet := make(map[string]struct{})
	for i := range rows {
		if rows[i].ModeUI != "" {
			modeSet[rows[i].ModeUI] = struct{}{}
		}
	}
	if len(modeSet) == 0 {
		return
	}
	modes := make([]string, 0, len(modeSet))
	for m := range modeSet {
		modes = append(modes, m)
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(modes)), ",")
	q := fmt.Sprintf(`SELECT mode_en, name FROM mode_name_tr WHERE lang = 'fr' AND mode_en IN (%s)`, placeholders)
	args := make([]any, len(modes))
	for i, m := range modes {
		args[i] = m
	}
	qrows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return
	}
	defer qrows.Close()
	fr := make(map[string]string, len(modes))
	for qrows.Next() {
		var en, name string
		if scanErr := qrows.Scan(&en, &name); scanErr != nil {
			continue
		}
		if strings.TrimSpace(name) != "" {
			fr[en] = name
		}
	}
	for i := range rows {
		if t, ok := fr[rows[i].ModeUI]; ok {
			rows[i].ModeUI = t
		}
	}
}

// resolveModeUIsFromPairID remplit ModeUI (EN normalisé) des rows LIVE qui n'ont
// pas de mode — le MatchInfo brut n'expose pas de PublicName — depuis l'AssetId du
// PlaylistMapModePair via shared.match_registry (pair_id → pair_name). Les joueurs
// suivis ont déjà vu les pairs matchmaking communs → la plupart résolvent (sinon le
// match reste "Inconnu", dégradation acceptable). Best-effort : silencieux sur erreur.
func (r *ExplorerRepo) resolveModeUIsFromPairID(ctx context.Context, rows []domain.ExplorerTargetRecentMatch) {
	ids := make(map[string]struct{})
	for i := range rows {
		if rows[i].ModeUI == "" && rows[i].ModePairAssetID != "" {
			ids[rows[i].ModePairAssetID] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return
	}
	idList := make([]string, 0, len(ids))
	for id := range ids {
		idList = append(idList, id)
	}
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return
	}
	defer release()
	placeholders := strings.TrimRight(strings.Repeat("?,", len(idList)), ",")
	q := fmt.Sprintf(`SELECT pair_id, MIN(pair_name), MIN(pair_name_fr)
		FROM match_registry WHERE pair_id IN (%s) AND pair_name IS NOT NULL
		GROUP BY pair_id`, placeholders)
	args := make([]any, len(idList))
	for i, id := range idList {
		args[i] = id
	}
	qrows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return
	}
	defer qrows.Close()
	type pairNames struct{ name, nameFR sql.NullString }
	byID := make(map[string]pairNames, len(idList))
	for qrows.Next() {
		var id string
		var p pairNames
		if scanErr := qrows.Scan(&id, &p.name, &p.nameFR); scanErr == nil {
			byID[id] = p
		}
	}
	for i := range rows {
		if rows[i].ModeUI != "" || rows[i].ModePairAssetID == "" {
			continue
		}
		p, ok := byID[rows[i].ModePairAssetID]
		if !ok {
			continue
		}
		var namePtr, frPtr *string
		if p.name.Valid {
			namePtr = &p.name.String
		}
		if p.nameFR.Valid {
			frPtr = &p.nameFR.String
		}
		if mode := analysis.ResolveModeUI(namePtr, frPtr); mode != nil {
			rows[i].ModeUI = *mode
		}
	}
}

// scanTargetRecentMatch projette une ligne de Q19cTargetRecentMatches vers le
// DTO domain. rank est NULL pour les DNF/non classés (→ *int nil) ; damage_* est
// stocké en DOUBLE et arrondi à l'entier (les graphes n'affichent pas de décimale).
func scanTargetRecentMatch(rows *sql.Rows) (domain.ExplorerTargetRecentMatch, error) {
	var m domain.ExplorerTargetRecentMatch
	var rank sql.NullInt64
	var damageDealt, damageTaken float64
	var pairName, pairNameFR *string
	if err := rows.Scan(
		&m.MatchID, &m.StartTime, &m.MapUI, &pairName, &pairNameFR,
		&m.Outcome, &rank,
		&m.Kills, &m.Deaths, &m.Assists, &m.KDA,
		&m.Score, &damageDealt, &damageTaken,
		&m.MaxKillingSpree, &m.PerfectKills,
	); err != nil {
		return m, err
	}
	// Libellé de mode normalisé + localisé (FR-preferring) — helper canonique
	// partagé avec home / match-view / historique (ResolveModeUI =
	// COALESCE(pair_name_fr, pair_name) → NormalizeModeLabel).
	if mode := analysis.ResolveModeUI(pairName, pairNameFR); mode != nil {
		m.ModeUI = *mode
	}
	if rank.Valid {
		v := int(rank.Int64)
		m.Rank = &v
	}
	m.DamageDealt = int(damageDealt)
	m.DamageTaken = int(damageTaken)
	return m, nil
}

// ResolveXUIDByGamertag résout un gamertag en xuid via shared.v_gamertag_lookup (ILIKE).
//
// Source : la vue v_gamertag_lookup (cascade xuid_aliases ∪ match_participants
// avec fallback bots officiels). Plus robuste que la table xuid_aliases seule
// car elle capture aussi les joueurs qui sont apparus dans match_participants
// avant d'être synchronisés dans xuid_aliases. Bots filtrés (xuid 'bid(...)').
func (r *ExplorerRepo) ResolveXUIDByGamertag(ctx context.Context, gamertag string) (string, error) {
	const q = `
		SELECT xuid FROM v_gamertag_lookup
		WHERE gamertag ILIKE ? AND xuid NOT LIKE 'bid(%'
		LIMIT 1
	`
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return "", fmt.Errorf("ExplorerRepo.ResolveXUIDByGamertag(%q): %w", gamertag, err)
	}
	defer release()

	var xuid string
	if err := db.QueryRowContext(ctx, q, gamertag).Scan(&xuid); err != nil {
		return "", fmt.Errorf("ExplorerRepo.ResolveXUIDByGamertag(%q): %w", gamertag, err)
	}
	return xuid, nil
}

// GetTopWeaponsForMatches retourne le top `limit` armes (par kills) du joueur sur
// les matchs donnés : COUNT(*) par effective_weapon_id dans shared.v_weapon_kills
// (1 ligne = 1 kill event). Labels résolus
// depuis metadata.weapon_labels. Best-effort : nil si entrée vide / erreur
// (feature secondaire, jamais fatale). Armes 0/1/2 exclues (melee/grenade génériques).
func (r *ExplorerRepo) GetTopWeaponsForMatches(
	ctx context.Context, xuid string, matchIDs []string, limit int,
) ([]domain.WeaponHighlight, error) {
	if strings.TrimSpace(xuid) == "" || len(matchIDs) == 0 || limit <= 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(matchIDs)), ",")
	q := fmt.Sprintf(`
		SELECT effective_weapon_id, COUNT(*) AS kills
		FROM v_weapon_kills
		WHERE xuid = ? AND match_id IN (%s) AND effective_weapon_id NOT IN (0, 1, 2)
		GROUP BY effective_weapon_id
		ORDER BY kills DESC
		LIMIT %d`, placeholders, limit) //nolint:gosec — limit/placeholders maîtrisés

	args := make([]any, 0, 1+len(matchIDs))
	args = append(args, xuid)
	for _, mid := range matchIDs {
		args = append(args, mid)
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.DebugContext(ctx, "ExplorerRepo.GetTopWeaponsForMatches: shared reader (best-effort)", "err", err)
		return nil, nil //nolint:nilerr
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		slog.DebugContext(ctx, "ExplorerRepo.GetTopWeaponsForMatches: query (best-effort)", "err", err)
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var out []domain.WeaponHighlight
	for rows.Next() {
		var wid UBigint
		var kills int
		if scanErr := rows.Scan(&wid, &kills); scanErr != nil {
			return nil, fmt.Errorf("ExplorerRepo.GetTopWeaponsForMatches: scan: %w", scanErr)
		}
		idStr := strconv.FormatUint(uint64(wid), 10) //nolint:gosec
		out = append(out, domain.WeaponHighlight{
			WeaponID: wid.Int64(),
			Kills:    kills,
			LabelFR:  idStr,
			LabelEN:  idStr,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.resolveWeaponLabels(ctx, out)
	return out, nil
}

// resolveWeaponLabels remplit LabelFR/LabelEN en place depuis metadata.weapon_labels.
// Best-effort : laisse l'ID décimal si metadata absent / arme inconnue. IDs injectés
// en littéraux décimaux (cast UBIGINT non bindable proprement, cf. compare_repo).
func (r *ExplorerRepo) resolveWeaponLabels(ctx context.Context, weapons []domain.WeaponHighlight) {
	if len(weapons) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return
	}
	ids := make([]int64, 0, len(weapons))
	for _, w := range weapons {
		ids = append(ids, w.WeaponID)
	}
	// PASSAGE PRINCIPAL P4 : nom via le registre + weapon_labels (parité). label =
	// name_fr>name_en (= LabelFR) ; LabelEN = name_en sinon le label FR (parité avec
	// l'ancien COALESCE inversé).
	meta := resolveWeaponMeta(ctx, r.pdb.Metadata, r.pdb.TitleSlug, ids)
	for i := range weapons {
		m, ok := meta[weapons[i].WeaponID]
		if !ok || m.label == "" {
			continue
		}
		weapons[i].LabelFR = m.label
		if m.nameEN != "" {
			weapons[i].LabelEN = m.nameEN
		} else {
			weapons[i].LabelEN = m.label
		}
	}
}
