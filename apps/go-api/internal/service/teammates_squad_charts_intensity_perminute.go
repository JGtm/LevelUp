// Package service - teammates_squad_charts_intensity_perminute.go : builders
// squad intensity profile + per-minute stats. Decoupe de
// teammates_squad_charts.go (god-file split, refactor 2026-05-27).
package service

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

func (s *TeammatesService) buildSquadIntensityProfile(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag string,
	selectedGamertags []string,
	allTeamLabel string,
) *domain.SquadIntensityProfile {
	if s.repo == nil || len(allSquadRows) == 0 {
		return nil
	}

	// 1. Liste de matchs uniques + métadonnées affichage (date + carte).
	type matchMeta struct {
		startTime int64
		mapUI     string
		dateLabel string
	}
	metas := make(map[string]matchMeta)
	matchOrder := make([]string, 0, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, ok := metas[m.MatchID]; ok {
			continue
		}
		metas[m.MatchID] = matchMeta{
			startTime: m.StartTime.Unix(),
			mapUI:     m.MapUI,
			dateLabel: m.StartTime.Format("02/01"),
		}
		matchOrder = append(matchOrder, m.MatchID)
	}
	if len(matchOrder) < intensityMinMatches {
		return nil
	}
	// Tri chronologique (oldest → newest).
	sort.SliceStable(matchOrder, func(i, j int) bool {
		return metas[matchOrder[i]].startTime < metas[matchOrder[j]].startTime
	})

	// 2. Charger les kill events. LoadImpactEvents retourne kills + deaths +
	//    parfois autres types ; on filtre côté calcul.
	events, err := s.repo.LoadImpactEvents(ctx, matchOrder)
	if err != nil || len(events) == 0 {
		if err != nil {
			slog.WarnContext(ctx, "teammates_intensity_events_load_failed", "err", err)
		}
		return nil
	}
	// T0 (§4.A-bis) : ramener les TimeMS au référentiel gameplay (countdown
	// pré-match retranché) AVANT le calcul de durée et le bucketing. Le profil
	// est auto-normalisé (dénominateur = max event time), mais sans correction
	// le countdown gonfle le 1ᵉʳ bucket. T0 lu depuis allSquadRows (Q30.t0_ms).
	events = correctSquadImpactEvents(ctx, "teammates.13", events, timeline.BuildTimelinesFromSquadRows(allSquadRows))

	// 3. Pour chaque match, calculer la durée approximée = max(time_ms) sur les
	//    events corrigés. Les events pré-gameplay (TimeMS<0, countdown) sont
	//    ignorés — ils ne doivent ni gonfler le dénominateur ni être bucketés.
	maxTimeByMatch := make(map[string]int64, len(matchOrder))
	for _, e := range events {
		if e.TimeMS < 0 {
			continue
		}
		if e.TimeMS > maxTimeByMatch[e.MatchID] {
			maxTimeByMatch[e.MatchID] = e.TimeMS
		}
	}

	// 4. Résoudre xuid pour le main + chaque teammate. squadLoader.LoadFor est
	// obligatoire : playerMatchesRepo est bound au main, donc tous les toggles
	// "par joueur" affichaient le même xuid (celui du main) → mêmes kill events.
	xuidByGT := make(map[string]string)
	resolveXUID := func(gt string) {
		if _, ok := xuidByGT[gt]; ok {
			return
		}
		if s.squadLoader == nil {
			return
		}
		rows, err := s.squadLoader.LoadFor(ctx, s.titleSlug, gt, port.PlayerMatchFilters{})
		if err != nil || len(rows) == 0 {
			return
		}
		xuidByGT[gt] = rows[0].Self.Identity.XUID
	}
	resolveXUID(mainGamertag)
	for _, gt := range selectedGamertags {
		resolveXUID(gt)
	}

	// 5. Construire les options du toggle.
	options := []domain.SquadIntensityOption{{Key: "all", Label: allTeamLabel}}
	for _, gt := range append([]string{mainGamertag}, selectedGamertags...) {
		options = append(options, domain.SquadIntensityOption{Key: gt, Label: gt})
	}

	// 6. Builder per-option : 1 row par match avec 10 phases normalisées.
	buildRows := func(filterXUID string) []domain.SquadIntensityMatchRow {
		out := make([]domain.SquadIntensityMatchRow, 0, len(matchOrder))
		for _, mid := range matchOrder {
			meta := metas[mid]
			label := meta.dateLabel
			if meta.mapUI != "" {
				label = meta.mapUI + " — " + meta.dateLabel
			}
			row := domain.SquadIntensityMatchRow{MatchID: mid, Label: label}
			duration := maxTimeByMatch[mid]
			if duration <= 0 {
				out = append(out, row)
				continue
			}
			var counts [intensityBuckets]int
			for _, e := range events {
				if e.MatchID != mid {
					continue
				}
				if e.EventType != analysis.EventTypeKill {
					continue
				}
				if e.TimeMS < 0 {
					// Event pré-gameplay (countdown) après correction T0 — ignoré.
					continue
				}
				if filterXUID != "" && e.XUID != filterXUID {
					continue
				}
				bucket := int((e.TimeMS * intensityBuckets) / duration)
				if bucket < 0 {
					bucket = 0
				}
				if bucket >= intensityBuckets {
					bucket = intensityBuckets - 1
				}
				counts[bucket]++
			}
			// Normalisation par max bucket du match.
			maxC := 0
			for _, c := range counts {
				if c > maxC {
					maxC = c
				}
			}
			if maxC > 0 {
				for i, c := range counts {
					row.Phases[i] = round2(float64(c) / float64(maxC))
				}
			}
			out = append(out, row)
		}
		return out
	}

	rowsByOpt := make(map[string][]domain.SquadIntensityMatchRow, len(options))
	hasAny := false
	for _, opt := range options {
		filterXUID := ""
		if opt.Key != "all" {
			filterXUID = xuidByGT[opt.Key]
			if filterXUID == "" {
				// Joueur dont on n'a pas pu résoudre le xuid → ligne vide pour transparence
				rowsByOpt[opt.Key] = buildRows("__missing__")
				continue
			}
		}
		rows := buildRows(filterXUID)
		rowsByOpt[opt.Key] = rows
		// Tester si au moins une cellule non nulle pour marquer hasAny.
		for _, r := range rows {
			for _, p := range r.Phases {
				if p > 0 {
					hasAny = true
					break
				}
			}
			if hasAny {
				break
			}
		}
	}
	if !hasAny {
		return nil
	}

	return &domain.SquadIntensityProfile{Options: options, Rows: rowsByOpt}
}

