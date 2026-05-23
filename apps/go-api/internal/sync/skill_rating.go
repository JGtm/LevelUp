// Package sync — skill_rating.go : algorithme LUSR (TrueSkill 2 adapté).
//
// Portage de src/analysis/skill_rating.py + _trueskill_math.py + _composite.py.
// Calcule un rating de compétence absolu par match en traitant les matchs
// séquentiellement dans l'ordre chronologique.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"time"

	"levelup/go-api/internal/analysis"
)

// ── PlayerState — état TrueSkill 2 entre deux matchs ────────────────────────

// PlayerState contient l'état mu/sigma d'un joueur pour un playlist_group.
type PlayerState struct {
	MU                   float64
	Sigma                float64
	MatchCount           int
	LastMatchTime        *time.Time
	AccuracyHistory      []float64
	DamageEffHistory     []float64
	MedalExploitHistory  []float64
	OffConversionHistory []float64
	DefResistanceHistory []float64
}

// NewPlayerState crée un état initial.
func NewPlayerState() *PlayerState {
	return &PlayerState{MU: InitialMU, Sigma: InitialSigma}
}

// ── Fonctions gaussiennes TrueSkill 2 ──────────────────────────────────────

func standardNormalPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2.0*math.Pi)
}

func standardNormalCDF(x float64) float64 {
	return 0.5 * (1.0 + math.Erf(x/math.Sqrt(2.0)))
}

func vWin(t, eps float64) float64 {
	x := t - eps
	denom := standardNormalCDF(x)
	if denom < 1e-10 {
		return -x
	}
	return standardNormalPDF(x) / denom
}

func wWin(t, eps float64) float64 {
	v := vWin(t, eps)
	x := t - eps
	return v * (v + x)
}

// ── TrueSkill update ────────────────────────────────────────────────────────

// trueskillUpdate met à jour (mu, sigma) après un match.
// Mu : Elo-style avec baseline dynamique basée sur muOpp.
//
//	expectedScore = 1 / (1 + exp(-(mu - muOpp) / (2 × Beta)))
//	deltaMU       = KElo × (actualScore - expectedScore) × wf
//
// Battre des adversaires plus forts (muOpp > mu) donne plus de gain ;
// battre des adversaires plus faibles en demi-mesure peut descendre mu.
// Sigma : réduction TrueSkill standard. weightFactor réservé pour pondération
// asymétrique (carry-adj Phase 1.bis cf. thought_log 2026-05-22).
//
//nolint:unparam // weightFactor toujours 1.0 aujourd'hui, signature configurable pour formule à venir.
func trueskillUpdate(mu, sigma, muOpp, sigmaOpp, actualScore, weightFactor float64) (float64, float64) {
	expectedScore := 1.0 / (1.0 + math.Exp(-(mu-muOpp)/(2.0*Beta)))
	deltaMU := KElo * (actualScore - expectedScore) * weightFactor
	newMU := math.Max(MinRating, mu+deltaMU)

	c2 := 2.0*Beta*Beta + sigma*sigma + sigmaOpp*sigmaOpp
	c := math.Sqrt(c2)
	eps := drawMargin(Beta)

	sigma2 := sigma * sigma
	w := wWin(0.0, eps/c)
	deltaSigma2 := sigma2 * (sigma2 / c2) * w * weightFactor

	newSigma2 := math.Max(MinSigma*MinSigma, sigma2-deltaSigma2)
	newSigma := math.Sqrt(newSigma2)
	newSigma = math.Min(math.Sqrt(newSigma*newSigma+Tau*Tau), MaxSigma)

	return newMU, newSigma
}

// applyInactivityDecay augmente sigma proportionnellement à l'inactivité.
func applyInactivityDecay(sigma, daysInactive float64) float64 {
	capped := math.Min(daysInactive, float64(MaxInactivityDays))
	if capped <= InactivityThresholdDay {
		return sigma
	}
	added := InactivitySigmaPerDay * (capped - InactivityThresholdDay)
	return clampF(sigma+added, MinSigma, MaxSigma)
}

// ── Score composite ─────────────────────────────────────────────────────────

