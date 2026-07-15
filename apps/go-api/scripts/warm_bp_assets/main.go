// warm_bp_assets — préchargement one-shot de tous les assets Battle Pass.
//
// Lit les tokens depuis db_profiles.json (premier joueur avec un refresh token
// valide dans sync_meta — MSAL cache ou oauth_refresh_token), fetche les
// définitions de tracks + images de couverture/fond + images des paliers
// (InventoryItems) depuis GameCMS, et les persiste dans :
//   - data/cache/bp-track-image/   (images binaires sur FS)
//   - metadata.duckdb asset_index  (index DuckDB)
//   - metadata.duckdb battlepass_item_definitions (upsert direct pour paliers)
//
// Usage :
//
//	go run ./scripts/warm_bp_assets/ [--batch 5] [--delay 600ms]
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// ---------------------------------------------------------------------------
// Structs JSON GameCMS
// ---------------------------------------------------------------------------

type trackDef struct {
	BattlePassImage     string    `json:"BattlePassImage"`
	BackgroundImagePath string    `json:"BackgroundImagePath"`
	XpPerRank           int       `json:"XpPerRank"`
	Ranks               []rankDef `json:"Ranks"`
}

type rankDef struct {
	Rank        int          `json:"Rank"`
	FreeRewards rewardBucket `json:"FreeRewards"`
	PaidRewards rewardBucket `json:"PaidRewards"`
}

type rewardBucket struct {
	InventoryRewards []inventoryReward `json:"InventoryRewards"`
}

type inventoryReward struct {
	InventoryItemPath string `json:"InventoryItemPath"`
}

type itemDef struct {
	CommonData itemCommonData `json:"CommonData"`
}

type itemCommonData struct {
	Quality     string          `json:"Quality"`
	ItemType    string          `json:"ItemType"`
	DisplayPath itemDisplayPath `json:"DisplayPath"`
}

type itemDisplayPath struct {
	Media itemMedia `json:"Media"`
}

type itemMedia struct {
	MediaUrl itemMediaUrl `json:"MediaUrl"`
}