// ---------------------------------------------------------------------------
// teammates.14 — Stats par minute (frags / morts / assists)
// ---------------------------------------------------------------------------

// buildSquadPerMinuteStats agrège sur le scope filtré, par joueur :
//   - main : depuis allSquadRows dédupliqué par match_id (Kills/Deaths/Assists/TimePlayedSecs
//     représentent le main player car c'est sa pipeline Q30).
//   - teammates : LoadPlayerMatches + filtre sur match_ids escouade, lit r.Self.{Kills,Deaths,Assists,TimePlayed}.
//
// Retourne nil si aucun joueur n'a de matchs avec time_played > 0.
func (s *TeammatesService) buildSquadPerMinuteStats(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag string,
	selectedGamertags []string,
	sessionMatchIDs map[string]bool,
) []domain.SquadPerMinuteEntry {
	if len(allSquadRows) == 0 {
		return nil
	}

	// 1. Set des match_ids du scope (post-filtres + dédup).
	matchIDsAllowed := make(map[string]struct{}, len(allSquadRows))
	for _, m := range allSquadRows {
		matchIDsAllowed[m.MatchID] = struct{}{}
	}
	if len(sessionMatchIDs) > 0 {
		filtered := make(map[string]struct{}, len(sessionMatchIDs))
		for id := range matchIDsAllowed {
			if sessionMatchIDs[id] {
				filtered[id] = struct{}{}
			}
		}
		matchIDsAllowed = filtered
	}

	// 2. Main : agrégat depuis allSquadRows dédup.
	type sums struct {
		k, d, a, secs, n int
	}
	mainSeen := make(map[string]struct{}, len(allSquadRows))
	mainAgg := sums{}
	for _, m := range allSquadRows {
		if _, dup := mainSeen[m.MatchID]; dup {
			continue
		}
		mainSeen[m.MatchID] = struct{}{}
		if _, ok := matchIDsAllowed[m.MatchID]; !ok {
			continue
		}
		mainAgg.k += m.Kills
		mainAgg.d += m.Deaths
		mainAgg.a += m.Assists
		mainAgg.secs += m.TimePlayedSecs
		mainAgg.n++
	}

	makeEntry := func(name string, agg sums) domain.SquadPerMinuteEntry {
		entry := domain.SquadPerMinuteEntry{Player: name, MatchCount: agg.n}
		if agg.secs > 0 {
			minutes := float64(agg.secs) / 60.0
			entry.KillsPerMinute = round2(float64(agg.k) / minutes)
			entry.DeathsPerMinute = round2(float64(agg.d) / minutes)
			entry.AssistsPerMinute = round2(float64(agg.a) / minutes)
		}
		return entry
	}

	entries := []domain.SquadPerMinuteEntry{makeEntry(mainGamertag, mainAgg)}

	// 3. Teammates : squadLoader.LoadFor (résolution per-gamertag) → filter par
	// match_ids → sum r.Self. squadLoader est obligatoire ici : playerMatchesRepo
	// est bound au gamertag principal et ignore l'arg gt (cf. doc TeammatesService),
	// l'utiliser ici réagrégeait les stats du main pour chaque coéquipier → toutes
	// les barres avaient la même valeur.
	for _, gt := range selectedGamertags {
		if s.squadLoader == nil {
			entries = append(entries, domain.SquadPerMinuteEntry{Player: gt})
			continue
		}
		mateRows, err := s.squadLoader.LoadFor(
			ctx, s.titleSlug, gt, port.PlayerMatchFilters{},
		)
		if err != nil {
			slog.WarnContext(ctx, "teammates_per_minute_load_failed",
				"gamertag", gt, "err", err)
			entries = append(entries, domain.SquadPerMinuteEntry{Player: gt})
			continue
		}
		mateAgg := sums{}
		for _, r := range mateRows {
			if _, ok := matchIDsAllowed[r.Summary.MatchID]; !ok {
				continue
			}
			mateAgg.k += intPtrOrZero(r.Self.Kills)
			mateAgg.d += intPtrOrZero(r.Self.Deaths)
			mateAgg.a += intPtrOrZero(r.Self.Assists)
			mateAgg.secs += intPtrOrZero(r.Self.TimePlayed)
			mateAgg.n++
		}
		entries = append(entries, makeEntry(gt, mateAgg))
	}

	// Si tous les joueurs ont 0 secondes, retourner nil (insufficient data).
	hasData := false
	for _, e := range entries {
		if e.KillsPerMinute > 0 || e.DeathsPerMinute > 0 || e.AssistsPerMinute > 0 {
			hasData = true
			break
		}
	}
	if !hasData {
		return nil
	}
	return entries
}
