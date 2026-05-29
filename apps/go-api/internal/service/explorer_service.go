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

// explorerTargetLiveBudget plafonne la durée des fetchs live de l'encart
// "Profil joueur cible" (identité live, stats carrière remote, privacy). Au-delà,
// le contexte est annulé : les sous-sections non prêtes restent nil et la réponse
// part avec ce qui est disponible (identité DB + sample stats locales). Même
// philosophie que CareerLiveBudget côté home — la page ne doit jamais bloquer
// 10s+ sur un fetch Halo lent. Un peu plus généreux car la carrière est le contenu
// principal de cet encart. Le calcul local (sample stats) n'est PAS borné par ce
// budget. Cf. plan explorer-target-profile-auth (volet C).
const explorerTargetLiveBudget = 3 * time.Second

// ExplorerTargetIdentityProvider abstrait le fetch de l'identité Spartan d'un
// xuid arbitraire (live cache+merge ou DB-only). Implémenté en production par
// *CareerLiveService — voir career_live_service.go.
//
// Interface locale (et non port.go) car le contrat est spécifique au flow
// Explorer (xuid tiers, no-tokens fallback) et n'a pas vocation à être
// implémenté ailleurs que par CareerLiveService.
type ExplorerTargetIdentityProvider interface {
	GetSpartanIdentityFor(ctx context.Context, xuid string) (*domain.HomeSpartanIdentityRow, error)
	GetSpartanIdentityFromDBOnly(ctx context.Context, xuid string) *domain.HomeSpartanIdentityRow
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

	// Providers optionnels pour l'encart "Profil joueur cible". Quand l'un
	// d'eux est nil, la goroutine correspondante est skip et le champ
	// correspondant de ExplorerTargetProfile reste nil. Le sample_stats
	// (calcul local) reste toujours produit même sans providers.
	identityProvider ExplorerTargetIdentityProvider
	remoteStats      port.PlayerStatsProvider
	privacyProvider  port.PrivacyProvider
	titleSlug        string
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

// WithTargetProfileProviders injecte les providers nécessaires à l'encart
// "Profil joueur cible" (identité Spartan live + stats carrière remote +
// privacy). Retourne le service pour chainer. Tous les providers sont
// optionnels — un nil skip la sous-section correspondante.
func (s *ExplorerService) WithTargetProfileProviders(
	identity ExplorerTargetIdentityProvider,
	remote port.PlayerStatsProvider,
	privacy port.PrivacyProvider,
	titleSlug string,
) *ExplorerService {
	s.identityProvider = identity
	s.remoteStats = remote
	s.privacyProvider = privacy
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

// buildTargetProfile orchestre les 4 sources de l'encart "Profil joueur cible".
//
// Détection précoce du cas no-tokens (user connecté sans OAuth Halo) : on
// court-circuite les 3 goroutines live (identity remote, career_stats, privacy)
// et on bascule l'identité sur DB locale. Le sample_stats reste calculé depuis
// DuckDB indépendamment des tokens.
//
// Tous les fetchs sont best-effort : une goroutine qui retourne nil ne
// bloque pas les autres. Le frontend rend les sections disponibles, masque
// celles à nil, et affiche un hint "Connexion Halo requise" si
// AuthAvailable=false.
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
		privacy     *domain.MatchPrivacyWarning
		sampleStats *domain.ExplorerTargetSampleStats
	)
	g, gctx := errgroup.WithContext(ctx)

	// Budget de latence sur les fetchs live (identité/carrière/privacy). Le
	// calcul local (sample stats) reste sur gctx, NON borné par ce budget.
	liveCtx, cancelLive := context.WithTimeout(gctx, explorerTargetLiveBudget)
	defer cancelLive()

	// 4 sources best-effort en parallèle : une source qui échoue/skip rend nil
	// sans bloquer les autres (le front masque les sections nil).
	g.Go(func() error { identity = s.fetchTargetIdentity(liveCtx, targetXUID, hasAuth); return nil })
	g.Go(func() error { careerStats = s.fetchTargetCareer(liveCtx, targetGamertag, hasAuth); return nil })
	g.Go(func() error { privacy = s.fetchTargetPrivacy(liveCtx, targetXUID, hasAuth); return nil })
	g.Go(func() error { sampleStats = s.computeTargetSampleStats(gctx, targetXUID, rawMatches); return nil })

	_ = g.Wait()

	// Observabilité : si le budget live a expiré, on a servi une réponse
	// dégradée (sous-sections live potentiellement nil). On le trace pour
	// distinguer "Halo lent → dégradation gracieuse" d'une vraie erreur.
	if hasAuth && liveCtx.Err() == context.DeadlineExceeded {
		slog.WarnContext(ctx, "explorer_target_live_budget_exceeded",
			"xuid", targetXUID,
			"budget", explorerTargetLiveBudget.String(),
			"career_served", careerStats != nil,
			"identity_served", identity != nil)
	}

	// On émet toujours un TargetProfile (même si toutes les sous-sections
	// sont nil) parce que AuthAvailable porte une info utile au front.
	return &domain.ExplorerTargetProfile{
		Identity:       identity,
		CareerStats:    careerStats,
		SampleStats:    sampleStats,
		PrivacyWarning: privacy,
		AuthAvailable:  hasAuth,
	}
}

// fetchTargetIdentity charge l'identité Spartan du target : live si auth dispo,
// DB locale sinon. Retourne nil si pas de provider ou en cas d'erreur (logguée).
func (s *ExplorerService) fetchTargetIdentity(ctx context.Context, targetXUID string, hasAuth bool) *domain.HomeSpartanIdentityRow {
	if s.identityProvider == nil {
		return nil
	}
	if !hasAuth {
		return s.identityProvider.GetSpartanIdentityFromDBOnly(ctx, targetXUID)
	}
	id, err := s.identityProvider.GetSpartanIdentityFor(ctx, targetXUID)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_identity_failed", "xuid", targetXUID, "err", err)
		return nil
	}
	return id
}

// fetchTargetCareer fetch les stats carrière remote du target (via le provider
// décoré d'un cache). Skip silencieusement sans auth ; nil en cas d'erreur.
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

// fetchTargetPrivacy interroge la privacy du target. Skip sans auth ; nil en
// cas d'erreur (logguée).
func (s *ExplorerService) fetchTargetPrivacy(ctx context.Context, targetXUID string, hasAuth bool) *domain.MatchPrivacyWarning {
	if s.privacyProvider == nil || !hasAuth {
		return nil
	}
	info, err := s.privacyProvider.GetMatchPrivacy(ctx, targetXUID)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_privacy_failed", "xuid", targetXUID, "err", err)
		return nil
	}
	if info == nil {
		return nil
	}
	return domain.NewPrivacyWarning(*info)
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
