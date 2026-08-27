package replay

// ground_weapon_flag_exclusion_test.go — LE GARDE-RAIL DE LA DECISION 1 du plan
// `.ai/V7.5/replay2d/PLAN_DRAPEAU_OBJET.md` : un objet d'objectif du manifeste ne fait JAMAIS
// un socle d'arme.
//
// POURQUOI UN GARDE-RAIL PLUTOT QU'UN COMMENTAIRE. Le drapeau etait deja ecarte AVANT ce lot,
// mais par ACCIDENT — son identifiant n'est pas au catalogue d'armes, exactement comme le bruit
// du balayage. Un accident ne se maintient pas : le jour ou l'identifiant entrerait au
// catalogue (repli d'alias, elargissement de l'enum), le rejeu publierait un socle de fusil a
// la base de chaque equipe, et rien ne le dirait. Le test ci-dessous echoue ce jour-la.
//
// LES DEUX TESTS NE DISENT PAS LA MEME CHOSE, et il faut les deux :
//
//	le premier prouve que l'exclusion PRIME sur le catalogue d'armes — il declare drapeau une
//	  famille qui EST au catalogue, et son temoin (meme scan, table vide) publie bien le socle
//	  que l'exclusion supprime. Sans ce temoin, un test vert ne prouverait rien ;
//	le second prouve que c'est le manifeste DU TITRE qui alimente la table, avec l'identifiant
//	  qu'il declare — pas une constante recopiee dans un test.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games/mappings"
)

// gwFlagScan monte QUATRE apparitions au repos au meme endroit sous l'identifiant `id` : le cas
// qui fait un socle quand l'identite se resout en arme (cf. gwTestPadScan, meme geometrie).
func gwFlagScan(id uint32) (WorldObjectScan, []filmdec.BipedPosition) {
	kf := []uint64{0, 20_000_000, 40_000_000, 60_000_000, 80_000_000}
	scan := WorldObjectScan{
		Scanned: true,
		Stats:   filmdec.EquipmentCreationStats{Slots: 8, Anchors: 40, Accepted: 4},
		Creations: []filmdec.EquipmentCreation{
			gwTestCreation(10, 0, 1_000_000, id, 10, 10),
			gwTestCreation(11, 0, 31_000_000, id, 10.2, 10.1),
			gwTestCreation(12, 0, 51_000_000, id, 9.9, 10.2),
			gwTestCreation(13, 0, 71_000_000, id, 10.1, 9.8),
		},
		Keyframes: filmdec.WorldObjectKeyframes{
			TimesUS: kf,
			SeenUS: map[filmdec.EquipmentLifeKey][]uint64{
				{Slot: 10}: {20_000_000}, {Slot: 11}: {40_000_000},
				{Slot: 12}: {60_000_000}, {Slot: 13}: {80_000_000},
			},
		},
	}
	return scan, gwTestPositions([]uint64{25_000_000, 45_000_000, 65_000_000}, 10, 10)
}

// TestUnObjetDObjectifNeFaitJamaisUnSocle : l'exclusion prime sur le catalogue d'armes, et le
// temoin montre le socle qu'elle supprime.
func TestUnObjetDObjectifNeFaitJamaisUnSocle(t *testing.T) {
	fam := gwTestFamily(t, 0) // une famille du CATALOGUE d'armes : le pire cas
	scan, pos := gwFlagScan(fam)

	// TEMOIN — sans table d'objets d'objectif, ces quatre apparitions font un socle. C'est ce
	// qui rend le test suivant capable d'echouer.
	temoin, _, covT := buildWeaponPads(PadScans{Weapons: scan}, pos, gwTestClock(), padCatalogs{})
	if len(temoin) != 1 {
		t.Fatalf("temoin : 1 socle attendu sans table d'objets d'objectif, %d obtenus", len(temoin))
	}
	if covT.Objectives != 0 {
		t.Fatalf("temoin : %d objets d'objectif comptes alors que la table est vide", covT.Objectives)
	}

	// LA REGLE — le meme identifiant declare DRAPEAU par le titre : aucun socle.
	pads, picks, cov := buildWeaponPads(PadScans{Weapons: scan}, pos, gwTestClock(),
		padCatalogs{ObjectiveObjects: map[uint32]Label{fam: {En: "Flag", Fr: "Drapeau"}}})
	for _, p := range pads {
		if p.Weapon == formatWeaponFamily(fam) {
			t.Errorf("un socle porte la famille DRAPEAU %q — un drapeau n'est pas un socle d'arme",
				p.Weapon)
		}
	}
	if len(pads) != 0 || len(picks) != 0 {
		t.Fatalf("%d socle(s) et %d occupation(s) publies pour un objet d'objectif, attendu 0 et 0",
			len(pads), len(picks))
	}
	if cov.Objectives != 4 || cov.Rejected != 4 || cov.Kept != 0 {
		t.Errorf("couverture : objetsDObjectif=%d ecartees=%d retenues=%d, attendu 4/4/0 — "+
			"l'ecart doit se COMPTER, pas disparaitre", cov.Objectives, cov.Rejected, cov.Kept)
	}
	if !cov.Balanced() {
		t.Errorf("invariant de couverture ROMPU : %+v", *cov)
	}
}

