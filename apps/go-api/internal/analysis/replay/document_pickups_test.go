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
	"fmt"
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

// LA FORME DES DEUX CÔTÉS EST DÉRIVÉE, JAMAIS RECOPIÉE. Un socle publie sa famille via
// `formatWeaponFamily` (`"0x"` + huit MAJUSCULES) ; un ramassage via `%08x` (huit minuscules,
// sans préfixe). La première version de ce test écrivait `"11223344"` des DEUX côtés — une
// forme que `buildWeaponPads` ne produit jamais : il passait avec ET sans le bogue de jointure.
// Les deux helpers ci-dessous ancrent le test aux fonctions de production : si l'une des deux
// conventions bouge, le test bouge avec elle.
func padWeaponForm(fam uint32) string { return formatWeaponFamily(fam) }

func pickupWeaponForm(fam uint32) string { return fmt.Sprintf("%08x", fam) }

func TestPadFamilyKeyNormalisesBothConventions(t *testing.T) {
	const fam = 0x11223344
	kPad, okPad := padFamilyKey(padWeaponForm(fam))
	kPick, okPick := padFamilyKey(pickupWeaponForm(fam))
	if !okPad || !okPick {
		t.Fatalf("les deux formes de production doivent être reconnues : socle=%v ramassage=%v", okPad, okPick)
	}
	if kPad != kPick {
		t.Errorf("clés divergentes : socle %q vs ramassage %q — la jointure ne peut pas marcher", kPad, kPick)
	}
	// UN NOM CANONIQUE DE POWER-UP N'EST PAS UNE FAMILLE, et c'est ainsi qu'on le distingue
	// sans connaître la liste des noms.
	if _, ok := padFamilyKey("overshield"); ok {
		t.Error("un nom canonique de power-up ne doit PAS passer pour une famille d arme")
	}
	if _, ok := padFamilyKey(""); ok {
		t.Error("une famille vide ne doit pas produire de clé")
	}
}

func TestDatePadPickupsAddsNeverRemoves(t *testing.T) {
	const famA, famB = 0x11223344, 0x55667788
	pads := []WeaponPad{
		{Weapon: padWeaponForm(famA)},
		{Weapon: padWeaponForm(famB)},
		// Socle de POWER-UP : identité = nom canonique, structurellement non joignable.
		{Weapon: "overshield"},
	}
	pickups := []Pickup{
		{T: 12, W: pickupWeaponForm(famA), Kind: PickupWeapon, XUID: "111"},
		{T: 40, W: pickupWeaponForm(famB), Kind: PickupWeapon, XUID: "222"}, // ambigu avec le suivant
		{T: 45, W: pickupWeaponForm(famB), Kind: PickupWeapon, XUID: "333"},
		{T: 70, W: pickupWeaponForm(famA), Kind: PickupItem, XUID: "444"}, // non-arme : jamais un socle d arme
		{T: 90, W: pickupWeaponForm(famA), Kind: PickupWeapon, XUID: "555"},
	}
	picks := []PadPickup{
		{Pad: 0, TLow: 10, THigh: 30}, // datable
		{Pad: 1, TLow: 35, THigh: 50}, // AMBIGU : deux candidats
		{Pad: 0, TLow: 60, THigh: 80}, // seul candidat est un non-arme -> non couvert
		{Pad: 2, TLow: 85, THigh: 95}, // socle de power-up -> hors jointure, pas « non couvert »
		{Pad: 9, TLow: 10, THigh: 30}, // index de socle hors bornes
	}
	st := datePadPickups(pads, picks, pickups)

	if picks[0].T == nil || *picks[0].T != 12 {
		t.Fatalf("occupation 0 : t = %v, attendu 12 — LA JOINTURE EST MORTE", picks[0].T)
	}
	if picks[0].XUID == nil || *picks[0].XUID != "111" {
		t.Errorf("occupation 0 : xuid = %v, attendu \"111\"", picks[0].XUID)
	}
	// L'AMBIGUITE NE SE TRANCHE PAS AU HASARD : deux joueurs ont pu prendre la même arme
	// ailleurs sur la carte pendant ces vingt secondes.
	if picks[1].T != nil || picks[1].XUID != nil {
		t.Errorf("occupation 1 ambiguë : t = %v, xuid = %v, attendu nil/nil", picks[1].T, picks[1].XUID)
	}
	// UN SOCLE D'ARME NE REND PAS DE L'EQUIPEMENT.
	if picks[2].T != nil {
		t.Errorf("occupation 2 : t = %v, attendu nil (le seul candidat n est pas une arme)", picks[2].T)
	}
	// UN SOCLE DE POWER-UP N'EST PAS UNE RECHERCHE INFRUCTUEUSE : il est hors jointure.
	if picks[3].T != nil {
		t.Errorf("occupation 3 (power-up) : t = %v, attendu nil", picks[3].T)
	}
	if picks[4].T != nil {
		t.Errorf("occupation 4 : index de socle hors bornes, t = %v, attendu nil", picks[4].T)
	}
	// RIEN N'EST EFFACE : les intervalles restent, datés ou non.
	for i, k := range picks {
		if k.TLow == 0 && k.THigh == 0 {
			t.Errorf("occupation %d : l intervalle a été effacé", i)
		}
	}
	if st.Occupations != 5 || st.Dated != 1 || st.Named != 1 || st.Ambiguous != 1 ||
		st.Uncovered != 2 || st.PowerupOccupations != 1 {
		t.Errorf("stats = %+v, attendu occupations=5 datées=1 nommées=1 ambiguës=1 nonCouvertes=2 powerup=1", st)
	}
}

