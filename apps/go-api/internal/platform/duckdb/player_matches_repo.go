// Package duckdb — player_matches_repo.go : implementation DuckDB du loader
// unifie des matchs joueur (port.PlayerMatchesRepository).
//
// Per-player : un PlayerMatchesRepo est lie a un PlayerDB precis. La resolution
// (slug, gamertag) -> PlayerDB se fait via pool.GetOrOpen au niveau de
// l'adapter qui consomme le repo (chunk ulterieur).
//
// Capability gating : laisse au service appelant pour cette implementation.
// Le repo execute la requete telle quelle ; si le titre n'a pas la capability
// "match.history", c'est au service de retourner games.ErrCapabilityNotSupported
// avant d'appeler le repo.
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// PlayerMatchesRepo charge les matchs d'un joueur depuis sa DB stats avec
// shared attache. Implemente une variante per-player de
// port.PlayerMatchesRepository (le slug et le gamertag sont fixes par le
// PlayerDB injecte au constructeur).
type PlayerMatchesRepo struct {
	pdb *PlayerDB
}

// NewPlayerMatchesRepo cree un PlayerMatchesRepo lie a un PlayerDB.
func NewPlayerMatchesRepo(pdb *PlayerDB) *PlayerMatchesRepo {
	return &PlayerMatchesRepo{pdb: pdb}
}

// Load charge les matchs du joueur en suivant les filtres fournis. Retourne
// les rows projetees en canonical.PlayerMatchRow. Trie par r.start_time DESC
// par defaut, override possible via filters.OrderBy (whitelist).
//
// L'appelant doit avoir valide les filtres via filters.Validate() en amont.
// Le repo re-applique aussi sa propre validation defensive (input untrusted).
//
// split+merge cross-DB.
//
//	Étape 1 (SharedReader) : query shared (v_match_full ⨝ match_participants ⨝
//	subquery medals_earned) avec tous les filtres shared (Period, Outcome,
//	IsFirefight, IsRanked, MinTimePlayed, BTBExcluded, PlaylistKind, MapIDs,
//	ExcludeFriendsXUIDs) + ORDER BY si tri sur colonne shared.
//	Étape 2 (pdb.Player) : player_match_enrichment WHERE match_id IN (...)
//	Étape 3 (pdb.Player) : match_skill_rank WHERE match_id IN (...)
//	Étape 4 (Go) : merge LEFT JOIN, application HadBotTeammate filter post-merge,
//	re-tri sur performance_score (PME) si nécessaire, LIMIT.
func (r *PlayerMatchesRepo) Load(
	ctx context.Context,
	filters port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Étape 1 : query shared.
	sharedResults, err := r.loadSharedRows(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: %w", err)
	}
	if len(sharedResults) == 0 {
		return nil, nil
	}

	matchIDs := make([]string, 0, len(sharedResults))
	for i := range sharedResults {
		matchIDs = append(matchIDs, sharedResults[i].matchID)
	}

	// Étape 2 + 3 : enrichments + skill ranks.
	enrichments, err := r.loadEnrichmentsForMatches(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: %w", err)
	}
	skillRanks, err := r.loadSkillRanksForMatches(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: %w", err)
	}
	// Étape 3b : season_id + measurement_matches_remaining depuis match_csrs
	// (shared DB) — match_skill_rank ne porte pas ces colonnes.
	csrMeta, err := r.loadMatchCSRMetaForMatches(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: %w", err)
	}

	// Étape 4 : merge + filtres player + tri PME + LIMIT.
	out := r.mergePlayerMatchRows(sharedResults, enrichments, skillRanks, csrMeta, filters)
	return out, nil
}

