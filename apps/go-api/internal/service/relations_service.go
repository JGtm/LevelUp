// Package service — relations_service.go : orchestration du hub Communauté >
// Relations. Combine RelationsRepository (agrégats SQL) + analysis/relations
// (badges, catégorie, aperçu) → domain.RelationsPageResponse. Aucun SQL.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/relations"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// coreRecentFormLimit : nombre de matchs de la frise « Derniers ensemble » de la
// carte Noyau dur (derniers matchs joués à tes côtés avec un fidèle).
const coreRecentFormLimit = 25

// RelationsService orchestre le hub Relations.
type RelationsService struct {
	repo      port.RelationsRepository
	filters   port.FiltersService        // optionnel : résout le scope match_id (Phase 2). nil → pas de segmentation.
	crossGame port.CrossGameCooccurrence // optionnel (Phase 3b) : badge cross-jeu. nil → inerte.
	now       func() time.Time           // injectable pour les tests (badges temporels)
}

// NewRelationsService crée un RelationsService.
func NewRelationsService(repo port.RelationsRepository) *RelationsService {
	return &RelationsService{repo: repo, now: time.Now}
}

// WithFilters injecte le FiltersService qui résout le sous-ensemble de match_id
// d'une sélection (expérience/classé, saison/période, playlist/mode, vue
// solo/escouade), via le MÊME pipeline cascade que /filters/resolve. Sans cette
// injection, la segmentation est inerte (tous les matchs).
func (s *RelationsService) WithFilters(f port.FiltersService) *RelationsService {
	s.filters = f
	return s
}

// withNow injecte une horloge déterministe (tests des badges temporels).
func (s *RelationsService) withNow(now func() time.Time) *RelationsService {
	s.now = now
	return s
}

// GetRelationsPage construit la page complète : aperçu (compteurs + binôme +
// bête noire) et liste des relations enrichies (badges + catégorie).
//
// Phase 2 : `input` porte la segmentation serveur. Si la sélection est active
// (filtre non trivial) ET qu'un FiltersService est injecté, le scope de match_id
// est résolu en amont et passé au repo qui restreint l'agrégation. Un input
// trivial (tout) court-circuite la résolution (scope nil) → coût Phase 1.
func (s *RelationsService) GetRelationsPage(ctx context.Context, input domain.FilterContextInput) (domain.RelationsPageResponse, error) {
	scope, err := s.resolveScope(ctx, input)
	if err != nil {
		return domain.RelationsPageResponse{}, fmt.Errorf("RelationsService.GetRelationsPage: scope: %w", err)
	}

	rawRows, err := s.repo.GetRelations(ctx, scope)
	if err != nil {
		return domain.RelationsPageResponse{}, fmt.Errorf("RelationsService.GetRelationsPage: %w", err)
	}

	stats := make([]relations.RelationStats, 0, len(rawRows))
	for _, r := range rawRows {
		stats = append(stats, relationStatsFromRaw(r))
	}

	now := s.now()
	insights := make([]domain.RelationInsight, 0, len(rawRows))
	for i := range rawRows {
		insights = append(insights, buildRelationInsight(rawRows[i], stats[i], now))
	}

	// Phase 3b (ADDITIF, best-effort) : badge cross-jeu sur les relations aussi
	// croisées sur un autre titre. No-op si crossGame non injecté ; toute erreur
	// cross-titre est avalée en interne → aucune régression de /relations.
	s.appendCrossGameBadges(ctx, insights)

	overview := buildOverview(stats)
	s.appendCoreEngagement(ctx, &overview, stats, scope)
	s.appendNemesisCSR(ctx, &overview, stats)

	return domain.RelationsPageResponse{
		Overview:  overview,
		Relations: insights,
	}, nil
}

// appendNemesisCSR enrichit l'aperçu avec le CSR courant de la bête noire (top
// nemesis), lu best-effort depuis match_csrs_latest via le repo. ADDITIF /
// best-effort strict (lot relations-G) : le CSR n'existe que pour le classé — pour
// une bête noire social/non collectée, le repo renvoie nil et rien n'est affiché
// (dégradation gracieuse). Toute erreur de lecture est loggée et avalée : un échec
// de cet enrichissement ne doit JAMAIS faire échouer /relations.
func (s *RelationsService) appendNemesisCSR(
	ctx context.Context, overview *domain.RelationsOverview, stats []relations.RelationStats,
) {
	if overview.TopNemesis == nil {
		return
	}
	// SelectTopNemesis est pur/déterministe : la bête noire est la même que dans
	// buildOverview (le XUID, dropé par topRefToRef, est requis pour la lecture CSR).
	nemesis := relations.SelectTopNemesis(stats)
	if nemesis == nil || nemesis.XUID == "" {
		return
	}
	csr, err := s.repo.GetLatestCSR(ctx, nemesis.XUID)
	if err != nil {
		slog.WarnContext(ctx, "RelationsService: enrich nemesis CSR failed", "err", err)
		return
	}
	overview.TopNemesis.CSR = csr
}

