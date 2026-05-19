package streaks

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// evaluator.go — orchestration de l'évaluation de progression des streaks.
//
// Appelé post-sync (cf. commit 6 hook) avec l'activité récente du joueur et
// la liste des types de streak à évaluer + leurs seuils personnels. Idempotent
// par bucket : appeler Evaluate plusieurs fois dans le même jour ne change pas
// l'état après la première mise à jour réussie.

// Transition décrit le changement d'état d'une streak après évaluation.
type Transition string

const (
	// TransitionNone : aucun changement (déjà incrémenté ce bucket, ou bucket
	// courant pas encore satisfait).
	TransitionNone Transition = "none"
	// TransitionStarted : nouvelle streak créée (premier bucket satisfait, pas
	// de streak active préexistante).
	TransitionStarted Transition = "started"
	// TransitionIncremented : streak existante incrémentée d'au moins 1.
	TransitionIncremented Transition = "incremented"
	// TransitionShielded : 1+ bucket(s) manqué(s) absorbé(s) par un shield.
	// Peut être combiné avec increment (recover après shield).
	TransitionShielded Transition = "shielded"
	// TransitionBroken : 1+ bucket(s) manqué(s) sans shield suffisant. Streak
	// passée à status=broken, length figée.
	TransitionBroken Transition = "broken"
)

// EvaluateInput contient tout ce dont l'évaluateur a besoin pour une passe.
type EvaluateInput struct {
	UserID    string
	TitleSlug string
	Now       time.Time
	// Matches : activité récente du joueur (suffisamment pour couvrir tous les
	// buckets depuis le dernier increment de chaque streak). L'orchestrateur
	// (post-sync hook) garantit qu'un nombre raisonnable de matchs récents
	// est fourni (typ. 30 derniers jours).
	Matches []MatchActivity
	// Thresholds : seuil personnel par type de streak. Optionnel (seuls les
	// types perf-based l'utilisent). Si une clé est absente, la streak de ce
	// type n'est PAS évaluée.
	Thresholds map[StreakType]float64
}

// EvaluationResult décrit l'effet de l'évaluation sur une streak.
type EvaluationResult struct {
	Streak     Streak
	Transition Transition
}

// Evaluator coordonne l'évaluation des streaks d'un joueur.
type Evaluator struct {
	repo  Repo
	idGen func() string
}

// NewEvaluator construit un Evaluator avec génération d'ID UUID v4 par défaut.
func NewEvaluator(repo Repo) *Evaluator {
	return &Evaluator{repo: repo, idGen: func() string { return uuid.New().String() }}
}

// WithIDGen surcharge le générateur d'ID (pour tests).
func (e *Evaluator) WithIDGen(f func() string) *Evaluator {
	e.idGen = f
	return e
}

// Evaluate exécute l'évaluation sur tous les types de streak listés dans
// `input.Thresholds`. Retourne un résultat par type évalué (incluant
// TransitionNone pour ceux sans changement, pour permettre l'observabilité).
func (e *Evaluator) Evaluate(ctx context.Context, input EvaluateInput) ([]EvaluationResult, error) {
	results := make([]EvaluationResult, 0, len(input.Thresholds))
	for st, threshold := range input.Thresholds {
		res, err := e.evaluateOne(ctx, input, st, threshold)
		if err != nil {
			return nil, fmt.Errorf("evaluate streak %s: %w", st, err)
		}
		results = append(results, res)
	}
	return results, nil
}

