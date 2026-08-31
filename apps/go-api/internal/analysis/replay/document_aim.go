package replay

// document_aim.go — LE POINT DE TRAJECTOIRE, ET LES DEUX ANGLES DE LA VISEE.
//
// # POURQUOI CE FICHIER EXISTE
//
// `Point` vivait dans `document.go`, qui depasse le seuil de 500 lignes du projet. Le type y
// est descendu avec la chronique du schema 13 plutot que d'y ajouter encore : la visee
// (cap + elevation) est desormais decrite en un seul endroit, et `document.go` repasse sous
// le seuil. Aucun champ ne change de nom ni de forme au passage.
//
// # CHRONIQUE DU SCHEMA 13 (2026-08-18, PLAN_EXPLOITATION_REGISTRE_FILM lot E phase 1)
//
// La version monte pour UN champ optionnel sur un SOUS-OBJET, ce qui d'ordinaire ne la monte
// pas. La raison est la meme que pour v9 -> v10 (l'origine des poses) : ce n'est pas un champ
// de plus, c'est le SENS DU CONE DE VISEE qui change.
//
// Jusqu'ici le client dessinait le cone a sa longueur maximale sur CHAQUE point porteur de
// cap. Cette longueur affirmait, sans le dire, que le joueur regardait a l'horizontale. Le
// film, lui, transmet l'elevation dans le MEME composant i21 que le cap, et elle n'est pas
// plate : sur les trois films de la mesure E.0.1, le mode de la distribution tombe a 1024 /
// 1013 / 1006 (le centre theorique, ou quelques degres dessous — on vise des corps, pas des
// yeux) mais le support s'etend sur [537, 1490], soit environ +/- 82 deg. Un artefact v12
// fait donc dessiner a plat des visees qui plongent ou qui montent, et la reprise du backfill
// se faisant par SchemaVersion, il continuerait a le faire sans montee de version.
//
// # CE QUE LA MESURE A ETABLI, ET CE QU'ELLE N'A PAS PU ETABLIR
//
// La convention est MESUREE (item E.0.1, journal `registre_film/LOTEF_PHASE0.md` §1) :
// binaire decale centre sur 1024, quantum 360/2048 = 0,17578 deg par pas — DEUX FOIS celui du
// cap, le candidat 180/2048 etant refute a 3,34x et 4,06x la meilleure somme des carres —
// positif = vers le HAUT. L'oracle est le kill : au moment du tir le reticule est sur la
// victime, donc l'angle vise est celui de la geometrie entre les deux bipedes au meme instant.
// Signe : 56 accords sur 58 kills a |dz| >= 1 m ; echelle : r = 0,930 / 0,916 / 0,969 sur
// 164 kills ; controle de bout en bout de l'accesseur contre la geometrie : ecart median
// 0,82 / 0,66 / 0,67 deg.
//
// RESERVE PUBLIEE, la meme que celle de l'accesseur (`filmdec.BipedPosition.AimPitchDeg`) :
// toutes les valeurs observees tiennent dans la MOITIE centrale du champ, si bien que « le
// champ couvre +/- 180 deg et le jeu borne le tangage » et « le champ ne code que +/- 90 deg »
// rendent EXACTEMENT les memes degres sur tout ce que le film transmet. La formule publiee est
// la premiere ; une valeur hors de cette moitie la remettrait en cause.

