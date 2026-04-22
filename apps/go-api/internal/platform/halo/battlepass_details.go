// Package halo — battlepass_details.go : persistance des définitions de tracks Battle Pass.
//
// Pattern symétrique à challenges_details.go :
//   - loadTrackDefinitionFromMetadata : cache metadata → nil si absent
//   - storeTrackDefinitionInMetadata  : UPDATE is_current=false + UPSERT definitions + translations
//   - fetchRewardTrackDefinition      : cache metadata → fallback GameCMS → store
//
// La source de vérité est GameCMS (/hi/Progression/file/{trackPath}), pas le payload
// d'opérations /economy/operations qui ne contient que la progression joueur.
package halo

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// bpMetaWriteMu sérialise toutes les écritures dans metadata.duckdb depuis ce package.
// DuckDB ne supporte pas les transactions concurrentes sur le même fichier.
// Sans ce mutex, les 30 goroutines preCacheBPItemDefinitions (une par track)
// créent des violations de clé primaire → FatalException.
var bpMetaWriteMu sync.Mutex

// battlepassTrackDefinitionRaw représente la définition d'un Reward Track depuis GameCMS.
// Structure : /hi/Progression/file/{RewardTrackPath}
type battlepassTrackDefinitionRaw struct {
	Name                any                    `json:"Name"`
	Description         any                    `json:"Description"`
	BattlePassImage     string                 `json:"BattlePassImage"`
	BackgroundImagePath string                 `json:"BackgroundImagePath"`
	XpPerRank           int                    `json:"XpPerRank"`
	Ranks               []battlepassRankDefRaw `json:"Ranks"`
}

type battlepassRankDefRaw struct {
	Rank        int                       `json:"Rank"`
	FreeRewards battlepassRewardBucketRaw `json:"FreeRewards"`
	PaidRewards battlepassRewardBucketRaw `json:"PaidRewards"`
}

type battlepassRewardBucketRaw struct {
	InventoryRewards []battlepassInventoryRewardRaw `json:"InventoryRewards"`
}

type battlepassInventoryRewardRaw struct {
	InventoryItemPath string `json:"InventoryItemPath"`
	Amount            int    `json:"Amount"`
}

// battlepassItemDefinitionRaw représente le JSON d'un item inventaire depuis GameCMS.
// Structure : GET /hi/Progression/file/{InventoryItemPath}
type battlepassItemDefinitionRaw struct {
	CommonData battlepassItemCommonDataRaw `json:"CommonData"`
}

type battlepassItemCommonDataRaw struct {
	Title       any                          `json:"Title"`
	Description any                          `json:"Description"`
	Quality     string                       `json:"Quality"`
	ItemType    string                       `json:"ItemType"`
	DisplayPath battlepassItemDisplayPathRaw `json:"DisplayPath"`
}

type battlepassItemDisplayPathRaw struct {
	Media battlepassItemMediaRaw `json:"Media"`
}

type battlepassItemMediaRaw struct {
	MediaUrl battlepassItemMediaUrlRaw `json:"MediaUrl"`
}

type battlepassItemMediaUrlRaw struct {
	Path string `json:"Path"`
}

