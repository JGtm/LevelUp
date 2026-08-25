package replay

// inventory_rules_test.go — LES QUATRE REGLES D ANCRAGE DE LA GRAMMAIRE D INVENTAIRE.
//
// POURQUOI CE FICHIER. `inventory_test.go` couvrait le sous-bloc de MUNITIONS sur un flux
// fabrique ; les quatre regles qui decident OU lire ne l etaient pas du tout — sept fonctions a
// 0 % de couverture, le plus gros trou du paquet. Or ce sont elles qui portent toute la valeur
// du decodage : le bloc de munitions se parse bien, encore faut-il le chercher au bon endroit.
//
// LES REGLES, TELLES QU ELLES ONT ETE ENONCEES AVANT TOUTE CONFRONTATION AU TERRAIN (c est la
// seule facon qu un accord signifie quelque chose) :
//
//	R1 capacite  ancre 28 bits 0x8CAC57A, puis dans les 60 bits suivants le motif 20 bits
//	             0b00000000000000010010 ; les 3 bits qui suivent sont l index. Retenue
//	             SEULEMENT si l ancre est UNIQUE dans le record.
//	R2 grenades  PREMIER motif i22 (R(3)=4 puis quatre R(8) bornes, somme > 0) situe APRES
//	             l ancre de capacite — dont la position vient de R1, donc sans aucune
//	             information de grenade.
//	R3 armes     familles connues trouvees dans le record, dans l ORDRE DES BITS.
//	R4 munitions le bloc i30..i42 se termine EXACTEMENT sur le bit de porte d i43, juste avant
//	             la premiere famille d arme. Critere de LARGEUR, jamais de contenu.
//
// CE QUE CES TESTS FONT, ET CE QU ILS NE FONT PAS. Ils verifient la MECANIQUE de chaque regle
// sur des flux construits bit a bit — ou l on sait ce qui doit etre trouve et ce qui ne doit pas
// l etre. Ils ne verifient AUCUNE valeur de jeu : la confrontation au terrain (8/8 sur la
// grenade portee, 8/8 sur le nom de la capacite, 7/7 sur chargeur et reserve) est une mesure
// datee, elle ne se rejoue pas dans un test. Ce qui se rejoue, et qui est ici, c est le
// RESULTAT DES REGLES SUR LE BINAIRE REEL de la mini-bobine.

import "testing"

// abilityAnchorBits / abilityPatternBits : les largeurs des deux constantes de R1.
const (
	abilityAnchorBits  = 28
	abilityPatternBits = 20
	abilityIndexBits   = 3
)

// buildAbilityRecord ecrit [ancre][gap][motif][index] precede et suivi de 1 — le remplissage a
// UN est deliberé : aucune fenetre de 1 ne peut valoir l ancre ni le motif, donc tout ce que la
// regle trouve, c est ce qu on y a mis.
func buildAbilityRecord(gap int, index uint32, anchors int) []byte {
	w := &bitWriter{}
	for a := 0; a < anchors; a++ {
		w.put(invAbilityAnchor, abilityAnchorBits)
		for i := 0; i < gap; i++ {
			w.put(1, 1)
		}
		w.put(invAbilityPattern, abilityPatternBits)
		w.put(index, abilityIndexBits)
		for i := 0; i < 32; i++ { // separation entre deux ancres
			w.put(1, 1)
		}
	}
	return w.buf
}

