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

	// Dépendances optionnelles pour l'encart "Profil joueur cible" :
	//   - localIdentity : identité Spartan local-only (nil si cible non suivie)
	//   - remoteStats   : stats carrière agrégées (servicerecord live, caché)
	// Quand une dépendance est nil, la sous-section correspondante reste nil.
	// Le sample_stats (calcul local) reste toujours produit. Aucune privacy
	// n'est fetchée pour l'Explorer (bruit sans valeur — décision produit).
	localIdentity ExplorerLocalIdentityResolver
	remoteStats   port.PlayerStatsProvider
	titleSlug     string
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
// cible" : le résolveur d'identité locale + le provider de stats carrière remote.
// Retourne le service pour chainer. Les deux sont optionnels — un nil laisse la
// sous-section correspondante à nil.
func (s *ExplorerService) WithTargetProfileProviders(
	localIdentity ExplorerLocalIdentityResolver,
	remote port.PlayerStatsProvider,
	titleSlug string,
) *ExplorerService {
	s.localIdentity = localIdentity
	s.remoteStats = remote
	s.titleSlug = titleSlug
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

// buildTargetProfile orchestre les 3 sources de l'encart "Profil joueur cible" :
//   - identité Spartan : LOCAL-only (DB du joueur cible s'il est suivi, nil sinon)
//   - carrière agrégée : fetch live servicerecord (seul appel réseau, borné +
//     caché) — fonctionne pour tout joueur, gated sur la présence de tokens
//   - sample stats     : agrégat local sur les matchs communs
//
// Aucune privacy n'est fetchée (bruit sans valeur). Tout est best-effort : une
// source qui échoue/rend nil ne bloque pas les autres (le front masque les
// sections nil). AuthAvailable reflète la présence de tokens (la carrière en
// dépend) et pilote le hint front.
func (s *ExplorerService) buildTargetProfile(
	ctx context.Context,
	targetXUID, targetGamertag string,
	rawMatches []domain.CommonMatchRaw,
) *domain.ExplorerTargetProfile {
	tokens := ctxkeys.HaloTokens(ctx)
	hasAuth := tokens != nil && tokens.SpartanToken != ""

	var (
		identity    *domain.HomeSpartanIdentityRow
		careerStats *domain.NormalizedPlayerStats
		sampleStats *domain.ExplorerTargetSampleStats
	)
	g, gctx := errgroup.WithContext(ctx)

	// Budget de latence sur le SEUL fetch live (carrière). Identité et sample
	// sont 100% locaux → sur gctx, NON bornés par ce budget.
	liveCtx, cancelLive := context.WithTimeout(gctx, explorerTargetLiveBudget)
	defer cancelLive()

	g.Go(func() error { identity = s.fetchTargetIdentity(gctx, targetGamertag); return nil })
	g.Go(func() error { careerStats = s.fetchTargetCareer(liveCtx, targetGamertag, hasAuth); return nil })
	g.Go(func() error { sampleStats = s.computeTargetSampleStats(gctx, targetXUID, rawMatches); return nil })

	_ = g.Wait()

	// Observabilité : si le budget carrière a expiré, on a servi une réponse
	// dégradée (carrière nil). On le trace pour distinguer "Halo lent →
	// dégradation gracieuse" d'une vraie erreur.
	if hasAuth && careerStats == nil && liveCtx.Err() == context.DeadlineExceeded {
		slog.WarnContext(ctx, "explorer_target_live_budget_exceeded",
			"gamertag", targetGamertag,
			"budget", explorerTargetLiveBudget.String())
	}

	return &domain.ExplorerTargetProfile{
		Identity:      identity,
		CareerStats:   careerStats,
		SampleStats:   sampleStats,
		AuthAvailable: hasAuth,
	}
}

// fetchTargetIdentity résout l'identité Spartan du target STRICTEMENT en local :
// si la cible est un joueur suivi, son identité (rang/emblem/peaks) est lue
// depuis SA propre DB ; sinon nil (un adversaire n'a pas d'identité publiée).
// Aucun fetch live.
func (s *ExplorerService) fetchTargetIdentity(ctx context.Context, targetGamertag string) *domain.HomeSpartanIdentityRow {
	if s.localIdentity == nil {
		return nil
	}
	id := s.localIdentity.LocalSpartanIdentity(ctx, targetGamertag)
	if id == nil {
		// Cas normal pour un adversaire (non suivi localement) : pas d'identité.
		// Tracé en debug pour distinguer "non suivi" d'une vraie absence de data.
		slog.DebugContext(ctx, "explorer_target_identity_not_local", "gamertag", targetGamertag)
	}
	return id
}

// fetchTargetCareer fetch les stats carrière remote du target (servicerecord via
// le provider décoré d'un cache). Skip silencieusement sans auth ; nil en cas
// d'erreur.
func (s *ExplorerService) fetchTargetCareer(ctx context.Context, targetGamertag string, hasAuth bool) *domain.NormalizedPlayerStats {
	if s.remoteStats == nil {
		return nil
	}
	if !hasAuth {
		slog.DebugContext(ctx, "explorer_target_career_skipped",
			"gamertag", targetGamertag, "reason", "no_auth_tokens")
		return nil
	}
	stats, err := s.remoteStats.FetchRemoteStats(ctx, targetGamertag, s.titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_career_failed", "gamertag", targetGamertag, "err", err)
		return nil
	}
	return stats
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
