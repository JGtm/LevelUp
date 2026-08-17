package replay

// ground_weapon_pads_test.go — LES TESTS DE LA CHAINE DES SOCLES, sur des entrees SYNTHETIQUES.
//
// POURQUOI SANS FILM. Les verdicts du chantier (combien de socles, quel cycle, quel accord)
// vivent dans les instruments sous garde, qui lisent de vrais films et publient leurs
// denominateurs au plan. Ce fichier-ci verrouille autre chose, et c'est ce que le gate ordinaire
// peut tenir : que la chaine RETIENNE ce qu'elle doit retenir et ECARTE ce qu'elle doit ecarter.
//
// LES QUATRE REFUS SONT TESTES, parce qu'ils sont la moitie du resultat : pas de socle pour une
// arme lachee a une mort, pas de socle pour un objet qui a bouge, pas de cycle instable publie,
// pas de ramasseur. Un test qui ne verifierait que les publications laisserait revenir chacun.

import (
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

// gwTestClock est l'axe de temps des tests : 100 ms de pas, 801 frames (80 s de film).
func gwTestClock() replayClock {
	return replayClock{origin: 0, step: 100_000, frames: 801}
}

// gwTestFamily rend une famille d'arme du catalogue de production, choisie par un ordre TOTAL
// pour que le test ne depende pas de l'ordre d'iteration d'une map.
func gwTestFamily(t *testing.T, rank int) uint32 {
	t.Helper()
	fams := make([]uint32, 0, len(weaponv3.KnownWeaponHigh32))
	for f := range weaponv3.KnownWeaponHigh32 {
		fams = append(fams, f)
	}
	sort.Slice(fams, func(i, j int) bool { return fams[i] < fams[j] })
	if rank >= len(fams) {
		t.Fatalf("catalogue d'armes trop court : %d familles", len(fams))
	}
	return fams[rank]
}

// gwTestCreation fabrique un record de creation porteur d'une identite d'arme.
func gwTestCreation(slot, gen uint32, atUS uint64, fam uint32, x, y float32) filmdec.EquipmentCreation {
	c := filmdec.EquipmentCreation{Slot: slot, Gen: gen, TimestampUS: atUS, X: x, Y: y}
	c.MPPPresent[filmdec.MPPWord32] = true
	c.MPPVal[filmdec.MPPWord32] = uint64(fam)
	return c
}

// gwTestPositions fabrique un nuage de bipedes : un joueur LOIN du socle sur toute la duree
// (sa vie ne s'acheve donc jamais pres de lui), plus un passage a la position `px, py` a chacun
// des instants demandes.
//
// LE NUAGE DEBORDE LA DERNIERE IMAGE-CLE, et ce n'est pas un detail : la fin du film est le
// dernier instant qu'une des trois sources porte, et le recensement d'une vie est restreint a
// [creation, fin). Un nuage qui s'arreterait PILE a la derniere image-cle exclurait celle-ci du
// recensement, et l'objet encore present a la fin ne se verrait plus comme tel.
func gwTestPositions(passes []uint64, px, py float32) []filmdec.BipedPosition {
	var out []filmdec.BipedPosition
	for t := uint64(0); t <= 85_000_000; t += 1_000_000 {
		out = append(out, filmdec.BipedPosition{Slot: 1, TimestampUS: t, X: 100, Y: 100, HasWorld: true})
	}
	for _, t := range passes {
		out = append(out, filmdec.BipedPosition{Slot: 2, TimestampUS: t, X: px, Y: py, HasWorld: true})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TimestampUS < out[j].TimestampUS })
	return out
}

