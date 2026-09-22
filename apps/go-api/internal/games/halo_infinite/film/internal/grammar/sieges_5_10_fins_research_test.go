//go:build research

package grammar

// sieges_5_10_fins_research_test.go — LES FINS DE VIE D UN VEHICULE : ce que le RECENSEMENT
// borne, ce que la SUPPRESSION date, et le temoin nomme du lot.
//
// SORTI DE `sieges_5_10_parent_research_test.go` par DEPLACEMENT PUR : le fichier passait le
// seuil de 500 lignes (CLAUDE.md regle 5, ratchet `archlint/film_file_size_test.go`). La passe
// partagee, elle, vit dans `sieges_5_10_passe_research_test.go`.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestSieges510FinsDeVie confronte, vie par vie, LE RECENSEMENT (les images-cles qui listent la
// vie) et LA SUPPRESSION (`recDel`). C est la mesure qui dit si le record de suppression est LE
// canal de la fin de vie non destructrice, et a quelle distance il tombe du dernier recensement.
func TestSieges510FinsDeVie(t *testing.T) {
	rec, ok := s510Passe(t)
	if !ok {
		return
	}
	kf := ScanWorldObjectKeyframes(rec.fc.Film(), VehicleTypeIndex)
	dernier := kf.LastTimeUS()
	type fin struct {
		key            types.EquipmentLifeKey
		premier, ultim uint64
		supUS          uint64
		supprime       bool
	}
	var fins []fin
	for k, vus := range kf.SeenUS {
		if len(vus) == 0 {
			continue
		}
		f := fin{key: k, premier: vus[0], ultim: vus[len(vus)-1]}
		for _, d := range rec.dels {
			if d.slot == k.Slot && d.gen == k.Gen && d.ts >= f.ultim {
				f.supUS, f.supprime = d.ts, true
				break
			}
		}
		fins = append(fins, f)
	}
	sort.SliceStable(fins, func(i, j int) bool { return fins[i].ultim < fins[j].ultim })
	var closes, avecSup, jusquAuBout int
	for _, f := range fins {
		if f.ultim >= dernier {
			jusquAuBout++
			continue
		}
		closes++
		if f.supprime {
			avecSup++
		}
		t.Logf("  vie %d/%d : recensee %s -> %s · suppression %s",
			f.key.Slot, f.key.Gen, s510Horloge(rec.ms(f.premier)), s510Horloge(rec.ms(f.ultim)),
			s510SupTexte(rec, f.supprime, f.supUS, f.ultim))
	}
	t.Logf("FINS DE VIE ti=40 : %d vies recensees · %d finissent AVEC le film · %d cessent d etre "+
		"recensees, dont %d portent une SUPPRESSION datee (%.1f %%)",
		len(fins), jusquAuBout, closes, avecSup, m533bPart(avecSup, closes))
}

// s510SupTexte rend la colonne « suppression » d une ligne de fin de vie.
func s510SupTexte(rec *s510Rec, supprime bool, supUS, ultim uint64) string {
	if !supprime {
		return "AUCUNE"
	}
	return fmt.Sprintf("%s (+%.1f s apres le dernier recensement)",
		s510Horloge(rec.ms(supUS)), float64(supUS-ultim)/1e6)
}

// s510Horloge rend un instant du film en m:ss.d.
func s510Horloge(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	return fmt.Sprintf("%d:%04.1f", ms/60000, float64(ms%60000)/1000)
}
