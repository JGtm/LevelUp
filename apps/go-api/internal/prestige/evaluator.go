package prestige

import "time"

// evaluator.go — évaluation d'un défi à partir de données fraîches.
//
// Référence : Axe 7 (state machine) + Annexe A (médailles) + win_rate min matches.
//
// L'évaluateur est PUR : prend un défi + des données déjà chargées, retourne
// un résultat (nouvelle valeur, transition éventuelle). Il n'écrit ni ne lit
// la DB — c'est le service qui orchestre.

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
	EvalReasonInsufficient   EvalReason = "insufficient_matches" // pas assez de matchs (win_rate)
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
// Calcule la moyenne de la métrique sur les matchs fournis.
// Si win_rate avec fenêtre dont matchs < min requis → EvalReasonInsufficient.
// Si moyenne ≥ target → completed.
// Sinon → progress.
//
// Précondition : matches fournis sont DÉJÀ filtrés sur la fenêtre du défi
// (l'appelant a fait le tri sessions/jours/deadline).
func EvaluateThreshold(t Tuning, c Challenge, matches []MatchSample, now time.Time) EvaluationResult {
	// Cas spécial : fenêtre deadline expirée
	if c.WindowType == WindowDeadline && isDeadlinePassed(c.WindowValue, now) {
		// Évaluer une dernière fois la moyenne ; si pas atteinte → expired
		avg := computeAverage(matches, c.Metric)
		if len(matches) >= minMatchesForMetric(t, c) && avg >= c.Target {
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

	// Vérifier minimum de matchs (cas win_rate)
	minMatches := minMatchesForMetric(t, c)
	if len(matches) < minMatches {
		// Progression affichée mais pas validable
		return EvaluationResult{
			NewValue:      computeAverage(matches, c.Metric),
			NewStatus:     c.Status, // inchangé
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
		NewStatus:     c.Status, // inchangé
		StatusChanged: false,
		Reason:        EvalReasonProgress,
	}
}

// EvaluateCumulative évalue un défi de type cumulative (compteur d'événements).
//
// Pour les défis "5× Killtacular ce mois" : somme les Count des MedalEvent
// dont l'ID match la métrique du défi (filtré par l'appelant).
//
// Si total ≥ target → completed.
// Si deadline passée → expired (cible non atteinte) ou completed (si atteinte).
func EvaluateCumulative(t Tuning, c Challenge, events []MedalEvent, now time.Time) EvaluationResult {
	total := 0
	for _, ev := range events {
		total += ev.Count
	}
	totalF := float64(total)

	if c.WindowType == WindowDeadline && isDeadlinePassed(c.WindowValue, now) {
		if totalF >= c.Target {
			return EvaluationResult{
				NewValue:      totalF,
				NewStatus:     StatusCompleted,
				StatusChanged: true,
				Reason:        EvalReasonTargetReached,
			}
		}
		return EvaluationResult{
			NewValue:      totalF,
			NewStatus:     StatusExpired,
			StatusChanged: true,
			Reason:        EvalReasonDeadlinePassed,
		}
	}

	if totalF >= c.Target {
		return EvaluationResult{
			NewValue:      totalF,
			NewStatus:     StatusCompleted,
			StatusChanged: true,
			Reason:        EvalReasonTargetReached,
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
// Pour la métrique win_rate, calcule (#wins / #matches) × 100 pour rester
// dans la même échelle que la cible (exprimée en %).
func computeAverage(matches []MatchSample, metric string) float64 {
	if len(matches) == 0 {
		return 0
	}
	if metric == "FieldWinRate" || metric == "win_rate" {
		wins := 0
		for _, m := range matches {
			if m.IsWin {
				wins++
			}
		}
		return float64(wins) / float64(len(matches)) * 100.0
	}
	var sum float64
	for _, m := range matches {
		sum += m.MetricValue
	}
	return sum / float64(len(matches))
}

// minMatchesForMetric retourne le nb minimal de matchs pour évaluer le défi.
//
// Pour win_rate avec session/rolling_days, utilise tuning.WinRateMin.
// Pour les autres métriques, retourne 0 (toujours évaluable).
func minMatchesForMetric(t Tuning, c Challenge) int {
	if c.Metric != "FieldWinRate" && c.Metric != "win_rate" {
		return 0
	}
	return t.WinRateMinForWindow(c.WindowType, c.WindowValue)
}

// isDeadlinePassed parse la deadline ISO et compare avec now.
//
// Si parsing échoue, retourne false (le défi n'expire pas faute de date claire).
func isDeadlinePassed(windowValue string, now time.Time) bool {
	if windowValue == "" {
		return false
	}
	deadline, err := time.Parse(time.RFC3339, windowValue)
	if err != nil {
		// Tentative format simple YYYY-MM-DD
		deadline, err = time.Parse("2006-01-02", windowValue)
		if err != nil {
			return false
		}
	}
	return now.After(deadline)
}
