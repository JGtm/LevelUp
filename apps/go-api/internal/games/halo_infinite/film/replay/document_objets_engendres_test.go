package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// piste construit la trajectoire d'une vie (slot, génération 1) : un point par entrée, en
// microsecondes du film.
func piste(slot uint32, pts ...types.ProjectileSample) types.ProjectileTrack {
	return types.ProjectileTrack{Slot: slot, Gen: 1, Pts: pts}
}

func echantillon(us uint64, x, y, z float32, auRepos bool) types.ProjectileSample {
	return types.ProjectileSample{TimestampUS: us, X: x, Y: y, Z: z, AtRest: auRepos}
}

// champsDeTest : le champ de réparation de TestChampDeReparationFinLue (5676a9ba, porteur 539), dont
// la pile i28 nomme l'objet engendré 1123 puis 1138.
func champsDeTest(projectiles []types.ProjectileTrack) entreeDEquipementPorte {
	reads := []types.CarriedEquipmentRead{
		lecture(90183.0, 2039, 539, types.CarriedNew),
		avecSuivis(lecture(118775.4, 2039, 539, types.CarriedDelta), 1123),
		avecSuivis(lecture(134024.8, 2039, 539, types.CarriedDelta), 1138),
		lecture(134997.8, 2039, 539, types.CarriedDel),
	}
	return entreeDEquipementPorte{reads: reads, palette: paletteDuTest(),
		changes: []types.EquipmentChange{{TimestampUS: 90183000, Slot: 539, Kind: types.EquipmentTaken, Rank: 23}},
		couts:   coutsParFamille(catalogueDuTest(), famillesDuTest), projectiles: projectiles,
		tracks: []Track{{Slot: 539}}, origin: origine5676, step: pas100ms}
}

// TestObjetEngendreSitueParSaTrajectoire : l'objet engendré 1123 porte la position de repos de la
// trajectoire de la MÊME vie : l'image et la position de son dernier point publié (un seul point par
// image : le premier), la hauteur de l'échantillon brut correspondant.
func TestObjetEngendreSitueParSaTrajectoire(t *testing.T) {
	entree := champsDeTest([]types.ProjectileTrack{
		piste(1123,
			echantillon(118_700_000, 10.001, 20.001, 5.5, false),
			echantillon(118_760_000, 10.2, 20.2, 5.6, false), // meme image que le premier : ecarte
			echantillon(118_800_000, 11.123, 21.456, 5.25, false),
			echantillon(118_900_000, 12.345, 22.678, 4.987, true)),
		piste(999, // une autre vie : ne situe rien
			echantillon(118_700_000, 1, 1, 1, false), echantillon(118_800_000, 2, 2, 2, false),
			echantillon(118_900_000, 3, 3, 3, true)),
	})
	out, _, cov := buildCarriedEquipment(entree)
	if len(out) != 1 || len(out[0].Tracked) != 2 {
		t.Fatalf("publies %d, couverture %+v", len(out), cov)
	}
	a := out[0].Tracked[0]
	if a.Land == nil || *a.Land != 945 || a.X == nil || a.Y == nil || a.Z == nil ||
		*a.X != 12.35 || *a.Y != 22.68 || *a.Z != 4.99 {
		t.Fatalf("objet 1123 : %+v", a)
	}
	b := out[0].Tracked[1]
	if b.Land != nil || b.X != nil || b.Y != nil || b.Z != nil {
		t.Fatalf("objet 1138 sans trajectoire : aucune position ne se devine, %+v", b)
	}
	if cov.TrackedObjects != 2 || cov.TrackedPositioned != 1 || cov.TrackedUnpositioned != 1 {
		t.Fatalf("couverture %+v", cov)
	}
}

// TestObjetEngendreVolCoupe : un vol coupé à un pas impossible se publie jusqu'à la coupure ; l'objet
// se situe sur le dernier point publié, jamais sur l'échantillon qui suit.
func TestObjetEngendreVolCoupe(t *testing.T) {
	entree := champsDeTest([]types.ProjectileTrack{
		piste(1123,
			echantillon(118_700_000, 10, 20, 5, false),
			echantillon(118_800_000, 11, 21, 5.1, false),
			echantillon(118_900_000, 95, 21, 5.2, false), // 84 m en 100 ms : coupe
			echantillon(119_000_000, 96, 21, 5.3, true)),
	})
	out, _, _ := buildCarriedEquipment(entree)
	a := out[0].Tracked[0]
	if a.Land == nil || *a.Land != 944 || *a.X != 11 || *a.Y != 21 || *a.Z != 5.1 {
		t.Fatalf("objet 1123 : %+v", a)
	}
}

// TestObjetEngendreTrajectoireNonPubliee : une trajectoire que le rejeu ne publie pas (moins de deux
// points de grille) ne situe rien.
func TestObjetEngendreTrajectoireNonPubliee(t *testing.T) {
	entree := champsDeTest([]types.ProjectileTrack{
		piste(1123,
			echantillon(118_700_000, 10, 20, 5, false),
			echantillon(118_710_000, 10, 20, 5, false),
			echantillon(118_720_000, 10, 20, 5, true)),
	})
	out, _, cov := buildCarriedEquipment(entree)
	if a := out[0].Tracked[0]; a.Land != nil || a.X != nil || cov.TrackedPositioned != 0 || cov.TrackedUnpositioned != 2 {
		t.Fatalf("objet 1123 : %+v, couverture %+v", a, cov)
	}
}

