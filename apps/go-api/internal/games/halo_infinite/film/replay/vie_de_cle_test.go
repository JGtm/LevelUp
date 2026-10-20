package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// vie_de_cle_test.go — LA VIE D'UNE CLE D'OBJET DU MONDE FINIT A LA VIE SUIVANTE DE SON SLOT (RB2-3,
// lot J5.4 du plan de suite de l'audit du decodeur, 2026-09-27).
//
// Un slot d'objet ne porte qu'une entite a la fois : la creation PROUVEE suivante sur ce slot —
// quelle que soit sa generation, et qu'elle soit publiee ou non — prouve que l'objet precedent n'y
// est plus. Avant ce lot, la vie d'une cle n'etait bornee que par la reprise de la MEME cle parmi
// les seules creations PUBLIEES : un objet d'objectif ecarte ou un successeur d'une autre
// generation laissait le recensement prolonger l'objet precedent. Une creation que ni la regle
// d'identite ni le recensement ne confirme n'est pas prouvee, et ne borne rien.

// TestVieDUneCle_BorneeParLaVieSuivanteDuSlot : trois figures, une par chaine.
//
// ROUGE AVANT : borne haute 40 s (au lieu de 15 s) derriere un objet d'objectif, 20 s (au lieu de
// 15 s) derriere une autre generation, et une pose affichee jusqu'a la frame 80 (au lieu de 50).
//
// MUTATION : borner par la seule reprise de la meme cle retenue -> rouge.
func TestVieDUneCle_BorneeParLaVieSuivanteDuSlot(t *testing.T) {
	fam := gwTestFamily(t, 0)
	kf := []uint64{0, 10_000_000, 20_000_000, 40_000_000}
	pos := []grammar.BipedPosition{{Slot: 1, TimestampUS: 45_000_000, X: 100, Y: 100, HasWorld: true}}
	const drapeau = 0x0F1A6000
	objectifs := map[uint32]Label{drapeau: {En: "Flag", Fr: "Drapeau"}}
	t.Run("objet d'objectif ecarte sur la meme cle", func(t *testing.T) {
		scan := WorldObjectScan{Scanned: true, Stats: types.EquipmentCreationStats{Accepted: 2},
			Creations: []types.EquipmentCreation{
				gwTestCreation(40, 1, 5_000_000, fam, 0, 0),
				gwTestCreation(40, 1, 15_000_000, drapeau, 0, 0), // objet d'objectif : ecarte, reel
			},
			Keyframes: grammar.WorldObjectKeyframes{TimesUS: kf,
				SeenUS: map[types.LifeKey][]uint64{{Slot: 40, Gen: 1}: {10_000_000, 20_000_000}}},
		}
		objs, _ := padObjects(scan, weaponPadRule(objectifs), nil, pos)
		if len(objs) != 1 || objs[0].Bounds.HighUS != 15_000_000 {
			t.Fatalf("objets %+v : attendu un objet borne a 15 s (la creation suivante du slot)", objs)
		}
	})
	t.Run("successeur d'une autre generation", func(t *testing.T) {
		scan := WorldObjectScan{Scanned: true, Stats: types.EquipmentCreationStats{Accepted: 2},
			Creations: []types.EquipmentCreation{
				gwTestCreation(41, 1, 5_000_000, fam, 0, 0),
				gwTestCreation(41, 2, 15_000_000, fam, 0, 0),
			},
			Keyframes: grammar.WorldObjectKeyframes{TimesUS: kf,
				SeenUS: map[types.LifeKey][]uint64{{Slot: 41, Gen: 1}: {10_000_000}}},
		}
		objs, _ := padObjects(scan, weaponPadRule(nil), nil, pos)
		for _, o := range objs {
			if o.Key.Gen == 1 && o.Bounds.HighUS != 15_000_000 {
				t.Fatalf("objet (41, 1) borne a %d, attendu 15 s (la generation 2 prend le slot)",
					o.Bounds.HighUS)
			}
		}
	})
	t.Run("successeur non prouve", func(t *testing.T) {
		// Une creation d'identite inconnue que le recensement ne confirme pas n'est pas un objet :
		// elle ne borne rien (mesure : 29 contradictions du recensement sur 8 mini-bobines).
		scan := WorldObjectScan{Scanned: true, Stats: types.EquipmentCreationStats{Accepted: 2},
			Creations: []types.EquipmentCreation{
				gwTestCreation(42, 1, 5_000_000, fam, 0, 0),
				gwTestCreation(42, 2, 15_000_000, 0xDEADBEEF, 0, 0),
			},
			Keyframes: grammar.WorldObjectKeyframes{TimesUS: kf,
				SeenUS: map[types.LifeKey][]uint64{{Slot: 42, Gen: 1}: {10_000_000, 20_000_000}}},
		}
		objs, _ := padObjects(scan, weaponPadRule(nil), nil, pos)
		if len(objs) != 1 || objs[0].Bounds.HighUS != 40_000_000 {
			t.Fatalf("objets %+v : attendu un objet borne a 40 s (successeur non prouve ignore)", objs)
		}
	})
	t.Run("reprise non prouvee de la meme cle", func(t *testing.T) {
		// Le recensement d'une cle REPRISE ne dit pas lequel des deux objets il voit : il ne
		// prouve donc pas le successeur.
		scan := WorldObjectScan{Scanned: true, Stats: types.EquipmentCreationStats{Accepted: 2},
			Creations: []types.EquipmentCreation{
				gwTestCreation(44, 1, 5_000_000, fam, 0, 0),
				gwTestCreation(44, 1, 15_000_000, 0xDEADBEEF, 0, 0),
			},
			Keyframes: grammar.WorldObjectKeyframes{TimesUS: kf,
				SeenUS: map[types.LifeKey][]uint64{{Slot: 44, Gen: 1}: {10_000_000, 20_000_000}}},
		}
		objs, _ := padObjects(scan, weaponPadRule(nil), nil, pos)
		if len(objs) != 1 || objs[0].Bounds.HighUS != 40_000_000 {
			t.Fatalf("objets %+v : attendu un objet borne a 40 s (reprise non prouvee ignoree)", objs)
		}
	})
	t.Run("successeur confirme par le recensement", func(t *testing.T) {
		scan := WorldObjectScan{Scanned: true, Stats: types.EquipmentCreationStats{Accepted: 2},
			Creations: []types.EquipmentCreation{
				gwTestCreation(43, 1, 5_000_000, fam, 0, 0),
				gwTestCreation(43, 2, 15_000_000, 0xDEADBEEF, 0, 0),
			},
			Keyframes: grammar.WorldObjectKeyframes{TimesUS: kf, SeenUS: map[types.LifeKey][]uint64{
				{Slot: 43, Gen: 1}: {10_000_000}, {Slot: 43, Gen: 2}: {20_000_000}}},
		}
		objs, _ := padObjects(scan, weaponPadRule(nil), nil, pos)
		if len(objs) != 1 || objs[0].Bounds.HighUS != 15_000_000 {
			t.Fatalf("objets %+v : attendu un objet borne a 15 s (successeur recense)", objs)
		}
	})
	t.Run("poses d'equipement", func(t *testing.T) {
		raw := []types.EquipmentPlacement{
			pePose(types.LifeKey{Slot: 900, Gen: 1}, 2_000_000),
			pePose(types.LifeKey{Slot: 900, Gen: 2}, 6_000_000),
		}
		census := grammar.WorldObjectKeyframes{
			TimesUS: []uint64{5_000_000, 9_000_000, 13_000_000},
			SeenUS:  map[types.LifeKey][]uint64{{Slot: 900, Gen: 1}: {5_000_000}},
		}
		ends := placementEnds(raw, census, peClock())
		if ends[0].untilMax != 50 {
			t.Fatalf("pose (900, 1) : borne haute frame %d, attendu 50 (6 s, la pose suivante du slot)",
				ends[0].untilMax)
		}
	})
}
