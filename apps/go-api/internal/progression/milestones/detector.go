package milestones

import (
	"context"
	"fmt"
	"time"
)

// detector.go — détection de milestones débloqués à partir des stats agrégées
// d'un joueur.
//
// Stratégie : pour chaque entrée du catalogue NON encore débloquée pour ce
// joueur, on regarde si la valeur courante de la métrique atteint le seuil.
// Si oui → Append earned + notif `milestone_unlocked` (par l'orchestrateur).
// Sinon, si la valeur est >= seuil × (1 - NearMissRatio), on signale un
// near-miss dans le DetectionResult (pour usage downstream par le coach).
//
// Les "stats agrégées" sont passées en input — le détecteur ne touche pas
// à la DB pour les calculer (responsabilité de l'orchestrateur, commit 6).

// PlayerStats encapsule les métriques cumulatives d'un joueur sur un titre.
// Les clés sont les `metric` du catalogue (matches_played, wins, kills, etc.).
type PlayerStats struct {
	Metrics map[string]float64
}

// DetectInput rassemble tout ce dont le détecteur a besoin pour une passe.
type DetectInput struct {
	UserID    string
	TitleSlug string
	Stats     PlayerStats
	Now       time.Time
}

// DetectionResult décrit l'effet de la détection pour un milestone.
//
// Earned=true : le milestone vient d'être débloqué (Append fait).
// AlreadyHad=true : le milestone était déjà débloqué (no-op).
// NearMiss=true : la valeur est proche du seuil sans l'atteindre.
// Progress : ratio valeur/seuil (0..1+).
type DetectionResult struct {
	Milestone  CatalogEntry
	Earned     bool
	AlreadyHad bool
	NearMiss   bool
	Progress   float64
}

// Detector orchestre la détection de milestones pour un joueur.
type Detector struct {
	catalog CatalogRepo
	earned  EarnedRepo
}

// NewDetector construit un détecteur.
func NewDetector(catalog CatalogRepo, earned EarnedRepo) *Detector {
	return &Detector{catalog: catalog, earned: earned}
}

// Detect exécute une passe de détection. Pour chaque milestone du catalogue
// du titre courant : check si déjà earned, sinon évalue le seuil et persiste
// si débloqué. Retourne un résultat par milestone pour permettre l'observabilité
// et l'émission de notifs par l'orchestrateur.
func (d *Detector) Detect(ctx context.Context, input DetectInput) ([]DetectionResult, error) {
	entries, err := d.catalog.ListByTitle(ctx, input.TitleSlug)
	if err != nil {
		return nil, fmt.Errorf("milestones: list catalog: %w", err)
	}
	results := make([]DetectionResult, 0, len(entries))
	for _, entry := range entries {
		res, err := d.detectOne(ctx, input, entry)
		if err != nil {
			return nil, fmt.Errorf("milestones: detect %s: %w", entry.ID, err)
		}
		results = append(results, res)
	}
	return results, nil
}

// detectOne traite une seule entrée du catalogue.
func (d *Detector) detectOne(ctx context.Context, input DetectInput, entry CatalogEntry) (DetectionResult, error) {
	value := input.Stats.Metrics[entry.Metric]
	progress := 0.0
	if entry.Threshold > 0 {
		progress = value / entry.Threshold
	}
	result := DetectionResult{Milestone: entry, Progress: progress}

	already, err := d.earned.IsEarned(ctx, input.UserID, input.TitleSlug, entry.ID)
	if err != nil {
		return result, fmt.Errorf("isEarned: %w", err)
	}
	if already {
		result.AlreadyHad = true
		return result, nil
	}

	switch {
	case value >= entry.Threshold:
		// Débloqué.
		ent := Earned{
			UserID:      input.UserID,
			TitleSlug:   input.TitleSlug,
			MilestoneID: entry.ID,
			EarnedAt:    input.Now,
		}
		if err := d.earned.Append(ctx, ent); err != nil {
			return result, fmt.Errorf("append earned: %w", err)
		}
		result.Earned = true

	case value >= entry.Threshold*(1-NearMissRatio):
		// Proche du seuil mais pas encore débloqué.
		result.NearMiss = true
	}
	return result, nil
}
