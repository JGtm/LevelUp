package filmdec

// r7_recensement_research_test.go — lot R7 : le RECENSEMENT des types en TETE de liste, sur
// le parc des films temoins. Il n'etablit rien a lui seul : il ORDONNE le travail Ghidra
// (quels lecteurs decompiler d'abord) et fournit le denominateur « paquets delta » de toutes
// les mesures du lot.
//
// LECTURE SEULE, skip par defaut, CGO_ENABLED=0, balayage borne au parc R7_IDS.
//
//	CGO_ENABLED=0 R7_ROOT=<repo>/data/cache/film_chunks R7_IDS=a,b,c \
//	  go test ./internal/analysis/filmdec/ -run '^TestR7Recensement$' -count=1 -timeout 30m -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	r7RootEnv = "R7_ROOT"
	r7IDsEnv  = "R7_IDS"
	r7ArtsEnv = "R7_ARTS"
	r7CatEnv  = "R7_CAT"
	r7MapsEnv = "R7_MAPS"
)

// r7Films rend la liste des films demandes, ou skip l'instrument.
func r7Films(t *testing.T) (string, []string) {
	t.Helper()
	root, ids := os.Getenv(r7RootEnv), os.Getenv(r7IDsEnv)
	if root == "" || ids == "" {
		t.Skipf("instrument R7 : definir %s et %s", r7RootEnv, r7IDsEnv)
	}
	var out []string
	for _, id := range strings.Split(ids, ",") {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	return root, out
}

// r7TeteType rend le type de l'evenement de tete d'un paquet delta, ou -1 si la liste est
// vide / le paquet trop court. Grammaire : [1 bit config][1 bit continuation][R(7) type].
func r7TeteType(pay []byte) int {
	if len(pay) < 2 || pay[0]&0xC0 != 0xC0 {
		return -1
	}
	return int(pay[0]&0x3F)<<1 | int(pay[1]>>7)
}

// TestR7Recensement compte les types de tete par film et sur le parc.
func TestR7Recensement(t *testing.T) {
	root, ids := r7Films(t)
	parc := map[int]int{}
	totalDelta, totalListeVide := 0, 0
	for _, id := range ids {
		dir := filepath.Join(root, id)
		n := CountFilmChunks(dir)
		if n == 0 {
			t.Logf("film %s : aucun chunk — ignore", id)
			continue
		}
		film := map[int]int{}
		nd, nv := 0, 0
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeDelta || pk.Size < 2 {
					continue
				}
				nd++
				typ := r7TeteType(pk.Payload(data))
				if typ < 0 {
					nv++
					continue
				}
				film[typ]++
			}
		}
		totalDelta += nd
		totalListeVide += nv
		for k, v := range film {
			parc[k] += v
		}
		t.Logf("film %s : %d paquets delta, %d listes vides, %d types de tete", id, nd, nv, len(film))
	}
	t.Logf("")
	t.Logf("=== PARC : %d paquets delta, %d listes vides (%.1f %%) ===",
		totalDelta, totalListeVide, 100*float64(totalListeVide)/float64(max(1, totalDelta)))
	type kv struct {
		typ, n int
	}
	var l []kv
	for k, v := range parc {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	for i, e := range l {
		t.Logf("  %2d. type %3d %-38s %8d  (%.2f %%)", i+1, e.typ, r7Noms[e.typ], e.n,
			100*float64(e.n)/float64(max(1, totalDelta)))
	}
	var absents []string
	for typ := 0; typ < 123; typ++ {
		if parc[typ] == 0 {
			absents = append(absents, fmt.Sprintf("%d", typ))
		}
	}
	t.Logf("  types SANS aucune tete sur le parc (%d) : %s", len(absents), strings.Join(absents, ","))
}
