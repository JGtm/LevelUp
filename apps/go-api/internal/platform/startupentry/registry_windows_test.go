//go:build windows

package startupentry

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// TestRegistryStore_OnAPrivateKey exerce le vrai registre sur une clé privée à nom aléatoire
// (HKCU\Software\LevelUpTest-<hex>) : la vraie clé Run n'est jamais ouverte. La clé est supprimée à la fin.
func TestRegistryStore_OnAPrivateKey(t *testing.T) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	base := `Software\LevelUpTest-` + hex.EncodeToString(b[:])
	runPath, approvedPath := base+`\Run`, base+`\Approved`
	t.Cleanup(func() {
		for _, p := range []string{runPath, approvedPath, base} {
			_ = registry.DeleteKey(registry.CURRENT_USER, p) // clé de test ; absente = déjà nettoyée
		}
	})
	s := registryStore{run: runPath, approved: approvedPath}

	if _, present, err := s.RunValue(EntryName); err != nil || present {
		t.Fatalf("clé absente : present=%v err=%v", present, err)
	}
	if err := s.DeleteRunValue(EntryName); err != nil {
		t.Fatalf("suppression sur clé absente : %v", err)
	}
	for _, p := range []string{runPath, approvedPath} {
		k, _, err := registry.CreateKey(registry.CURRENT_USER, p, registry.SET_VALUE)
		if err != nil {
			t.Fatal(err)
		}
		if err := k.SetStringValue(EntryName, `"C:\x\levelup-desktop.exe" --minimized`); err != nil {
			t.Fatal(err)
		}
		k.Close()
	}
	got, present, err := s.RunValue(EntryName)
	if err != nil || !present || got != `"C:\x\levelup-desktop.exe" --minimized` {
		t.Fatalf("lecture : %q present=%v err=%v", got, present, err)
	}
	out, err := RemoveForInstall(s, `C:\x`)
	if err != nil || out != OutcomeRemoved {
		t.Fatalf("issue = %q err = %v", out, err)
	}
	if _, present, _ := s.RunValue(EntryName); present {
		t.Fatal("la valeur Run est encore là")
	}
	if k, err := registry.OpenKey(registry.CURRENT_USER, approvedPath, registry.QUERY_VALUE); err == nil {
		defer k.Close()
		if _, _, err := k.GetStringValue(EntryName); err == nil {
			t.Fatal("l'approbation est encore là")
		}
	}
	if err := s.DeleteApprovedValue(EntryName); err != nil {
		t.Fatalf("suppression répétée : %v", err)
	}
}
