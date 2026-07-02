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

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
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

// compositeMatchRow contient les champs nécessaires au score composite.
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
		slog.DebugContext(ctx, "batchComputeLUSR: matchs exclus filtrés",
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
	// En mode force, on ne filtre pas les LUSR existants — l'INSERT append-only
	// ajoute une nouvelle version et la vue match_skill_rank_latest renvoie la plus récente.
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
	//    En mode force : tout recalculer (upsertLUSRRatings ajoute une nouvelle
	//    version append-only, la vue latest réflète automatiquement la dernière).
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
	results := computeSkillRatingsBatch(ctxkeys.TitleSlug(ctx), toProcess, participantsByMatch, states, medalExploitByMatch)
	if len(results) == 0 {
		return 0, nil
	}

	// 7. Écrire les résultats.
	n, err := upsertLUSRRatings(ctx, playerDB, results, existingCSR, existingLUSRForUpsert, seedRatings)
	if err != nil {
		return n, err
	}

	// 8. PAS de compaction des versions superseded. Le DELETE per-row sur
	//    match_skill_rank (id NOT IN MAX(id)…) déclenche le bug ART DuckDB amont
	//    #23046 — il a FAIT CRASHER JGtm (2026-06-20) malgré mono-writer + PK BIGINT :
	//    l'hypothèse historique « sérialisé + PK BIGINT = sûr » est FAUSSE (#23046
	//    corrompt le heap file-backed sous churn, pas seulement sous concurrence).
	//    La table reste append-only PUR (INSERT seul) ; la vue match_skill_rank_latest
	//    (MAX(id)) reste correcte avec les versions superseded présentes. Croissance
	//    bornée en pratique (force-recompute rare) ; compaction éventuelle = job
	//    offline (serveur arrêté), jamais en runtime.
	_ = force
	return n, nil
}

// ── Dry-run LUSR (preview, sans écriture) ──────────────────────────────────

// LUSRPlaylistPreview compare l'état persisté actuel d'un playlist_group avec
// l'état qui serait écrit après recompute. Permet de valider que le rebuild
// ART change effectivement les valeurs LUSR comme attendu (cf. cibles squad
// dans memory/reference_lusr_target_levels.md).
func computeSkillRatingsBatch(
	titleSlug string,
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
		// Title-aware (C6) : un titre avec classifier dédié (Halo 5) classe ses
		// propres modes ; sinon retombe sur le classifier défaut (Infinite). Évite
		// que les modes h5 collapsent tous dans arena_slayer. titleSlug==""/halo_infinite
		// → défaut → byte-identique HINF (aucun classifier per-title HINF enregistré).
		chain := GetLUSRChainForTitle(titleSlug, pairName)
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
//
// Le baseline PV-pour-tuer est SCALE-INVARIANT ici : offConv/defRes sont ensuite
// comparés à la moyenne glissante DU MÊME joueur (avgOffConv/avgDefRes dans
// computeCompositeScoreWithBreakdown), tous calculés avec la même constante → le
// facteur s'annule dans le ratio et n'affecte PAS le classement LUSR. On utilise
// donc le baseline Infinite par défaut (games.DefaultEffectiveHpToKill) comme
// simple constante d'échelle nommée — title-agnostic par construction, sans avoir
// à threader le slug sur ce hot path. Le baseline title-aware (225 Infinite / 115
// Halo 5) ne concerne que le KPI Rendement/Résistance AFFICHÉ (match-view), résolu
// séparément via games.EffectiveHpToKill(slug).
func computeCombatYield(m lusrMatchData) (offConv, defRes float64) {
	const hpToKill = games.DefaultEffectiveHpToKill
	if m.DamageDealt > 0 {
		offConv = hpToKill * (m.Kills + m.Assists/3.0) / m.DamageDealt
	}
	if m.DamageTaken > 0 && m.Deaths > 0 {
		defRes = m.DamageTaken / (hpToKill * m.Deaths)
	}
	return
}
