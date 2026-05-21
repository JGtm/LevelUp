// Package sync — csr_shared_writes.go : capture CSR par-participant dans
// shared.match_csrs au sync d'un match ranked (Option A du plan pipeline CSR).
//
// Avant : ExtractCSRRowIfRanked + UpsertCSRRow ne persistaient que le CSR du
// joueur dont on syncait la DB (player.match_skill_rank). Les autres joueurs
// du match avaient leur CSR dans le payload Halo mais on le jetait.
//
// Maintenant : ExtractAllSharedCSRRows itère sur skillData[xuid] pour tous les
// participants ranked, et UpsertSharedCSRs batch INSERT dans shared.match_csrs.
// Permet les comparaisons cross-joueurs (Squad, "qui était mieux classé",
// coéquipiers en placement, etc.) sans dépendre de la player DB de chacun.
package sync

import (
	"database/sql"
	"fmt"
	"time"
)

// SharedMatchCSRRow représente une ligne shared.match_csrs prête à écrire.
// Différent de MatchCSRRow (player DB) qui porte des invariants supplémentaires
// (rating_value NOT NULL → 0.0 forcé en placement). Ici on accepte NULL via
// pointeur pour expliciter le placement state au lecteur.
type SharedMatchCSRRow struct {
	MatchID                     string
	XUID                        string
	RatingType                  string // "CSR" (LUSR pas écrit ici — c'est local)
	RatingValue                 *float64
	Tier                        string
	SubTier                     int
	TierLabel                   string
	RatingDelta                 *float64
	MeasurementMatchesRemaining int
	SeasonID                    string
	StartTime                   time.Time
}

// ExtractAllSharedCSRRows construit une SharedMatchCSRRow pour chaque joueur
// du match ranked qui a un PostMatchCSR dans le payload skill. Skip les
// joueurs sans données skill ou sans CSR (bots, joueurs non-classés du lobby).
// Skip aussi si le match n'est pas ranked.
//
// La saison est lue depuis reg.SeasonID (peuplé depuis matchInfo.SeasonId par
// transforms.go). Fallback à "" si absent — les anciens matchs auront NULL
// season_id, le display l'utilisera comme indication de saison inconnue.
func ExtractAllSharedCSRRows(reg *MatchRegistryRow, skillByXUID map[string]*MatchSkillData) []SharedMatchCSRRow {
	if reg == nil || !reg.IsRanked || len(skillByXUID) == 0 {
		return nil
	}
	seasonID := ""
	if reg.SeasonID != nil {
		seasonID = *reg.SeasonID
	}
	out := make([]SharedMatchCSRRow, 0, len(skillByXUID))
	for xuid, skill := range skillByXUID {
		if skill == nil || skill.PostMatchCSR == nil {
			continue
		}
		post := skill.PostMatchCSR
		// Garde-fou identique à ExtractCSRRowIfRanked : payload tronqué = skip.
		if post.Tier == "" && post.MeasurementMatchesRemaining == 0 {
			continue
		}
		row := SharedMatchCSRRow{
			MatchID:                     reg.MatchID,
			XUID:                        xuid,
			RatingType:                  "CSR",
			Tier:                        post.Tier,
			SubTier:                     post.SubTier,
			MeasurementMatchesRemaining: post.MeasurementMatchesRemaining,
			SeasonID:                    seasonID,
			StartTime:                   reg.StartTime,
		}
		if post.MeasurementMatchesRemaining > 0 || post.Tier == "" {
			// Placement : tier="Placement", value=NULL, label=formatté.
			row.Tier = TierLabelPlacement
			row.SubTier = 0
			row.TierLabel = formatCSRTierLabel("", "", 0, 0, post.MeasurementMatchesRemaining)
			// Pas de RatingDelta significatif en placement.
		} else {
			// Rang stable.
			v := post.Value
			row.RatingValue = &v
			row.TierLabel = formatCSRTierLabel(post.Tier, translateTierFR(post.Tier), post.SubTier, int(post.Value), 0)
			if skill.PreMatchCSR != nil {
				delta := post.Value - skill.PreMatchCSR.Value
				row.RatingDelta = &delta
			}
		}
		out = append(out, row)
	}
	return out
}

// UpsertSharedCSRs écrit ou met à jour les rows shared.match_csrs.
// ON CONFLICT (match_id, xuid) DO UPDATE SET : préserve la donnée la plus
// fraîche. Non-bloquant : retourne nil si rows est vide (rien à faire).
func UpsertSharedCSRs(sharedDB *sql.DB, rows []SharedMatchCSRRow) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, row := range rows {
		_, err := sharedDB.Exec(`
			INSERT INTO match_csrs (
				match_id, xuid, rating_type, rating_value,
				tier, sub_tier, tier_label, rating_delta,
				measurement_matches_remaining, season_id,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (match_id, xuid) DO UPDATE SET
				rating_type                   = EXCLUDED.rating_type,
				rating_value                  = EXCLUDED.rating_value,
				tier                          = EXCLUDED.tier,
				sub_tier                      = EXCLUDED.sub_tier,
				tier_label                    = EXCLUDED.tier_label,
				rating_delta                  = EXCLUDED.rating_delta,
				measurement_matches_remaining = EXCLUDED.measurement_matches_remaining,
				season_id                     = COALESCE(EXCLUDED.season_id, match_csrs.season_id),
				updated_at                    = EXCLUDED.updated_at
		`,
			row.MatchID, row.XUID, row.RatingType, row.RatingValue,
			row.Tier, row.SubTier, row.TierLabel, row.RatingDelta,
			row.MeasurementMatchesRemaining, sqlNullableString(row.SeasonID),
			now, now,
		)
		if err != nil {
			return fmt.Errorf("UpsertSharedCSRs(%s/%s): %w", row.MatchID, row.XUID, err)
		}
	}
	return nil
}

// sqlNullableString retourne une *string pour les zéro values qu'on veut
// stocker NULL en DB (au lieu de la chaîne vide).
func sqlNullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
