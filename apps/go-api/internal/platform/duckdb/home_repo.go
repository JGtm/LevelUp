// Package duckdb â€” home_repo.go : accÃ¨s DB pour la page d'accueil Mission Control.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/legacymatch"
)

const homeStaticTitleSlug = "halo_infinite"

const homeIdentityAssetBasePath = "/api/v1/assets/spartan"

// HomeRepo fournit les donnÃ©es de la page d'accueil depuis DuckDB.
type HomeRepo struct {
	pdb *PlayerDB
}

// NewHomeRepo crÃ©e un HomeRepo pour un joueur.
func NewHomeRepo(pdb *PlayerDB) *HomeRepo {
	return &HomeRepo{pdb: pdb}
}

// LoadHomeMatches charge tous les matchs du joueur (Q26).
func (r *HomeRepo) LoadHomeMatches(ctx context.Context) ([]legacymatch.HomeMatchRow, error) {
	rows, err := r.pdb.ReadDB().Query(ctx, Q26HomeMatches, r.pdb.XUID, r.pdb.XUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []legacymatch.HomeMatchRow
	for rows.Next() {
		var row legacymatch.HomeMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapID,
			&row.MapName,
			&row.MapNameFR,
			&row.PairID,
			&row.PairName,
			&row.PairNameFR,
			&row.GameVariantID,
			&row.GameVariantName,
			&row.PlaylistID,
			&row.PlaylistName,
			&row.PlaylistNameFR,
			&row.IsFirefight,
			&row.IsRanked,
			&row.SessionLabel,
			&row.IsWithFriends,
			&row.Outcome,
			&row.TeamID,
			&row.Team0Score,
			&row.Team1Score,
			&row.DominanceFlag,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.KDA,
			&row.Ratio,
			&row.Accuracy,
			&row.AvgLifeSeconds,
			&row.TimePlayedSecs,
			&row.DamageDealt,
			&row.DamageTaken,
			&row.TeamMMR,
			&row.EnemyMMR,
			&row.PerformanceScore,
			&row.SkillRatingValue,
			&row.SkillRatingType,
			&row.SkillTier,
			&row.SkillSubTier,
			&row.SkillTierLabel,
			&row.SkillRatingDelta,
			&row.SkillPlaylistGroup,
			&row.RankInTeam,
			&row.HeadshotKills,
			&row.PerfectKills,
			&row.MaxKillingSpree,
		); err != nil {
			return nil, err
		}
		tier := ""
		if row.SkillTier != nil {
			tier = *row.SkillTier
		}
		tierLabel := ""
		if row.SkillTierLabel != nil {
			tierLabel = *row.SkillTierLabel
		}
		row.SkillRankImageURL = buildHomeSkillPeakBadgeURL(tier, tierLabel, row.SkillSubTier, homeStaticTitleSlug)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	r.enrichHomeMatchTranslations(ctx, result)
	return result, nil
}

// LoadSpartanIdentity charge le bloc record compact depuis career_progression et metadata.
// DÃ©grade silencieusement si la carriÃ¨re n'est pas synchronisÃ©e pour le joueur.
//
//nolint:gocyclo // sÃ©rie de checks Valid sur 7 NullString + appels async (skill_peak_csr/lusr/identity)
func (r *HomeRepo) LoadSpartanIdentity(ctx context.Context) (*domain.HomeSpartanIdentityRow, error) {
	var row domain.HomeSpartanIdentityRow
	var spartanID sql.NullString
	var rankName sql.NullString
	var rankTier sql.NullString
	var bannerImageURL sql.NullString
	var emblemImageURL sql.NullString
	var backdropImageURL sql.NullString
	var adornmentImagePath sql.NullString

	err := r.pdb.ReadDB().QueryRow(ctx, Q26cHomeSpartanIdentity).Scan(
		&row.RankNumber,
		&row.CurrentXP,
		&row.XPForNextRank,
		&row.IsMaxRank,
		&spartanID,
		&rankName,
		&rankTier,
		&bannerImageURL,
		&emblemImageURL,
		&backdropImageURL,
		&adornmentImagePath,
	)
	if err != nil {
		if err == sql.ErrNoRows || isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}

	if spartanID.Valid {
		row.SpartanID = stringPtr(spartanID.String)
	}
	if rankName.Valid {
		row.RankName = stringPtr(rankName.String)
	}
	if rankTier.Valid {
		row.RankTier = stringPtr(rankTier.String)
	}
	if bannerImageURL.Valid {
		row.BannerImageURL = buildHomeIdentityAssetURL("banner", r.titleSlug(), bannerImageURL.String)
	}
	if emblemImageURL.Valid {
		row.EmblemImageURL = buildHomeIdentityAssetURL("emblem", r.titleSlug(), emblemImageURL.String)
	}
	if backdropImageURL.Valid {
		row.BackdropImageURL = buildHomeIdentityAssetURL("backdrop", r.titleSlug(), backdropImageURL.String)
	}
	if adornmentImagePath.Valid {
		row.AdornmentImageURL = buildHomeIdentityAssetURL("career-rank", r.titleSlug(), adornmentImagePath.String)
	}

	r.enrichSpartanIdentity(ctx, &row)
	row.HighestCSR = r.loadHomeSkillPeak(ctx, "CSR")
	row.HighestLUSR = r.loadHomeSkillPeak(ctx, "LUSR")

	if row.SpartanID == nil && row.RankNumber <= 0 &&
		row.BannerImageURL == nil && row.EmblemImageURL == nil && row.BackdropImageURL == nil &&
		row.HighestCSR == nil && row.HighestLUSR == nil {
		return nil, nil
	}
	return &row, nil
}

// enrichSpartanIdentity hydrate les paths d'assets visuels du rang carriÃ¨re
// (image rang + adornment) depuis metadata.duckdb. Les libellÃ©s (rang courant,
// rang suivant) sont rÃ©solus en aval par le service via le SemanticAdapter
// (cf. mappings.RankCatalog) â€” ils ne passent plus par le repo storage.
func (r *HomeRepo) enrichSpartanIdentity(ctx context.Context, row *domain.HomeSpartanIdentityRow) {
	if row == nil || row.RankNumber <= 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return
	}

	var imagePath, adornmentPath sql.NullString
	if err := r.pdb.Metadata.QueryRow(ctx, Q26dHomeCareerRankMeta, row.RankNumber).Scan(&imagePath, &adornmentPath); err != nil {
		return
	}
	if imagePath.Valid {
		row.RankImageURL = buildHomeIdentityAssetURL("career-rank", r.titleSlug(), imagePath.String)
	}
	if row.AdornmentImageURL == nil && adornmentPath.Valid {
		row.AdornmentImageURL = buildHomeIdentityAssetURL("career-rank", r.titleSlug(), adornmentPath.String)
	}
}

