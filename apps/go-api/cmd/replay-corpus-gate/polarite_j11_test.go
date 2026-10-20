package main

// polarite_j11_test.go — LE VERDICT DU G-CORPUS J11 REJOUE SOUS LA CLASSIFICATION TOTALE (lot R4).
//
// `testdata/gcorpus_j11_2026-09-28.json` est le rapport JSON du gate de corpus J11 (17 temoins,
// `.ai/V7.5/film_re/G_CORPUS_J11_2026-09-28.md`), tel que le gate l'a ecrit. Il porte les
// valeurs BRUTES (`ancien`, `nouveau`) de chaque ligne perte et changement — pas celles des
// gains, que le rapport compte sans les nommer. Ce test rejoue donc la classification des seules
// lignes nommees : chaque ligne de couverture numerique est recomparee par `replaydiff.Comparer`
// (la polarite de la table courante), puis passe par `classerPourLeVerdict` (denominateurs,
// telemetrie), et le camp obtenu est confronte au camp ecrit en 2026-09-28.
//
// CE QU'IL NE PEUT PAS DIRE : les ~70 compteurs d'echec qui MONTAIENT et sortaient en gain. Le
// rapport ne les nomme pas ; ils ne se chiffrent qu'en relancant le gate (hors de ce lot).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/replaydiff"
)

// campDuVerdict ramene un sens de verdict a son camp de rapport.
func campDuVerdict(sens string) string {
	switch sens {
	case replaydiff.SensPerte, replaydiff.SensDisparu:
		return "perte"
	case replaydiff.SensChangement:
		return "changement"
	case sensTelemetrie:
		return "telemetrie"
	}
	return "gain"
}

// mesureDEcart relit une valeur d'ecart du rapport ; vide = mesure absente.
func mesureDEcart(s string) (*replaydiff.Mesure, bool) {
	if s == "" {
		return nil, true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, false
	}
	return &replaydiff.Mesure{EstNum: true, Num: v}, true
}

// reclasserLigne recompare UNE ligne de couverture numerique du rapport sous la table courante.
func reclasserLigne(d detailJSON) (replaydiff.Difference, bool) {
	if _, couverture := replaydiff.PolariteDe(d.Metrique); !couverture {
		return replaydiff.Difference{}, false
	}
	a, okA := mesureDEcart(d.Ancien)
	b, okB := mesureDEcart(d.Nouveau)
	if !okA || !okB {
		return replaydiff.Difference{}, false
	}
	k := d.Axe + "/" + d.Metrique
	ea := replaydiff.Empreinte{Mesures: map[string]replaydiff.Mesure{}}
	eb := replaydiff.Empreinte{Mesures: map[string]replaydiff.Mesure{}}
	if a != nil {
		ea.Mesures[k] = *a
	}
	if b != nil {
		eb.Mesures[k] = *b
	}
	rap := replaydiff.Comparer(ea, eb)
	if len(rap.Differences) != 1 {
		return replaydiff.Difference{}, false
	}
	return rap.Differences[0], true
}

// denominateurNonNomme : une ligne que le gate avait rangee en CHANGEMENT parce que son
// denominateur avait bouge (`classerPourLeVerdict`), et dont le denominateur n est pas nomme au
// rapport (il etait un gain). Sans sa valeur, le rejeu ne peut pas refaire le rapport : la ligne
// est comptee INDECIDABLE plutot que basculee a tort en perte.
func denominateurNonNomme(avant, apres, metrique string,
	parMetrique map[string]replaydiff.Difference) bool {
	if avant != "changement" || apres != "perte" {
		return false
	}
	denom, rejet := denominateurDe(metrique)
	if !rejet || denom == "" {
		return false
	}
	_, nomme := parMetrique[denom]
	return !nomme
}

type bascule struct{ avant, apres, metrique string }