// appendCoreEngagement enrichit l'aperçu avec le WR perso de référence (lift de
// la carte Noyau dur) et la frise « forme récente » jouée avec un membre du
// noyau. ADDITIF / best-effort : toute erreur est loggée et avalée — un échec de
// cet enrichissement ne doit jamais faire échouer /relations.
func (s *RelationsService) appendCoreEngagement(
	ctx context.Context, overview *domain.RelationsOverview, stats []relations.RelationStats, scope []string,
) {
	coreXUIDs := make([]string, 0)
	for i := range stats {
		if relations.IsCore(stats[i]) {
			coreXUIDs = append(coreXUIDs, stats[i].XUID)
		}
	}
	eng, err := s.repo.GetCoreEngagement(ctx, coreXUIDs, scope, coreRecentFormLimit)
	if err != nil {
		slog.WarnContext(ctx, "RelationsService: enrich core engagement failed", "err", err)
		return
	}
	overview.PlayerWinRate = eng.PlayerWinRate
	overview.CoreRecentForm = eng.RecentForm

	// Forme récente du binôme (top_ally) pour sa sparkline — best-effort, séparé
	// (le binôme n'est pas forcément dans le noyau).
	if ally := relations.SelectTopAlly(stats); ally != nil {
		form, formErr := s.repo.GetRelationRecentForm(ctx, ally.XUID, scope, coreRecentFormLimit)
		if formErr != nil {
			slog.WarnContext(ctx, "RelationsService: enrich top-ally form failed", "err", formErr)
		} else {
			overview.TopAllyRecentForm = form
		}
	}

	// Miroir ennemi : forme récente CONTRE la bête noire (top_nemesis) pour la
	// sparkline « Derniers matchs » de sa carte — best-effort, symétrique du binôme.
	if nem := relations.SelectTopNemesis(stats); nem != nil {
		form, formErr := s.repo.GetRelationEnemyRecentForm(ctx, nem.XUID, scope, coreRecentFormLimit)
		if formErr != nil {
			slog.WarnContext(ctx, "RelationsService: enrich top-nemesis form failed", "err", formErr)
		} else {
			overview.TopNemesisRecentForm = form
		}
	}
}

// resolveScope retourne le sous-ensemble de match_id correspondant à `input` :
//   - nil  → pas de segmentation (sélection triviale OU FiltersService absent)
//     → le repo agrège sur tous les matchs (comportement Phase 1)
//   - []   → sélection active mais aucun match en périmètre → le repo court-circuite
//   - [..] → sélection active → le repo restreint l'agrégation à ce set
func (s *RelationsService) resolveScope(ctx context.Context, input domain.FilterContextInput) ([]string, error) {
	if s.filters == nil || !filterContextIsActive(input) {
		return nil, nil
	}
	ids, err := s.filters.ResolveMatchIDs(ctx, input)
	if err != nil {
		return nil, err
	}
	// ResolveMatchIDs renvoie nil pour une population vide ; on normalise vers
	// un slice non-nil vide pour que le repo distingue "tout" (nil) de
	// "rien en périmètre" (vide) sans ambiguïté.
	if ids == nil {
		return []string{}, nil
	}
	return ids, nil
}

// filterContextIsActive indique si `input` restreint réellement la population :
// une période bornée, une session pickée, une cascade non vide, ou une vue
// solo/escouade. Un input trivial (tout) court-circuite la résolution coûteuse.
func filterContextIsActive(in domain.FilterContextInput) bool {
	if in.Period.StartDate != nil || in.Period.EndDate != nil {
		return true
	}
	if hasPickedSessions(in.Sessions) {
		return true
	}
	c := in.Cascade
	if len(c.ExperienceTypes) > 0 || len(c.Playlists) > 0 || len(c.Modes) > 0 || len(c.Maps) > 0 {
		return true
	}
	switch in.MatchContext {
	case domain.MatchContextSolo, domain.MatchContextSquad:
		return true
	}
	return false
}

