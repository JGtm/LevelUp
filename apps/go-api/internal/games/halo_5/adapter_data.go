// Package halo_5 — adapter_data.go : implementation games.TitleDataAdapter pour
// Halo 5, adossee au CLIENT LIVE (pas de DuckDB en Phase 1 read-only).
//
// Divergence majeure vs Halo Infinite : l'identite joueur est le GAMERTAG (l'API
// h5 ne fournit jamais de xuid). Le parametre `xuid` des methodes Load* est donc
// interprete comme le GAMERTAG cote Halo 5 (la resolution player -> cle de titre
// est faite en amont par le wiring multi-titre).
//
// DESIGN TOKEN (active-ready) : Halo 5 est 100% live et son SpartanToken vit dans
// le CONTEXTE de requete (par joueur + par session, rotatif). L'adapter ne capture
// donc PAS un client/token fixe ; il detient une SourceFactory `ctx -> source` et
// resout le token au moment de chaque appel (cf. review Phase 1a, finding blocker).
// La factory de prod (NewSpartanTokenSource) lit ctxkeys.HaloTokens(ctx).
//
// Pas de SemanticAdapter dans ce package : Halo 5 utilise le games.Generic
// SemanticAdapter partage (le semantic adapter n'a aucune logique title-specific).
package halo_5

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/classification"
)

// h5Source est la surface live minimale consommee par l'adapter (interface ->
// mockable en tests sans reseau). *Client l'implemente.
type h5Source interface {
	GetServiceRecords(ctx context.Context, gamertag, recordType string) (*H5ServiceRecordResponse, error)
	GetPlayerMatches(ctx context.Context, gamertag string, start, count int) (*H5MatchesResponse, error)
	GetMatchCarnage(ctx context.Context, matchID, mode string) (*H5CarnageResponse, error)
	GetMatchEvents(ctx context.Context, matchID string) (*h5MatchEventsResponse, error)
}

var _ h5Source = (*Client)(nil)

// SourceFactory produit une source live h5 a partir du contexte de requete (le
// SpartanToken vit dans ctx, par joueur+session). Retourne une erreur si le token
// est absent (-> degradation gracieuse cote adapter, pas de panique).
type SourceFactory func(ctx context.Context) (h5Source, error)

// MatchHistorySource lit l'historique de matchs Halo 5 depuis le substrat LOCAL
// synchronisé (shared_matches_v2.duckdb du titre : match_registry ⨝
// match_participants) et le projette en canonical.MatchSummary. PAS de fetch live :
// la donnée y est déjà écrite par le livesync (cf. AXE A du prod-gate). C'est le
// pendant h5 du MatchHistoryRepo d'Infinite.
//
// L'identité joueur est PORTÉE par la source (fixée à la construction depuis le
// PlayerDB, comme ExplorerRepo d'Infinite) — d'où l'absence de paramètre gamertag
// dans GetMatchSummaries, conforme à la signature de TitleDataAdapter.LoadMatchSummaries.
//
// matchIDs nil/vide → les N derniers matchs du joueur (ORDER BY start_time DESC).
// matchIDs non vide → filtre sur ces IDs, ORDRE D'ENTRÉE PRÉSERVÉ (un match absent
// du shared est silencieusement omis).
//
// Interface définie côté package h5 (le consommateur) : l'implémentation DuckDB
// (internal/platform/duckdb) la satisfait STRUCTURELLEMENT, sans cycle d'import.
type MatchHistorySource interface {
	GetMatchSummaries(ctx context.Context, matchIDs []string) ([]canonical.MatchSummary, error)
}

// NewSpartanTokenSource est la SourceFactory de PRODUCTION : lit le SpartanToken du
// contexte (ctxkeys.HaloTokens) et construit un Client live h5. Erreur si pas de
// token (le caller dégrade). C'est le point de jonction wiring (Phase 1b) -> client.
func NewSpartanTokenSource(ctx context.Context) (h5Source, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil || tokens.SpartanToken == "" {
		return nil, errors.New("h5: SpartanToken absent du contexte (re-auth requise)")
	}
	return NewClient(tokens.SpartanToken, 0), nil
}

