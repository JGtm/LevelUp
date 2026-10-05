//go:build research

package killsource

// va_v2_profil_research_test.go — LOT VA, ETAPE V2, CORRECTIONS DU CONTROLE (2026-10-05) : LE
// PROFIL DE BALAYAGE QUE LA CUISSON DU REJEU PORTE, ECRIT EN JSON.
//
// L etape `vehicles` du rejeu marche les morts sous le profil que `killsource` calibre sur le film
// (`Result.ProfilCalibre`, pose par `replay.poserProfilPuisCarte` : generation stricte, traversee,
// largeur d axe absolue). La sonde de la marche des morts (`grammar/va_v2_corr_research_test.go`)
// le relit pour se placer dans le contexte de l etape publiee.
//
//	VA_RACINE=<film_chunks> VA_CATALOGUE=<map_quant_bounds.json> VA_CARTES="id=Carte;..." \
//	VA_SORTIE=<dir hors data> go test -tags=research -count=1 -run '^TestVAV2ProfilCalibre$' ./internal/games/halo_infinite/film/internal/facts/killsource/

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestVAV2ProfilCalibre ecrit `<id>.profil.json` par film de VA_CARTES.
func TestVAV2ProfilCalibre(t *testing.T) {
	racine, sortie := os.Getenv("VA_RACINE"), os.Getenv("VA_SORTIE")
	if racine == "" || sortie == "" {
		t.Skip("VA_RACINE, VA_CATALOGUE, VA_CARTES et VA_SORTIE requis")
	}
	cat, err := profile.LoadMapQuantCatalog(os.Getenv("VA_CATALOGUE"))
	if err != nil {
		t.Fatal(err)
	}
	for _, kv := range strings.Split(os.Getenv("VA_CARTES"), ";") {
		id, nom, _ := strings.Cut(kv, "=")
		e, err := cat.Lookup(nom)
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		src, err := source.LoadDir(filepath.Join(racine, id), nil)
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		opts := DefaultOptions()
		opts.Carte = &e
		res, err := Decode(context.Background(), id, src, &opts)
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		blob, err := json.MarshalIndent(res.ProfilCalibre, "", " ")
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		if err := os.WriteFile(filepath.Join(sortie, id+".profil.json"), blob, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