// gwTestPadScan monte le cas de reference : QUATRE apparitions au meme endroit, recensees une
// image-cle chacune, dont trois suivies d'un passage de joueur. La quatrieme est encore recensee
// a la derniere image-cle : le socle ne s'est pas vide.
func gwTestPadScan(t *testing.T) (GroundWeaponScan, []filmdec.BipedPosition) {
	t.Helper()
	fam := gwTestFamily(t, 0)
	kf := []uint64{0, 20_000_000, 40_000_000, 60_000_000, 80_000_000}
	scan := GroundWeaponScan{
		Scanned: true,
		Stats:   filmdec.EquipmentCreationStats{Slots: 8, Anchors: 40, Accepted: 4},
		Creations: []filmdec.EquipmentCreation{
			gwTestCreation(10, 0, 1_000_000, fam, 10, 10),
			gwTestCreation(11, 0, 31_000_000, fam, 10.2, 10.1),
			gwTestCreation(12, 0, 51_000_000, fam, 9.9, 10.2),
			gwTestCreation(13, 0, 71_000_000, fam, 10.1, 9.8),
		},
		Keyframes: filmdec.GroundWeaponKeyframes{
			TimesUS: kf,
			SeenUS: map[filmdec.EquipmentLifeKey][]uint64{
				{Slot: 10}: {20_000_000},
				{Slot: 11}: {40_000_000},
				{Slot: 12}: {60_000_000},
				{Slot: 13}: {80_000_000},
			},
		},
	}
	return scan, gwTestPositions([]uint64{25_000_000, 45_000_000, 65_000_000}, 10, 10)
}

// TestBuildWeaponPadsPublieUnSocleEtSesOccupations : quatre apparitions au meme endroit font UN
// socle ; chaque occupation porte ses trois instants ; seules celles qui s'achevent donnent une
// occupation achevee.
func TestBuildWeaponPadsPublieUnSocleEtSesOccupations(t *testing.T) {
	scan, pos := gwTestPadScan(t)
	pads, picks, cov := buildWeaponPads(scan, pos, gwTestClock())
	if len(pads) != 1 {
		t.Fatalf("1 socle attendu, %d obtenus : %+v", len(pads), pads)
	}
	p := pads[0]
	if got := len(p.Spawns); got != 4 {
		t.Fatalf("4 apparitions attendues sur le socle, %d : %+v", got, p.Spawns)
	}
	if p.Weapon != formatWeaponFamily(gwTestFamily(t, 0)) {
		t.Fatalf("famille publiee %q, attendu %q", p.Weapon, formatWeaponFamily(gwTestFamily(t, 0)))
	}
	// Premiere occupation : apparue a 1,0 s (frame 10), prouvee presente a 20 s (frame 200),
	// prouvee absente a 40 s (frame 400).
	if got := p.Presence[0]; got.T0 != 10 || got.TLow != 200 || got.THigh != 400 {
		t.Fatalf("presence [10,200,400] attendue, obtenue %+v", got)
	}
	// Derniere occupation : encore recensee a la DERNIERE image-cle — le socle ne s'est pas
	// vide, donc aucune occupation achevee pour elle.
	if len(picks) != 3 {
		t.Fatalf("3 occupations achevees attendues (la 4e est encore recensee), %d : %+v",
			len(picks), picks)
	}
	for i, k := range picks {
		if k.Pad != 0 {
			t.Fatalf("occupation %d rattachee au socle %d, attendu 0", i, k.Pad)
		}
		if k.XUID != nil {
			t.Fatalf("occupation %d publie un ramasseur %v — le gate 2.5 ne l'autorise pas",
				i, *k.XUID)
		}
		if k.TLow >= k.THigh {
			t.Fatalf("occupation %d : bornes [%d,%d] non ordonnees", i, k.TLow, k.THigh)
		}
	}
	if cov.Pads != 1 || cov.Occupancies != 4 || cov.Dated != 3 || cov.Never != 1 || cov.Unknown != 0 {
		t.Fatalf("couverture inattendue : %+v", cov)
	}
	if !cov.Balanced() {
		t.Fatalf("couverture desequilibree : %+v", cov)
	}
}

// TestBuildWeaponPadsPublieLeCycleQuandIlEstEtabli : trois ecarts de 6,0 s exactement — le
// cycle se publie, mesure DEPUIS la disparition.
func TestBuildWeaponPadsPublieLeCycleQuandIlEstEtabli(t *testing.T) {
	scan, pos := gwTestPadScan(t)
	pads, _, cov := buildWeaponPads(scan, pos, gwTestClock())
	c := pads[0].Cycle
	if c == nil {
		t.Fatalf("cycle attendu etabli (trois ecarts identiques) : %+v", pads[0])
	}
	if c.Gaps != 3 || c.MedianS < 5.9 || c.MedianS > 6.1 {
		t.Fatalf("cycle de 6,0 s sur 3 ecarts attendu, obtenu %+v", c)
	}
	if cov.Cycles != 1 {
		t.Fatalf("1 cycle etabli attendu a la couverture, %d", cov.Cycles)
	}
}

