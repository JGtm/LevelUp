package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// equipment_episodes_test.go — l'assembleur d'épisodes d'état actif, sur données
// synthétiques : la MESURE des canaux vit dans les instruments gardés de filmdec
// (i28_camo_test.go, i5_overshield_test.go) ; ici on verrouille la MACHINE À ÉTATS
// (ouverture au passage actif, fermeture mesurée ou à la mort, clamp de fenêtre).

const (
	eqOrigin = uint64(1_000_000)
	eqStep   = uint64(100_000) // 100 ms — la grille par défaut du rejeu
)

// eqTS rend l'horodatage film de la frame f.
func eqTS(f int) uint64 { return eqOrigin + uint64(f)*eqStep }

func camoRead(slot uint32, frame int, q uint16) filmdec.CamoRead {
	return filmdec.CamoRead{Slot: slot, TimestampUS: eqTS(frame), Q: q}
}

func shieldPos(slot uint32, frame int, q uint8) filmdec.BipedPosition {
	p := filmdec.BipedPosition{Slot: slot, TimestampUS: eqTS(frame)}
	p.HasShield = true
	p.Shield.Q = q
	return p
}

func TestCamoEpisodeOuvreEtFermeSurLesTransitionsMesurees(t *testing.T) {
	tracks := []Track{{Slot: 512, StartFrame: 0, EndFrame: 100}}
	camo := []filmdec.CamoRead{
		camoRead(512, 2, filmdec.CamoInactiveQ),
		camoRead(512, 10, filmdec.CamoActiveQ),
		camoRead(512, 20, filmdec.CamoActiveQ), // même état : ne rouvre rien
		camoRead(512, 34, filmdec.CamoInactiveQ),
	}
	eps, nonBinary := buildEquipmentEpisodes(nil, camo, eqOrigin, eqStep, tracks)
	if nonBinary != 0 {
		t.Fatalf("aucune lecture non binaire attendue, obtenu %d", nonBinary)
	}
	if len(eps) != 1 {
		t.Fatalf("attendu 1 épisode, obtenu %d : %+v", len(eps), eps)
	}
	e := eps[0]
	if e.Fam != EquipFamilyCamo || e.Slot != 512 || e.T0 != 10 || e.T1 != 34 || !e.EndRead {
		t.Errorf("épisode camo mal borné : %+v (attendu slot 512, t0=10, t1=34, endRead)", e)
	}
}

func TestCamoEpisodeOuvertALaMortSeFermeALaFinDeLaVie(t *testing.T) {
	tracks := []Track{{Slot: 512, StartFrame: 5, EndFrame: 42}}
	camo := []filmdec.CamoRead{camoRead(512, 30, filmdec.CamoActiveQ)}
	eps, _ := buildEquipmentEpisodes(nil, camo, eqOrigin, eqStep, tracks)
	if len(eps) != 1 {
		t.Fatalf("attendu 1 épisode, obtenu %d : %+v", len(eps), eps)
	}
	e := eps[0]
	if e.T0 != 30 || e.T1 != 42 || e.EndRead {
		t.Errorf("un épisode ouvert doit se fermer à la fin de la vie SANS fin mesurée : %+v", e)
	}
}

func TestCamoActivationAnterieureALOrigineSeClampeAuDebutDeLaVie(t *testing.T) {
	tracks := []Track{{Slot: 512, StartFrame: 0, EndFrame: 42}}
	camo := []filmdec.CamoRead{
		{Slot: 512, TimestampUS: eqOrigin - 500_000, Q: filmdec.CamoActiveQ}, // avant la frame 0
		camoRead(512, 8, filmdec.CamoInactiveQ),
	}
	eps, _ := buildEquipmentEpisodes(nil, camo, eqOrigin, eqStep, tracks)
	if len(eps) != 1 {
		t.Fatalf("attendu 1 épisode, obtenu %d : %+v", len(eps), eps)
	}
	if e := eps[0]; e.T0 != 0 || e.T1 != 8 || !e.EndRead {
		t.Errorf("l'état actif avant l'origine doit se clamper à la frame 0, pas s'inventer plus tard : %+v", e)
	}
}

func TestCamoLectureNonBinaireCompteeMaisSansEffet(t *testing.T) {
	tracks := []Track{{Slot: 512, StartFrame: 0, EndFrame: 42}}
	camo := []filmdec.CamoRead{
		camoRead(512, 10, filmdec.CamoActiveQ),
		camoRead(512, 15, 2048), // jamais observée sur le corpus : ni ouvre, ni ferme
		camoRead(512, 20, filmdec.CamoInactiveQ),
	}
	eps, nonBinary := buildEquipmentEpisodes(nil, camo, eqOrigin, eqStep, tracks)
	if nonBinary != 1 {
		t.Fatalf("la lecture non binaire doit être COMPTÉE, obtenu %d", nonBinary)
	}
	if len(eps) != 1 || eps[0].T0 != 10 || eps[0].T1 != 20 {
		t.Errorf("la lecture non binaire ne doit pas fabriquer de transition : %+v", eps)
	}
}

