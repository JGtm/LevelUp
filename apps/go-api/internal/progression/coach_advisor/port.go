package coach_advisor

import (
	"context"
	"errors"
	"time"
)

// ErrProposalNotFound est retournée par Repo.Get quand l'id est inconnu.
var ErrProposalNotFound = errors.New("coach_advisor: proposal not found")

// Repo persiste les proposals coach_advisor dans stats.duckdb (par joueur).
//
// Implémenté par platform/duckdb.CoachProposalRepo. Les méthodes sont scopées
// par (user_id, title_slug) — c'est le service qui passe ces valeurs.
//
// Pattern d'écriture (cf. CLAUDE.md §"Phase 4 ART") : append + SELECT-then-
// UPDATE-or-INSERT pour les transitions de status. Pas de
// ON CONFLICT DO UPDATE sur les colonnes mutables — uniquement DO NOTHING
// si jamais on duplique un id (idempotence du Create).
type Repo interface {
	// Create insère une nouvelle proposal (status='pending' par défaut, sauf si
	// p.Status est positionné explicitement). Erreur si id en collision (PK).
	Create(ctx context.Context, p Proposal) error

	// Get retourne une proposal par id. ErrProposalNotFound si inconnue.
	Get(ctx context.Context, id string) (Proposal, error)

	// ListByUser retourne les proposals filtrées par status.
	// Si status est vide, retourne tous les status. Tri par created_at DESC.
	ListByUser(ctx context.Context, userID, titleSlug string, status ProposalStatus) ([]Proposal, error)

	// ListPendingBySignalScope retourne les proposals 'pending' qui ciblent
	// la même (metric, axis) que les paramètres fournis. Utilisé pour la
	// supersession (cf. ADR 0020 §3) — au moins l'un des deux doit matcher.
	ListPendingBySignalScope(ctx context.Context, userID, titleSlug, metric, axis string) ([]Proposal, error)

	// ListPendingByAxis retourne les proposals 'pending' qui ciblent ce
	// radar_axis. Utilisé pour l'obsolescence sur completion d'un challenge
	// accepté coach (cf. ADR 0020 §3).
	ListPendingByAxis(ctx context.Context, userID, titleSlug, axis string) ([]Proposal, error)

	// MarkAccepted positionne status='accepted', resolved_at=now,
	// resolved_ref=ref (challenge_id ou arc_id selon Kind).
	MarkAccepted(ctx context.Context, id, ref string, now time.Time) error

	// MarkDismissed positionne status='dismissed', resolved_at=now.
	MarkDismissed(ctx context.Context, id string, now time.Time) error

	// MarkSuperseded positionne status='superseded', superseded_at=now,
	// superseded_by=newID. Idempotent (no-op si déjà dans cet état).
	MarkSuperseded(ctx context.Context, id, newID string, now time.Time) error

	// MarkObsoleted positionne status='obsoleted', obsoleted_at=now.
	// Appelé quand un challenge coach-accepté est complété sur même axis.
	MarkObsoleted(ctx context.Context, id string, now time.Time) error
}
