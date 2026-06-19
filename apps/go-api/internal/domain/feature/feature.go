// Package feature — types purs de la matrice de features produit (Phase 1.7b).
//
// Une FEATURE est une surface produit (page/section) exposée à l'utilisateur,
// dérivée par CASCADE des capabilities du titre (cf. internal/games/feature.go
// ComputeFeatureMatrix). Ce package ne contient que des types résultat : aucune
// dépendance DB/HTTP/games (les définitions feature→capability + la cascade
// vivent dans internal/games, là où sont les capabilities).
package feature

// Key identifie une feature produit (surface UI). Stable, title-agnostic.
type Key string

const (
	KeyMatchHistory Key = "match_history"
	KeyMatchDetail  Key = "match_detail"
	KeySkillRating  Key = "skill_rating"
	KeyCareer       Key = "career"
	KeyPveStats     Key = "pve_stats"
	KeyTimeseries   Key = "timeseries"
	KeyCitations    Key = "citations"
	KeyEngagement   Key = "engagement"
	KeyBattlePass   Key = "battlepass"
	KeyChallenges   Key = "challenges"
)

// Status reflète l'état d'une feature pour un titre donné (3 états, Phase 1.7b).
type Status string

const (
	// StatusAvailable : toutes les capabilities requises sont supported.
	StatusAvailable Status = "available"
	// StatusDegraded : utilisable mais partielle (une capability requise est
	// degraded, ou une capability d'enrichissement est absente/degraded).
	StatusDegraded Status = "degraded"
	// StatusUnavailable : une capability primaire requise est absente ou
	// not_exposed → la feature ne peut pas être exposée (dégradation gracieuse).
	StatusUnavailable Status = "unavailable"
)

// Available retourne vrai si la feature est exposable (available ou degraded).
func (s Status) Available() bool {
	return s == StatusAvailable || s == StatusDegraded
}

// Matrix est l'état de toutes les features produit d'un titre à un instant T.
type Matrix map[Key]Status

// Get retourne le statut d'une feature (StatusUnavailable si absente).
func (m Matrix) Get(k Key) Status {
	if s, ok := m[k]; ok {
		return s
	}
	return StatusUnavailable
}
