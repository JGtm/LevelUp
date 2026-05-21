// Package duckdb — MatchViewRepo : données pour la vue détail d'un match.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
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
// Exécutée sur SharedReader (ADR 0016) — Q13 lit match_registry (shared-only).
func (r *MatchViewRepo) GetMatchMeta(ctx context.Context, matchID string) (*domain.MatchMetaRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchMeta: shared reader: %w", err)
	}
	defer release()

	var row domain.MatchMetaRaw
	err = sharedDB.QueryRowContext(ctx, Q13MatchMeta, matchID).Scan(
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
		&row.PairNameFR,
		&row.PairAssetID,
		&row.GameVariantAssetID,
	)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchMeta: %w", err)
	}
	// Résolution unifiée des noms d'asset via MetadataRepo.ResolveAssetName.
	// Cascade FR-FR → fr → en-US → en (PreferredLangsForLocale("fr")), une seule
	// requête SQL par asset, source unique de vérité (asset_translations).
	if row.MapAssetID != nil {
		row.MapNameFR = r.resolveAssetName(ctx, "map", *row.MapAssetID)
		row.MapNameEN = r.resolveAssetNameEN(ctx, "map", *row.MapAssetID)
		row.MapImageURL = r.lookupMapImageURL(ctx, *row.MapAssetID)
	}
	if row.PlaylistAssetID != nil {
		row.PlaylistNameFR = r.resolveAssetName(ctx, "playlist", *row.PlaylistAssetID)
	}
	// Résolution du libellé de mode — même cascade que applyMatchHistoryFRTranslations :
	// ResolveAssetNamesBulk(pair) → loadModeFRBatch(mode_name_tr) → ResolvePairNameFR.
	// pair_name_fr est toujours NULL en DB (non écrit par sync) : seul ce chemin
	// produit "Capture du drapeau" au lieu de l'EN normalisé.
	{
		var pairAssetName string
		if r.pdb.Metadata != nil && row.PairAssetID != nil {
			metaRepo := NewMetadataRepoFromDB(r.pdb.Metadata)
			langs := PreferredLangsForLocale("fr")
			pairNames, _ := metaRepo.ResolveAssetNamesBulk(ctx, "pair", []string{*row.PairAssetID}, langs)
			pairAssetName = strings.TrimSpace(pairNames[*row.PairAssetID])
		}

		rawPairName := derefString(row.PairName)
		modeENSet := make(map[string]struct{})
		if en := analysis.NormalizeModeLabel(rawPairName); en != "" {
			modeENSet[en] = struct{}{}
		}
		if en := analysis.NormalizeModeLabel(pairAssetName); en != "" {
			modeENSet[en] = struct{}{}
		}
		modeFR := loadModeFRBatch(ctx, r.pdb, modeENSet)

		if fr := analysis.ResolvePairNameFR(rawPairName, derefString(row.PairNameFR), pairAssetName, modeFR); fr != "" {
			row.ModeNameFR = &fr
		}
	}
	return &row, nil
}

// resolveAssetName est un helper qui appelle MetadataRepo.ResolveAssetName avec
// les préférences locale=FR par défaut. Retourne nil si la métadonnée DB n'est
// pas disponible ou si l'asset n'a aucune traduction.
func (r *MatchViewRepo) resolveAssetName(ctx context.Context, assetType, assetID string) *string {
	if r.pdb.Metadata == nil {
		return nil
	}
	meta := NewMetadataRepoFromDB(r.pdb.Metadata)
	name, _, ok, err := meta.ResolveAssetName(ctx, assetType, assetID, PreferredLangsForLocale("fr"))
	if err != nil || !ok || strings.TrimSpace(name) == "" {
		return nil
	}
	return &name
}

