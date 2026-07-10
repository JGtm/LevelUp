package records

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// detector.go — détection de PB battus à partir de l'activité joueur récente.
//
// Stratégie : pour chaque (métrique, fenêtre), on calcule la meilleure valeur
// atteinte sur la fenêtre temporelle parmi les matchs fournis. Si cette
// valeur dépasse le PB courant en DB ET que la fenêtre a au moins
// MinMatchesForRecord matchs joués, on enregistre un nouveau PB (Upsert
// player_records + Append record_history). La notification d'alerte coach
// (catégorie `personal_record`) est émise par l'orchestrateur post-sync via
// le DetectionResult retourné — le detector ne touche pas au package
// notifications pour rester pur et testable.
//
// Une fonction utilitaire `IsNearMiss` repère les valeurs proches du PB sans
// le battre, pour usage downstream du coach (commit 5).

// MatchInput est l'input minimal pour le détecteur : un match avec ses
// métriques pertinentes pré-extraites par l'orchestrateur.
type MatchInput struct {
	MatchID  string
	PlayedAt time.Time
	Metrics  map[TrackedMetric]float64
}

// DetectInput rassemble tout ce dont le détecteur a besoin pour une passe.
type DetectInput struct {
	XUID      string
	UserID    string // identique au xuid dans le contexte joueur courant
	TitleSlug string
	Now       time.Time
	// Matches : activité récente sur la fenêtre la plus large (typ. > 90j).
	// L'orchestrateur garantit qu'au moins MinMatchesForRecord matchs sont
	// fournis pour permettre la détection PB sur les 3 fenêtres.
	Matches []MatchInput
	// Metrics : sous-ensemble optionnel à évaluer. Si nil → DefaultTrackedMetrics.
	Metrics []TrackedMetric
}

// DetectionResult décrit l'effet de la détection pour une métrique × fenêtre.
//
// NewPB=true : un nouveau record a été persisté (PBRepo.Upsert + history append).
// NearMiss=true (et NewPB=false) : la valeur courante est proche du PB sans
// le battre, à utiliser par le coach pour une alerte « approche ».
type DetectionResult struct {
	Metric        TrackedMetric
	Period        RecordPeriod
	Value         float64
	PreviousValue *float64
	MatchID       string
	NewPB         bool
	NearMiss      bool
}

// Detector orchestre la détection de PB pour un joueur.
type Detector struct {
	pbRepo      PBRepo
	historyRepo HistoryRepo
	idGen       func() string
}

// NewDetector construit un détecteur avec UUID v4 par défaut.
func NewDetector(pbRepo PBRepo, historyRepo HistoryRepo) *Detector {
	return &Detector{
		pbRepo:      pbRepo,
		historyRepo: historyRepo,
		idGen:       func() string { return uuid.New().String() },
	}
}

// WithIDGen surcharge le générateur d'ID (pour tests).
func (d *Detector) WithIDGen(f func() string) *Detector {
	d.idGen = f
	return d
}

// Detect exécute une passe de détection. Pour chaque (métrique, fenêtre) :
//  1. calcule la meilleure valeur sur la fenêtre depuis input.Matches
//  2. compare au PB courant lu via pbRepo
//  3. si nouveau PB : Upsert player_records + Append record_history
//  4. sinon, si valeur > (PB × (1-NearMissRatio)) : marqué near-miss dans le résultat
//
// Retourne un DetectionResult par (métrique, fenêtre) évalué (peut inclure
// des résultats avec NewPB=false && NearMiss=false pour observabilité).
func (d *Detector) Detect(ctx context.Context, input DetectInput) ([]DetectionResult, error) {
	metrics := input.Metrics
	if metrics == nil {
		metrics = DefaultTrackedMetrics()
	}
	results := make([]DetectionResult, 0, len(metrics)*3)

	for _, metric := range metrics {
		for _, period := range AllRecordPeriods() {
			res, err := d.detectOne(ctx, input, metric, period)
			if err != nil {
				return nil, fmt.Errorf("detect %s/%s: %w", metric, period, err)
			}
			results = append(results, res)
		}
	}
	return results, nil
}

