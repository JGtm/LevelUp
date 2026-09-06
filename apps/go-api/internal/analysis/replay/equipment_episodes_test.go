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

// TestEpisodeDUneVieAnterieureEstPublie — LA RÉGRESSION DU BALAYAGE DU PARC, FIGÉE AVEC SA
// PROVENANCE (instruction des régressions, candidate 3, 2026-09-06).
//
// LE CAS RÉEL. `82f29378` (Oasis) et `13d92593` (Dredge) perdaient leur UNIQUE épisode de
// surbouclier (1 -> 0) et `084a804d` (Fortitude Heavies) 2 épisodes de camouflage sur 21, à
// partir de `48cf4905d` (schéma 36, « une track = une vie »). `trackFrameWindows` n'indexait
// qu'UNE fenêtre par slot — la dernière vie — et `close` bornait l'épisode d'une vie
// antérieure à une fenêtre postérieure, ce qui le rendait vide (`t1 < t0`) donc muet.
//
// CE QUE LE TEST VERROUILLE : chaque vie du slot garde SES épisodes.
func TestEpisodeDUneVieAnterieureEstPublie(t *testing.T) {
	// Deux vies du même slot, disjointes ; un épisode de camouflage dans chacune.
	tracks := []Track{
		{Slot: 512, StartFrame: 0, EndFrame: 50},
		{Slot: 512, StartFrame: 200, EndFrame: 260},
	}
	camo := []filmdec.CamoRead{
		camoRead(512, 10, filmdec.CamoActiveQ), camoRead(512, 20, filmdec.CamoInactiveQ),
		camoRead(512, 210, filmdec.CamoActiveQ), camoRead(512, 230, filmdec.CamoInactiveQ),
	}
	eps, _ := buildEquipmentEpisodes(nil, camo, eqOrigin, eqStep, tracks)
	if len(eps) != 2 {
		t.Fatalf("%d épisode(s), attendu 2 — celui de la vie ANTÉRIEURE a été jeté : %+v", len(eps), eps)
	}
	if eps[0].T0 != 10 || eps[0].T1 != 20 {
		t.Errorf("épisode de la première vie [%d..%d], attendu [10..20]", eps[0].T0, eps[0].T1)
	}
	if eps[1].T0 != 210 || eps[1].T1 != 230 {
		t.Errorf("épisode de la seconde vie [%d..%d], attendu [210..230]", eps[1].T0, eps[1].T1)
	}
	cov := equipmentCoverage(eps, tracks)
	if cov.CamoEpisodes != 2 || cov.CamoLives != 2 {
		t.Errorf("couverture %+v, attendu camoEpisodes=2 et camoLives=2 — deux VIES du même "+
			"slot (constat C2 de la revue REG-R1 : le compteur indexait par slot)", cov)
	}
}

// TestEpisodeOuvertEnFinDeVieAnterieureSeFermeSurSaPropreVie — la fermeture « à la mort »
// suit la vie de l'ouverture, pas la dernière du slot.
//
// Sans cela, un camouflage ouvert dans la première vie et jamais éteint se serait fermé à la
// fin de la SECONDE vie : un épisode de 25 secondes traversant une réapparition.
func TestEpisodeOuvertEnFinDeVieAnterieureSeFermeSurSaPropreVie(t *testing.T) {
	tracks := []Track{
		{Slot: 512, StartFrame: 0, EndFrame: 50},
		{Slot: 512, StartFrame: 200, EndFrame: 260},
	}
	camo := []filmdec.CamoRead{camoRead(512, 40, filmdec.CamoActiveQ)}
	eps, _ := buildEquipmentEpisodes(nil, camo, eqOrigin, eqStep, tracks)
	if len(eps) != 1 {
		t.Fatalf("%d épisode(s), attendu 1 : %+v", len(eps), eps)
	}
	if eps[0].T0 != 40 || eps[0].T1 != 50 || eps[0].EndRead {
		t.Errorf("épisode [%d..%d] endRead=%v, attendu [40..50] endRead=false — la fermeture "+
			"doit suivre la vie de l'ouverture", eps[0].T0, eps[0].T1, eps[0].EndRead)
	}
}

