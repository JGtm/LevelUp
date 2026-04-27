// Package domain — squad_v2.go : DTOs pour la nouvelle version de la page
// Squad construite sur les fondations Phase 0 (PLAN_META_FOUNDATIONS_GO).
//
// Vit en parallèle de squad.go (legacy, mono-coéquipier) jusqu'à migration
// complète des consommateurs frontend (cf. PLAN_SQUAD_GO_PORTAGE).
package domain

import (
	"time"

	"levelup/go-api/internal/games/canonical"
)

// SquadPageV2Response est le DTO de la page Squad V2.
//
// Phase 1 chunk S1 : structure squelette avec uniquement l'intersection des
// matchs partagés. Les sections riches (KPI, score d'équipe, charts synergies,
// impact 8 rôles, radar, etc.) sont remplies par les chunks S2-S11.
//
// Capabilities porte les CapabilityGap rencontrées (joueurs avec capability
// match.history absente, sections impossibles à remplir) pour que le frontend
// affiche un <CapabilityGap mode="placeholder|cta"> approprié.
type SquadPageV2Response struct {
	MainPlayer         string             `json:"main_player"`
	Teammates          []string           `json:"teammates"`
	Period             string             `json:"period"`
	SharedMatchesCount int                `json:"shared_matches_count"`
	SharedMatches      []SquadSharedMatch `json:"shared_matches"`
	// Capabilities reprend canonical.CapabilityGap (CapabilityKey + ReasonCode +
	// Severity + Message + Retryable) pour signaler les sections degradees.
	Capabilities []canonical.CapabilityGap `json:"capabilities,omitempty"`
}

// SquadSharedMatch représente un match commun entre tous les joueurs sélectionnés.
//
// Players[gamertag] donne accès aux stats du joueur sur ce match précis. Les
// champs au niveau de la struct (StartedAt, Map, Outcome) sont hydratés depuis
// le joueur principal — ces données sont identiques pour tous les joueurs du
// match (même match_id).
type SquadSharedMatch struct {
	MatchID   string                              `json:"match_id"`
	StartedAt time.Time                           `json:"started_at_utc"`
	Map       *canonical.AssetReference           `json:"map,omitempty"`
	Mode      *canonical.AssetReference           `json:"mode,omitempty"`
	Playlist  *canonical.AssetReference           `json:"playlist,omitempty"`
	Outcome   canonical.Outcome                   `json:"outcome"`
	Players   map[string]canonical.PlayerMatchRow `json:"players"`
}
