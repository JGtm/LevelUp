// tmp_filmmanifest — demander a l'API Halo un manifeste de film FRAIS et l'ecrire au format
// attendu par cmd/fetch_film_chunks.
//
// POURQUOI. Les URL de blob des manifestes en cache sont PRE-SIGNEES et expirent : sur les
// 62 chunks manquants du cache, 62 rendent 404. Un manifeste frais rend des URL signees a
// nouveau. Sans cet outil il n'existe aucun chemin Go pour rafraichir un manifeste — seul
// cmd/refresh_golden_fixture le fait, et il passe par le chemin d'auth LEGACY (`.env.local`)
// dont les valeurs sont vides depuis la migration ADR 0023.
//
// AUTH — chemin canonique ADR 0023, aucune re-capture de jeton. On lit le
// MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json) via
// auth.RefreshHaloTokensViaStoreFirst. Si le refresh token est mort, on echoue bruyamment :
// re-capturer serait masquer la panne, pas la reparer.
//
// PRUDENCE SUR LE FORMAT. La forme exacte de la reponse /spectate n'est parsee nulle part
// dans le depot, et aucun fixture n'en subsiste. On ne devine donc pas : le brut est ecrit
// sur disque AVANT toute conversion, et la conversion cherche ses champs par un parcours
// generique de l'arbre JSON plutot que par une structure figee. Si la conversion echoue, le
// brut reste exploitable a la main.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	authpkg "levelup/go-api/internal/platform/auth"
)

const spectateURL = "https://discovery-infiniteugc.svc.halowaypoint.com/hi/films/matches/%s/spectate"

// cachedManifest est le format que cmd/fetch_film_chunks sait lire (« manifests Python
// herites »). On le reproduit a l'identique.
type cachedManifest struct {
	BlobPrefix string        `json:"blob_prefix"`
	Chunks     []cachedChunk `json:"chunks"`
}

type cachedChunk struct {
	Index            int    `json:"index"`
	ChunkType        int    `json:"chunk_type"`
	StartMS          int    `json:"start_ms"`
	DurationMS       int    `json:"duration_ms"`
	FileRelativePath string `json:"file_relative_path"`
}

