// Package api — SquadProfileProvider basé sur les axes de PERFORMANCE.
//
// Source = composite de PERFORMANCE par dimension (lusr_component_history /
// CompositeWeights) : 8 composantes calculées live, normalisées 0..1. C'est la
// décomposition par axe utilisée pour le coaching — le rating LUSR v2 est un
// (μ, σ) win/loss SANS axes (et ne se décomposera pas), donc ce composite de
// perf est la source PÉRENNE du coaching. La perf est le bon levier ACTIONNABLE
// (« renforce tel axe » > « monte ton μ ») ; le lien vers le sous-tier suivant
// est un proxy (meilleure perf → plus de victoires → μ). Cohérent avec le coach
// SOLO qui utilise déjà ces composantes (axisForLUSRComponent).
//
// Implémente prestige.SquadProfileProvider : par membre-user, agrège les
// composantes de performance par axe (moyenne des CurrentAvg). L'axe le plus
// faible de l'escouade = le levier de progression collectif.

package wire

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/profile"
)

// profileWindowDaysSquad : fenêtre d'analyse pour le profil de perf d'escouade.
const profileWindowDaysSquad = 30

// squadPerfProfileProvider source les axes de performance par membre.
type squadPerfProfileProvider struct {
	players   func() ([]domain.PlayerSummary, error) // annuaire (xuid → slug)
	resolve   PlayerResolver                         // slug → PlayerDB
	titleSlug string
}

// NewSquadPerfProfileProvider construit le provider.
func NewSquadPerfProfileProvider(players func() ([]domain.PlayerSummary, error), resolve PlayerResolver, titleSlug string) *squadPerfProfileProvider {
	return &squadPerfProfileProvider{players: players, resolve: resolve, titleSlug: titleSlug}
}

var _ prestige.SquadProfileProvider = (*squadPerfProfileProvider)(nil)

// SquadAxes retourne, pour chaque membre-user ayant assez de données, son profil
// de performance par axe. Les membres hors-app (pas de slug) ou sans données
// sont ignorés (best-effort).
func (p *squadPerfProfileProvider) SquadAxes(ctx context.Context, rosterXUIDs []string, _ string) ([]map[string]float64, error) {
	if len(rosterXUIDs) == 0 || p.players == nil || p.resolve == nil {
		return nil, nil
	}
	slugByXUID, err := p.slugByXUID()
	if err != nil {
		return nil, nil // annuaire indisponible → pas de biais (best-effort)
	}
	now := time.Now().UTC()
	var out []map[string]float64
	for _, xuid := range rosterXUIDs {
		slug := slugByXUID[xuid]
		if slug == "" {
			continue // membre hors-app : pas de profil
		}
		if axes := p.memberAxes(ctx, slug, now); len(axes) > 0 {
			out = append(out, axes)
		}
	}
	slog.DebugContext(ctx, "prestige: squad profile axes",
		"roster", len(rosterXUIDs), "with_profile", len(out))
	return out, nil
}

// slugByXUID construit la map xuid → player_slug depuis l'annuaire db_profiles.
func (p *squadPerfProfileProvider) slugByXUID() (map[string]string, error) {
	players, err := p.players()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(players))
	for _, pl := range players {
		if pl.XUID != "" {
			m[pl.XUID] = pl.PlayerSlug
		}
	}
	return m, nil
}

// memberAxes construit le profil de perf par axe d'un membre via son profil.
// Best-effort : nil sur résolution/build/données insuffisantes.
func (p *squadPerfProfileProvider) memberAxes(ctx context.Context, slug string, now time.Time) map[string]float64 {
	pdb, err := p.resolve(ctx, slug)
	if err != nil || pdb == nil || pdb.Player == nil {
		return nil
	}
	prof, err := profile.NewServiceFromPlayerDB(pdb).BuildProfile(ctx, pdb.XUID, p.titleSlug, profileWindowDaysSquad, now)
	if err != nil || !prof.HasEnoughData {
		return nil
	}
	return componentsToAxes(prof.LUSRComponents)
}

// componentsToAxes agrège les composantes de performance d'un membre par axe
// (moyenne des CurrentAvg des composantes mappées sur le même axe).
func componentsToAxes(comps []profile.LUSRComponentBreakdown) map[string]float64 {
	sums := make(map[string]float64)
	counts := make(map[string]int)
	for _, c := range comps {
		axis := perfComponentAxis(c.Name)
		if axis == "" {
			continue
		}
		sums[axis] += c.CurrentAvg
		counts[axis]++
	}
	out := make(map[string]float64, len(sums))
	for axis, s := range sums {
		out[axis] = s / float64(counts[axis])
	}
	return out
}

// perfComponentAxis mappe une des 8 composantes de performance (CompositeWeights,
// sync/skill_config.go) vers un axe radar. Couvre les 8 réellement stockées.
// support/objective n'ont pas de composante directe → absents.
func perfComponentAxis(name string) string {
	switch name {
	case "kills_vs_expected", "accuracy_delta", "offensive_conversion":
		return "combat"
	case "deaths_vs_expected", "defensive_resistance":
		return "survival"
	case "win_factor":
		return "score"
	case "damage_efficiency", "medal_exploit":
		return "impact"
	}
	return ""
}
