package title

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestFilmFactsPath_FrereDesArtefactsEtJamaisDedans — l'invariant de LIEU.
//
// Le sidecar de raster tactique vit SOUS le dossier des artefacts et ne survit que parce que les
// deux parcours de ce dossier sautent les répertoires. Les faits de film, eux, doivent naître
// FRÈRES : sous `data/cache/`, hors de `data/cache/replays/`. Ce test épingle la différence —
// un futur déplacement sous les artefacts les exposerait aux deux parcours.
func TestFilmFactsPath_FrereDesArtefactsEtJamaisDedans(t *testing.T) {
	p := NewPathResolver("/depot")
	dir := p.FilmFactsDir("halo_infinite")
	artefacts := p.ReplayArtifactsDir("halo_infinite")
	if strings.HasPrefix(filepath.ToSlash(dir), filepath.ToSlash(artefacts)) {
		t.Fatalf("les faits de film sont RANGES SOUS les artefacts (%s sous %s) : les deux "+
			"parcours du dossier d'artefacts ne comptent que les `.json` de premier niveau et "+
			"suppriment ce qu'ils ne savent pas dater.", dir, artefacts)
	}
	cache := filepath.ToSlash(p.CacheRootDir())
	if !strings.HasPrefix(filepath.ToSlash(dir), cache+"/") {
		t.Fatalf("FilmFactsDir = %q, attendu sous la racine de cache %q — le sous-chemin "+
			"`data/cache` a UNE source, CacheRootDir.", dir, cache)
	}
	if !strings.HasSuffix(filepath.ToSlash(dir), "/"+SousDossierFilmFacts+"/halo_infinite") {
		t.Fatalf("FilmFactsDir = %q : le rangement est PAR TITRE, sous %q.", dir,
			SousDossierFilmFacts)
	}
}

// TestFilmFactsPath_MemeCheminPourLesDeuxFormes — même clé que l'artefact et que les chunks.
func TestFilmFactsPath_MemeCheminPourLesDeuxFormes(t *testing.T) {
	p := NewPathResolver("/depot")
	complet := p.FilmFactsPath("halo_infinite", "000d5950-1234-4abc-9def-0123456789ab")
	court := p.FilmFactsPath("halo_infinite", "000d5950")
	if complet != court {
		t.Fatalf("le match_id complet et sa forme courte donnent deux chemins :\n  %s\n  %s",
			complet, court)
	}
	if !strings.HasSuffix(complet, "000d5950"+ExtensionFilmFacts) {
		t.Fatalf("FilmFactsPath = %q, attendu suffixe %q", complet,
			"000d5950"+ExtensionFilmFacts)
	}
}

// TestExtensionFilmFacts_NeSeConfondPasAvecLesFaitsDeLaBase — la garde de NOM.
//
// `<short8>.facts.json` existe déjà et veut dire l'INVERSE (les faits que la BASE sait du match,
// `internal/replaybuild/facts_file.go`). Une extension en `.facts.` ferait des deux fichiers des
// homonymes pour des contenus inverses, et le premier diagnostic les confondrait.
func TestExtensionFilmFacts_NeSeConfondPasAvecLesFaitsDeLaBase(t *testing.T) {
	if ExtensionFilmFacts == ".facts.bin" || ExtensionFilmFacts == ".facts.json" {
		t.Fatalf("ExtensionFilmFacts = %q : réservé aux faits que la BASE sait du match.",
			ExtensionFilmFacts)
	}
	if !strings.HasPrefix(ExtensionFilmFacts, ".filmfacts") {
		t.Fatalf("ExtensionFilmFacts = %q, attendu un préfixe `.filmfacts` — le nom doit dire "+
			"que les faits viennent DU FILM.", ExtensionFilmFacts)
	}
}
