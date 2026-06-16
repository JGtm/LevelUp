// Package auth — migrate_watcher_tokens.go : copy-migration boot du store de tokens
// watcher vers le layout namespacé titre (Phase 1.5 PMT-2 leg 5, MT-02).
//
// Le chemin du store est passé de data/auth/watcher_tokens (legacy global) à
// data/titles/{slug}/auth/watcher_tokens (namespacé titre). Pour ne pas déconnecter les
// utilisateurs existants (leurs {xuid}.json vivaient dans l'ancien dossier), cette fonction
// RECOPIE les fichiers du legacy vers le nouveau chemin au démarrage.
//
// GARDE-FOU : NON DESTRUCTIVE — l'ancien dossier est PRÉSERVÉ tel quel (filet de retour).
// IDEMPOTENTE — n'écrase JAMAIS un fichier déjà présent dans le nouveau dossier (un token
// rafraîchi post-migration ne doit pas être réécrasé par la version legacy périmée).
// Appelée au boot serveur ET par les CLI qui écrivent des tokens (token-capture/import)
// AVANT toute création de MultiUserTokenStore.
package auth

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// MigrateWatcherTokens recopie les fichiers {xuid}.json de legacyDir vers newDir si absents.
// Retourne le nombre de fichiers copiés. No-op si legacyDir == newDir ou si legacyDir absent.
// Ne supprime jamais legacyDir (filet de retour). Idempotente.
func MigrateWatcherTokens(legacyDir, newDir string) (copied int, err error) {
	if legacyDir == "" || newDir == "" || legacyDir == newDir {
		return 0, nil
	}

	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // pas de legacy → rien à migrer (install neuve)
		}
		return 0, fmt.Errorf("migrate_watcher_tokens: scan legacy %s: %w", legacyDir, err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		dst := filepath.Join(newDir, name)
		if _, statErr := os.Stat(dst); statErr == nil {
			continue // déjà présent (potentiellement plus récent) → ne JAMAIS écraser
		}
		data, rerr := os.ReadFile(filepath.Join(legacyDir, name))
		if rerr != nil {
			slog.Warn("migrate_watcher_tokens: lecture legacy échouée, ignoré", "name", name, "err", rerr)
			continue
		}
		if mkErr := os.MkdirAll(newDir, 0o700); mkErr != nil {
			return copied, fmt.Errorf("migrate_watcher_tokens: mkdir %s: %w", newDir, mkErr)
		}
		// Write-to-temp + rename atomique (cohérent avec MultiUserTokenStore.upsertLocked).
		tmp := dst + ".tmp"
		if werr := os.WriteFile(tmp, data, 0o600); werr != nil {
			return copied, fmt.Errorf("migrate_watcher_tokens: écriture %s: %w", dst, werr)
		}
		if rnErr := os.Rename(tmp, dst); rnErr != nil {
			_ = os.Remove(tmp)
			return copied, fmt.Errorf("migrate_watcher_tokens: rename %s: %w", dst, rnErr)
		}
		copied++
	}

	if copied > 0 {
		slog.Info("migrate_watcher_tokens: tokens recopiés vers le layout namespacé titre",
			"copied", copied, "from", legacyDir, "to", newDir)
	}
	return copied, nil
}
