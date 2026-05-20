// Package duckdb — career_live_repo.go : accès DB pour le flow live carrière
// (lecture per-field-merged + INSERT-if-changed).
//
// Ce repo est utilisé par CareerLiveService (service layer) pour découpler la
// synchronisation XP/identité Spartan du post-sync matchs.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CareerRankRow est la projection raw d'une ligne `career_progression` après
// per-field merge (dernière valeur non-vide par colonne via ARG_MAX + FILTER).
// Les URLs d'images sont les valeurs absolues stockées en DB (résolues à la
// sync précédente) — la construction des URLs d'asset internes (/api/v1/...)
// reste l'affaire du service consommateur.
type CareerRankRow struct {
	Rank             int
	RankName         string
	RankTier         string
	CurrentXP        int
	XPForNextRank    int
	XPTotal          int
	IsMaxRank        bool
	SpartanID        string
	BannerImageURL   string
	EmblemImageURL   string
	BackdropImageURL string
	AdornmentPath    string
}

// IsEmpty retourne true si la row ne porte aucune donnée exploitable
// (utilisé pour décider si on retourne nil au caller).
func (r *CareerRankRow) IsEmpty() bool {
	if r == nil {
		return true
	}
	return r.Rank <= 0 &&
		r.CurrentXP == 0 &&
		r.SpartanID == "" &&
		r.BannerImageURL == "" &&
		r.EmblemImageURL == "" &&
		r.BackdropImageURL == ""
}

// CareerLiveRepo expose les opérations DB du flow live carrière.
type CareerLiveRepo struct {
	pdb *PlayerDB
}

// NewCareerLiveRepo crée un repo lié au PlayerDB du joueur courant.
func NewCareerLiveRepo(pdb *PlayerDB) *CareerLiveRepo {
	return &CareerLiveRepo{pdb: pdb}
}

// qLoadLastCareerRank : projection per-field-merged de career_progression.
// Pattern ARG_MAX + FILTER identique à Q26cHomeSpartanIdentity, étendu à
// xp_total et adornment_path. La table contient potentiellement plusieurs
// xuids historiques (cas rare), on filtre explicitement.
const qLoadLastCareerRank = `
SELECT
    COALESCE(ARG_MAX(rank,              recorded_at), 0)                                                    AS rank,
    NULLIF(TRIM(ARG_MAX(rank_name,      recorded_at)), '')                                                  AS rank_name,
    NULLIF(TRIM(ARG_MAX(rank_tier,      recorded_at)), '')                                                  AS rank_tier,
    COALESCE(ARG_MAX(current_xp,        recorded_at), 0)                                                    AS current_xp,
    COALESCE(ARG_MAX(xp_for_next_rank,  recorded_at), 0)                                                    AS xp_for_next_rank,
    COALESCE(ARG_MAX(xp_total,          recorded_at), 0)                                                    AS xp_total,
    COALESCE(ARG_MAX(is_max_rank,       recorded_at), FALSE)                                                AS is_max_rank,
    ARG_MAX(spartan_id,         recorded_at) FILTER (WHERE NULLIF(TRIM(spartan_id),         '') IS NOT NULL) AS spartan_id,
    ARG_MAX(banner_image_url,   recorded_at) FILTER (WHERE NULLIF(TRIM(banner_image_url),   '') IS NOT NULL) AS banner_image_url,
    ARG_MAX(emblem_image_url,   recorded_at) FILTER (WHERE NULLIF(TRIM(emblem_image_url),   '') IS NOT NULL) AS emblem_image_url,
    ARG_MAX(backdrop_image_url, recorded_at) FILTER (WHERE NULLIF(TRIM(backdrop_image_url), '') IS NOT NULL) AS backdrop_image_url,
    ARG_MAX(adornment_path,     recorded_at) FILTER (WHERE NULLIF(TRIM(adornment_path),     '') IS NOT NULL) AS adornment_path
FROM career_progression
WHERE xuid = ?`

// LoadLastCareerRank retourne la projection per-field-merged de la dernière
// row connue. Garantit un fallback robuste : si la dernière row a un emblem
// vide mais qu'une row antérieure portait un emblem, cette dernière valeur
// est remontée. Retourne (nil, nil) si la table est vide/inexistante ou si
// aucun xuid n'a de données.
func (r *CareerLiveRepo) LoadLastCareerRank(ctx context.Context, xuid string) (*CareerRankRow, error) {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return nil, nil
	}
	var (
		row           CareerRankRow
		rankName      sql.NullString
		rankTier      sql.NullString
		spartanID     sql.NullString
		bannerURL     sql.NullString
		emblemURL     sql.NullString
		backdropURL   sql.NullString
		adornmentPath sql.NullString
	)
	err := r.pdb.Player.QueryRow(ctx, qLoadLastCareerRank, xuid).Scan(
		&row.Rank,
		&rankName,
		&rankTier,
		&row.CurrentXP,
		&row.XPForNextRank,
		&row.XPTotal,
		&row.IsMaxRank,
		&spartanID,
		&bannerURL,
		&emblemURL,
		&backdropURL,
		&adornmentPath,
	)
	if err != nil {
		if err == sql.ErrNoRows || isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("LoadLastCareerRank: %w", err)
	}
	if rankName.Valid {
		row.RankName = rankName.String
	}
	if rankTier.Valid {
		row.RankTier = rankTier.String
	}
	if spartanID.Valid {
		row.SpartanID = spartanID.String
	}
	if bannerURL.Valid {
		row.BannerImageURL = bannerURL.String
	}
	if emblemURL.Valid {
		row.EmblemImageURL = emblemURL.String
	}
	if backdropURL.Valid {
		row.BackdropImageURL = backdropURL.String
	}
	if adornmentPath.Valid {
		row.AdornmentPath = adornmentPath.String
	}
	if row.IsEmpty() {
		return nil, nil
	}
	return &row, nil
}