// compositeMatchRow contient les champs nécessaires au score composite.
type compositeMatchRow struct {
	Kills               float64
	Deaths              float64
	Assists             float64
	KillsExpected       float64
	DeathsExpected      float64
	Outcome             *int
	DamageDealt         float64
	DamageTaken         float64
	Accuracy            float64
	MedalExploitScore   float64 // score brut ComputeMedalExploitScore, 0 si absent
	OffensiveConversion float64 // 225*(kills+assists/3)/damage_dealt, 0 si absent
	DefensiveResistance float64 // damage_taken/(225*deaths), 0 si absent
}

// computeCompositeScore calcule le score composite [0,1] d'un match. Wrapper
// rétro-compatible autour de computeCompositeScoreWithBreakdown qui ne retourne
// que le composite (utilisé par les tests existants).
//
// Les composantes manquantes (valeur 0 ou avg nil) sont ignorées et les poids
// renormalisés. 4 params réservés pour futures composantes du score composite
// (enemyAvgKE = carry adjustment vs adversaires ; avgMedalExploit = bonus exploit ;
// avgOffConv = offensive conversion ; avgDefRes = defensive resistance).
//
//nolint:unparam // signature stable pour formule PerfTier roadmap, callers passent nil aujourd'hui.
func computeCompositeScore(
	row *compositeMatchRow,
	avgAccuracy *float64,
	enemyAvgKE *float64,
	avgDamageEff *float64,
	avgMedalExploit *float64,
	avgOffConv *float64,
	avgDefRes *float64,
) float64 {
	composite, _ := computeCompositeScoreWithBreakdown(row, avgAccuracy, enemyAvgKE, avgDamageEff, avgMedalExploit, avgOffConv, avgDefRes)
	return composite
}

