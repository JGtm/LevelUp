package filmdec

// biped_creation_test.go — L'INSTRUMENT BIT-EXACT DU LECTEUR DE CRÉATION DE BIPÈDE.
//
// Les records sont construits BIT À BIT, à la spécification du sondage E2 (§2.2). Le test ne
// vérifie pas que le lecteur « marche » : il vérifie qu'il lit AUX BONNES POSITIONS, et les
// mutations le prouvent — décaler l'index d'un seul bit, ou déplacer la porte inversée, doit
// faire ROUGIR. Un lecteur bit à bit qu'aucune mutation ne fait rougir ne lit rien de précis.

import "testing"

// bipedCreationRecord écrit un record de CRÉATION de bipède complet, à la spécification :
//
//	+0  R(1)=0  « pas un delta »        +16 R(2)  génération
//	+1  R(2)=1  recNew                  +18 R(6)  typeIndex = 35
//	+3  R(13)   slot                    +24 le default-state commence ici
//
// puis le prologue du default-state : g0=1, version R(8), gRep=1, représentation R(32),
// porte INVERSÉE R(1)=0, index R(5).
type bipedCreationRecord struct {
	slot, gen  uint32
	version    uint32
	rep        uint32
	gateFermee bool // porte de l'index FERMÉE (bit à 1) : le record ne porte pas d'index
	index      uint32
	// portesFermees force les portes du prologue à ZÉRO, pour bâtir une ancre SANS la forme.
	pasDeVersion, pasDeRep bool
}

func ecrireBipedCreation(w *bitWriter, r bipedCreationRecord) {
	w.bit(0)                   // R(1) = 0 : pas un delta
	w.bits(1, 2)               // R(2) = 1 : recNew
	w.bits(uint64(r.slot), 13) // R(13) : slot
	w.bits(uint64(r.gen), 2)   // R(2) : génération du handle
	w.bits(BipedTypeIndex, 6)  // R(6) : typeIndex
	if r.pasDeVersion {
		w.bit(0)
	} else {
		w.bit(1)
		w.bits(uint64(r.version), 8)
	}
	if r.pasDeRep {
		w.bit(0)
		return
	}
	w.bit(1)
	w.bits(uint64(r.rep), 32)
	if r.gateFermee {
		w.bit(1) // porte INVERSÉE : un bit à 1 signifie « aucune valeur »
		return
	}
	w.bit(0)
	w.bits(uint64(r.index), bipedCreationIndexBits)
}

// bandeDeTest rend une bande dense contenant exactement les slots donnés.
func bandeDeTest(slots ...uint32) SlotBand {
	m := map[uint32]bool{}
	for _, s := range slots {
		m[s] = true
	}
	return NewSlotBand(m)
}

// balayerPourTest fait passer un payload par le MÊME chemin que la production
// (`bipedCreationWalk.scanPayload`), et rend les records avec les compteurs.
func balayerPourTest(pay []byte, band SlotBand) ([]BipedCreation, BipedCreationStats) {
	var st BipedCreationStats
	st.Slots = band.Count()
	w := bipedCreationWalk{band: band, st: &st, autres: map[uint32]int{}}
	out := w.scanPayload(pay, FilmPacket{Index: 7, TimestampUS: 1234}, 3)
	st.OtherWord, st.OtherWordCount = motAlternatifModal(w.autres)
	return out, st
}

