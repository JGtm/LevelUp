// Package duckdb - season_pass_repo_builders.go : builders track summary +
// content summary + tier summaries + reward buckets. Decoupe de
// season_pass_repo.go (god-file split, refactor 2026-05-27).
package duckdb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// buildTrackSummary construit un SeasonPassTrackSummary depuis les données brutes.
func buildTrackSummary(
	row seasonPassTrackRow,
	state trackSnapshotState,
	itemMap map[string]seasonPassItemMeta,
) domain.SeasonPassTrackSummary {
	payload := parseTrackPayload(row.rawPayloadJSON)
	name := row.rewardTrackPath
	if row.trackName.Valid && row.trackName.String != "" {
		name = row.trackName.String
	} else if payloadName := localizedText(payloadNameValue(payload)); payloadName != "" {
		name = payloadName
	}
	description := payloadDescription(payload)
	xpPerRank := resolveXPPerRank(row, payload)
	maxRank := resolveMaxRank(payload, state)
	tiers, activeTierRank := buildTierSummaries(payload, itemMap, state)
	activeTierProgressPercent := computeActiveTierProgressPercent(state, xpPerRank, activeTierRank)

	status := computeSeasonPassStatus(state)
	isOwned := state.IsOwned || state.Rank > 0 || state.IsActive

	s := domain.SeasonPassTrackSummary{
		RewardTrackPath: row.rewardTrackPath,
		Name:            name,
		Description:     description,
		Status:          status,
		IsActive:        state.IsActive,
		IsOwned:         isOwned,
		// PremiumOwned = signal brut d'achat premium (state.IsOwned), SANS la
		// dilution progression/actif de IsOwned ci-dessus.
		PremiumOwned:              state.IsOwned,
		HasReachedMaxRank:         state.HasReachedMaxRank,
		CurrentRank:               state.Rank,
		PartialProgress:           state.Partial,
		MaxRank:                   maxRank,
		CompletionPercent:         computeCompletionPercent(state, maxRank),
		ActiveTierRank:            activeTierRank,
		ActiveTierProgressPercent: activeTierProgressPercent,
		Tiers:                     tiers,
		Content:                   computeContentSummary(payload, itemMap, 0),
		RemainingContent:          computeContentSummary(payload, itemMap, state.Rank),
	}
	if xpPerRank != nil {
		v := *xpPerRank
		s.XPPerRank = &v
	}
	if imageURL := resolveTrackImageURL(row, payload); imageURL != nil {
		s.ImageURL = imageURL
	}
	if backgroundURL := resolveTrackBackgroundURL(row, payload); backgroundURL != nil {
		s.BackgroundImageURL = backgroundURL
	}
	if !state.SnapshotAt.IsZero() {
		ts := state.SnapshotAt.UTC().Format(time.RFC3339)
		s.SnapshotAt = &ts
	}
	return s
}

// computeContentSummary agrège le contenu d'un track (currencies + cosmétiques + raretés).
// N'agrège que les paliers dont le rang est STRICTEMENT supérieur à minRank :
//   - minRank = 0           → tout le pass (inventaire complet, indépendant de l'ownership).
//   - minRank = rang courant → uniquement le RESTANT (paliers pas encore atteints).
//
// Retourne nil si aucun palier ne reste (ex. RemainingContent au rang max).
func computeContentSummary(
	payload *seasonPassTrackPayload,
	itemMap map[string]seasonPassItemMeta,
	minRank int,
) *domain.SeasonPassContentSummary {
	if payload == nil || len(payload.Ranks) == 0 {
		return nil
	}
	s := &domain.SeasonPassContentSummary{
		RarityBreakdown: map[string]int{},
		TypeBreakdown:   map[string]int{},
	}
	seen := map[string]struct{}{}
	for _, rank := range payload.Ranks {
		if rank.Rank <= minRank {
			continue
		}
		s.TotalTiers++
		aggregateRewardBucket(rank.FreeRewards, itemMap, s, seen)
		aggregateRewardBucket(rank.PaidRewards, itemMap, s, seen)
	}
	if s.TotalTiers == 0 {
		return nil
	}
	if len(s.RarityBreakdown) == 0 {
		s.RarityBreakdown = nil
	}
	if len(s.TypeBreakdown) == 0 {
		s.TypeBreakdown = nil
	}
	return s
}

