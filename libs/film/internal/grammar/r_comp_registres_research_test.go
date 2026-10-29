//go:build research

package grammar

// r_comp_registres_research_test.go — CHANTIER « comp » DE LA CAMPAGNE DE GRAMMAIRE (R-L3, R-HOM,
// 2026-10-02) : le REGISTRE de chaque film, ecrit tel quel (archetype, index, nom, niveau).
//
// Sert deux recherches : R-L3 (1) compare le registre `ti=0` / `ti=2` des builds HI_1_4_1,
// version-31 et version-33 a celui de HI_1_13_0 ; R-HOM part de l union des noms pour recenser,
// dans l executable, les composants HOMONYMES (meme nom, deux tables de composant).
// Instrument de recherche : aucun fichier de production touche, aucune sortie ne change.
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -run '^TestRCompRegistres$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"hash/fnv"
	"testing"
)

// TestRCompRegistres ecrit `r_comp_registres.tsv` (une ligne par composant) et
// `r_comp_registres_empreintes.tsv` (une empreinte par film et par archetype, noms + niveaux).
func TestRCompRegistres(t *testing.T) {
	racine, sortie, films := b2Env(t)
	lignes := []string{"film\tbuild\tti\ti\tnom\tniveau"}
	empreintes := []string{"film\tbuild\tti\tcomposants\tempreinte"}
	for _, id := range films {
		fc, _, _ := ContexteDeFilm(racine + "/" + id)
		if fc == nil {
			t.Errorf("%s : film illisible", id)
			continue
		}
		reg, err := fc.Registry()
		if err != nil {
			t.Errorf("%s : registre : %v", id, err)
			continue
		}
		build := cmBuild(fc)
		for _, a := range reg.Archetypes {
			h := fnv.New32a()
			for i, nom := range a.Components {
				lignes = append(lignes, fmt.Sprintf("%s\t%s\t%d\t%d\t%s\t%d", id, build, a.Index, i, nom, a.Level(i)))
				fmt.Fprintf(h, "%s/%d;", nom, a.Level(i))
			}
			empreintes = append(empreintes, fmt.Sprintf("%s\t%s\t%d\t%d\t%08x", id, build, a.Index,
				len(a.Components), h.Sum32()))
		}
		t.Logf("%s %s : %d archetypes", id, build, len(reg.Archetypes))
	}
	b2Ecrire(t, sortie, "r_comp_registres.tsv", lignes)
	b2Ecrire(t, sortie, "r_comp_registres_empreintes.tsv", empreintes)
}
