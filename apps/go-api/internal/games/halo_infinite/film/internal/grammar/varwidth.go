package grammar

// varwidth.go — LES NEUF CATEGORIES DE PLAGE DE `FUN_1406d3140`, RELUES CHEZ L ECRIVAIN
// (lot 1.9.1 bis, pas 2 bis, 2026-09-15, image base 140000000).
//
// # CE QUE CE FICHIER CORRIGE, ET POURQUOI PERSONNE NE L AVAIT VU
//
// `FUN_1406d3140` est l ENTIER A LARGEUR VARIABLE du flux : une valeur de `W` bits suivie de
// deux bits de queue, lue par `object-parent-state` (i10), `equipment-activated` (i21),
// `equipment-control-signal` (i22), la pile de handles d equipement (i28), les boucles de
// slots de l unite, plusieurs etats par defaut. Son prologue est :
//
//	uVar7 = DAT_144706100 ;                                  // 0x1FFF
//	si (DAT_144706104 != 0) { uVar8 = (&DAT_1451f98d0)[param_3*2] ;   // la BASE
//	                          uVar7 = (&DAT_1451f98d4)[param_3*2] ; } // la PLAGE
//	si (param_3 == 1 && R(1) != 0 && DAT_144706104 != 0) {
//	    uVar8 = DAT_1451f98f0 ; uVar7 = DAT_1451f98f4 ; }     // = l entree 4
//	W = FUN_1406d310c(uVar7) ;                               // ceil(log2) de la plage
//
// LE DEPOT LISAIT `W = 13` POUR TOUT LE MONDE parce que la table `DAT_1451f98d0` est NULLE
// dans l image statique, et qu un commentaire de 2026-08 en avait conclu « slot de table
// statiquement nul, donc inutilise ». **C EST L INITIALISEUR QUI MANQUAIT** : `FUN_140d10bb0`
// (appele par `FUN_140d10a78`) remplit les neuf entrees et POSE `DAT_144706104 = 1` —
//
//	140d10bb0:  DAT_144706104 = 1 ; iVar1 = DAT_144706100 ;
//	            i = 0 ; repeter :
//	              i == 0 ou 1 -> plage = iVar1 - 0x200 ; base = 0x200
//	              i == 2      -> plage = 0x100         ; base = 0x200
//	              i == 3      -> plage = 0x100         ; base = 0x300
//	              i == 4      -> plage = 0x200         ; base = 0x200
//	              i == 5      -> plage = 0x100         ; base = 0x400
//	              i == 6      -> plage = 0x200         ; base = 0
//	              sinon (7, 8)-> plage = iVar1         ; base = 0
//	            jusqu a i > 8
//
// — et ce sont des CONSTANTES. Aucune donnee de carte, aucun compte d objets, aucun tag de
// scenario n y entre : c est le point qui distingue ce champ des largeurs d axe d i0. Les deux
// autres ecrivains (`FUN_1408f1618` @1423503d3, `FUN_142f2f0cc` @142f2f2b1) ne touchent QUE les
// entrees 0, 1, 7, 8 et `DAT_144706100` ; les entrees 2 a 6 ne sont ecrites que par
// `FUN_140d10bb0`, donc elles sont invariantes.
//
// CONSEQUENCE : `W` ne vaut 13 que pour les categories 0, 1, 7 et 8 — celles dont la plage
// derive de `DAT_144706100` (0x1FFF, et `bitLen(0x1DFF) == bitLen(0x1FFF) == 13`, donc le
// `-0x200` de l initialiseur et le `-0x1FF` des deux autres sont indiscernables en largeur).
// Pour 2, 3, 5 il vaut 8 ; pour 4 et 6 il vaut 9. Et la categorie 1 BASCULE sur l entree 4
// quand son bit de sonde vaut 1 : elle lit alors 9 bits, pas 13.
//
// LA BASE N EST PAS PORTEE ICI, ET C EST ECRIT : `FUN_1406d3140` rend `(queue << 30) |
// (base + valeur)`, avec une base de 0x200 / 0x300 / 0x400 selon la categorie. Elle ne change
// AUCUN bit lu — elle decale des IDENTIFIANTS publies (`ObjectParentState.FreeID`, la sonde
// « le projectile pointe-t-il son tireur »). La changer se juge au gate de decodage, qui n est
// pas disponible pour ce lot ; consigne au §4 du PLAN_DECODEUR_FILM.

// varWidthDefaultRange est `DAT_144706100` : la plage de config du flux, 0x1FFF dans l image.
// Les trois ecrivains de la table la relisent, aucun ne la calcule a partir d une carte.
const varWidthDefaultRange uint32 = 0x1FFF

