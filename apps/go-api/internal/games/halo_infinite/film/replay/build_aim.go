package replay

// build_aim.go — L'ECRITURE DES DEUX ANGLES DE VISEE DANS LE DOCUMENT.
//
// Les deux helpers vivaient dans `build.go`, qui depasse le seuil de 500 lignes du projet ;
// ils en sortent avec l'arrivee du second angle plutot que d'y ajouter (`build.go` passe de
// 585 a 581 lignes au net, cablage du nouvel angle compris). Ils sont ici ENSEMBLE parce
// qu'ils font le meme geste sur la meme mesure — arrondir au dixieme de degre — et qu'ils
// le font en SENS CONTRAIRE face a `omitempty` : c'est le seul endroit du code ou cette
// opposition se lit d'un coup d'oeil.
//
// La forme publiee et la convention mesuree sont documentees sur les champs eux-memes
// (`document_aim.go`), la convention de decodage sur `filmdec.BipedPosition.AimPitchDeg`.

import "math"

// headingForJSON arrondit le cap au dixième de degré (la visée est quantifiée à
// 360/4096 ≈ 0,088°, une décimale ne perd donc rien) et évite le PIÈGE omitempty : un cap
// qui s'arrondit à 0 serait omis et relu comme « pas de visée ». On publie 360, qui est le
// même cap et reste sérialisé.
func headingForJSON(v float32) float32 {
	r := float32(math.Round(float64(v)*10) / 10)
	if r <= 0 {
		return 360
	}
	return r
}

// pitchForJSON arrondit l'élévation de visée au dixième de degré et APPLIQUE LE CONTRAT DE
// L'ABSENCE : sous 0,05° en valeur absolue, elle rend 0, donc `omitempty` l'omet.
//
// C'est le contraire du geste de headingForJSON, et c'est voulu. Un cap qui s'arrondit à 0
// doit être REPÊCHÉ (on publie 360, le même angle) parce que son omission se relirait « pas
// de visée ». Une élévation qui s'arrondit à 0 est, elle, exactement ce que son omission
// veut dire : À PLAT. Aucun repêchage n'est d'ailleurs possible — 0 et 360 sont le même cap,
// 0 et 180 ne sont pas la même élévation.
//
// Le seuil de 0,05° est celui de l'arrondi lui-même (la moitié du dernier chiffre publié),
// pas un seuil de mesure : il ne jette rien que la décimale n'aurait déjà jeté. Le quantum de
// la source vaut 0,17578° — trois pas de quantum séparent donc « publié » de « omis ».
func pitchForJSON(v float32) float32 {
	r := float32(math.Round(float64(v)*10) / 10)
	if r == 0 { // -0 compris : le zéro négatif de math.Round s'omet comme le zéro positif
		return 0
	}
	return r
}

// zoomHoldUS borne le MAINTIEN d'un palier de lunette, en microsecondes.
//
// POURQUOI UN PLAFOND EXISTE. Les sorties de lunette sont sous-comptees par le scanner (seuls
// les evenements portes EN TETE d'un paquet sont lus, cf. `filmdec.ScanFilmZoomEvents`). Deux
// entrees consecutives sur un meme slot prouvent qu'une sortie a echappe a la lecture — sans
// plafond, la premiere periode s'etendrait jusqu'a la seconde entree, et une entree jamais
// refermee tiendrait jusqu'a la fin du match. Le plafond transforme une lacune de mesure en
// ABSENCE D'AFFIRMATION, ce qui est le bon defaut : on cesse de dire « il est a la lunette »
// plutot que de le dire a tort.
//
// LA VALEUR EST CALIBREE SUR LA DONNEE, PAS SUR LA VERITE TERRAIN — la distinction est
// importante : se caler sur la chronologie relevee a la main reviendrait a s'ajuster a la
// reponse que la mesure doit justement contröler. Elle vient donc de la distribution des
// periodes REELLEMENT FERMEES (une entree suivie de sa sortie, toutes deux lues) sur le film
// de reference : 139 periodes, mediane 1,13 s, p75 1,75 s, p90 2,65 s, **p95 3,49 s**,
// maximum 5,72 s. Le plafond retient le p95 arrondi : il couvre la quasi-totalite des periodes
// reelles et eteint les entrees orphelines en quelques secondes.
const zoomHoldUS uint64 = 3_500_000