// h5RecordModeArena : seul le service record arena est consomme en Phase 1
// (warzone = PvE-like, Phase 2).
const h5RecordModeArena = "arena"

// h5RequestTimeout borne chaque appel live (defensif contre un endpoint lent).
const h5RequestTimeout = 12 * time.Second

// DataAdapter est l'implementation games.TitleDataAdapter d'Halo 5.
type DataAdapter struct {
	newSource      SourceFactory
	staticCaps     games.CapabilityMap
	placementTotal int // TitleDescriptor.PlacementMatches (0 -> defaut h5DefaultPlacementMatches)
	logger         *slog.Logger
	// classifier (optionnel) — résout le caractère classé/PvE d'un match depuis le
	// HopperId (set-membership). Quand nil, les refs IsRanked/IsPvE du header
	// MatchDetail restent INDETERMINEES (nil) — comportement conservateur identique
	// à mapMatchSummaries(nil). Injecté via WithRankedClassifier au boot/wiring.
	classifier classification.RankedClassifier
	// matchHistory (optionnel) — source de lecture de l'historique LOCAL (shared h5)
	// projeté en canonical.MatchSummary. nil → LoadMatchSummaries dégrade en
	// ErrCapabilityNotSupported. Injecté via WithMatchHistorySource au builder
	// player-scoped (porte l'identité du joueur, cf. MatchHistorySource).
	matchHistory MatchHistorySource
	// commendationDefs (optionnel) — référentiel natif (nom + icône) des commendations
	// par UUID, lu dans la metadata h5 (commendation_definitions). nil → les
	// commendations du MatchDetail restent brutes (ID + count, le front dégrade sur
	// l'ID court). Injecté via WithCommendationDefs au builder (AXE B définitions).
	commendationDefs CommendationDefSource
	// commendationTotals (optionnel) — totaux à vie par commendation (dernier progress
	// absolu) du joueur, lus dans shared.match_commendations. nil →
	// LoadCommendationTotals dégrade en ErrCapabilityNotSupported. Injecté via
	// WithCommendationTotals au builder player-scoped (AXE B totaux).
	commendationTotals CommendationTotalsSource
	// careerLocal (optionnel) — career lu depuis le DuckDB synchronisé (meilleur CSR +
	// SR). Quand injecté, LoadCareerSnapshot le PRÉFÈRE à l'API live (sert le rang
	// hors-ligne en démo, aucun token). nil → voie live inchangée. Injecté via
	// WithCareerSource au builder player-scoped (gaté DemoMode).
	careerLocal CareerLocalSource
	// eventsLocal (optionnel) — kill-feed lu depuis le DuckDB synchronisé
	// (killer_victim_pairs + weapon_kills + kill_positions). Quand injecté,
	// LoadMatchEvents le PRÉFÈRE à l'API /events live (timeline hors-ligne en démo).
	// nil → voie live inchangée. Injecté via WithMatchEventsSource (gaté DemoMode).
	eventsLocal MatchEventsLocalSource
}

var _ games.TitleDataAdapter = (*DataAdapter)(nil)

// NewDataAdapter construit l'adapter Halo 5 adosse a une source-factory.
// newSource nil -> l'adapter est inerte (toutes les capabilities live degradees a
// not_exposed, toutes les methodes -> ErrCapabilityNotSupported).
func NewDataAdapter(newSource SourceFactory, logger *slog.Logger) *DataAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &DataAdapter{newSource: newSource, logger: logger}
}

// WithCapabilities injecte la CapabilityMap chargee depuis capabilities.toml.
// nil -> fallbackCapabilities (filet de securite boot). Chainable.
func (a *DataAdapter) WithCapabilities(caps games.CapabilityMap) *DataAdapter {
	a.staticCaps = caps
	return a
}

