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
//
// Note: GameCMS utilise deux noms différents pour l'image principale du pass selon
// les saisons. BattlePassImage existe sur S05+, mais S03/S04 n'ont que SummaryImagePath.
// On expose les deux et le consommateur prend BattlePassImage en priorité, fallback Summary.
type battlepassTrackDefinitionRaw struct {
	Name                any                    `json:"Name"`
	Description         any                    `json:"Description"`
	BattlePassImage     string                 `json:"BattlePassImage"`
	SummaryImagePath    string                 `json:"SummaryImagePath"`
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
	if p.trackDefPersister != nil {
		raw := jp.RawJSON
		go func() {
			if err := p.trackDefPersister.UpsertTrackDefinition(context.Background(), trimmed, raw); err != nil {
				slog.Warn("halo_provider: track definition persist failed", "path", trimmed, "err", err)
			}
		}()
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
	var imageRefs []assets.Ref
	if bp := strings.TrimSpace(def.BattlePassImage); bp != "" {
		imageRefs = append(imageRefs, assets.Ref{
			Kind:    assets.KindBPTrackImage,
			TitleID: p.titleID(),
			ID:      bp,
		})
	}
	if bg := strings.TrimSpace(def.BackgroundImagePath); bg != "" {
		imageRefs = append(imageRefs, assets.Ref{
			Kind:    assets.KindBPBackground,
			TitleID: p.titleID(),
			ID:      bg,
		})
	}
	if len(imageRefs) > 0 {
		p.assetResolver.Warm(ctx, imageRefs...)
	}

	// Résoudre + persister chaque item BP individuellement pour déclencher
	// UpsertItemDefinition sur chaque JSON résolu (structured write path).
	itemPaths := p.collectTrackItemPaths(def)
	for _, ip := range itemPaths {
		p.resolveAndPersistItem(ctx, ip)
	}
}

// collectTrackItemPaths extrait les InventoryItemPath uniques d'une définition de track.
func (p *HaloProvider) collectTrackItemPaths(def *battlepassTrackDefinitionRaw) []string {
	seen := make(map[string]struct{})
	for _, rank := range def.Ranks {
		for _, r := range rank.FreeRewards.InventoryRewards {
			if ip := strings.TrimSpace(r.InventoryItemPath); ip != "" {
				seen[ip] = struct{}{}
			}
		}
		for _, r := range rank.PaidRewards.InventoryRewards {
			if ip := strings.TrimSpace(r.InventoryItemPath); ip != "" {
				seen[ip] = struct{}{}
			}
		}
	}
	paths := make([]string, 0, len(seen))
	for ip := range seen {
		paths = append(paths, ip)
	}
	return paths
}

// resolveAndPersistItem résout un item via KindBPItemDefinition et persiste son JSON
// structuré si un ItemDefinitionPersister est câblé.
func (p *HaloProvider) resolveAndPersistItem(ctx context.Context, itemPath string) {
	ref := assets.Ref{
		Kind:    assets.KindBPItemDefinition,
		TitleID: p.titleID(),
		ID:      itemPath,
	}
	resolved, err := p.assetResolver.Get(ctx, ref)
	if err != nil {
		slog.DebugContext(ctx, "halo_provider: bp item resolver miss", "path", itemPath, "err", err)
		return
	}
	if p.itemDefPersister == nil {
		return
	}
	jp, ok := resolved.Payload.(assets.JSONPayload)
	if !ok {
		return
	}
	raw := jp.RawJSON
	if err := p.itemDefPersister.UpsertItemDefinition(context.Background(), itemPath, raw); err != nil {
		slog.Warn("halo_provider: item definition persist failed", "path", itemPath, "err", err)
	}
}
