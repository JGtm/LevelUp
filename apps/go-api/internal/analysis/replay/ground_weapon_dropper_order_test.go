package replay

// ground_weapon_dropper_order_test.go — LE LACHEUR PUBLIE EST DETERMINISTE.
//
// Meme famille de defaut, meme forme de preuve que `filmdec/projectile_track_order_test.go`
// (item 0.4bis de PLAN_CUISSON_PERF, 2026-09-02) : `gwPadsClass` rend le PREMIER slot dont une
// vie s'acheve dans la fenetre ET dans le rayon. En iterant la map `lives` directement,
// « le premier » dependait de l'ordre d'iteration — aleatoire a chaque execution. Mesure
// d'origine sur `000d5950` : quatre armes au sol publiaient un `dropper` different d'une cuisson
// a l'autre (553 ou 556, 599 ou 602), pour le meme film et le meme code.
//
// La preuve : plusieurs slots EX AEQUO (tous dans la fenetre et dans le rayon), la meme entree
// rejouee, une seule sortie — le plus petit slot.

import "testing"

// apparitionTemoin : une apparition d'arme au sol a un instant et une position connus.
func apparitionTemoin() gwPadApparition {
	return gwPadApparition{Kind: "weapon", Family: "sniper", X: 10, Y: 20, Z: 30, TUS: 1_000_000}
}

// vieQuiSAcheveIci : une vie de bipede qui s'arrete A la position de l'apparition, decalee de
// `decalage` metres sur X et de `gapUS` microsecondes.
func vieQuiSAcheveIci(decalage float32, gapUS uint64) equipLife {
	a := apparitionTemoin()
	return equipLife{from: a.TUS - 5_000_000, to: a.TUS + gapUS, x: a.X + decalage, y: a.Y, z: a.Z}
}

func TestGwPadsClassRendToujoursLeMemeLacheurEntreExAequo(t *testing.T) {
	a := apparitionTemoin()
	// Trois slots dont une vie s'acheve AU MEME instant et A LA MEME distance : la regle ne les
	// separe pas, et c'est exactement le cas qui tirait au sort.
	slots := []uint32{556, 553, 601}
	for tour := 0; tour < 50; tour++ {
		lives := map[uint32][]equipLife{}
		// L'ordre d'INSERTION change de tour en tour ; celui d'ITERATION change de lui-meme.
		for i := range slots {
			s := slots[(i+tour)%len(slots)]
			lives[s] = []equipLife{vieQuiSAcheveIci(0.5, 0)}
		}
		classe, dropper := gwPadsClass(lives, a)
		if classe != gwClassDropped {
			t.Fatalf("tour %d : classe attendue %q, obtenue %q", tour, gwClassDropped, classe)
		}
		if dropper != 553 {
			t.Fatalf("tour %d : lacheur attendu 553 (le plus petit slot ex aequo), obtenu %d",
				tour, dropper)
		}
	}
}

// TestGwPadsClassNePrendPasLePlusPetitSlotHorsRegle : le tri departage les EX AEQUO, il ne
// contourne pas la regle — un slot plus petit hors fenetre ou hors rayon ne gagne pas.
func TestGwPadsClassNePrendPasLePlusPetitSlotHorsRegle(t *testing.T) {
	a := apparitionTemoin()
	cas := []struct {
		nom     string
		ecarte  equipLife
		attendu int
		classe  string
	}{
		{
			nom:     "le plus petit slot est HORS FENETRE",
			ecarte:  vieQuiSAcheveIci(0.5, originDropWindowUS+1),
			attendu: 600,
			classe:  gwClassDropped,
		},
		{
			nom:     "le plus petit slot est HORS RAYON",
			ecarte:  vieQuiSAcheveIci(originDropMaxDist+1, 0),
			attendu: 600,
			classe:  gwClassDropped,
		},
	}
	for _, c := range cas {
		lives := map[uint32][]equipLife{
			100: {c.ecarte},
			600: {vieQuiSAcheveIci(0.5, 0)},
		}
		classe, dropper := gwPadsClass(lives, a)
		if classe != c.classe || dropper != c.attendu {
			t.Errorf("%s : attendu (%s, %d), obtenu (%s, %d)", c.nom, c.classe, c.attendu, classe, dropper)
		}
	}
}

// TestGwPadsClassSansVieProcheEstSpawned : aucune vie dans la fenetre et le rayon — l'arme est
// apparue a son socle, et il n'y a pas de lacheur a nommer.
func TestGwPadsClassSansVieProcheEstSpawned(t *testing.T) {
	lives := map[uint32][]equipLife{
		100: {vieQuiSAcheveIci(originDropMaxDist+1, 0)},
		600: {vieQuiSAcheveIci(0.5, originDropWindowUS+1)},
	}
	classe, dropper := gwPadsClass(lives, apparitionTemoin())
	if classe != gwClassSpawned || dropper != -1 {
		t.Errorf("attendu (%s, -1), obtenu (%s, %d)", gwClassSpawned, classe, dropper)
	}
	if classe, dropper := gwPadsClass(nil, apparitionTemoin()); classe != gwClassSpawned || dropper != -1 {
		t.Errorf("sans aucune vie : attendu (%s, -1), obtenu (%s, %d)", gwClassSpawned, classe, dropper)
	}
}