// WithPlacementTotal injecte le nombre de matchs de placement du titre
// (TitleDescriptor.PlacementMatches). <= 0 -> defaut h5DefaultPlacementMatches au
// mapping. Chainable.
func (a *DataAdapter) WithPlacementTotal(n int) *DataAdapter {
	a.placementTotal = n
	return a
}

// WithRankedClassifier injecte le classifier ranked/PvE (set-membership HopperId)
// utilisé pour renseigner IsRanked/IsPvE des refs header de LoadMatchDetail. nil
// (défaut) -> verdicts INDETERMINES (header sans isRanked), comme l'historique
// sans classifier. Chainable.
func (a *DataAdapter) WithRankedClassifier(c classification.RankedClassifier) *DataAdapter {
	a.classifier = c
	return a
}

// WithMatchHistorySource injecte la source de lecture de l'historique LOCAL (shared
// h5 → canonical.MatchSummary) utilisée par LoadMatchSummaries. nil (défaut) ->
// LoadMatchSummaries reste dégradé (ErrCapabilityNotSupported). Chainable.
func (a *DataAdapter) WithMatchHistorySource(s MatchHistorySource) *DataAdapter {
	a.matchHistory = s
	return a
}

// WithCommendationDefs injecte le référentiel natif (nom + icône) des commendations
// utilisé pour enrichir MatchDetail.Commendations (AXE B définitions). nil (défaut)
// -> commendations laissées brutes (ID + count). Chainable.
func (a *DataAdapter) WithCommendationDefs(s CommendationDefSource) *DataAdapter {
	a.commendationDefs = s
	return a
}

// WithCommendationTotals injecte la source des totaux à vie par commendation (dernier
// progress absolu) utilisée par LoadCommendationTotals (AXE B totaux). nil (défaut)
// -> LoadCommendationTotals dégrade en ErrCapabilityNotSupported. Chainable.
func (a *DataAdapter) WithCommendationTotals(s CommendationTotalsSource) *DataAdapter {
	a.commendationTotals = s
	return a
}

// WithCareerSource injecte la source de career LOCAL (DuckDB synchronisé). Quand
// présente, LoadCareerSnapshot la PRÉFÈRE à l'API cryptum live (rang servi hors-ligne
// en démo, où aucun token n'est disponible). nil (défaut) -> voie live inchangée.
// Chainable.
func (a *DataAdapter) WithCareerSource(s CareerLocalSource) *DataAdapter {
	a.careerLocal = s
	return a
}

// WithMatchEventsSource injecte la source de kill-feed LOCAL (DuckDB synchronisé).
// Quand présente, LoadMatchEvents la PRÉFÈRE à l'API /events live (timeline servie
// hors-ligne en démo). nil (défaut) -> voie live inchangée. Chainable.
func (a *DataAdapter) WithMatchEventsSource(s MatchEventsLocalSource) *DataAdapter {
	a.eventsLocal = s
	return a
}

// TitleSlug retourne l'identite du titre (constante de package, pas un gating).
func (a *DataAdapter) TitleSlug() string { return TitleSlug }

// Capabilities decrit l'etat des capabilities Halo 5 exposees par cet adapter.
// Source nominale : capabilities.toml via WithCapabilities ; fallback code sinon.
// DEGRADATION RUNTIME : si aucune source-factory n'est cablee, l'adapter ne peut
// rien servir live -> toutes les capabilities sont rétrogradées a not_exposed (on
// ne force jamais Has()==true au-dessus de ce qui est reellement servable).
func (a *DataAdapter) Capabilities() games.CapabilityMap {
	base := a.staticCaps
	if base == nil {
		base = fallbackCapabilities()
	}
	out := make(games.CapabilityMap, len(base))
	for k, v := range base {
		if a.newSource == nil {
			out[k] = games.CapNotExposed
			continue
		}
		out[k] = v
	}
	return out
}