// resolveAssetNameEN retourne le nom EN canonique d'un asset (sans cascade
// FR) — utilisé pour les lookups d'image qui dépendent du nom EN canonique
// (l'adapter `AssetURLAdapter` indexe `static/maps/{titleSlug}/{name}.{ext}`
// par nom EN, pas par nom localisé).
func (r *MatchViewRepo) resolveAssetNameEN(ctx context.Context, assetType, assetID string) *string {
	if r.pdb.Metadata == nil {
		return nil
	}
	meta := NewMetadataRepoFromDB(r.pdb.Metadata)
	name, _, ok, err := meta.ResolveAssetName(ctx, assetType, assetID, PreferredLangsForLocale("en"))
	if err != nil || !ok || strings.TrimSpace(name) == "" {
		return nil
	}
	return &name
}

// lookupMapImageURL retourne l'URL de l'image de map depuis map_images_registry
// par map_id (UUID stable). Pattern identique à home_repo.loadHomeMapImageURLs.
// Nil si map_id absent du registry ou si metadata.duckdb indisponible.
func (r *MatchViewRepo) lookupMapImageURL(ctx context.Context, mapAssetID string) *string {
	if r.pdb.Metadata == nil {
		return nil
	}
	const q = `
		SELECT local_path FROM map_images_registry
		WHERE title_id = ? AND map_id = ?
		  AND TRIM(local_path) != ''
		LIMIT 1`
	rows, err := r.pdb.Metadata.Query(ctx, q, halo_infinite.TitleSlug, mapAssetID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		var path string
		if err := rows.Scan(&path); err == nil && path != "" {
			return &path
		}
	}
	return nil
}

