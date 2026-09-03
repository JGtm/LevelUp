package filmdec

// vehicules_v10_deadstate_test.go — INSTRUMENT DE MESURE (lot V10, signal `i11 object-dead-state`
// de `ti=40`). LECTURE SEULE, garde par V10_FILMS.
//
// CE QUE GHIDRA A ETABLI AVANT CETTE MESURE (2026-09-03, chaine statique du lot V9) :
//
//	chaine ASCII `object-dead-state-component` @0x143c99320 — UNIQUE occurrence du binaire :
//	    il n'existe AUCUN `object-dead-state-dynamic-precision-component`, donc AUCUN piege
//	    equivalent a celui d'i2/i3 (verifie par /search_strings, 1 seule chaine) ;
//	xref DATA unique -> thunk `name()` @0x14064c6d0 (`48 8D 05 49 CC 64 03 C3`) ;
//	slot qui stocke ce thunk @0x143d0ba48 -> vtable du descripteur @0x143d0ba40 ;
//	vtable[0x28] = 0x14076ce9c = LE THUNK -> deser = vtable[0x30] = **0x140c1dce0**.
//
//	FUN_140c1dce0 decompile :
//	    mort = R(1) -> comp+0x70
//	    si ti == 0x23 || ti == 0x28 : FUN_140c1dd44(comp+0x74, br)
//	    si ti == 0x23 : R(1) -> comp+0xc4
//
//	=> pour `ti=40` (0x28) la forme est EXACTEMENT celle du bipede MOINS le bit de queue, et
//	FUN_140c1dd44 ne recoit PAS le typeIndex : le corps lourd est le MEME objet que celui qui a
//	resolu l'arme du kill a 97,6 % sur le bipede. Le depot le portait deja
//	(`consumeObjectDeadStateBipedTI`, branche `typeIndex == 0x28` de `consumeByName`).
//
// CE QUI MANQUAIT, ET C'EST TOUT LE SUJET DE CE LOT : le balayage OFFLINE (`scanRecordDirs`)
// s'arretait au premier composant non modelise, donc n'atteignait JAMAIS i11 sur un record qui
// declare i6..i10. Ce test mesure d'abord COMBIEN de records `ti=40` declarent i11 et ce que
// leur masque porte AVANT lui — c'est-a-dire le cout exact du portage a faire.
//
// USAGE (depuis apps/go-api, cache Go ISOLE) :
//
//	CGO_ENABLED=0 V10_FILM_ROOT=<repo>/data/cache V10_FILMS="0d76e8f1:behemoth" \
//	  V10_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  go test ./internal/analysis/filmdec/ -run '^TestV10MasqueDeadState$' -v -timeout 90m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type v10FilmSpec struct{ short8, mapKey string }

// TestV10MasqueDeadState recense les MASQUES des records `ti=40` acceptes par le balayage
// offline : combien declarent i11, et quels composants les precedent. Aucune grammaire n'est
// exercee ici — c'est un recensement de masques, donc insensible a toute erreur de curseur.
func TestV10MasqueDeadState(t *testing.T) {
	films := v10ParseFilms(t)
	root := v10Root()
	cat := v10LoadBounds(t)

	release := LockProcessDecode()
	defer release()

	totalRec, totalI11 := 0, 0
	patterns := map[string]int{}
	precede := map[int]int{} // index de composant present AVANT i11 -> nb de records
	occur := map[int]int{}   // index de composant -> nb de records ti=40 qui le declarent
	sizes := map[int]int{}   // taille du masque -> nb de records
	for _, f := range films {
		entry, err := cat.Lookup(f.mapKey)
		if err != nil {
			t.Fatalf("%s : bornes de %q introuvables : %v", f.short8, f.mapKey, err)
		}
		dir := filepath.Join(root, "film_chunks", f.short8)
		kf := ScanFilmWorldObjectKeyframes(dir, VehicleTypeIndex)
		if len(kf.Band) == 0 {
			t.Fatalf("%s : aucun slot ti=%d aux images-cles", f.short8, VehicleTypeIndex)
		}
		nRec, nI11 := 0, 0
		prev := recordMaskHook
		SetRecordMaskHook(func(idx []int, _ []byte, _ int) {
			nRec++
			sizes[len(idx)]++
			has11 := false
			for _, id := range idx {
				occur[id]++
				if id == 11 {
					has11 = true
				}
			}
			if !has11 {
				return
			}
			nI11++
			patterns[v10Key(idx)]++
			for _, id := range idx {
				if id < 11 && id > 0 {
					precede[id]++
				}
			}
		})
		opt := ScanFilmOptions{RequireTag1: false, DropSaturated: true, CaptureDirs: true,
			QuantaOnly: true, DynPrecOrientation: true}
		lay := entry.Layout()
		if lay.Valid() {
			opt.Layout = &lay
		}
		_, err = ScanFilmBipedPositionsForBand(dir, kf.Band, opt)
		SetRecordMaskHook(prev)
		if err != nil {
			t.Fatalf("%s : balayage : %v", f.short8, err)
		}
		totalRec += nRec
		totalI11 += nI11
		t.Logf("V10 %s — %d records ti=40 acceptes · %d portent i11 (%.2f %%)",
			f.short8, nRec, nI11, 100*v10Frac(nI11, nRec))
		v10ControlBiped(t, dir, f.short8, entry)
	}

	t.Logf("\n########## MASQUES ti=40 PORTANT i11 — %d / %d records (%.2f %%) ##########",
		totalI11, totalRec, 100*v10Frac(totalI11, totalRec))
	occKeys := make([]int, 0, len(occur))
	for k := range occur {
		occKeys = append(occKeys, k)
	}
	sort.Ints(occKeys)
	for _, k := range occKeys {
		t.Logf("  [TOUS RECORDS] i%-2d declare par %d / %d records (%.2f %%)",
			k, occur[k], totalRec, 100*v10Frac(occur[k], totalRec))
	}
	szKeys := make([]int, 0, len(sizes))
	for k := range sizes {
		szKeys = append(szKeys, k)
	}
	sort.Ints(szKeys)
	for _, k := range szKeys {
		t.Logf("  [TAILLE MASQUE] %d composants : %d records", k, sizes[k])
	}
	keys := make([]int, 0, len(precede))
	for k := range precede {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		t.Logf("  composant i%-2d present avant i11 dans %d / %d records porteurs (%.0f %%)",
			k, precede[k], totalI11, 100*v10Frac(precede[k], totalI11))
	}
	type pc struct {
		p string
		n int
	}
	var ps []pc
	for p, n := range patterns {
		ps = append(ps, pc{p, n})
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].n > ps[j].n })
	for i, p := range ps {
		if i >= 20 {
			t.Logf("  ... et %d autres motifs", len(ps)-20)
			break
		}
		t.Logf("  motif %-40s %d records", p.p, p.n)
	}
}

