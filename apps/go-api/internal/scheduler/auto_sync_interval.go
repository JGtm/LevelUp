// Package scheduler — auto_sync_interval.go : l'intervalle réglé de l'auto-sync.
package scheduler

import "time"

const defaultIntervalHours = 6

// CurrentInterval retourne l'intervalle courant depuis les settings.
// Utilise _minutes en priorité si > 0, sinon fallback sur _hours.
// Retourne defaultIntervalHours en cas d'erreur de lecture.
func (s *AutoSyncScheduler) CurrentInterval() time.Duration {
	cfg, err := s.settings.Load()
	if err != nil {
		return intervalFromHours(0)
	}
	return resolveInterval(cfg.SpnkrAutoSyncIntervalMinutes, cfg.SpnkrAutoSyncIntervalHours)
}

// resolveInterval retourne la durée à utiliser. minutes > 0 prime sur hours.
func resolveInterval(minutes, hours int) time.Duration {
	if minutes > 0 {
		return time.Duration(minutes) * time.Minute
	}
	return intervalFromHours(hours)
}

// intervalFromHours convertit un nombre d'heures en Duration.
// Retourne defaultIntervalHours si h <= 0.
func intervalFromHours(h int) time.Duration {
	if h <= 0 {
		return defaultIntervalHours * time.Hour
	}
	return time.Duration(h) * time.Hour
}