// computeCompositeScoreWithBreakdown calcule composite + breakdown détaillé.
// Le breakdown contient les valeurs des composantes effectivement calculées
// (clés = noms canoniques de CompositeWeights). Les composantes absentes du
// match (donnée manquante, poids 0) ne figurent pas dans la map — leur lecture
// retourne 0 côté Go, ce qui simplifie l'INSERT en batch.
//
// Utilisé par computeSkillRatingsBatch (commit V2-1) pour persister
// `lusr_component_history` en plus de `match_skill_rank`.
func computeCompositeScoreWithBreakdown(
	row *compositeMatchRow,
	avgAccuracy *float64,
	enemyAvgKE *float64,
	avgDamageEff *float64,
	avgMedalExploit *float64,
	avgOffConv *float64,
	avgDefRes *float64,
) (float64, map[string]float64) {
	w := CompositeWeights
	type entry struct {
		key    string
		score  float64
		weight float64
	}
	var valid []entry

	// 1. kills_vs_expected
	// Carry adjustment asymétrique : compresse le bonus quand les adversaires
	// sont faibles (playerKE >> enemyAvgKE), mais ne touche pas aux pénalités.
	// Référence : enemyAvgKE (difficulté réelle des adversaires), pas les
	// coéquipiers — évite de pénaliser un carry pour la faiblesse de son équipe.
	// Floor carryAdj à 1.0 : pas d'amplification si les adversaires sont plus forts.
	if row.KillsExpected > 0 {
		score := sigmoidRatio(row.Kills, row.KillsExpected)
		if enemyAvgKE != nil && *enemyAvgKE > 0 {
			carryRatio := row.KillsExpected / *enemyAvgKE
			carryAdj := clampF(carryRatio, 1.0, 2.0)
			if score > 0.5 {
				score = clampF(0.5+(score-0.5)/carryAdj, 0.0, 1.0)
			}
			// score ≤ 0.5 : pénalité pleine, non modifiée
		}
		valid = append(valid, entry{MetricKeyKillsVsExpected, score, w[MetricKeyKillsVsExpected]})
	}

	// 2. deaths_vs_expected (inversé)
	if row.DeathsExpected > 0 {
		score := sigmoidRatio(row.DeathsExpected, math.Max(1.0, row.Deaths))
		valid = append(valid, entry{MetricKeyDeathsVsExpected, score, w[MetricKeyDeathsVsExpected]})
	}

	// 3. win_factor
	if row.Outcome != nil {
		var winScore float64
		switch *row.Outcome {
		case 2:
			winScore = 1.0
		case 1:
			winScore = 0.5
		case 3:
			winScore = 0.0
		case 4:
			winScore = 0.15
		default:
			winScore = 0.5
		}
		valid = append(valid, entry{MetricKeyWinFactor, winScore, w[MetricKeyWinFactor]})
	}

	// 4. damage_efficiency
	total := row.DamageDealt + row.DamageTaken
	if total > 0 {
		rawEff := clampF(row.DamageDealt/total, 0.0, 1.0)
		scoreEff := rawEff
		if avgDamageEff != nil && *avgDamageEff > 0 {
			scoreEff = sigmoidRatio(rawEff, *avgDamageEff)
		}
		valid = append(valid, entry{MetricKeyDamageEfficiency, scoreEff, w[MetricKeyDamageEfficiency]})
	}

	// 5. accuracy_delta
	if row.Accuracy > 0 && avgAccuracy != nil && *avgAccuracy > 0 {
		score := sigmoidRatio(row.Accuracy, *avgAccuracy)
		valid = append(valid, entry{MetricKeyAccuracyDelta, score, w[MetricKeyAccuracyDelta]})
	}

	// 6. medal_exploit (optional — 0 si médailles absentes)
	if row.MedalExploitScore > 0 {
		ref := 5.0
		if avgMedalExploit != nil && *avgMedalExploit > 1e-9 {
			ref = *avgMedalExploit
		}
		valid = append(valid, entry{MetricKeyMedalExploit, sigmoidRatio(row.MedalExploitScore, ref), w[MetricKeyMedalExploit]})
	}

	// 7. offensive_conversion (optional — 0 si damage_dealt absent)
	if row.OffensiveConversion > 0 {
		ref := analysis.OffensiveConversionP80
		if avgOffConv != nil && *avgOffConv > 1e-9 {
			ref = *avgOffConv
		}
		valid = append(valid, entry{MetricKeyOffensiveConv, sigmoidRatio(row.OffensiveConversion, ref), w[MetricKeyOffensiveConv]})
	}

	// 8. defensive_resistance (optional — 0 si deaths=0)
	if row.DefensiveResistance > 0 {
		ref := analysis.DefensiveResistanceP80
		if avgDefRes != nil && *avgDefRes > 1e-9 {
			ref = *avgDefRes
		}
		valid = append(valid, entry{MetricKeyDefensiveResist, sigmoidRatio(row.DefensiveResistance, ref), w[MetricKeyDefensiveResist]})
	}

	if len(valid) == 0 {
		return 0.5, nil
	}
	totalWeight := 0.0
	for _, e := range valid {
		totalWeight += e.weight
	}
	if totalWeight < 1e-12 {
		return 0.5, nil
	}
	sum := 0.0
	breakdown := make(map[string]float64, len(valid))
	for _, e := range valid {
		sum += e.score * e.weight
		breakdown[e.key] = e.score
	}
	return clampF(sum/totalWeight, 0.0, 1.0), breakdown
}

// ── Estimation μ individuel ─────────────────────────────────────────────────

func estimateIndividualMU(killsExpected, matchAvgKE, matchStdKE, baseMU float64) float64 {
	if matchStdKE < 1e-6 {
		return baseMU
	}
	z := (killsExpected - matchAvgKE) / matchStdKE
	return baseMU + IndividualMUAlpha*z
}

// computeEnemyStrength calcule la force estimée de l'adversaire (muOpp, sigmaOpp).
func computeEnemyStrength(enemyKEs []float64, matchAvgKE, matchStdKE, playerMU float64) (float64, float64) { //nolint:unparam // sigmaOpp actuellement fixe à DefaultOpponentSigma, extensible
	if len(enemyKEs) == 0 {
		return playerMU, DefaultOpponentSigma
	}
	muEstimates := make([]float64, 0, len(enemyKEs))
	for _, ke := range enemyKEs {
		est := estimateIndividualMU(ke, matchAvgKE, matchStdKE, playerMU)
		muEstimates = append(muEstimates, est)
	}
	avgMU := 0.0
	for _, m := range muEstimates {
		avgMU += m
	}
	avgMU /= float64(len(muEstimates))

	if len(muEstimates) <= 1 {
		return avgMU, DefaultOpponentSigma
	}
	variance := 0.0
	for _, m := range muEstimates {
		variance += (m - avgMU) * (m - avgMU)
	}
	variance /= float64(len(muEstimates))
	sigma := clampF(math.Sqrt(variance)+MinSigma, MinSigma, DefaultOpponentSigma)
	return avgMU, sigma
}