type itemMediaUrl struct {
	Path string `json:"Path"`
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	batchSize := flag.Int("batch", 5, "Nombre de tracks traités en parallèle")
	delay := flag.Duration("delay", 600*time.Millisecond, "Pause entre chaque batch")
	repoRoot := flag.String("repo", ".", "Racine du repo (contient data/)")
	titleSlug := flag.String("title", "halo_infinite", "Slug du titre (halo_infinite)")
	directAccessToken := flag.String("access-token", "", "Access token Microsoft direct (bypass MSAL)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// Charger .env.local pour exposer SPNKR_OAUTH_REFRESH_TOKEN_* et autres vars.
	loadEnvLocal(filepath.Join(*repoRoot, ".env.local"))

	ctx := context.Background()

	// 1. Chemins des bases de données — structure data/titles/{slug}/...
	titleDataDir := filepath.Join(*repoRoot, "data", "titles", *titleSlug)
	metaDBPath := filepath.Join(titleDataDir, "warehouse", "metadata.duckdb")
	cacheRoot := filepath.Join(*repoRoot, "data", "cache")

	if _, err := os.Stat(metaDBPath); os.IsNotExist(err) {
		slog.Error("DB introuvable", "path", metaDBPath)
		os.Exit(1)
	}

	// 1b. Vérification anticipée : metadata.duckdb doit être accessible en écriture.
	// On échoue immédiatement (avant l'auth) pour ne pas gaspiller un Device Code Flow.
	if err := checkMetaDBWritable(metaDBPath); err != nil {
		slog.Error("metadata.duckdb inaccessible en écriture — arrêtez le serveur avant de relancer",
			"err", err,
			"hint", "taskkill //F //IM server.exe")
		os.Exit(1)
	}

	// 2. Obtenir un access_token.
	var accessToken string
	var err error
	if *directAccessToken != "" {
		slog.Info("auth: token fourni directement via --access-token")
		accessToken = *directAccessToken
	} else {
		// Essayer tous les joueurs de db_profiles.json via le provider (MSAL + OAuth v2).
		dbProfilesPath := filepath.Join(*repoRoot, "db_profiles.json")
		accessToken, err = readAccessTokenFromProfiles(ctx, dbProfilesPath, *titleSlug, *repoRoot)
		if err != nil || accessToken == "" {
			// Fallback: watcher_tokens.json (refresh_token si présent)
			slog.Warn("auth: aucun token trouvé dans les joueurs, essai watcher_tokens.json")
			accessToken, err = readWatcherToken(*repoRoot)
		}
		if err != nil || accessToken == "" {
			// Dernier recours: Device Code Flow interactif.
			slog.Warn("auth: aucun token valide trouvé, démarrage du Device Code Flow…")
			accessToken, err = runDeviceFlow(ctx)
			if err == nil && accessToken != "" {
				_ = saveWatcherToken(*repoRoot, accessToken) // persistance best-effort
			}
		}
		if err != nil || accessToken == "" {
			slog.Error("auth: impossible d'obtenir un access_token",
				"hint", "passez --access-token ou relancez le device-flow")
			os.Exit(1)
		}
	}
	slog.Info("auth: access_token OK, échange vers tokens Halo…")

	// 3. Exchange access_token → spartan/clearance. Si l'exchange échoue (token expiré),
	// on retente avec un Device Code Flow interactif.
	result, err := auth.ExchangeAccessToken(ctx, accessToken)
	if err != nil {
		slog.Warn("auth: exchange échoué (token probablement expiré), démarrage Device Code Flow", "err", err)
		accessToken, err = runDeviceFlow(ctx)
		if err != nil || accessToken == "" {
			slog.Error("auth: Device Code Flow échoué", "err", err)
			os.Exit(1)
		}
		_ = saveWatcherToken(*repoRoot, accessToken) // persistance best-effort
		result, err = auth.ExchangeAccessToken(ctx, accessToken)
		if err != nil {
			slog.Error("auth: exchange échoué après device flow", "err", err)
			os.Exit(1)
		}
	}
	haloTokens := result.Tokens
	// S9 (sécurité, lot S) : ne jamais logger de fragment de token (même tronqué).
	// Signal binaire seul.
	slog.Info("auth: spartan token OK")

	// 4. Asset resolver (FS + DuckDB index + GameCMS).
	tokenFn := assets.TokenProvider(func(_ context.Context) (*domain.HaloTokens, error) {
		return haloTokens, nil
	})
	resolver, err := assets.New(assets.AssetConfig{
		CacheRootDir:  cacheRoot,
		MetaDBPath:    metaDBPath,
		TokenProvider: tokenFn,
	})
	if err != nil {
		slog.Error("resolver: init échouée", "err", err)
		os.Exit(1)
	}
	defer resolver.Close(ctx) //nolint:errcheck

	// 5. Lecture des track paths + ouverture unique de metadata.duckdb en écriture.
	// Connexion unique partagée pour loadTrackPaths et tous les upserts — DuckDB n'accepte pas
	// plusieurs connexions read-write concurrentes sur le même fichier.
	metaDB, err := sql.Open("duckdb", metaDBPath)
	if err != nil {
		slog.Error("metadata: ouverture DB échouée", "err", err)
		os.Exit(1)
	}
	metaDB.SetMaxOpenConns(1)
	defer metaDB.Close() //nolint:errcheck

	trackPaths, err := loadTrackPaths(metaDB)
	if err != nil {
		slog.Error("metadata: lecture track paths échouée", "err", err)
		os.Exit(1)
	}
	if len(trackPaths) == 0 {
		slog.Info("battlepass_track_definitions vide — fallback sur battlepass_snapshots des joueurs")
		trackPaths, err = loadTrackPathsFromPlayerDBs(
			filepath.Join(*repoRoot, "db_profiles.json"), *titleSlug, *repoRoot)
		if err != nil {
			slog.Warn("fallback snapshots échoué", "err", err)
		}
	}
	if len(trackPaths) == 0 {
		slog.Warn("metadata: aucun track path trouvé (ni battlepass_track_definitions ni battlepass_snapshots)")
		os.Exit(0)
	}
	slog.Info("tracks à traiter", "count", len(trackPaths))
	var (
		totalTracks int32
		totalImages int32
		totalItems  int32
		totalErrors int32
	)

	httpClient := &http.Client{Timeout: 30 * time.Second}

	for i := 0; i < len(trackPaths); i += *batchSize {
		end := i + *batchSize
		if end > len(trackPaths) {
			end = len(trackPaths)
		}
		batch := trackPaths[i:end]
		slog.Info("batch", "from", i+1, "to", end, "total", len(trackPaths))

		for _, trackPath := range batch {
			n, ni, ne := processTrack(ctx, trackPath, resolver, haloTokens, metaDB, cacheRoot, httpClient)
			atomic.AddInt32(&totalTracks, 1)
			atomic.AddInt32(&totalImages, int32(n))
			atomic.AddInt32(&totalItems, int32(ni))
			atomic.AddInt32(&totalErrors, int32(ne))
		}

		if end < len(trackPaths) {
			slog.Info("pause entre batches", "delay", *delay)
			time.Sleep(*delay)
		}
	}

	// Le defer resolver.Close(ctx) en tête de fonction fermera le WriteQueue
	// et attendra le flush. On laisse quand même 2 s pour les goroutines fire-and-forget.
	time.Sleep(2 * time.Second)

	slog.Info("terminé",
		"tracks", totalTracks,
		"images_fetched", totalImages,
		"items_fetched", totalItems,
		"errors", totalErrors,
	)
}

// processTrack fetche la définition d'un track + toutes ses images + tous ses items.
// Retourne (images_ok, items_ok, errors).
func processTrack(
	ctx context.Context,
	trackPath string,
	resolver *assets.DefaultResolver,
	tokens *domain.HaloTokens,
	metaDB *sql.DB, cacheRoot string,
	httpClient *http.Client,
) (int, int, int) {
	log := slog.With("track", trackPath)

	// a) Fetcher la définition JSON du track (token requis, stocké dans asset_index).
	ref := assets.Ref{Kind: assets.KindRewardTrackDefinition, TitleID: "halo_infinite", ID: trackPath}
	resolved, err := resolver.Get(ctx, ref)
	if err != nil {
		log.Error("track def: erreur", "err", err)
		return 0, 0, 1
	}

	jp, ok := resolved.Payload.(assets.JSONPayload)
	if !ok {
		log.Warn("track def: payload inattendu (non-JSON)")
		return 0, 0, 0
	}

	var def trackDef
	if err := json.Unmarshal(jp.RawJSON, &def); err != nil {
		log.Error("track def: parse JSON échoué", "err", err)
		return 0, 0, 1
	}

	// Persister la définition dans battlepass_track_definitions.
	if err := upsertTrackDef(metaDB, trackPath, &def, jp.RawJSON); err != nil {
		log.Warn("track def: upsert DB échoué", "err", err)
	}

	images := 0
	errors := 0

	// b) Images de couverture et fond du track.
	for _, imgPath := range []string{def.BattlePassImage, def.BackgroundImagePath} {
		imgPath = strings.TrimSpace(imgPath)
		if imgPath == "" {
			continue
		}
		kind := assets.KindBPTrackImage
		if imgPath == def.BackgroundImagePath {
			kind = assets.KindBPBackground
		}
		imgRef := assets.Ref{Kind: kind, TitleID: "halo_infinite", ID: imgPath}
		if _, err := resolver.Get(ctx, imgRef); err != nil {
			log.Warn("image track: erreur", "path", imgPath, "err", err)
			errors++
		} else {
			images++
		}
	}

	// c) Collecter les InventoryItemPaths uniques de tous les paliers.
	itemPaths := collectItemPaths(&def)
	log.Info("items à fetcher", "count", len(itemPaths))

	itemsOK := 0
	for _, itemPath := range itemPaths {
		ok2, imgOK := processItem(ctx, itemPath, tokens, metaDB, cacheRoot, httpClient)
		if ok2 {
			itemsOK++
		} else {
			errors++
		}
		if imgOK {
			images++
		}
	}

	log.Info("track traité", "images", images, "items", itemsOK, "errors", errors)
	return images, itemsOK, errors
}

// processItem fetche le JSON d'un item, l'upsert dans battlepass_item_definitions
// et fetche son image via GameCMS directement (endpoint public /hi/images/file/).
// Retourne (item_ok, image_ok).
func processItem(
	ctx context.Context,
	itemPath string,
	tokens *domain.HaloTokens,
	metaDB *sql.DB, cacheRoot string,
	httpClient *http.Client,
) (bool, bool) {
	log := slog.With("item", itemPath)

	// Fetch JSON depuis GameCMS (authentifié).
	const gameCMSBase = "https://gamecms-hacs.svc.halowaypoint.com"
	cleanPath := strings.TrimLeft(itemPath, "/")
	url := fmt.Sprintf("%s/hi/Progression/file/%s", gameCMSBase, cleanPath)

	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		log.Error("item: création requête", "err", err)
		return false, false
	}
	req.Header.Set("x-343-authorization-spartan", tokens.SpartanToken)
	if tokens.ClearanceToken != "" {
		req.Header.Set("343-clearance", tokens.ClearanceToken)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Error("item: GET échoué", "err", err)
		return false, false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Warn("item: 404")
		return false, false
	}
	if resp.StatusCode != http.StatusOK {
		log.Warn("item: statut inattendu", "status", resp.StatusCode)
		return false, false
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("item: lecture body", "err", err)
		return false, false
	}

	var def itemDef
	if err := json.Unmarshal(raw, &def); err != nil {
		log.Error("item: parse JSON", "err", err)
		return false, false
	}

	displayPath := strings.TrimSpace(def.CommonData.DisplayPath.Media.MediaUrl.Path)

	// Upsert dans battlepass_item_definitions.
	if err := upsertItemDef(metaDB, itemPath, def.CommonData.Quality, def.CommonData.ItemType, displayPath, raw); err != nil {
		log.Warn("item: upsert BDD échoué", "err", err)
		// Non bloquant : l'image peut quand même être fetchée.
	}

	// Fetch image depuis /hi/images/file/ (endpoint public).
	imageOK := false
	if displayPath != "" {
		imgOK := fetchAndSaveBPImage(ctx, displayPath, cacheRoot, tokens, httpClient)
		imageOK = imgOK
		if !imgOK {
			log.Warn("item: image non récupérée", "display_path", displayPath)
		}
	}

	return true, imageOK
}

