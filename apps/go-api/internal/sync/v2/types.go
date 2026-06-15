// Package v2 — types.go : types publics partagés par les 6 phases.
package v2

import "time"

// PlayerProfile identifie un joueur tracké pour le cycle.
// Champs strictement nécessaires à l'orchestration ; les détails (token,
// settings) sont résolus via les services injectés (HaloClient pinned,
// FriendsLoader, etc.).
type PlayerProfile struct {
	Gamertag   string
	XUID       string
	PlayerSlug string
	// TitleSlug porte le titre du joueur (MT-11 / PMT-3) pour que le pipeline V2
	// construise le moteur sur les bonnes DB (parité avec le path V1 scheduler).
	// Vide = halo_infinite (fallback DefaultSlug au point de construction).
	TitleSlug string
}

// CycleResult agrège le résultat d'un cycle V2 complet.
// Mappé sur scheduler.RunOnceResult au moment du retour au scheduler.
type CycleResult struct {
	StartedAt      time.Time
	Duration       time.Duration
	UniqueMatches  int                      // après dedup Phase 2
	PerPlayer      map[string]PlayerOutcome // keyed by PlayerSlug
	PhaseDurations map[string]time.Duration // keyed by phase name (cf. PhaseNames)
}

// PlayerOutcome capture le résultat par joueur dans un cycle V2.
type PlayerOutcome struct {
	Gamertag        string
	XUID            string
	MatchesUnknown  int    // découverts en Phase 1
	MatchesInserted int    // commitas en Phase 5 attribués à ce joueur
	MatchesSkipped  int    // déjà connus
	Status          string // "ok" | "partial" | "failed"
	FirstError      string // vide si OK
	Warnings        []string
}

// PhaseNames sert de clés stables pour PhaseDurations et le logging.
// Toute évolution est un changement d'API observable — à coordonner avec
// les dashboards expvar et la suite contract_test.
const (
	PhaseDiscovery   = "discovery"
	PhaseDedup       = "dedup"
	PhaseFetchShared = "fetch_shared"
	PhaseFetchPlayer = "fetch_player"
	PhasePersist     = "persist"
	PhasePostSync    = "post_sync"
)
