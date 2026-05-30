// Package service — ExplorerService : matchs communs, recherche croisée.
//
// Port Go de apps/api/app/routers/explorer.py.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// outcomeWin est le code de victoire (Halo Infinite outcome = 2).
const outcomeWin = 2

// explorerTargetLiveBudget plafonne la durée du SEUL fetch live de l'encart
// "Profil joueur cible" : les stats carrière remote (servicerecord). Au-delà,
// le contexte est annulé : la carrière reste nil et la réponse part avec ce qui
// est disponible (identité locale + sample stats locales). Même philosophie que
// CareerLiveBudget côté home — la page ne doit jamais bloquer 10s+ sur un fetch
// Halo lent. L'identité (local-only) et le sample (calcul local) ne sont PAS
// bornés par ce budget.
const explorerTargetLiveBudget = 3 * time.Second

// ExplorerLocalIdentityResolver résout l'identité Spartan d'une cible STRICTEMENT
// depuis les données locales : si la cible est un joueur suivi (db_profiles), on
// lit son identité depuis SA propre player DB ; sinon nil. Aucun fetch live Halo
// (l'identité d'un adversaire n'est de toute façon jamais publiée).
//
// Interface locale (et non port.go) car le contrat est spécifique au flow
// Explorer. Implémentée en production par une closure câblée dans le registry
// (resolveByGT + HomeRepo.LoadSpartanIdentity).
type ExplorerLocalIdentityResolver interface {
	LocalSpartanIdentity(ctx context.Context, targetGamertag string) *domain.HomeSpartanIdentityRow
}

// LocalIdentityResolverFunc adapte une closure en ExplorerLocalIdentityResolver
// (le registry câble la résolution resolveByGT + HomeRepo.LoadSpartanIdentity).
type LocalIdentityResolverFunc func(ctx context.Context, targetGamertag string) *domain.HomeSpartanIdentityRow

// LocalSpartanIdentity implémente ExplorerLocalIdentityResolver.
func (f LocalIdentityResolverFunc) LocalSpartanIdentity(ctx context.Context, targetGamertag string) *domain.HomeSpartanIdentityRow {
	return f(ctx, targetGamertag)
}

// ExplorerTargetIdentityProvider fetch l'identité Spartan live d'un xuid
// arbitraire (career rank + emblem + backdrop). Implémenté par *CareerLiveService
// (GetSpartanIdentityFor). Sert de secours quand la cible n'est pas un joueur
// suivi localement.
type ExplorerTargetIdentityProvider interface {
	GetSpartanIdentityFor(ctx context.Context, xuid string) (*domain.HomeSpartanIdentityRow, error)
}

// ExplorerTargetCSRProvider fetch les CSR par playlist (saison) d'un xuid
// arbitraire. Implémenté par une closure câblée dans le registry (sync client
// GetPlayerCSRs + mapping → domain.CareerPlaylistCSR). Contrat : retourne
// (nil, nil) si les tokens auth sont absents/insuffisants (dégradation
// gracieuse, pas d'erreur).
type ExplorerTargetCSRProvider interface {
	SeasonCSRs(ctx context.Context, xuid, seasonID string) ([]domain.CareerPlaylistCSR, error)
}

// CSRProviderFunc adapte une closure en ExplorerTargetCSRProvider.
type CSRProviderFunc func(ctx context.Context, xuid, seasonID string) ([]domain.CareerPlaylistCSR, error)

// SeasonCSRs implémente ExplorerTargetCSRProvider.
func (f CSRProviderFunc) SeasonCSRs(ctx context.Context, xuid, seasonID string) ([]domain.CareerPlaylistCSR, error) {
	return f(ctx, xuid, seasonID)
}

