// Package atomicfile — écriture de fichier « atomique d'abord, in-place en repli ».
//
// POURQUOI (constat deploy v7.2.0, 2026-07-25). Le pattern canonique d'écriture
// atomique (fichier temporaire dans le même répertoire puis os.Rename) échoue
// SYSTÉMATIQUEMENT quand la cible est bind-montée FICHIER dans un conteneur : le
// montage épingle l'inode, et rename(2) répond EBUSY (« device or resource busy »).
// app_settings.json est exactement dans ce cas en production — toute écriture
// runtime de settings (last_notified_version, toggles admin) échouait, d'où la
// notification Discord « nouvelle version » rejouée à chaque redémarrage.
//
// STRATÉGIE. Tenter l'écriture atomique ; si l'environnement l'INTERDIT (EBUSY au
// rename, ou répertoire parent non inscriptible pour le temporaire), replier sur
// une écriture in-place (truncate + write + fsync) qui réutilise l'inode existant
// — la seule qui traverse un bind-mount fichier. Toute autre erreur est remontée
// telle quelle : le repli couvre une contrainte d'environnement connue, pas un
// diagnostic manquant.
//
// LIMITE ASSUMÉE DU REPLI : l'écriture in-place N'EST PAS atomique. Entre le
// truncate et la fin du write, un crash process/machine laisse un fichier tronqué
// ou vide. Le risque est BORNÉ, pas éliminé :
//   - le contenu complet est sérialisé EN MÉMOIRE par l'appelant — aucun calcul ni
//     aucune I/O de lecture entre le truncate et le write ;
//   - un SEUL appel à Write (pas de flux incrémental) ;
//   - fsync avant fermeture : au retour de WriteFile le contenu est sur disque.
//
// Reste la fenêtre truncate→write (de l'ordre de la microseconde). Les fichiers
// visés (settings JSON) sont RECONSTRUCTIBLES : un fichier vide relit les défauts,
// aucune donnée métier n'est perdue. Ne PAS utiliser ce package pour un fichier
// dont la troncature serait une perte irréversible.
package atomicfile

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
)

// tmpPattern — préfixe des temporaires créés dans le répertoire de la cible.
const tmpPattern = ".levelup-atomic-*"

// Points d'injection pour les tests (simulation d'un rename EBUSY et d'un
// répertoire parent non inscriptible). JAMAIS réassignés en production.
var (
	renameFile = os.Rename
	createTemp = os.CreateTemp
)

// WriteFile écrit `data` dans `path`, atomiquement quand l'environnement le
// permet, sinon in-place (cf. LIMITE ASSUMÉE en tête de package).
//
// `perm` ne s'applique qu'à la CRÉATION : sur un fichier existant (cas du
// bind-mount) le mode d'origine est conservé — on ne chmod jamais l'inode monté.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	tmp, err := createTemp(filepath.Dir(path), tmpPattern)
	if err != nil {
		// Répertoire parent non inscriptible : le temporaire est impossible, mais
		// l'inode cible peut rester ouvrable en écriture (bind-mount fichier).
		slog.Warn("atomicfile: temporaire impossible, repli in-place NON ATOMIQUE",
			"path", path, "err", err)
		return writeInPlace(path, data, perm)
	}
	tmpName := tmp.Name()
	if err := finalizeTemp(tmp, data, perm); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := renameFile(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		if !isBusy(err) {
			return fmt.Errorf("atomicfile: rename %s → %s: %w", tmpName, path, err)
		}
		slog.Warn("atomicfile: rename EBUSY (cible bind-montée fichier), repli in-place NON ATOMIQUE",
			"path", path, "err", err)
		return writeInPlace(path, data, perm)
	}
	return nil
}

// finalizeTemp écrit le contenu dans le temporaire, aligne son mode et le ferme.
func finalizeTemp(f *os.File, data []byte, perm os.FileMode) error {
	name := f.Name()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("atomicfile: write %s: %w", name, err)
	}
	// Chmod du TEMPORAIRE seulement (os.CreateTemp crée en 0600) : la cible finale
	// hérite de ce mode au rename.
	if err := f.Chmod(perm); err != nil {
		_ = f.Close()
		return fmt.Errorf("atomicfile: chmod %s: %w", name, err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("atomicfile: sync %s: %w", name, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("atomicfile: close %s: %w", name, err)
	}
	return nil
}

// writeInPlace tronque `path` et y écrit `data` en UN SEUL Write, suivi d'un
// fsync. Non atomique — cf. LIMITE ASSUMÉE en tête de package.
func writeInPlace(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("atomicfile: open in-place %s: %w", path, err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("atomicfile: write in-place %s: %w", path, err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("atomicfile: sync in-place %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("atomicfile: close in-place %s: %w", path, err)
	}
	return nil
}

// isBusy — rename(2) refusé parce que la cible est épinglée (bind-mount fichier).
// C'EST le seul échec de rename qui justifie le repli : tout le reste (ENOSPC,
// EACCES sur la cible, EXDEV…) est une anomalie à remonter, pas à contourner.
func isBusy(err error) bool {
	return errors.Is(err, syscall.EBUSY)
}
