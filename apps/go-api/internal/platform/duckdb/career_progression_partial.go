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
	// LastFetchStatus (Phase 6 PLAN_V2) trace l'issue du dernier fetch live.
	// Valeurs : "ok" / "api_empty" / "forbidden_403" / "auth_missing" / "failed".
	// Set même quand tous les autres champs sont nil (signal "j'ai essayé").
	LastFetchStatus *string
}

// IsEmpty retourne true si aucun champ n'est set.
// IsEmpty retourne true si aucun champ de DATA n'est set. Le LastFetchStatus
// est ignoré ici : un partial avec status="forbidden_403" et tout le reste nil
// est considéré "empty data" mais on l'écrira quand même pour tracer la
// tentative (cf. HasOnlyStatus).
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

// HasOnlyStatus retourne true si seul LastFetchStatus est set (data vide).
// Sert au caller à décider s'il insère quand même pour tracer la tentative.
func (p *CareerProgressionPartial) HasOnlyStatus() bool {
	if p == nil || p.LastFetchStatus == nil {
		return false
	}
	return p.IsEmpty()
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
	// Phase 6 PLAN_V2 : un partial avec uniquement LastFetchStatus est inséré
	// pour tracer une tentative qui n'a rien rendu (ex: 403 sur customization
	// pour XxDaemonGamerxX). Sinon vide-vide → skip.
	if partial == nil {
		return false, nil
	}
	if partial.IsEmpty() && partial.LastFetchStatus == nil {
		return false, nil
	}

	last, err := r.LoadLastCareerRank(ctx, xuid)
	if err != nil {
		return false, fmt.Errorf("InsertCareerProgressionPartial load last: %w", err)
	}
	// Skip uniquement si le partial est strictement identique à la last
	// ET qu'il n'apporte pas de nouveau status (un status est toujours utile à
	// tracer même si data identique : permet de mesurer le taux de succès).
	if partial.MatchesLast(last) && partial.LastFetchStatus == nil {
		return false, nil
	}

	cols, placeholders, args := buildPartialInsertColumns(xuid, partial)
	sql := fmt.Sprintf(`INSERT INTO career_progression (%s) VALUES (%s)`,
		strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	if _, err := r.pdb.Player.Exec(ctx, sql, args...); err != nil {
		return false, fmt.Errorf("InsertCareerProgressionPartial exec: %w", err)
	}
	return true, nil
}

// buildPartialInsertColumns prépare cols + placeholders + args pour l'INSERT
// dynamique. Extrait pour respecter la règle 80L par fonction (arch-rules).
// Inclut systématiquement xuid + recorded_at ; les autres colonnes sont
// présentes uniquement si le champ correspondant est set.
func buildPartialInsertColumns(xuid string, p *CareerProgressionPartial) ([]string, []string, []interface{}) {
	cols := []string{"xuid", "recorded_at"}
	placeholders := []string{"?", "?"}
	args := []interface{}{xuid, time.Now().UTC()}

	add := func(name string, value interface{}) {
		cols = append(cols, name)
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}
	if p.Rank != nil {
		add("rank", *p.Rank)
	}
	if p.CurrentXP != nil {
		add("current_xp", *p.CurrentXP)
	}
	if p.XPForNextRank != nil {
		add("xp_for_next_rank", *p.XPForNextRank)
	}
	if p.XPTotal != nil {
		add("xp_total", *p.XPTotal)
	}
	if p.IsMaxRank != nil {
		add("is_max_rank", *p.IsMaxRank)
	}
	if p.RankName != nil {
		add("rank_name", *p.RankName)
	}
	if p.RankTier != nil {
		add("rank_tier", *p.RankTier)
	}
	if p.SpartanID != nil {
		add("spartan_id", *p.SpartanID)
	}
	if p.BannerImageURL != nil {
		add("banner_image_url", *p.BannerImageURL)
	}
	if p.EmblemImageURL != nil {
		add("emblem_image_url", *p.EmblemImageURL)
	}
	if p.BackdropImageURL != nil {
		add("backdrop_image_url", *p.BackdropImageURL)
	}
	if p.AdornmentPath != nil {
		add("adornment_path", *p.AdornmentPath)
	}
	if p.LastFetchStatus != nil {
		add("last_fetch_status", *p.LastFetchStatus)
	}
	return cols, placeholders, args
}
