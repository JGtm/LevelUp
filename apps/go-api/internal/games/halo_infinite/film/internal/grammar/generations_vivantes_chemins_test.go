package grammar

// generations_vivantes_chemins_test.go — LES CHEMINS DE PRODUCTION DU FILTRE DE GENERATION VIVANTE
// (correction de revue du jalon J5 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat GB-1).
//
// La revue a joue trois mutations qui laissaient la suite verte : les tests J5.2 exercent le coeur pur
// ([ScanBipedRecords], filtre fourni a la main) et les huit canaux delta, mais aucun ne passait par les
// sites qui VONT CHERCHER les generations vivantes du film ([ScanBipedPositionsForBand],
// [ScanBipedAimOnly], [scanEquipRecoveryPacket]) ni par le marcheur pour la generation qu il PORTE
// (les tests J5.3 l injectent). Ces tests fabriquent un corps de generation 2 dans les OCTETS de la
// mini-bobine versionnee (en memoire : le fichier n est pas touche) et exigent que ces chemins le lisent.
//
// LA FABRICATION. Aucune mini-bobine du depot ne porte de generation >= 2. Le slot d un corps est donc
// SCINDE a un instant : ses records delta (avec i0 et visee seule) posterieurs passent a la generation
// 2, et son record de CREATION aussi — c est lui qui designe la generation 2 comme vivante
// ([FilmContext.GenerationsVivantes]) ; les images-cles continuent de designer la generation 1, qui
// reste vivante pour les records anterieurs. Le slot porte alors deux corps successifs, comme un slot
// recycle d un BTB long.

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// coupureGB1 : l instant (horloge du film, us) ou les slots scindes changent de corps. Choisi sur la
// mini-bobine 000d5950 : le slot 517 y emet i48 aux compteurs c5, c6 AVANT et c7 APRES, le slot 527 son
// emplacement d arme i45 une fois avant et une fois apres.
const coupureGB1 uint64 = 4_600_000_000

// Slots de la mini-bobine 000d5950 retenus pour ce qu ils emettent (cf. coupureGB1) ; 520 porte 166
// visees seules.
const (
	slotChaineI48 uint32 = 517
	slotArmeI45   uint32 = 527
	slotVisee     uint32 = 520
)

// siteDeGeneration : les deux bits de generation d un handle, dans un payload du film.
type siteDeGeneration struct {
	pay []byte
	bit int
}

// comptesScission : ce que la scission d un slot a reecrit.
type comptesScission struct {
	avecI0, viseeSeule, creations int
}

// chargerBobineFamilles charge la mini-bobine versionnee 000d5950 en memoire.
func chargerBobineFamilles(t *testing.T) *source.Film {
	t.Helper()
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	return film
}