// ── Batch compute LUSR ──────────────────────────────────────────────────────
// Structs lusrMatchData, lusrParticipant, lusrResult et loaders SQL → skill_rating_loaders.go.
// Constante LUSRMaxDelta → skill_config.go.

// BatchComputeLUSR est le wrapper public de batchComputeLUSR. Utilisé par
// RecomputeAfterARTRebuild (phase 4.4) et tout caller hors-package qui doit
// recompute la cascade LUSR (typiquement après un rebuild ART, un changement
// de formule, ou un backfill manuel). medal exploit map laissée nil — les
// callers qui veulent l'override exploit-aware (sync engine) utilisent encore
// le helper privé batchComputeLUSR.
func BatchComputeLUSR(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string, force bool) (int, error) {
	return batchComputeLUSR(ctx, playerDB, sharedDB, xuid, nil, force)
}

// batchComputeLUSR calcule le LUSR pour tous les matchs non classés.
// medalExploitByMatch : match_id → score brut d'exploit médailles (nil = pas de données).
// force : si true, recalcule même les matchs déjà présents (utile après changement de formule).
// Retourne le nombre de matchs mis à jour.
func batchComputeLUSR(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string, medalExploitByMatch map[string]float64, force bool) (int, error) {
	// 1. Charger les matchs non classés, non-firefight, triés chronologiquement.
	matches, err := loadLUSRMatchData(ctx, sharedDB, xuid)
	if err != nil {
		return 0, err
	}
	if len(matches) == 0 {
		return 0, nil
	}

	// Filtrer les matchs marqués `is_excluded` côté playerDB : ils ne doivent ni
	// alimenter la cascade TrueSkill ni recevoir de rating LUSR.
	excluded, err := loadExcludedMatchIDs(ctx, playerDB)
	if err != nil {
		return 0, fmt.Errorf("batchComputeLUSR: %w", err)
	}
	if len(excluded) > 0 {
		before := len(matches)
		filtered := matches[:0]
		for _, m := range matches {
			if excluded[m.MatchID] {
				continue
			}
			filtered = append(filtered, m)
		}
		matches = filtered
		slog.Debug("batchComputeLUSR: matchs exclus filtrés",
			"xuid", xuid, "filtered", before-len(matches), "remaining", len(matches))
		if len(matches) == 0 {
			return 0, nil
		}
	}

	// 2. Charger les participants pour calcul de force adverse.
	matchIDs := make([]string, len(matches))
	for i, m := range matches {
		matchIDs[i] = m.MatchID
	}
	participantsByMatch, err := loadLUSRParticipants(ctx, sharedDB, matchIDs)
	if err != nil {
		return 0, err
	}

	// 3. Charger les matchs déjà classés CSR (protéger) et LUSR (pour mode incrémental).
	existingCSR, err := loadExistingRatingIDs(ctx, playerDB, "CSR")
	if err != nil {
		return 0, fmt.Errorf("batchComputeLUSR: %w", err)
	}
	existingLUSR, err := loadExistingRatingIDs(ctx, playerDB, "LUSR")
	if err != nil {
		return 0, fmt.Errorf("batchComputeLUSR: %w", err)
	}
	// En mode force, on ne filtre pas les LUSR existants à l'upsert — ON CONFLICT DO UPDATE écrase.
	existingLUSRForUpsert := existingLUSR
	if force {
		existingLUSRForUpsert = make(map[string]bool)
	}

	// 4. En mode force : recalcul depuis zéro (état vierge, pas de seed).
	//    En mode incrémental : reprendre depuis le dernier état persisté.
	var states map[string]*PlayerState
	seedRatings := make(map[string]float64)
	if force {
		states = make(map[string]*PlayerState)
	} else {
		states = loadExistingLUSRStates(ctx, playerDB)
		for pg, st := range states {
			seedRatings[pg] = st.MU
		}
	}

	// 5. En mode normal : filtrer les matchs déjà calculés.
	//    En mode force : tout recalculer (upsertLUSRRatings écrase via ON CONFLICT).
	toProcess := matches
	if !force {
		toProcess = make([]lusrMatchData, 0, len(matches))
		for _, m := range matches {
			if existingCSR[m.MatchID] || existingLUSR[m.MatchID] {
				continue
			}
			toProcess = append(toProcess, m)
		}
		if len(toProcess) == 0 {
			return 0, nil
		}
	}

	// 6. Calculer les ratings via TrueSkill 2 séquentiel.
	results := computeSkillRatingsBatch(toProcess, participantsByMatch, states, medalExploitByMatch)
	if len(results) == 0 {
		return 0, nil
	}

	// 7. Écrire les résultats.
	return upsertLUSRRatings(ctx, playerDB, results, existingCSR, existingLUSRForUpsert, seedRatings)
}

