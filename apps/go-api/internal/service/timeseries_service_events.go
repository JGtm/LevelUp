// Package service — timeseries_service_events.go : highlight events loader +
// agregations associees (intensity heatmap, first events distribution).
// Decoupe de timeseries_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"context"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// enrichMatchesMaxKillingSpree calcule la folie meurtrière max PAR MATCH depuis les
// events kill/death (analysis.ComputeMaxKillingSpree) et la pose sur les StatsMatchRow
// dont la valeur native est ABSENTE (nil). Sert les titres sans max_killing_spree
// natif dans le carnage (Halo 5) : les kills horodatés vivent dans killer_victim_pairs
// et sont synthétisés en events (XUID = acteur réel) par le HighlightEventsRepo.
//
// NO-OP pour les titres à valeur native (Infinite) : on n'écrase JAMAIS une valeur
// déjà présente. La spree est order-based (invariante par décalage T0), donc calculée
// sur les events BRUTS (non T0-corrigés), comme le fallback escouade.
func enrichMatchesMaxKillingSpree(
	matches []legacymatch.StatsMatchRow,
	events []canonical.HighlightEvent,
	playerXUID string,
) {
	if playerXUID == "" || len(events) == 0 {
		return
	}
	byMatch := make(map[string][]canonical.HighlightEvent, len(matches))
	for _, ev := range events {
		byMatch[ev.MatchID] = append(byMatch[ev.MatchID], ev)
	}
	for i := range matches {
		if matches[i].MaxKillingSpree != nil {
			continue // valeur native déjà présente → ne pas écraser
		}
		evs := byMatch[matches[i].MatchID]
		if len(evs) == 0 {
			continue
		}
		spree := analysis.ComputeMaxKillingSpree(evs, playerXUID)
		matches[i].MaxKillingSpree = &spree
	}
}

// ---------------------------------------------------------------------------
// First events distribution (chart .11)
// ---------------------------------------------------------------------------

// loadHighlightEvents charge les events bruts (kill / death / first_kill /
// first_death) pour les match_ids fournis. Source unique partagée par chart
// .11 (premier événement) et le heatmap d'intensité.
func (s *TimeseriesService) loadHighlightEvents(
	ctx context.Context, matchIDs []string,
) ([]canonical.HighlightEvent, error) {
	filters := port.HighlightEventFilters{
		MatchIDs: matchIDs,
		EventTypes: []canonical.HighlightEventType{
			canonical.EventKill,
			canonical.EventDeath,
			canonical.EventFirstKill,
			canonical.EventFirstDeath,
		},
	}
	if err := filters.Validate(); err != nil {
		return nil, err
	}
	return s.highlightEventsRepo.Load(ctx, filters)
}

// buildIntensityRows agrège les frags du joueur (events où KillerXUID == xuid)
// en 10 buckets normalisés [0..1] par match. Réutilise narrative.ComputeMatchIntensityProfiles
// + NormalizeIntensityBuckets sur les events filtrés solo.
//
// Le label affiché côté front est "Map — dd/MM" (lookup depuis matches).
// matchOrder : préserve l'ordre des matches du scope (latest first comme
// LoadPlayerMatches), idem squad SquadIntensityProfile.
func buildIntensityRows(
	events []canonical.HighlightEvent,
	matches []legacymatch.StatsMatchRow,
	playerXUID string,
	gameplayDurationsMS map[string]int64,
) []domain.IntensityMatchRow {
	// Filtrer les events où le joueur est tueur (frags du joueur uniquement).
	playerKills := make([]canonical.HighlightEvent, 0, len(events))
	for _, ev := range events {
		killer := ""
		if ev.KillerXUID != nil {
			killer = *ev.KillerXUID
		} else {
			killer = ev.XUID // fallback legacy : XUID = tueur sur kill events
		}
		if killer == playerXUID && (ev.EventType == string(canonical.EventKill) ||
			ev.EventType == string(canonical.EventFirstKill)) {
			playerKills = append(playerKills, ev)
		}
	}
	if len(playerKills) == 0 {
		return nil
	}
	profiles := narrative.ComputeMatchIntensityProfiles(playerKills, 10, gameplayDurationsMS)
	if len(profiles) == 0 {
		return nil
	}

	// Index match_rows pour récupérer le label « #N nom de map » — même convention
	// que les autres graphes Progression (buildMatchCategories : numéro de match +
	// carte). `matches` est déjà trié ASC (cf. GetPage), donc N = position ASC :
	// #1 = plus ancien, aligné sur la numérotation des autres charts. La date est
	// retirée (portée par l'axe X ailleurs).
	type matchMeta struct {
		label    string
		startUTC time.Time
	}
	metaByID := make(map[string]matchMeta, len(matches))
	for _, m := range matches {
		mapName := m.MapNameFR
		if mapName == "" {
			mapName = m.MapName
		}
		// Contrat du builder heatmap (squadIntensityHeatmapChart) : label
		// "Carte — date" ; le numéro #N est posé par le builder web (le
		// doubler ici rendait "#1 #1 Carte" à l'écran).
		label := m.StartTime.Format("02/01")
		if mapName != "" {
			label = mapName + " — " + label
		}
		metaByID[m.MatchID] = matchMeta{
			label:    label,
			startUTC: m.StartTime,
		}
	}

	out := make([]domain.IntensityMatchRow, 0, len(profiles))
	for _, p := range profiles {
		normalized := narrative.NormalizeIntensityBuckets(p.Buckets)
		var phases [10]float64
		for i := 0; i < 10 && i < len(normalized); i++ {
			phases[i] = normalized[i]
		}
		row := domain.IntensityMatchRow{
			MatchID: p.MatchID,
			Phases:  phases,
		}
		if meta, ok := metaByID[p.MatchID]; ok {
			row.Label = meta.label
		} else {
			row.Label = p.MatchID
		}
		out = append(out, row)
	}
	// Tri chronologique ASC (plus ancien en premier) — cohérent avec match_rows
	// (buildMatchRows préserve l'ordre ASC de `matches`) et la numérotation #N
	// (#1 = plus ancien). ComputeMatchIntensityProfiles trie par MatchID, donc on
	// ré-ordonne ici par start_time. Le heatmap front consomme cet ordre tel quel
	// (#1 en haut via yAxis.inverse), comme la page Escouade — pas de reverse client.
	sort.SliceStable(out, func(i, j int) bool {
		mi := metaByID[out[i].MatchID]
		mj := metaByID[out[j].MatchID]
		return mi.startUTC.Before(mj.startUTC)
	})
	return out
}

