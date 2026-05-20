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
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	nameplateHost            = "https://gamecms-hacs.svc.halowaypoint.com"
	nameplateResolverTimeout = 8 * time.Second
)

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

	resolvedCfg := cfg
	if resolvedCfg <= 0 {
		resolvedCfg = resolvePositiveEmblemCfg(ctx, trimmed, spartanToken, clearanceToken)
		if resolvedCfg <= 0 {
			slog.WarnContext(ctx, "nameplate_resolver: aucun cfg positif trouvé",
				"emblem_path", trimmed, "original_cfg", cfg)
			return ""
		}
	}

	return fmt.Sprintf("%s/hi/Waypoint/file/images/nameplates/%s_%d.png",
		nameplateHost, stem, resolvedCfg)
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
	cmsURL := fmt.Sprintf("%s/hi/progression/file/%s",
		nameplateHost, strings.TrimPrefix(emblemPath, "/"))

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
