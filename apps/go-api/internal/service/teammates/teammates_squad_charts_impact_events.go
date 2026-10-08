// Package service - teammates_squad_charts_impact_events.go : builders du premier frag /
// premiere mort et des kills par arme (la matrice d'impact vit dans teammates_squad_impact.go).
package teammates

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/observability/timing"
)

// intPtrOrZero retourne *p si non nil, 0 sinon.
func intPtrOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// ---------------------------------------------------------------------------
// Premier frag / première mort — séries par joueur (chart lanes, onglet Dynamique)
// ---------------------------------------------------------------------------

// buildSquadFirstBlood charge les events highlight de l'escouade, les ramène au
// référentiel gameplay (T0) et produit UNE série par joueur avec les valeurs PAR
// MATCH (first_kill_sec / first_death_sec, nil si l'événement est absent).
//
// Aucun bucketing serveur : médianes, écart et fenêtre d'axe sont dérivés côté
// front (FirstBloodLanes). L'agrégation « premier événement » est celle de
// narrative.ComputeFirstEventsByActor — noyau partagé avec les surfaces solo
// (Timeseries / Session).
//
// Retourne nil si aucun event, aucun joueur résolu, ou aucune série exploitable.
func (s *TeammatesService) buildSquadFirstBlood(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) []domain.FirstBloodPlayerSeries {
	defer timing.FromContext(ctx).Section("first_blood")()
	if s.repo == nil {
		return nil
	}
	// 1. Périmètre : matchs (chronologiques) + joueurs de l'escouade.
	matchIDs, xuidsOrdered, gtByXUID := firstBloodScope(allSquadRows, mainGamertag, mainXUID, teammates)
	if len(matchIDs) == 0 || len(xuidsOrdered) == 0 {
		return nil
	}
	// Métadonnées d'affichage (carte/mode/date) par match, pour le tooltip —
	// DEC-4 (retours utilisateur 2026-08-29) : plus jamais l'uuid du match.
	metaByMatch := squadFirstBloodMeta(allSquadRows)

	// 2. Charger les events, puis les ramener au référentiel gameplay (T0 /
	//    countdown pré-match retranché, §4.A-bis). CRITIQUE ici : le chart lit des
	//    valeurs absolues (secondes depuis le début du gameplay). T0 lu depuis
	//    allSquadRows (Q30.t0_ms) ; match sans T0 connu → identité.
	events, err := s.repo.LoadImpactEvents(ctx, matchIDs)
	if err != nil || len(events) == 0 {
		if err != nil {
			slog.WarnContext(ctx, "teammates_first_blood_load_failed", "err", err)
		}
		return nil
	}
	events = CorrectSquadImpactEvents(ctx, "squad.first_blood", events, timeline.BuildTimelinesFromSquadRows(allSquadRows))

	// 3. Agrégation partagée (min des TimeMS >= 0 par (joueur, match, type)).
	//    Les events pré-gameplay (TimeMS < 0 après correction T0) sont écartés
	//    par ComputeFirstEventsByActor : le « premier frag » est celui du
	//    GAMEPLAY, pas du countdown.
	actors := make([]narrative.FirstEventActor, 0, len(events))
	for _, e := range events {
		switch e.EventType {
		case highlightevent.EventTypeKill:
			actors = append(actors, narrative.FirstEventActor{
				MatchID: e.MatchID, XUID: e.XUID, IsKill: true, TimeMS: e.TimeMS,
			})
		case highlightevent.EventTypeDeath:
			actors = append(actors, narrative.FirstEventActor{
				MatchID: e.MatchID, XUID: e.XUID, IsKill: false, TimeMS: e.TimeMS,
			})
		}
	}
	byXUID := narrative.ComputeFirstEventsByActor(actors, xuidsOrdered, matchIDs)

	// 4. Projection en séries produit — un joueur sans aucun événement exploitable
	//    n'a pas de bande (pas de lane vide dans le chart).
	out := make([]domain.FirstBloodPlayerSeries, 0, len(xuidsOrdered))
	for _, xuid := range xuidsOrdered {
		points := make([]domain.FirstBloodMatchPoint, 0, len(matchIDs))
		for _, r := range byXUID[xuid] {
			points = append(points, domain.NewFirstBloodPoint(
				r.MatchID, r.FirstKillMS, r.FirstDeathMS, metaByMatch[r.MatchID]))
		}
		series := domain.FirstBloodPlayerSeries{Player: gtByXUID[xuid], Matches: points}
		if series.HasEvents() {
			out = append(out, series)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// squadFirstBloodMeta indexe carte/mode/date par match_id pour le tooltip du
// chart « premier frag / première mort » (DEC-4). MapUI est déjà résolu sur
// SquadMatchRow (Q30, enrichSquadMatchAssets tourne avant l'appel — cf.
// teammates_service.go) ; ModeUI réutilise squadModeUI, le résolveur canonique
// déjà partagé avec SquadMatchHistoryRow (teammates_service_assets.go) — ne
// pas dupliquer sa logique pair-sinon-variant. allSquadRows porte plusieurs
// lignes par match (une par coéquipier) : première occurrence retenue, comme
// firstBloodScope ci-dessous (les métadonnées de match sont invariantes par
// coéquipier).
func squadFirstBloodMeta(rows []domain.SquadMatchRow) map[string]domain.FirstBloodMatchMeta {
	meta := make(map[string]domain.FirstBloodMatchMeta, len(rows))
	for _, m := range rows {
		if _, ok := meta[m.MatchID]; ok {
			continue
		}
		meta[m.MatchID] = domain.FirstBloodMatchMeta{
			MapUI:     m.MapUI,
			ModeUI:    squadModeUI(m),
			StartTime: m.StartTime,
		}
	}
	return meta
}

// firstBloodScope dérive le périmètre du chart « premier frag / première mort » :
//   - matchIDs : identifiants uniques triés par start_time ASC (l'ordre ne change
//     pas les médianes mais rend le payload lisible et stable) ;
//   - xuidsOrdered : main puis coéquipiers, ordre canonique de la page ;
//   - gtByXUID : résolution xuid → gamertag pour l'étiquetage des bandes.
func firstBloodScope(
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) (matchIDs []string, xuidsOrdered []string, gtByXUID map[string]string) {
	startByMatch := make(map[string]time.Time, len(allSquadRows))
	matchIDs = make([]string, 0, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, ok := startByMatch[m.MatchID]; ok {
			continue
		}
		startByMatch[m.MatchID] = m.StartTime
		matchIDs = append(matchIDs, m.MatchID)
	}
	slices.SortStableFunc(matchIDs, func(a, b string) int {
		return startByMatch[a].Compare(startByMatch[b])
	})

	gtByXUID = make(map[string]string, 1+len(teammates))
	xuidsOrdered = make([]string, 0, 1+len(teammates))
	if mainXUID != "" {
		xuidsOrdered = append(xuidsOrdered, mainXUID)
		gtByXUID[mainXUID] = mainGamertag
	}
	for _, tm := range teammates {
		if tm.XUID == nil || *tm.XUID == "" {
			continue
		}
		if _, dup := gtByXUID[*tm.XUID]; dup {
			continue
		}
		xuidsOrdered = append(xuidsOrdered, *tm.XUID)
		gtByXUID[*tm.XUID] = tm.Gamertag
	}
	return matchIDs, xuidsOrdered, gtByXUID
}

// ---------------------------------------------------------------------------
// teammates.09 — Kills par arme — comparatif multi-joueurs
// ---------------------------------------------------------------------------

// buildSquadWeaponKills charge `LoadWeaponKills` via le squadLoader pour le
// main + chaque teammate (via leur xuid), agrège par weapon_id et trie ASC
// par total escouade (peu utilisées en haut).
//
// Match set : union des matchs où au moins un coéquipier sélectionné a joué
// avec le main (cf. spec Python `load_weapon_kills_data` qui passe les
// match_ids par joueur). Le SQL filtre par (xuid, match_id) — un joueur
// absent d'un match n'apparaît tout simplement pas dans les rows agrégés
// pour ce match. L'intersection stricte excluait tout coéquipier qui n'a
// pas joué exactement les mêmes matchs que les autres → chart absent.
//
// Renvoie nil si :
//   - squadLoader == nil (DI non câblée)
//   - aucun match commun avec au moins un coéquipier
//   - aucun joueur avec xuid résolu
//   - le repo ne renvoie aucune donnée (capability absente ou tables vides)
//
//nolint:funlen // chart-builder cohésif (load weapons → resolve xuid → kill counts).