// ExplorerService orchestre les requêtes de l'Explorer.
type ExplorerService struct {
	repo port.ExplorerRepository
	xuid string
	// dataAdapter (optionnel, Phase 2 plan finition multi-titres) :
	// quand fourni, le service mesure la capability match.history pour
	// loguer une éventuelle dégradation. La bascule fonctionnelle reste
	// future car canonical.MatchSummary ne couvre pas encore le filtrage
	// Explorer (joueur commun + plage temporelle).
	dataAdapter games.TitleDataAdapter

	// Dépendances optionnelles pour l'encart "Profil joueur cible" (cf.
	// TargetProfileDeps). Une dépendance nil laisse la sous-section à nil ;
	// sample_stats (local) reste toujours produit. Aucune privacy n'est fetchée.
	deps ExplorerTargetProfileDeps
}

// ExplorerTargetProfileDeps regroupe les dépendances de l'encart "Profil joueur
// cible". Tout est optionnel (nil → sous-section masquée).
type ExplorerTargetProfileDeps struct {
	// LocalIdentity : identité Spartan lue depuis la DB locale de la cible (si
	// joueur suivi). Source prioritaire.
	LocalIdentity ExplorerLocalIdentityResolver
	// LiveIdentity : identité Spartan live (career rank/emblem/backdrop) pour un
	// xuid arbitraire — secours quand la cible n'est pas locale.
	LiveIdentity ExplorerTargetIdentityProvider
	// RemoteStats : service record agrégé (stats + temps de jeu + médailles).
	RemoteStats port.ServiceRecordProvider
	// MedalDefs : métadonnées médailles (label/description) pour le top médailles.
	MedalDefs port.MedalDefinitionsRepository
	// CSR : classements CSR saison courante de la cible (live).
	CSR ExplorerTargetCSRProvider
	// CurrentSeasonID : saison CSR courante (pour le fetch CSR).
	CurrentSeasonID string
	// Seasons : calendrier des saisons (plages temporelles) pour le bucketing
	// "matchs par saison" — résolu côté registry depuis SeasonsCatalog.
	Seasons   []SeasonCatalogEntry
	TitleSlug string
}

// NewExplorerService crée un ExplorerService.
func NewExplorerService(repo port.ExplorerRepository, xuid string) *ExplorerService {
	return &ExplorerService{repo: repo, xuid: xuid}
}

// WithDataAdapter injecte un games.TitleDataAdapter optionnel (Phase 2 plan
// finition multi-titres). Permet de logger le statut des capabilities et
// d'amorcer la bascule vers la couche canonique.
func (s *ExplorerService) WithDataAdapter(a games.TitleDataAdapter) *ExplorerService {
	s.dataAdapter = a
	return s
}

// WithTargetProfileProviders injecte les dépendances de l'encart "Profil joueur
// cible". Retourne le service pour chainer.
func (s *ExplorerService) WithTargetProfileProviders(deps ExplorerTargetProfileDeps) *ExplorerService {
	s.deps = deps
	return s
}

// logCapabilityIfMissing log un warning si la capability est absente du
// DataAdapter injecté. No-op si pas de DataAdapter.
func (s *ExplorerService) logCapabilityIfMissing(ctx context.Context, cap games.CapabilityKey, caller string) {
	if s.dataAdapter == nil {
		return
	}
	if !s.dataAdapter.Capabilities().Has(cap) {
		slog.WarnContext(ctx, "capability_not_supported",
			"title_slug", s.dataAdapter.TitleSlug(),
			"capability", string(cap),
			"caller", caller,
		)
	}
}

