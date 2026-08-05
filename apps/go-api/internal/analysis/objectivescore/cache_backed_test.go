package objectivescore

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// cache_backed_test.go — test bout-à-bout sur un VRAI film caché (7344d24f,
// Strongholds:Arena, finals DB 193-112). Skip si le cache n'est pas présent
// (environnement CI sans data/cache). Valide que team0 est monotone et finit
// EXACTEMENT au final calibré.

const (
	// cacheRoot = racine du cache film (statique, lecture seule). Skip-if-missing.
	cacheRoot   = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache`
	shGTShort   = "7344d24f" // Strongholds:Arena ground-truth
	shGTFinalT0 = 193
	shGTFinalT1 = 112
)

type testManifest struct {
	Chunks []struct {
		Index     int `json:"index"`
		ChunkType int `json:"chunk_type"`
		StartMS   int `json:"start_ms"`
	} `json:"chunks"`
}

// loadCachedChunks lit + décompresse les chunks d'un film caché vers []ChunkInput.
// Renvoie ok=false si le manifest est absent (skip).
func loadCachedChunks(t *testing.T, short string) ([]ChunkInput, bool) {
	t.Helper()
	mfPath := filepath.Join(cacheRoot, "film_manifests", short+".json")
	raw, err := os.ReadFile(mfPath)
	if os.IsNotExist(err) {
		return nil, false
	}
	if err != nil {
		t.Fatalf("read manifest %s: %v", short, err)
	}
	var mf testManifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		t.Fatalf("unmarshal manifest %s: %v", short, err)
	}
	var chunks []ChunkInput
	for _, c := range mf.Chunks {
		p := filepath.Join(cacheRoot, "film_chunks", short, sprintfChunk(c.Index))
		rawChunk, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		chunks = append(chunks, ChunkInput{
			Data: decompress(rawChunk), StartMS: c.StartMS, ChunkType: c.ChunkType,
		})
	}
	return chunks, true
}

func sprintfChunk(i int) string {
	const hex = "0123456789"
	if i < 10 {
		return "chunk_0" + string(hex[i]) + ".bin"
	}
	return "chunk_" + string(hex[i/10]) + string(hex[i%10]) + ".bin"
}

// decompress = zlib si magic 0x78, sinon renvoie raw (miroir des décodeurs film).
func decompress(raw []byte) []byte {
	if len(raw) >= 2 && raw[0] == 0x78 {
		if z, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			defer z.Close()
			if d, e2 := io.ReadAll(z); e2 == nil {
				return d
			}
		}
	}
	return raw
}

func TestDecodeStrongholds_CacheBacked_7344d24f(t *testing.T) {
	chunks, ok := loadCachedChunks(t, shGTShort)
	if !ok {
		t.Skipf("cache absent (%s) — test cache-backed sauté", cacheRoot)
	}

	frames := DecodeStrongholds(chunks, shGTFinalT0, shGTFinalT1)
	if len(frames) == 0 {
		t.Fatalf("aucun ScoreFrame décodé sur %s (chunks=%d)", shGTShort, len(chunks))
	}

	// team0 monotone croissant.
	for i := 1; i < len(frames); i++ {
		if frames[i].Team0 < frames[i-1].Team0 {
			t.Fatalf("Team0 non monotone à i=%d : %d < %d", i, frames[i].Team0, frames[i-1].Team0)
		}
	}
	// Dernière frame == final DB exact (calibration).
	if last := frames[len(frames)-1].Team0; last != shGTFinalT0 {
		t.Errorf("dernière frame Team0 = %d, want %d (final DB)", last, shGTFinalT0)
	}
	if last := frames[len(frames)-1].Team1; last != shGTFinalT1 {
		t.Errorf("dernière frame Team1 = %d, want %d (final DB)", last, shGTFinalT1)
	}
	// La timeline démarre à 0 (score initial).
	if frames[0].Team0 != 0 {
		t.Errorf("première frame Team0 = %d, want 0", frames[0].Team0)
	}
	t.Logf("7344d24f : %d frames, team0 %d→%d, team1 %d→%d",
		len(frames), frames[0].Team0, frames[len(frames)-1].Team0,
		frames[0].Team1, frames[len(frames)-1].Team1)
}
