// Package sync — fetch_cache.go : cache fetch intermédiaire pour le refactor
// Collect→Persist (cf. REFACTOR_COLLECT_PERSIST.md §3.5 / §10 Q7).
//
// **Pourquoi** : économie quota API + recovery sans re-fetch + debug
// (voir exactement ce que l'API a renvoyé pour un match donné).
//
// **Architecture** : wrapping transparent autour d'un HaloClient existant.
// Le cache est un système fichier simple sous `data/sync_cache/{cycle_id}/` :
//
//	data/sync_cache/
//	└── {cycle_id}/                          (créé au début de run())
//	    ├── match_{id}_stats.json            (raw GetMatchStats response)
//	    ├── match_{id}_skill.json            (raw GetMatchSkill response)
//	    └── match_{id}_highlight_chunk.bin   (raw chunk highlight events)
//
// **Cycle de vie** :
//   - cycle_id = event_id du run() (déjà créé via logging.WithEvent).
//   - Création lazy au 1er write.
//   - Lecture : si fichier existe, skip l'appel API.
//   - Pas de delete par batch — purge par âge (>7 jours) via PurgeOldCache.
//
// **Désactivation** : `LEVELUP_PERSIST_NO_FETCH_CACHE=1` → wrapper inactif
// (renvoie directement les appels à l'inner client, sans cache).

package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// FetchCacheConfig configure un cachedHaloClient.
type FetchCacheConfig struct {
	// CacheDir : répertoire de cache (typiquement `data/sync_cache/{cycle_id}/`).
	// Créé lazy au 1er write. Si vide → cache désactivé (fall-through inner).
	CacheDir string
	// Disabled : court-circuite complètement le cache (lecture ET écriture).
	// Activé par `LEVELUP_PERSIST_NO_FETCH_CACHE=1`.
	Disabled bool
}

// cachedHaloClient wrap un HaloClient et ajoute un cache disque sur les
// méthodes coûteuses (GetMatchStats, GetMatchSkill, GetHighlightEventsChunk).
// Les autres méthodes (history, career, CSRs) sont pass-through (peu de
// bénéfice à cacher l'history qui change à chaque sync).
type cachedHaloClient struct {
	inner HaloClient
	cfg   FetchCacheConfig
}

// NewCachedHaloClient construit un wrapping cache autour d'un HaloClient.
// Si cfg.Disabled, retourne directement inner (pas de wrapping).
func NewCachedHaloClient(inner HaloClient, cfg FetchCacheConfig) HaloClient {
	if cfg.Disabled || cfg.CacheDir == "" {
		return inner
	}
	return &cachedHaloClient{inner: inner, cfg: cfg}
}

// ─── Pass-through (non cachés) ────────────────────────────────────────────

func (c *cachedHaloClient) GetMatchHistory(ctx context.Context, gamertag, matchType string, start, count int) ([]MatchHistoryEntry, error) {
	return c.inner.GetMatchHistory(ctx, gamertag, matchType, start, count)
}

func (c *cachedHaloClient) GetMatchFilm(ctx context.Context, matchID string) (map[int]FilmChunkData, bool, error) {
	// Films chunks = volume gros. Pas cachés ici — utiliser LocalFilmCache pour ça.
	return c.inner.GetMatchFilm(ctx, matchID)
}

// GetFilmChunks : passe-plat vers inner. IL N'EST PAS OPTIONNEL, ET C'EST LA LEÇON DU
// 2026-08-29.
//
// `GetFilmChunks` ne fait pas partie de l'interface HaloClient (délibéré : les mocks des
// autres étapes n'ont pas à la porter). L'étape 1.57 du post-sync l'obtient donc par
// ASSERTION DE TYPE sur le client. Or ce wrapper est posé SYSTÉMATIQUEMENT sur le chemin V1
// (engine.go, NewCachedHaloClient) : sans cette méthode, l'assertion échouait, l'étape sortait
// en silence, et `assist_known` restait FALSE — le défaut exact que cette étape existe pour
// corriger. Un wrapper qui n'expose pas ce qu'il enveloppe le DÉSACTIVE.
//
// Le repli sur `inner` qui ne porterait pas la méthode rend `found = false` sans erreur : un
// client de test minimal reste utilisable, et le collecteur traite ce cas comme « film absent ».
func (c *cachedHaloClient) GetFilmChunks(ctx context.Context, matchID string) ([]FilmChunk, bool, error) {
	f, ok := c.inner.(interface {
		GetFilmChunks(ctx context.Context, matchID string) ([]FilmChunk, bool, error)
	})
	if !ok {
		return nil, false, nil
	}
	return f.GetFilmChunks(ctx, matchID)
}

