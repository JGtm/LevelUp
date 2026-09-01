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
//   - match_view_builders_team.go    : buildTeamTabFull + convertEncounters + convertRelationBadges + buildMediaTab + buildNemesisMap.
//   - match_view_converters.go       : convertTugBinsToDomain + convertImpactBadgesToDomain + convertKDPointsToDomain + computeScoreboardRowCombatYield + indexBulk{Medals,Weapons}ByXUID.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/positions"
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
	// assetURL (optionnel) : adapter d'URLs d'assets (image map, badge rang).
	// Injecté via WithAssetURL au boot. Si nil, MapImageURL et IconURL restent
	// vides — le front affiche les fallbacks texte (dégradation gracieuse).
	assetURL games.TitleAssetURLAdapter
	// modeTaxonomy (optionnel) : résout pair_name → catégorie custom de mode pour
	// MatchViewHeader.ModeCategory. Injecté via WithModeTaxonomy avec la MÊME
	// taxonomie que MediaRepo (wire.haloInfiniteModeTaxonomy — une seule résolution
	// par titre). Zéro-value → ModeCategory reste vide (dégradation gracieuse).
	modeTaxonomy analysis.ModeTaxonomy
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
	// objectiveEventsRepo (optionnel) : loader des events objectif v3 (timeline
	// mode-agnostique) pour l'endpoint /objective-events. Si nil,
	// GetObjectiveEvents retourne games.ErrCapabilityNotSupported (le titre
	// n'expose pas la timeline objectif).
	objectiveEventsRepo port.ObjectiveEventsRepository
	// playerPositionsRepo (optionnel) : loader des positions joueurs keyframe v3
	// (match-level, §N) pour l'endpoint /positions. Si nil, GetMatchPositions
	// retourne games.ErrCapabilityNotSupported (le titre n'expose pas le film).
	playerPositionsRepo port.PlayerPositionsRepository
	// regulationSeconds (optionnel) : temps réglementaire par game_variant_name
	// du TITRE COURANT (regulation.toml, chargé au boot comme les autres
	// mappings). Nil/vide → le header n'expose jamais le flag « Prolongation »
	// (titre sans mesure, ex. Halo 5). Jamais de comparaison de slug ici : la
	// table injectée EST le titre.
	regulationSeconds map[string]int
	// roundsDecide (optionnel) : game_variant_name → le RÉSULTAT du match se lit en
	// MANCHES et non en points (regulation.toml [rounds_decide]). Nil/absent → l'en-tête
	// affiche le score de l'API, comportement d'avant le 2026-08-29.
	roundsDecide map[string]bool
	// killDistanceRepo (optionnel) : loader « distance par arme, par joueur »
	// (POC LOT G.3, plan retours-utilisateur §3bis DEC-8). Nil, ou titre sans
	// capability film.kill_source => pas de bloc, jamais d'erreur. Dégradation
	// gracieuse posée au câblage (même gate, cf. wire.killDistanceRepoFor).
	killDistanceRepo port.KillDistanceRepository
	// replaySvc (optionnel) : service du rejeu 2D, interrogé UNIQUEMENT pour la
	// présence de l'artefact (IsAvailable = un os.Stat). Nil → ReplayAvailable
	// reste faux et le front ne pose aucun lien : un titre qui ne produit pas de
	// rejeu n'a rien à afficher, pas une erreur à remonter.
	replaySvc port.ReplayService
}

