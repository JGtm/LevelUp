// Package service — ExplorerService : matchs communs, recherche croisée.
//
// Port Go de apps/api/app/routers/explorer.py.
package service

import (
	"context"
	"fmt"
	"log/slog"

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
		"target_profile_built", targetProfile != nil)

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

	// Goroutine 1 : identity Spartan — live si auth dispo, DB locale sinon.
	g.Go(func() error {
		if s.identityProvider == nil {
			return nil
		}
		if !hasAuth {
			identity = s.identityProvider.GetSpartanIdentityFromDBOnly(gctx, targetXUID)
			return nil
		}
		id, idErr := s.identityProvider.GetSpartanIdentityFor(gctx, targetXUID)
		if idErr != nil {
			slog.WarnContext(gctx, "explorer_target_identity_failed",
				"xuid", targetXUID, "err", idErr)
			return nil
		}
		identity = id
		return nil
	})

	// Goroutine 2 : stats carrière remote via PlayerStatsProvider.
	g.Go(func() error {
		if s.remoteStats == nil {
			return nil
		}
		if !hasAuth {
			slog.DebugContext(gctx, "explorer_target_career_skipped",
				"xuid", targetXUID, "reason", "no_auth_tokens")
			return nil
		}
		stats, rErr := s.remoteStats.FetchRemoteStats(gctx, targetGamertag, s.titleSlug)
		if rErr != nil {
			slog.WarnContext(gctx, "explorer_target_career_failed",
				"gamertag", targetGamertag, "err", rErr)
			return nil
		}
		careerStats = stats
		return nil
	})

	// Goroutine 3 : privacy warning du target.
	g.Go(func() error {
		if s.privacyProvider == nil {
			return nil
		}
		if !hasAuth {
			return nil
		}
		info, pErr := s.privacyProvider.GetMatchPrivacy(gctx, targetXUID)
		if pErr != nil {
			slog.WarnContext(gctx, "explorer_target_privacy_failed",
				"xuid", targetXUID, "err", pErr)
			return nil
		}
		if info != nil {
			privacy = domain.NewPrivacyWarning(*info)
		}
		return nil
	})

	// Goroutine 4 : sample stats (toujours calculé, indépendant des tokens).
	g.Go(func() error {
		matchIDs := extractCommonMatchIDs(rawMatches)
		if len(matchIDs) == 0 {
			return nil
		}
		agg, sErr := s.repo.GetParticipantStatsForMatches(gctx, targetXUID, matchIDs)
		if sErr != nil {
			slog.WarnContext(gctx, "explorer_target_sample_stats_failed",
				"xuid", targetXUID, "err", sErr)
			return nil
		}
		medals, mErr := s.repo.GetMedalCountsForMatches(gctx, targetXUID, matchIDs)
		if mErr != nil {
			slog.WarnContext(gctx, "explorer_target_medals_failed",
				"xuid", targetXUID, "err", mErr)
			// medals est nil → BuildSampleStats l'ignorera, ce n'est pas bloquant.
		}
		sampleStats = analysis.BuildSampleStats(agg, medals, len(matchIDs))
		return nil
	})

	_ = g.Wait()

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