// TestBuildWeaponPadsTaitUnCycleInstable : le seuil de la decision 3 ne se contourne pas au
// niveau du document non plus — un socle dont les ecarts varient publie `cycle: null`.
func TestBuildWeaponPadsTaitUnCycleInstable(t *testing.T) {
	scan, _ := gwTestPadScan(t)
	// La deuxieme arme reapparait tout de suite apres la disparition de la premiere, la
	// troisieme longtemps apres : les ecarts passent de 1 s a 16 s.
	scan.Creations[1].TimestampUS = 26_000_000
	pos := gwTestPositions([]uint64{25_000_000, 45_000_000, 65_000_000}, 10, 10)
	pads, _, cov := buildWeaponPads(scan, pos, gwTestClock())
	if len(pads) != 1 {
		t.Fatalf("1 socle attendu, %d", len(pads))
	}
	if pads[0].Cycle != nil {
		t.Fatalf("cycle instable publie : %+v — la decision 3 l'interdit", pads[0].Cycle)
	}
	if cov.Cycles != 0 {
		t.Fatalf("aucun cycle etabli attendu, %d", cov.Cycles)
	}
}

// TestBuildWeaponPadsEcarteLesLachersEtLesObjetsQuiOntBouge : les deux populations que la
// mesure a disqualifiees ne font ni socle ni occupation.
//
//	LACHEE   une vie de bipede s'acheve a moins de 2 frames et 1,5 m de l'apparition : c'est une
//	         arme tombee a une mort, et sa disparition est une despawn (l'oracle y passe SOUS son
//	         propre temoin, 32,1 % contre 65,0 %).
//	QUI A BOUGE  l'objet a une piste de position : il n'est pas apparu au repos. Ce sont presque
//	         toujours des armes de depart lachees en ramassant autre chose.
func TestBuildWeaponPadsEcarteLesLachersEtLesObjetsQuiOntBouge(t *testing.T) {
	fam := gwTestFamily(t, 1)
	kf := []uint64{0, 20_000_000, 40_000_000, 60_000_000, 80_000_000}
	scan := GroundWeaponScan{
		Scanned: true,
		Stats:   filmdec.EquipmentCreationStats{Accepted: 4},
		Creations: []filmdec.EquipmentCreation{
			// Deux apparitions au meme endroit, mais LACHEES : une vie de joueur s'acheve la.
			gwTestCreation(20, 0, 10_000_000, fam, 30, 30),
			gwTestCreation(21, 0, 50_000_000, fam, 30.1, 30),
			// Deux apparitions au meme endroit, mais qui ONT BOUGE (piste delta).
			gwTestCreation(22, 0, 10_000_000, fam, 60, 60),
			gwTestCreation(23, 0, 50_000_000, fam, 60.1, 60),
		},
		Keyframes: filmdec.GroundWeaponKeyframes{TimesUS: kf},
		Tracks: []filmdec.ProjectileTrack{
			{Slot: 22, Pts: []filmdec.ProjectileSample{{TimestampUS: 10_000_000, X: 60, Y: 60}}},
			{Slot: 23, Pts: []filmdec.ProjectileSample{{TimestampUS: 50_000_000, X: 60.1, Y: 60}}},
		},
	}
	// Deux vies de joueur qui S'ACHEVENT sur la premiere position, aux deux instants voulus :
	// des echantillons isoles, separes par plus de `lifeGapUS`.
	pos := []filmdec.BipedPosition{
		{Slot: 3, TimestampUS: 9_950_000, X: 30, Y: 30, HasWorld: true},
		{Slot: 3, TimestampUS: 49_950_000, X: 30.1, Y: 30, HasWorld: true},
	}
	pads, picks, cov := buildWeaponPads(scan, pos, gwTestClock())
	if len(pads) != 0 || len(picks) != 0 {
		t.Fatalf("aucun socle ni occupation attendus, %d socles / %d occupations : %+v",
			len(pads), len(picks), pads)
	}
	if cov.Dropped != 2 {
		t.Fatalf("2 apparitions lachees attendues, %d : %+v", cov.Dropped, cov)
	}
	if cov.Spawned != 2 || cov.AtRest != 0 {
		t.Fatalf("2 apparitions non lachees dont 0 au repos attendues : %+v", cov)
	}
	if !cov.Balanced() {
		t.Fatalf("couverture desequilibree : %+v", cov)
	}
}

