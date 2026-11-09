//go:build e2e_windows && windows

// e2e_refresh_only_test.go — bout en bout de -mode refresh-only sur le binaire compilé (ADR 0045
// D-10) : sortie 0, stdout vide (ce n'est pas le mode `desktop` de la coquille), aucune base ouverte,
// journal sur stderr et sous la racine des journaux, verrou rendu à la sortie ; et, quand l'application
// tient déjà le verrou, sortie 0 sans rien faire. La rotation d'un jeton contre un faux serveur OAuth
// est couverte en processus par refresh_only_test.go : le binaire parle au vrai Microsoft, ce qu'un
// test de CI ne doit pas faire.
//
// Voie : job go-test-windows-cgo de la CI (`-tags=e2e_windows -run TestE2E`).
package main

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/instancelock"
)

const e2eRefreshOnlyTimeout = 90 * time.Second

// e2eRunRefreshOnly lance le binaire en -mode refresh-only sur root et rend sa sortie standard,
// sa sortie d'erreur et son code de sortie.
func e2eRunRefreshOnly(t *testing.T, bin, root string) (stdout, stderr string, code int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), e2eRefreshOnlyTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-mode", "refresh-only")
	cmd.Dir = root
	cmd.Env = e2eEnv(root, 0)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("refresh-only n'est pas sorti en %s\n%s", e2eRefreshOnlyTimeout, errb.String())
	}
	code = 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("lancement : %v", err)
		}
		code = exitErr.ExitCode()
	}
	return out.String(), errb.String(), code
}

// TestE2ERefreshOnly : racine vide, puis verrou tenu par l'application.
func TestE2ERefreshOnly(t *testing.T) {
	bin := e2eBuildServer(t)
	root := e2eNewDataRoot(t)
	lockPath := title.NewPathResolver(root).InstanceLockPath()

	t.Run("racine_vide", func(t *testing.T) {
		stdout, stderr, code := e2eRunRefreshOnly(t, bin, root)
		if code != 0 {
			t.Fatalf("code de sortie %d, attendu 0\n%s", code, stderr)
		}
		if stdout != "" {
			t.Errorf("stdout non vide en refresh-only : %q", stdout)
		}
		if !strings.Contains(stderr, "refresh-only: terminé") {
			t.Errorf("journal de fin absent de stderr :\n%s", stderr)
		}
		e2eAssertNoDatabaseUnder(t, root)
		e2eAssertLogged(t, filepath.Join(root, "logs"), "refresh-only: terminé")
		lock, err := instancelock.Acquire(lockPath)
		if err != nil {
			t.Fatalf("verrou non rendu à la sortie : %v", err)
		}
		_ = lock.Release()
	})

	t.Run("verrou_tenu_par_l_application", func(t *testing.T) {
		held, err := instancelock.Acquire(lockPath)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = held.Release() }()

		stdout, stderr, code := e2eRunRefreshOnly(t, bin, root)
		if code != 0 {
			t.Fatalf("code de sortie %d, attendu 0\n%s", code, stderr)
		}
		if stdout != "" {
			t.Errorf("stdout non vide : %q", stdout)
		}
		if !strings.Contains(stderr, "une instance tient le dossier de données") {
			t.Errorf("le verrou tenu n'est pas journalisé :\n%s", stderr)
		}
		if strings.Contains(stderr, "refresh-only: terminé") {
			t.Error("la vérification des jetons a tourné alors que l'application tient le verrou")
		}
	})
}

// e2eAssertNoDatabaseUnder exige qu'aucune base ni WAL n'existe sous root.
func e2eAssertNoDatabaseUnder(t *testing.T, root string) {
	t.Helper()
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			name := strings.ToLower(d.Name())
			if strings.HasSuffix(name, ".duckdb") || strings.HasSuffix(name, ".wal") {
				t.Errorf("refresh-only a ouvert une base : %s", p)
			}
		}
		return nil
	})
}

// e2eAssertLogged exige qu'un fichier *.log sous dir contienne msg.
func e2eAssertLogged(t *testing.T, dir, msg string) {
	t.Helper()
	found := false
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".log") {
			return nil
		}
		if data, rerr := os.ReadFile(p); rerr == nil && strings.Contains(string(data), msg) {
			found = true
		}
		return nil
	})
	if !found {
		t.Errorf("aucun journal sous %s ne contient %q", dir, msg)
	}
}
