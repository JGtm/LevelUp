package coach

import (
	"context"
	"strconv"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/patterns"
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

	// PatternReport : rapport patterns (phases 1-3). Peut être nil si patterns non calculés.
	PatternReport *patterns.PatternReport

	// CombatMedians : médianes OC/DR + résidu d'engagement sur la fenêtre courante.
	// Nil si aucune donnée de dégâts disponible pour le joueur.
	CombatMedians *CombatMedians
}

// CombatMedians regroupe les médianes OC/DR et le résidu d'engagement moyen
// calculés sur la fenêtre post-sync (120 jours).
type CombatMedians struct {
	MedianOC    float64 // médiane offensive_conversion sur la fenêtre
	MedianDR    float64 // médiane defensive_resistance sur la fenêtre
	AvgResidual float64 // moyenne du résidu d'engagement brut (peut être 0 si absent)
	HasResidual bool    // true si au moins 10 matchs avec engagement_score_brut
}

// Generator orchestre la production d'alertes coach.
type Generator struct{}

// NewGenerator construit un générateur (stateless).
func NewGenerator() *Generator { return &Generator{} }

// Generate produit la liste complète des alertes à émettre, sans dédup
// (la dédup est appliquée en aval). Ordre des alertes : streaks → records
// → milestones → LUSR → trends → comeback → patterns → combat (déterministe pour tests).
func (g *Generator) Generate(_ context.Context, input GenerateInput) []Alert {
	out := make([]Alert, 0, 16)
	out = append(out, buildStreakAlerts(input)...)
	out = append(out, buildRecordAlerts(input)...)
	out = append(out, buildMilestoneAlerts(input)...)
	out = append(out, buildLUSRTierApproachAlert(input)...)
	out = append(out, buildLOWESSAlerts(input)...)
	out = append(out, buildLOWESSSoftNegativeAlerts(input)...)
	out = append(out, buildComebackAlert(input)...)
	out = append(out, buildPatternAlerts(input)...)
	out = append(out, buildCombatPatternAlerts(input)...)
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

// recordPeriodRank ordonne les fenêtres de la plus large à la plus étroite :
// all_time (3) > 90d (2) > 30d (1). Utilisé par keepWidestPeriod (DP3).
func recordPeriodRank(p records.RecordPeriod) int {
	switch p {
	case records.RecordPeriodAllTime:
		return 3
	case records.RecordPeriod90d:
		return 2
	case records.RecordPeriod30d:
		return 1
	default:
		return 0
	}
}

// keepWidestPeriod (DP3) collapse les DetectionResult actionnables (NewPB ou
// NearMiss) par métrique en ne gardant que la fenêtre la PLUS LARGE. Un record
// KDA battu simultanément sur 30d+90d+all_time ne doit produire qu'UNE alerte
// (all_time), pas trois. Ordre de sortie déterministe (parcours des résultats
// d'entrée). Les résultats non actionnables sont écartés (buildRecordAlerts les
// ignorait déjà).
func keepWidestPeriod(results []records.DetectionResult) []records.DetectionResult {
	widest := make(map[records.TrackedMetric]int, len(results))
	for _, r := range results {
		if !r.NewPB && !r.NearMiss {
			continue
		}
		if rank := recordPeriodRank(r.Period); rank > widest[r.Metric] {
			widest[r.Metric] = rank
		}
	}
	out := make([]records.DetectionResult, 0, len(results))
	emitted := make(map[records.TrackedMetric]bool, len(results))
	for _, r := range results {
		if !r.NewPB && !r.NearMiss {
			continue
		}
		if recordPeriodRank(r.Period) != widest[r.Metric] || emitted[r.Metric] {
			continue
		}
		emitted[r.Metric] = true
		out = append(out, r)
	}
	return out
}

// buildRecordAlerts émet :
//   - personal_record (RecordBroken) si NewPB=true ET PreviousValue != nil
//     (DP12 : premier record = seed silencieux, le PB est persisté sans notif)
//   - record_near_miss si NearMiss=true (et NewPB=false)
//
// DP3 : une seule alerte par métrique, sur la fenêtre la plus large (keepWidestPeriod).
func buildRecordAlerts(input GenerateInput) []Alert {
	var out []Alert
	for _, r := range keepWidestPeriod(input.RecordResults) {
		switch {
		case r.NewPB:
			// DP12 : un NewPB sans référence antérieure (première évaluation,
			// player_records vide) n'est pas une nouvelle — seed silencieux.
			// Garde jumelle de oldRec.Loaded dans post_sync_deltas.go.
			if r.PreviousValue == nil {
				continue
			}
			params := map[string]any{
				alertParamMetric: string(r.Metric),
				"period":         string(r.Period),
				"value":          r.Value,
				"match_id":       r.MatchID,
			}
			params["previous_value"] = *r.PreviousValue
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

// LOWESSSoftNegativeThreshold : pente (sur la fenêtre, échelle 0..1) en-dessous
// de laquelle une composante en baisse soutenue déclenche un signal de
// stabilisation doux. Plancher anti-bruit (le gate de strength en aval filtre
// encore davantage).
const LOWESSSoftNegativeThreshold = -0.10

// buildLOWESSSoftNegativeAlerts émet une alerte par composante en tendance
// NÉGATIVE soutenue (slope < LOWESSSoftNegativeThreshold) sur au moins
// LOWESSObservationWindow jours — symétrique de buildLOWESSAlerts. Cadrage
// produit : opportunité de consolidation, jamais reproche.
func buildLOWESSSoftNegativeAlerts(input GenerateInput) []Alert {
	var out []Alert
	for _, t := range input.LOWESSTrends {
		if t.Slope >= LOWESSSoftNegativeThreshold || t.Window < LOWESSObservationWindow {
			continue
		}
		out = append(out, Alert{
			Type:     AlertTypeLOWESSSoftNegative,
			Severity: notifications.SeverityInfo,
			Params: map[string]any{
				"component": t.Component,
				"slope":     t.Slope,
				"window":    t.Window,
			},
			DedupKey: t.Component + "|soft_neg",
		})
	}
	return out
}

// buildPatternAlerts émet les alertes issues du Pattern Engine (phases 1-3).
// Retourne nil si input.PatternReport == nil.
func buildPatternAlerts(input GenerateInput) []Alert {
	if input.PatternReport == nil {
		return nil
	}
	report := input.PatternReport
	var out []Alert
	for _, p := range report.ContextPatterns {
		switch p.Signal {
		case patterns.SignalStrength:
			out = append(out, Alert{
				Type:     AlertPatternStrength,
				Severity: notifications.SeveritySuccess,
				Params: map[string]any{
					"context_type": string(p.Type),
					"key":          p.Key,
					"win_rate":     p.WinRate,
					"delta":        p.Delta,
				},
				DedupKey: string(p.Type) + "|" + p.Key,
			})
		case patterns.SignalWeakness:
			out = append(out, Alert{
				Type:     AlertPatternWeakness,
				Severity: notifications.SeverityWarn,
				Params: map[string]any{
					"context_type": string(p.Type),
					"key":          p.Key,
					"win_rate":     p.WinRate,
					"delta":        p.Delta,
				},
				DedupKey: string(p.Type) + "|" + p.Key,
			})
		}
	}
	for _, b := range report.BehaviorPatterns {
		if b.Severity == patterns.SeverityLow {
			continue
		}
		out = append(out, Alert{
			Type:     AlertPatternBehavior,
			Severity: notifications.SeverityWarn,
			Params: map[string]any{
				"behavior_type": string(b.Type),
				"trigger":       b.Trigger,
				"evidence":      b.Evidence,
				"severity":      string(b.Severity),
			},
			DedupKey: string(b.Type),
		})
	}
	for _, l := range report.Levers {
		if l.Rank > 3 || l.Impact <= 0.3 {
			continue
		}
		out = append(out, Alert{
			Type:     AlertPatternLever,
			Severity: notifications.SeverityInfo,
			// F3 : le levier ne porte plus de phrase — on sert les données
			// structurées (axe + contexte visé). La phrase est composée à
			// l'affichage via le gabarit i18n par axe (title-agnostic).
			Params: map[string]any{
				"rank":          l.Rank,
				"axis":          l.Axis,
				"context_key":   l.ContextKey,
				"context_label": l.ContextLabel,
				"impact":        l.Impact,
				"horizon":       l.Horizon,
			},
			DedupKey: l.Axis,
		})
	}
	return out
}

// Seuils de référence OC/DR du coach : ce sont les MÊMES valeurs que les seuils
// de milestones combat (post_sync_progression_queries.go), et elles sont
// volontairement DISTINCTES des frontières élite analysis.OffensiveConversionP80
// (0,90) et analysis.DefensiveResistanceP80 (1,65) — celles-ci normalisent
// l'échelle visuelle des jauges, pas le déclenchement d'un conseil. Le coach doit
// alerter sur un décrochage par rapport à un niveau ATTEIGNABLE, d'où le repère
// plus bas. Les aligner sur analysis changerait la sensibilité des alertes ;
// ce n'est donc PAS une dérive à corriger.
//
// Le suffixe « P80 » des noms est historique et conservé pour ne pas casser les
// tests qui les référencent (generator_test.go) — lire « seuil de référence », pas
// « percentile 80 de la population ».
const (
	combatOCP80Threshold = analysis.CombatOCReferenceThreshold
	combatDRP80Threshold = analysis.CombatDRReferenceThreshold
)

// buildCombatPatternAlerts émet des alertes proactives basées sur OC/DR/activité.
//
//   - actif   : OC basse (< 70% P80) et résidu d'engagement élevé (> +5) — joueur actif qui peut améliorer sa précision
//   - discret : résidu d'engagement très bas (< -5) — activité faible, signal de désengagement
//   - fragile : DR basse (< 70% P80) de manière systématique — défense à renforcer
//
// Les 3 alertes sont indépendantes. Retourne nil si CombatMedians == nil.
func buildCombatPatternAlerts(input GenerateInput) []Alert {
	if input.CombatMedians == nil {
		return nil
	}
	m := input.CombatMedians
	var out []Alert

	if m.MedianOC < combatOCP80Threshold*0.70 && m.HasResidual && m.AvgResidual > 5 {
		out = append(out, Alert{
			Type:     AlertTypeCombatPatternActif,
			Severity: notifications.SeverityInfo,
			Params: map[string]any{
				"median_oc":    m.MedianOC,
				"avg_residual": m.AvgResidual,
			},
			DedupKey: "combat_actif",
		})
	}

	if m.HasResidual && m.AvgResidual < -5 {
		out = append(out, Alert{
			Type:     AlertTypeCombatPatternDiscret,
			Severity: notifications.SeverityInfo,
			Params: map[string]any{
				"avg_residual": m.AvgResidual,
			},
			DedupKey: "combat_discret",
		})
	}

	// MedianDR > 0 requis : un titre sans damage_taken (ex. Halo 5) a MedianDR=0
	// → sans cette garde l'alerte « fragile » se déclencherait pour TOUS ses
	// joueurs (signal = bruit). Un vrai joueur fragile a MedianDR > 0.
	if m.MedianDR > 0 && m.MedianDR < combatDRP80Threshold*0.70 {
		out = append(out, Alert{
			Type:     AlertTypeCombatPatternFragile,
			Severity: notifications.SeverityInfo,
			Params: map[string]any{
				"median_dr": m.MedianDR,
			},
			DedupKey: "combat_fragile",
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
