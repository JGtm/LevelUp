// Package sync — csr_writes.go : extraction et persistance des CSR par-match
// depuis le payload skill Halo (RankRecap.PostMatchCsr).
//
// Le sync nominal fetche déjà l'endpoint /hi/matches/{id}/skill pour tous les
// matchs (cf. processMatch + le path fetchedMatch dans engine.go). Ce fichier
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
	"database/sql"
	"fmt"
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

// formatCSRTierLabel construit le tier_label affiché côté front. Format :
//   - placement : "Placement (N restants)"
//   - Onyx (pas de sous-tier) : "Onyx 1850"
//   - autres tiers (Bronze…Diamond) : "Or 3" (FR + sub_tier romain ou arabe)
func formatCSRTierLabel(tier, tierFR string, subTier int, value int, measurementRemaining int) string {
	if measurementRemaining > 0 || tier == "" {
		return fmt.Sprintf("Placement (%d restant%s)", measurementRemaining, pluralS(measurementRemaining))
	}
	if tier == TierOnyx {
		return fmt.Sprintf("Onyx %d", value)
	}
	if subTier > 0 {
		return fmt.Sprintf("%s %d", tierFR, subTier)
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
	if post.MeasurementMatchesRemaining > 0 || post.Tier == "" {
		row.Tier = TierLabelPlacement
		row.TierFR = TierLabelPlacement
		row.SubTier = 0
		row.TierLabel = formatCSRTierLabel("", "", 0, 0, post.MeasurementMatchesRemaining)
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

// UpsertCSRRow écrit ou remplace une ligne CSR dans match_skill_rank côté
// player DB. ON CONFLICT (match_id) DO UPDATE SET rating_type='CSR' : si une
// ligne LUSR existait pour ce match, elle est remplacée par la CSR (un match
// classé n'a jamais de LUSR valide — le LUSR est exclu des matchs ranked par
// loadLUSRMatchData). La garde-fou SQL inverse (LUSR ne peut pas écraser CSR)
// reste dans upsertLUSRRatings et n'est pas affectée.
func UpsertCSRRow(playerDB *sql.DB, row *MatchCSRRow) error {
	if row == nil {
		return nil
	}
	now := time.Now().UTC()
	_, err := playerDB.Exec(`
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, rating_deviation,
			 tier, tier_fr, sub_tier, tier_label,
			 rating_delta, playlist_group, start_time,
			 measurement_matches_remaining,
			 created_at, updated_at)
		VALUES (?, 'CSR', ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (match_id) DO UPDATE SET
			rating_type                   = 'CSR',
			rating_value                  = EXCLUDED.rating_value,
			rating_deviation              = NULL,
			tier                          = EXCLUDED.tier,
			tier_fr                       = EXCLUDED.tier_fr,
			sub_tier                      = EXCLUDED.sub_tier,
			tier_label                    = EXCLUDED.tier_label,
			rating_delta                  = EXCLUDED.rating_delta,
			playlist_group                = EXCLUDED.playlist_group,
			start_time                    = EXCLUDED.start_time,
			measurement_matches_remaining = EXCLUDED.measurement_matches_remaining,
			updated_at                    = EXCLUDED.updated_at`,
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
