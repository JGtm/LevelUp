//go:build research

package replaybuild

// va_v2_etapes_research_test.go — LOT VA, ETAPE V2 (2026-10-05) : LES VALEURS DES ETAPES QUE
// `replay-equiv` HACHE, ECRITES EN JSON pour les instruire etape par etape. Un instrument de
// recherche : la cuisson est celle du harnais (`cmd/replay-equiv`, branche du decodage), un film a la
// fois, sous une racine factice. VA_FAITS (facultatif) : le repertoire des `<film>.facts.json` quand
// ce n est pas celui des references de `replay-equiv` (films du gate de corpus hors de ce corpus).
//
//	VA_RACINE=<racine factice> VA_FILMS=<id,id> VA_ETAPES=<etape,etape> VA_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestVAV2Etapes$' ./internal/replaybuild/

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// TestVAV2Etapes ecrit, par film de VA_FILMS, la valeur de chaque etape de VA_ETAPES.
func TestVAV2Etapes(t *testing.T) {
	racine, sortie := os.Getenv("VA_RACINE"), os.Getenv("VA_SORTIE")
	films, etapes := strings.Split(os.Getenv("VA_FILMS"), ","), strings.Split(os.Getenv("VA_ETAPES"), ",")
	if racine == "" || sortie == "" || films[0] == "" || etapes[0] == "" {
		t.Skip("VA_RACINE, VA_FILMS, VA_ETAPES et VA_SORTIE requis")
	}
	ctx := context.Background()
	cacheRoot := title.NewPathResolver(racine).CacheRootDir()
	repFaits := os.Getenv("VA_FAITS")
	if repFaits == "" {
		repFaits = filepath.Join(racine, "apps", "go-api", "internal", "games", "halo_infinite", "film", "replay",
			"testdata", "equivalence")
	}
	for _, film := range films {
		faits, err := ReadFactsFile(filepath.Join(repFaits, film+".facts.json"))
		if err != nil {
			t.Fatalf("%s : %v", film, err)
		}
		b, err := NewBuilder(ctx, racine, title.DefaultSlug)
		if err != nil {
			t.Fatalf("%s : %v", film, err)
		}
		b.WithObserver(func(step string, v any) {
			if !slices.Contains(etapes, step) {
				return
			}
			blob, err := vaJSON(v)
			if err != nil {
				t.Errorf("%s %s : %v", film, step, err)
				return
			}
			if err := os.WriteFile(filepath.Join(sortie, film+"."+step+".json"), blob, 0o600); err != nil {
				t.Errorf("%s %s : %v", film, step, err)
			}
		})
		b.SansFaitsPersistes()
		if _, err := b.BuildBytes(ctx, faits.MatchID, faits.MapNames, filmcache.ChunkDir(cacheRoot, film),
			faits.MatchFacts); err != nil {
			t.Fatalf("%s : %v", film, err)
		}
		t.Logf("%s : etapes ecrites", film)
	}
}

// vaJSON rend la valeur en JSON ; une structure que `encoding/json` refuse en bloc (cle de table non
// textuelle) est rendue champ par champ, chaque champ illisible remplace par son rendu `%+v`.
func vaJSON(v any) ([]byte, error) {
	if blob, err := json.MarshalIndent(v, "", " "); err == nil {
		return blob, nil
	}
	rv := reflect.Indirect(reflect.ValueOf(v))
	if rv.Kind() != reflect.Struct {
		return json.MarshalIndent(fmt.Sprintf("%+v", v), "", " ")
	}
	champs := map[string]json.RawMessage{}
	for i := range rv.NumField() {
		f := rv.Type().Field(i)
		if !f.IsExported() {
			continue
		}
		blob, err := json.Marshal(rv.Field(i).Interface())
		if err != nil {
			blob, _ = json.Marshal(fmt.Sprintf("%+v", rv.Field(i).Interface()))
		}
		champs[f.Name] = blob
	}
	return json.MarshalIndent(champs, "", " ")
}
