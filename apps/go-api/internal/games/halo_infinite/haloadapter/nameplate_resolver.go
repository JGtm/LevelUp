package haloadapter

// nameplate_resolver.go — la bannière (nameplate) Spartan dérivée de l'emblème, quand
// `/customization/appearance` ne rend pas de chemin de bannière (port du flux Python
// `resolve_positive_emblem_cfg`).
//
// Ordre : la table officielle emblème -> nameplate (`mapping.json`, cf. Grunt
// GameCms_GetEmblemMapping) donne le chemin EXACT pour la configuration équipée, même négative
// (palettes servies sous `_n<abs(cfg)>.png`) ; à défaut, une configuration > 0 donne l'URL
// directe ; à défaut encore, le descripteur CMS de l'emblème donne sa PREMIÈRE configuration
// positive (les négatives sont des palettes « Test » sans image servie) — palette approchée mais
// image servable.
//
// Sorties : `http.DefaultClient` (aucun délai de client : chaque GET de la brique porte son
// échéance de 8 s), gardé par netguard sous deux surfaces (table, descripteur). Le cache de la
// table (6 h) est de processus.

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/haloapi/infinite"
)

// emblemMappingTTL : la table change peu (nouveaux emblèmes à chaque lot de contenu) ; 6 h, comme
// la personnalisation.
const emblemMappingTTL = 6 * time.Hour

// emblemMappingEntry : une entrée de la table emblème -> nameplate.
type emblemMappingEntry = infinite.EmblemMappingEntry

// Cache de processus de la table : radical d'emblème -> ConfigurationId -> entrée ; rafraîchi à
// l'expiration (au mieux : sur échec, la dernière valeur connue reste servie).
var (
	emblemMappingMu      sync.RWMutex
	emblemMappingData    map[string]map[string]emblemMappingEntry
	emblemMappingFetched time.Time
)

// nameplateAPI rend le client de la brique pour une sortie de la résolution nameplate : surface
// `surface`, adresses du titre du contexte (résolveur partagé, sans journal sur une clé absente).
func nameplateAPI(ctx context.Context, surface string) *infinite.Client {
	e := adressesDuTitre(ctx, games.DefaultEndpointResolver(), ctxkeys.TitleSlug(ctx), false, infinite.CleNameplate)
	return infinite.New(netguardDoer{surface: surface, next: http.DefaultClient}, infinite.Retry{}).WithEndpoints(e)
}

// getEmblemMappingEntry rend l'entrée exacte de (emblemID, cfg) depuis le cache, rafraîchi s'il a
// expiré.
func getEmblemMappingEntry(ctx context.Context, emblemID string, cfg int64, auth infinite.Auth) (emblemMappingEntry, bool) {
	emblemMappingMu.RLock()
	stale := emblemMappingData == nil || time.Since(emblemMappingFetched) > emblemMappingTTL
	emblemMappingMu.RUnlock()
	if stale {
		refreshEmblemMapping(ctx, auth)
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

// refreshEmblemMapping relit la table et la met en cache ; au mieux (un échec garde l'ancienne,
// la brique journalise la cause).
func refreshEmblemMapping(ctx context.Context, auth infinite.Auth) {
	parsed, err := nameplateAPI(ctx, surfaceNameplateMapping).GetEmblemMapping(ctx, auth)
	if err != nil {
		return
	}
	emblemMappingMu.Lock()
	emblemMappingData = parsed
	emblemMappingFetched = time.Now()
	emblemMappingMu.Unlock()
	slog.InfoContext(ctx, "nameplate_resolver: mapping refreshed", "entries", len(parsed))
}

// ResolveNameplateURL rend l'URL de la nameplate dérivée d'un chemin d'emblème et d'une
// configuration (voir l'en-tête) ; "" si le chemin est vide, n'est pas un emblème, ou si la
// résolution échoue (journalisé). Mince enveloppe de [resolveNameplate], que
// [DiagnoseNameplate] partage.
func ResolveNameplateURL(ctx context.Context, emblemPath string, cfg int64, spartanToken, clearanceToken string) string {
	url, _, _ := resolveNameplate(ctx, emblemPath, cfg, spartanToken, clearanceToken)
	return url
}

// resolveNameplate est le cœur UNIQUE de la résolution : l'URL et son verdict de diagnostic.
// Verdicts : ok (table ou configuration positive) ; upstream_missing (absent de la table et
// descripteur lu SANS configuration positive : absence durable, emblème de nouvelle génération) ;
// transient (échec indéterminé, se répare au rafraîchissement suivant).
func resolveNameplate(ctx context.Context, emblemPath string, cfg int64, spartanToken, clearanceToken string) (url string, verdict Verdict, detail Detail) {
	trimmed := strings.TrimSpace(emblemPath)
	if trimmed == "" {
		return "", VerdictTransient, DetailNoEmblemPath
	}
	stem := infinite.ExtractEmblemStem(trimmed)
	if stem == "" {
		return "", VerdictTransient, DetailNonEmblemPath
	}
	auth := infinite.Auth{Spartan: spartanToken, Clearance: clearanceToken}
	if entry, ok := getEmblemMappingEntry(ctx, stem, cfg, auth); ok && entry.NameplateCmsPath != "" {
		return infinite.NameplateURLDepuisMapping(nameplateAPI(ctx, surfaceNameplateMapping).Endpoints(), entry.NameplateCmsPath), VerdictOK, DetailMappingHit
	}
	resolvedCfg := cfg
	if resolvedCfg <= 0 {
		var definitive bool
		resolvedCfg, definitive = resolvePositiveEmblemCfg(ctx, trimmed, auth)
		if resolvedCfg <= 0 {
			if definitive {
				// État NORMAL et durable des emblèmes de nouvelle génération : absents de la table,
				// une seule configuration négative, aucun PNG servi. La lecture servira la dernière
				// bannière connue (jamais vide).
				slog.InfoContext(ctx, "nameplate_resolver: emblème sans nameplate upstream (mapping.json miss + aucune cfg positive) — la dernière bannière connue sera servie",
					"emblem_path", trimmed, "stem", stem, "original_cfg", cfg, "xuid", ctxkeys.HaloXUID(ctx))
				return "", VerdictUpstreamMissing, DetailNoPositiveCfg
			}
			slog.WarnContext(ctx, "nameplate_resolver: résolution nameplate échouée (fetch CMS emblem KO, indéterminé)",
				"emblem_path", trimmed, "stem", stem, "original_cfg", cfg, "xuid", ctxkeys.HaloXUID(ctx))
			return "", VerdictTransient, DetailCMSHTTPError
		}
	}
	return infinite.NameplateURLDepuisCfg(nameplateAPI(ctx, surfaceNameplateEmblem).Endpoints(), stem, resolvedCfg), VerdictOK, DetailMappingMiss
}

// resolvePositiveEmblemCfg lit le descripteur CMS de l'emblème et rend sa première configuration
// positive. `definitive` : true quand le descripteur a été lu (l'absence de configuration
// positive est alors un fait durable) ; false sur un échec de transport, de statut ou de lecture
// (indéterminé, potentiellement transitoire).
func resolvePositiveEmblemCfg(ctx context.Context, emblemPath string, auth infinite.Auth) (result int64, definitive bool) {
	cfg, err := nameplateAPI(ctx, surfaceNameplateEmblem).GetEmblemPositiveCfg(ctx, auth, emblemPath)
	if err != nil {
		return 0, false
	}
	return cfg, true
}
