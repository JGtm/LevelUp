// Package sync — csr_writes.go : extraction et persistance des CSR par-match
// depuis le payload skill Halo (RankRecap.PostMatchCsr).
//
// Le sync nominal fetche déjà l'endpoint /hi/matches/{id}/skill pour tous les
// matchs (cf. fetchMatchData → le path fetchedMatch dans engine_fetch.go). Ce fichier
// branche un consommateur supplémentaire sur le même payload : pour chaque
// match classé (registry.IsRanked == TRUE), on lit RankRecap dans MatchSkillData
// et on persiste une ligne dans match_skill_rank côté player DB avec
// rating_type='CSR'.
//
// L'UPSERT remplace silencieusement une éventuelle ligne LUSR pré-existante
// sur le même match : un match classé doit toujours porter le CSR officiel
// Microsoft, pas le LUSR calculé localement (qui est exclu des matchs ranked
// dans le pipeline LUSR de toute façon — cf. loadLUSRMatchData WHERE
// COALESCE(mr.is_ranked, FALSE) = FALSE). La garde-fou SQL inverse — sur le
// LUSR — reste en place dans skill_rating_loaders.go pour empêcher l'écrasement
// CSR → LUSR.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// MatchCSRRow représente une ligne à écrire dans match_skill_rank avec
// rating_type='CSR'. Pour les matchs de placement (PostMatchCSR.Value=0
// et MeasurementMatchesRemaining > 0) :
//   - RatingValue = nil (NULL en DB)
//   - Tier = "Placement", TierFR = "Placement"
//   - TierLabel = "Placement (N restants)"
//   - RatingDelta = nil (le delta n'a pas de sens en placement)
type MatchCSRRow struct {
	MatchID                     string
	RatingValue                 *float64
	Tier                        string
	TierFR                      string
	SubTier                     int
	TierLabel                   string
	RatingDelta                 *float64
	PlaylistGroup               string
	StartTime                   time.Time
	MeasurementMatchesRemaining int
}

// tierENtoFR mappe le tier EN renvoyé par l'API Halo vers son équivalent FR.
// Les valeurs vides ("" → placement) sont gérées en amont par ExtractCSRRowIfRanked.
var tierENtoFR = map[string]string{
	TierBronze:   TierLabelBronze,
	TierSilver:   TierLabelArgent,
	TierGold:     TierLabelOr,
	TierPlatinum: TierLabelPlatine,
	TierDiamond:  TierLabelDiamant,
	TierOnyx:     TierLabelOnyx,
}

// translateTierFR retourne la traduction FR du tier EN ; retourne le tier EN
// inchangé si non reconnu (forward-compat : si Microsoft ajoute un tier).
func translateTierFR(tierEN string) string {
	if fr, ok := tierENtoFR[tierEN]; ok {
		return fr
	}
	return tierEN
}

var csrSubTierRoman = [7]string{"", "I", "II", "III", "IV", "V", "VI"}

// formatCSRTierLabel construit le tier_label affiché côté front. Format :
//   - placement : "Placement (N restants)"
//   - Onyx (pas de sous-tier) : "Onyx 1850"
//   - autres tiers (Bronze…Diamond) : "Or III" (FR + sous-tier en chiffres romains)
func formatCSRTierLabel(tier, tierFR string, subTier int, value int, measurementRemaining int) string {
	if measurementRemaining > 0 || tier == "" {
		return fmt.Sprintf("Placement (%d restant%s)", measurementRemaining, pluralS(measurementRemaining))
	}
	if tier == TierOnyx {
		return fmt.Sprintf("Onyx %d", value)
	}
	if subTier >= 1 && subTier <= 6 {
		return fmt.Sprintf("%s %s", tierFR, csrSubTierRoman[subTier])
	}
	return tierFR
}

func pluralS(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}

