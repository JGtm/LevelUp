package prestige

import (
	"strconv"
	"time"

	"levelup/go-api/internal/analysis"
)

// evaluator.go — évaluation d'un défi à partir de données fraîches.
//
// Référence : Axe 7 (state machine) + Annexe A (médailles) + win_rate min matches.
//
// L'évaluateur est PUR : prend un défi + des données déjà chargées, retourne
// un résultat (nouvelle valeur, transition éventuelle). Il n'écrit ni ne lit
// la DB — c'est le service qui orchestre.
//
// Fenêtres supportées :
//   - WindowLastNMatches : l'appelant fournit les N derniers matchs ; N = WindowValue.
//     Expiration via ExpiresAt (calculé à la création selon tier + mode).
//   - WindowDeadline     : backward compat, expiration via WindowValue ISO ou ExpiresAt.
//   - WindowRollingDays  : déprécié — migrer vers WindowLastNMatches.
//   - WindowSession      : usage interne.

// EvaluationResult est le résultat d'une passe d'évaluation.
type EvaluationResult struct {
	NewValue      float64         // valeur courante mesurée
	NewStatus     ChallengeStatus // statut résultant (peut être inchangé)
	StatusChanged bool            // true si la transition a eu lieu
	Reason        EvalReason      // raison du résultat (utile pour log/UI)
}

// EvalReason classe la raison du résultat d'évaluation.
type EvalReason string

const (
	EvalReasonProgress       EvalReason = "progress"             // valeur mise à jour, défi continue
	EvalReasonTargetReached  EvalReason = "target_reached"       // cible atteinte → completed
	EvalReasonDeadlinePassed EvalReason = "deadline_passed"      // fenêtre écoulée → expired
	EvalReasonInsufficient   EvalReason = "insufficient_matches" // pas assez de matchs
)

// MatchSample représente un match exploité pour l'évaluation threshold.
type MatchSample struct {
	StartedAt   time.Time
	MetricValue float64
	IsWin       bool
}

// MedalEvent représente un événement de médaille pour l'évaluation cumulative.
type MedalEvent struct {
	StartedAt time.Time
	MedalID   string
	Count     int
}

// EvaluateThreshold évalue un défi de type threshold (moyenne sur fenêtre).
//
// Pour WindowLastNMatches : l'appelant fournit les N derniers matchs.
// Si len(matches) < N → EvalReasonInsufficient (joueur n'a pas encore joué N matchs).
// Si moyenne ≥ target → completed.
// Si ExpiresAt dépassé et cible non atteinte → expired.
//
// Précondition : matches fournis sont DÉJÀ filtrés sur la fenêtre du défi.
func EvaluateThreshold(t Tuning, c Challenge, matches []MatchSample, now time.Time) EvaluationResult {
	if isExpirationPassed(c, now) {
		avg := computeAverage(matches, c.Metric)
		minReq := minMatchesForMetric(t, c)
		if (minReq == 0 || len(matches) >= minReq) && avg >= c.Target {
			return EvaluationResult{
				NewValue:      avg,
				NewStatus:     StatusCompleted,
				StatusChanged: true,
				Reason:        EvalReasonTargetReached,
			}
		}
		return EvaluationResult{
			NewValue:      avg,
			NewStatus:     StatusExpired,
			StatusChanged: true,
			Reason:        EvalReasonDeadlinePassed,
		}
	}

	minReq := minMatchesForMetric(t, c)
	if len(matches) < minReq {
		return EvaluationResult{
			NewValue:      computeAverage(matches, c.Metric),
			NewStatus:     c.Status,
			StatusChanged: false,
			Reason:        EvalReasonInsufficient,
		}
	}

	avg := computeAverage(matches, c.Metric)
	if avg >= c.Target {
		return EvaluationResult{
			NewValue:      avg,
			NewStatus:     StatusCompleted,
			StatusChanged: true,
			Reason:        EvalReasonTargetReached,
		}
	}
	return EvaluationResult{
		NewValue:      avg,
		NewStatus:     c.Status,
		StatusChanged: false,
		Reason:        EvalReasonProgress,
	}
}

