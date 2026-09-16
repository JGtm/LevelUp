package grammar

// ecs_table_level_gate_test.go — LA PORTE DE REGENERATION DE LA COLONNE `level`.
//
// Elle vit a part de `ecs_table_guard_test.go` parce que ce fichier tient QUATRE garde-rails et
// frolait le seuil de 500 lignes : une porte de regeneration est un outil, pas un controle.

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"testing"
)

// updateNiveauxECS : LA PORTE DE REGENERATION DE LA SEULE COLONNE `level`, ET D AUCUNE AUTRE.
//
// POURQUOI ELLE EXISTE (lot 1.2, 2026-09-14). Le reste de la table est ECRIT A LA MAIN — statut
// de portage, adresse de deser, sens, confiance — et doit le rester. La colonne `level`, elle,
// est une DONNEE DU FILM : la recopier a la main, c'est se donner une chance de la recopier
// faux, et c'est exactement ce qui est arrive quand le cadrage du registre a change (178 lignes
// annotees `niveau_jeu=N`, 189 lignes reellement decalees — 11 silencieuses).
//
// ELLE REECRIT PUIS ECHOUE, comme les autres portes du paquet : une regeneration ne rend jamais
// `ok`, sans quoi elle se confond avec un run vert.
//
//	ECS_TABLE_FILM=../killsource/testdata/minibobine_000d5950 \
//	  go test ./internal/games/halo_infinite/film/filmdec/ -run G2 -update-ecs-table-level
var updateNiveauxECS = flag.Bool("update-ecs-table-level", false,
	"reecrire la colonne `level` de testdata/ecs_table.tsv depuis le registre du PREMIER film "+
		"de ECS_TABLE_FILM — CETTE colonne seulement")

// reecrireNiveauxECS remet la colonne `level` de chaque ligne de registre (ti >= 0) a la valeur
// que le film donne, et NE TOUCHE A RIEN D AUTRE : ni les alias (ti = -1, level = -1), ni une
// ligne absente du registre de ce film — elle serait signalee par le controle ordinaire.
func reecrireNiveauxECS(t *testing.T, got map[string]uint32, film string) {
	t.Helper()
	raw, err := os.ReadFile(ecsTablePath)
	if err != nil {
		t.Fatalf("table ECS illisible : %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	change, absents := 0, 0
	for n := 1; n < len(lines); n++ {
		c := strings.Split(lines[n], "\t")
		if len(c) != ecsTableColumns || strings.HasPrefix(c[0], "-") {
			continue
		}
		lv, ok := got[c[0]+"|"+c[2]+"|"+c[3]]
		if !ok {
			absents++
			continue
		}
		if v := strconv.FormatUint(uint64(lv), 10); v != c[4] {
			c[4] = v
			lines[n] = strings.Join(c, "\t")
			change++
		}
	}
	if err := os.WriteFile(ecsTablePath, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("ecriture de la table : %v", err)
	}
	t.Fatalf("colonne `level` reecrite depuis %s : %d ligne(s) changee(s), %d ligne(s) absente(s) "+
		"du registre de ce film (laissees telles quelles) ; relancer sans -update-ecs-table-level "+
		"pour verifier", film, change, absents)
}
