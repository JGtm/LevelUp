// Package service — match_history_service_briefing_ranked.go : module « Classement »
// du bandeau de briefing de l'Explorer (mode Matchs).
//
// Émet une entrée PAR TYPE de rating (CSR, LUSR) présent dans le scope, jamais
// fusionnées (P-3) : progression de paliers (premier / dernier match chronologique
// du type portant un palier, via SkillTierLabel déjà résolu FR) + moyenne de delta
// par match (Value/Count du bucket). Les paliers de placement sont exposés par flags
// (D-D), jamais parsés côté front. Le bloc « attendu vs réel » a été retiré (décision
// produit 2026-07-16). Extrait de match_history_service_briefing.go pour rester sous
// le seuil de taille de fichier (CLAUDE.md §5).
package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"levelup/go-api/internal/domain"
)

// buildBriefingRanked émet le module « Classement » : une entrée PAR TYPE de
// rating (CSR, LUSR) présent dans le scope, jamais fusionnées (P-3). Le type
// majoritaire est toujours émis ; un type secondaire seulement s'il atteint
// minRankedKindMatches. Chaque entrée porte la progression de paliers (premier /
// dernier match chronologique du type portant un palier, via SkillTierLabel déjà
// résolu FR) et la moyenne de delta par match (Value/Count du bucket). Les paliers
// de placement sont exposés par flags (D-D), jamais parsés côté front. L'appelant
// a déjà gaté sur rankedCapable (capability match.skill.snapshot, exclut H5).
// Retourne nil si aucune entrée émise (pas de bucket de rating dans le scope).
func buildBriefingRanked(ctx context.Context, scope []domain.MatchHistoryRawRow, scopedKPIs *domain.KPIStats) *domain.ExplorerBriefingRanked {
	if scopedKPIs == nil || len(scopedKPIs.RankDeltas) == 0 {
		return nil
	}
	var majorityKind string
	if scopedKPIs.RankDelta != nil {
		majorityKind = scopedKPIs.RankDelta.Kind
	}
	kinds := make([]domain.ExplorerBriefingRankedKind, 0, len(scopedKPIs.RankDeltas))
	for _, rd := range scopedKPIs.RankDeltas {
		if rd.Kind != majorityKind && rd.Count < minRankedKindMatches {
			continue
		}
		kinds = append(kinds, buildRankedKind(ctx, scope, rd))
	}
	if len(kinds) == 0 {
		return nil
	}
	return &domain.ExplorerBriefingRanked{Kinds: kinds}
}

// buildRankedKind construit la progression d'UN type de rating : moyenne par match
// (Value/Count) + paliers de début/fin résolus depuis les rows du scope de CE type,
// triées chronologiquement. Les paliers de placement sont signalés par flags (D-D).
func buildRankedKind(ctx context.Context, scope []domain.MatchHistoryRawRow, rd domain.RankDelta) domain.ExplorerBriefingRankedKind {
	entry := domain.ExplorerBriefingRankedKind{Kind: rd.Kind, Matches: rd.Count}
	if rd.Count > 0 {
		v := rd.Value / float64(rd.Count)
		entry.DeltaPerMatch = &v
	}
	// Rows du scope de CE type, datées, triées chronologiquement. Casse : les raw
	// rows portent « CSR »/« LUSR » (majuscule) et rd.Kind « csr »/« lusr »
	// (canonical, minuscule) → comparaison insensible à la casse.
	typed := make([]domain.MatchHistoryRawRow, 0, len(scope))
	for _, r := range scope {
		if r.StartTime == nil || r.SkillRatingType == nil {
			continue
		}
		if strings.EqualFold(*r.SkillRatingType, rd.Kind) {
			typed = append(typed, r)
		}
	}
	sort.SliceStable(typed, func(i, j int) bool { return typed[i].StartTime.Before(*typed[j].StartTime) })

	first := firstTieredRow(typed)
	last := lastTieredRow(typed)
	if first == nil {
		// Aucun match du type ne porte de palier : progression omise (dégradation
		// best-effort documentée, jamais d'erreur avalée — CLAUDE.md §3).
		slog.DebugContext(ctx, "briefing ranked: no tier label for type", "kind", rd.Kind, "matches", rd.Count)
		return entry
	}
	applyTierStart(&entry, first)
	applyTierEnd(&entry, last)
	return entry
}

// firstTieredRow / lastTieredRow retournent le premier / dernier row portant un
// SkillTierLabel non nil (les rows sont déjà triées chronologiquement).
func firstTieredRow(rows []domain.MatchHistoryRawRow) *domain.MatchHistoryRawRow {
	for i := range rows {
		if rows[i].SkillTierLabel != nil {
			return &rows[i]
		}
	}
	return nil
}

func lastTieredRow(rows []domain.MatchHistoryRawRow) *domain.MatchHistoryRawRow {
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].SkillTierLabel != nil {
			return &rows[i]
		}
	}
	return nil
}

// isPlacementRow : le match est en phase de placement (PlacementTotal renseigné —
// cf. domain.MatchHistoryRawRow : PlacementDone/PlacementTotal nil hors placement).
func isPlacementRow(r *domain.MatchHistoryRawRow) bool { return r.PlacementTotal != nil }

// applyTierStart pose le palier de DÉBUT. En placement → flag sans compteur (D-D :
// « Placement → Platine VI »). Sinon → label brut FR déjà résolu.
func applyTierStart(entry *domain.ExplorerBriefingRankedKind, r *domain.MatchHistoryRawRow) {
	if isPlacementRow(r) {
		entry.TierStartIsPlacement = true
		return
	}
	entry.TierStartLabel = r.SkillTierLabel
}

// applyTierEnd pose le palier de FIN. En placement → nombre de matchs restants
// (D-D : « Placement (N restants) »). Sinon → label brut FR déjà résolu.
func applyTierEnd(entry *domain.ExplorerBriefingRankedKind, r *domain.MatchHistoryRawRow) {
	if isPlacementRow(r) {
		done := 0
		if r.PlacementDone != nil {
			done = *r.PlacementDone
		}
		remaining := *r.PlacementTotal - done
		if remaining < 0 {
			remaining = 0
		}
		entry.TierEndPlacementRemaining = &remaining
		return
	}
	entry.TierEndLabel = r.SkillTierLabel
}
