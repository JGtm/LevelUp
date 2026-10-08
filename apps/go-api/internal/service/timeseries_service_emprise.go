// Package service — timeseries_service_emprise.go : L'ONGLET « USAGES » DES SÉRIES TEMPORELLES,
// l'Emprise appliquée aux matchs solo de la fenêtre (plan
// `.ai/V7.5/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`, lot L2 ; type publié :
// domain.SoloEmpriseBlock).
//
// Orchestration seule : l'assemblage est `buildSoloEmpriseBlock` (solo_emprise_block.go, partagé avec
// la page Sessions), avec la grille par carte. Le périmètre est celui du reste de la page (les matchs
// déjà filtrés).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// usagesDeps — les dépendances de l'onglet « Usages » propres à l'Emprise solo, embarquées par
// TimeseriesService (le résumé d'usage, lui, est `sessionUsageRepo`, partagé avec d'autres blocs).
type usagesDeps struct {
	// empriseRepo : la feuille de match (frags aux armes spéciales), câblée pour tout titre.
	empriseRepo port.SquadEmpriseRepository
	// vehicleRepo : la ressource véhicules, câblée sous `film.vehicle_usage` ; nil = non mesurée.
	vehicleRepo port.SquadVehicleRepository
	// emblemLoader : l'emblème du joueur consulté ; nil = initiale côté web.
	emblemLoader port.EmblemURLLoader
	// livesRepo : les vies du joueur (câblé sous `film.kill_positions`) ; nil = carte absente.
	livesRepo port.SoloLivesRepository
	// radarRange : game_variant_name -> portée du radar en mètres (`regulation.toml [radar_range_m]`).
	radarRange map[string]int
}

// WithEmprise injecte la feuille de match de l'Emprise. Câblage inconditionnel : la colonne est
// écrite par tous les titres ; un repo qui rend games.ErrCapabilityNotSupported retire seulement
// les frags aux armes spéciales.
func (s *TimeseriesService) WithEmprise(repo port.SquadEmpriseRepository) *TimeseriesService {
	s.usages.empriseRepo = repo
	return s
}

// WithVehicleUsage injecte le lecteur de la ressource véhicules (câblé sous `film.vehicle_usage`).
func (s *TimeseriesService) WithVehicleUsage(repo port.SquadVehicleRepository) *TimeseriesService {
	s.usages.vehicleRepo = repo
	return s
}

// WithEmblemLoader injecte le chargeur d'emblèmes (le même que les fiches de médailles de l'Escouade).
func (s *TimeseriesService) WithEmblemLoader(l port.EmblemURLLoader) *TimeseriesService {
	s.usages.emblemLoader = l
	return s
}

// attachEmprise pose le bloc `emprise` sur la réponse. `lu` : les lectures du résumé d'usage déjà
// faites sur la fenêtre (partagées avec les autres blocs, ADR 0036 I4) ; nil = à faire ici.
func (s *TimeseriesService) attachEmprise(
	ctx context.Context, resp *domain.TimeseriesPageResponse, filteredCanon []canonical.PlayerMatchRow,
	locale string, lu *squadagg.LecturesUsage,
) {
	if len(filteredCanon) == 0 || s.playerXUID == "" {
		return
	}
	defer timing.FromContext(ctx).Section("emprise")()
	resp.Emprise = buildSoloEmpriseBlock(ctx, soloEmpriseQuery{
		Page: "timeseries", Player: s.gamertag, PlayerXUID: s.playerXUID,
		RepoRoot: s.repoRoot, TitleSlug: s.titleSlug, Locale: locale,
		Current: timeseriesEmpriseMatches(filteredCanon, locale), Lectures: lu,
		UsageRepo: s.sessionUsageRepo, EmpriseRepo: s.usages.empriseRepo, VehicleRepo: s.usages.vehicleRepo,
		WithMaps: true,
	})
}

// attachEmblem pose l'emblème du joueur consulté (best-effort : rien sans chargeur ou sans emblème).
func (s *TimeseriesService) attachEmblem(ctx context.Context, resp *domain.TimeseriesPageResponse) {
	if s.usages.emblemLoader == nil || s.gamertag == "" {
		slog.DebugContext(ctx, "timeseries_emblem_sans_chargeur", "player", s.gamertag)
		return
	}
	resp.PlayerEmblemURL = s.usages.emblemLoader.LoadEmblemURLs(ctx, s.titleSlug, []string{s.gamertag})[s.gamertag]
}

// timeseriesEmpriseMatches projette la fenêtre en matchs de l'Emprise : carte (identifiant et nom
// dans la langue de la requête, depuis le canonique déjà chargé — même arbitrage que
// timeseriesFormesMetas) et résultat du joueur. L'ordre est rétabli par le calcul (chronologique).
func timeseriesEmpriseMatches(rows []canonical.PlayerMatchRow, locale string) []squademprise.Match {
	out := make([]squademprise.Match, 0, len(rows))
	for _, r := range rows {
		m := squademprise.Match{
			MatchID: r.Summary.MatchID, StartTime: r.Summary.StartedAtUTC, Outcome: r.Self.Outcome,
			MapLabel: labelPourLocale(r.Summary.Map, locale),
		}
		if r.Summary.Map != nil {
			m.MapKey = r.Summary.Map.ID
		}
		if r.Enrichment.SessionLabel != nil {
			m.SessionLabel = *r.Enrichment.SessionLabel
		}
		out = append(out, m)
	}
	return out
}
