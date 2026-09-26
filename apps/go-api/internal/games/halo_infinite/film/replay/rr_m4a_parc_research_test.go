//go:build research

package replay

// rr_m4a_parc_research_test.go — RECONSTRUCTION DU PARC DEPUIS LES FAITS, HORS DE `data/`
// (retours du rejeu 2026-09-23, lot M4a : validation de parc avant / apres).
//
// # CE QU IL FAIT
//
// Il relit CHAQUE fichier de faits persistes du parc (`data/cache/film_facts/<slug>/*.bin`, en
// LECTURE SEULE), assemble le document par la porte de production [BuildFromFacts] — aucun octet
// de film, aucune grammaire exercee — et ecrit le document dans un dossier HORS de `data/`. Joue
// une fois au code de BASE (commit qui ne porte que cet instrument) et une fois au code de la
// BRANCHE, il rend deux dossiers que `cmd/replay-diff` compare calque par calque : seuls les
// calques vises par le lot doivent bouger.
//
// LES OPTIONS SONT CELLES D UN ASSEMBLAGE HORS CONTEXTE (catalogue de libelles du depot, bornes de
// la carte du catalogue versionne) : ni roster de la base, ni statborg, ni zones. C est voulu —
// les deux passes emploient les MEMES options, donc toute difference vient du code.
//
// UN FILM A LA FOIS, sequentiellement, dans ce seul processus ; le verrou de la voie film est pose
// par l appelant. Un fichier de faits perime pour ce binaire (codec, schema) est COMPTE et saute,
// jamais relu de force.
//
//	RR_M4A_ROOT=<depot qui porte data/> RR_M4A_OUT=<dossier hors data/> \
//	  go test -tags research -count=1 -timeout 60m -run '^TestRRM4AParcDepuisLesFaits$' \
//	  ./internal/games/halo_infinite/film/replay/

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

func TestRRM4AParcDepuisLesFaits(t *testing.T) {
	root, out := os.Getenv("RR_M4A_ROOT"), os.Getenv("RR_M4A_OUT")
	if root == "" || out == "" {
		t.Skip("instrument de parc : RR_M4A_ROOT et RR_M4A_OUT requis")
	}
	if strings.Contains(filepath.ToSlash(filepath.Clean(out)), "/data/") {
		t.Fatalf("le dossier de sortie doit etre HORS de data/ : %s", out)
	}
	if err := os.MkdirAll(out, 0o750); err != nil {
		t.Fatal(err)
	}
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(root, "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "data", "cache", "film_facts", "halo_infinite")
	noms, err := filepath.Glob(filepath.Join(dir, "*.filmfacts.bin"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(noms)
	labels := goldenCatalog(t)
	var ecrits, perimes, sansCarte int
	for _, chemin := range noms {
		id := strings.TrimSuffix(filepath.Base(chemin), ".filmfacts.bin")
		switch rrM4AReconstruire(t, cat, labels, chemin, filepath.Join(out, id+".json")) {
		case "ok":
			ecrits++
		case "perime":
			perimes++
		default:
			sansCarte++
		}
		debug.FreeOSMemory()
	}
	t.Logf("PARC : %d fichiers de faits, %d documents ecrits, %d perimes, %d sans carte",
		len(noms), ecrits, perimes, sansCarte)
}

// rrM4AReconstruire assemble UN document depuis ses faits et l ecrit. Rend « ok », « perime »
// (en-tete illisible pour ce binaire) ou « carte » (module absent du catalogue de bornes).
func rrM4AReconstruire(t *testing.T, cat *profile.MapQuantCatalog, labels LabelCatalog,
	chemin, sortie string,
) string {
	t.Helper()
	blob, err := os.ReadFile(chemin) //nolint:gosec // instrument de mesure, chemin du parc
	if err != nil {
		t.Fatal(err)
	}
	head, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Logf("perime : %s : %v", filepath.Base(chemin), err)
		return "perime"
	}
	var entry profile.MapQuantEntry
	trouve := false
	for _, e := range cat.Maps {
		if e.Module == head.MapModule {
			entry, trouve = e, true
			break
		}
	}
	if !trouve {
		t.Logf("carte : %s : module %q absent du catalogue", filepath.Base(chemin), head.MapModule)
		return "carte"
	}
	f, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Logf("perime : %s : %v", filepath.Base(chemin), err)
		return "perime"
	}
	id := strings.TrimSuffix(filepath.Base(chemin), ".filmfacts.bin")
	doc := BuildFromFacts(id, "halo_infinite", f, Options{MapQuant: &entry, Labels: labels})
	enc, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sortie, enc, 0o600); err != nil {
		t.Fatal(err)
	}
	return "ok"
}
