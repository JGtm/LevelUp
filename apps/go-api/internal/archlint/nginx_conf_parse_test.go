// Package archlint — nginx_conf_parse_test.go : lecture de la syntaxe nginx pour le
// garde-rail nginx_conf_test.go (directives, blocs, héritage, sélection de location).
// Couvre les constructions des configurations versionnées (directives, blocs, chaînes
// entre guillemets, commentaires) ; une accolade ou une chaîne non fermée fait échouer la
// lecture, donc le garde-rail.
package archlint

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ngxToken est un mot de la configuration ; quoted distingue "" (valeur vide) d'un mot.
type ngxToken struct {
	text   string
	quoted bool
	line   int
}

// ngxNode est une directive (nom, arguments, bloc éventuel) ; la racine n'a pas de nom.
type ngxNode struct {
	name   string
	args   []ngxToken
	line   int
	parent *ngxNode
	body   []*ngxNode
}

func (n *ngxNode) arg(i int) string {
	if i < len(n.args) {
		return n.args[i].text
	}
	return ""
}

func (n *ngxNode) all(name string) []*ngxNode {
	var out []*ngxNode
	for _, c := range n.body {
		if c.name == name {
			out = append(out, c)
		}
	}
	return out
}

func (n *ngxNode) last(name string) *ngxNode {
	all := n.all(name)
	if len(all) == 0 {
		return nil
	}
	return all[len(all)-1]
}

// effective rend la directive la plus proche en remontant les blocs englobants
// (héritage nginx des directives simples).
func (n *ngxNode) effective(name string) *ngxNode {
	for cur := n; cur != nil; cur = cur.parent {
		if d := cur.last(name); d != nil {
			return d
		}
	}
	return nil
}

// effectiveProxyHeaders rend les proxy_set_header en vigueur : ceux du bloc le plus
// proche qui en déclare (un bloc qui en déclare un seul n'hérite plus d'aucun).
func (n *ngxNode) effectiveProxyHeaders() map[string]string {
	for cur := n; cur != nil; cur = cur.parent {
		decl := cur.all("proxy_set_header")
		if len(decl) == 0 {
			continue
		}
		out := map[string]string{}
		for _, d := range decl {
			out[strings.ToLower(d.arg(0))] = d.arg(1)
		}
		return out
	}
	return map[string]string{}
}

func (n *ngxNode) bodyCap() (int64, bool) {
	d := n.last("client_max_body_size")
	if d == nil {
		return 0, false
	}
	return parseNginxSize(d.arg(0)), true
}

// effectiveBodyCap rend le plafond de corps en vigueur et dit s'il est déclaré par la
// location elle-même.
func (n *ngxNode) effectiveBodyCap() (int64, bool) {
	if size, ok := n.bodyCap(); ok {
		return size, true
	}
	if d := n.effective("client_max_body_size"); d != nil {
		return parseNginxSize(d.arg(0)), false
	}
	return nginxDefaultBodyCap, false
}

func (n *ngxNode) listensTLS() bool {
	for _, d := range n.all("listen") {
		if strings.Contains(d.arg(0), "443") {
			return true
		}
	}
	return false
}

func (n *ngxNode) matcher() string {
	parts := make([]string, len(n.args))
	for i, a := range n.args {
		parts[i] = a.text
	}
	return strings.Join(parts, " ")
}

func (n *ngxNode) walkLocations(visit func(*ngxNode)) {
	for _, c := range n.body {
		if c.name == "location" {
			visit(c)
			c.walkLocations(visit)
		}
	}
}

