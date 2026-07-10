package livesync

// csr_match.go — CSR PAR MATCH Halo 5 : projette le CurrentCsr (post-match) du carnage
// arena vers match_skill_rank (rating_type='CSR'). La vue match_skill_rank_latest
// priorise CSR > LUSR → les matchs classés affichent le CSR, les sociaux le LUSR.
// Partagé par le hook live (PostScore) et l'outil de backfill (cmd/h5-csr-match-backfill).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/persist"
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
	pp := persist.NewPlayerPersister(playerDB)
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
		// Écritures per-match via la couche persist (PlayerPersister), jamais en
		// ExecContext direct — invariant ADR 0019 (E8/DEC-8). Le CSR va dans
		// match_skill_rank (append-only), le rang SR dans career_progression (dédup
		// par xuid+recorded_at, fait par le persister).
		var skill *persist.SkillRankInsert
		if ranked.Bool {
			skill = buildPerMatchCSRInsert(id, stat.CurrentCsr)
		}
		var career *persist.CareerProgressionInsert
		if stat.XpInfo != nil {
			career = buildCareerProgressionInsert(xuid, stat.XpInfo.SpartanRank, stat.XpInfo.TotalXP, start)
		}
		csrW, srW, err := pp.PersistPerMatchRating(ctx, skill, career)
		if err != nil {
			slog.WarnContext(ctx, "h5 PersistPerMatchRatings: persist per-match rating échoué",
				"err", err, "match_id", id)
			continue
		}
		if csrW {
			csrN++
		}
		if srW {
			srN++
		}
	}
	return csrN, srN
}

// buildPerMatchCSRInsert construit la row match_skill_rank (rating_type='CSR',
// playlist_group='h5_arena') depuis le CurrentCsr post-match. nil si pas de CSR
// (placement). Le start_time n'est PAS porté : la vue _latest joint match_registry
// pour le temps canonique (parité avec persistSkillRank du flux Infinite).
func buildPerMatchCSRInsert(matchID string, cur *halo5.H5Csr) *persist.SkillRankInsert {
	if cur == nil {
		return nil
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
	rv := float64(cur.Csr)
	pg := "h5_arena"
	return &persist.SkillRankInsert{
		MatchID:       matchID,
		RatingType:    "CSR",
		RatingValue:   &rv,
		Tier:          &tier,
		SubTier:       &sub,
		TierLabel:     &label,
		PlaylistGroup: &pg,
	}
}

// buildCareerProgressionInsert construit le snapshot career_progression (rang XP H5)
// depuis le SR. nil si SR hors borne ou recorded_at absent. rank_name = "SR N"
// (title-aware) via halo5.SpartanRankLabel (sinon la Home retombe sur "Rang N").
// Dérivation SR→XP via halo5.SpartanRankProgression (source unique). La déduplication
// par (xuid, recorded_at) est faite par PlayerPersister.PersistPerMatchRating.
func buildCareerProgressionInsert(xuid string, spartanRank, totalXP int, start sql.NullTime) *persist.CareerProgressionInsert {
	if !start.Valid {
		return nil
	}
	currentXP, xpForNext, xpTotal, isMax, ok := halo5.SpartanRankProgression(spartanRank, totalXP)
	if !ok {
		return nil
	}
	return &persist.CareerProgressionInsert{
		XUID:          xuid,
		Rank:          spartanRank,
		RankName:      halo5.SpartanRankLabel(spartanRank),
		CurrentXP:     currentXP,
		XPForNextRank: xpForNext,
		XPTotal:       xpTotal,
		IsMaxRank:     isMax,
		RecordedAt:    start.Time,
	}
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
