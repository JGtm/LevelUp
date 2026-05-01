// Package prestige — système Prestige (défis, arcs, PP, leaderboard).
//
// Ce package est le cœur métier du système de défis personnels et d'escouade.
// Il est conçu comme une API interne autonome : les autres packages
// (sync, api/handlers, ui) le consomment via le contrat `Service`.
//
// Couches :
//   - types.go     : structs persistées et transportées
//   - enums.go     : énumérations avec marshalling JSON
//   - constants.go : valeurs de référence (couleurs paliers, sources d'événements)
//   - repository.go: interfaces de persistance (impl dans platform/duckdb/)
package prestige

import (
	"encoding/json"
	"fmt"
)

// ---------- ChallengeStatus ----------

// ChallengeStatus représente l'état courant d'un défi dans son cycle de vie.
type ChallengeStatus string

const (
	StatusDraft     ChallengeStatus = "draft"
	StatusActive    ChallengeStatus = "active"
	StatusCompleted ChallengeStatus = "completed"
	StatusExpired   ChallengeStatus = "expired"
	StatusAbandoned ChallengeStatus = "abandoned"
	StatusArchived  ChallengeStatus = "archived"
)

// IsTerminal retourne true si le statut représente une fin de vie du défi.
func (s ChallengeStatus) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusExpired, StatusAbandoned, StatusArchived:
		return true
	}
	return false
}

// Valid retourne true si le statut est connu.
func (s ChallengeStatus) Valid() bool {
	switch s {
	case StatusDraft, StatusActive, StatusCompleted, StatusExpired, StatusAbandoned, StatusArchived:
		return true
	}
	return false
}

func (s ChallengeStatus) String() string { return string(s) }

func (s *ChallengeStatus) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	candidate := ChallengeStatus(raw)
	if !candidate.Valid() {
		return fmt.Errorf("prestige: invalid ChallengeStatus %q", raw)
	}
	*s = candidate
	return nil
}

// ---------- Tier ----------

// Tier représente le palier de difficulté calculé d'un défi.
type Tier string

const (
	TierNormal    Tier = "normal"
	TierHeroic    Tier = "heroic"
	TierLegendary Tier = "legendary"
	TierMythic    Tier = "mythic"
)

// Valid retourne true si le palier est connu.
func (t Tier) Valid() bool {
	switch t {
	case TierNormal, TierHeroic, TierLegendary, TierMythic:
		return true
	}
	return false
}

func (t Tier) String() string { return string(t) }

func (t *Tier) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	candidate := Tier(raw)
	if !candidate.Valid() {
		return fmt.Errorf("prestige: invalid Tier %q", raw)
	}
	*t = candidate
	return nil
}

// ---------- Cadence ----------

// Cadence représente le rythme d'attribution/évaluation d'un défi.
type Cadence string

const (
	CadenceDaily   Cadence = "daily"
	CadenceWeekly  Cadence = "weekly"
	CadenceMonthly Cadence = "monthly"
	CadenceFree    Cadence = "free"
)

func (c Cadence) Valid() bool {
	switch c {
	case CadenceDaily, CadenceWeekly, CadenceMonthly, CadenceFree:
		return true
	}
	return false
}

func (c Cadence) String() string { return string(c) }

func (c *Cadence) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	candidate := Cadence(raw)
	if !candidate.Valid() {
		return fmt.Errorf("prestige: invalid Cadence %q", raw)
	}
	*c = candidate
	return nil
}

// ---------- EvalType ----------

// EvalType distingue les défis "moyenne sur fenêtre" des défis "compteur cumulé".
type EvalType string

const (
	EvalThreshold  EvalType = "threshold"
	EvalCumulative EvalType = "cumulative"
)

func (e EvalType) Valid() bool {
	switch e {
	case EvalThreshold, EvalCumulative:
		return true
	}
	return false
}

func (e EvalType) String() string { return string(e) }

func (e *EvalType) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	candidate := EvalType(raw)
	if !candidate.Valid() {
		return fmt.Errorf("prestige: invalid EvalType %q", raw)
	}
	*e = candidate
	return nil
}

// ---------- WindowType ----------

