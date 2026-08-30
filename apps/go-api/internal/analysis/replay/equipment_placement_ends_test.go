package replay

// equipment_placement_ends_test.go — la fin d'affichage observée des poses (schéma 27).
//
// POURQUOI CES TESTS EXISTENT. Le golden d'assemblage ne porte aucun recensement ti=37 dans
// son fixture ; sans eux, le bornage par images-clés et la fermeture d'une vie de clé par la
// pose suivante n'auraient aucune couverture.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// peClock : origine 1 s, pas 100 ms, 200 frames.
func peClock() replayClock {
	return replayClock{origin: 1_000_000, step: 100_000, frames: 200}
}

func pePose(life filmdec.EquipmentLifeKey, t0US uint64) filmdec.EquipmentPlacement {
	return filmdec.EquipmentPlacement{Life: life, T0US: t0US, T1US: t0US}
}

func TestPlacementEndsDisparitionBornee(t *testing.T) {
	life := filmdec.EquipmentLifeKey{Slot: 900, Gen: 1}
	raw := []filmdec.EquipmentPlacement{pePose(life, 2_000_000)}
	census := filmdec.WorldObjectKeyframes{
		TimesUS: []uint64{5_000_000, 9_000_000, 13_000_000},
		SeenUS:  map[filmdec.EquipmentLifeKey][]uint64{life: {5_000_000}},
	}
	ends := placementEnds(raw, census, peClock())
	if ends[0].end != GroundWeaponEndSeen {
		t.Fatalf("end = %q, attendu %q : vue a 5 s, absente a 9 s — la disparition est bornee",
			ends[0].end, GroundWeaponEndSeen)
	}
	if ends[0].until != 40 || ends[0].untilMax != 80 {
		t.Errorf("bornes = [%d, %d], attendu [40, 80] : derniere preuve de presence (5 s), "+
			"premiere preuve d absence (9 s)", ends[0].until, ends[0].untilMax)
	}
}

func TestPlacementEndsEncoreRecenseeALaFin(t *testing.T) {
	life := filmdec.EquipmentLifeKey{Slot: 900, Gen: 1}
	raw := []filmdec.EquipmentPlacement{pePose(life, 2_000_000)}
	census := filmdec.WorldObjectKeyframes{
		TimesUS: []uint64{5_000_000, 9_000_000},
		SeenUS:  map[filmdec.EquipmentLifeKey][]uint64{life: {5_000_000, 9_000_000}},
	}
	ends := placementEnds(raw, census, peClock())
	if ends[0].end != GroundWeaponEndOpen || ends[0].until != 199 {
		t.Fatalf("ends = %+v : recensee a la DERNIERE image-cle, rien ne prouve la "+
			"disparition — la pose reste affichee jusqu au bout", ends[0])
	}
}

func TestPlacementEndsLaPoseSuivanteFermeLaVie(t *testing.T) {
	// Le pool de cles reboucle : deux poses successives de la MEME cle sont deux objets. Le
	// recensement posterieur a la seconde ne doit pas prolonger la premiere.
	life := filmdec.EquipmentLifeKey{Slot: 900, Gen: 1}
	raw := []filmdec.EquipmentPlacement{
		pePose(life, 2_000_000),
		pePose(life, 10_000_000),
	}
	census := filmdec.WorldObjectKeyframes{
		TimesUS: []uint64{5_000_000, 12_000_000, 16_000_000},
		// 12 s et 16 s recensent la SECONDE vie ; la premiere n est vue qu a 5 s.
		SeenUS: map[filmdec.EquipmentLifeKey][]uint64{life: {5_000_000, 12_000_000, 16_000_000}},
	}
	ends := placementEnds(raw, census, peClock())
	if ends[0].end != GroundWeaponEndSeen || ends[0].until != 40 {
		t.Fatalf("premiere pose : %+v — le recensement de la vie SUIVANTE ne doit pas la "+
			"prolonger", ends[0])
	}
	if ends[1].end != GroundWeaponEndOpen {
		t.Errorf("seconde pose : %+v — recensee a la derniere image-cle, elle reste ouverte",
			ends[1])
	}
}

func TestPlacementEndsSansRecensement(t *testing.T) {
	// Sans recensement du tout (census vide — le cas des tests d assemblage sur positions
	// figees), rien ne prouve aucune disparition : tout sort `open`, jamais une borne inventee.
	life := filmdec.EquipmentLifeKey{Slot: 900, Gen: 1}
	raw := []filmdec.EquipmentPlacement{pePose(life, 2_000_000)}
	ends := placementEnds(raw, filmdec.WorldObjectKeyframes{}, peClock())
	if ends[0].end != GroundWeaponEndOpen {
		t.Fatalf("end = %q, attendu %q : sans image-cle, aucune absence n est prouvable",
			ends[0].end, GroundWeaponEndOpen)
	}
}

func TestBuildEquipmentPlacementsPublieLesFins(t *testing.T) {
	// Le bout-en-bout du tally : la pose publiee porte until/untilMax/end, et la couverture
	// les compte — sans quoi « 229 poses » se lirait comme « 229 fins mesurees ».
	life := filmdec.EquipmentLifeKey{Slot: 900, Gen: 1}
	raw := []filmdec.EquipmentPlacement{pePose(life, 2_000_000)}
	census := filmdec.WorldObjectKeyframes{
		TimesUS: []uint64{5_000_000, 9_000_000, 13_000_000},
		SeenUS:  map[filmdec.EquipmentLifeKey][]uint64{life: {5_000_000}},
	}
	st := filmdec.EquipmentPlacementStats{Scanned: true}
	out, cov := buildEquipmentPlacements(raw, st, nil, peClock(), census)
	if len(out) != 1 {
		t.Fatalf("poses publiees = %d, attendu 1", len(out))
	}
	p := out[0]
	if p.End != GroundWeaponEndSeen || p.Until != 40 || p.UntilMax != 80 {
		t.Fatalf("fin publiee = end=%q until=%d untilMax=%d, attendu seen/40/80", p.End, p.Until, p.UntilMax)
	}
	if cov.EndSeen != 1 || cov.EndOpen != 0 {
		t.Errorf("couverture = endSeen=%d endOpen=%d, attendu 1/0", cov.EndSeen, cov.EndOpen)
	}
}
