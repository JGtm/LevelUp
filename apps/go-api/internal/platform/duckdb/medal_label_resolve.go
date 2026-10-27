// Package duckdb — medal_label_resolve.go : source unique de la résolution label + description d'une
// médaille dans la langue de la requête.
//
// Les textes d'une médaille vivent dans medal_translations, une ligne par (médaille, étiquette
// Halo) ; medal_definitions ne porte plus que des colonnes non textuelles (difficulté, type,
// score personnel, sprite). Les lecteurs (vue Match, tuile Home, Explorer/Squad, page Médailles,
// tiroir d'actifs) lisent leurs colonnes non textuelles puis passent par ce helper, au lieu de
// recopier une chaîne de repli en SQL.
//
// Chaîne de repli (ADR 0040 D-4, game names) : texte de la langue demandée, sinon texte de la
// langue de repli (`en`), sinon vide ; aucune traduction n'est inventée. Une langue vide ou
// inconnue se lit comme la langue de repli.
package duckdb

import (
	"context"
	"strings"

	"levelup/go-api/internal/i18n"
)

// medalText est le label et la description d'une médaille dans une langue de lecture.
type medalText struct {
	label       string
	description string
}

// medalReadLocale retourne la langue de lecture d'une langue demandée : la langue elle-même si
// elle figure dans la liste, sinon la langue de repli (un nom de jeu n'est jamais servi dans la
// langue par défaut faute de mieux).
func medalReadLocale(locale string) i18n.Locale {
	if l, ok := i18n.Lookup(locale); ok {
		return l
	}
	return i18n.Fallback
}

// pickMedalText choisit label et description dans la langue de lecture (repli compris).
func pickMedalText(t DefinitionTexts, loc i18n.Locale) medalText {
	return medalText{
		label:       strings.TrimSpace(t.Name.Pick(loc)),
		description: strings.TrimSpace(t.Description.PickOptional(loc)),
	}
}

// loadMedalLabels lit medal_translations pour `ids` et résout label et description dans la langue
// demandée. Une médaille sans aucune ligne de traduction est absente du résultat : l'appelant
// applique son repli (citation_mappings, identifiant).
func loadMedalLabels(ctx context.Context, q metaQuerier, ids []int64, locale string) (map[int64]medalText, error) {
	texts, err := LoadMedalTexts(ctx, q, uniqueInt64s(ids))
	if err != nil {
		return nil, err
	}
	loc := medalReadLocale(locale)
	out := make(map[int64]medalText, len(texts))
	for id, t := range texts {
		out[id] = pickMedalText(t, loc)
	}
	return out, nil
}
