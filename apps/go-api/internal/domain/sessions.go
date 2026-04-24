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
	SessionID          int      `json:"session_id"`
	SessionLabel       string   `json:"session_label"`
	MatchIDs           []string `json:"match_ids"`
	DurationSeconds    int      `json:"duration_seconds"`     // span (dernier start_time − premier start_time)
	TotalPlayedSeconds int      `json:"total_played_seconds"` // somme des time_played_seconds
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

// TeamChangeMode contrôle comment les changements de composition d'équipe
// déclenchent une rupture de session.
type TeamChangeMode string

const (
	// TeamChangeModeIgnore : ignorer la composition — sessions découpées par le
	// temps uniquement.
	TeamChangeModeIgnore TeamChangeMode = "ignore"

	// TeamChangeModeGroup : nouvelle session dès qu'un coéquipier quitte ou rejoint
	// (comportement par défaut).
	TeamChangeModeGroup TeamChangeMode = "group"

	// TeamChangeModeFriends : nouvelle session uniquement si un joueur de
	// FriendsXUIDs change.
	TeamChangeModeFriends TeamChangeMode = "friends"
)

// SessionComputeOptions configure l'algorithme de découpage en sessions.
type SessionComputeOptions struct {
	// GapMinutes est le délai minimal (en minutes) entre deux matchs pour démarrer
	// une nouvelle session. Défaut = 120 (2 heures).
	GapMinutes int

	// CutoffHour est l'heure (0-23) avant laquelle une session de la veille est
	// considérée comme terminée. Défaut = 8 (8h du matin).
	CutoffHour int

	// FriendsXUIDs est la liste des XUIDs d'amis. Utilisé en mode TeamChangeModeFriends.
	FriendsXUIDs []string

	// SplitOnRankedChange déclenche une rupture de session quand le mode ranked/social change.
	SplitOnRankedChange bool

	// TeamChangeMode contrôle la réaction aux changements de coéquipiers.
	// Valeurs : "ignore" | "group" (défaut) | "friends".
	// Chaîne vide → comportement rétrocompatible (group si FriendsXUIDs vide,
	// friends sinon).
	TeamChangeMode TeamChangeMode

	// Mode choisit l'algorithme (gap simple ou context).
	Mode SessionComputeMode
}

// SessionsQueryRequest est le corps de POST sessions/compute.
type SessionsQueryRequest struct {
	Options SessionComputeOptions `json:"options"`
}

// SessionsResponse est la réponse de la liste des sessions d'un joueur.
type SessionsResponse struct {
	Sessions    []SessionGroup      `json:"sessions"`
	Assignments []SessionAssignment `json:"assignments"`
	BucketInfo  BucketInfo          `json:"bucket_info"`
	TotalDays   float64             `json:"total_days"`
}
