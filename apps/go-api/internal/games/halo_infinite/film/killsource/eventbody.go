package killsource

// eventbody.go — LES CORPS D EVENEMENTS, ET SEULEMENT CE QU IL FAUT POUR ENCHAINER.
//
// Le chainage ne sert qu a VALIDER une position de kill-event : il n a besoin que de la
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

// evBody : avance le curseur de la longueur du CORPS de l evenement `code`. Faux = corps non
// modelise ou debordement : la chaine s arrete la.
func evBody(r *evReader, code int, gate15 bool) bool {
	if evStub[code] {
		return true
	}
	if n, ok := evFixed[code]; ok {
		if r.bp+n > len(r.pl)*8 {
			r.over = true
			return false
		}
		r.bp += n
		return true
	}
	if code == killEventCode {
		k := readKillEvent(r.pl, r.bp)
		if k.end < 0 {
			r.over = true
			return false
		}
		r.bp = k.end
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

// killEventCode : le code d evenement du KILL. C est le seul dont le contenu est lu.
const killEventCode = 85

// evBody1 : corps du code 1 (FUN_140968368). 10 a 33 bits. Le << +16 >> qu on lisait autrefois
// n est PAS ici : c est la boucle de presence du dispatcher, donc de l encadrement.
func evBody1(r *evReader) {
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
// flux : il se tranche par film. Le compteur R(10) est une LONGUEUR EN BITS, pas un nombre
// d enregistrements — confirme au desassemblage.
func evBody15(r *evReader, gate15 bool) {
	if gate15 {
		r.rd(15)
	}
	r.rd(13)
	n := int(r.rd(10))
	r.rd(n)
}

// evBody0 : corps du code 0 (FUN_1407f15a4). Modele bit-exact, plancher 84 bits, maximum 241.
func evBody0(r *evReader) {
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
	v58 := int(r.rd(4))
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
func evBody82(r *evReader) bool {
	r.rd(32)
	r.rd(8)
	n1 := int(r.rd(3))
	for i := 0; i < n1; i++ {
		r.rd(32)
		if !evVariantA(r) {
			return false
		}
	}
	if r.g1() == 1 {
		r.rd(32)
		n2 := int(r.rd(3))
		for i := 0; i < n2; i++ {
			evVariantB(r)
		}
	}
	r.rd(32)
	return true
}

// evVariantA : FUN_14080ef08. Le tag 7 est un vecteur monde quantifie par une configuration
// RUNTIME : il n est pas modelisable statiquement, et la chaine s arrete plutot que de deviner.
func evVariantA(r *evReader) bool {
	switch int(r.rd(3)) {
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
func evVariantB(r *evReader) {
	switch int(r.rd(3)) {
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
