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

// EvaluateCumulative évalue un défi de type cumulative (compteur cumulé).
//
// `total` est la SOMME de la métrique du défi sur les matchs comptabilisés,
// déjà agrégée par l'appelant depuis la borne basse `evalSince` du défi
// (cf. service.evaluateCumulativeNow → BaselineProvider.CumulativeSince).
//
// Un cumulatif ne MOYENNE JAMAIS. « 220 tirs à la tête » se compare à un total,
// pas à une moyenne par match. Défaut corrigé le 2026-07-26 : les DEUX chemins
// d'évaluation (persistance `evaluateOne` et affichage `computeCurrentValue`)
// retombaient sur EvaluateThreshold, si bien que la jauge affichait 1.25 (une
// moyenne) contre une cible de 220 — soit 0,6 % de progression sur un objectif
// en réalité presque atteint.
//
// La forme précédente consommait des `[]MedalEvent` qu'AUCUNE source de
// production n'a jamais produits (le type n'avait ni écrivain ni appelant hors
// tests) : elle a été remplacée par le total, seule donnée dont l'agrégation est
// réellement mesurable en base. Pas de fenêtre par nombre de matchs — la
// profondeur est temporelle (created_at) et l'échéance est portée par ExpiresAt.
//
// Si total ≥ target → completed (priorité sur l'expiration).
// Si ExpiresAt dépassé et cible non atteinte → expired.
func EvaluateCumulative(_ Tuning, c Challenge, total float64, now time.Time) EvaluationResult {
	// La cible est vérifiée avant l'expiration : si atteinte pile à la deadline → completed.
	if total >= c.Target {
		return EvaluationResult{
			NewValue:      total,
			NewStatus:     StatusCompleted,
			StatusChanged: true,
			Reason:        EvalReasonTargetReached,
		}
	}

	if isExpirationPassed(c, now) {
		return EvaluationResult{
			NewValue:      total,
			NewStatus:     StatusExpired,
			StatusChanged: true,
			Reason:        EvalReasonDeadlinePassed,
		}
	}

	return EvaluationResult{
		NewValue:      total,
		NewStatus:     c.Status,
		StatusChanged: false,
		Reason:        EvalReasonProgress,
	}
}

// evalSince calcule la borne basse temporelle des matchs comptés pour un défi :
//
//   - JAMAIS avant createdAt → un défi ne se complète pas rétroactivement avec
//     l'historique antérieur à sa création (invariant anti-complétion-rétroactive) ;
//   - pour une fenêtre rolling_days, pas avant now - N jours (la plus récente des
//     deux bornes l'emporte).
//
// Les fenêtres session / last_n_matches sont bornées par createdAt seul (leur
// profondeur est portée par un compteur de matchs, pas par le temps).
//
// SOURCE UNIQUE de la règle : partagée par les défis d'escouade (squadEvalSince)
// et par les défis personnels cumulatifs (service.evaluateCumulativeNow). Deux
// copies de cette borne divergeraient (CLAUDE.md n°6).
func evalSince(createdAt time.Time, windowType WindowType, windowValue string, now time.Time) time.Time {
	since := createdAt
	if windowType != WindowRollingDays {
		return since
	}
	n, err := strconv.Atoi(windowValue)
	if err != nil || n <= 0 {
		return since
	}
	if cutoff := now.AddDate(0, 0, -n); cutoff.After(since) {
		since = cutoff
	}
	return since
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
