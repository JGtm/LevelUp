package replay

// Tests — LE CROISEMENT « ALLUMÉS SEULEMENT ».
//
// CE QUE CE FICHIER VERROUILLE, et chaque point EST la décision produit du 2026-08-19
// (« on ne les affiche que si allumés ») :
//   - un emplacement du catalogue que le match NE confirme PAS ne part jamais au client.
//     C'est le cas du Super Fiesta sur Cliffhanger : dix-sept au fichier, zéro à l'écran ;
//   - un emplacement confirmé part avec la position DU FICHIER, au centimètre, et l'index
//     du socle de match qui le confirme — la présence reste celle du match ;
//   - un socle du match qu'aucun emplacement ne réclame n'est pas cité : il reste publié
//     par `weaponPads` et se dessine comme avant. Le film fait foi, le catalogue complète ;
//   - un socle de match ne confirme qu'UN emplacement : deux emplacements voisins ne se
//     dessinent pas deux fois au même endroit ;
//   - `catalogN` dit toujours combien la carte en porte, confirmés ou non.

import (
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// spot fabrique un emplacement de catalogue.
func spot(x, y, z float64, fam string) MapWeaponPadSpot {
	return MapWeaponPadSpot{Pos: mapvar.Vec3{X: x, Y: y, Z: z}, TypeID: "0x6253CFC0",
		Family: fam, Objects: 1}
}

// filmPad fabrique un socle tel que l'artefact du match le publie.
func filmPad(x, y, z float32) WeaponPad {
	return WeaponPad{X: x, Y: y, Z: z, Weapon: "0x0A1992BC", Spawns: []int{0}}
}

// carteTemoin — trois emplacements au fichier, dont deux seulement seront confirmés.
func carteTemoin() MapWeaponPadsEntry {
	return MapWeaponPadsEntry{MapID: "m1", MvarFile: "m1.mvar", Pads: []MapWeaponPadSpot{
		spot(-9.738, -0.003, 22.403, "power"),
		spot(5.160, -0.003, 26.501, "rack"),
		spot(0.257, -0.003, 21.360, "powerup"), // celui que le film ne verra pas
	}}
}

func TestBuildMapWeaponPads_AllumesSeulement(t *testing.T) {
	// Le film voit les deux premiers, au centimètre près, et rien d'autre.
	pads := []WeaponPad{filmPad(-9.74, 0, 22.40), filmPad(5.16, 0, 26.51)}
	got := BuildMapWeaponPads(carteTemoin(), pads)
	if got == nil {
		t.Fatal("calque nil alors que deux emplacements sont confirmés")
	}
	if len(got.Pads) != 2 {
		t.Fatalf("%d emplacements servis, attendu 2 : %+v", len(got.Pads), got.Pads)
	}
	if got.CatalogN != 3 {
		t.Errorf("catalogN = %d, attendu 3 — le calque doit dire ce qu'il n'affiche pas", got.CatalogN)
	}
	// LA POSITION EST CELLE DU FICHIER, pas celle du film : c'est tout l'intérêt.
	if got.Pads[0].X != -9.738 || got.Pads[0].Pad != 0 {
		t.Errorf("premier emplacement = %+v, attendu la position du catalogue et le socle 0", got.Pads[0])
	}
	if got.Pads[1].Pad != 1 {
		t.Errorf("second emplacement lié au socle %d, attendu 1", got.Pads[1].Pad)
	}
	// LE TROISIÈME N'EST PAS SERVI : aucun socle du match ne le confirme.
	for _, p := range got.Pads {
		if p.X > 0.2 && p.X < 0.3 {
			t.Errorf("un emplacement NON confirmé est parti au client : %+v", p)
		}
	}
}

// TestBuildMapWeaponPads_AucunConfirme — le témoin du Super Fiesta : la carte porte des
// emplacements, le match n'en sert aucun, RIEN ne part.
func TestBuildMapWeaponPads_AucunConfirme(t *testing.T) {
	if got := BuildMapWeaponPads(carteTemoin(), nil); got != nil {
		t.Fatalf("calque servi sans aucun socle au match : %+v", got)
	}
	// Et un socle très loin des emplacements ne confirme rien non plus.
	if got := BuildMapWeaponPads(carteTemoin(), []WeaponPad{filmPad(300, 300, 300)}); got != nil {
		t.Fatalf("calque servi alors que le socle du match est à 300 m : %+v", got)
	}
}

// TestBuildMapWeaponPads_SocleFilmSansCatalogue — un socle que le catalogue ignore n'est
// cité par aucun emplacement ; il reste au client par `weaponPads`, inchangé.
func TestBuildMapWeaponPads_SocleFilmSansCatalogue(t *testing.T) {
	pads := []WeaponPad{filmPad(-9.74, 0, 22.40), filmPad(100, 100, 5)}
	got := BuildMapWeaponPads(carteTemoin(), pads)
	if got == nil || len(got.Pads) != 1 {
		t.Fatalf("un seul emplacement doit être confirmé : %+v", got)
	}
	if got.Pads[0].Pad != 0 {
		t.Errorf("emplacement lié au socle %d, attendu 0", got.Pads[0].Pad)
	}
}

// TestBuildMapWeaponPads_UnSocleUnEmplacement — deux emplacements du fichier à moins d'un
// mètre du MÊME socle : un seul est confirmé, l'autre reste éteint.
func TestBuildMapWeaponPads_UnSocleUnEmplacement(t *testing.T) {
	e := MapWeaponPadsEntry{MapID: "m1", Pads: []MapWeaponPadSpot{
		spot(0, 0, 0, "rack"),
		spot(0.4, 0, 0, "rack"),
	}}
	got := BuildMapWeaponPads(e, []WeaponPad{filmPad(0.05, 0, 0)})
	if got == nil || len(got.Pads) != 1 {
		t.Fatalf("un socle ne confirme qu'un emplacement : %+v", got)
	}
	if got.Pads[0].X != 0 {
		t.Errorf("le PLUS PROCHE doit l'emporter, servi : %+v", got.Pads[0])
	}
	if got.CatalogN != 2 {
		t.Errorf("catalogN = %d, attendu 2", got.CatalogN)
	}
}

// TestBuildMapWeaponPads_SeuilStrict — le mètre est une borne, pas une approximation.
func TestBuildMapWeaponPads_SeuilStrict(t *testing.T) {
	e := MapWeaponPadsEntry{MapID: "m1", Pads: []MapWeaponPadSpot{spot(0, 0, 0, "rack")}}
	if got := BuildMapWeaponPads(e, []WeaponPad{filmPad(0.999, 0, 0)}); got == nil {
		t.Error("99,9 cm doit confirmer")
	}
	if got := BuildMapWeaponPads(e, []WeaponPad{filmPad(1.001, 0, 0)}); got != nil {
		t.Errorf("1,001 m ne doit rien confirmer : %+v", got)
	}
	// ET LA TROISIÈME DIMENSION COMPTE : même x/y, un étage plus haut, ce n'est pas le
	// même socle (Cliffhanger en porte de superposés).
	if got := BuildMapWeaponPads(e, []WeaponPad{filmPad(0, 0, 3)}); got != nil {
		t.Errorf("trois mètres plus haut, ce n'est pas le même socle : %+v", got)
	}
}

// TestBuildMapWeaponPads_CarteVide — une carte MESURÉE sans socle ne sert rien, et ne
// panique pas.
func TestBuildMapWeaponPads_CarteVide(t *testing.T) {
	e := MapWeaponPadsEntry{MapID: "m1", Pads: nil}
	if got := BuildMapWeaponPads(e, []WeaponPad{filmPad(0, 0, 0)}); got != nil {
		t.Fatalf("carte sans emplacement : %+v", got)
	}
}