// buildSharedQuery compose la partie shared (v_match_full ⨝ match_participants
// + subquery medals_earned) avec les filtres applicables shared-only.
// HadBotTeammate (PME) reste filtré côté Go après merge.
//
// ORDER BY : si sur colonne shared (start_time), ajouté ici + LIMIT propagé.
// Si sur colonne player (performance_score), order/limit appliqués post-merge.
func (r *PlayerMatchesRepo) buildSharedQuery(f port.PlayerMatchFilters) (string, []any, sharedQueryHints, error) {
	var sb strings.Builder
	// Set perfect-kill title-aware (source unique analysis.PerfectKillMedalIDs ;
	// HINF byte-identique = me.medal_name_id IN (1512363953)).
	sb.WriteString(resolvePerfectKillClause(
		playerMatchesSharedBaseSelect, "me.medal_name_id", pdbTitleSlug(r.pdb),
	))

	args := []any{r.pdb.XUID}

	appendPlayerMatchScalarFilters(&sb, &args, f)
	if err := appendPlayerMatchSetFilters(&sb, &args, f); err != nil {
		return "", nil, sharedQueryHints{}, err
	}
	// Masquage read-side des modes exclus du titre (Halo 5 : Campagne). Alias "r" =
	// v_match_full dans playerMatchesSharedBaseSelect.
	if clause, exArgs := excludedVariantClause(pdbTitleSlug(r.pdb), "r"); clause != "" {
		sb.WriteString(clause)
		args = append(args, exArgs...)
	}

	hints, orderBy, err := classifyOrderBy(f.OrderBy)
	if err != nil {
		return "", nil, sharedQueryHints{}, err
	}
	sb.WriteString(" ORDER BY ")
	sb.WriteString(orderBy)

	// LIMIT côté SQL uniquement si ORDER BY shared ET pas de filtre PME.
	// Sinon (PME order ou filtre HadBotTeammate), récupère tout et applique en Go.
	if hints.canPushLimit && f.Limit > 0 && f.HadBotTeammate == nil {
		sb.WriteString(" LIMIT ?")
		args = append(args, f.Limit)
	}

	return sb.String(), args, hints, nil
}

// StartTimeCanonicalSQL délègue à analysis.SQLStartTimeCanonical — source unique
// du fragment timezone canonique (règle CLAUDE.md n°8). Conservé comme alias local
// pour les repos platform/duckdb (appel sans préfixe de package). Le garde-rail
// analysis/start_time_canonical_test.go interdit le littéral brut ailleurs.
// Unification H1 (2026-07-04) : le corps a migré vers internal/analysis car
// analysis/match_filter.go en a besoin et ne peut pas importer platform/duckdb.
func StartTimeCanonicalSQL(alias string) string {
	return analysis.SQLStartTimeCanonical(alias)
}

// startTimeCanonicalSQL : expression canonique pour l'alias `r` (projection,
// filtre Period, ORDER BY de player_matches_repo). Source unique via le helper.
var startTimeCanonicalSQL = StartTimeCanonicalSQL("r")

// appendPlayerMatchScalarFilters ajoute les filtres scalaires (Period, IsFirefight,
// IsRanked, MinTimePlayedSeconds, BTBExcluded).
func appendPlayerMatchScalarFilters(sb *strings.Builder, args *[]any, f port.PlayerMatchFilters) {
	if since := periodSince(f.Period); since != nil {
		sb.WriteString(" AND " + startTimeCanonicalSQL + " >= ?")
		*args = append(*args, *since)
	}
	if f.IsFirefight != nil {
		sb.WriteString(" AND COALESCE(r.is_firefight, FALSE) = ?")
		*args = append(*args, *f.IsFirefight)
	}
	if f.IsRanked != nil {
		sb.WriteString(` AND (CASE
			WHEN COALESCE(r.is_ranked, FALSE)
				OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
				OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
			THEN TRUE ELSE FALSE END) = ?`)
		*args = append(*args, *f.IsRanked)
	}
	if f.MinTimePlayedSeconds != nil {
		sb.WriteString(" AND COALESCE(p.time_played_seconds, 0) >= ?")
		*args = append(*args, *f.MinTimePlayedSeconds)
	}
	if f.BTBExcluded {
		sb.WriteString(" AND (r.pair_name IS NULL OR LOWER(r.pair_name) NOT LIKE '%btb%')")
	}
}

