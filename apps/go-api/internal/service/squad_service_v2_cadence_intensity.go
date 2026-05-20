// Package service — squad_service_v2_cadence_intensity.go : helpers Cadence
// + Intensite pour l'onglet Synergies de la page Squad V2 (cf.
// PLAN_SQUAD_GO_PORTAGE Phase P6).
//
//	Cadence    : ChartSeries[ChartPointStacked] - bars empilees par phase de
//	             60s, components = 1 par xuid du squad. Decoupage de l'activite
//	             kills sur la timeline du match.
//	Intensite  : ChartSeries[ChartPointHeatmap] - heatmap match x bucket
//	             normalise (0..N-1), valeur = densite events normalisee.
//
// Les deux helpers consomment []canonical.HighlightEvent charges via le
// repo HighlightEventsRepository (capability match.detail.events). Ils
// s'appuient sur les algos purs analysis/narrative/{cadence,intensity}.
package service

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// Clés ChartSeries.Key et Meta keys partagées par les helpers Squad V2.
const (
	chartKeySquadCadence  = "squad.synergies.cadence"
	chartMetaPhaseSeconds = "phase_seconds"
	chartMetaGamertag     = "gamertag"
)

// DefaultCadencePhaseSeconds est la taille de phase par defaut pour le
// decoupage cadence. 60 secondes correspond au reglage UX Squad V2 pilote.
const DefaultCadencePhaseSeconds = 60

// DefaultIntensityBuckets est le nombre de buckets normalises par defaut
// pour le profil intensite. 10 = decoupage par decile (cf. heatmap).
const DefaultIntensityBuckets = 10

// BuildCadenceChart construit une serie ChartPointStacked decrivant la
// cadence kills agregee par phase de phaseSeconds (default 60s).
//
//	Category   : "phase_<idx>" (idx 0..K-1, K = max bucket count observe).
//	Components : map xuid -> kills sur la phase.
//
// Si plusieurs matchs sont fournis, la cadence est sommee bucket par bucket
// pour donner une vision agregee (audit § 3.6 : zoom session). Le wrapper
// front rend ca en bar chart empile (1 couleur par xuid).
//
// squadXUIDs : map gamertag -> xuid (le service amont la fournit). Les
// gamertags sont utilises comme cles de Components (front-friendly).
func BuildCadenceChart(
	events []canonical.HighlightEvent,
	squadXUIDs map[string]string,
	phaseSeconds int,
) domain.ChartSeries[domain.ChartPointStacked] {
	if phaseSeconds <= 0 {
		phaseSeconds = DefaultCadencePhaseSeconds
	}
	if len(squadXUIDs) == 0 {
		return domain.ChartSeries[domain.ChartPointStacked]{
			Key:      chartKeySquadCadence,
			LabelKey: "squad.synergies.cadence_title",
			Meta: map[string]any{
				chartMetaPhaseSeconds: phaseSeconds,
			},
		}
	}
	xuids := make([]string, 0, len(squadXUIDs))
	for _, x := range squadXUIDs {
		xuids = append(xuids, x)
	}
	xuidToGT := reverseMap(squadXUIDs)

	profiles := narrative.ComputeCadenceProfiles(events, xuids, phaseSeconds)
	if len(profiles) == 0 {
		return domain.ChartSeries[domain.ChartPointStacked]{
			Key:      chartKeySquadCadence,
			LabelKey: "squad.synergies.cadence_title",
			Meta: map[string]any{
				chartMetaPhaseSeconds: phaseSeconds,
			},
		}
	}

	// Trouver le K max (longest match en buckets).
	maxBuckets := 0
	for _, p := range profiles {
		if len(p.Buckets) > maxBuckets {
			maxBuckets = len(p.Buckets)
		}
	}

	// Initialiser une matrice [bucket][gt] = 0.
	bucketAgg := make([]map[string]float64, maxBuckets)
	for i := range bucketAgg {
		bucketAgg[i] = make(map[string]float64)
		// Initialiser tous les gts a 0 pour stabilite.
		for gt := range squadXUIDs {
			bucketAgg[i][gt] = 0
		}
	}

	// Sommer les buckets par xuid -> gt.
	for _, p := range profiles {
		gt := xuidToGT[p.XUID]
		if gt == "" {
			continue
		}
		for i, count := range p.Buckets {
			if i >= maxBuckets {
				break
			}
			bucketAgg[i][gt] += float64(count)
		}
	}

	dps := make([]domain.ChartPointStacked, 0, maxBuckets)
	for i, comp := range bucketAgg {
		dps = append(dps, domain.ChartPointStacked{
			Category:   bucketCategoryLabel(i),
			Components: comp,
		})
	}
	return domain.ChartSeries[domain.ChartPointStacked]{
		Key:        chartKeySquadCadence,
		LabelKey:   "squad.synergies.cadence_title",
		Datapoints: dps,
		Meta: map[string]any{
			chartMetaPhaseSeconds: phaseSeconds,
			"bucket_count":        maxBuckets,
		},
	}
}

// bucketCategoryLabel formate un index de bucket en categorie stable.
// Le wrapper front affiche le label "0-60s", "60-120s", etc., calcule a
// partir de l'index + meta.phase_seconds. Le format "phase_NN" zero-padde
// sur 2 digits garantit le tri alphabetique stable cote ECharts (0-9 < 10-19).
func bucketCategoryLabel(idx int) string {
	return fmt.Sprintf("phase_%02d", idx)
}

// BuildIntensityHeatmap construit une heatmap ChartPointHeatmap match x
// bucket normalise. Chaque cellule = densite normalisee (0..1) du bucket
// dans le match.
//
//	X : "bucket_NN" (idx 0..nBuckets-1).
//	Y : MatchID (le wrapper resout en label match cote front si besoin).
//	Value : valeur normalisee [0..1].
//
// Tri : par MatchID asc.
func BuildIntensityHeatmap(
	events []canonical.HighlightEvent,
	nBuckets int,
) domain.ChartSeries[domain.ChartPointHeatmap] {
	if nBuckets <= 0 {
		nBuckets = DefaultIntensityBuckets
	}
	profiles := narrative.ComputeMatchIntensityProfiles(events, nBuckets)
	if len(profiles) == 0 {
		return domain.ChartSeries[domain.ChartPointHeatmap]{
			Key:      "squad.synergies.intensity",
			LabelKey: "squad.synergies.intensity_title",
			Meta: map[string]any{
				"n_buckets": nBuckets,
			},
		}
	}

	// Tri stable par MatchID (ComputeMatchIntensityProfiles garantit deja le tri,
	// on confirme).
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].MatchID < profiles[j].MatchID
	})

	dps := make([]domain.ChartPointHeatmap, 0, len(profiles)*nBuckets)
	for _, p := range profiles {
		norm := narrative.NormalizeIntensityBuckets(p.Buckets)
		for i, v := range norm {
			dps = append(dps, domain.ChartPointHeatmap{
				X:     bucketCategoryLabel(i),
				Y:     p.MatchID,
				Value: v,
				Detail: map[string]any{
					"raw":   p.Buckets[i],
					"total": p.Total,
				},
			})
		}
	}
	return domain.ChartSeries[domain.ChartPointHeatmap]{
		Key:        "squad.synergies.intensity",
		LabelKey:   "squad.synergies.intensity_title",
		Datapoints: dps,
		Meta: map[string]any{
			"n_buckets": nBuckets,
		},
	}
}
