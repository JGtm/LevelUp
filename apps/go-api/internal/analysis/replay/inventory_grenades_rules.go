package replay

// inventory_grenades_rules.go — LES DEUX VOIES DE LECTURE DES COMPTEURS DE GRENADE (i22) d'un
// record d'image-clé. Elles vivent ici, et non dans inventory_decode.go, pour la seule raison
// du seuil de taille du dépôt (CLAUDE.md n°5) : le fichier voisin était à 500 lignes.
//
// LE MOTIF i22, dans les deux voies : un compteur `R(3)` valant 4 — le nombre de types de
// grenade — suivi de quatre `R(8)` bornés par DefaultGrenadeMax. C'est une GRAMMAIRE, et elle
// est faible : un record en porte 8 à 9 occurrences en moyenne (mesuré sur 1 167 records). Ce
// n'est donc jamais le motif qui identifie i22 : c'est sa POSITION.
//
// R2a — LA VOIE PAR L'ANCRE (invGrenadesAfter), historique et prioritaire. Le premier motif de
// somme non nulle situé après l'ancre de capacité (R1). Elle est exacte quand elle s'applique,
// mais elle n'existe que si R1 existe : la mesure du 2026-08-24
// (MESURE_TROUS_INVENTAIRE_2026-08-24.md) a chiffré ce couplage à 4 278 records sur 6 721
// (63,7 %) ARMÉS — munitions et emplacement dégainé lus — et pourtant sans une seule grenade,
// parce que l'ancre 28 bits n'est présente que dans 19,1 % des records (et dans AUCUN record
// de 11 films sur 24).
//
// R2b — LA VOIE POSITIONNELLE (invGrenadesNearAmmo), le REPLI livré le 2026-08-25. Elle ne doit
// RIEN à l'ancre : elle se borne au bloc de munitions, dont R4 établit le début au bit près par
// un critère de LARGEUR (le parse doit atterrir exactement sur le bit de porte d'i43). i22
// précède ce bloc à une distance mesurée, et c'est cette distance qui est la règle.
//
// LA LOI DE POSITION, ÉTABLIE PAR MESURE AVANT TOUTE IMPLÉMENTATION (24 films, 1 167 records
// où R1, R2a et R4 réussissent tous les trois — donc où la position VRAIE d'i22 est connue) :
//
//	offset(i22) − début du bloc de munitions ∈ [−204, −139]
//
// Cinq repères candidats ont été mis en concurrence sur ce même corpus, et c'est la mesure qui
// a tranché : début de record (76 valeurs distinctes), fin de record (490), dernière famille
// d'arme (81) ne bornent rien. Le début du bloc de munitions concentre sur 22 valeurs, dans une
// plage de 66 bits — et surtout, la règle « le PREMIER motif de la fenêtre » y est exacte
// 1 167 fois sur 1 167.
//
// POURQUOI « PREMIER » ET NON « UNIQUE » : sur le même corpus, exiger l'unicité ne rend que
// 71,5 % des lectures (mais 100 % justes) ; prendre le DERNIER en rend 91,6 % dont seulement
// 78 % justes. Le premier motif de la fenêtre est donc la seule des trois stratégies à être à
// la fois complète et exacte.
//
// LE CONTRÔLE NÉGATIF, qui est ce qui rend la règle réfutable (et la mesure du 2026-08-24 avait
// montré qu'un motif i22 « par hasard » existe bel et bien) : la MÊME règle appliquée à une
// fenêtre décalée de deux largeurs vers l'amont ne rend RIEN — 0 lecture sur les records cibles,
// contre 100 % pour la fenêtre de la loi. Et le candidat parasite le plus proche EN DESSOUS
// d'une position vraie est à 105 bits (minimum sur 1 167 records) : la fenêtre ci-dessous, même
// élargie de 12 bits sous la loi, en reste à plus de 90 bits.
//
// LA SOMME NULLE EST UNE MESURE, et c'est la seconde chose que R2b change. R2a exigeait
// `somme > 0` faute de pouvoir se fier à la position : un Spartan ayant lancé toutes ses
// grenades produit un motif entièrement nul, et la règle le rejetait — « zéro grenade » devenait
// indistinguable d'une non-lecture (104 records sur 104 dans ce cas, mesure du 2026-08-24 §4.4).
// Bornée par la POSITION, R2b n'a plus besoin de ce garde-fou de VALEUR : elle publie le zéro.
// R2a, elle, garde le sien — sans position sûre, il reste son seul rempart.

