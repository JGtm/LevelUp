//go:build cgo

package parquetbackup

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/instancelock"
)

// TestRunRestoreCommand_ForceDoesNotBypassTheInstanceLock : --force autorise le remplacement de bases existantes,
// pas le passage outre un serveur en marche : le verrou tenu refuse toujours (S2). Le dry-run, qui n'écrit rien,
// l'ignore.
func TestRunRestoreCommand_ForceDoesNotBypassTheInstanceLock(t *testing.T) {
	ctx := context.Background()
	pr, authDir := seedRoot(t)
	dest := t.TempDir()
	m, err := Run(ctx, newOptions(pr, authDir, dest))
	if err != nil {
		t.Fatal(err)
	}
	set := filepath.Join(dest, m.Name)

	// Racine cible dont un serveur tient le verrou (tous profils : desktop, hébergé, dev, démo).
	to := filepath.Join(t.TempDir(), "data")
	target := title.NewPathResolverRoots(title.Roots{Resources: t.TempDir(), Data: to, DataSet: true})
	lock, err := instancelock.Acquire(target.InstanceLockPath())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Release() }()

	var out bytes.Buffer
	err = RunRestoreCommand(ctx, RestoreCommand{From: set, To: to, Force: true}, title.Roots{}, &out)
	if err == nil || !strings.Contains(err.Error(), "tourne") {
		t.Fatalf("restauration --force sous un verrou tenu : %v, attendu un refus", err)
	}
	if err := RunRestoreCommand(ctx, RestoreCommand{From: set, To: to, DryRun: true}, title.Roots{}, &out); err != nil {
		t.Fatalf("le dry-run n'écrit rien et ignore le verrou : %v", err)
	}

	_ = lock.Release()
	if err := RunRestoreCommand(ctx, RestoreCommand{From: set, To: to}, title.Roots{}, &out); err != nil {
		t.Fatalf("verrou libre : %v\n%s", err, out.String())
	}
	if got := countRows(t, target.SharedDBPath(testSlug), "match_registry"); got != 3 {
		t.Fatalf("lignes restaurées = %d, attendu 3", got)
	}
}

// TestParseRestoreArgs : les deux portes (serveur, alias) lisent les mêmes drapeaux.
func TestParseRestoreArgs(t *testing.T) {
	c, err := ParseRestoreArgs("levelup backup restore", []string{"--from", "a", "--to", "b", "--force", "--dry-run"}, &bytes.Buffer{})
	if err != nil || c != (RestoreCommand{From: "a", To: "b", Force: true, DryRun: true}) {
		t.Fatalf("%+v %v", c, err)
	}
	if _, err := ParseRestoreArgs("levelup backup restore", []string{"--from", "a"}, &bytes.Buffer{}); err == nil ||
		!strings.Contains(err.Error(), "levelup backup restore") {
		t.Fatalf("sans --to : %v, attendu l'usage de la commande", err)
	}
}
