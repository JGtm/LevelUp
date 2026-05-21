// Package port — git.go : interface GitProvider.
//
// P8.10 (revue 2026-04-29 gap #4) : abstraction de l'accès git pour rendre
// les services qui en dépendent (release notes) testables sans dépendance
// au binaire git, et pour faire respecter la séparation handler ↔ logique
// métier (l'exec.Command vit désormais dans `platform/git/`).
package port

import "context"

// GitProvider expose les opérations git nécessaires à la reconstruction de
// l'historique des release notes.
type GitProvider interface {
	// LogSHAs retourne les SHAs (du plus récent au plus ancien) ayant
	// modifié `relPath` (chemin relatif au repoRoot). `--all` couvre toutes
	// les branches.
	LogSHAs(ctx context.Context, repoRoot, relPath string) ([]string, error)

	// ShowFile retourne le contenu de `relPath` au commit `sha`.
	ShowFile(ctx context.Context, repoRoot, sha, relPath string) (string, error)
}
