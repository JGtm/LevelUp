package replay

import (
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// camo_duration_distribution_test.go — LA DISTRIBUTION DES DUREES DES EPISODES CAMO DU
// GOLDEN, PUBLIEE ET VERIFIEE.
//
// POURQUOI CE TEST. Le commentaire de CamoStates (golden_inputs_test.go) et celui de
// renderEquipment (golden_assembly_test.go) attribuaient les 698 lectures binaires du
// canal i28 queue[1] sur le film de reference (000d5950, Fiesta, 8 joueurs, 0 porteur
// d'equipement rang 8) a un « POWER-UP de camouflage ». C'EST FAUX : ce mode ne pose
// aucun equipement au sol a ramasser. Ce que le canal capte est le DASH du mode Fiesta,
// qui declenche un camouflage court a chaque activation (enseignement utilisateur du
// 2026-08-16) — rien n'est ramasse.
//
// CE QUE CE TEST A MESURE, ET CE QUE CETTE MESURE A REFUTE AU PASSAGE. La duree brute
// d'un episode FUSIONNE (premiere lecture active -> derniere lecture inactive mesuree,
// T1-T0 : deux POINTS sur l'axe des frames, pas un compte inclusif — cette derniere
// convention est celle de l'AFFICHAGE, cf. equipmentFx.test.ts, distincte de la duree
// mesuree ici) N'EST PAS bornee a 1-2 s : mediane 2,7 s, jusqu'a 34,5 s. Instrumente
// (activationsInWindow ci-dessous), la raison est le CHAINAGE : 21 des 36 episodes du
// golden contiennent PLUSIEURS lectures actives brutes avant la lecture inactive qui les
// ferme — plusieurs dashes reactivent le canal avant que le precedent ne soit retombe a
// zero, et la fenetre [T0,T1] fusionne ces reactivations en un seul episode continu que
// le decodage ne peut pas decomposer (le film ne date que des TRANSITIONS, jamais des
// reaffirmations d'un etat inchange).
//
// CE QUI RESTE VRAI, MESURE ICI, ET FALSIFIABLE : un episode SANS reactivation (une
// seule lecture active dans sa fenetre — le cas le plus proche d'un dash isole, non
// chaine) ne depasse JAMAIS quelques secondes sur ce golden (max mesure 9,6 s, sur 15
// episodes de ce type). Un power-up d'Active Camo ramasse dure typiquement plusieurs
// dizaines de secondes (le surbouclier du meme lot, lui plus proche d'un ramassage
// franc qu'un declencheur, atteint 61,6 s sur le temoin 084a804d) : si le canal
// capturait un ramassage, les episodes a activation unique s'etaleraient jusque-la. Ils
// ne le font pas — c'est la prediction que ce test verrouille.
//
// ZERO OCTET DE FILM : le meme fixture fige que golden_inputs_test.go /
// golden_assembly_test.go — aucune regeneration necessaire pour faire tourner ce test.

// singleActivationMaxMS borne haute d'un episode camo A ACTIVATION UNIQUE (aucune
// reactivation mesuree dans sa fenetre) — tres au-dessus du maximum mesure (9,6 s) pour
// laisser de la marge au bruit de decodage, tres en-dessous d'une duree de power-up Halo
// typique (plusieurs dizaines de secondes, cf. le surbouclier a 61,6 s cite plus haut).
const singleActivationMaxMS = 15_000

// singleActivationFloor : nombre MINIMAL d'episodes a activation unique attendu sur ce
// golden. Il en porte 15 aujourd'hui ; le plancher est bas (10) pour ne pas re-figer une
// mesure a la ligne pres, mais il n'est PAS zero — c'est lui qui empeche la prediction
// ci-dessous de passer sur une population vide.
const singleActivationFloor = 10

// TestCamoEpisodeDurationDistributionDuGolden construit le document du film de reference
// et publie la distribution des durees de ses episodes de camouflage actif.
func TestCamoEpisodeDurationDistributionDuGolden(t *testing.T) {
	g := loadGoldenInputs(t)
	doc := BuildFromPositions(goldenFilm, "halo_infinite", g.Positions, g.Fire, g.options())
	if doc.FrameIntervalMS <= 0 {
		t.Fatalf("intervalle de frame nul ou negatif : %d", doc.FrameIntervalMS)
	}
	if len(g.Positions) == 0 {
		t.Fatal("golden sans position : fixture vide")
	}

	// MEME calcul d'origine que BuildFromPositions (build.go) : le minimum des
	// horodatages de positions. Recalcule ici plutot qu'expose, pour ne pas ajouter une
	// dependance de plus a la frontiere du paquet pour un seul test.
	origin := g.Positions[0].TimestampUS
	for _, p := range g.Positions {
		if p.TimestampUS < origin {
			origin = p.TimestampUS
		}
	}
	step := uint64(doc.FrameIntervalMS) * 1000

	bySlot := map[uint32][]filmdec.CamoRead{}
	for _, r := range g.CamoStates {
		bySlot[r.Slot] = append(bySlot[r.Slot], r)
	}
	// activationsInWindow compte les lectures ACTIVES BRUTES dans [t0,t1] : 1 = episode
	// isole (rien ne l'a reactive avant la fermeture mesuree), >1 = chaine.
	activationsInWindow := func(slot uint32, t0, t1 int) int {
		n := 0
		for _, r := range bySlot[slot] {
			if r.Q != filmdec.CamoActiveQ {
				continue
			}
			if f := frameOf(r.TimestampUS, origin, step); f >= t0 && f <= t1 {
				n++
			}
		}
		return n
	}

	type episode struct {
		durationMS int
		single     bool
	}
	var eps []episode
	measuredEnd, closedAtDeath := 0, 0
	for _, e := range doc.EquipmentEpisodes {
		if e.Fam != EquipFamilyCamo {
			continue
		}
		eps = append(eps, episode{
			durationMS: (e.T1 - e.T0) * doc.FrameIntervalMS,
			single:     activationsInWindow(e.Slot, e.T0, e.T1) <= 1,
		})
		if e.EndRead {
			measuredEnd++
		} else {
			closedAtDeath++
		}
	}
	if len(eps) == 0 {
		t.Fatal("aucun episode camo dans le golden — le fixture a change, revoir ce test avant de le corriger")
	}

	durationsMS := make([]int, len(eps))
	for i, e := range eps {
		durationsMS[i] = e.durationMS
	}
	sort.Ints(durationsMS)

	// Histogramme : bornes hautes EXCLUES, sauf la derniere (ouverte).
	edgesMS := []int{500, 1000, 1500, 2000, 3000, 5000, 10000}
	labels := []string{"<0,5s", "0,5-1s", "1-1,5s", "1,5-2s", "2-3s", "3-5s", "5-10s", ">=10s"}
	counts := make([]int, len(labels))
	for _, d := range durationsMS {
		i := 0
		for i < len(edgesMS) && d >= edgesMS[i] {
			i++
		}
		counts[i]++
	}

	sum, singleMax, chained := 0, 0, 0
	for _, e := range eps {
		sum += e.durationMS
		if e.single {
			if e.durationMS > singleMax {
				singleMax = e.durationMS
			}
		} else {
			chained++
		}
	}
	median := durationsMS[len(durationsMS)/2]
	mean := sum / len(durationsMS)

	t.Logf("golden %s : %d episode(s) camo (%d fin(s) MESUREE(s), %d ferme(s) a la mort)",
		goldenFilm, len(eps), measuredEnd, closedAtDeath)
	for i, lbl := range labels {
		t.Logf("  %-7s : %d/%d", lbl, counts[i], len(eps))
	}
	t.Logf("  min=%dms mediane=%dms moyenne=%dms max=%dms",
		durationsMS[0], median, mean, durationsMS[len(durationsMS)-1])
	t.Logf("  %d/%d episode(s) CHAINE(s) (>1 lecture active dans la fenetre, aucune fin mesuree entre) — "+
		"max d'un episode a ACTIVATION UNIQUE : %dms sur %d episode(s) de ce type",
		chained, len(eps), singleMax, len(eps)-chained)

	// LA POPULATION D'ABORD, LA PREDICTION ENSUITE. `singleMax` part de 0 et n'est releve
	// que par des episodes A ACTIVATION UNIQUE : si un changement de decodage faisait
	// tomber cette population a zero (par exemple des reaffirmations d'etat comptees comme
	// des reactivations), `0 > 15000` resterait faux et ce test PASSERAIT SANS RIEN
	// ASSERTER — un gate qui ne peut plus echouer ne garde rien. Le plancher est donc
	// verifie AVANT la prediction, et il echoue bruyamment.
	if single := len(eps) - chained; single < singleActivationFloor {
		t.Fatalf("%d episode(s) camo a ACTIVATION UNIQUE, plancher %d : la prediction ci-dessous "+
			"n'aurait plus de population a verifier — le decodage du canal i28 a change, "+
			"mesurer avant de toucher a ce test", single, singleActivationFloor)
	}

	// PREDICTION FALSIFIABLE : un episode A ACTIVATION UNIQUE (rien ne l'a reactive) est
	// la lecture la plus proche d'un dash isole. S'il s'agissait d'un power-up ramasse,
	// ces episodes isoles s'etaleraient jusqu'a plusieurs dizaines de secondes (sa duree
	// de ramassage) — cf. le surbouclier a 61,6 s cite en tete de fichier. Ils ne le
	// font pas sur ce golden : la lecture dash tient.
	if singleMax > singleActivationMaxMS {
		t.Errorf("prediction 'dash Fiesta' refutee : un episode a activation UNIQUE dure %dms (> %dms) — "+
			"aussi long qu'un ramassage, revoir le commentaire de CamoStates et de renderEquipment",
			singleMax, singleActivationMaxMS)
	}
}
