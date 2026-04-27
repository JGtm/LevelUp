package canonical

// Outcome représente le résultat d'un match pour un joueur ou une équipe.
//
// Les valeurs sont stables et alignées avec internal/domain/Outcome (la
// migration vers ce package est progressive, voir Phase F du plan).
type Outcome string

const (
	OutcomeWin  Outcome = "win"
	OutcomeLoss Outcome = "loss"
	OutcomeTie  Outcome = "tie"
	OutcomeDNF  Outcome = "dnf"
)

// IsKnownOutcome valide qu'un Outcome est dans l'enum canonique.
func IsKnownOutcome(o Outcome) bool {
	switch o {
	case OutcomeWin, OutcomeLoss, OutcomeTie, OutcomeDNF:
		return true
	}
	return false
}

// AllOutcomes retourne la liste exhaustive des Outcome supportés.
func AllOutcomes() []Outcome {
	return []Outcome{OutcomeWin, OutcomeLoss, OutcomeTie, OutcomeDNF}
}

// MatchType classe fonctionnellement un match dans le canonique.
type MatchType string

const (
	MatchTypeRanked    MatchType = "ranked"
	MatchTypeSocial    MatchType = "social"
	MatchTypeCustom    MatchType = "custom"
	MatchTypeFirefight MatchType = "firefight"
	MatchTypeUnknownMT MatchType = "unknown"
)

// RatingType discrimine les variantes de skill rating publiées par le produit.
type RatingType string

const (
	RatingTypeCSR  RatingType = "csr"
	RatingTypeLUSR RatingType = "lusr"
)

// Bucket désigne la granularité d'agrégation temporelle.
type Bucket string

const (
	BucketDay   Bucket = "day"
	BucketWeek  Bucket = "week"
	BucketMonth Bucket = "month"
)

// GroupBy désigne une dimension d'agrégation utilisée par les services
// timeseries / explorer.
type GroupBy string

const (
	GroupByPlaylist GroupBy = "playlist"
	GroupByMode     GroupBy = "mode"
	GroupByMap      GroupBy = "map"
)

// DominanceFlag classe la nature narrative d'un match (W/L) pour les badges
// affichés sur Career top matches, MatchView header, Synthesis top by week.
//
// Calculé au sync à partir des scores live + outcome final + écarts
// intermédiaires (cf. _medal_verdicts.go côté sync).
type DominanceFlag int

// Constantes des flags de dominance. DominanceNone = pas calculé ou pas
// de catégorie attribuée (match ordinaire).
const (
	DominanceNone        DominanceFlag = 0
	DominanceDomination  DominanceFlag = 1
	DominanceHumiliation DominanceFlag = 2
	DominanceRemontada   DominanceFlag = 3
	DominanceDebandade   DominanceFlag = 4
	DominanceContreRem   DominanceFlag = 5
)

// IsKnownDominanceFlag retourne true si la valeur est l'une des constantes
// connues (y compris DominanceNone).
func IsKnownDominanceFlag(d DominanceFlag) bool {
	switch d {
	case DominanceNone, DominanceDomination, DominanceHumiliation,
		DominanceRemontada, DominanceDebandade, DominanceContreRem:
		return true
	}
	return false
}

// AllDominanceFlags retourne la liste exhaustive des flags supportés (None
// inclus, en premier).
func AllDominanceFlags() []DominanceFlag {
	return []DominanceFlag{
		DominanceNone,
		DominanceDomination,
		DominanceHumiliation,
		DominanceRemontada,
		DominanceDebandade,
		DominanceContreRem,
	}
}

// HighlightEventType discrimine les types d'events filmés stockés dans
// shared.highlight_events. Utilisé par les filtres
// `port.HighlightEventFilters.EventTypes`.
type HighlightEventType string

// Constantes des types d'events filmés admis. Les valeurs miroitent les
// chaînes stockées dans `shared.highlight_events.event_type`.
const (
	EventKill       HighlightEventType = "kill"
	EventDeath      HighlightEventType = "death"
	EventAssist     HighlightEventType = "assist"
	EventMedal      HighlightEventType = "medal"
	EventFinisher   HighlightEventType = "finisher"
	EventClutch     HighlightEventType = "clutch"
	EventFirstKill  HighlightEventType = "first_kill"
	EventFirstDeath HighlightEventType = "first_death"
)

// IsKnownHighlightEventType valide qu'un type est dans l'enum canonique.
func IsKnownHighlightEventType(t HighlightEventType) bool {
	switch t {
	case EventKill, EventDeath, EventAssist, EventMedal,
		EventFinisher, EventClutch, EventFirstKill, EventFirstDeath:
		return true
	}
	return false
}

// AllHighlightEventTypes retourne la liste exhaustive des types supportés.
func AllHighlightEventTypes() []HighlightEventType {
	return []HighlightEventType{
		EventKill, EventDeath, EventAssist, EventMedal,
		EventFinisher, EventClutch, EventFirstKill, EventFirstDeath,
	}
}