// ── Dry-run LUSR (preview, sans écriture) ──────────────────────────────────

// LUSRPlaylistPreview compare l'état persisté actuel d'un playlist_group avec
// l'état qui serait écrit après recompute. Permet de valider que le rebuild
// ART change effectivement les valeurs LUSR comme attendu (cf. cibles squad
// dans memory/reference_lusr_target_levels.md).
type LUSRPlaylistPreview struct {
	PlaylistGroup string
	OldMU         float64 // rating persisté actuel (0 si pas de seed)
	OldSigma      float64
	NewMU         float64 // dernier rating qui serait écrit
	NewSigma      float64
	MatchCount    int                // nombre de matchs qui contribueraient à ce playlist_group
	ComponentAvgs map[string]float64 // moyenne par composante sur tous les matchs du groupe
}

// DeltaMU retourne NewMU - OldMU. Positif = LUSR remonte (joueur sous-évalué
// avant), négatif = LUSR descend.
func (p LUSRPlaylistPreview) DeltaMU() float64 { return p.NewMU - p.OldMU }

// LUSRDryRunReport agrège le résultat d'une exécution dry-run.
type LUSRDryRunReport struct {
	XUID             string
	MatchesProcessed int
	Playlists        []LUSRPlaylistPreview
}

// HasChanges retourne true si au moins un playlist_group montre un delta
// significatif (> 1.0 MU pour filtrer le bruit numérique).
func (r *LUSRDryRunReport) HasChanges() bool {
	for _, p := range r.Playlists {
		if absF(p.DeltaMU()) > 1.0 {
			return true
		}
	}
	return false
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// batchComputeLUSRPreview est le pendant dry-run de batchComputeLUSR :
// reproduit les étapes 1-6 (load + compute) mais court-circuite l'écriture
// (étape 7). À la place, agrège un LUSRDryRunReport qui compare l'état
// persisté à l'état qui serait écrit.
//
// Toujours en mode force=true (recompute depuis zéro pour pouvoir comparer
// l'ensemble du résultat avec l'état actuel — un dry-run incrémental serait
// trivialement vide).
func batchComputeLUSRPreview(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	medalExploitByMatch map[string]float64,
) (*LUSRDryRunReport, error) {
	report := &LUSRDryRunReport{XUID: xuid}

	// Étapes 1-2 identiques à batchComputeLUSR.
	matches, err := loadLUSRMatchData(ctx, sharedDB, xuid)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return report, nil
	}
	excluded, err := loadExcludedMatchIDs(ctx, playerDB)
	if err != nil {
		return nil, fmt.Errorf("batchComputeLUSRPreview: %w", err)
	}
	if len(excluded) > 0 {
		filtered := matches[:0]
		for _, m := range matches {
			if !excluded[m.MatchID] {
				filtered = append(filtered, m)
			}
		}
		matches = filtered
		if len(matches) == 0 {
			return report, nil
		}
	}
	matchIDs := make([]string, len(matches))
	for i, m := range matches {
		matchIDs[i] = m.MatchID
	}
	participantsByMatch, err := loadLUSRParticipants(ctx, sharedDB, matchIDs)
	if err != nil {
		return nil, err
	}

	// État actuel (persisté) — comparaison du Before.
	oldStates := loadExistingLUSRStates(ctx, playerDB)

	// Recompute depuis zéro (force=true, pas de seed).
	results := computeSkillRatingsBatch(matches, participantsByMatch,
		map[string]*PlayerState{}, medalExploitByMatch)

	// Agréger par playlist_group : dernier résultat chronologique = état final.
	// computeSkillRatingsBatch maintient un état interne mais ne le retourne pas ;
	// on reconstruit l'état final en prenant le rating du DERNIER match traité
	// par playlist_group (les matches sont déjà triés chronologiquement par
	// loadLUSRMatchData → COALESCE start_time_utc).
	finalByPG := make(map[string]*lusrResult, len(results))
	countByPG := make(map[string]int, len(results))
	compSums := make(map[string]map[string]float64)
	compCounts := make(map[string]map[string]int)
	for i := range results {
		r := &results[i]
		finalByPG[r.PlaylistGroup] = r
		countByPG[r.PlaylistGroup]++
		if compSums[r.PlaylistGroup] == nil {
			compSums[r.PlaylistGroup] = make(map[string]float64)
			compCounts[r.PlaylistGroup] = make(map[string]int)
		}
		for comp, val := range r.Components {
			compSums[r.PlaylistGroup][comp] += val
			compCounts[r.PlaylistGroup][comp]++
		}
	}

	// Construire le rapport — un PlaylistPreview par playlist_group.
	report.MatchesProcessed = len(results)
	for pg, r := range finalByPG {
		avgs := make(map[string]float64, len(compSums[pg]))
		for comp, sum := range compSums[pg] {
			avgs[comp] = sum / float64(compCounts[pg][comp])
		}
		preview := LUSRPlaylistPreview{
			PlaylistGroup: pg,
			NewMU:         r.RatingValue,
			NewSigma:      r.RatingDeviation,
			MatchCount:    countByPG[pg],
			ComponentAvgs: avgs,
		}
		if old, ok := oldStates[pg]; ok && old != nil {
			preview.OldMU = old.MU
			preview.OldSigma = old.Sigma
		}
		report.Playlists = append(report.Playlists, preview)
	}
	return report, nil
}

