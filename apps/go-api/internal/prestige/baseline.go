package prestige

import "time"

// baseline.go — calcul de la baseline personnelle et gestion de la fraîcheur.
//
// Référence : Axe 2 du plan conceptuel.
// La baseline est la moyenne d'une métrique sur les N derniers matchs
// du même mode (N=20 par défaut, configurable via tuning.Baseline.WindowMatches).
//
// Si le joueur n'a pas joué pendant tuning.AntiSmurf.BaselineStaleDays jours,
// la baseline est marquée stale et le joueur entre en phase de recovery
// (les RecoveryMatches premiers matchs après le retour sont en data_tier=estimated).

// MatchData est l'extrait minimal d'un match nécessaire au calcul de baseline.
//
// Le mode (PvP / Ranked / PvE / etc.) est représenté par une chaîne libre
// pour rester découplé du domaine `domain` — c'est l'appelant qui filtre
// les matchs du bon mode avant de fournir cette liste.
type MatchData struct {
	MatchID     string
	StartedAt   time.Time
	MetricValue float64
}

// ComputeBaseline calcule la baseline personnelle pour une métrique.
//
// Prend les N derniers matchs (déjà filtrés par mode et triés par date desc).
// La fenêtre exacte = min(len(matches), tuning.Baseline.WindowMatches).
//
// Retourne aussi le DataTier dérivé du nombre de matchs disponibles :
//   - len < MatchesEstimated (5)  → DataTracking
//   - len < MatchesFull      (10) → DataEstimated
//   - sinon                       → DataFull
//
// Précondition : matches non-nil (peut être vide).
func ComputeBaseline(t Tuning, userID, titleSlug, metric string, matches []MatchData) Baseline {
	limit := t.Baseline.WindowMatches
	if len(matches) < limit {
		limit = len(matches)
	}
	window := matches[:limit]

	dataTier := classifyDataTier(t, len(window))

	var sum float64
	for _, m := range window {
		sum += m.MetricValue
	}
	var value float64
	if len(window) > 0 {
		value = sum / float64(len(window))
	}

	return Baseline{
		UserID:     userID,
		TitleSlug:  titleSlug,
		Metric:     metric,
		Value:      value,
		MatchCount: len(window),
		DataTier:   dataTier,
		ComputedAt: time.Now().UTC(),
	}
}

// classifyDataTier retourne le DataTier selon le nombre de matchs disponibles.
func classifyDataTier(t Tuning, matchCount int) DataTier {
	if matchCount < t.Baseline.MatchesEstimated {
		return DataTracking
	}
	if matchCount < t.Baseline.MatchesFull {
		return DataEstimated
	}
	return DataFull
}

// CheckStaleness retourne true si la baseline est obsolète (> N jours d'inactivité).
//
// Si lastMatchAt est nil, considère que le joueur est nouveau — la baseline
// n'est pas stale, juste vide.
func CheckStaleness(t Tuning, lastMatchAt *time.Time, now time.Time) bool {
	if lastMatchAt == nil {
		return false
	}
	threshold := time.Duration(t.AntiSmurf.BaselineStaleDays) * 24 * time.Hour
	return now.Sub(*lastMatchAt) > threshold
}

// RecoveryDataTier retourne le DataTier applicable pendant la phase de recovery.
//
// Si recoveryMatchesRemaining > 0, on est encore en recovery → DataEstimated
// quel que soit le nombre de matchs (sauf si vraiment < MatchesEstimated, alors DataTracking).
// Sinon, classification normale.
func RecoveryDataTier(t Tuning, st BaselineState, matchCount int) DataTier {
	if st.RecoveryMatchesRemaining > 0 {
		// En recovery, plafonner à DataEstimated même si on a beaucoup de matchs
		standard := classifyDataTier(t, matchCount)
		if standard == DataFull {
			return DataEstimated
		}
		return standard
	}
	return classifyDataTier(t, matchCount)
}

// AdvanceRecovery décrémente le compteur de recovery après un nouveau match.
//
// Retourne le nouveau BaselineState (avec recovery_matches_remaining réduit).
// Quand recovery atteint 0, la baseline est de nouveau au plein bonus.
func AdvanceRecovery(st BaselineState, now time.Time) BaselineState {
	updated := st
	if updated.RecoveryMatchesRemaining > 0 {
		updated.RecoveryMatchesRemaining--
	}
	if updated.IsStale && updated.RecoveryMatchesRemaining == 0 {
		updated.IsStale = false
	}
	updated.UpdatedAt = now
	return updated
}

// MarkStale marque la baseline comme stale et active la phase de recovery.
//
// Appelé quand on détecte une réactivation après inactivité longue.
func MarkStale(t Tuning, st BaselineState, now time.Time) BaselineState {
	updated := st
	updated.IsStale = true
	updated.RecoveryMatchesRemaining = t.AntiSmurf.RecoveryMatches
	updated.UpdatedAt = now
	return updated
}
