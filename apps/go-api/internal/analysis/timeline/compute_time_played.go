package timeline

import (
	"time"

	"levelup/go-api/internal/domain"
)

// TimePlayedQuality qualifie la fiabilité du time_played recalculé pour un
// (match, joueur). Permet de tracer les cas dégradés lors du backfill.
type TimePlayedQuality string

const (
	// TimePlayedOK : calcul nominal depuis first_joined / last_leave.
	TimePlayedOK TimePlayedQuality = "ok"
	// TimePlayedNoData : first_joined_time absent → non calculable, ne pas
	// stocker (conserver la valeur API existante).
	TimePlayedNoData TimePlayedQuality = "no_data"
	// TimePlayedClampedZero : joined ≥ left après bornage (incohérence timing) →
	// time_played = 0, suspect.
	TimePlayedClampedZero TimePlayedQuality = "clamped_zero"
)

// Computed indique si la qualité correspond à une valeur stockable.
func (q TimePlayedQuality) Computed() bool {
	return q == TimePlayedOK || q == TimePlayedClampedZero
}

// TimePlayedInput porte la donnée de participation d'un joueur sur un match.
type TimePlayedInput struct {
	// FirstJoinedTime : instant d'arrivée du joueur (UTC). Zero si inconnu.
	FirstJoinedTime time.Time
	// LastLeaveTime : instant de départ (UTC). Nil si le joueur était encore
	// présent à la fin du match → on borne sur gameplayEnd.
	LastLeaveTime *time.Time
}

// ComputeTimePlayed recalcule le temps de jeu réel d'un joueur (en secondes)
// dans la fenêtre de gameplay [gameplayStart, gameplayEnd], en gérant quitters
// et latecomers. Algorithme (cf. PLAN_MATCH_TIMELINE_T0 §7.2) :
//
//	joined       = MAX(first_joined, gameplayStart)   // clamp latecomer / countdown
//	left         = last_leave (ou gameplayEnd si présent jusqu'au bout)
//	left_clamped = MIN(left, gameplayEnd)             // clamp dépassement film
//	time_played  = MAX(0, left_clamped − joined)      // borné [0, gameplay_duration]
//
// gameplayStart / gameplayEnd sont résolus par l'appelant :
//
//	gameplayStart = start_time_utc + t0_ms/1000   (t0=0 si indisponible)
//	gameplayEnd   = start_time_utc + duration_seconds
//
// Retourne (secondes, quality). Si quality.Computed() est faux (NoData), la
// valeur ne doit pas écraser la valeur API existante.
func ComputeTimePlayed(in TimePlayedInput, gameplayStart, gameplayEnd time.Time) (int64, TimePlayedQuality) {
	if in.FirstJoinedTime.IsZero() {
		return 0, TimePlayedNoData
	}
	if !gameplayEnd.After(gameplayStart) {
		// Fenêtre de gameplay invalide (durée ≤ 0) → non exploitable.
		return 0, TimePlayedNoData
	}

	joined := in.FirstJoinedTime
	if joined.Before(gameplayStart) {
		joined = gameplayStart
	}

	left := gameplayEnd
	if in.LastLeaveTime != nil && !in.LastLeaveTime.IsZero() {
		left = *in.LastLeaveTime
	}
	if left.After(gameplayEnd) {
		left = gameplayEnd
	}

	secs := int64(left.Sub(joined).Seconds())
	if secs <= 0 {
		return 0, TimePlayedClampedZero
	}
	return secs, TimePlayedOK
}

// ComputeTimePlayedFor est la variante branchée sur l'abstraction
// domain.MatchTimeline : la fenêtre de gameplay [GameplayStartUTC, GameplayEndUTC]
// est dérivée du timeline (source unique du vrai début/fin). Retourne NoData si
// l'horloge absolue n'est pas renseignée (HasClock() faux).
func ComputeTimePlayedFor(in TimePlayedInput, tl domain.MatchTimeline) (int64, TimePlayedQuality) {
	if !tl.HasClock() {
		return 0, TimePlayedNoData
	}
	return ComputeTimePlayed(in, tl.GameplayStartUTC(), tl.GameplayEndUTC())
}