// buildSoloFirstBlood projette les rows d'agrégation « premier événement » du
// joueur suivi en série produit (chart lanes « Premier frag / première mort »),
// enrichie des métadonnées d'affichage du match (carte/mode/date — DEC-4,
// retours utilisateur 2026-08-29 : le tooltip ne doit plus jamais montrer
// l'uuid du match).
//
// Solo : une seule série. Partagé par la page Timeseries et la page Session —
// même contrat par match, même conversion ms → secondes (domain.NewFirstBloodPoint).
// matches sert UNIQUEMENT à la résolution carte/mode/date (même scope que rows
// côté appelants : les matchIDs qui produisent rows sont dérivés de ces mêmes
// matches) — StartTime est déjà la valeur canonique de la ligne, jamais
// recalculée ici (règle 8, timezone canonique).
// Retourne nil si aucun match ne porte de premier frag ni de première mort.
func buildSoloFirstBlood(
	player string,
	rows []narrative.FirstEventsRow,
	matches []legacymatch.StatsMatchRow,
) []domain.FirstBloodPlayerSeries {
	if player == "" || len(rows) == 0 {
		return nil
	}
	metaByMatch := make(map[string]domain.FirstBloodMatchMeta, len(matches))
	for _, m := range matches {
		metaByMatch[m.MatchID] = statsMatchRowFirstBloodMeta(m)
	}
	points := make([]domain.FirstBloodMatchPoint, 0, len(rows))
	for _, r := range rows {
		points = append(points, domain.NewFirstBloodPoint(
			r.MatchID, r.FirstKillMS, r.FirstDeathMS, metaByMatch[r.MatchID]))
	}
	series := domain.FirstBloodPlayerSeries{Player: player, Matches: points}
	if !series.HasEvents() {
		return nil
	}
	return []domain.FirstBloodPlayerSeries{series}
}

// statsMatchRowFirstBloodMeta résout les métadonnées d'affichage (carte, mode,
// date) d'un match pour le chart « premier frag / première mort ». Carte : FR
// si disponible sinon l'anglais brut (même repli que buildIntensityRows
// ci-dessus ; 2e occurrence du pattern, sous le seuil ≤2 copies avant
// centralisation — règle CLAUDE.md). Mode : analysis.ResolveModeUIWithVariant,
// résolveur canonique pair-sinon-variant déjà utilisé dans tout le package
// service — ne pas dupliquer sa logique ici.
func statsMatchRowFirstBloodMeta(m legacymatch.StatsMatchRow) domain.FirstBloodMatchMeta {
	mapUI := m.MapNameFR
	if mapUI == "" {
		mapUI = m.MapName
	}
	meta := domain.FirstBloodMatchMeta{MapUI: mapUI, StartTime: m.StartTime}
	if modeUI := analysis.ResolveModeUIWithVariant(
		&m.PairName, &m.PairNameFR, &m.GameVariantName, &m.GameVariantNameFR,
	); modeUI != nil {
		meta.ModeUI = *modeUI
	}
	return meta
}
