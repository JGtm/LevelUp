// Package duckdb — MatchViewRepo : données pour la vue détail d'un match.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite"
)

// MatchViewRepo implémente port.MatchViewRepository.
type MatchViewRepo struct {
	pdb  *PlayerDB
	xuid string
}

// NewMatchViewRepo crée un MatchViewRepo.
func NewMatchViewRepo(pdb *PlayerDB, xuid string) *MatchViewRepo {
	return &MatchViewRepo{pdb: pdb, xuid: xuid}
}

// GetMatchMeta retourne les métadonnées du match (Q13).
func (r *MatchViewRepo) GetMatchMeta(ctx context.Context, matchID string) (*domain.MatchMetaRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var row domain.MatchMetaRaw
	err := r.pdb.ReadDB().QueryRow(ctx, Q13MatchMeta, matchID).Scan(
		&row.MatchID,
		&row.StartTime,
		&row.DurationSeconds,
		&row.MapName,
		&row.PairName,
		&row.PlaylistName,
		&row.IsFirefight,
		&row.IsRanked,
		&row.PlayableDurationSeconds,
		&row.MapAssetID,
		&row.GameVariantName,
		&row.PlaylistAssetID,
		&row.Team0Score,
		&row.Team1Score,
	)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchMeta: %w", err)
	}
	if row.MapAssetID != nil {
		row.MapNameFR = r.lookupMapNameFR(ctx, *row.MapAssetID)
	}
	if row.PairName != nil {
		if modeEN := analysis.NormalizeModeLabel(*row.PairName); modeEN != "" {
			row.ModeNameFR = r.lookupModeNameFR(ctx, modeEN)
		}
	}
	if row.PlaylistAssetID != nil {
		row.PlaylistNameFR = r.lookupPlaylistNameFR(ctx, *row.PlaylistAssetID)
	}
	return &row, nil
}

// lookupMapNameFR retourne le nom FR d'une map via asset_translations, ou nil.
func (r *MatchViewRepo) lookupMapNameFR(ctx context.Context, mapAssetID string) *string {
	if r.pdb.Metadata == nil {
		return nil
	}
	const q = `
		SELECT name FROM asset_translations
		WHERE asset_type = 'map' AND asset_id = ?
		  AND lang IN ('fr-FR', 'fr')
		  AND name IS NOT NULL AND TRIM(name) != ''
		ORDER BY CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END
		LIMIT 1`
	rows, err := r.pdb.Metadata.Query(ctx, q, mapAssetID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil && name != "" {
			return &name
		}
	}
	return nil
}

// lookupModeNameFR retourne le nom FR d'un mode via mode_name_tr, ou nil.
// modeEN doit être le label normalisé (via analysis.NormalizeModeLabel).
func (r *MatchViewRepo) lookupModeNameFR(ctx context.Context, modeEN string) *string {
	if r.pdb.Metadata == nil {
		return nil
	}
	const q = `
		SELECT name FROM mode_name_tr
		WHERE lang = 'fr' AND mode_en = ?
		LIMIT 1`
	rows, err := r.pdb.Metadata.Query(ctx, q, modeEN)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil && name != "" {
			return &name
		}
	}
	return nil
}

// lookupPlaylistNameFR retourne le nom FR d'une playlist via asset_translations, ou nil.
// Pattern identique à lookupMapNameFR (asset_type = 'playlist').
func (r *MatchViewRepo) lookupPlaylistNameFR(ctx context.Context, playlistAssetID string) *string {
	if r.pdb.Metadata == nil {
		return nil
	}
	const q = `
		SELECT name FROM asset_translations
		WHERE asset_type = 'playlist' AND asset_id = ?
		  AND lang IN ('fr-FR', 'fr')
		  AND name IS NOT NULL AND TRIM(name) != ''
		ORDER BY CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END
		LIMIT 1`
	rows, err := r.pdb.Metadata.Query(ctx, q, playlistAssetID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil && name != "" {
			return &name
		}
	}
	return nil
}