// fallbackCapabilities est la CapabilityMap par defaut (filet boot si capabilities.toml
// n'a pas pu etre injecte). HONNETE : seules les methodes REELLEMENT cablees sont
// exposees. career.progression = supported (LoadCareerSnapshot) ; match.detail.core =
// supported (LoadMatchDetail, carnage → canonical) ; match.history = supported
// (LoadMatchSummaries, shared h5 local → canonical, AXE A prod-gate). Le reste =
// not_exposed tant que la methode est un stub (remonte a mesure du cablage :
// scoreboard, timeseries...). Parite avec config/titles/halo_5/mappings/capabilities.toml
// (capabilities_parity_test).
func fallbackCapabilities() games.CapabilityMap {
	return games.CapabilityMap{
		games.CapMatchHistory:       games.CapSupported, // CÂBLÉ : LoadMatchSummaries (shared h5 local → canonical.MatchSummary)
		games.CapMatchDetailCore:    games.CapSupported, // CÂBLÉ : LoadMatchDetail (carnage → canonical.MatchDetail)
		games.CapScoreboardExtra:    games.CapNotExposed,
		games.CapMatchSkillSnapshot: games.CapNotExposed,
		games.CapCareerProgression:  games.CapSupported,
		games.CapTimeseries:         games.CapNotExposed,
		games.CapEngagement:         games.CapNotExposed,
		games.CapCitationsEngine:    games.CapNotExposed,
		// Commendations NATIVES par match : CÂBLÉ (AXE B). carnage
		// ProgressiveCommendationDeltas → shared.match_commendations (ingest) +
		// MatchDetail.Commendations (LoadMatchDetail). DISTINCTE de citations.engine.
		games.CapCommendationsNative: games.CapSupported,
		games.CapPveFirefight:        games.CapNotExposed,
		games.CapBattlePass:          games.CapNotExposed,
		games.CapChallenges:          games.CapNotExposed,
		// Canonical MatchEvents : CÂBLÉ Phase 1 (LoadMatchEvents → events.go).
		games.CapMatchEventsTimeline:  games.CapSupported,
		games.CapMatchKillfeedPerKill: games.CapSupported,
		games.CapMatchEventsSpatial:   games.CapSupported,
		// Précision par arme : CÂBLÉ. ShotsFired/ShotsLanded natifs des events
		// weapon_drop → table weapon_accuracy (cf. ingest/weapon_accuracy.go).
		games.CapWeaponAccuracy: games.CapSupported,
	}
}

// resolveSource resout la source live depuis le contexte (token). Retourne
// (nil, ErrCapabilityNotSupported) si pas de factory ; (nil, err) si la factory
// echoue (token absent) — le caller decide de la degradation.
func (a *DataAdapter) resolveSource(ctx context.Context) (h5Source, error) {
	if a.newSource == nil {
		return nil, games.ErrCapabilityNotSupported
	}
	return a.newSource(ctx)
}

// LoadPlayerStats projette le service record arena (live) vers PlayerStats.
// `xuid` = GAMERTAG cote Halo 5. Indisponibilite gracieuse (404/410 ou token
// expire 401/403) -> PlayerStats vide identite-seule + warn (pas une erreur dure).
func (a *DataAdapter) LoadPlayerStats(ctx context.Context, xuid string, _ canonical.StatsScope) (*canonical.PlayerStats, error) {
	gamertag := xuid
	src, err := a.resolveSource(ctx)
	if err != nil {
		return nil, games.ErrCapabilityNotSupported
	}
	ctx, cancel := context.WithTimeout(ctx, h5RequestTimeout)
	defer cancel()

	resp, err := src.GetServiceRecords(ctx, gamertag, h5RecordModeArena)
	if err != nil {
		if a.degradeUnavailable(ctx, err, gamertag, "LoadPlayerStats") {
			return &canonical.PlayerStats{Identity: h5Identity(gamertag)}, nil
		}
		return nil, fmt.Errorf("h5 LoadPlayerStats(%s): %w", gamertag, err)
	}
	if stats := aggregatePlayerStats(resp, gamertag); stats != nil {
		return stats, nil
	}
	return &canonical.PlayerStats{Identity: h5Identity(gamertag)}, nil
}

