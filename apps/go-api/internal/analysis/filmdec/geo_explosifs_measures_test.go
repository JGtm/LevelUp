package filmdec

// geo_explosifs_measures_test.go — mesures M2/M3/M4 et appariement fatal de l'instrument
// geo_explosifs_research_test.go (scinde pour le seuil de 500 lignes).

import "testing"

// geoMaxChunks borne le balayage (RAM : un film BTB est gros). 16 = compromis arene/BTB.
const geoMaxChunks = 16

// geoConfMargin / geoConfAngle : un choix geometrique est CONFIANT si sa marge de cout sur le
// second candidat depasse geoConfMargin ET si le gagnant vise la victime a moins de geoConfAngle.
const (
	geoConfMargin = 20.0 // deg de cout
	geoConfAngle  = 25.0 // deg d'alignement
)

// geoAngleBucket incremente l'histogramme <5,<15,<30,<60,>=60 degres.
func geoAngleBucket(a float64, h *[5]int) {
	switch {
	case a < 5:
		h[0]++
	case a < 15:
		h[1]++
	case a < 30:
		h[2]++
	case a < 60:
		h[3]++
	default:
		h[4]++
	}
}

// geoMatchFatal marque chaque touche appariee a une mort de la MEME victime dans geoMatchWin
// (la touche fatale precede/coincide la mort), et lui attache le tueur EnumB.
func geoMatchFatal(touch []geoTouch, kills []geoKill) {
	for i := range touch {
		if !touch[i].hasVict {
			continue
		}
		var bestK *geoKill
		var bd uint64
		for j := range kills {
			k := &kills[j]
			if k.victSlot != touch[i].victSlot {
				continue
			}
			d := touch[i].ts - k.ts
			if k.ts > touch[i].ts {
				d = k.ts - touch[i].ts
			}
			if d <= geoMatchWin && (bestK == nil || d < bd) {
				bestK, bd = k, d
			}
		}
		if bestK != nil {
			touch[i].fatal = true
			touch[i].killer = bestK.killer
		}
	}
}

// geoValidateAim VALIDE la geometrie de visee independamment, sur les degats DIRECTS (ref1 =
// tireur BIPEDE, victime ref0 connue) : l'angle entre la visee du tireur et la direction vers sa
// victime, a l'instant du degat, doit etre PETIT (hitscan/projectile rapide vise la cible). Si
// cette mediane est petite, la convention visee->vecteur et les positions sont correctes ; une
// mediane elevee ici invaliderait tout le reste.
func geoValidateAim(t *testing.T, raws []geoRawDmg, base int, tracks map[uint32][]geoAimSample) {
	t.Helper()
	bi := 0
	for i, b := range lot1chBases {
		if b == base {
			bi = i
		}
	}
	var ang []float64
	buck := [5]int{} // <5, <15, <30, <60, >=60
	for _, e := range raws {
		if !e.bip0[bi] || !e.bip1[bi] || e.idx0 < 0 || e.idx1 < 0 {
			continue
		}
		att, oka := geoLookup(tracks[uint32(base+e.idx1)], e.ts, geoPosTolUS)
		vic, okv := geoLookup(tracks[uint32(base+e.idx0)], e.ts, geoPosTolUS)
		if !oka || !okv {
			continue
		}
		a, ok := geoAngleToVictim(att, vic)
		if !ok {
			continue
		}
		ang = append(ang, a)
		geoAngleBucket(a, &buck)
	}
	t.Logf("M0 VALIDATION VISEE sur degats DIRECTS (tireur=ref1 bipede vise sa victime ref0) :")
	t.Logf("   n=%d · mediane %.1f deg · <5:%d <15:%d <30:%d <60:%d >=60:%d",
		len(ang), geoMedian(ang), buck[0], buck[1], buck[2], buck[3], buck[4])
	t.Logf("   LECTURE : mediane PETITE = convention visee + positions correctes (socle de M2/M3/M4).")
}

// geoM2Coverage : couverture d'attribution + ambiguite + couverture de la visee + alignement
// du gagnant geometrique par classe d'arme.
func geoM2Coverage(t *testing.T, touch []geoTouch, heavy []geoShot, tracks map[uint32][]geoAimSample, speedByWid map[uint64]float64) {
	t.Helper()
	nVict, nCand, nConfirm := 0, 0, 0
	amb := [4]int{} // 0,1,2,>=3 tireurs distincts
	candShots, candAim := 0, 0
	ah := [5]int{} // histogramme du meilleur alignement : <5,<15,<30,<60,>=60
	for _, tc := range touch {
		if tc.hasVict {
			nVict++
		}
		cands := geoFindCandidates(tc, heavy, tracks)
		if len(cands) == 0 {
			continue
		}
		nCand++
		ds := geoDistinctShooters(cands)
		switch {
		case ds >= 3:
			amb[3]++
		default:
			amb[ds]++
		}
		for _, c := range cands {
			candShots++
			if c.hasAngle {
				candAim++
			}
		}
		if best, ok := geoBestAligned(cands); ok {
			geoAngleBucket(best.angle, &ah)
			if best.angle < geoAlignConfirm {
				nConfirm++
			}
		}
	}
	t.Logf("M2 COUVERTURE / AMBIGUITE / VISEE :")
	t.Logf("   touches %d · victime resolue %d · >=1 tir lourd candidat %d (%.1f %% des victimes resolues)",
		len(touch), nVict, nCand, lot1Pct(nCand, nVict))
	t.Logf("   ambiguite (tireurs lourds distincts dans la fenetre) : 1=%d · 2=%d · >=3=%d (0 candidat exclu)",
		amb[1], amb[2], amb[3])
	t.Logf("   couverture VISEE (i21) a l'instant du tir : %d/%d records candidats (%.1f %%)",
		candAim, candShots, lot1Pct(candAim, candShots))
	t.Logf("   meilleur alignement par touche : <5:%d <15:%d <30:%d <60:%d >=60:%d",
		ah[0], ah[1], ah[2], ah[3], ah[4])
	t.Logf("   CONFIRMEES geometriquement (meilleur alignement < %.0f deg) : %d/%d touches a candidat (%.1f %%)",
		geoAlignConfirm, nConfirm, nCand, lot1Pct(nConfirm, nCand))
	t.Logf("   LECTURE : la coincidence temporelle catch ~57 deg au hasard ; l'alignement isole les VRAIES touches.")
}