// GetPlayerMatchStats retourne les stats du joueur pour ce match (Q17).
// Exécutée sur SharedReader (ADR 0016) — Q17 lit match_participants (shared-only).
func (r *MatchViewRepo) GetPlayerMatchStats(ctx context.Context, xuid, matchID string) (*domain.PlayerMatchStatsRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return &domain.PlayerMatchStatsRaw{}, nil //nolint:nilerr
	}
	defer release()

	var s domain.PlayerMatchStatsRaw
	err = sharedDB.QueryRowContext(ctx, Q17PlayerMatchStats, matchID, xuid).Scan(
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
// Exécutée sur SharedReader (ADR 0016) — Q12 lit medals_earned + weapon_kills
// + match_participants + v_gamertag_lookup (shared-only).
func (r *MatchViewRepo) GetMatchScoreboard(ctx context.Context, matchID string) ([]domain.ScoreboardRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard: shared reader: %w", err)
	}
	defer release()

	// Q12 utilise 3 fois match_id : medals CTE, weapons CTE, WHERE
	rows, err := sharedDB.QueryContext(ctx, Q12MatchScoreboard, matchID, matchID, matchID)
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
// Exécutée sur SharedReader (ADR 0016) — Q14 lit medals_earned (shared-only).
func (r *MatchViewRepo) GetMatchMedals(ctx context.Context, xuid, matchID string) ([]domain.MedalRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchMedals: shared reader: %w", err)
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q14MatchMedals, xuid, matchID)
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
// Exécutée sur SharedReader (ADR 0016, shared-only).
func (r *MatchViewRepo) GetMatchEvents(ctx context.Context, matchID string) ([]domain.EventRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
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

// GetMatchWeaponKills retourne les kills par arme du joueur (Q16).
// Applique la fusion variante→canonique (Duelist Energy Sword → Energy Sword,
// M392 Bandit → Bandit Evo, etc.) avant le lookup pour regrouper les skins
// — comportement aligné sur la Python resolve_weapon_display.
func (r *MatchViewRepo) GetMatchWeaponKills(ctx context.Context, xuid, matchID string) ([]domain.WeaponKillRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Q16 lit v_weapon_kills (shared-only) — via SharedReader (ADR 0016).
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q16WeaponKills, xuid, matchID)
	if err != nil {
		return nil, nil //nolint:nilerr
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

type weaponMetaEntry struct {
	label  string
	nameEN string
}

// lookupWeaponMeta résout label (FR>EN) + name_en depuis weapon_labels.
// name_en est nécessaire pour construire l'URL image via AssetURLAdapter.WeaponImageURL.
func (r *MatchViewRepo) lookupWeaponMeta(ctx context.Context, weaponIDs []int64) map[int64]weaponMetaEntry {
	result := map[int64]weaponMetaEntry{}
	if len(weaponIDs) == 0 || r.pdb.Metadata == nil {
		return result
	}
	unique := uniqueInt64s(weaponIDs)
	parts := make([]string, len(unique))
	for i, id := range unique {
		parts[i] = fmt.Sprintf("%d", uint64(id)) //nolint:gosec
	}
	query := fmt.Sprintf( //nolint:gosec
		`SELECT weapon_id,
		        COALESCE(name_fr, name_en, CAST(weapon_id AS VARCHAR)) AS label,
		        COALESCE(name_en, '') AS name_en
		 FROM weapon_labels
		 WHERE weapon_id IN (%s)`,
		strings.Join(parts, ","),
	)
	rows, err := r.pdb.Metadata.Query(ctx, query)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id UBigint
		var label, nameEN string
		if err := rows.Scan(&id, &label, &nameEN); err == nil && label != "" {
			result[id.Int64()] = weaponMetaEntry{label: label, nameEN: nameEN}
		}
	}
	return result
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
// Exécutée sur SharedReader (ADR 0016, shared-only).
func (r *MatchViewRepo) GetMatchKVPairs(ctx context.Context, matchID string) ([]domain.KVPairRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, Q20KVPairs, matchID)
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
// Exécutée sur SharedReader (ADR 0016).
func (r *MatchViewRepo) GetMatchNeighbors(ctx context.Context, xuid, matchID string) (*domain.MatchNeighbors, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return &domain.MatchNeighbors{TotalMatches: 0}, nil
	}
	defer release()
	row := sharedDB.QueryRowContext(ctx, Q25NeighborMatches, xuid, matchID)
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

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return &domain.MatchNeighbors{TotalMatches: 0}, nil
	}
	defer release()
	row := sharedDB.QueryRowContext(ctx, query, args...)
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
//
// Cross-DB split (ADR 0016) : on lit d'abord match_skill_rank sur la conn
// Player (Q22a — rating_type_raw + tier/value/delta), puis on enrichit
// rating_type via match_registry sur SharedReader (Q22b — is_ranked +
// playlist_name + pair_name). Le calcul CASE/STRPOS qui était inline dans
// Q22 est réimplémenté en Go pour respecter la séparation des connexions.
func (r *MatchViewRepo) GetMatchSkillRank(ctx context.Context, matchID string) (*domain.SkillRankRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Phase A — player.match_skill_rank.
	var (
		row           domain.SkillRankRaw
		ratingTypeRaw string
	)
	err := r.pdb.ReadDB().QueryRow(ctx, Q22aMatchSkillRankPlayer, matchID).Scan(
		&ratingTypeRaw,
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

	// Phase B — shared.match_registry (best-effort : si la lecture échoue
	// ou si le match n'est pas dans le registry, on retombe sur ratingTypeRaw).
	row.RatingType = resolveMatchRatingType(ctx, r.pdb.SharedReadDB(), matchID, ratingTypeRaw)
	return &row, nil
}

// resolveMatchRatingType reproduit le CASE/STRPOS inline de l'ancien Q22 :
//   - si match_registry présent : CSR si is_ranked OU playlist_name/pair_name
//     contient "ranked" (case-insensitive), sinon LUSR.
//   - sinon : fallback sur le rating_type_raw stocké dans match_skill_rank
//     ("CSR" si égal après TRIM/UPPER, sinon LUSR).
func resolveMatchRatingType(ctx context.Context, sr SharedReader, matchID, ratingTypeRaw string) string {
	if sr != nil {
		db, release, err := sr.Get(ctx)
		if err == nil {
			defer release()
			var (
				isRanked     bool
				playlistName string
				pairName     string
			)
			scanErr := db.QueryRowContext(ctx, Q22bMatchRegistryRankedFlag, matchID).
				Scan(&isRanked, &playlistName, &pairName)
			if scanErr == nil {
				if isRanked ||
					strings.Contains(strings.ToLower(playlistName), "ranked") ||
					strings.Contains(strings.ToLower(pairName), "ranked") {
					return "CSR"
				}
				return "LUSR"
			}
		}
	}
	if strings.EqualFold(strings.TrimSpace(ratingTypeRaw), "CSR") {
		return "CSR"
	}
	return "LUSR"
}

// GetMatchEncounters retourne l'historique de rencontres avec les participants (Q23).
// Exécutée sur SharedReader (ADR 0016) — Q23 lit match_participants +
// v_gamertag_lookup (shared-only).
func (r *MatchViewRepo) GetMatchEncounters(ctx context.Context, matchID, myXUID string) ([]domain.EncounterRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounters: shared reader: %w", err)
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q23MatchEncounters,
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
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounterStats: shared reader: %w", err)
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q23bMatchEncounterStats,
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
			s          domain.EncounterStatsRaw
			lastSeenAt sql.NullTime
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
			&lastSeenAt,
		); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounterStats scan: %w", err)
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
func (r *MatchViewRepo) GetMatchMedia(ctx context.Context, matchID string) ([]domain.MediaAssocRaw, error) {
	if r.pdb.SharedSocial == nil {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.SharedSocial.Query(ctx, Q24MatchMedia, matchID)
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

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
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

// GetMatchBulkMedals retourne les médailles de tous les joueurs du match (Q27).
// Exécutée sur SharedReader (ADR 0016) — Q27 lit medals_earned (shared-only).
func (r *MatchViewRepo) GetMatchBulkMedals(ctx context.Context, matchID string) ([]domain.BulkMedalRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q27BulkMedals, matchID)
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

	metas := r.lookupMedalMeta(ctx, medalIDs)
	for i := range results {
		if m, ok := metas[results[i].MedalID]; ok {
			results[i].Label = m.label
			results[i].Difficulty = m.difficulty
		} else {
			results[i].Label = strconv.FormatInt(results[i].MedalID, 10)
		}
	}
	return results, nil
}

// GetHistoryForAvg retourne les 50 derniers matchs du joueur (Q29) pour le
// calcul des moyennes historiques K/D/A + spree/headshots/perfect.
// Exécutée sur SharedReader (ADR 0016) — Q29 lit match_participants +
// match_registry + medals_earned (shared-only).
func (r *MatchViewRepo) GetHistoryForAvg(ctx context.Context, xuid string) ([]domain.MatchHistAvgRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "GetHistoryForAvg shared reader failed", "err", err)
		return nil, nil //nolint:nilerr
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q29HistoryForAvg, xuid, xuid)
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
// Exécutée sur SharedReader (ADR 0016) — Q28 lit weapon_kills (shared-only).
func (r *MatchViewRepo) GetMatchBulkWeaponKills(ctx context.Context, matchID string) ([]domain.BulkWeaponKillRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q28BulkWeaponKills, matchID)
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

	weapMeta := r.lookupWeaponMeta(ctx, weaponIDs)
	for i := range results {
		if m, ok := weapMeta[results[i].WeaponID]; ok {
			results[i].WeaponLabel = m.label
			results[i].NameEN = m.nameEN
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
	err := r.pdb.ReadDB().QueryRow(ctx, q, gameVariantName).Scan(
		&m.GameVariantName,
		&m.Intercept, &m.CoefKills, &m.CoefDeaths,
		&m.CoefDamageDealt, &m.CoefDamageTaken, &m.CoefMMRDelta,
		&m.R2, &m.N,
	)
	if err != nil {
		return nil, nil //nolint:nilerr — table absente ou mode inconnu : dégradation gracieuse
	}
	return &m, nil
}

// sanitizeF64 remplace les valeurs NaN/Inf par nil. json.Marshal rejette NaN
// et +/-Inf (non représentables en JSON), ce qui provoque un corps HTTP vide
// quand writeJSON ignorait silencieusement l'erreur.
func sanitizeF64(f *float64) *float64 {
	if f == nil || math.IsNaN(*f) || math.IsInf(*f, 0) {
		return nil
	}
	return f
}