// TestR1AbilityAnchorThenPattern : l ancre, puis le motif dans les 60 bits, puis l index.
func TestR1AbilityAnchorThenPattern(t *testing.T) {
	pay := buildAbilityRecord(0, 5, 1)
	hits := invAbilityIn(pay, 0, len(pay)*8)
	if len(hits) != 1 {
		t.Fatalf("%d occurrence(s) de l ancre, attendu 1 : %+v", len(hits), hits)
	}
	if hits[0].low != 5 {
		t.Errorf("bits bas du rang %d, attendu 5 — les 3 bits lus ne suivent plus le motif",
			hits[0].low)
	}
	if got := invAbilityRankOf(hits[0].low); got != 21 {
		t.Errorf("rang reconstruit %d, attendu 21 — le motif porte deja les bits de poids fort",
			got)
	}
	if hits[0].anchorBit != 0 {
		t.Errorf("l ancre est rapportee au bit %d, attendu 0 — R2 cherche les grenades APRES "+
			"cette position, un decalage la ferait chercher au mauvais endroit", hits[0].anchorBit)
	}
}

// TestR1PatternMustFollowWithinSixtyBits : la fenetre de 60 bits est une BORNE, pas un detail.
//
// Sans elle, la regle trouverait le motif n importe ou dans le record et l index lu serait celui
// d un autre composant. Le test place le motif juste apres la borne et exige un refus.
func TestR1PatternMustFollowWithinSixtyBits(t *testing.T) {
	if hits := invAbilityIn(buildAbilityRecord(60, 5, 1), 0, 1<<20); len(hits) != 1 {
		t.Fatalf("motif a +60 bits : %d occurrence(s), attendu 1 (la borne est inclusive)", len(hits))
	}
	if hits := invAbilityIn(buildAbilityRecord(61, 5, 1), 0, 1<<20); len(hits) != 0 {
		t.Errorf("motif a +61 bits : %d occurrence(s), attendu 0 — la fenetre de 60 bits ne "+
			"borne plus rien", len(hits))
	}
}

// TestR1DoubleAnchorIsNotDisambiguated : DEUX ancres = une lecture qu on ne sait pas departager.
//
// La regle exige l unicite, et le decodeur ne publie alors NI capacite NI grenades (R2 depend de
// la position de R1). C est la forme que prend ici la doctrine du chantier : on ne tranche pas
// au hasard, on ne publie pas.
func TestR1DoubleAnchorIsNotDisambiguated(t *testing.T) {
	hits := invAbilityIn(buildAbilityRecord(0, 5, 2), 0, 1<<20)
	if len(hits) != 2 {
		t.Fatalf("%d occurrence(s) pour deux ancres, attendu 2", len(hits))
	}
	if len(hits) == 1 {
		t.Error("un record a deux ancres ne doit jamais produire UNE occurrence : ce serait un " +
			"departage silencieux")
	}
}

// buildGrenadeBlock ecrit un motif i22 : R(3)=4 puis quatre R(8). Le remplissage est a UN, qui
// ne peut pas produire le prefixe 100.
func buildGrenadeBlock(lead int, counts [4]uint32, tail int) []byte {
	w := &bitWriter{}
	for i := 0; i < lead; i++ {
		w.put(1, 1)
	}
	w.put(4, 3)
	for _, c := range counts {
		w.put(c, 8)
	}
	for i := 0; i < tail; i++ {
		w.put(1, 1)
	}
	return w.buf
}

// TestR2GrenadesAreSearchedAfterTheAbilityAnchor : LE POINT DE DEPART EST LA PREUVE.
//
// R2 ne cherche qu APRES l ancre de capacite, dont la position a ete etablie sans aucune
// information de grenade. Un motif identique place AVANT ce point doit rester invisible : c est
// ce qui empeche la regle de se valider elle-meme.
func TestR2GrenadesAreSearchedAfterTheAbilityAnchor(t *testing.T) {
	pay := buildGrenadeBlock(0, [4]uint32{2, 1, 0, 0}, 64)
	if got, ok := invGrenadesAfter(pay, 0, len(pay)*8, DefaultGrenadeMax); !ok {
		t.Fatal("le motif place au debut doit etre trouve quand la recherche part de 0")
	} else if got != [4]uint32{2, 1, 0, 0} {
		t.Errorf("compteurs %v, attendu [2 1 0 0]", got)
	}
	if _, ok := invGrenadesAfter(pay, 8, len(pay)*8, DefaultGrenadeMax); ok {
		t.Error("un motif ANTERIEUR au point de depart a ete trouve : R2 ne partirait plus de " +
			"l ancre, et sa position cesserait d etre independante des grenades")
	}
}