// invGrenadeWinLo / invGrenadeWinHi bornent la fenêtre de R2b, en bits et RELATIVEMENT au
// premier bit du bloc de munitions (donc négatives : i22 précède le bloc).
//
// LA LOI MESURÉE EST [−204, −139] ; la fenêtre l'élargit de 12 bits de chaque côté pour ne pas
// river la règle aux extrêmes exacts d'un corpus de 24 films. Cet élargissement est SANS COÛT
// mesurable : le parasite le plus proche sous une position vraie est à 105 bits.
const (
	invGrenadeWinLo = -216
	invGrenadeWinHi = -127
)

// invGrenadeCountsAt lit le motif i22 supposé commencer au bit b : `R(3)` puis quatre `R(8)`.
// Rend les compteurs, leur somme, et si la grammaire est respectée.
//
// UNE SEULE COPIE DE LA GRAMMAIRE pour les deux voies : R2a et R2b ne doivent pas pouvoir
// diverger sur ce qu'est un motif i22 — elles ne diffèrent que par l'endroit où elles cherchent
// et par ce qu'elles acceptent comme somme.
func invGrenadeCountsAt(
	pay []byte, b int, maxVal uint32,
) (c [invGrenadeSlots]uint32, sum uint32, ok bool) {
	if invBits(pay, b, 3) != invGrenadeSlots {
		return c, 0, false
	}
	for i := 0; i < invGrenadeSlots; i++ {
		v := invBits(pay, b+3+8*i, 8)
		if v > maxVal {
			return c, 0, false
		}
		c[i] = v
		sum += v
	}
	return c, sum, true
}

// invGrenadesAfter est R2a : le PREMIER motif i22 de somme NON NULLE situé après `from`, qui
// est la position de l'ancre de capacité. Voir l'en-tête pour la raison du critère de somme.
func invGrenadesAfter(
	pay []byte, from, to int, maxVal uint32,
) (c [invGrenadeSlots]uint32, ok bool) {
	for b := from; b+invGrenadeMotifBits <= to; b++ {
		got, sum, valid := invGrenadeCountsAt(pay, b, maxVal)
		if valid && sum > 0 {
			return got, true
		}
	}
	return c, false
}

// invGrenadeMotifBits est la largeur du motif i22 : le compteur R(3) et les quatre R(8).
const invGrenadeMotifBits = 3 + 8*invGrenadeSlots

// invGrenadesNearAmmo est R2b : le PREMIER motif i22 dont le début tombe dans la fenêtre de la
// loi de position, en amont du bloc de munitions. La somme nulle est ACCEPTÉE — la position
// suffit à distinguer « aucune grenade portée » d'une non-lecture.
//
// `ammoStart` vient de R4 et de lui seul ; `from` est le premier bit du record, qui borne la
// fenêtre par la gauche quand le record est plus court qu'elle.
func invGrenadesNearAmmo(
	pay []byte, ammoStart, from int, maxVal uint32,
) (c [invGrenadeSlots]uint32, ok bool) {
	lo, hi := ammoStart+invGrenadeWinLo, ammoStart+invGrenadeWinHi
	if lo < from {
		lo = from
	}
	for b := lo; b <= hi; b++ {
		if got, _, valid := invGrenadeCountsAt(pay, b, maxVal); valid {
			return got, true
		}
	}
	return c, false
}
