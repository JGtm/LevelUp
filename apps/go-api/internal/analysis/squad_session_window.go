// Package analysis — squad_session_window.go : fenêtre adaptative au rythme
// du joueur pour le timeline « Performance d'escouade par session » (teammates.04).
//
// Problème : un nombre fixe de sessions couvre ~2 semaines pour un hardcore mais
// ~4 mois pour un occasionnel ; une fenêtre en semaines fixe affiche 30 sessions
// pour l'un et 2 pour l'autre. La fenêtre est donc dimensionnée par le rythme
// (écart médian entre sessions), bornée pour rester lisible.
package analysis

import "sort"

// SquadSessionWindowConfig paramètre la fenêtre adaptative des sessions.
type SquadSessionWindowConfig struct {
	TargetSessions int // nombre de sessions visé pour dimensionner l'horizon
	MinSessions    int // plancher : on garde toujours au moins ce nombre
	MaxSessions    int // plafond : densité visuelle maximale
	MinDays        int // horizon minimal (jours)
	MaxDays        int // horizon maximal (jours)
}

// DefaultSquadSessionWindow renvoie les paramètres par défaut (stratégie
// adaptative hybride retenue avec l'utilisateur).
func DefaultSquadSessionWindow() SquadSessionWindowConfig {
	return SquadSessionWindowConfig{
		// Fenêtre élargie ~+50% (retour user : la précédente était trop sévère et
		// montrait trop peu de sessions).
		TargetSessions: 18,
		MinSessions:    9,
		MaxSessions:    30,
		MinDays:        21,
		MaxDays:        180,
	}
}

const secondsPerDay = 86400.0

// SquadSessionWindowKeep renvoie le nombre de sessions de queue (les plus
// récentes) à conserver, en fonction du rythme de jeu.
//
// firstSeenUnixAsc : timestamps Unix (secondes) de début de chaque session,
// triés par ordre chronologique ASCENDANT.
//
// Principe : horizon_jours = clamp(target × écart_médian_jours, MinDays, MaxDays),
// puis on garde les sessions dont le début est ≥ (dernière_session − horizon),
// le tout borné par [MinSessions, MaxSessions]. Concrètement :
//   - hardcore (écart ≈ 1 j)    → ~14 derniers jours
//   - régulier (écart ≈ 3 j)    → ~5 dernières semaines
//   - occasionnel (écart ≈ 10 j) → plafonné à ~120 j et 20 sessions
func SquadSessionWindowKeep(firstSeenUnixAsc []int64, cfg SquadSessionWindowConfig) int {
	n := len(firstSeenUnixAsc)
	if n <= cfg.MinSessions {
		return n
	}

	// Écart médian (jours) entre sessions consécutives.
	gaps := make([]float64, 0, n-1)
	for i := 1; i < n; i++ {
		d := float64(firstSeenUnixAsc[i]-firstSeenUnixAsc[i-1]) / secondsPerDay
		if d < 0 {
			d = 0 // sécurité : timestamps non monotones
		}
		gaps = append(gaps, d)
	}
	g := MedianFloat(gaps)
	if g <= 0 {
		g = 1 // sessions le même jour → l'horizon minimal s'appliquera
	}

	horizonDays := float64(cfg.TargetSessions) * g
	if horizonDays < float64(cfg.MinDays) {
		horizonDays = float64(cfg.MinDays)
	}
	if horizonDays > float64(cfg.MaxDays) {
		horizonDays = float64(cfg.MaxDays)
	}

	cutoff := firstSeenUnixAsc[n-1] - int64(horizonDays*secondsPerDay)
	keep := 0
	for i := n - 1; i >= 0; i-- {
		if firstSeenUnixAsc[i] >= cutoff {
			keep++
		} else {
			break // série triée ASC → tout ce qui précède est plus ancien
		}
	}

	if keep < cfg.MinSessions {
		keep = cfg.MinSessions
	}
	if keep > cfg.MaxSessions {
		keep = cfg.MaxSessions
	}
	if keep > n {
		keep = n
	}
	return keep
}

// MedianFloat retourne la médiane de xs (0 si vide). Canonique (K1n 2026-07-05) :
// réutilisé par le post-sync (medianStat) et la fenêtre de session escouade, plutôt
// que des recalculs sort+milieu dupliqués.
func MedianFloat(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	m := len(s) / 2
	if len(s)%2 == 1 {
		return s[m]
	}
	return (s[m-1] + s[m]) / 2
}
