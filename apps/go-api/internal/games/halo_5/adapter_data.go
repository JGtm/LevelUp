// Package halo_5 — adapter_data.go : implementation games.TitleDataAdapter pour
// Halo 5, adossee au CLIENT LIVE (pas de DuckDB en Phase 1 read-only).
//
// Divergence majeure vs Halo Infinite : l'identite joueur est le GAMERTAG (l'API
// h5 ne fournit jamais de xuid). Le parametre `xuid` des methodes Load* est donc
// interprete comme le GAMERTAG cote Halo 5 (la resolution player -> cle de titre
// est faite en amont par le wiring multi-titre).
//
// Pas de SemanticAdapter dans ce package : Halo 5 utilise le games.Generic
// SemanticAdapter partage (le semantic adapter n'a aucune logique title-specific,
// cf. semantic_adapter.go). La divergence h5 vit ici (DataAdapter) + dans les TOML.
package halo_5

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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

// h5RecordModeArena : seul le service record arena est consomme en Phase 1
// (warzone = PvE-like, Phase 2).
const h5RecordModeArena = "arena"

// h5RequestTimeout borne chaque appel live (defensif contre un endpoint lent).
const h5RequestTimeout = 12 * time.Second

// DataAdapter est l'implementation games.TitleDataAdapter d'Halo 5.
type DataAdapter struct {
	source     h5Source
	staticCaps games.CapabilityMap
	logger     *slog.Logger
}

var _ games.TitleDataAdapter = (*DataAdapter)(nil)

// NewDataAdapter construit l'adapter Halo 5 adosse a une source live.
// source nil -> toutes les methodes live retournent ErrCapabilityNotSupported.
func NewDataAdapter(source h5Source, logger *slog.Logger) *DataAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &DataAdapter{source: source, logger: logger}
}

// WithCapabilities injecte la CapabilityMap chargee depuis capabilities.toml.
// nil -> fallbackCapabilities (filet de securite boot). Chainable.
func (a *DataAdapter) WithCapabilities(caps games.CapabilityMap) *DataAdapter {
	a.staticCaps = caps
	return a
}

// TitleSlug retourne l'identite du titre (constante de package, pas un gating).
func (a *DataAdapter) TitleSlug() string { return TitleSlug }

// Capabilities decrit l'etat des capabilities Halo 5 exposees par cet adapter.
// Source nominale : capabilities.toml via WithCapabilities ; fallback code sinon.
func (a *DataAdapter) Capabilities() games.CapabilityMap {
	base := a.staticCaps
	if base == nil {
		base = fallbackCapabilities()
	}
	out := make(games.CapabilityMap, len(base))
	for k, v := range base {
		out[k] = v
	}
	return out
}

// fallbackCapabilities est la CapabilityMap par defaut (filet boot si capabilities.toml
// n'a pas pu etre injecte). Miroir de config/titles/halo_5/mappings/capabilities.toml
// (parite gardee par capabilities_parity_test.go).
func fallbackCapabilities() games.CapabilityMap {
	return games.CapabilityMap{
		games.CapMatchHistory:       games.CapSupported,
		games.CapMatchDetailCore:    games.CapSupported,
		games.CapScoreboardExtra:    games.CapSupported,
		games.CapMatchSkillSnapshot: games.CapDegraded,
		games.CapCareerProgression:  games.CapSupported,
		games.CapTimeseries:         games.CapDegraded,
		games.CapEngagement:         games.CapDegraded,
		games.CapCitationsEngine:    games.CapNotExposed,
		games.CapPveFirefight:       games.CapNotExposed,
		games.CapBattlePass:         games.CapNotExposed,
		games.CapChallenges:         games.CapNotExposed,
	}
}

// LoadPlayerStats projette le service record arena (live) vers PlayerStats.
// `xuid` = GAMERTAG cote Halo 5. Un joueur sans record arena (404/vide) ->
// PlayerStats vide (pas une erreur).
func (a *DataAdapter) LoadPlayerStats(ctx context.Context, xuid string, _ canonical.StatsScope) (*canonical.PlayerStats, error) {
	if a.source == nil {
		return nil, games.ErrCapabilityNotSupported
	}
	gamertag := xuid
	ctx, cancel := context.WithTimeout(ctx, h5RequestTimeout)
	defer cancel()

	resp, err := a.source.GetServiceRecords(ctx, gamertag, h5RecordModeArena)
	if err != nil {
		if isNotFoundErr(err) {
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
	if a.source == nil {
		return nil, games.ErrCapabilityNotSupported
	}
	gamertag := xuid
	ctx, cancel := context.WithTimeout(ctx, h5RequestTimeout)
	defer cancel()

	resp, err := a.source.GetServiceRecords(ctx, gamertag, h5RecordModeArena)
	if err != nil {
		if isNotFoundErr(err) {
			return &canonical.CareerSnapshot{Player: h5Identity(gamertag)}, nil
		}
		return nil, fmt.Errorf("h5 LoadCareerSnapshot(%s): %w", gamertag, err)
	}
	if snap := mapCareerSnapshot(resp, gamertag); snap != nil {
		return snap, nil
	}
	return &canonical.CareerSnapshot{Player: h5Identity(gamertag)}, nil
}

// isNotFoundErr detecte un HTTPError 404/410 (ressource absente = pas une erreur
// metier : le joueur n'a simplement pas de record sur ce mode).
func isNotFoundErr(err error) bool {
	var he *HTTPError
	if errors.As(err, &he) {
		return he.StatusCode == http.StatusNotFound || he.StatusCode == http.StatusGone
	}
	return false
}

// --- Methodes non cablees en Phase 1 (degradation gracieuse explicite) ---

// LoadMatchSummaries : la signature est ID-based, mais l'historique Halo 5 est
// player+page-based (GetPlayerMatches). Le cablage de l'historique vers le
// canonique (via mapMatchSummaries, deja teste) est Phase 2 (necessite un chemin
// player-history dans le wiring). Stub en attendant.
func (a *DataAdapter) LoadMatchSummaries(_ context.Context, _ []string) ([]canonical.MatchSummary, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadMatchDetail : carnage report (2e appel) = Phase 2 (scoreboard etendu + CSR pre/post).
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