// WindowType représente le type de fenêtre temporelle d'un défi.
//
// "matches_internal" est réservé aux calculs de baseline et n'est jamais
// exposé dans l'UI.
type WindowType string

const (
	WindowSession         WindowType = "session"
	WindowRollingDays     WindowType = "rolling_days"   // déprécié — préférer WindowLastNMatches
	WindowLastNMatches    WindowType = "last_n_matches" // fenêtre par compteur de matchs (N = WindowValue)
	WindowDeadline        WindowType = "deadline"
	WindowMatchesInternal WindowType = "matches_internal"
)

func (w WindowType) Valid() bool {
	switch w {
	case WindowSession, WindowRollingDays, WindowLastNMatches, WindowDeadline, WindowMatchesInternal:
		return true
	}
	return false
}

func (w WindowType) String() string { return string(w) }

func (w *WindowType) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	candidate := WindowType(raw)
	if !candidate.Valid() {
		return fmt.Errorf("prestige: invalid WindowType %q", raw)
	}
	*w = candidate
	return nil
}

// ---------- ChallengeMode ----------

// ChallengeMode distingue les défis créés librement par le joueur des défis
// pilotés (auto-attribués par le système avec contraintes de cadence/cooldown).
type ChallengeMode string

const (
	ModeLibre  ChallengeMode = "libre"
	ModePilote ChallengeMode = "pilote"
)

func (m ChallengeMode) Valid() bool {
	switch m {
	case ModeLibre, ModePilote:
		return true
	}
	return false
}

func (m ChallengeMode) String() string { return string(m) }

func (m *ChallengeMode) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	candidate := ChallengeMode(raw)
	if !candidate.Valid() {
		return fmt.Errorf("prestige: invalid ChallengeMode %q", raw)
	}
	*m = candidate
	return nil
}

// ---------- DataTier ----------

// DataTier indique la qualité des données disponibles pour calculer la baseline
// et donc les bonus PP applicables.
//
//   - DataFull      : ≥ 10 matchs sur le mode → palier plein, PP plein
//   - DataEstimated : 5–9 matchs → palier estimé, PP réduit de moitié
//   - DataTracking  : < 5 matchs → tracking pur, aucun bonus PP
type DataTier string

const (
	DataFull      DataTier = "full"
	DataEstimated DataTier = "estimated"
	DataTracking  DataTier = "tracking"
)

func (d DataTier) Valid() bool {
	switch d {
	case DataFull, DataEstimated, DataTracking:
		return true
	}
	return false
}

func (d DataTier) String() string { return string(d) }

func (d *DataTier) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	candidate := DataTier(raw)
	if !candidate.Valid() {
		return fmt.Errorf("prestige: invalid DataTier %q", raw)
	}
	*d = candidate
	return nil
}

// ---------- SquadMode ----------

// SquadMode distingue les défis d'escouade collaboratifs des compétitifs.
type SquadMode string

const (
	SquadCollective  SquadMode = "collective"
	SquadCompetitive SquadMode = "competitive"
)

func (s SquadMode) Valid() bool {
	switch s {
	case SquadCollective, SquadCompetitive:
		return true
	}
	return false
}

func (s SquadMode) String() string { return string(s) }

func (s *SquadMode) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	candidate := SquadMode(raw)
	if !candidate.Valid() {
		return fmt.Errorf("prestige: invalid SquadMode %q", raw)
	}
	*s = candidate
	return nil
}

// ---------- RejectReason ----------

// RejectReason est retournée par le calcul de palier quand un défi est refusé.
type RejectReason string

const (
	RejectNone             RejectReason = ""
	RejectTooEasy          RejectReason = "too_easy"          // stretch < 1.08
	RejectInsufficientData RejectReason = "insufficient_data" // < 5 matchs sur la métrique
	RejectStaleBaseline    RejectReason = "stale_baseline"    // > 60 jours d'inactivité
	RejectCeilingHit       RejectReason = "ceiling_hit"       // cible dépasse le plafond physique
	RejectQuotaExceeded    RejectReason = "quota_exceeded"    // plafond simultanés mode pilote
	RejectInvalidWindow    RejectReason = "invalid_window"    // fenêtre incohérente avec le type de défi
)
