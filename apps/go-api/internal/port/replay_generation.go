package port

// replay_generation.go — la génération à la demande du rejeu d'un match qui n'en a pas : ce que le handler
// demande (ReplayGeneration), la file qui exécute (ReplayGenerationQueue) et ce que la base et le disque disent
// du match (ReplayMatchProbe).

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
)

// ReplayGeneration : l'entrée du handler. Une instance par requête, liée au joueur de la page par la fabrique.
type ReplayGeneration interface {
	// RequestGeneration met le match en file. created=false : la demande était déjà en file ou en cours
	// (idempotence). Les refus se rendent par les erreurs de domain (ErrReplayGenerationNotSupported,
	// ErrReplayFilmUnavailable, ErrReplayGenerationQueueFull) ou games.ErrCapabilityNotSupported.
	RequestGeneration(ctx context.Context, titleSlug, matchID string) (domain.ReplayGenerationStatus, bool, error)
	// GenerationStatus rend l'état du match, sans rien lancer.
	GenerationStatus(ctx context.Context, titleSlug, matchID string) (domain.ReplayGenerationStatus, error)
}

// ReplayGenerationQueue : la file des demandes explicites (satisfaite par *replayregen.Service).
type ReplayGenerationQueue interface {
	// Enqueue met la demande en file. created=false : déjà en file ou en cours. Plafond atteint :
	// domain.ErrReplayGenerationQueueFull ; file non reliée : domain.ErrReplayGenerationUnbound.
	Enqueue(d domain.ReplayGenerationDemand) (domain.ReplayGenerationStatus, bool, error)
	// DemandStatus rend l'état connu de la file pour ce match ; ok=false quand elle n'en sait rien.
	DemandStatus(titleSlug, matchID string) (domain.ReplayGenerationStatus, bool)
}

// ReplayMatchProbeResult : ce que la base et le disque disent d'un match.
type ReplayMatchProbeResult struct {
	// FullID : le match_id complet du registre.
	FullID string
	// Known : le match est au registre.
	Known bool
	// UnmanagedMode : le rejeu ne fabrique pas le mode du match.
	UnmanagedMode bool
	// FilmAbsent : le bit « film absent » du registre est posé.
	FilmAbsent bool
	// ArtifactPresent : le rejeu est rangé.
	ArtifactPresent bool
	// EndedAt : l'heure de fin du match (UTC) ; zéro si le registre ne la porte pas.
	EndedAt time.Time
	// FilmArchived : le film complet est dans l'archive locale (la génération n'a besoin d'aucun jeton).
	FilmArchived bool
}

// ReplayMatchProbe lit l'état d'un match pour la génération. Implémenté au câblage (api/wire).
type ReplayMatchProbe interface {
	ProbeReplayMatch(ctx context.Context, titleSlug, matchID string) (ReplayMatchProbeResult, error)
}

// ReplayTokenPresence : le joueur a-t-il un jeton de rafraîchissement utilisable au magasin (présent, sans
// reconnexion requise) ? Lecture seule, aucun échange.
type ReplayTokenPresence func(xuid string) bool