// fetchAndSaveBPImage télécharge une image depuis /hi/images/file/ et la sauvegarde
// dans data/cache/bp-track-image/halo_infinite/{displayPath}.tier.png (atomique).
func fetchAndSaveBPImage(ctx context.Context, displayPath, cacheRoot string, tokens *domain.HaloTokens, httpClient *http.Client) bool {
	const gameCMSBase = "https://gamecms-hacs.svc.halowaypoint.com"
	cleanPath := strings.TrimLeft(strings.ReplaceAll(displayPath, "\\", "/"), "/")
	url := fmt.Sprintf("%s/hi/images/file/%s", gameCMSBase, cleanPath)

	// Chemin de destination (même convention que LocalFSStore avec variant "tier").
	destPath := filepath.Join(cacheRoot, "bp-track-image", "halo_infinite", filepath.FromSlash(cleanPath)+".tier.png")

	// Si déjà en cache, skip.
	if _, err := os.Stat(destPath); err == nil {
		return true
	}

	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	// Tokens optionnels pour les images publiques.
	if tokens != nil {
		req.Header.Set("x-343-authorization-spartan", tokens.SpartanToken)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return false
	}
	tmp := destPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return false
	}
	if err := os.Rename(tmp, destPath); err != nil {
		_ = os.Remove(tmp)
		return false
	}
	slog.Debug("image sauvegardée", "path", destPath, "bytes", len(data))
	return true
}