// TestCreationBipedeLitLIndexALaPositionExacte : un record bâti à la spécification rend son
// slot, sa génération et son index de participant — et les valeurs sont celles qu'on a écrites.
func TestCreationBipedeLitLIndexALaPositionExacte(t *testing.T) {
	w := &bitWriter{}
	w.bits(0, 5) // amorce : le record ne commence pas au bit 0
	ecrireBipedCreation(w, bipedCreationRecord{
		slot: 515, gen: 1, version: bipedCreationVersion,
		rep: BipedRepresentationName, index: 3,
	})
	w.bits(0, 64) // queue : le balayage doit pouvoir dépasser le record

	recs, st := balayerPourTest(w.buf, bandeDeTest(515))
	if len(recs) != 1 {
		t.Fatalf("records lus = %d, attendu 1 (anchors=%d shapeBad=%d sig=%d)",
			len(recs), st.Anchors, st.ShapeBad, st.SignatureMismatch)
	}
	got := recs[0]
	if got.Slot != 515 || got.Generation != 1 {
		t.Fatalf("vie lue = (slot %d, gen %d), attendu (515, 1)", got.Slot, got.Generation)
	}
	if !got.HasIndex || got.ParticipantIndex != 3 {
		t.Fatalf("index lu = %d (present=%t), attendu 3", got.ParticipantIndex, got.HasIndex)
	}
	if got.Version != bipedCreationVersion || got.Representation != BipedRepresentationName {
		t.Fatalf("signature lue = (version %d, rep %#x), attendu (13, %#x)",
			got.Version, got.Representation, BipedRepresentationName)
	}
	if got.BitPos != 5 {
		t.Fatalf("position de l'en-tête = %d, attendu 5", got.BitPos)
	}
	if got.Chunk != 3 || got.PacketIndex != 7 || got.TimestampUS != 1234 {
		t.Fatalf("localisation = (chunk %d, paquet %d, %d us), attendu (3, 7, 1234)",
			got.Chunk, got.PacketIndex, got.TimestampUS)
	}
	if st.Accepted != 1 || st.SignatureMismatch != 0 || st.GateClosed != 0 {
		t.Fatalf("compteurs = accepted %d, sig %d, gateClosed %d ; attendu 1, 0, 0",
			st.Accepted, st.SignatureMismatch, st.GateClosed)
	}
}

// TestCreationBipedeLesTrenteDeuxIndexSeLisent : le champ fait CINQ bits, et le lecteur doit
// rendre les 32 valeurs telles quelles. Un masque trop court se verrait ici, et nulle part
// ailleurs — sur un film réel l'index ne dépasse jamais le roster.
func TestCreationBipedeLesTrenteDeuxIndexSeLisent(t *testing.T) {
	for idx := uint32(0); idx < 32; idx++ {
		w := &bitWriter{}
		ecrireBipedCreation(w, bipedCreationRecord{
			slot: 600, gen: 3, version: bipedCreationVersion,
			rep: BipedRepresentationName, index: idx,
		})
		w.bits(0, 64)
		recs, _ := balayerPourTest(w.buf, bandeDeTest(600))
		if len(recs) != 1 || recs[0].ParticipantIndex != idx {
			t.Fatalf("index %d : records=%d valeur=%v", idx, len(recs), recs)
		}
	}
}

// TestCreationBipedeDecalerLIndexDUnBitLeChange — LA MUTATION.
//
// Le même record, avec UN SEUL bit de plus glissé avant l'index (la porte inversée écrite deux
// fois). Si le lecteur lisait « quelque part par là », il rendrait encore 3 ; il doit rendre
// autre chose, ou refuser. C'est la preuve que la position `+67` est lue, pas approchée.
func TestCreationBipedeDecalerLIndexDUnBitLeChange(t *testing.T) {
	ref := &bitWriter{}
	ecrireBipedCreation(ref, bipedCreationRecord{
		slot: 515, gen: 1, version: bipedCreationVersion,
		rep: BipedRepresentationName, index: 3,
	})
	ref.bits(0, 64)
	sain, _ := balayerPourTest(ref.buf, bandeDeTest(515))
	if len(sain) != 1 || sain[0].ParticipantIndex != 3 {
		t.Fatalf("témoin sain cassé : %v", sain)
	}

	// Le décalage : en-tête et prologue identiques jusqu'à la représentation, puis UN bit
	// parasite avant la porte inversée. L'index se retrouve à +68 au lieu de +67.
	mut := &bitWriter{}
	mut.bit(0)
	mut.bits(1, 2)
	mut.bits(515, 13)
	mut.bits(1, 2)
	mut.bits(BipedTypeIndex, 6)
	mut.bit(1)
	mut.bits(uint64(bipedCreationVersion), 8)
	mut.bit(1)
	mut.bits(uint64(BipedRepresentationName), 32)
	mut.bit(0)                          // bit parasite : occupe la place de la porte
	mut.bit(0)                          // l'ancienne porte, décalée
	mut.bits(3, bipedCreationIndexBits) // l'index, décalé lui aussi
	mut.bits(0, 64)

	decale, _ := balayerPourTest(mut.buf, bandeDeTest(515))
	if len(decale) == 1 && decale[0].ParticipantIndex == 3 {
		t.Fatalf("le lecteur rend le MÊME index (3) sur un record décalé d'un bit : " +
			"il ne lit pas la position +67, il devine")
	}
}