// TestObjetEngendreCleRecyclee : la génération d'une vie revient dans le match. L'objet 1123 nommé à la
// frame 943 se situe sur la trajectoire née là, pas sur celle que le même couple (slot, génération)
// porte 500 s plus tard.
func TestObjetEngendreCleRecyclee(t *testing.T) {
	entree := champsDeTest([]types.ProjectileTrack{
		piste(1123,
			echantillon(118_700_000, 10, 20, 5, false),
			echantillon(118_800_000, 11, 21, 5.1, false),
			echantillon(118_900_000, 12, 22, 5.2, true)),
		piste(1123,
			echantillon(617_000_000, 70, 80, 9, false),
			echantillon(617_100_000, 71, 81, 9.1, false),
			echantillon(617_200_000, 72, 82, 9.2, true)),
	})
	out, _, cov := buildCarriedEquipment(entree)
	a := out[0].Tracked[0]
	if a.Land == nil || *a.Land != 945 || *a.X != 12 || *a.Y != 22 || *a.Z != 5.2 {
		t.Fatalf("objet 1123 : %+v", a)
	}
	// Seule la trajectoire recyclee existe pour la vie 1138 : elle naît loin de la pile, rien ne s'y joint.
	entree = champsDeTest([]types.ProjectileTrack{
		piste(1138,
			echantillon(617_000_000, 70, 80, 9, false),
			echantillon(617_100_000, 71, 81, 9.1, false),
			echantillon(617_200_000, 72, 82, 9.2, true)),
	})
	out, _, cov = buildCarriedEquipment(entree)
	if b := out[0].Tracked[1]; b.Land != nil || cov.TrackedPositioned != 0 || cov.TrackedUnpositioned != 2 {
		t.Fatalf("objet 1138 : %+v, couverture %+v", b, cov)
	}
}

// cand construit une trajectoire de repos née à la frame `n`, reconnaissable par son `land`.
func cand(n, land int) reposEngendre { return reposEngendre{naissance: n, land: land} }

// TestReposNeAPeuPresBornes : la borne est INCLUSE à toleranceDeNaissance frames et refuse la suivante,
// avant comme après la frame de l'objet.
func TestReposNeAPeuPresBornes(t *testing.T) {
	const t0 = 1000
	cas := []struct {
		nom       string
		naissance int
		attendu   bool
	}{
		{"ecart 0", t0, true},
		{"ecart +borne", t0 + toleranceDeNaissance, true},
		{"ecart -borne", t0 - toleranceDeNaissance, true},
		{"ecart +borne+1", t0 + toleranceDeNaissance + 1, false},
		{"ecart -borne-1", t0 - toleranceDeNaissance - 1, false},
	}
	for _, c := range cas {
		r, ok := reposNeAPeuPres([]reposEngendre{cand(c.naissance, 7)}, t0)
		if ok != c.attendu || (ok && r.land != 7) {
			t.Errorf("%s : situe=%v, attendu %v (%+v)", c.nom, ok, c.attendu, r)
		}
	}
	if toleranceDeNaissance != 3 {
		t.Errorf("la tolerance vaut %d : les bornes ci-dessus sont ecrites pour 3 (4 refuse, 3 accepte)",
			toleranceDeNaissance)
	}
	if _, ok := reposNeAPeuPres(nil, t0); ok {
		t.Error("aucune candidate : rien ne se situe")
	}
}

// TestReposNeAPeuPresLaPlusProche : parmi plusieurs candidates dans la tolerance, la plus proche gagne,
// quel que soit son rang ; a egalite d'ecart (une avant, une apres), la premiere de la liste.
func TestReposNeAPeuPresLaPlusProche(t *testing.T) {
	const t0 = 1000
	// La plus proche est au milieu : ni la premiere ni la derniere.
	r, ok := reposNeAPeuPres([]reposEngendre{cand(t0-3, 1), cand(t0+1, 2), cand(t0+3, 3)}, t0)
	if !ok || r.land != 2 {
		t.Fatalf("la plus proche (ecart 1) attendue, obtenu %+v", r)
	}
	// La plus proche est la premiere.
	r, _ = reposNeAPeuPres([]reposEngendre{cand(t0, 1), cand(t0+2, 2)}, t0)
	if r.land != 1 {
		t.Fatalf("la plus proche (ecart 0, premiere) attendue, obtenu %+v", r)
	}
	// Egalite : ecart 2 avant puis ecart 2 apres, et l'inverse.
	r, _ = reposNeAPeuPres([]reposEngendre{cand(t0-2, 1), cand(t0+2, 2)}, t0)
	if r.land != 1 {
		t.Fatalf("egalite : la premiere attendue, obtenu %+v", r)
	}
	r, _ = reposNeAPeuPres([]reposEngendre{cand(t0+2, 2), cand(t0-2, 1)}, t0)
	if r.land != 2 {
		t.Fatalf("egalite : la premiere attendue, obtenu %+v", r)
	}
	// Une candidate hors tolerance ne l'emporte pas, meme seule apres une proche.
	r, ok = reposNeAPeuPres([]reposEngendre{cand(t0+3, 1), cand(t0+4, 2)}, t0)
	if !ok || r.land != 1 {
		t.Fatalf("seule la candidate a 3 frames est eligible, obtenu %+v", r)
	}
}
