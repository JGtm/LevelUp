package canonical

import "time"

// StatsScope encapsule les filtres d'un appel LoadPlayerStats.
//
// Convention : From/To en UTC, IncludeRanked nil = pas de filtre actif.
// Les slices vides signifient "pas de filtre" et non "aucun match".
type StatsScope struct {
	From          time.Time
	To            time.Time
	PlaylistIDs   []string
	IncludePvE    bool
	IncludeRanked *bool
}

// TimeseriesQuery décrit une requête sur les séries temporelles.
type TimeseriesQuery struct {
	Metric  FieldKey
	Bucket  Bucket
	From    time.Time
	To      time.Time
	GroupBy []GroupBy
}

// CareerOptions contrôle la profondeur d'un appel LoadCareerSnapshot.
//
// HistoryLimit = 0 signifie "limite raisonnable par défaut" (50), pas
// "tout l'historique" — pour récupérer l'historique complet, utiliser
// LoadTimeseries avec un From à zéro.
type CareerOptions struct {
	IncludeHistory bool
	HistoryLimit   int
}

// DefaultCareerHistoryLimit est la borne par défaut quand HistoryLimit = 0.
const DefaultCareerHistoryLimit = 50
