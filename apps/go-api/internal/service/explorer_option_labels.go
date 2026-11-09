package service

import (
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// optionLabels range les libellés affichés des options d'un filtre, par CLÉ de filtre. La clé est le
// nom de la langue par défaut (indépendante de la langue de la requête) ; le libellé est celui de la
// langue de la requête. Le premier libellé non vide d'une clé l'emporte.
type optionLabels map[string]string

// add enregistre le libellé de la clé ; une clé vide ou un libellé vide n'inscrit rien, un libellé vide
// laisse la clé sans libellé propre (valueOrKey la rend alors).
func (o optionLabels) add(key, label string) {
	if key == "" {
		return
	}
	if _, ok := o[key]; !ok || o[key] == "" {
		o[key] = label
	}
}

// options rend les options triées par libellé puis par clé. Un libellé absent est remplacé par la clé
// (le libellé de la langue par défaut). Dans la langue par défaut le libellé EST la clé : l'ordre est celui
// du tri alphabétique des clés.
func (o optionLabels) options() []domain.LabelValue {
	out := make([]domain.LabelValue, 0, len(o))
	for key, label := range o {
		if label == "" {
			label = key
		}
		out = append(out, domain.LabelValue{Label: label, Value: key})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Label != out[j].Label {
			return out[i].Label < out[j].Label
		}
		return out[i].Value < out[j].Value
	})
	return out
}

// labelOr rend le libellé de la clé, à défaut la clé elle-même (le libellé de la langue par défaut).
func (o optionLabels) labelOr(key string) string {
	if v := o[key]; v != "" {
		return v
	}
	return key
}

// localizeCascadeOptions pose, sur les options de playlist, de mode et de carte du filtre global, le
// libellé de la langue `loc` : la VALEUR (le nom de la langue par défaut, clé de la cascade, de l'URL et
// des filtres enregistrés) ne bouge pas. La couche pure (ResolveFiltersFromRows) reste canonique ; le
// service localise au point d'entrée, comme pour les types d'expérience (localizeExperienceOptions).
// Les libellés viennent des lignes du joueur (une clé sans ligne garde sa clé pour libellé). Chaque liste
// est ensuite rangée par libellé puis par clé : dans la langue par défaut le libellé est la clé et l'ordre
// ne change pas.
func localizeCascadeOptions(opts *domain.AvailableFilterOptions, rows []domain.FilterMatchRow, loc i18n.Locale) {
	playlists, modes, maps := optionLabels{}, optionLabels{}, optionLabels{}
	for _, r := range rows {
		playlists.add(playlistUI(r), r.PlaylistName.Pick(loc))
		maps.add(mapUI(r), r.MapName.Pick(loc))
		modes.add(modeUI(r), derefString(analysis.ResolveModeUIWithVariantIn(loc, r.PairName, r.GameVariantName)))
	}
	relabel := func(list []domain.LabelValue, labels optionLabels) {
		for i := range list {
			list[i].Label = labels.labelOr(list[i].Value)
		}
		sort.SliceStable(list, func(i, j int) bool { return list[i].Label < list[j].Label })
	}
	relabel(opts.Playlists, playlists)
	relabel(opts.Modes, modes)
	relabel(opts.Maps, maps)
}
