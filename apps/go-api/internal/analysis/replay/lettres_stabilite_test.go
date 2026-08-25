package replay

// lettres_stabilite_test.go — LA CLAUSE DE STABILITE du lot « lettres A/B/C des bases » :
// les films d'une MEME carte rendent-ils la MEME permutation zone -> rang de lettre ?
//
// CE TEST NE DECODE AUCUN FILM. Il relit les TSV que `TestLettresOrdreSlots` ecrit film par
// film, donc il tourne en une fraction de seconde, se rejoue sans re-payer un balayage, et ne
// peut rien faire exploser en memoire. C'est le MEME partage que `zone_state_p2a_test.go` et
// `zone_state_p2a_stabilite_test.go` du lot C-bis, et il vaut ici pour la meme raison de fond :
// la comparaison inter-match n'a aucune raison de vivre dans le processus qui decode. C'est
// aussi ce qui garde chacun des deux fichiers sous le seuil du depot (500 lignes) — la scission
// suit une frontiere reelle, elle ne decoupe pas pour decouper.
//
// USAGE (depuis apps/go-api, une fois les films mesures) :
//
//	$env:LETTRES_OUT="<worktree>/.ai/V7.5/replay2d/registre_film/lotLettres"
//	go test -count=1 -run TestLettresOrdreStabilite -v ./internal/analysis/replay/

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestLettresOrdreStabilite compare les permutations des films d'une MEME carte.
//
// CE TEST NE DECODE AUCUN FILM — il relit les TSV produits film par film, donc il tourne en une
// fraction de seconde et ne peut rien faire exploser. Meme patron que
// `TestZoneEtatPhase2aStabilite` : la comparaison inter-match n'a aucune raison de vivre dans le
// processus qui decode.
func TestLettresOrdreStabilite(t *testing.T) {
	dir := os.Getenv(lettresOutEnv)
	if dir == "" {
		t.Skipf("comparaison non demandee : %s vide", lettresOutEnv)
	}
	byMap := lettresLitMesures(t, dir)
	if len(byMap) == 0 {
		t.Skipf("aucune mesure sous %s — jouer TestLettresOrdreSlots film par film d'abord", dir)
	}
	ids := make([]string, 0, len(byMap))
	for k := range byMap {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	comparees, stables := 0, 0
	for _, id := range ids {
		rows := byMap[id]
		if len(rows) < 2 {
			t.Logf("  CARTE %-14s : un seul film mesure (%s) — non comparable", rows[0].carte,
				rows[0].short)
			continue
		}
		comparees++
		if lettresMemePermutation(t, rows) {
			stables++
		}
	}
	t.Logf("VERDICT — %d cartes a >= 2 films mesures, %d rendent la MEME permutation", comparees,
		stables)
	if comparees > 0 && stables < comparees {
		t.Errorf("%d carte(s) INSTABLE(S) : le fallback des lettres ne tient pas en l'etat",
			comparees-stables)
	}
}

// lettresLigne est une ligne de mesure relue d'un TSV.
type lettresLigne struct {
	short, carte, mapID, perm, slots string
}

// lettresLitMesures relit toutes les mesures d'un repertoire, groupees par carte.
func lettresLitMesures(t *testing.T, dir string) map[string][]lettresLigne {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("repertoire de mesures illisible (%s) : %v", dir, err)
	}
	out := map[string][]lettresLigne{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_lettres.tsv") {
			continue
		}
		blob, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("mesure illisible (%s) : %v", e.Name(), err)
		}
		for _, line := range strings.Split(string(blob), "\n") {
			f := strings.Split(strings.TrimRight(line, "\r"), "\t")
			if len(f) < 28 || f[0] != "film" {
				continue
			}
			out[f[3]] = append(out[f[3]],
				lettresLigne{short: f[1], carte: f[2], mapID: f[3], perm: f[27], slots: f[29]})
		}
	}
	for id := range out {
		rows := out[id]
		sort.Slice(rows, func(i, j int) bool { return rows[i].short < rows[j].short })
	}
	return out
}

// lettresMemePermutation dit si tous les films COMPLETS d'une carte rendent la meme permutation.
func lettresMemePermutation(t *testing.T, rows []lettresLigne) bool {
	t.Helper()
	ref, refShort, complets, identiques := "", "", 0, 0
	for _, r := range rows {
		if r.perm == "INCOMPLETE" {
			t.Logf("    %s %-10s : permutation INCOMPLETE (aucune lettre publiee) · %s", r.carte,
				r.short, r.slots)
			continue
		}
		complets++
		switch {
		case ref == "":
			ref, refShort, identiques = r.perm, r.short, 1
		case r.perm == ref:
			identiques++
		default:
			t.Errorf("    %s %s : permutation %s, mais %s rend %s — ORDRE INSTABLE", r.carte,
				r.short, r.perm, refShort, ref)
		}
		t.Logf("    %s %-10s : %s · %s", r.carte, r.short, r.perm, r.slots)
	}
	verdict := complets >= 2 && identiques == complets
	t.Logf("  CARTE %-14s : %d films, %d permutations completes, %d identiques — stable=%v",
		rows[0].carte, len(rows), complets, identiques, verdict)
	return verdict
}