// GetCommonMatches retourne l'historique paginé de matchs communs avec un autre
// joueur, enrichi des badges encounter (ally_plus, tough_enemy, ordinal).
// page est 1-indexé ; PageSizeCommonMatches = 20 éléments par page.
func (s *ExplorerService) GetCommonMatches(
	ctx context.Context,
	otherGamertag string,
	page int,
) (domain.ExplorerPlayerQueryResponse, error) {
	s.logCapabilityIfMissing(ctx, games.CapMatchHistory, "explorer_service.GetCommonMatches")

	otherXUID, err := s.repo.ResolveXUIDByGamertag(ctx, otherGamertag)
	if err != nil {
		return domain.ExplorerPlayerQueryResponse{},
			fmt.Errorf("ExplorerService: résolution gamertag %q: %w", otherGamertag, err)
	}

	rawMatches, err := s.repo.GetCommonMatches(ctx, s.xuid, otherXUID)
	if err != nil {
		return domain.ExplorerPlayerQueryResponse{},
			fmt.Errorf("ExplorerService: matchs communs: %w", err)
	}

	kv, err := s.repo.GetKillerVictimBetween(ctx, s.xuid, otherXUID)
	if err != nil {
		// Dégradation gracieuse : les badges tough_enemy ne seront pas calculés,
		// mais le reste de la réponse reste valide.
		slog.WarnContext(ctx, "explorer_kv_between_failed",
			"xuid1", s.xuid, "xuid2", otherXUID, "err", err)
		kv = domain.KillerVictimAggregate{}
	}

	totalCount := len(rawMatches)
	if page < 1 {
		page = 1
	}
	pageSize := domain.PageSizeCommonMatches
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > totalCount {
		end = totalCount
	}

	var pageMatches []domain.CommonMatchRaw
	if offset < totalCount {
		pageMatches = rawMatches[offset:end]
	}

	rows := convertCommonMatches(pageMatches)

	stats := buildEncounterStats(otherXUID, otherGamertag, rawMatches, kv)
	badges := narrative.ComputeEncounterBadges(stats, totalCount)
	wins, losses := countWinsLosses(rawMatches)
	encounterStats := convertEncounterStatsToExplorer(stats, totalCount)
	activityHeatmap := analysis.ComputeActivityHeatmapFromCommonMatches(rawMatches)

	// Encart "Profil joueur cible" : 4 sources fetch en parallèle (best-effort).
	targetProfile := s.buildTargetProfile(ctx, otherXUID, otherGamertag, rawMatches)

	slog.DebugContext(ctx, "explorer_common_matches",
		"xuid", s.xuid, "other_xuid", otherXUID,
		"total", totalCount, "page", page, "badges", len(badges),
		"heatmap_cells", len(activityHeatmap),
		"target_profile_built", targetProfile != nil,
		"auth_available", targetProfile != nil && targetProfile.AuthAvailable,
		"career_served", targetProfile != nil && targetProfile.CareerStats != nil)

	return domain.ExplorerPlayerQueryResponse{
		TargetGamertag:  otherGamertag,
		TargetXUID:      otherXUID,
		CommonMatches:   rows,
		Badges:          convertEncounterBadges(badges),
		EncounterStats:  encounterStats,
		Total:           len(rows),
		TotalCount:      totalCount,
		WinsTogether:    wins,
		LossesTogether:  losses,
		Page:            page,
		PageSize:        pageSize,
		ActivityHeatmap: activityHeatmap,
		TargetProfile:   targetProfile,
	}, nil
}

