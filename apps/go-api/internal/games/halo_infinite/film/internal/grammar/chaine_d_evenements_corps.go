package grammar

// chaine_d_evenements_corps.go — LES CORPS D EVENEMENTS DE LA CHAINE DU RATTRAPAGE DES KILLS
// (`chaine_d_evenements.go`), ET SEULEMENT CE QU IL FAUT POUR ENCHAINER.
//
// Le chainage ne sert qu a VALIDER une position de message de kill : il n a besoin que de la
// LONGUEUR des evenements, jamais de leur contenu. Un code non modelise arrete la chaine — il
// n existe AUCUNE extrapolation, et c est ce qui rend le critere honnete : un faux positif ne
// peut pas se faire passer pour une chaine en profitant d une longueur devinee.
//
// COUVERTURE ASSUMEE : 28 codes sur 123. Le seuil de 3 evenements a ete mesure SOUS CETTE
// COUVERTURE (RE_LOG 7ter.25 (3)) — l elargir changerait le seuil, pas seulement le rappel.
//
// LE CODE 36 EST VOLONTAIREMENT ABSENT. Son modele mesure sort 1/11 : il est FAUX. Le porter
// ferait avancer le curseur d une longueur erronee, ce qui est pire qu arreter la chaine — une
// chaine trop courte perd un candidat, une chaine fausse en fabrique un.

// evStub dit si `code` est l un des 13 codes dont le deserialiseur est un `return 1` — corps de ZERO
// bit.
func evStub(code int) bool {
	switch code {
	case 3, 4, 23, 24, 25, 26, 33, 49, 54, 57, 59, 92, 103:
		return true
	}
	return false
}

// evFixed rend la longueur de CORPS du code `code` quand elle est fixe, mesuree contre l oracle du
// dispatcher sous la grammaire corrigee. Les codes a un seul echantillon (12, 21, 34, 40) sont
// plausibles mais NON confirmes statistiquement — c est ecrit ici plutot que dissimule.
func evFixed(code int) (int, bool) {
	switch code {
	case 5:
		return 111, true
	case 6:
		return 93, true
	case 7:
		return 118, true
	case 9:
		return 36, true
	case 12:
		return 94, true
	case 21:
		return 2, true
	case 34:
		return 59, true
	case 38:
		return 10, true
	case 40:
		return 78, true
	case 75:
		return 54, true
	case 76:
		return 104, true
	}
	return 0, false
}

// evBody : avance le curseur de la longueur du CORPS de l evenement `code`. Faux = corps non
// modelise ou debordement : la chaine s arrete la. Le message de kill est le seul dont le contenu
// est lu ([lireLeKillDeLaChaine]).
func evBody(r *curseurEv, code int, gate15 bool) bool {
	if evStub(code) {
		return true
	}
	if n, ok := evFixed(code); ok {
		r.skip(n)
		return !r.over
	}
	if code == GenreJoueurTue {
		_, fin := lireLeKillDeLaChaine(r.octets(), r.pos())
		if fin < 0 {
			r.over = true
			return false
		}
		r.aller(fin)
		return true
	}
	switch code {
	case 0:
		evBody0(r)
	case 1:
		evBody1(r)
	case 15:
		evBody15(r, gate15)
	case 82:
		if !evBody82(r) {
			return false
		}
	default:
		return false
	}
	return !r.over
}

// evBody1 : corps du code 1 (FUN_140968368). 10 a 33 bits. Le << +16 >> qu on lisait autrefois
// n est PAS ici : c est la boucle de presence du dispatcher, donc de l encadrement.
func evBody1(r *curseurEv) {
	r.rd(5)
	if r.g1() == 0 {
		r.rd(4)
	}
	r.rd(3)
	if r.g1() != 0 {
		r.rd(19)
	}
}

// evBody15 : corps du code 15 (FUN_14080bb4c). `gate15` est un etat RUNTIME du jeu, pas un bit du
// flux : il se tranche par film ([trancherGate15]). Le compteur R(10) est une LONGUEUR EN BITS, pas
// un nombre d enregistrements — confirme au desassemblage.
func evBody15(r *curseurEv, gate15 bool) {
	if gate15 {
		r.rd(15)
	}
	r.rd(13)
	n := int(r.rd(10)) //nolint:gosec // R(10)
	r.rd(n)
}

// evBody0 : corps du code 0 (FUN_1407f15a4). Modele bit-exact, plancher 84 bits, maximum 241.
func evBody0(r *curseurEv) {
	if r.g1() != 0 {
		r.rd(32)
	}
	if r.g1() == 0 {
		r.rd(5)
	}
	r.rd(19)
	if r.g1() != 0 {
		r.rd(19)
		r.rd(12)
	}
	r.rd(5)
	r.rd(5)
	r.rd(6)
	if r.g1() != 0 {
		r.rd(5)
	}
	r.rd(14)
	if r.g1() != 0 {
		r.rd(32)
	}
	r.rd(1)
	r.rd(3)
	r.rd(5)
	r.rd(5)
	r.rd(1)
	r.rd(4)
	if r.g1() == 0 {
		r.rd(10)
	}
	v58 := int(r.rd(4)) //nolint:gosec // R(4)
	if r.g1() != 0 {
		r.rd(32)
	}
	if v58 == 1 {
		r.rd(8)
	}
	r.rd(4)
	if r.g1() != 0 {
		r.rd(13)
		r.rd(2)
	}
}

// evBody82 : corps du code 82. Deux listes de variantes, chacune dispatchee sur un tag de 3 bits.
func evBody82(r *curseurEv) bool {
	r.rd(32)
	r.rd(8)
	n1 := int(r.rd(3)) //nolint:gosec // R(3)
	for range n1 {
		r.rd(32)
		if !evVariantA(r) {
			return false
		}
	}
	if r.g1() == 1 {
		r.rd(32)
		n2 := int(r.rd(3)) //nolint:gosec // R(3)
		for range n2 {
			evVariantB(r)
		}
	}
	r.rd(32)
	return true
}

// evVariantA : FUN_14080ef08. Le tag 7 est un vecteur monde quantifie par une configuration
// RUNTIME : il n est pas modelisable statiquement, et la chaine s arrete plutot que de deviner.
func evVariantA(r *curseurEv) bool {
	switch int(r.rd(3)) { //nolint:gosec // R(3)
	case 0: // ecrit un octet de sortie, zero bit lu
	case 1, 2, 3, 6:
		r.rd(32)
	case 4:
		r.rd(1)
	case 5:
		r.skip(128)
	default:
		return false
	}
	return true
}

// evVariantB : FUN_1407f0ebc.
func evVariantB(r *curseurEv) {
	switch int(r.rd(3)) { //nolint:gosec // R(3)
	case 0: // ecrit un octet, zero bit
	case 1:
		if r.g1() == 0 {
			r.rd(5)
		}
	case 2:
		if r.g1() == 0 {
			r.rd(32)
		} else {
			r.rd(24)
		}
	default:
		r.rd(32)
	}
}