// ---------------------------------------------------------------------------
// Helpers DB
// loadEnvLocal lit un fichier .env.local et injecte les variables dans l'environnement
// du processus, sans écraser les variables déjà définies.
func loadEnvLocal(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

// ---------------------------------------------------------------------------

// checkMetaDBWritable ouvre metadata.duckdb en lecture-écriture et la referme immédiatement.
// Retourne une erreur si la DB est verrouillée par un autre processus (ex: server.exe).
// À appeler AVANT l'auth pour ne pas gaspiller un Device Code Flow.
func checkMetaDBWritable(metaDBPath string) error {
	db, err := sql.Open("duckdb", metaDBPath)
	if err != nil {
		return err
	}
	// Ping pour forcer l'ouverture réelle du fichier.
	if err := db.Ping(); err != nil {
		db.Close() //nolint:errcheck
		return err
	}
	return db.Close()
}

// saveWatcherToken persiste l'access_token dans watcher_tokens.json (best-effort).
// Cela évite de devoir refaire un Device Code Flow au prochain lancement du script.
func saveWatcherToken(repoRoot, accessToken string) error {
	path := title.NewPathResolver(repoRoot).WatcherTokensPath()

	// Lire le fichier existant pour conserver les autres champs (xsts_*, gamertag…).
	raw, _ := os.ReadFile(path)
	var existing map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &existing)
	}
	if existing == nil {
		existing = make(map[string]any)
	}

	existing["access_token"] = accessToken
	existing["oauth_expires_at"] = time.Now().Add(1 * time.Hour).Format(time.RFC3339)

	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	slog.Info("auth: access_token sauvegardé dans watcher_tokens.json")
	return nil
}

