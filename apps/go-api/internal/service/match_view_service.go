// Package service — MatchViewService : vue complète d'un match.
//
// Port Go de apps/api/app/services/match_view_service.py.
// Assemble les données des 4 onglets + header à partir du repo.
//
// Le fichier joue le rôle de façade : il expose la struct MatchViewService,
// son constructeur et les méthodes publiques (GetMatchView, GetMatchNeighbors).
//
// Découpage thématique (audit #1 god files) :
//   - match_view_data_loaders.go : matchViewData + loadMatchViewDataParallel +
//     buildMatchViewFromData + detectPartialMatchData + matchModeFamilyFromMeta +
//     loadAwardsForScoreboard.
//   - match_view_builders_header.go  : buildMatchHeader + buildScoreLabelFromMeta + buildRankBlock.
//   - match_view_builders_summary.go : buildSummaryTabFull + computeExpectedAssists + buildExpectedStats + convertMedals + buildCitationsTab + toIntPtr.
//   - match_view_builders_combat.go  : buildCombatTabFull + buildKillerVictimPairs + buildTugEvents + buildImpactInput + buildKDEvents.
//   - match_view_builders_team.go    : buildTeamTabFull + convertEncounters + buildEncounterBadges* + buildMediaTab + buildNemesisMap.
//   - match_view_converters.go       : convertTugBinsToDomain + convertImpactBadgesToDomain + convertKDPointsToDomain + computeScoreboardRowCombatYield + indexBulk{Medals,Weapons}ByXUID.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
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
	// la volée) APRÈS correction T0 (vrai début de match). Dégradation gracieuse
	// si nil : on retombe sur GetMatchEvents (horloge film, T0 non appliqué).
	//
	// Type = highlightEventsLoader (même seam prouvé que Timeseries) : la PlayerDB
	// fixe déjà le titre, donc Load(filters) suffit — pas besoin de la signature
	// multi-titres LoadHighlightEvents(slug, filters). Le port aspirationnel
	// HighlightEventsRepository n'a aucun implémenteur de prod (LoadHighlightEvents
	// stub sur le DataAdapter, pas d'InvalidateMatch côté duckdb).
	eventsRepo highlightEventsLoader
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
	// viewerGamertag (optionnel) : gamertag du joueur de la page. Posé dans le
	// ctx (ctxkeys.WithViewerGamertag) avant l'appel adapter de la voie canonique
	// (LoadMatchDetail), indispensable aux titres GAMERTAG-keyés (Halo 5) dont
	// l'API match ne porte aucun xuid. Vide pour les titres xuid-keyés (HINF) :
	// la voie repo ne le consulte jamais. Injecté via WithViewerGamertag au boot.
	viewerGamertag string
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