// appendPlayerMatchSetFilters ajoute les filtres IN (OutcomeIn, ExcludeFriendsXUIDs,
// MapIDs) et PlaylistKind. Peut retourner une erreur sur PlaylistKind invalide.
func appendPlayerMatchSetFilters(sb *strings.Builder, args *[]any, f port.PlayerMatchFilters) error {
	if len(f.OutcomeIn) > 0 {
		placeholders := make([]string, 0, len(f.OutcomeIn))
		for _, o := range f.OutcomeIn {
			placeholders = append(placeholders, "?")
			*args = append(*args, outcomeToInt(o))
		}
		fmt.Fprintf(sb, " AND COALESCE(p.outcome, 0) IN (%s)",
			strings.Join(placeholders, ","))
	}
	if len(f.ExcludeFriendsXUIDs) > 0 {
		placeholders := make([]string, 0, len(f.ExcludeFriendsXUIDs))
		for _, x := range f.ExcludeFriendsXUIDs {
			placeholders = append(placeholders, "?")
			*args = append(*args, x)
		}
		fmt.Fprintf(sb, " AND p.match_id NOT IN (SELECT match_id FROM match_participants WHERE xuid IN (%s))",
			strings.Join(placeholders, ","))
	}
	if f.PlaylistKind != nil {
		clause, err := playlistKindClause(*f.PlaylistKind)
		if err != nil {
			return err
		}
		if clause != "" {
			sb.WriteString(" AND ")
			sb.WriteString(clause)
		}
	}
	if len(f.MapIDs) > 0 {
		placeholders := make([]string, 0, len(f.MapIDs))
		for _, id := range f.MapIDs {
			placeholders = append(placeholders, "?")
			*args = append(*args, id)
		}
		fmt.Fprintf(sb, " AND COALESCE(r.map_id, '') IN (%s)",
			strings.Join(placeholders, ","))
	}
	return nil
}

// sharedQueryHints regroupe les hints sur le découpage ORDER BY + LIMIT entre
// SQL et Go.
type sharedQueryHints struct {
	canPushLimit  bool // ORDER BY est sur colonne shared, LIMIT peut être SQL
	postMergeSort string
}

// classifyOrderBy détermine si l'ORDER BY peut s'appliquer côté SQL (shared col)
// ou doit s'appliquer post-merge en Go (PME col). Retourne aussi la clause SQL
// à utiliser (vers shared cols seulement).
func classifyOrderBy(s string) (sharedQueryHints, string, error) {
	switch strings.TrimSpace(s) {
	case "", "start_time DESC":
		return sharedQueryHints{canPushLimit: true}, startTimeCanonicalSQL + " DESC", nil
	case "start_time ASC":
		return sharedQueryHints{canPushLimit: true}, startTimeCanonicalSQL + " ASC", nil
	case "performance_score DESC":
		// Tri post-merge. SQL garde un ordre stable mais non significatif.
		return sharedQueryHints{canPushLimit: false, postMergeSort: "performance_score DESC"},
			startTimeCanonicalSQL + " DESC", nil
	case "performance_score ASC":
		return sharedQueryHints{canPushLimit: false, postMergeSort: "performance_score ASC"},
			startTimeCanonicalSQL + " DESC", nil
	}
	return sharedQueryHints{}, "", fmt.Errorf("%w: %q", ErrUnknownOrderBy, s)
}

