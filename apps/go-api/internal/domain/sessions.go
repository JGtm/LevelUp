// Package domain — types pour le découpage en sessions.
//
// Port Go de src/analysis/sessions.py (LevelUp-no-streamlit).
package domain

import "time"

// SessionMatchRow est le type de transfert brut entre la couche platform et les services.
// Seuls les champs nécessaires à l'algorithme de sessions sont présents.
type SessionMatchRow struct {
	MatchID        string
	StartTime      time.Time
	EndTime        *time.Time // start_time + time_played_seconds (si dispo)
	TeammatesSig   *string    // signature JSON des coéquipiers (xuid concaténés triés)
	IsRanked       bool
	TimePlayedSecs *int
}

// SessionGroup représente un groupe de matchs formant une session.
type SessionGroup struct {
	SessionID    int    // 0-based
	SessionLabel string // "DD/MM/YYYY HH:MM–HH:MM (N)"
	MatchIDs     []string
}

// SessionAssignment associe chaque match à un session_id et un label.
type SessionAssignment struct {
	MatchID      string `json:"match_id"`
	SessionID    int    `json:"session_id"`
	SessionLabel string `json:"session_label"`
}

// BucketType représente la granularité temporelle pour les graphiques de séries.
type BucketType string

const (
	BucketMatch BucketType = "match"
	BucketHour  BucketType = "hour"
	BucketDay   BucketType = "day"
	BucketWeek  BucketType = "week"
	BucketMonth BucketType = "month"
)

// BucketInfo contient le type et le label lisible d'un bucket.
type BucketInfo struct {
	Type  BucketType `json:"type"`
	Label string     `json:"label"` // "partie", "heure", "jour", "semaine", "mois"
}

// SessionComputeMode indique l'algorithme de découpage.
type SessionComputeMode string

const (
	SessionModeGap     SessionComputeMode = "gap"     // gap seul (mode simple)
	SessionModeContext SessionComputeMode = "context" // gap + coéquipiers + ranked
)

// SessionComputeOptions configure l'algorithme de découpage en sessions.
type SessionComputeOptions struct {
	// GapMinutes est le délai minimal (en minutes) entre deux matchs pour démarrer
	// une nouvelle session. Défaut = 120 (2 heures).
	GapMinutes int

	// CutoffHour est l'heure (0-23) avant laquelle une session de la veille est
	// considérée comme terminée. Défaut = 8 (8h du matin).
	CutoffHour int

	// FriendsXUIDs est la liste des XUIDs d'amis. Quand non vide, seuls les amis
	// déclenchent une rupture de session sur changement de coéquipiers.
	FriendsXUIDs []string

	// SplitOnRankedChange déclenche une rupture de session quand le mode ranked/social change.
	SplitOnRankedChange bool

	// Mode choisit l'algorithme (gap simple ou context).
	Mode SessionComputeMode
}

// SessionsQueryRequest est le corps de POST sessions/compute.
type SessionsQueryRequest struct {
	Options SessionComputeOptions `json:"options"`
}

// SessionsResponse est la réponse de la liste des sessions d'un joueur.
type SessionsResponse struct {
	Sessions   []SessionGroup      `json:"sessions"`
	Assignments []SessionAssignment `json:"assignments"`
	BucketInfo BucketInfo          `json:"bucket_info"`
	TotalDays  float64             `json:"total_days"`
}