// GetPlayerMatchStats retourne les stats du joueur pour ce match (Q17).
func (r *MatchViewRepo) GetPlayerMatchStats(ctx context.Context, xuid, matchID string) (*domain.PlayerMatchStatsRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var s domain.PlayerMatchStatsRaw
	err := r.pdb.ReadDB().QueryRow(ctx, Q17PlayerMatchStats, matchID, xuid).Scan(
		&s.OutcomeCode,
		&s.TeamID,
		&s.RankInTeam,
		&s.Kills,
		&s.Deaths,
		&s.Assists,
		&s.KDA,
		&s.Accuracy,
		&s.PersonalScore,
		&s.AvgLifeSeconds,
		&s.TimePlayedSeconds,
		&s.ShotsFired,
		&s.ShotsHit,
		&s.DamageDealt,
		&s.DamageTaken,
		&s.TeamMMR,
		&s.EnemyMMR,
		&s.HeadshotKills,
		&s.MaxKillingSpree,
	)
	if err != nil {
		// Le joueur peut ne pas avoir participé → retourner une stats vide
		return &domain.PlayerMatchStatsRaw{}, nil
	}
	return &s, nil
}

// GetMatchEnrichment retourne l'enrichissement pour ce match (Q18).
func (r *MatchViewRepo) GetMatchEnrichment(ctx context.Context, matchID string) (*domain.MatchEnrichmentRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var e domain.MatchEnrichmentRaw
	err := r.pdb.ReadDB().QueryRow(ctx, Q18MatchEnrichment, matchID).Scan(
		&e.PerformanceScore,
		&e.IsWithFriends,
		&e.IsExcluded,
		&e.DominanceFlag,
	)
	if err != nil {
		// Pas d'enrichissement → retourner vide
		return &domain.MatchEnrichmentRaw{}, nil
	}
	return &e, nil
}

// GetMatchScoreboard retourne les stats de tous les joueurs (Q12).
func (r *MatchViewRepo) GetMatchScoreboard(ctx context.Context, matchID string) ([]domain.ScoreboardRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Q12 utilise 3 fois match_id : medals CTE, weapons CTE, WHERE
	rows, err := r.pdb.ReadDB().Query(ctx, Q12MatchScoreboard, matchID, matchID, matchID)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard: %w", err)
	}
	defer rows.Close()

	var results []domain.ScoreboardRaw
	for rows.Next() {
		var s domain.ScoreboardRaw
		// top_weapon_id est UBIGINT côté DuckDB → scanner en *uint64 pour
		// éviter l'overflow int64 sur les hash de filmshell (bit63=1).
		var topWeaponU *uint64
		if err := rows.Scan(
			&s.XUID,
			&s.Gamertag,
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
			&s.PerfectKills,
			&topWeaponU,
			&s.KillsExpected,
			&s.DeathsExpected,
			&s.KillsStdDev,
			&s.DeathsStdDev,
		); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard scan: %w", err)
		}
		if topWeaponU != nil {
			v := int64(*topWeaponU) //nolint:gosec
			s.TopWeaponID = &v
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Résolution des labels d'armes top-weapon pour chaque joueur du scoreboard
	var topWeaponIDs []int64
	for _, s := range results {
		if s.TopWeaponID != nil {
			topWeaponIDs = append(topWeaponIDs, *s.TopWeaponID)
		}
	}
	if len(topWeaponIDs) > 0 {
		labels := r.lookupWeaponLabels(ctx, topWeaponIDs)
		for i := range results {
			if results[i].TopWeaponID != nil {
				results[i].TopWeaponLabel = labels[*results[i].TopWeaponID]
			}
		}
	}

	return results, nil
}