// TestR2RejectsImplausibleCounters : les bornes ecartent les motifs qui ressemblent a i22.
//
// UN SPARTAN PORTE DEUX GRENADES PAR TYPE. La borne ne contraint pas une valeur reelle : elle
// rejette les coincidences. Et la somme non nulle interdit le motif tout-a-zero, qui se trouve
// partout dans un flux.
func TestR2RejectsImplausibleCounters(t *testing.T) {
	if _, ok := invGrenadesAfter(buildGrenadeBlock(0, [4]uint32{5, 0, 0, 0}, 64),
		0, 1<<20, DefaultGrenadeMax); ok {
		t.Error("un compteur a 5 (borne 2) a ete accepte : la borne ne discrimine plus")
	}
	if _, ok := invGrenadesAfter(buildGrenadeBlock(0, [4]uint32{0, 0, 0, 0}, 64),
		0, 1<<20, DefaultGrenadeMax); ok {
		t.Error("le motif tout-a-zero a ete accepte : il se trouve partout dans un flux, " +
			"l accepter reviendrait a lire du bourrage comme un inventaire")
	}
}

// TestR2StopsAtTheFirstMatch : le PREMIER motif, pas le meilleur.
//
// La regle ne choisit pas : elle prend le premier. Un choix supposerait un critere de qualite,
// donc une preference, donc un arbitrage — exactement ce que ce chantier retire partout.
func TestR2StopsAtTheFirstMatch(t *testing.T) {
	w := &bitWriter{}
	w.put(4, 3)
	for _, c := range []uint32{1, 0, 0, 0} {
		w.put(c, 8)
	}
	for i := 0; i < 64; i++ {
		w.put(1, 1)
	}
	w.put(4, 3)
	for _, c := range []uint32{2, 2, 2, 2} {
		w.put(c, 8)
	}
	got, ok := invGrenadesAfter(w.buf, 0, w.n, DefaultGrenadeMax)
	if !ok {
		t.Fatal("aucun motif trouve alors que deux sont ecrits")
	}
	if got != [4]uint32{1, 0, 0, 0} {
		t.Errorf("compteurs %v : c est le SECOND motif qui a ete retenu, or la regle prend le "+
			"premier", got)
	}
}

// Deux familles reelles du catalogue de production, choisies distinctes jusqu au dernier bit.
const (
	famMA40     uint32 = 0x48C19D2D
	famSidekick uint32 = 0xF408190F
)

// TestR3FirstKnownFamilyInBitOrder : l ORDRE DES BITS, et rien d autre.
func TestR3FirstKnownFamilyInBitOrder(t *testing.T) {
	w := &bitWriter{}
	const lead = 40
	for i := 0; i < lead; i++ {
		w.put(1, 1)
	}
	w.put(famMA40, 32)
	for i := 0; i < 16; i++ {
		w.put(1, 1)
	}
	w.put(famSidekick, 32)
	known := map[uint32]bool{famMA40: true, famSidekick: true}
	at, ok := invFirstFamily(w.buf, 0, w.n, known)
	if !ok {
		t.Fatal("aucune famille trouvee alors que deux sont ecrites")
	}
	if at != lead {
		t.Errorf("premiere famille au bit %d, attendu %d — c est cette position qui borne le "+
			"bloc de munitions par la DROITE (R4) ; un decalage deplace tout le parse", at, lead)
	}
	// Le catalogue fait la selectivite : sans la famille, le balayage ne trouve rien.
	if _, ok := invFirstFamily(w.buf, 0, w.n, map[uint32]bool{famSidekick: true}); !ok {
		t.Error("la seconde famille n est pas trouvee quand la premiere sort du catalogue")
	}
	if _, ok := invFirstFamily(w.buf, 0, w.n, map[uint32]bool{0xDEADBEEF: true}); ok {
		t.Error("une famille absente du catalogue a ete reconnue : le predicat ne borne plus rien")
	}
}