// TestGroundWeaponObjectsEcarteLesCreationsSansIdentite : le garde-fou de la decouverte 2 — une
// creation dont le mot MPP ne se resout PAS dans le catalogue d'armes est ecartee et comptee.
// Sans lui, une bande fantome rendait 398 creations acceptees contre 366 pour la vraie bande.
func TestGroundWeaponObjectsEcarteLesCreationsSansIdentite(t *testing.T) {
	fam := gwTestFamily(t, 0)
	sansMPP := gwTestCreation(30, 0, 1_000_000, fam, 0, 0)
	sansMPP.MPPPresent[filmdec.MPPWord32] = false
	scan := GroundWeaponScan{
		Scanned: true,
		Stats:   filmdec.EquipmentCreationStats{Accepted: 3},
		Creations: []filmdec.EquipmentCreation{
			gwTestCreation(31, 0, 2_000_000, fam, 1, 1),
			gwTestCreation(32, 0, 3_000_000, 0xDEADBEEF, 2, 2), // identite hors catalogue
			sansMPP,
		},
		Keyframes: filmdec.GroundWeaponKeyframes{TimesUS: []uint64{0, 20_000_000}},
	}
	objs := groundWeaponObjects(scan, nil, nil)
	if len(objs) != 1 {
		t.Fatalf("1 creation retenue attendue, %d : %+v", len(objs), objs)
	}
	if objs[0].FamilyID != fam {
		t.Fatalf("identite retenue 0x%08x, attendu 0x%08x", objs[0].FamilyID, fam)
	}
	_, _, cov := buildWeaponPads(scan, nil, gwTestClock())
	if cov.Kept != 1 || cov.Rejected != 2 {
		t.Fatalf("1 retenue et 2 ecartees attendues : %+v", cov)
	}
}

// TestGroundWeaponObjectsBorneParLaRepriseDeCle : deux objets successifs sur la MEME cle
// (slot, gen) — la generation ne fait que 2 bits. Sans cette borne, le recensement du second
// prouverait la survie du premier.
func TestGroundWeaponObjectsBorneParLaRepriseDeCle(t *testing.T) {
	fam := gwTestFamily(t, 0)
	scan := GroundWeaponScan{
		Scanned: true,
		Stats:   filmdec.EquipmentCreationStats{Accepted: 2},
		Creations: []filmdec.EquipmentCreation{
			gwTestCreation(40, 1, 5_000_000, fam, 0, 0),
			gwTestCreation(40, 1, 25_000_000, fam, 0, 0),
		},
		Keyframes: filmdec.GroundWeaponKeyframes{
			TimesUS: []uint64{0, 20_000_000, 40_000_000},
			SeenUS:  map[filmdec.EquipmentLifeKey][]uint64{{Slot: 40, Gen: 1}: {20_000_000, 40_000_000}},
		},
	}
	// Le nuage deborde la derniere image-cle : sans cela la fin du film tomberait dessus et
	// l'exclurait du recensement (la vie est restreinte a [creation, fin)).
	pos := []filmdec.BipedPosition{{Slot: 1, TimestampUS: 45_000_000, X: 100, Y: 100, HasWorld: true}}
	objs := groundWeaponObjects(scan, nil, pos)
	if len(objs) != 2 {
		t.Fatalf("2 objets attendus sur la meme cle, %d", len(objs))
	}
	if objs[0].Bounds.HighUS != 25_000_000 {
		t.Fatalf("le premier objet doit etre borne par la REPRISE de cle (25 s), pas par"+
			" l'image-cle suivante : %+v", objs[0].Bounds)
	}
	if objs[0].Status == gwPickupStatusNever {
		t.Fatalf("le premier objet n'est pas recense a la derniere image-cle de SA vie : %+v",
			objs[0])
	}
	if objs[1].Status != gwPickupStatusNever {
		t.Fatalf("le second objet est recense a la derniere image-cle du film : %+v", objs[1])
	}
}

