package replay

// ctf_closure_research_test.go — INSTRUMENT DE RECHERCHE #3 (v7.5 voie B).
//
// # CE QU'IL EST DEVENU, ET POURQUOI IL A MAIGRI
//
// Ce fichier portait sa PROPRE implémentation des deux fermetures, le temps de les mesurer. Elles
// sont depuis passées en production (`closures.go`), et la copie de recherche a été SUPPRIMÉE :
// deux implémentations du même raisonnement divergeraient, et la règle du dépôt l'interdit.
//
// Ce qu'il mesure désormais : que les fermetures DE PRODUCTION reproduisent les chiffres publiés
// au §7.5 du verdict. C'est un gate de non-régression de la MESURE, pas un banc d'essai.
//
// Le réglage de la fenêtre de réapparition a été tranché et n'est plus rejoué ici : `[p05, p95]`
// rendait 13 vies sur 13 contestées sur `64e8adfa` (le p95 monte à 51,7 s), `médiane ± 750 ms`
// gagne de 3,5 à 5,5 points sur trois films. L'échec est consigné au §7.5, pas dans du code.
//
//	CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
//	  CTF_CLOSURE_FILMS="64e8adfa:Catalyst,..." \
//	  go test ./internal/analysis/replay/ -run CTFExclusionClosure -timeout 90m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const ctfClosureFilmsEnv = "CTF_CLOSURE_FILMS"

func TestCTFExclusionClosure(t *testing.T) {
	spec := os.Getenv(ctfClosureFilmsEnv)
	if spec == "" {
		t.Skipf("fermeture non demandée : %s vide", ctfClosureFilmsEnv)
	}
	cache, outDir := os.Getenv(ctfCacheEnv), os.Getenv(ctfOutEnv)
	if cache == "" || outDir == "" {
		t.Fatalf("%s et %s sont requis", ctfCacheEnv, ctfOutEnv)
	}
	cat := loadCTFQuantCatalog(t)
	for _, item := range strings.Split(spec, ",") {
		short, mapName, ok := strings.Cut(strings.TrimSpace(item), ":")
		if !ok {
			t.Fatalf("entrée mal formée %q", item)
		}
		t.Run(short, func(t *testing.T) {
			b := ctfClosureReport(t, cat, filepath.Join(cache, "film_chunks", short), short, mapName)
			if err := os.WriteFile(filepath.Join(outDir, short+"_fermeture.txt"), []byte(b), 0o644); err != nil {
				t.Fatalf("écriture : %v", err)
			}
			t.Logf("\n%s", b)
		})
	}
}

func ctfClosureReport(t *testing.T, cat *filmdec.MapQuantCatalog, dir, short, mapName string) string {
	t.Helper()
	pos, fire, deaths, table := ctfDecodeFilm(t, cat, dir, mapName)
	sort.SliceStable(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	tracks := indexBySlot(pos)

	owners, lives, off := ctfReadingOnlyOwners(tracks, deaths, table)
	before := ctfCoverageOf(tracks, owners, fire)
	closed, rep := closeBridge(tracks, owners, lives, deaths, off, table.ByXUID, fireRefs(fire))
	after := ctfCoverageOf(tracks, closed, fire)

	var b strings.Builder
	fmt.Fprintf(&b, "film\t%s\ncarte\t%s\n", short, mapName)
	fmt.Fprintf(&b, "slots_pont_avant\t%d\nslots_pont_apres\t%d\n", len(owners), len(closed))
	fmt.Fprintf(&b, "fermeture_par_tir\t%d\tfermeture_par_reapparition\t%d\tcontestees\t%d\trefusees_recouvrement\t%d\n",
		rep.byShot, rep.byRespawn, rep.contested, rep.refused)
	fmt.Fprintf(&b, "\n# couverture des tirs\n")
	fmt.Fprintf(&b, "avant\trattaches\t%d\tsur\t%d\ttaux\t%.4f\tsans_slot\t%d\tambigus\t%d\n",
		before.Attached, before.Available, ratio(before.Attached, before.Available), before.NoSlot, before.Ambiguous)
	fmt.Fprintf(&b, "apres\trattaches\t%d\tsur\t%d\ttaux\t%.4f\tsans_slot\t%d\tambigus\t%d\n",
		after.Attached, after.Available, ratio(after.Attached, after.Available), after.NoSlot, after.Ambiguous)
	fmt.Fprintf(&b, "gain_points\t%.2f\n",
		100*(ratio(after.Attached, after.Available)-ratio(before.Attached, before.Available)))
	return b.String()
}

// ctfDecodeFilm est le préambule commun des instruments : décoder un film et lire son pont.
// Factorisé au TROISIÈME exemplaire, comme l'exige la règle du dépôt.
func ctfDecodeFilm(t *testing.T, cat *filmdec.MapQuantCatalog, dir, mapName string) (
	[]filmdec.BipedPosition, []filmdec.FireEvent, []Death, PlayerIndexTable) {
	t.Helper()
	entry, err := cat.Lookup(mapName)
	if err != nil {
		t.Fatalf("bornes de %s : %v", mapName, err)
	}
	world := entry.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange, scan.CaptureDirs = &world, true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions : %v", err)
	}
	fire, err := filmdec.ScanFilmFireEvents(dir)
	if err != nil {
		t.Fatalf("tirs : %v", err)
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("morts : %v", err)
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Fatalf("index : %v", err)
	}
	table, _ := injectiveOrEmpty(idx)
	return pos, fire, deaths, table
}

// ctfCoverageOf rend la couverture du calque des tirs pour un pont donné — même porte que
// buildShots, sans le filtre des trajectoires publiées (mesuré nul sur les sept films) : on
// compare deux ponts, pas deux pipelines.
func ctfCoverageOf(tracks map[uint32]slotTrack, owner map[uint32]int, fire []filmdec.FireEvent) LayerCoverage {
	cov := LayerCoverage{Available: len(fire)}
	for _, e := range fire {
		_, reason := slotFor(tracks, owner, e.FilmIndex, e.TimestampUS)
		cov.count(reason)
	}
	return cov
}
