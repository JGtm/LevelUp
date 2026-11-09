// Package duckdb — season_pass_locale.go : ordre de lecture des libellés Battle Pass par langue.
//
// GameCMS stocke les traductions fr-FR ET en-US (battlepass_{track,item}_translations) et le
// payload brut porte ses clés de langue : l'ordre de lecture se dérive de i18n.Chain(loc)
// (langue demandée, repli, défaut), jamais d'un booléen « préfère l'anglais ». Une langue
// sans source (SQL ou clé JSON) ne contribue pas : une troisième langue ajoute ses sources ici
// et ses clés dans le payload, sans toucher l'ordonnancement.
package duckdb

import (
	"fmt"
	"strings"

	"levelup/go-api/internal/i18n"
)

// bpDefaultKey est la clé du payload qui porte le texte canonique sans langue : elle se lit
// juste après la langue de repli (l'anglais), avant les langues qui suivent dans la chaîne.
const bpDefaultKey = "default"

// bpLangSources regroupe, par langue, les expressions SQL (ordonnées) qui fournissent un
// libellé dans cette langue. Pour l'anglais, `value` (chaîne canonique en-US de Waypoint) est
// une source ANGLAISE : elle suit `.translations.en-US` dans la liste anglaise.
type bpLangSources map[i18n.Locale][]string

// bpLocaleOrderedCoalesce ordonne un COALESCE par préférence de locale : les sources de chaque
// langue de i18n.Chain(loc), dans cet ordre. Centralise l'ordre locale-aware des libellés
// d'items Battle Pass (loadItemMetadataMap + fillItemsFromAssetIndex + nom de piste) pour
// éviter une copie divergente de l'ordonnancement.
func bpLocaleOrderedCoalesce(loc i18n.Locale, sources bpLangSources) string {
	var ordered []string
	for _, l := range i18n.Chain(loc) {
		ordered = append(ordered, sources[l]...)
	}
	return "COALESCE(" + strings.Join(ordered, ", ") + ")"
}

// bpPayloadKeys retourne l'ordre des clés de langue d'un texte multilingue du payload pour la
// langue `loc` : une clé par langue de i18n.Chain(loc), la clé « default » suivant la langue
// de repli.
func bpPayloadKeys(loc i18n.Locale) []string {
	var keys []string
	for _, l := range i18n.Chain(loc) {
		keys = append(keys, string(l))
		if l == i18n.Fallback {
			keys = append(keys, bpDefaultKey)
		}
	}
	return keys
}

// bpTag est l'étiquette Halo principale de la langue : la valeur de `lang` des tables de
// traduction GameCMS et la clé `translations.<étiquette>` du payload brut.
func bpTag(l i18n.Locale) string { return i18n.HaloTags(l)[0] }

// bpAlias est l'alias SQL de la jointure de traduction de la langue.
func bpAlias(l i18n.Locale) string { return "t_" + string(l) }

// bpTranslationJoins rend un LEFT JOIN de `table` par langue de config/locales.toml (alias
// bpAlias, filtré sur l'étiquette de la langue), apparié à la définition `d` sur `keyCol` et
// `content_hash`. Les valeurs injectées viennent de locales.toml, jamais d'une entrée.
func bpTranslationJoins(table, keyCol string) string {
	var b strings.Builder
	for _, l := range i18n.Locales() {
		a := bpAlias(l)
		fmt.Fprintf(&b, "\n\t\tLEFT JOIN %s %s\n\t\t       ON %s.%s = d.%s\n\t\t      AND %s.content_hash = d.content_hash\n\t\t      AND %s.lang = '%s'",
			table, a, a, keyCol, keyCol, a, a, bpTag(l))
	}
	return b.String()
}

// bpJSONSources rend, par langue, les expressions JSON du payload brut (`rawCol`) qui portent le
// texte du champ `jsonKey` : `translations.<étiquette>` pour chaque langue, et `value` (chaîne
// canonique de Waypoint, source de la langue de repli) en plus pour celle-ci.
// `withColumn` ajoute, avant le JSON, la colonne dénormalisée `col` de la
// jointure de la langue.
func bpJSONSources(rawCol, jsonKey, col string, withColumn bool) bpLangSources {
	out := bpLangSources{}
	for _, l := range i18n.Locales() {
		var src []string
		if withColumn {
			src = append(src, bpAlias(l)+"."+col)
		}
		src = append(src, fmt.Sprintf("json_extract_string(%s, '$.CommonData.%s.translations.%s')", rawCol, jsonKey, bpTag(l)))
		if l == i18n.Fallback {
			src = append(src, fmt.Sprintf("json_extract_string(%s, '$.CommonData.%s.value')", rawCol, jsonKey))
		}
		out[l] = src
	}
	return out
}
