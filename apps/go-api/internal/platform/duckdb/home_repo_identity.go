// Package duckdb — home_repo_identity.go : Spartan identity Home (rank,
// emblem, banner, backdrop, adornment) — Q26c/Q26d + enrichments.
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path"
	"strings"

	"levelup/go-api/internal/domain"
)

const homeIdentityAssetBasePath = "/api/v1/assets/spartan"

// LoadSpartanIdentity charge le bloc record compact depuis career_progression et metadata.
// Dégrade silencieusement si la carrière n'est pas synchronisée pour le joueur.
//
//nolint:gocyclo // série de checks Valid sur 7 NullString + appels async (skill_peak_csr/lusr/identity)
func (r *HomeRepo) LoadSpartanIdentity(ctx context.Context) (*domain.HomeSpartanIdentityRow, error) {
	var row domain.HomeSpartanIdentityRow
	var spartanID sql.NullString
	var rankName sql.NullString
	var rankTier sql.NullString
	var bannerImageURL sql.NullString
	var emblemImageURL sql.NullString
	var backdropImageURL sql.NullString
	var adornmentImagePath sql.NullString

	err := r.pdb.ReadDB().QueryRow(ctx, Q26cHomeSpartanIdentity, r.pdb.XUID).Scan(
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

// BuildSpartanIdentityFromCareerRow assemble un HomeSpartanIdentityRow à
// partir d'une CareerRankRow déjà mergée (live + per-field DB fallback) plus,
// si includePeaks, les skill peaks CSR/LUSR lus depuis match_skill_rank.
//
// Utilisé par CareerLiveService pour servir l'identité Spartan sans
// dépendre de la query Q26c (qui reste exposée via LoadSpartanIdentity pour
// les chemins de fallback / compatibilité ascendante).
//
// includePeaks : les skill peaks sont lus sur la player DB de CE repo
// (`r.pdb.Player`). Ils ne sont donc valides que si le sujet de l'identité est
// le propriétaire de cette DB. Le caller doit passer false quand il construit
// l'identité d'un joueur tiers (cas Explorer : joueur cible recherché) — sinon
// on afficherait les peaks du propriétaire de la page sur la carte d'un autre
// joueur, et on paierait 2 scans `match_skill_rank` inutiles. Cf. plan
// explorer-target-profile-auth (volet B).
//
// Retourne nil si tous les champs visibles sont vides (joueur jamais sync'd).
func (r *HomeRepo) BuildSpartanIdentityFromCareerRow(ctx context.Context, careerRow *CareerRankRow, includePeaks bool) *domain.HomeSpartanIdentityRow {
	if r == nil {
		return nil
	}
	row := &domain.HomeSpartanIdentityRow{}
	if careerRow != nil {
		r.populateIdentityFromCareerRow(ctx, row, careerRow)
	}
	if includePeaks {
		row.HighestCSR = r.loadHomeSkillPeak(ctx, "CSR")
		row.HighestLUSR = r.loadHomeSkillPeak(ctx, "LUSR")
	}

	if isEmptyHomeIdentity(row) {
		return nil
	}
	return row
}

// populateIdentityFromCareerRow copie les champs visibles + URLs d'assets
// depuis CareerRankRow vers HomeSpartanIdentityRow, puis hydrate via metadata.
func (r *HomeRepo) populateIdentityFromCareerRow(ctx context.Context, row *domain.HomeSpartanIdentityRow, careerRow *CareerRankRow) {
	row.RankNumber = careerRow.Rank
	row.CurrentXP = careerRow.CurrentXP
	row.XPForNextRank = careerRow.XPForNextRank
	row.IsMaxRank = careerRow.IsMaxRank
	if careerRow.SpartanID != "" {
		row.SpartanID = stringPtr(careerRow.SpartanID)
	}
	if careerRow.RankName != "" {
		row.RankName = stringPtr(careerRow.RankName)
	}
	if careerRow.RankTier != "" {
		row.RankTier = stringPtr(careerRow.RankTier)
	}
	if careerRow.BannerImageURL != "" {
		row.BannerImageURL = buildHomeIdentityAssetURL("banner", r.titleSlug(), careerRow.BannerImageURL)
	}
	if careerRow.EmblemImageURL != "" {
		row.EmblemImageURL = buildHomeIdentityAssetURL("emblem", r.titleSlug(), careerRow.EmblemImageURL)
	}
	if careerRow.BackdropImageURL != "" {
		row.BackdropImageURL = buildHomeIdentityAssetURL("backdrop", r.titleSlug(), careerRow.BackdropImageURL)
	}
	if careerRow.AdornmentPath != "" {
		row.AdornmentImageURL = buildHomeIdentityAssetURL("career-rank", r.titleSlug(), careerRow.AdornmentPath)
	}
	r.enrichSpartanIdentity(ctx, row)
}

// isEmptyHomeIdentity retourne true si la row n'a aucun champ visible (joueur jamais sync'd).
func isEmptyHomeIdentity(row *domain.HomeSpartanIdentityRow) bool {
	return row.SpartanID == nil && row.RankNumber <= 0 &&
		row.BannerImageURL == nil && row.EmblemImageURL == nil && row.BackdropImageURL == nil &&
		row.HighestCSR == nil && row.HighestLUSR == nil
}

// enrichSpartanIdentity hydrate les paths d'assets visuels du rang carrière
// (image rang + adornment) depuis metadata.duckdb. Les libellés (rang courant,
// rang suivant) sont résolus en aval par le service via le SemanticAdapter
// (cf. mappings.RankCatalog) — ils ne passent plus par le repo storage.
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
