package i18n

// LocaleEN et LocaleFR sont les codes des deux langues `required` de `config/locales.toml`, pour
// les seuls points qui parlent encore une langue précise : les colonnes SQL `_en` / `_fr` (jusqu'à
// l'étape 8 du plan i18n) et les champs de contrat JSON suffixés (jusqu'à l'étape 6). Tout autre
// code choisit sa langue par une `Locale` reçue, jamais par ces constantes.
const (
	LocaleEN Locale = "en"
	LocaleFR Locale = "fr"
)

// FromColumns assemble un Text depuis une paire de colonnes `_en` / `_fr` : une valeur vide
// n'est pas inscrite (Pick replie alors sur Fallback). Pont de lecture des colonnes SQL tant que
// leur schéma reste à deux langues (étape 8).
func FromColumns(en, fr string) Text {
	t := Text{}
	if en != "" {
		t[LocaleEN] = en
	}
	if fr != "" {
		t[LocaleFR] = fr
	}
	return t
}
