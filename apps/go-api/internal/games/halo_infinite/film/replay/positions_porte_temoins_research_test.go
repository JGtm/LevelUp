//go:build research

package replay

// positions_porte_temoins_research_test.go — LES TROIS TEMOINS DU LOT M1 DES RETOURS DU REJEU
// (2026-09-23), sur leurs FAITS PERSISTES : la porte des positions rejouee sur les films que
// l utilisateur a signales et que l annexe `RAPPORT_positions_limbe.md` a mesures. Les
// identifiants de match et les instants sont ICI, dans le test ; le code de production n en porte
// aucun.
//
// LECTURE SEULE de trois fichiers de faits (aucun film, aucune ecriture), sous la voie film :
//
//	M1_FAITS=<depot>/data/cache/film_facts/halo_infinite \
//	  go test -tags research -count=1 -run '^TestPortePositionsTemoins$' -v ./internal/games/halo_infinite/film/replay/

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/testutil"
)

func TestPortePositionsTemoins(t *testing.T) {
	faits := os.Getenv("M1_FAITS")
	if faits == "" {
		t.Skip("temoins sur faits reels : M1_FAITS requis")
	}
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(root, "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	doc := func(court, carte string) ReplayDocument {
		entry, err := cat.Lookup(carte)
		if err != nil {
			t.Fatal(err)
		}
		blob, err := os.ReadFile(filepath.Join(faits, court+".filmfacts.bin")) //nolint:gosec // temoin
		if err != nil {
			t.Fatal(err)
		}
		f, err := DecodeFilmFactsFile(blob, entry)
		if err != nil {
			t.Fatal(err)
		}
		return BuildFromFacts(court, "halo_infinite", f, Options{FrameIntervalMS: DefaultFrameIntervalMS})
	}
	t.Run("81c02726 Madina et le Mongoose 770", func(t *testing.T) { temoin81c02726(t, doc("81c02726", "Isolation")) })
	t.Run("ab526724 vies 769 et 770", func(t *testing.T) {
		for _, v := range doc("ab526724", "Starboard").Vehicles {
			if v.Slot == 769 || v.Slot == 770 {
				t.Errorf("vie de vehicule %d/%d publiee : tous ses echantillons sont hors de l emprise", v.Slot, v.Gen)
			}
		}
	})
	t.Run("879a4dba excursions 770 et 774", func(t *testing.T) {
		d := doc("879a4dba", "Fortitude")
		temoinSansEchantillonA(t, d, 770, 576)
		temoinSansEchantillonA(t, d, 774, 653)
	})
}

// temoin81c02726 : la vie de Madina (slot 523) commence au vrai point d apparition (frame 692),
// plus au point ecrit 4,65 s avant la creation de son corps ; le Mongoose 770 n a plus
// l echantillon lointain de la frame 983, et son retour porte la lacune.
//
// CE QUE CE TEMOIN EPINGLE, ET CE QU IL N EPINGLE PAS (revue adverse du 2026-09-24). Le point de
// Madina est a la fois anterieur a la creation (R-B1) et hors de l emprise (F-1) : c est
// `avantCreation >= 1` qui prouve R-B1 sur faits reels (la porte tourne R-B1 d abord), le debut a
// 692 prouve seulement que le point ne revient par aucune des deux voies. L echantillon 983 du
// Mongoose est hors de l emprise : F-1 le retire. F-2 N A PAS DE TEMOIN REEL parmi les trois films
// du plan (plafond de trois fichiers de faits) : ses deux declenchements au parc (1cd3848a, slot
// 799 du nuage, publie dans la vie 798/1 par fusion de relais ; 7b0d89c4, slot 768 — un
// echantillon chacun) sont suivis par l instrument de parc
// (`replaybuild/m1_parc_depuis_faits_research_test.go`, compteur `echantillonsAuTraversDUnSilence`).
func temoin81c02726(t *testing.T, d ReplayDocument) {
	t.Helper()
	vue := false
	for _, tr := range d.Tracks {
		if tr.Slot != 523 || tr.EndFrame < 645 || tr.StartFrame > 741 {
			continue
		}
		vue = true
		if tr.StartFrame != 692 {
			t.Errorf("vie de Madina : debut frame %d, attendu 692 (le record de creation)", tr.StartFrame)
		}
	}
	if !vue {
		t.Fatal("vie de Madina (slot 523, frames 645..741) absente du document : une vie disparue " +
			"ne prouve pas qu elle commence au record")
	}
	temoinSansEchantillonA(t, d, 770, 983)
	for _, v := range d.Vehicles {
		for _, s := range v.Samples {
			if v.Slot == 770 && s.T == 1035 && s.G <= 0 {
				t.Errorf("le retour du Mongoose 770 (frame 1035) ne porte pas la lacune : %+v", s)
			}
		}
	}
	if d.Coverage.Tracks.AvantCreation < 1 {
		t.Errorf("avantCreation = %d, attendu au moins le point de Madina", d.Coverage.Tracks.AvantCreation)
	}
}

// temoinSansEchantillonA verifie qu aucune vie du slot ne publie d echantillon a cette frame.
func temoinSansEchantillonA(t *testing.T, d ReplayDocument, slot uint32, frame int) {
	t.Helper()
	for _, v := range d.Vehicles {
		for _, s := range v.Samples {
			if v.Slot == slot && s.T == frame {
				t.Errorf("vehicule %d/%d : echantillon aberrant encore publie a la frame %d (%+v)",
					v.Slot, v.Gen, frame, s)
			}
		}
	}
}