// geoM3Truth : accord GEOMETRIE vs TEMPOREL contre le dead-state, sur les touches FATALES dont
// le tueur est mappe en FilmIndex.
func geoM3Truth(t *testing.T, touch []geoTouch, heavy []geoShot, tracks map[uint32][]geoAimSample, speedByWid map[uint64]float64, table map[int32]int) {
	t.Helper()
	var evalN, geoOK, tmpOK int
	var ambN, ambGeoOK, ambTmpOK int
	for _, tc := range touch {
		if !tc.fatal {
			continue
		}
		trueFilm, ok := table[tc.killer]
		if !ok {
			continue // tueur non mappe (n'a pas tire, ou victime jamais tireuse) : hors mesure
		}
		cands := geoFindCandidates(tc, heavy, tracks)
		if len(cands) == 0 {
			continue
		}
		evalN++
		ambiguous := geoDistinctShooters(cands) >= 2
		if ambiguous {
			ambN++
		}
		if gw, _, ok := geoGeometricWinner(cands, speedByWid); ok && gw.shot.film == trueFilm {
			geoOK++
			if ambiguous {
				ambGeoOK++
			}
		}
		if tw, ok := geoTemporalWinner(cands); ok && tw.shot.film == trueFilm {
			tmpOK++
			if ambiguous {
				ambTmpOK++
			}
		}
	}
	t.Logf("M3 VERITE TERRAIN (touche fatale · gagnant == tueur dead-state EnumB) :")
	t.Logf("   evaluables %d · GEOMETRIE %d (%.1f %%) · TEMPOREL-recent %d (%.1f %%)",
		evalN, geoOK, lot1Pct(geoOK, evalN), tmpOK, lot1Pct(tmpOK, evalN))
	t.Logf("   sous-ensemble AMBIGU (>=2 tireurs) : %d · GEOMETRIE %d (%.1f %%) · TEMPOREL %d (%.1f %%)",
		ambN, ambGeoOK, lot1Pct(ambGeoOK, ambN), ambTmpOK, lot1Pct(ambTmpOK, ambN))
	t.Logf("   LECTURE : l'ecart geometrie-temporel sur le sous-ensemble AMBIGU est le gain propre de la geometrie.")
}

// geoM4Gain : sur les touches AMBIGUES (>=2 tireurs), la geometrie tranche-t-elle avec confiance
// la ou le temporel-unique abstient ? Et sur le sous-ensemble fatal mappe, avec quelle justesse ?
func geoM4Gain(t *testing.T, touch []geoTouch, heavy []geoShot, tracks map[uint32][]geoAimSample, speedByWid map[uint64]float64, table map[int32]int) {
	t.Helper()
	var ambN, confident int
	var fatalAmb, fatalGeoOK, fatalTmpOK int
	for _, tc := range touch {
		cands := geoFindCandidates(tc, heavy, tracks)
		if geoDistinctShooters(cands) < 2 {
			continue
		}
		ambN++
		gw, margin, ok := geoGeometricWinner(cands, speedByWid)
		conf := ok && gw.hasAngle && margin >= geoConfMargin && gw.angle <= geoConfAngle
		if conf {
			confident++
		}
		if tc.fatal {
			if trueFilm, okm := table[tc.killer]; okm {
				fatalAmb++
				if ok && gw.shot.film == trueFilm {
					fatalGeoOK++
				}
				if tw, okt := geoTemporalWinner(cands); okt && tw.shot.film == trueFilm {
					fatalTmpOK++
				}
			}
		}
	}
	t.Logf("M4 GAIN BTB (touches ambigues : temporel-UNIQUE abstient par definition) :")
	t.Logf("   touches ambigues %d · geometrie tranche avec CONFIANCE (marge>=%.0f, alignement<=%.0f) %d (%.1f %%)",
		ambN, geoConfMargin, geoConfAngle, confident, lot1Pct(confident, ambN))
	t.Logf("   sous-ensemble AMBIGU + FATAL mappe %d : geometrie juste %d (%.1f %%) · temporel-recent juste %d (%.1f %%)",
		fatalAmb, fatalGeoOK, lot1Pct(fatalGeoOK, fatalAmb), fatalTmpOK, lot1Pct(fatalTmpOK, fatalAmb))
	t.Logf("   LECTURE : sur ce film, la part de touches ambigues situe le besoin ; la justesse geometrique situe l'apport.")
}