// NewMatchViewService crée un MatchViewService.
func NewMatchViewService(repo port.MatchViewRepository, xuid string) *MatchViewService {
	return &MatchViewService{repo: repo, xuid: xuid}
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

// WithObjectiveEventsRepo injecte le loader des events objectif v3 (timeline
// mode-agnostique) consommé par GetObjectiveEvents. Dégradation gracieuse si
// nil : GetObjectiveEvents retourne games.ErrCapabilityNotSupported.
func (s *MatchViewService) WithObjectiveEventsRepo(r port.ObjectiveEventsRepository) *MatchViewService {
	s.objectiveEventsRepo = r
	return s
}

// WithReplay injecte le service de rejeu 2D pour publier `replay_available` dans le
// header. SEULE IsAvailable est appelée ici (un os.Stat) : la Match View ne charge
// jamais l'artefact. Dégradation gracieuse si nil (pas de lien côté front).
func (s *MatchViewService) WithReplay(svc port.ReplayService) *MatchViewService {
	s.replaySvc = svc
	return s
}

// WithKillDistanceRepo injecte le loader « distance par arme, par joueur »
// (POC LOT G.3, plan retours-utilisateur §3bis DEC-8).
//
// Dégradation gracieuse si nil ou si le titre n'a pas la capability : le bloc
// combat_tab.kill_distance_by_weapon reste absent, exactement comme avant ce lot.
func (s *MatchViewService) WithKillDistanceRepo(r port.KillDistanceRepository) *MatchViewService {
	s.killDistanceRepo = r
	return s
}

// WithPlayerPositionsRepo injecte le loader des positions joueurs keyframe v3
// (match-level, §N) consommé par GetMatchPositions. Dégradation gracieuse si
// nil : GetMatchPositions retourne games.ErrCapabilityNotSupported.
func (s *MatchViewService) WithPlayerPositionsRepo(r port.PlayerPositionsRepository) *MatchViewService {
	s.playerPositionsRepo = r
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

// WithRoundsDecide injecte la table `game_variant_name → le résultat se lit en MANCHES`
// (regulation.toml [rounds_decide]). Sans injection, l'en-tête affiche le score de l'API —
// le comportement d'avant le 2026-08-29, jamais une régression.
func (s *MatchViewService) WithRoundsDecide(roundsDecide map[string]bool) *MatchViewService {
	s.roundsDecide = roundsDecide
	return s
}

// WithRegulation injecte la table `game_variant_name → temps réglementaire`
// (secondes) du titre courant, socle du flag « Prolongation ». Sans injection
// (ou table vide), le header ne porte jamais le flag — dégradation sûre.
func (s *MatchViewService) WithRegulation(regulationSeconds map[string]int) *MatchViewService {
	s.regulationSeconds = regulationSeconds
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
		// L'ABSENCE ET LA PANNE NE SONT PAS LE MÊME 404 (correctif 2026-08-29).
		//
		// Absence (sql.ErrNoRows, préservé par le wrapping %w du repo) : match jamais
		// synchronisé — 404 propre et typé, title-agnostic. AUCUN fetch live vers l'API
		// du titre depuis cette page : décision user 2026-07-19 (BACKLOG "Retirer le
		// fallback LIVE du Match view") — cette décision porte sur le refus du fetch,
		// PAS sur le mapping des erreurs. Le front affiche « pas encore synchronisé »
		// sur ce code (match_not_found).
		if errors.Is(err, sql.ErrNoRows) {
			slog.InfoContext(ctx, "match_view: match absent du substrat local (pas encore synchronisé)",
				"match_id", matchID, "err", err)
			return domain.MatchViewResponse{}, domain.ErrNotFound("match", matchID)
		}
		// Panne technique (schéma en retard, timeout, verrou, I/O) : la déguiser en
		// « pas encore synchronisé » a masqué pendant des heures une panne TOTALE de la
		// page (2026-08-29 : Binder Error sur snapshot au schéma en retard → 404 sur
		// TOUS les matchs, log en Info que personne ne lit). Une panne se dit : 500,
		// log ERROR, et l'état d'erreur générique du front.
		slog.ErrorContext(ctx, "match_view: lecture des métadonnées en échec (pas une absence)",
			"match_id", matchID, "err", err)
		return domain.MatchViewResponse{}, fmt.Errorf("match_view: métadonnées illisibles pour %s: %w", matchID, err)
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

// GetObjectiveEvents retourne les events objectif v3 (timeline mode-agnostique)
// d'un match. Si le repo n'est pas câblé (titre sans capability film, ou wiring
// non injecté en test), retourne games.ErrCapabilityNotSupported. Sinon délègue
// à LoadMatch et propage l'erreur telle quelle (y compris
// ErrCapabilityNotSupported remontée par le repo si les tables sont absentes).
func (s *MatchViewService) GetObjectiveEvents(ctx context.Context, matchID string) ([]domain.ObjectiveEvent, error) {
	if s.objectiveEventsRepo == nil {
		return nil, games.ErrCapabilityNotSupported
	}
	return s.objectiveEventsRepo.LoadMatch(ctx, matchID)
}

// GetMatchPositions retourne les positions joueurs keyframe v3 (match-level, §N)
// d'un match. Si le repo n'est pas câblé (titre sans capability film, ou wiring
// non injecté en test), retourne games.ErrCapabilityNotSupported. Sinon délègue
// à LoadMatch et propage l'erreur telle quelle (y compris
// ErrCapabilityNotSupported remontée par le repo si la table est absente).
func (s *MatchViewService) GetMatchPositions(ctx context.Context, matchID string) ([]positions.PlayerPosition, error) {
	if s.playerPositionsRepo == nil {
		return nil, games.ErrCapabilityNotSupported
	}
	return s.playerPositionsRepo.LoadMatch(ctx, matchID)
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
