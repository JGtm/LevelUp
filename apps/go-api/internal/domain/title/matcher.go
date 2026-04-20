// Package title — matcher.go : résolution d'un title_id/steam_app_id vers un TitleDescriptor.
//
// Utilisé par le watcher pour déterminer si un événement de présence
// correspond à un titre tracké.
package title

import "log/slog"

// MatchByXboxTitleID cherche un titre dont le XboxTitleID correspond.
// Retourne nil si aucun titre ne matche.
func (r *Registry) MatchByXboxTitleID(xboxTitleID string) *TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, td := range r.titles {
		if td.XboxTitleID != "" && td.XboxTitleID == xboxTitleID {
			slog.Debug("title_match: Xbox title ID match",
				"xbox_title_id", xboxTitleID,
				"slug", td.Slug,
			)
			return td
		}
	}
	return nil
}

// MatchBySteamAppID cherche un titre dont le SteamAppID correspond.
// Retourne nil si aucun titre ne matche.
func (r *Registry) MatchBySteamAppID(steamAppID string) *TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, td := range r.titles {
		if td.SteamAppID != "" && td.SteamAppID == steamAppID {
			slog.Debug("title_match: Steam app ID match",
				"steam_app_id", steamAppID,
				"slug", td.Slug,
			)
			return td
		}
	}
	return nil
}

// MatchPresence tente de résoudre un title_id (Xbox RTA) vers un titre.
// Alias sémantique de MatchByXboxTitleID.
func (r *Registry) MatchPresence(titleID string) *TitleDescriptor {
	return r.MatchByXboxTitleID(titleID)
}