// EvaluateCumulative évalue un défi de type cumulative (compteur d'événements).
//
// Pour les défis "5× Killtacular" : somme les Count des MedalEvent
// dont l'ID match la métrique du défi (filtré par l'appelant).
// Pas de fenêtre par matchs — uniquement le timer d'expiration (ExpiresAt).
//
// Si total ≥ target → completed (priorité sur l'expiration).
// Si ExpiresAt dépassé et cible non atteinte → expired.
func EvaluateCumulative(t Tuning, c Challenge, events []MedalEvent, now time.Time) EvaluationResult {
	total := 0
	for _, ev := range events {
		total += ev.Count
	}
	totalF := float64(total)

	// La cible est vérifiée avant l'expiration : si atteinte pile à la deadline → completed.
	if totalF >= c.Target {
		return EvaluationResult{
			NewValue:      totalF,
			NewStatus:     StatusCompleted,
			StatusChanged: true,
			Reason:        EvalReasonTargetReached,
		}
	}

	if isExpirationPassed(c, now) {
		return EvaluationResult{
			NewValue:      totalF,
			NewStatus:     StatusExpired,
			StatusChanged: true,
			Reason:        EvalReasonDeadlinePassed,
		}
	}

	return EvaluationResult{
		NewValue:      totalF,
		NewStatus:     c.Status,
		StatusChanged: false,
		Reason:        EvalReasonProgress,
	}
}

// computeAverage retourne la moyenne de la métrique sur les matchs fournis.
//
// Pour win_rate, calcule (#wins / #matches) × 100.
// TODO(expiry:2026-12-31) P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
func computeAverage(matches []MatchSample, metric string) float64 {
	if len(matches) == 0 {
		return 0
	}
	if metric == MetricWinRatePascal || metric == MetricWinRateSnake {
		wins := 0
		for _, m := range matches {
			if m.IsWin {
				wins++
			}
		}
		return analysis.WinRate(wins, len(matches)) * 100.0
	}
	var sum float64
	for _, m := range matches {
		sum += m.MetricValue
	}
	return sum / float64(len(matches))
}

// minMatchesForMetric retourne le nb minimal de matchs requis pour évaluer le défi.
//
// Pour WindowLastNMatches : N = WindowValue (parsé), fallback sur tuning.RequiredMatchCount.
// Pour win_rate avec rolling_days/session : utilise WinRateMin du tuning.
// Autres cas : 0 (toujours évaluable).
func minMatchesForMetric(t Tuning, c Challenge) int {
	if c.WindowType == WindowLastNMatches {
		n, err := strconv.Atoi(c.WindowValue)
		if err != nil || n <= 0 {
			return t.RequiredMatchCount(c.Tier)
		}
		return n
	}
	if c.Metric != MetricWinRatePascal && c.Metric != MetricWinRateSnake {
		return 0
	}
	return t.WinRateMinForWindow(c.WindowType, c.WindowValue)
}

// isExpirationPassed retourne true si le défi est expiré.
//
// Priorité : ExpiresAt (champ unifié, calculé à la création).
// Fallback backward compat : WindowDeadline avec WindowValue ISO.
func isExpirationPassed(c Challenge, now time.Time) bool {
	if c.ExpiresAt != nil {
		return now.After(*c.ExpiresAt)
	}
	if c.WindowType == WindowDeadline {
		return isDeadlinePassed(c.WindowValue, now)
	}
	return false
}

// isDeadlinePassed parse la deadline ISO et compare avec now.
// Conservé pour backward compat avec WindowDeadline sans ExpiresAt.
func isDeadlinePassed(windowValue string, now time.Time) bool {
	if windowValue == "" {
		return false
	}
	deadline, err := time.Parse(time.RFC3339, windowValue)
	if err != nil {
		deadline, err = time.Parse("2006-01-02", windowValue)
		if err != nil {
			return false
		}
	}
	return now.After(deadline)
}
