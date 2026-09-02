package replay

// vehicle_families_test.go — LE GARDE-RAIL de la table d identite des chassis.
//
// CE QU IL PROTEGE. La table est une donnee de reference recopiee a la main depuis des rapports
// de retro-ingenierie : le risque n est pas qu elle soit vide, c est qu une entree y soit fausse
// (un chassis pointant un sprite qui n existe pas) ou incoherente (deux ecritures du meme
// identifiant). Les deux se voient ici, sans film et sans environnement.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// vehicleSpriteIndexPath : l index du lot A, seule source de verite des familles SERVIES.
// Chemin relatif au paquet — quatre remontees jusqu a la racine du depot.
const vehicleSpriteIndexPath = "../../../../../static/vehicles-assets/halo_infinite/replay/index.json"

// TestVehicleFamilyOfResoutLesChassisObserves : les valeurs `MPPWord32` OBSERVEES dans les
// films du corpus (RE_DEFAULTSTATE_TI40_2026-08-31 § 8.4) sont celles dont la resolution est
// verifiee de bout en bout. `0xfe32c0f4`, longtemps non resolu (labels.tsv sans banque de
// sons), a ete tranche le 2026-09-02 par la chaine de destruction : vehi -> hlmt daf7f543 =
// le hlmt du Warthog (V3D_DESTRUCTION_SONS_2026-09-02, table des verdicts).
func TestVehicleFamilyOfResoutLesChassisObserves(t *testing.T) {
	for _, c := range []struct {
		chassis uint32
		want    string
		note    string
	}{
		{0x5b80c406, "ghost", "labels.tsv : vehi 5b80c406, sb_010_veh_cv_ghost"},
		{0xc6e79dcc, "banshee", "labels.tsv : vehi c6e79dcc, sb_010_veh_cv_banshee"},
		{0xaf31ab1a, "mongoose", "REWORK_WARTHOG_GUNGOOSE § 1 : variante de chassis mongoose"},
		{0x0000254b, "falcon", "V4_RAPPORT_SPRITES § 4 : classe par ses noms de maillage"},
		{0x3d4a8a5a, "chopper", "labels.tsv : deux entrees a porteur unique, sb_010_veh_bt_chopper"},
		{0xfe32c0f4, "warthog", "V3D : vehi fe32c0f4 -> hlmt daf7f543 = hlmt du Warthog"},
		{0xcb96ca07, "warthog", "V3D, meme table : meme hlmt daf7f543 ; non observe en film"},
	} {
		if got := vehicleFamilyOf(c.chassis); got != c.want {
			t.Errorf("vehicleFamilyOf(%#08x) = %q, attendu %q (%s)", c.chassis, got, c.want, c.note)
		}
	}
}

// TestVehicleFamilyOfInconnuRendVide : une valeur hors table ne prend PAS la famille d une
// voisine. C est la regle de degradation du calque, et elle vaut d etre figee : un `map` Go rend
// deja la valeur zero, mais rien n empeche un futur contributeur d y coller un repli.
func TestVehicleFamilyOfInconnuRendVide(t *testing.T) {
	for _, chassis := range []uint32{0, 0xdeadbeef, 0xffffffff} {
		if got := vehicleFamilyOf(chassis); got != "" {
			t.Errorf("vehicleFamilyOf(%#08x) = %q, attendu vide : emprunter la famille d un "+
				"voisin dessinerait un Warthog en Banshee", chassis, got)
		}
	}
}

// TestFormatChassisIDEstHexa8Minuscule : la MEME convention d ecriture que `Loadout.W` et
// `GroundWeapon.W`. Deux conventions dans un meme document rendraient toute jointure cote client
// impossible.
func TestFormatChassisIDEstHexa8Minuscule(t *testing.T) {
	for _, c := range []struct {
		in   uint32
		want string
	}{
		{0x5b80c406, "5b80c406"},
		{0x0000254b, "0000254b"},
		{0, "00000000"},
		{0xFFFFFFFF, "ffffffff"},
	} {
		if got := formatChassisID(c.in); got != c.want {
			t.Errorf("formatChassisID(%#08x) = %q, attendu %q", c.in, got, c.want)
		}
	}
}

// TestVehicleFamillesServiesParLeLotA : CHAQUE famille de la table pointe un sprite QUI EXISTE.
//
// C EST LE GARDE-RAIL QUI COMPTE. Une entree dont la famille n a pas de PNG servi produit une URL
// morte : le client demande une image, recoit un 404, et n affiche rien — sans qu aucun compteur
// ne le dise (la couverture, elle, verrait la famille comme RESOLUE). Le test lit l index du
// lot A ; s il est absent (checkout partiel), il SAUTE plutot que d echouer, mais il ne se tait
// jamais sur une famille manquante quand l index est la.
func TestVehicleFamillesServiesParLeLotA(t *testing.T) {
	raw, err := os.ReadFile(filepath.Clean(vehicleSpriteIndexPath))
	if err != nil {
		t.Skipf("index des sprites de vehicule absent (%v) — garde-rail saute", err)
	}
	var index []struct {
		File    string `json:"file"`
		Famille string `json:"famille"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatalf("index des sprites illisible : %v", err)
	}
	servies := make(map[string]bool, len(index))
	for _, e := range index {
		servies[e.Famille] = true
	}
	if len(servies) == 0 {
		t.Fatal("index des sprites vide : le garde-rail ne verifie plus rien")
	}
	for chassis, famille := range vehicleFamilyByChassis {
		if famille == "" {
			t.Errorf("chassis %#08x mappe sur une famille VIDE : une entree sans famille ne "+
				"doit pas figurer dans la table, l absence suffit", chassis)
			continue
		}
		if !servies[famille] {
			t.Errorf("chassis %#08x -> famille %q, ABSENTE de l index des sprites du lot A : "+
				"l URL composee serait morte, et la couverture compterait la famille comme "+
				"resolue", chassis, famille)
		}
	}
}
