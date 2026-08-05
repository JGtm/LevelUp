// Package service — match_history_placement.go : calcul du placement (X/Y)
// par match pour CSR (officiel Microsoft) et LUSR (interne LevelUp).
//
// CSR  : parse "Placement (N restants)" depuis tier_label + threshold dynamique
//
//	via csr_placement_thresholds (5 depuis S3, 10 avant).
//
// LUSR : stratégie "hack" — pour chaque chaîne (arena_slayer / arena_objectif /
//
//	btb / chaos) NON ENCORE CALIBRÉE (aucun match avec LUSR), les 10 matchs
//	LUSR-éligibles les plus anciens sont marqués comme placement 1/10, 2/10,
//	..., 10/10. Pas de tier='Placement' en DB pour LUSR (contrairement à CSR),
//	on dérive du nombre de matchs joués par chaîne. Une chaîne CALIBRÉE (≥ 1
//	LUSR existant) ne reçoit plus AUCUN signal de placement : ses matchs sans
//	LUSR (DNF/exclus, qui n'en auront jamais) sont des trous structurels, pas
//	un placement en cours (bug JGtm V7.3 : Super Fiesta DNF récents affichés
//	à tort "1/10..10/10" sur une chaîne déjà classée — cf. computePerfPlacements
//	qui applique la même doctrine côté performance).
package service

import (
	"context"
	"log/slog"
	"regexp"
	"sort"
	"strconv"

	"levelup/go-api/internal/ctxkeys"
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
	perfCount := applyPerfPlacements(ctx, rows)
	if csrCount > 0 || lusrCount > 0 || perfCount > 0 {
		// DebugContext : volume potentiellement élevé (1 appel/page Explorer).
		// Active via LEVELUP_LOG_LEVEL=debug ; dispatch auto vers logs/service.log.
		slog.DebugContext(ctx, "match_history.placements_applied",
			"rows_total", len(rows),
			"csr_placements", csrCount,
			"lusr_placements", lusrCount,
			"perf_placements", perfCount,
		)
	}
}

// perfPlacementInput décrit un match pour le calcul du placement de la chaîne de
// PERFORMANCE (perf_score) : sa chaîne perf, son éligibilité (comptée par le batch
// perf) et s'il porte déjà un perf_score. Structure pivot partagée par l'Explorer/
// Sessions (MatchHistoryRawRow) et l'Accueil (canonical) — une seule implémentation
// de l'algorithme (computePerfPlacements) pour toutes les surfaces (CLAUDE.md §6).
type perfPlacementInput struct {
	matchID string
	chain   string
	// eligible : le match est-il COMPTÉ par le batch perf dans sa chaîne ?
	// Miroir EXACT du périmètre de batchComputePerformanceScores (sync/performance.go) :
	//   - WHERE de loadHistoryForPerf : mr.start_time IS NOT NULL
	//     ET COALESCE(mp.outcome, 0) != 4 (DNF) ;
	//   - filtre loadExcludedMatchIDs : is_excluded retiré de allMatches AVANT la
	//     boucle, donc jamais compté dans la fenêtre de chaîne.
	// Toute divergence fait afficher un « X/10 » qui ne correspond pas au compteur
	// réel du batch (contre-revue V7.2 : le critère start_time manquait).
	eligible bool
	hasScore bool // perf_score déjà calculé (chaîne calibrée)
}

