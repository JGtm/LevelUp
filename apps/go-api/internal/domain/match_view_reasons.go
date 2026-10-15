package domain

// Raisons partielles « lecture en échec » (E4, D-8) : une source de la vue dont la
// lecture a ÉCHOUÉ, par opposition à `<source>_empty` (lecture réussie, rien à
// afficher). Codes stables, lus par le web (bandeau « données indisponibles »).
const (
	// Raisons « lecture réussie, rien à afficher » (detectPartialMatchData).
	PartialReasonScoreboardEmpty        = "scoreboard_empty"
	PartialReasonEventsEmpty            = "events_empty"
	PartialReasonPlayerStatsEmpty       = "player_stats_empty"
	PartialReasonMedalsEmpty            = "medals_empty"
	PartialReasonScoreboardUnavailable  = "scoreboard_unavailable"
	PartialReasonEventsUnavailable      = "events_unavailable"
	PartialReasonPlayerStatsUnavailable = "player_stats_unavailable"
	PartialReasonMedalsUnavailable      = "medals_unavailable"
	// Code générique : un chargeur SANS raison dédiée a échoué (la source est dans le journal).
	PartialReasonSectionUnavailable = "section_unavailable"
)

// IsUnavailableReason dit si code est l'une des raisons « lecture en échec ». Ensemble
// fermé : tout autre code `*_unavailable` (structurel, propre à un match) garde son sens.
func IsUnavailableReason(code string) bool {
	switch code {
	case PartialReasonScoreboardUnavailable, PartialReasonEventsUnavailable,
		PartialReasonPlayerStatsUnavailable, PartialReasonMedalsUnavailable,
		PartialReasonSectionUnavailable:
		return true
	}
	return false
}

// HasUnavailableSource : la vue a été construite avec au moins une source en échec. Une
// telle vue est transitoire (la panne peut disparaître au prochain appel) : elle ne doit
// être réutilisée par aucun cache (D-9).
func (r MatchViewResponse) HasUnavailableSource() bool {
	for _, code := range r.PartialReasons {
		if IsUnavailableReason(code) {
			return true
		}
	}
	return false
}
