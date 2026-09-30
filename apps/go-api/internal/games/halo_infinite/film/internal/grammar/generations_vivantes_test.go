package grammar

// generations_vivantes_test.go — LES TESTS DU FILTRE DE GENERATION VIVANTE (lot J5.2 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DT-8, constat GB-1).
//
// Les trois premiers sont SYNTHETIQUES : un record bipede ecrit par l ecrivain partage des tests
// (`writeBipedRecord`), lu par le coeur pur [ScanBipedRecords]. Le quatrieme travaille sur les
// OCTETS REELS de la mini-bobine du depot, dont il fait passer un corps a la generation 2 : c est la
// seule facon de faire traverser au record les huit canaux delta, qui lisent le registre du film.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// slotDeTestGB1 : un slot bipede de la mini-bobine de reference.
const slotDeTestGB1 uint32 = 517

// lireUnRecord ecrit UN record bipede de la generation `tag` sur `slot` et le lit sous `gens`.
func lireUnRecord(t *testing.T, slot uint32, tag uint64, gens *GenerationsVivantes) []BipedPosition {
	t.Helper()
	w := &bitWriter{}
	w.bits(0, 7) // le balayage est bit a bit, pas aligne
	writeBipedRecord(w, slot, tag, 3, 4096, 5000, 8192)
	w.bits(0, 64)
	opt := scanOptWorld()
	opt.Generations = gens
	return ScanBipedRecords(w.buf, NewSlotBand(map[uint32]bool{slot: true}), cliffLayout, opt, ContexteParDefaut())
}

// TestEnTeteBipede_TagDeLaGenerationVivanteAccepte : un corps de generation 2, connu d une lecture
// de production, a ses positions — c est le corps que GB-1 perdait.
func TestEnTeteBipede_TagDeLaGenerationVivanteAccepte(t *testing.T) {
	gens := NouvellesGenerationsVivantes([]types.LifeKey{{Slot: slotDeTestGB1, Gen: 1}, {Slot: slotDeTestGB1, Gen: 2}})
	for _, tag := range []uint64{1, 2} {
		if got := lireUnRecord(t, slotDeTestGB1, tag, gens); len(got) != 1 {
			t.Errorf("generation %d VIVANTE du slot %d : %d position(s), attendu 1 (constat GB-1 : un corps "+
				"de generation >= 2 n avait aucune position)", tag, slotDeTestGB1, len(got))
		}
	}
}

// TestEnTeteBipede_TagDUneGenerationMorteRefuse : une generation qu aucune lecture ne designe pour
// ce slot reste ecartee — le filtre garde son pouvoir discriminant, y compris contre la generation 1.
func TestEnTeteBipede_TagDUneGenerationMorteRefuse(t *testing.T) {
	gens := NouvellesGenerationsVivantes([]types.LifeKey{{Slot: slotDeTestGB1, Gen: 2}})
	for _, tag := range []uint64{0, 1, 3} {
		if got := lireUnRecord(t, slotDeTestGB1, tag, gens); len(got) != 0 {
			t.Errorf("generation %d MORTE du slot %d (seule la 2 est vivante) : %d position(s) lue(s), "+
				"attendu 0", tag, slotDeTestGB1, len(got))
		}
	}
	if got := lireUnRecord(t, slotDeTestGB1, 2, gens); len(got) != 1 {
		t.Errorf("generation 2 vivante : %d position(s), attendu 1", len(got))
	}
}

// TestEnTeteBipede_SlotSansCreationRetombeSurLeRepliNomme : un slot qu aucune lecture ne designe
// retombe sur le repli `repli_generation_vivante_inconnue_tag1` — generation 1 seulement — et le
// repli se COMPTE (un slot distinct), la ou un slot connu ne compte pas.
func TestEnTeteBipede_SlotSansCreationRetombeSurLeRepliNomme(t *testing.T) {
	const slotConnu uint32 = 600
	gens := NouvellesGenerationsVivantes([]types.LifeKey{{Slot: slotConnu, Gen: 2}})
	lus := lireUnRecord(t, slotDeTestGB1, 1, gens)
	if len(lus) != 1 {
		t.Fatalf("slot %d sans generation connue, tag 1 : %d position(s), attendu 1 (le repli garde tag == 1)",
			slotDeTestGB1, len(lus))
	}
	if got := lireUnRecord(t, slotDeTestGB1, 2, gens); len(got) != 0 {
		t.Errorf("slot %d sans generation connue, tag 2 : %d position(s), attendu 0", slotDeTestGB1, len(got))
	}
	connus := lireUnRecord(t, slotConnu, 2, gens)
	if n := gens.SlotsEnRepli(append(append([]BipedPosition{}, lus...), lus...)); n != 1 {
		t.Errorf("repli compte %d slot(s) pour deux positions d UN slot sans generation connue, attendu 1", n)
	}
	if n := gens.SlotsEnRepli(connus); n != 0 {
		t.Errorf("repli compte %d slot(s) pour un slot CONNU, attendu 0", n)
	}
	if n := ToutesLesGenerations().SlotsEnRepli(lus); n != 0 {
		t.Errorf("filtre leve : repli compte %d, attendu 0", n)
	}
}