// computePerfPlacements renvoie matchID → [done, total] pour les matchs SANS
// perf_score dont la chaîne de performance est réellement SOUS LE SEUIL, c.-à-d.
// compte ≤ threshold matchs perf-éligibles (donc structurellement aucun perf_score :
// le 1er score n'arrive qu'au (threshold+1)-ième match de la chaîne). `done` = nombre
// de matchs éligibles de la chaîne (IDENTIQUE pour tous les matchs de la chaîne — c'est
// une progression de calibration, pas un rang par-match), `total` = threshold.
//
// Une chaîne à > threshold matchs est CALIBRÉE : ses matchs sans perf_score sont des
// trous structurels (fenêtre de calibration passée) ou un retard de sync — jamais un
// « placement » (on ne masque pas un bug de fraîcheur, cf. V72-32) → non renseignés.
// Les matchs non éligibles (DNF, exclus) ne comptent pas et ne sont jamais renseignés.
func computePerfPlacements(inputs []perfPlacementInput, threshold int) map[string][2]int {
	counts := make(map[string]int, 6)
	for _, in := range inputs {
		if in.eligible && in.chain != "" {
			counts[in.chain]++
		}
	}
	out := make(map[string][2]int)
	for _, in := range inputs {
		if !in.eligible || in.hasScore || in.chain == "" {
			continue
		}
		c := counts[in.chain]
		if c >= 1 && c <= threshold {
			out[in.matchID] = [2]int{c, threshold}
		}
	}
	return out
}

