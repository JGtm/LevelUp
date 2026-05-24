package testfixtures

import (
	"path/filepath"
	"runtime"
	"sync"
)

// repoRoot retourne le chemin absolu de la racine du repo (parent d'apps/).
// Calculé une seule fois via runtime.Caller(0) puis caché.
//
// Logique : ce fichier est à apps/go-api/internal/testfixtures/paths.go.
// On remonte 4 niveaux pour atteindre la racine du repo.
var (
	repoRootOnce sync.Once
	repoRootPath string
)

// RepoRoot retourne le chemin absolu de la racine du repo LevelUp.
//
// Utilisable depuis n'importe quel package de test : la fonction se localise
// elle-même via runtime.Caller, indépendante du cwd du test.
func RepoRoot() string {
	repoRootOnce.Do(func() {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			panic("testfixtures: runtime.Caller(0) a echoue — impossible de localiser le repo root")
		}
		// file = .../apps/go-api/internal/testfixtures/paths.go
		// Dir(file) = .../apps/go-api/internal/testfixtures
		// 4 remontees = .../ (racine repo)
		dir := filepath.Dir(file)
		for i := 0; i < 4; i++ {
			dir = filepath.Dir(dir)
		}
		repoRootPath = dir
	})
	return repoRootPath
}

// TestdataDir retourne le chemin du dossier testdata sous apps/go-api/internal/sync/.
//
// Convention : toutes les fixtures partagees vivent sous ce dossier (le nom
// "testdata" est ignore par le compilateur Go, donc safe pour stocker des
// binaires lourds sans casser go build).
func TestdataDir() string {
	return filepath.Join(RepoRoot(), "apps", "go-api", "internal", "sync", "testdata")
}

// JGtmFullMatchDir retourne le chemin du fixture E2E JGtm full match.
//
// Contenu attendu (cf. testdata/jgtm_full_match/README.md, gitignore line 206-209) :
//   - manifest_raw.json
//   - chunks/filmChunk0..29
//   - api_match_stats.json
//   - api_skill.json
//   - api_match_history_page0.json
//
// Si le fixture n'a pas ete telecharge localement, JGtmFullMatchAvailable()
// retourne false et les tests doivent t.Skip.
func JGtmFullMatchDir() string {
	return filepath.Join(TestdataDir(), "jgtm_full_match")
}

// WALDir retourne le chemin du dossier data/wal/ a la racine du repo
// (69 batches WAL JSON captures en production, 7.1 MB).
//
// Le contenu de ce dossier n'est PAS versionne (gitignore line 144) mais
// reste present sur les machines dev pour les tests.
func WALDir() string {
	return filepath.Join(RepoRoot(), "data", "wal")
}
