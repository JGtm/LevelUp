package digest

// grammar.go — LA VERSION DE LA GRAMMAIRE, ET LA LIGNE QUI LA PORTE DANS LES FICHIERS FIGES.
//
// # POURQUOI UN MARQUEUR DE VERSION
//
// Un fichier de digests figes ne dit RIEN de la grammaire sous laquelle il a ete ecrit. Quand
// le rendu de ce paquet change, les references d'hier deviennent incomparables — et l'ecart se
// lit comme une REGRESSION DU DECODEUR. Ce n'est pas une hypothese : le 2026-09-02, six des
// neuf TSV du corpus etaient restes sous la v1 pendant que le harnais rendait de la v2, et rien
// dans les fichiers ne permettait de le voir.
//
// La ligne `# digest-grammar: N` ouvre donc chaque fichier de references. Le harnais la lit
// AVANT toute comparaison : une version differente est une panne d'INFRASTRUCTURE nominale
// (« re-figer par -update »), jamais un ecart d'etape.

import (
	"strconv"
	"strings"
)

// GrammarVersion : la version du RENDU de ce paquet. Elle monte des qu'une meme valeur rend
// d'autres octets — c'est-a-dire des que les references figees deviennent incomparables.
//
//	v1 — rendu SANS prefixe de longueur ; paires de map triees par le seul rendu de la cle.
//	v2 — prefixes `s:` (chaine), `b:` (octets imbriques) et `n:` (nul), qui ferment les quatre
//	     collisions constructibles ; et DEPARTAGE des paires de map par le rendu de la VALEUR
//	     quand deux cles rendent les memes octets.
const GrammarVersion = 2

// grammarPrefix ouvre la ligne de version. L'ecriture est UNIQUE (GrammarLine ecrit,
// ParseGrammarLine relit) : un producteur et un lecteur qui divergeraient sur ce litteral
// rendraient le marqueur inutile — c'est exactement le defaut qu'il repare.
const grammarPrefix = "# digest-grammar: "

// GrammarLine rend la ligne a ecrire EN TETE d'un fichier de digests figes.
func GrammarLine() string { return grammarPrefix + strconv.Itoa(GrammarVersion) }

// ParseGrammarLine lit une ligne de version. `ok` est faux quand la ligne n'en est pas une —
// un fichier fige avant l'introduction du marqueur, ou tronque.
func ParseGrammarLine(ligne string) (version int, ok bool) {
	reste, coupe := strings.CutPrefix(strings.TrimSpace(ligne), grammarPrefix)
	if !coupe {
		return 0, false
	}
	v, err := strconv.Atoi(strings.TrimSpace(reste))
	if err != nil {
		return 0, false
	}
	return v, true
}