// fetchRewardTrackDefinition charge la définition d'un track depuis le cache metadata,
// ou le fetche depuis GameCMS en cas de miss, puis le persiste.
// Retourne nil immédiatement si battlepassMetaPath est vide (pas de persistance → fetch inutile).
func (p *HaloProvider) fetchRewardTrackDefinition(
	ctx context.Context,
	tokens *domain.HaloTokens,
	trackPath string,
) *battlepassTrackDefinitionRaw {
	trimmed := strings.TrimSpace(trackPath)
	if trimmed == "" {
		return nil
	}
	// Sans chemin de persistance il n'y a rien à stocker : skip silencieux.
	if strings.TrimSpace(p.battlepassMetaPath) == "" {
		return nil
	}

	sfKey := strings.TrimSpace(p.battlepassMetaPath) + "|" + trimmed
	value, err, _ := battlePassTrackFetchSFGroup.Do(sfKey, func() (any, error) {
		if cached, err := p.loadTrackDefinitionFromMetadata(ctx, trimmed); err == nil && cached != nil {
			slog.DebugContext(ctx, "halo_provider: reward track definition served from metadata cache",
				"path", trimmed)
			// Pré-cacher les définitions d'items manquantes en arrière-plan (best-effort).
			var tokCopy domain.HaloTokens
			if tokens != nil {
				tokCopy = *tokens
			}
			go p.preCacheBPItemDefinitions(cached.Ranks, tokCopy)
			return cached, nil
		} else if err != nil {
			slog.DebugContext(ctx, "halo_provider: reward track definition metadata cache read failed",
				"path", trimmed, "err", err)
		}

		base := p.gameCMSBaseURL
		if base == "" {
			base = defaultGameCMSHost
		}
		url := fmt.Sprintf("%s/hi/Progression/file/%s", strings.TrimRight(base, "/"), strings.TrimLeft(trimmed, "/"))
		body, err := p.doGet(ctx, url, tokens)
		if err != nil {
			return nil, err
		}

		var def battlepassTrackDefinitionRaw
		if err := json.Unmarshal(body, &def); err != nil {
			return nil, fmt.Errorf("decode reward track definition: %w", err)
		}

		if err := p.storeTrackDefinitionInMetadata(ctx, trimmed, body, &def); err != nil {
			slog.WarnContext(ctx, "halo_provider: reward track definition metadata cache write failed",
				"path", trimmed, "err", err)
		} else {
			slog.InfoContext(ctx, "halo_provider: reward track definition persisted from GameCMS",
				"path", trimmed, "xp_per_rank", def.XpPerRank)
			// Pré-cacher les images du track en arrière-plan (fire-and-forget).
			// Tokens copiés par valeur pour éviter toute race condition.
			var tokCopy domain.HaloTokens
			if tokens != nil {
				tokCopy = *tokens
			}
			go p.preCacheBPTrackImages(def.BattlePassImage, def.BackgroundImagePath, tokCopy)
			// Pré-cacher les définitions d'items en arrière-plan (best-effort).
			go p.preCacheBPItemDefinitions(def.Ranks, tokCopy)
		}

		return &def, nil
	})
	if err != nil {
		slog.DebugContext(ctx, "halo_provider: reward track definition fetch failed",
			"path", trimmed, "err", err)
		return nil
	}
	def, _ := value.(*battlepassTrackDefinitionRaw)
	return def
}

// ---------------------------------------------------------------------------
// Cache local des définitions d'items inventaire Battle Pass
// ---------------------------------------------------------------------------

