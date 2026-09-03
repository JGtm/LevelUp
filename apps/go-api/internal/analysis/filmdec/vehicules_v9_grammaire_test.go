package filmdec

// vehicules_v9_grammaire_test.go — INSTRUMENT DE MESURE (lot V9 : grammaire d'i2/i3 pour
// ti=40, et ce qu'elle rend d'i4). LECTURE SEULE, garde par V9_FILMS.
//
// LA QUESTION. La retro-ingenierie du 2026-09-03 dit que ti=40 porte, pour i2 et i3, DEUX
// deserialiseurs distincts de ceux du bipede (FUN_140c5f7ec / FUN_140d87740 contre
// FUN_14076e278 / FUN_140d70998). Cet instrument mesure ce que CHAQUE hypothese de
// grammaire fait a la VALEUR d'i4 (object-body-vitality) sur la bande ti=40, et compare au
// TEMOIN bipede (ti=35), dont la sante est validee en production.
//
// L'ORACLE, ecrit AVANT la mesure (le meme que celui qui a refute la tentative du lot V2b) :
// une VRAIE sante rend (a) une part de pas DECROISSANTS dans la fourchette bipede mesuree
// 13-26 %, et (b) un histogramme de quanta CONCENTRE pres du plein. Un curseur decale rend
// du bruit : ~50 % de pas decroissants et un histogramme uniforme.
//
// COMMENT il balaie sans dupliquer le marcheur : `SetRecordMaskHook` livre, pour chaque
// record ACCEPTE par `ScanBipedRecords`, le triplet (masque, charge utile, bit de depart).
// L'instrument le memorise puis rejoue `scanRecordDirs` sur ces memes triplets avec chaque
// grammaire. Un SEUL balayage de film pour N hypotheses, et aucune copie de la grammaire.
//
// USAGE (depuis apps/go-api, cache Go ISOLE) :
//
//	CGO_ENABLED=0 V9_FILM_ROOT=<repo>/data/cache V9_FILMS="0d76e8f1,fccc61cd" \
//	  V9_BOUNDS=<repo>/.../map_quant_bounds.json V9_MAPS="behemoth,launch site" \
//	  go test ./internal/analysis/filmdec/ -run TestV9Grammaire -v -timeout 90m

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// v9Record est un record ti=40 accepte, fige pour etre rejoue sous plusieurs grammaires.
type v9Record struct {
	idx []int
	pay []byte
	at  int
}

// v9Verdict agrege ce qu'une grammaire rend sur i4.
type v9Verdict struct {
	name             string
	samples          int
	hist             [8]int
	down, up, flat   int
	reachedI4        int // records dont le masque porte i4 ET qui l'ont atteint
	declaredI4       int // records dont le masque porte i4
	dirs             int // i2 a rendu une direction
	consecutivePairs int
}

func (v v9Verdict) downPct() float64 {
	n := v.down + v.up
	if n == 0 {
		return 0
	}
	return 100 * float64(v.down) / float64(n)
}

// topHeavy rend la part des quanta dans les DEUX buckets hauts (192..255) : une sante se
// tient pres du plein, un curseur decale s'etale.
func (v v9Verdict) topHeavy() float64 {
	tot := 0
	for _, c := range v.hist {
		tot += c
	}
	if tot == 0 {
		return 0
	}
	return 100 * float64(v.hist[6]+v.hist[7]) / float64(tot)
}

func TestV9Grammaire(t *testing.T) {
	films := v9Films(t)
	root := os.Getenv("V9_FILM_ROOT")
	if root == "" {
		t.Skip("V9_FILM_ROOT absent")
	}
	release := LockProcessDecode()
	defer release()

	for _, f := range films {
		t.Run(f, func(t *testing.T) { v9Film(t, root+"/film_chunks/"+f, f) })
	}
}