// TestCreationBipedeRefuseUneAutreSignature : un record dont le mot de 32 bits n'est pas la
// constante est COMPTÉ et IGNORÉ, jamais interprété — et le mot alternatif est publié.
func TestCreationBipedeRefuseUneAutreSignature(t *testing.T) {
	w := &bitWriter{}
	ecrireBipedCreation(w, bipedCreationRecord{
		slot: 515, gen: 0, version: bipedCreationVersion,
		rep: 0xDEADBEEF, index: 5,
	})
	w.bits(0, 64)
	recs, st := balayerPourTest(w.buf, bandeDeTest(515))
	if len(recs) != 0 {
		t.Fatalf("un record de signature étrangère a été INTERPRÉTÉ : %v", recs)
	}
	if st.SignatureMismatch != 1 {
		t.Fatalf("signatureMismatch = %d, attendu 1 (le refus doit se COMPTER)", st.SignatureMismatch)
	}
	if st.OtherWord != 0xDEADBEEF || st.OtherWordCount != 1 {
		t.Fatalf("mot alternatif = %#x×%d, attendu 0xDEADBEEF×1", st.OtherWord, st.OtherWordCount)
	}
}

// TestCreationBipedeRefuseUneAutreVersion : le gate exige la version 13 ÉCRITE. Une version
// différente est une ancre sans la forme — comptée en `ShapeBad`, jamais en désaccord de
// signature (ce serait alarmer sur du bruit d'ancrage).
func TestCreationBipedeRefuseUneAutreVersion(t *testing.T) {
	w := &bitWriter{}
	ecrireBipedCreation(w, bipedCreationRecord{
		slot: 515, version: 12, rep: BipedRepresentationName, index: 5,
	})
	w.bits(0, 64)
	recs, st := balayerPourTest(w.buf, bandeDeTest(515))
	if len(recs) != 0 {
		t.Fatalf("un record de version 12 a été lu : %v", recs)
	}
	if st.ShapeBad != 1 || st.SignatureMismatch != 0 {
		t.Fatalf("shapeBad = %d, signatureMismatch = %d ; attendu 1 et 0", st.ShapeBad, st.SignatureMismatch)
	}
}

// TestCreationBipedePorteFermeeNEstPasUnIndexNul : la porte du ref est INVERSÉE ; à 1 elle ne
// transmet rien. Rendre `index = 0` serait attribuer tous ces corps au participant 0.
func TestCreationBipedePorteFermeeNEstPasUnIndexNul(t *testing.T) {
	w := &bitWriter{}
	ecrireBipedCreation(w, bipedCreationRecord{
		slot: 515, version: bipedCreationVersion,
		rep: BipedRepresentationName, gateFermee: true,
	})
	w.bits(0, 64)
	recs, st := balayerPourTest(w.buf, bandeDeTest(515))
	if len(recs) != 0 {
		t.Fatalf("un record à porte fermée a été rendu : %v", recs)
	}
	if st.GateClosed != 1 {
		t.Fatalf("gateClosed = %d, attendu 1", st.GateClosed)
	}
}