// buildTargetProfile orchestre les sources de l'encart "Profil joueur cible" :
//   - identité Spartan : DB locale (si joueur suivi) sinon live (xuid arbitraire,
//     career rank/emblem/backdrop) — bannière fallback sur le backdrop
//   - carrière + médailles : service record live (stats + temps de jeu + top
//     médailles lifetime) — un seul appel, borné + caché
//   - CSR saison : classements ranked live de la saison courante (tout xuid)
//   - sample stats : agrégat local sur les matchs communs
//
// Aucune privacy n'est fetchée. Tout est best-effort : une source qui échoue
// rend nil sans bloquer les autres. Les fetchs live sont bornés par
// explorerTargetLiveBudget ; les sources locales (sample) ne le sont pas.
func (s *ExplorerService) buildTargetProfile(
	ctx context.Context,
	targetXUID, targetGamertag string,
	rawMatches []domain.CommonMatchRaw,
) *domain.ExplorerTargetProfile {
	tokens := ctxkeys.HaloTokens(ctx)
	hasAuth := tokens != nil && tokens.SpartanToken != ""

	var (
		identity     *domain.HomeSpartanIdentityRow
		careerStats  *domain.NormalizedPlayerStats
		topMedals    []domain.MedalDigestItem
		seasonCSRs   []domain.CareerPlaylistCSR
		matchsPerSea []domain.SeasonMatchCount
		sampleStats  *domain.ExplorerTargetSampleStats
	)
	g, gctx := errgroup.WithContext(ctx)

	// Budget de latence sur les fetchs live (identité live / carrière / CSR).
	// Les calculs locaux (sample, matchs par saison) restent sur gctx, NON bornés.
	liveCtx, cancelLive := context.WithTimeout(gctx, explorerTargetLiveBudget)
	defer cancelLive()

	g.Go(func() error {
		identity = s.fetchTargetIdentity(liveCtx, targetXUID, targetGamertag, hasAuth)
		return nil
	})
	g.Go(func() error {
		careerStats, topMedals = s.fetchTargetServiceRecord(liveCtx, targetGamertag, hasAuth)
		return nil
	})
	g.Go(func() error { seasonCSRs = s.fetchTargetCSR(liveCtx, targetXUID, hasAuth); return nil })
	g.Go(func() error { matchsPerSea = s.computeMatchesPerSeason(gctx, targetXUID); return nil })
	g.Go(func() error { sampleStats = s.computeTargetSampleStats(gctx, targetXUID, rawMatches); return nil })

	_ = g.Wait()

	if hasAuth && careerStats == nil && liveCtx.Err() == context.DeadlineExceeded {
		slog.WarnContext(ctx, "explorer_target_live_budget_exceeded",
			"gamertag", targetGamertag, "budget", explorerTargetLiveBudget.String())
	}

	return &domain.ExplorerTargetProfile{
		Identity:         identity,
		CareerStats:      careerStats,
		TopMedals:        topMedals,
		SeasonCSRs:       seasonCSRs,
		MatchesPerSeason: matchsPerSea,
		SampleStats:      sampleStats,
		AuthAvailable:    hasAuth,
	}
}

// computeMatchesPerSeason agrège les matchs du target par saison (calcul local
// DuckDB, indépendant des tokens) : start_times depuis shared.match_participants
// rangés dans les plages de saison. nil si pas de saisons câblées / pas de matchs.
func (s *ExplorerService) computeMatchesPerSeason(ctx context.Context, targetXUID string) []domain.SeasonMatchCount {
	if len(s.deps.Seasons) == 0 {
		return nil
	}
	starts, err := s.repo.GetMatchStartTimesForXUID(ctx, targetXUID)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_matches_per_season_failed", "xuid", targetXUID, "err", err)
		return nil
	}
	return buildMatchesPerSeason(starts, s.deps.Seasons)
}

// fetchTargetIdentity résout l'identité Spartan du target : DB locale d'abord
// (si joueur suivi), sinon live pour un xuid arbitraire (career rank/emblem/
// backdrop). Applique le fallback bannière → backdrop. Retourne nil si rien.
func (s *ExplorerService) fetchTargetIdentity(ctx context.Context, targetXUID, targetGamertag string, hasAuth bool) *domain.HomeSpartanIdentityRow {
	if s.deps.LocalIdentity != nil {
		if id := s.deps.LocalIdentity.LocalSpartanIdentity(ctx, targetGamertag); id != nil {
			applyBannerFallback(id)
			return id
		}
	}
	if hasAuth && s.deps.LiveIdentity != nil && targetXUID != "" {
		id, err := s.deps.LiveIdentity.GetSpartanIdentityFor(ctx, targetXUID)
		if err != nil {
			slog.WarnContext(ctx, "explorer_target_identity_live_failed", "xuid", targetXUID, "err", err)
			return nil
		}
		if id != nil {
			applyBannerFallback(id)
			return id
		}
	}
	slog.DebugContext(ctx, "explorer_target_identity_unavailable", "gamertag", targetGamertag)
	return nil
}

// applyBannerFallback : si la bannière (nameplate background) est absente — cas
// fréquent pour une cible non locale, l'API officielle n'expose pas de
// nameplate dédié — on retombe sur le backdrop du joueur (présent dans
// l'appearance). Mutation en place, best-effort.
func applyBannerFallback(id *domain.HomeSpartanIdentityRow) {
	if id == nil {
		return
	}
	if (id.BannerImageURL == nil || *id.BannerImageURL == "") &&
		id.BackdropImageURL != nil && *id.BackdropImageURL != "" {
		id.BannerImageURL = id.BackdropImageURL
	}
}

