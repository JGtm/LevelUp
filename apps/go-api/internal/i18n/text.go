package i18n

import (
	"expvar"
	"fmt"
	"maps"
	"sort"
	"strings"
)

// Compteurs de repli (ADR 0009, expvar, `/debug/vars`). Atomiques : un Pick concurrent ne
// demande aucun verrou. `Pick` n'a pas de contexte, donc pas de journal : c'est l'appelant
// qui veut la clé du libellé manquant qui utilise Resolve et journalise lui-même.
var (
	fallbackTotal = expvar.NewInt("levelup.i18n.text_fallback_total")
	emptyTotal    = expvar.NewInt("levelup.i18n.text_empty_total")
)

// Text est un libellé écrit dans plusieurs langues : `Locale -> texte`.
//
// Une table TOML `{ en = "...", fr = "..." }` se décode directement dans un Text (les clés
// sont les codes de `config/locales.toml`). Une langue absente ou vide n'est pas une
// erreur de lecture : Validate la refuse pour les langues `required`, et Pick replie sur
// Fallback pour les autres (noms de jeu, ADR 0040 D-4). Jamais de traduction inventée :
// un texte absent reste absent.
type Text map[Locale]string

// Pick retourne le texte de la langue demandée ; s'il est vide ou absent, celui de
// Fallback ; sinon "". Chaque repli pris est compté (expvar `levelup.i18n.text_fallback_total`),
// et chaque texte sans aucune valeur aussi (`levelup.i18n.text_empty_total`).
func (t Text) Pick(l Locale) string {
	v, _ := t.Resolve(l)
	return v
}

// PickOptional fait comme Pick pour un champ FACULTATIF (description, condition, motif) : un texte qui
// ne porte aucune valeur dans aucune langue rend "" sans incrémenter `levelup.i18n.text_empty_total`
// (l'absence d'un champ facultatif n'est pas un libellé manquant) ; sinon comme Pick, repli compris.
func (t Text) PickOptional(l Locale) string {
	if t.Empty() {
		return ""
	}
	return t.Pick(l)
}

// Resolve fait comme Pick et indique en plus si le repli sur Fallback a été pris
// (`true` seulement quand la valeur rendue n'est pas celle de la langue demandée et n'est
// pas vide). L'appelant qui dispose d'un contexte et de la clé du libellé s'en sert pour
// journaliser le repli (`slog.DebugContext`) avec de quoi retrouver la donnée manquante.
func (t Text) Resolve(l Locale) (text string, fellBack bool) {
	if v := t[l]; v != "" {
		return v, false
	}
	if l != Fallback {
		if v := t[Fallback]; v != "" {
			fallbackTotal.Add(1)
			return v, true
		}
	}
	emptyTotal.Add(1)
	return "", false
}

// Has indique si le texte porte une valeur non vide dans la langue demandée, sans repli
// et sans compter : c'est la question « cette langue est-elle renseignée ? ».
func (t Text) Has(l Locale) bool {
	return t[l] != ""
}

// Clone retourne une copie indépendante (nil reste nil).
func (t Text) Clone() Text {
	if t == nil {
		return nil
	}
	out := make(Text, len(t))
	for l, v := range t {
		out[l] = v
	}
	return out
}

// Trimmed retourne une copie dont les valeurs sont débarrassées de leurs espaces de bord.
// Les chargeurs l'appliquent avant Validate : un libellé d'espaces n'est pas un libellé.
func (t Text) Trimmed() Text {
	if t == nil {
		return nil
	}
	out := make(Text, len(t))
	for l, v := range t {
		out[l] = strings.TrimSpace(v)
	}
	return out
}

// CheckKnown vérifie que chaque clé du texte est une langue de la liste de l'application, sans
// exiger de langue `required`. Pour les tables partielles par nature (noms de jeu : une
// langue absente se replie sur Fallback) où seule une faute de frappe sur le code est à
// refuser. L'erreur nomme toutes les langues inconnues, triées.
func (t Text) CheckKnown() error {
	var unknown []string
	for l := range t {
		if !Known(l) {
			unknown = append(unknown, string(l))
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fmt.Errorf("langue(s) inconnue(s) %q", unknown)
}

// Validate vérifie que chaque langue `required` de la liste de l'application porte un texte
// non vide (espaces de bord exclus), et que chaque clé du texte est une langue connue. L'erreur
// nomme toutes les langues fautives (triées), pas seulement la première : une langue ajoutée à
// `config/locales.toml` fait apparaître d'un coup tous les endroits à compléter.
func (t Text) Validate() error {
	var missing []string
	for _, l := range Locales() {
		if Required(l) && strings.TrimSpace(t[l]) == "" {
			missing = append(missing, string(l))
		}
	}
	unknownErr := t.CheckKnown()
	switch {
	case len(missing) > 0 && unknownErr != nil:
		return fmt.Errorf("langue(s) manquante(s) %q, %w", missing, unknownErr)
	case len(missing) > 0:
		return fmt.Errorf("langue(s) manquante(s) %q", missing)
	}
	return unknownErr
}

// Empty indique si le texte ne porte aucune valeur non vide, dans aucune langue. Sans repli et
// sans compter : c'est la question « ce libellé existe-t-il ? ».
func (t Text) Empty() bool {
	for _, v := range t {
		if v != "" {
			return false
		}
	}
	return true
}

// Ptr rend un pointeur vers le texte de la langue demandée (sans repli, sans compter), ou nil
// quand cette langue n'est pas renseignée. Pour les API historiques qui prennent un `*string`
// où nil veut dire « absent ».
func (t Text) Ptr(l Locale) *string {
	v := t[l]
	if v == "" {
		return nil
	}
	return &v
}

// Set inscrit la valeur d'une langue, en créant le texte s'il est nil. Une valeur vide est
// inscrite telle quelle (Pick la traite comme absente).
func (t *Text) Set(l Locale, v string) {
	if *t == nil {
		*t = Text{}
	}
	(*t)[l] = v
}

// Equal indique si deux textes portent exactement les mêmes langues et les mêmes valeurs (une
// valeur vide et une langue absente sont deux états distincts). Un Text est une table : il n'est
// pas comparable avec `==`.
func (t Text) Equal(o Text) bool {
	return maps.Equal(t, o)
}

// Compare ordonne deux textes langue par langue, dans l'ordre alphabétique des codes de la liste
// (FlatCodes), sur leurs valeurs (une langue absente vaut ""). Ordre TOTAL sur les langues de la
// liste : deux textes qu'il ne sépare pas portent les mêmes valeurs dans chacune. Il sert aux tris
// déterministes qui départageaient autrefois deux champs `En` puis `Fr`. Une clé hors liste n'est
// pas comparée.
func (t Text) Compare(o Text) int {
	for _, l := range FlatCodes() {
		if c := strings.Compare(t[l], o[l]); c != 0 {
			return c
		}
	}
	return 0
}