// LoadCareerSnapshot projette la carrière Halo 5 (palier CSR + rang XP « SR ») vers
// CareerSnapshot. `xuid` = GAMERTAG. Stratégie LIVE-FIRST → FALLBACK LOCAL : on tente
// d'abord le live (rang FRAIS, token du joueur) ; s'il est indisponible — token du
// joueur mort (RT révoqué), démo (aucun token), ou panne gracieuse — on sert le rang
// PERSISTÉ depuis le DuckDB synchronisé (title-agnostic, sans dépendre du token du
// joueur consulté). Ainsi la carrière d'un joueur SUIVI par l'app ne disparaît jamais
// parce que SON refresh_token est mort.
func (a *DataAdapter) LoadCareerSnapshot(ctx context.Context, xuid string, _ canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	gamertag := xuid
	if snap, ok := a.liveCareerSnapshot(ctx, gamertag); ok {
		return snap, nil
	}
	if a.careerLocal != nil {
		if snap := a.localCareerSnapshot(ctx, gamertag); snap != nil {
			return snap, nil
		}
	}
	// Aucune source exploitable : si même la voie live n'est pas câblée (builder
	// global live-only, newSource nil) ET pas de local → capability indisponible
	// (le caller masque). Sinon snapshot d'identité minimal (live câblé mais vide).
	if a.newSource == nil && a.careerLocal == nil {
		return nil, games.ErrCapabilityNotSupported
	}
	return &canonical.CareerSnapshot{Player: h5Identity(gamertag)}, nil
}

// liveCareerSnapshot tente la voie LIVE (CSR service record + SR carnage du dernier
// match). (snap, true) si une source live a répondu et produit un snapshot ;
// (nil, false) si la source/token est absente ou l'appel échoue — gracieux
// (401/403/404, signal re-auth) OU panne dure (réseau/5xx/décodage), tous deux logués
// et NON propagés : le fallback local peut couvrir. Le caller bascule alors dessus.
func (a *DataAdapter) liveCareerSnapshot(ctx context.Context, gamertag string) (*canonical.CareerSnapshot, bool) {
	src, err := a.resolveSource(ctx)
	if err != nil {
		return nil, false
	}
	rctx, cancel := context.WithTimeout(ctx, h5RequestTimeout)
	defer cancel()
	resp, err := src.GetServiceRecords(rctx, gamertag, h5RecordModeArena)
	if err != nil {
		if !a.degradeUnavailable(rctx, err, gamertag, "LoadCareerSnapshot") {
			a.logger.WarnContext(rctx, "h5 career live: échec dur (fallback local)", "player", gamertag, "err", err)
		}
		return nil, false
	}
	snap := mapCareerSnapshot(resp, gamertag, a.placementTotal)
	if snap == nil {
		return nil, false
	}
	a.enrichSpartanRank(rctx, src, gamertag, snap)
	return snap, true
}