// relationStatsFromRaw projette la ligne brute repo en stats d'analyse (pur),
// calculant les taux de victoire et le duel ratio.
func relationStatsFromRaw(r domain.RelationRawRow) relations.RelationStats {
	st := relations.RelationStats{
		XUID:            r.XUID,
		Gamertag:        r.Gamertag,
		TotalMatches:    r.TotalMatches,
		TeammateMatches: r.TeammateCount,
		EnemyMatches:    r.EnemyCount,
		TeammateWins:    r.TeammateWins,
		EnemyWins:       r.EnemyWins,
		KillsDealt:      r.KillsDealt,
		DeathsSuffered:  r.DeathsSuffered,
	}
	st.TeammateWinRate = relationWinRate(r.TeammateWins, r.TeammateLosses)
	st.EnemyWinRate = relationWinRate(r.EnemyWins, r.EnemyLosses)
	st.DuelRatio = duelRatio(r.KillsDealt, r.DeathsSuffered)
	if !r.FirstSeen.IsZero() {
		t := r.FirstSeen
		st.FirstSeen = &t
	}
	if !r.LastSeen.IsZero() {
		t := r.LastSeen
		st.LastSeen = &t
	}
	return st
}

// buildRelationInsight assemble le DTO d'une relation (stats + badges +
// catégorie) à partir de la ligne brute et des stats d'analyse.
func buildRelationInsight(r domain.RelationRawRow, st relations.RelationStats, now time.Time) domain.RelationInsight {
	insight := domain.RelationInsight{
		XUID:            r.XUID,
		Gamertag:        r.Gamertag,
		TotalMatches:    r.TotalMatches,
		TeammateMatches: r.TeammateCount,
		TeammateWins:    r.TeammateWins,
		TeammateWinRate: st.TeammateWinRate,
		EnemyMatches:    r.EnemyCount,
		EnemyWins:       r.EnemyWins,
		EnemyWinRate:    st.EnemyWinRate,
		AvgKDAWith:      r.AvgKDAWith,
		AvgKDAAgainst:   r.AvgKDAAgainst,
		KillsDealt:      r.KillsDealt,
		DeathsSuffered:  r.DeathsSuffered,
		DuelRatio:       st.DuelRatio,
		FirstSeenAt:     formatRFC3339(r.FirstSeen),
		LastSeenAt:      formatRFC3339(r.LastSeen),
		Category:        relations.Categorize(st),
		IsCore:          relations.IsCore(st),
		Badges:          projectBadges(relations.ComputeBadges(st, now)),
	}
	return insight
}

// buildOverview agrège les compteurs et sélectionne binôme + bête noire.
func buildOverview(stats []relations.RelationStats) domain.RelationsOverview {
	c := relations.ComputeCounts(stats)
	return domain.RelationsOverview{
		DistinctPlayers: c.DistinctPlayers,
		AlliesCount:     c.AlliesCount,
		RivalsCount:     c.RivalsCount,
		CoreCount:       c.CoreCount,
		TopAlly:         topRefToRef(relations.SelectTopAlly(stats)),
		TopNemesis:      topRefToRef(relations.SelectTopNemesis(stats)),
	}
}

// projectBadges projette []relations.Badge → []domain.RelationBadge.
func projectBadges(badges []relations.Badge) []domain.RelationBadge {
	out := make([]domain.RelationBadge, 0, len(badges))
	for _, b := range badges {
		out = append(out, domain.RelationBadge{
			LabelKey:   b.LabelKey,
			ColorToken: b.ColorToken,
			Style:      b.Style,
			Detail:     b.Detail,
		})
	}
	return out
}

// topRefToRef projette *relations.TopRef → *domain.RelationRef (nil-safe).
func topRefToRef(t *relations.TopRef) *domain.RelationRef {
	if t == nil {
		return nil
	}
	return &domain.RelationRef{Gamertag: t.Gamertag, WinRate: t.WinRate, Matches: t.Matches}
}

// relationWinRate : nil si W+L == 0, sinon ratio 0..1.
func relationWinRate(wins, losses int) *float64 {
	total := wins + losses
	if total == 0 {
		return nil
	}
	rate := analysis.WinRate(wins, total)
	return &rate
}

// duelRatio : kills/deaths. nil si deaths==0 ET kills==0 (aucune donnée de
// duel). Si deaths==0 et kills>0 → ratio = float64(kills) (domination totale).
func duelRatio(kills, deaths int) *float64 {
	if deaths == 0 && kills == 0 {
		return nil
	}
	var ratio float64
	if deaths > 0 {
		ratio = float64(kills) / float64(deaths)
	} else {
		ratio = float64(kills)
	}
	return &ratio
}

// formatRFC3339 : nil pour un time zéro, sinon RFC3339 UTC.
func formatRFC3339(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
