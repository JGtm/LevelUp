// Package archlint — no_second_artifact_sink_test.go : ratchet du PUITS D'ARTEFACTS
// (lot B v7.5, notification Discord groupée des rejeux).
//
// `replaybuild.SetArtifactStoredSink` câble le puits par lequel passent TOUTES les
// écritures d'artefact du serveur. Il est fait pour être appelé UNE SEULE FOIS, au boot
// (cmd/server/main.go). Un second câblage n'ajouterait pas un second observateur : il
// REMPLACERAIT le premier — le fichier qui gagne dépendrait de l'ordre de boot, et le
// symptôme serait « des notifications qui manquent », impossible à rattacher à sa cause.
//
// Le puits lui-même (internal/replaybuild/artifact_events.go) reste libre de définir la
// fonction ; ce ratchet ne compte que les APPELS, hors tests.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// artifactSinkSetter : l'appel surveillé. Le point d'appel légitime est unique et déclaré
// ci-dessous ; toute autre occurrence hors test est une violation.
const artifactSinkSetter = "SetArtifactStoredSink("

// artifactSinkAllowedCallers : sites AUTORISÉS, chemin relatif à apps/go-api.
// Toute entrée ajoutée ici doit porter une justification écrite et datée.
var artifactSinkAllowedCallers = map[string]string{
	"internal/api/wire/registry_replay_notify.go": "2026-08-26 — câblage unique au boot (InstallReplayNotify)",
	"internal/replaybuild/artifact_events.go":     "2026-08-26 — définition du setter lui-même",
}

func TestNoSecondArtifactStoredSink(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // internal/archlint -> internal -> apps/go-api

	var violations []string
	err := filepath.WalkDir(apiRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", ".git", "node_modules", "tmp":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(apiRoot, path)
		rel = filepath.ToSlash(rel)
		if _, allowed := artifactSinkAllowedCallers[rel]; allowed {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, artifactSinkSetter) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk apps/go-api: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("second câblage du puits d'artefacts détecté (%d) — il REMPLACERAIT le "+
			"câblage de boot, pas ne s'y ajouterait :\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}
