package positions

// positions_test.go — valide le décodeur sur le match cache 000d5950 (§N).
//
// Skip-if-missing : le cache film vit dans le MAIN tree (data/cache/film_chunks),
// un worktree git ne le réplique pas. Si le cache est absent (CI sans data), le
// test est SKIPPÉ. Les helpers de résolution main-tree (filmCacheRoot /
// loadCachedChunk) reprennent le pattern de weaponv3/pi_resolver_test.go.

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// refMatchShort8 — 8 premiers chars hex de l'UUID du match de référence (§N).
const refMatchShort8 = "000d5950"

type refManifest struct {
	Chunks []struct {
		Index     int `json:"index"`
		ChunkType int `json:"chunk_type"`
		StartMS   int `json:"start_ms"`
	} `json:"chunks"`
}

// loadRefChunkInputs charge le manifeste + les chunks décompressés du match de
// référence en ChunkInput. Renvoie nil si le cache est absent (→ skip).
func loadRefChunkInputs(t *testing.T) []ChunkInput {
	t.Helper()
	root := filmCacheRoot(t)
	if root == "" {
		return nil
	}
	manPath := filepath.Join(filepath.Dir(root), "film_manifests", refMatchShort8+".json")
	mb, err := os.ReadFile(manPath)
	if err != nil {
		return nil
	}
	var m refManifest
	if err := json.Unmarshal(mb, &m); err != nil {
		t.Fatalf("manifest %s illisible: %v", refMatchShort8, err)
	}
	var chunks []ChunkInput
	for _, c := range m.Chunks {
		data := loadCachedChunk(t, refMatchShort8, fmt.Sprintf("chunk_%02d.bin", c.Index))
		if data == nil {
			continue
		}
		chunks = append(chunks, ChunkInput{
			Data: data, StartMS: c.StartMS, ChunkType: c.ChunkType,
		})
	}
	return chunks
}

// TestDecodeKeyframePositions_Ref valide le décodeur sur 000d5950 : nb de
// positions full-state dans [60,120], bornes collant à §N, et présence d'au
// moins une position éloignée de l'origine (mouvement réel).
func TestDecodeKeyframePositions_Ref(t *testing.T) {
	chunks := loadRefChunkInputs(t)
	if len(chunks) == 0 {
		t.Skip("cache film 000d5950 absent — skip décodage positions")
	}

	ps := DecodeKeyframePositions(chunks)
	n := len(ps)
	if n < 60 || n > 120 {
		t.Fatalf("nb positions full-state = %d, attendu [60,120]", n)
	}

	xmin, xmax := ps[0].X, ps[0].X
	ymin, ymax := ps[0].Y, ps[0].Y
	zmin, zmax := ps[0].Z, ps[0].Z
	farFromOrigin := false
	for _, p := range ps {
		xmin, xmax = minF(xmin, p.X), maxF(xmax, p.X)
		ymin, ymax = minF(ymin, p.Y), maxF(ymax, p.Y)
		zmin, zmax = minF(zmin, p.Z), maxF(zmax, p.Z)
		if math.Abs(float64(p.X)) > 20 {
			farFromOrigin = true
		}
	}
	t.Logf("positions=%d bornes x[%.1f,%.1f] y[%.1f,%.1f] z[%.1f,%.1f]",
		n, xmin, xmax, ymin, ymax, zmin, zmax)

	assertInRange(t, "x.min", float64(xmin), -10, 40)
	assertInRange(t, "x.max", float64(xmax), -10, 40)
	assertInRange(t, "y.min", float64(ymin), -30, 30)
	assertInRange(t, "y.max", float64(ymax), -30, 30)
	assertInRange(t, "z.min", float64(zmin), -4, 4)
	assertInRange(t, "z.max", float64(zmax), -4, 4)

	if !farFromOrigin {
		t.Fatalf("aucune position |x|>20 — mouvement réel non détecté (§N attend du déplacement)")
	}
}

func assertInRange(t *testing.T, name string, v, lo, hi float64) {
	t.Helper()
	if v < lo || v > hi {
		t.Fatalf("%s = %.2f hors borne [%.1f,%.1f] (§N)", name, v, lo, hi)
	}
}

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// --- Résolution du cache film dans le MAIN tree (pattern weaponv3) ---

// filmCacheRoot retourne data/cache/film_chunks en testant plusieurs racines.
// Le cache vit dans le MAIN tree ; un worktree git ne le réplique pas.
func filmCacheRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	candidates := make([]string, 0, 4)
	if moduleRoot := findModuleRoot(wd); moduleRoot != "" {
		candidates = append(candidates,
			filepath.Join(moduleRoot, "..", "..", "data", "cache", "film_chunks"),
			filepath.Join(moduleRoot, "data", "cache", "film_chunks"),
		)
	}
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

func findModuleRoot(wd string) string {
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

func mainTreeRootFromWorktree(wd string) string {
	norm := filepath.ToSlash(wd)
	const marker = "/.claude/worktrees/"
	idx := strings.Index(norm, marker)
	if idx < 0 {
		return ""
	}
	return filepath.FromSlash(norm[:idx])
}

// loadCachedChunk lit un chunk et le décompresse (zlib, magic 0x78). Renvoie nil
// si le fichier est absent (le caller décide de skipper).
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