// TestDatePadPickupsFailsOnBrokenJoinKey — L'INVERSION, ET ELLE APPELLE VRAIMENT LA JOINTURE.
//
// La première version de ce test (ronde 1) comparait deux chaînes et n'appelait JAMAIS
// `datePadPickups` : elle ne prouvait rien sur la fonction, et portait en prime une branche
// morte (la même condition que son `t.Skip`, trois lignes plus haut). Correctif de ronde 2 :
// on exerce la fonction DEUX FOIS sur les mêmes données — une fois avec la forme de production
// du socle, une fois avec une clé volontairement cassée — et on exige que la seconde ne date
// RIEN. C'est la panne du P0 rejouée en test.
func TestDatePadPickupsFailsOnBrokenJoinKey(t *testing.T) {
	const fam = 0x11223344
	pickups := []Pickup{{T: 12, W: pickupWeaponForm(fam), Kind: PickupWeapon, XUID: "111"}}

	// (1) Forme de PRODUCTION des deux côtés : la jointure trouve.
	bon := []PadPickup{{Pad: 0, TLow: 10, THigh: 30}}
	stBon := datePadPickups([]WeaponPad{{Weapon: padWeaponForm(fam)}}, bon, pickups)
	if stBon.Dated != 1 || bon[0].T == nil {
		t.Fatalf("forme de production : datées=%d t=%v, attendu 1 et 12 — la jointure ne marche pas",
			stBon.Dated, bon[0].T)
	}

	// (2) MÊME famille, mais écrite dans une convention que la normalisation ne rapproche pas
	// (un nom canonique, comme un socle de power-up) : rien ne doit être daté.
	casse := []PadPickup{{Pad: 0, TLow: 10, THigh: 30}}
	stCasse := datePadPickups([]WeaponPad{{Weapon: "famille-11223344"}}, casse, pickups)
	if stCasse.Dated != 0 || casse[0].T != nil || casse[0].XUID != nil {
		t.Errorf("clé non joignable : datées=%d t=%v xuid=%v, attendu 0/nil/nil",
			stCasse.Dated, casse[0].T, casse[0].XUID)
	}
	if stCasse.PowerupOccupations != 1 {
		t.Errorf("clé non joignable : powerupOccupations=%d, attendu 1 (hors jointure, pas « non couvert »)",
			stCasse.PowerupOccupations)
	}

	// (3) LE PIÈGE D'ORIGINE : la forme du socle comparée SANS normalisation à celle du
	// ramassage. Les deux conventions doivent rester distinctes — sinon `padFamilyKey` ne sert
	// plus à rien et ce test doit le dire.
	if padWeaponForm(fam) == pickupWeaponForm(fam) {
		t.Error("les deux conventions coïncident désormais : la normalisation est devenue " +
			"inutile, et le P0 ne peut plus se reproduire — revoir ce test")
	}
}

// TestDatePadPickupsWithoutNativeChannelIsInert : sans ramassage natif, RIEN ne bouge. C'est la
// garantie de compatibilite du schema 30 — un film dont le canal natif ne rend rien produit
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
