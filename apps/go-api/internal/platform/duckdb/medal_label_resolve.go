// Package duckdb — medal_label_resolve.go : source unique de l'expression SQL de
// résolution label + description d'une médaille (locale-aware).
//
// Pourquoi : quatre lecteurs résolvent le nom/description d'une médaille depuis
// medal_definitions (+ medal_translations) — la vue Match (lookupMedalMeta), la
// tuile Home (resolveMedalLabels), l'Explorer/Squad (MedalDefinitionsRepo.
// LookupByIDs) et le tab Assets (ListMedalsByTitle). Avant centralisation, une
// copie (Explorer/Squad) avait dérivé en OMETTANT les colonnes FR
// (md.name_fr / md.description_fr) → médailles non traduites sur la page Escouade
// ET le tab assets, sur Halo 5 ET Infinite (medal_translations est vide : aucun
// INSERT Go ne la peuple, la vraie source FR est medal_definitions.name_fr).
//
// Ce helper émet la chaîne canonique LOCALE-AWARE pour empêcher une 5e copie de
// rederiver. Calqué sur medal_citation_fallback.go (source unique du fallback).
//
// Chaîne de priorité (alias attendus dans la requête : `md` = medal_definitions,
// `mt_loc` = medal_translations[locale], `mt_en` = medal_translations[en-US]) :
//
//	locale FR : label       = mt_loc.name → md.name_fr → mt_en.name → md.name_en
//	            description = md.description_fr → mt_loc.description → md.description_en
//	locale EN : label       = mt_loc.name → mt_en.name → md.name_en   (PAS name_fr)
//	            description = mt_loc.description → md.description_en    (PAS description_fr)
//
// L'ordre FR préserve la priorité historique de la vue Match (md.name_fr AVANT
// mt_en.name) pour ne pas régresser ; il NE FAUT PAS injecter name_fr/
// description_fr en locale EN (sinon un client EN reçoit du FR).
package duckdb

// medalLabelDescCoalesceSQL retourne les deux expressions SELECT (label,
// description) à interpoler dans une requête médaille, en fonction de la locale
// applicative ("fr" ou "en"). Les alias `md`, `mt_loc`, `mt_en` doivent exister
// dans la requête appelante (cf. medalTranslationJoinsSQL).
func medalLabelDescCoalesceSQL(locale string) (labelExpr, descExpr string) {
	if medalLangCode(locale) == LangCodeFR {
		labelExpr = `COALESCE(
			NULLIF(TRIM(mt_loc.name),''),
			NULLIF(TRIM(md.name_fr),''),
			NULLIF(TRIM(mt_en.name),''),
			NULLIF(TRIM(md.name_en),''),
			''
		)`
		descExpr = `COALESCE(
			NULLIF(TRIM(md.description_fr),''),
			NULLIF(TRIM(mt_loc.description),''),
			NULLIF(TRIM(md.description_en),''),
			''
		)`
		return labelExpr, descExpr
	}
	// Locale EN (défaut) : ne jamais injecter les colonnes FR.
	labelExpr = `COALESCE(
		NULLIF(TRIM(mt_loc.name),''),
		NULLIF(TRIM(mt_en.name),''),
		NULLIF(TRIM(md.name_en),''),
		''
	)`
	descExpr = `COALESCE(
		NULLIF(TRIM(mt_loc.description),''),
		NULLIF(TRIM(md.description_en),''),
		''
	)`
	return labelExpr, descExpr
}

// medalTranslationJoinsSQL retourne les deux LEFT JOIN (mt_loc[locale] + mt_en[en-US])
// sur medal_translations à coller après `FROM medal_definitions md`. Tolère une
// table medal_translations vide (LEFT JOIN → NULL → COALESCE retombe sur md.*).
func medalTranslationJoinsSQL(locale string) string {
	return `LEFT JOIN medal_translations mt_loc
			    ON mt_loc.medal_name_id = md.medal_name_id AND mt_loc.lang = '` + medalLangCode(locale) + `'
			 LEFT JOIN medal_translations mt_en
			    ON mt_en.medal_name_id = md.medal_name_id AND mt_en.lang = '` + LangCodeEN + `'`
}
