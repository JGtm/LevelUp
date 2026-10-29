package infinite

// nameplate.go — bannière (nameplate) Spartan dérivée d'un emblème : table officielle
// emblème -> nameplate (`mapping.json`) et descripteur CMS d'un emblème (GET, DTO, extraction du
// radical, première configuration positive, gabarits d'URL). Les GET passent par le Doer du client (plus de client par
// défaut), une seule tentative, sous une échéance de [DelaiNameplate]. Le cache de la table (6 h),
// la résolution et les verdicts de diagnostic sont à l'appelant.

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

// DelaiNameplate : échéance de chaque GET de la résolution nameplate.
const DelaiNameplate = 8 * time.Second

// emblemMappingPathSuffix : chemin de la table emblème -> nameplate, après le segment de jeu.
const emblemMappingPathSuffix = "/Waypoint/file/images/emblems/mapping.json"

// EmblemMappingEntry : une entrée de la table (projection minimale du JSON officiel).
type EmblemMappingEntry struct {
	NameplateCmsPath string `json:"nameplateCmsPath"`
	EmblemCmsPath    string `json:"emblemCmsPath"`
	TextColor        string `json:"textColor"`
}

// EmblemMappingDTO : radical d'emblème -> ConfigurationId (en chaîne) -> entrée.
type EmblemMappingDTO map[string]map[string]EmblemMappingEntry

// GetEmblemMapping récupère la table emblème -> nameplate ([CleNameplate]).
func (c *Client) GetEmblemMapping(ctx context.Context, auth Auth) (EmblemMappingDTO, error) {
	body, err := c.getUnique(ctx, requete{url: c.endpoints.base(CleNameplate) + emblemMappingPathSuffix, auth: auth})
	if err != nil {
		slog.DebugContext(ctx, "nameplate_resolver: mapping fetch en échec", "err", err)
		return nil, err
	}
	var parsed EmblemMappingDTO
	if err := json.Unmarshal(body, &parsed); err != nil {
		slog.WarnContext(ctx, "nameplate_resolver: mapping parse failed", "err", err, "size", len(body))
		return nil, fmt.Errorf("emblem mapping parse: %w", err)
	}
	return parsed, nil
}

// GetEmblemPositiveCfg lit le descripteur CMS d'un emblème ([CleNameplate],
// `/{jeu}/progression/file/{chemin}`) et rend la première `AvailableConfigurations[].ConfigurationId`
// positive. (0, nil) : descripteur lu SANS configuration positive (absence durable) ; une erreur :
// échec indéterminé (transport, statut, JSON illisible).
func (c *Client) GetEmblemPositiveCfg(ctx context.Context, auth Auth, emblemPath string) (int64, error) {
	cmsURL := c.endpoints.base(CleNameplate) + "/progression/file/" + strings.TrimPrefix(emblemPath, "/")
	body, err := c.getUnique(ctx, requete{url: cmsURL, auth: auth})
	if err != nil {
		slog.DebugContext(ctx, "nameplate_resolver: CMS emblème en échec", "emblem_path", emblemPath, "err", err)
		return 0, err
	}
	cfg, lu := FirstPositiveEmblemCfg(body)
	if !lu {
		return 0, fmt.Errorf("emblem CMS parse: JSON illisible (%d octets)", len(body))
	}
	return cfg, nil
}

// FirstPositiveEmblemCfg lit le corps JSON CMS d'un emblème : (cfg > 0, true) première
// configuration positive ; (0, true) aucune ; (0, false) JSON illisible. Fonction pure.
func FirstPositiveEmblemCfg(body []byte) (cfg int64, parsed bool) {
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, false
	}
	configs, _ := data["AvailableConfigurations"].([]any)
	for _, c := range configs {
		entry, _ := c.(map[string]any)
		var v int64
		switch t := entry["ConfigurationId"].(type) {
		case float64:
			v = int64(t)
		case json.Number:
			v, _ = t.Int64() // illisible = 0 : non positif
		}
		if v > 0 {
			return v, true
		}
	}
	return 0, true
}

// ExtractEmblemStem rend le radical d'un chemin d'emblème
// (« Inventory/Spartan/Emblems/104-001-olympus-campa-2ddbe23b.json » -> « 104-001-olympus-campa-2ddbe23b ») ;
// "" hors de `/Spartan/Emblems/`.
func ExtractEmblemStem(emblemPath string) string {
	const marker = "/Spartan/Emblems/"
	idx := strings.Index(emblemPath, marker)
	if idx < 0 {
		return ""
	}
	tail := emblemPath[idx+len(marker):]
	if slash := strings.LastIndex(tail, "/"); slash >= 0 {
		tail = tail[slash+1:]
	}
	if dot := strings.LastIndex(tail, "."); dot >= 0 {
		tail = tail[:dot]
	}
	return tail
}

// NameplateURLDepuisMapping rend l'URL d'une nameplate désignée par la table (`nameplateCmsPath`).
func NameplateURLDepuisMapping(e Endpoints, nameplateCmsPath string) string {
	return fmt.Sprintf("%s/Waypoint/file/%s", e.base(CleNameplate), strings.TrimPrefix(nameplateCmsPath, "/"))
}

// NameplateURLDepuisCfg rend l'URL de la nameplate d'un radical d'emblème pour une configuration.
func NameplateURLDepuisCfg(e Endpoints, stem string, cfg int64) string {
	return fmt.Sprintf("%s/Waypoint/file/images/nameplates/%s_%d.png", e.base(CleNameplate), stem, cfg)
}

// getUnique exécute UN GET (sans reprise ni limiteur) sous l'échéance [DelaiNameplate] ; tout
// statut non-200 rend un [*HTTPError].
func (c *Client) getUnique(ctx context.Context, rq requete) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, DelaiNameplate)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rq.url, nil)
	if err != nil {
		return nil, err
	}
	poserEntetes(req, rq)
	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }() // la fermeture ne change pas l'issue
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, drainBlobMax)) // drainage best-effort
		return nil, &HTTPError{StatusCode: resp.StatusCode, URL: rq.url, Err: fmt.Errorf("HTTP %d", resp.StatusCode)}
	}
	return io.ReadAll(resp.Body)
}