// FetchMvarForMap : passe-plat vers inner. MEME LECON QUE `GetFilmChunks` JUSTE AU-DESSUS, et
// elle vient d'etre re-payee.
//
// Le rattrapage du catalogue de cartes obtient cette capacite par ASSERTION DE TYPE sur le
// client. Or ce wrapper est pose SYSTEMATIQUEMENT sur le chemin V1 (engine.go,
// NewCachedHaloClient) : sans cette methode, l'assertion echouait, le rattrapage sortait en
// silence, et AUCUNE carte absente n'entrait jamais au catalogue — indistinguable d'un lot non
// deploye. Un wrapper qui n'expose pas ce qu'il enveloppe le DESACTIVE.
//
// Le repli sur un `inner` qui ne porterait pas la methode rend une erreur explicite plutot
// qu'un succes vide : ici, se taire ferait croire a un `.mvar` introuvable cote serveur.
func (c *cachedHaloClient) FetchMvarForMap(ctx context.Context, mapID, mvarFile string,
) ([]byte, string, error) {
	f, ok := c.inner.(interface {
		FetchMvarForMap(ctx context.Context, mapID, mvarFile string) ([]byte, string, error)
	})
	if !ok {
		return nil, "", errPasDeCapaciteMvar
	}
	return f.FetchMvarForMap(ctx, mapID, mvarFile)
}

// errPasDeCapaciteMvar : le client enveloppe ne sait pas rapatrier de variante de carte.
var errPasDeCapaciteMvar = errors.New("client sans capacite FetchMvarForMap")

func (c *cachedHaloClient) GetCareerRank(ctx context.Context, xuid string) (*CareerRankData, error) {
	return c.inner.GetCareerRank(ctx, xuid)
}

func (c *cachedHaloClient) GetPlayerCSRs(ctx context.Context, xuid, seasonID string) ([]PlayerPlaylistCSR, error) {
	return c.inner.GetPlayerCSRs(ctx, xuid, seasonID)
}

func (c *cachedHaloClient) GetPlaylistCsr(ctx context.Context, playlistID, xuid, seasonID string) (*PlayerPlaylistCSR, error) {
	return c.inner.GetPlaylistCsr(ctx, playlistID, xuid, seasonID)
}

// ─── Cachés ───────────────────────────────────────────────────────────────

// GetMatchStats : check cache → si miss, call API + write cache.
func (c *cachedHaloClient) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	path := c.statsPath(matchID)
	if data, err := os.ReadFile(path); err == nil {
		var out map[string]any
		if jerr := json.Unmarshal(data, &out); jerr == nil {
			slog.DebugContext(ctx, "fetch_cache: hit GetMatchStats",
				"match_id", matchID, "path", path)
			return out, nil
		}
		// Cache corrompu → fall-through API et écrase.
		slog.WarnContext(ctx, "fetch_cache: corrupted GetMatchStats cache, refetching",
			"match_id", matchID, "path", path)
	}
	out, err := c.inner.GetMatchStats(ctx, matchID)
	if err != nil {
		return out, err
	}
	c.writeJSON(path, out, "GetMatchStats", matchID)
	return out, nil
}

// GetMatchSkill : idem. Skip cache si erreur côté inner.
func (c *cachedHaloClient) GetMatchSkill(ctx context.Context, matchID string, xuids []string) (map[string]*MatchSkillData, error) {
	path := c.skillPath(matchID)
	if data, err := os.ReadFile(path); err == nil {
		var out map[string]*MatchSkillData
		if jerr := json.Unmarshal(data, &out); jerr == nil {
			slog.DebugContext(ctx, "fetch_cache: hit GetMatchSkill",
				"match_id", matchID, "path", path)
			return out, nil
		}
	}
	out, err := c.inner.GetMatchSkill(ctx, matchID, xuids)
	if err != nil {
		return out, err
	}
	c.writeJSON(path, out, "GetMatchSkill", matchID)
	return out, nil
}

