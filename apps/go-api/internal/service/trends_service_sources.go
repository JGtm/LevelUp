package service

// trends_service_sources.go : les lectures optionnelles de la page Tendances
// (objectifs, équipement, médailles). Chacune est faite une fois par requête, sous
// sa propre section de durée. Une source non câblée donne un bloc absent sans
// journal ; une lecture en échec est journalisée, son bloc est omis et la page
// reste servie.

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/trends"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// trendsSampleDays : profondeur des lectures d'objectif, d'équipement et de
// médailles (deux fois l'horizon le plus long, pour la période d'avant).
const trendsSampleDays = 730

// Noms des blocs dégradés dans les journaux.
const (
	trendsBlockObjectives = "objectives"
	trendsBlockEquipment  = "equipment"
	trendsBlockMedals     = "medals"
)

// objectiveRoleRowsLoader : la lecture des lignes (match, joueur, famille) projetées par rôle, les
// deux camps. duckdb.ObjectiveStatsRepo l'implémente ; non câblée (titre sans
// match.objective.stats), le bloc objectifs est absent.
type objectiveRoleRowsLoader interface {
	LoadObjectiveRoleRows(ctx context.Context, matchIDs []string) ([]sessionusage.ObjectiveRow, error)
}

// flagGrabsNetLoader : la lecture OPTIONNELLE des prises nettes de drapeau. Interface séparée de
// la précédente : les deux grandeurs viennent de deux tables alimentées par deux producteurs (l'API
// pour les rôles, le film pour les prises nettes), et un montage qui n'a que l'une doit pouvoir
// servir l'autre sans l'implémenter.
type flagGrabsNetLoader interface {
	LoadFlagGrabsNet(ctx context.Context, matchIDs []string) ([]sessionusage.FlagGrabsNetRow, error)
}

// WithPlayerXUID fixe le xuid du joueur, clé de jonction des lectures d'objectif,
// d'équipement et de médailles. Vide : ces blocs restent absents.
func (s *TrendsService) WithPlayerXUID(xuid string) *TrendsService {
	s.playerXUID = xuid
	return s
}

// WithObjectives branche la lecture des rôles d'objectif. Les prises nettes de
// drapeau se lisent si roles les sert (flagGrabsNetLoader) ; participants donne
// les camps et effectifs.
func (s *TrendsService) WithObjectives(roles objectiveRoleRowsLoader, participants port.SessionUsageRepository) *TrendsService {
	s.objectiveRoles = roles
	s.participants = participants
	return s
}

// WithEquipmentUsage branche la lecture de l'usage de l'équipement.
func (s *TrendsService) WithEquipmentUsage(repo port.SessionUsageRepository) *TrendsService {
	s.equipment = repo
	return s
}

// WithMedals branche la lecture des médailles et celle de leurs noms.
func (s *TrendsService) WithMedals(medals port.MedalsByXUIDRepository, defs port.MedalDefinitionsRepository) *TrendsService {
	s.medals = medals
	s.medalDefs = defs
	return s
}

// trendsSources regroupe ce que les lectures optionnelles ont rendu ; un champ nil
// signifie bloc absent.
type trendsSources struct {
	objectives map[string]trends.ObjectiveSample
	equipment  map[string]trends.EquipmentSample
	medals     []trends.MedalCount
	medalNames map[int64]string
}

// readSources fait les lectures optionnelles sur les matchs ids.
func (s *TrendsService) readSources(ctx context.Context, ids []string, locale string) trendsSources {
	var out trendsSources
	if s.playerXUID == "" || len(ids) == 0 {
		return out
	}
	out.objectives = s.readObjectives(ctx, ids)
	out.equipment = s.readEquipment(ctx, ids)
	out.medals, out.medalNames = s.readMedals(ctx, ids, locale)
	return out
}

// degrade journalise une lecture en échec : au niveau debug si le titre ne porte
// pas la capacité, en avertissement sinon.
func (s *TrendsService) degrade(ctx context.Context, block string, err error) {
	if errors.Is(err, games.ErrCapabilityNotSupported) {
		slog.DebugContext(ctx, "trends: capacité absente, bloc omis",
			"titleSlug", s.titleSlug, "player", s.gamertag, "block", block, "err", err)
		return
	}
	slog.WarnContext(ctx, "trends: lecture en échec, bloc omis",
		"titleSlug", s.titleSlug, "player", s.gamertag, "block", block, "err", err)
}