// GetMatchObjectiveScore retourne la somme award_score des awards de
// catégorie 'objective' pour (xuid, matchID). Mirror du squad
// LoadObjectiveScores (cf. squad_v2_adapter.go) appliqué à un seul match.
// Dégradation silencieuse à 0 si la table est absente ou ligne vide.
func (r *MatchViewRepo) GetMatchObjectiveScore(ctx context.Context, xuid, matchID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var tableCount int
	if err := r.pdb.ReadDB().QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'personal_score_awards'
	`).Scan(&tableCount); err != nil || tableCount == 0 {
		return 0, nil
	}

	var total int
	err := r.pdb.ReadDB().QueryRow(ctx, `
		SELECT COALESCE(SUM(award_score), 0)::INTEGER
		FROM personal_score_awards
		WHERE award_category = 'objective'
		  AND xuid = ?
		  AND match_id = ?
	`, xuid, matchID).Scan(&total)
	if err != nil {
		return 0, nil
	}
	return total, nil
}

// GetMatchMedals retourne les médailles du joueur dans ce match (Q14).
func (r *MatchViewRepo) GetMatchMedals(ctx context.Context, xuid, matchID string) ([]domain.MedalRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q14MatchMedals, xuid, matchID)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchMedals: %w", err)
	}
	defer rows.Close()

	var results []domain.MedalRaw
	var medalIDs []int64
	for rows.Next() {
		var m domain.MedalRaw
		if err := rows.Scan(&m.MedalID, &m.Count); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchMedals scan: %w", err)
		}
		results = append(results, m)
		medalIDs = append(medalIDs, m.MedalID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	meta := r.lookupMedalMeta(ctx, medalIDs)
	for index := range results {
		if m, ok := meta[results[index].MedalID]; ok {
			results[index].Label = m.label
			results[index].Description = m.description
			results[index].Difficulty = m.difficulty
		} else {
			results[index].Label = strconv.FormatInt(results[index].MedalID, 10)
		}
	}
	return results, nil
}

// GetMatchEvents retourne les events highlight du match (Q21).
func (r *MatchViewRepo) GetMatchEvents(ctx context.Context, matchID string) ([]domain.EventRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q21MatchEventsWithXUID, matchID)
	if err != nil {
		// La table peut être absente sur certains matchs → retourner vide
		return nil, nil
	}
	defer rows.Close()

	var results []domain.EventRaw
	for rows.Next() {
		var e domain.EventRaw
		if err := rows.Scan(&e.EventType, &e.TimeMS, &e.XUID); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchEvents scan: %w", err)
		}
		results = append(results, e)
	}
	return results, rows.Err()
}

// GetMatchWeaponKills retourne les kills par arme du joueur (Q16).
// Applique la fusion variante→canonique (Duelist Energy Sword → Energy Sword,
// M392 Bandit → Bandit Evo, etc.) avant le lookup pour regrouper les skins
// — comportement aligné sur la Python resolve_weapon_display.
func (r *MatchViewRepo) GetMatchWeaponKills(ctx context.Context, xuid, matchID string) ([]domain.WeaponKillRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q16WeaponKills, xuid, matchID)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	// Étape 1 : scan + fusion variant→canonique + regroupement par ID canonique.
	killsByID := make(map[int64]int)
	orderedIDs := make([]int64, 0, 16)
	for rows.Next() {
		var widU uint64
		var kills int
		if err := rows.Scan(&widU, &kills); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchWeaponKills scan: %w", err)
		}
		canonicalU := widU
		if canon, ok := analysis.WeaponFusionMapID[widU]; ok {
			canonicalU = canon
		}
		canonicalID := int64(canonicalU) //nolint:gosec
		if _, seen := killsByID[canonicalID]; !seen {
			orderedIDs = append(orderedIDs, canonicalID)
		}
		killsByID[canonicalID] += kills
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Étape 2 : assembler trié par kills DESC (Q16 est déjà ORDER BY kills DESC,
	// mais la fusion peut réordonner — re-trier garde le contrat).
	results := make([]domain.WeaponKillRaw, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		results = append(results, domain.WeaponKillRaw{WeaponID: id, Kills: killsByID[id]})
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].Kills > results[j].Kills })

	// Étape 3 : résolution labels.
	weaponIDs := make([]int64, 0, len(results))
	for _, w := range results {
		weaponIDs = append(weaponIDs, w.WeaponID)
	}
	labels := r.lookupWeaponLabels(ctx, weaponIDs)
	for index := range results {
		if label, ok := labels[results[index].WeaponID]; ok {
			results[index].WeaponLabel = label
			continue
		}
		results[index].WeaponLabel = strconv.FormatInt(results[index].WeaponID, 10)
	}
	return results, nil
}

type medalMeta struct {
	label       string
	description string
	difficulty  string
}

// lookupMedalMeta résout label + description + difficulty depuis medal_definitions
// (chaîne BCP-47 medal_translations fr-FR/en-US > medal_definitions name_fr/name_en).
// Fallback citation_mappings.citation_name_display si la médaille n'est pas dans medal_definitions.
func (r *MatchViewRepo) lookupMedalMeta(ctx context.Context, medalIDs []int64) map[int64]medalMeta {
	result := make(map[int64]medalMeta, len(medalIDs))
	if len(medalIDs) == 0 || r.pdb.Metadata == nil {
		return result
	}
	q, args, ok := buildLookupQuery(
		`SELECT md.medal_name_id,
		        COALESCE(
		            NULLIF(TRIM(mt_fr.name),''),
		            NULLIF(TRIM(md.name_fr),''),
		            NULLIF(TRIM(mt_en.name),''),
		            NULLIF(TRIM(md.name_en),'')
		        ) AS label,
		        COALESCE(
		            NULLIF(TRIM(md.description_fr),''),
		            NULLIF(TRIM(md.description_en),''),
		            ''
		        ) AS description,
		        COALESCE(NULLIF(TRIM(md.difficulty),''), 'Normal') AS difficulty
		 FROM medal_definitions md
		 LEFT JOIN medal_translations mt_fr
		     ON mt_fr.medal_name_id = md.medal_name_id AND mt_fr.lang = 'fr-FR'
		 LEFT JOIN medal_translations mt_en
		     ON mt_en.medal_name_id = md.medal_name_id AND mt_en.lang = 'en-US'
		 WHERE md.medal_name_id IN (%s)`,
		medalIDs,
	)
	if !ok {
		return result
	}
	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var label, desc, diff string
		if err := rows.Scan(&id, &label, &desc, &diff); err == nil && label != "" {
			result[id] = medalMeta{label: label, description: desc, difficulty: diff}
		}
	}
	// Fallback citation_mappings pour les IDs absents de medal_definitions.
	missing := make([]int64, 0)
	for _, id := range medalIDs {
		if _, ok := result[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		fb := lookupLabelsByID(
			ctx,
			r.pdb.Metadata,
			`SELECT medal_id, citation_name_display
			 FROM citation_mappings
			 WHERE medal_id IN (%s)
			   AND citation_name_display IS NOT NULL
			   AND citation_name_display <> ''`,
			missing,
		)
		for id, label := range fb {
			result[id] = medalMeta{label: label, difficulty: "Normal"}
		}
	}
	return result
}

// lookupMedalLabels est conservé pour les usages internes (bulk scoreboard).
func (r *MatchViewRepo) lookupMedalLabels(ctx context.Context, medalIDs []int64) map[int64]string {
	return lookupLabelsByID(
		ctx,
		r.pdb.Metadata,
		`SELECT medal_id, citation_name_display
		 FROM citation_mappings
		 WHERE medal_id IN (%s)
		   AND citation_name_display IS NOT NULL
		   AND citation_name_display <> ''`,
		medalIDs,
	)
}

func (r *MatchViewRepo) lookupWeaponLabels(ctx context.Context, weaponIDs []int64) map[int64]string {
	labels := map[int64]string{}
	if len(weaponIDs) == 0 || r.pdb.Metadata == nil {
		return labels
	}
	// Contournement driver : database/sql ne supporte pas uint64 avec bit63=1.
	// On injecte les IDs comme littéraux décimaux (valeurs internes, pas user input).
	unique := uniqueInt64s(weaponIDs)
	parts := make([]string, len(unique))
	for i, id := range unique {
		parts[i] = fmt.Sprintf("%d", uint64(id)) //nolint:gosec
	}
	query := fmt.Sprintf( //nolint:gosec
		`SELECT weapon_id, COALESCE(name_fr, name_en, CAST(weapon_id AS VARCHAR)) AS weapon_label
		 FROM weapon_labels
		 WHERE weapon_id IN (%s)`,
		strings.Join(parts, ","),
	)
	rows, err := r.pdb.Metadata.Query(ctx, query)
	if err != nil {
		return labels
	}
	defer rows.Close()
	for rows.Next() {
		// weapon_id UBIGINT scanné via UBigint (cf. ubigint_scanner.go) — sinon
		// overflow silencieux pour les hash filmshell bit63=1 (Mutilateur,
		// MK50 Sidekick, Fuel Rod SPNKr…).
		var id UBigint
		var label string
		if err := rows.Scan(&id, &label); err == nil && label != "" {
			labels[id.Int64()] = label
		}
	}
	return labels
}

func lookupLabelsByID(ctx context.Context, db *DB, queryTemplate string, ids []int64) map[int64]string {
	labels := map[int64]string{}
	query, args, ok := buildLookupQuery(queryTemplate, ids)
	if !ok || db == nil {
		return labels
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return labels
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var label string
		if err := rows.Scan(&id, &label); err == nil && label != "" {
			labels[id] = label
		}
	}
	return labels
}

func buildLookupQuery(queryTemplate string, ids []int64) (string, []interface{}, bool) {
	uniqueIDs := uniqueInt64s(ids)
	if len(uniqueIDs) == 0 {
		return "", nil, false
	}

	placeholders := make([]string, 0, len(uniqueIDs))
	args := make([]interface{}, 0, len(uniqueIDs))
	for _, id := range uniqueIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	return fmt.Sprintf(queryTemplate, strings.Join(placeholders, ",")), args, true
}

func uniqueInt64s(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

// GetMatchKVPairs retourne les paires killer→victim du match (Q20).
func (r *MatchViewRepo) GetMatchKVPairs(ctx context.Context, matchID string) ([]domain.KVPairRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q20KVPairs, matchID)
	if err != nil {
		// Vue v_killer_victim_full peut être absente dans certaines DBs → vide
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

// GetMatchNeighbors retourne les matchs précédent/suivant pour la navigation (Q25).
func (r *MatchViewRepo) GetMatchNeighbors(ctx context.Context, xuid, matchID string) (*domain.MatchNeighbors, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	row := r.pdb.ReadDB().QueryRow(ctx, Q25NeighborMatches, xuid, matchID)
	var nextID, prevID *string
	var currentIdx, total int
	if err := row.Scan(&nextID, &prevID, &currentIdx, &total); err != nil {
		// Match introuvable dans le scope → voisins vides
		return &domain.MatchNeighbors{TotalMatches: 0}, nil
	}
	return &domain.MatchNeighbors{
		PreviousMatchID: prevID,
		NextMatchID:     nextID,
		CurrentIndex:    currentIdx,
		TotalMatches:    total,
	}, nil
}

// GetMatchNeighborsFiltered : variante paramétrable Phase 2b. spec=nil ou
// vide → délègue à GetMatchNeighbors (chronologie globale).
//
// Le fragment SQL est produit par analysis.BuildNeighborsWhereClause avec
// halo_infinite.PairNamePrefixesForCategory injecté. Pour les titres futurs
// sans la notion ModeCategory, l'adapter dégradera silencieusement.
func (r *MatchViewRepo) GetMatchNeighborsFiltered(
	ctx context.Context,
	xuid, matchID string,
	spec *domain.MatchFilterSpec,
) (*domain.MatchNeighbors, error) {
	if spec.IsEmpty() {
		return r.GetMatchNeighbors(ctx, xuid, matchID)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	clauseRes := analysis.BuildNeighborsWhereClause(spec, halo_infinite.PairNamePrefixesForCategory)

	if len(clauseRes.IgnoredFilters) > 0 {
		slog.WarnContext(ctx, "neighbors: filters ignored",
			"match_id", matchID, "ignored", clauseRes.IgnoredFilters)
	}

	query := strings.Replace(Q25NeighborMatchesTemplate, "/*EXTRA_WHERE*/", clauseRes.SQL, 1)
	args := make([]any, 0, len(clauseRes.Args)+2)
	args = append(args, xuid)
	args = append(args, clauseRes.Args...)
	args = append(args, matchID)

	row := r.pdb.ReadDB().QueryRow(ctx, query, args...)
	var nextID, prevID *string
	var currentIdx, total int
	if err := row.Scan(&nextID, &prevID, &currentIdx, &total); err != nil {
		// Match hors scope filtré → voisins vides (cas normal : utilisateur
		// arrivé sur un match qui ne matche plus les filtres).
		return &domain.MatchNeighbors{TotalMatches: 0}, nil
	}
	return &domain.MatchNeighbors{
		PreviousMatchID: prevID,
		NextMatchID:     nextID,
		CurrentIndex:    currentIdx,
		TotalMatches:    total,
	}, nil
}

// GetMatchSkillRank retourne le rang compétitif pour ce match (Q22).
// Utilise la player DB (match_skill_rank).
func (r *MatchViewRepo) GetMatchSkillRank(ctx context.Context, matchID string) (*domain.SkillRankRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var row domain.SkillRankRaw
	err := r.pdb.ReadDB().QueryRow(ctx, Q22MatchSkillRank, matchID).Scan(
		&row.RatingType,
		&row.TierLabel,
		&row.RatingValue,
		&row.RatingDelta,
		&row.PlaylistGroup,
		&row.Tier,
		&row.SubTier,
	)
	if err != nil {
		// Absent pour les matchs non-ranked ou sans donnée skill → nil sans erreur
		return nil, nil //nolint:nilerr
	}
	return &row, nil
}

// GetMatchEncounters retourne l'historique de rencontres avec les participants (Q23).
// Utilise la player DB (avec shared. attaché).
func (r *MatchViewRepo) GetMatchEncounters(ctx context.Context, matchID, myXUID string) ([]domain.EncounterRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q23MatchEncounters,
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
		if err := rows.Scan(&enc.XUID, &enc.Gamertag, &enc.CountTogether, &enc.IsAlly); err != nil {
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

	rows, err := r.pdb.ReadDB().Query(ctx, Q23bMatchEncounterStats,
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
		var s domain.EncounterStatsRaw
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
		); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounterStats scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetMatchMedia retourne les médias associés au match (Q24).
// Utilise shared_social DB.
func (r *MatchViewRepo) GetMatchMedia(ctx context.Context, matchID, playerSlug string) ([]domain.MediaAssocRaw, error) {
	if r.pdb.SharedSocial == nil {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.SharedSocial.Query(ctx, Q24MatchMedia, matchID, playerSlug)
	if err != nil {
		// Absence de la table ou DB non configurée → résultat vide sans erreur bloquante
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
// Utilise la player DB (avec shared. attaché).
func (r *MatchViewRepo) GetMatchExpectedStats(ctx context.Context, matchID, xuid string) (*domain.ExpectedStatsRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var row domain.ExpectedStatsRaw
	err := r.pdb.ReadDB().QueryRow(ctx, Q26MatchExpectedStats, matchID, xuid).Scan(
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

// GetMatchBulkMedals retourne les médailles de tous les joueurs du match (Q27).
func (r *MatchViewRepo) GetMatchBulkMedals(ctx context.Context, matchID string) ([]domain.BulkMedalRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q27BulkMedals, matchID)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var results []domain.BulkMedalRaw
	var medalIDs []int64
	for rows.Next() {
		var m domain.BulkMedalRaw
		if err := rows.Scan(&m.XUID, &m.MedalID, &m.Count); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchBulkMedals scan: %w", err)
		}
		results = append(results, m)
		medalIDs = append(medalIDs, m.MedalID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	labels := r.lookupMedalLabels(ctx, medalIDs)
	for i := range results {
		if label, ok := labels[results[i].MedalID]; ok {
			results[i].Label = label
		} else {
			results[i].Label = strconv.FormatInt(results[i].MedalID, 10)
		}
	}
	return results, nil
}

// GetHistoryForAvg retourne les 50 derniers matchs du joueur (Q29) pour le
// calcul des moyennes historiques K/D/A + spree/headshots/perfect.
func (r *MatchViewRepo) GetHistoryForAvg(ctx context.Context, xuid string) ([]domain.MatchHistAvgRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q29HistoryForAvg, xuid, xuid)
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
		); err != nil {
			return nil, fmt.Errorf("GetHistoryForAvg scan: %w", err)
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// GetMatchBulkWeaponKills retourne les kills par arme de tous les joueurs (Q28).
// Applique la fusion variante→canonique par xuid (regroupe Duelist Energy Sword
// + Elite Bloodblade + Energy Sword sous le même canonique pour chaque joueur).
func (r *MatchViewRepo) GetMatchBulkWeaponKills(ctx context.Context, matchID string) ([]domain.BulkWeaponKillRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q28BulkWeaponKills, matchID)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	type key struct {
		xuid string
		wid  int64
	}
	killsByKey := make(map[key]int)
	ordered := make([]key, 0, 32)
	for rows.Next() {
		var xuid string
		var widU uint64
		var kills int
		if err := rows.Scan(&xuid, &widU, &kills); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchBulkWeaponKills scan: %w", err)
		}
		canonicalU := widU
		if canon, ok := analysis.WeaponFusionMapID[widU]; ok {
			canonicalU = canon
		}
		k := key{xuid: xuid, wid: int64(canonicalU)} //nolint:gosec
		if _, seen := killsByKey[k]; !seen {
			ordered = append(ordered, k)
		}
		killsByKey[k] += kills
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	results := make([]domain.BulkWeaponKillRaw, 0, len(ordered))
	weaponIDs := make([]int64, 0, len(ordered))
	for _, k := range ordered {
		results = append(results, domain.BulkWeaponKillRaw{
			XUID:     k.xuid,
			WeaponID: k.wid,
			Kills:    killsByKey[k],
		})
		weaponIDs = append(weaponIDs, k.wid)
	}

	labels := r.lookupWeaponLabels(ctx, weaponIDs)
	for i := range results {
		if label, ok := labels[results[i].WeaponID]; ok {
			results[i].WeaponLabel = label
			continue
		}
		// Fallback : weapon_id en string pour les variantes absentes de
		// metadata.weapon_labels (cohérent avec GetMatchWeaponKills L428).
		// Évite que le frontend ait à gérer un weapon_label vide (`??` ne
		// fallback pas sur "").
		results[i].WeaponLabel = strconv.FormatInt(results[i].WeaponID, 10)
	}
	return results, nil
}