// applyPerfPlacements remplit rows[i].PerfPlacementDone/Total pour les matchs en
// placement de chaîne de PERFORMANCE. Modifie rows en place. Idempotent. `rows` DOIT
// contenir TOUS les matchs du joueur (comptage par chaîne sur l'historique global).
// Retourne le nombre de matchs marqués.
func applyPerfPlacements(ctx context.Context, rows []domain.MatchHistoryRawRow) int {
	slug := ctxkeys.TitleSlug(ctx)
	inputs := make([]perfPlacementInput, len(rows))
	for i := range rows {
		r := &rows[i]
		pair := ""
		if r.PairName != nil {
			pair = *r.PairName
		}
		inputs[i] = perfPlacementInput{
			matchID: r.MatchID,
			chain:   sync.GetPerformanceChain(slug, pair, r.IsRanked, r.IsFirefight),
			// Les 3 critères du batch, dans l'ordre du commentaire de
			// perfPlacementInput.eligible. `r.StartTime == nil` reproduit le
			// `mr.start_time IS NOT NULL` de loadHistoryForPerf : un match sans
			// start_time n'entre PAS dans l'historique du batch, il ne doit donc
			// pas gonfler le « X/10 » affiché.
			eligible: r.StartTime != nil && r.Outcome != domain.OutcomeDNF && !r.IsExcluded,
			hasScore: r.PerformanceScore != nil,
		}
	}
	placements := computePerfPlacements(inputs, sync.MinMatchesPerChainForRelative)
	count := 0
	for i := range rows {
		dt, ok := placements[rows[i].MatchID]
		if !ok {
			continue
		}
		d, t := dt[0], dt[1]
		rows[i].PerfPlacementDone = &d
		rows[i].PerfPlacementTotal = &t
		count++
	}
	return count
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

// lusrPlacementInput décrit un match pour le calcul du placement de chaîne LUSR :
// sa chaîne, s'il porte déjà un LUSR (signal de calibration de la chaîne) et son
// éligibilité au batch LUSR. Structure pivot symétrique de perfPlacementInput
// (même fichier) — même doctrine : on ne masque pas un trou structurel derrière
// un état de placement inventé.
type lusrPlacementInput struct {
	matchID string
	when    int64 // start_time.UnixNano, tri sans alloc
	chain   string
	hasLUSR bool
	// eligible : le match est-il un candidat LUSR possible ? Miroir PARTIEL du
	// périmètre balayé par le batch LUSR (loadShadowMatches + classifyLUSREligibility,
	// internal/sync/skill/skill_v2_shadow.go) :
	//   - DNF (domain.OutcomeDNF) : outcomeToTeamResult (skill_v2_shadow.go) ne
	//     mappe pas le code Halo 4 (DNF) → ces matchs sont skippedNonTwoTeam par
	//     le batch, jamais candidats à un LUSR ;
	//   - is_excluded : LoadExcludedMatchIDs (exclusion_filter.go) est retiré
	//     explicitement "de la cascade LUSR" selon son doc-comment ;
	//   - déjà noté CSR : garde défensive, un match Ranked ne doit jamais
	//     recevoir un placement LUSR même si pair_name résolvait une chaîne.
	// Le batch filtre EN PLUS sur rosters 2-équipes / imbalance / durée mini
	// (isTeamImbalanceTooHigh, buildTwoTeamRosters) : critères non exposés par
	// MatchHistoryRawRow, donc non reproduits ici — cf. rapport (Découvertes).
	eligible bool
}

// computeLUSRPlacements renvoie matchID → [done, total] pour les matchs SANS
// LUSR dont la chaîne n'est PAS ENCORE CALIBRÉE (aucun match de la chaîne ne
// porte de LUSR). `done` = rang chronologique du match dans les
// `limit` plus anciens candidats éligibles de sa chaîne, `total` = limit.
//
// Une chaîne calibrée (≥ 1 LUSR) est CLASSÉE : ses matchs sans LUSR restants
// (DNF, exclus, retard de sync) sont des trous structurels — jamais un
// "placement" — donc non renseignés (même doctrine que computePerfPlacements).
func computeLUSRPlacements(inputs []lusrPlacementInput, limit int) map[string][2]int {
	calibrated := make(map[string]bool, 4)
	for _, in := range inputs {
		if in.hasLUSR && in.chain != "" {
			calibrated[in.chain] = true
		}
	}
	type candidate struct {
		matchID string
		when    int64
	}
	byChain := make(map[string][]candidate, 4)
	for _, in := range inputs {
		if in.hasLUSR || !in.eligible || in.chain == "" || calibrated[in.chain] {
			continue
		}
		byChain[in.chain] = append(byChain[in.chain], candidate{matchID: in.matchID, when: in.when})
	}
	out := make(map[string][2]int)
	for _, cands := range byChain {
		sort.Slice(cands, func(a, b int) bool { return cands[a].when < cands[b].when })
		n := len(cands)
		if n > limit {
			n = limit
		}
		for k := 0; k < n; k++ {
			out[cands[k].matchID] = [2]int{k + 1, limit}
		}
	}
	return out
}

// applyLUSRPlacements : pour chaque chaîne LUSR NON CALIBRÉE, identifie les
// sync.LUSRPlacementThreshold (=10) matchs éligibles les plus anciens du joueur
// SANS LUSR existant, et leur attribue placement 1/10, 2/10, ..., 10/10 dans
// l'ordre chronologique. Modifie rows en place. Idempotent.
//
// Stratégie validée avec l'utilisateur (PR2 plan) : pas de tier='Placement'
// en DB pour LUSR, on dérive du compte de matchs par chaîne.
// Retourne le nombre total de matchs marqués (pour visibilité via slog).
func applyLUSRPlacements(rows []domain.MatchHistoryRawRow) int {
	inputs := make([]lusrPlacementInput, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		if r.PairName == nil || r.StartTime == nil {
			continue
		}
		chain := sync.GetLUSRChain(*r.PairName)
		if chain == "" {
			// Ranked (→ CSR) ou Firefight (→ PvE) : exclus du LUSR par construction.
			continue
		}
		isCSR := r.SkillRatingType != nil && *r.SkillRatingType == "CSR"
		inputs = append(inputs, lusrPlacementInput{
			matchID: r.MatchID,
			when:    r.StartTime.UnixNano(),
			chain:   chain,
			hasLUSR: r.SkillRatingType != nil && *r.SkillRatingType == "LUSR",
			eligible: !isCSR &&
				r.Outcome != domain.OutcomeDNF &&
				!r.IsExcluded,
		})
	}
	limit := sync.LUSRPlacementThreshold
	placements := computeLUSRPlacements(inputs, limit)
	count := 0
	for i := range rows {
		dt, ok := placements[rows[i].MatchID]
		if !ok {
			continue
		}
		d, t := dt[0], dt[1]
		rows[i].PlacementDone = &d
		rows[i].PlacementTotal = &t
		count++
	}
	return count
}
