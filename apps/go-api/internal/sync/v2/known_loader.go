// Package v2 — known_loader.go : implémentation V2-native de KnownLoader
// (D6.2 du plan ADR 0027).
//
// Réimplémentation indépendante de V1 (engine.go::loadKnownMatchIDs) pour
// satisfaire la règle de duplication ciblée : V1 ne reçoit aucune
// modification. Algorithme identique à V1 :
//
//  1. SELECT match_id FROM player_match_enrichment    (source player)
//  2. SELECT DISTINCT match_id FROM match_participants
//     WHERE xuid || ” = ?                            (source shared, cross-player dedup)
//
// Cast défensif `xuid || ”` aligne sur recompute_after_art_rebuild.go:156
// pour éviter les mismatchs silencieux si la colonne drift en type
// (VARCHAR vs UBIGINT) — incident observé en prod 2026-05-24.
package v2

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// PlayerDBOpener ouvre la stats.duckdb d'un joueur en read-only et retourne
// le handle + une fonction de libération. Le caller doit appeler release()
// après usage (defer). Wrappe OpenPlayerDB + dblease.AcquireReader côté
// runtime ; le test injecte un opener qui ouvre un temp DuckDB.
//
// Les implémentations doivent être thread-safe (peut être appelée
// concurremment pour N joueurs en Phase 1).
type PlayerDBOpener func(ctx context.Context, gamertag string) (db *sql.DB, release func(), err error)

// knownLoaderV2 implémente KnownLoader sans dépendance à internal/sync.
// Utilise *sql.DB du package standard — aucun lien avec V1.
type knownLoaderV2 struct {
	openPlayerDB PlayerDBOpener
	getSharedDB  func() *sql.DB // retourne la connexion shared courante à chaque appel
}

// NewKnownLoader construit un KnownLoader prêt à être injecté dans le
// CycleOrchestrator. getSharedDB peut retourner nil (source 2 désactivée,
// comportement V1-compatible — la source 1 player suffit pour la dedup
// intra-player). Utiliser un getter plutôt qu'un *sql.DB fixe évite les
// connexions stales après un swap provider RO→RW→RO.
func NewKnownLoader(playerDBOpener PlayerDBOpener, getSharedDB func() *sql.DB) KnownLoader {
	return &knownLoaderV2{
		openPlayerDB: playerDBOpener,
		getSharedDB:  getSharedDB,
	}
}

// LoadKnown retourne l'union player_match_enrichment ∪ shared.match_participants
// pour le xuid du joueur. Erreurs partielles (1 source échoue) → log WARN +
// retour de l'autre source (best-effort).
//
// Erreur fatale uniquement si openPlayerDB échoue (impossible d'avoir la
// source 1).
func (l *knownLoaderV2) LoadKnown(ctx context.Context, p PlayerProfile) (map[string]bool, error) {
	known := make(map[string]bool, 512)

	// Source 1 : player_match_enrichment (DB per-joueur).
	playerDB, release, err := l.openPlayerDB(ctx, p.Gamertag)
	if err != nil {
		return nil, fmt.Errorf("open player DB %s: %w", p.Gamertag, err)
	}
	defer release()

	rows, err := playerDB.QueryContext(ctx, "SELECT match_id FROM player_match_enrichment_latest")
	if err == nil {
		for rows.Next() {
			var id string
			if scanErr := rows.Scan(&id); scanErr == nil {
				known[id] = true
			}
		}
		_ = rows.Close()
	} else {
		// Table absente (schéma frais) ou autre erreur — pas fatal.
		slog.DebugContext(ctx, "v2 LoadKnown: player_match_enrichment query failed (table absent ?)",
			"gamertag", p.Gamertag, "err", err)
	}

	// Source 2 : shared.match_participants WHERE xuid (cross-player dedup).
	// getSharedDB() retourne la connexion courante (fraîche après chaque swap
	// provider RO→RW→RO) plutôt qu'un pointeur capturé au boot qui devient
	// stale après le premier cycle.
	sharedDB := l.getSharedDB()
	if sharedDB != nil && strings.TrimSpace(p.XUID) != "" {
		sharedRows, err := sharedDB.QueryContext(ctx,
			"SELECT DISTINCT match_id FROM match_participants WHERE xuid || '' = ?", p.XUID)
		if err != nil {
			// Warn explicite (cohérent avec V1) : known set partiel peut
			// dégrader la dedup cross-player → re-fetch inutile.
			slog.WarnContext(ctx, "v2 LoadKnown: shared.match_participants query failed — known set partiel",
				"gamertag", p.Gamertag, "xuid", p.XUID, "err", err)
		} else {
			addedFromShared := 0
			for sharedRows.Next() {
				var id string
				if scanErr := sharedRows.Scan(&id); scanErr == nil {
					if !known[id] {
						addedFromShared++
					}
					known[id] = true
				}
			}
			_ = sharedRows.Close()
			slog.DebugContext(ctx, "v2 LoadKnown: source 2 (shared) ajoutee",
				"gamertag", p.Gamertag, "xuid", p.XUID,
				"added_from_shared", addedFromShared, "total_known", len(known))
		}
	}

	return known, nil
}
