package main

import (
	"bufio"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/testutil"
)

// Chemins relatifs à la racine du dépôt.
const (
	staticHudRoot    = "static/weapons-assets/halo_infinite"
	replayLabelsTOML = "config/titles/halo_infinite/mappings/replay_labels.toml"
)

// depuisLaRacine rend le chemin absolu d'un chemin relatif à la racine du dépôt.
func depuisLaRacine(t *testing.T, rel string) string {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, filepath.FromSlash(rel))
}

// atlasSizes : nombre d'images appariées de chaque atlas lu, mesuré sur le jeu.
var atlasSizes = map[uint32]int{
	abilityAtlasTag:   18,
	equipmentAtlasTag: 21,
	grenadeAtlasTag:   5,
}

func TestHudIcons_TableIsWellFormed(t *testing.T) {
	if len(hudIcons) != 15 {
		t.Fatalf("la table porte %d lignes, 15 attendues (les vignettes hud/)", len(hudIcons))
	}
	files := map[string]bool{}
	for _, ic := range hudIcons {
		if files[ic.File] {
			t.Errorf("fichier %s écrit deux fois", ic.File)
		}
		files[ic.File] = true
		if !strings.HasSuffix(ic.File, ".png") {
			t.Errorf("%s : extension .png attendue", ic.File)
		}
		n, ok := atlasSizes[ic.Atlas]
		if !ok {
			t.Errorf("%s : atlas %08x inconnu", ic.File, ic.Atlas)
			continue
		}
		if ic.Index < 0 || ic.Index >= n {
			t.Errorf("%s : index %d hors de l'atlas %08x (%d images)", ic.File, ic.Index, ic.Atlas, n)
		}
	}
}

// Tout `icon = "hud/X"` du manifeste doit avoir sa ligne : sinon la vignette servie ne
// serait plus régénérée par l'outil.
func TestHudIcons_CoverEveryIconOfTheManifest(t *testing.T) {
	f, err := os.Open(depuisLaRacine(t, replayLabelsTOML))
	if err != nil {
		t.Fatalf("manifeste illisible : %v", err)
	}
	defer func() { _ = f.Close() }()
	inTable := map[string]bool{}
	for _, ic := range hudIcons {
		inTable[ic.File] = true
	}
	seen := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		i := strings.Index(line, `icon = "hud/`)
		if i < 0 {
			continue
		}
		rest := line[i+len(`icon = "`):]
		name := rest[:strings.Index(rest, `"`)] + ".png"
		seen++
		if !inTable[name] {
			t.Errorf("%s est déclaré par le manifeste mais absent de la table", name)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if seen == 0 {
		t.Fatal("aucune ligne `icon = \"hud/...\"` trouvée : le test ne contrôle plus rien")
	}
}

// Chaque ligne de la table désigne un fichier servi : les chemins ne changent pas, les URL
// sont cuites dans les rejeux.
func TestHudIcons_EveryTargetExists(t *testing.T) {
	for _, ic := range hudIcons {
		p := filepath.Join(depuisLaRacine(t, staticHudRoot), filepath.FromSlash(ic.File))
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s : %v", ic.File, err)
		}
	}
}

func TestSquareCanvas_CentersTheGlyph(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 6, 2))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	out := squareCanvas(src)
	if b := out.Bounds(); b.Dx() != 6 || b.Dy() != 6 {
		t.Fatalf("taille %v, 6x6 attendu", b)
	}
	for y := 0; y < 6; y++ {
		want := uint8(0)
		if y == 2 || y == 3 {
			want = 255
		}
		if got := out.Pix[out.PixOffset(0, y)+3]; got != want {
			t.Errorf("ligne %d : alpha %d, %d attendu", y, got, want)
		}
	}
}