// TestGroundWeaponObjectsHasDeltaEstParVieEtNonParCle : LE CORRECTIF DE LA REVUE DU 2026-08-17.
//
// Une cle (slot, gen) REPRISE porte deux objets successifs ; seul le PREMIER a une piste delta.
// Lire « la cle a une piste » (l'ancienne regle) faisait passer le SECOND pour un objet qui a
// bouge : il sortait du jeu `at_rest`, et la grappe de son socle perdait une apparition. La
// regle est desormais celle de la vie — la MEME fenetre que le recensement.
func TestGroundWeaponObjectsHasDeltaEstParVieEtNonParCle(t *testing.T) {
	fam := gwTestFamily(t, 0)
	scan := GroundWeaponScan{
		Scanned: true,
		Stats:   filmdec.EquipmentCreationStats{Accepted: 2},
		Creations: []filmdec.EquipmentCreation{
			gwTestCreation(50, 2, 5_000_000, fam, 7, 7),
			gwTestCreation(50, 2, 45_000_000, fam, 7, 7),
		},
		Keyframes: filmdec.GroundWeaponKeyframes{TimesUS: []uint64{0, 20_000_000, 40_000_000, 60_000_000}},
		// UNE SEULE piste, sur la PREMIERE vie : elle demarre a l'instant de la premiere
		// creation et s'acheve bien avant la seconde.
		Tracks: []filmdec.ProjectileTrack{{Slot: 50, Gen: 2, Pts: []filmdec.ProjectileSample{
			{TimestampUS: 5_000_000, X: 7, Y: 7},
			{TimestampUS: 9_000_000, X: 9, Y: 9},
		}}},
	}
	pos := []filmdec.BipedPosition{{Slot: 1, TimestampUS: 65_000_000, X: 100, Y: 100, HasWorld: true}}
	objs := groundWeaponObjects(scan, nil, pos)
	if len(objs) != 2 {
		t.Fatalf("2 objets attendus sur la meme cle, %d", len(objs))
	}
	if !objs[0].Appar.HasDelta || !objs[0].Moved {
		t.Fatalf("la PREMIERE vie porte la piste : elle a bouge — %+v", objs[0])
	}
	if objs[1].Appar.HasDelta || objs[1].Moved {
		t.Fatalf("la SECONDE vie n'a AUCUNE piste : elle est apparue au repos et doit rester"+
			" candidate a un socle — %+v", objs[1])
	}
	// La position de reference suit la MEME regle : dernier point de piste pour la premiere,
	// position de creation pour la seconde.
	if objs[0].Pos != [3]float32{9, 9, 0} {
		t.Fatalf("position de reference de la vie mobile : %v, attendu le dernier point (9, 9, 0)",
			objs[0].Pos)
	}
	if objs[1].Pos != [3]float32{7, 7, 0} {
		t.Fatalf("position de reference de la vie au repos : %v, attendu la creation (7, 7, 0)",
			objs[1].Pos)
	}
}

// TestBuildWeaponPadsSansBalayageNePublieRien : un film qu'on n'a pas su balayer publie ZERO
// socle et le DIT (`scanned: false`) — sans quoi il serait indistinguable d'un film sans socle.
func TestBuildWeaponPadsSansBalayageNePublieRien(t *testing.T) {
	pads, picks, cov := buildWeaponPads(GroundWeaponScan{}, nil, gwTestClock())
	if pads != nil || picks != nil {
		t.Fatalf("aucune publication attendue sans balayage : %+v / %+v", pads, picks)
	}
	if cov == nil || cov.Scanned {
		t.Fatalf("couverture attendue presente et `scanned: false` : %+v", cov)
	}
}