// extractItemPathsFromRanks collecte tous les InventoryItemPath uniques des ranks.
func extractItemPathsFromRanks(ranks []battlepassRankDefRaw) []string {
	seen := map[string]struct{}{}
	for _, rank := range ranks {
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
	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	return paths
}

// loadExistingItemPaths retourne l'ensemble des inventory_item_path déjà connus
// dans battlepass_item_definitions (is_current=TRUE).
func (p *HaloProvider) loadExistingItemPaths(ctx context.Context) map[string]struct{} {
	metaPath := strings.TrimSpace(p.battlepassMetaPath)
	if metaPath == "" {
		return map[string]struct{}{}
	}
	db, err := duckdb.OpenReadOnly(metaPath)
	if err != nil {
		return map[string]struct{}{}
	}
	defer db.Close()
	rows, err := db.Query(ctx, `SELECT DISTINCT inventory_item_path FROM battlepass_item_definitions WHERE is_current = TRUE`)
	if err != nil {
		return map[string]struct{}{}
	}
	defer rows.Close()
	existing := map[string]struct{}{}
	for rows.Next() {
		var itemPath string
		if err := rows.Scan(&itemPath); err == nil && itemPath != "" {
			existing[itemPath] = struct{}{}
		}
	}
	return existing
}

// preCacheBPItemDefinitions fetche en arrière-plan les définitions d'items inventaire
// manquants dans battlepass_item_definitions.
// Appelé dans une goroutine ; les tokens sont passés par valeur pour éviter les races.
func (p *HaloProvider) preCacheBPItemDefinitions(ranks []battlepassRankDefRaw, tokens domain.HaloTokens) {
	if len(ranks) == 0 {
		return
	}
	ctx := context.Background()
	allPaths := extractItemPathsFromRanks(ranks)
	if len(allPaths) == 0 {
		return
	}
	existing := p.loadExistingItemPaths(ctx)
	fetched := 0
	for _, itemPath := range allPaths {
		if _, ok := existing[itemPath]; ok {
			continue
		}
		if err := p.fetchAndStoreItemDefinition(ctx, itemPath, &tokens); err != nil {
			slog.DebugContext(ctx, "battlepass: item definition fetch failed",
				"path", itemPath, "err", err)
		} else {
			fetched++
		}
	}
	if fetched > 0 {
		slog.InfoContext(ctx, "battlepass: item definitions pre-cached", "count", fetched)
	}
}

// fetchAndStoreItemDefinition récupère la définition JSON d'un item inventaire depuis GameCMS
// (/hi/Progression/file/{itemPath}) et la persiste dans battlepass_item_definitions.
func (p *HaloProvider) fetchAndStoreItemDefinition(ctx context.Context, itemPath string, tokens *domain.HaloTokens) error {
	base := p.gameCMSBaseURL
	if base == "" {
		base = defaultGameCMSHost
	}
	url := fmt.Sprintf("%s/hi/Progression/file/%s",
		strings.TrimRight(base, "/"),
		strings.TrimLeft(strings.TrimSpace(itemPath), "/"),
	)
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return fmt.Errorf("item definition fetch %s: %w", itemPath, err)
	}
	var def battlepassItemDefinitionRaw
	if err := json.Unmarshal(body, &def); err != nil {
		return fmt.Errorf("item definition decode %s: %w", itemPath, err)
	}
	return p.storeItemDefinitionInMetadata(ctx, itemPath, body, &def)
}

// storeItemDefinitionInMetadata persiste une définition d'item inventaire dans metadata.duckdb :
//  1. Marque is_current=false pour les anciens content_hash du même item.
//  2. UPSERT dans battlepass_item_definitions.
//  3. UPSERT dans battlepass_item_translations pour chaque langue disponible.
func (p *HaloProvider) storeItemDefinitionInMetadata(
	ctx context.Context,
	itemPath string,
	body []byte,
	def *battlepassItemDefinitionRaw,
) error {
	metaPath := strings.TrimSpace(p.battlepassMetaPath)
	if metaPath == "" || def == nil || len(body) == 0 {
		return nil
	}

	bpMetaWriteMu.Lock()
	defer bpMetaWriteMu.Unlock()

	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return err
	}
	defer db.Close()

	contentHash := trackDefinitionContentHash(body)
	now := time.Now()
	displayPath := strings.TrimSpace(def.CommonData.DisplayPath.Media.MediaUrl.Path)
	quality := strings.TrimSpace(def.CommonData.Quality)
	itemType := strings.TrimSpace(def.CommonData.ItemType)

	tx, err := db.SQLDb().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storeItemDefinition: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 1. Invalider les anciens hashes.
	if _, err = tx.ExecContext(ctx, `
		UPDATE battlepass_item_definitions
		SET is_current = FALSE, last_seen_at = ?
		WHERE inventory_item_path = ? AND content_hash <> ? AND is_current = TRUE`,
		now, itemPath, contentHash); err != nil {
		return fmt.Errorf("storeItemDefinition: update is_current: %w", err)
	}

	// 2. UPSERT battlepass_item_definitions.
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO battlepass_item_definitions
			(inventory_item_path, content_hash, quality, item_type, display_path,
			 raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?)
		ON CONFLICT (inventory_item_path, content_hash) DO UPDATE SET
			quality          = excluded.quality,
			item_type        = excluded.item_type,
			display_path     = excluded.display_path,
			raw_payload_json = excluded.raw_payload_json,
			last_seen_at     = excluded.last_seen_at,
			is_current       = TRUE`,
		itemPath, contentHash,
		nullableTrackString(quality),
		nullableTrackString(itemType),
		nullableTrackString(displayPath),
		string(body),
		now, now,
	); err != nil {
		return fmt.Errorf("storeItemDefinition: upsert: %w", err)
	}

	// 3. UPSERT battlepass_item_translations.
	titleTranslations := collectTrackTranslations(def.CommonData.Title)
	descTranslations := collectTrackTranslations(def.CommonData.Description)
	langs := make(map[string]struct{}, len(titleTranslations)+len(descTranslations))
	for lang := range titleTranslations {
		langs[lang] = struct{}{}
	}
	for lang := range descTranslations {
		langs[lang] = struct{}{}
	}
	for lang := range langs {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO battlepass_item_translations
				(inventory_item_path, content_hash, lang, title, description, first_seen_at, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (inventory_item_path, content_hash, lang) DO UPDATE SET
				title        = excluded.title,
				description  = excluded.description,
				last_seen_at = excluded.last_seen_at`,
			itemPath, contentHash, lang,
			nullableTrackString(titleTranslations[lang]),
			nullableTrackString(descTranslations[lang]),
			now, now,
		); err != nil {
			return fmt.Errorf("storeItemDefinition: upsert translations lang=%s: %w", lang, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("storeItemDefinition: commit: %w", err)
	}
	slog.DebugContext(ctx, "battlepass: item definition persisted",
		"path", itemPath, "display_path", displayPath)
	return nil
}