// WithViewerGamertag configure le gamertag du joueur de la page. Posé dans le
// ctx avant l'appel LoadMatchDetail de la voie canonique (titres GAMERTAG-keyés,
// cf. ctxkeys.WithViewerGamertag). Dégradation gracieuse si "" : la voie repo
// (HINF) n'en a pas besoin et la voie canonique dégrade proprement sans viewer.
func (s *MatchViewService) WithViewerGamertag(gamertag string) *MatchViewService {
	s.viewerGamertag = gamertag
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
// des canonical.HighlightEvent (T0-corrigés) au lieu de convertir des EventRaw
// à la volée. Mêmes repo et seam que Timeseries (duckdb.HighlightEventsRepo).
//
// Dégradation gracieuse : si nil, le service retombe sur repo.GetMatchEvents
// (Q21 legacy) + conversion EventRaw → canonical.HighlightEvent (chunk MV2),
// SANS correction T0 (cadence/rôles sur l'horloge film, countdown inclus).
func (s *MatchViewService) WithHighlightEventsRepo(r highlightEventsLoader) *MatchViewService {
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
// Le flux est en 3 étapes (helpers extraits dans match_view_data_loaders.go) :
//  1. meta bloquant (Q11) — indispensable à la suite.
//  2. loadMatchViewDataParallel — errgroup ~20 chargements concurrents.
//  3. buildMatchViewFromData — assemblage séquentiel des builders.
func (s *MatchViewService) GetMatchView(ctx context.Context, matchID string) (domain.MatchViewResponse, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("match_view_get", time.Since(start).Milliseconds())
	}(time.Now())
	// --- Appels séquentiels bloquants (meta est nécessaire pour la suite) ---
	meta, err := s.repo.GetMatchMeta(ctx, matchID)
	if err != nil {
		// Routage repo-first / adapter-fallback (title-agnostic). Le repo ne sait
		// pas servir ce match (HINF : substrat absent → erreur ; h5 live-only :
		// PAS de DuckDB → GetMatchMeta erreur "no rows"). Si un DataAdapter est
		// câblé, on tente la voie canonique LIVE (LoadMatchDetail) avant de
		// remonter l'erreur d'origine. HINF garde la voie repo par défaut : son
		// substrat est présent → meta OK → on ne tombe jamais ici, donc AUCUNE
		// régression (byte-identique). Aucune comparaison de slug : le routage est
		// piloté par « le repo peut-il servir ce match ».
		if resp, ok := s.tryCanonicalMatchView(ctx, matchID); ok {
			return resp, nil
		}
		return domain.MatchViewResponse{}, fmt.Errorf("MatchViewService: meta: %w", err)
	}

	// Couche B (ADR 0029) : fail-fast si le joueur courant n'a pas participé à ce
	// match — avant les ~20 chargements parallèles. Évite la page "mal renseignée"
	// (stats perso vides + scoreboard d'autrui). Best-effort : une erreur DB ne
	// bloque pas (on ne veut pas masquer un match légitime sur incident transitoire).
	if err := s.assertParticipation(ctx, matchID); err != nil {
		return domain.MatchViewResponse{}, err
	}

	data, err := s.loadMatchViewDataParallel(ctx, matchID)
	if err != nil {
		return domain.MatchViewResponse{}, err
	}

	return s.buildMatchViewFromData(ctx, matchID, meta, data), nil
}

// tryCanonicalMatchView tente la voie canonique LIVE (DataAdapter.LoadMatchDetail)
// quand le repo ne sait pas servir le match (cf. routage repo-first dans
// GetMatchView). Retourne (resp, true) en cas de succès, (zéro, false) si aucun
// adapter n'est câblé OU si l'adapter dégrade (ErrCapabilityNotSupported / nil /
// erreur) — le caller remonte alors l'erreur d'origine (comportement actuel).
//
// Le gamertag du viewer est posé dans le ctx (ctxkeys.WithViewerGamertag) pour les
// titres GAMERTAG-keyés (Halo 5) dont LoadMatchDetail en a besoin pour retrouver
// l'entrée d'historique du match. No-op pour un titre xuid-keyé. Si le câblage
// (registry) a déjà posé la clé, WithViewerGamertag la ré-affirme avec la même
// valeur (idempotent) ; quand s.viewerGamertag est vide, on ne touche pas le ctx.
func (s *MatchViewService) tryCanonicalMatchView(ctx context.Context, matchID string) (domain.MatchViewResponse, bool) {
	if s.dataAdapter == nil {
		return domain.MatchViewResponse{}, false
	}
	if s.viewerGamertag != "" {
		ctx = ctxkeys.WithViewerGamertag(ctx, s.viewerGamertag)
	}
	detail, err := s.dataAdapter.LoadMatchDetail(ctx, matchID)
	if err != nil || detail == nil {
		// Dégradation gracieuse : token absent, match introuvable live, capability
		// non supportée. Le caller retombe sur l'erreur repo d'origine (404/500).
		if err != nil && !errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.WarnContext(ctx, "match_view: voie canonique échouée (fallback erreur repo)",
				"match_id", matchID, "err", err)
		}
		return domain.MatchViewResponse{}, false
	}
	// Enrichissement i18n des refs d'assets (map/playlist/game_variant) AVANT la
	// projection : la voie LIVE porte des AssetReference brutes (ID seul, Labels
	// nil) — sans ça le header affiche un mode/carte/playlist vide. Pendant LIVE
	// de HomeRepo.EnrichCanonicalAssetTranslations (voie home/DB). No-op si la
	// table asset_translations n'est pas peuplée (refs gardent leur DefaultLabel).
	s.enrichCanonicalDetailTranslations(ctx, detail)
	return s.buildMatchViewFromCanonical(ctx, detail), true
}

// participantChecker est une capability OPTIONNELLE du repo : vérifie l'existence
// du joueur dans match_participants. Seul l'adapter DuckDB réel l'implémente ; les
// mocks de test ne l'implémentent pas → le gating est alors gracieusement ignoré.
type participantChecker interface {
	IsParticipant(ctx context.Context, xuid, matchID string) (bool, error)
}

// assertParticipation renvoie domain.APIError{Code:"match_not_participant"} si le
// joueur courant n'a pas participé au match. No-op si le repo ne fournit pas la
// capability, si le xuid est inconnu, ou en cas d'erreur DB (best-effort).
func (s *MatchViewService) assertParticipation(ctx context.Context, matchID string) error {
	pc, ok := s.repo.(participantChecker)
	if !ok || s.xuid == "" {
		return nil
	}
	participated, err := pc.IsParticipant(ctx, s.xuid, matchID)
	if err != nil {
		slog.WarnContext(ctx, "match_view: vérification participation échouée",
			"match_id", matchID, "err", err)
		return nil
	}
	if !participated {
		slog.InfoContext(ctx, "match_view: accès match non-participé refusé",
			"match_id", matchID, "xuid", s.xuid)
		return &domain.APIError{
			Code:    "match_not_participant",
			Message: "vous n'avez pas participé à ce match",
		}
	}
	return nil
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
// Helpers transverses
// ---------------------------------------------------------------------------
// outcomeLabel et formatLifeSeconds sont définis dans match_history_service.go
// (même package).

// Phase 1 méta-plan § 6.1.3 — chunk MV3 cleanup hex codes.
// Les helpers outcomeColor et perfColor restent pour rétrocompat front V0 ;
// utiliser outcomeColorToken / perfColorToken pour les nouveaux champs.

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
	return formatDateLong(t, "fr")
}

// formatDateLong formate une date en "JJ mois AAAA, HH:MM", locale-aware (mois
// abrégés FR ou EN). Le header Match View figeait la date en FR sous UI EN (GH-9).
// Sortie FR strictement identique à l'ancienne formatDateFRLong.
func formatDateLong(t time.Time, locale string) string {
	monthsFR := [...]string{
		"janv.", "févr.", "mars", "avr.", "mai", "juin",
		"juil.", "août", "sept.", "oct.", "nov.", "déc.",
	}
	monthsEN := [...]string{
		"Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
	}
	months := monthsFR
	if locale == "en" {
		months = monthsEN
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
