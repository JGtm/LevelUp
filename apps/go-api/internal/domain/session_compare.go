// Package domain — session_compare.go : types pour la comparaison de sessions.
//
// Sprint 33 :
//
//	POST /api/v1/players/{slug}/pages/session-compare → SessionCompareResponse
package domain

// ---------------------------------------------------------------------------
// Requête
// ---------------------------------------------------------------------------

// SessionCompareRequest est le corps de POST /pages/session-compare.
type SessionCompareRequest struct {
	Filters  FilterContextInput `json:"filters"`
	SessionA *string            `json:"session_a,omitempty"`
	SessionB *string            `json:"session_b,omitempty"`
}

// ---------------------------------------------------------------------------
// Réponse
// ---------------------------------------------------------------------------

// SessionCompareEntry résume une session sélectionnée.
type SessionCompareEntry struct {
	SessionLabel     string   `json:"session_label"`
	StartTime        *string  `json:"start_time"`
	EndTime          *string  `json:"end_time"`
	TotalMatches     int      `json:"total_matches"`
	Wins             int      `json:"wins"`
	Losses           int      `json:"losses"`
	KDA              *float64 `json:"kda"`
	PerformanceScore *float64 `json:"performance_score"`
	WithFriends      bool     `json:"with_friends"`
	DominantCategory *string  `json:"dominant_category"`
}

// SessionCompareMetricRow est une ligne de comparaison métrique A vs B.
type SessionCompareMetricRow struct {
	Key    string  `json:"key"`
	Label  string  `json:"label"`
	ValueA string  `json:"value_a"`
	ValueB string  `json:"value_b"`
	Delta  *string `json:"delta"`
	Winner *string `json:"winner"` // "a" | "b" | "tie"
}

// SessionCompareResponse est la réponse de POST /pages/session-compare.
type SessionCompareResponse struct {
	SessionA          *SessionCompareEntry      `json:"session_a"`
	SessionB          *SessionCompareEntry      `json:"session_b"`
	AvailableSessions []string                  `json:"available_sessions"`
	Metrics           []SessionCompareMetricRow `json:"metrics"`
	MapsTable         []map[string]interface{}  `json:"maps_table"`
	ModesTable        []map[string]interface{}  `json:"modes_table"`
}