func (r *HomeRepo) loadHomeSkillPeak(ctx context.Context, ratingType string) *domain.HomeSkillPeakRow {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return nil
	}

	var ratingValue sql.NullFloat64
	var tierLabel sql.NullString
	var tier sql.NullString
	var subTier sql.NullInt16
	if err := r.pdb.ReadDB().QueryRow(ctx, Q26eHomeSkillPeakByType, ratingType).Scan(&ratingValue, &tierLabel, &tier, &subTier); err != nil {
		if err == sql.ErrNoRows || isTableNotFoundErr(err) {
			return nil
		}
		return nil
	}
	if !ratingValue.Valid {
		return nil
	}

	peak := &domain.HomeSkillPeakRow{RatingValue: ratingValue.Float64}
	if tierLabel.Valid {
		peak.TierLabel = stringPtr(tierLabel.String)
	}
	// LUSR est un rating cross-titre (LevelUp) : pas de slug de titre dans l'URL.
	titleSlug := homeStaticTitleSlug
	if strings.EqualFold(ratingType, "LUSR") {
		titleSlug = ""
	}
	peak.BadgeImageURL = buildHomeSkillPeakBadgeURL(optionalNullStringValue(tier), optionalNullStringValue(tierLabel), optionalNullInt16Value(subTier), titleSlug)
	return peak
}

