package title

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestFilmShortMatchID(t *testing.T) {
	cas := []struct{ in, want string }{
		{"000d5950-1234-4abc-9def-0123456789ab", "000d5950"},
		{"000d5950", "000d5950"},
		{"abcdef0123456789", "abcdef01"}, // sans tiret, plus long que 8 : on tronque
		{"abc", "abc"},                   // plus court que 8 : rendu tel quel, rien n'est fabriqué
		{"", ""},
		{"-abcdef", "-abcdef"}, // tiret en tête : rien à tronquer avant lui
	}
	for _, c := range cas {
		if got := FilmShortMatchID(c.in); got != c.want {
			t.Errorf("FilmShortMatchID(%q) = %q, attendu %q", c.in, got, c.want)
		}
	}
}

// TestReplayArtifactPath_MemeCheminPourLesDeuxFormes — l'invariant qui rend l'artefact
// atteignable depuis une route de l'application.
func TestReplayArtifactPath_MemeCheminPourLesDeuxFormes(t *testing.T) {
	p := NewPathResolver("/depot")
	court := p.ReplayArtifactPath(DefaultSlug, "000d5950")
	complet := p.ReplayArtifactPath(DefaultSlug, "000d5950-1234-4abc-9def-0123456789ab")
	if court != complet {
		t.Errorf("les deux formes du match_id donnent deux chemins :\n  court   %s\n  complet %s", court, complet)
	}
	if !strings.HasSuffix(filepath.ToSlash(court), "data/cache/replays/halo_infinite/000d5950.json") {
		t.Errorf("chemin inattendu : %s", court)
	}
}

// TestUneSeuleTroncatureDeMatchID — GARDE-RAIL (règle n°6 du dépôt).
//
// Cette troncature a existé en DEUX exemplaires (ici et dans le cache de films) et les
// deux ne se rencontraient jamais : l'un écrivait, l'autre cherchait ailleurs. Une
// troisième copie referait le même défaut, en silence.
func TestUneSeuleTroncatureDeMatchID(t *testing.T) {
	// Le motif de la troncature : un `[:i]` ou `[:8]` juste après une recherche de tiret.
	motif := regexp.MustCompile(`IndexByte\(\s*\w*[Mm]atch\w*\s*,\s*'-'\s*\)`)
	const proprietaire = "film_id.go"
	root := filepath.Join("..", "..", "..", "internal")
	var offenders []string
	vuChezLeProprietaire := false
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, rErr := os.ReadFile(filepath.Clean(path))
		if rErr != nil || !motif.Match(raw) {
			return nil
		}
		if filepath.Base(path) == proprietaire {
			vuChezLeProprietaire = true
			return nil
		}
		offenders = append(offenders, path)
		return nil
	})
	if err != nil {
		t.Fatalf("parcours du dépôt : %v", err)
	}
	// Le garde doit pouvoir ÉCHOUER : si le motif ne se trouve même plus chez son
	// propriétaire, c'est le test qui ne garde plus rien (leçon J4.0).
	if !vuChezLeProprietaire {
		t.Fatalf("motif introuvable dans %s : le garde-rail ne vérifie plus rien "+
			"(la fonction a-t-elle été renommée ou déplacée ?)", proprietaire)
	}
	if len(offenders) > 0 {
		t.Errorf("troncature de match_id recopiée hors de %s : %v — "+
			"appeler title.FilmShortMatchID", proprietaire, offenders)
	}
}

// TestTacticalRasterPath — le sidecar de raster suit l'artefact : MEME cle courte (les
// deux formes du match_id donnent un seul chemin) et SOUS-DOSSIER du dossier
// d'artefacts.
//
// LE SOUS-DOSSIER EST LA CONDITION DE COHABITATION, pas une preference de rangement :
// les deux parcours du dossier d'artefacts (AvailableSet, purge recurrente) ne comptent
// que les `.json` de PREMIER NIVEAU. A plat, un sidecar aurait ete lu comme l'artefact
// d'un match inexistant par l'un, et supprime comme un artefact indatable par l'autre.
func TestTacticalRasterPath(t *testing.T) {
	p := NewPathResolver("/depot")
	court := p.TacticalRasterPath(DefaultSlug, "000d5950")
	complet := p.TacticalRasterPath(DefaultSlug, "000d5950-1234-4abc-9def-0123456789ab")
	if court != complet {
		t.Errorf("les deux formes du match_id donnent deux chemins :\n  court   %s\n  complet %s", court, complet)
	}
	if !strings.HasSuffix(filepath.ToSlash(court), "data/cache/replays/halo_infinite/rasters/000d5950.json") {
		t.Errorf("chemin inattendu : %s", court)
	}
	// Le sidecar est SOUS le dossier des artefacts, et n'est PAS un fichier de son
	// premier niveau : c'est exactement ce que les deux parcours exigent.
	dirArtefacts := filepath.ToSlash(p.ReplayArtifactsDir(DefaultSlug))
	if !strings.HasPrefix(filepath.ToSlash(court), dirArtefacts+"/") {
		t.Errorf("le sidecar n'est pas sous le dossier des artefacts : %s", court)
	}
	if filepath.Dir(court) == p.ReplayArtifactsDir(DefaultSlug) {
		t.Errorf("le sidecar est POSE A PLAT dans le dossier des artefacts (%s) : "+
			"AvailableSet le compterait comme un match, la purge le supprimerait", court)
	}
}
