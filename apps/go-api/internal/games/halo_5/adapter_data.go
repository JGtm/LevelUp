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
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// h5Source est la surface live minimale consommee par l'adapter (interface ->
// mockable en tests sans reseau). *Client l'implemente.
type h5Source interface {
	GetServiceRecords(ctx context.Context, gamertag, recordType string) (*H5ServiceRecordResponse, error)
	GetPlayerMatches(ctx context.Context, gamertag string, start, count int) (*H5MatchesResponse, error)
}

var _ h5Source = (*Client)(nil)

// SourceFactory produit une source live h5 a partir du contexte de requete (le
// SpartanToken vit dans ctx, par joueur+session). Retourne une erreur si le token
// est absent (-> degradation gracieuse cote adapter, pas de panique).
type SourceFactory func(ctx context.Context) (h5Source, error)

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
// n'a pas pu etre injecte). HONNETE Phase 1a : seules les methodes REELLEMENT cablees
// sur le client live sont exposees. career.progression = supported (LoadCareerSnapshot).
// Tout le reste = not_exposed tant que la methode est un stub (remonte en Phase 2 a
// mesure du cablage : match.history, match.detail.core, scoreboard, timeseries...).
// Parite avec config/titles/halo_5/mappings/capabilities.toml (capabilities_parity_test).
func fallbackCapabilities() games.CapabilityMap {
	return games.CapabilityMap{
		games.CapMatchHistory:       games.CapNotExposed,
		games.CapMatchDetailCore:    games.CapNotExposed,
		games.CapScoreboardExtra:    games.CapNotExposed,
		games.CapMatchSkillSnapshot: games.CapNotExposed,
		games.CapCareerProgression:  games.CapSupported,
		games.CapTimeseries:         games.CapNotExposed,
		games.CapEngagement:         games.CapNotExposed,
		games.CapCitationsEngine:    games.CapNotExposed,
		games.CapPveFirefight:       games.CapNotExposed,
		games.CapBattlePass:         games.CapNotExposed,
		games.CapChallenges:         games.CapNotExposed,
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

// LoadCareerSnapshot projette le pic CSR natif (service record) vers CareerSnapshot.
// `xuid` = GAMERTAG. Halo 5 n'a pas de progression XP facon rang carriere HINF :
// seuls le palier CSR (RankTier/RankName) et la valeur Onyx sont alimentes.
func (a *DataAdapter) LoadCareerSnapshot(ctx context.Context, xuid string, _ canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	gamertag := xuid
	src, err := a.resolveSource(ctx)
	if err != nil {
		return nil, games.ErrCapabilityNotSupported
	}
	ctx, cancel := context.WithTimeout(ctx, h5RequestTimeout)
	defer cancel()

	resp, err := src.GetServiceRecords(ctx, gamertag, h5RecordModeArena)
	if err != nil {
		if a.degradeUnavailable(ctx, err, gamertag, "LoadCareerSnapshot") {
			return &canonical.CareerSnapshot{Player: h5Identity(gamertag)}, nil
		}
		return nil, fmt.Errorf("h5 LoadCareerSnapshot(%s): %w", gamertag, err)
	}
	if snap := mapCareerSnapshot(resp, gamertag, a.placementTotal); snap != nil {
		return snap, nil
	}
	return &canonical.CareerSnapshot{Player: h5Identity(gamertag)}, nil
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

// --- Methodes non cablees en Phase 1 (capabilities.toml les declare not_exposed) ---

// LoadMatchSummaries : l'historique Halo 5 est player+page-based (GetPlayerMatches),
// pas ID-based comme cette signature. Le cablage history->canonique (via le mapper
// mapMatchSummaries, deja teste) est Phase 2. match.history = not_exposed.
func (a *DataAdapter) LoadMatchSummaries(_ context.Context, _ []string) ([]canonical.MatchSummary, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadMatchDetail : carnage report (2e appel) = Phase 2. match.detail.core = not_exposed.
func (a *DataAdapter) LoadMatchDetail(_ context.Context, _ string) (*canonical.MatchDetail, error) {
	return nil, games.ErrCapabilityNotSupported
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

func (a *DataAdapter) LoadFriendsXUIDs(_ context.Context, _ string) ([]string, error) {
	return nil, games.ErrCapabilityNotSupported
}
