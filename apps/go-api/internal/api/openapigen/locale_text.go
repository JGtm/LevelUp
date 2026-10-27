package openapigen

import "levelup/go-api/internal/i18n"

// schemaTypeString est le type OpenAPI d'une propriété de langue.
const schemaTypeString = "string"

// schemaKeyType est la clé `type` d'un schéma OpenAPI.
const schemaKeyType = "type"

// Passe 3 — les libellés par langue à plat deviennent une propriété par langue.
//
// Un struct de réponse qui porte un `i18n.Text` à plat (cf. `i18n.MarshalFlat`) le déclare par un
// champ Go dont le tag JSON est `i18n.FlatMember` : Huma en dérive une propriété `i18n_text` de
// type objet à clés libres, qui ne dit rien du fil. Cette passe la remplace, dans chaque schéma
// qui la porte, par UNE PROPRIÉTÉ `string` PAR LANGUE de `config/locales.toml` (ordre
// alphabétique des codes, celui où le fil les écrit), et — quand `required:"true"` l'a rendue
// obligatoire — met ces propriétés à sa place dans `required`. Une langue ajoutée à la liste
// fait donc apparaître sa propriété dans le contrat sans aucune ligne de Go.
func expandLocaleTexts(node any) {
	switch v := node.(type) {
	case map[string]any:
		if props, ok := v["properties"].(map[string]any); ok {
			if _, has := props[i18n.FlatMember]; has {
				expandLocaleText(v, props)
			}
		}
		for _, child := range v {
			expandLocaleTexts(child)
		}
	case []any:
		for _, child := range v {
			expandLocaleTexts(child)
		}
	}
}

func expandLocaleText(schema, props map[string]any) {
	delete(props, i18n.FlatMember)
	codes := i18n.FlatCodes()
	for _, c := range codes {
		props[string(c)] = map[string]any{schemaKeyType: schemaTypeString}
	}
	req, ok := schema["required"].([]any)
	if !ok {
		return
	}
	out := make([]any, 0, len(req)+len(codes))
	for _, r := range req {
		if r != i18n.FlatMember {
			out = append(out, r)
			continue
		}
		for _, c := range codes {
			out = append(out, string(c))
		}
	}
	schema["required"] = out
}
