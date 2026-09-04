// Package modelabel — L'APPARIEMENT D'UN JETON DE MODE DANS UN LIBELLÉ, et il n'y en a
// qu'un dans le dépôt.
//
// POURQUOI CE PAQUET EXISTE, ET POURQUOI IL EST UNE FEUILLE. La règle d'appariement (le
// jeton doit apparaître comme MOT ENTIER, insensible à la casse ; le jeton le plus LONG
// gagne) est écrite une seule fois, ici. Elle était jusqu'au 2026-09-03 privée du paquet
// `analysis` — or `games/mappings` en a besoin à son tour pour la table `[score_timeline]`
// de `regulation.toml`, et il ne peut pas importer `analysis` (`analysis` importe déjà
// `games/mappings` : cycle). La sortir dans un paquet FEUILLE — aucune dépendance hors
// `strings` — donne aux deux le même appariement, plutôt qu'une deuxième implémentation
// qui divergerait au premier ajustement (règle CLAUDE.md n°6).
//
// IL PORTE AUSSI LE RETRAIT DU SUFFIXE DE CARTE (`StripMapSuffix`), pour la même raison :
// l'appariement du bloc « Score dans le temps » a besoin du retrait SANS le reste de la
// normalisation — celle-ci mange le jeton de mode sur toute une famille de pair_name
// (« Super Fiesta:Slayer » devient « Super Fiesta », 429 matchs du registre local).
//
// `analysis.ExtractKnownMode` et l'étape 3 de `analysis.NormalizeModeLabel` délèguent ici,
// et aucun de leurs appelants n'a bougé.
package modelabel

import "strings"

// ExtractKnownMode retrouve un jeton de mode CANONIQUE à l'intérieur d'un libellé déjà
// normalisé (typiquement par analysis.NormalizeModeLabel) mais non reconnu tel quel —
// « Legacy Slayer BR » → « Slayer », « Tactical Slayer » → « Slayer ».
//
// Cherche le jeton connu LE PLUS LONG apparaissant comme MOT ENTIER (insensible à la
// casse) : c'est ce qui fait gagner « Super Fiesta » sur « Fiesta » quand les deux sont
// déclarés. Retourne "" si aucun jeton ne matche — l'appelant garde alors son repli.
//
// Fonction pure (aucun accès DB, aucune allocation hors la mise en minuscules).
func ExtractKnownMode(label string, knownTokens []string) string {
	label = strings.TrimSpace(label)
	if label == "" || len(knownTokens) == 0 {
		return ""
	}
	low := strings.ToLower(label)
	best := ""
	for _, m := range knownTokens {
		m = strings.TrimSpace(m)
		if m == "" || len(m) <= len(best) {
			continue // garde le match le plus long (ex. "Super Fiesta" > "Fiesta")
		}
		if wholeWordIndex(low, strings.ToLower(m)) >= 0 {
			best = m
		}
	}
	return best
}

// wholeWordIndex retourne l'index de needle dans haystack en exigeant des frontières
// de mot (lettres/chiffres) de part et d'autre, sinon -1. Évite que "Slayer" matche
// au milieu d'un autre mot. haystack/needle doivent être déjà en minuscules ASCII.
func wholeWordIndex(haystack, needle string) int {
	for from := 0; from <= len(haystack)-len(needle); {
		i := strings.Index(haystack[from:], needle)
		if i < 0 {
			return -1
		}
		idx := from + i
		beforeOK := idx == 0 || !isWordChar(haystack[idx-1])
		end := idx + len(needle)
		afterOK := end >= len(haystack) || !isWordChar(haystack[end])
		if beforeOK && afterOK {
			return idx
		}
		from = idx + 1
	}
	return -1
}

// isWordChar : caractère de mot ASCII (a-z, 0-9, _). haystack est déjà en minuscules.
func isWordChar(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}
