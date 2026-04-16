// Package domain — last_match.go : types pour la résolution du dernier match.
//
// Sprint 33 :
//
//	POST /api/v1/players/{slug}/pages/last-match/resolve → LastMatchResolveResponse
package domain

// ---------------------------------------------------------------------------
// Requête
// ---------------------------------------------------------------------------

// LastMatchResolveRequest est le corps de POST /pages/last-match/resolve.
type LastMatchResolveRequest struct {
	Filters      FilterContextInput `json:"filters"`
	CurrentIndex *int               `json:"current_index,omitempty"`
}

// ---------------------------------------------------------------------------
// Réponse
// ---------------------------------------------------------------------------

// LastMatchResolveResponse est la réponse de résolution du dernier match.
type LastMatchResolveResponse struct {
	CurrentMatchID      string  `json:"current_match_id"`
	TotalMatchesInScope int     `json:"total_matches_in_scope"`
	CurrentIndex        int     `json:"current_index"`
	PreviousMatchID     *string `json:"previous_match_id"`
	NextMatchID         *string `json:"next_match_id"`
	SessionTrackingKey  string  `json:"session_tracking_key"`
}
