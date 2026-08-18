package objectiveevents

// slotidentity_deaths_bomb_test.go — LE GARDE-RAIL DU BLOQUANT DE PRODUCTION DU 2026-08-18.
//
// CE QUI S'EST PASSE. Le pont par instants de mort DEROULE la progression du compteur : une unite
// gagnee = un instant ajoute a la serie. Le garde d'entree ecartait les valeurs NEGATIVES (les
// ancrages parasites) et rien d'autre. Une emission POSITIVE aberrante — le champ lu au mauvais
// endroit sur un film dont la grammaire n'est pas celle qu'on croit — faisait donc boucler le
// deroulage jusqu'a cette valeur : `cmd/replay-build --facts` montait a 19-22 Go et ne rendait
// jamais la main.
//
// CE QUE CE TEST TIENT, ET POURQUOI IL EST SUR UN COMPTEUR ET PAS SUR UN CHRONOMETRE. Un test de
// duree serait flou (une machine chargee le fait tomber) et surtout il ne DIRAIT rien. Celui-ci
// mesure le nombre d'ALLOCATIONS, c'est-a-dire exactement la grandeur qui explosait : une lecture
// aberrante doit couter ZERO deroulage, pas « un peu moins ». Il tombe a la seconde ou le
// plafond disparait, et il tombe avec le bon message.

import (
	"runtime"
	"testing"
)

// bombRecord fabrique une emission du compteur de morts d'un slot de joueur.
func bombRecord(slot, timeMS int, deaths int64) StatRecord {
	return StatRecord{
		Slot: slot, TimeMS: timeMS,
		Comps: map[int]StatValue{coreKillsComp: {B: deaths}},
	}
}

// TestUneLectureAberranteNeDerouleRien : une valeur au-dela du plafond est JETEE, comme une
// negative — jamais deroulee.
func TestUneLectureAberranteNeDerouleRien(t *testing.T) {
	// 400 millions : l'ordre de grandeur qui produisait les 19-22 Go observes en production.
	const aberrante = int64(400_000_000)
	recs := []StatRecord{bombRecord(10, 1000, 3), bombRecord(10, 2000, aberrante)}

	var avant, apres runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&avant)
	got := deathProgressions(recs)
	runtime.ReadMemStats(&apres)

	if n := len(got[10]); n != 3 {
		t.Fatalf("%d instants deroules pour le slot 10, attendu 3 — l'emission aberrante a ete "+
			"DEROULEE au lieu d'etre jetee", n)
	}
	// Une seule serie de 3 entiers : quelques centaines d'octets. Le seuil est large a dessein —
	// ce qu'il interdit, c'est l'ordre de grandeur du defaut (des gigaoctets), pas le bruit.
	const plafondOctets = 1 << 20
	if d := apres.TotalAlloc - avant.TotalAlloc; d > plafondOctets {
		t.Errorf("%d octets alloues pour deux emissions, plafond %d — le deroulage n'est plus "+
			"borne", d, plafondOctets)
	}
}

// TestUneProgressionNormaleSeDerouleEncore : le plafond ne coute AUCUNE lecture vraie.
//
// SANS CE SECOND TEST, le premier serait satisfait par un garde qui jette tout — et le pont
// d'identite ne nommerait plus personne, en silence.
func TestUneProgressionNormaleSeDerouleEncore(t *testing.T) {
	recs := []StatRecord{
		bombRecord(10, 1000, 1), bombRecord(10, 2000, 2), bombRecord(10, 5000, 4),
		bombRecord(12, 1500, 1),
	}
	got := deathProgressions(recs)
	if want := []int{1000, 2000, 5000, 5000}; len(got[10]) != len(want) {
		t.Fatalf("slot 10 : %v, attendu %v — une progression ordinaire doit se derouler", got[10], want)
	}
	for i, v := range []int{1000, 2000, 5000, 5000} {
		if got[10][i] != v {
			t.Errorf("slot 10, instant %d : %d, attendu %d", i, got[10][i], v)
		}
	}
	if len(got[12]) != 1 {
		t.Errorf("slot 12 : %v, attendu une seule mort", got[12])
	}
}

// TestLePlafondEstAuBordEtPasEnDessous : la valeur du plafond elle-meme reste une lecture VRAIE.
//
// Le plafond est une BORNE DE SURETE, pas un seuil de plausibilite : le poser au plus juste
// ferait jeter des lectures vraies sur un mode ou une partie qu'on n'a pas encore vue.
func TestLePlafondEstAuBordEtPasEnDessous(t *testing.T) {
	got := deathProgressions([]StatRecord{bombRecord(10, 1000, maxDeathsPerSlot)})
	if n := len(got[10]); n != maxDeathsPerSlot {
		t.Errorf("%d instants pour une emission a la valeur du plafond, attendu %d — le plafond "+
			"doit etre INCLUS", n, maxDeathsPerSlot)
	}
	auDela := deathProgressions([]StatRecord{bombRecord(10, 1000, maxDeathsPerSlot+1)})
	if n := len(auDela[10]); n != 0 {
		t.Errorf("%d instants pour une emission JUSTE au-dessus du plafond, attendu 0", n)
	}
}