// scinderALaGenerationDeux reecrit, DANS LES OCTETS du film de `fc`, la generation 1 -> 2 : pour
// chaque slot de `depuis`, ses records delta bipedes (avec i0 et visee seule) dont le paquet date de
// `depuis[slot]` ou plus tard, et ses records de creation. Les sites sont TOUS releves avant la
// premiere ecriture, sous le filtre de production du film intact. `fc` ne doit plus servir ensuite :
// ses derivations memorisees decrivent le film d avant.
func scinderALaGenerationDeux(t *testing.T, fc *FilmContext, depuis map[uint32]uint64) map[uint32]comptesScission {
	t.Helper()
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatalf("decoupage i0 de la mini-bobine : %v", err)
	}
	band, gens := fc.BipedSlots(), fc.GenerationsVivantes()
	comptes := map[uint32]comptesScission{}
	var sites []siteDeGeneration
	retenir := func(pay []byte, p int, ts uint64, viseeSeule bool) {
		h := LireHandleDelta(pay, p)
		d, ok := depuis[h.Slot]
		if !ok || h.Gen != 1 || ts < d {
			return
		}
		sites = append(sites, siteDeGeneration{pay: pay, bit: p + prefixeDeltaBits + handleSlotBits})
		c := comptes[h.Slot]
		if viseeSeule {
			c.viseeSeule++
		} else {
			c.avecI0++
		}
		comptes[h.Slot] = c
	}
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
			walkDeltaBipedPayload(pay, band, lay, gens, func(r deltaBipedRecord) {
				retenir(pay, r.I0-bipedHeaderBits-bipedIndexBits*len(r.Mask), pk.TimestampUS, false)
			})
			debutsDesViseesSeules(fc, pay, band, gens, func(p int) { retenir(pay, p, pk.TimestampUS, true) })
		}
	}
	cre, _, err := fc.CreationsDeBipede()
	if err != nil {
		t.Fatalf("creations de la mini-bobine : %v", err)
	}
	for _, x := range cre {
		if _, ok := depuis[x.Slot]; !ok || x.Generation != 1 {
			continue
		}
		data, pks, ok := fc.ChunkAt(x.Chunk)
		if !ok || x.PacketIndex >= len(pks) {
			t.Fatalf("creation %+v : paquet introuvable", x)
		}
		sites = append(sites, siteDeGeneration{pay: pks[x.PacketIndex].Payload(data), bit: x.BitPos + woNewTypeBits + handleSlotBits})
		c := comptes[x.Slot]
		c.creations++
		comptes[x.Slot] = c
	}
	for slot := range depuis {
		if comptes[slot].creations == 0 {
			t.Fatalf("aucune creation du slot %d : la generation 2 ne serait designee par aucune lecture", slot)
		}
	}
	for _, s := range sites {
		poserDeuxBits(s.pay, s.bit, 2)
	}
	return comptes
}

// debutsDesViseesSeules rend le bit de debut de chaque record de VISEE SEULE d un payload, avec
// l ancrage et l avance de [ScanBipedAimRecords] (qui ne publie pas ce bit).
func debutsDesViseesSeules(fc *FilmContext, pay []byte, band SlotBand, gens *GenerationsVivantes, visit func(int)) {
	total := len(pay) * 8
	br := LecteurSur(pay)
	br.PoserContexte(fc.ContexteDeLecture())
	for p := 0; p+bipedHeaderBits <= total; {
		at, _, ok := matchAimOnlyRecord(br, pay, p, band, gens)
		if !ok {
			p++
			continue
		}
		var d componentDirs
		readAimingVectorComponent(pay, at, total, &d)
		if !d.HasYaw {
			p++
			continue
		}
		visit(p)
		p = at + 1 + aimYawBits + aimPitchBits
	}
}

// exigerGenerationDeuxVivante echoue si la scission n a pas rendu la generation 2 du slot vivante.
func exigerGenerationDeuxVivante(t *testing.T, fc *FilmContext, slot uint32) {
	t.Helper()
	if !fc.GenerationsVivantes().Accepte(types.LifeKey{Slot: slot, Gen: 2}) {
		t.Fatalf("la generation 2 du slot %d n est pas vivante apres reecriture de sa creation", slot)
	}
}

// resumeDuSlot : nombre de positions du slot, dont posterieures a la coupure, et dernier instant.
func resumeDuSlot(pos []BipedPosition, slot uint32) (n, apres int, dernier uint64) {
	for _, p := range pos {
		if p.Slot != slot {
			continue
		}
		n++
		if p.TimestampUS >= coupureGB1 {
			apres++
		}
		dernier = max(dernier, p.TimestampUS)
	}
	return n, apres, dernier
}