// LoadRecentPlaylistRanks retourne les 3 derniÃ¨res playlists distinctes jouÃ©es avec leur
// dernier rang compÃ©titif connu (Q26g). Retourne (nil, nil) si aucune donnÃ©e.
// Le nom de playlist est rÃ©solu depuis asset_translations (metadata) puis adaptÃ© Ã  la locale.
func (r *HomeRepo) LoadRecentPlaylistRanks(ctx context.Context, locale string) ([]domain.HomePlaylistRank, error) {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return nil, nil
	}

	rows, err := r.pdb.ReadDB().Query(ctx, Q26gHomePlaylistRanks, r.pdb.XUID)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	type rawItem struct {
		playlistID   string
		playlistName string
		item         domain.HomePlaylistRank
	}

	var raws []rawItem
	for rows.Next() {
		var playlistID sql.NullString
		var playlistName sql.NullString
		var isRanked sql.NullBool
		var ratingType sql.NullString
		var ratingValue sql.NullFloat64
		var tier sql.NullString
		var tierFR sql.NullString
		var subTier sql.NullInt16
		var tierLabel sql.NullString

		if err := rows.Scan(&playlistID, &playlistName, &isRanked, &ratingType, &ratingValue, &tier, &tierFR, &subTier, &tierLabel); err != nil {
			return nil, err
		}

		item := domain.HomePlaylistRank{
			PlaylistName: playlistName.String,
			IsRanked:     isRanked.Bool,
		}
		if ratingValue.Valid {
			rt := strings.ToUpper(strings.TrimSpace(ratingType.String))
			item.RatingType = &rt
			item.RatingValue = &ratingValue.Float64
			if tierLabel.Valid {
				item.TierLabel = stringPtr(tierLabel.String)
			}
			item.BadgeImageURL = buildHomeSkillPeakBadgeURL(
				optionalNullStringValue(tier),
				optionalNullStringValue(tierLabel),
				optionalNullInt16Value(subTier),
				homeStaticTitleSlug,
			)
		}
		raws = append(raws, rawItem{
			playlistID:   playlistID.String,
			playlistName: playlistName.String,
			item:         item,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Enrichissement FR depuis asset_translations (mÃªme source que les tuiles de matchs).
	playlistIDs := make([]string, 0, len(raws))
	for _, raw := range raws {
		if raw.playlistID != "" {
			playlistIDs = append(playlistIDs, raw.playlistID)
		}
	}
	assetNames, _ := r.loadHomeAssetTranslationNames(ctx, "playlist", playlistIDs)

	result := make([]domain.HomePlaylistRank, 0, len(raws))
	for _, raw := range raws {
		nameFR := strings.TrimSpace(assetNames[raw.playlistID])
		raw.item.PlaylistName = resolvePlaylistNameForLocale(locale, nameFR, raw.playlistName)
		result = append(result, raw.item)
	}
	return result, nil
}

// resolvePlaylistNameForLocale retourne le nom de playlist adaptÃ© Ã  la locale.
// Pour "en*" â†’ prÃ©fÃ¨re l'anglais ; sinon â†’ prÃ©fÃ¨re le franÃ§ais.
func resolvePlaylistNameForLocale(locale, fr, en string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "en") {
		if strings.TrimSpace(en) != "" {
			return en
		}
		return fr
	}
	if strings.TrimSpace(fr) != "" {
		return fr
	}
	return en
}

func (r *HomeRepo) titleSlug() string {
	if r == nil || r.pdb == nil {
		return titlepkg.DefaultSlug
	}
	trimmed := strings.TrimSpace(r.pdb.TitleSlug)
	if trimmed == "" {
		return titlepkg.DefaultSlug
	}
	return trimmed
}

func buildHomeIdentityAssetURL(imageType string, titleID string, value string) *string {
	cleaned := normalizeHomeIdentityAssetPath(value)
	if cleaned == "" {
		return nil
	}

	resolved := fmt.Sprintf("%s/%s/%s/%s", homeIdentityAssetBasePath, imageType, titleID, cleaned)
	return &resolved
}

func normalizeHomeIdentityAssetPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	lower := strings.ToLower(trimmed)
	if strings.HasSuffix(lower, ".json") {
		return ""
	}
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return ""
		}
		trimmed = strings.TrimSpace(parsed.Path)
	}

	cleaned := path.Clean(strings.TrimLeft(trimmed, "/"))
	if cleaned == "" || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	return cleaned
}

func stringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalNullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}

func optionalNullInt16Value(value sql.NullInt16) int {
	if !value.Valid {
		return 0
	}
	return int(value.Int16)
}

// buildHomeSkillPeakBadgeURL construit l'URL du badge de rang.
// titleSlug : slug de titre (ex "halo_infinite") pour les ratings game-specific (CSR).
// Passer "" pour les ratings cross-titre (LUSR) â€” l'URL n'inclut pas de slug.
func buildHomeSkillPeakBadgeURL(tier string, tierLabel string, subTier int, titleSlug string) *string {
	normalizedTier, normalizedSubTier := normalizeHomeSkillPeakBadgeParts(tier, tierLabel, subTier)
	if normalizedTier == "" {
		return nil
	}
	// P5.4 (gap #9, ADR 0012) : déléguer à halo_infinite.AssetURLAdapter pour
	// le format `120px-HINF-CSR_*` (Halo-only). Évite la duplication du format.
	adapter := halo_infinite.NewAssetURLAdapter()
	var rawURL string
	if strings.EqualFold(normalizedTier, "Onyx") {
		rawURL = adapter.CSRRankImageURLOnyx()
	} else {
		if normalizedSubTier < 1 || normalizedSubTier > 6 {
			return nil
		}
		rawURL = adapter.CSRRankImageURL(normalizedTier, normalizedSubTier)
	}
	if rawURL == "" {
		return nil
	}
	// Quand titleSlug != adapter.TitleSlug() (LUSR cross-titre), recomposer
	// le path sans slug. L'adapter renvoie /static/ranks/<slug>/<id>.png.
	if titleSlug == "" {
		// Strip /static/ranks/halo_infinite/ → /static/ranks/
		prefix := static.MountPoint + "/" + static.Folder(static.KindCSRRank) + "/" + adapter.TitleSlug() + "/"
		if strings.HasPrefix(rawURL, prefix) {
			rawURL = path.Join(static.MountPoint, static.Folder(static.KindCSRRank), strings.TrimPrefix(rawURL, prefix))
		}
	}
	return &rawURL
}