// ExtractCSRRowIfRanked construit une MatchCSRRow depuis le skillData d'un
// joueur sur un match donné. Retourne nil si :
//   - le match n'est pas classé (reg.IsRanked == false)
//   - le skillData n'a pas de PostMatchCSR (RankRecap absent — match social
//     mal classifié côté registry, ou résultat API tronqué)
//   - le PostMatchCSR a une signature de payload tronqué (Tier vide ET
//     MeasurementMatchesRemaining = 0) — protection contre l'écrasement
//     accidentel d'une CSR valide par un placeholder bidon issu d'un drift
//     ponctuel de l'API Halo (cf. audit garde-fous CSR 2026-05-16).
//
// Le PlaylistGroup est figé à "ranked" — toutes les playlists classées Halo
// Infinite partagent un seul rating CSR par joueur, contrairement au LUSR
// qui distingue Open vs SoloAndDuo via GetLUSRChain.
func ExtractCSRRowIfRanked(reg *MatchRegistryRow, skill *MatchSkillData) *MatchCSRRow {
	if reg == nil || !reg.IsRanked {
		return nil
	}
	if skill == nil || skill.PostMatchCSR == nil {
		return nil
	}
	post := skill.PostMatchCSR

	// Garde-fou contre les payloads tronqués/erronés. Un PostMatchCsr légitime
	// porte SOIT un Tier non vide (rang stable Bronze..Onyx) SOIT un
	// MeasurementMatchesRemaining > 0 (placement en cours). Si les deux sont
	// nuls, on suppose le payload invalide et on rejette la row plutôt que
	// d'écraser une CSR existante avec un placeholder "Placement (0 restant)".
	// Le user perdrait alors une donnée Microsoft valide sur un re-fetch
	// (notamment via `--csr --force`).
	if post.Tier == "" && post.MeasurementMatchesRemaining == 0 {
		return nil
	}

	row := &MatchCSRRow{
		MatchID:                     reg.MatchID,
		Tier:                        post.Tier,
		SubTier:                     post.SubTier,
		PlaylistGroup:               PerfChainRanked,
		StartTime:                   reg.StartTime,
		MeasurementMatchesRemaining: post.MeasurementMatchesRemaining,
	}

	// Placement : pas de rating final, pas de delta significatif.
	// Note : rating_value=0.0 stocké (au lieu de NULL) pour respecter la
	// contrainte NOT NULL du schéma match_skill_rank.rating_value. Le caller
	// distingue placement vs rating réel via MeasurementMatchesRemaining > 0
	// (highest_csr/lusr filtrera 0.0 implicitement via ORDER BY rating DESC).
	if post.MeasurementMatchesRemaining > 0 || post.Tier == "" {
		row.Tier = TierLabelPlacement
		row.TierFR = TierLabelPlacement
		row.SubTier = 0
		row.TierLabel = formatCSRTierLabel("", "", 0, 0, post.MeasurementMatchesRemaining)
		zero := 0.0
		row.RatingValue = &zero
		return row
	}

	// Rang stable : value, tier_fr, tier_label, delta.
	v := post.Value
	row.RatingValue = &v
	row.TierFR = translateTierFR(post.Tier)
	row.TierLabel = formatCSRTierLabel(post.Tier, row.TierFR, post.SubTier, int(post.Value), 0)
	if skill.PreMatchCSR != nil {
		delta := post.Value - skill.PreMatchCSR.Value
		row.RatingDelta = &delta
	}
	return row
}