// detectOne traite une seule paire (metric, period) pour ce joueur.
func (d *Detector) detectOne(ctx context.Context, input DetectInput, metric TrackedMetric, period RecordPeriod) (DetectionResult, error) {
	best, matchID, bestPlayedAt, count := computeBestInWindow(input.Matches, metric, period, input.Now)
	result := DetectionResult{Metric: metric, Period: period, Value: best, MatchID: matchID}

	if count < MinMatchesForRecord {
		return result, nil
	}

	current, err := d.pbRepo.Get(ctx, input.XUID, string(metric), period)
	if err != nil {
		return result, err
	}

	// Nouveau PB ?
	if current == nil || best > current.Value {
		var prev *float64
		var prevAt *time.Time
		if current != nil {
			p := current.Value
			prev = &p
			if current.AchievedAt != nil {
				t := *current.AchievedAt
				prevAt = &t
			}
		}
		pb := PersonalRecord{
			XUID:               input.XUID,
			Metric:             string(metric),
			Period:             period,
			Value:              best,
			AchievedAt:         &bestPlayedAt,
			AchievedMatchID:    matchID,
			PreviousValue:      prev,
			PreviousAchievedAt: prevAt,
			UpdatedAt:          input.Now,
		}
		if err := d.pbRepo.Upsert(ctx, pb); err != nil {
			return result, fmt.Errorf("upsert PB: %w", err)
		}

		hist := RecordHistory{
			ID:         d.idGen(),
			UserID:     input.UserID,
			TitleSlug:  input.TitleSlug,
			Metric:     string(metric),
			Period:     period,
			Value:      best,
			AchievedAt: bestPlayedAt,
		}
		if err := d.historyRepo.Append(ctx, hist); err != nil {
			return result, fmt.Errorf("append history: %w", err)
		}

		result.NewPB = true
		result.PreviousValue = prev
		return result, nil
	}

	// Near-miss ?
	if IsNearMiss(best, current.Value) {
		result.NearMiss = true
		p := current.Value
		result.PreviousValue = &p
	}
	return result, nil
}

// computeBestInWindow retourne la meilleure valeur de `metric` sur la fenêtre
// `period` dans les matchs fournis. Retourne (0, "", time.Time{}, 0) si aucun
// match éligible.
func computeBestInWindow(matches []MatchInput, metric TrackedMetric, period RecordPeriod, now time.Time) (best float64, matchID string, playedAt time.Time, count int) {
	cutoff := periodCutoff(period, now)
	for _, m := range matches {
		if !cutoff.IsZero() && m.PlayedAt.Before(cutoff) {
			continue
		}
		v, ok := m.Metrics[metric]
		if !ok {
			continue
		}
		count++
		if count == 1 || v > best {
			best = v
			matchID = m.MatchID
			playedAt = m.PlayedAt
		}
	}
	return
}

// periodCutoff retourne la borne inférieure d'une fenêtre temporelle. Time
// zéro pour all_time (pas de cutoff).
func periodCutoff(period RecordPeriod, now time.Time) time.Time {
	switch period {
	case RecordPeriod30d:
		return now.AddDate(0, 0, -30)
	case RecordPeriod90d:
		return now.AddDate(0, 0, -90)
	default:
		return time.Time{}
	}
}

// IsNearMiss retourne true si `current` est proche de `target` sans l'atteindre,
// dans la bande SIGNIFICATIVE : target×(1-NearMissRatio) <= current <=
// target×(1-NearMissMinGapRatio).
//
// La borne haute (`<= target×(1-NearMissMinGapRatio)`, DP11) remplace l'ancien
// `< target` : sur la fenêtre all_time le PB stocké a été posé par un match
// toujours présent dans la fenêtre, donc le best courant est égal (ou quasi
// égal en float) au PB à chaque passe. Avec `< target`, l'incident prod
// 2026-07-03 portait target=73.333336, value=73.33 — strictement inférieur en
// float mais identique à l'affichage (2 décimales) → notif « 73.33 vs 73.33 »
// absurde. L'écart doit être significatif pour un joueur (>= 2 % du PB), pas
// seulement non-nul en float.
//
// Exposé pour usage par le coach (commit 5) qui peut détecter des near-miss
// même sur des évaluations partielles non passées par le Detector complet.
func IsNearMiss(current, target float64) bool {
	if target <= 0 {
		return false
	}
	return current >= target*(1-NearMissRatio) && current <= target*(1-NearMissMinGapRatio)
}
