// Package service — squad_service_v2_contributions.go : helpers Contributions
// pour la page Squad V2 (cf. PLAN_SQUAD_GO_PORTAGE Phase P7).
//
//	Per-minute stats           : ChartSeries[ChartPointStacked] - 3 barres
//	                             (kills/deaths/assists per min) par joueur.
//	Frags/Deaths combined      : ChartSeries[ChartPointStacked] - 2 barres
//	                             (kills + deaths) par joueur, agreges.
//	Killing spree max          : ChartSeries[ChartPoint2D] - timeseries du
//	                             max killing spree par match, lisse via
//	                             RollingMeanAdaptive.
//	HS+PK stacked              : ChartSeries[ChartPointStacked] - barres
//	                             empilees headshots + power weapon kills par
//	                             joueur.
//
// Tous les helpers consomment le map gamertag -> []canonical.PlayerMatchRow
// fournit par le service amont.
package service

import (
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// SpreeRollingMinWindow / SpreeRollingPct controlent la fenetre adaptative du
// lissage RollingMeanAdaptive sur le killing spree (cf. portage Python pct=10).
const (
	SpreeRollingMinWindow = 5
	SpreeRollingPct       = 10
)

// BuildPerMinuteStats construit une serie de barres groupees : pour chaque
// joueur du squad, 3 valeurs (kills_per_min, deaths_per_min,
// assists_per_min) agregees sur l'ensemble des matchs partages.
//
// Calcul : sum(kills) / sum(duration_min) sur les rows ou duration et stat
// sont dispo. Les rows sans DurationSeconds sont skippees (eviter division
// par zero / biais). Les stats nil sont traitees comme 0 sur les rows
// retenues.
//
// Wrapper attendu : <BarGrouped> (3 barres par categorie joueur).
func BuildPerMinuteStats(rowsByPlayer map[string][]canonical.PlayerMatchRow) domain.ChartSeries[domain.ChartPointStacked] {
	if len(rowsByPlayer) == 0 {
		return domain.ChartSeries[domain.ChartPointStacked]{
			Key:      "squad.contrib.per_minute",
			LabelKey: "squad.contrib.per_minute_title",
		}
	}
	gts := sortedGamertags(rowsByPlayer)

	dps := make([]domain.ChartPointStacked, 0, len(gts))
	for _, gt := range gts {
		var totalMin float64
		var kills, deaths, assists int
		for _, r := range rowsByPlayer[gt] {
			if r.Summary.DurationSeconds == nil || *r.Summary.DurationSeconds <= 0 {
				continue
			}
			totalMin += float64(*r.Summary.DurationSeconds) / 60.0
			kills += derefInt(r.Self.Kills)
			deaths += derefInt(r.Self.Deaths)
			assists += derefInt(r.Self.Assists)
		}
		if totalMin <= 0 {
			continue
		}
		dps = append(dps, domain.ChartPointStacked{
			Category: gt,
			Components: map[string]float64{
				"kills_per_min":   float64(kills) / totalMin,
				"deaths_per_min":  float64(deaths) / totalMin,
				"assists_per_min": float64(assists) / totalMin,
			},
		})
	}
	return domain.ChartSeries[domain.ChartPointStacked]{
		Key:        "squad.contrib.per_minute",
		LabelKey:   "squad.contrib.per_minute_title",
		Datapoints: dps,
	}
}

// BuildFragsDeathsCombined construit une serie ou chaque categorie est un
// joueur, components = "kills" et "deaths" (totaux agreges sur les matchs).
//
// Le wrapper <BarGrouped> rend ca en barres groupees vert (kills) / rouge
// (deaths). Le delta visuel est immediat.
func BuildFragsDeathsCombined(rowsByPlayer map[string][]canonical.PlayerMatchRow) domain.ChartSeries[domain.ChartPointStacked] {
	if len(rowsByPlayer) == 0 {
		return domain.ChartSeries[domain.ChartPointStacked]{
			Key:      "squad.contrib.frags_deaths",
			LabelKey: "squad.contrib.frags_deaths_title",
		}
	}
	gts := sortedGamertags(rowsByPlayer)

	dps := make([]domain.ChartPointStacked, 0, len(gts))
	for _, gt := range gts {
		var kills, deaths int
		for _, r := range rowsByPlayer[gt] {
			kills += derefInt(r.Self.Kills)
			deaths += derefInt(r.Self.Deaths)
		}
		dps = append(dps, domain.ChartPointStacked{
			Category: gt,
			Components: map[string]float64{
				analysis.StatLabelKills:  float64(kills),
				analysis.StatLabelDeaths: float64(deaths),
			},
		})
	}
	return domain.ChartSeries[domain.ChartPointStacked]{
		Key:        "squad.contrib.frags_deaths",
		LabelKey:   "squad.contrib.frags_deaths_title",
		Datapoints: dps,
	}
}

// BuildKillingSpreeMax construit une serie multi-joueurs ou chaque trace
// represente le max_killing_spree lisse via RollingMeanAdaptive (fenetre
// pct=10 du portage Python). Y = spree lisse, X = StartedAtUTC.
//
// Le brut max_killing_spree par match est dans MatchParticipant.MaxKillingSpree
// (valeur NATIVE, cas Infinite). Quand elle est absente (Halo 5), on la CALCULE
// depuis les events kill/death du match (analysis.ComputeMaxKillingSpree) — la
// capability events-timeline du titre rend ce calcul possible. NO-OP Infinite :
// native présente → pas de recalcul. `events` est la liste des events kill/death
// (canonical, incl. synthèse kvPairs côté repo) ; `squadXUIDs` mappe gamertag→xuid
// pour attribuer les events au bon joueur. Si events/squadXUIDs sont absents, seules
// les valeurs natives sont tracées (rows sans native skippees).
func BuildKillingSpreeMax(
	rowsByPlayer map[string][]canonical.PlayerMatchRow,
	events []canonical.HighlightEvent,
	squadXUIDs map[string]string,
) []domain.ChartSeries[domain.ChartPoint2D] {
	if len(rowsByPlayer) == 0 {
		return nil
	}
	gts := sortedGamertags(rowsByPlayer)

	// Events kill/death groupes par match — pour le calcul-fallback (native absente).
	eventsByMatch := groupKillEventsByMatch(events)

	out := make([]domain.ChartSeries[domain.ChartPoint2D], 0, len(gts))
	for _, gt := range gts {
		sorted := sortedByStartedAt(rowsByPlayer[gt])
		xuid := squadXUIDs[gt]

		var rawValid []float64
		var validRows []canonical.PlayerMatchRow
		for _, r := range sorted {
			spree, ok := resolveSpreeForRow(r, xuid, eventsByMatch)
			if !ok {
				continue
			}
			rawValid = append(rawValid, float64(spree))
			validRows = append(validRows, r)
		}
		if len(rawValid) == 0 {
			continue
		}

		smoothed := temporal.RollingMeanAdaptive(rawValid, SpreeRollingMinWindow, SpreeRollingPct)
		dps := make([]domain.ChartPoint2D, 0, len(smoothed))
		for i, v := range smoothed {
			if isNaN(v) {
				continue
			}
			ts := validRows[i].Summary.StartedAtUTC
			dps = append(dps, domain.ChartPoint2D{
				X: ts,
				Y: v,
			})
		}
		if len(dps) == 0 {
			continue
		}
		out = append(out, domain.ChartSeries[domain.ChartPoint2D]{
			Key:        "squad.contrib.killing_spree." + gt,
			LabelKey:   "squad.contrib.killing_spree_title",
			Datapoints: dps,
			Meta:       map[string]any{chartMetaGamertag: gt},
		})
	}
	return out
}

// groupKillEventsByMatch indexe les events kill/death par match_id. nil si aucun
// event (le calcul-fallback est alors inopérant → seules les valeurs natives comptent).
func groupKillEventsByMatch(events []canonical.HighlightEvent) map[string][]canonical.HighlightEvent {
	if len(events) == 0 {
		return nil
	}
	byMatch := make(map[string][]canonical.HighlightEvent)
	for _, e := range events {
		byMatch[e.MatchID] = append(byMatch[e.MatchID], e)
	}
	return byMatch
}

// resolveSpreeForRow retourne le max killing spree d'un row : la valeur NATIVE quand
// elle existe (Infinite — fait foi, pas de recalcul), sinon la valeur CALCULÉE depuis
// les events kill/death du match (Halo 5). ok=false quand ni l'une ni l'autre n'est
// disponible (titre sans native ni events, ou xuid non résolu) → la row est skippée.
func resolveSpreeForRow(
	r canonical.PlayerMatchRow,
	xuid string,
	eventsByMatch map[string][]canonical.HighlightEvent,
) (int, bool) {
	if r.Self.MaxKillingSpree != nil {
		return *r.Self.MaxKillingSpree, true
	}
	if xuid == "" || eventsByMatch == nil {
		return 0, false
	}
	evs := eventsByMatch[r.Summary.MatchID]
	if len(evs) == 0 {
		return 0, false
	}
	return analysis.ComputeMaxKillingSpree(evs, xuid), true
}

// BuildHsPkStacked construit une serie de barres empilees Headshots +
// PowerWeaponKills par joueur (totaux agreges sur les matchs).
//
// Le wrapper <BarStacked> rend ca empile : headshots en bas, power weapons
// au-dessus. Records overlay = future iteration (prop recordsOverlay sur
// le wrapper).
func BuildHsPkStacked(rowsByPlayer map[string][]canonical.PlayerMatchRow) domain.ChartSeries[domain.ChartPointStacked] {
	if len(rowsByPlayer) == 0 {
		return domain.ChartSeries[domain.ChartPointStacked]{
			Key:      "squad.contrib.hs_pk",
			LabelKey: "squad.contrib.hs_pk_title",
		}
	}
	gts := sortedGamertags(rowsByPlayer)

	dps := make([]domain.ChartPointStacked, 0, len(gts))
	for _, gt := range gts {
		var hs, pk int
		for _, r := range rowsByPlayer[gt] {
			hs += derefInt(r.Self.HeadshotKills)
			pk += derefInt(r.Self.PowerWeaponKills)
		}
		dps = append(dps, domain.ChartPointStacked{
			Category: gt,
			Components: map[string]float64{
				"headshots":     float64(hs),
				"power_weapons": float64(pk),
			},
		})
	}
	return domain.ChartSeries[domain.ChartPointStacked]{
		Key:        "squad.contrib.hs_pk",
		LabelKey:   "squad.contrib.hs_pk_title",
		Datapoints: dps,
	}
}

// sortedGamertags renvoie les cles de rowsByPlayer triees alpha. Stabilise
// l'ordre des series cote consommateur.
func sortedGamertags(rowsByPlayer map[string][]canonical.PlayerMatchRow) []string {
	gts := make([]string, 0, len(rowsByPlayer))
	for gt := range rowsByPlayer {
		gts = append(gts, gt)
	}
	sort.Strings(gts)
	return gts
}

// sortedByStartedAt clone et trie chronologiquement ASC.
func sortedByStartedAt(rows []canonical.PlayerMatchRow) []canonical.PlayerMatchRow {
	sorted := make([]canonical.PlayerMatchRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Summary.StartedAtUTC.Before(sorted[j].Summary.StartedAtUTC)
	})
	return sorted
}

// derefInt resout un *int avec fallback 0 (utilise pour agreger des stats
// optionnelles sans biais).
func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// _ ensure time package usage in contributions (helper reserved for explicit
// timestamp formatting if needed in tests).
var _ = time.RFC3339
