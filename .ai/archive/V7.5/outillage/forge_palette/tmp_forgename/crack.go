package main

// crack.go — recherche exhaustive d'un nom snake_case dont le murmur3_x86_32(seed=0)
// retombe sur un hash observe. C'est la methode qui a peuple `objectives.go`.
//
// GARDE-FOU, repris tel quel de ce fichier : une telle recherche produit des
// collisions fortuites. L'esperance est calculee et imprimee ; seules les
// correspondances SEMANTIQUEMENT coherentes avec le domaine Halo doivent etre
// retenues. Un hash non explique reste INCONNU — on ne devine pas de libelle.
//
//	tmp_forgename crack <fichier_cibles> <fichier_vocabulaire> [profondeur=3]

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

func readLines(path string) []string {
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		for _, tok := range strings.Fields(sc.Text()) {
			tok = strings.TrimSpace(tok)
			if tok != "" && !strings.HasPrefix(tok, "#") {
				out = append(out, tok)
			}
		}
	}
	return out
}

func cmdCrack(targetFile, vocabFile string, depth int) {
	targets := map[int32]bool{}
	for _, t := range readLines(targetFile) {
		if v, err := strconv.ParseInt(t, 10, 64); err == nil {
			targets[int32(v)] = true
		}
	}
	vocab := readLines(vocabFile)
	fmt.Fprintf(os.Stderr, "cibles : %d, vocabulaire : %d mots, profondeur : %d\n",
		len(targets), len(vocab), depth)

	tried := 0
	hits := 0
	var rec func(prefix string, level int)
	rec = func(prefix string, level int) {
		for _, w := range vocab {
			s := w
			if prefix != "" {
				s = prefix + "_" + w
			}
			tried++
			if targets[mapvar.LabelHash(s)] {
				hits++
				fmt.Printf("%-12d %s\n", mapvar.LabelHash(s), s)
			}
			if level < depth {
				rec(s, level+1)
			}
		}
	}
	rec("", 1)
	// Esperance de collisions fortuites : essais x cibles / 2^32.
	exp := float64(tried) * float64(len(targets)) / 4294967296.0
	fmt.Fprintf(os.Stderr, "essais : %d, correspondances : %d, collisions fortuites attendues : %.3f\n",
		tried, hits, exp)
}