// TestR4AmmoBlockLandsOnTheGateBitOfTheFirstFamily : LE CRITERE EST UNE LARGEUR.
//
// Le bloc doit atterrir EXACTEMENT sur le bit de porte d i43, juste avant la premiere famille.
// Aucune valeur des emplacements 0 et 1 — ceux que la confrontation au terrain mesure — n entre
// dans le critere : sans quoi on choisirait la lecture qui donne le resultat attendu.
func TestR4AmmoBlockLandsOnTheGateBitOfTheFirstFamily(t *testing.T) {
	pay, firstFamilyBit, mag, res := buildRecordWithAmmoThenFamily(25, 75, 1)
	inv := KeyframeInventory{AbilityRank: -1, DrawnSlot: -1}
	readAmmo(pay, &inv, 0, firstFamilyBit)
	if !inv.AmmoRead {
		t.Fatal("aucun bloc de munitions resolu : aucun debut n atterrit sur la porte d i43")
	}
	if inv.Ammo[0].Mag == nil || *inv.Ammo[0].Mag != mag {
		t.Errorf("chargeur lu %v, attendu %d — le depart retenu n est pas le bon", inv.Ammo[0].Mag, mag)
	}
	if inv.Ammo[0].Res == nil || *inv.Ammo[0].Res != res {
		t.Errorf("reserve lue %v, attendu %d", inv.Ammo[0].Res, res)
	}
	if inv.DrawnSlot != 1 {
		t.Errorf("emplacement degaine %d, attendu 1", inv.DrawnSlot)
	}
	if inv.AmmoCandidates < 1 {
		t.Errorf("%d candidat(s) publie(s) : le departage doit rester VISIBLE, meme quand il "+
			"n a pas eu lieu", inv.AmmoCandidates)
	}
}

// TestR4WithoutAFamilyNothingIsRead : sans borne droite, pas de lecture.
//
// R4 s appuie entierement sur R3. Un record dont aucune famille n est connue ne rend AUCUNE
// munition — et c est juste : lire un bloc sans savoir ou il finit, c est lire au hasard.
func TestR4WithoutAFamilyNothingIsRead(t *testing.T) {
	pay, _, _, _ := buildRecordWithAmmoThenFamily(25, 75, 1)
	if _, ok := invFirstFamily(pay, 0, len(pay)*8, map[uint32]bool{0x00000001: true}); ok {
		t.Fatal("le banc d essai porte une famille du catalogue de controle : le test ne " +
			"mesure pas ce qu il croit")
	}
	inv := KeyframeInventory{AbilityRank: -1, DrawnSlot: -1}
	if inv.AmmoRead {
		t.Fatal("etat initial incoherent")
	}
	// Aucune famille -> `keyframeInventories` n appelle jamais readAmmo. On verifie la porte
	// elle-meme : un bloc dont la borne droite tombe AVANT son debut ne rend rien.
	readAmmo(pay, &inv, 400, 100)
	if inv.AmmoRead {
		t.Error("un bloc a ete lu alors que sa borne droite precede son debut")
	}
}

