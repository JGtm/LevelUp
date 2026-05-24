// cmd/gen_test_fixtures — orchestration de la generation des fixtures de
// test (G.1 du PLAN_FIX_SYNC_TESTS_STRATEGY_2026-05-24).
//
// 3 modes :
//
//	complete-chunks : telecharge les chunks manquants (00, 01, etc.) pour
//	  les matchs locaux dans external_film_dir, en utilisant le manifest
//	  correspondant pour reconstruire l'URL blob CDN public.
//	  Usage : gen_test_fixtures complete-chunks \
//	            -src C:/Users/Guillaume/Downloads/film_chunks \
//	            -manifests C:/Users/Guillaume/Downloads/film_manifests
//
//	download-full-match : telecharge un match complet (manifest + tous les
//	  chunks + 3 API responses) dans testdata/<name>_full_match/. Tokens
//	  SpartanToken + ClearanceToken requis (via env ou flags).
//	  Usage : gen_test_fixtures download-full-match \
//	            -match-id b71d39db-... \
//	            -xuid 2533274823110022 \
//	            -spartan "$SPARTAN" \
//	            -clearance "$CLEARANCE" \
//	            -dest internal/sync/testdata/jgtm_full_match
//
//	list-external : audit du dataset externe — compte les matchs, identifie
//	  les chunks manquants pour chacun.
//	  Usage : gen_test_fixtures list-external \
//	            -src C:/Users/Guillaume/Downloads/film_chunks \
//	            -manifests C:/Users/Guillaume/Downloads/film_manifests
//
// Reference :
//   - internal/sync/testdata/jgtm_full_match/README.md (procedure manuelle)
//   - internal/testfixtures/jgtm_full_match.go (loaders)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	subcmd := os.Args[1]
	args := os.Args[2:]

	switch subcmd {
	case "complete-chunks":
		runCompleteChunks(args)
	case "download-full-match":
		runDownloadFullMatch(args)
	case "list-external":
		runListExternal(args)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "ERROR: unknown subcommand %q\n\n", subcmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println(`gen_test_fixtures — generation des fixtures de test sync

Usage :
  gen_test_fixtures complete-chunks  -src <dir> -manifests <dir>
  gen_test_fixtures download-full-match -match-id <uuid> -xuid <id> -dest <dir> [-spartan TOK -clearance TOK]
  gen_test_fixtures list-external    -src <dir> -manifests <dir>

Variables d'env utilisees si flags absents :
  LEVELUP_SPARTAN_TOKEN   (download-full-match)
  LEVELUP_CLEARANCE_TOKEN (download-full-match)`)
}

// ─────────────────────────────────────────────────────────────────────────
// complete-chunks
// ─────────────────────────────────────────────────────────────────────────

type filmManifest struct {
	BlobStoragePathPrefix string `json:"BlobStoragePathPrefix"`
	CustomData            struct {
		Chunks []struct {
			Index            int    `json:"Index"`
			ChunkType        int    `json:"ChunkType"`
			ChunkSize        int    `json:"ChunkSize"`
			FileRelativePath string `json:"FileRelativePath"`
		} `json:"Chunks"`
	} `json:"CustomData"`
}

