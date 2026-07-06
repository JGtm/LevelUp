// Package service - teammates_squad_charts_weapons_perf.go : builders
// teammates weapon kills + performance series. Decoupe de
// teammates_squad_charts.go (god-file split, refactor 2026-05-27).
package teammates

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

func (s *TeammatesService) buildSquadWeaponKills(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) *domain.SquadWeaponKills {
	if s.squadLoader == nil || len(allSquadRows) == 0 || len(teammates) == 0 {
		slog.DebugContext(ctx, "teammates_weapon_kills_skipped",
			"squad_loader_nil", s.squadLoader == nil,
			"squad_rows", len(allSquadRows),
			"teammates", len(teammates))
		return nil
	}

	// 1. Union des matchs — chaque match unique présent dans allSquadRows
	// (= match où le main a joué avec au moins un coéquipier sélectionné).
	matchSet := make(map[string]struct{})
	for _, m := range allSquadRows {
		matchSet[m.MatchID] = struct{}{}
	}
	sharedMatches := make([]string, 0, len(matchSet))
	for mid := range matchSet {
		sharedMatches = append(sharedMatches, mid)
	}
	if len(sharedMatches) == 0 {
		return nil
	}

	// 2. xuid map (main + teammates avec xuid résolu).
	xuidByPlayer := make(map[string]string)
	playersOrdered := make([]string, 0, 1+len(teammates))
	if mainXUID != "" {
		xuidByPlayer[mainGamertag] = mainXUID
		playersOrdered = append(playersOrdered, mainGamertag)
	}
	for _, tm := range teammates {
		if tm.XUID == nil || *tm.XUID == "" {
			continue
		}
		xuidByPlayer[tm.Gamertag] = *tm.XUID
		playersOrdered = append(playersOrdered, tm.Gamertag)
	}
	if len(playersOrdered) == 0 {
		return nil
	}

	xuids := make([]string, 0, len(playersOrdered))
	for _, p := range playersOrdered {
		xuids = append(xuids, xuidByPlayer[p])
	}

	// 3. 1 seul appel pour tous les xuids.
	rows, err := s.squadLoader.LoadWeaponKills(ctx, s.titleSlug, port.WeaponKillFilters{
		MatchIDs:            sharedMatches,
		XUIDs:               xuids,
		IncludeGrenadeMelee: true,
	})
	if err != nil {
		slog.WarnContext(ctx, "teammates_weapon_kills_load_failed",
			"err", err,
			"matches", len(sharedMatches),
			"xuids", len(xuids))
		return nil
	}
	if len(rows) == 0 {
		slog.DebugContext(ctx, "teammates_weapon_kills_empty_rows",
			"matches", len(sharedMatches),
			"xuids", len(xuids))
		return nil
	}

	// 4. Réindexation xuid → gamertag pour l'agrégation.
	gtByXUID := make(map[string]string, len(xuidByPlayer))
	for gt, x := range xuidByPlayer {
		gtByXUID[x] = gt
	}

	// 5. Group by weapon_id.
	type barAgg struct {
		weaponID       int64
		label          string
		isGrenadeMelee bool
		kills          map[string]int
		total          int
	}
	bars := make(map[int64]*barAgg)
	for _, r := range rows {
		gt, ok := gtByXUID[r.XUID]
		if !ok {
			continue
		}
		b, exists := bars[r.WeaponID]
		if !exists {
			b = &barAgg{
				weaponID:       r.WeaponID,
				label:          r.Label,
				isGrenadeMelee: r.IsGrenadeMelee,
				kills:          make(map[string]int),
			}
			bars[r.WeaponID] = b
		}
		b.kills[gt] += r.Kills
		b.total += r.Kills
		// Privilégier un label non-vide.
		if b.label == "" && r.Label != "" {
			b.label = r.Label
		}
	}
	if len(bars) == 0 {
		return nil
	}

	// 6. Tri ASC par TotalSquad (peu utilisées en haut), tie-break par label.
	out := make([]domain.SquadWeaponBar, 0, len(bars))
	for _, b := range bars {
		out = append(out, domain.SquadWeaponBar{
			WeaponID:       b.weaponID,
			Label:          b.label,
			IsGrenadeMelee: b.isGrenadeMelee,
			KillsByPlayer:  b.kills,
			TotalSquad:     b.total,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TotalSquad != out[j].TotalSquad {
			return out[i].TotalSquad < out[j].TotalSquad
		}
		return out[i].Label < out[j].Label
	})

	return &domain.SquadWeaponKills{
		Players: playersOrdered,
		Bars:    out,
	}
}

// buildSquadKillMechanics agrège les mécaniques de kill NATIVES Halo 5 par joueur
// (assassinats + compétences spartiate) en barres empilées (1 barre par mécanique,
// segments = coéquipiers). Même résolution xuid↔gamertag + matchs partagés que
// buildSquadWeaponKills. nil si pas de loader / pas de données / aucune mécanique
// (titre sans la capability → colonnes à 0).
func (s *TeammatesService) buildSquadKillMechanics(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) *domain.SquadKillMechanics {
	if s.squadLoader == nil || len(allSquadRows) == 0 || len(teammates) == 0 {
		return nil
	}

	matchSet := make(map[string]struct{})
	for _, m := range allSquadRows {
		matchSet[m.MatchID] = struct{}{}
	}
	sharedMatches := make([]string, 0, len(matchSet))
	for mid := range matchSet {
		sharedMatches = append(sharedMatches, mid)
	}
	if len(sharedMatches) == 0 {
		return nil
	}

	xuidByPlayer := make(map[string]string)
	playersOrdered := make([]string, 0, 1+len(teammates))
	if mainXUID != "" {
		xuidByPlayer[mainGamertag] = mainXUID
		playersOrdered = append(playersOrdered, mainGamertag)
	}
	for _, tm := range teammates {
		if tm.XUID == nil || *tm.XUID == "" {
			continue
		}
		xuidByPlayer[tm.Gamertag] = *tm.XUID
		playersOrdered = append(playersOrdered, tm.Gamertag)
	}
	if len(playersOrdered) == 0 {
		return nil
	}
	xuids := make([]string, 0, len(playersOrdered))
	for _, p := range playersOrdered {
		xuids = append(xuids, xuidByPlayer[p])
	}

	rows, err := s.squadLoader.LoadKillMechanics(ctx, s.titleSlug, port.WeaponKillFilters{
		MatchIDs: sharedMatches,
		XUIDs:    xuids,
	})
	if err != nil {
		slog.DebugContext(ctx, "teammates_kill_mechanics_skipped", "err", err)
		return nil
	}
	if len(rows) == 0 {
		return nil
	}

	gtByXUID := make(map[string]string, len(xuidByPlayer))
	for gt, x := range xuidByPlayer {
		gtByXUID[x] = gt
	}

	// 1 barre par mécanique (ordre stable), segments = joueurs.
	mechs := []struct {
		key  string
		pick func(port.KillMechanicsRow) int
	}{
		{"assassination", func(r port.KillMechanicsRow) int { return r.Assassinations }},
		{"ground_pound", func(r port.KillMechanicsRow) int { return r.GroundPound }},
		{"shoulder_bash", func(r port.KillMechanicsRow) int { return r.ShoulderBash }},
	}
	killsByMech := make([]map[string]int, len(mechs))
	totalByMech := make([]int, len(mechs))
	for i := range mechs {
		killsByMech[i] = make(map[string]int)
	}
	for _, r := range rows {
		gt, ok := gtByXUID[r.XUID]
		if !ok {
			continue
		}
		for i := range mechs {
			if v := mechs[i].pick(r); v > 0 {
				killsByMech[i][gt] += v
				totalByMech[i] += v
			}
		}
	}

	bars := make([]domain.SquadKillMechanicBar, 0, len(mechs))
	for i := range mechs {
		if totalByMech[i] == 0 {
			continue
		}
		bars = append(bars, domain.SquadKillMechanicBar{
			Mechanic:      mechs[i].key,
			KillsByPlayer: killsByMech[i],
			TotalSquad:    totalByMech[i],
		})
	}
	if len(bars) == 0 {
		return nil
	}
	return &domain.SquadKillMechanics{
		Players: playersOrdered,
		Bars:    bars,
	}
}

// ---------------------------------------------------------------------------
// teammates.16 — Charts de performance escouade (8 sous-charts par joueur)
// ---------------------------------------------------------------------------

// buildSquadPerformanceSeries construit la time-series par match × joueur
// sur l'INTERSECTION des matchs (= matchs où tous les coéquipiers
// sélectionnés + le main player étaient présents).
//
// Les MatchOrder sont alignés (0..N-1) sur les matchs triés chronologiquement
// par StartTime ASC. Tous les joueurs ont la même longueur de série.
//
//nolint:funlen // chart-builder cohésif (intersect matchs → perfs per joueur → align MatchOrder).
func (s *TeammatesService) buildSquadPerformanceSeries(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	selectedGamertags []string,
	teammates []domain.TeammateRow,
) map[string][]domain.SquadPerformanceSeriesPoint {
	if len(allSquadRows) == 0 || len(selectedGamertags) == 0 {
		return nil
	}

	// 0. Index carte localisée par match_id (MapUI est déjà enrichi en FR par
	// enrichSquadMatchAssets avant cet appel).
	mapNameByMatch := make(map[string]string, len(allSquadRows))
	for _, r := range allSquadRows {
		if _, ok := mapNameByMatch[r.MatchID]; !ok && r.MapUI != "" {
			mapNameByMatch[r.MatchID] = r.MapUI
		}
	}

	// 1. Matchs partagés : allSquadRows est déjà l'intersection composition
	// exacte — 1 row par match, dédupliquée par intersectSquadRowsByMatchID
	// (commit 851e10ef5). L'ancien comptage d'occurrences
	// (n >= len(selectedGamertags)) datait de l'ère "union avec doublons par
	// coéquipier" : sur l'input dédupliqué il rendait la série TOUJOURS vide
	// dès 2 coéquipiers sélectionnés (1 >= 2 faux).
	startTimeByMatch := make(map[string]int64, len(allSquadRows))
	sharedMatches := make([]string, 0, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, ok := startTimeByMatch[m.MatchID]; !ok {
			startTimeByMatch[m.MatchID] = m.StartTime.Unix()
			sharedMatches = append(sharedMatches, m.MatchID)
		}
	}
	if len(sharedMatches) == 0 {
		return nil
	}
	// Tri chronologique ASC.
	sort.SliceStable(sharedMatches, func(i, j int) bool {
		return startTimeByMatch[sharedMatches[i]] < startTimeByMatch[sharedMatches[j]]
	})
	matchOrderIndex := make(map[string]int, len(sharedMatches))
	for i, mid := range sharedMatches {
		matchOrderIndex[mid] = i
	}

	out := make(map[string][]domain.SquadPerformanceSeriesPoint, 1+len(selectedGamertags))

	// MaxKillingSpree : supportée dès que le titre fournit des kills HORODATÉS par
	// match (events kill/death) — true pour Halo 5 (capability events-timeline). La
	// valeur native (match_participants.max_killing_spree, cas Infinite) fait foi quand
	// elle existe ; sinon on la CALCULE depuis les events du match (calcul title-agnostic
	// analysis.ComputeMaxKillingSpree). NO-OP Infinite : native présent → pas de calcul.
	provideSpree := games.ProvidesMaxKillingSpree(s.titleSlug)

	// xuid par gamertag (main + coéquipiers résolus) — sert à attribuer les events
	// kill/death d'un match au bon joueur pour le calcul-fallback de la spree.
	xuidByGT := make(map[string]string, 1+len(teammates))
	if mainXUID != "" {
		xuidByGT[mainGamertag] = mainXUID
	}
	for _, tm := range teammates {
		if tm.XUID != nil && *tm.XUID != "" {
			xuidByGT[tm.Gamertag] = *tm.XUID
		}
	}

	// Events kill/death par match (canonical), chargés une seule fois pour le calcul
	// du max killing spree quand la valeur native est absente. Le squadLoader/repo
	// applique déjà le fallback de synthèse kvPairs → kill/death pour les titres dont
	// highlight_events ne porte pas les kills (Halo 5). Vide si provideSpree=false ou
	// si aucun event (titre sans substrat) → on retombe alors sur la valeur native (nil).
	var eventsByMatch map[string][]canonical.HighlightEvent
	if provideSpree {
		eventsByMatch = s.loadCanonicalKillEventsByMatch(ctx, sharedMatches)
	}

	// squadLoader.LoadFor résout les rows par gamertag (playerMatchesRepo est bound
	// au main et ignore l'arg gt → toutes les séries affichaient les stats du main).
	loadFor := func(gt string) []domain.SquadPerformanceSeriesPoint {
		if s.squadLoader == nil {
			return nil
		}
		rows, err := s.squadLoader.LoadFor(ctx, s.titleSlug, gt, port.PlayerMatchFilters{})
		if err != nil {
			slog.WarnContext(ctx, "teammates_perf_series_load_failed", "gamertag", gt, "err", err)
			return nil
		}
		series := make([]domain.SquadPerformanceSeriesPoint, 0, len(sharedMatches))
		for _, r := range rows {
			idx, ok := matchOrderIndex[r.Summary.MatchID]
			if !ok {
				continue
			}
			pt := domain.SquadPerformanceSeriesPoint{
				MatchID:    r.Summary.MatchID,
				StartTime:  r.Summary.StartedAtUTC.Format("2006-01-02T15:04:05Z"),
				MatchOrder: idx,
				MapName:    mapNameByMatch[r.Summary.MatchID],
				Kills:      intPtrOrZero(r.Self.Kills),
				Deaths:     intPtrOrZero(r.Self.Deaths),
				Assists:    intPtrOrZero(r.Self.Assists),
			}
			if r.Self.KDA != nil {
				v := round2(*r.Self.KDA)
				pt.KDA = &v
			}
			if r.Self.Accuracy != nil {
				v := round2(*r.Self.Accuracy)
				pt.Accuracy = &v
			}
			if r.Self.AvgLifeSeconds != nil {
				v := round2(*r.Self.AvgLifeSeconds)
				pt.AvgLifeSeconds = &v
			}
			if r.Enrichment.PerformanceScore != nil {
				v := round2(*r.Enrichment.PerformanceScore)
				pt.PerformanceScore = &v
			}
			if provideSpree {
				if r.Self.MaxKillingSpree != nil {
					// Valeur native (Infinite) — fait foi, PAS de recalcul.
					v := *r.Self.MaxKillingSpree
					pt.MaxKillingSpree = &v
				} else if xuid := xuidByGT[gt]; xuid != "" {
					// Native absente (Halo 5) — calcul depuis les events kill/death du match.
					if evs := eventsByMatch[r.Summary.MatchID]; len(evs) > 0 {
						v := analysis.ComputeMaxKillingSpree(evs, xuid)
						pt.MaxKillingSpree = &v
					}
				}
			}
			if r.Self.HeadshotKills != nil {
				v := *r.Self.HeadshotKills
				pt.HeadshotKills = &v
			}
			if r.Self.PerfectKills != nil {
				v := *r.Self.PerfectKills
				pt.PerfectKills = &v
			}
			// Ventilation par type de frag (barres empilées « répartition des frags »).
			if r.Self.MeleeKills != nil {
				v := *r.Self.MeleeKills
				pt.MeleeKills = &v
			}
			if r.Self.PowerWeaponKills != nil {
				v := *r.Self.PowerWeaponKills
				pt.PowerWeaponKills = &v
			}
			if r.Self.GrenadeKills != nil {
				v := *r.Self.GrenadeKills
				pt.GrenadeKills = &v
			}
			if r.Self.DamageDealt != nil {
				dd := *r.Self.DamageDealt
				pt.DamageDealt = &dd // brut, pour le chart dégâts/frag
				if dd > 0 {
					v := round2(SynergyOffensiveConversion(pt.Kills, pt.Assists, float64(dd), games.EffectiveHpToKill(s.titleSlug)))
					pt.RendementOffensif = &v
				}
			}
			if r.Self.DamageTaken != nil {
				dt := *r.Self.DamageTaken
				pt.DamageTaken = &dt // brut, pour le chart dégâts/mort
				v := round2(SynergyDefensiveResistance(float64(dt), pt.Deaths, games.EffectiveHpToKill(s.titleSlug)))
				pt.ResistanceDefensive = &v
			}
			if r.Enrichment.TeamMMR != nil {
				v := round2(*r.Enrichment.TeamMMR)
				pt.TeamMMR = &v
			}
			if ss := r.Enrichment.SkillSnapshot; ss != nil {
				if ss.RatingValue != nil {
					v := round2(*ss.RatingValue)
					pt.SkillRating = &v
				}
				if ss.Delta != nil {
					v := round2(*ss.Delta)
					pt.SkillDelta = &v
				}
				pt.SkillRatingType = string(ss.RatingType)
				pt.SkillPlaylistGroup = ss.PlaylistGroup
				pt.SkillSeasonID = ss.SeasonID
				pt.SkillMeasurementRemaining = ss.MeasurementRemaining
			}
			series = append(series, pt)
		}
		// Sort by MatchOrder pour garantir l'alignement avec les autres joueurs.
		sort.SliceStable(series, func(i, j int) bool {
			return series[i].MatchOrder < series[j].MatchOrder
		})
		return series
	}

	for _, gt := range append([]string{mainGamertag}, selectedGamertags...) {
		s := loadFor(gt)
		if len(s) > 0 {
			out[gt] = s
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ---------------------------------------------------------------------------
// teammates.06 — Radar synergie escouade (6 axes)
// ---------------------------------------------------------------------------

// synergyRadarThresholds retourne les seuils du radar scalés par nShared.
// Les axes absolus (combat/support/score/objective) sont proportionnels au
// nombre de matchs partagés. Impact et Survival sont des ratios agrégés →
// seuil fixe étiré au-dessus du P80 observé pour préserver la variance
// inter-joueurs (un squad de joueurs corrects dépasse tous le P80, ce qui
// aplatissait les axes à 100).
//
// Combat calibration : accuracy est stockée 0..100 en DB (voir transforms.go).
// Dans loadSynergyMateAxes/synergyMainFallbackAxes acc est normalisé /100
// avant usage → multiplicateur réel = 1 + (acc/100)×0.4 ≈ 1.2 pour 50%.
// Un bon joueur (~12K, 5HS, 2PK, 55% acc) produit ~19/match, seuil 25 ≈ 76%.
