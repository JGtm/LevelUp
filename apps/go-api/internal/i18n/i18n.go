// Package i18n porte la liste des langues de l'application et la normalisation d'une langue
// demandée (ADR 0040).
//
// Couche domaine : le package n'importe aucun package interne, pour que `domain`, `ctxkeys`,
// les handlers et les services en dépendent sans cycle. La liste vient de `config/locales.toml`
// par `locales_gen.go` (généré, jamais édité) ; le Go ne lit pas le TOML à l'exécution.
package i18n

import (
	"context"
	"slices"
	"strings"
)

// Locale est la sous-étiquette primaire BCP-47 d'une langue, en minuscules ("fr", "en").
// Une valeur construite hors Parse n'est pas garantie listée : Known la vérifie.
type Locale string

// entry est une ligne de la table générée.
type entry struct {
	code     Locale
	intl     string
	halo     []string
	required bool
}

// Locales retourne les langues de l'application dans leur ordre d'affichage.
func Locales() []Locale {
	out := make([]Locale, len(table))
	for i, e := range table {
		out[i] = e.code
	}
	return out
}

// Known indique si la langue figure dans la liste de l'application.
func Known(l Locale) bool {
	_, ok := lookup(l)
	return ok
}

// Required indique si la langue doit couvrir 100 % des textes de l'interface.
func Required(l Locale) bool {
	e, ok := lookup(l)
	return ok && e.required
}

// Parse normalise une langue demandée : espaces retirés, minuscules, sous-étiquette primaire
// ("en-US", "en_us", "EN", "en-GB,fr;q=0.8" donnent "en"). Une langue vide ou absente de la
// liste donne Default : jamais de repli binaire fr/en, jamais de valeur hors liste.
func Parse(s string) Locale {
	if l, ok := Lookup(s); ok {
		return l
	}
	return Default
}

// Lookup normalise comme Parse mais signale si la langue demandée figure dans la liste ; sinon
// elle retourne ("", false). Pour l'appelant qui ne veut pas confondre « inconnue » avec
// Default (noms de jeu : une langue sans données retombe sur Fallback, pas sur Default).
func Lookup(s string) (Locale, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.IndexAny(s, "-_,; "); i >= 0 {
		s = s[:i]
	}
	l := Locale(s)
	if !Known(l) {
		return "", false
	}
	return l, true
}

// IntlTag retourne l'étiquette BCP-47 de formatage (nombres, dates) de la langue ; une langue
// inconnue prend celle de Default.
func IntlTag(l Locale) string {
	return resolve(l).intl
}

// HaloTags retourne les étiquettes Halo de la langue, par ordre de préférence ; une langue
// inconnue prend celles de Default. Le résultat est une copie.
func HaloTags(l Locale) []string {
	return append([]string(nil), resolve(l).halo...)
}

// Chain retourne l'ordre de lecture d'une donnée par langue pour la langue demandée : elle-même,
// puis la langue de repli, puis la langue par défaut, sans doublon. Une langue inconnue prend
// Default. Pour les lecteurs qui ordonnent des SOURCES (colonnes, tables de traduction) ; un
// texte déjà rassemblé dans un Text se choisit par Text.Pick.
func Chain(l Locale) []Locale {
	if !Known(l) {
		l = Default
	}
	out := []Locale{l}
	for _, c := range []Locale{Fallback, Default} {
		if !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	return out
}

// FromHaloTag retourne la langue de l'application qui porte l'étiquette Halo `tag` (« fr-FR »,
// « es-MX ») ; à défaut d'étiquette exacte, la langue de sa sous-étiquette primaire (« fr »,
// « en-GB » donnent fr, en). ok vaut false quand aucune langue de la liste ne la porte.
func FromHaloTag(tag string) (Locale, bool) {
	tag = strings.TrimSpace(tag)
	for _, e := range table {
		for _, h := range e.halo {
			if strings.EqualFold(h, tag) {
				return e.code, true
			}
		}
	}
	return Lookup(tag)
}

type contextKey struct{}

// WithContext place la langue de la requête dans le contexte. Les services qui localisent des
// libellés la relisent par FromContext sans la threader dans chaque signature.
func WithContext(ctx context.Context, l Locale) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext retourne la langue posée par WithContext, Default si le contexte n'en porte pas.
func FromContext(ctx context.Context) Locale {
	if l, ok := ctx.Value(contextKey{}).(Locale); ok && Known(l) {
		return l
	}
	return Default
}

func lookup(l Locale) (entry, bool) {
	for _, e := range table {
		if e.code == l {
			return e, true
		}
	}
	return entry{}, false
}

func resolve(l Locale) entry {
	if e, ok := lookup(l); ok {
		return e
	}
	e, _ := lookup(Default)
	return e
}