func TestCamoVieNonPublieeNeProduitAucunEpisode(t *testing.T) {
	tracks := []Track{{Slot: 512, StartFrame: 0, EndFrame: 42}}
	camo := []filmdec.CamoRead{camoRead(999, 10, filmdec.CamoActiveQ)}
	eps, _ := buildEquipmentEpisodes(nil, camo, eqOrigin, eqStep, tracks)
	if eps != nil {
		t.Errorf("un slot sans trajectoire publiée n'a aucune fiche où poser l'épisode : %+v", eps)
	}
}

func TestOvershieldEpisodeSuitLaRegleQSup64(t *testing.T) {
	tracks := []Track{{Slot: 700, StartFrame: 0, EndFrame: 100}}
	pos := []filmdec.BipedPosition{
		shieldPos(700, 3, 60),  // bouclier normal
		shieldPos(700, 12, 64), // plein EXACT : pas un surbouclier (règle stricte q > 64)
		shieldPos(700, 18, 223),
		shieldPos(700, 30, 120),
		shieldPos(700, 45, 64), // épuisement : retour au plein — fin MESURÉE
	}
	eps, _ := buildEquipmentEpisodes(pos, nil, eqOrigin, eqStep, tracks)
	if len(eps) != 1 {
		t.Fatalf("attendu 1 épisode, obtenu %d : %+v", len(eps), eps)
	}
	e := eps[0]
	if e.Fam != EquipFamilyOvershield || e.T0 != 18 || e.T1 != 45 || !e.EndRead {
		t.Errorf("épisode surbouclier mal borné : %+v (attendu t0=18, t1=45, endRead)", e)
	}
}

func TestOvershieldMortEnSurboucliersFermeALaFinDeLaVie(t *testing.T) {
	tracks := []Track{{Slot: 700, StartFrame: 0, EndFrame: 60}}
	pos := []filmdec.BipedPosition{shieldPos(700, 50, 200)}
	eps, _ := buildEquipmentEpisodes(pos, nil, eqOrigin, eqStep, tracks)
	if len(eps) != 1 {
		t.Fatalf("attendu 1 épisode, obtenu %d : %+v", len(eps), eps)
	}
	if e := eps[0]; e.T0 != 50 || e.T1 != 60 || e.EndRead {
		t.Errorf("mort en surbouclier : fermeture à la fin de la vie, fin NON mesurée : %+v", e)
	}
}

func TestEquipmentEpisodesTriesEtCouvertureComptee(t *testing.T) {
	tracks := []Track{
		{Slot: 512, StartFrame: 0, EndFrame: 100},
		{Slot: 700, StartFrame: 0, EndFrame: 100},
		{Slot: 800, StartFrame: 0, EndFrame: 100},
	}
	camo := []filmdec.CamoRead{
		camoRead(512, 40, filmdec.CamoActiveQ),
		camoRead(512, 44, filmdec.CamoInactiveQ),
		camoRead(512, 60, filmdec.CamoActiveQ),
		camoRead(512, 70, filmdec.CamoInactiveQ),
	}
	pos := []filmdec.BipedPosition{
		shieldPos(700, 10, 200),
		shieldPos(700, 25, 60),
	}
	eps, _ := buildEquipmentEpisodes(pos, camo, eqOrigin, eqStep, tracks)
	if len(eps) != 3 {
		t.Fatalf("attendu 3 épisodes, obtenu %d : %+v", len(eps), eps)
	}
	// Tri par T0 : le surbouclier (t0=10) précède les deux épisodes camo (40, 60).
	if eps[0].Fam != EquipFamilyOvershield || eps[1].T0 != 40 || eps[2].T0 != 60 {
		t.Errorf("épisodes non triés par T0 : %+v", eps)
	}
	cov := equipmentCoverage(eps, tracks)
	if cov.TracksTotal != 3 || cov.CamoLives != 1 || cov.CamoEpisodes != 2 ||
		cov.OvershieldLives != 1 || cov.OvershieldEpisodes != 1 {
		t.Errorf("couverture fausse : %+v", cov)
	}
}

func TestEquipmentEpisodesSansDonneesRendNil(t *testing.T) {
	tracks := []Track{{Slot: 512, StartFrame: 0, EndFrame: 100}}
	if eps, _ := buildEquipmentEpisodes(nil, nil, eqOrigin, eqStep, tracks); eps != nil {
		t.Errorf("sans lecture, rien n'est inventé : %+v", eps)
	}
	if eps, _ := buildEquipmentEpisodes(nil, []filmdec.CamoRead{camoRead(512, 10, filmdec.CamoActiveQ)},
		eqOrigin, eqStep, nil); eps != nil {
		t.Errorf("sans trajectoire publiée, rien n'est publié : %+v", eps)
	}
}