// playerMatchesSharedBaseSelect : (ADR 0016) — partie shared du split
// PlayerMatchesRepo.Load. Toutes les tables/vues référencées sont au niveau root
// du catalogue shared_matches_v2.duckdb (pas de préfixe `shared.`).
//
// 40 colonnes : match metadata + participant stats + team_id + team_0/1_score
// + perfect_kills (subquery sur medals_earned) + t0_ms (countdown pré-match,
// Match Timeline T0 Phase 3). Les colonnes PME (session,
// performance, dominance, had_bot, is_with_friends) et match_skill_rank (tier,
// rating, etc.) sont hydratées en étape 2/3 (cf. mergePlayerMatchRows).
//
// Bug #2/#7 : on ne fallback PAS sur l'EN dans la projection FR. Si NULL en
// DB, on renvoie chaîne vide ; HomeRepo.EnrichCanonicalAssetTranslations
// remplit ensuite Labels["fr"] depuis metadata.asset_translations.
//
// Bug #3 : projeter damage_dealt / damage_taken pour ComputeCombatYield.
var playerMatchesSharedBaseSelect = `
SELECT
    p.match_id,
    ` + StartTimeCanonicalSQL("r") + ` AS start_time,
    COALESCE(r.duration_seconds, 0)                   AS duration_seconds,
    COALESCE(r.map_id, '')                            AS map_id,
    COALESCE(r.map_name, '')                          AS map_name,
    COALESCE(r.map_name_fr, '')                       AS map_name_fr,
    COALESCE(r.playlist_id, '')                       AS playlist_id,
    COALESCE(r.playlist_name, '')                     AS playlist_name,
    COALESCE(r.playlist_name_fr, '')                  AS playlist_name_fr,
    COALESCE(r.game_variant_id, '')                   AS variant_id,
    COALESCE(r.game_variant_name, '')                 AS variant_name,
    COALESCE(r.pair_id, '')                           AS pair_id,
    COALESCE(r.pair_name, '')                         AS pair_name,
    COALESCE(r.pair_name_fr, '')                      AS pair_name_fr,
    CASE
        WHEN COALESCE(r.is_ranked, FALSE)
            OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
            OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
        THEN TRUE ELSE FALSE
    END                                                  AS is_ranked,
    COALESCE(r.is_firefight, FALSE)                   AS is_firefight,
    COALESCE(p.team_id, 0)                            AS team_id,
    p.outcome                                         AS outcome_code,
    COALESCE(p.kills, 0)                              AS kills,
    COALESCE(p.deaths, 0)                             AS deaths,
    COALESCE(p.assists, 0)                            AS assists,
    p.kda,
    COALESCE(p.headshot_kills, 0)                     AS headshot_kills,
    p.accuracy,
    COALESCE(p.time_played_seconds, 0)                AS time_played_seconds,
    p.avg_life_seconds,
    p.damage_dealt,
    p.damage_taken,
    p.team_mmr,
    p.enemy_mmr,
    COALESCE(r.team_0_score, -1)                      AS team_0_score,
    COALESCE(r.team_1_score, -1)                      AS team_1_score,
    p.max_killing_spree,
    p.personal_score,
    p.rank                                               AS rank_in_match,
    p.grenade_kills,
    p.melee_kills,
    p.power_weapon_kills,
    p.assassination_kills,
    p.ground_pound_kills,
    p.shoulder_bash_kills,
    p.shots_fired,
    p.shots_hit,
    COALESCE((
        SELECT SUM(me.count)
        FROM medals_earned me
        WHERE me.match_id = p.match_id
          AND me.xuid = p.xuid
          AND /*__PERFECT_KILL_IN__*/
    ), 0)::INTEGER                                       AS perfect_kills,
    -- T0 offset (Match Timeline T0, Phase 3) : countdown pré-match en ms.
    -- NULL si real_start_time absent → fallback runtime T0=0.
    CASE
        WHEN r.real_start_time IS NOT NULL THEN
            epoch_ms(r.real_start_time AT TIME ZONE 'UTC')
            - epoch_ms(` + StartTimeCanonicalSQL("r") + `)
    END                                                  AS t0_ms
FROM match_participants p
JOIN v_match_full r ON r.match_id = p.match_id
WHERE p.xuid = ?`

// loadSharedRows exécute l'étape 1 du split (query shared) et retourne les
// playerMatchScanResult partiellement remplis (cols shared seulement).
// LobbySizesAtCompletion compte, par match_id, les participants présents à la fin
// (present_at_completion = TRUE, bots inclus) sur la DB partagée. Sert à dimensionner
// l'axe du breakdown de placements : taille de lobby résistante au churn (départs/arrivées),
// contrairement à un COUNT(DISTINCT) brut sur match_participants. Best-effort : un match
// sans flag present_at_completion peuplé (data legacy) ressort à 0 et est ignoré côté front.