// aggregateRewardBucket accumule les currencies et items cosmétiques d'un bucket.
func aggregateRewardBucket(
	bucket seasonPassRewardBucket,
	itemMap map[string]seasonPassItemMeta,
	s *domain.SeasonPassContentSummary,
	seen map[string]struct{},
) {
	for _, reward := range bucket.CurrencyRewards {
		slug := currencySlug(reward.CurrencyPath)
		switch slug {
		case "cr":
			s.Credits += reward.Amount
		case "softcurrency":
			s.SpartanPoints += reward.Amount
		case "xpboost":
			s.XPBoosts++
		case "rerollcurrency":
			s.ChallengeSwaps++
		}
	}
	for _, reward := range bucket.InventoryRewards {
		p := strings.TrimSpace(reward.InventoryItemPath)
		if p == "" {
			continue
		}
		if _, already := seen[p]; already {
			continue
		}
		seen[p] = struct{}{}
		s.CosmeticsTotal++
		if meta, ok := itemMap[p]; ok {
			if meta.Quality != nil && *meta.Quality != "" {
				s.RarityBreakdown[strings.ToLower(*meta.Quality)]++
			}
			if meta.ItemType != nil && *meta.ItemType != "" {
				s.TypeBreakdown[*meta.ItemType]++
			}
		}
	}
}

func parseTrackPayload(raw sql.NullString) *seasonPassTrackPayload {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	var payload seasonPassTrackPayload
	if err := json.Unmarshal([]byte(raw.String), &payload); err != nil {
		return nil
	}
	if len(payload.Ranks) == 0 && payload.BattlePassImage == "" && payload.BackgroundImagePath == "" {
		return nil
	}
	return &payload
}

func collectTrackItemPaths(payload *seasonPassTrackPayload) []string {
	if payload == nil || len(payload.Ranks) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	for _, rank := range payload.Ranks {
		collectRewardBucketItemPaths(rank.FreeRewards, seen)
		collectRewardBucketItemPaths(rank.PaidRewards, seen)
	}
	return mapKeys(seen)
}

func collectRewardBucketItemPaths(bucket seasonPassRewardBucket, seen map[string]struct{}) {
	for _, reward := range bucket.InventoryRewards {
		path := strings.TrimSpace(reward.InventoryItemPath)
		if path == "" {
			continue
		}
		seen[path] = struct{}{}
	}
}

func buildTierSummaries(
	payload *seasonPassTrackPayload,
	itemMap map[string]seasonPassItemMeta,
	state trackSnapshotState,
) ([]domain.SeasonPassTierSummary, *int) {
	if payload == nil || len(payload.Ranks) == 0 {
		return nil, nil
	}
	ranks := append([]seasonPassRankRaw(nil), payload.Ranks...)
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].Rank < ranks[j].Rank
	})
	maxRank := ranks[len(ranks)-1].Rank
	activeTierRank := resolveActiveTierRank(state, maxRank)
	tiers := make([]domain.SeasonPassTierSummary, 0, len(ranks))
	for _, rank := range ranks {
		tiers = append(tiers, buildTierSummary(rank, itemMap, state, activeTierRank))
	}
	return tiers, intPtr(activeTierRank)
}

func buildTierSummary(
	rank seasonPassRankRaw,
	itemMap map[string]seasonPassItemMeta,
	state trackSnapshotState,
	activeTierRank int,
) domain.SeasonPassTierSummary {
	meta, isPremium := selectTierPreview(rank, itemMap, state.IsOwned)
	title := meta.Title
	if title == "" {
		title = fmt.Sprintf("Palier %d", rank.Rank)
	}
	return domain.SeasonPassTierSummary{
		Rank:        rank.Rank,
		Title:       title,
		Description: meta.Description,
		ImageURL:    meta.ImageURL,
		Quality:     meta.Quality,
		ItemType:    meta.ItemType,
		IsObtained:  rank.Rank <= state.Rank,
		IsCurrent:   rank.Rank == activeTierRank,
		IsPremium:   isPremium,
		FreeRewards: buildFreeRewardSummaries(rank.FreeRewards, itemMap),
	}
}

// buildFreeRewardSummaries construit la liste des récompenses gratuites d'un palier.
// Retourne nil si le bucket est vide (omitempty côté JSON).
func buildFreeRewardSummaries(
	bucket seasonPassRewardBucket,
	itemMap map[string]seasonPassItemMeta,
) []domain.SeasonPassItemSummary {
	items := make([]domain.SeasonPassItemSummary, 0, len(bucket.InventoryRewards))
	for _, reward := range bucket.InventoryRewards {
		path := strings.TrimSpace(reward.InventoryItemPath)
		if path == "" {
			continue
		}
		meta, ok := itemMap[path]
		if !ok {
			continue
		}
		title := meta.Title
		if title == "" {
			title = path
		}
		items = append(items, domain.SeasonPassItemSummary{
			Title:       title,
			Description: meta.Description,
			ImageURL:    meta.ImageURL,
			Quality:     meta.Quality,
			ItemType:    meta.ItemType,
		})
	}
	// Currency rewards gratuits
	for _, reward := range bucket.CurrencyRewards {
		meta, ok := currencyRewardMeta(reward)
		if !ok {
			continue
		}
		items = append(items, domain.SeasonPassItemSummary{
			Title:    meta.Title,
			ImageURL: meta.ImageURL,
		})
	}
	return items
}
