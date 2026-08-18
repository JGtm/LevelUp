package replay

// powerup_pads_test.go — LA VOIE `ti=37` DE LA CHAINE DES SOCLES, sur des entrees SYNTHETIQUES.
//
// POURQUOI SANS FILM, meme raison que `ground_weapon_pads_test.go` : les verdicts de corpus
// (quel socle, ou, a quel cycle) vivent dans les instruments sous garde, qui lisent de vrais
// films et publient leurs denominateurs au plan. Ce fichier-ci verrouille ce que le gate
// ordinaire peut tenir — que la voie RETIENNE ce qu'elle doit retenir et ECARTE ce qu'elle doit
// ecarter.
//
// LES TROIS REFUS SONT TESTES, parce qu'ils sont la moitie du resultat :
//
//	UNE APPARITION SEULE N'EST PAS UN SOCLE. Le temoin fantome du balayage de creations retient
//	jusqu'a 5,9 creations isolees sur un film mesure : sans le seuil de recurrence, ce bruit-la
//	deviendrait des socles.
//	UN OBJET QUI A BOUGE N'EST PAS UN SOCLE. C'est le power-up LACHE a une mort — il tombe, donc
//	il emet des positions delta. Il reste publie par `equipmentPlacements` avec son origine.
//	UNE FAMILLE QUE LE MANIFESTE NE NOMME PAS NE DONNE RIEN. C'est le filtre d'identite, seul
//	garde-fou selectif de la voie : l'en-tete de creation, lui, ne l'est pas.