// enrichSpartanRank ajoute le rang XP (SR) au CareerSnapshot, lu dans la carnage du
// DERNIER match (XpInfo.SpartanRank/TotalXP — seule source du SR : ni la liste de
// matchs ni le service record ne le portent). BEST-EFFORT : toute indisponibilité
// (token/404/absence/match introuvable) laisse le snapshot CSR intact, sans erreur.
func (a *DataAdapter) enrichSpartanRank(ctx context.Context, src h5Source, gamertag string, snap *canonical.CareerSnapshot) {
	matches, err := src.GetPlayerMatches(ctx, gamertag, 0, 1)
	if err != nil || matches == nil || len(matches.Results) == 0 {
		return
	}
	m := matches.Results[0]
	if m.Id.MatchId == "" {
		return
	}
	carnage, err := src.GetMatchCarnage(ctx, m.Id.MatchId, h5GameModeSegment(m.Id.GameMode))
	if err != nil || carnage == nil {
		return
	}
	for i := range carnage.PlayerStats {
		p := &carnage.PlayerStats[i]
		if p.XpInfo != nil && strings.EqualFold(p.Player.Gamertag, gamertag) {
			applySpartanRank(snap, p.XpInfo.SpartanRank, p.XpInfo.TotalXP)
			return
		}
	}
}

// degradeUnavailable retourne true (et logue) si l'erreur est une indisponibilite
// gracieuse : 404/410 (le joueur n'a pas de record sur ce mode) OU 401/403 (token
// expire/insuffisant -> signal de re-auth, pas une panne data ; un endpoint
// read-only de profil ne doit pas casser la page). Les autres erreurs (reseau,
// 5xx, decode) sont des pannes a propager.
func (a *DataAdapter) degradeUnavailable(ctx context.Context, err error, gamertag, op string) bool {
	var he *HTTPError
	if !errors.As(err, &he) {
		return false
	}
	switch he.StatusCode {
	case http.StatusNotFound, http.StatusGone:
		a.logger.DebugContext(ctx, "h5 record absent", "op", op, "player", gamertag, "status", he.StatusCode)
		return true
	case http.StatusUnauthorized, http.StatusForbidden:
		a.logger.WarnContext(ctx, "h5 token expire/insuffisant (re-auth requise)", "op", op, "player", gamertag, "status", he.StatusCode)
		return true
	}
	return false
}

// LoadMatchSummaries projette l'historique de matchs Halo 5 vers []MatchSummary
// depuis le substrat LOCAL synchronisé (shared_matches_v2.duckdb du titre :
// match_registry ⨝ match_participants), comme le MatchHistoryRepo d'Infinite —
// PAS de fetch live (la donnée y est déjà écrite par le livesync).
//
//   - matchIDs nil/vide → les N derniers matchs du joueur (ORDER BY start_time DESC) ;
//   - matchIDs non vide → filtre sur ces IDs, ordre d'entrée préservé.
//
// Dégradation propre : source non câblée (nil) → ErrCapabilityNotSupported (le
// caller dégrade). C'est aussi le cas sous le builder global live-only (pas de
// PlayerDB) : seul le builder player-scoped injecte la source.
func (a *DataAdapter) LoadMatchSummaries(ctx context.Context, matchIDs []string) ([]canonical.MatchSummary, error) {
	if a.matchHistory == nil {
		return nil, games.ErrCapabilityNotSupported
	}
	summaries, err := a.matchHistory.GetMatchSummaries(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("h5 LoadMatchSummaries: %w", err)
	}
	return summaries, nil
}

// h5MatchDetailMaxPages borne la recherche du matchID dans l'historique du viewer
// (GetPlayerMatches est player+page-based, pas ID-based). 8 pages × 25 = 200 matchs
// récents couverts ; au-delà, on dégrade (un match très ancien n'est de toute façon
// pas une cible Match View live réaliste).
const (
	h5MatchDetailMaxPages = 8
	h5MatchDetailPageSize = 25
)