// TestCreationBipedeIgnoreUnSlotHorsBande : la bande est la première contrainte de l'ancrage.
func TestCreationBipedeIgnoreUnSlotHorsBande(t *testing.T) {
	w := &bitWriter{}
	ecrireBipedCreation(w, bipedCreationRecord{
		slot: 515, version: bipedCreationVersion,
		rep: BipedRepresentationName, index: 3,
	})
	w.bits(0, 64)
	recs, st := balayerPourTest(w.buf, bandeDeTest(700, 701))
	if len(recs) != 0 || st.Anchors != 0 {
		t.Fatalf("un record hors bande a été ancré : records=%d anchors=%d", len(recs), st.Anchors)
	}
}

// TestCreationBipedeIgnoreUnAutreArchetype : le R(6) typeIndex est ce qui rend l'ancrage
// sélectif. Un record de création d'un AUTRE archétype sur un slot de la bande ne doit rien
// produire — sans quoi le lecteur lirait le prologue d'un désérialiseur qui n'est pas le sien.
func TestCreationBipedeIgnoreUnAutreArchetype(t *testing.T) {
	w := &bitWriter{}
	w.bit(0)
	w.bits(1, 2)
	w.bits(515, 13)
	w.bits(0, 2)
	w.bits(EquipmentTypeIndex, 6) // ti=37, pas 35
	w.bit(1)
	w.bits(uint64(bipedCreationVersion), 8)
	w.bit(1)
	w.bits(uint64(BipedRepresentationName), 32)
	w.bit(0)
	w.bits(3, bipedCreationIndexBits)
	w.bits(0, 64)
	recs, st := balayerPourTest(w.buf, bandeDeTest(515))
	if len(recs) != 0 || st.Anchors != 0 {
		t.Fatalf("un record ti=%d a été lu comme un bipède : records=%d anchors=%d",
			EquipmentTypeIndex, len(recs), st.Anchors)
	}
}

// TestCreationBipedeDeuxRecordsSuccessifs : deux créations dans le même payload sont rendues
// toutes les deux, et le curseur ne re-balaye pas un record accepté (pas de doublon décalé).
func TestCreationBipedeDeuxRecordsSuccessifs(t *testing.T) {
	w := &bitWriter{}
	ecrireBipedCreation(w, bipedCreationRecord{
		slot: 512, gen: 0, version: bipedCreationVersion,
		rep: BipedRepresentationName, index: 0,
	})
	ecrireBipedCreation(w, bipedCreationRecord{
		slot: 513, gen: 2, version: bipedCreationVersion,
		rep: BipedRepresentationName, index: 7,
	})
	w.bits(0, 64)
	recs, st := balayerPourTest(w.buf, bandeDeTest(512, 513))
	if len(recs) != 2 {
		t.Fatalf("records lus = %d, attendu 2 : %v", len(recs), recs)
	}
	if recs[0].Slot != 512 || recs[0].ParticipantIndex != 0 {
		t.Fatalf("premier record = slot %d index %d", recs[0].Slot, recs[0].ParticipantIndex)
	}
	if recs[1].Slot != 513 || recs[1].Generation != 2 || recs[1].ParticipantIndex != 7 {
		t.Fatalf("second record = slot %d gen %d index %d",
			recs[1].Slot, recs[1].Generation, recs[1].ParticipantIndex)
	}
	if st.Accepted != 2 {
		t.Fatalf("accepted = %d, attendu 2", st.Accepted)
	}
}

// TestCreationBipedeLaCleDeVieDistingueLesGenerations : deux vies du MÊME slot ne se
// confondent pas. Un slot seul les fusionnerait, et c'est exactement le défaut P0-2.
func TestCreationBipedeLaCleDeVieDistingueLesGenerations(t *testing.T) {
	a := BipedCreation{Slot: 549, Generation: 0}
	b := BipedCreation{Slot: 549, Generation: 1}
	if a.LifeKey() == b.LifeKey() {
		t.Fatalf("deux générations du slot 549 partagent la clé de vie %d", a.LifeKey())
	}
	if a.LifeKey() != (BipedCreation{Slot: 549}).LifeKey() {
		t.Fatal("la clé de vie n'est pas stable pour la même paire (slot, génération)")
	}
}