// TestLeDrapeauDuManifesteNeFaitJamaisUnSocle : la table vient du MANIFESTE DU TITRE, et
// l'identifiant qu'il declare ne peut pas devenir un socle.
func TestLeDrapeauDuManifesteNeFaitJamaisUnSocle(t *testing.T) {
	cat := goldenCatalog(t)
	if len(cat.ObjectiveObjects) == 0 {
		t.Fatal("le manifeste du titre ne declare AUCUN objet d'objectif de famille drapeau — " +
			"la chaine des socles ne peut alors rien reconnaitre (cf. replay_labels.toml, " +
			"[[objective_objects]])")
	}
	for id := range cat.ObjectiveObjects {
		scan, pos := gwFlagScan(id)
		pads, _, cov := buildWeaponPads(PadScans{Weapons: scan}, pos, gwTestClock(),
			padCatalogs{ObjectiveObjects: cat.ObjectiveObjects})
		if len(pads) != 0 {
			t.Errorf("l'identifiant de drapeau %s du manifeste a produit %d socle(s)",
				formatWeaponFamily(id), len(pads))
		}
		if cov.Objectives != 4 {
			t.Errorf("%s : %d objets d'objectif comptes, attendu 4",
				formatWeaponFamily(id), cov.Objectives)
		}
	}
}

// TestLeManifesteNommeSesObjetsDObjectifDansLesDeuxLangues : le nom vit dans le TOML, jamais en
// Go — et CHAQUE famille portee doit avoir au moins un representant nomme.
//
// LES DEUX FAMILLES SONT EXIGEES SEPAREMENT (2026-08-27). Compter globalement laisserait passer
// la disparition d'une famille entiere tant que l'autre survit : le jour ou l'entree du crane
// serait retiree du manifeste, l'exclusion des socles d'armes redeviendrait ACCIDENTELLE sur
// Oddball sans qu'aucun test ne l'annonce — exactement le defaut que ce fichier existe pour
// empecher sur le drapeau.
func TestLeManifesteNommeSesObjetsDObjectifDansLesDeuxLangues(t *testing.T) {
	labels := goldenReplayLabels(t)
	attendues := []string{mappings.ObjectiveFamilyFlag, mappings.ObjectiveFamilyBall}
	nb := map[string]int{}
	for id, o := range labels.ObjectiveObjects() {
		nb[o.Family]++
		if o.Label.En == "" || o.Label.Fr == "" {
			t.Errorf("objet d'objectif %s (famille %q) nomme dans une seule langue (en=%q fr=%q)",
				formatWeaponFamily(id), o.Family, o.Label.En, o.Label.Fr)
		}
	}
	for _, f := range attendues {
		if nb[f] == 0 {
			t.Errorf("aucun objet d'objectif de famille %q au manifeste du titre — la chaine des "+
				"socles ne reconnaitrait plus cet objet, elle l'ecarterait par accident "+
				"(cf. replay_labels.toml, [[objective_objects]])", f)
		}
	}
}
