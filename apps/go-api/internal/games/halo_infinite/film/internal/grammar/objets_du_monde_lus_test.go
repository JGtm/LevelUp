package grammar

// objets_du_monde_lus_test.go — LE CONTRAT DES OBJETS DU MONDE DERRIERE LA MARCHE : la marche designe
// les records et leur archetype, les lecteurs de toujours les lisent, la passe ne rend que ce que la
// marche n a pas lu. Temoins de la mini-bobine (registre compris) : le slot 1596, que la bande des
// armes au sol porte et que la marche lit comme un equipement ; la creation du slot 1476 au paquet
// 4:576, que la passe trouve dans une trame que la fermeture prouve depuis son debut, ou la marche ne
// lit aucun record de ce slot.

import (
	"reflect"
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// contexteDeLaMiniBobine rend un contexte neuf de la mini-bobine et les bornes de ses pistes.
func contexteDeLaMiniBobine(t *testing.T) (*FilmContext, profile.Vec3Range) {
	t.Helper()
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	return NewFilmContext(film), profile.QuantRangeCEBiped()
}

// echantillonsDuSlot compte les echantillons des pistes du slot `slot`.
func echantillonsDuSlot(pistes []types.ProjectileTrack, slot uint32) int {
	n := 0
	for _, p := range pistes {
		if p.Slot == slot {
			n += len(p.Pts)
		}
	}
	return n
}

// preuvesDesEchantillons rend, pour les echantillons que la passe a releves pour le slot `slot` sur la
// bande `band`, si la trame de leur paquet est prouvee, par instant ; et combien la passe en releve
// dans une trame prouvee ou la marche a lu ce slot. La passe doit avoir ete faite sur ce contexte.
func preuvesDesEchantillons(t *testing.T, fc *FilmContext, wr profile.Vec3Range, band map[uint32]bool,
	slot uint32) (prouveeA map[uint64]bool, prouves int) {
	t.Helper()
	p, ok := fc.recup.pistesDe(wr, fc.ProfilDeBalayage().LargeursObjetDuMonde(), slotsDeLaBande(band))
	if !ok || fc.recup.lectures == nil {
		t.Fatal("la passe et la marche doivent avoir ete faites sur ce contexte")
	}
	prouveeA = map[uint64]bool{}
	for _, s := range p.brut {
		if s.slot != slot {
			continue
		}
		tr, vu := fc.recup.lectures.trames[paquetDuFlux{s.Chunk, s.paquet}]
		prouvee := vu && tr.prouveeDes != rienDeProuve
		prouveeA[s.TimestampUS] = prouvee
		if prouvee && slices.Contains(tr.slots, slot) {
			prouves++
		}
	}
	return prouveeA, prouves
}

// pisteDuSlot dit si des pistes portent le slot `slot`.
func pisteDuSlot(pistes []types.ProjectileTrack, slot uint32) bool {
	for _, p := range pistes {
		if p.Slot == slot {
			return true
		}
	}
	return false
}

// TestUnePisteNEstRendueQuASonArchetype : les bandes des armes au sol et de l equipement se recouvrent ;
// la passe seule rend le slot 1596 aux deux. La marche le lit comme un equipement : dans une trame que
// sa fermeture prouve, la passe ne le rend plus aux armes au sol ; dans une trame qu elle ne prouve
// pas, l archetype que la marche donne au slot n y est pas prouve, et la passe le rend encore.
// MUTATION — rendre tous les echantillons de la passe : ROUGE.
func TestUnePisteNEstRendueQuASonArchetype(t *testing.T) {
	const temoin = 1596
	fc, wr := contexteDeLaMiniBobine(t)
	armes, equipement := worldObjectSlotBand(fc, GroundWeaponTypeIndex), worldObjectSlotBand(fc, EquipmentTypeIndex)
	if !armes[temoin] || !equipement[temoin] {
		t.Fatalf("le slot %d doit etre dans les deux bandes : le temoin ne distingue plus rien", temoin)
	}
	seule, err := ScanWorldObjectsForBand(fc, &wr, ArchetypeDeBandeInconnu, armes)
	if err != nil || !pisteDuSlot(seule, temoin) {
		t.Fatalf("la passe seule doit rendre le slot %d a la bande des armes au sol (%v)", temoin, err)
	}
	pistesArmes, err := ScanWorldObjectsForBand(fc, &wr, GroundWeaponTypeIndex, armes)
	if err != nil {
		t.Fatal(err)
	}
	prouveeA, prouves := preuvesDesEchantillons(t, fc, wr, armes, temoin)
	if prouves == 0 {
		t.Fatalf("la passe ne releve le slot %d dans aucune trame prouvee : le temoin ne distingue plus rien", temoin)
	}
	for _, p := range pistesArmes {
		for _, s := range p.Pts {
			if p.Slot == temoin && prouveeA[s.TimestampUS] {
				t.Fatalf("slot %d, lu comme un equipement par la marche, rendu aux armes au sol a %d dans une trame "+
					"que la fermeture prouve", temoin, s.TimestampUS)
			}
		}
	}
	if n, m := echantillonsDuSlot(pistesArmes, temoin), echantillonsDuSlot(seule, temoin); n == 0 || n >= m {
		t.Fatalf("slot %d aux armes au sol : %d echantillons derriere la marche contre %d par la passe seule ; "+
			"ceux des trames non prouvees doivent rester", temoin, n, m)
	}
	pistesEquipement, err := ScanWorldObjectsForBand(fc, &wr, EquipmentTypeIndex, equipement)
	if err != nil || !pisteDuSlot(pistesEquipement, temoin) {
		t.Fatalf("le slot %d n est plus rendu a l equipement (%v)", temoin, err)
	}
	if fc.ComptesDesReplis().PistesDuMondeApresLaMarche == 0 {
		t.Fatal("la bobine porte des trames que la marche ne lit pas : la passe doit en rendre, comptees")
	}
}

// TestUneCreationFortuiteNeSeRendPas : la creation du slot 1476 au paquet 4:576 est dans une trame que
// la fermeture prouve depuis son debut, ou la marche ne lit aucun record de ce slot : la passe seule la
// rend, la marche d abord ne la rend pas. Aux creations que les deux rendent au meme en-tete, la valeur
// est la meme. MUTATION — rendre toutes les creations de la passe : ROUGE.
func TestUneCreationFortuiteNeSeRendPas(t *testing.T) {
	fc, wr := contexteDeLaMiniBobine(t)
	band := worldObjectSlotBand(fc, EquipmentTypeIndex)
	w, err := fc.marcheDeCreation(EquipmentTypeIndex, &wr, band)
	if err != nil {
		t.Fatal(err)
	}
	brut, _ := releverLesCreations(fc, []equipCreationWalk{w})
	fortuite := func(cr []types.EquipmentCreation) bool {
		for _, x := range cr {
			if x.Chunk == 4 && x.PacketIndex == 576 && x.Slot == 1476 {
				return true
			}
		}
		return false
	}
	if !fortuite(brut[0]) {
		t.Fatal("la passe seule doit rendre la creation fortuite : le temoin ne distingue plus rien")
	}
	cre, _, err := ScanEquipmentCreationsForBand(fc, &wr, band)
	if err != nil || fortuite(cre) {
		t.Fatalf("creation fortuite rendue derriere la marche (%v)", err)
	}
	type cle struct {
		chunk, paquet, bit int
	}
	parLaPasse := map[cle]types.EquipmentCreation{}
	for _, x := range brut[0] {
		parLaPasse[cle{x.Chunk, x.PacketIndex, x.BitPos}] = x
	}
	communes := 0
	for _, x := range cre {
		if y, ok := parLaPasse[cle{x.Chunk, x.PacketIndex, x.BitPos}]; ok {
			communes++
			if !reflect.DeepEqual(x, y) {
				t.Fatalf("creation %d:%d bit %d : %+v par la marche, %+v par la passe", x.Chunk, x.PacketIndex, x.BitPos, x, y)
			}
		}
	}
	if communes == 0 {
		t.Fatal("aucune creation commune aux deux lectures : la comparaison ne garde rien")
	}
}
