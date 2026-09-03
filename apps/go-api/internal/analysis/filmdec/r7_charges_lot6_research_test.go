package filmdec

// r7_charges_lot6_research_test.go — LE TYPE 36 `action_weapon_fire` (29 038 tetes sur
// 430 046 paquets : le type le plus frequent du corpus) et son jumeau 35
// `request_weapon_fire`.
//
// CE QUI A DEBLOQUE LE TYPE 36. Le repartiteur `FUN_14080a9d4` appelle `vtable+0x68` avec
// `param_5 = 1` : toutes les branches `param_5 == 0` sont MORTES dans le film (le R(32) au
// lieu d'une reference dans la boucle cibles, `FUN_1406cd5b8`, le R(1) final). Et la boucle
// des COMPOSANTES n'est PAS a largeur runtime, contrairement a la reserve du dossier : la
// largeur est `base = 12 si Ncomp==1 sinon 4`, ecrasee a `min(base,6)` quand la cible
// referencee porte kind==1 (`FUN_14102bd24`). Seule la boucle CIBLES porte une reference
// d'entite, dont la largeur est celle, deja resolue, du domaine 1.
//
// CONTRE-EPREUVE INDEPENDANTE : depliee sur le temoin canonique du depot (ref0 domaine 1
// porte=1 sonde=1, refs 1-2 absentes, D garde=0, E garde=0, F garde=1, estBloc=0, z=1), la
// grammaire place l'arme haute aux bits 44..75, l'arme basse 76..107, et la visee au bit
// 113 — EXACTEMENT les offsets valides en production dans `fire_events.go`, et
// « post-comptes = 111 » comme mesure par `fire_aim_modal.go` sur 5 films. Les deux bits du
// « +2 » sont identifies : ce sont les portes c1 (`FUN_140c9e4d8`) et d (`FUN_1408eff64`).
//
// PIEGE D'ORDRE : le flux porte [Ncomp][Ncib] alors que la boucle CIBLES tourne EN PREMIER.

// r7Comptes lit le bloc de comptes du type 36 (FUN_14080cc68) et rend (Ncomp, Ncib).
func r7Comptes(br *BitReader) (int, int) {
	if br.ReadBit() { // z : les deux comptes sont nuls
		return 0, 0
	}
	nComp := 1
	if !br.ReadBit() { // u
		nComp = int(br.ReadBits(4))
	}
	if br.ReadBit() { // z2
		return nComp, 0
	}
	nCib := 1
	if !br.ReadBit() { // u2
		nCib = int(br.ReadBits(4))
	}
	return nComp, nCib
}

// r7Composite consomme le bloc composite O ou P (FUN_140c9e4d8 / FUN_1408eff64) et rend la
// valeur de sa porte. `queue` active la queue propre a O (le bloc c2/c3 puis R(15)+R(7)).
func r7Composite(br *BitReader, queue bool, largeurA, largeurB int) bool {
	porte := br.ReadBit()
	if !porte {
		return false
	}
	switch br.ReadBits(2) { // kind
	case 1:
		r7RefCharge(br, 1)
		if br.ReadBit() {
			br.Skip(6)
		}
	case 2:
		r7RefCharge(br, 2)
	}
	if !queue {
		return true
	}
	if !br.ReadBit() { // c2
		br.Skip(4 + 4)
		if !br.ReadBit() { // c3
			return true
		}
	}
	if !br.ReadBit() { // FUN_14076d528, POLARITE INVERSEE
		br.Skip(largeurA)
	}
	br.Skip(largeurB)
	return true
}

// r7SkipChargeLot6 : les types 36 et 35. Rend false pour tout type non ferme.
func r7SkipChargeLot6(br *BitReader, typ int, ctx r7Ctx) bool {
	switch typ {
	case 36:
		return r7Charge36(br, ctx)
	case 35:
		return r7Charge35(br, ctx)
	}
	return false
}

