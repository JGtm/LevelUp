// Package ops — snapshot_retention.go : rétention des versions de snapshot.
//
// Modèle de production retenu : FULL RE-EXPORT change-gated. Chaque cut réécrit
// l'intégralité des partitions de la version (tous les matchs ready), il n'y a donc
// PAS de petites partitions par-batch qui s'accumulent → AUCUNE compaction n'est
// nécessaire (on n'écrit pas de `compactMonth` : ce serait du code mort). La seule
// hygiène disque est de borner le nombre de versions complètes conservées (rollback).
package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// listSnapshotVersions retourne les numéros de version présents dans snapshotsDir
// (répertoires `vNNN…`), triés croissant. Ignore CURRENT.json et tout autre fichier.
func listSnapshotVersions(snapshotsDir string) ([]int64, error) {
	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("lecture snapshots dir: %w", err)
	}
	var versions []int64
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "v") {
			continue
		}
		v, err := strconv.ParseInt(strings.TrimPrefix(name, "v"), 10, 64)
		if err != nil {
			continue // répertoire non conforme → ignoré
		}
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	return versions, nil
}

// applyRetention conserve les `keep` versions les plus récentes ET la version active
// `currentVersion` (jamais supprimée, même si plus ancienne que les keep — sécurité
// rollback), et supprime les autres. keep <= 0 désactive la rétention (no-op).
// Retourne la liste des versions effectivement supprimées.
func applyRetention(snapshotsDir string, keep int, currentVersion int64) ([]int64, error) {
	if keep <= 0 {
		return nil, nil
	}
	versions, err := listSnapshotVersions(snapshotsDir)
	if err != nil {
		return nil, err
	}
	if len(versions) <= keep {
		return nil, nil
	}
	// Versions à garder : les `keep` plus hautes (versions est trié croissant).
	protected := make(map[int64]bool, keep+1)
	for _, v := range versions[len(versions)-keep:] {
		protected[v] = true
	}
	protected[currentVersion] = true // jamais supprimer la version active

	var removed []int64
	for _, v := range versions {
		if protected[v] {
			continue
		}
		dir := filepath.Join(snapshotsDir, snapshotVersionName(v))
		if err := os.RemoveAll(dir); err != nil {
			return removed, fmt.Errorf("suppression version %d: %w", v, err)
		}
		removed = append(removed, v)
	}
	return removed, nil
}
