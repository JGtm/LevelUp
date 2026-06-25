package livesync

// csr_match.go — CSR PAR MATCH Halo 5 : projette le CurrentCsr (post-match) du carnage
// arena vers match_skill_rank (rating_type='CSR'). La vue match_skill_rank_latest
// priorise CSR > LUSR → les matchs classés affichent le CSR, les sociaux le LUSR.
// Partagé par le hook live (PostScore) et l'outil de backfill (cmd/h5-csr-match-backfill).

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	halo5 "levelup/go-api/internal/games/halo_5"
)

// PersistPerMatchRatings écrit, pour chaque matchID, depuis UN SEUL fetch du carnage :
//   - le CSR post-match (CurrentCsr) dans match_skill_rank si le match est CLASSÉ ;
//   - le rang SR (XpInfo.SpartanRank/TotalXP) dans career_progression (snapshot par
//     match → rang/XP carrière dans le temps, title-agnostic — C3).
//
// Best-effort par match (saute carnage KO, social pour le CSR, SR hors borne). xuid =
// xuid du joueur (clé career_progression). Retourne (csrÉcrits, srÉcrits). Append-only
// pour le CSR ; career_progression dédupliqué par (xuid, recorded_at).
func PersistPerMatchRatings(ctx context.Context, src halo5.CaptureSource, playerDB, shared *sql.DB, gamertag, xuid string, matchIDs []string) (int, int) {
	csrN, srN := 0, 0
	for _, id := range matchIDs {
		var ranked sql.NullBool
		var start sql.NullTime
		if err := shared.QueryRowContext(ctx,
			`SELECT COALESCE(is_ranked, FALSE), COALESCE(start_time_utc, start_time) FROM match_registry WHERE match_id = ?`,
			id).Scan(&ranked, &start); err != nil {
			continue
		}
		carnage, cerr := src.GetMatchCarnage(ctx, id, "arena")
		if cerr != nil {
			continue
		}
		stat := ownerStat(carnage, gamertag)
		if stat == nil {
			continue
		}
		if ranked.Bool && writePerMatchCSR(ctx, playerDB, id, stat.CurrentCsr, start) {
			csrN++
		}
		if stat.XpInfo != nil && writeCareerSR(ctx, playerDB, xuid, stat.XpInfo.SpartanRank, stat.XpInfo.TotalXP, start) {
			srN++
		}
	}
	return csrN, srN
}

// writePerMatchCSR insère une ligne CSR (match_skill_rank). false si pas de CSR
// (placement) ou erreur.
func writePerMatchCSR(ctx context.Context, playerDB *sql.DB, matchID string, cur *halo5.H5Csr, start sql.NullTime) bool {
	if cur == nil {
		return false
	}
	tier := h5DesignationTierEN(cur.DesignationId)
	sub := cur.Tier
	label := tier
	if tier != "" && !strings.EqualFold(tier, "Onyx") && sub > 0 {
		label = fmt.Sprintf("%s %d", tier, sub)
	}
	if strings.EqualFold(tier, "Onyx") {
		sub = 0
	}
	_, err := playerDB.ExecContext(ctx, `
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, tier, sub_tier, tier_label, playlist_group, start_time)
		VALUES (?, 'CSR', ?, ?, ?, ?, 'h5_arena', ?)`,
		matchID, float64(cur.Csr), tier, sub, label, start)
	return err == nil
}

// writeCareerSR insère un snapshot de rang SR dans career_progression (rang XP H5),
// dédupliqué par (xuid, recorded_at). false si SR hors borne, recorded_at absent ou
// snapshot déjà présent. Dérivation SR→XP via halo5.SpartanRankProgression (source unique).
func writeCareerSR(ctx context.Context, playerDB *sql.DB, xuid string, spartanRank, totalXP int, start sql.NullTime) bool {
	if !start.Valid {
		return false
	}
	currentXP, xpForNext, xpTotal, isMax, ok := halo5.SpartanRankProgression(spartanRank, totalXP)
	if !ok {
		return false
	}
	// rank_name = "SR N" (title-aware) : sinon la Home retombe sur le fallback
	// générique "Rang N" (career.rank_catalog = not_exposed pour h5). Source unique
	// du libellé : halo5.SpartanRankLabel (partagé avec l'asset ref canonique).
	res, err := playerDB.ExecContext(ctx, `
		INSERT INTO career_progression (xuid, rank, rank_name, current_xp, xp_for_next_rank, xp_total, is_max_rank, recorded_at)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?
		WHERE NOT EXISTS (SELECT 1 FROM career_progression WHERE xuid = ? AND recorded_at = ?)`,
		xuid, spartanRank, halo5.SpartanRankLabel(spartanRank), currentXP, xpForNext, xpTotal, isMax, start.Time, xuid, start.Time)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// ownerStat trouve l'entrée du joueur (par gamertag) dans le carnage. nil si absent.
func ownerStat(c *halo5.H5CarnageResponse, gamertag string) *halo5.H5CarnagePlayer {
	if c == nil {
		return nil
	}
	for i := range c.PlayerStats {
		if c.PlayerStats[i].Player.Gamertag == gamertag {
			return &c.PlayerStats[i]
		}
	}
	return nil
}