// varWidthProbeSlot est l entree sur laquelle la categorie 1 bascule quand son bit de sonde
// vaut 1 (`DAT_1451f98f0` / `DAT_1451f98f4` = l entree 4 de la table, adresse verifiee :
// 1451f98d0 + 4*8 = 1451f98f0).
const varWidthProbeSlot = 4

// varWidthProbeCategory est la SEULE categorie qui depense un bit de sonde. La condition du
// jeu teste `param_3 == 1` en PREMIER : aucune autre categorie ne lit ce bit.
const varWidthProbeCategory = 1

// varWidthRange rend la plage de la categorie `param_3`, telle que `FUN_140d10bb0` l ecrit.
// Toute categorie hors [0, 8] retombe sur la plage par defaut — c est ce que fait le jeu
// quand la garde `DAT_144706104` n est pas levee, et c est la lecture la plus prudente.
func varWidthRange(param3 int) uint32 {
	switch param3 {
	case 0, 1:
		return varWidthDefaultRange - 0x200
	case 2, 3, 5:
		return 0x100
	case 4, 6:
		return 0x200
	default:
		return varWidthDefaultRange
	}
}

// varWidthBits rend `W = FUN_1406d310c(plage)` : le nombre de bits du champ de valeur.
// `bitLen` EST `FUN_1406d310c` (rang du bit de poids fort, plus un si des bits plus bas sont
// mis) — la meme primitive que celle d `object-dissolver`.
func varWidthBits(param3 int) uint {
	w := bitLen(varWidthRange(param3))
	if w < 0 {
		return 0
	}
	return uint(w) //nolint:gosec // bitLen d un uint32 tient dans [0, 32]
}

// CE QUE LE PAS 2 QUATER A TROUVE, ET QUI QUALIFIE TOUT CE QUI PRECEDE (2026-09-15).
//
// `DAT_144706104` — la garde qui decide si la table s'applique — N'EST PAS UNE CONSTANTE.
// `FUN_142e2bfd0`, le lecteur d'etat complet, l'ECRIT DEPUIS LE FLUX en tete de paquet :
//
//	142e2bfd0+:  uVar7 = FUN_1428e1c0c(&DAT_144c23178) ;   // la VERSION du format
//	             si (7 < uVar7) DAT_144706104 = FUN_1406cf008(param_1) ;  // un R(1) DU FILM
//
// Donc : sur un format de version <= 7 la garde garde la valeur posee par `FUN_140d10bb0`
// (1, table ACTIVE) ; au-dela, c'est UN BIT DU FILM qui tranche, une fois par paquet
// d'image-cle.
//
// CORRECTION DU 2026-09-15 (lot 1.9.1 ter) — LA VERSION VIENT DU FILM, ET SON OFFSET EST CONNU.
// Ce commentaire disait « ce bit ne se lit pas offline sans connaitre la version, qui vient d'un
// objet de configuration du jeu et non du film ». C'est FAUX, et la relecture du chargeur le
// montre : `FUN_1428e1c0c` rend `*(structure_du_film + 4)`, et cette structure EST l'image de
// `chunk_00` (`FUN_14299b198` ecrit `film+0` et `film+4` comme les deux premiers u32 du chunk).
// La version est donc `chunk_00+4` — cf. `film_format_version.go`, [FilmFormatVersionFromHeader].
//
// CE QUE CELA DECIDE, ET CE QUE CELA NE DECIDE PAS. Les 1 351 films du cache portent un format
// de 20 a 27 : TOUS sont > 7, donc sur TOUS le bit de garde est ecrit dans le flux, en tete du
// paquet d'image-cle (`FUN_142e2bfd0` @142e2c020) — et sur le chemin DELTA (`FUN_142987460`) il
// l'est sans meme de condition de version. La branche « garde statique a 1 » est donc MORTE pour
// tout film reel, et ce qui reste ouvert n'est plus « peut-on savoir si le bit est lu » (oui,
// toujours) mais « que VAUT-il », film par film. Sa position est le premier bit du paquet, la
// meme que `PacketPreambleBits` porte deja sur le chemin delta (`frame_records.go`), ou il vaut
// 1 dans 100,00 % des 30 418 payloads mesures de `000d5950`. La table reste donc posee.
//
// CE QUE LA MESURE DIT MALGRE TOUT, ET POURQUOI LA TABLE RESTE POSEE. Sur les sept bobines, le
// record de `ti=37` qui se met a fermer ne ferme QUE si i21, i22 ET i28 sont corriges ENSEMBLE
// (mesure d'attribution du pas 2 bis, chaque correction neutralisee a tour de role) : trois
// largeurs independantes qui tombent juste ensemble sur la frontiere ne sont pas une
// coincidence. La table est donc ACTIVE sur ce film. Ce n'est PAS une preuve qu'elle l'est sur
// tous — et c'est pour cela que la descente de trois lignes du golden de fermeture a ete
// remontee au pilote plutot que refigee en silence.
