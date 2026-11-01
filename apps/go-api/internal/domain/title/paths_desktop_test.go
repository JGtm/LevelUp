package title

import (
	"path/filepath"
	"testing"
)

// Le port stable et le verrou d'instance vivent sous la racine de données, jamais sous la racine
// de ressources (ADR 0008 amendé, ADR 0045 D-9 et D-10).
func TestPathResolver_DesktopPortEtVerrouSousLaRacineDeDonnees(t *testing.T) {
	pr := NewPathResolverRoots(Roots{Resources: "/res", Data: "/donnees", Logs: "/journaux", DataSet: true})
	if got, want := pr.DesktopPortPath(), filepath.Join("/donnees", "desktop", "port"); got != want {
		t.Errorf("DesktopPortPath() = %q, attendu %q", got, want)
	}
	if got, want := pr.InstanceLockPath(), filepath.Join("/donnees", "levelup.lock"); got != want {
		t.Errorf("InstanceLockPath() = %q, attendu %q", got, want)
	}
}
