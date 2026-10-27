package duckdb

import "levelup/go-api/internal/i18n"

// setTextPtr inscrit la valeur d'une langue quand le pointeur n'est pas nil (nil = non résolu :
// le texte est laissé tel quel).
func setTextPtr(t *i18n.Text, l i18n.Locale, v *string) {
	if v != nil {
		t.Set(l, *v)
	}
}
