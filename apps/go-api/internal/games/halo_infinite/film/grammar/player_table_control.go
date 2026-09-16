package grammar

// player_table_control.go — LE CONTROLE DU PROFIL PAR LE FILM (lot 1.5.2).
//
// D-3 d'ADR 0034 : le profil DIT la largeur du bloc de personnalisation, et le film ne la dit
// pas — il la CONTROLE. Ce fichier porte ce controle, et rien d'autre : il compare ce que le
// balayage mesure a ce que la grammaire predit, range chaque ecart dans sa famille, et publie la
// transposition modale du film dans l'unite du profil. AUCUNE LECTURE N'EN DEPEND : si le
// controle contredisait le profil, la table serait lue quand meme, et la contradiction comptee.
//
// Sorti de `player_table.go` au passage des 500 lignes (CLAUDE.md, seuil 5) : deplacement PUR.

// controlerCalibrage confronte le PROFIL a ce que le FILM mesure, et ne decide rien.
//
// IL TRAVAILLE SUR LE BALAYAGE, PAS SUR LA MARCHE, et c'est ce qui le rend informatif : la
// marche avance de la longueur PREDITE, donc y mesurer « mesure - predit » rendrait zero par
// construction. Le balayage, lui, trouve les enregistrements sans rien supposer de leur longueur.
//
// Pour chaque couple de candidats consecutifs, l'ecart mesure est compare a la longueur predite :
// egal = accord ; surplus valant un nombre ENTIER de slots vacants, chacun verifie par le
// predicat grammatical a sa position calculee = vacant ; intervalle qui contient un
// enregistrement que la MARCHE a lu et que le balayage n'a pas vu = invisible ; tout le reste =
// CONTRADICTION, comptee et lisible par un relecteur, jamais corrigee en silence (D-3, D-14).
//
// IL N'Y A PAS DE FAMILLE « PARASITE », ET C'EST UNE CONSEQUENCE DU CRITERE D'ACCEPTATION, pas un
// oubli. Une position PARASITE du balayage (motif d'en-tete fortuit) se reconnait a son champ de
// nom illisible, et elle est ecartee AVANT d'arriver ici, par le meme filtre que `candidatsReels`.
// Ce qui reste est exactement l'ensemble que la marche doit VISITER EN ENTIER pour etre recevable
// (`chercherDepart`) : les deux bouts de chaque couple sont donc toujours retenus. Une famille
// « un bout non retenu » serait inatteignable — mesure sur les 1 351 films du cache : 0.
//
// La transposition MODALE est publiee comme calibrage du film, dans l'unite du profil. La MODE
// est choisie plutot que la moyenne parce qu'elle est insensible aux quelques ecarts que les
// vacants et les enregistrements invisibles deforment.
func controlerCalibrage(d []byte, candidats []int, finBit, persoBits int, slots []PlayerSlot,
	rep *PlayerTableReport) {
	ctl := controleEcarts{d: d, vide: slotVacantBits(persoBits), finBit: finBit,
		retenus: positionsRetenues(slots)}
	surplus := map[int]int{}
	var precedent slotEnr
	vuPrecedent := false
	for _, c := range candidats {
		e, ok := decodeSlot(d, c, finBit, persoBits)
		if !ok || !gamertagImprimable(e.slot.Gamertag) {
			continue
		}
		if vuPrecedent {
			apres := precedent.slot.Bit + longueurPredite(precedent, persoBits)
			surplus[c-apres]++
			ctl.classer(precedent.slot.Bit, c, apres, c-apres, rep)
		}
		precedent, vuPrecedent = e, true
	}
	modal, gaps := modeEcart(surplus)
	rep.FilmDeltaBits = modal + rep.ProfileDeltaBits
	rep.FilmDeltaGaps = gaps
	rep.CalibrationAgrees = gaps > 0 && modal == 0
}

// controleEcarts porte ce qu'il faut pour ranger un ecart dans sa famille.
type controleEcarts struct {
	d            []byte
	vide, finBit int
	retenus      map[int]bool
}

// classer range l'ecart entre les candidats `a` et `b` dans l'une des quatre familles.
func (ctl controleEcarts) classer(a, b, apres, surplus int, rep *PlayerTableReport) {
	switch {
	case surplus == 0:
		rep.GapsAgree++
	case surplus > 0 && surplus%ctl.vide == 0 && ctl.tousVacants(apres, surplus/ctl.vide):
		rep.GapsVacant++
	case ctl.retenuEntre(a, b):
		rep.GapsHidden++
	default:
		rep.GapsContradict++
	}
}

// retenuEntre dit si la marche a lu un enregistrement STRICTEMENT entre `a` et `b` : le balayage
// ne l'a pas vu, la grammaire l'a atteint.
func (ctl controleEcarts) retenuEntre(a, b int) bool {
	for p := range ctl.retenus {
		if p > a && p < b {
			return true
		}
	}
	return false
}

// tousVacants verifie `n` enregistrements vacants consecutifs a partir de `a`.
func (ctl controleEcarts) tousVacants(a, n int) bool {
	for k := 0; k < n; k++ {
		if !slotVacant(ctl.d, a+k*ctl.vide, ctl.finBit) {
			return false
		}
	}
	return true
}

// modeEcart rend la valeur la plus frequente d'une distribution d'ecarts et le nombre d'ecarts
// qu'elle couvre. A egalite, la plus petite valeur, pour que la mesure soit reproductible.
func modeEcart(m map[int]int) (valeur, couverts int) {
	for k, n := range m {
		if n > couverts || (n == couverts && k < valeur) {
			valeur, couverts = k, n
		}
	}
	return valeur, couverts
}

// positionsRetenues rend l'ensemble des positions de bit que la marche a retenues.
func positionsRetenues(slots []PlayerSlot) map[int]bool {
	out := make(map[int]bool, len(slots))
	for _, s := range slots {
		out[s.Bit] = true
	}
	return out
}
