// Package sync — spartan_nameplate_resolver.go : port du flow Python
// `resolve_positive_emblem_cfg` pour récupérer la bannière (nameplate) Spartan
// quand l'endpoint /customization/appearance ne retourne pas BannerImagePath.
//
// Pourquoi : depuis le commit 22cb84d5 (8 mai 2026), le Go a supprimé les
// fallbacks fallbackCustomization*URL qui inventaient des URLs nameplate au
// format /hi/Waypoint/file/images/nameplates/<stem>_<cfg>.png. Constat prod :
// 403 systématique quand cfg était négatif (palette "Test" sans image CDN).
//
// **Pièce manquante** : le Python (src/ui/profile_api_urls.py
// resolve_positive_emblem_cfg) fetche le JSON emblem GameCMS, parse
// AvailableConfigurations[], et prend le **premier ConfigurationId > 0**
// (les négatifs sont des palettes Test sans image servie par le CDN).
//
// Vérifié en prod le 2026-05-20 pour JGtm : emblem cfg=-809699482 (Test)
// → JSON CMS contient AvailableConfigurations avec [3]=651339664 positif
// → URL nameplate construite avec ce cfg retourne 200 OK + image/png 10585 b.
package haloclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
)

// nameplateHostFor résout l'host gamecms/nameplate pour le titre courant (MT-01).
// Free function (les résolveurs nameplate ne sont pas des méthodes HaloAPIClient) :
// consulte le resolver partagé de boot, fallback const Halo legacy.
func nameplateHostFor(ctx context.Context) string {
	res := games.DefaultEndpointResolver()
	if res == nil {
		return nameplateHost
	}
	if host, ok := res.HostFor(ctxkeys.TitleSlug(ctx), games.EndpointNameplate); ok {
		return host
	}
	return nameplateHost
}

const (
	nameplateHost            = "https://gamecms-hacs.svc.halowaypoint.com"
	nameplateResolverTimeout = 8 * time.Second
	// emblemMappingURL : endpoint officiel Microsoft (cf. Grunt
	// GameCms_GetEmblemMapping). Mappe (emblem_id, configuration_id) vers
	// le nameplateCmsPath exact + textColor. Sans ce mapping, on prenait
	// la 1ère cfg positive comme fallback → palette potentiellement
	// inversée par rapport à celle équipée par le joueur.
	// emblemMappingPathSuffix : la partie post-préfixe de jeu de l'URL mapping
	// (le segment /hi|/h5 est injecté à l'usage via gamePrefixForCtx, MT-01).
	emblemMappingPathSuffix = "/Waypoint/file/images/emblems/mapping.json"
	// emblemMappingTTL : la table change peu (nouveaux emblems quand Halo
	// release un set). 6h est cohérent avec le TTL customization.
	emblemMappingTTL = 6 * time.Hour
)

// emblemMappingEntry projection minimale du JSON Microsoft.
type emblemMappingEntry struct {
	NameplateCmsPath string `json:"nameplateCmsPath"`
	EmblemCmsPath    string `json:"emblemCmsPath"`
	TextColor        string `json:"textColor"`
}

// emblemMappingCache : process-level, thread-safe, refresh on TTL miss.
// Structure : emblemID → ConfigurationId(string) → entry
var (
	emblemMappingMu      sync.RWMutex
	emblemMappingData    map[string]map[string]emblemMappingEntry
	emblemMappingFetched time.Time
)

// getEmblemMappingEntry consulte le cache process-level et retourne l'entry
// exacte pour (emblemID, cfg). Refresh le cache si TTL expiré (best-effort
// fetch, retombe sur dernière valeur connue en cas d'échec).
func getEmblemMappingEntry(ctx context.Context, emblemID string, cfg int64, spartanToken, clearanceToken string) (emblemMappingEntry, bool) {
	emblemMappingMu.RLock()
	stale := emblemMappingData == nil || time.Since(emblemMappingFetched) > emblemMappingTTL
	emblemMappingMu.RUnlock()

	if stale {
		refreshEmblemMapping(ctx, spartanToken, clearanceToken)
	}

	emblemMappingMu.RLock()
	defer emblemMappingMu.RUnlock()
	if emblemMappingData == nil {
		return emblemMappingEntry{}, false
	}
	cfgs, ok := emblemMappingData[emblemID]
	if !ok {
		return emblemMappingEntry{}, false
	}
	entry, ok := cfgs[strconv.FormatInt(cfg, 10)]
	return entry, ok
}

