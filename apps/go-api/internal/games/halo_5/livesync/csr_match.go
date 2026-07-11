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
	"levelup/go-api/internal/sync/skill"
)

// carnageGetter — sous-ensemble de halo5.CaptureSource réellement utilisé par le CSR
// par match (interface-segregation). Permet de tester PersistPerMatchRatings sans
// implémenter tout h5Source (dont un type de retour non exporté). *halo5.Client et
// toute CaptureSource le satisfont.
type carnageGetter interface {
	GetMatchCarnage(ctx context.Context, matchID, mode string) (*halo5.H5CarnageResponse, error)
}

// PerMatchRatingsSummary — bilan instrumenté d'un run PersistPerMatchRatings :
// écritures + ventilation des skips PAR RAISON. Aucun skip n'est silencieux (règle
// logging n°3 : jamais de continue avalé) — chaque skip est compté ici ET loggé
// (Debug par match, Info agrégé côté caller). Les compteurs départagent la cause D2
// (carnage KO / joueur absent / placement CSR null).
type PerMatchRatingsSummary struct {
	Processed        int // matchs parcourus (len(matchIDs))
	CSRWritten       int // lignes match_skill_rank CSR écrites
	SRWritten        int // snapshots career_progression écrits
	SkipRegistry     int // lecture match_registry KO (match absent / erreur SQL)
	SkipCarnage      int // fetch carnage KO (token mort, 404, réseau)
	SkipOwnerAbsent  int // joueur (gamertag) absent du carnage
	PlacementCSRNull int // match CLASSÉ mais CurrentCsr nil (placement présumé)
	SkipPersist      int // persist per-match échoué
}

// PersistPerMatchRatings écrit, pour chaque matchID, depuis UN SEUL fetch du carnage :
//   - le CSR post-match (CurrentCsr) dans match_skill_rank si le match est CLASSÉ ;
//   - le rang SR (XpInfo.SpartanRank/TotalXP) dans career_progression (snapshot par
//     match → rang/XP carrière dans le temps, title-agnostic — C3).
//
// Best-effort par match. Retourne un PerMatchRatingsSummary : chaque skip est compté
// par raison et loggé (aucun continue silencieux). Append-only pour le CSR ;
// career_progression dédupliqué par (xuid, recorded_at).
func PersistPerMatchRatings(ctx context.Context, src carnageGetter, playerDB, shared *sql.DB, gamertag, xuid string, matchIDs []string) PerMatchRatingsSummary {
	pp := persist.NewPlayerPersister(playerDB)
	sum := PerMatchRatingsSummary{Processed: len(matchIDs)}
	for _, id := range matchIDs {
		var ranked sql.NullBool
		var start sql.NullTime
		if err := shared.QueryRowContext(ctx,
			`SELECT COALESCE(is_ranked, FALSE), COALESCE(start_time_utc, start_time) FROM match_registry WHERE match_id = ?`,
			id).Scan(&ranked, &start); err != nil {
			sum.SkipRegistry++
			slog.DebugContext(ctx, "h5 PersistPerMatchRatings: skip lecture registre", "match_id", id, "err", err)
			continue
		}
		carnage, cerr := src.GetMatchCarnage(ctx, id, "arena")
		if cerr != nil {
			sum.SkipCarnage++
			slog.DebugContext(ctx, "h5 PersistPerMatchRatings: skip carnage KO", "match_id", id, "err", cerr)
			continue
		}
		stat := ownerStat(carnage, gamertag)
		if stat == nil {
			sum.SkipOwnerAbsent++
			slog.DebugContext(ctx, "h5 PersistPerMatchRatings: skip joueur absent du carnage", "match_id", id, "gamertag", gamertag)
			continue
		}
		// Match classé sans CurrentCsr = placement présumé (compté à part pour D2/DEC-4).
		if ranked.Bool && stat.CurrentCsr == nil {
			sum.PlacementCSRNull++
			slog.DebugContext(ctx, "h5 PersistPerMatchRatings: classé sans CSR (placement présumé)", "match_id", id)
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
			sum.SkipPersist++
			slog.WarnContext(ctx, "h5 PersistPerMatchRatings: persist per-match rating échoué",
				"err", err, "match_id", id)
			continue
		}
		if csrW {
			sum.CSRWritten++
		}
		if srW {
			sum.SRWritten++
		}
	}
	slog.InfoContext(ctx, "h5 PersistPerMatchRatings: bilan",
		"gamertag", gamertag, "processed", sum.Processed,
		"csr_written", sum.CSRWritten, "sr_written", sum.SRWritten,
		"skip_registry", sum.SkipRegistry, "skip_carnage", sum.SkipCarnage,
		"skip_owner_absent", sum.SkipOwnerAbsent, "placement_csr_null", sum.PlacementCSRNull,
		"skip_persist", sum.SkipPersist)
	return sum
}

// buildPerMatchCSRInsert construit la row match_skill_rank (rating_type='CSR',
// playlist_group='h5_arena') depuis le CurrentCsr post-match. Le start_time n'est
// PAS porté : la vue _latest joint match_registry pour le temps canonique (parité
// avec persistSkillRank du flux Infinite).
//
// cur == nil sur un match CLASSÉ = PLACEMENT (le carnage arena H5 ne porte pas de
// CurrentCsr tant que le joueur n'est pas classé sur la playlist ; cause prouvée à
// 100 % des matchs classés sans CSR — cf. LOT D). On écrit alors une ligne
// « Placement » (tier=TierLabelPlacement, rating_value=0 pour la contrainte NOT NULL
// de la player DB) : le header match view ignore la valeur pour ce tier (buildRankBlock,
// isPlacement) et affiche « Placement » au lieu d'un rang vide, ET --missing-only ne
// re-fetch plus ces matchs (ils portent désormais une ligne CSR). DEC-4.
func buildPerMatchCSRInsert(matchID string, cur *halo5.H5Csr) *persist.SkillRankInsert {
	pg := "h5_arena"
	if cur == nil {
		zero := 0.0
		tier := skill.TierLabelPlacement
		label := skill.TierLabelPlacement
		sub := 0
		return &persist.SkillRankInsert{
			MatchID:       matchID,
			RatingType:    "CSR",
			RatingValue:   &zero,
			Tier:          &tier,
			SubTier:       &sub,
			TierLabel:     &label,
			PlaylistGroup: &pg,
		}
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