// LoadMatchDetail assemble un canonical.MatchDetail d'un match Halo 5 depuis la
// donnée LIVE (best-effort). Flux :
//  1. résoudre le GAMERTAG du viewer depuis le contexte (ctxkeys.ViewerGamertag) —
//     Halo 5 est gamertag-keyé et la carnage exige le MODE + les refs header issus
//     de l'entrée d'historique du joueur (GetPlayerMatches) ;
//  2. paginer GetPlayerMatches(gamertag) pour retrouver l'entrée du matchID
//     (→ mode du carnage + refs header via le mapper summary réutilisé) ;
//  3. fetch GetMatchCarnage(matchID, mode) (roster complet) ;
//  4. projeter carnage + header → MatchDetail (mappers parallèles à l'ingestion).
//
// DEGRADATION (→ nil, ErrCapabilityNotSupported, le service Part B retombe sur le
// repo) : viewer gamertag absent du contexte, source/token indisponible, matchID
// introuvable dans l'historique récent, carnage 404/vide. Toute indisponibilité
// gracieuse (token expiré, 404) est loguée, pas propagée en erreur dure.
func (a *DataAdapter) LoadMatchDetail(ctx context.Context, matchID string) (*canonical.MatchDetail, error) {
	matchID = strings.TrimSpace(matchID)
	if matchID == "" {
		return nil, games.ErrCapabilityNotSupported
	}
	gamertag := strings.TrimSpace(ctxkeys.ViewerGamertag(ctx))
	if gamertag == "" {
		// Sans viewer, ni le MODE de la carnage ni les refs header ne sont
		// résolvables (Player.Xuid null, pas d'historique). Dégradation propre.
		a.logger.DebugContext(ctx, "h5 LoadMatchDetail: viewer gamertag absent du contexte (dégradation)",
			"match_id", matchID)
		return nil, games.ErrCapabilityNotSupported
	}
	src, err := a.resolveSource(ctx)
	if err != nil {
		return nil, games.ErrCapabilityNotSupported
	}
	ctx, cancel := context.WithTimeout(ctx, h5RequestTimeout)
	defer cancel()

	header, gameMode, found := a.findMatchInHistory(ctx, src, gamertag, matchID)
	if !found {
		a.logger.DebugContext(ctx, "h5 LoadMatchDetail: match introuvable dans l'historique récent (dégradation)",
			"player", gamertag, "match_id", matchID)
		return nil, games.ErrCapabilityNotSupported
	}

	carnage, err := src.GetMatchCarnage(ctx, matchID, h5GameModeSegment(gameMode))
	if err != nil {
		if a.degradeUnavailable(ctx, err, gamertag, "LoadMatchDetail") {
			return nil, games.ErrCapabilityNotSupported
		}
		return nil, fmt.Errorf("h5 LoadMatchDetail(%s): %w", matchID, err)
	}
	detail := mapCarnageToCanonicalDetail(matchID, gamertag, header, carnage)
	if detail == nil {
		a.logger.DebugContext(ctx, "h5 LoadMatchDetail: carnage vide (dégradation)",
			"player", gamertag, "match_id", matchID)
		return nil, games.ErrCapabilityNotSupported
	}
	a.enrichCommendations(ctx, detail) // best-effort : noms + icônes natifs (AXE B)
	return detail, nil
}

// findMatchInHistory pagine l'historique du viewer pour retrouver l'entrée du
// matchID. Retourne le header canonique (refs map/playlist/variant + dates +
// isRanked, via le mapper summary réutilisé) + le GameMode (carnage) + found.
// Best-effort : toute erreur de page (token/404/réseau) arrête la pagination et
// renvoie found=false (le caller dégrade).
func (a *DataAdapter) findMatchInHistory(ctx context.Context, src h5Source, gamertag, matchID string) (*canonical.MatchSummary, int, bool) {
	seen := make(map[string]struct{}) // garde anti-boucle si l'API n'honore pas `start`
	for page := 0; page < h5MatchDetailMaxPages; page++ {
		resp, err := src.GetPlayerMatches(ctx, gamertag, page*h5MatchDetailPageSize, h5MatchDetailPageSize)
		if err != nil {
			a.degradeUnavailable(ctx, err, gamertag, "LoadMatchDetail.history")
			return nil, 0, false
		}
		if resp == nil || len(resp.Results) == 0 {
			return nil, 0, false // fin de l'historique
		}
		for i := range resp.Results {
			r := &resp.Results[i]
			if _, dup := seen[r.Id.MatchId]; dup {
				return nil, 0, false // pagination non avançante -> stop défensif
			}
			seen[r.Id.MatchId] = struct{}{}
			if h5HeaderMatchID(r.Id.MatchId, matchID) {
				summary := mapOneMatchSummary(r, gamertag, a.classifier)
				return &summary, r.Id.GameMode, true
			}
		}
	}
	return nil, 0, false
}

