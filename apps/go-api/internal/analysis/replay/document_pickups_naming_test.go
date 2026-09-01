package replay

// document_pickups_naming_test.go — SCHÉMA 31 : LE NOM DE L'OBJET RAMASSÉ.
//
// DISCIPLINE DE CE FICHIER, et elle vient d'une leçon payée. La ronde 2 de la revue du chantier
// précédent a trouvé une fixture qui écrivait `typ: bipedPickupType` — LES CONSTANTES DU CODE
// TESTÉ : les permuter dans la production laissait tout vert. Ici, les natures et les slugs
// attendus sont des LITTÉRAUX. Permuter `PickupGrenade` et `PickupEquipment` dans
// `document_pickups.go` doit faire tomber ce fichier.

import (
	"fmt"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// pkClock construit l'horloge de test avec sa table d'équipement. Les trois quarts des cas de ce
// fichier n'ont besoin que de ça.
func pkClock(equipement map[uint32]string) replayClock {
	return replayClock{origin: 1_000_000, step: 100_000, families: equipement}
}

// TestBuildPickupsResolvesFamilyFromTitleCatalogs — la résolution nominale, les deux sources.
//
// LES SLUGS ATTENDUS SONT DES LITTÉRAUX. Les dériver des tables passées en entrée rendrait le
// test tautologique : il passerait même si `pickupFamily` rendait la première valeur venue.
func TestBuildPickupsResolvesFamilyFromTitleCatalogs(t *testing.T) {
	// Deux catalogues DISJOINTS, comme en production (mesuré : 0 identifiant commun).
	equipement := map[uint32]string{0xbcabbe43: "grenade_frag", 0xeef5d48d: "thruster"}
	armes := map[uint32]string{0x767db96d: "hinf_ma40_ar"}
	in := []filmdec.BipedPickup{
		{TimestampUS: 1_000_000, Slot: 520, CatalogID: 0x767db96d, Class: 0}, // arme
		{TimestampUS: 1_100_000, Slot: 520, CatalogID: 0xbcabbe43, Class: 2}, // grenade
		{TimestampUS: 1_200_000, Slot: 520, CatalogID: 0xeef5d48d, Class: 3}, // équipement
	}
	got, cov := buildPickups(in, pkClock(equipement), nil, filmdec.BipedPickupStats{}, armes)
	if len(got) != 3 {
		t.Fatalf("publiés = %d, attendu 3", len(got))
	}
	if got[0].Family != "hinf_ma40_ar" {
		t.Errorf("arme : family = %q, attendu \"hinf_ma40_ar\" (weapon_key du catalogue d armes)", got[0].Family)
	}
	if got[1].Family != "grenade_frag" {
		t.Errorf("grenade : family = %q, attendu \"grenade_frag\" (manifeste d équipement)", got[1].Family)
	}
	if got[2].Family != "thruster" {
		t.Errorf("équipement : family = %q, attendu \"thruster\"", got[2].Family)
	}
	if cov.UnknownFamilies != 0 {
		t.Errorf("unknownFamilies = %d, attendu 0 : les trois identifiants sont catalogués", cov.UnknownFamilies)
	}
}

// TestBuildPickupsKindPerClassLiteral — la table classe -> nature, valeur par valeur.
//
// LA CLASSE DE REPLI EST TESTÉE, et c'est le point : le R(3) porte huit valeurs, quatre sont
// observées sur le corpus. Une cinquième ne doit pas se publier comme une grenade au motif
// qu'elle n'est pas une arme — elle sort `item`, la valeur du schéma 30 qui n'a PAS été
// renommée.
func TestBuildPickupsKindPerClassLiteral(t *testing.T) {
	cas := []struct {
		classe uint8
		veut   string
	}{{0, "weapon"}, {1, "weapon"}, {2, "grenade"}, {3, "equipment"}, {5, "item"}, {7, "item"}}
	for _, c := range cas {
		in := []filmdec.BipedPickup{{TimestampUS: 1_000_000, Slot: 520, CatalogID: 1, Class: c.classe}}
		got, _ := buildPickups(in, pkClock(nil), nil, filmdec.BipedPickupStats{}, nil)
		if len(got) != 1 {
			t.Fatalf("classe %d : %d publié(s), attendu 1", c.classe, len(got))
		}
		if string(got[0].Kind) != c.veut {
			t.Errorf("classe %d : kind = %q, attendu %q", c.classe, got[0].Kind, c.veut)
		}
	}
}

// TestBuildPickupsUnknownFamilyIsCountedNotInvented — un identifiant hors catalogue sort SANS
// nom, et le compteur le dit.
//
// C'EST LA MESURE QUI PROTÈGE DU SILENCE. `family` étant `omitempty`, un artefact où rien ne se
// résout est indiscernable d'un artefact où tout va bien — sauf par ce compteur. Le manifeste
// ne déclare que 21 objets ; un 22e doit se VOIR.
func TestBuildPickupsUnknownFamilyIsCountedNotInvented(t *testing.T) {
	in := []filmdec.BipedPickup{
		{TimestampUS: 1_000_000, Slot: 520, CatalogID: 0xbcabbe43, Class: 2}, // connu
		{TimestampUS: 1_100_000, Slot: 520, CatalogID: 0xdeadbeef, Class: 3}, // INCONNU
		{TimestampUS: 1_200_000, Slot: 520, CatalogID: 0xfeedface, Class: 0}, // arme INCONNUE
	}
	got, cov := buildPickups(in, pkClock(map[uint32]string{0xbcabbe43: "grenade_frag"}),
		nil, filmdec.BipedPickupStats{}, map[uint32]string{})
	if len(got) != 3 {
		t.Fatalf("publiés = %d, attendu 3", len(got))
	}
	if got[1].Family != "" || got[2].Family != "" {
		t.Errorf("identifiants inconnus : family = %q et %q, attendu vides — on ne devine pas un nom",
			got[1].Family, got[2].Family)
	}
	if cov.UnknownFamilies != 2 {
		t.Errorf("unknownFamilies = %d, attendu 2", cov.UnknownFamilies)
	}
	if got[0].Family != "grenade_frag" {
		t.Errorf("l identifiant connu doit rester nommé : family = %q", got[0].Family)
	}
}

// TestPickupFamilyNeverCrossesCatalogs — L'INVERSION : le catalogue est choisi par la NATURE.
//
// On donne au test des tables VOLONTAIREMENT croisées — le même identifiant des deux côtés sous
// deux noms différents — et on exige que chaque nature lise SA table. Si la production essayait
// les deux catalogues « au cas où », ce test tomberait. Et il doit tomber : les deux espaces
// sont disjoints en production, donc une résolution croisée serait la preuve d'une erreur,
// jamais un repli utile.
func TestPickupFamilyNeverCrossesCatalogs(t *testing.T) {
	const id = 0x12345678
	equipement := map[uint32]string{id: "cote_equipement"}
	armes := map[uint32]string{id: "cote_arme"}
	in := []filmdec.BipedPickup{
		{TimestampUS: 1_000_000, Slot: 520, CatalogID: id, Class: 0}, // arme
		{TimestampUS: 1_100_000, Slot: 520, CatalogID: id, Class: 3}, // équipement
		{TimestampUS: 1_200_000, Slot: 520, CatalogID: id, Class: 6}, // repli : aucun catalogue
	}
	got, _ := buildPickups(in, pkClock(equipement), nil, filmdec.BipedPickupStats{}, armes)
	if got[0].Family != "cote_arme" {
		t.Errorf("classe arme : family = %q, attendu \"cote_arme\" — le catalogue d armes doit primer", got[0].Family)
	}
	if got[1].Family != "cote_equipement" {
		t.Errorf("classe équipement : family = %q, attendu \"cote_equipement\"", got[1].Family)
	}
	if got[2].Family != "" {
		t.Errorf("classe de repli : family = %q, attendu vide — sa nature n est pas établie, "+
			"aucun catalogue ne la concerne", got[2].Family)
	}

	// LE CAS QUI FAIT VRAIMENT TOMBER UN REPLI CROISÉ, et il manquait à la première version de
	// ce test. Ci-dessus, la table d'équipement TROUVE toujours : une production qui essaierait
	// le catalogue d'armes « en repli quand l'équipement ne rend rien » resterait verte, parce
	// que le repli n'est jamais atteint. Vérifié par inversion le 2026-09-01 : le repli croisé
	// passait ce test. Ici l'identifiant est ABSENT de la table d'équipement et PRÉSENT dans
	// celle des armes — un repli le nommerait, et il ne doit pas.
	const orphelin = 0x87654321
	seul := []filmdec.BipedPickup{
		{TimestampUS: 1_000_000, Slot: 520, CatalogID: orphelin, Class: 3}, // équipement
		{TimestampUS: 1_100_000, Slot: 520, CatalogID: orphelin, Class: 2}, // grenade
	}
	gotSeul, cov := buildPickups(seul, pkClock(map[uint32]string{}), nil,
		filmdec.BipedPickupStats{}, map[uint32]string{orphelin: "hinf_ma40_ar"})
	for i, p := range gotSeul {
		if p.Family != "" {
			t.Errorf("non-arme %d : family = %q, attendu vide — l identifiant n est PAS dans le "+
				"manifeste d équipement, et le catalogue d armes ne doit pas servir de repli", i, p.Family)
		}
	}
	if cov.UnknownFamilies != 2 {
		t.Errorf("unknownFamilies = %d, attendu 2 : les deux doivent être comptés comme non résolus",
			cov.UnknownFamilies)
	}

	// Et la symétrique : une ARME absente du catalogue d'armes ne doit pas piocher dans le
	// manifeste d'équipement.
	arme := []filmdec.BipedPickup{{TimestampUS: 1_000_000, Slot: 520, CatalogID: orphelin, Class: 0}}
	gotArme, _ := buildPickups(arme, pkClock(map[uint32]string{orphelin: "grenade_frag"}), nil,
		filmdec.BipedPickupStats{}, map[uint32]string{})
	if gotArme[0].Family != "" {
		t.Errorf("arme : family = %q, attendu vide — le manifeste d équipement ne doit pas servir de repli",
			gotArme[0].Family)
	}
}

// TestPickupNamingSurvivesTheThreeWritingConventions — LE PIÈGE DE FORMAT, réglé à la source.
//
// Le chantier précédent a payé un P0 pour avoir compare deux ÉCRITURES d'une même famille. Trois
// conventions coexistent dans ce document :
//
//	`%08x` nu                pickups[].w, weaponChanges[].w         ex. bcabbe43
//	`"0x"` + MAJUSCULES      loadouts[].w, weaponPads[].weapon      ex. 0xBCABBE43
//	`"0x"` + minuscules      les identifiants du manifeste du titre ex. 0xbcabbe43
//
// LA RÉSOLUTION DU NOM N'EN TRAVERSE AUCUNE : les deux catalogues sont keyés par `uint32` et
// `BipedPickup.CatalogID` EST cet `uint32`. Ce test verrouille les trois écritures et vérifie
// qu'elles se ramènent au même entier ; le contrôle sur le manifeste RÉEL (ci-dessous) ferme la
// chaîne côté fichier.
func TestPickupNamingSurvivesTheThreeWritingConventions(t *testing.T) {
	const id uint32 = 0xbcabbe43
	nu := fmt.Sprintf("%08x", id)
	maj := formatWeaponFamily(id)
	min := "0x" + fmt.Sprintf("%08x", id)
	if nu != "bcabbe43" || maj != "0xBCABBE43" || min != "0xbcabbe43" {
		t.Fatalf("les trois écritures ont changé : %q / %q / %q — revoir ce test avant la production",
			nu, maj, min)
	}
	for _, s := range []string{nu, maj, min} {
		k, ok := padFamilyKey(s)
		if !ok || k != "bcabbe43" {
			t.Errorf("%q se normalise en %q (ok=%v), attendu \"bcabbe43\"", s, k, ok)
		}
	}
	// Et la résolution de production ne dépend d'AUCUNE d'elles : elle passe par l'entier.
	if got := pickupFamily(id, "grenade", map[uint32]string{id: "grenade_frag"}, nil); got != "grenade_frag" {
		t.Errorf("pickupFamily = %q, attendu \"grenade_frag\"", got)
	}
}

// TestPickupFamilyResolvesAgainstTheRealManifest — le manifeste du titre, tel qu'il est sur
// disque, et non une table réécrite à la main.
//
// POURQUOI CE TEST EXISTE. Les précédents prouvent que la production résout ce qu'on lui donne ;
// celui-ci prouve que le MANIFESTE RÉEL arrive jusqu'à elle, casse comprise. Les 21 entrées de
// `[[equipment_objects]]` s'écrivent `"0x"` + minuscules (vérifié sur pièces le 2026-09-01) et
// `tagGlobalID32` les parse en `uint32`. Si quelqu'un changeait cette écriture ou le parseur, ce
// test le dirait.
//
// LES SIX IDENTIFIANTS SONT CEUX QUE LES DEUX FILMS DE RÉFÉRENCE PORTENT, et deux d'entre eux
// ont été établis par corrélation AVANT que le manifeste soit consulté (`eef5d48d` = propulseur,
// `8e2dc574` = mur). Ils ne sont pas choisis pour la commodité du test.
func TestPickupFamilyResolvesAgainstTheRealManifest(t *testing.T) {
	familles := goldenReplayLabels(t).EquipmentObjects()
	cas := []struct {
		id   uint32
		veut string
	}{
		{0xeef5d48d, "thruster"},
		{0x8e2dc574, "wall"},
		{0xbcabbe43, "grenade_frag"},
		{0xcaaadcb0, "grenade_plasma"},
		{0x8c77ffe7, "grapple"},
		{0x72199cba, "sensor"},
	}
	for _, c := range cas {
		if got := pickupFamily(c.id, "equipment", familles, nil); got != c.veut {
			t.Errorf("%08x : family = %q, attendu %q (manifeste du titre)", c.id, got, c.veut)
		}
	}
	// Un identifiant absent du manifeste ne rend RIEN, même s'il ressemble à un voisin.
	if got := pickupFamily(0xeef5d48e, "equipment", familles, nil); got != "" {
		t.Errorf("identifiant absent : family = %q, attendu vide", got)
	}
}
