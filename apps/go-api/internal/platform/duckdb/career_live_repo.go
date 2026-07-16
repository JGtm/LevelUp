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

	"levelup/go-api/internal/domain"
)

// CareerRankRow / CareerProgressionPartial : DTOs relocalisés dans domain
// (Phase 2 — le service ne dépend plus de duckdb). Alias conservés pour le code
// interne au package duckdb. La définition + les méthodes vivent dans
// internal/domain/career_live.go.
type CareerRankRow = domain.CareerRankRow

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
//
// Les champs CARRIÈRE (rank, rank_name, rank_tier, current_xp, xp_for_next_rank,
// xp_total, is_max_rank) portent TOUS un FILTER WHERE rank IS NOT NULL. Raison :
// career_progression mélange deux types de lignes — les snapshots CARRIÈRE
// (rank renseigné, écrits par le sync de rang) ET les lignes APPEARANCE-ONLY
// (spartan_id/emblem/banner renseignés, rank/current_xp/xp_* = NULL/0 — écrites
// par le backfill d'apparence H5, cf. appearance_persist). Sans ce FILTER,
// ARG_MAX(current_xp, recorded_at) pickait la ligne appearance la PLUS RÉCENTE
// (current_xp=0, mais rank=NULL ignoré par ARG_MAX → rank correct) → barre XP
// vide alors que la vraie ligne carrière portait l'XP. Le FILTER lit les champs
// carrière depuis la dernière ligne RÉELLEMENT carrière, indépendamment des
// lignes appearance. (Couvre aussi l'ancien cas "partial" : xp_for_next_rank
// NULL → ARG_MAX le saute de toute façon.) Les champs APPEARANCE gardent leur
// propre FILTER (dernière ligne non vide). Défense en profondeur S1.
//
// Note `xuid || ”` : workaround d'une corruption d'index ART connue sur
// player_db (cf. docs/INCIDENT_2026-05-20_match_participants_index.md ET
// diag 2026-05-21 sur career_progression). Sans cette concat, le filter
// pushdown sur l'index PK retournait un sous-ensemble strict des rows et
// l'ARG_MAX(banner_image_url) FILTER pickait un snapshot vide alors qu'une
// row récente non visible portait la bannière. La concat défait le pushdown
// (force un table-scan complet — perf négligeable, table < 1k rows par
// joueur sur la cadence 5 min). Migration `rebuild_career_progression…`
// reconstruit la table pour éliminer la corruption à la source, mais ce
// workaround reste en place comme défense permanente (DuckDB est connu pour
// ces régressions d'index, cf. duckdb/duckdb#9999 et apparentés).
//
// Bannière/emblème/backdrop : champs d'apparence INDÉPENDANTS (directive
// produit 2026-07-08) — chacun sert sa dernière valeur non vide, sans aucun
// couplage entre eux (« jamais vide » : un champ irrésoluble au dernier
// snapshot conserve sa dernière valeur connue). Cas réel documenté : les
// emblèmes nouvelle génération (`<id>-SpartanEmblem`, ex. 3806589 JGtm
// 2026-07-03) n'ont AUCUNE nameplate upstream (absents de mapping.json,
// aucune cfg positive, 404 CDN) — la bannière servie reste alors la dernière
// connue jusqu'à publication Microsoft (auto-réparation au refresh suivant).
const qLoadLastCareerRank = `
SELECT
    COALESCE(ARG_MAX(rank,              recorded_at) FILTER (WHERE rank IS NOT NULL), 0)                     AS rank,
    NULLIF(TRIM(ARG_MAX(rank_name,      recorded_at) FILTER (WHERE rank IS NOT NULL)), '')                  AS rank_name,
    NULLIF(TRIM(ARG_MAX(rank_tier,      recorded_at) FILTER (WHERE rank IS NOT NULL)), '')                  AS rank_tier,
    COALESCE(ARG_MAX(current_xp,        recorded_at) FILTER (WHERE rank IS NOT NULL), 0)                    AS current_xp,
    COALESCE(ARG_MAX(xp_for_next_rank,  recorded_at) FILTER (WHERE rank IS NOT NULL), 0)                    AS xp_for_next_rank,
    COALESCE(ARG_MAX(xp_total,          recorded_at) FILTER (WHERE rank IS NOT NULL), 0)                    AS xp_total,
    COALESCE(ARG_MAX(is_max_rank,       recorded_at) FILTER (WHERE rank IS NOT NULL), FALSE)                AS is_max_rank,
    ARG_MAX(spartan_id,         recorded_at) FILTER (WHERE NULLIF(TRIM(spartan_id),         '') IS NOT NULL) AS spartan_id,
    ARG_MAX(banner_image_url,   recorded_at) FILTER (WHERE NULLIF(TRIM(banner_image_url),   '') IS NOT NULL) AS banner_image_url,
    ARG_MAX(emblem_image_url,   recorded_at) FILTER (WHERE NULLIF(TRIM(emblem_image_url),   '') IS NOT NULL) AS emblem_image_url,
    ARG_MAX(backdrop_image_url, recorded_at) FILTER (WHERE NULLIF(TRIM(backdrop_image_url), '') IS NOT NULL) AS backdrop_image_url,
    ARG_MAX(adornment_path,     recorded_at) FILTER (WHERE NULLIF(TRIM(adornment_path),     '') IS NOT NULL) AS adornment_path
FROM career_progression
WHERE xuid || '' = ?`

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
	rows, err := r.pdb.Player.QueryRowRecovered(ctx, qLoadLastCareerRank, xuid)
	if err != nil {
		if err == sql.ErrNoRows || isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("LoadLastCareerRank: %w", err)
	}
	defer rows.Close()
	if err := rows.Scan(
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
	); err != nil {
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
		// Table career_ranks absente (titre sans catalogue de rangs de carrière,
		// ex. Halo 5 dont le metadata n'a pas cette table) : on dégrade comme un
		// ErrNoRows — pas d'enrichissement, pas d'erreur dure qui casserait le
		// flow live carrière (S1 vol.3b).
		if errors.Is(err, sql.ErrNoRows) || isTableNotFoundErr(err) {
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
		// Cohérent avec la 1re requête : table absente → on n'écrase pas xp_total
		// avec une somme partielle, on garde la valeur en place (best-effort).
		if errors.Is(err, sql.ErrNoRows) || isTableNotFoundErr(err) {
			return nil
		}
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
