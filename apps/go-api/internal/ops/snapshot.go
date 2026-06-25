// Package ops — snapshot.go : producteur de snapshots Parquet immuables versionnés
// (Phase 2 du PLAN_DURABILITE_SNAPSHOT_IMMUABLE).
//
// ProduceSnapshot lit l'ensemble des matchs `snapshot_ready_at IS NOT NULL` (union des
// joueurs du titre), réexporte les FAITS SHARED match-immutables en Parquet dans une
// NOUVELLE version, écrit le manifest, puis flippe ATOMIQUEMENT CURRENT.json. Le
// cut est :
//   - change-gated : no-op propre si 0 match ready, ou si (compte ready + watermark)
//     est identique à la version courante (rien de neuf à figer) ;
//   - hors fenêtre RW : à câbler après libération du write-lease du sync (cf. cycle V2) ;
//   - sans ATTACH : shared et chaque player DB ouvertes séparément en RO (OpenReadForQuery
//     via les openers injectés — JAMAIS `?access_mode=read_only` direct, incident 2026-06-01).
//
// Aucune écriture en place : produire = écrire une version + flipper le pointeur, donc
// une lecture en cours sur la version N n'est jamais perturbée (modèle lakehouse).
package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"levelup/go-api/internal/domain/title"
)

// Catégories d'échec d'un cut (enum fermé pour les métriques — cf. classification côté
// caller via errors.Is). Wrappées au niveau de ProduceSnapshot.
var (
	ErrSnapshotRead     = errors.New("snapshot: lecture")  // gather ready / ouverture RO
	ErrSnapshotCopy     = errors.New("snapshot: copy")     // export Parquet (COPY)
	ErrSnapshotManifest = errors.New("snapshot: manifest") // writeManifest / flipCurrent
)

// snapshotLog : logger taggé module=snapshot → tous les logs producteur/lecture du
// sous-système atterrissent dans logs/snapshot.log (le package ops route sinon vers
// logs/general.log). Diagnostic centralisé cut/fallback/readiness.
var snapshotLog = slog.With("module", "snapshot")

// SnapshotOptions configure un cut. Paths/Shared/PlayerOpener sont obligatoires.
type SnapshotOptions struct {
	TitleSlug     string
	Paths         *title.PathResolver
	Shared        SharedReadOpener
	PlayerOpener  PlayerReadOpener
	Players       []string  // gamertags des joueurs du titre
	Now           time.Time // injectable (tests) ; time.Now() si zéro
	RetentionKeep int       // versions complètes conservées (0 = pas de rétention)
}

// SnapshotResult résume l'issue d'un cut.
type SnapshotResult struct {
	Produced          bool   // false ⇒ no-op (NoopReason renseigné)
	Version           int64  // version produite (ou version courante si no-op unchanged)
	ReadyMatchCount   int    // matchs uniques inclus
	PartialMatchCount int    // dont avec partial_reasons non vide
	NoopReason        string // "no_ready_matches" | "unchanged" | ""
}

// ProduceSnapshot produit (au besoin) une nouvelle version de snapshot pour un titre.
func ProduceSnapshot(ctx context.Context, opts SnapshotOptions) (SnapshotResult, error) {
	slug := opts.TitleSlug
	if slug == "" {
		slug = title.DefaultSlug
	}
	if opts.Paths == nil || opts.Shared == nil || opts.PlayerOpener == nil {
		return SnapshotResult{}, fmt.Errorf("ProduceSnapshot: Paths/Shared/PlayerOpener requis")
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	snapshotsDir := opts.Paths.SnapshotsDir(slug)
	if err := os.MkdirAll(snapshotsDir, 0o755); err != nil {
		return SnapshotResult{}, fmt.Errorf("mkdir snapshots dir: %w", err)
	}
	currentPath := opts.Paths.SnapshotCurrentManifestPath(slug)
	currentVersion, err := readCurrent(currentPath)
	if err != nil {
		return SnapshotResult{}, err
	}

	readyIDs, partial, watermark, err := gatherReadySet(ctx, opts)
	if err != nil {
		// Lecture du set ready tronquée : NE PAS figer un snapshot partiel ni consulter
		// le change-gate (un faux "unchanged" gèlerait le backlog). No-op visible, retry
		// au prochain cycle. Renvoie aussi l'erreur catégorisée pour la métrique de cut.
		snapshotLog.WarnContext(ctx, "snapshot: gather ready incomplet, cut sauté (retry prochain cycle)",
			"titleSlug", slug, "err", err)
		return SnapshotResult{NoopReason: "read_incomplete"}, fmt.Errorf("%w: %w", ErrSnapshotRead, err)
	}
	if len(readyIDs) == 0 {
		return SnapshotResult{NoopReason: "no_ready_matches"}, nil
	}
	if unchanged := snapshotUnchanged(opts.Paths, slug, currentVersion, len(readyIDs), watermark); unchanged {
		return SnapshotResult{Version: currentVersion, ReadyMatchCount: len(readyIDs),
			PartialMatchCount: len(partial), NoopReason: "unchanged"}, nil
	}

	version := currentVersion + 1
	versionDir := opts.Paths.SnapshotVersionDir(slug, version)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return SnapshotResult{}, fmt.Errorf("mkdir version dir: %w", err)
	}

	// Le snapshot ne contient que les FAITS SHARED (match-immutables) filtrés au set
	// ready : la lecture scoped MatchView sert le shared depuis le snapshot, et lit les
	// dérivés player (perf/LUSR/citations) sur la player DB live (non B-swap, pas de
	// stall) — donc aucun dérivé à exporter ici.
	parts, err := exportSharedFacts(ctx, opts.Shared, versionDir, readyIDs)
	if err != nil {
		return SnapshotResult{}, fmt.Errorf("%w: %w", ErrSnapshotCopy, err)
	}

	m := SnapshotManifest{
		Version:           version,
		TitleSlug:         slug,
		CreatedAt:         now.UTC().Format(time.RFC3339),
		Watermark:         watermark.UTC().Format(time.RFC3339),
		ReadyMatchCount:   len(readyIDs),
		PartialMatchCount: len(partial),
		SchemaVersion:     SnapshotSchemaVersion,
		Partitions:        parts,
	}
	if err := writeManifest(versionDir, m); err != nil {
		return SnapshotResult{}, fmt.Errorf("%w: %w", ErrSnapshotManifest, err)
	}
	if err := flipCurrent(currentPath, version, now.UTC().Format(time.RFC3339)); err != nil {
		return SnapshotResult{}, fmt.Errorf("%w: %w", ErrSnapshotManifest, err)
	}
	if removed, err := applyRetention(snapshotsDir, opts.RetentionKeep, version); err != nil {
		snapshotLog.WarnContext(ctx, "snapshot: rétention échouée (non bloquant)", "err", err, "titleSlug", slug)
	} else if len(removed) > 0 {
		snapshotLog.InfoContext(ctx, "snapshot: versions purgées", "titleSlug", slug, "removed", removed)
	}

	snapshotLog.InfoContext(ctx, "snapshot: version produite",
		"titleSlug", slug, "version", version, "ready_matches", len(readyIDs),
		"partial_matches", len(partial), "partitions", len(parts))
	return SnapshotResult{Produced: true, Version: version, ReadyMatchCount: len(readyIDs),
		PartialMatchCount: len(partial)}, nil
}

