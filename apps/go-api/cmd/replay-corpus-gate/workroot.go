package main

// workroot.go — LE CYCLE DE VIE DE LA RACINE DE TRAVAIL : creee jetable, nettoyee par defaut.

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// prepareWorkRoot cree la racine de travail et rend sa fonction de nettoyage. Un chemin
// EXPLICITE (`--work-root`) n'est jamais efface automatiquement — l'operateur qui l'a nomme en
// garde la responsabilite ; seul un dossier temporaire genere ICI est nettoye, sauf
// `--keep-work` (debug : inspecter l'artefact frais, le manifeste copie, les chunks recus).
func prepareWorkRoot(workRootFlag string, keepWork bool) (workRoot string, cleanup func(), err error) {
	if workRootFlag != "" {
		if err := os.MkdirAll(workRootFlag, 0o750); err != nil {
			return "", nil, fmt.Errorf("racine de travail explicite %s : %w", workRootFlag, err)
		}
		return workRootFlag, func() {}, nil
	}
	dir, err := os.MkdirTemp("", "replay-corpus-gate-*")
	if err != nil {
		return "", nil, fmt.Errorf("racine de travail temporaire : %w", err)
	}
	cleanup = func() {
		if keepWork {
			slog.Info("replay-corpus-gate: racine de travail conservee (--keep-work)", "chemin", dir)
			return
		}
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			slog.Warn("replay-corpus-gate: nettoyage de la racine de travail", "chemin", dir, "err", rmErr)
		}
	}
	return dir, cleanup, nil
}

// ecrireJSONGenerique depose une valeur serialisable — utilise par le rapport JSON optionnel.
func ecrireJSONGenerique(path string, v any) error {
	blob, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation du rapport : %w", err)
	}
	if err := os.WriteFile(path, append(blob, '\n'), 0o600); err != nil {
		return fmt.Errorf("ecriture du rapport %s : %w", path, err)
	}
	return nil
}
