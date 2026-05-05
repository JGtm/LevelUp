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
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"

	"golang.org/x/sync/errgroup"
)

// outcomeColors : couleur hex par code d'outcome Halo Infinite.
//
// Deprecated: anti-pattern (CLAUDE.md règle 20 — aucun hex côté backend).
// Conservé pour rétrocompat avec les consommateurs front V0 qui n'ont pas
// encore migré vers tokenCssVar(). Utiliser outcomeColorToken pour les
// nouveaux champs (Phase 1 méta-plan § 6.1.3 — chunk MV3 cleanup).
//
// (outcomeLabels est défini dans match_history_service.go)
var outcomeColors = map[int]string{
	1: "#8b5cf6", // Égalité
	2: "#22c55e", // Victoire
	3: "#ef4444", // Défaite
	4: "#8b5cf6", // Non terminé
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
		weapons        []domain.WeaponKillRaw
		kvPairs        []domain.KVPairRaw
		skillRank      *domain.SkillRankRaw
		encounters     []domain.EncounterRaw
		media          []domain.MediaAssocRaw
		expected       *domain.ExpectedStatsRaw
		bulkMedals     []domain.BulkMedalRaw
		bulkWeapons    []domain.BulkWeaponKillRaw
		matchCitations []domain.CitationMatchViewRow
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
		weapons, e = s.repo.GetMatchWeaponKills(gctx, s.xuid, matchID)
		if e != nil {
			slog.Warn("match_view: weapons indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
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
		// playerSlug = xuid (identifiant de stockage dans shared_social)
		media, e = s.repo.GetMatchMedia(gctx, matchID, s.xuid)
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
	if s.citationsRepo != nil {
		g.Go(func() error {
			var e error
			matchCitations, e = s.citationsRepo.LoadMatchCitationsForView(gctx, matchID)
			if e != nil {
				slog.Warn("match_view: citations indisponibles", "match_id", matchID, "err", e)
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

	header := buildMatchHeader(ctx, matchID, meta, stats, enrich, scoreboard, s.assetURL, isFavorite)
	rank := buildRankBlock(skillRank, s.assetURL)
	summary := buildSummaryTabFull(stats, medals, expected)
	combat := buildCombatTabFull(matchID, weapons, events, canonicalEvents, kvPairs, scoreboard, s.xuid, durationMS)
	team := buildTeamTabFull(scoreboard, kvPairs, encounters, encounterStats, bulkMedals, bulkWeapons, s.xuid)
	mediaTab := buildMediaTab(media)

	// MV4.B' : radar 6 axes pour TOUS les xuids du scoreboard (pas juste le main).
	// Charge les awards en série après le scoreboard. Latence supplémentaire
	// acceptable (~50-100ms pour 1 query DuckDB sur l'index match_id+xuid).
	rawAwards := s.loadAwardsForScoreboard(ctx, matchID, scoreboard)
	var radar []any
	if len(rawAwards) > 0 {
		modeFamily := matchModeFamilyFromMeta(meta)
		series := BuildMatchRadar(rawAwards, scoreboard, modeFamily)
		for _, s := range series {
			radar = append(radar, s)
		}
	}

	return domain.MatchViewResponse{
		Header:       header,
		Rank:         rank,
		SummaryTab:   summary,
		CombatTab:    combat,
		TeamTab:      team,
		MediaTab:     mediaTab,
		CitationsTab: buildCitationsTab(matchCitations, medals),
		Radar:        radar,
	}, nil
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
		OutcomeColor: "#94a3b8",
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
	if meta.ModeNameFR != nil && *meta.ModeNameFR != "" {
		h.ModeUI = *meta.ModeNameFR
	} else if meta.PairName != nil {
		h.ModeUI = *meta.PairName
	}
	// Playlist : priorité à la traduction FR (asset_translations), fallback
	// nom brut EN (match_registry.playlist_name).
	if meta.PlaylistNameFR != nil && *meta.PlaylistNameFR != "" {
		h.PlaylistLabel = *meta.PlaylistNameFR
	} else if meta.PlaylistName != nil {
		h.PlaylistLabel = *meta.PlaylistName
	}
	// MapImageURL : résolu via TitleAssetURLAdapter à partir du nom EN brut
	// (l'adapter Halo Infinite mappe nameEN → /static/maps/halo_infinite/{name}.png).
	// Dégradation gracieuse : nil si adapter absent ou nameEN inconnu.
	if assetURL != nil && meta.MapName != nil && *meta.MapName != "" {
		if url := assetURL.MapImageURL(*meta.MapName); url != "" {
			h.MapImageURL = &url
		} else {
			slog.WarnContext(ctx, "match_header: map image missing for known map",
				"match_id", matchID, "map_name", *meta.MapName)
		}
	}
	h.PlayableDurationSeconds = meta.PlayableDurationSeconds
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
		if enrich.PerformanceScore != nil {
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

	// Badge image — uniquement pour CSR (LUSR n'a pas de badge officiel).
	// Onyx : pas de sub-tier → CSRRankImageURLOnyx().
	// Autres tiers (Bronze, Silver, Gold, Platinum, Diamond) : tier + sub-tier.
	// Sources : match_skill_rank.tier (EN, TitleCase) + match_skill_rank.sub_tier.
	if assetURL == nil || sr.RatingType != "CSR" || sr.Tier == nil || *sr.Tier == "" {
		return rank
	}
	tier := *sr.Tier
	if strings.EqualFold(tier, "Onyx") {
		rank.IconURL = assetURL.CSRRankImageURLOnyx()
	} else if sr.SubTier != nil {
		rank.IconURL = assetURL.CSRRankImageURL(tier, *sr.SubTier)
	}
	return rank
}

// ---------------------------------------------------------------------------
// Summary Tab
// ---------------------------------------------------------------------------

func buildSummaryTabFull(stats *domain.PlayerMatchStatsRaw, medals []domain.MedalRaw, expected *domain.ExpectedStatsRaw) domain.MatchSummaryTab {
	tab := domain.MatchSummaryTab{
		KPIs:           domain.MatchSummaryKpis{},
		PersonalResult: domain.MatchPersonalResult{OutcomeLabel: "-", OutcomeColor: "#94a3b8"},
		Medals:         convertMedals(medals),
		Citations:      []domain.MatchCitation{},
		ExpectedStats:  buildExpectedStats(expected),
	}

	if stats == nil {
		return tab
	}

	tab.KPIs = domain.MatchSummaryKpis{
		Kills:         &stats.Kills,
		Deaths:        &stats.Deaths,
		Assists:       &stats.Assists,
		KDA:           stats.KDA,
		DamageDealt:   stats.DamageDealt,
		AverageLife:   formatLifeSeconds(stats.AvgLifeSeconds),
		Accuracy:      stats.Accuracy,
		PersonalScore: toIntPtr(stats.PersonalScore),
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

// buildExpectedStats construit le bloc de stats attendues.
func buildExpectedStats(e *domain.ExpectedStatsRaw) domain.MatchExpectedStats {
	if e == nil {
		return domain.MatchExpectedStats{}
	}
	return domain.MatchExpectedStats{
		HasExpectedData: e.KillsExpected != nil || e.DeathsExpected != nil,
		ExpectedKills:   e.KillsExpected,
		ExpectedDeaths:  e.DeathsExpected,
		ExpectedAssists: e.AssistsExpected,
		HasHistAvg:      false, // calculé séparément via ComputeModeCategoryAverages si historique disponible
	}
}

func toIntPtr(f *float64) *int {
	if f == nil {
		return nil
	}
	v := int(math.Round(*f))
	return &v
}

func convertMedals(raw []domain.MedalRaw) []domain.MatchMedal {
	if len(raw) == 0 {
		return []domain.MatchMedal{}
	}
	medals := make([]domain.MatchMedal, 0, len(raw))
	for _, r := range raw {
		medals = append(medals, domain.MatchMedal{
			MedalNameID: r.MedalID,
			Name:        r.Label,
			Count:       r.Count,
		})
	}
	return medals
}

// buildCitationsTab construit l'onglet Citations depuis les données chargées.
func buildCitationsTab(citations []domain.CitationMatchViewRow, medals []domain.MedalRaw) domain.MatchCitationsTab {
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
		Medals:        convertMedals(medals),
	}
}

// ---------------------------------------------------------------------------
// Combat Tab
// ---------------------------------------------------------------------------

func buildCombatTabFull(
	matchID string,
	weapons []domain.WeaponKillRaw,
	events []domain.EventRaw,
	canonicalEvents []canonical.HighlightEvent,
	kvPairs []domain.KVPairRaw,
	scoreboard []domain.ScoreboardRaw,
	myXUID string,
	durationMS int64,
) domain.MatchCombatTab {
	wkList := make([]domain.MatchWeaponKill, 0, len(weapons))
	for _, w := range weapons {
		wkList = append(wkList, domain.MatchWeaponKill{
			WeaponID:    w.WeaponID,
			WeaponLabel: w.WeaponLabel,
			KillCount:   w.Kills,
		})
	}

	evtList := make([]domain.MatchHighlightEvent, 0, len(events))
	for _, e := range events {
		evtList = append(evtList, domain.MatchHighlightEvent{
			EventType:   e.EventType,
			EventTimeMS: e.TimeMS,
			ActorXUID:   e.XUID,
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
			EventType: "kill",
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
) domain.MatchTeamTab {
	// Index bulk medals et weapons par XUID pour O(1).
	medalsByXUID := make(map[string][]domain.PlayerMedalRow, len(scoreboard))
	for _, m := range bulkMedals {
		medalsByXUID[m.XUID] = append(medalsByXUID[m.XUID], domain.PlayerMedalRow{
			MedalID: m.MedalID,
			Count:   m.Count,
			Label:   m.Label,
		})
	}
	weaponsByXUID := make(map[string][]domain.PlayerWeaponKillRow, len(scoreboard))
	for _, w := range bulkWeapons {
		weaponsByXUID[w.XUID] = append(weaponsByXUID[w.XUID], domain.PlayerWeaponKillRow{
			WeaponID: w.WeaponID,
			Kills:    w.Kills,
			Label:    w.WeaponLabel,
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
			AssistsStdDev:       s.AssistsStdDev,
			Medals:              medalsByXUID[s.XUID],
			WeaponKills:         weaponsByXUID[s.XUID],
		}
		if s.TeamID != nil {
			team := fmt.Sprintf("t%d", *s.TeamID)
			row.TeamSide = &team
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
		result = append(result, domain.MatchEncounterRow{
			XUID:          e.XUID,
			Gamertag:      e.Gamertag,
			CountTogether: e.CountTogether,
			IsAlly:        e.IsAlly,
			Badges:        badges,
		})
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
	return "#94a3b8"
}

func perfColor(score float64) string {
	switch {
	case score >= 80:
		return "#22c55e"
	case score >= 60:
		return "#3b82f6"
	case score >= 40:
		return "#f59e0b"
	default:
		return "#ef4444"
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
