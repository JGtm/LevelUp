package temporal

import "levelup/go-api/internal/games/canonical"

// engagement_signals.go — vecteur de signaux d'engagement title-agnostic (chantier
// F7 : engagement title-agnostic gradue).
//
// Idee : l'engagement devient un sous-systeme qui recoit un VECTEUR extensible de
// signaux. Chaque titre alimente ce qu'il expose (H5 peut fournir PLUS qu'Infinite :
// impulses objectif, mecaniques de kill riches...). Le moteur temporal ne connait
// AUCUN titre : la richesse est deduite de la COMPOSITION des events, jamais du slug
// (les events title-specific sont projetes en amont dans highlight_events par l'ingest
// title-owned — cf. games/halo_5/ingest/objective_impulses.go). Un titre futur s'active
// en fournissant son mapping d'ingest + ses coefficients calibres, ZERO modification
// de ce moteur.
//
// Deux portes de degradation gouvernent le statut du score (cf. plan F7 DE-3) :
//  1. Suffisance (ici) : sous l'ensemble minimal de signaux -> Insufficient ; minimal
//     seul -> Partial ; minimal + signaux riches -> Full (eligible supported).
//  2. Calibration (par titre, cote capability/service) : coefficients non valides pour
//     le titre -> plafonne a degraded.
//
// Les signaux optionnels absents ne pesent PAS (poids nul). Ils ne PAIENT qu'apres
// calibration (DE-5) : en l'etat, ils ne modifient PAS le score (la courbe reste
// calculee sur le meme flux d'events pondere par engagement_weights.go) ; ils
// gouvernent la SUFFISANCE et la confiance par match.

// SignalSufficiency classe la richesse du vecteur de signaux d'un match.
type SignalSufficiency int

const (
	// SufficiencyInsufficient : l'ensemble minimal (pace joueur datee + ancre lobby
	// + duree exploitable) n'est pas reuni -> aucun score fiable.
	SufficiencyInsufficient SignalSufficiency = iota
	// SufficiencyPartial : ensemble minimal present, aucun signal riche -> score
	// exploitable mais degrade.
	SufficiencyPartial
	// SufficiencyFull : minimal + au moins un signal riche present -> eligible au
	// statut supported (sous reserve de la 2e porte, calibration par titre).
	SufficiencyFull
)

// String rend le libelle machine-friendly (expose dans le champ signal_basis de la
// reponse API, cf. plan E3b).
func (s SignalSufficiency) String() string {
	switch s {
	case SufficiencyFull:
		return "full"
	case SufficiencyPartial:
		return "partial"
	default:
		return "insufficient"
	}
}

// EngagementSignals est le vecteur EXTENSIBLE de signaux d'engagement d'un match,
// agnostique du titre. Les pointeurs/absences encodent le masque de presence : un
// champ nil = famille de signal non fournie par le titre pour ce match (le moteur ne
// substitue AUCUNE valeur par defaut deguisee).
type EngagementSignals struct {
	// --- Ensemble minimal (requis pour tout score) ---

	// HasTimedPlayerEvents : le joueur a au moins un frag/mort date -> pace_joueur
	// calculable. Sans lui, la courbe joueur serait plate (signal absent).
	HasTimedPlayerEvents bool
	// HasLobbyPace : events lobby presents -> ancre de l'attendu (modele lobby-anchored).
	HasLobbyPace bool
	// DurationMS : duree du match. >= MinMatchDurationMS requis pour un signal exploitable.
	DurationMS int64

	// --- Signaux riches optionnels (extensibles ; un titre peut en fournir PLUS) ---

	// ObjectiveEvents : nombre d'events objectif ("mode") du joueur (porteur d'objectif,
	// leadership). Present chez les 2 titres (Infinite natif + H5 impulses allowlist).
	// nil = non fourni pour ce match.
	ObjectiveEvents *int
	// RichKillMechanics : nombre d'events de kill « riches » (finisher / clutch /
	// first_kill) — nuance de mecanique de combat. nil = non fourni.
	RichKillMechanics *int
}

// Sufficiency evalue la porte de suffisance du vecteur (1re porte de degradation F7).
func (s EngagementSignals) Sufficiency() SignalSufficiency {
	if !s.HasTimedPlayerEvents || !s.HasLobbyPace || s.DurationMS < MinMatchDurationMS {
		return SufficiencyInsufficient
	}
	if s.hasRichSignals() {
		return SufficiencyFull
	}
	return SufficiencyPartial
}

// hasRichSignals indique la presence d'au moins un signal riche NON nul (un champ
// present mais a 0 ne compte pas comme un signal riche effectif).
func (s EngagementSignals) hasRichSignals() bool {
	if s.ObjectiveEvents != nil && *s.ObjectiveEvents > 0 {
		return true
	}
	if s.RichKillMechanics != nil && *s.RichKillMechanics > 0 {
		return true
	}
	return false
}

// IsZero indique un vecteur non construit (aucun champ minimal renseigne). Sert de
// sentinelle a ComputeEngagementScore pour deriver le vecteur depuis ses inputs quand
// l'appelant (test legacy) ne le fournit pas.
func (s EngagementSignals) IsZero() bool {
	return !s.HasTimedPlayerEvents && !s.HasLobbyPace && s.DurationMS == 0 &&
		s.ObjectiveEvents == nil && s.RichKillMechanics == nil
}

// SignalsFromEvents construit le vecteur de signaux depuis les flux d'events deja
// partitionnes (joueur / lobby) et la duree du match. Helper title-AGNOSTIC partage
// par les points de construction (sync + serving) et par ComputeEngagementScore
// (derivation de reference). La richesse est deduite de la composition des events.
func SignalsFromEvents(playerEvents, lobbyEvents []canonical.HighlightEvent, durationMS int64) EngagementSignals {
	sig := EngagementSignals{
		HasTimedPlayerEvents: hasTimedFragOrDeath(playerEvents),
		HasLobbyPace:         len(lobbyEvents) > 0,
		DurationMS:           durationMS,
	}
	if obj := countEventType(playerEvents, "mode"); obj > 0 {
		sig.ObjectiveEvents = &obj
	}
	if rich := countRichKillMechanics(playerEvents); rich > 0 {
		sig.RichKillMechanics = &rich
	}
	return sig
}

// hasTimedFragOrDeath retourne vrai si le joueur a au moins un event kill/death date
// (le socle de la courbe joueur).
func hasTimedFragOrDeath(playerEvents []canonical.HighlightEvent) bool {
	for _, e := range playerEvents {
		switch canonical.HighlightEventType(e.EventType) {
		case canonical.EventKill, canonical.EventDeath,
			canonical.EventFirstKill, canonical.EventFirstDeath:
			return true
		}
	}
	return false
}

// countEventType compte les events d'un type brut donne dans une liste deja filtree.
func countEventType(events []canonical.HighlightEvent, eventType string) int {
	n := 0
	for _, e := range events {
		if e.EventType == eventType {
			n++
		}
	}
	return n
}

// countRichKillMechanics compte les events de kill « riches » (finisher / clutch /
// first_kill) du joueur — famille de signal riche extensible.
func countRichKillMechanics(playerEvents []canonical.HighlightEvent) int {
	n := 0
	for _, e := range playerEvents {
		switch canonical.HighlightEventType(e.EventType) {
		case canonical.EventFinisher, canonical.EventClutch, canonical.EventFirstKill:
			n++
		}
	}
	return n
}