// UpsertCSRRow écrit une ligne CSR dans match_skill_rank côté player DB.
//
// **Sémantique append-only** (Phase 2.E du refactor ART) : chaque appel
// produit un INSERT pur. La table contient N versions par (match_id,
// rating_type) ; la vue `match_skill_rank_latest` expose la version la
// plus récente avec priorité CSR > LUSR.
//
// Avant la Phase 2.E, cette fonction faisait `INSERT ... ON CONFLICT
// (match_id) DO UPDATE` ce qui déclenchait empiriquement le bug ART
// DuckDB sous concurrence (cf. csr_art_repro_test.go : 19/20 workers
// crashent). L'INSERT pur élimine ce risque par construction.
//
// L'idempotence fonctionnelle est portée par la vue latest : un appel
// répété avec les mêmes données produit N rows physiques mais une seule
// version visible (la plus récente, donc identique aux précédentes).
func UpsertCSRRow(ctx context.Context, playerDB *sql.DB, row *MatchCSRRow) error {
	if row == nil {
		return nil
	}
	now := time.Now().UTC()
	_, err := playerDB.ExecContext(ctx, `
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, rating_deviation,
			 tier, tier_fr, sub_tier, tier_label,
			 rating_delta, playlist_group, start_time,
			 measurement_matches_remaining,
			 created_at, updated_at)
		VALUES (?, 'CSR', ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.MatchID, row.RatingValue,
		row.Tier, row.TierFR, row.SubTier, row.TierLabel,
		row.RatingDelta, row.PlaylistGroup, row.StartTime,
		row.MeasurementMatchesRemaining,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("UpsertCSRRow(%s): %w", row.MatchID, err)
	}
	return nil
}

// BackfillCSRFromShared projette le CSR par-match d'un joueur depuis
// shared.match_csrs vers player.match_skill_rank (rating_type='CSR').
//
// Contexte : l'import OpenSpartan écrit le CSR par-match dans shared.match_csrs
// (extrait de RankRecap, cf. ExtractAllSharedCSRRows), mais l'UI lit le rating
// via player.match_skill_rank_latest. Cette fonction reprojette donc les lignes
// shared du joueur vers sa player DB, en réutilisant le formatage tier_fr /
// placement de la chaîne CSR. Pur local — aucun appel API.
//
// La shared row a déjà appliqué le formatage placement (tier="Placement",
// rating_value NULL) ; on réplique ici l'invariant player (rating_value=0.0 en
// placement pour la contrainte NOT NULL) et on dérive tier_fr.
//
// Idempotent : les matchs portant déjà une ligne CSR côté player DB sont skippés
// (évite la prolifération de versions append-only au réimport). Retourne le
// nombre de lignes écrites.
func BackfillCSRFromShared(ctx context.Context, sharedDB, playerDB *sql.DB, xuid string) (int, error) {
	if sharedDB == nil || playerDB == nil {
		return 0, fmt.Errorf("BackfillCSRFromShared: nil DB")
	}
	if strings.TrimSpace(xuid) == "" {
		return 0, fmt.Errorf("BackfillCSRFromShared: xuid vide")
	}

	existing, err := loadExistingCSRMatchIDs(ctx, playerDB)
	if err != nil {
		return 0, err
	}

	rows, err := sharedDB.QueryContext(ctx, `
		SELECT c.match_id, c.rating_value, c.tier, c.sub_tier, c.tier_label,
		       c.rating_delta, c.measurement_matches_remaining, r.start_time
		FROM match_csrs_latest c
		JOIN match_registry r ON r.match_id = c.match_id
		WHERE c.xuid = ? AND c.rating_type = 'CSR'`, xuid)
	if err != nil {
		return 0, fmt.Errorf("BackfillCSRFromShared query: %w", err)
	}
	defer rows.Close()

	written := 0
	for rows.Next() {
		var (
			matchID   string
			ratingVal sql.NullFloat64
			tier      sql.NullString
			subTier   sql.NullInt64
			tierLabel sql.NullString
			ratingDel sql.NullFloat64
			measRem   sql.NullInt64
			startTime sql.NullTime
		)
		if err := rows.Scan(&matchID, &ratingVal, &tier, &subTier, &tierLabel,
			&ratingDel, &measRem, &startTime); err != nil {
			return written, fmt.Errorf("BackfillCSRFromShared scan: %w", err)
		}
		if existing[matchID] {
			continue
		}
		row := &MatchCSRRow{
			MatchID:                     matchID,
			Tier:                        tier.String,
			TierFR:                      translateTierFR(tier.String),
			SubTier:                     int(subTier.Int64),
			TierLabel:                   tierLabel.String,
			PlaylistGroup:               PerfChainRanked,
			StartTime:                   startTime.Time,
			MeasurementMatchesRemaining: int(measRem.Int64),
		}
		// Placement : rating_value=0.0 (NOT NULL côté player), pas de delta.
		if ratingVal.Valid {
			v := ratingVal.Float64
			row.RatingValue = &v
			if ratingDel.Valid {
				d := ratingDel.Float64
				row.RatingDelta = &d
			}
		} else {
			zero := 0.0
			row.RatingValue = &zero
		}
		if err := UpsertCSRRow(ctx, playerDB, row); err != nil {
			return written, err
		}
		written++
	}
	if err := rows.Err(); err != nil {
		return written, fmt.Errorf("BackfillCSRFromShared iterate: %w", err)
	}
	return written, nil
}

// loadExistingCSRMatchIDs retourne l'ensemble des match_id ayant déjà une ligne
// CSR côté player DB (toutes versions append-only confondues).
func loadExistingCSRMatchIDs(ctx context.Context, playerDB *sql.DB) (map[string]bool, error) {
	rows, err := playerDB.QueryContext(ctx,
		`SELECT DISTINCT match_id FROM match_skill_rank WHERE rating_type = 'CSR'`)
	if err != nil {
		return nil, fmt.Errorf("loadExistingCSRMatchIDs: %w", err)
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("loadExistingCSRMatchIDs scan: %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}