// buildRecordWithAmmoThenFamily ecrit [bourrage][bloc de munitions][porte i43][famille].
// Rend le payload, le bit de la premiere famille, et les valeurs plantees.
func buildRecordWithAmmoThenFamily(mag, res uint32, sel int) ([]byte, int, uint32, uint32) {
	w := &bitWriter{}
	const lead = 20
	for i := 0; i < lead; i++ {
		w.put(1, 1)
	}
	m := mag
	writeAmmoSlot(w, &m, nil, res)
	writeAmmoSlot(w, nil, nil, 0)
	writeAmmoSlot(w, nil, nil, 0)
	writeAmmoSlot(w, nil, nil, 0)
	w.put(0, 3) // i42 : en-tete
	w.put(0, 1) // porte active-bas : valeur presente
	w.put(uint32(sel), 2)
	w.put(1, 1) // seconde porte, fermee
	w.put(0, 1) // LA PORTE d i43 : c est sur ce bit que le bloc doit atterrir
	familyBit := w.n
	w.put(famMA40, 32)
	for i := 0; i < 32; i++ {
		w.put(1, 1)
	}
	return w.buf, familyBit, mag, res
}

// buildGrenadeSelTail ecrit [off bits a UN][masque 6b][selection 3b][tail a UN] : la queue de
// record apres la fin de la derniere famille (famEnd = 0 dans ces tests). Le remplissage a UN
// ne peut pas valoir un masque valide (ses deux bits hauts sont toujours nuls) : tout ce que la
// regle trouve, c est ce qu on y a mis.
func buildGrenadeSelTail(off int, mask, sel uint32, tail int) []byte {
	w := &bitWriter{}
	for i := 0; i < off; i++ {
		w.put(1, 1)
	}
	w.put(mask, 6)
	w.put(sel, 3)
	for i := 0; i < tail; i++ {
		w.put(1, 1)
	}
	return w.buf
}

// TestR5GrenadeSelectionInWindow : le motif i47 est lu DANS la fenetre, refuse au-dela.
func TestR5GrenadeSelectionInWindow(t *testing.T) {
	gren := [invGrenadeSlots]uint32{0, 2, 1, 0} // rangs 1 et 2 portes : masque 0b000110
	pay := buildGrenadeSelTail(invGrenadeSelLo, 0b110, 3, 40)
	if got := invGrenadeSelection(pay, 0, len(pay)*8, gren); got != 2 {
		t.Errorf("rang %d, attendu 2 — le motif en fenetre n est plus lu", got)
	}
	pay = buildGrenadeSelTail(invGrenadeSelHi+1, 0b110, 3, 40)
	if got := invGrenadeSelection(pay, 0, len(pay)*8, gren); got != -1 {
		t.Errorf("motif HORS fenetre lu (rang %d), attendu -1 — la fenetre ne borne plus rien", got)
	}
}

// TestR5MaskMustMatchCounters : le masque doit etre EXACTEMENT le bitmap des compteurs i22, et
// la selection designer un rang porte. Sans ces deux gardes, neuf bits se trouveraient partout.
func TestR5MaskMustMatchCounters(t *testing.T) {
	gren := [invGrenadeSlots]uint32{2, 0, 0, 0} // masque attendu 0b000001
	pay := buildGrenadeSelTail(invGrenadeSelLo, 0b000010, 2, 40)
	if got := invGrenadeSelection(pay, 0, len(pay)*8, gren); got != -1 {
		t.Errorf("masque etranger accepte (rang %d), attendu -1", got)
	}
	pay = buildGrenadeSelTail(invGrenadeSelLo, 0b000001, 2, 40)
	if got := invGrenadeSelection(pay, 0, len(pay)*8, gren); got != -1 {
		t.Errorf("selection HORS masque acceptee (rang %d), attendu -1", got)
	}
}

// TestR5ContradictoryReadsRefuse : deux occurrences en fenetre qui ne disent pas la meme
// selection = non lu. On ne departage pas au hasard — meme doctrine que R1.
func TestR5ContradictoryReadsRefuse(t *testing.T) {
	gren := [invGrenadeSlots]uint32{0, 2, 1, 0}
	w := &bitWriter{}
	for i := 0; i < invGrenadeSelLo; i++ {
		w.put(1, 1)
	}
	w.put(0b110, 6)
	w.put(2, 3) // rang 1
	w.put(0b110, 6)
	w.put(3, 3) // rang 2, a +209 : encore dans la fenetre
	for i := 0; i < 40; i++ {
		w.put(1, 1)
	}
	if got := invGrenadeSelection(w.buf, 0, len(w.buf)*8, gren); got != -1 {
		t.Errorf("deux lectures contradictoires rendent %d, attendu -1", got)
	}
}

