package haloadapter

// provider_operations.go — passe de combat (opération active) et défis du joueur, en direct :
// jetons et xuid lus dans le contexte, filet 401/403 ([retryOnAuth]), corps brut rendu pour la
// persistance par l'appelant.

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/haloapi/infinite"
)

// GetBattlePass rend l'opération de passe de combat active du joueur du contexte ;
// available=false, error_hint="auth_required" sans authentification dans le contexte.
func (p *Provider) GetBattlePass(ctx context.Context) domain.BattlePassResponse {
	resp, _ := p.GetBattlePassWithRaw(ctx)
	return resp
}

// GetBattlePassWithRaw rend la passe de combat et le JSON brut reçu (pour persistance) ; brut nil
// en cas d'erreur ou d'absence d'authentification.
func (p *Provider) GetBattlePassWithRaw(ctx context.Context) (domain.BattlePassResponse, []byte) {
	tokens := ctxkeys.HaloTokens(ctx)
	xuid := ctxkeys.HaloXUID(ctx)
	if tokens == nil || xuid == "" {
		hint := errHintAuthRequired
		return domain.BattlePassResponse{Available: false, ErrorHint: &hint}, nil
	}
	type bpResult struct {
		resp domain.BattlePassResponse
		raw  []byte
	}
	out, err := retryOnAuth(ctx, func(c context.Context) (bpResult, error) {
		r, raw, e := p.fetchBattlePass(c, ctxkeys.HaloTokens(c), xuid)
		return bpResult{resp: r, raw: raw}, e
	})
	if err != nil {
		slog.WarnContext(ctx, "halo_provider: battle_pass fetch failed", "xuid", xuid, "err", err)
		hint := errHintFetchError
		return domain.BattlePassResponse{Available: false, ErrorHint: &hint}, nil
	}
	return out.resp, out.raw
}

// fetchBattlePass lit les opérations du joueur et rend la réponse de domaine, le JSON brut (pour
// persistance) et l'erreur éventuelle (corps illisible : brut rendu avec l'erreur).
func (p *Provider) fetchBattlePass(ctx context.Context, tokens *domain.HaloTokens, xuid string) (domain.BattlePassResponse, []byte, error) {
	dto, body, err := p.api(ctx, surfaceProviderGet, infinite.RetryProvider(), infinite.CleEconomy).GetOperations(ctx, authDe(tokens), xuid)
	if err != nil {
		return domain.BattlePassResponse{}, body, err
	}
	rank, progress, trackPath := parseBattlePassTrack(dto.ActiveOperationRewardTrackPath, dto.OperationRewardTracks)
	slog.DebugContext(ctx, "halo_provider: battle_pass fetched", "xuid", xuid, "rank", rank, "track", trackPath)

	// Pré-cache sans attente des définitions GameCMS de toutes les pistes, par le résolveur : le
	// contexte de la requête n'est pas transmis (le préchauffage survit à la réponse).
	if p.assetResolver != nil {
		var trackRefs []assets.Ref
		for _, t := range dto.OperationRewardTracks {
			if t.RewardTrackPath == "" {
				continue
			}
			trackRefs = append(trackRefs, assets.Ref{Kind: assets.KindRewardTrackDefinition, TitleID: p.titleID(), ID: t.RewardTrackPath})
		}
		if len(trackRefs) > 0 {
			go p.assetResolver.Warm(context.Background(), trackRefs...)
		}
	}
	return domain.BattlePassResponse{Available: true, Rank: &rank, Progress: &progress, RewardTrack: &trackPath}, body, nil
}

// parseBattlePassTrack extrait rang, progression et piste de l'opération active ; à défaut, de la
// première opération.
func parseBattlePassTrack(activePath string, tracks []infinite.BattlePassTrackDTO) (rank, progress int, trackPath string) {
	for _, t := range tracks {
		if t.RewardTrackPath == activePath {
			return t.CurrentProgress.Rank, t.CurrentProgress.PartialProgress, activePath
		}
	}
	if len(tracks) > 0 {
		return tracks[0].CurrentProgress.Rank, tracks[0].CurrentProgress.PartialProgress, tracks[0].RewardTrackPath
	}
	return 0, 0, activePath
}

