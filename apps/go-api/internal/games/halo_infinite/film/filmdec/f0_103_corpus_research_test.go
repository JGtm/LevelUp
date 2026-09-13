package filmdec

// f0_103_corpus_research_test.go — LOT F.0 : LE RECENSEMENT DU PARC D ARTEFACTS.
//
// La question 4 porte sur le CHAMP DE REPARATION, le capteur et le traqueur. Encore faut-il
// des films qui en portent une consommation de charge exploitable : sur les dix films imposes
// par le plan, il n y en a AUCUNE pour le champ de reparation. Ce recensement lit les
// artefacts SEULS (aucun film decode, quelques secondes) et dit, famille par famille, combien
// de consommations exploitables le parc porte et dans quels films — c est lui qui CHOISIT les
// films de la mesure, au lieu de conclure sur un corpus muet.
//
// LECTURE SEULE. Garde : `F0_ARTS` (racine des artefacts) et `F0_LABELS` (manifeste).
//
//	CGO_ENABLED=0 F0_ARTS=<depot>/data/cache/replays/halo_infinite \
//	  F0_LABELS=<depot>/config/titles/halo_infinite/mappings/replay_labels.toml \
//	  go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestF0CorpusSpent$' -v

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// f0CorpusFilms borne le nombre de films listes par famille dans le rapport.
const f0CorpusFilms = 12

// TestF0CorpusSpent recense les consommations de charge exploitables du parc d artefacts.
func TestF0CorpusSpent(t *testing.T) {
	dir := os.Getenv(f0ArtsEnv)
	if dir == "" {
		t.Skipf("instrument F.0 : definir %s", f0ArtsEnv)
	}
	manifeste := f0ManifesteRangs(t)
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("racine des artefacts illisible : %v", err)
	}
	parFamille := map[string]int{}
	parFilm := map[string]map[string]int{}
	films, sansPalette := 0, 0
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".json") || strings.Contains(nom, ".derived.") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, nom))
		if err != nil {
			continue
		}
		var a f0Art
		if err := json.Unmarshal(raw, &a); err != nil {
			continue
		}
		films++
		rangs := manifeste
		if len(a.AbilityLabels) > 0 {
			rangs = map[int]string{}
			for k, v := range a.AbilityLabels {
				var r int
				if _, err := fmt.Sscanf(k, "%d", &r); err == nil && v.Family != "" {
					rangs[r] = v.Family
				}
			}
		} else {
			sansPalette++
		}
		id := strings.TrimSuffix(nom, ".json")
		for _, c := range a.EquipmentChanges {
			if c.Kind != "spent" || c.Gap > 0 {
				continue
			}
			fam := rangs[c.From]
			if fam == "" {
				continue
			}
			parFamille[fam]++
			if parFilm[fam] == nil {
				parFilm[fam] = map[string]int{}
			}
			parFilm[fam][id]++
		}
	}
	t.Logf("parc : %d artefacts lus, %d sans table de palette (repli manifeste)", films, sansPalette)
	familles := make([]string, 0, len(parFamille))
	for f := range parFamille {
		familles = append(familles, f)
	}
	sort.Slice(familles, func(i, j int) bool { return parFamille[familles[i]] > parFamille[familles[j]] })
	for _, fam := range familles {
		t.Logf("  %-22s %4d consommations exploitables · meilleurs films : %s",
			fam, parFamille[fam], f0TopFilms(parFilm[fam]))
	}
}

// f0TopFilms rend les films les plus riches d une famille, « id:compte » separes par une
// virgule — directement reutilisable en `F0_IDS`.
func f0TopFilms(m map[string]int) string {
	type kv struct {
		k string
		v int
	}
	var l []kv
	for k, v := range m {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].v != l[j].v {
			return l[i].v > l[j].v
		}
		return l[i].k < l[j].k
	})
	var parts []string
	for i, e := range l {
		if i >= f0CorpusFilms {
			break
		}
		parts = append(parts, e.k+":"+strconv.Itoa(e.v))
	}
	return strings.Join(parts, ",")
}
