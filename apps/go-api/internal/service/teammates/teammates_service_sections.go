// Package teammates — teammates_service_sections.go : LES SECTIONS DE LA POPULATION ESCOUADE
// de GetPage — tableaux (cartes, historique, frise) puis graphes —, calculées en séquence et
// interrompues à la première annulation de la requête (D2.7, lot perf L2, 2026-09-23).
//
// Extrait de GetPage, déjà orchestrateur au-delà du seuil funlen : y poser la vérification
// d'annulation entre chaque section l'aurait allongé. Aucune section n'y a changé d'entrée ni
// d'ordre : c'est le même enchaînement, section par section, qu'avant l'extraction.
package teammates

import (
	"context"
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// populationEscouade : ce que lisent les sections calculées sur la population escouade.
type populationEscouade struct {
	playerXUID string
	req        domain.TeammatesQueryRequest
	// rows : la population filtrée (allSquadRows) ; rowsTimeline : l'historique complet de la
	// composition (allSquadRowsForTimeline), non filtré par session ni période.
	rows, rowsTimeline []domain.SquadMatchRow
	teammates          []domain.TeammateRow
	allies             []domain.AllyParticipant
	mainTeamByMatch    map[string]map[string]struct{}
	sessionMatchIDs    map[string]bool
	selectedXUIDs      []string
	extraPool          map[string]struct{}
	issues             *dataIssues
}

// sectionsEscouade : les sections de la réponse calculées sur la population escouade. Toutes
// nulles quand la population est vide ; celles d'après une annulation restent nulles (GetPage
// rend alors l'erreur, pas une page partielle).
type sectionsEscouade struct {
	timeseries          []domain.SquadTimeseriesPoint
	mapBreakdown        []domain.MapBreakdownRow
	matchHistory        []domain.SquadMatchHistoryRow
	sessionTimeline     []domain.SquadSessionPoint
	mapHeatmap          *domain.SquadMapHeatmap
	impactMatrix        *domain.SquadImpactMatrix
	perMinuteStats      []domain.SquadPerMinuteEntry
	synergyRadar        []domain.SquadSynergyRadarSeries
	intensityProfile    *domain.SquadIntensityProfile
	performanceSeries   map[string][]domain.SquadPerformanceSeriesPoint
	weaponKills         *domain.SquadWeaponKills
	weaponAccuracy      *domain.SquadWeaponAccuracy
	fragClasses         map[string][]domain.FragClassEntry
	nativeKillMechanics *domain.SquadKillMechanics
	firstBlood          []domain.FirstBloodPlayerSeries
	assistPairs         *domain.SquadAssistPairs
	echange             *domain.SquadEchange
	rangeProfiles       *domain.MatchRangeBlock
	medalDigest         []domain.MedalDigestEntry
}

// sectionsDeLaPopulation calcule, sur la population escouade (composition exacte comprise),
// les tableaux puis les graphes de la page. Population vide : aucune section.
func (s *TeammatesService) sectionsDeLaPopulation(
	ctx context.Context, p populationEscouade,
) sectionsEscouade {
	var out sectionsEscouade
	if len(p.rows) == 0 {
		return out
	}
	s.tableauxDeLaPopulation(ctx, p, &out)
	s.graphesDeLaPopulation(ctx, p, &out)
	return out
}

// tableauxDeLaPopulation : séries temporelles, cartes (et leur historique « avec cette
// escouade »), historique des matchs et frise des sessions.
func (s *TeammatesService) tableauxDeLaPopulation(
	ctx context.Context, p populationEscouade, out *sectionsEscouade,
) {
	// Résout map/playlist/mode FR sur les rows (mode via la cascade
	// canonique asset_translations + mode_name_tr, cf. enrichSquadMatchAssets).
	siVivante(ctx, func() { enrichSquadMatchAssets(ctx, s.repo, p.rows) })
	out.timeseries = analysis.ComputeSquadTimeseries(p.rows, 20)
	out.mapBreakdown = computeMapBreakdown(p.rows)

	var squadStats map[string]domain.MapSquadStats
	siVivante(ctx, func() { squadStats = s.statsCartesAvecLEscouade(ctx, p) })
	out.mapBreakdown = enrichMapBreakdownWithSquadStats(out.mapBreakdown, squadStats)
	siVivante(ctx, func() {
		stop := timing.FromContext(ctx).Section("match_history")
		out.matchHistory = buildSquadMatchHistory(
			p.rows, squadStatsToWinTotal(squadStats), s.titleSlug,
			s.replayAvailability(ctx), s.roundsDecide)
		stop()
		stop = timing.FromContext(ctx).Section("session_timeline")
		out.sessionTimeline = buildSquadSessionTimeline(p.rowsTimeline)
		stop()
	})
}

// statsCartesAvecLEscouade : l'historique par carte « avec cette escouade ». squadXUIDs =
// coéquipiers sélectionnés (selectedXUIDs) ; excludeXUIDs = autres coéquipiers connus
// (extraPool) → anti-join qui écarte les matchs où l'un d'eux était sur l'équipe du main.
// L'anti-join n'est posé QUE sous l'option composition exacte : sinon la référence
// historique porterait sur une population plus étroite que les nombres affichés (parité
// stricte avec allSquadRows). Aucun filtre période/session : référence historique complète.
func (s *TeammatesService) statsCartesAvecLEscouade(
	ctx context.Context, p populationEscouade,
) map[string]domain.MapSquadStats {
	var excludeXUIDs []string
	if p.req.FilterExactComposition {
		excludeXUIDs = sortedXUIDSlice(p.extraPool)
	}
	stop := timing.FromContext(ctx).Section("map_stats")
	squadStats, err := s.repo.LoadMapStatsForSquad(ctx, p.playerXUID, p.selectedXUIDs, excludeXUIDs)
	stop()
	if err != nil {
		p.issues.add(ctx, domain.DataIssueMapStats, "", err)
	}
	return squadStats
}

// graphesDeLaPopulation : les graphes et blocs d'analyse, chacun sauté si la requête a été
// annulée pendant le précédent.
func (s *TeammatesService) graphesDeLaPopulation(
	ctx context.Context, p populationEscouade, out *sectionsEscouade,
) {
	gt, px, sel, tm := s.gamertag, p.playerXUID, p.req.SelectedGamertags, p.teammates
	siVivante(ctx, func() { out.mapHeatmap = s.buildSquadMapHeatmap(ctx, p.rows, sel, p.issues) })
	siVivante(ctx, func() { out.impactMatrix = s.buildSquadImpactMatrix(ctx, p.rows, px, sel, tm, p.allies) })
	siVivante(ctx, func() { out.perMinuteStats = s.buildSquadPerMinuteStats(ctx, p.rows, gt, sel, p.sessionMatchIDs) })
	siVivante(ctx, func() { out.synergyRadar = s.buildSquadSynergyRadar(ctx, p.rows, gt, sel) })
	siVivante(ctx, func() {
		out.intensityProfile = s.buildSquadIntensityProfile(ctx, p.rows, gt, px, sel, tm, p.mainTeamByMatch)
	})
	siVivante(ctx, func() { out.performanceSeries = s.buildSquadPerformanceSeries(ctx, p.rows, gt, px, sel, tm) })
	siVivante(ctx, func() {
		out.weaponKills, out.fragClasses = s.buildSquadWeaponKills(ctx, p.rows, gt, px, tm, out.performanceSeries)
	})
	siVivante(ctx, func() { out.weaponAccuracy = s.buildSquadWeaponAccuracy(ctx, p.rows, gt, px, tm) })
	siVivante(ctx, func() { out.nativeKillMechanics = s.buildSquadKillMechanics(ctx, p.rows, gt, px, tm) })
	siVivante(ctx, func() { out.firstBlood = s.buildSquadFirstBlood(ctx, p.rows, gt, px, tm) })
	siVivante(ctx, func() { out.assistPairs = s.buildSquadAssistPairs(ctx, p.rows, gt, px, tm) })
	// L'échange compare DEUX périmètres : les matchs filtrés (rows) et l'historique complet
	// de la composition (rowsTimeline, non filtré par session/période) — la baseline
	// « habituelle », dont le périmètre filtré est toujours un sous-ensemble. Même mécanique
	// que buildBriefingBaseline.
	siVivante(ctx, func() { out.echange = s.buildSquadEchange(ctx, p.rows, p.rowsTimeline, gt, px, tm) })
	// Roles de portee (D22-5) : MEME cadrage de perimetre et de roster que l'echange
	// ci-dessus, sur les seuls matchs filtres — la tendance se lit sur ce que la page
	// affiche, jamais sur un historique que le filtre a ecarte.
	siVivante(ctx, func() { out.rangeProfiles = s.buildSquadRange(ctx, p.rows, gt, px, tm) })
	siVivante(ctx, func() { out.medalDigest = s.buildMedalDigest(ctx, p.rows, gt, px, tm, p.req.Locale) })
}

// siVivante lance la section si la requête vit encore, la saute sinon (D2.7, lot perf L2) :
// une section n'est plus lancée pour un client parti — jusque-là une page abandonnée
// (onglet fermé, délai dépassé) se calculait jusqu'au bout, lectures comprises.
func siVivante(ctx context.Context, section func()) {
	if ctx.Err() == nil {
		section()
	}
}

// requeteAnnulee : ce que rend GetPage quand la requête a été annulée entre deux sections —
// l'erreur du contexte, enveloppée (errors.Is(err, context.Canceled) tient), jamais une page
// partielle.
func requeteAnnulee(err error) (domain.TeammatesPageResponse, error) {
	return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService: requete annulee: %w", err)
}
