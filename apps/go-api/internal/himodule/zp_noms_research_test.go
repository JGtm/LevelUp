//go:build research

// zp_noms_research_test.go — SONDE : les NOMS des proprietes reseau `ti=13 i0` (identifiants de
// chaine R(32)) sont-ils l'empreinte murmur3 (graine 0, chaine normalisee, cf.
// `objectif_bombstate_test.go` cote grammaire) d'une chaine en clair des scripts Lua (`hsc*`) du
// jeu installe ?
//
// La sonde hache chaque suite imprimable (>= 2 caracteres) de chaque script, telle quelle et
// normalisee, et rend celles dont l'empreinte est une des cibles. Temoin positif : les VALEURS
// d'identifiant de chaine (`round_result_reason_win`...) s'y retrouvent ; resultat sur les noms des
// blocs de zone du rejeu (`replay/zone_states_owner_nom.go`) au journal du lot (2026-10-07).
//
// Lecture seule des modules installes, aucun fichier ecrit. `ZP_MOT` (facultatif) liste en plus les
// scripts dont une chaine contient ce mot.
//
//	ZP_DEPLOY=<...>/Halo Infinite/deploy ZP_CIBLES="904941267=proprietaireA,..." go test \
//	  -tags=research -count=1 -v -run '^TestZPNoms$' ./internal/himodule/
package himodule_test

import (
	"math/bits"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/himodule"
)

func zpMM3(data []byte) uint32 {
	var h uint32
	const c1, c2 = 0xcc9e2d51, 0x1b873593
	n := len(data) / 4
	for i := 0; i < n; i++ {
		k := uint32(data[i*4]) | uint32(data[i*4+1])<<8 | uint32(data[i*4+2])<<16 | uint32(data[i*4+3])<<24
		k *= c1
		k = bits.RotateLeft32(k, 15)
		k *= c2
		h ^= k
		h = bits.RotateLeft32(h, 13)
		h = h*5 + 0xe6546b64
	}
	tail := data[n*4:]
	var k uint32
	switch len(tail) {
	case 3:
		k ^= uint32(tail[2]) << 16
		fallthrough
	case 2:
		k ^= uint32(tail[1]) << 8
		fallthrough
	case 1:
		k ^= uint32(tail[0])
		k *= c1
		k = bits.RotateLeft32(k, 15)
		k *= c2
		h ^= k
	}
	h ^= uint32(len(data))
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

func zpNorm(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "_")
	return strings.ReplaceAll(s, " ", "_")
}

// zpRuns rend les suites imprimables de longueur >= 2.
func zpRuns(data []byte) []string {
	var out []string
	start := -1
	for i := 0; i <= len(data); i++ {
		ok := i < len(data) && data[i] >= 0x20 && data[i] < 0x7f
		if ok && start < 0 {
			start = i
		}
		if !ok && start >= 0 {
			if i-start >= 2 {
				out = append(out, string(data[start:i]))
			}
			start = -1
		}
	}
	return out
}

// TestZPNoms cherche les cibles parmi les chaines des scripts Lua.
func TestZPNoms(t *testing.T) {
	deploy := os.Getenv("ZP_DEPLOY")
	if deploy == "" {
		t.Skip("ZP_DEPLOY requis")
	}
	cibles := map[uint32]string{}
	for _, c := range strings.Split(os.Getenv("ZP_CIBLES"), ",") {
		v, lbl, ok := strings.Cut(c, "=")
		if !ok {
			continue
		}
		n, err := strconv.ParseUint(v, 10, 32)
		if err == nil {
			cibles[uint32(n)] = lbl
		}
	}
	if zpMM3([]byte("bombobjectstate")) != 0x19813E20 {
		t.Fatal("murmur3 faux")
	}
	mot := strings.ToLower(os.Getenv("ZP_MOT"))
	var mods []string
	_ = filepath.Walk(filepath.Join(deploy, "any"), func(p string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(p, ".module") {
			mods = append(mods, p)
		}
		return nil
	})
	sort.Strings(mods)
	trouves := map[string]map[string]bool{}
	tags, mots := 0, 0
	for _, p := range mods {
		m, err := himodule.Open(p)
		if err != nil {
			t.Logf("%s : %v", filepath.Base(p), err)
			continue
		}
		for _, f := range m.Files("hsc*") {
			data, err := m.Extract(f)
			if err != nil {
				continue
			}
			tags++
			mentionne := false
			for _, r := range zpRuns(data) {
				mots++
				if mot != "" && strings.Contains(strings.ToLower(r), mot) {
					mentionne = true
				}
				for _, s := range []string{r, zpNorm(r)} {
					if lbl, ok := cibles[zpMM3([]byte(s))]; ok {
						if trouves[lbl] == nil {
							trouves[lbl] = map[string]bool{}
						}
						trouves[lbl][s+" @"+filepath.Base(p)+"/"+strconv.FormatUint(uint64(f.GlobalID), 16)] = true
					}
				}
			}
			if mentionne {
				t.Logf("hsc* %08x (%s) mentionne %q, %d octets", f.GlobalID, filepath.Base(p), mot, len(data))
			}
		}
		_ = m.Close()
	}
	t.Logf("%d modules, %d tags hsc*, %d chaines", len(mods), tags, mots)
	lbls := make([]string, 0, len(trouves))
	for l := range trouves {
		lbls = append(lbls, l)
	}
	sort.Strings(lbls)
	for _, l := range lbls {
		for s := range trouves[l] {
			t.Logf("TROUVE %s = %s", l, s)
		}
	}
}
