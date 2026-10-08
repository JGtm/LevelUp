//go:build research

package grammar

// ri27d1_fermeture_portee_research_test.go — 2.7.d1 (`.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`,
// §1.2, commande I-ferm) : la sonde de fermeture d image-cle par archetype (`KeyframeClosure`) sur les
// films de l instrument 2.7.d0, dans le contexte de la cuisson. Un seul mode : la marche de production
// a la tete (etiquette `tete`) ; la reference de chaque etape du plan dit a quelles lignes la comparer.
//
//	RI27C_FILMS=<id,...> RI27C_RACINE=<film_chunks> RI27C_OUT=<dossier> [RI27C_CARTES=<id=Carte;...>] \
//	  go test -tags=research -count=1 -run '^TestRI27d1FermetureCorpus$' ./...grammar/

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// TestRI27d1FermetureCorpus : [KeyframeClosure] par film, dans le contexte de la cuisson (carte posee,
// MPP declare : [ri27cContexte]). Sortie : fermeture_corpus.tsv — mode, film, archetype, fermes, total,
// bloquant.
func TestRI27d1FermetureCorpus(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	var lignes []string
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		st, err := KeyframeClosure(fc)
		if err != nil {
			t.Fatalf("%s : %v", court, err)
		}
		tis := make([]int, 0, len(st))
		for ti := range st {
			tis = append(tis, int(ti))
		}
		sort.Ints(tis)
		for _, ti := range tis {
			s := st[uint32(ti)] //nolint:gosec // cle uint32
			lignes = append(lignes, fmt.Sprintf("tete\t%s\t%d\t%d\t%d\t%s", court, ti, s.Closed, s.Total, s.Blocking))
		}
		runtime.GC()
	}
	ri27cEcrire(t, filepath.Join(sortie, "fermeture_corpus.tsv"), lignes)
}

// TestRI27d1Formats : la version de format de chaque film de RI27C_FILMS (journal seulement).
func TestRI27d1Formats(t *testing.T) {
	films, racine, _ := ri27cEnv(t)
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, false)
		v, ok := FilmFormatVersion(fc.film)
		t.Logf("FORMAT %s %d %v", court, v, ok)
	}
}