func runCompleteChunks(args []string) {
	fs := flag.NewFlagSet("complete-chunks", flag.ExitOnError)
	src := fs.String("src", "", "Dossier racine des chunks (ex: film_chunks)")
	manifests := fs.String("manifests", "", "Dossier des manifests (ex: film_manifests)")
	dryRun := fs.Bool("dry-run", false, "Ne telecharge pas, liste seulement les manquants")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *src == "" || *manifests == "" {
		fs.Usage()
		os.Exit(2)
	}

	entries, err := os.ReadDir(*src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: read src: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var (
		matchesAudit  int
		chunksToFetch int
		downloaded    int
		failed        int
	)
	for _, dir := range entries {
		if !dir.IsDir() {
			continue
		}
		matchesAudit++
		shortID := dir.Name()
		manifestPath := filepath.Join(*manifests, shortID+".json")
		mfRaw, err := os.ReadFile(manifestPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: manifest absent\n", shortID)
			continue
		}
		var mf filmManifest
		if err := json.Unmarshal(mfRaw, &mf); err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: manifest invalide: %v\n", shortID, err)
			continue
		}
		chunksDir := filepath.Join(*src, shortID)
		for _, ch := range mf.CustomData.Chunks {
			localPath := filepath.Join(chunksDir, fmt.Sprintf("chunk_%02d.bin", ch.Index))
			if _, err := os.Stat(localPath); err == nil {
				continue // deja present
			}
			chunksToFetch++
			if *dryRun {
				fmt.Printf("MISSING %s chunk_%02d.bin (type=%d, size=%d)\n",
					shortID, ch.Index, ch.ChunkType, ch.ChunkSize)
				continue
			}
			url := mf.BlobStoragePathPrefix +
				strings.TrimLeft(ch.FileRelativePath, "/")
			if err := downloadFile(client, url, localPath); err != nil {
				failed++
				fmt.Fprintf(os.Stderr, "FAIL %s chunk_%02d: %v\n", shortID, ch.Index, err)
			} else {
				downloaded++
				if downloaded%10 == 0 {
					fmt.Printf("downloaded %d chunks...\n", downloaded)
				}
			}
		}
	}

	fmt.Println()
	fmt.Println("─── COMPLETE CHUNKS ───")
	fmt.Printf("matches audites : %d\n", matchesAudit)
	fmt.Printf("chunks manquants: %d\n", chunksToFetch)
	if *dryRun {
		fmt.Println("(dry-run : aucun telechargement effectue)")
	} else {
		fmt.Printf("downloaded      : %d\n", downloaded)
		fmt.Printf("failed          : %d\n", failed)
	}
}

// ─────────────────────────────────────────────────────────────────────────
// download-full-match
// ─────────────────────────────────────────────────────────────────────────