func (a *DataAdapter) LoadEncounters(_ context.Context, _ string) ([]canonical.EncounterRow, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadLUSRHistory : Halo 5 n'a pas de rating LUSR (CSR natif via service record).
func (a *DataAdapter) LoadLUSRHistory(_ context.Context, _ string) ([]canonical.LUSRCheckpoint, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadTopMatches(_ context.Context, _ string) ([]canonical.CareerTopMatch, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadTargetRecentMatches(_ context.Context, _ string, _ int) ([]canonical.RecentMatchRow, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadParticipantStats(_ context.Context, _ string, _ []string) (*canonical.PlayerMatchSetStats, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadPlayerIntersection(_ context.Context, _, _ string) (*canonical.PlayerIntersection, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadTimeseries(_ context.Context, _ string, _ canonical.TimeseriesQuery) (*canonical.MetricSeries, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadMatchScoreboard(_ context.Context, _ string) ([]canonical.MatchParticipant, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadHighlightEvents(_ context.Context, _ string) ([]canonical.HighlightEvent, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadMatchEvents récupère la timeline d'events NATIVE Halo 5 (/h5/matches/{id}/
// events) et la mappe en canonical (kill-feed + arme-par-kill + médailles +
// positions). Surface on-demand. Token/404 indisponible → timeline vide (dégradé),
// pas d'erreur dure (cf. degradeUnavailable). matchID vide / source absente →
// ErrCapabilityNotSupported.
func (a *DataAdapter) LoadMatchEvents(ctx context.Context, matchID string, opts canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error) {
	if strings.TrimSpace(matchID) == "" {
		return nil, games.ErrCapabilityNotSupported
	}
	// Source LOCALE (DuckDB synchronisé) prioritaire si injectée : kill-feed servi
	// hors-ligne (démo, aucun token → l'API /events live échouerait). Dégrade vers le
	// live uniquement si la lecture DB échoue ET qu'une source live existe.
	if a.eventsLocal != nil {
		if tl, lerr := a.eventsLocal.GetMatchEvents(ctx, matchID, opts); lerr == nil && tl != nil {
			return tl, nil
		} else if lerr != nil {
			a.logger.DebugContext(ctx, "h5 kill-feed local: lecture échouée (dégradation)", "match_id", matchID, "err", lerr)
		}
		if a.newSource == nil {
			return &canonical.MatchEventTimeline{MatchID: matchID}, nil
		}
	}
	src, err := a.resolveSource(ctx)
	if err != nil {
		return nil, games.ErrCapabilityNotSupported
	}
	ctx, cancel := context.WithTimeout(ctx, h5RequestTimeout)
	defer cancel()

	resp, err := src.GetMatchEvents(ctx, matchID)
	if err != nil {
		if a.degradeUnavailable(ctx, err, matchID, "LoadMatchEvents") {
			return &canonical.MatchEventTimeline{MatchID: matchID}, nil
		}
		return nil, fmt.Errorf("h5 LoadMatchEvents(%s): %w", matchID, err)
	}
	return &canonical.MatchEventTimeline{MatchID: matchID, Events: mapH5Events(resp, opts)}, nil
}

func (a *DataAdapter) LoadFriendsXUIDs(_ context.Context, _ string) ([]string, error) {
	return nil, games.ErrCapabilityNotSupported
}
