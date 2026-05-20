// Package service — MatchViewService : vue complète d'un match.
//
// Port Go de apps/api/app/services/match_view_service.py.
// Assemble les données des 4 onglets + header à partir du repo.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"

	"golang.org/x/sync/errgroup"
)

// outcomeColors : couleur hex par code d'outcome Halo Infinite.
//
// Couleurs hex legacy (rétrocompat front V0). Externalisées en constantes pour
// centraliser et permettre le lint goconst (toute nouvelle UI doit passer par
// outcomeColorToken / token CSS sémantique, cf. CLAUDE.md règle 20).
const (
	mvHexOutcomeWin     = "#22c55e" // Victoire
	mvHexOutcomeLoss    = "#ef4444" // Défaite
	mvHexOutcomeNeutral = "#8b5cf6" // Égalité / DNF
	mvHexOutcomeUnknown = "#94a3b8" // Fallback gris (outcome inconnu)
	mvHexPerfMedium     = "#3b82f6" // perfColor 60–80
	mvHexPerfLow        = "#f59e0b" // perfColor 40–60
)

// Deprecated: anti-pattern (CLAUDE.md règle 20 — aucun hex côté backend).
// Conservé pour rétrocompat avec les consommateurs front V0 qui n'ont pas
// encore migré vers tokenCssVar(). Utiliser outcomeColorToken pour les
// nouveaux champs (Phase 1 méta-plan § 6.1.3 — chunk MV3 cleanup).
//
// (outcomeLabels est défini dans match_history_service.go)
var outcomeColors = map[int]string{
	1: mvHexOutcomeNeutral, // Égalité
	2: mvHexOutcomeWin,     // Victoire
	3: mvHexOutcomeLoss,    // Défaite
	4: mvHexOutcomeNeutral, // Non terminé
}

// outcomeColorToken retourne le token sémantique pour un code outcome.
// Le front résout via tokenCssVar(token) (SemanticToken).
//
// Code -> token mapping :
//
//	1 (égalité)     -> "outcome-draw"
//	2 (victoire)    -> "outcome-win"
//	3 (défaite)     -> "outcome-loss"
//	4 (non terminé) -> "outcome-dnf"
//	autre/0         -> "" (pas de couleur sémantique applicable)
func outcomeColorToken(code int) string {
	switch code {
	case 1:
		return "outcome-draw"
	case 2:
		return "outcome-win"
	case 3:
		return "outcome-loss"
	case 4:
		return "outcome-dnf"
	}
	return ""
}

// perfColorToken retourne le token sémantique pour un score de performance.
// 5 paliers ordinaux mappés sur perf-tier-1..5.
//
// Score >= 80 -> "perf-tier-1" (meilleur)
// Score >= 60 -> "perf-tier-2"
// Score >= 40 -> "perf-tier-3"
// Score >= 20 -> "perf-tier-4"
// Score <  20 -> "perf-tier-5" (pire)
func perfColorToken(score float64) string {
	switch {
	case score >= 80:
		return "perf-tier-1"
	case score >= 60:
		return "perf-tier-2"
	case score >= 40:
		return "perf-tier-3"
	case score >= 20:
		return "perf-tier-4"
	}
	return "perf-tier-5"
}

// MatchViewService assemble la réponse Match View.
type MatchViewService struct {
	repo          port.MatchViewRepository
	citationsRepo port.CitationsRepository
	// eventsRepo (optionnel, Phase 1 méta-plan § 6.1.3 — chunk MV4.A) :
	// loader unifié des highlight_events qui remplace progressivement
	// repo.GetMatchEvents. Quand non-nil, les builders narrative cadence/impact
	// consomment directement des canonical.HighlightEvent (pas de conversion à
	// la volée). Dégradation gracieuse si nil : on retombe sur GetMatchEvents.
	eventsRepo port.HighlightEventsRepository
	// awardsRepo (optionnel, chunk MV4.B) : loader des personal_score_awards
	// pour le radar 6 axes via narrative.ComputeParticipationProfile. Si nil,
	// le radar reste vide (axes à 0).
	awardsRepo port.PersonalScoreAwardsRepository
	xuid       string
	// titleSlug est nécessaire pour HighlightEventsRepository (capability check
	// + selection de la DB shared). Injecté via WithTitleSlug.
	titleSlug string
	// dataAdapter (optionnel, Phase C+ multi-titres) : point d'extension pour
	// router LoadMatchDetail via la couche canonique. À ce jour, le service
	// utilise le repo direct car canonical.MatchDetail ne couvre pas encore
	// la totalité du payload Match View (4 onglets + header). Le hook est en
	// place pour permettre une bascule incrémentale.
	dataAdapter games.TitleDataAdapter
	// assetURL (optionnel) : adapter d'URLs d'assets (image map, badge rang).
	// Injecté via WithAssetURL au boot. Si nil, MapImageURL et IconURL restent
	// vides — le front affiche les fallbacks texte (dégradation gracieuse).
	assetURL games.TitleAssetURLAdapter
	// socialRepo (optionnel) : repo des données sociales (favoris). Injecté
	// via WithSocial. Si nil ou shared_social indisponible, IsFavorite reste
	// false — le bouton favori côté front reste fonctionnel mais idempotent.
	socialRepo port.SocialRepository
	// playerSlug : nécessaire pour les lookups socialRepo (clé de la table
	// match_favorites). Injecté via WithSocial avec le slug courant.
	playerSlug string
	// friendsExtras (optionnel) : loader des extras per-friend pour le panneau
	// d'expander scoreboard (perf score + skill rank + had_bot_teammate). Si
	// nil, la section "Local" du panneau ne s'affiche que pour `is_me`.
	// Cf. port.FriendsExtrasResolver et registry.MatchView.
	friendsExtras port.FriendsExtrasResolver
	// metadataRepo (optionnel) : lookup des coefs assists_model_coefs pour
	// calculer expected_assists à la volée. Dégradation gracieuse si nil.
	metadataRepo port.MetadataRepository
}

// NewMatchViewService crée un MatchViewService.
func NewMatchViewService(repo port.MatchViewRepository, xuid string) *MatchViewService {
	return &MatchViewService{repo: repo, xuid: xuid}
}

// WithDataAdapter injecte le DataAdapter multi-titres pour activer une
// future bascule LoadMatchDetail. Dégradation gracieuse si nil.
func (s *MatchViewService) WithDataAdapter(a games.TitleDataAdapter) *MatchViewService {
	s.dataAdapter = a
	return s
}

// WithCitationsRepo injecte le CitationsRepository pour peupler l'onglet Citations.
// Dégradation gracieuse si nil (onglet vide).
func (s *MatchViewService) WithCitationsRepo(r port.CitationsRepository) *MatchViewService {
	s.citationsRepo = r
	return s
}

// WithHighlightEventsRepo injecte le loader unifié des highlight_events
// (Phase 0 méta-plan, chunk 7). Quand câblé, le service consomme directement
// des canonical.HighlightEvent au lieu de convertir des EventRaw à la volée.
//
// Dégradation gracieuse : si nil, le service retombe sur repo.GetMatchEvents
// (Q21 legacy) + conversion EventRaw → canonical.HighlightEvent (chunk MV2).
func (s *MatchViewService) WithHighlightEventsRepo(r port.HighlightEventsRepository) *MatchViewService {
	s.eventsRepo = r
	return s
}

// WithTitleSlug configure le titre courant pour les calls qui en ont besoin
// (HighlightEventsRepository, capability gating). Injecté par le wiring HTTP
// avec ctxkeys.TitleSlug ou un fallback "halo_infinite".
func (s *MatchViewService) WithTitleSlug(slug string) *MatchViewService {
	s.titleSlug = slug
	return s
}

// WithFriendsExtras injecte le loader d'extras per-friend (perf score +
// skill rank + had_bot_teammate) pour le panneau d'expander du scoreboard.
// Dégradation gracieuse si nil : section "Local" du panneau active seulement
// pour le joueur principal (`is_me`).
func (s *MatchViewService) WithFriendsExtras(loader port.FriendsExtrasResolver) *MatchViewService {
	s.friendsExtras = loader
	return s
}

// WithMetadataRepo injecte le MetadataRepository pour le lookup des coefs
// assists_model_coefs (expected_assists à la volée). Dégradation gracieuse si nil.
func (s *MatchViewService) WithMetadataRepo(r port.MetadataRepository) *MatchViewService {
	s.metadataRepo = r
	return s
}

// WithAwardsRepo injecte le loader personal_score_awards pour le radar
// 6 axes (chunk MV4.B). Dégradation gracieuse si nil : radar à 0.
func (s *MatchViewService) WithAwardsRepo(r port.PersonalScoreAwardsRepository) *MatchViewService {
	s.awardsRepo = r
	return s
}

