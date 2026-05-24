package testfixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// FilmManifest est la representation publique du manifest film Halo, miroir
// exact de la struct privee internal/sync.filmManifest. Decouplee pour eviter
// un cycle d'import (testfixtures importe par sync lui-meme).
type FilmManifest struct {
	BlobStoragePathPrefix string         `json:"BlobStoragePathPrefix"`
	CustomData            FilmCustomData `json:"CustomData"`
	FilmStatusBond        int            `json:"FilmStatusBond,omitempty"`
}

// FilmCustomData regroupe les chunks decrits par le manifest.
type FilmCustomData struct {
	FilmLength       int         `json:"FilmLength"`
	FilmMajorVersion int         `json:"FilmMajorVersion"`
	Chunks           []FilmChunk `json:"Chunks"`
}

// FilmChunk decrit un segment du film. ChunkType :
//
//	1 = HEADER (init data, 1er chunk de chaque film)
//	2 = REPLICATION_DATA (gameplay, le coeur du film)
//	3 = HIGHLIGHT_EVENTS (events extraits, dernier chunk en general)
type FilmChunk struct {
	Index                            int    `json:"Index"`
	ChunkType                        int    `json:"ChunkType"`
	ChunkSize                        int    `json:"ChunkSize"`
	ChunkStartTimeOffsetMilliseconds int    `json:"ChunkStartTimeOffsetMilliseconds"`
	DurationMilliseconds             int    `json:"DurationMilliseconds"`
	FileRelativePath                 string `json:"FileRelativePath"`
}

// JGtmFullMatch regroupe toutes les donnees du fixture E2E JGtm.
//
// Match : b71d39db-e3af-40e4-b7f9-e7c34c367981 (Arena, 8 participants, ~9 min,
// joue le 2026-05-19, JGtm xuid 2533274823110022).
type JGtmFullMatch struct {
	// Manifest film decode depuis manifest_raw.json.
	Manifest FilmManifest

	// ChunksDir : chemin absolu du dossier chunks/ (filmChunk0..filmChunk29).
	// Charger un chunk specifique via LoadChunk(idx).
	ChunksDir string

	// MatchStatsRaw : JSON brut de l'API stats (8 participants), pour decode
	// par le caller selon ses besoins (canonical row, struct domaine, etc.).
	MatchStatsRaw []byte

	// SkillRaw : JSON brut de l'API skill (CSR + MMR pour le xuid demande).
	SkillRaw []byte

	// MatchHistoryRaw : JSON brut de la page 0 du match history (5 derniers
	// matchs du joueur).
	MatchHistoryRaw []byte
}

// LoadChunk retourne le contenu binaire du chunk d'index donne.
//
// idx doit etre dans [0, len(Manifest.CustomData.Chunks)-1]. Pour les chunks
// REPLICATION_DATA, idx va typiquement de 1 a N-1 (0 = header, N = highlight).
func (m JGtmFullMatch) LoadChunk(t *testing.T, idx int) []byte {
	t.Helper()
	path := filepath.Join(m.ChunksDir, fmt.Sprintf("filmChunk%d", idx))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("testfixtures.LoadChunk(%d): %v", idx, err)
	}
	return data
}

// ReplicationChunkIndices retourne les indices des chunks REPLICATION_DATA
// (ChunkType=2). Pour le fixture JGtm : 28 chunks (indices 1..28).
func (m JGtmFullMatch) ReplicationChunkIndices() []int {
	out := make([]int, 0, len(m.Manifest.CustomData.Chunks))
	for _, c := range m.Manifest.CustomData.Chunks {
		if c.ChunkType == 2 {
			out = append(out, c.Index)
		}
	}
	return out
}

// HighlightChunkIndex retourne l'index du chunk HIGHLIGHT_EVENTS (ChunkType=3).
// Retourne -1 si absent (cas pathologique : film sans highlight chunk).
//
// Pour le fixture JGtm : index 29 (dernier chunk).
func (m JGtmFullMatch) HighlightChunkIndex() int {
	for _, c := range m.Manifest.CustomData.Chunks {
		if c.ChunkType == 3 {
			return c.Index
		}
	}
	return -1
}

// JGtmFullMatchAvailable retourne true si le fixture est present sur le
// disque local. Les tests qui en dependent doivent appeler cette fonction
// avant d'appeler LoadJGtmFullMatch — sinon Skip le test.
//
// Le fixture est gitignored (taille 6 MB + binaires), il faut le regenerer
// via `go run ./cmd/gen_test_fixtures download-full-match` (tokens requis).
func JGtmFullMatchAvailable() bool {
	manifestPath := filepath.Join(JGtmFullMatchDir(), "manifest_raw.json")
	chunksDir := filepath.Join(JGtmFullMatchDir(), "chunks")
	if _, err := os.Stat(manifestPath); err != nil {
		return false
	}
	// Verifie aussi qu'on a au moins le chunk 0 (sentinel).
	if _, err := os.Stat(filepath.Join(chunksDir, "filmChunk0")); err != nil {
		return false
	}
	return true
}

// LoadJGtmFullMatch charge le fixture E2E JGtm complet.
//
// Si le fixture est absent (non telecharge), appelle t.Skip avec instructions
// pour le regenerer. Permet aux tests d'etre safe sur CI sans le fixture.
//
// Usage typique :
//
//	if !testfixtures.JGtmFullMatchAvailable() {
//	    t.Skip("jgtm_full_match fixture absent — run cmd/gen_test_fixtures download-full-match")
//	}
//	fixture := testfixtures.LoadJGtmFullMatch(t)
//	chunk0 := fixture.LoadChunk(t, 0)
func LoadJGtmFullMatch(t *testing.T) JGtmFullMatch {
	t.Helper()
	if !JGtmFullMatchAvailable() {
		t.Skipf("jgtm_full_match fixture absent dans %s — regenerer via `go run ./cmd/gen_test_fixtures download-full-match`",
			JGtmFullMatchDir())
	}

	dir := JGtmFullMatchDir()
	var fx JGtmFullMatch

	// Manifest
	manifestRaw, err := os.ReadFile(filepath.Join(dir, "manifest_raw.json"))
	if err != nil {
		t.Fatalf("LoadJGtmFullMatch: read manifest: %v", err)
	}
	if err := json.Unmarshal(manifestRaw, &fx.Manifest); err != nil {
		t.Fatalf("LoadJGtmFullMatch: decode manifest: %v", err)
	}
	fx.ChunksDir = filepath.Join(dir, "chunks")

	// Reponses API — toleres si absentes (fixture partiellement telecharge).
	if data, err := os.ReadFile(filepath.Join(dir, "api_match_stats.json")); err == nil {
		fx.MatchStatsRaw = data
	}
	if data, err := os.ReadFile(filepath.Join(dir, "api_skill.json")); err == nil {
		fx.SkillRaw = data
	}
	if data, err := os.ReadFile(filepath.Join(dir, "api_match_history_page0.json")); err == nil {
		fx.MatchHistoryRaw = data
	}

	return fx
}
