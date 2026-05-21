package coach

import (
	"context"
	"strconv"
	"time"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// generator.go — orchestrateur de génération d'alertes coach.
//
// Le coach consomme les sorties des autres détecteurs (streaks, records,
// milestones) PLUS des signaux qu'il calcule lui-même (LUSR tier approach,
// LOWESS positive, comeback). Il produit des Alert que l'orchestrateur
// post-sync (commit 6) émet via notifications.Emitter.
//
// Le générateur est pur : pas d'I/O. La dédup est appliquée en aval via
// FilterRecent (cf. dedup.go) avec une liste de notifications récentes
// fournie par l'orchestrateur.

// alertParamMetric est la clé "metric" du payload Params{} d'une Alert coach.
// Centralisée pour réduire la duplication littérale (records + milestones).
const alertParamMetric = "metric"

// LUSRSnapshot capture l'état LUSR pour la passe d'évaluation.
type LUSRSnapshot struct {
	// Mu courant.
	Mu float64
	// NextTierName : nom du prochain sub-tier (utilisé en param de notif).
	// Vide si le joueur est au tier max.
	NextTierName string
	// NextTierMu : seuil μ du prochain sub-tier (pour calculer le gap).
	// 0 si tier max.
	NextTierMu float64
}

// LOWESSTrend décrit la tendance LOWESS sur une composante LUSR.
type LOWESSTrend struct {
	Component string  // ex: "accuracy", "kills_vs_expected"
	Slope     float64 // pente sur la fenêtre — >0 = amélioration
	Window    int     // jours d'observation (utilisé en param)
}

// GenerateInput rassemble tous les signaux nécessaires au coach.
type GenerateInput struct {
	UserID    string
	TitleSlug string
	Now       time.Time

	// Sorties des détecteurs amont (peuvent être vides).
	StreakResults    []streaks.EvaluationResult
	RecordResults    []records.DetectionResult
	MilestoneResults []milestones.DetectionResult

	// État LUSR + tendances temporelles.
	LUSR         LUSRSnapshot
	LOWESSTrends []LOWESSTrend

	// LastMatchAt : date du dernier match avant cette passe. Si > pause
	// threshold ET la passe contient au moins 1 nouveau match → alerte
	// `comeback_welcome`. Nil si jamais joué.
	LastMatchAt *time.Time
	// HasNewActivity : true si au moins 1 match a été ingéré dans la
	// fenêtre récente (l'orchestrateur calcule).
	HasNewActivity bool
}

// Generator orchestre la production d'alertes coach.
type Generator struct{}

// NewGenerator construit un générateur (stateless).
func NewGenerator() *Generator { return &Generator{} }

// Generate produit la liste complète des alertes à émettre, sans dédup
// (la dédup est appliquée en aval). Ordre des alertes : streaks → records
// → milestones → LUSR → trends → comeback (déterministe pour tests).
func (g *Generator) Generate(_ context.Context, input GenerateInput) []Alert {
	out := make([]Alert, 0, 16)
	out = append(out, buildStreakAlerts(input)...)
	out = append(out, buildRecordAlerts(input)...)
	out = append(out, buildMilestoneAlerts(input)...)
	out = append(out, buildLUSRTierApproachAlert(input)...)
	out = append(out, buildLOWESSAlerts(input)...)
	out = append(out, buildComebackAlert(input)...)
	return out
}

// streakMilestoneThresholds liste les paliers de streak qui déclenchent une
// alerte `streak_milestone`. Cf. plan §4.3 (multiplicateur PP) — alignés
// sur les paliers de bonus (4j, 8j, 15j, 30j).
var streakMilestoneThresholds = []int{4, 8, 15, 30}

// buildStreakAlerts émet :
//   - StreakMilestone quand la nouvelle longueur atteint un palier
//   - (Pas d'alerte sur transition broken — feedback positif seulement)
func buildStreakAlerts(input GenerateInput) []Alert {
	var out []Alert
	for _, r := range input.StreakResults {
		if r.Transition != streaks.TransitionIncremented && r.Transition != streaks.TransitionShielded {
			continue
		}
		// Atteint-on un palier pile à cette longueur ?
		for _, th := range streakMilestoneThresholds {
			if r.Streak.CurrentLength == th {
				out = append(out, Alert{
					Type:     AlertTypeStreakMilestone,
					Severity: notifications.SeveritySuccess,
					Params: map[string]any{
						"streak_type": string(r.Streak.Type),
						"length":      r.Streak.CurrentLength,
						"multiplier":  streaks.PPMultiplier(r.Streak.CurrentLength),
					},
					DedupKey: string(r.Streak.Type) + "|" + strconv.Itoa(th),
				})
				break
			}
		}
	}
	return out
}

// buildRecordAlerts émet :
//   - personal_record (RecordBroken) si NewPB=true
//   - record_near_miss si NearMiss=true (et NewPB=false)
func buildRecordAlerts(input GenerateInput) []Alert {
	var out []Alert
	for _, r := range input.RecordResults {
		switch {
		case r.NewPB:
			params := map[string]any{
				alertParamMetric: string(r.Metric),
				"period":         string(r.Period),
				"value":          r.Value,
				"match_id":       r.MatchID,
			}
			if r.PreviousValue != nil {
				params["previous_value"] = *r.PreviousValue
			}
			out = append(out, Alert{
				Type:     AlertTypeRecordBroken,
				Severity: notifications.SeveritySuccess,
				Params:   params,
				DedupKey: string(r.Metric) + "|" + string(r.Period),
			})
		case r.NearMiss:
			params := map[string]any{
				alertParamMetric: string(r.Metric),
				"period":         string(r.Period),
				"value":          r.Value,
			}
			if r.PreviousValue != nil {
				params["target"] = *r.PreviousValue
			}
			out = append(out, Alert{
				Type:     AlertTypeRecordNearMiss,
				Severity: notifications.SeverityInfo,
				Params:   params,
				DedupKey: string(r.Metric) + "|" + string(r.Period),
			})
		}
	}
	return out
}

// buildMilestoneAlerts émet :
//   - milestone_unlocked si Earned=true
//   - milestone_near_miss si NearMiss=true
func buildMilestoneAlerts(input GenerateInput) []Alert {
	var out []Alert
	for _, r := range input.MilestoneResults {
		switch {
		case r.Earned:
			out = append(out, Alert{
				Type:     AlertTypeMilestoneUnlocked,
				Severity: notifications.SeveritySuccess,
				Params: map[string]any{
					"milestone_id":   r.Milestone.ID,
					"title_en":       r.Milestone.TitleEN,
					"title_fr":       r.Milestone.TitleFR,
					alertParamMetric: r.Milestone.Metric,
					"threshold":      r.Milestone.Threshold,
				},
				DedupKey: r.Milestone.ID,
			})
		case r.NearMiss:
			out = append(out, Alert{
				Type:     AlertTypeMilestoneNearMiss,
				Severity: notifications.SeverityInfo,
				Params: map[string]any{
					"milestone_id":   r.Milestone.ID,
					"title_en":       r.Milestone.TitleEN,
					"title_fr":       r.Milestone.TitleFR,
					alertParamMetric: r.Milestone.Metric,
					"threshold":      r.Milestone.Threshold,
					"progress":       r.Progress,
				},
				DedupKey: r.Milestone.ID,
			})
		}
	}
	return out
}

// buildLUSRTierApproachAlert émet une alerte si μ est à moins de
// LUSRTierApproachDelta points du prochain sub-tier.
func buildLUSRTierApproachAlert(input GenerateInput) []Alert {
	if input.LUSR.NextTierMu <= 0 || input.LUSR.NextTierName == "" {
		return nil
	}
	gap := input.LUSR.NextTierMu - input.LUSR.Mu
	if gap <= 0 || gap > LUSRTierApproachDelta {
		return nil
	}
	return []Alert{{
		Type:     AlertTypeLUSRTierApproach,
		Severity: notifications.SeverityInfo,
		Params: map[string]any{
			"current_mu":     input.LUSR.Mu,
			"next_tier_name": input.LUSR.NextTierName,
			"next_tier_mu":   input.LUSR.NextTierMu,
			"gap":            gap,
		},
		DedupKey: input.LUSR.NextTierName,
	}}
}

// buildLOWESSAlerts émet une alerte par composante en tendance positive sur
// au moins LOWESSObservationWindow jours.
func buildLOWESSAlerts(input GenerateInput) []Alert {
	var out []Alert
	for _, t := range input.LOWESSTrends {
		if t.Slope <= 0 || t.Window < LOWESSObservationWindow {
			continue
		}
		out = append(out, Alert{
			Type:     AlertTypeLOWESSPositive,
			Severity: notifications.SeverityInfo,
			Params: map[string]any{
				"component": t.Component,
				"slope":     t.Slope,
				"window":    t.Window,
			},
			DedupKey: t.Component,
		})
	}
	return out
}

// buildComebackAlert émet `comeback_welcome` si LastMatchAt est antérieur au
// seuil de pause ET HasNewActivity=true (l'utilisateur revient effectivement).
func buildComebackAlert(input GenerateInput) []Alert {
	if input.LastMatchAt == nil || !input.HasNewActivity {
		return nil
	}
	if input.Now.Sub(*input.LastMatchAt) < ComebackPauseThreshold {
		return nil
	}
	return []Alert{{
		Type:     AlertTypeComebackWelcome,
		Severity: notifications.SeveritySuccess,
		Params: map[string]any{
			"days_away": int(input.Now.Sub(*input.LastMatchAt).Hours() / 24),
		},
		DedupKey: "",
	}}
}