// gatherReadySet agrège l'union des matchs ready de tous les joueurs du titre, le set
// des matchs partiels (partial_reasons non vide) et le watermark (max snapshot_ready_at).
// Best-effort par joueur : une player DB absente/illisible est loggée et ignorée (un
// titre fraîchement activé peut n'avoir aucune table _latest).
func gatherReadySet(ctx context.Context, opts SnapshotOptions) (ids []string, partial map[string]bool, watermark time.Time, err error) {
	idSet := make(map[string]bool)
	partial = make(map[string]bool)
	for _, gt := range opts.Players {
		db, release, oerr := opts.PlayerOpener.OpenPlayerRO(ctx, gt)
		if oerr != nil {
			// Player DB absente/illisible (joueur offline, titre fraîchement activé) :
			// best-effort, on ignore CE joueur (pas une troncature de données existantes).
			snapshotLog.WarnContext(ctx, "snapshot: player DB indisponible (ready ignoré)", "player", gt, "err", oerr)
			continue
		}
		wm, rerr := collectPlayerReady(ctx, db, idSet, partial)
		release()
		if rerr != nil {
			// Lecture interrompue → set ready POTENTIELLEMENT TRONQUÉ. On refuse de figer
			// un snapshot partiel : erreur dure, le caller re-tentera au prochain cycle.
			return nil, nil, time.Time{}, fmt.Errorf("player %s: %w", gt, rerr)
		}
		if wm.After(watermark) {
			watermark = wm
		}
	}
	ids = make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Strings(ids) // déterministe (manifest + tests reproductibles)
	return ids, partial, watermark, nil
}

// collectPlayerReady lit les matchs ready d'un joueur dans idSet/partial et retourne
// son watermark local. Retourne une ERREUR si la lecture échoue/s'interrompt (query,
// scan, rows.Err) : le caller NE DOIT PAS figer un set ready tronqué (perte silencieuse
// de matchs interdite par la doctrine convergent-sync). Une player DB simplement absente
// est gérée en amont (OpenPlayerRO échoue → joueur ignoré), pas ici.
func collectPlayerReady(ctx context.Context, db *sql.DB, idSet, partial map[string]bool) (time.Time, error) {
	var watermark time.Time
	rows, err := db.QueryContext(ctx, `
		SELECT match_id, COALESCE(partial_reasons, '[]'), snapshot_ready_at
		FROM player_match_enrichment_latest
		WHERE snapshot_ready_at IS NOT NULL`)
	if err != nil {
		return watermark, fmt.Errorf("lecture ready: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var id, reasons string
		var readyAt time.Time
		if err := rows.Scan(&id, &reasons, &readyAt); err != nil {
			return watermark, fmt.Errorf("scan ready: %w", err)
		}
		idSet[id] = true
		if reasons != "[]" && reasons != "" {
			partial[id] = true
		}
		if readyAt.After(watermark) {
			watermark = readyAt
		}
	}
	if err := rows.Err(); err != nil {
		return watermark, fmt.Errorf("itération ready: %w", err)
	}
	return watermark, nil
}

// snapshotUnchanged compare le cut candidat (compte ready + watermark) au manifest de
// la version courante : si identiques, rien de neuf à figer (no-op). Toute erreur de
// lecture du manifest courant ⇒ traiter comme "changé" (on reproduit, plus sûr).
func snapshotUnchanged(paths *title.PathResolver, slug string, currentVersion int64, readyCount int, watermark time.Time) bool {
	if currentVersion == 0 {
		return false
	}
	data, err := os.ReadFile(filepath.Join(paths.SnapshotVersionDir(slug, currentVersion), "manifest.json"))
	if err != nil {
		return false
	}
	var m SnapshotManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return false
	}
	return m.ReadyMatchCount == readyCount && m.Watermark == watermark.UTC().Format(time.RFC3339)
}