// TestPositions_CorpsDeGenerationDeuxPublieParLeCheminDeProduction : ScanBipedPositions — l entree
// de la cuisson, options par defaut (Generations nil) — publie les positions d un corps de generation
// 2 exactement comme a la generation 1, et la trajectoire du slot n est pas tronquee a la coupure.
// C est le site qui va chercher les generations vivantes du film (offline_biped_band.go) : laisse a
// nil, il retombe sur la seule generation 1 et le corps disparait (constat GB-1 : le rejeu de
// 1c4c63c2 s arretait a 826 s pour un film de 1 334 s).
func TestPositions_CorpsDeGenerationDeuxPublieParLeCheminDeProduction(t *testing.T) {
	film := chargerBobineFamilles(t)
	opt := DefaultScanFilmOptions()
	opt.QuantaOnly = true
	avant, err := ScanBipedPositions(NewFilmContext(film), opt)
	if err != nil {
		t.Fatalf("positions de la mini-bobine : %v", err)
	}
	nAvant, apresAvant, dernierAvant := resumeDuSlot(avant, slotChaineI48)
	if apresAvant == 0 {
		t.Fatalf("slot %d sans position apres la coupure : le test ne prouverait rien", slotChaineI48)
	}
	comptes := scinderALaGenerationDeux(t, NewFilmContext(film), map[uint32]uint64{slotChaineI48: coupureGB1})
	if comptes[slotChaineI48].avecI0 == 0 {
		t.Fatalf("aucun record avec i0 du slot %d reecrit", slotChaineI48)
	}
	fc := NewFilmContext(film)
	exigerGenerationDeuxVivante(t, fc, slotChaineI48)
	apres, err := ScanBipedPositions(fc, opt)
	if err != nil {
		t.Fatalf("positions apres scission : %v", err)
	}
	nApres, apresApres, dernierApres := resumeDuSlot(apres, slotChaineI48)
	if nApres != nAvant || apresApres != apresAvant || dernierApres != dernierAvant {
		t.Fatalf("slot %d scinde (generation 2 des %d us, %d records reecrits) : %d position(s) dont %d "+
			"apres la coupure, derniere a %d us ; attendu %d dont %d, derniere a %d us — le corps de "+
			"generation 2 n est pas publie et la trajectoire est tronquee (constat GB-1)", slotChaineI48,
			coupureGB1, comptes[slotChaineI48].avecI0, nApres, apresApres, dernierApres, nAvant, apresAvant, dernierAvant)
	}
	if !reflect.DeepEqual(apres, avant) {
		t.Errorf("les positions publiees different apres le passage d un corps a la generation 2")
	}
}

// TestVisee_CorpsDeGenerationDeuxLuParScanBipedAimOnly : la visee d un corps de generation 2 qui ne
// replique pas sa position (occupant de vehicule) est lue par le balayage de production.
func TestVisee_CorpsDeGenerationDeuxLuParScanBipedAimOnly(t *testing.T) {
	film := chargerBobineFamilles(t)
	avant, err := ScanBipedAimOnly(NewFilmContext(film))
	if err != nil {
		t.Fatalf("visees de la mini-bobine : %v", err)
	}
	comptes := scinderALaGenerationDeux(t, NewFilmContext(film), map[uint32]uint64{slotVisee: 0})
	if comptes[slotVisee].viseeSeule == 0 {
		t.Fatalf("aucune visee seule du slot %d reecrite : le test ne prouverait rien", slotVisee)
	}
	fc := NewFilmContext(film)
	exigerGenerationDeuxVivante(t, fc, slotVisee)
	apres, err := ScanBipedAimOnly(fc)
	if err != nil {
		t.Fatalf("visees apres passage a la generation 2 : %v", err)
	}
	compter := func(l []BipedAim) (n int) {
		for _, a := range l {
			if a.Slot == slotVisee {
				n++
			}
		}
		return n
	}
	if compter(apres) != compter(avant) || !reflect.DeepEqual(apres, avant) {
		t.Fatalf("slot %d a la generation 2 (%d visees seules reecrites) : %d visee(s) lue(s), %d a la "+
			"generation 1 — le balayage ne voit pas le corps de generation 2 (constat GB-1)",
			slotVisee, comptes[slotVisee].viseeSeule, compter(apres), compter(avant))
	}
}
