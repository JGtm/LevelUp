package i18n

import (
	"encoding/json"
	"fmt"
)

// Textes stockés dans une langue par clé (ADR 0040, D-3).
//
// Une notification ou une proposition du coach garde ses paramètres en JSON dans une base
// joueur. Un libellé qu'elle porte y est écrit UNE FOIS PAR LANGUE de l'application, sous la
// clé `<base>_<code>` (`title_fr`, `title_en`, `title_es`...). À la lecture, le serveur remplace
// la famille par une seule clé `<base>` dans la langue de la requête : le web ne voit jamais de
// paire par langue. Une ligne écrite avant l'arrivée d'une langue n'a pas la clé de celle-ci ;
// la résolution replie alors sur Fallback, puis Default, puis la première langue renseignée.
//
// Le format stocké est donc le format courant : aucune migration des lignes existantes n'est
// nécessaire quand une langue s'ajoute.

// ParamKey retourne la clé de stockage du libellé `base` dans la langue `l` ("title", "fr" ->
// "title_fr").
func ParamKey(base string, l Locale) string {
	return base + "_" + string(l)
}

// AddParams écrit le texte dans `m`, une clé `<base>_<code>` par langue de l'application ; une
// langue absente du texte est écrite vide (la lecture la replie).
func (t Text) AddParams(m map[string]any, base string) {
	for _, l := range Locales() {
		m[ParamKey(base, l)] = t[l]
	}
}

// ResolveParams retourne une copie de `m` où chaque famille `<base>_<code>` des `bases`
// données est remplacée par une clé `<base>` unique, valeur de la langue `l` (repli : Fallback,
// Default, première langue renseignée). Une base sans aucune clé de langue est laissée telle
// quelle. `m` n'est pas modifiée.
func ResolveParams(m map[string]any, bases []string, l Locale) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	for _, base := range bases {
		found := map[Locale]string{}
		for _, loc := range Locales() {
			key := ParamKey(base, loc)
			if v, ok := m[key]; ok {
				delete(out, key)
				s, _ := v.(string)
				found[loc] = s
			}
		}
		if len(found) == 0 {
			continue
		}
		out[base] = Text(found).pickParam(l)
	}
	return out
}

// pickParam choisit la valeur d'une famille stockée : la langue demandée, Fallback, Default, puis
// la première langue renseignée dans l'ordre de la liste.
func (t Text) pickParam(l Locale) string {
	for _, c := range []Locale{l, Fallback, Default} {
		if v := t[c]; v != "" {
			return v
		}
	}
	for _, c := range Locales() {
		if v := t[c]; v != "" {
			return v
		}
	}
	return ""
}

// ResolveParamsJSON fait comme ResolveParams sur un objet JSON sérialisé. Un JSON vide rend ""
// ; un JSON illisible rend l'erreur (l'appelant la journalise et décide de la dégradation).
func ResolveParamsJSON(raw string, bases []string, l Locale) (string, error) {
	if raw == "" {
		return "", nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return "", fmt.Errorf("i18n: paramètres illisibles : %w", err)
	}
	out, err := json.Marshal(ResolveParams(m, bases, l))
	if err != nil {
		return "", fmt.Errorf("i18n: paramètres non sérialisables : %w", err)
	}
	return string(out), nil
}
