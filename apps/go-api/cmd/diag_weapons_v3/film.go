package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// matchRef relie le préfixe cache (8-char, clé de film_chunks/ + film_manifests/)
// à l'UUID complet du registre.
type matchRef struct {
	short string // préfixe cache (nom du dossier chunks + manifest)
	full  string // match_id complet (registre / participants)
}

// resolveMatchIDs construit la liste des matchs à traiter selon cfg :
//   - -match : résout le préfixe/UUID vers l'UUID complet via le registre.
//   - -all   : énumère les dossiers film_chunks/ cachés et garde ceux présents
//     dans match_registry (jointure par préfixe).
func resolveMatchIDs(ctx context.Context, db *sql.DB, cfg runConfig) ([]matchRef, error) {
	if cfg.matchArg != "" {
		return resolveMatchArgs(ctx, db, cfg.matchArg)
	}
	return cachedMatchRefs(ctx, db, cfg.cacheDir)
}

// resolveMatchArgs résout -match : un préfixe/UUID, ou plusieurs séparés par
// virgule (panel de validation). Chaque arg est résolu via le registre.
func resolveMatchArgs(ctx context.Context, db *sql.DB, arg string) ([]matchRef, error) {
	var refs []matchRef
	for _, raw := range strings.Split(arg, ",") {
		a := strings.TrimSpace(raw)
		if a == "" {
			continue
		}
		full, ok, err := resolveFullMatchID(ctx, db, a)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("match %q absent de match_registry", a)
		}
		refs = append(refs, matchRef{short: shortPrefix(full), full: full})
	}
	return refs, nil
}

// cachedMatchRefs énumère les films cachés (dossiers film_chunks/<short>/) et ne
// garde que ceux ayant un manifest ET une ligne match_registry correspondante.
func cachedMatchRefs(ctx context.Context, db *sql.DB, cacheDir string) ([]matchRef, error) {
	entries, err := os.ReadDir(filmcache.ChunksRoot(cacheDir))
	if err != nil {
		return nil, fmt.Errorf("read film_chunks dir: %w", err)
	}
	var refs []matchRef
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		short := e.Name()
		if _, err := os.Stat(filmcache.ManifestPath(cacheDir, short)); err != nil {
			continue // pas de manifest -> on saute (dégradation gracieuse)
		}
		full, ok, err := resolveFullMatchID(ctx, db, short)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue // film orphelin (match purgé du registre)
		}
		refs = append(refs, matchRef{short: short, full: full})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].short < refs[j].short })
	return refs, nil
}

// shortPrefix renvoie le préfixe 8-char (clé cache) d'un UUID de match.
func shortPrefix(matchID string) string {
	if len(matchID) >= 8 {
		return matchID[:8]
	}
	return matchID
}