// evaluateOne traite une seule (streak type, threshold) pour ce joueur.
func (e *Evaluator) evaluateOne(ctx context.Context, input EvaluateInput, st StreakType, threshold float64) (EvaluationResult, error) {
	now := input.Now
	current, err := e.repo.GetActive(ctx, input.UserID, input.TitleSlug, st)
	if err != nil {
		return EvaluationResult{}, err
	}

	// Cas 1 : pas de streak active → checker si on en démarre une.
	if current == nil {
		bs := BucketStart(now, st)
		be := BucketEnd(now, st)
		if !IsBucketSatisfied(input.Matches, bs, be, st, threshold) {
			return EvaluationResult{Transition: TransitionNone}, nil
		}
		fresh := newStreak(e.idGen(), input.UserID, input.TitleSlug, st, threshold, now)
		if err := e.repo.Upsert(ctx, fresh); err != nil {
			return EvaluationResult{}, err
		}
		return EvaluationResult{Streak: fresh, Transition: TransitionStarted}, nil
	}

	// Cas 2 : streak active ou paused → calcul des buckets passés.
	RegenerateIfNewMonth(current, now)

	ref := current.StartedAt
	if current.LastIncrementAt != nil {
		ref = *current.LastIncrementAt
	}

	// Parcours des buckets de (ref + 1) à (bucket de now) inclus.
	increments, missed := walkBuckets(input.Matches, ref, now, st, threshold)

	// Application des shields si buckets manqués.
	if missed > 0 {
		if !TryConsume(current, missed) {
			current.Status = StreakStatusBroken
			bt := now
			current.BrokenAt = &bt
			if err := e.repo.Upsert(ctx, *current); err != nil {
				return EvaluationResult{}, err
			}
			return EvaluationResult{Streak: *current, Transition: TransitionBroken}, nil
		}
		current.Status = StreakStatusPaused
	}

	// Application des increments.
	if increments > 0 {
		current.CurrentLength += increments
		if current.CurrentLength > current.BestLength {
			current.BestLength = current.CurrentLength
		}
		incrementAt := now
		current.LastIncrementAt = &incrementAt
		current.Status = StreakStatusActive

		if err := e.repo.Upsert(ctx, *current); err != nil {
			return EvaluationResult{}, err
		}
		tr := TransitionIncremented
		if missed > 0 {
			tr = TransitionShielded
		}
		return EvaluationResult{Streak: *current, Transition: tr}, nil
	}

	// Pas d'increment mais shield consommé : persister le paused.
	if missed > 0 {
		if err := e.repo.Upsert(ctx, *current); err != nil {
			return EvaluationResult{}, err
		}
		return EvaluationResult{Streak: *current, Transition: TransitionShielded}, nil
	}

	return EvaluationResult{Streak: *current, Transition: TransitionNone}, nil
}

// walkBuckets parcourt les buckets de (ref+1) à (now) inclus et retourne
// le nombre de buckets satisfaits (= candidats à incrément) et le nombre
// de buckets manqués (= candidats à shield).
//
// Sémantique du shield (cf. plan §4.2) : un shield « préserve » la streak
// sans incrémenter le compteur pour le jour manqué. Les buckets satisfaits
// AVANT et APRÈS un miss shielded sont tous comptés en increments —
// la chaîne logique se ressoude. Si trop de misses pour les shields,
// le break est total et increments deviennent inutiles (rebranchement
// downstream de cette fonction).
//
// Cas particulier : si ref est dans le même bucket que now, retourne (0, 0).
func walkBuckets(matches []MatchActivity, ref, now time.Time, st StreakType, threshold float64) (increments, missed int) {
	refBucketStart := BucketStart(ref, st)
	nowBucketStart := BucketStart(now, st)
	if !nowBucketStart.After(refBucketStart) {
		return 0, 0
	}

	cur := nextBucketStart(refBucketStart, st)
	for !cur.After(nowBucketStart) {
		if IsBucketSatisfied(matches, cur, BucketEnd(cur, st), st, threshold) {
			increments++
		} else {
			missed++
		}
		cur = nextBucketStart(cur, st)
	}
	return increments, missed
}

// nextBucketStart retourne le début du bucket suivant celui contenant t.
func nextBucketStart(t time.Time, st StreakType) time.Time {
	bs := BucketStart(t, st)
	if isWeeklyType(st) {
		return bs.AddDate(0, 0, 7)
	}
	return bs.AddDate(0, 0, 1)
}

// newStreak construit une nouvelle streak active à partir du premier bucket satisfait.
func newStreak(id, userID, titleSlug string, st StreakType, threshold float64, now time.Time) Streak {
	var thr *float64
	if isPerfType(st) {
		t := threshold
		thr = &t
	}
	incrementAt := now
	return Streak{
		ID:               id,
		UserID:           userID,
		TitleSlug:        titleSlug,
		Type:             st,
		StartedAt:        BucketStart(now, st),
		CurrentLength:    1,
		BestLength:       1,
		LastIncrementAt:  &incrementAt,
		Threshold:        thr,
		ShieldsUsed:      0,
		ShieldsAvailable: MaxShieldsPerMonth,
		Status:           StreakStatusActive,
	}
}

// isPerfType retourne true si le type utilise un seuil de performance.
func isPerfType(st StreakType) bool {
	return st == StreakTypeDailyPerf || st == StreakTypeWeeklyKDAThreshold
}