// ---------------------------------------------------------------------------

// loadTrackPathsFromPlayerDBs charge les reward_track_path depuis battlepass_snapshots
// de chaque joueur quand battlepass_track_definitions est vide.
func loadTrackPathsFromPlayerDBs(dbProfilesPath, titleSlug, repoRoot string) ([]string, error) {
	raw, err := os.ReadFile(dbProfilesPath)
	if err != nil {
		return nil, fmt.Errorf("db_profiles.json: %w", err)
	}
	var profiles struct {
		Profiles map[string]map[string]struct {
			DBPath string `json:"db_path"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(raw, &profiles); err != nil {
		return nil, fmt.Errorf("db_profiles.json parse: %w", err)
	}
	players, ok := profiles.Profiles[titleSlug]
	if !ok || len(players) == 0 {
		return nil, nil
	}

	seen := map[string]struct{}{}
	for gamertag, p := range players {
		dbPath := filepath.Join(repoRoot, filepath.FromSlash(p.DBPath))
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			continue
		}
		pdb, openErr := sql.Open("duckdb", dbPath+"?access_mode=read_only")
		if openErr != nil {
			slog.Warn("track_paths: ouverture DB échouée", "gamertag", gamertag, "err", openErr)
			continue
		}
		pdb.SetMaxOpenConns(1)
		rows, qErr := pdb.Query(`SELECT DISTINCT reward_track_path FROM battlepass_snapshots ORDER BY reward_track_path`)
		if qErr != nil {
			slog.Warn("track_paths: query snapshots échouée", "gamertag", gamertag, "err", qErr)
			pdb.Close() //nolint:errcheck
			continue
		}
		for rows.Next() {
			var tp string
			if scanErr := rows.Scan(&tp); scanErr == nil && tp != "" {
				seen[tp] = struct{}{}
			}
		}
		rows.Close() //nolint:errcheck
		pdb.Close()  //nolint:errcheck
		slog.Info("track_paths: snapshots chargés", "gamertag", gamertag, "count", len(seen))
	}
	paths := make([]string, 0, len(seen))
	for tp := range seen {
		paths = append(paths, tp)
	}
	sort.Strings(paths)
	return paths, nil
}

// loadTrackPaths retourne tous les reward_track_path depuis battlepass_track_definitions.
func loadTrackPaths(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT DISTINCT reward_track_path FROM battlepass_track_definitions WHERE is_current = TRUE ORDER BY reward_track_path`)
	if err != nil {
		return nil, fmt.Errorf("battlepass_track_definitions: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

// upsertTrackDef insère ou met à jour une définition de track dans battlepass_track_definitions.
// Utilise UPDATE-first + INSERT-if-zero pour éviter le FATAL DuckDB sur index recréé.
func upsertTrackDef(db *sql.DB, trackPath string, def *trackDef, raw []byte) error {
	hash := contentHash(raw)
	now := time.Now()

	var xpArg any
	if def.XpPerRank > 0 {
		xpArg = def.XpPerRank
	}
	bpImg := nullStr(strings.TrimSpace(def.BattlePassImage))
	bgImg := nullStr(strings.TrimSpace(def.BackgroundImagePath))

	// Invalider les anciens hashes pour ce track.
	_, _ = db.Exec(`
		UPDATE battlepass_track_definitions
		SET is_current = FALSE, last_seen_at = ?
		WHERE reward_track_path = ? AND content_hash <> ? AND is_current = TRUE`,
		now, trackPath, hash)

	// UPDATE d'abord.
	res, err := db.Exec(`
		UPDATE battlepass_track_definitions
		SET xp_per_rank = ?, battlepass_image_path = ?, background_image_path = ?,
		    raw_payload_json = ?, last_seen_at = ?, is_current = TRUE
		WHERE reward_track_path = ? AND content_hash = ?`,
		xpArg, bpImg, bgImg, string(raw), now,
		trackPath, hash,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		return nil
	}
	_, err = db.Exec(`
		INSERT INTO battlepass_track_definitions
			(reward_track_path, content_hash, xp_per_rank,
			 battlepass_image_path, background_image_path,
			 raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?)`,
		trackPath, hash, xpArg, bpImg, bgImg, string(raw), now, now,
	)
	return err
}

// upsertItemDef insère ou met à jour un item dans battlepass_item_definitions.
func upsertItemDef(db *sql.DB, itemPath, quality, itemType, displayPath string, raw []byte) error {
	hash := contentHash(raw)
	now := time.Now()

	var dpArg any
	if strings.TrimSpace(displayPath) != "" {
		dpArg = displayPath
	}

	// UPDATE d'abord, puis INSERT si aucune ligne touchée.
	// On évite ON CONFLICT DO UPDATE qui déclenche un FATAL DuckDB
	// sur des tables dont l'index interne a été recréé (suffixe _N).
	res, err := db.Exec(`
		UPDATE battlepass_item_definitions
		SET quality = ?, item_type = ?, display_path = ?,
		    raw_payload_json = ?, last_seen_at = ?, is_current = TRUE
		WHERE inventory_item_path = ? AND content_hash = ?`,
		nullStr(quality), nullStr(itemType), dpArg, string(raw), now,
		itemPath, hash,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		return nil
	}
	_, err = db.Exec(`
		INSERT INTO battlepass_item_definitions
			(inventory_item_path, content_hash, quality, item_type, display_path,
			 raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?)`,
		itemPath, hash, nullStr(quality), nullStr(itemType), dpArg, string(raw), now, now,
	)
	return err
}

// readAccessToken lit le cache MSAL ou l'oauth_refresh_token depuis sync_meta
// et retourne un access_token Microsoft valide.
//
// readAccessTokenFromProfiles itère sur tous les joueurs de db_profiles.json
// pour le titleSlug donné et tente, via le provider, un refresh MSAL silencieux
// ou OAuth v2 sur chaque sync_meta. Retourne le premier access_token obtenu.
func readAccessTokenFromProfiles(ctx context.Context, dbProfilesPath, titleSlug, repoRoot string) (string, error) {
	raw, err := os.ReadFile(dbProfilesPath)
	if err != nil {
		return "", fmt.Errorf("db_profiles.json: %w", err)
	}
	var profiles struct {
		Profiles map[string]map[string]struct {
			DBPath string `json:"db_path"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(raw, &profiles); err != nil {
		return "", fmt.Errorf("db_profiles.json parse: %w", err)
	}
	players, ok := profiles.Profiles[titleSlug]
	if !ok || len(players) == 0 {
		return "", fmt.Errorf("aucun joueur pour le titre %q dans db_profiles.json", titleSlug)
	}

	provider := auth.NewSISUProvider()

	for gamertag, p := range players {
		dbPath := filepath.Join(repoRoot, filepath.FromSlash(p.DBPath))
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			slog.Debug("auth: DB absente, joueur ignoré", "gamertag", gamertag, "path", dbPath)
			continue
		}

		db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
		if err != nil {
			slog.Warn("auth: ouverture DB échouée", "gamertag", gamertag, "err", err)
			continue
		}
		db.SetMaxOpenConns(1)

		var cacheJSON, refreshToken string
		_ = db.QueryRowContext(ctx, "SELECT value FROM sync_meta WHERE key = 'msal_token_cache'").Scan(&cacheJSON)
		_ = db.QueryRowContext(ctx, "SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'").Scan(&refreshToken)
		db.Close() //nolint:errcheck

		if cacheJSON != "" {
			token, err := provider.TrySilentRefresh(ctx, cacheJSON)
			if err == nil && token != "" {
				slog.Info("auth: MSAL silent refresh OK", "gamertag", gamertag)
				return token, nil
			}
			slog.Warn("auth: MSAL silent refresh raté", "gamertag", gamertag, "err", err)
		}

		if refreshToken != "" {
			token, err := provider.TryOAuthRefresh(ctx, refreshToken)
			if err == nil && token != "" {
				slog.Info("auth: OAuth v2 refresh OK", "gamertag", gamertag)
				return token, nil
			}
			slog.Warn("auth: OAuth v2 refresh raté", "gamertag", gamertag, "err", err)
		}

		// Fallback: SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> depuis .env.local
		envKey := "SPNKR_OAUTH_REFRESH_TOKEN_" + strings.ToUpper(gamertag)
		if envRT := os.Getenv(envKey); envRT != "" {
			token, err := provider.TryOAuthRefresh(ctx, envRT)
			if err == nil && token != "" {
				slog.Info("auth: OAuth v2 refresh OK (env var)", "gamertag", gamertag, "env", envKey)
				return token, nil
			}
			slog.Warn("auth: OAuth v2 refresh raté (env var)", "gamertag", gamertag, "err", err)
		}

		slog.Debug("auth: aucun token utilisable", "gamertag", gamertag)
	}

	return "", nil
}

// runDeviceFlow déclenche un Device Code Flow interactif (SISU) et retourne l'access_token.
func runDeviceFlow(ctx context.Context) (string, error) {
	slog.Info("auth: init Device Code Flow…")
	flow, err := auth.NewSISUProvider().InitDeviceFlow(ctx)
	if err != nil {
		return "", fmt.Errorf("device flow init: %w", err)
	}
	// Afficher les instructions clairement.
	fmt.Printf("\n=== AUTHENTIFICATION REQUISE ===\n")
	fmt.Printf("1. Ouvrez :  %s\n", flow.GetVerificationURL())
	fmt.Printf("2. Entrez le code :  %s\n", flow.GetUserCode())
	fmt.Printf("3. Connectez-vous avec votre compte Microsoft Xbox\n")
	fmt.Printf("Code valide %d secondes.\n\n", flow.GetExpiresIn())

	// Attendre la complétion (bloquant).
	token, err := flow.AcquireToken(ctx)
	if err != nil {
		return "", fmt.Errorf("device flow acquire: %w", err)
	}
	slog.Info("auth: Device Code Flow OK — token acquis")
	return token, nil
}

// readWatcherToken lit l'access_token depuis data/cache/watcher_tokens.json.
func readWatcherToken(repoRoot string) (string, error) {
	path := title.NewPathResolver(repoRoot).WatcherTokensPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var obj struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", err
	}
	if obj.AccessToken == "" {
		return "", fmt.Errorf("watcher_tokens.json: access_token vide")
	}
	slog.Info("auth: token lu depuis watcher_tokens.json")
	return obj.AccessToken, nil
}

// ---------------------------------------------------------------------------
// Utilitaires
// ---------------------------------------------------------------------------

func collectItemPaths(def *trackDef) []string {
	seen := map[string]struct{}{}
	for _, rank := range def.Ranks {
		for _, r := range rank.FreeRewards.InventoryRewards {
			if p := strings.TrimSpace(r.InventoryItemPath); p != "" {
				seen[p] = struct{}{}
			}
		}
		for _, r := range rank.PaidRewards.InventoryRewards {
			if p := strings.TrimSpace(r.InventoryItemPath); p != "" {
				seen[p] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	return out
}

func contentHash(data []byte) string {
	// SHA256[:8] en hex — même convention que le resolver Go.
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8])
}

func nullStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
