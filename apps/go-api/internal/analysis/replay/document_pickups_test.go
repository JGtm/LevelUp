package replay

// document_pickups_test.go — LE FILET DES DEUX CALQUES PURS du ramassage natif.
//
// POURQUOI ICI ET PAS DANS LE GOLDEN. Le fixture d'assemblage ne porte NI `weaponChanges`, NI
// `equipmentChanges`, NI les ramassages : `goldenInputs.options()` ne les transmet pas. C'est
// une lacune ANTÉRIEURE à ce lot (les deux premiers canaux vivent déjà en production sans
// couverture de golden) et ce lot ne la corrige pas — ce serait un fix hors périmètre. Mais
// laisser un calque de production SANS filet ne se fait pas : `buildPickups` et
// `datePadPickups` sont PURS, ils se testent donc directement, sur des entrées écrites à la
// main où chaque cas limite est visible.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

func TestBuildPickupsProjectsAndNames(t *testing.T) {
	const origin, step = 1_000_000, 100_000 // frame = (ts - 1 s) / 100 ms
	in := []filmdec.BipedPickup{
		{TimestampUS: 900_000, Slot: 520, CatalogID: 0xAABBCCDD, Class: 0}, // avant l'origine
		{TimestampUS: 1_000_000, Slot: 520, CatalogID: 0x11223344, Class: 0},
		{TimestampUS: 1_250_000, Slot: 999, CatalogID: 0x55667788, Class: 2}, // slot sans pont
		{TimestampUS: 1_500_000, Slot: 521, CatalogID: 0x99AABBCC, Class: 3},
	}
	slotXUID := map[uint32]uint64{520: 111, 521: 222}
	st := filmdec.BipedPickupStats{MultiEvent: 7, RefusedOffBand: 1}

	got, cov := buildPickups(in, origin, step, slotXUID, st)
	if len(got) != 3 {
		t.Fatalf("publies = %d, attendu 3 (l evenement anterieur a l origine est ecarte)", len(got))
	}
	if cov.BeforeOrigin != 1 {
		t.Errorf("beforeOrigin = %d, attendu 1", cov.BeforeOrigin)
	}
	if got[0].T != 0 || got[1].T != 2 || got[2].T != 5 {
		t.Errorf("frames = %d/%d/%d, attendu 0/2/5", got[0].T, got[1].T, got[2].T)
	}
	if got[0].W != "11223344" {
		t.Errorf("W = %q, attendu \"11223344\" (hexa 8 chiffres, meme convention que Loadout.W)", got[0].W)
	}
	if got[0].Kind != PickupWeapon || got[1].Kind != PickupItem || got[2].Kind != PickupItem {
		t.Errorf("natures = %q/%q/%q, attendu weapon/item/item", got[0].Kind, got[1].Kind, got[2].Kind)
	}
	// LA CLASSE BRUTE SURVIT : ce qui distingue 2 de 3 n'est pas etabli, et le jour ou ce le
	// sera les artefacts deja cuits doivent porter la valeur.
	if got[2].Class != 3 {
		t.Errorf("classe brute = %d, attendu 3", got[2].Class)
	}
	// LE RAMASSEUR EST NOMME QUAND LE PONT LE PERMET, et l'evenement est publie SANS xuid
	// sinon — un ramassage anonyme vaut mieux qu'un ramassage efface.
	if got[0].XUID != "111" || got[2].XUID != "222" {
		t.Errorf("xuid = %q/%q, attendu \"111\"/\"222\"", got[0].XUID, got[2].XUID)
	}
	if got[1].XUID != "" {
		t.Errorf("xuid = %q sur un slot sans pont, attendu vide", got[1].XUID)
	}
	if cov.Named != 2 || cov.Published != 3 || cov.Weapons != 1 || cov.Items != 2 {
		t.Errorf("couverture = %+v, attendu nommes=2 publies=3 armes=1 objets=2", cov)
	}
	// LES TEMOINS DU DECODEUR VOYAGENT JUSQU'A LA COUVERTURE : sans eux, un lecteur ne peut
	// pas juger ce que le canal ne voit pas.
	if cov.MultiEvent != 7 || cov.Refused != 1 {
		t.Errorf("multiEvent = %d et refuses = %d, attendu 7 et 1", cov.MultiEvent, cov.Refused)
	}
}

