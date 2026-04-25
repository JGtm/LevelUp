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