// r7Charge36 consomme la charge de `action_weapon_fire`.
func r7Charge36(br *BitReader, ctx r7Ctx) bool {
	estCourt := br.ReadBit()
	estBloc := br.ReadBit()
	br.Skip(8) // indice tireur (R(7) puis R(1), FUN_141fcf670)
	r7Porte5(br)
	r7Porte2Inv(br)
	r7Porte32(br)
	br.Skip(32) // arme, moitie basse
	br.Skip(2)  // deux R(1)
	blocHoro := false
	if estBloc {
		br.Skip(1)
		blocHoro = br.ReadBit()
		if blocHoro {
			if br.ReadBit() {
				br.Skip(10)
			}
		}
	}
	if estCourt {
		br.Skip(10) // FUN_14076dc04(largeur=0xa) puis FIN du record
		return true
	}
	nComp, nCib := r7Comptes(br)
	kindUn := map[int]bool{}
	for i := 0; i < nCib; i++ {
		kindUn[i] = br.ReadBits(2) == 1
		br.Skip(1)
		r7RefCharge(br, 1)
	}
	base := 4
	if nComp == 1 {
		base = 12
	}
	dernierQ, vuQ := uint64(1), false
	for i := 0; i < nComp; i++ {
		br.Skip(4)
		if !br.ReadBit() { // p
			continue
		}
		dernierQ, vuQ = br.ReadBits(3), true
		idx := 0
		if nCib < 3 {
			idx = int(br.ReadBits(1))
		} else {
			idx = int(br.ReadBits(4))
		}
		br.Skip(16)
		w := base
		if kindUn[idx] && w > 6 {
			w = 6
		}
		br.Skip(3 * w)
	}
	if !(vuQ && dernierQ == 0) { // porte : la derniere composante a p==1 saute O, P et Q
		c1 := r7Composite(br, true, 15, 7)
		r7Composite(br, false, 0, 0)
		if !c1 {
			br.Skip(30) // VISEE
		}
	}
	return r7Queue36(br, ctx, estBloc, blocHoro)
}

// r7Queue36 consomme la queue commune du type 36 (section R de la grammaire).
func r7Queue36(br *BitReader, ctx r7Ctx, estBloc, blocHoro bool) bool {
	if !blocHoro {
		if br.ReadBit() {
			br.Skip(6)
		}
		br.Skip(6)
	}
	if estBloc {
		switch br.ReadBits(2) { // FUN_1431a0cbc
		case 1:
			br.Skip(19)
		case 0:
			if !r7VecteurQuantifie(br, ctx, 16) {
				return false
			}
		}
		br.Skip(4) // FUN_141102ed0(0x24) = 3 >= 2
	} else {
		br.Skip(2) // FUN_14080cb98
		a2 := br.ReadBit()
		if br.ReadBit() { // b2
			br.Skip(4)
		}
		if a2 {
			for i := 0; i < 2; i++ { // FUN_14320c36c puis FUN_142a40f18
				if br.ReadBit() {
					br.Skip(12)
				}
			}
		}
	}
	br.Skip(6)
	if br.ReadBit() {
		br.Skip(7)
	}
	if br.ReadBit() {
		return r7VecteurQuantifie(br, ctx, 16)
	}
	return true
}

// r7Charge35 consomme la charge de `request_weapon_fire` : strictement sequentiel, sans
// variante courte, sans bloc horodatage, et avec une VISEE INCONDITIONNELLE.
func r7Charge35(br *BitReader, ctx r7Ctx) bool {
	br.Skip(8) // indice tireur
	r7Porte2Inv(br)
	r7Porte32(br)
	br.Skip(32)
	br.Skip(2) // deux R(1)
	nCib := int(br.ReadBits(4))
	for i := 0; i < nCib; i++ {
		br.Skip(2 + 1)
		r7RefCharge(br, 1)
	}
	nComp := int(br.ReadBits(4))
	// FUN_1407eda24 sous FUN_141102ed0(0x23) = 4 >= 2 : w = min(64/Ncomp, 12).
	w := 12
	if nComp > 0 {
		if v := 64 / nComp; v < 12 {
			w = v
		}
	}
	for i := 0; i < nComp; i++ {
		br.Skip(4)
		if !br.ReadBit() {
			continue
		}
		br.Skip(3 + 4 + 16)
		br.Skip(3 * w)
	}
	r7Composite(br, true, 20, 14)
	br.Skip(30) // VISEE, inconditionnelle
	r7Composite(br, false, 0, 0)
	if br.ReadBit() {
		br.Skip(6)
	}
	br.Skip(6)
	br.Skip(2) // FUN_14080cb98
	a2 := br.ReadBit()
	if br.ReadBit() { // b2
		br.Skip(4)
	}
	if a2 {
		for i := 0; i < 2; i++ {
			if br.ReadBit() {
				br.Skip(12)
			}
		}
	}
	br.Skip(6)
	if br.ReadBit() {
		br.Skip(7)
	}
	if br.ReadBit() {
		return r7VecteurQuantifie(br, ctx, 16)
	}
	return true
}
