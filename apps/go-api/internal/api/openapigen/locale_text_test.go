package openapigen

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/i18n"
)

func TestExpandLocaleTexts_UneProprieteParLangue(t *testing.T) {
	tree := map[string]any{
		"components": map[string]any{"schemas": map[string]any{
			"Obligatoire": map[string]any{
				schemaKeyType: "object",
				"properties": map[string]any{
					"a":             map[string]any{schemaKeyType: schemaTypeString},
					i18n.FlatMember: map[string]any{schemaKeyType: "object"},
					"z":             map[string]any{schemaKeyType: schemaTypeString},
				},
				"required": []any{"a", i18n.FlatMember, "z"},
			},
			"Optionnel": map[string]any{
				schemaKeyType: "object",
				"properties": map[string]any{
					i18n.FlatMember: map[string]any{schemaKeyType: "object"},
				},
			},
			"Sans": map[string]any{
				schemaKeyType: "object",
				"properties":  map[string]any{"a": map[string]any{schemaKeyType: schemaTypeString}},
			},
		}},
	}
	expandLocaleTexts(tree)
	schemas := tree["components"].(map[string]any)["schemas"].(map[string]any)

	codes := i18n.FlatCodes()
	wantRequired := []any{"a"}
	for _, c := range codes {
		wantRequired = append(wantRequired, string(c))
	}
	wantRequired = append(wantRequired, "z")

	ob := schemas["Obligatoire"].(map[string]any)
	if got := ob["required"]; !reflect.DeepEqual(got, wantRequired) {
		t.Errorf("required = %v, attendu %v (les langues prennent la place du repère)", got, wantRequired)
	}
	opt := schemas["Optionnel"].(map[string]any)
	if _, has := opt["required"]; has {
		t.Errorf("un libellé optionnel ne crée pas de required : %v", opt["required"])
	}
	for nom, s := range map[string]any{"Obligatoire": ob, "Optionnel": opt} {
		props := s.(map[string]any)["properties"].(map[string]any)
		if _, has := props[i18n.FlatMember]; has {
			t.Errorf("%s : le repère %q survit dans le contrat", nom, i18n.FlatMember)
		}
		for _, c := range codes {
			if p, ok := props[string(c)].(map[string]any); !ok || p[schemaKeyType] != schemaTypeString {
				t.Errorf("%s : propriété %q manquante ou non `string` : %v", nom, c, props[string(c)])
			}
		}
	}
	sans := schemas["Sans"].(map[string]any)["properties"].(map[string]any)
	if len(sans) != 1 {
		t.Errorf("un schéma sans libellé à plat est laissé tel quel : %v", sans)
	}
}
