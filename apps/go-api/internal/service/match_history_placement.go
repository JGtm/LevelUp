// Package service — match_history_placement.go : calcul du placement (X/Y)
// par match pour CSR (officiel Microsoft) et LUSR (interne LevelUp).
//
// CSR  : parse "Placement (N restants)" depuis tier_label + threshold dynamique
//
//	via csr_placement_thresholds (5 depuis S3, 10 avant).
//
// LUSR : stratégie "hack" — pour chaque chaîne (arena_slayer / arena_objectif /
//
//	btb / chaos), les 10 matchs non classés sans LUSR les plus anciens
//	sont marqués comme placement 1/10, 2/10, ..., 10/10. Pas de tier=
//	'Placement' en DB pour LUSR (contrairement à CSR), on dérive du
//	nombre de matchs joués par chaîne.
package service

import (
	"context"
	"log/slog"
	"regexp"
	"sort"
	"strconv"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync"
)

// CSRThresholdResolver lookup season_id → threshold de placement CSR (5 ou 10).
// Implémenté par *duckdb.CSRThresholdsRepo. Callback plutôt qu'interface pour
// éviter de remonter un port pour une seule méthode triviale.
type CSRThresholdResolver func(ctx context.Context, seasonID string) int

// placementRemainingRe parse "Placement (4 restants)" → 4. Duplique le helper
// de home_repo_playlist_ranks.go pour éviter d'importer platform/duckdb dans
// service (cycle : sync→duckdb, et on importe sync ici).
var placementRemainingRe = regexp.MustCompile(`\((\d+)\s*restant`)

func parsePlacementRemaining(label string) int {
	m := placementRemainingRe.FindStringSubmatch(label)
	if len(m) < 2 {
		return sync.LUSRPlacementThreshold
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 0 || n > 100 {
		return sync.LUSRPlacementThreshold
	}
	return n
}

// applyMatchPlacements remplit rows[i].PlacementDone/PlacementTotal pour les
// matchs en phase de placement. Modifie rows en place. Idempotent.
//
// rows DOIT contenir TOUS les matchs du joueur (pas seulement la page courante)
// car le calcul LUSR a besoin de l'ordre chronologique global pour identifier
// les 10 plus anciens par chaîne.
func applyMatchPlacements(
	ctx context.Context,
	rows []domain.MatchHistoryRawRow,
	csrThreshold CSRThresholdResolver,
) {
	csrCount := applyCSRPlacements(ctx, rows, csrThreshold)
	lusrCount := applyLUSRPlacements(rows)
	if csrCount > 0 || lusrCount > 0 {
		// DebugContext : volume potentiellement élevé (1 appel/page Explorer).
		// Active via LEVELUP_LOG_LEVEL=debug ; dispatch auto vers logs/service.log.
		slog.DebugContext(ctx, "match_history.placements_applied",
			"rows_total", len(rows),
			"csr_placements", csrCount,
			"lusr_placements", lusrCount,
		)
	}
}

// applyCSRPlacements : pour les matchs CSR avec tier_label "Placement (N restants)",
// calcule done = threshold - remaining (threshold via csr_placement_thresholds).
// Retourne le nombre de matchs marqués (pour visibilité via slog).
func applyCSRPlacements(
	ctx context.Context,
	rows []domain.MatchHistoryRawRow,
	csrThreshold CSRThresholdResolver,
) int {
	count := 0
	for i := range rows {
		r := &rows[i]
		if r.SkillRatingType == nil || *r.SkillRatingType != "CSR" {
			continue
		}
		if r.SkillTierLabel == nil {
			continue
		}
		label := *r.SkillTierLabel
		// "Placement" est le préfixe FR et EN ; pas besoin de détecter la langue.
		if len(label) < 9 || label[:9] != "Placement" {
			continue
		}
		seasonID := ""
		if r.SeasonID != nil {
			seasonID = *r.SeasonID
		}
		threshold := defaultCSRPlacementThreshold
		if csrThreshold != nil {
			threshold = csrThreshold(ctx, seasonID)
		}
		remaining := parsePlacementRemaining(label)
		if remaining > threshold {
			remaining = threshold
		}
		done := threshold - remaining
		if done < 0 {
			done = 0
		}
		t := threshold
		d := done
		r.PlacementDone = &d
		r.PlacementTotal = &t
		count++
	}
	return count
}

// defaultCSRPlacementThreshold : fallback quand csrThreshold est nil. Aligné
// sur duckdb.CSRPlacementThresholdDefault (5 depuis S3, 2023-03-07).
const defaultCSRPlacementThreshold = 5

// applyLUSRPlacements : pour chaque chaîne LUSR, identifie les
// sync.LUSRPlacementThreshold (=10) matchs les plus anciens du joueur SANS
// LUSR existant, et leur attribue placement 1/10, 2/10, ..., 10/10 dans
// l'ordre chronologique.
//
// Stratégie validée avec l'utilisateur (PR2 plan) : pas de tier='Placement'
// en DB pour LUSR, on dérive du compte de matchs par chaîne.
// Retourne le nombre total de matchs marqués (pour visibilité via slog).
func applyLUSRPlacements(rows []domain.MatchHistoryRawRow) int {
	type candidate struct {
		idx  int   // index dans rows pour update direct
		when int64 // start_time.UnixNano pour tri sans alloc
	}
	byChain := make(map[string][]candidate, 4)
	for i := range rows {
		r := &rows[i]
		// Déjà un LUSR → pas en placement (a déjà atteint le 11e match).
		if r.SkillRatingType != nil && *r.SkillRatingType == "LUSR" {
			continue
		}
		// Déjà un CSR (placement ou tier officiel) → pas concerné par LUSR.
		if r.SkillRatingType != nil && *r.SkillRatingType == "CSR" {
			continue
		}
		if r.PairName == nil || r.StartTime == nil {
			continue
		}
		chain := sync.GetLUSRChain(*r.PairName)
		if chain == "" {
			// Ranked (→ CSR) ou Firefight (→ PvE) : exclus du LUSR par construction.
			continue
		}
		byChain[chain] = append(byChain[chain], candidate{idx: i, when: r.StartTime.UnixNano()})
	}
	limit := sync.LUSRPlacementThreshold
	count := 0
	for _, cands := range byChain {
		sort.Slice(cands, func(a, b int) bool { return cands[a].when < cands[b].when })
		n := len(cands)
		if n > limit {
			n = limit
		}
		for k := 0; k < n; k++ {
			done := k + 1
			total := limit
			rows[cands[k].idx].PlacementDone = &done
			rows[cands[k].idx].PlacementTotal = &total
			count++
		}
	}
	return count
}
