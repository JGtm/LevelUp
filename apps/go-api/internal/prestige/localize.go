package prestige

import "levelup/go-api/internal/i18n"

// localize.go — libellés servis dans la langue de la requête (ADR 0040 D-3).
//
// Un Template ou un PresetArc lu depuis la base porte ses textes dans toutes les langues
// (`Labels`, `Titles`...). Le contrat HTTP sert le texte d'UNE langue (`label`, `title`...) : le
// service pose ces champs au moment de rendre la réponse, avec la `Locale` du contexte de la
// requête, jamais avant (une valeur lue ou mise en cache est indépendante de la langue).

// Localized rend une copie dont `Label` et `Description` sont ceux de la langue `loc`, avec le
// repli `en` de i18n.Text.Pick. Une description absente reste vide sans compter de repli.
func (t Template) Localized(loc i18n.Locale) Template {
	t.Label = t.Labels.Pick(loc)
	t.Description = t.Descriptions.PickOptional(loc)
	return t
}

// LocalizeTemplates rend une tranche neuve de templates localisés dans `loc` (nil reste nil).
func LocalizeTemplates(templates []Template, loc i18n.Locale) []Template {
	if templates == nil {
		return nil
	}
	out := make([]Template, len(templates))
	for i := range templates {
		out[i] = templates[i].Localized(loc)
	}
	return out
}

// Localized rend une copie dont `Title` et `Description` sont ceux de la langue `loc`.
func (p PresetArc) Localized(loc i18n.Locale) PresetArc {
	p.Title = p.Titles.Pick(loc)
	p.Description = p.Descriptions.PickOptional(loc)
	return p
}

// StoredLabel est le libellé qu'un défi recopie de son template à sa création (`Challenge.Label`).
// Ce texte est STOCKÉ, dans une seule langue : celle par défaut de l'application, avec le repli
// `en`, quelle que soit la langue de la requête qui crée le défi (comportement historique : le
// libellé était toujours le français). Servir un défi dans la langue de la requête suppose un
// libellé par langue en base, hors périmètre du contrat des templates.
func (t Template) StoredLabel() string {
	return t.Labels.Pick(i18n.Default)
}

// StoredTitle et StoredDescription : titre et description d'un arc preset recopiés à l'adoption
// dans l'arc du joueur (texte stocké, langue par défaut, repli `en` : voir StoredLabel).
func (p PresetArc) StoredTitle() string {
	return p.Titles.Pick(i18n.Default)
}

// StoredDescription : voir StoredTitle. Une description absente reste vide.
func (p PresetArc) StoredDescription() string {
	return p.Descriptions.PickOptional(i18n.Default)
}

// ServedLabel retourne le libellé d'un défi issu de ce modèle, dans la langue `loc` : le libellé
// du modèle quand le libellé STOCKÉ est exactement celui que la création a recopié (StoredLabel),
// donc que le joueur ne l'a pas modifié ; sinon le libellé stocké tel quel (un libellé édité ou
// libre n'est jamais remplacé). Le stockage reste à une langue (Challenge.Label) : l'origine
// (Challenge.TemplateID) retrouve les autres.
func (t Template) ServedLabel(stored string, loc i18n.Locale) string {
	if stored != t.StoredLabel() {
		return stored
	}
	return t.Labels.Pick(loc)
}

// ServedTitle : voir Template.ServedLabel, pour le titre d'un arc adopté depuis ce preset
// (Arc.PresetID) ; le titre stocké est celui de StoredTitle.
func (p PresetArc) ServedTitle(stored string, loc i18n.Locale) string {
	if stored != p.StoredTitle() {
		return stored
	}
	return p.Titles.Pick(loc)
}

// ServedDescription : voir ServedTitle, pour la description (facultative : une absence ne compte
// pas comme libellé manquant).
func (p PresetArc) ServedDescription(stored string, loc i18n.Locale) string {
	if stored != p.StoredDescription() {
		return stored
	}
	return p.Descriptions.PickOptional(loc)
}
