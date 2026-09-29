package grammar

// generations_vivantes_datees_test.go — LA GARDE DATEE DU FILTRE DE GENERATION (lot R2 du plan de
// suite de l audit du decodeur, 2026-09-28, constat C2 du G-corpus J11.1) : un record delta dont le
// handle designe un corps AVANT le record de creation de ce corps n est la replication d aucun corps.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// gensDatees : le slot de test porte un corps de generation 1 cree a 100, puis un corps de
// generation 2 cree a 1 000 (horloge des paquets, en microsecondes).
func gensDatees(cre ...BipedCreation) *GenerationsVivantes {
	var vies []types.LifeKey
	for _, c := range cre {
		vies = append(vies, types.LifeKey{Slot: c.Slot, Gen: c.Generation})
	}
	return NouvellesGenerationsVivantes(vies).avecCreations(cre)
}

func creationDe(slot, gen uint32, tUS uint64) BipedCreation {
	return BipedCreation{Slot: slot, Generation: gen, TimestampUS: tUS, HasIndex: true}
}

func TestGenerationDatee_CorpsSuivantAvantSaCreationRefuse(t *testing.T) {
	g := gensDatees(creationDe(slotDeTestGB1, 1, 100), creationDe(slotDeTestGB1, 2, 1_000))
	deux := types.LifeKey{Slot: slotDeTestGB1, Gen: 2}
	un := types.LifeKey{Slot: slotDeTestGB1, Gen: 1}
	if g.A(500).Accepte(deux) {
		t.Errorf("record (slot %d, gen 2) a t=500, AVANT la creation de son corps (t=1000) : lu, attendu "+
			"refuse (constat C2 : en-tete de generation >= 2 publie en position aberrante)", slotDeTestGB1)
	}
	for _, ts := range []uint64{1_000, 1_500} {
		if !g.A(ts).Accepte(deux) {
			t.Errorf("record (slot %d, gen 2) a t=%d, corps cree a 1000 : refuse, attendu lu", slotDeTestGB1, ts)
		}
	}
	if !g.A(500).Accepte(un) {
		t.Errorf("record (slot %d, gen 1) a t=500, corps vivant depuis 100 : refuse, attendu lu", slotDeTestGB1)
	}
	// Sur de vrais octets : le balayage du coeur pur recoit le filtre date.
	opt := scanOptWorld()
	opt.Generations = g.A(500)
	if n := len(lireAvecOptions(t, slotDeTestGB1, 2, opt)); n != 0 {
		t.Errorf("balayage d un record gen 2 avant sa creation : %d position(s), attendu 0", n)
	}
	opt.Generations = g.A(1_200)
	if n := len(lireAvecOptions(t, slotDeTestGB1, 2, opt)); n != 1 {
		t.Errorf("balayage d un record gen 2 apres sa creation : %d position(s), attendu 1", n)
	}
}

func TestGenerationDatee_PremierCorpsDeGenerationUnLaisseARBUn(t *testing.T) {
	g := gensDatees(creationDe(slotDeTestGB1, 1, 100), creationDe(slotDeTestGB1, 2, 1_000))
	if !g.A(50).Accepte(types.LifeKey{Slot: slotDeTestGB1, Gen: 1}) {
		t.Errorf("premier corps de generation 1 avant sa creation : refuse par la grammaire, attendu lu " +
			"(la regle R-B1 du rejeu l ecarte ET le compte dans coverage.tracks.avantCreation)")
	}
}

func TestGenerationDatee_PremierCorpsDUneAutreGenerationGarde(t *testing.T) {
	g := gensDatees(creationDe(slotDeTestGB1, 2, 100))
	if g.A(50).Accepte(types.LifeKey{Slot: slotDeTestGB1, Gen: 2}) {
		t.Errorf("premier corps LU de generation 2 (slot desarme pour R-B1) avant sa creation : lu, attendu refuse")
	}
	if !g.A(150).Accepte(types.LifeKey{Slot: slotDeTestGB1, Gen: 2}) {
		t.Errorf("premier corps de generation 2 apres sa creation : refuse, attendu lu")
	}
}

func TestGenerationDatee_SansDateOuSansCreationLeFiltreResteAtemporel(t *testing.T) {
	g := gensDatees(creationDe(slotDeTestGB1, 1, 100), creationDe(slotDeTestGB1, 2, 1_000))
	if !g.Accepte(types.LifeKey{Slot: slotDeTestGB1, Gen: 2}) {
		t.Errorf("filtre NON date : gen 2 refusee, attendu lue (instruments, coeur pur)")
	}
	// Corps connu par les seules images-cles : aucune date, aucune garde.
	vies := []types.LifeKey{{Slot: slotDeTestGB1, Gen: 1}, {Slot: slotDeTestGB1, Gen: 2}}
	k := NouvellesGenerationsVivantes(vies).avecCreations([]BipedCreation{creationDe(slotDeTestGB1, 1, 100)})
	if !k.A(50).Accepte(types.LifeKey{Slot: slotDeTestGB1, Gen: 2}) {
		t.Errorf("corps sans creation lue : refuse, attendu lu (rien ne le date)")
	}
	if ToutesLesGenerations().A(0) == nil || !ToutesLesGenerations().A(0).Accepte(types.LifeKey{Slot: 1, Gen: 3}) {
		t.Errorf("filtre leve : doit tout accepter, date ou non")
	}
	var nul *GenerationsVivantes
	if nul.A(10) != nil {
		t.Errorf("filtre nil date : attendu nil")
	}
}

// TestGenerationDatee_GenerationQuiRevientSurLeSiege : le pool reboucle au-dela de quatre corps, la
// generation revient. Le corps qui TIENT le siege a l instant fait foi.
func TestGenerationDatee_GenerationQuiRevientSurLeSiege(t *testing.T) {
	g := gensDatees(creationDe(slotDeTestGB1, 1, 100), creationDe(slotDeTestGB1, 2, 1_000),
		creationDe(slotDeTestGB1, 1, 2_000))
	un := types.LifeKey{Slot: slotDeTestGB1, Gen: 1}
	if !g.A(500).Accepte(un) || !g.A(2_500).Accepte(un) {
		t.Errorf("generation 1 tenant le siege : refusee")
	}
	if g.A(1_500).Accepte(un) {
		t.Errorf("generation 1 a t=1500 (siege tenu par la gen 2, la gen 1 revient a 2000) : lue, attendu refusee")
	}
}

// lireAvecOptions : un record bipede (slot, tag) ecrit dans un payload, balaye avec ces options.
func lireAvecOptions(t *testing.T, slot uint32, tag uint64, opt ScanFilmOptions) []BipedPosition {
	t.Helper()
	w := &bitWriter{}
	w.bits(0, 7)
	writeBipedRecord(w, slot, tag, 3, 4096, 5000, 8192)
	w.bits(0, 64)
	return ScanBipedRecords(w.buf, NewSlotBand(map[uint32]bool{slot: true}), cliffLayout, opt, ContexteParDefaut())
}
