package replay

// equipment_changes_film_research_test.go — le calque des ramassages et des consommations
// d'equipement, VU DE BOUT EN BOUT sur un vrai film. Le golden d'assemblage ne peut pas le
// faire : son fixture d'entrees a ete fige avant ce calque et ne porte aucun changement.
//
// GARDE : PICKUP_FILM (repertoire du film) et PICKUP_MAP (nom de carte du catalogue de bornes),
// les memes que l'instrument des armes — les deux calques se mesurent sur le meme assemblage.

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

func TestEquipmentChangesSurFilmReel(t *testing.T) {
	dir, mapName := os.Getenv("PICKUP_FILM"), os.Getenv("PICKUP_MAP")
	if dir == "" || mapName == "" {
		t.Skip("PICKUP_FILM / PICKUP_MAP absents : instrument de mesure saute")
	}
	path := filepath.Join("..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	entry, err := cat.Lookup(mapName)
	if err != nil {
		t.Fatalf("carte %q : %v", mapName, err)
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("chunks du film illisibles : %v", err)
	}
	doc, err := BuildFromFilm("mesure-equipement", "halo_infinite", film, Options{MapQuant: &entry})
	if err != nil {
		t.Fatalf("assemblage : %v", err)
	}
	if doc.Coverage.EquipmentChanges == nil {
		t.Fatal("coverage.equipmentChanges absente : elle doit etre publiee MEME vide, sinon un " +
			"film sans ramassage et un film qu'on n'a pas su balayer se confondent")
	}
	cov := *doc.Coverage.EquipmentChanges
	t.Logf("COUVERTURE : decodes=%d publies=%d ramassages=%d consommations=%d reapparitions=%d "+
		"vies=%d", cov.Decoded, cov.Published, cov.Taken, cov.Spent, cov.Spawned, cov.Lives)
	t.Logf("TEMOIN DE COMPLETUDE : manqueesEstimees=%d sautsCompteur=%d premiereHorsNorme=%d "+
		"repetitions=%d", cov.MissedEstimate, cov.CounterJumps, cov.LivesFirstOffSpec, cov.Repeats)

	// LE DOCUMENT PEUT PORTER MOINS QUE LA COUVERTURE : les changements dont le slot n'a pas de
	// trajectoire publiee sont ecartes apres coup, comme pour les lectures de capacite.
	if len(doc.EquipmentChanges) > cov.Published {
		t.Errorf("document=%d entrees pour %d publies : le filtre des pistes ne peut qu'en "+
			"retirer", len(doc.EquipmentChanges), cov.Published)
	}
	if cov.Repeats > 0 {
		t.Errorf("repetitions=%d : le compteur de rotation ne doit JAMAIS rester immobile d'une "+
			"emission a la suivante — c'est la propriete qui fonde le calque", cov.Repeats)
	}
	ranks := map[int]bool{}
	for _, e := range doc.EquipmentChanges {
		if e.T < 0 || e.T > doc.FrameCount {
			t.Errorf("instant %d hors de l'axe [0, %d]", e.T, doc.FrameCount)
		}
		switch e.Kind {
		case EquipmentSpent:
			if e.R != NoAbilityRank {
				t.Errorf("consommation avec r=%d : apres une consommation le joueur ne porte "+
					"plus rien", e.R)
			}
		case EquipmentTaken:
			if e.R == NoAbilityRank {
				t.Errorf("ramassage sans rang : un ramassage nomme forcement ce qui est ramasse")
			}
			ranks[e.R] = true
		default:
			t.Errorf("nature %q inattendue dans le document", e.Kind)
		}
	}
	// LE NOMMAGE N'EST PAS MESURE ICI, et un zero ne veut donc rien dire : cet assemblage est
	// monte SANS le manifeste du titre (Options.Labels vide), donc aucune palette n'est classee
	// et aucun rang n'est nomme. En production c'est l'appelant qui fournit les libelles.
	t.Logf("RANGS RAMASSES : %v ; nommes par le catalogue : %d sur %d (0 attendu sans manifeste)",
		ranks, countNamed(ranks, doc.AbilityLabels), len(ranks))
	for i, e := range doc.EquipmentChanges {
		if i >= 10 {
			break
		}
		t.Logf("   t=%-5d vie=%-4d %-6s r=%-4d from=%d", e.T, e.Slot, e.Kind, e.R, e.From)
	}
	t.Log("JUGE : la PLAUSIBILITE, et le temoin de completude la borne. Quelques ramassages par " +
		"match, des consommations qui les suivent, et un nombre de manquees petit devant les vues.")
}

// countNamed compte les rangs que le catalogue du film sait nommer.
func countNamed(ranks map[int]bool, labels map[string]Label) int {
	n := 0
	for r := range ranks {
		if _, ok := labels[strconv.Itoa(r)]; ok {
			n++
		}
	}
	return n
}
