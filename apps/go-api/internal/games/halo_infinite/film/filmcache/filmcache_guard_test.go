package filmcache_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// filmcache_guard_test.go — GARDE-RAIL de la centralisation.
//
// La source disque du cache film existait en double (cmd/diag_weapons_v3 et les tests
// d'objectiveevents) avant que ce paquet ne la rassemble. La regle du depot est qu'une
// factorisation SANS garde-rail re-diverge : sans ce test, la troisieme copie
// reapparaitrait au premier outil qui a besoin d'un film, et le jour ou la disposition du
// cache changerait, deux lecteurs sur trois rendraient « rien a decoder » en silence.

// sourceImpl reconnait une implementation de objectiveevents.FilmSource : la methode
// Chunks() rendant une tranche de ChunkMeta.
var sourceImpl = regexp.MustCompile(`func \([^)]*\) Chunks\(\) \[\](objectiveevents\.)?ChunkMeta`)

// allowlist : fichiers autorises a porter leur propre implementation, avec la RAISON et
// la date. Toute nouvelle entree doit porter les deux.
//
//	internal/analysis/objectiveevents/extract_test.go (2026-08-08) — CYCLE D'IMPORT.
//	filmcache importe objectiveevents pour ses types ; un test DANS le paquet
//	objectiveevents ne peut donc pas importer filmcache. Le lever demanderait de
//	convertir ce fichier en paquet de test externe — hors du perimetre du lot v7.5-4.
//	internal/analysis/objectiveevents/statborg_rounds_test.go (2026-08-18) — MEME CYCLE
//	D'IMPORT. Ce test fige les corrections de production du decodeur d'enregistrements sur des
//	vecteurs reels ; il vit DANS le paquet objectiveevents (il lit des symboles non exportes),
//	donc il ne peut pas importer filmcache, qui l'importe. Sa source n'est d'ailleurs pas une
//	source disque : c'est un buffer construit en memoire, repete — la disposition du cache
//	n'entre pas dans son sujet. Entree posee au lot A phase 1, apres que le garde-rail a
//	rougi (le test etait arrive au commit precedent sans elle).
var allowlist = map[string]string{
	filepath.FromSlash("internal/analysis/objectiveevents/extract_test.go"):         "cycle d'import (2026-08-08)",
	filepath.FromSlash("internal/analysis/objectiveevents/statborg_rounds_test.go"): "cycle d'import (2026-08-18)",
}

func TestUneSeuleSourceDisqueDeFilm(t *testing.T) {
	root := moduleRoot(t)
	var offenders []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == "node_modules" || name == "testdata" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if strings.HasPrefix(rel, filepath.FromSlash("internal/games/halo_infinite/film/filmcache")) {
			return nil // le paquet canonique
		}
		if _, ok := allowlist[rel]; ok {
			return nil
		}
		blob, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if sourceImpl.Match(blob) {
			offenders = append(offenders, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours du module : %v", err)
	}
	if len(offenders) > 0 {
		t.Errorf("implementation(s) de FilmSource hors de filmcache : %v\n"+
			"Utiliser filmcache.Open, ou justifier une entree d'allowlist datee dans ce fichier.",
			offenders)
	}
}

// TestLAllowlistNeCouvreQueDesFichiersExistants empeche l'allowlist de survivre a ce
// qu'elle autorise : une entree perimee rendrait le garde-rail plus permissif sans que
// personne ne le voie.
func TestLAllowlistNeCouvreQueDesFichiersExistants(t *testing.T) {
	root := moduleRoot(t)
	for rel, raison := range allowlist {
		p := filepath.Join(root, rel)
		blob, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("entree d'allowlist %q (%s) : fichier absent — la retirer", rel, raison)
			continue
		}
		if !sourceImpl.Match(blob) {
			t.Errorf("entree d'allowlist %q (%s) : le fichier n'implemente plus FilmSource — la retirer",
				rel, raison)
		}
	}
}

// moduleRoot remonte jusqu'au go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 12; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("go.mod introuvable au-dessus du repertoire de test")
	return ""
}