// computeSkillRatingsBatch calcule mu/sigma pour chaque match séquentiellement.
// medalExploitByMatch : map optionnelle match_id → score brut médailles.
func computeSkillRatingsBatch(
	matches []lusrMatchData,
	participantsByMatch map[string][]lusrParticipant,
	states map[string]*PlayerState,
	medalExploitByMatch map[string]float64,
) []lusrResult {
	if states == nil {
		states = make(map[string]*PlayerState)
	}
	results := make([]lusrResult, 0, len(matches))

	for _, match := range matches {
		pairName := ""
		if match.PairName != nil {
			pairName = *match.PairName
		}
		chain := GetLUSRChain(pairName)
		if chain == "" {
			continue // exclu : Ranked (→ CSR) ou Firefight (→ PvE)
		}

		state, exists := states[chain]
		if !exists {
			state = NewPlayerState()
			states[chain] = state
		}

		// Inactivité decay
		if state.LastMatchTime != nil {
			delta := match.StartTime.Sub(*state.LastMatchTime)
			days := delta.Hours() / 24.0
			state.Sigma = applyInactivityDecay(state.Sigma, days)
		}

		// Participants du match
		allParts := participantsByMatch[match.MatchID]
		matchAvgKE, matchStdKE := computeMatchKEStats(allParts)

		// Séparer coéquipiers et adversaires (teammateKEs inutilisé depuis v2 carry fix)
		_, enemyKEs := splitParticipantKEs(match.TeamID, allParts)

		// Force adversaire (ancrée sur state.MU)
		muOpp, sigmaOpp := computeEnemyStrength(enemyKEs, matchAvgKE, matchStdKE, state.MU)

		// Moyennes historiques (nil = pas assez de données → composante ignorée)
		avgAcc := rollingAvgPtr(state.AccuracyHistory)
		avgDmgEff := rollingAvgPtr(state.DamageEffHistory)
		avgMedalExploit := rollingAvgPtr(state.MedalExploitHistory)
		avgOffConv := rollingAvgPtr(state.OffConversionHistory)
		avgDefRes := rollingAvgPtr(state.DefResistanceHistory)

		// Enemy avg KE — référence du carry adjustment (difficulté des adversaires).
		var enemyAvgKE *float64
		if len(enemyKEs) > 0 {
			sum := 0.0
			for _, ke := range enemyKEs {
				sum += ke
			}
			avg := sum / float64(len(enemyKEs))
			enemyAvgKE = &avg
		}

		// Guard : match sans outcome
		if match.Outcome == nil {
			state.MatchCount++
			t := match.StartTime
			state.LastMatchTime = &t
			results = append(results, lusrResult{
				MatchID:         match.MatchID,
				RatingValue:     math.Round(state.MU*10) / 10,
				RatingDeviation: math.Round(state.Sigma*10) / 10,
				PlaylistGroup:   chain,
			})
			continue
		}

		// Calcul des métriques dérivées
		offConv, defRes := computeCombatYield(match)
		medalScore := medalExploitByMatch[match.MatchID]

		// Score composite
		cRow := &compositeMatchRow{
			Kills:               match.Kills,
			Deaths:              match.Deaths,
			Assists:             match.Assists,
			KillsExpected:       match.KillsExpected,
			DeathsExpected:      match.DeathsExpected,
			Outcome:             match.Outcome,
			DamageDealt:         match.DamageDealt,
			DamageTaken:         match.DamageTaken,
			Accuracy:            match.Accuracy,
			MedalExploitScore:   medalScore,
			OffensiveConversion: offConv,
			DefensiveResistance: defRes,
		}
		composite, breakdown := computeCompositeScoreWithBreakdown(cRow, avgAcc, enemyAvgKE, avgDmgEff, avgMedalExploit, avgOffConv, avgDefRes)

		// Update TrueSkill
		newMU, newSigma := trueskillUpdate(state.MU, state.Sigma, muOpp, sigmaOpp, composite, 1.0)
		state.MU = newMU
		state.Sigma = newSigma
		state.MatchCount++
		t := match.StartTime
		state.LastMatchTime = &t

		// Mise à jour des historiques glissants
		appendToHistory(&state.AccuracyHistory, match.Accuracy)
		totalDmg := match.DamageDealt + match.DamageTaken
		if totalDmg > 0 {
			appendToHistory(&state.DamageEffHistory, clampF(match.DamageDealt/totalDmg, 0, 1))
		}
		if medalScore > 0 {
			appendToHistory(&state.MedalExploitHistory, medalScore)
		}
		if offConv > 0 {
			appendToHistory(&state.OffConversionHistory, offConv)
		}
		if defRes > 0 {
			appendToHistory(&state.DefResistanceHistory, defRes)
		}

		results = append(results, lusrResult{
			MatchID:         match.MatchID,
			RatingValue:     math.Round(state.MU*10) / 10,
			RatingDeviation: math.Round(state.Sigma*10) / 10,
			PlaylistGroup:   chain,
			Components:      breakdown,
		})
	}
	return results
}