// comptesDesHuitCanaux rend le nombre de records bipedes que chacun des huit canaux delta a ancres.
func comptesDesHuitCanaux(t *testing.T, fc *FilmContext) map[string]int {
	t.Helper()
	out := map[string]int{}
	ajouter := func(nom string, n int, err error) {
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		out[nom] = n
	}
	_, ch, err := ScanAbilityCharges(fc)
	ajouter("abilityCharges", ch.Records, err)
	_, im, err := ScanAbilityImpulses(fc)
	ajouter("abilityImpulses", im.Records, err)
	_, rk, err := ScanAbilityRanks(fc)
	ajouter("abilityRanks", rk.Records, err)
	_, ca, err := ScanCamoStates(fc)
	ajouter("camoStates", ca.Records, err)
	_, eq, err := ScanEquipmentChanges(fc, nil)
	ajouter("equipmentChanges", eq.Walk.Records, err)
	_, gr, err := ScanGrappleReads(fc)
	ajouter("grappleReads", gr.Records, err)
	_, hw, err := ScanHeldWeaponChanges(fc, nil)
	ajouter("heldWeaponChanges", hw.Records, err)
	_, iv, err := ScanInventoryDeltas(fc)
	ajouter("inventoryDeltas", iv.Records, err)
	return out
}

// poserDeuxBits ecrit la valeur `v` (2 bits, MSB d abord) a la position `pos` du tampon.
func poserDeuxBits(buf []byte, pos int, v uint8) {
	for k := range 2 {
		bit := (v >> (1 - k)) & 1
		octet, masque := (pos+k)>>3, byte(0x80)>>uint((pos+k)&7)
		if bit == 1 {
			buf[octet] |= masque
		} else {
			buf[octet] &^= masque
		}
	}
}

// passerUnCorpsALaGenerationDeux reecrit, DANS LES OCTETS du film, la generation du handle de tous
// les records delta du slot `slot` et de ses records de creation : 1 -> 2. Les images-cles, elles,
// continuent de designer (slot, 1). Rend le nombre de records delta reecrits.
func passerUnCorpsALaGenerationDeux(t *testing.T, fc *FilmContext, slot uint32) int {
	t.Helper()
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatalf("decoupage i0 de la mini-bobine : %v", err)
	}
	band := fc.BipedSlots()
	type site struct {
		pay []byte
		bit int
	}
	var sites []site
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			walkDeltaBipedPayload(pay, band, lay, ToutesLesGenerations(), func(r deltaBipedRecord) {
				p := r.I0 - bipedHeaderBits - bipedIndexBits*len(r.Mask)
				if h := LireHandleDelta(pay, p); h.Slot == slot && h.Gen == 1 {
					sites = append(sites, site{pay: pay, bit: p + prefixeDeltaBits + handleSlotBits})
				}
			})
		}
	}
	cre, _, err := fc.CreationsDeBipede()
	if err != nil {
		t.Fatalf("creations de la mini-bobine : %v", err)
	}
	creees := 0
	for _, x := range cre {
		if x.Slot != slot || x.Generation != 1 {
			continue
		}
		data, pks, ok := fc.ChunkAt(x.Chunk)
		if !ok || x.PacketIndex >= len(pks) {
			t.Fatalf("creation %+v : paquet introuvable", x)
		}
		sites = append(sites, site{pay: pks[x.PacketIndex].Payload(data), bit: x.BitPos + woNewTypeBits + handleSlotBits})
		creees++
	}
	if creees == 0 {
		t.Fatalf("aucune creation du slot %d sur la mini-bobine : le corps de generation 2 ne serait "+
			"designe par aucune lecture", slot)
	}
	for _, s := range sites {
		poserDeuxBits(s.pay, s.bit, 2)
	}
	return len(sites) - creees
}

// TestMarcheurDelta_CorpsDeGenerationDeuxVuParLesHuitCanaux : un corps dont le handle porte la
// generation 2 — et qu un record de creation designe — est lu par les huit canaux delta exactement
// comme il l etait a la generation 1. Avant le lot J5.2, le marcheur passait `true` en dur au filtre
// d en-tete : ce corps disparaissait des huit canaux (constat GB-1).
func TestMarcheurDelta_CorpsDeGenerationDeuxVuParLesHuitCanaux(t *testing.T) {
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	avant := comptesDesHuitCanaux(t, NewFilmContext(film))
	fc := NewFilmContext(film)
	cre, _, err := fc.CreationsDeBipede()
	if err != nil || len(cre) == 0 {
		t.Fatalf("creations de la mini-bobine : %d, err %v", len(cre), err)
	}
	slot := cre[0].Slot
	reecrits := passerUnCorpsALaGenerationDeux(t, fc, slot)
	if reecrits == 0 {
		t.Fatalf("aucun record delta du slot %d : le test ne prouverait rien", slot)
	}
	apres := NewFilmContext(film)
	if !apres.GenerationsVivantes().Accepte(types.LifeKey{Slot: slot, Gen: 2}) {
		t.Fatalf("la generation 2 du slot %d n est pas vivante apres reecriture de sa creation", slot)
	}
	for nom, n := range comptesDesHuitCanaux(t, apres) {
		if n != avant[nom] {
			t.Errorf("%s : %d records ancres, %d avant le passage du slot %d a la generation 2 (%d "+
				"records reecrits) — le canal ne voit pas le corps de generation 2 (constat GB-1)",
				nom, n, avant[nom], slot, reecrits)
		}
	}
}
