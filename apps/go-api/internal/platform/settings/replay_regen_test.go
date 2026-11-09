package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReplayRegenPending_PersisteSansToucherAuxAutresCles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app_settings.json")
	if err := os.WriteFile(path, []byte(`{"lang":"fr","cle_inconnue":42}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(path)
	if on, err := s.ReplayRegenPending(); err != nil || on {
		t.Fatalf("défaut = %v, %v ; attendu faux", on, err)
	}
	if err := s.SetReplayRegenPending(true); err != nil {
		t.Fatal(err)
	}
	if on, _ := NewStore(path).ReplayRegenPending(); !on {
		t.Fatal("le drapeau survit à un nouveau magasin (redémarrage)")
	}
	raw, _ := os.ReadFile(path)
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if string(m["cle_inconnue"]) != "42" || string(m["lang"]) != `"fr"` || string(m["replay_regen_pending"]) != "true" {
		t.Fatalf("fichier = %s", raw)
	}
	if err := s.SetReplayRegenPending(false); err != nil {
		t.Fatal(err)
	}
	if on, _ := s.ReplayRegenPending(); on {
		t.Fatal("effacé")
	}
}

func TestReplayRegenPending_N_EntrePasDansLaReponseDeLAPI(t *testing.T) {
	cfg := defaultSettings()
	cfg.ReplayRegenPending = true
	raw, err := json.Marshal(ToResponse(cfg))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	if _, ok := m["replay_regen_pending"]; ok {
		t.Fatal("le drapeau est un état interne : jamais servi par GET /settings")
	}
}