// rollingAvgPtr retourne la moyenne d'une slice si elle a au moins
// MinMatchesForAccuracyDelta éléments, nil sinon. Le seuil est toujours
// MinMatchesForAccuracyDelta (constant unique) — inline pour eviter le bruit
// unparam.
func rollingAvgPtr(hist []float64) *float64 {
	if len(hist) < MinMatchesForAccuracyDelta {
		return nil
	}
	sum := 0.0
	for _, v := range hist {
		sum += v
	}
	avg := sum / float64(len(hist))
	return &avg
}

// appendToHistory ajoute v à *hist et tronque a AccuracyHistorySize elements.
// La taille max est toujours AccuracyHistorySize (constant unique) — inline
// pour eviter le bruit unparam.
func appendToHistory(hist *[]float64, v float64) {
	*hist = append(*hist, v)
	if len(*hist) > AccuracyHistorySize {
		*hist = (*hist)[len(*hist)-AccuracyHistorySize:]
	}
}

// computeCombatYield calcule offensive_conversion et defensive_resistance depuis un match.
func computeCombatYield(m lusrMatchData) (offConv, defRes float64) {
	if m.DamageDealt > 0 {
		offConv = 225.0 * (m.Kills + m.Assists/3.0) / m.DamageDealt
	}
	if m.DamageTaken > 0 && m.Deaths > 0 {
		defRes = m.DamageTaken / (225.0 * m.Deaths)
	}
	return
}