import (
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// LES DEUX IDENTITES `eqip` DU MANIFESTE (replay_labels.toml), ecrites ici pour que le test
// n'ait pas a lire le TOML : ce qu'il verrouille est la REGLE, pas le catalogue. Le catalogue,
// lui, est verrouille par `TestPowerupPadRuleSuitLeManifesteDuTitre` ci-dessous.
const (
	puTestOvershield = uint32(0xb781197a)
	puTestCamo       = uint32(0xe7be9f5c)
	// puTestMur est un identifiant d'equipement du MEME catalogue qui n'est PAS un power-up :
	// c'est le temoin negatif du filtre de famille.
	puTestMur = uint32(0x8e2dc574)
)

// puTestFamilies est la table d'identite `eqip` que la voie consomme, reduite a ce que ces
// tests exercent.
func puTestFamilies() map[uint32]string {
	return map[uint32]string{
		puTestOvershield: "powerup_overshield",
		puTestCamo:       "powerup_camo",
		puTestMur:        "wall",
	}
}

// puTestCatalogs rend les deux tables d'identite de la chaine, avec la seule voie `ti=37`
// renseignee.
func puTestCatalogs() padCatalogs {
	return padCatalogs{EquipmentFamilies: puTestFamilies()}
}

// puTestScan monte le cas de reference de la voie `ti=37` : TROIS creations du meme power-up au
// meme point, a des slots DIFFERENTS (c'est ce que la mesure a vu — le socle prend un slot neuf
// a chaque reapparition), recensees une image-cle chacune.
//
// AUCUNE PISTE DELTA, et c'est le coeur du lot : un objet de socle ne bouge jamais, donc il
// n'emet aucune position. C'est precisement ce qui le rendait invisible a `confirmPlacements`.
func puTestScan(id uint32, x, y float32) WorldObjectScan {
	return WorldObjectScan{
		Scanned: true,
		Stats:   filmdec.EquipmentCreationStats{Slots: 12, Anchors: 60, Accepted: 3},
		Creations: []filmdec.EquipmentCreation{
			puTestCreation(20, 0, 1_000_000, id, x, y),
			puTestCreation(21, 0, 31_000_000, id, x+0.2, y-0.1),
			puTestCreation(22, 0, 51_000_000, id, x-0.1, y+0.2),
		},
		Keyframes: filmdec.WorldObjectKeyframes{
			TimesUS: []uint64{0, 20_000_000, 40_000_000, 60_000_000, 80_000_000},
			SeenUS: map[filmdec.EquipmentLifeKey][]uint64{
				{Slot: 20}: {20_000_000},
				{Slot: 21}: {40_000_000},
				{Slot: 22}: {60_000_000},
			},
		},
	}
}

// puTestCreation fabrique un record de creation `ti=37` porteur d'une identite `eqip`.
func puTestCreation(
	slot, gen uint32, atUS uint64, id uint32, x, y float32,
) filmdec.EquipmentCreation {
	c := filmdec.EquipmentCreation{Slot: slot, Gen: gen, TimestampUS: atUS, X: x, Y: y}
	c.MPPPresent[filmdec.MPPWord32] = true
	c.MPPVal[filmdec.MPPWord32] = uint64(id)
	return c
}

// TestPowerupPadsGrappeUnSocle : trois creations du meme power-up a moins d'un metre font UN
// socle, publie sous sa FAMILLE et non sous son identifiant `eqip`.
func TestPowerupPadsGrappeUnSocle(t *testing.T) {
	pos := gwTestPositions(nil, 0, 0)
	pads, _, cov := buildWeaponPads(
		PadScans{Powerups: puTestScan(puTestOvershield, 10, 10)}, pos, gwTestClock(),
		puTestCatalogs())
	if len(pads) != 1 {
		t.Fatalf("1 socle de power-up attendu, %d obtenus : %+v", len(pads), pads)
	}
	if pads[0].Weapon != "powerup_overshield" {
		t.Errorf("le socle publie %q — la FAMILLE du manifeste est attendue, pas l'hexadecimal"+
			" (`weaponLabels` ne nomme que des armes, et la regle de taille du calque lit la"+
			" famille)", pads[0].Weapon)
	}
	if got := len(pads[0].Spawns); got != 3 {
		t.Errorf("3 apparitions attendues sur le socle, %d : %+v", got, pads[0].Spawns)
	}
	if cov.PowerupPads != 1 || cov.PowerupKept != 3 || cov.PowerupAccepted != 3 {
		t.Errorf("couverture : socles=%d retenues=%d acceptees=%d, attendu 1/3/3",
			cov.PowerupPads, cov.PowerupKept, cov.PowerupAccepted)
	}
	if !cov.PowerupScanned {
		t.Error("powerupScanned faux alors que la lecture a abouti : un film non lu et un film" +
			" sans power-up se liraient pareil")
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestPowerupPadsSeuilDeRecurrence : une apparition SEULE n'est pas un socle.
//
// C'EST LE GARDE-FOU QUE LA MESURE IMPOSE : le temoin fantome du balayage de creations retient
// jusqu'a 5,9 creations isolees sur `64e8adfa`. Sans ce seuil, ce bruit deviendrait des socles.
func TestPowerupPadsSeuilDeRecurrence(t *testing.T) {
	scan := puTestScan(puTestOvershield, 10, 10)
	scan.Creations = scan.Creations[:1]
	scan.Stats.Accepted = 1
	pads, _, cov := buildWeaponPads(
		PadScans{Powerups: scan}, gwTestPositions(nil, 0, 0), gwTestClock(), puTestCatalogs())
	if len(pads) != 0 {
		t.Fatalf("une apparition isolee ne fait pas un socle, %d publies : %+v", len(pads), pads)
	}
	if cov.PowerupKept != 1 {
		t.Errorf("la creation devait etre RETENUE par l'identite (%d) — c'est la GRAPPE qui la"+
			" refuse, et la couverture doit le montrer", cov.PowerupKept)
	}
	if cov.PowerupPads != 0 {
		t.Errorf("%d socle(s) comptes pour zero publie", cov.PowerupPads)
	}
}

// TestPowerupPadsEcarteLesCreationsAVieDelta : un power-up qui a BOUGE n'est pas un socle.
//
// C'est le power-up LACHE a une mort : il tombe, donc il emet des positions delta. Il reste
// publie par `equipmentPlacements` avec son origine `dropped` — cette voie-ci ne le double pas.
func TestPowerupPadsEcarteLesCreationsAVieDelta(t *testing.T) {
	scan := puTestScan(puTestOvershield, 10, 10)
	scan.Tracks = puTestTracks(scan.Creations)
	pads, _, cov := buildWeaponPads(
		PadScans{Powerups: scan}, gwTestPositions(nil, 0, 0), gwTestClock(), puTestCatalogs())
	if len(pads) != 0 {
		t.Fatalf("un objet qui a bouge n'est pas un socle, %d publies : %+v", len(pads), pads)
	}
	if cov.PowerupKept != 3 {
		t.Errorf("les trois creations devaient etre RETENUES par l'identite (%d) — c'est la vie"+
			" DELTA qui les refuse", cov.PowerupKept)
	}
}

// puTestTracks donne a chaque creation une piste delta qui commence a son instant : la vie a
// donc BOUGE au sens de `gwPickupLifeTrack`.
func puTestTracks(cre []filmdec.EquipmentCreation) []filmdec.ProjectileTrack {
	out := make([]filmdec.ProjectileTrack, 0, len(cre))
	for _, c := range cre {
		out = append(out, filmdec.ProjectileTrack{
			Slot: c.Slot, Gen: c.Gen,
			Pts: []filmdec.ProjectileSample{
				{TimestampUS: c.TimestampUS, X: c.X, Y: c.Y, Z: c.Z},
				{TimestampUS: c.TimestampUS + 200_000, X: c.X + 1.5, Y: c.Y, Z: c.Z},
			},
		})
	}
	return out
}

// TestPowerupPadsFamilleInconnueNeDonneRien : une identite que le manifeste ne resout pas, et
// une identite qu'il resout en AUTRE CHOSE qu'un power-up, ne donnent aucun socle.
//
// LES DEUX CAS SONT DISTINCTS, et il faut les deux : le premier est le bruit du balayage (un
// mot de 32 bits quelconque), le second est un objet d'equipement REEL — un mur pose — que rien
// n'autorise a devenir un socle de power-up.
func TestPowerupPadsFamilleInconnueNeDonneRien(t *testing.T) {
	for _, cas := range []struct {
		nom string
		id  uint32
	}{
		{"identite hors manifeste", 0xdeadbeef},
		{"famille connue mais pas un power-up", puTestMur},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			pads, _, cov := buildWeaponPads(
				PadScans{Powerups: puTestScan(cas.id, 10, 10)}, gwTestPositions(nil, 0, 0),
				gwTestClock(), puTestCatalogs())
			if len(pads) != 0 {
				t.Fatalf("%d socle(s) publies : %+v", len(pads), pads)
			}
			if cov.PowerupKept != 0 {
				t.Errorf("%d creation(s) retenues alors que l'identite ne resout aucun power-up",
					cov.PowerupKept)
			}
			if cov.PowerupAccepted != 3 {
				t.Errorf("le DENOMINATEUR doit rester (%d acceptees, attendu 3) : sans lui, un"+
					" film non lu et un film sans power-up se liraient pareil",
					cov.PowerupAccepted)
			}
		})
	}
}

// TestPowerupPadsIndexDOccupationEstGlobal : les occupations de la voie `ti=37` designent leur
// socle dans le tableau PUBLIE, pas dans leur propre voie.
//
// POURQUOI CE TEST EXISTE. `padPickups[].pad` est un index dans `weaponPads`, et les deux voies
// s'y concatenent. Chaque voie numerote d'abord ses socles a partir de zero ; sans le decalage,
// les occupations des power-ups designeraient les socles d'ARME — une erreur silencieuse, qui
// ne se verrait qu'a l'ecran, sur la mauvaise infobulle.
func TestPowerupPadsIndexDOccupationEstGlobal(t *testing.T) {
	armes, pos := gwTestPadScan(t)
	pads, picks, cov := buildWeaponPads(
		PadScans{Weapons: armes, Powerups: puTestScan(puTestOvershield, 40, 40)},
		pos, gwTestClock(), puTestCatalogs())
	if len(pads) != 2 {
		t.Fatalf("2 socles attendus (1 arme + 1 power-up), %d : %+v", len(pads), pads)
	}
	if pads[0].Weapon == "powerup_overshield" || pads[1].Weapon != "powerup_overshield" {
		t.Fatalf("l'ordre publie doit etre ARMES puis POWER-UPS : %q, %q",
			pads[0].Weapon, pads[1].Weapon)
	}
	if cov.Pads != 1 || cov.PowerupPads != 1 {
		t.Errorf("couverture : socles d'arme=%d, de power-up=%d, attendu 1/1 (et leur somme est"+
			" la longueur de weaponPads)", cov.Pads, cov.PowerupPads)
	}
	for _, k := range picks {
		if k.Pad < 0 || k.Pad >= len(pads) {
			t.Fatalf("occupation hors du tableau des socles : pad=%d pour %d socles",
				k.Pad, len(pads))
		}
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestPowerupPadsVoieNonLueNeDitPasZero : une voie `ti=37` non balayee laisse `powerupScanned`
// faux — le seul moyen de distinguer « pas de power-up » de « pas lu ».
func TestPowerupPadsVoieNonLueNeDitPasZero(t *testing.T) {
	armes, pos := gwTestPadScan(t)
	_, _, cov := buildWeaponPads(PadScans{Weapons: armes}, pos, gwTestClock(), puTestCatalogs())
	if cov.PowerupScanned {
		t.Error("powerupScanned vrai alors qu'aucune lecture ti=37 n'a eu lieu")
	}
	if cov.PowerupAccepted != 0 || cov.PowerupKept != 0 || cov.PowerupPads != 0 {
		t.Errorf("compteurs non nuls sans lecture : %d/%d/%d",
			cov.PowerupAccepted, cov.PowerupKept, cov.PowerupPads)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestPowerupPadRuleSuitLeManifesteDuTitre : les identites que ces tests emploient sont bien
// celles que le MANIFESTE du titre nomme, et la regle les accepte.
//
// SANS CE TEST, LES PRECEDENTS SE MENTIRAIENT A EUX-MEMES : ils fabriquent leur propre table de
// familles, donc ils passeraient meme si le manifeste changeait d'identifiant. Celui-ci relie
// les deux.
func TestPowerupPadRuleSuitLeManifesteDuTitre(t *testing.T) {
	familles := goldenCatalog(t).EquipmentFamilies
	if len(familles) == 0 {
		t.Skip("catalogue d'identite `eqip` vide : le manifeste du titre n'a pas ete lu")
	}
	rule := powerupPadRule(familles)
	for id, veut := range map[uint32]string{
		puTestOvershield: "powerup_overshield",
		puTestCamo:       "powerup_camo",
	} {
		got, ok := rule.Family(id)
		if !ok || got != veut {
			t.Errorf("%#08x : la regle rend (%q, %t), attendu (%q, true) — le manifeste a-t-il"+
				" change d'identifiant ?", id, got, ok, veut)
		}
	}
	if _, ok := rule.Family(puTestMur); ok {
		t.Errorf("%#08x (%q) est accepte comme power-up", puTestMur, familles[puTestMur])
	}
	// LA REGLE EST UN PREFIXE, PAS UNE LISTE DE DEUX : tout power-up que le manifeste ajoutera
	// doit passer sans qu'on revienne ici. Ce controle le verifie sur le catalogue reel.
	var vus []string
	for id, f := range familles {
		if !strings.HasPrefix(f, padPowerupPrefix) {
			continue
		}
		if _, ok := rule.Family(id); !ok {
			t.Errorf("%#08x (%q) est un power-up du manifeste que la regle REFUSE", id, f)
		}
		vus = append(vus, f)
	}
	sort.Strings(vus)
	if len(vus) < 2 {
		t.Errorf("le manifeste ne nomme que %d power-up(s) (%v) — les deux du jeu sont attendus",
			len(vus), vus)
	}
}