// TestInventoryRulesOnRealBinary : LES QUATRE REGLES, SUR LE BINAIRE REEL.
//
// Les tests ci-dessus verifient la mecanique de chaque regle sur un flux construit. Celui-ci
// verifie ce qu elles RENDENT ENSEMBLE sur les images-cles du film de reference — les seuls
// chiffres qui disent si la grammaire tient encore.
//
// LES NON-LECTURES SONT PUBLIEES, ET C EST LE POINT : 184 etats pour 132 capacites et 120
// compteurs de grenade lus. Un decodeur qui remonterait a 184/184 ne serait pas « meilleur » —
// il aurait cesse de refuser, et il faudrait comprendre pourquoi avant de s en rejouir.
func TestInventoryRulesOnRealBinary(t *testing.T) {
	inv, _, err := ScanFilmKeyframeInventory(MiniFilmDir, loadoutFamilies(), 0)
	if err != nil {
		t.Fatalf("ScanFilmKeyframeInventory : %v", err)
	}
	var ability, grenades, ammo, multi, drawn, grenSel int
	for _, i := range inv {
		if i.AbilityRank >= 0 {
			ability++
		}
		if i.GrenadesRead {
			grenades++
		}
		if i.AmmoRead {
			ammo++
		}
		if i.AmmoCandidates > 1 {
			multi++
		}
		if i.DrawnSlot >= 0 {
			drawn++
		}
		if i.SelectedGrenadeRank >= 0 {
			grenSel++
			if !i.GrenadesRead || i.SelectedGrenadeRank >= invGrenadeSlots ||
				i.Grenades[i.SelectedGrenadeRank] == 0 {
				t.Fatalf("slot %d : selection de grenade rang %d sans compteur porte — la garde "+
					"masque==i22 ne tient plus", i.Slot, i.SelectedGrenadeRank)
			}
		}
		for k := range i.Ammo {
			if i.Ammo[k].Mag != nil && i.Ammo[k].Gauge != nil {
				t.Fatalf("slot %d : chargeur ET jauge sur le meme emplacement — la largeur 22 "+
					"n existe pas dans la carte memoire", i.Slot)
			}
		}
	}
	for _, c := range []struct {
		nom       string
		got, want int
	}{
		{"etats", len(inv), wantInventoryRead},
		{"capacite lue (R1)", ability, wantInvAbility},
		{"grenades lues (R2)", grenades, wantInvGrenades},
		{"munitions lues (R3+R4)", ammo, wantInvAmmo},
		{"lectures a plusieurs candidats", multi, wantInvMultiCandidate},
		{"selection de grenade lue (R5)", grenSel, wantInvGrenadeSel},
	} {
		if c.got != c.want {
			t.Errorf("%s : %d, attendu %d — une regle d ancrage a change de rendement",
				c.nom, c.got, c.want)
		}
	}
	if drawn != ammo {
		t.Errorf("%d emplacement(s) degaine(s) pour %d bloc(s) de munitions lus : le selecteur "+
			"i42 fait partie du MEME parse, les deux comptes ne peuvent pas diverger", drawn, ammo)
	}
}

// Le rendement mesure de chaque regle sur le film de reference. Ces valeurs sont ECRITES, pas
// derivees : une valeur attendue qui se recalcule depuis la sortie ne teste rien.
const (
	wantInvAbility        = 132
	wantInvGrenades       = 120
	wantInvAmmo           = 150
	wantInvMultiCandidate = 51
	wantInvGrenadeSel     = 92
)