// GetHighlightEventsChunk : check cache (chunk bin + meta json).
func (c *cachedHaloClient) GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error) {
	binPath := c.highlightChunkPath(matchID)
	metaPath := binPath + ".meta.json"

	if binData, err := os.ReadFile(binPath); err == nil {
		if metaData, mErr := os.ReadFile(metaPath); mErr == nil {
			var meta struct {
				Version int  `json:"version"`
				Found   bool `json:"found"`
			}
			if json.Unmarshal(metaData, &meta) == nil {
				slog.DebugContext(ctx, "fetch_cache: hit GetHighlightEventsChunk",
					"match_id", matchID, "bytes", len(binData))
				return binData, meta.Version, meta.Found, nil
			}
		}
	}
	data, version, found, err := c.inner.GetHighlightEventsChunk(ctx, matchID)
	if err != nil {
		return data, version, found, err
	}
	if found && len(data) > 0 {
		if err := c.ensureCacheDir(); err == nil {
			_ = os.WriteFile(binPath, data, 0o644)
			meta := struct {
				Version int  `json:"version"`
				Found   bool `json:"found"`
			}{Version: version, Found: found}
			if mData, jErr := json.Marshal(meta); jErr == nil {
				_ = os.WriteFile(metaPath, mData, 0o644)
			}
		}
	}
	return data, version, found, nil
}

// ─── Helpers cache ─────────────────────────────────────────────────────────

func (c *cachedHaloClient) statsPath(matchID string) string {
	return filepath.Join(c.cfg.CacheDir, "match_"+matchID+"_stats.json")
}

func (c *cachedHaloClient) skillPath(matchID string) string {
	return filepath.Join(c.cfg.CacheDir, "match_"+matchID+"_skill.json")
}

func (c *cachedHaloClient) highlightChunkPath(matchID string) string {
	return filepath.Join(c.cfg.CacheDir, "match_"+matchID+"_highlight_chunk.bin")
}

func (c *cachedHaloClient) ensureCacheDir() error {
	return os.MkdirAll(c.cfg.CacheDir, 0o755)
}

// writeJSON sérialise + écrit best-effort (log warn si fail, ne propage pas).
//
// **NaN handling** : certains payloads Halo (notamment GetMatchSkill avec
// kills_expected/deaths_expected non calculables) contiennent des float64
// `NaN` ou `±Inf` que `encoding/json` refuse de sérialiser. C'est attendu
// et non-bloquant pour le sync — on log en DEBUG (pas WARN) pour ne pas
// spammer les logs. Le cache MISS au prochain run refetch normalement.
func (c *cachedHaloClient) writeJSON(path string, v any, op, matchID string) {
	if err := c.ensureCacheDir(); err != nil {
		slog.Warn("fetch_cache: mkdir failed", "dir", c.cfg.CacheDir, "err", err)
		return
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		// Erreur courante : "json: unsupported value: NaN". Pas critique,
		// le cache devient juste un miss au prochain fetch.
		slog.Debug("fetch_cache: marshal skipped (likely NaN/Inf in payload)",
			"op", op, "match_id", matchID, "err", err)
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		slog.Warn("fetch_cache: write failed", "op", op, "match_id", matchID, "path", path, "err", err)
	}
}

// PurgeOldFetchCache supprime les sous-dossiers (cycle_id) du root cache dir
// plus vieux que maxAge. Best-effort : continue sur erreur fichier individuel,
// retourne le nombre de dossiers supprimés.
//
// Cas d'usage : janitor périodique (1× / jour) qui retire les cycles anciens.
// Default maxAge recommandé : 7 jours (cf. REFACTOR_COLLECT_PERSIST.md §3.5).
func PurgeOldFetchCache(rootDir string, maxAge time.Duration) (int, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // dir absent = rien à purger
		}
		return 0, fmt.Errorf("fetch_cache: read root %s: %w", rootDir, err)
	}
	cutoff := time.Now().Add(-maxAge)
	purged := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		path := filepath.Join(rootDir, e.Name())
		if err := os.RemoveAll(path); err != nil {
			slog.Warn("fetch_cache: purge dir failed", "dir", path, "err", err)
			continue
		}
		purged++
	}
	if purged > 0 {
		slog.Info("fetch_cache: purge",
			"root", rootDir, "purged_dirs", purged, "max_age", maxAge)
	}
	return purged, nil
}
