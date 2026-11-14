// Package service - teammates_squad_charts_intensity_perminute.go : builders
// squad intensity profile. Decoupe de
// teammates_squad_charts.go (god-file split, refactor 2026-05-27).
package teammates

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/observability/timing"
)

// intensityMatchMeta : métadonnées d'affichage d'un match du profil (date + carte).
type intensityMatchMeta struct {
	startTime int64
	mapUI     string
	dateLabel string
}

// intensityEventFilter décide si un kill event (match, tueur) compte pour une
// ligne du profil : joueur (xuid exact), équipe (xuid dans les alliés du match),
// lobby (tout le match).
type intensityEventFilter func(matchID, xuid string) bool

// intensityMatchOrder dédoublonne les matchs du scope et les trie du plus
// ancien au plus récent.
func intensityMatchOrder(allSquadRows []domain.SquadMatchRow) (map[string]intensityMatchMeta, []string) {
	metas := make(map[string]intensityMatchMeta)
	order := make([]string, 0, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, ok := metas[m.MatchID]; ok {
			continue
		}
		metas[m.MatchID] = intensityMatchMeta{
			startTime: m.StartTime.Unix(),
			mapUI:     m.MapUI,
			dateLabel: m.StartTime.Format("02/01"),
		}
		order = append(order, m.MatchID)
	}
	sort.SliceStable(order, func(i, j int) bool {
		return metas[order[i]].startTime < metas[order[j]].startTime
	})
	return metas, order
}

// intensityRowsBuilder produit, pour un filtre d'events, 1 ligne par match
// avec 10 phases normalisées par le max-bucket du match.
type intensityRowsBuilder struct {
	metas          map[string]intensityMatchMeta
	matchOrder     []string
	events         []domain.ImpactEventRow
	maxTimeByMatch map[string]int64
}

func (b intensityRowsBuilder) build(keep intensityEventFilter) []domain.SquadIntensityMatchRow {
	out := make([]domain.SquadIntensityMatchRow, 0, len(b.matchOrder))
	for _, mid := range b.matchOrder {
		meta := b.metas[mid]
		label := meta.dateLabel
		if meta.mapUI != "" {
			label = meta.mapUI + " — " + meta.dateLabel
		}
		row := domain.SquadIntensityMatchRow{MatchID: mid, Label: label}
		duration := b.maxTimeByMatch[mid]
		if duration <= 0 {
			out = append(out, row)
			continue
		}
		var counts [intensityBuckets]int
		for _, e := range b.events {
			if e.MatchID != mid || e.EventType != highlightevent.EventTypeKill {
				continue
			}
			if e.TimeMS < 0 {
				// Event pré-gameplay (countdown) après correction T0 — ignoré.
				continue
			}
			if !keep(mid, e.XUID) {
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

// intensityRowsHaveSignal : au moins une phase non nulle sur une ligne.
func intensityRowsHaveSignal(rows []domain.SquadIntensityMatchRow) bool {
	for _, r := range rows {
		for _, p := range r.Phases {
			if p > 0 {
				return true
			}
		}
	}
	return false
}

// buildSquadIntensityProfile charge les kill events highlight pour les matchs
// du scope et calcule, pour chaque ligne, un profil d'intensité 10 buckets × N
// matchs (normalisé par match) :
//   - `team` (domain.SquadIntensityKeyTeam) : frags des ALLIÉS du joueur
//     principal (main inclus), lus dans mainTeamByMatch (Q32b, chargé une fois
//     par GetPage). Un match sans équipe résolue → phases nulles sur ce match ;
//   - `lobby` (domain.SquadIntensityKeyLobby) : tous les frags du match, les
//     deux camps (les highlight_events du film couvrent tout le lobby) ;
//   - un gamertag par joueur (main + sélectionnés). Son xuid est celui de la page —
//     mainXUID et teammates[].XUID, les mêmes que premier frag, échange et armes —,
//     plus jamais une relecture de l'historique du joueur par LoadFor (D2.3, lot perf
//     L2) : un coéquipier sans xuid résolu garde une ligne vide.
//
// Renvoie nil si <3 matchs (section masquée), aucun kill event, ou aucune
// ligne ne produit de profil.
func (s *TeammatesService) buildSquadIntensityProfile(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	selectedGamertags []string, teammates []domain.TeammateRow,
	mainTeamByMatch map[string]map[string]struct{},
) *domain.SquadIntensityProfile {
	defer timing.FromContext(ctx).Section("intensity_profile")()
	if s.repo == nil || len(allSquadRows) == 0 {
		return nil
	}

	// 1. Liste de matchs uniques + métadonnées affichage, tri chronologique.
	metas, matchOrder := intensityMatchOrder(allSquadRows)
	if len(matchOrder) < intensityMinMatches {
		return nil
	}

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
	// le countdown gonfle le 1er bucket. T0 lu depuis allSquadRows (Q30.t0_ms).
	events = CorrectSquadImpactEvents(ctx, "teammates.13", events, timeline.BuildTimelinesFromSquadRows(allSquadRows))

	// 3. Pour chaque match, durée approximée = max(time_ms) sur les events
	//    corrigés. Les events pré-gameplay (TimeMS<0, countdown) sont ignorés —
	//    ils ne doivent ni gonfler le dénominateur ni être bucketés.
	maxTimeByMatch := make(map[string]int64, len(matchOrder))
	for _, e := range events {
		if e.TimeMS < 0 {
			continue
		}
		if e.TimeMS > maxTimeByMatch[e.MatchID] {
			maxTimeByMatch[e.MatchID] = e.TimeMS
		}
	}

	// 4. Options + filtre d'events par option. Les lignes agrégées portent leur
	//    clé en libellé (le front traduit) ; les joueurs, leur gamertag.
	xuidByGT := resolveSquadScope(allSquadRows, mainGamertag, mainXUID, teammates).xuidByPlayer
	options := []domain.SquadIntensityOption{
		{Key: domain.SquadIntensityKeyTeam, Label: domain.SquadIntensityKeyTeam},
		{Key: domain.SquadIntensityKeyLobby, Label: domain.SquadIntensityKeyLobby},
	}
	filters := map[string]intensityEventFilter{
		domain.SquadIntensityKeyTeam: func(matchID, xuid string) bool {
			_, ok := mainTeamByMatch[matchID][xuid]
			return ok
		},
		domain.SquadIntensityKeyLobby: func(string, string) bool { return true },
	}
	for _, gt := range append([]string{mainGamertag}, selectedGamertags...) {
		options = append(options, domain.SquadIntensityOption{Key: gt, Label: gt})
		// Joueur dont on n'a pas pu résoudre le xuid → ligne vide pour transparence.
		xuid := xuidByGT[gt]
		filters[gt] = func(_, x string) bool { return xuid != "" && x == xuid }
	}

	// 5. Une ligne par match et par option.
	builder := intensityRowsBuilder{metas: metas, matchOrder: matchOrder, events: events, maxTimeByMatch: maxTimeByMatch}
	rowsByOpt := make(map[string][]domain.SquadIntensityMatchRow, len(options))
	hasAny := false
	for _, opt := range options {
		rows := builder.build(filters[opt.Key])
		rowsByOpt[opt.Key] = rows
		hasAny = hasAny || intensityRowsHaveSignal(rows)
	}
	if !hasAny {
		return nil
	}

	return &domain.SquadIntensityProfile{Options: options, Rows: rowsByOpt}
}