func runDownloadFullMatch(args []string) {
	fs := flag.NewFlagSet("download-full-match", flag.ExitOnError)
	matchID := fs.String("match-id", "", "Match UUID complet")
	xuid := fs.String("xuid", "", "XUID du joueur owner")
	dest := fs.String("dest", "", "Dossier de destination (ex: internal/sync/testdata/jgtm_full_match)")
	spartan := fs.String("spartan", os.Getenv("LEVELUP_SPARTAN_TOKEN"), "SpartanToken (ou env LEVELUP_SPARTAN_TOKEN)")
	clearance := fs.String("clearance", os.Getenv("LEVELUP_CLEARANCE_TOKEN"), "ClearanceToken (ou env LEVELUP_CLEARANCE_TOKEN)")
	userAgent := fs.String("ua", "SHIVA-2043073184/06.10122.05904.0 (release; PC)", "User-Agent")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *matchID == "" || *xuid == "" || *dest == "" {
		fs.Usage()
		fmt.Fprintln(os.Stderr, "ERROR: -match-id, -xuid, -dest requis")
		os.Exit(2)
	}
	if *spartan == "" || *clearance == "" {
		fmt.Fprintln(os.Stderr, "ERROR: SpartanToken + ClearanceToken requis (cf. cmd/get-token)")
		os.Exit(2)
	}

	if err := os.MkdirAll(filepath.Join(*dest, "chunks"), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: mkdir: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	hdr := func(req *http.Request) {
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-343-Authorization-Spartan", *spartan)
		req.Header.Set("343-Clearance", *clearance)
		req.Header.Set("User-Agent", *userAgent)
	}

	// 1. Manifest film
	manifestURL := fmt.Sprintf(
		"https://discovery-infiniteugc.svc.halowaypoint.com/hi/films/matches/%s/spectate",
		*matchID)
	if err := fetchJSON(client, manifestURL, hdr,
		filepath.Join(*dest, "manifest_raw.json")); err != nil {
		fmt.Fprintf(os.Stderr, "manifest fetch: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK manifest_raw.json")

	// 2. Decode manifest pour blob prefix + chunks
	mfRaw, _ := os.ReadFile(filepath.Join(*dest, "manifest_raw.json"))
	var mf filmManifest
	if err := json.Unmarshal(mfRaw, &mf); err != nil {
		fmt.Fprintf(os.Stderr, "manifest decode: %v\n", err)
		os.Exit(1)
	}

	// 3. Chunks (CDN public, sans auth)
	chunksDir := filepath.Join(*dest, "chunks")
	for _, ch := range mf.CustomData.Chunks {
		url := mf.BlobStoragePathPrefix + strings.TrimLeft(ch.FileRelativePath, "/")
		path := filepath.Join(chunksDir, fmt.Sprintf("filmChunk%d", ch.Index))
		if err := downloadFile(client, url, path); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL chunk %d: %v\n", ch.Index, err)
			continue
		}
	}
	fmt.Printf("OK %d chunks dans chunks/\n", len(mf.CustomData.Chunks))

	// 4. API responses
	endpoints := []struct {
		url  string
		file string
	}{
		{
			fmt.Sprintf("https://halostats.svc.halowaypoint.com/hi/matches/%s/stats", *matchID),
			"api_match_stats.json",
		},
		{
			fmt.Sprintf("https://skill.svc.halowaypoint.com/hi/matches/%s/skill?players=xuid(%s)", *matchID, *xuid),
			"api_skill.json",
		},
		{
			fmt.Sprintf("https://halostats.svc.halowaypoint.com/hi/players/xuid(%s)/matches?start=0&count=5", *xuid),
			"api_match_history_page0.json",
		},
	}
	for _, ep := range endpoints {
		if err := fetchJSON(client, ep.url, hdr, filepath.Join(*dest, ep.file)); err != nil {
			fmt.Fprintf(os.Stderr, "WARN %s: %v\n", ep.file, err)
			continue
		}
		fmt.Printf("OK %s\n", ep.file)
	}

	fmt.Println()
	fmt.Printf("Fixture cree dans : %s\n", *dest)
}

// ─────────────────────────────────────────────────────────────────────────
// list-external
// ─────────────────────────────────────────────────────────────────────────

func runListExternal(args []string) {
	fs := flag.NewFlagSet("list-external", flag.ExitOnError)
	src := fs.String("src", "", "Dossier racine des chunks")
	manifests := fs.String("manifests", "", "Dossier des manifests")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *src == "" || *manifests == "" {
		fs.Usage()
		os.Exit(2)
	}

	entries, err := os.ReadDir(*src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	var totalMatches, totalMissing int
	for _, dir := range entries {
		if !dir.IsDir() {
			continue
		}
		totalMatches++
		shortID := dir.Name()
		mfRaw, err := os.ReadFile(filepath.Join(*manifests, shortID+".json"))
		if err != nil {
			fmt.Printf("%s : manifest absent\n", shortID)
			continue
		}
		var mf filmManifest
		if err := json.Unmarshal(mfRaw, &mf); err != nil {
			fmt.Printf("%s : manifest invalide\n", shortID)
			continue
		}
		chunksDir := filepath.Join(*src, shortID)
		missing := 0
		for _, ch := range mf.CustomData.Chunks {
			localPath := filepath.Join(chunksDir, fmt.Sprintf("chunk_%02d.bin", ch.Index))
			if _, err := os.Stat(localPath); err != nil {
				missing++
			}
		}
		if missing > 0 {
			fmt.Printf("%s : %d/%d chunks manquants\n",
				shortID, missing, len(mf.CustomData.Chunks))
			totalMissing += missing
		}
	}
	fmt.Println()
	fmt.Printf("Total : %d matches, %d chunks manquants au total\n",
		totalMatches, totalMissing)
}

// ─────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────

func fetchJSON(client *http.Client, url string, hdrFn func(*http.Request), dest string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	hdrFn(req)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return fmt.Errorf("empty body")
	}
	return os.WriteFile(dest, body, 0o644)
}

func downloadFile(client *http.Client, url, dest string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