// Point est une position echantillonnee au pas de temps T. X/Y = plan horizontal de la
// carte ; Z (optionnel) = altitude, pour l'indication d'etage — non critique au rendu 2D.
type Point struct {
	T int     `json:"t"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
	// H (optionnel) est le CAP DE VISÉE en degrés dans le plan XY, même origine et même
	// sens que atan2(Y, X) : 0 = +X, 90 = +Y. Décodé du composant i21
	// (unit-desired-aiming-vector) du même record que la position, donc au même instant.
	// Présent sur ~44 % des points (le record ne réplique pas toujours la visée) ; le
	// client oriente alors le marqueur sur son déplacement, ou pas du tout.
	// PIÈGE omitempty évité à l'écriture : un cap qui s'arrondirait à 0 est publié comme
	// 360 (même angle), sans quoi il serait omis et relu comme « pas de visée ».
	H float32 `json:"h,omitempty"`
	// P (optionnel) est l'ELEVATION DE VISEE en degres : positif = vers le HAUT, 0 = a plat.
	// Decodee du R(11) qui suit le cap dans le composant i21, donc du MEME record que la
	// position et que `H` — meme instant, meme couverture.
	//
	// LE CONTRAT DE L'ABSENCE, ET C'EST LUI QUI COMPTE : `p` absent avec `h` present se lit
	// « A PLAT », jamais « inconnu ». C'est le PIEGE omitempty documente sur `H` ci-dessus,
	// et ici il est ASSUME au lieu d'etre contourne : une elevation qui s'arrondit a 0 est
	// omise, parce que 0 EST la valeur par defaut du lecteur. La regle d'ecriture (cf.
	// `pitchForJSON`) est donc explicite : publie quand |p| >= 0,05 deg, omis en dessous.
	// Contrairement au cap, aucune valeur de repli n'est possible — 0 et 360 sont le meme
	// cap, mais 0 et 180 ne sont pas la meme elevation.
	//
	// ARRONDI AU DIXIEME DE DEGRE : le quantum de la source vaut 0,17578 deg (360/2048), une
	// decimale ne perd donc rien et divise par deux le cout du champ dans l'artefact.
	//
	// CE QUE LE CLIENT EN FAIT : le cone de visee garde son ANGLE et sa LONGUEUR porte
	// l'elevation, signe compris — `AIM_LENGTH x (1 + 0,55 x sin(p))`, ecrete a +/-90 deg,
	// court vers le bas, long vers le haut. Le modele precedent multipliait par
	// `max(0,35 ; cos(p))`, PAIR donc muet sur le sens, ce qui obligeait a coller un tick a la
	// pointe du cone ; ce tick a ete retire le 2026-08-29 (demande utilisateur) en meme temps
	// que le cosinus. Convention, oracle et reserve : en tete de ce fichier et sur
	// `filmdec.BipedPosition.AimPitchDeg`.
	P float32 `json:"p,omitempty"`
	// Sh (optionnel) est la FRACTION DE BOUCLIER dans [0, 1], décodée du composant i5
	// (object-shield-vitality) du MÊME record que la position — donc au même instant.
	// Présente sur ~16 % des points : le film ne réplique le bouclier que lorsqu'il change.
	//
	// POINTEUR, PAS float32 : c'est le PIÈGE omitempty documenté ailleurs dans ce fichier,
	// et ici il serait fatal — un bouclier à ZÉRO est l'information la plus utile de tout
	// le champ (bouclier brisé), et `float32 + omitempty` l'omettrait exactement comme une
	// absence de mesure. Un pointeur n'est omis que s'il est nil, donc « 0 » reste publié.
	//
	// CE QUE LE CHAMP GARANTIT (mesuré sur le film 000d5950, cf. cmd/tmp_vitals) :
	// les 27 404 quanta observés tombent TOUS dans [0, 64], c'est-à-dire exactement la plage
	// [0, 1] d'un bouclier standard, alors que la sérialisation en autorise 0..255 (25,4 %
	// attendus d'un champ lu au mauvais endroit). C'est le témoin décisif du décodage.
	// Témoin du MOMENT, sur une source indépendante (les instants de mort viennent des fins
	// de vie des trajectoires, pas du bouclier) : bouclier médian 0,00 dans la demi-seconde
	// avant une mort contre 0,23 ailleurs, écart jamais atteint par 10 000 permutations des
	// étiquettes. NUANCE PUBLIÉE : le test binaire « bouclier nul ? » ne donne que 1,32x
	// (50,5 % contre 38,2 %) — le film ne réplique le bouclier que lorsqu'il CHANGE, une
	// mesure isolée est donc déjà une mesure de combat.
	//
	// CE QU'IL NE GARANTIT PAS : la RECHARGE. Le témoin de remontée monotone ÉCHOUE
	// (4 suites croissantes réelles contre 7 pour le même échantillon dont on a mélangé
	// l'ordre) : l'échantillonnage est trop lâche pour lire une régénération.
	Sh *float32 `json:"sh,omitempty"`
	// Hp (optionnel) est la FRACTION DE VIE dans [0, 1] (composant i4, object-body-vitality),
	// même record, même instant. Même choix de pointeur, même raison.
	//
	// PUBLIÉ MAIS NON DESTINÉ À UNE BARRE : la couverture est de 0,6 % (974 points sur
	// 171 826). Le décodage est crédible — les 974 quanta tombent tous dans la moitié
	// POSITIVE de la plage sérialisée [-1, +1] (49,6 % attendus au hasard), et la médiane
	// passe de 0,79 chez un joueur vivant à 0,55 dans la demi-seconde avant sa mort
	// (p < 10⁻⁴ par permutation des étiquettes). Mais à 0,6 % de couverture, toute barre
	// affichée serait, 99 % du temps, une valeur périmée présentée comme actuelle.
	Hp *float32 `json:"hp,omitempty"`
	// S (optionnel) est le PALIER DE LUNETTE en vigueur : absent ou 0 = le joueur n'est PAS
	// a la lunette ; 1 et au-dela = il l'est. Le grossissement n'y est pas — il appartient a
	// l'ARME, le film ne transmet que le palier.
	//
	// D'OU IL VIENT, ET POURQUOI C'EST UNE AUTRE SOURCE QUE `H` ET `P`. Le cap et l'elevation
	// sont lus dans le record de position ; la lunette, elle, est un EVENEMENT
	// (`unit_zoom`, type 21) porte par la liste d'evenements en tete de paquet. C'est un etat
	// A BASCULE : une entree vaut jusqu'a la sortie, et non jusqu'au prochain echantillon.
	// La reconstruction (maintien borne, sous-comptage des sorties assume) est decrite sur
	// `filmdec.ZoomStateAt`.
	//
	// CE QUI L'A ETABLI — VERITE TERRAIN, pas inference. L'utilisateur a releve a la main
	// dans Theater, en premiere personne, les six periodes de lunette d'un joueur sur le film
	// 00162144. Les evenements decodes apparient les SIX debuts a moins de 1,2 s (41 -> 41,7 ;
	// 49 -> 49,6 ; 61 -> 60,6 ; 68 -> 69,2 ; 71 -> 71,5 ; 85 -> 85,7), et le controle par
	// translation de la chronologie sur ~3 200 decalages temoins — maximum repris sur les
	// 58 unites a chaque decalage — n'atteint JAMAIS ce score : p = 0,00 %. Le pont vers le
	// joueur est une fermeture, pas une correlation : index de reference + 512 tombe sur un
	// slot bipede existant dans 98 % des cas, contre 0 % pour toute autre base.
	//
	// CE QUE LE CLIENT EN FAIT : le cone de visee RESSERRE SON OUVERTURE — il ne change PAS
	// de longueur, celle-ci etant deja portee par l'elevation `P`. Les deux mecaniques sont
	// orthogonales et lisibles ensemble : un joueur qui vise a la lunette vers le bas a un
	// cone a la fois plus etroit et plus court.
	//
	// RESERVE PUBLIEE : seuls les evenements portes EN TETE d'un paquet sont lus aujourd'hui.
	// Les sorties de lunette sont donc sous-comptees, ce que le maintien borne compense en
	// refusant d'affirmer au-dela d'un delai. Une absence de `s` se lit « pas a la lunette »,
	// jamais « inconnu » — meme contrat que `P`.
	S int `json:"s,omitempty"`
}
