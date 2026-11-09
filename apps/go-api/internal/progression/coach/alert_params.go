package coach

import (
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/progression/milestones"
)

// milestoneParams complète les paramètres d'une alerte de jalon avec son titre, écrit une fois par
// langue (`title_fr`, `title_en`... : le serveur le résout à la lecture, notifications.LocalizeParams).
func milestoneParams(m milestones.CatalogEntry, params map[string]any) map[string]any {
	m.Title.AddParams(params, notifications.ParamTitle)
	return params
}
