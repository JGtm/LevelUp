// Package testutil fournit des helpers de test transverses.
//
// jsonshape: garde-rail contre la régression "slice Go nil → JSON null → crash
// frontend". Le frontend type les slices comme `T[]` non-nullable ; quand un
// service Go renvoie un slice nil, la sérialisation JSON produit `null` et le
// front crashe sur `.filter()` / `.map()`.
//
// Cf. crash 2026-05-27 sur FilterOmnibar (filters_service.emptyResolved oubliait
// d'initialiser Playlists/Modes/Maps).
package testutil

import (
	"fmt"
	"reflect"
	"strings"
)

// TestingT capture la surface de *testing.T utilisée par les helpers — permet
// d'écrire des méta-tests sur le helper lui-même via un mock.
type TestingT interface {
	Errorf(format string, args ...any)
	Helper()
}

// RequireNoNilSlicesWithoutOmitempty parcourt récursivement v et fait échouer
// le test si un champ slice exposé en JSON sans `omitempty` est nil.
//
// Règle : un champ JSON sans `omitempty` est promis présent dans la sortie. Si
// le slice est nil, le JSON contient `null` au lieu de `[]`, ce qui viole le
// contrat avec un consommateur typé non-nullable.
//
// Les champs taggés `json:"-"` sont ignorés. Les champs avec `omitempty` sont
// tolérés nil (ils seront omis du JSON ; le type front correspondant doit alors
// être marqué optionnel).
func RequireNoNilSlicesWithoutOmitempty(t TestingT, v any) {
	t.Helper()
	walkForNilSlices(t, reflect.ValueOf(v), "root")
}

func walkForNilSlices(t TestingT, v reflect.Value, path string) {
	t.Helper()
	// Déréférencer pointeurs / interfaces.
	for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		rt := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := rt.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := f.Tag.Get("json")
			if tag == "-" {
				continue
			}
			parts := strings.Split(tag, ",")
			name := parts[0]
			if name == "" {
				name = f.Name
			}
			hasOmitempty := false
			for _, p := range parts[1:] {
				if p == "omitempty" {
					hasOmitempty = true
					break
				}
			}
			childPath := path + "." + name
			fv := v.Field(i)
			if fv.Kind() == reflect.Slice && fv.IsNil() && !hasOmitempty {
				t.Errorf("%s: slice field is nil but JSON tag has no omitempty — will marshal as JSON null and crash a non-nullable frontend consumer", childPath)
				continue
			}
			walkForNilSlices(t, fv, childPath)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkForNilSlices(t, v.Index(i), fmt.Sprintf("%s[%d]", path, i))
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			walkForNilSlices(t, iter.Value(), fmt.Sprintf("%s[%v]", path, iter.Key()))
		}
	}
}