// v10ControlBiped est LE TEMOIN QUI DECIDE DE LA PORTEE DU RECENSEMENT ci-dessus. Le balayage
// ancre n'accepte qu'un record dont le masque COMMENCE par i0 et dont i0 est ABSOLU : s'il ne
// voyait JAMAIS d'i11, y compris sur le BIPEDE dont la mort est un fait etabli du corpus (le
// dead-state lourd resout l'arme du kill), alors « 0 record ti=40 porte i11 » ne dirait rien du
// vehicule et tout de l'ancre. Ce temoin mesure donc le meme i11 sur la bande `ti=35` du MEME
// film, avec la MEME ancre.
func v10ControlBiped(t *testing.T, dir, short8 string, entry MapQuantEntry) {
	nRec, nI11 := 0, 0
	occur := map[int]int{}
	prev := recordMaskHook
	SetRecordMaskHook(func(idx []int, _ []byte, _ int) {
		nRec++
		for _, id := range idx {
			occur[id]++
			if id == 11 {
				nI11++
			}
		}
	})
	opt := ScanFilmOptions{RequireTag1: true, DropSaturated: true, CaptureDirs: true, QuantaOnly: true}
	lay := entry.Layout()
	if lay.Valid() {
		opt.Layout = &lay
	}
	_, err := ScanFilmBipedPositions(dir, opt)
	SetRecordMaskHook(prev)
	if err != nil {
		t.Logf("    [CONTROLE bipede %s] balayage impossible : %v", short8, err)
		return
	}
	keys := make([]int, 0, len(occur))
	for k := range occur {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var top []string
	for _, k := range keys {
		if occur[k]*200 >= nRec { // >= 0,5 % des records
			top = append(top, fmt.Sprintf("i%d:%d", k, occur[k]))
		}
	}
	t.Logf("    [CONTROLE bipede %s] %d records ti=35 acceptes · %d portent i11 (%.2f %%) · composants >= 0,5 %% : %s",
		short8, nRec, nI11, 100*v10Frac(nI11, nRec), strings.Join(top, " "))
}

func v10Key(idx []int) string {
	parts := make([]string, 0, len(idx))
	for _, id := range idx {
		parts = append(parts, fmt.Sprint(id))
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func v10Frac(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func v10ParseFilms(t *testing.T) []v10FilmSpec {
	raw := os.Getenv("V10_FILMS")
	if raw == "" {
		t.Skipf("V10_FILMS absent : instrument dead-state saute")
	}
	var out []v10FilmSpec
	for _, tok := range strings.Split(raw, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		i := strings.Index(tok, ":")
		if i < 0 {
			t.Fatalf("V10_FILMS : entree %q sans ':'", tok)
		}
		out = append(out, v10FilmSpec{strings.TrimSpace(tok[:i]), strings.TrimSpace(tok[i+1:])})
	}
	if len(out) == 0 {
		t.Skipf("V10_FILMS vide")
	}
	return out
}

func v10Root() string {
	if r := os.Getenv("V10_FILM_ROOT"); r != "" {
		return r
	}
	return `C:\Users\Guillaume\Projects\LevelUp\data\cache`
}

func v10LoadBounds(t *testing.T) *MapQuantCatalog {
	path := os.Getenv("V10_BOUNDS")
	if path == "" {
		path = `C:\Users\Guillaume\Projects\LevelUp\data\titles\halo_infinite\reference\map_quant_bounds.json`
	}
	cat, err := LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible : %v", err)
	}
	return cat
}
