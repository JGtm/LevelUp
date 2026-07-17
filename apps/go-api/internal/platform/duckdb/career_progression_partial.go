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

	"levelup/go-api/internal/domain"
)

// CareerProgressionPartial : DTO relocalisé dans domain (Phase 2). Alias conservé
// pour le code interne au package duckdb ; définition + méthodes (IsEmpty,
// HasOnlyStatus, MatchesLast) dans internal/domain/career_live.go.
type CareerProgressionPartial = domain.CareerProgressionPartial

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

	// E5 (revue 2026-07) : ExecRecovered (Reopen+retry) — écriture player-DB tolérant
	// un Reopen concurrent (« database is closed »), comme les lectures du sweep.
	if _, err := r.pdb.Player.ExecRecovered(ctx, sql, args...); err != nil {
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
