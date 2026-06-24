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

// PersistPerMatchCSR écrit le CSR post-match du joueur dans match_skill_rank pour les
// matchIDs CLASSÉS de la liste. Best-effort par match : saute le social (is_ranked
// FALSE), le placement (CurrentCsr nil) et les carnages indisponibles. Retourne le
// nombre de lignes CSR écrites. Append-only (INSERT pur, id+written_at auto).
func PersistPerMatchCSR(ctx context.Context, src halo5.CaptureSource, playerDB, shared *sql.DB, gamertag string, matchIDs []string) int {
	written := 0
	for _, id := range matchIDs {
		var ranked sql.NullBool
		var start sql.NullTime
		if err := shared.QueryRowContext(ctx,
			`SELECT COALESCE(is_ranked, FALSE), COALESCE(start_time_utc, start_time) FROM match_registry WHERE match_id = ?`,
			id).Scan(&ranked, &start); err != nil || !ranked.Bool {
			continue
		}
		carnage, cerr := src.GetMatchCarnage(ctx, id, "arena")
		if cerr != nil {
			continue
		}
		cur := ownerCurrentCsr(carnage, gamertag)
		if cur == nil {
			continue // placement / pas de CSR sur ce match
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
		if _, err := playerDB.ExecContext(ctx, `
			INSERT INTO match_skill_rank
				(match_id, rating_type, rating_value, tier, sub_tier, tier_label, playlist_group, start_time)
			VALUES (?, 'CSR', ?, ?, ?, ?, 'h5_arena', ?)`,
			id, float64(cur.Csr), tier, sub, label, start); err != nil {
			continue
		}
		written++
	}
	return written
}

// ownerCurrentCsr trouve le CurrentCsr du joueur (par gamertag) dans le carnage. nil
// si absent / social / placement.
func ownerCurrentCsr(c *halo5.H5CarnageResponse, gamertag string) *halo5.H5Csr {
	if c == nil {
		return nil
	}
	for i := range c.PlayerStats {
		if c.PlayerStats[i].Player.Gamertag == gamertag {
			return c.PlayerStats[i].CurrentCsr
		}
	}
	return nil
}
