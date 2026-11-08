//go:build research

package replay

// fin_de_vie_temoin_research_test.go — LE TEMOIN REEL DE LA FIN DE VIE A LA MORT ECRITE : le match
// `20c703a7`, slot 579 (Like15Ninjaz), que le film cesse d'ecrire a la frame 3920 et dont il ecrit
// la mort a la frame 3957 (400 537 ms du match, origine 4 735 ms, pas de 100 ms).
//
// LECTURE SEULE d'un fichier de faits persiste (aucun film, aucune ecriture) :
//
//	FIN_DE_VIE_FAITS=<depot>/data/cache/film_facts/halo_infinite go test -tags research -count=1 \
//	  -run '^TestFinDeVieTemoin20c703a7$' -v ./replay/
//
// Le catalogue des bornes de cartes est celui du depot hote (`racineDepot`) ; a defaut, celui de
// FIN_DE_VIE_CATALOGUE.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/profile"
)

func TestFinDeVieTemoin20c703a7(t *testing.T) {
	faits := os.Getenv("FIN_DE_VIE_FAITS")
	if faits == "" {
		t.Skip("temoin sur faits reels : FIN_DE_VIE_FAITS requis")
	}
	catalogue := os.Getenv("FIN_DE_VIE_CATALOGUE")
	if catalogue == "" {
		catalogue = filepath.Join(racineDepot(t), "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json")
	}
	cat, err := profile.LoadMapQuantCatalog(catalogue)
	if err != nil {
		t.Fatal(err)
	}
	blob, err := os.ReadFile(filepath.Join(faits, "20c703a7.filmfacts.bin")) //nolint:gosec // temoin
	if err != nil {
		t.Skipf("faits absents : %v", err)
	}
	tete, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatal(err)
	}
	var entry profile.MapQuantEntry
	for _, e := range cat.Maps {
		if e.Module == tete.MapModule {
			entry = e
		}
	}
	f, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatal(err)
	}
	doc := BuildFromFacts(context.Background(), "20c703a7", "halo_infinite", f, Options{MapQuant: &entry})

	var trouvee bool
	for i, tr := range doc.Tracks {
		if tr.Slot == 579 && tr.XUID == "2533274865072358" {
			trouvee = true
			if dernier := tr.Points[len(tr.Points)-1].T; dernier != 3920 || tr.EndFrame != 3957 {
				t.Errorf("slot 579 : dernier point %d, fin %d, attendu 3920 et 3957", dernier, tr.EndFrame)
			}
		}
		// UNE VIE NE CHEVAUCHE JAMAIS LA SUIVANTE DU MEME SLOT.
		for _, suiv := range doc.Tracks[i+1:] {
			if suiv.Slot == tr.Slot && suiv.StartFrame <= tr.EndFrame && suiv.StartFrame > tr.StartFrame {
				t.Errorf("slot %d : la vie [%d,%d] recouvre la suivante, ouverte a %d",
					tr.Slot, tr.StartFrame, tr.EndFrame, suiv.StartFrame)
			}
		}
	}
	if !trouvee {
		t.Fatal("la vie du slot 579 n'est pas publiee")
	}
}
