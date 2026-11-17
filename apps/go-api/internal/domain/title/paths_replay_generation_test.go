package title

import (
	"path/filepath"
	"testing"
)

// Le fichier des demandes de génération vit sous AdminStateDir, dérivé de la racine de données.
func TestPathResolver_ReplayGenerationQueueSousAdminState(t *testing.T) {
	pr := NewPathResolverRoots(Roots{Resources: "/res", Data: "/donnees", Logs: "/journaux", DataSet: true})
	want := filepath.Join("/donnees", "global", "admin_state", "replay_generation_queue.json")
	if got := pr.ReplayGenerationQueuePath(); got != want {
		t.Errorf("ReplayGenerationQueuePath() = %q, attendu %q", got, want)
	}
	if filepath.Dir(pr.ReplayGenerationQueuePath()) != pr.AdminStateDir() {
		t.Errorf("le fichier doit être directement sous AdminStateDir")
	}
}