// homeMedalIconURL retourne l'URL d'une icÃ´ne de mÃ©daille Ã  partir de son ID.
func homeMedalIconURL(medalID int64) string {
	return static.URL(static.KindMedal, homeStaticTitleSlug, strconv.FormatInt(medalID, 10), ".png")
}

func normalizeHomeSkillPeakBadgeParts(tier string, tierLabel string, subTier int) (string, int) {
	normalizedTier := canonicalHomeSkillTierName(tier)
	derivedSubTier := subTier
	if normalizedTier == "" && strings.TrimSpace(tierLabel) != "" {
		parts := strings.Fields(strings.TrimSpace(tierLabel))
		if len(parts) > 0 {
			normalizedTier = canonicalHomeSkillTierName(parts[0])
		}
		if derivedSubTier <= 0 && len(parts) > 1 {
			derivedSubTier = parseHomeSkillPeakSubTier(parts[len(parts)-1])
		}
	}
	return normalizedTier, derivedSubTier
}

func canonicalHomeSkillTierName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bronze":
		return "Bronze"
	case "silver":
		return "Silver"
	case "gold":
		return "Gold"
	case "platinum":
		return "Platinum"
	case "diamond":
		return "Diamond"
	case "onyx":
		return "Onyx"
	default:
		return ""
	}
}

func parseHomeSkillPeakSubTier(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	if numeric, err := strconv.Atoi(trimmed); err == nil {
		return numeric
	}
	switch strings.ToUpper(trimmed) {
	case "I":
		return 1
	case "II":
		return 2
	case "III":
		return 3
	case "IV":
		return 4
	case "V":
		return 5
	case "VI":
		return 6
	default:
		return 0
	}
}

//nolint:gocyclo // 4 enrichments sÃ©quentiels (map/pair/playlist/variant) avec multiples Valid checks
func (r *HomeRepo) enrichHomeMatchTranslations(ctx context.Context, matches []legacymatch.HomeMatchRow) {
	if len(matches) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return
	}

	mapNames, _ := r.loadHomeAssetTranslationNames(ctx, "map", collectMissingHomeAssetIDs(matches, "map"))
	pairNames, _ := r.loadHomeAssetTranslationNames(ctx, "pair", collectMissingHomeAssetIDs(matches, "pair"))
	gameVariantNames, _ := r.loadHomeAssetTranslationNames(ctx, "game_variant", collectMissingHomeAssetIDs(matches, "game_variant"))
	playlistNames, _ := r.loadHomeAssetTranslationNames(ctx, "playlist", collectMissingHomeAssetIDs(matches, "playlist"))

	// mode_name_tr : collecte les modes EN de TOUS les matchs (sans filtre needsTranslation).
	// Quand pair_name est un UUID brut, fallback sur le nom dans pairNames (asset_translations).
	modeENSet := make(map[string]struct{})
	for _, m := range matches {
		if en := analysis.NormalizeModeLabel(m.PairName); en != "" {
			modeENSet[en] = struct{}{}
		}
		// Si pair_name est un UUID (non normalisable en mode lisible), enrichir depuis asset_translations
		if assetName := strings.TrimSpace(pairNames[m.PairID]); assetName != "" {
			if en2 := analysis.NormalizeModeLabel(assetName); en2 != "" {
				modeENSet[en2] = struct{}{}
			}
		}
	}
	modeENList := make([]string, 0, len(modeENSet))
	for k := range modeENSet {
		modeENList = append(modeENList, k)
	}
	modeNamesFR, _ := r.loadHomeModeNameTranslations(ctx, modeENList)

	for i := range matches {
		if needsHomeAssetTranslation(matches[i].MapNameFR, matches[i].MapName) {
			if name := strings.TrimSpace(mapNames[matches[i].MapID]); name != "" {
				matches[i].MapNameFR = name
			}
		}
		// PrioritÃ© 1 : mode_name_tr appliquÃ© sur tous les matchs (pair_name_fr peut contenir une valeur EN non traduite)
		modeEN := analysis.NormalizeModeLabel(matches[i].PairName)
		modeFR := modeNamesFR[modeEN]
		// Si PairName est un UUID (non normalisable), tenter via le nom dans asset_translations
		if modeFR == "" {
			if assetName := strings.TrimSpace(pairNames[matches[i].PairID]); assetName != "" {
				modeFR = modeNamesFR[analysis.NormalizeModeLabel(assetName)]
			}
		}
		if modeFR != "" {
			matches[i].PairNameFR = modeFR
		} else if needsHomeAssetTranslation(matches[i].PairNameFR, matches[i].PairName) {
			// PrioritÃ© 2 : asset_translations (nom complet de paire, fallback)
			if name := strings.TrimSpace(pairNames[matches[i].PairID]); name != "" {
				matches[i].PairNameFR = name
			}
		}
		if needsHomeAssetTranslation(matches[i].GameVariantNameFR, matches[i].GameVariantName) {
			if name := strings.TrimSpace(gameVariantNames[matches[i].GameVariantID]); name != "" {
				matches[i].GameVariantNameFR = name
			}
		}
		if needsHomeAssetTranslation(matches[i].PlaylistNameFR, matches[i].PlaylistName) {
			if name := strings.TrimSpace(playlistNames[matches[i].PlaylistID]); name != "" {
				matches[i].PlaylistNameFR = name
			}
		}
	}
}

