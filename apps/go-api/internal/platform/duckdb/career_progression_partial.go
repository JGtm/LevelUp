// Package duckdb — career_progression_partial.go : INSERT "partial" pour
// career_progression où chaque colonne est indépendante.
//
// Pourquoi : le legacy InsertCareerProgressionIfChanged écrivait toujours
// TOUS les champs (live + carry-forward depuis dbLast). Conséquence : si le
// live rendait `BannerImageURL=""` alors qu'une valeur existait en DB, la
// nouvelle ligne contenait soit le vide live soit la copie dbLast — dans
// les 2 cas la ligne n'apportait aucune information neuve sur la bannière.
//
// Le partial INSERT n'écrit QUE les champs effectivement rendus non-vides
// par l'API live. Les autres restent NULL dans la nouvelle ligne. Combiné
// au `ARG_MAX FILTER WHERE NULLIF(TRIM(col), ”) IS NOT NULL` côté lecture,
// chaque champ a sa propre histoire de fraîcheur indépendante.
//
// Exemple : si l'API rend custom OK mais progress nil (Halo Economy down),
// l'INSERT inclut spartan_id/banner/emblem/backdrop mais omet rank/xp.
// Le rang affiché à la prochaine lecture reste le dernier connu, la
// bannière utilise la nouvelle valeur.
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CareerProgressionPartial = sous-ensemble des colonnes de career_progression.
// Chaque champ pointer = nil (non renseigné par l'API) ou set (rendu non-vide).
// IsEmpty() permet de skip un INSERT bidon.
type CareerProgressionPartial struct {
	Rank             *int
	CurrentXP        *int
	XPForNextRank    *int
	XPTotal          *int
	IsMaxRank        *bool
	RankName         *string
	RankTier         *string
	SpartanID        *string
	BannerImageURL   *string
	EmblemImageURL   *string
	BackdropImageURL *string
	AdornmentPath    *string
}

// IsEmpty retourne true si aucun champ n'est set.
func (p *CareerProgressionPartial) IsEmpty() bool {
	if p == nil {
		return true
	}
	return p.Rank == nil && p.CurrentXP == nil && p.XPForNextRank == nil &&
		p.XPTotal == nil && p.IsMaxRank == nil &&
		p.RankName == nil && p.RankTier == nil &&
		p.SpartanID == nil &&
		p.BannerImageURL == nil && p.EmblemImageURL == nil &&
		p.BackdropImageURL == nil && p.AdornmentPath == nil
}

// MatchesLast retourne true si tous les champs set de p ont déjà la même valeur
// dans last. Permet de skip un INSERT redondant qui n'apporterait rien.
// Si last est nil, on considère qu'il y a forcément du nouveau (sauf si p est empty).
func (p *CareerProgressionPartial) MatchesLast(last *CareerRankRow) bool {
	if p == nil || p.IsEmpty() {
		return true
	}
	if last == nil {
		return false
	}
	if p.Rank != nil && *p.Rank != last.Rank {
		return false
	}
	if p.CurrentXP != nil && *p.CurrentXP != last.CurrentXP {
		return false
	}
	if p.XPForNextRank != nil && *p.XPForNextRank != last.XPForNextRank {
		return false
	}
	if p.XPTotal != nil && *p.XPTotal != last.XPTotal {
		return false
	}
	if p.IsMaxRank != nil && *p.IsMaxRank != last.IsMaxRank {
		return false
	}
	if p.SpartanID != nil && *p.SpartanID != last.SpartanID {
		return false
	}
	if p.BannerImageURL != nil && *p.BannerImageURL != last.BannerImageURL {
		return false
	}
	if p.EmblemImageURL != nil && *p.EmblemImageURL != last.EmblemImageURL {
		return false
	}
	if p.BackdropImageURL != nil && *p.BackdropImageURL != last.BackdropImageURL {
		return false
	}
	if p.AdornmentPath != nil && *p.AdornmentPath != last.AdornmentPath {
		return false
	}
	if p.RankName != nil && *p.RankName != last.RankName {
		return false
	}
	if p.RankTier != nil && *p.RankTier != last.RankTier {
		return false
	}
	return true
}

// InsertCareerProgressionPartial INSERT une nouvelle ligne dans
// career_progression avec UNIQUEMENT les colonnes set dans partial.
//
// Les colonnes omises sont laissées à leur DEFAULT (typiquement NULL ou 0
// suivant le schéma de la player DB). Le SELECT ARG_MAX FILTER WHERE
// NULLIF(TRIM(col), ”) IS NOT NULL ignore les chaînes vides ; pour les
// numeric (rank), un filter `WHERE rank > 0` est ajouté côté query.
//
// Retourne :
//   - (true, nil) : ligne insérée
//   - (false, nil) : partial vide OU strictement identique à la dernière connue
//   - (false, err) : erreur DB
//
// Cette méthode remplace InsertCareerProgressionIfChanged pour les chemins
// post-refactor V2. Le legacy reste pour compat tests existants.
func (r *CareerLiveRepo) InsertCareerProgressionPartial(
	ctx context.Context,
	xuid string,
	partial *CareerProgressionPartial,
) (bool, error) {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return false, fmt.Errorf("InsertCareerProgressionPartial: pdb non disponible")
	}
	if xuid == "" {
		return false, fmt.Errorf("InsertCareerProgressionPartial: xuid vide")
	}
	if partial.IsEmpty() {
		return false, nil
	}

	last, err := r.LoadLastCareerRank(ctx, xuid)
	if err != nil {
		return false, fmt.Errorf("InsertCareerProgressionPartial load last: %w", err)
	}
	if partial.MatchesLast(last) {
		return false, nil
	}

	cols := []string{"xuid", "recorded_at"}
	placeholders := []string{"?", "?"}
	args := []interface{}{xuid, time.Now().UTC()}

	add := func(name string, value interface{}) {
		cols = append(cols, name)
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}
	if partial.Rank != nil {
		add("rank", *partial.Rank)
	}
	if partial.CurrentXP != nil {
		add("current_xp", *partial.CurrentXP)
	}
	if partial.XPForNextRank != nil {
		add("xp_for_next_rank", *partial.XPForNextRank)
	}
	if partial.XPTotal != nil {
		add("xp_total", *partial.XPTotal)
	}
	if partial.IsMaxRank != nil {
		add("is_max_rank", *partial.IsMaxRank)
	}
	if partial.RankName != nil {
		add("rank_name", *partial.RankName)
	}
	if partial.RankTier != nil {
		add("rank_tier", *partial.RankTier)
	}
	if partial.SpartanID != nil {
		add("spartan_id", *partial.SpartanID)
	}
	if partial.BannerImageURL != nil {
		add("banner_image_url", *partial.BannerImageURL)
	}
	if partial.EmblemImageURL != nil {
		add("emblem_image_url", *partial.EmblemImageURL)
	}
	if partial.BackdropImageURL != nil {
		add("backdrop_image_url", *partial.BackdropImageURL)
	}
	if partial.AdornmentPath != nil {
		add("adornment_path", *partial.AdornmentPath)
	}

	sql := fmt.Sprintf(`INSERT INTO career_progression (%s) VALUES (%s)`,
		strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	if _, err := r.pdb.Player.Exec(ctx, sql, args...); err != nil {
		return false, fmt.Errorf("InsertCareerProgressionPartial exec: %w", err)
	}
	return true, nil
}