// selectLocation applique la sélection de nginx aux locations de premier niveau :
// correspondance exacte, puis plus long préfixe (définitif s'il porte ^~), puis
// première expression régulière qui correspond, sinon le plus long préfixe.
func (n *ngxNode) selectLocation(uri string) *ngxNode {
	var prefix *ngxNode
	prefixLen := -1
	for _, loc := range n.all("location") {
		switch loc.arg(0) {
		case "=":
			if loc.arg(1) == uri {
				return loc
			}
		case "~", "~*":
		case "^~":
			if strings.HasPrefix(uri, loc.arg(1)) && len(loc.arg(1)) > prefixLen {
				prefix, prefixLen = loc, len(loc.arg(1))
			}
		default:
			if strings.HasPrefix(uri, loc.arg(0)) && len(loc.arg(0)) > prefixLen {
				prefix, prefixLen = loc, len(loc.arg(0))
			}
		}
	}
	if prefix != nil && prefix.arg(0) == "^~" {
		return prefix
	}
	for _, loc := range n.all("location") {
		if loc.arg(0) != "~" && loc.arg(0) != "~*" {
			continue
		}
		pattern := loc.arg(1)
		if loc.arg(0) == "~*" {
			pattern = "(?i)" + pattern
		}
		if re, err := regexp.Compile(pattern); err == nil && re.MatchString(uri) {
			return loc
		}
	}
	return prefix
}

// parseNginxSize lit une taille nginx (« 1m », « 600m », « 16k », « 2g », octets nus) ;
// 0 (illimité) et une valeur illisible valent l'infini.
func parseNginxSize(s string) int64 {
	s = strings.ToLower(strings.TrimSpace(s))
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "k"):
		mult, s = 1<<10, strings.TrimSuffix(s, "k")
	case strings.HasSuffix(s, "m"):
		mult, s = 1<<20, strings.TrimSuffix(s, "m")
	case strings.HasSuffix(s, "g"):
		mult, s = 1<<30, strings.TrimSuffix(s, "g")
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 1 << 62
	}
	return n * mult
}

// parseNginxConf construit l'arbre des directives (commentaires `#` ignorés).
func parseNginxConf(src string) (*ngxNode, error) {
	toks, err := tokenizeNginx(src)
	if err != nil {
		return nil, err
	}
	root := &ngxNode{}
	cur := root
	var pending *ngxNode
	for _, tok := range toks {
		switch {
		case !tok.quoted && tok.text == ";":
			if pending == nil {
				return nil, fmt.Errorf("ligne %d : « ; » sans directive", tok.line)
			}
			cur.body = append(cur.body, pending)
			pending = nil
		case !tok.quoted && tok.text == "{":
			if pending == nil {
				return nil, fmt.Errorf("ligne %d : « { » sans directive", tok.line)
			}
			cur.body = append(cur.body, pending)
			cur, pending = pending, nil
		case !tok.quoted && tok.text == "}":
			if pending != nil || cur.parent == nil {
				return nil, fmt.Errorf("ligne %d : « } » inattendue", tok.line)
			}
			cur = cur.parent
		case pending == nil:
			pending = &ngxNode{name: tok.text, line: tok.line, parent: cur}
		default:
			pending.args = append(pending.args, tok)
		}
	}
	if pending != nil || cur != root {
		return nil, fmt.Errorf("configuration tronquée (directive ou bloc non fermé)")
	}
	return root, nil
}

func tokenizeNginx(src string) ([]ngxToken, error) {
	var toks []ngxToken
	line := 1
	runes := []rune(src)
	for i := 0; i < len(runes); {
		c := runes[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case c == '#':
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
		case c == ';' || c == '{' || c == '}':
			toks = append(toks, ngxToken{text: string(c), line: line})
			i++
		case c == '"' || c == '\'':
			start := line
			var b strings.Builder
			i++
			for i < len(runes) && runes[i] != c {
				if runes[i] == '\\' && i+1 < len(runes) {
					i++
				}
				if runes[i] == '\n' {
					line++
				}
				b.WriteRune(runes[i])
				i++
			}
			if i >= len(runes) {
				return nil, fmt.Errorf("ligne %d : chaîne non fermée", start)
			}
			i++
			toks = append(toks, ngxToken{text: b.String(), quoted: true, line: start})
		default:
			start := i
			for i < len(runes) && !strings.ContainsRune(" \t\r\n;{}", runes[i]) {
				i++
			}
			toks = append(toks, ngxToken{text: string(runes[start:i]), line: line})
		}
	}
	return toks, nil
}
