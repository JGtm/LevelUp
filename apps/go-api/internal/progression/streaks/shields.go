package streaks

import "time"

// shields.go — logique de régénération et consommation des shields.
//
// Règles (cf. PLAN §4.2) :
//   - 1 shield par mois calendaire, régénéré le 1er du mois
//   - 1 shield couvre exactement 1 bucket manqué (1 jour ou 1 semaine)
//   - Au-delà → la streak break
//   - Shield consommé reste consommé même si on regagne la streak après

// RegenerateIfNewMonth réinitialise shields_used à 0 si le dernier incrément
// (ou la création de la streak si jamais incrémentée) est dans un mois passé.
//
// Idempotent : appelé à chaque évaluation sans effet de bord en cours de mois.
//
// Mutation in-place du Streak fourni.
func RegenerateIfNewMonth(s *Streak, now time.Time) {
	ref := s.StartedAt
	if s.LastIncrementAt != nil {
		ref = *s.LastIncrementAt
	}
	if !SameMonth(ref, now) {
		s.ShieldsUsed = 0
	}
}

// AvailableShields retourne le nombre de shields encore consommables ce mois.
// Toujours >= 0.
func AvailableShields(s *Streak) int {
	avail := s.ShieldsAvailable - s.ShieldsUsed
	if avail < 0 {
		return 0
	}
	return avail
}

// TryConsume tente de consommer `n` shields. Retourne true si l'opération a
// réussi (shields suffisants), false sinon (avec mutation in-place dans le
// cas positif).
//
// La mutation : ShieldsUsed += n. Status doit être mis à jour par l'appelant
// (typiquement StreakStatusPaused).
func TryConsume(s *Streak, n int) bool {
	if n <= 0 {
		return true
	}
	if AvailableShields(s) < n {
		return false
	}
	s.ShieldsUsed += n
	return true
}
