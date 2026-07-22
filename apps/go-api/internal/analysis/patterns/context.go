package patterns

import "levelup/go-api/internal/domain"

// context.go — analyse des patterns contextuels (by_mode, by_map, by_squad).

// analyzeContext calcule les patterns contextuels pour les 3 axes :
// mode, carte et composition d'équipe (squad/solo).
func analyzeContext(rows []MatchRow, cfg PatternConfig) []ContextualPattern {
	globalWR := globalWinRate(rows)
	var out []ContextualPattern
	out = append(out, analyzeByMode(rows, cfg, globalWR)...)
	out = append(out, analyzeByMap(rows, cfg, globalWR)...)
	out = append(out, analyzeBySquad(rows, cfg, globalWR)...)
	return out
}

// analyzeByMode groupe les matchs par mode et calcule les patterns.
func analyzeByMode(rows []MatchRow, cfg PatternConfig, globalWR float64) []ContextualPattern {
	groups := make(map[string][]MatchRow)
	for _, r := range rows {
		groups[r.Mode] = append(groups[r.Mode], r)
	}
	return buildContextPatterns(groups, ContextByMode, cfg, globalWR)
}

// analyzeByMap groupe les matchs par MapID et calcule les patterns.
func analyzeByMap(rows []MatchRow, cfg PatternConfig, globalWR float64) []ContextualPattern {
	groups := make(map[string][]MatchRow)
	for _, r := range rows {
		groups[r.MapID] = append(groups[r.MapID], r)
	}
	return buildContextPatterns(groups, ContextByMap, cfg, globalWR)
}

// analyzeBySquad analyse la différence squad/solo (by_squad).
// N'émet que si les deux groupes ont >= 10 matchs chacun.
func analyzeBySquad(rows []MatchRow, cfg PatternConfig, globalWR float64) []ContextualPattern {
	const minSquadMatches = 10
	var squadRows, soloRows []MatchRow
	for _, r := range rows {
		if r.IsWithFriends {
			squadRows = append(squadRows, r)
		} else {
			soloRows = append(soloRows, r)
		}
	}
	if len(squadRows) < minSquadMatches || len(soloRows) < minSquadMatches {
		return nil
	}
	squadPat := buildSinglePattern(ContextBySquad, "with_friends", squadRows, cfg, globalWR)
	soloPat := buildSinglePattern(ContextBySquad, "solo", soloRows, cfg, globalWR)
	return []ContextualPattern{squadPat, soloPat}
}

// buildContextPatterns crée les ContextualPattern pour chaque groupe.
func buildContextPatterns(groups map[string][]MatchRow, ctxType ContextType, cfg PatternConfig, globalWR float64) []ContextualPattern {
	var out []ContextualPattern
	for key, rows := range groups {
		if len(rows) < cfg.MinMatchesPerGroup {
			continue
		}
		pat := buildSinglePattern(ctxType, key, rows, cfg, globalWR)
		out = append(out, pat)
	}
	return out
}

// buildSinglePattern calcule un ContextualPattern pour un groupe de rows.
func buildSinglePattern(ctxType ContextType, key string, rows []MatchRow, cfg PatternConfig, globalWR float64) ContextualPattern {
	count := len(rows)
	wins := 0
	sumKDA, sumOC, sumDR := 0.0, 0.0, 0.0
	var perfVals, deltaCSRVals, deltaLUSRVals []float64

	for _, r := range rows {
		if r.Outcome == domain.OutcomeWin {
			wins++
		}
		sumKDA += r.KDA
		sumOC += r.OC
		sumDR += r.DR
		if r.PerfScore != nil {
			perfVals = append(perfVals, *r.PerfScore)
		}
		if r.DeltaCSR != nil {
			deltaCSRVals = append(deltaCSRVals, *r.DeltaCSR)
		}
		if r.DeltaLUSR != nil {
			deltaLUSRVals = append(deltaLUSRVals, *r.DeltaLUSR)
		}
	}

	wr := float64(wins) / float64(count)
	delta := wr - globalWR

	pat := ContextualPattern{
		Type:       ctxType,
		Key:        key,
		MatchCount: count,
		WinRate:    wr,
		AvgKDA:     sumKDA / float64(count),
		AvgOC:      sumOC / float64(count),
		AvgDR:      sumDR / float64(count),
		Delta:      delta,
		Signal:     classifySignal(delta, count, cfg),
	}

	if len(perfVals) > 0 {
		avg := meanFloat(perfVals)
		pat.AvgPerf = &avg
	}
	if len(deltaCSRVals) > 0 {
		avg := meanFloat(deltaCSRVals)
		pat.AvgDeltaCSR = &avg
	}
	if len(deltaLUSRVals) > 0 {
		avg := meanFloat(deltaLUSRVals)
		pat.AvgDeltaLUSR = &avg
	}

	return pat
}

// MinMatchesForSignal est le nombre de matchs minimum d'un groupe pour qu'un
// signal Force/Faiblesse soit statistiquement crédible. En dessous, le front
// affiche un badge neutre « Échantillon faible » (DEC-8) — le seuil est servi
// dans PatternReport pour ne pas le coder en dur côté client.
const MinMatchesForSignal = 10

// classifySignal détermine le signal en fonction du delta de win rate.
// Strength : delta > seuil ET count >= MinMatchesForSignal.
// Weakness : delta < -seuil.
func classifySignal(delta float64, count int, cfg PatternConfig) Signal {
	switch {
	case delta > cfg.StrengthWinRateDelta && count >= MinMatchesForSignal:
		return SignalStrength
	case delta < -cfg.WeaknessWinRateDelta:
		return SignalWeakness
	default:
		return SignalNeutral
	}
}

// globalWinRate calcule le taux de victoire global sur toutes les rows.
func globalWinRate(rows []MatchRow) float64 {
	if len(rows) == 0 {
		return 0
	}
	wins := 0
	for _, r := range rows {
		if r.Outcome == domain.OutcomeWin {
			wins++
		}
	}
	return float64(wins) / float64(len(rows))
}

// meanFloat calcule la moyenne d'une slice de float64.
func meanFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