func collectMissingHomeAssetIDs(matches []legacymatch.HomeMatchRow, assetType string) []string {
	ids := make(map[string]struct{})
	for _, match := range matches {
		var assetID string
		var labelFR string
		var labelEN string

		switch assetType {
		case "map":
			assetID = match.MapID
			labelFR = match.MapNameFR
			labelEN = match.MapName
		case "pair":
			assetID = match.PairID
			labelFR = match.PairNameFR
			labelEN = match.PairName
		case "game_variant":
			assetID = match.GameVariantID
			labelFR = match.GameVariantNameFR
			labelEN = match.GameVariantName
		case "playlist":
			assetID = match.PlaylistID
			labelFR = match.PlaylistNameFR
			labelEN = match.PlaylistName
		default:
			return nil
		}

		if strings.TrimSpace(assetID) == "" || !needsHomeAssetTranslation(labelFR, labelEN) {
			continue
		}
		ids[assetID] = struct{}{}
	}

	result := make([]string, 0, len(ids))
	for assetID := range ids {
		result = append(result, assetID)
	}
	return result
}

func (r *HomeRepo) loadHomeAssetTranslationNames(ctx context.Context, assetType string, assetIDs []string) (map[string]string, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(assetIDs)), ",")
	query := fmt.Sprintf(`
		SELECT asset_id, name, lang
		FROM asset_translations
		WHERE asset_type = ?
		  AND lang IN ('fr-FR', 'fr')
		  AND name IS NOT NULL
		  AND TRIM(name) != ''
		  AND asset_id IN (%s)
		ORDER BY asset_id, CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END
	`, placeholders)

	args := make([]any, 0, len(assetIDs)+1)
	args = append(args, assetType)
	for _, assetID := range assetIDs {
		args = append(args, assetID)
	}

	rows, err := r.pdb.Metadata.Query(ctx, query, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	translations := make(map[string]string)
	for rows.Next() {
		var assetID string
		var name string
		var lang string
		if err := rows.Scan(&assetID, &name, &lang); err != nil {
			return nil, err
		}
		if _, exists := translations[assetID]; !exists {
			translations[assetID] = name
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return translations, nil
}

// loadHomeModeNameTranslations rÃ©sout les noms FR des modes depuis mode_name_tr.
// modeENNames est la liste de noms EN extraits de pair_name via NormalizeModeLabel.
func (r *HomeRepo) loadHomeModeNameTranslations(ctx context.Context, modeENNames []string) (map[string]string, error) {
	if len(modeENNames) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(modeENNames)), ",")
	query := fmt.Sprintf(`
		SELECT mode_en, name
		FROM mode_name_tr
		WHERE lang = 'fr'
		  AND mode_en IN (%s)
	`, placeholders)

	args := make([]any, len(modeENNames))
	for i, name := range modeENNames {
		args[i] = name
	}

	rows, err := r.pdb.Metadata.Query(ctx, query, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	translations := make(map[string]string)
	for rows.Next() {
		var modeEN, nameFR string
		if err := rows.Scan(&modeEN, &nameFR); err != nil {
			continue
		}
		if strings.TrimSpace(nameFR) != "" {
			translations[modeEN] = nameFR
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return translations, nil
}

func needsHomeAssetTranslation(labelFR, labelEN string) bool {
	trimmedFR := strings.TrimSpace(labelFR)
	if trimmedFR == "" {
		return true
	}
	trimmedEN := strings.TrimSpace(labelEN)
	return trimmedEN != "" && strings.EqualFold(trimmedFR, trimmedEN)
}

// CountPlayerMatches retourne le nombre total de matchs du joueur (Q26b).
func (r *HomeRepo) CountPlayerMatches(ctx context.Context) (int, error) {
	var count int
	err := r.pdb.ReadDB().QueryRow(ctx, Q26bCountPlayerMatches, r.pdb.XUID).Scan(&count)
	return count, err
}

// LoadHomeSessions charge les sessions avec label depuis player_match_enrichment (Q27).
func (r *HomeRepo) LoadHomeSessions(ctx context.Context) ([]legacymatch.HomeSessionRow, error) {
	rows, err := r.pdb.ReadDB().Query(ctx, Q27HomeSessions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []legacymatch.HomeSessionRow
	for rows.Next() {
		var row legacymatch.HomeSessionRow
		if err := rows.Scan(
			&row.MatchID,
			&row.SessionID,
			&row.SessionLabel,
			&row.IsWithFriends,
			&row.StartTime,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadRecentMedia charge les mÃ©dias rÃ©cents du joueur (Q28).
// Retourne une liste vide si la table media_files n'existe pas.
func (r *HomeRepo) LoadRecentMedia(ctx context.Context, limit int) ([]domain.HomeMediaRow, error) {
	rows, err := r.pdb.ReadDB().Query(ctx, Q28RecentMedia, limit)
	if err != nil {
		// La table media_files peut ne pas exister â€” dÃ©gradation silencieuse.
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeMediaRow
	for rows.Next() {
		var row domain.HomeMediaRow
		var matchID sql.NullString
		var matchStartTime sql.NullTime
		if err := rows.Scan(&row.FileName, &matchID, &matchStartTime); err != nil {
			return nil, err
		}
		if matchID.Valid {
			row.MatchID = &matchID.String
		}
		if matchStartTime.Valid {
			row.MatchStartTime = &matchStartTime.Time
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadFavoriteWeapon retourne le nom localisÃ© et le nombre de kills de l'arme la plus utilisÃ©e
// par le joueur sur l'ensemble de ses matchs (Q26k).
// DÃ©gradation silencieuse : retourne ("", 0, nil) si la table weapon_kills est vide ou absente.
func (r *HomeRepo) LoadFavoriteWeapon(ctx context.Context, locale string) (string, int, error) {
	var weaponID uint64
	var totalKills int
	err := r.pdb.ReadDB().QueryRow(ctx, Q26kFavoriteWeapon, r.pdb.XUID).Scan(&weaponID, &totalKills)
	if err != nil {
		return "", 0, nil //nolint:nilerr // dÃ©gradation silencieuse
	}

	// RÃ©solution du label depuis metadata.
	// Contournement driver : database/sql ne supporte pas uint64 avec bit63=1.
	// weapon_id est une valeur interne (pas user input) â†’ littÃ©ral dÃ©cimal sÃ»r.
	nameCol := "COALESCE(name_fr, name_en, '')"
	if locale == "en" {
		nameCol = "COALESCE(name_en, name_fr, '')"
	}
	var weaponName string
	metaErr := r.pdb.Metadata.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM weapon_labels WHERE weapon_id = %d", nameCol, weaponID), //nolint:gosec
	).Scan(&weaponName)
	if metaErr != nil || weaponName == "" {
		weaponName = "Inconnue"
		if locale == "en" {
			weaponName = "Unknown"
		}
	}
	return weaponName, totalKills, nil
}

// LoadMatchMedals charge les mÃ©dailles d'un joueur pour un lot de matchs (Q26h).
// Retourne un map match_id â†’ []domain.RecentMatchMedal, triÃ© par count DESC.
// Labels rÃ©solus via medal_definitions (name_fr) en prioritÃ©, citation_mappings en fallback.
func (r *HomeRepo) LoadMatchMedals(ctx context.Context, matchIDs []string) (map[string][]domain.RecentMatchMedal, error) {
	result := make(map[string][]domain.RecentMatchMedal)
	if len(matchIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, 0, len(matchIDs)+1)
	args = append(args, r.pdb.XUID)
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(Q26hMatchMedalsTemplate, strings.Join(placeholders, ", "))

	rows, err := r.pdb.ReadDB().Query(ctx, query, args...)
	if err != nil {
		return result, nil // dÃ©gradation silencieuse
	}
	defer rows.Close()

	type rawRow struct {
		matchID string
		medalID int64
		count   int
	}
	var rawRows []rawRow
	var medalIDsList []int64
	seen := make(map[int64]struct{})
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.matchID, &rr.medalID, &rr.count); err != nil {
			continue
		}
		rawRows = append(rawRows, rr)
		if _, ok := seen[rr.medalID]; !ok {
			seen[rr.medalID] = struct{}{}
			medalIDsList = append(medalIDsList, rr.medalID)
		}
	}
	if err := rows.Err(); err != nil {
		return result, nil
	}

	metaMap := resolveMedalLabels(ctx, r.pdb.Metadata, medalIDsList)

	for _, rr := range rawRows {
		meta := metaMap[rr.medalID]
		result[rr.matchID] = append(result[rr.matchID], domain.RecentMatchMedal{
			MedalID:     rr.medalID,
			Name:        meta.label,
			Count:       rr.count,
			Description: meta.description,
			ImageURL:    homeMedalIconURL(rr.medalID),
		})
	}
	return result, nil
}

// medalLabel contient le nom localisÃ© et la description d'une mÃ©daille.
type medalLabel struct {
	label       string
	description string
}

// resolveMedalLabels rÃ©sout les labels de mÃ©dailles avec la chaÃ®ne BCP-47 complÃ¨te :
//
//	medal_translations (fr-FR) â†’ medal_definitions.name_fr
//	â†’ medal_translations (en-US) â†’ medal_definitions.name_en
//
// Miroir de resolve_medal_name(id, lang) dans src/data/medal_definitions.py.
func resolveMedalLabels(ctx context.Context, db *DB, medalIDs []int64) map[int64]medalLabel {
	result := make(map[int64]medalLabel, len(medalIDs))
	if len(medalIDs) == 0 || db == nil {
		return result
	}

	// ChaÃ®ne BCP-47 : medal_translations (fr-FR, en-US) > medal_definitions (name_fr, name_en).
	q, mArgs, ok := buildLookupQuery(
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
		        ) AS description
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
	mRows, err := db.Query(ctx, q, mArgs...)
	if err != nil {
		return result
	}
	defer mRows.Close()
	for mRows.Next() {
		var id int64
		var name, desc string
		if err := mRows.Scan(&id, &name, &desc); err == nil && name != "" {
			result[id] = medalLabel{label: name, description: desc}
		}
	}
	return result
}

// LoadMatchCitations charge les citations progressÃ©es pour un lot de matchs (Q26i + Q26j).
// Retourne un map match_id â†’ []domain.HomeMatchCitationRaw, dÃ©gradation silencieuse.
func (r *HomeRepo) LoadMatchCitations(ctx context.Context, matchIDs []string) (map[string][]domain.HomeMatchCitationRaw, error) {
	result := make(map[string][]domain.HomeMatchCitationRaw)
	if len(matchIDs) == 0 || r.pdb == nil || r.pdb.Player == nil {
		return result, nil
	}

	// Ã‰tape 1 : charger les deltas + cumulatifs depuis match_citations (player DB).
	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(Q26iMatchCitationsTemplate, strings.Join(placeholders, ", "))

	rows, err := r.pdb.ReadDB().Query(ctx, query, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return result, nil
		}
		return result, nil // dÃ©gradation silencieuse
	}
	defer rows.Close()

	type citIntermediate struct {
		norm       string
		delta      int
		cumulative int
	}
	rawByMatch := make(map[string][]citIntermediate)
	normsSeen := make(map[string]struct{})
	for rows.Next() {
		var matchID, norm string
		var delta, cumulative int
		if err := rows.Scan(&matchID, &norm, &delta, &cumulative); err != nil {
			continue
		}
		rawByMatch[matchID] = append(rawByMatch[matchID], citIntermediate{norm, delta, cumulative})
		normsSeen[norm] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return result, nil
	}
	if len(rawByMatch) == 0 {
		return result, nil
	}

	// Ã‰tape 2 : charger les mÃ©tadonnÃ©es (display, image_path, tier_targets) depuis metadata.
	norms := make([]string, 0, len(normsSeen))
	for n := range normsSeen {
		norms = append(norms, n)
	}
	metaMap := r.loadCitationMappingMeta(ctx, norms)

	// Ã‰tape 3 : merger.
	for matchID, cits := range rawByMatch {
		for _, c := range cits {
			meta := metaMap[c.norm]
			var imgPath *string
			if meta.imagePath != "" {
				imgPath = &meta.imagePath
			}
			result[matchID] = append(result[matchID], domain.HomeMatchCitationRaw{
				Norm:        c.norm,
				Display:     meta.display,
				Description: meta.description,
				ImagePath:   safeStringValue(imgPath),
				TierTargets: meta.tierTargets,
				Delta:       c.delta,
				Cumulative:  c.cumulative,
			})
		}
	}
	return result, nil
}

type citationMeta struct {
	display     string
	imagePath   string
	tierTargets string
	description string
}

// loadCitationMappingMeta interroge citation_mappings sur pdb.Metadata pour un ensemble de norms.
func (r *HomeRepo) loadCitationMappingMeta(ctx context.Context, norms []string) map[string]citationMeta {
	result := make(map[string]citationMeta, len(norms))
	if len(norms) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return result
	}

	placeholders := make([]string, len(norms))
	args := make([]interface{}, len(norms))
	for i, n := range norms {
		placeholders[i] = "?"
		args[i] = n
	}
	query := fmt.Sprintf(Q26jCitationMappingsForNormsTemplate, strings.Join(placeholders, ", "))

	rows, err := r.pdb.Metadata.Query(ctx, query, args...)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var norm, display, imagePath, tierTargets, description string
		if err := rows.Scan(&norm, &display, &imagePath, &tierTargets, &description); err != nil {
			continue
		}
		result[norm] = citationMeta{display: display, imagePath: imagePath, tierTargets: tierTargets, description: description}
	}
	return result
}

func safeStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// isTableNotFoundErr dÃ©tecte les erreurs "table not found" DuckDB.
func isTableNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Table with name") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "no such table")
}

// ---------------------------------------------------------------------------
// BattlePassCacheRepository implementation
// ---------------------------------------------------------------------------

// LoadCachedBattlePass retourne les donnÃ©es BP depuis battlepass_snapshots si une
// entrÃ©e rÃ©cente du joueur existe dans la fenÃªtre ttl.
func (r *HomeRepo) LoadCachedBattlePass(ctx context.Context, ttl time.Duration) (*domain.BattlePassResponse, bool, error) {
	secs := int64(ttl.Seconds())
	query := fmt.Sprintf(`
		SELECT reward_track_path, current_rank, partial_progress
		FROM battlepass_snapshots
		WHERE xuid = ?
		  AND snapshot_at > CURRENT_TIMESTAMP - INTERVAL '%d' SECOND
		ORDER BY is_active DESC, snapshot_at DESC
		LIMIT 1`, secs)

	var trackPath string
	var rank, progress int
	err := r.pdb.ReadDB().QueryRow(ctx, query, r.pdb.XUID).Scan(&trackPath, &rank, &progress)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		if isTableNotFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("home_repo: cache BP query: %w", err)
	}

	resp := &domain.BattlePassResponse{
		Available:   true,
		Rank:        &rank,
		Progress:    &progress,
		RewardTrack: &trackPath,
		FromCache:   true,
	}
	return resp, true, nil
}

// challengeSnapshotRow est une ligne agrÃ©gÃ©e pour la reconstruction ChallengesResponse.
type challengeSnapshotRow struct {
	status    string
	xpReward  int
	expiresAt sql.NullTime
}

// LoadCachedChallenges retourne un rÃ©sumÃ© des snapshots rÃ©cents depuis challenge_snapshots
// (la snapshot la plus rÃ©cente par challenge_path dans la fenÃªtre ttl).
func (r *HomeRepo) LoadCachedChallenges(ctx context.Context, ttl time.Duration) (*domain.ChallengesResponse, bool, error) {
	secs := int64(ttl.Seconds())
	// SÃ©lectionne la snapshot la plus rÃ©cente par challenge_path dans la fenÃªtre TTL.
	query := fmt.Sprintf(`
		SELECT status, xp_reward, expires_at
		FROM (
			SELECT status, xp_reward, expires_at,
			       ROW_NUMBER() OVER (PARTITION BY challenge_path ORDER BY snapshot_at DESC) AS rn
			FROM challenge_snapshots
			WHERE xuid = ?
			  AND snapshot_at > CURRENT_TIMESTAMP - INTERVAL '%d' SECOND
		) t
		WHERE rn = 1`, secs)

	rows, err := r.pdb.ReadDB().Query(ctx, query, r.pdb.XUID)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("home_repo: cache challenges query: %w", err)
	}
	defer rows.Close()

	var snapshots []challengeSnapshotRow
	for rows.Next() {
		var s challengeSnapshotRow
		if err := rows.Scan(&s.status, &s.xpReward, &s.expiresAt); err != nil {
			return nil, false, fmt.Errorf("home_repo: cache challenges scan: %w", err)
		}
		snapshots = append(snapshots, s)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if len(snapshots) == 0 {
		return nil, false, nil
	}

	total := len(snapshots)
	completed := 0
	xpAvailable := 0
	var earliestExpiry *time.Time

	for _, s := range snapshots {
		if strings.EqualFold(s.status, "Completed") {
			completed++
		} else {
			xpAvailable += s.xpReward
		}
		if s.expiresAt.Valid {
			t := s.expiresAt.Time
			if earliestExpiry == nil || t.Before(*earliestExpiry) {
				earliestExpiry = &t
			}
		}
	}

	resp := &domain.ChallengesResponse{
		Available: true,
		Total:     &total,
		Completed: &completed,
		FromCache: true,
	}
	if xpAvailable > 0 {
		resp.XPAvailable = &xpAvailable
	}
	if earliestExpiry != nil {
		s := earliestExpiry.Format(time.RFC3339)
		resp.NextExpiry = &s
	}
	return resp, true, nil
}