// readObjectives lit les rôles d'objectif et les prises nettes sur ids, puis les
// camps des seuls matchs qui portent au moins une ligne de rôle.
func (s *TrendsService) readObjectives(ctx context.Context, ids []string) map[string]trends.ObjectiveSample {
	if s.objectiveRoles == nil || s.participants == nil {
		return nil
	}
	stop := timing.FromContext(ctx).Section("trends_objectives")
	rows, err := s.objectiveRoles.LoadObjectiveRoleRows(ctx, ids)
	var grabs []sessionusage.FlagGrabsNetRow
	if err == nil {
		if loader, ok := s.objectiveRoles.(flagGrabsNetLoader); ok {
			grabs, err = loader.LoadFlagGrabsNet(ctx, ids)
		}
	}
	stop()
	if err != nil {
		s.degrade(ctx, trendsBlockObjectives, err)
		return nil
	}
	roleMatches := distinctRoleMatchIDs(rows)
	if len(roleMatches) == 0 {
		return nil
	}
	stop = timing.FromContext(ctx).Section("trends_participants")
	participants, err := s.participants.LoadParticipants(ctx, roleMatches)
	stop()
	if err != nil {
		s.degrade(ctx, trendsBlockObjectives, err)
		return nil
	}
	return trends.ObjectiveSamples(trends.ObjectiveInput{
		PlayerXUID: s.playerXUID,
		Rows:       rows,
		FlagGrabs:  grabs,
		Teams:      sessionusage.BuildTeamContext(s.playerXUID, participants),
	})
}

// distinctRoleMatchIDs retourne, dans l'ordre de première apparition, les matchs
// qui portent au moins une ligne de rôle.
func distinctRoleMatchIDs(rows []sessionusage.ObjectiveRow) []string {
	seen := make(map[string]bool, len(rows))
	out := make([]string, 0, len(rows))
	for i := range rows {
		if id := rows[i].MatchID; !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// readEquipment lit les lignes d'usage de l'équipement sur ids.
func (s *TrendsService) readEquipment(ctx context.Context, ids []string) map[string]trends.EquipmentSample {
	if s.equipment == nil {
		return nil
	}
	stop := timing.FromContext(ctx).Section("trends_equipment")
	players, err := s.equipment.LoadUsagePlayers(ctx, ids)
	stop()
	if err != nil {
		s.degrade(ctx, trendsBlockEquipment, err)
		return nil
	}
	return trends.EquipmentSamples(s.playerXUID, players)
}

// readMedals lit les médailles du joueur sur ids, puis leurs noms dans la locale
// demandée. Un échec de l'une ou l'autre lecture omet le bloc (nil).
func (s *TrendsService) readMedals(ctx context.Context, ids []string, locale string) ([]trends.MedalCount, map[int64]string) {
	if s.medals == nil {
		return nil, nil
	}
	stop := timing.FromContext(ctx).Section("trends_medals")
	defer stop()
	rows, err := s.medals.LoadMedalsForMatchesByXUID(ctx, s.titleSlug, port.MedalsByXUIDFilters{
		MatchIDs: ids, XUIDs: []string{s.playerXUID},
	})
	if err != nil {
		s.degrade(ctx, trendsBlockMedals, err)
		return nil, nil
	}
	counts := make([]trends.MedalCount, 0, len(rows))
	seen := map[int64]bool{}
	medalIDs := make([]int64, 0)
	for _, r := range rows {
		counts = append(counts, trends.MedalCount{MatchID: r.MatchID, MedalID: r.MedalID, Count: r.Count})
		if !seen[r.MedalID] {
			seen[r.MedalID] = true
			medalIDs = append(medalIDs, r.MedalID)
		}
	}
	names := map[int64]string{}
	if s.medalDefs != nil && len(medalIDs) > 0 {
		defs, err := s.medalDefs.LookupByIDs(ctx, medalIDs, trendsLocale(locale))
		if err != nil {
			s.degrade(ctx, trendsBlockMedals, err)
			return nil, nil
		}
		for id, d := range defs {
			names[id] = d.Label
		}
	}
	return counts, names
}

// trendsLocale normalise la locale de la requête : "en" pour toute variante de
// l'anglais, "fr" sinon (défaut).
func trendsLocale(locale string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "en") {
		return "en"
	}
	return "fr"
}