// fetchTargetServiceRecord fetch le service record live (stats + temps de jeu +
// médailles lifetime) et en dérive les stats carrière + le top médailles. Skip
// sans auth ; (nil, nil) en cas d'erreur.
func (s *ExplorerService) fetchTargetServiceRecord(ctx context.Context, targetGamertag string, hasAuth bool) (*domain.NormalizedPlayerStats, []domain.MedalDigestItem) {
	if s.deps.RemoteStats == nil {
		return nil, nil
	}
	if !hasAuth {
		slog.DebugContext(ctx, "explorer_target_career_skipped",
			"gamertag", targetGamertag, "reason", "no_auth_tokens")
		return nil, nil
	}
	rec, err := s.deps.RemoteStats.FetchServiceRecord(ctx, targetGamertag, s.deps.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_career_failed", "gamertag", targetGamertag, "err", err)
		return nil, nil
	}
	if rec == nil {
		return nil, nil
	}
	stats := rec.Stats
	medals := buildTargetTopMedals(ctx, s.deps.MedalDefs, rec.Medals, s.deps.TitleSlug, ctxkeys.Locale(ctx))
	return &stats, medals
}

// fetchTargetCSR fetch les classements CSR de la saison courante de la cible
// (endpoint skill public — tout xuid). Skip sans auth / sans provider / sans
// saison. nil en cas d'erreur (logguée).
func (s *ExplorerService) fetchTargetCSR(ctx context.Context, targetXUID string, hasAuth bool) []domain.CareerPlaylistCSR {
	if s.deps.CSR == nil || !hasAuth || targetXUID == "" || s.deps.CurrentSeasonID == "" {
		return nil
	}
	csrs, err := s.deps.CSR.SeasonCSRs(ctx, targetXUID, s.deps.CurrentSeasonID)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_csr_failed", "xuid", targetXUID, "err", err)
		return nil
	}
	return csrs
}

// computeTargetSampleStats agrège les stats du target sur les matchs communs
// (calcul local DuckDB, indépendant des tokens). nil si aucun match commun.
func (s *ExplorerService) computeTargetSampleStats(ctx context.Context, targetXUID string, rawMatches []domain.CommonMatchRaw) *domain.ExplorerTargetSampleStats {
	matchIDs := extractCommonMatchIDs(rawMatches)
	if len(matchIDs) == 0 {
		return nil
	}
	agg, err := s.repo.GetParticipantStatsForMatches(ctx, targetXUID, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_sample_stats_failed", "xuid", targetXUID, "err", err)
		return nil
	}
	medals, mErr := s.repo.GetMedalCountsForMatches(ctx, targetXUID, matchIDs)
	if mErr != nil {
		slog.WarnContext(ctx, "explorer_target_medals_failed", "xuid", targetXUID, "err", mErr)
		// medals est nil → BuildSampleStats l'ignorera, ce n'est pas bloquant.
	}
	return analysis.BuildSampleStats(agg, medals, len(matchIDs))
}

// extractCommonMatchIDs extrait la liste des match_id d'un slice de common matches.
func extractCommonMatchIDs(rawMatches []domain.CommonMatchRaw) []string {
	if len(rawMatches) == 0 {
		return nil
	}
	ids := make([]string, 0, len(rawMatches))
	for i := range rawMatches {
		ids = append(ids, rawMatches[i].MatchID)
	}
	return ids
}

// convertEncounterStatsToExplorer projette narrative.EncounterStats → domain.ExplorerEncounterStats.
// Les compteurs ally/enemy et K/D croisés sont retournés en pointeurs pour
// distinguer "absent" de "zéro" (cohérent avec MatchEncounterRow JSON).
func convertEncounterStatsToExplorer(s narrative.EncounterStats, totalCount int) *domain.ExplorerEncounterStats {
	if totalCount == 0 {
		return nil
	}
	ally, enemy := s.AllyCount, s.EnemyCount
	kills, deaths := s.KillsDealt, s.DeathsSuffered
	out := &domain.ExplorerEncounterStats{
		CountTogether:  totalCount,
		AllyCount:      &ally,
		EnemyCount:     &enemy,
		KillsDealt:     &kills,
		DeathsSuffered: &deaths,
		WinrateAsAlly:  s.WinrateAsAlly,
		WinrateVsEnemy: s.WinrateVsEnemy,
		LastSeenAt:     s.LastSeen,
	}
	return out
}

