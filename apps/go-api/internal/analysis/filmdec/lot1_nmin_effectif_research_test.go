package filmdec

// lot1_nmin_effectif_research_test.go — MESURE de l'effectif par cle (joueur, arme), pour
// FIXER le seuil Nmin (migration.WeaponHitsMinShots) de la porte de publication de la vue (b)
// « precision par arme selon la distance » (plan PRECISION_ARME_DISTANCE, Lot 1).
//
// QUESTION : au grain `match x joueur x arme`, combien de TIRS et combien de TOUCHES (tirs
// apparies a un damage_aftermath du meme attaquant, W = 250 ms) tombe-t-il par cle ? La
// distribution dit a partir de quel effectif une forme (taux, histogramme de distance) cesse
// d'etre du bruit. On ne fabrique rien : on reutilise l'attribution PAR LE TIR deja mesuree
// (attribCollectShots + attribBuildIndex + lot1mtNear), on regroupe par (FilmIndex, WeaponID).
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule, borne a
// deltaWitnessChunks. Lancer une fois par film (000d5950, 01e1f945, 00502e52) et lire la
// distribution ci-dessous pour retenir Nmin dans le doc-header de la migration.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// nminKey identifie une ventilation candidate : le tireur (index interne au film) et l'arme.
type nminKey struct {
	fidx int
	wid  uint64
}

// nminSurvivors compte les cles dont le compte atteint chaque seuil candidat.
func nminSurvivors(counts []int, thresholds []int) []int {
	out := make([]int, len(thresholds))
	for _, c := range counts {
		for i, th := range thresholds {
			if c >= th {
				out[i]++
			}
		}
	}
	return out
}

// nminMedian rend la mediane entiere d'un echantillon de comptes (0 si vide).
func nminMedian(counts []int) int {
	if len(counts) == 0 {
		return 0
	}
	cp := append([]int(nil), counts...)
	sort.Ints(cp)
	return cp[len(cp)/2]
}

// TestLot1NminEffectifParArme mesure, par cle (joueur, arme), la distribution du nombre de tirs
// et de touches, pour fixer Nmin.
func TestLot1NminEffectifParArme(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	t.Logf("== film %s · %d chunks balayes (effectif Nmin par cle joueur/arme) ==", filepath.Base(dir), n)

	shots := attribCollectShots(t, dir, n)
	dmg, _ := sondeScanDamage(t, dir, reg, n)
	idx := attribBuildIndex(shots, dmg)

	shotsBy := map[nminKey]int{}
	hitsBy := map[nminKey]int{}
	for _, s := range shots {
		if !s.has {
			continue
		}
		k := nminKey{s.fidx, s.wid}
		shotsBy[k]++
		if lot1mtNear(idx.dmgTsByResp[s.att], s.ts, attribW) {
			hitsBy[k]++
		}
	}

	shotCounts := make([]int, 0, len(shotsBy))
	for _, c := range shotsBy {
		shotCounts = append(shotCounts, c)
	}
	hitCounts := make([]int, 0, len(hitsBy))
	for _, c := range hitsBy {
		hitCounts = append(hitCounts, c)
	}

	th := []int{1, 3, 5, 8, 10, 15}
	shotSurv := nminSurvivors(shotCounts, th)
	hitSurv := nminSurvivors(hitCounts, th)

	t.Logf("cles (joueur, arme) distinctes : %d (tirs) · %d (avec >=1 touche)", len(shotsBy), len(hitsBy))
	t.Logf("mediane tirs/cle : %d · mediane touches/cle : %d", nminMedian(shotCounts), nminMedian(hitCounts))
	for i, s := range th {
		t.Logf("   seuil >=%2d : cles par TIRS %3d · cles par TOUCHES %3d", s, shotSurv[i], hitSurv[i])
	}
	t.Logf("LECTURE Nmin : retenir le seuil ou les cles survivantes restent des armes REELLEMENT")
	t.Logf("   utilisees (pas ramassees pour 1-2 tirs) ET ou l'effectif de TOUCHES reste lisible.")
}
