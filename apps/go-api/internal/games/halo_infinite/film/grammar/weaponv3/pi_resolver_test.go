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
	2533274826120416, 2533274823110022, 2533274980284321, 2535467794760703, // team0
	2533274882097883, 2533274815845110, 2535444178793711, 2535437947245250, // team1
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
		// Le module est apps/go-api ; les data/ sont 2 niveaux au-dessus.
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
	return ""
}

// mainTreeRootFromWorktree détecte le segment .claude/worktrees/<nom> dans wd
// et renvoie la racine du repo principal (le parent de .claude).
func mainTreeRootFromWorktree(wd string) string {
	norm := filepath.ToSlash(wd)
	const marker = "/.claude/worktrees/"
	idx := strings.Index(norm, marker)
	if idx < 0 {
		return ""
	}
	return filepath.FromSlash(norm[:idx])
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
	if len(raw) >= 2 && raw[0] == 0x78 {
		zr, err := zlib.NewReader(bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("zlib NewReader %s/%s: %v", short8, name, err)
		}
		defer zr.Close()
		inf, err := io.ReadAll(zr)
		if err != nil {
			t.Fatalf("zlib read %s/%s: %v", short8, name, err)
		}
		return inf
	}
	return raw
}

// TestResolveXuidToPI_GroundTruth — sur le chunk_01 de 000d5950, les pi résolus
// doivent être DISTINCTS et l'xuid …0022 doit tomber sur pi==2.
func TestResolveXuidToPI_GroundTruth(t *testing.T) {
	chunk := loadCachedChunk(t, "000d5950", "chunk_01.bin")
	if chunk == nil {
		t.Skip("cache film 000d5950/chunk_01.bin absent — skip ground-truth pi")
	}

	pis := ResolveXuidToPI(piverifyXuids, chunk)
	if len(pis) == 0 {
		t.Fatalf("aucun pi résolu sur chunk_01")
	}

	// Ground truth : 2533274823110022 → pi 2 (suffixe …0022).
	const gt = uint64(2533274823110022)
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
	chunk := loadCachedChunk(t, "000d5950", "chunk_01.bin")
	if chunk == nil {
		t.Skip("cache film 000d5950/chunk_01.bin absent — skip variante strings")
	}

	roster := []string{
		"2533274823110022", // ground truth pi 2
		"bid(1)",           // bot — ignoré
		"not-a-number",     // ignoré
	}
	pis := ResolveXuidToPIStrings(roster, chunk)
	if pi, ok := pis["2533274823110022"]; !ok || pi != 2 {
		t.Fatalf("attendu pi==2 pour 2533274823110022, obtenu %d (ok=%v)", pi, ok)
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
	c1 := loadCachedChunk(t, "000d5950", "chunk_01.bin")
	if c1 == nil {
		t.Skip("cache film 000d5950 absent — skip ResolveBest")
	}
	c2 := loadCachedChunk(t, "000d5950", "chunk_02.bin")

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
