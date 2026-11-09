package weaponv3

import (
	"bytes"
	"compress/zlib"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// piverifyXuids — roster ground-truth du match 000d5950 (cf. piverify/main.go).
var piverifyXuids = []uint64{
	2633090228607708, 2603225103667623, 2543372234349153, 2667725237021739, // team0
	2603004317949713, 2742063072864889, 2812089420052706, 2709083604009238, // team1
}

// filmCacheRoot retourne le répertoire data/cache/film_chunks en testant
// plusieurs racines candidates. Le cache film vit dans le MAIN tree
// (data/cache/film_chunks/<short8>) ; un worktree git ne le réplique pas.
// On essaie : la racine du module courant (worktree), puis la racine du repo
// principal déduite du chemin .claude/worktrees/<nom>.
func filmCacheRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	candidates := make([]string, 0, 4)
	moduleRoot := findModuleRoot(t, wd)
	if moduleRoot != "" {
		// Le module est libs/film ; les data/ sont 2 niveaux au-dessus.
		candidates = append(candidates, filepath.Join(moduleRoot, "..", "..", "data", "cache", "film_chunks"))
		// Fallback : data/ à la racine du module (cas legacy).
		candidates = append(candidates, filepath.Join(moduleRoot, "data", "cache", "film_chunks"))
	}
	// Si on est sous .claude/worktrees/<nom>, le MAIN tree est la racine au-dessus.
	if mainRoot := mainTreeRootFromWorktree(wd); mainRoot != "" {
		candidates = append(candidates, filepath.Join(mainRoot, "data", "cache", "film_chunks"))
	}

	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}

// findModuleRoot remonte depuis wd jusqu'au go.mod du module.
func findModuleRoot(t *testing.T, wd string) string {
	t.Helper()
	dir := wd
	for range 12 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// mainTreeRootFromWorktree détecte le segment .claude/worktrees/<nom> dans wd
// et renvoie la racine du repo principal (le parent de .claude).
func mainTreeRootFromWorktree(wd string) string {
	norm := filepath.ToSlash(wd)
	const marker = "/.claude/worktrees/"
	before, _, ok := strings.Cut(norm, marker)
	if !ok {
		return ""
	}
	return filepath.FromSlash(before)
}

// inflateChunk décompresse un chunk zlib (magic 0x78) ; un chunk brut est rendu tel quel.
func inflateChunk(t *testing.T, label string, raw []byte) []byte {
	t.Helper()
	if len(raw) >= 2 && raw[0] == 0x78 {
		zr, err := zlib.NewReader(bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("zlib NewReader %s: %v", label, err)
		}
		defer func() { _ = zr.Close() }()
		inf, err := io.ReadAll(zr)
		if err != nil {
			t.Fatalf("zlib read %s: %v", label, err)
		}
		return inf
	}
	return raw
}

// loadCachedChunk lit un chunk de cache et le décompresse (zlib, magic 0x78).
// Renvoie nil si le fichier est absent (le caller décide de skipper).
func loadCachedChunk(t *testing.T, short8, name string) []byte {
	t.Helper()
	root := filmCacheRoot(t)
	if root == "" {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(root, short8, name))
	if err != nil {
		return nil
	}
	return inflateChunk(t, short8+"/"+name, raw)
}

// fixtureChunk lit un chunk de la mini-bobine versionnée de 000d5950 et le décompresse. Ses
// identifiants sont les fictifs du registre : c'est la seule source qui s'accorde aux littéraux de
// piverifyXuids, avec ou sans le cache du parc (dont les films portent les identifiants réels).
// Une bobine absente ou illisible est une erreur, pas un saut : elle est versionnée.
func fixtureChunk(t *testing.T, name string) []byte {
	t.Helper()
	chemin := filepath.Join("..", "..", "..", "replay", "testdata", "minifilm_000d5950", name)
	raw, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("fixture %s : %v", chemin, err)
	}
	return inflateChunk(t, chemin, raw)
}

// TestResolveXuidToPI_GroundTruth — sur le chunk_01 de la mini-bobine 000d5950, les pi résolus
// doivent être DISTINCTS et l'xuid …0022 doit tomber sur pi==2.
func TestResolveXuidToPI_GroundTruth(t *testing.T) {
	chunk := fixtureChunk(t, "chunk_01.bin")

	pis := ResolveXuidToPI(piverifyXuids, chunk)
	if len(pis) == 0 {
		t.Fatalf("aucun pi résolu sur chunk_01")
	}

	// Ground truth : 2603225103667623 → pi 2 (suffixe …0022).
	const gt = uint64(2603225103667623)
	if pi, ok := pis[gt]; !ok {
		t.Fatalf("xuid %d non résolu", gt)
	} else if pi != 2 {
		t.Fatalf("ground truth : xuid %d attendu pi==2, obtenu %d", gt, pi)
	}

	// Les pi résolus doivent être distincts (un pi == un slot joueur).
	seen := make(map[int]uint64, len(pis))
	for x, pi := range pis {
		if other, dup := seen[pi]; dup {
			t.Fatalf("pi %d partagé par les xuids %d et %d", pi, other, x)
		}
		seen[pi] = x
		if pi < 0 || pi > 31 {
			t.Fatalf("pi hors borne 0-31 : %d (xuid %d)", pi, x)
		}
	}
}

// TestResolveXuidToPIStrings_IgnoresBots — les xuids string décimaux sont
// résolus, les bots (préfixe "bid") et chaînes non numériques sont ignorés.
func TestResolveXuidToPIStrings_IgnoresBots(t *testing.T) {
	chunk := fixtureChunk(t, "chunk_01.bin")

	roster := []string{
		"2603225103667623", // ground truth pi 2
		"bid(1)",           // bot — ignoré
		"not-a-number",     // ignoré
	}
	pis := ResolveXuidToPIStrings(roster, chunk)
	if pi, ok := pis["2603225103667623"]; !ok || pi != 2 {
		t.Fatalf("attendu pi==2 pour 2603225103667623, obtenu %d (ok=%v)", pi, ok)
	}
	if _, ok := pis["bid(1)"]; ok {
		t.Fatalf("le bot bid(1) ne devrait pas être résolu")
	}
	if _, ok := pis["not-a-number"]; ok {
		t.Fatalf("une chaîne non numérique ne devrait pas être résolue")
	}
}

// TestResolveBest_MergesChunks — la fusion couvre au moins autant de xuids que
// le meilleur chunk seul (premier chunk qui trouve un xuid gagne).
func TestResolveBest_MergesChunks(t *testing.T) {
	c1 := fixtureChunk(t, "chunk_01.bin")
	c2 := fixtureChunk(t, "chunk_02.bin")

	single := ResolveXuidToPI(piverifyXuids, c1)
	merged := ResolveBest(piverifyXuids, [][]byte{c1, c2})

	if len(merged) < len(single) {
		t.Fatalf("la fusion (%d) couvre moins que chunk_01 seul (%d)", len(merged), len(single))
	}
	// Le premier chunk gagne : les pi de c1 doivent être préservés.
	for x, pi := range single {
		if mpi, ok := merged[x]; !ok || mpi != pi {
			t.Fatalf("xuid %d : pi du 1er chunk (%d) non préservé dans la fusion (%d, ok=%v)", x, pi, mpi, ok)
		}
	}
}
