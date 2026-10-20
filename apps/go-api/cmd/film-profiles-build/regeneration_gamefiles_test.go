//go:build gamefiles

package main

// regeneration_gamefiles_test.go — CATALOGUE COMMIS = CATALOGUE REGENERE, pour la part
// FABRIQUEE du catalogue des profils de film.
//
// # LA CHAINE QUE CE TEST FERME
//
//	fichiers du jeu (.module)  --cmd/mapquant-build-->  map_quant_bounds.json
//	map_quant_bounds.json      --cmd/film-profiles-build-->  film_profiles.json (bloc `derived`)
//
// Le test rejoue la chaine ENTIERE depuis l installation locale et exige que le fichier commis
// soit, a l octet, celui que la chaine produit. Sans lui, la part derivee du profil serait une
// affirmation : « ces bornes viennent du jeu » — vraie le jour ou elle a ete ecrite, et
// invérifiable ensuite.
//
// La moitie SANS jeu (le bloc derive decrit-il le catalogue de bornes qui est dans l arbre ?)
// est un test ordinaire du paquet `filmprofile`, joue par la CI :
// `TestBlocDeriveSolidaireDesBornesCommises`. Les deux se completent — celui-ci prouve que les
// bornes commises sont bien celles du jeu, celui-la que le profil en est solidaire.
//
// # CE QU IL COUTE, ET POURQUOI IL EST QUAND MEME TAGUE
//
// Mesure du 2026-09-16 sur ce poste : 8,4 s pour `cmd/mapquant-build` (79 cartes). C est peu
// au regard du corpus `himap` (203 s a 1 246 s), mais la regle du depot ne porte pas sur la
// duree : un test qui OUVRE l installation du jeu porte le tag et le suffixe `_gamefiles`,
// sans quoi il tourne dans le build par defaut (ratchet `archlint.TestCorpusGamefilesEstTague`).
//
// Commande :
//
//	CGO_ENABLED=1 go test -tags=gamefiles ./cmd/film-profiles-build/ -count=1 -v
//
// CGO est OBLIGATOIRE : `internal/himap` tire `internal/ooz` (decompression), dont les
// fichiers sont exclus sans cgo. Sans jeu installe, le test se SKIPPE (c est le cas de la CI).

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/filmprofile"
	"levelup/go-api/internal/himap"
	"levelup/go-api/internal/testutil"
)

// deployVariantTest : la variante d installation ou vivent les bornes monde du BSP, la meme
// que celle de `cmd/mapquant-build`.
const deployVariantTest = "ds"

// TestCatalogueCommisEgaleCatalogueRegenere — LE GATE.
func TestCatalogueCommisEgaleCatalogueRegenere(t *testing.T) {
	if _, err := himap.LevelsDir(deployVariantTest); err != nil {
		t.Skipf("installation de Halo Infinite absente (%v) — la part derivee du profil ne se "+
			"regenere que la ou le jeu est installe", err)
	}
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	goAPI := filepath.Join(racine, "apps", "go-api")
	res := title.NewPathResolver(racine)
	cheminBornesCommis := res.MapQuantBoundsPath(title.DefaultSlug)
	cheminProfilsCommis := res.FilmProfilesPath(title.DefaultSlug)
	bornesRegenerees := filepath.Join(t.TempDir(), "map_quant_bounds.json")

	// 1. Les bornes, regenerees depuis les .module du jeu, dans une sortie JETABLE : ce test
	//    ne reecrit jamais un fichier versionne.
	lancer(t, goAPI, "./cmd/mapquant-build", "--out", bornesRegenerees)

	// 2. Les bornes commises sont celles du jeu — comparaison structurelle (le champ `source`
	//    du catalogue cite le chemin d installation de la machine, pas une donnee du jeu).
	resumeRegenere := resumer(t, bornesRegenerees)
	resumeCommis := resumer(t, cheminBornesCommis)
	if resumeCommis.Empreinte != resumeRegenere.Empreinte {
		t.Fatalf("les bornes commises ne sont PAS celles du jeu installe :\n  commis    %d carte(s), %s"+
			"\n  regenere  %d carte(s), %s\nrejouer `CGO_ENABLED=1 go run ./cmd/mapquant-build`",
			resumeCommis.Cartes, resumeCommis.Empreinte, resumeRegenere.Cartes, resumeRegenere.Empreinte)
	}

	// 3. Le catalogue des profils commis est, a l octet, celui que la chaine produit A PARTIR
	//    DES BORNES REGENEREES. `--check` n ecrit rien.
	lancer(t, goAPI, "./cmd/film-profiles-build", "--check",
		"--out", cheminProfilsCommis, "--bounds", bornesRegenerees)
}

// lancer execute une commande de la chaine de fabrication et echoue avec sa sortie.
func lancer(t *testing.T, dir, paquet string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", paquet}, args...)...) //nolint:gosec // paquets du module, ecrits ici
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	sortie, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run %s %v : %v\n%s", paquet, args, err, sortie)
	}
}

// resumer rend l empreinte structurelle d un catalogue de bornes.
func resumer(t *testing.T, chemin string) filmprofile.BornesResumees {
	t.Helper()
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture de %s : %v", chemin, err)
	}
	r, err := filmprofile.ResumerBornesDeCarte(blob)
	if err != nil {
		t.Fatalf("resume de %s : %v", chemin, err)
	}
	return r
}