// convertCommonMatches convertit les lignes brutes en CommonMatchRow avec
// were_teammates et outcome_label résolus.
func convertCommonMatches(raw []domain.CommonMatchRaw) []domain.CommonMatchRow {
	if len(raw) == 0 {
		return []domain.CommonMatchRow{}
	}
	result := make([]domain.CommonMatchRow, 0, len(raw))
	for i := range raw {
		r := &raw[i]
		wereTeammates := r.Player1TeamID != nil &&
			r.Player2TeamID != nil &&
			*r.Player1TeamID == *r.Player2TeamID
		result = append(result, domain.CommonMatchRow{
			MatchID:       r.MatchID,
			StartTime:     r.StartTime,
			MapUI:         r.MapUI,
			ModeUI:        r.ModeUI,
			WereTeammates: wereTeammates,
			PlayerOutcome: r.Player1Outcome,
			OutcomeLabel:  outcomeLabel(r.Player1Outcome),
			Kills:         r.Player1Kills,
			Deaths:        r.Player1Deaths,
			KDA:           r.Player1KDA,
		})
	}
	return result
}

// buildEncounterStats construit un narrative.EncounterStats depuis les données
// brutes de matchs communs et les kills croisés.
func buildEncounterStats(xuid, gamertag string, raw []domain.CommonMatchRaw, kv domain.KillerVictimAggregate) narrative.EncounterStats {
	stats := narrative.EncounterStats{
		XUID:            xuid,
		Gamertag:        gamertag,
		TotalEncounters: len(raw),
		KillsDealt:      kv.KillsDealt,
		DeathsSuffered:  kv.DeathsSuffered,
	}

	var allyWins, allyTotal, enemyWins, enemyTotal int
	for i := range raw {
		r := &raw[i]
		wereTeammates := r.Player1TeamID != nil &&
			r.Player2TeamID != nil &&
			*r.Player1TeamID == *r.Player2TeamID
		if wereTeammates {
			allyTotal++
			if r.Player1Outcome == outcomeWin {
				allyWins++
			}
		} else {
			enemyTotal++
			if r.Player1Outcome == outcomeWin {
				enemyWins++
			}
		}
		if stats.LastSeen == nil || r.StartTime.After(*stats.LastSeen) {
			t := r.StartTime
			stats.LastSeen = &t
		}
	}

	stats.AllyCount = allyTotal
	stats.EnemyCount = enemyTotal
	if allyTotal > 0 {
		wr := float64(allyWins) / float64(allyTotal)
		stats.WinrateAsAlly = &wr
	}
	if enemyTotal > 0 {
		wr := float64(enemyWins) / float64(enemyTotal)
		stats.WinrateVsEnemy = &wr
	}
	return stats
}

// countWinsLosses compte les victoires et défaites sur l'ensemble des matchs.
func countWinsLosses(raw []domain.CommonMatchRaw) (wins, losses int) {
	for i := range raw {
		switch raw[i].Player1Outcome {
		case outcomeWin:
			wins++
		default:
			losses++
		}
	}
	return
}

// convertEncounterBadges convertit les badges narrative en types domain.
func convertEncounterBadges(badges []narrative.EncounterBadge) []domain.MatchEncounterBadge {
	if len(badges) == 0 {
		return nil
	}
	result := make([]domain.MatchEncounterBadge, 0, len(badges))
	for _, b := range badges {
		result = append(result, domain.MatchEncounterBadge{
			Kind:       string(b.Kind),
			LabelKey:   b.LabelKey,
			ColorToken: b.ColorToken,
			Detail:     b.Detail,
		})
	}
	return result
}