// loadTrackDefinitionFromMetadata charge la définition depuis battlepass_track_definitions
// avec JOIN battlepass_track_translations (COALESCE fr-FR / en-US).
// Retourne nil sans erreur si la table est absente ou si aucune entrée n'existe.
func (p *HaloProvider) loadTrackDefinitionFromMetadata(
	ctx context.Context,
	trackPath string,
) (*battlepassTrackDefinitionRaw, error) {
	metaPath := strings.TrimSpace(p.battlepassMetaPath)
	if metaPath == "" {
		return nil, nil
	}

	db, err := duckdb.OpenReadOnly(metaPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	row := db.QueryRow(ctx, `
		SELECT d.xp_per_rank,
		       d.battlepass_image_path,
		       d.background_image_path,
		       d.raw_payload_json,
		       COALESCE(t_fr.track_name, t_en.track_name) AS track_name
		FROM battlepass_track_definitions d
		LEFT JOIN battlepass_track_translations t_fr
		       ON t_fr.reward_track_path = d.reward_track_path
		      AND t_fr.content_hash = d.content_hash
		      AND t_fr.lang = 'fr-FR'
		LEFT JOIN battlepass_track_translations t_en
		       ON t_en.reward_track_path = d.reward_track_path
		      AND t_en.content_hash = d.content_hash
		      AND t_en.lang = 'en-US'
		WHERE d.reward_track_path = ? AND d.is_current = TRUE
		ORDER BY d.last_seen_at DESC
		LIMIT 1`, trackPath)

	var xpPerRank sql.NullInt64
	var bpImagePath sql.NullString
	var bgImagePath sql.NullString
	var rawPayload sql.NullString
	var trackName sql.NullString

	if err := row.Scan(&xpPerRank, &bpImagePath, &bgImagePath, &rawPayload, &trackName); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Reconstruire le struct depuis raw_payload_json (source de vérité), enrichi des colonnes indexées.
	def := &battlepassTrackDefinitionRaw{}
	if rawPayload.Valid && strings.TrimSpace(rawPayload.String) != "" {
		_ = json.Unmarshal([]byte(rawPayload.String), def)
	}
	if xpPerRank.Valid {
		def.XpPerRank = int(xpPerRank.Int64)
	}
	if bpImagePath.Valid && bpImagePath.String != "" {
		def.BattlePassImage = bpImagePath.String
	}
	if bgImagePath.Valid && bgImagePath.String != "" {
		def.BackgroundImagePath = bgImagePath.String
	}
	// Injecter le nom localisé depuis les translations si le JSON ne le fournit pas.
	if trackName.Valid && trackName.String != "" {
		nameStr := resolveChallengeLocalizedValue(def.Name, "fr-FR")
		if nameStr == "" {
			def.Name = trackName.String
		}
	}

	return def, nil
}

// storeTrackDefinitionInMetadata persiste une définition de track dans metadata.duckdb :
//  1. Marque is_current=false pour les anciens content_hash du même track.
//  2. UPSERT dans battlepass_track_definitions (raw_payload_json + colonnes indexées).
//  3. UPSERT dans battlepass_track_translations pour chaque langue disponible.
func (p *HaloProvider) storeTrackDefinitionInMetadata(
	ctx context.Context,
	trackPath string,
	body []byte,
	def *battlepassTrackDefinitionRaw,
) error {
	metaPath := strings.TrimSpace(p.battlepassMetaPath)
	if metaPath == "" || def == nil || len(body) == 0 {
		return nil
	}
	bpMetaWriteMu.Lock()
	defer bpMetaWriteMu.Unlock()
	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return err
	}
	defer db.Close()

	contentHash := trackDefinitionContentHash(body)
	now := time.Now()

	tx, err := db.SQLDb().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storeTrackDefinition: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 1. Invalider les anciens hashes.
	if _, err = tx.ExecContext(ctx, `
		UPDATE battlepass_track_definitions
		SET is_current = FALSE,
		    last_seen_at = ?
		WHERE reward_track_path = ?
		  AND content_hash <> ?
		  AND is_current = TRUE`, now, trackPath, contentHash); err != nil {
		return fmt.Errorf("storeTrackDefinition: update is_current: %w", err)
	}

	// 2. UPSERT battlepass_track_definitions.
	bpImage := strings.TrimSpace(def.BattlePassImage)
	bgImage := strings.TrimSpace(def.BackgroundImagePath)
	var xpPerRank any
	if def.XpPerRank > 0 {
		xpPerRank = def.XpPerRank
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO battlepass_track_definitions
			(reward_track_path, content_hash, xp_per_rank,
			 battlepass_image_path, background_image_path,
			 raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?)
		ON CONFLICT (reward_track_path, content_hash) DO UPDATE SET
			xp_per_rank             = excluded.xp_per_rank,
			battlepass_image_path   = excluded.battlepass_image_path,
			background_image_path   = excluded.background_image_path,
			raw_payload_json        = excluded.raw_payload_json,
			last_seen_at            = excluded.last_seen_at,
			is_current              = TRUE`,
		trackPath,
		contentHash,
		xpPerRank,
		nullableTrackString(bpImage),
		nullableTrackString(bgImage),
		string(body),
		now,
		now,
	); err != nil {
		return fmt.Errorf("storeTrackDefinition: upsert definitions: %w", err)
	}

	// 3. UPSERT battlepass_track_translations pour chaque langue disponible.
	nameTranslations := collectTrackTranslations(def.Name)
	descTranslations := collectTrackTranslations(def.Description)

	langs := make(map[string]struct{}, len(nameTranslations)+len(descTranslations))
	for lang := range nameTranslations {
		langs[lang] = struct{}{}
	}
	for lang := range descTranslations {
		langs[lang] = struct{}{}
	}

	for lang := range langs {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO battlepass_track_translations
				(reward_track_path, content_hash, lang, track_name, first_seen_at, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT (reward_track_path, content_hash, lang) DO UPDATE SET
				track_name   = excluded.track_name,
				last_seen_at = excluded.last_seen_at`,
			trackPath,
			contentHash,
			lang,
			nullableTrackString(nameTranslations[lang]),
			now,
			now,
		); err != nil {
			return fmt.Errorf("storeTrackDefinition: upsert translations lang=%s: %w", lang, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("storeTrackDefinition: commit: %w", err)
	}
	return nil
}

// trackDefinitionContentHash retourne les 8 premiers octets du SHA-256 en hex.
func trackDefinitionContentHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8])
}

