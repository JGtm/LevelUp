package i18n

import (
	"context"
	"embed"
	"expvar"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

// Catalogue des messages rédigés par le serveur (ADR 0040, D-3/D-4).
//
// Un message est une feuille TOML dont les clés sont les codes de `config/locales.toml` :
//
//	[coach.arc.title.combat]
//	fr = "Domination en combat"
//	en = "Combat Mastery"
//
// Les variables s'écrivent `{nom}` ; leur ensemble est identique dans toutes les langues d'un
// message (test de complétude). Une langue ajoutée à la liste exige le message dans cette langue
// pour chaque clé : le test refuse un catalogue incomplet, jamais servi en repli muet. Au
// runtime, une langue qui manquerait quand même retombe sur Fallback, puis Default, et le repli
// est compté.
//
// Les fichiers sont embarqués : le catalogue ne se relit pas à l'exécution. Une structure
// invalide (TOML illisible, langue inconnue, feuille mêlée de branches) arrête le démarrage ;
// la complétude, elle, est vérifiée par les tests.

//go:embed messages/*.toml
var messagesFS embed.FS

// Params porte les valeurs des variables `{nom}` d'un message.
type Params map[string]any

// Compteurs expvar (ADR 0009, `/debug/vars`). Un repli ou une clé absente ne doit jamais rester
// invisible : le garde-rail des clés les interdit à la compilation, ces compteurs disent s'il en
// reste en production.
var (
	messageFallbackTotal     = expvar.NewInt("levelup.i18n.message_fallback_total")
	messageMissingTotal      = expvar.NewInt("levelup.i18n.message_missing_total")
	messageParamMissingTotal = expvar.NewInt("levelup.i18n.message_param_missing_total")
	documentFallbackTotal    = expvar.NewInt("levelup.i18n.document_fallback_total")
)

// CountDocumentFallback compte un document produit (notes de version, aide) servi dans la langue de
// repli faute de fichier dans la langue demandée (ADR 0040, D-6). L'appelant journalise le détail.
func CountDocumentFallback() {
	documentFallbackTotal.Add(1)
}

// messagesDir est le dossier des fichiers du catalogue dans messagesFS.
const messagesDir = "messages"

var catalog = mustLoadCatalog(messagesFS)

// missingLogged retient les clés déjà journalisées : une clé absente appelée à chaque requête
// ne doit pas inonder le journal.
var missingLogged sync.Map

var placeholderRE = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// Msg retourne le message `key` dans la langue `l`, variables remplacées. Une langue sans le
// message retombe sur Fallback puis Default (compté) ; une clé absente du catalogue rend la clé
// elle-même (compté et journalisé une fois) : un texte visible vaut mieux qu'un trou, et le
// garde-rail archlint empêche une clé inconnue d'être écrite.
func Msg(l Locale, key string, p Params) string {
	t, ok := catalog[key]
	if !ok {
		reportMissing(key)
		return key
	}
	tpl := pickTemplate(t, l)
	if tpl == "" {
		reportMissing(key)
		return key
	}
	return fill(tpl, p)
}

// MsgText retourne le message `key` dans TOUTES les langues de l'application, variables
// remplacées : pour les champs de données qui portent un Text (libellés de palier, de chaîne,
// de badge) et sont résolus plus loin dans la langue de la requête.
func MsgText(key string, p Params) Text {
	out := make(Text, len(table))
	for _, l := range Locales() {
		out[l] = Msg(l, key, p)
	}
	return out
}

// MsgTextWith fait comme MsgText quand une variable dépend de la langue (un nom d'axe, un
// mois) : `params` est appelée une fois par langue de l'application.
func MsgTextWith(key string, params func(Locale) Params) Text {
	out := make(Text, len(table))
	for _, l := range Locales() {
		out[l] = Msg(l, key, params(l))
	}
	return out
}

// MsgAll retourne les formes distinctes du message `key` sur toutes les langues, sans variables
// remplacées. Sert à reconnaître un texte écrit par le serveur quel que soit sa langue (un titre
// de section lu dans un fichier).
func MsgAll(key string) []string {
	t, ok := catalog[key]
	if !ok {
		reportMissing(key)
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, l := range Locales() {
		if v := t[l]; v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// MessageKeys retourne toutes les clés du catalogue, triées. Pour les garde-rails.
func MessageKeys() []string {
	keys := make([]string, 0, len(catalog))
	for k := range catalog {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func pickTemplate(t Text, l Locale) string {
	if v := t[l]; v != "" {
		return v
	}
	for _, alt := range []Locale{Fallback, Default} {
		if v := t[alt]; v != "" {
			messageFallbackTotal.Add(1)
			return v
		}
	}
	return ""
}

func fill(tpl string, p Params) string {
	if !strings.Contains(tpl, "{") {
		return tpl
	}
	return placeholderRE.ReplaceAllStringFunc(tpl, func(m string) string {
		name := m[1 : len(m)-1]
		v, ok := p[name]
		if !ok {
			messageParamMissingTotal.Add(1)
			return m
		}
		return fmt.Sprint(v)
	})
}

func reportMissing(key string) {
	messageMissingTotal.Add(1)
	if _, seen := missingLogged.LoadOrStore(key, true); !seen {
		slog.ErrorContext(context.Background(), "i18n: message absent du catalogue", "key", key)
	}
}

func mustLoadCatalog(fsys fs.FS) map[string]Text {
	c, err := loadCatalog(fsys)
	if err != nil {
		panic("i18n: catalogue de messages invalide : " + err.Error())
	}
	return c
}

// loadCatalog lit tous les `*.toml` du dossier et aplatit les tables en clés pointées. Une
// table dont toutes les valeurs sont des chaînes est une feuille (ses clés sont des codes de
// langue connus) ; une table dont toutes les valeurs sont des tables est une branche.
func loadCatalog(fsys fs.FS) (map[string]Text, error) {
	entries, err := fs.ReadDir(fsys, messagesDir)
	if err != nil {
		return nil, err
	}
	out := map[string]Text{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		file := path.Join(messagesDir, e.Name())
		raw, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, err
		}
		var doc map[string]any
		if err := toml.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("%s : %w", file, err)
		}
		if err := flatten("", doc, file, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func flatten(prefix string, node map[string]any, file string, out map[string]Text) error {
	strs, tables := 0, 0
	for _, v := range node {
		switch v.(type) {
		case string:
			strs++
		case map[string]any:
			tables++
		default:
			return fmt.Errorf("%s : %q : valeur ni texte ni table", file, prefix)
		}
	}
	if strs > 0 && tables > 0 {
		return fmt.Errorf("%s : %q mêle texte de langue et sous-tables", file, prefix)
	}
	if strs > 0 {
		return addLeaf(prefix, node, file, out)
	}
	for name, v := range node {
		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		if err := flatten(key, v.(map[string]any), file, out); err != nil {
			return err
		}
	}
	return nil
}

func addLeaf(key string, node map[string]any, file string, out map[string]Text) error {
	if key == "" {
		return fmt.Errorf("%s : texte de langue hors de toute table", file)
	}
	if _, dup := out[key]; dup {
		return fmt.Errorf("%s : clé %q déclarée deux fois", file, key)
	}
	t := make(Text, len(node))
	for code, v := range node {
		l := Locale(code)
		if !Known(l) {
			return fmt.Errorf("%s : clé %q : langue inconnue %q", file, key, code)
		}
		t[l] = v.(string)
	}
	out[key] = t
	return nil
}