func main() {
	matchID := flag.String("match", "", "identifiant COMPLET du match (UUID)")
	xuid := flag.String("xuid", "2533274823110022", "xuid du joueur dont on emprunte les jetons")
	gamertag := flag.String("gamertag", "JGtm", "gamertag correspondant")
	repoRoot := flag.String("root", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration`, "racine du depot (pour data/)")
	rawOut := flag.String("raw", "", "chemin d'ecriture de la reponse BRUTE (defaut : a cote du manifeste)")
	flag.Parse()

	if *matchID == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_filmmanifest -match <UUID> [-xuid X] [-gamertag G]")
		os.Exit(2)
	}
	ctx := context.Background()

	spartan, clearance, err := resolveTokens(ctx, *repoRoot, *xuid, *gamertag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "auth :", err)
		os.Exit(1)
	}
	fmt.Printf("jetons obtenus (clearance : %v)\n", clearance != "")

	raw, err := fetchSpectate(ctx, spartan, clearance, *matchID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "appel /spectate :", err)
		os.Exit(1)
	}

	short := strings.SplitN(*matchID, "-", 2)[0]
	manifestPath := filepath.Join(*repoRoot, "data", "cache", "film_manifests", short+".json")
	rawPath := *rawOut
	if rawPath == "" {
		rawPath = manifestPath + ".raw"
	}
	if err := os.MkdirAll(filepath.Dir(rawPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(rawPath, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "ecriture du brut :", err)
		os.Exit(1)
	}
	fmt.Printf("reponse brute -> %s (%d octets)\n", rawPath, len(raw))

	mf, err := convert(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "conversion :", err)
		fmt.Fprintln(os.Stderr, "le brut est conserve — inspecter sa structure a la main.")
		os.Exit(1)
	}
	out, err := json.Marshal(mf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(manifestPath, out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "ecriture du manifeste :", err)
		os.Exit(1)
	}
	repl := 0
	for _, c := range mf.Chunks {
		if c.ChunkType == 2 {
			repl++
		}
	}
	fmt.Printf("manifeste -> %s\n", manifestPath)
	fmt.Printf("  prefixe blob : %s\n", mf.BlobPrefix)
	fmt.Printf("  %d chunks, dont %d de type REPLICATION_DATA\n", len(mf.Chunks), repl)
	fmt.Printf("\nEnchainer : go run ./cmd/fetch_film_chunks/ -cache %s\n",
		filepath.Join(*repoRoot, "data", "cache"))
}

// resolveTokens suit le chemin canonique ADR 0023. Aucune re-capture.
func resolveTokens(ctx context.Context, root, xuid, gamertag string) (string, string, error) {
	if t := os.Getenv("SPARTAN_TOKEN"); t != "" {
		return t, os.Getenv("CLEARANCE_TOKEN"), nil
	}
	store := authpkg.NewMultiUserTokenStore(filepath.Join(root, "data", "auth", "watcher_tokens"))
	res, err := authpkg.RefreshHaloTokensViaStoreFirst(
		ctx, store, authpkg.NewSISUProvider(), xuid, gamertag, authpkg.LegacyAuthInputs{})
	if err != nil {
		return "", "", fmt.Errorf("refresh depuis le store (xuid %s) : %w", xuid, err)
	}
	tok := authpkg.HaloTokensFromExchange(res)
	if tok == nil || tok.SpartanToken == "" {
		return "", "", fmt.Errorf(
			"aucun jeton exploitable pour %s (xuid %s) — diagnostiquer la chaine de refresh, ne PAS re-capturer",
			gamertag, xuid)
	}
	return tok.SpartanToken, tok.ClearanceToken, nil
}

func fetchSpectate(ctx context.Context, spartan, clearance, matchID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(spectateURL, matchID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-343-authorization-spartan", spartan)
	if clearance != "" {
		req.Header.Set("343-clearance", clearance)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d — %s", resp.StatusCode, truncate(string(body), 300))
	}
	return body, nil
}

// convert extrait le prefixe de blob et la liste des chunks SANS supposer la profondeur a
// laquelle ils se trouvent : on parcourt l'arbre et on prend la premiere chaine qui ressemble
// a un prefixe de blob et le premier tableau dont les elements portent un chemin de chunk.
func convert(raw []byte) (cachedManifest, error) {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return cachedManifest{}, fmt.Errorf("JSON illisible : %w", err)
	}
	var mf cachedManifest
	walk(root, &mf)
	if mf.BlobPrefix == "" {
		return mf, fmt.Errorf("prefixe de blob introuvable dans la reponse")
	}
	if len(mf.Chunks) == 0 {
		return mf, fmt.Errorf("aucun chunk trouve dans la reponse")
	}
	return mf, nil
}

func walk(n any, mf *cachedManifest) {
	switch v := n.(type) {
	case map[string]any:
		for k, val := range v {
			lk := strings.ToLower(k)
			if s, ok := val.(string); ok && mf.BlobPrefix == "" &&
				(strings.Contains(lk, "blobstorage") || strings.Contains(lk, "blob_prefix")) &&
				strings.HasPrefix(s, "http") {
				mf.BlobPrefix = s
			}
			if arr, ok := val.([]any); ok && len(mf.Chunks) == 0 && strings.Contains(lk, "chunk") {
				if cs := asChunks(arr); len(cs) > 0 {
					mf.Chunks = cs
				}
			}
			walk(val, mf)
		}
	case []any:
		for _, e := range v {
			walk(e, mf)
		}
	}
}

// asChunks convertit un tableau en chunks si ses elements portent bien un chemin relatif.
// Les noms de champs de l'API different de ceux du cache : on accepte les deux graphies.
func asChunks(arr []any) []cachedChunk {
	var out []cachedChunk
	for _, e := range arr {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		path := firstString(m, "FileRelativePath", "file_relative_path")
		if path == "" {
			return nil // ce tableau n'est pas une liste de chunks
		}
		out = append(out, cachedChunk{
			Index:            firstInt(m, "Index", "index"),
			ChunkType:        firstInt(m, "ChunkType", "chunk_type"),
			StartMS:          firstInt(m, "ChunkStartTimeOffsetMilliseconds", "start_ms"),
			DurationMS:       firstInt(m, "DurationMilliseconds", "duration_ms"),
			FileRelativePath: path,
		})
	}
	return out
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok {
			return s
		}
	}
	return ""
}

func firstInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if f, ok := m[k].(float64); ok {
			return int(f)
		}
	}
	return 0
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
