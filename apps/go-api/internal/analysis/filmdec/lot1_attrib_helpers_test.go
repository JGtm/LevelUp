package filmdec

// lot1_attrib_helpers_test.go — ADAPTATEURS de recherche autour du numerateur productionise
// (weapon_hits.go). Plusieurs instruments (lot1_nmin, lot1_cadence_detente, lot1_degats_type1)
// consomment les tirs sous la forme historique attribShot et un index temporel attribIndex. Ces
// helpers NE RE-DECODENT RIEN : attribCollectShots delegue a ScanFilmWeaponShots, attribBuildIndex
// n'assemble que des index temporels (lot1mtIndexByKey). Une seule copie du decodage / pairing vit
// dans la production ; ceci n'est que du confort de recherche.

import "testing"

// attribShot : un tir horodate, forme historique consommee par les instruments de recherche.
type attribShot struct {
	ts   uint64
	att  uint64
	wid  uint64
	fidx int
	has  bool
}

// attribIndex : index temporels tries pour les mesures de fiabilite au temoin.
type attribIndex struct {
	dmgTsByResp map[uint64][]uint64 // degats indexes par responsable (ref1) -> ts tries
	shotTsByAtt map[uint64][]uint64 // tirs indexes par attaquant (ref0) -> ts tries
}

// attribCollectShots decode les tirs LONGS 0xD2 via le code de production (ScanFilmWeaponShots) et
// les mappe vers la forme historique attribShot.
func attribCollectShots(t *testing.T, dir string, n int) []attribShot {
	t.Helper()
	shots, err := ScanFilmWeaponShots(dir, n)
	if err != nil {
		t.Fatalf("collecte des tirs : %v", err)
	}
	out := make([]attribShot, len(shots))
	for i, s := range shots {
		out[i] = attribShot{ts: s.TimestampUS, att: s.Attacker, wid: s.WeaponID, fidx: s.FilmIndex, has: s.HasPair}
	}
	return out
}

// attribBuildIndex assemble les index temporels par attaquant (tirs) et responsable (degats).
func attribBuildIndex(shots []attribShot, dmg []sondeDmgEvt) attribIndex {
	var dmgTs, dmgResp []uint64
	for _, e := range dmg {
		if e.idx1 < 0 {
			continue
		}
		dmgTs = append(dmgTs, e.ts)
		dmgResp = append(dmgResp, uint64(e.idx1))
	}
	var shotTs, shotAtt []uint64
	for _, s := range shots {
		if !s.has {
			continue
		}
		shotTs = append(shotTs, s.ts)
		shotAtt = append(shotAtt, s.att)
	}
	return attribIndex{
		dmgTsByResp: lot1mtIndexByKey(dmgTs, dmgResp),
		shotTsByAtt: lot1mtIndexByKey(shotTs, shotAtt),
	}
}

// attribMin / attribMax : bornes d'un echantillon non vide (utilitaires de recherche).
func attribMin(xs []float64) float64 {
	m := xs[0]
	for _, x := range xs[1:] {
		if x < m {
			m = x
		}
	}
	return m
}

func attribMax(xs []float64) float64 {
	m := xs[0]
	for _, x := range xs[1:] {
		if x > m {
			m = x
		}
	}
	return m
}