// GetChallenges rend le résumé des défis actifs du joueur du contexte ; available=false,
// error_hint="auth_required" sans authentification dans le contexte.
func (p *Provider) GetChallenges(ctx context.Context) domain.ChallengesResponse {
	resp, _ := p.GetChallengesWithRaw(ctx)
	return resp
}

// GetChallengesWithRaw rend les défis et le JSON brut reçu (pour persistance) ; brut nil en cas
// d'erreur ou d'absence d'authentification. Les lectures concurrentes d'un même joueur sont
// dédupliquées.
func (p *Provider) GetChallengesWithRaw(ctx context.Context) (domain.ChallengesResponse, []byte) {
	tokens := ctxkeys.HaloTokens(ctx)
	xuid := ctxkeys.HaloXUID(ctx)
	if tokens == nil || xuid == "" {
		hint := errHintAuthRequired
		return domain.ChallengesResponse{Available: false, ErrorHint: &hint}, nil
	}
	type challengesFetchResult struct {
		response domain.ChallengesResponse
		rawBody  []byte
	}
	out, err := retryOnAuth(ctx, func(c context.Context) (challengesFetchResult, error) {
		v, ferr, _ := challengesFetchSFGroup.Do(p.challengesFetchKey(c, xuid), func() (interface{}, error) {
			resp, raw, fetchErr := p.fetchChallenges(c, ctxkeys.HaloTokens(c), xuid)
			if fetchErr != nil {
				return nil, fetchErr
			}
			return challengesFetchResult{response: resp, rawBody: raw}, nil
		})
		if ferr != nil {
			return challengesFetchResult{}, ferr
		}
		r, ok := v.(challengesFetchResult)
		if !ok {
			return challengesFetchResult{}, fmt.Errorf("challenges fetch invalid result type")
		}
		return r, nil
	})
	if err != nil {
		slog.WarnContext(ctx, "halo_provider: challenges fetch failed", "xuid", xuid, "err", err)
		hint := errHintFetchError
		return domain.ChallengesResponse{Available: false, ErrorHint: &hint}, nil
	}
	return out.response, out.rawBody
}

func (p *Provider) challengesFetchKey(ctx context.Context, xuid string) string {
	return xuid + "|" + p.hostFor(ctx, infinite.CleChallenges)
}

// fetchChallenges lit les decks du joueur et rend la réponse de domaine, le JSON brut et l'erreur
// éventuelle (corps illisible : brut rendu avec l'erreur).
func (p *Provider) fetchChallenges(ctx context.Context, tokens *domain.HaloTokens, xuid string) (domain.ChallengesResponse, []byte, error) {
	dto, body, err := p.api(ctx, surfaceProviderGet, infinite.RetryProvider(), infinite.CleChallenges).GetDecks(ctx, authDe(tokens), xuid)
	if err != nil {
		return domain.ChallengesResponse{}, body, err
	}
	total, completed, xpAvail, nextExpiry := aggregateChallenges(dto.AssignedDecks)
	items := p.buildActiveChallengeItems(ctx, tokens, dto.AssignedDecks)
	slog.InfoContext(ctx, "halo_provider: challenges fetched", "xuid", xuid, "total", total, "completed", completed)
	resp := domain.ChallengesResponse{Available: true, Total: &total, Completed: &completed, XPAvailable: &xpAvail, Items: items}
	if nextExpiry != "" {
		resp.NextExpiry = &nextExpiry
	}
	return resp, body, nil
}

// aggregateChallenges calcule les totaux des decks assignés.
func aggregateChallenges(decks []challengeDeckRaw) (total, completed, xpAvail int, nextExpiry string) {
	for _, deck := range decks {
		activeCnt := len(deck.ActiveChallenges)
		completedCnt := len(deck.CompletedChallenges)
		total += activeCnt + completedCnt
		completed += completedCnt
		for _, ch := range deck.ActiveChallenges {
			if ch.XPReward != nil {
				xpAvail += *ch.XPReward
			}
		}
		if exp := deck.Expiration.ISO8601Date; exp != "" {
			if nextExpiry == "" || exp < nextExpiry {
				nextExpiry = exp
			}
		}
	}
	return
}