// WithAssetURL configure l'adapter d'URLs d'assets (map image, rank icon).
// Dégradation gracieuse : si nil ou si l'adapter retourne "", les champs
// restent vides côté response et le front affiche les fallbacks texte.
func (s *MatchViewService) WithAssetURL(a games.TitleAssetURLAdapter) *MatchViewService {
	s.assetURL = a
	return s
}

// WithSocial configure le repo social (favoris) et le slug joueur.
// Pattern identique à HomeService.WithSocial — IsFavorite reste false si nil.
func (s *MatchViewService) WithSocial(r port.SocialRepository, playerSlug string) *MatchViewService {
	s.socialRepo = r
	s.playerSlug = playerSlug
	return s
}

// GetMatchView retourne la réponse complète pour un match.
//
//nolint:funlen,cyclop // 11 sections séquentielles d'enrichissement bloquant : meta + analyses + médias
func (s *MatchViewService) GetMatchView(ctx context.Context, matchID string) (domain.MatchViewResponse, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("match_view_get", time.Since(start).Milliseconds())
	}(time.Now())
	// --- Appels séquentiels bloquants (meta est nécessaire pour la suite) ---
	meta, err := s.repo.GetMatchMeta(ctx, matchID)
	if err != nil {
		return domain.MatchViewResponse{}, fmt.Errorf("MatchViewService: meta: %w", err)
	}

	// --- Appels parallèles via errgroup ---
	var (
		stats      *domain.PlayerMatchStatsRaw
		enrich     *domain.MatchEnrichmentRaw
		scoreboard []domain.ScoreboardRaw
		medals     []domain.MedalRaw
		events     []domain.EventRaw
		// canonicalEvents : chargés via port.HighlightEventsRepository si câblé
		// (chunk MV4.A, loader unifié Phase 0). Sinon, conversion à la volée
		// depuis events (chunk MV2 legacy). Consommés par les builders narrative
		// (cadence + impact 8 rôles).
		canonicalEvents []canonical.HighlightEvent
		// encounterStats : stats riches par encounter (chunk MV4.C') chargées
		// via Q23b. Permet narrative.ComputeEncounterBadges (ally_plus +
		// tough_enemy). Optionnel — degradation gracieuse vers badge ordinal
		// seul si la repo retourne nil.
		encounterStats []domain.EncounterStatsRaw
		kvPairs        []domain.KVPairRaw
		skillRank      *domain.SkillRankRaw
		encounters     []domain.EncounterRaw
		media          []domain.MediaAssocRaw
		expected       *domain.ExpectedStatsRaw
		bulkMedals     []domain.BulkMedalRaw
		bulkWeapons    []domain.BulkWeaponKillRaw
		matchCitations []domain.CitationMatchViewRow
		richCitations  []domain.HomeMatchCitationRaw
		histRows       []domain.MatchHistAvgRow
		objectiveScore int
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var e error
		stats, e = s.repo.GetPlayerMatchStats(gctx, s.xuid, matchID)
		if e != nil {
			slog.Warn("match_view: stats indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		enrich, e = s.repo.GetMatchEnrichment(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: enrichment indisponible", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		scoreboard, e = s.repo.GetMatchScoreboard(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: scoreboard indisponible", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		// Score PSA catégorie 'objective' pour le joueur courant — alimente l'axe
		// Objective du radar synergie. Dégradation silencieuse à 0.
		objectiveScore, e = s.repo.GetMatchObjectiveScore(gctx, s.xuid, matchID)
		if e != nil {
			slog.Warn("match_view: objective score indisponible", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		medals, e = s.repo.GetMatchMedals(gctx, s.xuid, matchID)
		if e != nil {
			slog.Warn("match_view: medals indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		events, e = s.repo.GetMatchEvents(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: events indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	// MV4.A : chargement parallèle des events via le loader unifié si câblé.
	// Si l'eventsRepo n'est pas injecté, canonicalEvents reste nil et les
	// builders narrative retomberont sur la conversion à la volée (chunk MV2).
	if s.eventsRepo != nil {
		g.Go(func() error {
			filters := port.HighlightEventFilters{MatchIDs: []string{matchID}}
			if e := filters.Validate(); e != nil {
				slog.WarnContext(gctx, "match_view: HighlightEventFilters invalides",
					"match_id", matchID, "err", e)
				return nil
			}
			canonicalEv, e := s.eventsRepo.LoadHighlightEvents(gctx, s.titleSlug, filters)
			if e != nil {
				if !errors.Is(e, games.ErrCapabilityNotSupported) {
					slog.WarnContext(gctx, "match_view: LoadHighlightEvents echec",
						"match_id", matchID, "err", e)
				}
				return nil
			}
			canonicalEvents = canonicalEv
			return nil
		})
	}

	// MV4.B' : awards chargés après l'errgroup principal car ils dépendent du
	// scoreboard (xuids). Voir l'appel `s.loadAwardsForScoreboard(...)` plus
	// bas, après `g.Wait()`.
	g.Go(func() error {
		var e error
		kvPairs, e = s.repo.GetMatchKVPairs(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: kv_pairs indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		skillRank, e = s.repo.GetMatchSkillRank(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: skill_rank indisponible", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		encounters, e = s.repo.GetMatchEncounters(gctx, matchID, s.xuid)
		if e != nil {
			slog.Warn("match_view: encounters indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	// MV4.C' : chargement parallele des stats encounter riches (Q23b).
	g.Go(func() error {
		var e error
		encounterStats, e = s.repo.GetMatchEncounterStats(gctx, matchID, s.xuid)
		if e != nil {
			slog.Warn("match_view: encounter_stats indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		// Q24 retourne tous les auteurs (cross-joueur) : un coéquipier peut
		// avoir uploadé un media pour ce match.
		media, e = s.repo.GetMatchMedia(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: media indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		expected, e = s.repo.GetMatchExpectedStats(gctx, matchID, s.xuid)
		if e != nil {
			slog.Warn("match_view: expected_stats indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		bulkMedals, e = s.repo.GetMatchBulkMedals(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: bulk_medals indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		bulkWeapons, e = s.repo.GetMatchBulkWeaponKills(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: bulk_weapons indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		histRows, e = s.repo.GetHistoryForAvg(gctx, s.xuid)
		if e != nil {
			slog.Warn("match_view: hist_avg indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	if s.citationsRepo != nil {
		g.Go(func() error {
			var e error
			matchCitations, e = s.citationsRepo.LoadMatchCitationsForView(gctx, matchID)
			if e != nil {
				slog.Warn("match_view: citations indisponibles", "match_id", matchID, "err", e)
			}
			return nil
		})
		g.Go(func() error {
			var e error
			richCitations, e = s.citationsRepo.LoadMatchCitationsRich(gctx, matchID)
			if e != nil {
				slog.Warn("match_view: rich citations indisponibles", "match_id", matchID, "err", e)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return domain.MatchViewResponse{}, err
	}

	// Durée pour les bins tug-of-war
	var durationMS int64
	if meta != nil && meta.PlayableDurationSeconds != nil {
		durationMS = *meta.PlayableDurationSeconds * 1000
	}

	// IsFavorite : lookup synchrone (cheap, indexé sur PK player_slug+match_id).
	// Dégradation gracieuse si socialRepo nil ou shared_social indisponible.
	isFavorite := false
	if s.socialRepo != nil && s.playerSlug != "" {
		if fav, ferr := s.socialRepo.IsMatchFavorite(ctx, s.playerSlug, matchID); ferr == nil {
			isFavorite = fav
		} else {
			slog.WarnContext(ctx, "match_view: IsMatchFavorite échoué",
				"match_id", matchID, "player", s.playerSlug, "err", ferr)
		}
	}

	// expected_assists — uniquement pour le joueur suivi (is_me), jamais pour
	// les autres lignes du scoreboard.
	// Chaîne de résolution :
	//   1. Modèle personnel OLS (player_assists_model dans stats.duckdb)
	//   2. Fallback modèle populationnel (assists_model_coefs dans metadata.duckdb)
	if stats != nil && meta != nil && meta.GameVariantName != nil {
		v := computeExpectedAssists(ctx, s.repo, s.metadataRepo, *meta.GameVariantName, stats)
		if v != nil {
			if expected == nil {
				expected = &domain.ExpectedStatsRaw{}
			}
			expected.AssistsExpected = v
			// Propager aussi sur la ligne is_me du scoreboard (expander PlayerDetailPanel).
			for i := range scoreboard {
				if scoreboard[i].XUID == s.xuid {
					scoreboard[i].AssistsExpected = v
					break
				}
			}
		}
	}

	header := buildMatchHeader(ctx, matchID, meta, stats, enrich, scoreboard, s.assetURL, isFavorite)
	rank := buildRankBlock(skillRank, s.assetURL)
	summary := buildSummaryTabFull(stats, medals, expected, histRows, meta, s.titleSlug, richCitations)
	combat := buildCombatTabFull(matchID, bulkWeapons, events, canonicalEvents, kvPairs, scoreboard, s.xuid, durationMS)
	// Extras per-friend (panneau d'expander scoreboard) : best-effort, on
	// charge depuis chaque player DB d'ami configuré. Si pas de loader injecté
	// → map vide (section "Local" inactive sauf pour `is_me`).
	var friendsExtras map[string]port.FriendMatchExtras
	if s.friendsExtras != nil {
		xuids := make([]string, 0, len(scoreboard))
		for _, sb := range scoreboard {
			if sb.XUID != "" && sb.XUID != s.xuid {
				xuids = append(xuids, sb.XUID)
			}
		}
		if len(xuids) > 0 {
			gvn := ""
			if meta != nil && meta.GameVariantName != nil {
				gvn = *meta.GameVariantName
			}
			friendsExtras = s.friendsExtras(ctx, matchID, gvn, xuids)
		}
	}
	team := buildTeamTabFull(scoreboard, kvPairs, encounters, encounterStats, bulkMedals, bulkWeapons, s.xuid, s.titleSlug, enrich, skillRank, friendsExtras, s.assetURL)
	mediaTab := buildMediaTab(media)

	// MV4.B' : radar 6 axes calculé depuis le scoreboard (kills/HS/PK/assists/
	// accuracy/deaths/damage/score). Mêmes formules que le radar squad
	// (loadSynergyMateAxes), appliquées à un seul match. Pas besoin de
	// personal_score_awards — toutes les colonnes nécessaires sont déjà dans
	// match_participants. L'axe Objective reste neutre (threshold=0).
	modeFamily := matchModeFamilyFromMeta(meta)
	radarSeries := BuildMatchRadarFromScoreboard(scoreboard, s.xuid, objectiveScore, modeFamily)
	var radar []any
	for _, s := range radarSeries {
		radar = append(radar, s)
	}

	// RC6 — détection sync incomplet : le match_registry est OK (sinon on aurait
	// court-circuité plus haut), mais une ou plusieurs sources secondaires sont
	// vides. Le front peut afficher un bandeau dégradé au lieu de l'écran
	// "Match introuvable ou erreur de chargement" full-page.
	partialReasons := detectPartialMatchData(stats, scoreboard, events, medals)

	return domain.MatchViewResponse{
		Header:         header,
		Rank:           rank,
		SummaryTab:     summary,
		CombatTab:      combat,
		TeamTab:        team,
		MediaTab:       mediaTab,
		CitationsTab:   buildCitationsTab(matchCitations, medals, s.titleSlug),
		Radar:          radar,
		IsPartial:      len(partialReasons) > 0,
		PartialReasons: partialReasons,
	}, nil
}

// strDeref retourne la valeur d'un *string ou "<nil>" pour les logs structurés.
// Évite les faux-positifs "<nil>" dans slog quand on veut juste tracer le contenu.
func strDeref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// detectPartialMatchData inspecte les sources secondaires d'un match et
// retourne la liste des raisons (codes stables) pour lesquelles la vue est
// considérée partielle. Vide si tout est plein.
//
// Codes utilisés (stables — front les mappe à des messages i18n) :
//   - "scoreboard_empty"     → Q12 a renvoyé 0 lignes
//   - "events_empty"         → Q21 a renvoyé 0 highlight events
//   - "player_stats_empty"   → Q17 stats joueur courant absentes (outcome = 0)
//   - "medals_empty"         → Q14 a renvoyé 0 médailles (rare ; certains modes
//     n'attribuent pas de médailles, donc pas critique pour la sync mais utile
//     pour le front)
func detectPartialMatchData(
	stats *domain.PlayerMatchStatsRaw,
	scoreboard []domain.ScoreboardRaw,
	events []domain.EventRaw,
	medals []domain.MedalRaw,
) []string {
	var reasons []string
	if len(scoreboard) == 0 {
		reasons = append(reasons, "scoreboard_empty")
	}
	if len(events) == 0 {
		reasons = append(reasons, "events_empty")
	}
	if stats == nil || stats.OutcomeCode == 0 {
		reasons = append(reasons, "player_stats_empty")
	}
	if len(medals) == 0 {
		reasons = append(reasons, "medals_empty")
	}
	return reasons
}

// loadAwardsForScoreboard charge les awards pour tous les xuids du scoreboard
// (chunk MV4.B'). Sérialisé après l'errgroup principal — la liste des xuids
// dépend du scoreboard chargé en parallèle. Dégradation gracieuse :
//
//	awardsRepo nil       -> retourne nil
//	scoreboard vide      -> retourne nil
//	capability absente   -> retourne nil (silencieux)
//	autre erreur         -> log warn + retourne nil
func (s *MatchViewService) loadAwardsForScoreboard(
	ctx context.Context,
	matchID string,
	scoreboard []domain.ScoreboardRaw,
) []port.PersonalScoreAwardRow {
	if s.awardsRepo == nil || len(scoreboard) == 0 {
		return nil
	}
	xuids := extractMatchSquadXUIDs(scoreboard)
	if len(xuids) == 0 {
		return nil
	}
	filters := port.PersonalScoreAwardsFilters{
		MatchIDs: []string{matchID},
		XUIDs:    xuids,
	}
	if err := filters.Validate(); err != nil {
		slog.WarnContext(ctx, "match_view: PersonalScoreAwardsFilters invalides",
			"match_id", matchID, "err", err)
		return nil
	}
	rows, err := s.awardsRepo.LoadPersonalScoreAwards(ctx, s.titleSlug, filters)
	if err != nil {
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.WarnContext(ctx, "match_view: LoadPersonalScoreAwards echec",
				"match_id", matchID, "err", err)
		}
		return nil
	}
	return rows
}

// matchModeFamilyFromMeta résout la mode family pour le calcul des seuils
// radar (narrative.DefaultThresholds). Best-effort : on inspecte pair_name
// pour identifier slayer / ctf / strongholds / oddball.
//
// Si la pair_name ne match aucun pattern connu, retourne "" (thresholds
// custom neutres).
func matchModeFamilyFromMeta(meta *domain.MatchMetaRaw) string {
	if meta == nil || meta.PairName == nil {
		return ""
	}
	name := strings.ToLower(*meta.PairName)
	switch {
	case strings.Contains(name, "slayer"):
		return "slayer"
	case strings.Contains(name, "ctf") || strings.Contains(name, "capture"):
		return "ctf"
	case strings.Contains(name, "stronghold"):
		return "strongholds"
	case strings.Contains(name, "oddball") || strings.Contains(name, "neutral"):
		return "oddball"
	}
	return ""
}

// GetMatchNeighbors retourne les matchs adjacents pour la navigation prev/next.
func (s *MatchViewService) GetMatchNeighbors(ctx context.Context, matchID string) (domain.MatchNeighbors, error) {
	slog.Info("match_view: GetMatchNeighbors", "match_id", matchID, "xuid", s.xuid)
	n, err := s.repo.GetMatchNeighbors(ctx, s.xuid, matchID)
	if err != nil {
		slog.Warn("match_view: neighbors indisponibles", "match_id", matchID, "err", err)
		return domain.MatchNeighbors{}, nil
	}
	if n == nil {
		return domain.MatchNeighbors{}, nil
	}
	slog.Info("match_view: neighbors chargés",
		"match_id", matchID,
		"current_index", n.CurrentIndex,
		"total", n.TotalMatches,
		"has_prev", n.PreviousMatchID != nil,
		"has_next", n.NextMatchID != nil,
	)
	return *n, nil
}

// GetMatchNeighborsFiltered : variante Phase 2b — filtres MatchFilterSpec
// transmis au repo pour Q25 paramétrable. spec nil/vide → comportement
// identique à GetMatchNeighbors.
func (s *MatchViewService) GetMatchNeighborsFiltered(
	ctx context.Context,
	matchID string,
	spec *domain.MatchFilterSpec,
) (domain.MatchNeighbors, error) {
	slog.DebugContext(ctx, "match_view: GetMatchNeighborsFiltered",
		"match_id", matchID, "filtered", !spec.IsEmpty())
	n, err := s.repo.GetMatchNeighborsFiltered(ctx, s.xuid, matchID, spec)
	if err != nil {
		slog.ErrorContext(ctx, "match_view: filtered neighbors query failed",
			"err", err, "match_id", matchID)
		return domain.MatchNeighbors{}, nil
	}
	if n == nil {
		return domain.MatchNeighbors{}, nil
	}
	out := *n
	if !spec.IsEmpty() {
		out.AppliedFilters = spec
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Header
// ---------------------------------------------------------------------------

func buildMatchHeader(
	ctx context.Context,
	matchID string,
	meta *domain.MatchMetaRaw,
	stats *domain.PlayerMatchStatsRaw,
	enrich *domain.MatchEnrichmentRaw,
	scoreboard []domain.ScoreboardRaw,
	assetURL games.TitleAssetURLAdapter,
	isFavorite bool,
) domain.MatchViewHeader {
	h := domain.MatchViewHeader{
		MatchID:      matchID,
		OutcomeLabel: "-",
		OutcomeColor: mvHexOutcomeUnknown,
		PerfDisplay:  "-",
		IsFavorite:   isFavorite,
	}

	if meta == nil {
		return h
	}

	h.StartTime = meta.StartTime
	if meta.StartTime != nil {
		h.StartTimeLabel = formatDateFRLong(*meta.StartTime)
	}
	if meta.MapNameFR != nil && *meta.MapNameFR != "" {
		h.MapUI = *meta.MapNameFR
	} else if meta.MapName != nil {
		h.MapUI = *meta.MapName
	}
	if meta.MapAssetID != nil {
		h.MapID = *meta.MapAssetID
	}
	// ModeNameFR est normalement déjà le résultat de analysis.ResolveModeUI
	// côté repo. Fallback défense-en-profondeur via le même helper si jamais
	// un caller externe construit un MatchMetaRaw sans pré-résoudre.
	modeUI := meta.ModeNameFR
	if modeUI == nil || *modeUI == "" {
		modeUI = analysis.ResolveModeUI(meta.PairName, meta.PairNameFR)
	}
	if modeUI != nil {
		h.ModeUI = *modeUI
	}
	// Playlist : priorité à la traduction FR (asset_translations), fallback
	// nom brut EN (match_registry.playlist_name).
	if meta.PlaylistNameFR != nil && *meta.PlaylistNameFR != "" {
		h.PlaylistLabel = *meta.PlaylistNameFR
	} else if meta.PlaylistName != nil {
		h.PlaylistLabel = *meta.PlaylistName
	}
	// MapImageURL : cascade de résolution
	//   1. map_images_registry (lookup par map_id stable, peuplé par
	//      cmd/migrate-static-maps).
	//   2. AssetURLAdapter avec **nom EN résolu** (asset_translations en-US) —
	//      l'adapter indexe les fichiers `static/maps/halo_infinite/{name}.{ext}`
	//      par nom EN. Sans le nom EN, on aurait l'UUID brut de
	//      match_registry.map_name → l'adapter rejette via uuidRe.
	//   3. AssetURLAdapter avec map_name brut (legacy fallback — utile si
	//      asset_translations EN absent et map_name est déjà un nom propre).
	if meta.MapImageURL != nil && *meta.MapImageURL != "" {
		h.MapImageURL = meta.MapImageURL
	} else if assetURL != nil {
		nameForAdapter := ""
		if meta.MapNameEN != nil && *meta.MapNameEN != "" {
			nameForAdapter = *meta.MapNameEN
		} else if meta.MapName != nil && *meta.MapName != "" {
			nameForAdapter = *meta.MapName
		}
		if nameForAdapter != "" {
			if url := assetURL.MapImageURL(nameForAdapter); url != "" {
				h.MapImageURL = &url
			} else {
				slog.WarnContext(ctx, "match_header: map image missing",
					"match_id", matchID,
					"map_name_used", nameForAdapter,
					"map_name_raw", strDeref(meta.MapName),
					"map_name_en", strDeref(meta.MapNameEN))
			}
		}
	}
	h.PlayableDurationSeconds = meta.PlayableDurationSeconds
	h.IsRanked = meta.IsRanked
	if meta.MapAssetID != nil {
		h.WaypointURL = fmt.Sprintf("https://www.halowaypoint.com/halo-infinite/matches/%s", matchID)
	}

	if stats != nil && stats.OutcomeCode != 0 {
		code := stats.OutcomeCode
		h.OutcomeCode = &code
		h.OutcomeLabel = outcomeLabel(code)
		h.OutcomeColor = outcomeColor(code)
		// Phase 1 méta-plan § 6.1.3 — chunk MV3 cleanup hex codes.
		// OutcomeColorToken est résolu côté front via tokenCssVar(),
		// remplace progressivement OutcomeColor (hex legacy).
		h.OutcomeColorToken = outcomeColorToken(code)
		h.ScoreLabel = buildScoreLabelFromMeta(meta, stats)
	}

	if enrich != nil {
		isDNF := stats != nil && stats.OutcomeCode == 4
		if enrich.PerformanceScore != nil && !isDNF {
			perf := *enrich.PerformanceScore
			display := fmt.Sprintf("%.0f", perf)
			h.PerfDisplay = display
			color := perfColor(perf)
			h.PerfColor = &color
			// Token sémantique perf-tier-1..5 (cf. MV3 cleanup hex).
			h.PerfColorToken = perfColorToken(perf)
		}
		h.IsExcluded = enrich.IsExcluded

		// Phase 1 méta-plan § 6.1.3 — pilote MatchView aligné sur les fondations
		// narrative. Résolution du badge typé via narrative.ResolveDominanceBadge.
		// Le bool legacy `dominance_flag` reste exposé pour rétrocompat (true si
		// un badge narratif s'applique).
		flag := canonical.DominanceFlag(enrich.DominanceFlag)
		if badge := narrative.ResolveDominanceBadge(flag); badge != nil {
			h.DominanceFlag = true
			h.DominanceBadge = &domain.MatchViewDominanceBadge{
				Flag:       int(badge.Flag),
				LabelKey:   badge.LabelKey,
				ColorToken: badge.ColorToken,
			}
		}
	}

	return h
}

// buildScoreLabelFromMeta construit "X-Y" depuis team_0_score/team_1_score de
// match_registry. L'équipe du joueur (stats.TeamID) est toujours affichée en
// premier (miroir de buildHomeScoreLabel dans analysis/home.go).
func buildScoreLabelFromMeta(meta *domain.MatchMetaRaw, stats *domain.PlayerMatchStatsRaw) string {
	if meta == nil || meta.Team0Score == nil || meta.Team1Score == nil {
		return ""
	}
	s0, s1 := int(*meta.Team0Score), int(*meta.Team1Score)
	if s0 < 0 || s1 < 0 {
		return ""
	}
	if stats != nil && stats.TeamID != nil && *stats.TeamID == 1 {
		return fmt.Sprintf("%d-%d", s1, s0)
	}
	return fmt.Sprintf("%d-%d", s0, s1)
}

// buildRankBlock construit le bloc rank depuis SkillRankRaw.
//
// IconURL est résolu via TitleAssetURLAdapter (CSRRankImageURL ou
// CSRRankImageURLOnyx selon le tier). Dégradation gracieuse :
//   - assetURL nil → IconURL = "" (front affiche fallback texte)
//   - rating_type CSR mais tier inconnu → IconURL = ""
//   - rating_type LUSR (custom games) → pas de badge officiel, IconURL = ""
func buildRankBlock(sr *domain.SkillRankRaw, assetURL games.TitleAssetURLAdapter) domain.MatchViewRank {
	if sr == nil {
		return domain.MatchViewRank{RatingType: "none"}
	}
	rank := domain.MatchViewRank{RatingType: sr.RatingType}
	if sr.TierLabel != nil {
		rank.TierLabel = sr.TierLabel
	}
	rank.NumericVal = sr.RatingValue
	rank.DeltaValue = sr.RatingDelta

	// ProgressPct : position dans le sous-tier (0.0–1.0).
	// CSR et LUSR Halo Infinite ont tous les deux des sous-tiers de 50 points.
	// Même constante que home_canonical.go (tierSize = 50).
	// Onyx : nil (pas de tier suivant défini).
	if sr.RatingValue != nil && sr.Tier != nil && !strings.EqualFold(*sr.Tier, "Onyx") {
		const tierSize = 50.0
		pts := math.Mod(*sr.RatingValue, tierSize)
		if pts < 0 {
			pts += tierSize
		}
		pct := pts / tierSize
		rank.ProgressPct = &pct
	}

	// Badge image — LUSR utilise les mêmes fichiers que CSR (même dossier static).
	// Onyx : pas de sub-tier → CSRRankImageURLOnyx().
	// Autres tiers (Bronze, Silver, Gold, Platinum, Diamond) : tier + sub-tier.
	// Sources : match_skill_rank.tier (EN, TitleCase) + match_skill_rank.sub_tier.
	if assetURL == nil || sr.Tier == nil || *sr.Tier == "" {
		return rank
	}
	tier := *sr.Tier
	subTier := 0
	if sr.SubTier != nil {
		subTier = *sr.SubTier
	}
	// Fallback : dériver sub_tier depuis tier_label quand sub_tier = 0 (défaut DB).
	if subTier <= 0 && sr.TierLabel != nil {
		parts := strings.Fields(strings.TrimSpace(*sr.TierLabel))
		if len(parts) > 1 {
			if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				subTier = n
			}
		}
	}
	if strings.EqualFold(tier, "Onyx") {
		rank.IconURL = assetURL.CSRRankImageURLOnyx()
	} else if subTier >= 1 && subTier <= 6 {
		rank.IconURL = assetURL.CSRRankImageURL(tier, subTier)
	}
	return rank
}

// ---------------------------------------------------------------------------
// Summary Tab
// ---------------------------------------------------------------------------

func buildSummaryTabFull(
	stats *domain.PlayerMatchStatsRaw,
	medals []domain.MedalRaw,
	expected *domain.ExpectedStatsRaw,
	histRows []domain.MatchHistAvgRow,
	meta *domain.MatchMetaRaw,
	titleSlug string,
	richCitations []domain.HomeMatchCitationRaw,
) domain.MatchSummaryTab {
	citations := analysis.BuildCitationSnippets(richCitations, math.MaxInt32)
	if citations == nil {
		citations = []domain.MatchCitationSnippet{}
	}
	tab := domain.MatchSummaryTab{
		KPIs:           domain.MatchSummaryKpis{},
		PersonalResult: domain.MatchPersonalResult{OutcomeLabel: "-", OutcomeColor: mvHexOutcomeUnknown},
		Medals:         convertMedals(medals, titleSlug),
		Citations:      citations,
		ExpectedStats:  buildExpectedStats(expected, histRows, meta),
	}

	if stats == nil {
		return tab
	}

	var deltaMMR *float64
	if stats.TeamMMR != nil && stats.EnemyMMR != nil {
		d := *stats.TeamMMR - *stats.EnemyMMR
		deltaMMR = &d
	}

	// perfect_kills depuis les médailles (medal_name_id 1512363953)
	const perfectKillMedalID = int64(1512363953)
	var perfectKills int
	for _, m := range medals {
		if m.MedalID == perfectKillMedalID {
			perfectKills += m.Count
		}
	}

	tab.KPIs = domain.MatchSummaryKpis{
		Kills:           &stats.Kills,
		Deaths:          &stats.Deaths,
		Assists:         &stats.Assists,
		KDA:             stats.KDA,
		DamageDealt:     stats.DamageDealt,
		AverageLife:     formatLifeSeconds(stats.AvgLifeSeconds),
		Accuracy:        stats.Accuracy,
		PersonalScore:   toIntPtr(stats.PersonalScore),
		TeamMMR:         stats.TeamMMR,
		EnemyMMR:        stats.EnemyMMR,
		DeltaMMR:        deltaMMR,
		HeadshotKills:   stats.HeadshotKills,
		MaxKillingSpree: stats.MaxKillingSpree,
		PerfectKills:    &perfectKills,
	}

	if stats.OutcomeCode != 0 {
		score := 0
		if stats.PersonalScore != nil {
			score = int(math.Round(*stats.PersonalScore))
		}
		tab.PersonalResult = domain.MatchPersonalResult{
			OutcomeLabel:      outcomeLabel(stats.OutcomeCode),
			OutcomeColor:      outcomeColor(stats.OutcomeCode),
			OutcomeColorToken: outcomeColorToken(stats.OutcomeCode),
			Score:             &score,
			RankInTeam:        stats.RankInTeam,
		}
	}

	return tab
}

// computeExpectedAssists calcule expected_assists pour le joueur suivi (is_me).
// Résolution : modèle personnel OLS → fallback modèle populationnel → nil.
func computeExpectedAssists(
	ctx context.Context,
	repo port.MatchViewRepository,
	metaRepo port.MetadataRepository,
	gameVariantName string,
	stats *domain.PlayerMatchStatsRaw,
) *float64 {
	kills := float64(stats.Kills)
	deaths := float64(stats.Deaths)
	dd := 0.0
	if stats.DamageDealt != nil {
		dd = *stats.DamageDealt
	}
	dt := 0.0
	if stats.DamageTaken != nil {
		dt = *stats.DamageTaken
	}
	mmrDelta := 0.0
	if stats.TeamMMR != nil && stats.EnemyMMR != nil {
		mmrDelta = *stats.TeamMMR - *stats.EnemyMMR
	}

	// 1. Modèle personnel
	if m, err := repo.GetPlayerAssistsModel(ctx, gameVariantName); err == nil && m != nil {
		raw := m.Intercept +
			m.CoefKills*kills +
			m.CoefDeaths*deaths +
			m.CoefDamageDealt*dd +
			m.CoefDamageTaken*dt +
			m.CoefMMRDelta*mmrDelta
		v := math.Round(raw*100) / 100
		return &v
	}

	// 2. Fallback modèle populationnel (slope × (personal_score + shots_hit) + intercept)
	if metaRepo == nil {
		return nil
	}
	slope, intercept, err := metaRepo.GetAssistsCoef(ctx, gameVariantName)
	if err != nil {
		return nil
	}
	ps := 0.0
	if stats.PersonalScore != nil {
		ps = *stats.PersonalScore
	}
	sh := 0.0
	if stats.ShotsHit != nil {
		sh = float64(*stats.ShotsHit)
	}
	v := math.Round((slope*(ps+sh)+intercept)*100) / 100
	return &v
}

// buildExpectedStats construit le bloc de stats attendues + moyennes historiques.
func buildExpectedStats(e *domain.ExpectedStatsRaw, histRows []domain.MatchHistAvgRow, meta *domain.MatchMetaRaw) domain.MatchExpectedStats {
	out := domain.MatchExpectedStats{}
	if e != nil {
		out.ExpectedKills = e.KillsExpected
		out.ExpectedDeaths = e.DeathsExpected
		out.ExpectedAssists = e.AssistsExpected
		out.HasExpectedData = out.ExpectedKills != nil || out.ExpectedDeaths != nil || out.ExpectedAssists != nil
	}
	if len(histRows) == 0 || meta == nil {
		return out
	}

	pairName := ""
	if meta.PairName != nil {
		pairName = *meta.PairName
	}
	targetCat := analysis.ComputeModeCategory(pairName, meta.IsFirefight, meta.IsRanked)

	var totalK, totalD, totalA, totalSpree, totalHS, totalPerfect, count int
	for _, row := range histRows {
		cat := analysis.ComputeModeCategory(row.PairName, row.IsFirefight, row.IsRanked)
		if cat != targetCat {
			continue
		}
		totalK += row.Kills
		totalD += row.Deaths
		totalA += row.Assists
		if row.HeadshotKills != nil {
			totalHS += *row.HeadshotKills
		}
		if row.MaxKillingSpree != nil {
			totalSpree += *row.MaxKillingSpree
		}
		totalPerfect += row.PerfectKills
		count++
	}
	if count == 0 {
		return out
	}
	n := float64(count)
	avgK := float64(totalK) / n
	avgD := float64(totalD) / n
	avgA := float64(totalA) / n
	avgHS := float64(totalHS) / n
	avgSpree := float64(totalSpree) / n
	avgPerfect := float64(totalPerfect) / n

	out.HasHistAvg = true
	out.HistAvgKills = &avgK
	out.HistAvgDeaths = &avgD
	out.HistAvgAssists = &avgA
	out.HistAvgHeadshotKills = &avgHS
	out.HistAvgSpree = &avgSpree
	out.HistAvgPerfectKills = &avgPerfect
	out.HistMatchCount = count
	out.HistModeCategory = targetCat
	return out
}

func toIntPtr(f *float64) *int {
	if f == nil {
		return nil
	}
	v := int(math.Round(*f))
	return &v
}

func convertMedals(raw []domain.MedalRaw, titleSlug string) []domain.MatchMedal {
	if len(raw) == 0 {
		return []domain.MatchMedal{}
	}
	medals := make([]domain.MatchMedal, 0, len(raw))
	for _, r := range raw {
		imgURL := static.URL(static.KindMedal, titleSlug, strconv.FormatInt(r.MedalID, 10), ".png")
		var desc *string
		if r.Description != "" {
			d := r.Description
			desc = &d
		}
		medals = append(medals, domain.MatchMedal{
			MedalNameID: r.MedalID,
			Name:        r.Label,
			Count:       r.Count,
			Description: desc,
			ImageURL:    imgURL,
			Difficulty:  r.Difficulty,
		})
	}
	return medals
}

// buildCitationsTab construit l'onglet Citations depuis les données chargées.
func buildCitationsTab(citations []domain.CitationMatchViewRow, medals []domain.MedalRaw, titleSlug string) domain.MatchCitationsTab {
	commendations := make([]domain.MatchCitation, 0, len(citations))
	for _, c := range citations {
		val := float64(c.Value)
		commendations = append(commendations, domain.MatchCitation{
			Key:   c.NameNorm,
			Label: c.NameDisplay,
			Value: &val,
		})
	}
	return domain.MatchCitationsTab{
		Commendations: commendations,
		Medals:        convertMedals(medals, titleSlug),
	}
}

// ---------------------------------------------------------------------------
// Combat Tab
// ---------------------------------------------------------------------------

func buildCombatTabFull(
	matchID string,
	bulkWeapons []domain.BulkWeaponKillRaw,
	events []domain.EventRaw,
	canonicalEvents []canonical.HighlightEvent,
	kvPairs []domain.KVPairRaw,
	scoreboard []domain.ScoreboardRaw,
	myXUID string,
	durationMS int64,
) domain.MatchCombatTab {
	wkList := make([]domain.MatchWeaponKill, 0)
	for _, w := range bulkWeapons {
		if w.XUID != myXUID {
			continue
		}
		wkList = append(wkList, domain.MatchWeaponKill{
			WeaponID:    w.WeaponID,
			WeaponLabel: w.WeaponLabel,
			KillCount:   w.Kills,
		})
	}

	evtList := make([]domain.MatchHighlightEvent, 0, len(events))
	for _, e := range events {
		evtList = append(evtList, domain.MatchHighlightEvent{
			EventType:     e.EventType,
			EventTimeMS:   e.TimeMS,
			ActorXUID:     e.XUID,
			ActorGamertag: e.Gamertag,
		})
	}

	// Tug-of-war
	tugEvents := buildTugEvents(kvPairs, myXUID)
	tugBins := analysis.ComputeTugOfWar(tugEvents, durationMS, 0)
	tugDomain := make([]domain.MatchTugOfWarBin, 0, len(tugBins))
	for _, b := range tugBins {
		allyKills := 0
		enemyKills := 0
		if b.Delta > 0 {
			allyKills = b.Delta
		} else {
			enemyKills = -b.Delta
		}
		tugDomain = append(tugDomain, domain.MatchTugOfWarBin{
			BinStart:   int(b.BinStartMS / 1000),
			BinEnd:     int(b.BinEndMS / 1000),
			TeamKills:  allyKills,
			EnemyKills: enemyKills,
			NetKills:   b.CumDelta,
		})
	}

	// Impact badges : calculés en périmètre team-wide alliée (parité Python
	// _match_impact_events::compute_single_match_impact). Le filtre par
	// team_id du main est appliqué dans buildImpactInput → seuls les
	// participants alliés sont passés à l'analyse, mais les events restent
	// full (first_blood reste global toutes équipes).
	impactInput := buildImpactInput(events, scoreboard, myXUID)
	allBadges := analysis.ComputeMatchImpactFull(impactInput)
	badgesDomain := make([]domain.MatchImpactBadge, 0, len(allBadges))
	for _, b := range allBadges {
		badge := domain.MatchImpactBadge{
			Key:        b.BadgeKey,
			Label:      b.BadgeFR,
			PlayerXUID: b.PlayerXUID,
		}
		if b.TimeMS > 0 {
			t := b.TimeMS
			badge.TimeMS = &t
		}
		badgesDomain = append(badgesDomain, badge)
	}

	// KD timeline
	kdEvents := buildKDEvents(kvPairs, myXUID)
	kdPoints := analysis.ComputeKDTimeline(kdEvents, myXUID)
	kdDomain := make([]domain.MatchKDTimelinePoint, 0, len(kdPoints))
	for _, p := range kdPoints {
		kdDomain = append(kdDomain, domain.MatchKDTimelinePoint{
			TimeSeconds: int(p.TimeMS / 1000),
			Kills:       p.CumKills,
			Deaths:      p.CumDeaths,
		})
	}

	// Phase 1 méta-plan § 6.1.3 — pilote MatchView aligné fondations narrative.
	// Cadence intra-match + 8 rôles narratifs en parallèle des badges legacy.
	//
	// MV4.A : si canonicalEvents est peuplé (loader unifié actif), on l'utilise
	// directement. Sinon fallback sur la conversion à la volée depuis EventRaw.
	var cadence *domain.ChartSeries[domain.ChartPointStacked]
	var impactRoles []domain.MatchViewImpactRole
	if len(canonicalEvents) > 0 {
		cadence = BuildMatchCadenceChartFromCanonical(canonicalEvents, scoreboard)
		impactRoles = BuildMatchImpactRoles8FromCanonical(canonicalEvents, scoreboard)
	} else {
		cadence = BuildMatchCadenceChart(events, scoreboard, matchID)
		impactRoles = BuildMatchImpactRoles8(events, scoreboard, matchID)
	}

	// Killer-victim aggregation (chart match_view.18 — antagonistes).
	killerVictim := buildKillerVictimPairs(kvPairs, scoreboard)

	return domain.MatchCombatTab{
		WeaponKills:     wkList,
		HighlightEvents: evtList,
		TugOfWar:        tugDomain,
		ImpactBadges:    badgesDomain,
		KDTimeline:      kdDomain,
		NemesisDuels:    []domain.MatchNemesisRow{},
		KillerVictim:    killerVictim,
		ImpactRoles:     impactRoles,
		Cadence:         cadence,
	}
}

// buildKillerVictimPairs agrège les kvPairs par (killer_xuid, victim_xuid).
// Résout les gamertags via le scoreboard quand kv.{Killer,Victim}GT est vide.
func buildKillerVictimPairs(
	kvPairs []domain.KVPairRaw,
	scoreboard []domain.ScoreboardRaw,
) []domain.MatchKillerVictimPair {
	if len(kvPairs) == 0 {
		return nil
	}
	gtMap := make(map[string]string, len(scoreboard))
	for _, s := range scoreboard {
		if s.Gamertag != "" {
			gtMap[s.XUID] = s.Gamertag
		}
	}
	resolveGT := func(xuid, fallback string) string {
		if gt, ok := gtMap[xuid]; ok && gt != "" {
			return gt
		}
		if fallback != "" {
			return fallback
		}
		return xuid
	}

	type pairKey struct {
		killer, victim string
	}
	agg := make(map[pairKey]*domain.MatchKillerVictimPair)
	order := make([]pairKey, 0)

	for _, kv := range kvPairs {
		if kv.KillerXUID == "" || kv.VictimXUID == "" {
			continue
		}
		k := pairKey{killer: kv.KillerXUID, victim: kv.VictimXUID}
		count := kv.KillCount
		if count <= 0 {
			count = 1
		}
		if existing, ok := agg[k]; ok {
			existing.KillCount += count
			continue
		}
		agg[k] = &domain.MatchKillerVictimPair{
			KillerXUID:     kv.KillerXUID,
			KillerGamertag: resolveGT(kv.KillerXUID, kv.KillerGT),
			VictimXUID:     kv.VictimXUID,
			VictimGamertag: resolveGT(kv.VictimXUID, kv.VictimGT),
			KillCount:      count,
		}
		order = append(order, k)
	}

	pairs := make([]domain.MatchKillerVictimPair, 0, len(order))
	for _, k := range order {
		pairs = append(pairs, *agg[k])
	}
	return pairs
}

func buildTugEvents(kvPairs []domain.KVPairRaw, myXUID string) []analysis.TugOfWarEvent {
	events := make([]analysis.TugOfWarEvent, 0, len(kvPairs))
	for _, kv := range kvPairs {
		isAlly := kv.KillerXUID == myXUID
		events = append(events, analysis.TugOfWarEvent{
			TimeMS:    kv.TimeMS,
			IsAlly:    isAlly,
			EventType: analysis.EventTypeKill,
		})
	}
	return events
}

// buildImpactInput convertit les données brutes du match vers MatchImpactInput.
// Les events highlight_events (kill/death + horodatage + acteur) alimentent
// les badges event-based ; le scoreboard fournit les stats par joueur pour
// les badges stat-based (top_killer, silent_hero, false_brother).
//
// Périmètre team-wide alliée : Participants ne contient QUE les joueurs de la
// même team_id que myXUID (le main). Les events restent full (first_blood
// nécessite tous les kills toutes équipes confondues). Si myXUID n'est pas
// trouvé dans le scoreboard ou n'a pas de team_id, on dégrade en passant tous
// les participants pour ne pas casser silencieusement le calcul.
func buildImpactInput(events []domain.EventRaw, scoreboard []domain.ScoreboardRaw, myXUID string) analysis.MatchImpactInput {
	impactEvents := make([]analysis.ImpactEvent, 0, len(events))
	for _, ev := range events {
		if ev.TimeMS == nil || ev.XUID == nil {
			continue
		}
		et := ev.EventType
		if et != "kill" && et != "death" {
			continue
		}
		impactEvents = append(impactEvents, analysis.ImpactEvent{
			TimeMS:    *ev.TimeMS,
			EventType: et,
			ActorXUID: *ev.XUID,
		})
	}

	var mainTeamID *int
	for _, p := range scoreboard {
		if p.XUID == myXUID && p.TeamID != nil {
			mainTeamID = p.TeamID
			break
		}
	}

	snaps := make([]analysis.ParticipantSnap, 0, len(scoreboard))
	for _, p := range scoreboard {
		if mainTeamID != nil && (p.TeamID == nil || *p.TeamID != *mainTeamID) {
			continue
		}
		snaps = append(snaps, analysis.ParticipantSnap{
			XUID:    p.XUID,
			Outcome: p.OutcomeCode,
			Kills:   p.Kills,
			Deaths:  p.Deaths,
			Assists: p.Assists,
		})
	}
	return analysis.MatchImpactInput{
		Events:       impactEvents,
		Participants: snaps,
	}
}

func buildKDEvents(kvPairs []domain.KVPairRaw, myXUID string) []analysis.KDEvent {
	events := make([]analysis.KDEvent, 0, len(kvPairs)*2)
	for _, kv := range kvPairs {
		if kv.KillerXUID == myXUID {
			events = append(events, analysis.KDEvent{
				TimeMS:    kv.TimeMS,
				IsKill:    true,
				ActorXUID: myXUID,
			})
		}
		if kv.VictimXUID == myXUID {
			events = append(events, analysis.KDEvent{
				TimeMS:    kv.TimeMS,
				IsKill:    false,
				ActorXUID: myXUID,
			})
		}
	}
	return events
}

// ---------------------------------------------------------------------------
// Team Tab
// ---------------------------------------------------------------------------

func buildTeamTabFull(
	scoreboard []domain.ScoreboardRaw,
	kvPairs []domain.KVPairRaw,
	encounters []domain.EncounterRaw,
	encounterStats []domain.EncounterStatsRaw,
	bulkMedals []domain.BulkMedalRaw,
	bulkWeapons []domain.BulkWeaponKillRaw,
	myXUID string,
	titleSlug string,
	myEnrich *domain.MatchEnrichmentRaw,
	mySkillRank *domain.SkillRankRaw,
	friendsExtras map[string]port.FriendMatchExtras,
	assetURL games.TitleAssetURLAdapter, //nolint:PLR0913 — coordinator function
) domain.MatchTeamTab {
	// Index bulk medals et weapons par XUID pour O(1). ImageURL via adapter
	// (medals = ID numérique, weapons = name_en → fichier slug).
	medalsByXUID := make(map[string][]domain.PlayerMedalRow, len(scoreboard))
	for _, m := range bulkMedals {
		var imgURL string
		if assetURL != nil {
			imgURL = assetURL.MedalImageURL(uint64(m.MedalID)) //nolint:gosec
		}
		medalsByXUID[m.XUID] = append(medalsByXUID[m.XUID], domain.PlayerMedalRow{
			MedalID:    m.MedalID,
			Count:      m.Count,
			Label:      m.Label,
			ImageURL:   imgURL,
			Difficulty: m.Difficulty,
		})
	}
	weaponsByXUID := make(map[string][]domain.PlayerWeaponKillRow, len(scoreboard))
	for _, w := range bulkWeapons {
		var imgURL string
		if assetURL != nil {
			imgURL = assetURL.WeaponImageURL(w.NameEN)
		}
		weaponsByXUID[w.XUID] = append(weaponsByXUID[w.XUID], domain.PlayerWeaponKillRow{
			WeaponID: w.WeaponID,
			Kills:    w.Kills,
			Label:    w.WeaponLabel,
			ImageURL: imgURL,
		})
	}

	extremes := analysis.ComputeMVPLVP(scoreboard)

	rows := make([]domain.MatchScoreboardRow, 0, len(scoreboard))
	for _, s := range scoreboard {
		// Combat Yield calculé pour ce joueur
		var oc, dr, dpk, dpd *float64
		if s.DamageDealt != nil && s.DamageTaken != nil {
			cy := analysis.ComputeCombatYield(s.Kills, s.Assists, *s.DamageDealt, *s.DamageTaken, s.Deaths)
			oc = &cy.OffensiveConversion
			dr = &cy.DefensiveResistance
		}
		if s.DamageDealt != nil && s.Kills > 0 {
			v := *s.DamageDealt / float64(s.Kills)
			dpk = &v
		}
		if s.DamageTaken != nil && s.Deaths > 0 {
			v := *s.DamageTaken / float64(s.Deaths)
			dpd = &v
		}

		row := domain.MatchScoreboardRow{
			XUID:                s.XUID,
			Gamertag:            s.Gamertag,
			IsBot:               s.IsBot,
			IsMe:                s.XUID == myXUID,
			IsMVP:               extremes.MVPXUID != "" && s.XUID == extremes.MVPXUID,
			IsLVP:               extremes.LVPXUID != "" && s.XUID == extremes.LVPXUID,
			Rank:                s.RankInTeam,
			Kills:               &s.Kills,
			Deaths:              &s.Deaths,
			Assists:             &s.Assists,
			KDA:                 s.KDA,
			Accuracy:            s.Accuracy,
			DamageDealt:         s.DamageDealt,
			DamageTaken:         s.DamageTaken,
			ShotsFired:          s.ShotsFired,
			ShotsHit:            s.ShotsHit,
			AvgLifeSeconds:      s.AvgLifeSeconds,
			HeadshotKills:       s.HeadshotKills,
			MaxKillingSpree:     s.MaxKillingSpree,
			GrenadeKills:        s.GrenadeKills,
			MeleeKills:          s.MeleeKills,
			PowerWeaponKills:    s.PowerWeaponKills,
			OutcomeLabel:        outcomeLabel(s.OutcomeCode),
			Score:               toIntPtr(s.PersonalScore),
			PerfectKills:        &s.PerfectKills,
			TopWeaponID:         s.TopWeaponID,
			TopWeaponLabel:      s.TopWeaponLabel,
			OffensiveConversion: oc,
			DefensiveResistance: dr,
			DamagePerKill:       dpk,
			DamagePerDeath:      dpd,
			ExpectedKills:       s.KillsExpected,
			ExpectedDeaths:      s.DeathsExpected,
			ExpectedAssists:     s.AssistsExpected,
			KillsStdDev:         s.KillsStdDev,
			DeathsStdDev:        s.DeathsStdDev,
			Medals:              medalsByXUID[s.XUID],
			WeaponKills:         weaponsByXUID[s.XUID],
		}
		if s.TeamID != nil {
			team := fmt.Sprintf("t%d", *s.TeamID)
			row.TeamSide = &team
		}
		// Section "Local" du panneau d'expander : populée différemment selon
		// que le joueur est `me` (depuis myEnrich + skillRank du contexte) ou
		// un ami (depuis friendsExtras chargés via loader per-DB).
		if row.IsMe {
			if myEnrich != nil {
				if myEnrich.PerformanceScore != nil {
					v := *myEnrich.PerformanceScore
					row.PerformanceScore = &v
				}
				// HadBotTeammate du main player : domain.MatchEnrichmentRaw
				// ne l'expose pas directement (cf. Q18 actuel). Le front lit
				// header.had_bot_teammate qui est rempli ailleurs (page card).
			}
			if mySkillRank != nil {
				row.SkillRank = &domain.MatchScoreboardSkillRank{
					RatingType:  mySkillRank.RatingType,
					TierLabel:   mySkillRank.TierLabel,
					RatingValue: mySkillRank.RatingValue,
					RatingDelta: mySkillRank.RatingDelta,
				}
			}
		} else if extras, ok := friendsExtras[s.XUID]; ok {
			row.PerformanceScore = extras.PerformanceScore
			row.HadBotTeammate = extras.HadBotTeammate
			row.SkillRank = extras.SkillRank
			if extras.AssistsModel != nil {
				dd := 0.0
				if s.DamageDealt != nil {
					dd = *s.DamageDealt
				}
				dt := 0.0
				if s.DamageTaken != nil {
					dt = *s.DamageTaken
				}
				mmrDelta := 0.0
				if s.TeamMMR != nil && s.EnemyMMR != nil {
					mmrDelta = *s.TeamMMR - *s.EnemyMMR
				}
				m := extras.AssistsModel
				raw := m.Intercept +
					m.CoefKills*float64(s.Kills) +
					m.CoefDeaths*float64(s.Deaths) +
					m.CoefDamageDealt*dd +
					m.CoefDamageTaken*dt +
					m.CoefMMRDelta*mmrDelta
				v := math.Round(raw*100) / 100
				row.ExpectedAssists = &v
			}
		}
		rows = append(rows, row)
	}

	// Nemesis depuis KV pairs
	nemesisByXUID := buildNemesisMap(kvPairs, myXUID, scoreboard)
	nemesisList := make([]domain.MatchNemesisRow, 0, len(nemesisByXUID))
	for xuid, n := range nemesisByXUID {
		nemesisList = append(nemesisList, domain.MatchNemesisRow{
			XUID:     xuid,
			Gamertag: n.Gamertag,
			KilledMe: n.KilledMe,
			IKilled:  n.IKilled,
		})
	}
	sortNemesisByKilledMe(nemesisList)

	return domain.MatchTeamTab{
		Roster:     []domain.MatchRosterRow{},
		Scoreboard: rows,
		Nemesis:    nemesisList,
		Encounters: convertEncounters(encounters, encounterStats),
	}
}

func convertEncounters(
	raw []domain.EncounterRaw,
	stats []domain.EncounterStatsRaw,
) []domain.MatchEncounterRow {
	if len(raw) == 0 {
		return []domain.MatchEncounterRow{}
	}
	// Index stats par xuid pour O(1) lookup. Optionnel : si stats nil ou
	// vide, on retombe sur le badge ordinal seul (chunk MV4.C legacy).
	statsByXUID := make(map[string]domain.EncounterStatsRaw, len(stats))
	for _, s := range stats {
		statsByXUID[s.XUID] = s
	}

	result := make([]domain.MatchEncounterRow, 0, len(raw))
	for _, e := range raw {
		s, hasStats := statsByXUID[e.XUID]
		var badges []domain.MatchEncounterBadge
		if hasStats {
			badges = buildEncounterBadgesFromStats(e, s)
		} else {
			// MV4.C fallback : seul le badge ordinal attribuable.
			badges = buildEncounterBadgesFromRaw(e)
		}
		row := domain.MatchEncounterRow{
			XUID:          e.XUID,
			Gamertag:      e.Gamertag,
			IsBot:         e.IsBot,
			CountTogether: e.CountTogether,
			IsAlly:        e.IsAlly,
			Badges:        badges,
		}
		if hasStats {
			ally, enemy := s.AllyCount, s.EnemyCount
			kills, deaths := s.KillsDealt, s.DeathsSuffered
			row.AllyCount = &ally
			row.EnemyCount = &enemy
			row.KillsDealt = &kills
			row.DeathsSuffered = &deaths
			row.WinrateAsAlly = encounterWinrate(s.WinsAsAlly, s.LossesAsAlly)
			row.WinrateVsEnemy = encounterWinrate(s.WinsVsEnemy, s.LossesVsEnemy)
			if !s.LastSeenAt.IsZero() {
				ts := s.LastSeenAt
				row.LastSeenAt = &ts
			}
		}
		result = append(result, row)
	}
	return result
}

// buildEncounterBadgesFromRaw : fallback MV4.C — sans stats riches, seul le
// badge ordinal est attribuable.
func buildEncounterBadgesFromRaw(e domain.EncounterRaw) []domain.MatchEncounterBadge {
	stats := narrative.EncounterStats{
		XUID:            e.XUID,
		Gamertag:        e.Gamertag,
		TotalEncounters: e.CountTogether,
	}
	ordinal := e.CountTogether - 1
	if ordinal < 0 {
		ordinal = 0
	}
	return convertNarrativeBadges(narrative.ComputeEncounterBadges(stats, ordinal))
}

// buildEncounterBadgesFromStats : MV4.C' — utilise les stats riches Q23b
// pour permettre à narrative.ComputeEncounterBadges d'attribuer ally_plus
// et tough_enemy en plus d'ordinal.
func buildEncounterBadgesFromStats(
	e domain.EncounterRaw,
	s domain.EncounterStatsRaw,
) []domain.MatchEncounterBadge {
	winrateAsAlly := encounterWinrate(s.WinsAsAlly, s.LossesAsAlly)
	winrateVsEnemy := encounterWinrate(s.WinsVsEnemy, s.LossesVsEnemy)
	stats := narrative.EncounterStats{
		XUID:            e.XUID,
		Gamertag:        e.Gamertag,
		TotalEncounters: e.CountTogether,
		AllyCount:       s.AllyCount,
		EnemyCount:      s.EnemyCount,
		WinrateAsAlly:   winrateAsAlly,
		WinrateVsEnemy:  winrateVsEnemy,
		KillsDealt:      s.KillsDealt,
		DeathsSuffered:  s.DeathsSuffered,
	}
	ordinal := e.CountTogether - 1
	if ordinal < 0 {
		ordinal = 0
	}
	return convertNarrativeBadges(narrative.ComputeEncounterBadges(stats, ordinal))
}

// encounterWinrate : nil si W+L == 0 (pas assez de matchs pour calculer),
// sinon ratio. narrative.ComputeEncounterBadges traite nil comme "pas
// d'attribution ally_plus".
func encounterWinrate(wins, losses int) *float64 {
	total := wins + losses
	if total == 0 {
		return nil
	}
	rate := analysis.WinRate(wins, total)
	return &rate
}

// convertNarrativeBadges : narrative.EncounterBadge -> domain.MatchEncounterBadge.
func convertNarrativeBadges(raw []narrative.EncounterBadge) []domain.MatchEncounterBadge {
	if len(raw) == 0 {
		return nil
	}
	out := make([]domain.MatchEncounterBadge, 0, len(raw))
	for _, b := range raw {
		out = append(out, domain.MatchEncounterBadge{
			Kind:       string(b.Kind),
			LabelKey:   b.LabelKey,
			ColorToken: b.ColorToken,
			Detail:     b.Detail,
		})
	}
	return out
}

// buildMediaTab construit l'onglet médias.
func buildMediaTab(media []domain.MediaAssocRaw) domain.MatchMediaTab {
	if len(media) == 0 {
		return domain.MatchMediaTab{MediaItems: []domain.MatchAssociatedMedia{}}
	}
	items := make([]domain.MatchAssociatedMedia, 0, len(media))
	for _, m := range media {
		items = append(items, domain.MatchAssociatedMedia{
			FileID:       m.FileID,
			FileName:     m.FileName,
			FilePath:     m.FilePath,
			ThumbnailURL: m.ThumbnailPath,
			CaptureTime:  m.CaptureTime,
			Liked:        m.Liked,
		})
	}
	return domain.MatchMediaTab{MediaItems: items}
}

type nemesisEntry struct {
	Gamertag string
	KilledMe int
	IKilled  int
}

func buildNemesisMap(
	kvPairs []domain.KVPairRaw,
	myXUID string,
	scoreboard []domain.ScoreboardRaw,
) map[string]*nemesisEntry {
	gtMap := make(map[string]string, len(scoreboard))
	for _, s := range scoreboard {
		gtMap[s.XUID] = s.Gamertag
	}

	result := make(map[string]*nemesisEntry)
	for _, kv := range kvPairs {
		if kv.VictimXUID == myXUID {
			if _, ok := result[kv.KillerXUID]; !ok {
				gt := gtMap[kv.KillerXUID]
				if gt == "" {
					gt = kv.KillerGT
				}
				result[kv.KillerXUID] = &nemesisEntry{Gamertag: gt}
			}
			result[kv.KillerXUID].KilledMe += kv.KillCount
		}
		if kv.KillerXUID == myXUID {
			if _, ok := result[kv.VictimXUID]; !ok {
				gt := gtMap[kv.VictimXUID]
				if gt == "" {
					gt = kv.VictimGT
				}
				result[kv.VictimXUID] = &nemesisEntry{Gamertag: gt}
			}
			result[kv.VictimXUID].IKilled += kv.KillCount
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
// outcomeLabel est défini dans match_history_service.go (même package).

func outcomeColor(code int) string {
	if c, ok := outcomeColors[code]; ok {
		return c
	}
	return mvHexOutcomeUnknown
}

func perfColor(score float64) string {
	switch {
	case score >= 80:
		return mvHexOutcomeWin
	case score >= 60:
		return mvHexPerfMedium
	case score >= 40:
		return mvHexPerfLow
	default:
		return mvHexOutcomeLoss
	}
}

// formatDateFRLong formate une date en "JJ mois AAAA, HH:MM" (FR long).
// Distinct de formatDateFR (match_history) qui utilise le format court.
func formatDateFRLong(t time.Time) string {
	months := [...]string{
		"janv.", "févr.", "mars", "avr.", "mai", "juin",
		"juil.", "août", "sept.", "oct.", "nov.", "déc.",
	}
	local := t.Local()
	return fmt.Sprintf("%02d %s %d, %02d:%02d",
		local.Day(), months[local.Month()-1], local.Year(),
		local.Hour(), local.Minute())
}

func sortInts(s []int) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func sortNemesisByKilledMe(s []domain.MatchNemesisRow) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].KilledMe > s[j-1].KilledMe; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// formatLifeSeconds est défini dans match_history_service.go (même package).
