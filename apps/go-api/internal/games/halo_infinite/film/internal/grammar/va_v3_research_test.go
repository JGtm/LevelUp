//go:build research

package grammar

// va_v3_research_test.go — LOT VA, ETAPE V3 (2026-10-06) : la marche de PRODUCTION du corps de
// `chunk_00` ([marcherLeCorps]) sur les films d un cache, comparee a la table des joueurs
// ([ReadPlayerTable]), et ce que la variante declare. Instrument de recherche. Rejouable (lecture
// seule ; CAMPAGNE_FILMS=TOUS : tout le cache) :
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id|TOUS> [V3_SORTIE=<fichier hors data>] \
//	  go test -tags=research -count=1 -run ^TestVAV3Corps$ ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestVAV3Corps(t *testing.T) {
	racine := os.Getenv("CAMPAGNE_RACINE")
	var ids []string
	if f := os.Getenv("CAMPAGNE_FILMS"); f != "TOUS" {
		ids = strings.Split(f, ",")
	} else {
		es, _ := os.ReadDir(racine)
		for _, e := range es {
			if e.IsDir() {
				ids = append(ids, e.Name())
			}
		}
	}
	compte := map[string]int{}
	var out []string
	for _, id := range ids {
		film, err := source.LoadDir(filepath.Join(racine, id), nil)
		if err != nil {
			compte["erreur"]++
			continue
		}
		reg, ok := FilmRegistryChunk(film)
		if !ok {
			compte["sans_chunk_00"]++
			continue
		}
		ident, err := ReadFilmIdentity(reg)
		if err != nil {
			compte["sans_identite"]++
			continue
		}
		v, fin := marcherLeCorps(reg, ident.BodyBit)
		_, rep, perr := ReadPlayerTable(reg, ident)
		ferme := "table_illisible"
		if perr == nil {
			ferme = fmt.Sprint(fin == rep.FirstRecordBit)
		}
		if v != ident.Variante {
			ferme += "_IDENTITE_DIFFERENTE"
		}
		k := fmt.Sprintf("%s ferme=%s lue=%v presente=%v moteur=%d killcam=%v potg=%v", ident.Build, ferme, v.Lue, v.Presente, v.TypeDeMoteur, v.KillcamEnabled, v.PlayOfTheGameEnabled)
		compte[k]++
		out = append(out, id+"\t"+k)
	}
	if s := os.Getenv("V3_SORTIE"); s != "" {
		_ = os.WriteFile(s, []byte(strings.Join(out, "\n")+"\n"), 0o644)
	}
	var cles []string
	for k := range compte {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	for _, k := range cles {
		t.Logf("%6d  %s", compte[k], k)
	}
}
