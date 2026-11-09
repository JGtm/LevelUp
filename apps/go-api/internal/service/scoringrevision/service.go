// Package scoringrevision rejoue, une fois au démarrage, notes de performance et LUSR de chaque
// joueur dont la révision de formule rangée diffère de celle du binaire (sync/skill.ScoringRevision).
//
// Une mise à jour qui change une formule n'a besoin d'aucun outil manuel : le premier démarrage
// qui suit recalcule l'historique de chaque joueur, une fois. Un joueur dont le recalcul échoue
// garde son ancienne révision et reste à rejouer au démarrage suivant ; son échec ne retient pas
// les autres.
package scoringrevision

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// Compteurs expvar (ADR 0009), par titre.
const (
	CompteurAppliquee = "scoring_revision_appliquee_total"
	CompteurAJour     = "scoring_revision_a_jour_total"
	CompteurErreurs   = "scoring_revision_erreurs_total"
)

// Service : le passage unique de rejeu.
type Service struct {
	a      port.ScoringRevisionApplier
	titres func() []string
}

// New construit le service ; [Service.Run] le fait travailler sur les titres du registre qui
// portent la capacité LUSR.
func New(a port.ScoringRevisionApplier) *Service {
	return &Service{a: a, titres: func() []string { return titresLUSR(title.DefaultRegistry().All()) }}
}

// titresLUSR rend les titres qui portent la capacité LUSR, dans l'ordre du registre.
func titresLUSR(tous []*title.TitleDescriptor) []string {
	var slugs []string
	for _, t := range tous {
		if t.HasCapability(title.CapLUSR) {
			slugs = append(slugs, t.Slug)
		}
	}
	return slugs
}

// Run passe en revue chaque joueur de chaque titre, un à la fois, et s'arrête entre deux joueurs
// dès que ctx est annulé. Une liste de joueurs illisible saute le titre ; les profils réservés aux
// jetons (AuthOnly) n'ont pas de base joueur. Un profil sans xuid est sauté et compté en erreur : le
// recalcul se joint sur le xuid, il n'y trouverait aucun match et effacerait toutes les notes.
func (s *Service) Run(ctx context.Context) {
	for _, slug := range s.titres() {
		players, err := s.a.FollowedPlayers(slug)
		if err != nil {
			observability.IncCounterT(slug, CompteurErreurs)
			slog.WarnContext(ctx, "scoring revision: liste des joueurs illisible, titre sauté",
				"titleSlug", slug, "err", err)
			continue
		}
		for _, p := range players {
			if ctx.Err() != nil {
				return
			}
			if p.AuthOnly {
				continue
			}
			if p.XUID == "" {
				observability.IncCounterT(slug, CompteurErreurs)
				slog.WarnContext(ctx, "scoring revision: profil sans xuid, notes laissées telles quelles",
					"player", p.Gamertag, "titleSlug", slug)
				continue
			}
			s.appliquer(ctx, slug, p)
		}
	}
}

// appliquer rejoue pour un joueur et compte l'issue.
func (s *Service) appliquer(ctx context.Context, slug string, p domain.PlayerSummary) {
	applied, err := s.a.ApplyForPlayer(ctx, slug, p)
	switch {
	case err != nil:
		observability.IncCounterT(slug, CompteurErreurs)
		slog.WarnContext(ctx, "scoring revision: recalcul échoué, retenté au prochain démarrage",
			"err", err, "player", p.Gamertag, "titleSlug", slug)
	case applied:
		observability.IncCounterT(slug, CompteurAppliquee)
		slog.InfoContext(ctx, "scoring revision: KPI recalculés après changement de formule",
			"player", p.Gamertag, "titleSlug", slug)
	default:
		observability.IncCounterT(slug, CompteurAJour)
	}
}