func v9Films(t *testing.T) []string {
	t.Helper()
	raw := os.Getenv("V9_FILMS")
	if raw == "" {
		t.Skip("V9_FILMS absent : instrument de mesure, garde par variable d'environnement")
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// v9Collect balaie la bande `ti` et fige les records acceptes.
func v9Collect(t *testing.T, dir string, ti int) []v9Record {
	t.Helper()
	kf := ScanFilmWorldObjectKeyframes(dir, ti)
	if len(kf.Band) == 0 {
		t.Fatalf("aucun slot ti=%d aux images-cles (%s)", ti, dir)
	}
	var recs []v9Record
	prev := recordMaskHook
	SetRecordMaskHook(func(idx []int, pay []byte, at int) {
		recs = append(recs, v9Record{idx: append([]int{}, idx...), pay: pay, at: at})
	})
	defer SetRecordMaskHook(prev)
	opt := ScanFilmOptions{RequireTag1: false, DropSaturated: true, CaptureDirs: true, QuantaOnly: true}
	if _, err := ScanFilmBipedPositionsForBand(dir, kf.Band, opt); err != nil {
		t.Fatalf("balayage ti=%d : %v", ti, err)
	}
	return recs
}

// v9Replay rejoue les records fige sous UNE grammaire et rend son verdict.
func v9Replay(name string, recs []v9Record, g dirsGrammar) v9Verdict {
	v := v9Verdict{name: name}
	var prevQ int = -1
	for _, r := range recs {
		declares := false
		for _, id := range r.idx[1:] {
			if id == 4 {
				declares = true
				break
			}
		}
		if declares {
			v.declaredI4++
		}
		dirs, vit := scanRecordDirs(r.pay, r.at, len(r.pay)*8, r.idx, g)
		if dirs.HasAim {
			v.dirs++
		}
		if !vit.HasBody {
			prevQ = -1
			continue
		}
		v.samples++
		if declares {
			v.reachedI4++
		}
		q := int(vit.Body.Q)
		v.hist[q/32]++
		if prevQ >= 0 {
			v.consecutivePairs++
			switch {
			case q < prevQ:
				v.down++
			case q > prevQ:
				v.up++
			default:
				v.flat++
			}
		}
		prevQ = q
	}
	return v
}

func v9Film(t *testing.T, dir, short8 string) {
	recs := v9Collect(t, dir, VehicleTypeIndex)
	t.Logf("  %s : %d records ti=%d acceptes", short8, len(recs), VehicleTypeIndex)

	dynP2 := dynPrecOrientationGrammar() // param lu dans paramByComponent (= 2 depuis V9)
	dyn := dynP2
	dyn.fwdUpParam = 1
	cases := []struct {
		name string
		g    dirsGrammar
	}{
		{"H0 bipede i2+i3 (grammaire du depot avant 2026-09-03)", dirsGrammar{}},
		{"H1 i2 dyn.-prec. SEUL", dirsGrammar{fwdUpDynPrec: true, fwdUpParam: 1}},
		{"H2 i3 dyn.-prec. SEUL", dirsGrammar{angVelDynPrec: true}},
		{"H3 i2+i3 dyn.-prec., param=1 (porte C NON lue)", dyn},
		{"H4 i2+i3 dyn.-prec., param de production (porte C lue)", dynP2},
		{"H5 i2 dyn.-prec. param=2 SEUL (i3 bipede)", dirsGrammar{fwdUpDynPrec: true, fwdUpParam: 2}},
	}
	for _, c := range cases {
		v := v9Replay(c.name, recs, c.g)
		t.Logf("    %-52s i4=%4d/%4d atteints · %5.1f %% down · haut(192+) %5.1f %% · hist %v · i2 dir %d",
			c.name, v.reachedI4, v.declaredI4, v.downPct(), v.topHeavy(), v.hist, v.dirs)
	}

	// TEMOIN : la MEME lecture sur la bande BIPEDE (ti=35), grammaire bipede — c'est la
	// sante validee en production. Il donne la fourchette de reference du gate.
	bip := v9Collect(t, dir, BipedTypeIndex)
	vb := v9Replay("temoin bipede", bip, dirsGrammar{})
	t.Logf("    %-52s i4=%4d/%4d atteints · %5.1f %% down · haut(192+) %5.1f %% · hist %v",
		"TEMOIN bipede ti=35 (sante validee)", vb.reachedI4, vb.declaredI4, vb.downPct(), vb.topHeavy(), vb.hist)
	if vb.samples == 0 {
		t.Fatalf("%s : temoin bipede vide — la mesure vehicule n'a pas de reference", short8)
	}
	fmt.Fprintln(os.Stderr) // separateur lisible entre films dans le journal
}
