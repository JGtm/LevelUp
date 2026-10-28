package replay

// equipment_placement_life_end_test.go — LA FIN LUE D UNE POSE (plan des durees d effets, B.1) : le lien par la vie,
// ses bornes, la cle recyclee, l ambiguite, la fin non lue.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

func poseBrute(slot, gen uint32, t0 uint64) types.EquipmentPlacement {
	return types.EquipmentPlacement{Life: types.LifeKey{Slot: slot, Gen: gen}, T0US: t0}
}

func nomme(ts uint64, slots ...uint32) types.CarriedEquipmentRead {
	l := types.CarriedEquipmentRead{TimestampUS: ts, Object: types.LifeKey{Slot: 2039, Gen: 1}, Carrier: 539,
		Kind: types.CarriedDelta, HasHandles: true}
	for _, s := range slots {
		l.Tracked = append(l.Tracked, types.LifeKey{Slot: s, Gen: 1})
	}
	return l
}

func supprime(ts uint64, slot uint32) types.CarriedEquipmentRead {
	return types.CarriedEquipmentRead{TimestampUS: ts, Object: types.LifeKey{Slot: 2039, Gen: 1}, Carrier: 539,
		Kind: types.CarriedTrackedEnd, Tracked: []types.LifeKey{{Slot: slot, Gen: 1}}}
}

// TestFinDePoseParLaVie : le panneau 1836 est nomme 33 ms apres sa creation, et sa suppression date la fin.
func TestFinDePoseParLaVie(t *testing.T) {
	raw := []types.EquipmentPlacement{poseBrute(1836, 1, 55_871_310)}
	f := finsLuesDesPoses(raw, []types.CarriedEquipmentRead{nomme(55_906_548, 1836), supprime(69_617_029, 1836)})
	if f[0].etat != liaisonUnique || f[0].ts != 69_617_029 {
		t.Fatalf("lien %+v", f[0])
	}
}

// TestFinDePoseBornesDuNommage : le nommage doit suivre la creation d au plus 200 ms (inclus) ; avant la creation,
// ou 1 us apres la borne, il ne lie rien.
func TestFinDePoseBornesDuNommage(t *testing.T) {
	const t0 = uint64(50_000_000)
	for _, k := range []struct {
		nom    string
		nomme  uint64
		attend int
	}{
		{"au meme instant", t0, liaisonUnique},
		{"a 200 ms", t0 + 200_000, liaisonUnique},
		{"a 200 ms + 1 us", t0 + 200_001, liaisonAucune},
		{"avant la creation", t0 - 1, liaisonAucune},
	} {
		f := finsLuesDesPoses([]types.EquipmentPlacement{poseBrute(1836, 1, t0)},
			[]types.CarriedEquipmentRead{nomme(k.nomme, 1836)})
		if f[0].etat != k.attend {
			t.Fatalf("%s : lien %d, attendu %d", k.nom, f[0].etat, k.attend)
		}
	}
}

// TestFinDePoseCleRecyclee : la meme cle (slot, generation) revient dans le match. Une pose se lie au nommage qui
// suit SA creation, jamais a celui qui suit une autre creation de la cle ; sa fin est la suppression de SA vie.
func TestFinDePoseCleRecyclee(t *testing.T) {
	raw := []types.EquipmentPlacement{poseBrute(1836, 1, 55_871_310), poseBrute(1836, 1, 400_000_000)}
	reads := []types.CarriedEquipmentRead{
		nomme(55_906_548, 1836), supprime(69_617_029, 1836),
		nomme(400_033_000, 1836), supprime(413_000_000, 1836),
	}
	f := finsLuesDesPoses(raw, reads)
	if f[0].etat != liaisonUnique || f[0].ts != 69_617_029 || f[1].etat != liaisonUnique || f[1].ts != 413_000_000 {
		t.Fatalf("liens %+v", f)
	}
}

// TestFinDePoseAmbigue : deux vies de la meme cle nommees dans la fenetre de la meme creation : ambigu, aucune fin.
func TestFinDePoseAmbigue(t *testing.T) {
	reads := []types.CarriedEquipmentRead{nomme(50_010_000, 1836), supprime(50_020_000, 1836), nomme(50_100_000, 1836)}
	f := finsLuesDesPoses([]types.EquipmentPlacement{poseBrute(1836, 1, 50_000_000)}, reads)
	if f[0].etat != liaisonAmbigue || f[0].ts != 0 {
		t.Fatalf("lien %+v", f[0])
	}
}

// TestFinDePoseSansPile : une pose qu aucune pile ne nomme (un lache, un capteur pose) n a pas de fin lue, et sans
// lecture de l equipement porte rien ne se lie.
func TestFinDePoseSansPile(t *testing.T) {
	raw := []types.EquipmentPlacement{poseBrute(1836, 1, 50_000_000)}
	if f := finsLuesDesPoses(raw, []types.CarriedEquipmentRead{nomme(50_010_000, 1999)}); f[0].etat != liaisonAucune {
		t.Fatalf("lien %+v", f[0])
	}
	if f := finsLuesDesPoses(raw, nil); len(f) != 1 || f[0].etat != liaisonAucune {
		t.Fatalf("lien %+v", f)
	}
}

// TestFinDePoseIgnoreLesLecturesDEffet : les candidates a l effet portent une vie suivie dans `Tracked` ; elles ne
// sont pas un nommage et n ouvrent aucune vie.
func TestFinDePoseIgnoreLesLecturesDEffet(t *testing.T) {
	l := nomme(50_010_000, 1836)
	l.Kind = grammar.CarriedEffectBirth
	f := finsLuesDesPoses([]types.EquipmentPlacement{poseBrute(1836, 1, 50_000_000)}, []types.CarriedEquipmentRead{l})
	if f[0].etat != liaisonAucune {
		t.Fatalf("lien %+v", f[0])
	}
}

// TestFinDePosePublier : la frame de fin lue (bornee a l axe), les comptes de la couverture, et l absence de fin
// quand la suppression n est pas lue, que le lien est ambigu ou qu il manque.
func TestFinDePosePublier(t *testing.T) {
	clock := replayClock{origin: 10_000_000, step: 100_000, frames: 1000}
	cov := &EquipmentPlacementCoverage{}
	if p := (finDePose{etat: liaisonUnique, ts: 69_617_029}).publier(cov, clock); p == nil || *p != 596 {
		t.Fatalf("fin %v", p)
	}
	if p := (finDePose{etat: liaisonUnique, ts: 900_000_000}).publier(cov, clock); p == nil || *p != 999 {
		t.Fatalf("fin hors axe %v", p)
	}
	if p := (finDePose{etat: liaisonUnique}).publier(cov, clock); p != nil {
		t.Fatalf("fin non lue %v", p)
	}
	if p := (finDePose{etat: liaisonAmbigue, ts: 69_617_029}).publier(cov, clock); p != nil {
		t.Fatalf("ambigu %v", p)
	}
	if p := (finDePose{}).publier(cov, clock); p != nil {
		t.Fatalf("sans lien %v", p)
	}
	if cov.LifeLinked != 3 || cov.LifeAmbiguous != 1 || cov.LifeEnds != 2 {
		t.Fatalf("couverture %+v", cov)
	}
}
