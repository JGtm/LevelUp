// Package teammates — teammates_service_emprise.go : LE BLOC « EMPRISE » de la page Escouade
// (lot L4 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26).
//
// Orchestration seule : le calcul est pur (analysis/squademprise.Build), les lectures (feuille de
// match, film, véhicules) sont celles de `squadagg.EmpriseLecteur`, partagées avec les Séries
// temporelles — chacune dégrade seule et le dit.
//
// Périmètre : le même que les autres blocs d'usage (D2). L'habitude (D5) n'est calculée qu'avec
// des coéquipiers sélectionnés, sur l'historique de la composition.
package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// WithEmprise injecte la feuille de match du bloc « Emprise ». Câblage inconditionnel : la
// colonne est écrite par tous les titres ; un repo qui rend games.ErrCapabilityNotSupported
// retire seulement les frags aux armes spéciales. Nil ⇒ idem, journalisé.
func (s *TeammatesService) WithEmprise(repo port.SquadEmpriseRepository) *TeammatesService {
	s.empriseRepo = repo
	return s
}

// perimetreLu — le périmètre D2 et les lectures communes du résumé d'usage déjà faites dessus
// (nil = non faites : l'Emprise les fait).
type perimetreLu struct {
	scope    []legacymatch.SynthesisMatchRow
	lectures *squadagg.LecturesUsage
}

// loadEmprise publie le bloc sur le périmètre D2. Nil quand le périmètre est vide.
func (s *TeammatesService) loadEmprise(
	ctx context.Context, playerXUID string, p porteeUsage, req domain.TeammatesQueryRequest, lu perimetreLu,
) *domain.SquadEmpriseBlock {
	scope := lu.scope
	if len(scope) == 0 || playerXUID == "" {
		return nil
	}
	defer timing.FromContext(ctx).Section("emprise")()
	current := empriseCurrent(scope, p.squadRows)
	var timeline []squademprise.Match
	if len(req.SelectedGamertags) > 0 {
		timeline = empriseTimeline(p.timelineRows)
	}
	in := squademprise.Input{
		PlayerXUID: playerXUID, Current: current, Timeline: timeline,
		Weapons:            squadagg.WeaponCatalog(ctx, s.repoRoot, s.titleSlug, req.Locale),
		SessionMatchCounts: compositionCounts(p.compositionSessions),
	}
	lecteur := squadagg.EmpriseLecteur{Page: "teammates", Player: s.gamertag, RepoRoot: s.repoRoot, TitleSlug: s.titleSlug}
	scopeIDs := teammatesMatchIDs(scope)
	in.PowerKills, in.SheetUnavailable = lecteur.Feuille(ctx, s.empriseRepo, scopeIDs)
	in.Journal = lecteur.Journal(ctx, s.empriseRepo, scopeIDs)
	in.Film, in.FilmUnavailable = lecteur.Film(ctx, s.sessionUsageRepo, current, timeline, lu.lectures)
	// La ressource véhicules vient de l'artefact, pas du résumé d'usage : sa propre lecture et sa propre
	// section, sur le périmètre ET les matchs de l'habitude.
	vehIDs := append(squadagg.EmpriseMatchIDs(current), squademprise.HabitCandidates(current, timeline)...)
	in.Vehicles, in.VehiclesUnavailable, in.VehicleLabels = lecteur.Vehicules(ctx, s.vehicleRepo, vehIDs, playerXUID, req.Locale)
	if in.Film != nil {
		in.Players = squadagg.SquadPlayers(playerXUID, s.gamertag, in.Film.Participants, p.membres)
	} else {
		in.Players = squadagg.SquadPlayers(playerXUID, s.gamertag, nil, nil)
	}
	if n := squademprise.WithoutTimeScale(&in); n > 0 {
		// Même règle et même trace que Sessions : exclus du rendement des bonus, jamais en silence.
		slog.WarnContext(ctx, "teammates_emprise_matchs_sans_echelle_de_temps_hors_rendement",
			"player", s.gamertag, "matchs_sans_echelle", n, "matchs", len(current))
	}
	block := squademprise.Build(in)
	slog.DebugContext(ctx, "teammates_emprise",
		"player", s.gamertag, "matchs", block.MatchesTotal, "mesures", block.MatchesMeasured,
		"ressources", len(block.Resources), "objets", len(block.Objects),
		"film", block.FilmUnavailable, "feuille", block.SheetUnavailable, "habitude", block.Habit != nil)
	return &block
}

// empriseCurrent — le périmètre, avec la famille de mode de chaque match (même libellé que
// l'historique de l'escouade : squadModeUI).
func empriseCurrent(scope []legacymatch.SynthesisMatchRow, squadRows []domain.SquadMatchRow) []squademprise.Match {
	family := make(map[string]string, len(squadRows))
	for _, r := range squadRows {
		family[r.MatchID] = squadModeUI(r)
	}
	out := make([]squademprise.Match, 0, len(scope))
	for _, m := range scope {
		em := squademprise.Match{MatchID: m.MatchID, StartTime: m.StartTime, Family: family[m.MatchID]}
		if m.SessionLabel != nil {
			em.SessionLabel = *m.SessionLabel
		}
		out = append(out, em)
	}
	return out
}

// empriseTimeline — l'historique de la composition, avec la famille de mode de chaque match.
func empriseTimeline(rows []domain.SquadMatchRow) []squademprise.Match {
	out := make([]squademprise.Match, 0, len(rows))
	for _, r := range rows {
		m := squademprise.Match{MatchID: r.MatchID, StartTime: r.StartTime, Family: squadModeUI(r)}
		if r.SessionLabel != nil {
			m.SessionLabel = *r.SessionLabel
		}
		out = append(out, m)
	}
	return out
}

// compositionCounts — libellé de session -> matchs de la composition : composition_sessions,
// seule source d'un compte de session en contexte escouade (ADR 0033).
func compositionCounts(sessions []domain.CompositionSessionEntry) map[string]int {
	out := make(map[string]int, len(sessions))
	for _, s := range sessions {
		out[s.Label] = s.MatchCount
	}
	return out
}