// rejouerRapportJ11 rend les bascules de camp, par temoin et ligne, et le nombre de lignes lues.
func rejouerRapportJ11(t *testing.T) ([]bascule, int, int, []string) {
	t.Helper()
	brut, err := os.ReadFile(filepath.Join("testdata", "gcorpus_j11_2026-09-28.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rap rapportJSON
	if err := json.Unmarshal(brut, &rap); err != nil {
		t.Fatal(err)
	}
	var bascules []bascule
	var statuts []string
	lignes, couverture := 0, 0
	for _, tem := range rap.Temoins {
		type ligne struct {
			avant string
			d     replaydiff.Difference
			recla bool
		}
		var ls []ligne
		parMetrique := map[string]replaydiff.Difference{}
		for _, grp := range []struct {
			camp  string
			diffs []detailJSON
		}{{"perte", tem.PertesDetail}, {"changement", tem.ChangementsDetail}} {
			for _, dj := range grp.diffs {
				d, ok := reclasserLigne(dj)
				if !ok {
					d = replaydiff.Difference{Axe: dj.Axe, Metrique: dj.Metrique, Sens: dj.Sens,
						Ancien: dj.Ancien, Nouveau: dj.Nouveau}
				}
				ls = append(ls, ligne{avant: grp.camp, d: d, recla: ok})
				parMetrique[d.Metrique] = d
			}
		}
		pertes, changements := 0, 0
		for _, l := range ls {
			lignes++
			if !l.recla {
				if l.avant == "perte" {
					pertes++
				} else {
					changements++
				}
				continue
			}
			couverture++
			apres := campDuVerdict(classerPourLeVerdict(l.d, parMetrique))
			if denominateurNonNomme(l.avant, apres, l.d.Metrique, parMetrique) {
				apres = "indecidable"
			}
			if apres != l.avant {
				bascules = append(bascules, bascule{l.avant, apres, l.d.Metrique})
			}
			switch apres {
			case "perte":
				pertes++
			case "changement", "indecidable":
				changements++
			}
		}
		statuts = append(statuts, fmt.Sprintf("%s %s pertes %d->%d changements %d->%d", tem.ID,
			tem.Statut, len(tem.PertesDetail), pertes, len(tem.ChangementsDetail), changements))
	}
	return bascules, lignes, couverture, statuts
}

// TestRejeuJ11_LignesQuiChangentDeCamp — le chiffrage du lot R4, fige.
func TestRejeuJ11_LignesQuiChangentDeCamp(t *testing.T) {
	bascules, lignes, couverture, statuts := rejouerRapportJ11(t)
	parSens := map[string]int{}
	parMetrique := map[string]int{}
	for _, b := range bascules {
		parSens[b.avant+" -> "+b.apres]++
		parMetrique[b.avant+" -> "+b.apres+"  "+b.metrique]++
	}
	var detail []string
	for k, n := range parMetrique {
		detail = append(detail, fmt.Sprintf("%3d  %s", n, k))
	}
	sort.Strings(detail)
	t.Logf("%d lignes nommees, dont %d de couverture numerique ; %d changent de camp : %v\n%s",
		lignes, couverture, len(bascules), parSens, strings.Join(detail, "\n"))
	t.Logf("par temoin :\n%s", strings.Join(statuts, "\n"))
	// FIGE LE 2026-09-28 (lot R4). 390 lignes nommees, 209 de couverture numerique :
	//   105 pertes -> gains       compteurs d'echec qui BAISSAIENT (holes*, *Unlocated, no_owner,
	//                             */unknown, desync, unconfirmed, uncovered, turretRides*...) ;
	//    17 pertes -> changements denominateurs et ventilations (population, aimRideFrames,
	//                             byKind.*, spawned, nonWeapon, shotsOnCarrier, deathOffsetMs...) ;
	//     1 changement -> gain    `vehicles.shotsAmbiguous` qui baisse ;
	//     1 changement -> indecidable  `vehicles.shotsNoRide` : son denominateur etait un gain,
	//                             non nomme au rapport.
	// Par temoin : 336 pertes -> 214, 54 changements -> 70 ; aucun temoin ne sort sans perte.
	attendu := map[string]int{
		"perte -> gain": 105, "perte -> changement": 17,
		"changement -> gain": 1, "changement -> indecidable": 1,
	}
	if fmt.Sprint(parSens) != fmt.Sprint(attendu) || lignes != 390 || couverture != 209 {
		t.Errorf("bascules %v sur %d lignes (%d de couverture), attendu %v sur 390 (209)", parSens,
			lignes, couverture, attendu)
	}
	avant, apres := 0, 0
	for _, s := range statuts {
		var id, statut string
		var pa, pn, ca, cn int
		if _, err := fmt.Sscanf(s, "%s %s pertes %d->%d changements %d->%d", &id, &statut, &pa, &pn,
			&ca, &cn); err != nil {
			t.Fatalf("%q : %v", s, err)
		}
		avant, apres = avant+pa, apres+pn
	}
	if avant != 336 || apres != 214 {
		t.Errorf("pertes %d -> %d, attendu 336 -> 214", avant, apres)
	}
}