// CareerRankRowEqualForInsert compare deux rows sur les champs qui justifient
// un nouveau snapshot. Volontairement excluant rank_name / rank_tier /
// xp_for_next_rank / xp_total / adornment_path qui sont dérivés du rank via
// l'enrichissement metadata (un changement métadonnée seul ne mérite pas un
// snapshot).
func CareerRankRowEqualForInsert(a, b *CareerRankRow) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Rank == b.Rank &&
		a.CurrentXP == b.CurrentXP &&
		a.IsMaxRank == b.IsMaxRank &&
		a.SpartanID == b.SpartanID &&
		a.BannerImageURL == b.BannerImageURL &&
		a.EmblemImageURL == b.EmblemImageURL &&
		a.BackdropImageURL == b.BackdropImageURL
}

// InsertCareerProgressionIfChanged INSERT une nouvelle row si au moins un
// champ d'identité (rank, current_xp, is_max_rank, spartan_id, banner, emblem,
// backdrop) diffère de la dernière row du xuid. Sinon no-op.
//
// Retourne (true, nil) si une row a été insérée, (false, nil) si skippée.
func (r *CareerLiveRepo) InsertCareerProgressionIfChanged(
	ctx context.Context,
	xuid string,
	data *CareerRankRow,
) (bool, error) {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return false, fmt.Errorf("InsertCareerProgressionIfChanged: pdb non disponible")
	}
	if xuid == "" || data == nil {
		return false, fmt.Errorf("InsertCareerProgressionIfChanged: xuid ou data vide")
	}

	last, err := r.LoadLastCareerRank(ctx, xuid)
	if err != nil {
		return false, fmt.Errorf("InsertCareerProgressionIfChanged load last: %w", err)
	}
	if last != nil && CareerRankRowEqualForInsert(last, data) {
		return false, nil
	}

	now := time.Now().UTC()
	const insertSQL = `
INSERT INTO career_progression (
    xuid, rank, rank_name, rank_tier,
    current_xp, xp_for_next_rank, xp_total,
    is_max_rank, adornment_path, spartan_id,
    banner_image_url, emblem_image_url, backdrop_image_url, recorded_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	if _, err := r.pdb.Player.Exec(ctx, insertSQL,
		xuid, data.Rank, data.RankName, data.RankTier,
		data.CurrentXP, data.XPForNextRank, data.XPTotal,
		data.IsMaxRank, data.AdornmentPath, data.SpartanID,
		data.BannerImageURL, data.EmblemImageURL, data.BackdropImageURL, now,
	); err != nil {
		return false, fmt.Errorf("InsertCareerProgressionIfChanged exec: %w", err)
	}
	return true, nil
}

// EnrichFromMetadata hydrate les champs dérivés du rank depuis
// `metadata.career_ranks` : rank_name, rank_tier, xp_for_next_rank, xp_total
// et adornment_path. Idempotent : ne fait rien si Metadata n'est pas attachée
// ou si la row n'a pas de rank valide. Retourne nil sans erreur si la ligne
// metadata n'existe pas pour ce rank_id (rare : nouveau palier non encore
// référencé).
//
// Pendant du sync.enrichCareerRankFromMetadata mais opérant sur CareerRankRow
// (au lieu de CareerRankData), pour servir le flow live découplé du sync.
func (r *CareerLiveRepo) EnrichFromMetadata(ctx context.Context, row *CareerRankRow) error {
	if r == nil || r.pdb == nil || r.pdb.Metadata == nil || row == nil {
		return nil
	}
	if row.Rank <= 0 {
		return nil
	}

	var (
		titleEN       string
		tierType      sql.NullString
		grade         sql.NullInt64
		xpRequired    int
		adornmentPath sql.NullString
	)
	err := r.pdb.Metadata.QueryRow(ctx,
		`SELECT title_en, tier_type, grade, xp_required, adornment_icon_path
		 FROM career_ranks
		 WHERE rank_id = ?`,
		row.Rank,
	).Scan(&titleEN, &tierType, &grade, &xpRequired, &adornmentPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("EnrichFromMetadata row: %w", err)
	}

	row.XPForNextRank = xpRequired
	if tierType.Valid {
		row.RankTier = strings.TrimSpace(tierType.String)
	}
	row.RankName = buildCareerRankNameDB(titleEN, row.RankTier, grade)
	if adornmentPath.Valid {
		row.AdornmentPath = strings.TrimSpace(adornmentPath.String)
	}

	var completedXP int
	if err := r.pdb.Metadata.QueryRow(ctx,
		`SELECT COALESCE(SUM(xp_required), 0)
		 FROM career_ranks
		 WHERE rank_id < ?`,
		row.Rank,
	).Scan(&completedXP); err != nil {
		return fmt.Errorf("EnrichFromMetadata sum: %w", err)
	}
	row.XPTotal = completedXP + row.CurrentXP
	return nil
}

// buildCareerRankNameDB compose le nom complet d'un rang depuis le titre EN,
// le tier et le grade. Pendant de sync.buildCareerRankName (DRY local pour
// éviter l'import croisé service↔sync).
func buildCareerRankNameDB(titleEN, tierType string, grade sql.NullInt64) string {
	titleEN = strings.TrimSpace(titleEN)
	tierType = strings.TrimSpace(tierType)
	if titleEN == "" {
		return ""
	}
	parts := []string{titleEN}
	if tierType != "" {
		parts = append(parts, tierType)
	}
	if grade.Valid && grade.Int64 > 0 {
		parts = append(parts, fmt.Sprintf("%d", grade.Int64))
	}
	return strings.Join(parts, " ")
}
