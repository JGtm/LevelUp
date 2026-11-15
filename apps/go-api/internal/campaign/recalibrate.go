package campaign

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// recalibrate.go — recalage, au démarrage, du point de départ des campagnes en cours sur l'axe radar
// « Objectif » : leur départ a été mesuré avec une ancienne mesure de l'axe, que le suivi n'utilise plus.

// AxisObjective est l'axe radar « Objectif » : mesuré par match comme sur le radar de la vue Match (note
// d'objectif du match, 0-100), et non par une colonne de match_participants.
const AxisObjective = "objective"

// ObjectiveAxisRev est la révision de la mesure de l'axe radar « Objectif » (échantillons par match de
// SampleProvider.LoadAxisSamples). Contrat : toute modification de cette mesure change la constante dans le
// MÊME commit. Un joueur dont la révision rangée (SyncMetaObjectiveAxisRev) diffère voit, au premier
// démarrage qui suit, le départ de ses campagnes en cours sur l'axe recalé avec la mesure courante
// (Service.ApplyObjectiveAxisRev) : sans quoi le départ et le suivi ne seraient plus comparables. La
// révision détecte un changement de valeur, pas de logique : changer de mesure sans changer la constante
// ne recale rien.
const ObjectiveAxisRev = "objective-2026-11-15"

// SyncMetaObjectiveAxisRev est la clé de `sync_meta` qui porte, par joueur, la dernière valeur de
// ObjectiveAxisRev dont le recalage a réussi.
const SyncMetaObjectiveAxisRev = "campaign_objective_axis_rev"

// MeasureStart est le nouveau point de départ d'une campagne en cours (Repo.RestartMeasure).
type MeasureStart struct {
	SnapshotValue     float64
	SnapshotSample    int
	MatchesSinceStart int
}

// RevisionStore range la révision de mesure d'un joueur (sync_meta de sa base).
type RevisionStore interface {
	// Stored rend la révision rangée, "" si aucune.
	Stored(ctx context.Context) (string, error)
	// Store range la révision, sous le verrou d'écriture de la base du joueur.
	Store(ctx context.Context, rev string) error
}

// ApplyObjectiveAxisRev recale le départ des campagnes en cours du joueur sur l'axe Objectif quand la
// révision rangée diffère de ObjectiveAxisRev, puis range la révision ; rend le nombre de campagnes
// recalées. Une révision à jour ne touche à rien.
//
// La révision n'est rangée qu'après le succès du recalage de TOUTES les campagnes : un échec laisse
// l'ancienne valeur et le prochain démarrage retente (idempotent : les campagnes déjà recalées le sont
// à l'identique, leur évaluation repart à vide).
func (s *Service) ApplyObjectiveAxisRev(
	ctx context.Context, store RevisionStore, userID, titleSlug string, now time.Time,
) (int, error) {
	stored, err := store.Stored(ctx)
	if err != nil {
		return 0, fmt.Errorf("campaign: lecture de la révision de l'axe Objectif: %w", err)
	}
	if stored == ObjectiveAxisRev {
		return 0, nil
	}
	n, err := s.recalibrateObjectiveStarts(ctx, userID, titleSlug, now)
	if err != nil {
		return n, err
	}
	if err := store.Store(ctx, ObjectiveAxisRev); err != nil {
		return n, fmt.Errorf("campaign: rangement de la révision de l'axe Objectif: %w", err)
	}
	return n, nil
}

// recalibrateObjectiveStarts recale chaque campagne en cours (active ou paused) du joueur sur l'axe radar
// Objectif ; les campagnes closes et celles des autres axes ne bougent pas. Un échec est journalisé et
// n'arrête pas les autres campagnes ; l'erreur rendue les rassemble. Un titre sans statistiques
// d'objectif (WithObjectiveMeasure(false)) n'a rien à recaler : ses campagnes ne sont pas réécrites.
func (s *Service) recalibrateObjectiveStarts(
	ctx context.Context, userID, titleSlug string, now time.Time,
) (int, error) {
	if s.objectiveUnmeasured {
		slog.InfoContext(ctx, "campaign: titre sans statistiques d'objectif, campagnes de l'axe Objectif laissées telles quelles",
			"user_id", userID, "title_slug", titleSlug)
		return 0, nil
	}
	open, err := s.repo.ListOpen(ctx, userID, titleSlug)
	if err != nil {
		return 0, fmt.Errorf("campaign: liste des campagnes en cours: %w", err)
	}
	var errs []error
	recalibrated := 0
	for _, c := range open {
		if c.AxisKind != AxisKindRadar || c.Axis != AxisObjective {
			continue
		}
		if err := s.recalibrateStart(ctx, c, now); err != nil {
			slog.ErrorContext(ctx, "campaign: recalage du départ échoué",
				"campaign_id", c.ID, "user_id", c.UserID, "title_slug", c.TitleSlug, "axis", c.Axis, "err", err)
			errs = append(errs, fmt.Errorf("campagne %s: %w", c.ID, err))
			continue
		}
		recalibrated++
	}
	return recalibrated, errors.Join(errs...)
}

// recalibrateStart recalcule le départ de `c` sur la fenêtre de son démarrage (loadStartSamples, comme
// StartCampaign), recompte les matchs depuis le départ comme l'évaluation et remet l'évaluation à vide :
// la valeur courante, le test de progression et la suggestion de clôture se recalculent à la prochaine
// évaluation, sur la mesure courante. Une fenêtre sans aucun match mesuré donne un départ non mesuré
// (SnapshotSample = 0, cf. ImprovementCampaign.StartMeasured) : écrit tel quel, jamais présenté comme un 0.
func (s *Service) recalibrateStart(ctx context.Context, c ImprovementCampaign, now time.Time) error {
	samples, err := s.loadStartSamples(ctx, c.startParams(), c.StartedAt)
	if err != nil {
		return fmt.Errorf("échantillons de départ: %w", err)
	}
	post, err := s.samples.LoadAxisSamples(ctx, c.UserID, c.TitleSlug, c.Axis, c.AxisKind, c.PlaylistGroup, c.StartedAt, now)
	if err != nil {
		return fmt.Errorf("échantillons depuis le départ: %w", err)
	}
	start := MeasureStart{SnapshotValue: mean(samples), SnapshotSample: len(samples), MatchesSinceStart: len(post)}
	if err := s.repo.RestartMeasure(ctx, c.ID, start); err != nil {
		return fmt.Errorf("écriture: %w", err)
	}
	slog.InfoContext(ctx, "campaign: départ recalé sur la mesure courante de l'axe",
		"campaign_id", c.ID, "user_id", c.UserID, "title_slug", c.TitleSlug, "axis", c.Axis, "status", c.Status,
		"old_snapshot_value", c.SnapshotValue, "old_snapshot_sample", c.SnapshotSample,
		"snapshot_value", start.SnapshotValue, "snapshot_sample", start.SnapshotSample,
		"matches_since_start", start.MatchesSinceStart)
	return nil
}
