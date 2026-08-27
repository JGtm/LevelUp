package main

// actions_delai.go — LE DELAI PORTE PAR UNE EVENT ACTION, et pourquoi il manquait.
//
// LE SIXIEME OUBLI DE FORMAT, MESURE ET CLOS. Le parseur ne lisait d'une action que son type
// et sa cible ; `couchesDeEvent` rend une couche par action et le rendu les sommait toutes a
// t = 0. Le mode `audit-actions` a tranche le layout sur pieces, avec son temoin negatif :
//
//	offset 7 (these)          8 701 / 8 701 actions « jouer » laissent EXACTEMENT 5 octets
//	offset 6 (temoin negatif)     0 / 8 701
//	offset 8 (temoin negatif)     0 / 8 701
//
// et deux identifiants de propriete seulement apparaissent sur les actions : 15 et 16.
//
// LES VALEURS NE SONT PAS DES FLOTTANTS. Lues comme des `float32` elles donnent des
// denormaux absurdes (2,8e-44 ; 5,6e-43 ; 1,5134e-41) ; lues comme des `int32` — ce que
// `AkPropValue` autorise, c'est une union — elles donnent 20, 400, 10 800. Des
// MILLISECONDES rondes. C'est la lecture entiere qui est la bonne, et le test est
// refutable : une mauvaise interpretation ne produirait pas des multiples de 5 ms.
//
//	idProp 15 = delai de l'action        (ms)
//	idProp 16 = duree de fondu           (ms)
//
// CE QUE CA CORRIGE, ET C'EST UN SYMPTOME DATE. L'evenement `71cb04b8` (avant apparition
// d'une nouvelle zone, KOTH) porte DEUX actions : la premiere sans delai, la seconde a
// idProp 15 = 400 ms. L'utilisateur, a l'ecoute du rendu, decrivait « un tres court son au
// debut qui me parait EN TROP » — c'etait la seconde couche remontee de 400 ms sur la
// premiere. L'evenement n'est pas un empilement : c'est un ENCHAINEMENT.

import (
	"encoding/binary"
	"math"
)

// Identifiants de proprietes portes par les Event Actions (valeurs ENTIERES, en ms).
const (
	propActionDelai = 15
	propActionFondu = 16
)

// delaiMaxActionS : borne de plausibilite du delai d'une action. Au-dela, c'est le layout
// qui a derive, pas le jeu qui attend une minute avant de jouer un son.
const delaiMaxActionS = 60.0

// lireDelaiAction rend le delai propre d'une Event Action, en SECONDES.
//
// La lecture n'est retenue que si la charge utile se referme exactement sur les 5 octets
// specifiques d'une action « jouer » — le meme controle que celui qui a valide l'offset.
// Un layout qui aurait derive echoue au lieu de rendre un decalage fantaisiste.
func lireDelaiAction(d []byte) (float32, bool) {
	vals, fin, ok := lirePropsEntieres(d, offsetPropsAction, 1)
	if !ok {
		return 0, false
	}
	// Le paquet RANGED suit ; il se lit sur deux composantes et referme la charge utile.
	_, fin2, ok := lirePropsEntieres(d, fin, 2)
	if !ok || len(d)-fin2 != resteAttenduPlay {
		return 0, false
	}
	ms, present := vals[propActionDelai]
	if !present {
		return 0, true
	}
	s := float64(ms[0]) / 1000
	if s < 0 || s > delaiMaxActionS {
		return 0, false
	}
	return float32(s), true
}

// lirePropsEntieres lit un AkPropBundle en interpretant chaque valeur comme un `int32`.
// Rend les valeurs, l'offset suivant, et si la lecture est structurellement possible.
func lirePropsEntieres(d []byte, off, largeur int) (map[byte][]int32, int, bool) {
	if off < 0 || off >= len(d) || largeur < 1 {
		return nil, off, false
	}
	n := int(d[off])
	if n > 16 {
		return nil, off, false
	}
	debutIDs := off + 1
	debutVals := debutIDs + n
	fin := debutVals + n*4*largeur
	if fin > len(d) {
		return nil, off, false
	}
	out := make(map[byte][]int32, n)
	for i := 0; i < n; i++ {
		comp := make([]int32, largeur)
		for c := 0; c < largeur; c++ {
			comp[c] = int32(binary.LittleEndian.Uint32(d[debutVals+(i*largeur+c)*4:]))
		}
		out[d[debutIDs+i]] = comp
	}
	return out, fin, true
}

// entierDepuisFlottant rend l'entier qu'un `float32` denormal represente reellement, pour
// les rapports d'audit qui ont deja lu la valeur en flottant.
func entierDepuisFlottant(v float32) int32 {
	return int32(math.Float32bits(v))
}
