// Package duckdb — asset_name_texts.go : noms d'affichage des cartes, playlists, paires et variantes
// de jeu rassemblés dans TOUTES les langues connues (asset_translations, mode_name_tr), puis choisis
// par la langue de la requête. Aucune langue n'est codée ici : les langues viennent de
// config/locales.toml (i18n.Locales), les étiquettes Halo de LoadAssetTexts. Une langue de plus est
// un jeu de lignes de plus (ADR 0040 D-4, D-13).
package duckdb

import (
	"context"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/i18n"
)

// assetNameTexts lit pour les actifs `ids` du type `kind` leur nom dans chaque langue qui en porte
// un (LoadAssetNames). Best-effort : l'échec est journalisé et consigné (bestEffortFailed : une
// lecture mise en cache qui l'appelle n'est pas cachée) et le résultat est vide, jamais nil. Une
// base sans la table (non migrée) n'est pas un échec.
func assetNameTexts(ctx context.Context, meta *DB, kind string, ids []string) map[string]i18n.Text {
	out := make(map[string]i18n.Text, len(ids))
	if meta == nil || len(ids) == 0 {
		return out
	}
	texts, err := LoadAssetNames(ctx, meta, kind, ids)
	if err != nil {
		bestEffortFailed(ctx, "asset_names_"+kind, err)
		return out
	}
	for id, t := range texts {
		out[id] = t.Name
	}
	return out
}

// namePlaceholder indique si `label` n'est pas un vrai nom dans sa langue : absent, ou identique au
// nom de la langue de repli (le COALESCE SQL de la base partagée recopie le nom anglais quand la
// colonne de la langue est vide).
func namePlaceholder(label, fallbackLabel string) bool {
	label = strings.TrimSpace(label)
	if label == "" {
		return true
	}
	fb := strings.TrimSpace(fallbackLabel)
	return fb != "" && strings.EqualFold(label, fb)
}

// fillUntranslated complète `field` avec les noms `from` (un par langue) : pour chaque langue autre
// que la langue de repli, un nom absent ou placeholder (namePlaceholder) est remplacé par la
// traduction non vide. Le nom de la langue de repli est celui du registre (référence des
// placeholders) : il n'est jamais écrasé, mais il est posé depuis `from` quand le registre n'en
// porte aucun (Halo 5 : le registre ne nomme ni carte ni playlist, la langue de repli vient alors de
// asset_translations, D-3). Un nom déjà traduit n'est jamais écrasé.
func fillUntranslated(field *i18n.Text, from i18n.Text) {
	if len(from) == 0 {
		return
	}
	if *field == nil {
		*field = i18n.Text{}
	}
	if strings.TrimSpace((*field)[i18n.Fallback]) == "" {
		if v := strings.TrimSpace(from[i18n.Fallback]); v != "" {
			(*field)[i18n.Fallback] = v
		}
	}
	fallback := (*field)[i18n.Fallback]
	for _, l := range i18n.Locales() {
		if l == i18n.Fallback {
			continue
		}
		v := strings.TrimSpace(from[l])
		if v != "" && namePlaceholder((*field)[l], fallback) {
			(*field)[l] = v
		}
	}
}

// fillAbsent complète `field` avec les noms `from` dans TOUTES les langues (langue de repli
// comprise) que le champ n'a pas déjà. N'écrit jamais une valeur vide : une langue sans nom reste
// absente (la lecture replie sur la langue de repli). Pour les titres dont le registre ne porte
// aucun nom (Halo 5).
func fillAbsent(field *i18n.Text, from i18n.Text) {
	for l, v := range from {
		v = strings.TrimSpace(v)
		if v == "" || field.Has(l) {
			continue
		}
		if *field == nil {
			*field = i18n.Text{}
		}
		(*field)[l] = v
	}
}

// pickName choisit le nom d'un actif dans la langue `loc` le long de i18n.Chain (repli croisé des
// libellés d'affichage, pickAcrossChain). `canonical` est le nom canonique de la langue de repli
// tenu par la base du joueur ou par le catalogue ; il prime sur la ligne de traduction de cette
// langue quand il est renseigné.
func pickName(loc i18n.Locale, translations i18n.Text, canonical string) string {
	t := translations.Clone()
	if t == nil {
		t = i18n.Text{}
	}
	if c := strings.TrimSpace(canonical); c != "" {
		t[i18n.Fallback] = c
	}
	return pickAcrossChain(loc, t)
}

// modeTextsByLocale organise les noms de mode_name_tr par langue puis par clé de mode, pour les
// clés `keys` : modes[langue][clé] = nom. Best-effort comme loadModeNamesForKeys (journalisé et
// consigné). Une langue sans ligne est absente.
func modeTextsByLocale(ctx context.Context, meta *DB, keys []string) map[i18n.Locale]map[string]string {
	out := make(map[i18n.Locale]map[string]string, len(i18n.Locales()))
	if meta == nil || len(keys) == 0 {
		return out
	}
	for _, l := range i18n.Locales() {
		if names := loadModeNamesForKeys(ctx, meta, l, keys); len(names) > 0 {
			out[l] = names
		}
	}
	return out
}

// modeKeySet est l'ensemble des clés de mode (noms normalisés de la langue de repli,
// analysis.NormalizeModeLabel) à traduire par mode_name_tr. Source unique de cette collecte pour
// l'historique, les tuiles de l'Accueil et le canonique.
type modeKeySet map[string]struct{}

// addLabel inscrit la clé d'un libellé de mode (un libellé qui ne se normalise pas est ignoré).
func (s modeKeySet) addLabel(label string) {
	if key := analysis.NormalizeModeLabel(label); key != "" {
		s[key] = struct{}{}
	}
}

// addTexts inscrit la clé du nom de chaque langue d'un actif (un pair_name peut être un UUID : le
// nom exploitable vient alors de asset_translations).
func (s modeKeySet) addTexts(t i18n.Text) {
	for _, l := range i18n.Locales() {
		s.addLabel(strings.TrimSpace(t[l]))
	}
}

// list rend les clés (ordre quelconque).
func (s modeKeySet) list() []string {
	return setKeys(s)
}