// resetEmblemMappingCacheForTest réinitialise le cache process-level entre
// tests pour éviter les interférences. Sealed via build tag-free pour
// que les tests puissent l'appeler.
func resetEmblemMappingCacheForTest() {
	emblemMappingMu.Lock()
	emblemMappingData = nil
	emblemMappingFetched = time.Time{}
	emblemMappingMu.Unlock()
}

// seedEmblemMappingCacheForTest seed le cache avec un mapping arbitraire
// (test-only). Permet de tester la branche "mapping hit" sans réseau.
func seedEmblemMappingCacheForTest(data map[string]map[string]emblemMappingEntry) {
	emblemMappingMu.Lock()
	emblemMappingData = data
	emblemMappingFetched = time.Now()
	emblemMappingMu.Unlock()
}

// refreshEmblemMapping : fetch + parse + cache. Best-effort, log warn sur
// échec sans paniquer (le caller fallback sur l'ancien comportement).
func refreshEmblemMapping(ctx context.Context, spartanToken, clearanceToken string) {
	reqCtx, cancel := context.WithTimeout(ctx, nameplateResolverTimeout)
	defer cancel()
	mappingURL := nameplateHostFor(ctx) + "/" + gamePrefixForCtx(ctx) + emblemMappingPathSuffix
	req, err := http.NewRequestWithContext(reqCtx, "GET", mappingURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-343-authorization-spartan", spartanToken)
	if clearanceToken != "" {
		req.Header.Set("343-clearance", clearanceToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.DebugContext(ctx, "nameplate_resolver: mapping fetch HTTP error", "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		slog.DebugContext(ctx, "nameplate_resolver: mapping non-200", "status", resp.StatusCode)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var parsed map[string]map[string]emblemMappingEntry
	if err := json.Unmarshal(body, &parsed); err != nil {
		slog.WarnContext(ctx, "nameplate_resolver: mapping parse failed", "err", err, "size", len(body))
		return
	}
	emblemMappingMu.Lock()
	emblemMappingData = parsed
	emblemMappingFetched = time.Now()
	emblemMappingMu.Unlock()
	slog.InfoContext(ctx, "nameplate_resolver: mapping refreshed", "entries", len(parsed))
}

// ResolveNameplateURL retourne l'URL nameplate (image PNG) dérivée d'un
// emblem path + configuration_id.
//
//   - Si emblemPath est vide → ""
//   - Si cfg > 0 → URL directe sans appel CMS
//   - Si cfg <= 0 → fetch JSON CMS de l'emblem, parser AvailableConfigurations,
//     prendre le 1er cfg > 0, construire URL avec ce cfg
//   - Tout échec (HTTP, parse, aucun cfg positif trouvé) → "" + log warn
//
// Spec exacte (Python `_waypoint_nameplate_png_from_emblem` +
// `resolve_positive_emblem_cfg`).
func ResolveNameplateURL(
	ctx context.Context,
	emblemPath string,
	cfg int64,
	spartanToken, clearanceToken string,
) string {
	trimmed := strings.TrimSpace(emblemPath)
	if trimmed == "" {
		return ""
	}
	stem := extractEmblemStem(trimmed)
	if stem == "" {
		return ""
	}

	// Pattern OFFICIEL Microsoft (cf. Grunt GameCms_GetEmblemMapping) :
	// le mapping.json donne nameplateCmsPath EXACT pour la cfg équipée,
	// même quand cfg est négative (palettes que le CDN sert via format
	// `_n<abs(cfg)>.png` au lieu de `_<cfg>.png`). Sans ce lookup, on
	// servait une palette de couleurs incorrecte pour la majorité des
	// joueurs (bug observé 2026-05-20 : "couleurs inversées" pour JGtm
	// et autres).
	if entry, ok := getEmblemMappingEntry(ctx, stem, cfg, spartanToken, clearanceToken); ok && entry.NameplateCmsPath != "" {
		return fmt.Sprintf("%s/%s/Waypoint/file/%s",
			nameplateHostFor(ctx), gamePrefixForCtx(ctx), strings.TrimPrefix(entry.NameplateCmsPath, "/"))
	}

	// Fallback (mapping.json indisponible ou stem absent) : ancien comportement
	// resolve_positive_emblem_cfg (palette positive arbitraire — couleurs
	// potentiellement incorrectes mais image servable).
	resolvedCfg := cfg
	if resolvedCfg <= 0 {
		resolvedCfg = resolvePositiveEmblemCfg(ctx, trimmed, spartanToken, clearanceToken)
		if resolvedCfg <= 0 {
			slog.WarnContext(ctx, "nameplate_resolver: aucun cfg positif trouvé (mapping miss)",
				"emblem_path", trimmed, "stem", stem, "original_cfg", cfg)
			return ""
		}
	}
	return fmt.Sprintf("%s/%s/Waypoint/file/images/nameplates/%s_%d.png",
		nameplateHostFor(ctx), gamePrefixForCtx(ctx), stem, resolvedCfg)
}

// extractEmblemStem retourne `104-001-olympus-campa-2ddbe23b` depuis
// `Inventory/Spartan/Emblems/104-001-olympus-campa-2ddbe23b.json`.
// Retourne "" si l'emblemPath n'est pas un chemin /Spartan/Emblems/.
func extractEmblemStem(emblemPath string) string {
	const marker = "/Spartan/Emblems/"
	idx := strings.Index(emblemPath, marker)
	if idx < 0 {
		return ""
	}
	tail := emblemPath[idx+len(marker):]
	// Strip trailing /<more>.json si plusieurs slashes (rare).
	if slash := strings.LastIndex(tail, "/"); slash >= 0 {
		tail = tail[slash+1:]
	}
	if dot := strings.LastIndex(tail, "."); dot >= 0 {
		tail = tail[:dot]
	}
	return tail
}

// resolvePositiveEmblemCfg fetche le JSON CMS de l'emblem et retourne le
// premier ConfigurationId > 0 dans AvailableConfigurations. Retourne 0 sur
// échec (HTTP, parse, aucun positif).
func resolvePositiveEmblemCfg(
	ctx context.Context,
	emblemPath, spartanToken, clearanceToken string,
) int64 {
	cmsURL := fmt.Sprintf("%s/%s/progression/file/%s",
		nameplateHostFor(ctx), gamePrefixForCtx(ctx), strings.TrimPrefix(emblemPath, "/"))

	reqCtx, cancel := context.WithTimeout(ctx, nameplateResolverTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, "GET", cmsURL, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-343-authorization-spartan", spartanToken)
	if clearanceToken != "" {
		req.Header.Set("343-clearance", clearanceToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.DebugContext(ctx, "nameplate_resolver: HTTP error",
			"emblem_path", emblemPath, "err", err)
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		slog.DebugContext(ctx, "nameplate_resolver: CMS non-200",
			"emblem_path", emblemPath, "status", resp.StatusCode)
		return 0
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return 0
	}
	configs, _ := data["AvailableConfigurations"].([]any)
	for _, c := range configs {
		entry, _ := c.(map[string]any)
		cfgRaw := entry["ConfigurationId"]
		var cfg int64
		switch v := cfgRaw.(type) {
		case float64:
			cfg = int64(v)
		case json.Number:
			cfg, _ = v.Int64()
		}
		if cfg > 0 {
			return cfg
		}
	}
	return 0
}
