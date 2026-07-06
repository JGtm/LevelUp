// Package api — progression_backfill_provider.go : adapter exposant un backfill
// progression V2 in-process, branché sur l'endpoint admin
// POST /_admin/progression/backfill/{slug} (handlers.ProgressionBackfillHandler).
//
// Réutilise EvaluateProgressionAfterSync (même évaluation idempotente que le hook
// post-sync) mais SANS coach ni émission de notifications : on ne veut que peupler
// streaks/records/milestones depuis l'historique existant, pas rejouer des alertes
// coach historiques.
//
// Pourquoi un endpoint et pas une CLI : la shared/player DB sont ouvertes par le
// process serveur (lease writer). Un binaire séparé entrerait en conflit de lease.
// L'endpoint tourne dans le process serveur → réutilise le pool, zéro conflit.
package wire

import (
	"context"
	"time"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// progressionBackfillAdapter implémente handlers.ProgressionBackfiller.
type progressionBackfillAdapter struct {
	pdb *duckdb.PlayerDB
}

// BackfillProgression force une évaluation progression idempotente puis renvoie
// le diag (counts) résultant. CoachGenerator + Emitter nil : seuls les
// détecteurs streaks/records/milestones persistent.
func (a *progressionBackfillAdapter) BackfillProgression(ctx context.Context, slug string) (*domain.ProgressionDiag, error) {
	deps := BuildPlayerProgressionDeps(a.pdb, nil)
	deps.CoachGenerator = nil
	// Titre RÉEL du joueur (pdb.TitleSlug), pas le slug HINF figé → le backfill
	// progression marche pour Halo 5 et tout futur titre (C2 title-agnostic).
	if _, err := EvaluateProgressionAfterSync(ctx, a.pdb, a.pdb.TitleSlug, deps, time.Now().UTC()); err != nil {
		return nil, err
	}
	return duckdb.NewProgressionDiagRepo(a.pdb).GetProgressionDiag(ctx, slug)
}

// ProgressionBackfillProvider implémente handlers.ProgressionBackfillFactory :
// résout slug → backfiller in-process.
func (r *ServiceRegistry) ProgressionBackfillProvider(ctx context.Context, slug string) (handlers.ProgressionBackfiller, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return &progressionBackfillAdapter{pdb: pdb}, nil
}