// TestCouvertureDesEpisodesCompteDesViesPasDesSlots — CONSTAT C2 DE LA REVUE REG-R1
// (2026-09-06) : `camoLives` / `overshieldLives` comptaient des SLOTS sous un commentaire qui
// parle de vies — le défaut symétrique de celui corrigé le même jour pour
// `coverage.grapple.pullLives`. Le cas ne pouvait pas se produire avant ce correctif (seuls les
// épisodes de la dernière vie survivaient, donc slot == vie trivialement) : c'est lui qui le rend
// atteignable, à lui de compter juste.
func TestCouvertureDesEpisodesCompteDesViesPasDesSlots(t *testing.T) {
	tracks := []Track{
		{Slot: 512, StartFrame: 0, EndFrame: 50},
		{Slot: 512, StartFrame: 200, EndFrame: 260},
	}
	eps := []EquipmentEpisode{
		{Slot: 512, Fam: EquipFamilyCamo, T0: 10, T1: 20},
		{Slot: 512, Fam: EquipFamilyCamo, T0: 210, T1: 230},
		{Slot: 512, Fam: EquipFamilyOvershield, T0: 30, T1: 40},
		{Slot: 512, Fam: EquipFamilyOvershield, T0: 240, T1: 250},
	}
	cov := equipmentCoverage(eps, tracks)
	if cov.CamoEpisodes != 2 || cov.CamoLives != 2 {
		t.Errorf("camo : %d épisodes / %d vies, attendu 2 / 2 — deux vies du MÊME slot",
			cov.CamoEpisodes, cov.CamoLives)
	}
	if cov.OvershieldEpisodes != 2 || cov.OvershieldLives != 2 {
		t.Errorf("surbouclier : %d épisodes / %d vies, attendu 2 / 2",
			cov.OvershieldEpisodes, cov.OvershieldLives)
	}
}

// TestEpisodeAChevalSurDeuxViesGardeSesBornesMesurees — UN TROU DE REPLICATION N'AMPUTE PAS UNE
// MESURE QUI L'ENJAMBE (correctif du 2026-09-06, corpus temoin).
//
// L'etat actif se lit PAR SLOT (i28 pour le camo) : ses deux bornes sont des transitions LUES,
// qui ne savent rien du decoupage en vies. Depuis le schema 36 (« une track = une vie ») un trou
// de plus de `lifeGapUS` coupe la piste — et pour le camouflage cet etat est justement ce qui
// PROVOQUE le trou : un porteur invisible et immobile cesse d'etre replique. Borner l'episode a
// la seule vie de recouvrement MAXIMAL jetait alors la part couverte par l'autre vie, dont
// l'instant d'ACTIVATION. Mesure : `084a804d`, slot 620, camo lu [3105..3672] (568 frames),
// publie [3173..3672] (500) — 68 frames perdues, dont 16 A L'INTERIEUR d'une vie publiee, et
// l'activation sonnee 6,8 s en retard.
func TestEpisodeAChevalSurDeuxViesGardeSesBornesMesurees(t *testing.T) {
	// Deux vies du meme slot, separees d'un trou ; la SECONDE recouvre bien plus que la
	// premiere — c'est elle que l'ancienne regle elisait, en rognant l'ouverture.
	tracks := []Track{
		{Slot: 620, StartFrame: 0, EndFrame: 50},
		{Slot: 620, StartFrame: 60, EndFrame: 300},
		{Slot: 620, StartFrame: 400, EndFrame: 500}, // vie NON recouverte : elle ne doit rien elargir
	}
	camo := []filmdec.CamoRead{
		camoRead(620, 45, filmdec.CamoActiveQ),
		camoRead(620, 250, filmdec.CamoInactiveQ),
	}
	eps, _ := buildEquipmentEpisodes(nil, camo, eqOrigin, eqStep, tracks)
	if len(eps) != 1 {
		t.Fatalf("%d episode(s), attendu 1 : %+v", len(eps), eps)
	}
	if eps[0].T0 != 45 || eps[0].T1 != 250 {
		t.Errorf("episode [%d..%d], attendu [45..250] — les deux bornes sont MESUREES, le trou "+
			"de replication entre 50 et 60 ne doit en rogner aucune", eps[0].T0, eps[0].T1)
	}
	if !eps[0].EndRead {
		t.Errorf("endRead=false, attendu true — la fin est une transition mesuree, pas une mort")
	}
}