func TestDatePadPickupsAddsNeverRemoves(t *testing.T) {
	pads := []WeaponPad{{Weapon: "11223344"}, {Weapon: "55667788"}}
	pickups := []Pickup{
		{T: 12, W: "11223344", Kind: PickupWeapon, XUID: "111"}, // dans la fenetre du socle 0
		{T: 40, W: "55667788", Kind: PickupWeapon, XUID: "222"}, // ambigu avec le suivant
		{T: 45, W: "55667788", Kind: PickupWeapon, XUID: "333"},
		{T: 70, W: "11223344", Kind: PickupItem, XUID: "444"}, // non-arme : jamais apparie a un socle
	}
	picks := []PadPickup{
		{Pad: 0, TLow: 10, THigh: 30}, // datable
		{Pad: 1, TLow: 35, THigh: 50}, // AMBIGU : deux candidats
		{Pad: 0, TLow: 60, THigh: 80}, // seul candidat est un non-arme -> non couvert
		{Pad: 9, TLow: 10, THigh: 30}, // index de socle hors bornes
	}
	st := datePadPickups(pads, picks, pickups)

	if picks[0].T == nil || *picks[0].T != 12 {
		t.Fatalf("occupation 0 : t = %v, attendu 12", picks[0].T)
	}
	if picks[0].XUID == nil || *picks[0].XUID != "111" {
		t.Errorf("occupation 0 : xuid = %v, attendu \"111\"", picks[0].XUID)
	}
	// L'AMBIGUITE NE SE TRANCHE PAS AU HASARD : deux joueurs ont pu prendre la meme arme
	// ailleurs sur la carte pendant ces vingt secondes.
	if picks[1].T != nil || picks[1].XUID != nil {
		t.Errorf("occupation 1 ambigue : t = %v, xuid = %v, attendu nil/nil", picks[1].T, picks[1].XUID)
	}
	// UN SOCLE D'ARME NE REND PAS DE L'EQUIPEMENT.
	if picks[2].T != nil {
		t.Errorf("occupation 2 : t = %v, attendu nil (le seul candidat n est pas une arme)", picks[2].T)
	}
	if picks[3].T != nil {
		t.Errorf("occupation 3 : index de socle hors bornes, t = %v, attendu nil", picks[3].T)
	}
	// RIEN N'EST EFFACE : les intervalles restent, dates ou non.
	for i, k := range picks {
		if k.TLow == 0 && k.THigh == 0 {
			t.Errorf("occupation %d : l intervalle a ete efface", i)
		}
	}
	if st.Occupations != 4 || st.Dated != 1 || st.Named != 1 || st.Ambiguous != 1 || st.Uncovered != 2 {
		t.Errorf("stats = %+v, attendu occupations=4 datees=1 nommees=1 ambigues=1 nonCouvertes=2", st)
	}
}

// TestDatePadPickupsWithoutNativeChannelIsInert : sans ramassage natif, RIEN ne bouge. C'est la
// garantie de compatibilite du schema 29 — un film dont le canal natif ne rend rien produit
// exactement les `padPickups` du schema 28.
func TestDatePadPickupsWithoutNativeChannelIsInert(t *testing.T) {
	pads := []WeaponPad{{Weapon: "11223344"}}
	picks := []PadPickup{{Pad: 0, TLow: 10, THigh: 30}}
	st := datePadPickups(pads, picks, nil)
	if picks[0].T != nil || picks[0].XUID != nil {
		t.Errorf("sans canal natif, l occupation a ete modifiee : t = %v, xuid = %v", picks[0].T, picks[0].XUID)
	}
	if st.Uncovered != 1 || st.Dated != 0 {
		t.Errorf("stats = %+v, attendu nonCouvertes=1 datees=0", st)
	}
}
