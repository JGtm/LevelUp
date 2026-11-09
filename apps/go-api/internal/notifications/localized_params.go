package notifications

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/i18n"
)

// Paramètres de notification qui portent un libellé écrit une fois par langue (`title_fr`,
// `title_en`, `medal_name_fr`... : voir i18n.ParamKey). Le web ne lit jamais ces paires : le
// serveur les remplace, à la lecture, par une clé unique (`title`, `medal_name`) dans la langue
// de la requête. Un libellé de plus dans une notification s'ajoute à cette liste.
const (
	ParamTitle     = "title"
	ParamMedalName = "medal_name"
	ParamRankName  = "rank_name"
)

var localizedParamBases = []string{ParamTitle, ParamMedalName, ParamRankName}

// LocalizeParams remplace, dans chaque notification de `items`, les familles de libellés par
// langue par une clé unique dans la langue `l`. Les lignes déjà stockées avec `title_fr` /
// `title_en` se lisent ainsi sans migration. Des paramètres illisibles sont journalisés et la
// notification est servie telle quelle (le texte reste affichable, sans interpolation).
func LocalizeParams(ctx context.Context, items []Notification, l i18n.Locale) {
	for i := range items {
		if len(items[i].Params) == 0 {
			continue
		}
		resolved, err := i18n.ResolveParamsJSON(string(items[i].Params), localizedParamBases, l)
		if err != nil {
			slog.WarnContext(ctx, "notifications: params illisibles, servis bruts",
				"err", err, "id", items[i].ID, "category", items[i].Category)
			continue
		}
		items[i].Params = []byte(resolved)
	}
}
