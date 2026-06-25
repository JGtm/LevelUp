// Package ops — snapshot_manifest.go : structures et primitives du manifest des
// snapshots Parquet immuables versionnés (durabilité / lecture découplée du B-swap,
// Phase 2 du plan PLAN_DURABILITE_SNAPSHOT_IMMUABLE).
//
// Un snapshot est une version cohérente et figée d'un sous-ensemble du dataset d'un
// titre : les faits bruts immuables (shared) + les dérivés ancrés (player DB), filtrés
// aux seuls matchs `snapshot_ready_at IS NOT NULL` (complets). Chaque version vit dans
// son répertoire `vNNN…/` ; un pointeur atomique `CURRENT.json` désigne la version
// active. Aucune écriture en place : produire = écrire une nouvelle version + flipper
// CURRENT (modèle lakehouse). Le flip par os.Rename garantit qu'un lecteur ne voit
// jamais un état mixte.
package ops

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SnapshotSchemaVersion : version du SCHÉMA du manifest/layout (incompatibilité =
// bump). Distinct du numéro de version monotone d'un snapshot (qui croît à chaque cut).
const SnapshotSchemaVersion = 1

// SharedReadOpener ouvre la base partagée d'un titre en LECTURE. Découple le
// producteur (`internal/ops`) du subsystème platform/duckdb : l'implémentation
// concrète (câblée au boot) passe par OpenReadForQuery (réutilise le handle caché
// RW/RO du process — JAMAIS `?access_mode=read_only` direct ni ATTACH, incident
// 2026-06-01). Le release retourné ne ferme le handle que s'il a été ouvert ici.
type SharedReadOpener interface {
	OpenSharedRO(ctx context.Context) (*sql.DB, func(), error)
}

// PlayerReadOpener ouvre la player DB d'un joueur (par gamertag) en LECTURE, même
// contrat que SharedReadOpener (OpenReadForQuery, zéro ATTACH).
type PlayerReadOpener interface {
	OpenPlayerRO(ctx context.Context, gamertag string) (*sql.DB, func(), error)
}

// SharedReadOpenerFunc adapte une closure en SharedReadOpener (câblage par fermeture
// sur OpenReadForQuery côté cmd/server, sans définir un type dédié).
type SharedReadOpenerFunc func(ctx context.Context) (*sql.DB, func(), error)

// OpenSharedRO satisfait SharedReadOpener.
func (f SharedReadOpenerFunc) OpenSharedRO(ctx context.Context) (*sql.DB, func(), error) {
	return f(ctx)
}

// PlayerReadOpenerFunc adapte une closure en PlayerReadOpener.
type PlayerReadOpenerFunc func(ctx context.Context, gamertag string) (*sql.DB, func(), error)

// OpenPlayerRO satisfait PlayerReadOpener.
func (f PlayerReadOpenerFunc) OpenPlayerRO(ctx context.Context, gamertag string) (*sql.DB, func(), error) {
	return f(ctx, gamertag)
}

// SnapshotManifest décrit une version de snapshot (écrit dans `vNNN…/manifest.json`).
type SnapshotManifest struct {
	Version           int64           `json:"version"`
	TitleSlug         string          `json:"title_slug"`
	CreatedAt         string          `json:"created_at"`          // RFC3339 UTC
	Watermark         string          `json:"watermark"`           // max(snapshot_ready_at) couvert, RFC3339
	ReadyMatchCount   int             `json:"ready_match_count"`   // matchs uniques (union joueurs) inclus
	PartialMatchCount int             `json:"partial_match_count"` // dont avec partial_reasons non vide
	SchemaVersion     int             `json:"schema_version"`
	Partitions        []PartitionInfo `json:"partitions"`
}

// PartitionInfo décrit un fichier Parquet du snapshot (un par table×mois côté shared,
// un par table×joueur côté dérivés). RelPath est relatif au répertoire de version.
type PartitionInfo struct {
	Table     string `json:"table"`
	Month     string `json:"month,omitempty"`  // "YYYY-MM" (faits shared) ; vide pour les dérivés
	Player    string `json:"player,omitempty"` // gamertag (dérivés) ; vide pour les faits shared
	RelPath   string `json:"rel_path"`
	RowCount  int64  `json:"row_count"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

// snapshotPointer est le contenu de CURRENT.json — un pointeur léger vers la version
// active (le manifest complet vit dans le répertoire de version).
type snapshotPointer struct {
	Version     int64  `json:"version"`
	ManifestRel string `json:"manifest_rel"`
	FlippedAt   string `json:"flipped_at"` // RFC3339 UTC
}

// writeManifest sérialise le manifest dans `versionDir/manifest.json`.
func writeManifest(versionDir string, m SnapshotManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return os.WriteFile(filepath.Join(versionDir, "manifest.json"), data, 0o644)
}

// readCurrent lit le numéro de version actif depuis CURRENT.json. Retourne 0 (et nil)
// si le fichier n'existe pas encore (répertoire vierge = aucune version) — le premier
// cut produira donc la version 1.
func readCurrent(currentPath string) (int64, error) {
	data, err := os.ReadFile(currentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("lecture CURRENT.json: %w", err)
	}
	var p snapshotPointer
	if err := json.Unmarshal(data, &p); err != nil {
		return 0, fmt.Errorf("parse CURRENT.json: %w", err)
	}
	return p.Version, nil
}

// flipCurrent bascule ATOMIQUEMENT le pointeur CURRENT.json vers `version`. Écrit un
// fichier temporaire dans le même répertoire puis os.Rename (atomique, remplace
// l'existant sur Windows comme POSIX) → un lecteur concurrent voit toujours soit N,
// soit N+1, jamais un fichier tronqué.
func flipCurrent(currentPath string, version int64, nowRFC3339 string) error {
	ptr := snapshotPointer{
		Version:     version,
		ManifestRel: filepath.Join(snapshotVersionName(version), "manifest.json"),
		FlippedAt:   nowRFC3339,
	}
	data, err := json.MarshalIndent(ptr, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal CURRENT.json: %w", err)
	}
	tmp := currentPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("écriture CURRENT.json.tmp: %w", err)
	}
	if err := os.Rename(tmp, currentPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename CURRENT.json: %w", err)
	}
	return nil
}

// snapshotVersionName retourne le nom de répertoire zéro-paddé d'une version
// (ordre lexicographique = ordre chronologique). Doit coïncider avec le suffixe
// produit par PathResolver.SnapshotVersionDir.
func snapshotVersionName(version int64) string {
	return fmt.Sprintf("v%020d", version)
}

// sha256File calcule le SHA-256 hexadécimal d'un fichier (intégrité des partitions
// dans le manifest).
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
