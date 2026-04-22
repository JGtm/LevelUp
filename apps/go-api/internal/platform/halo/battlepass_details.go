// Package halo — battlepass_details.go : définitions de tracks Battle Pass via assets.Resolver.
//
// Toutes les opérations (fetch, cache, images, items) sont déléguées au resolver unifié.
// La source de vérité est GameCMS (/hi/Progression/file/{trackPath}), pas le payload
// d'opérations /economy/operations qui ne contient que la progression joueur.
package halo

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/domain"
)

// battlepassTrackDefinitionRaw représente la définition d'un Reward Track depuis GameCMS.
// Structure : /hi/Progression/file/{RewardTrackPath}
type battlepassTrackDefinitionRaw struct {
	Name                any                    `json:"Name"`
	Description         any                    `json:"Description"`
	BattlePassImage     string                 `json:"BattlePassImage"`
	BackgroundImagePath string                 `json:"BackgroundImagePath"`
	XpPerRank           int                    `json:"XpPerRank"`
	Ranks               []battlepassRankDefRaw `json:"Ranks"`
}

type battlepassRankDefRaw struct {
	Rank        int                       `json:"Rank"`
	FreeRewards battlepassRewardBucketRaw `json:"FreeRewards"`
	PaidRewards battlepassRewardBucketRaw `json:"PaidRewards"`
}

type battlepassRewardBucketRaw struct {
	InventoryRewards []battlepassInventoryRewardRaw `json:"InventoryRewards"`
}

type battlepassInventoryRewardRaw struct {
	InventoryItemPath string `json:"InventoryItemPath"`
	Amount            int    `json:"Amount"`
}

// battlepassItemDefinitionRaw représente le JSON d'un item inventaire depuis GameCMS.
// Structure : GET /hi/Progression/file/{InventoryItemPath}
type battlepassItemDefinitionRaw struct {
	CommonData battlepassItemCommonDataRaw `json:"CommonData"`
}

type battlepassItemCommonDataRaw struct {
	Title       any                          `json:"Title"`
	Description any                          `json:"Description"`
	Quality     string                       `json:"Quality"`
	ItemType    string                       `json:"ItemType"`
	DisplayPath battlepassItemDisplayPathRaw `json:"DisplayPath"`
}

type battlepassItemDisplayPathRaw struct {
	Media battlepassItemMediaRaw `json:"Media"`
}

type battlepassItemMediaRaw struct {
	MediaUrl battlepassItemMediaUrlRaw `json:"MediaUrl"`
}

type battlepassItemMediaUrlRaw struct {
	Path string `json:"Path"`
}

// fetchRewardTrackDefinition charge la définition d'un track via le resolver unifié.
// Retourne nil si le resolver n'est pas configuré ou en cas d'erreur.
func (p *HaloProvider) fetchRewardTrackDefinition(
	ctx context.Context,
	_ *domain.HaloTokens,
	trackPath string,
) *battlepassTrackDefinitionRaw {
	trimmed := strings.TrimSpace(trackPath)
	if trimmed == "" || p.assetResolver == nil {
		return nil
	}

	ref := assets.Ref{
		Kind:    assets.KindRewardTrackDefinition,
		TitleID: p.titleID(),
		ID:      trimmed,
	}
	resolved, err := p.assetResolver.Get(ctx, ref)
	if err != nil {
		slog.DebugContext(ctx, "halo_provider: reward track definition resolver miss",
			"path", trimmed, "err", err)
		return nil
	}
	jp, ok := resolved.Payload.(assets.JSONPayload)
	if !ok {
		slog.DebugContext(ctx, "halo_provider: reward track definition unexpected payload type",
			"path", trimmed)
		return nil
	}
	var def battlepassTrackDefinitionRaw
	if err := json.Unmarshal(jp.RawJSON, &def); err != nil {
		slog.DebugContext(ctx, "halo_provider: reward track definition decode error",
			"path", trimmed, "err", err)
		return nil
	}
	go p.warmBPTrackAssets(ctx, &def)
	return &def
}

// warmBPTrackAssets pré-cache les images et les items d'un track via le resolver.
// Appelé dans une goroutine (fire-and-forget).
func (p *HaloProvider) warmBPTrackAssets(ctx context.Context, def *battlepassTrackDefinitionRaw) {
	if def == nil || p.assetResolver == nil {
		return
	}
	var refs []assets.Ref
	if bp := strings.TrimSpace(def.BattlePassImage); bp != "" {
		refs = append(refs, assets.Ref{
			Kind:    assets.KindBPTrackImage,
			TitleID: p.titleID(),
			ID:      bp,
		})
	}
	if bg := strings.TrimSpace(def.BackgroundImagePath); bg != "" {
		refs = append(refs, assets.Ref{
			Kind:    assets.KindBPBackground,
			TitleID: p.titleID(),
			ID:      bg,
		})
	}
	for _, rank := range def.Ranks {
		for _, r := range rank.FreeRewards.InventoryRewards {
			if ip := strings.TrimSpace(r.InventoryItemPath); ip != "" {
				refs = append(refs, assets.Ref{
					Kind:    assets.KindRewardTrackDefinition,
					TitleID: p.titleID(),
					ID:      ip,
				})
			}
		}
		for _, r := range rank.PaidRewards.InventoryRewards {
			if ip := strings.TrimSpace(r.InventoryItemPath); ip != "" {
				refs = append(refs, assets.Ref{
					Kind:    assets.KindRewardTrackDefinition,
					TitleID: p.titleID(),
					ID:      ip,
				})
			}
		}
	}
	if len(refs) > 0 {
		p.assetResolver.Warm(ctx, refs...)
	}
}