// TestCreationBipedeTemoinFantome : le PLANCHER DE FAUX POSITIFS du gate, mesuré par le MÊME
// code que la mesure.
//
// LE TÉMOIN N'EST PAS DU BRUIT PUR, et c'est ce qui en fait un témoin. Un payload aléatoire ne
// produit presque aucune ANCRE (mesuré : 0 sur 8 Kio), donc il ne contrôlerait rien du gate — il
// contrôlerait l'en-tête. Ici, MILLE ancres parfaites sont fabriquées (préfixe NEW, slot dans la
// bande, `ti=35`) et seul le PROLOGUE est pseudo-aléatoire : c'est exactement la population que
// la signature doit trier. Aucune ne doit passer.
//
// Le sondage E2 a mesuré le plancher du même gate à ZÉRO sur les cinq films, avec une bande
// fantôme décalée de 4 096 (§2.3).
func TestCreationBipedeTemoinFantome(t *testing.T) {
	const ancres = 1000
	w := &bitWriter{}
	x := uint32(0x2545F491)
	suivant := func(n int) uint64 {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		return uint64(x) & ((1 << uint(n)) - 1)
	}
	for i := 0; i < ancres; i++ {
		w.bit(0)                    // R(1) = 0
		w.bits(1, 2)                // R(2) = 1 : recNew
		w.bits(uint64(512+i%8), 13) // slot DANS la bande
		w.bits(suivant(2), 2)       // génération quelconque
		w.bits(BipedTypeIndex, 6)   // ti=35 : l'ancre est parfaite
		w.bits(suivant(32), 32)     // prologue pseudo-aléatoire
		w.bits(suivant(32), 32)     // (48 bits : la largeur du prologue plein)
		w.bits(suivant(11), 11)
	}
	band := bandeDeTest(512, 513, 514, 515, 516, 517, 518, 519)
	recs, st := balayerPourTest(w.buf, band)
	if st.Anchors < ancres {
		t.Fatalf("le témoin n'a produit que %d ancres sur %d : il ne contrôle pas le gate",
			st.Anchors, ancres)
	}
	if len(recs) != 0 {
		t.Fatalf("le témoin fantôme rend %d record(s) sur %d ancres : le gate laisse passer du "+
			"bruit\n%v", len(recs), st.Anchors, recs)
	}
	t.Logf("plancher du gate : anchors=%d shapeBad=%d signatureMismatch=%d gateClosed=%d",
		st.Anchors, st.ShapeBad, st.SignatureMismatch, st.GateClosed)
}

// TestCreationBipedePortesDuPrologueFermees : une ancre dont la porte de version ou celle de la
// représentation est FERMÉE n'a pas la forme d'une création de bipède. Elle se compte en
// `ShapeBad` — le rejet ordinaire de l'ancrage bit à bit — et surtout PAS en désaccord de
// signature : alarmer sur du bruit d'ancrage viderait le compteur de son sens.
func TestCreationBipedePortesDuPrologueFermees(t *testing.T) {
	for _, cas := range []struct {
		nom string
		rec bipedCreationRecord
	}{
		{"porte de version fermée", bipedCreationRecord{slot: 515, pasDeVersion: true}},
		{"porte de représentation fermée", bipedCreationRecord{
			slot: 515, version: bipedCreationVersion, pasDeRep: true}},
	} {
		w := &bitWriter{}
		ecrireBipedCreation(w, cas.rec)
		w.bits(0, 96)
		recs, st := balayerPourTest(w.buf, bandeDeTest(515))
		if len(recs) != 0 {
			t.Fatalf("%s : un record a été rendu : %v", cas.nom, recs)
		}
		if st.ShapeBad != 1 || st.SignatureMismatch != 0 {
			t.Fatalf("%s : shapeBad = %d, signatureMismatch = %d ; attendu 1 et 0",
				cas.nom, st.ShapeBad, st.SignatureMismatch)
		}
	}
}
