package himap

// zones_forge.go — LES ZONES DE CALLOUT D'UNE CARTE FORGE COMME CADRE ET COMME MASQUE du fond
// de carte.
//
// L'EXTRACTION des zones (objets `mapvar.TypeIDZoneNommee` -> polygones monde, rateliers
// ecartes) vit dans `mapvar.ZonesNommeesForge` : elle sert aussi le serveur, qui ne peut pas
// importer ce paquet (decodeur Oodle en cgo). Ce fichier ne garde que ce que la CUISSON du fond
// en fait : les contours pour `MasqueZones`, et les boites de cadrage. Il ne decide pas du
// rognage — cette decision se prend carte par carte, sur la mesure, comme sur la chaine native.

import (
	"math"

	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
)

// ContoursDeZones rend les seuls polygones, dans la forme qu'attend `MasqueZones`.
func ContoursDeZones(zs []mapvar.ZoneNommee) [][][2]float64 {
	if len(zs) == 0 {
		return nil
	}
	out := make([][][2]float64, 0, len(zs))
	for _, z := range zs {
		out = append(out, z.Contour)
	}
	return out
}

// MargeCadreZones : marge gardee autour des zones de callout quand elles servent de CADRE,
// en metres.
//
// Plus large que la marge de rognage (1 m) parce qu elle ne joue pas le meme role : la marge de
// rognage decide ce qu on EFFACE, celle-ci decide ce qu on MONTRE. Un couloir qui borde une zone
// nommee doit rester visible meme s il n est pas nomme.
const MargeCadreZones = 8.0

// BoiteDesZones rend le rectangle monde (minX, minY, maxX, maxY) qui contient toutes les zones
// de callout, elargi de `marge`. Le second retour est faux s il n y a pas de zone.
//
// A QUOI CA SERT, ET POURQUOI CE N EST PAS LA MEME CHOSE QUE LE ROGNAGE AUX ZONES. Le rognage
// efface la matiere hors des zones ; il ne peut rien contre une carte dont les zones elles-memes
// s etalent, ni contre du decor qui traverse une zone. Le CADRE, lui, borne l image a l emprise
// des lieux nommes : c est la reponse a « recadrer avec les zones de callout », demandee par
// l utilisateur le 2026-08-30 sur dix cartes dont le cadre atteignait la butee de 3 000 px.
func BoiteDesZones(zs []mapvar.ZoneNommee, marge float64) ([4]float64, bool) {
	var b [4]float64
	if len(zs) == 0 {
		return b, false
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, z := range zs {
		for _, p := range z.Contour {
			minX, maxX = math.Min(minX, p[0]), math.Max(maxX, p[0])
			minY, maxY = math.Min(minY, p[1]), math.Max(maxY, p[1])
		}
	}
	return [4]float64{minX - marge, minY - marge, maxX + marge, maxY + marge}, true
}

// MargeCadreAncres : marge gardee autour des ancres d objectifs quand elles servent de CADRE.
//
// 25 m, soit une portee de tir : une arene s etend au-dela de ses objectifs, et un cadre colle
// aux ancres couperait les abords ou l on se bat. Bien plus large que la marge des zones (8 m)
// parce que les ancres sont des POINTS, pas des surfaces.
const MargeCadreAncres = 25.0

// BoiteDesAncres rend le rectangle monde qui contient toutes les ancres d objectifs, elargi de
// `marge`. Le second retour est faux s il n y a pas d ancre.
//
// POURQUOI CE CADRE EXISTE, ET QUAND IL EST LE SEUL DISPONIBLE. Mesure du 2026-08-30 : sur dix
// cartes que l utilisateur voulait recadrer aux zones de callout, les zones se sont revelees
// INUTILISABLES comme cadre — leur emprise vaut 266 x 266 m sur Insolence et Smallhalla,
// 254 x 227 m sur Thunderhead, c est-a-dire le canevas entier. Ces cartes-la nomment de grands
// volumes, pas des recoins. Le rognage aux zones n y efface donc presque rien et le cadrage
// aux zones ne borne rien.
//
// Les ANCRES, elles, sont dans le jeu par construction — c est l invariant qui tient tout ce
// chantier. Leur emprise est un cadre honnete la ou le maillage manque (Thunderhead n en publie
// pas) ou ne se lit pas (Insolence : fichier-tag sans section TST1).
func BoiteDesAncres(ancres [][3]float64, marge float64) ([4]float64, bool) {
	var b [4]float64
	if len(ancres) == 0 {
		return b, false
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, a := range ancres {
		minX, maxX = math.Min(minX, a[0]), math.Max(maxX, a[0])
		minY, maxY = math.Min(minY, a[1]), math.Max(maxY, a[1])
	}
	return [4]float64{minX - marge, minY - marge, maxX + marge, maxY + marge}, true
}