// collectTrackTranslations extrait un mapping langue → texte depuis un champ Name/Description
// Waypoint (qui peut être une string simple ou un objet {value, translations: {lang: text}}).
func collectTrackTranslations(raw any) map[string]string {
	result := make(map[string]string)
	switch v := raw.(type) {
	case string:
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			result["en-US"] = trimmed
		}
	case map[string]any:
		if fallback, ok := v["value"].(string); ok {
			if trimmed := strings.TrimSpace(fallback); trimmed != "" {
				result["en-US"] = trimmed
			}
		}
		if nested, ok := v["translations"].(map[string]any); ok {
			for lang, localized := range nested {
				if text, ok := localized.(string); ok {
					if trimmed := strings.TrimSpace(text); trimmed != "" {
						result[lang] = trimmed
					}
				}
			}
		}
	}
	return result
}

// nullableTrackString retourne nil si la valeur est vide, la valeur sinon.
// Utilisé pour éviter d'écrire des chaînes vides là où NULL est préférable.
func nullableTrackString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// ---------------------------------------------------------------------------
// Cache local des images de tracks Battle Pass
// ---------------------------------------------------------------------------

// battlepassImageCacheDir dérive le répertoire de cache images depuis battlepassMetaPath.
// Remonte l'arborescence jusqu'au répertoire "data/" pour construire data/cache/battlepass_assets/,
// quelle que soit la profondeur du chemin metadata (legacy ou title-aware).
func (p *HaloProvider) battlepassImageCacheDir() string {
	metaPath := strings.TrimSpace(p.battlepassMetaPath)
	if metaPath == "" {
		return ""
	}
	current := filepath.Dir(filepath.Clean(metaPath))
	for {
		if strings.EqualFold(filepath.Base(current), "data") {
			return filepath.Join(current, "cache", "battlepass_assets")
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	// Fallback structurel : deux niveaux au-dessus du metadata (ne devrait pas arriver).
	return filepath.Join(filepath.Dir(filepath.Dir(filepath.Clean(metaPath))), "cache", "battlepass_assets")
}

// preCacheBPTrackImages télécharge en arrière-plan les images d'un track Battle Pass
// (image de couverture + background) dans data/cache/battlepass_assets/tracks/.
// Appelé dans une goroutine ; les tokens sont passés par valeur pour éviter les races.
func (p *HaloProvider) preCacheBPTrackImages(bpImage, bgImage string, tokens domain.HaloTokens) {
	ctx := context.Background()
	if bpImage != "" {
		p.ensureBPImageCached(ctx, bpImage, "tracks", &tokens)
	}
	if bgImage != "" {
		p.ensureBPImageCached(ctx, bgImage, "tracks", &tokens)
	}
}

// ensureBPImageCached télécharge une image depuis le CDN GameCMS et la sauvegarde
// dans imageDir/subCategory/{basename}.png. Sans effet si le fichier existe déjà.
// Les erreurs sont tracées en debug et ignorées (best-effort).
func (p *HaloProvider) ensureBPImageCached(
	ctx context.Context,
	imagePath, subCategory string,
	tokens *domain.HaloTokens,
) {
	trimmed := strings.TrimSpace(imagePath)
	if trimmed == "" {
		return
	}

	imageDir := p.battlepassImageCacheDir()
	if imageDir == "" {
		return
	}

	// Nom de fichier = dernier segment du chemin GameCMS
	// ex. "RewardTracks/Operations/S6/images/05dfaff81ec75eed4bc214c3.png" → "05dfaff81ec75eed4bc214c3.png"
	filename := path.Base(trimmed)
	if filename == "." || filename == "/" || filename == "" {
		return
	}

	localPath := filepath.Join(imageDir, subCategory, filename)

	// Déjà en cache → rien à faire.
	if _, err := os.Stat(localPath); err == nil {
		return
	}

	// Télécharger depuis le CDN GameCMS.
	base := p.gameCMSBaseURL
	if base == "" {
		base = defaultGameCMSHost
	}
	url := fmt.Sprintf("%s/hi/images/file/%s",
		strings.TrimRight(base, "/"),
		strings.TrimLeft(trimmed, "/"),
	)

	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		slog.DebugContext(ctx, "battlepass: image download failed",
			"path", imagePath, "err", err)
		return
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		slog.DebugContext(ctx, "battlepass: image dir creation failed",
			"local", localPath, "err", err)
		return
	}
	if err := os.WriteFile(localPath, body, 0o644); err != nil {
		slog.DebugContext(ctx, "battlepass: image write failed",
			"local", localPath, "err", err)
		return
	}
	slog.InfoContext(ctx, "battlepass: image mise en cache local",
		"path", imagePath, "local", localPath)
}
