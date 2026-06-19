package dbprofiles

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, content string) string {
	t.Helper()
	p := filepath.Join(dir, "db_profiles.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return p
}

func boolPtr(b bool) *bool { return &b }

func TestLoad_AbsentFile_ReturnsEmptyV3(t *testing.T) {
	p := filepath.Join(t.TempDir(), "db_profiles.json")
	f, err := NewStore(p).Load()
	if err != nil {
		t.Fatalf("Load absent: %v", err)
	}
	if f.Version != version3 || len(f.Profiles) != 0 {
		t.Fatalf("attendu v3 vide, reçu version=%q profiles=%d", f.Version, len(f.Profiles))
	}
}

func TestLoad_V2_MigratesUnderDefaultSlug(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `{"version":"2.1","profiles":{"JGtm":{"db_path":"x","xuid":"123"}}}`)
	f, err := NewStore(p).Load()
	if err != nil {
		t.Fatalf("Load v2: %v", err)
	}
	e, ok := f.Get(defaultSlug, "JGtm")
	if !ok || e.XUID != "123" {
		t.Fatalf("migration v2 ratée: %+v ok=%v", e, ok)
	}
}

func TestSaveLoad_RoundTrip_PreservesAdminAndUnknownTopLevel(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `{"version":"3.0","admin":"JGtm","weird_top":{"k":1},"profiles":{"halo_infinite":{"JGtm":{"db_path":"d","xuid":"1"}}}}`)
	st := NewStore(p)
	f, err := st.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if f.Admin != "JGtm" {
		t.Fatalf("admin perdu au Load: %q", f.Admin)
	}
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, _ := os.ReadFile(p)
	s := string(raw)
	if !strings.Contains(s, `"admin": "JGtm"`) {
		t.Fatalf("admin perdu au Save: %s", s)
	}
	if !strings.Contains(s, "weird_top") {
		t.Fatalf("clé top-level inconnue perdue: %s", s)
	}
}

// Régression : un champ d'entrée non typé (auth_only des followers présence) doit
// survivre à un round-trip Save (sinon corruption silencieuse du watcher).
func TestSaveLoad_PreservesUnknownEntryField_AuthOnly(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `{"version":"3.0","profiles":{"halo_infinite":{"Follower":{"db_path":"","xuid":"9","waypoint_player":"Follower","auth_only":true}}}}`)
	st := NewStore(p)
	f, err := st.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, _ := os.ReadFile(p)
	if !strings.Contains(string(raw), `"auth_only": true`) {
		t.Fatalf("auth_only perdu au round-trip: %s", string(raw))
	}
}

func TestMutate_AtomicReadModifyWrite(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `{"version":"3.0","profiles":{"halo_infinite":{"JGtm":{"db_path":"d","xuid":"1"}}}}`)
	st := NewStore(p)
	err := st.Mutate(func(f *File) error {
		f.Set("halo_5", "JGtm", Entry{DBPath: "d5", XUID: "1"})
		return nil
	})
	if err != nil {
		t.Fatalf("Mutate: %v", err)
	}
	f, _ := st.Load()
	if _, ok := f.Get("halo_5", "JGtm"); !ok {
		t.Fatalf("entrée halo_5 non persistée")
	}
	if _, ok := f.Get("halo_infinite", "JGtm"); !ok {
		t.Fatalf("entrée halo_infinite existante perdue")
	}
}

func TestMutate_ErrorRollsBack(t *testing.T) {
	dir := t.TempDir()
	orig := `{"version":"3.0","profiles":{"halo_infinite":{"JGtm":{"db_path":"d","xuid":"1"}}}}`
	p := writeFile(t, dir, orig)
	st := NewStore(p)
	wantErr := errors.New("boom")
	err := st.Mutate(func(f *File) error {
		f.Set("halo_5", "JGtm", Entry{DBPath: "x"})
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("attendu wantErr, reçu %v", err)
	}
	f, _ := st.Load()
	if _, ok := f.Get("halo_5", "JGtm"); ok {
		t.Fatalf("mutation persistée malgré erreur fn (pas de rollback)")
	}
}

func TestFindKey_CaseInsensitive(t *testing.T) {
	f := &File{Profiles: map[string]map[string]Entry{
		"halo_infinite": {"JGtm": {XUID: "1"}},
	}}
	key, ok := f.FindKey("halo_infinite", "jgTM")
	if !ok || key != "JGtm" {
		t.Fatalf("FindKey insensible à la casse ratée: %q ok=%v", key, ok)
	}
}

func TestSetSyncEnabled_RefusesLastActive(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `{"version":"3.0","profiles":{"halo_infinite":{"JGtm":{"db_path":"d","xuid":"1"}}}}`)
	st := NewStore(p)
	err := st.SetSyncEnabled("halo_infinite", "JGtm", false)
	if !errors.Is(err, ErrLastActiveTitle) {
		t.Fatalf("désactiver le dernier titre actif devrait échouer, reçu %v", err)
	}
}

func TestSetSyncEnabled_AllowedWhenAnotherActive(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `{"version":"3.0","profiles":{"halo_infinite":{"JGtm":{"db_path":"d","xuid":"1"}},"halo_5":{"JGtm":{"db_path":"d5","xuid":"1"}}}}`)
	st := NewStore(p)
	if err := st.SetSyncEnabled("halo_5", "JGtm", false); err != nil {
		t.Fatalf("pause d'un titre alors qu'un autre est actif devrait passer: %v", err)
	}
	f, _ := st.Load()
	e, _ := f.Get("halo_5", "JGtm")
	if e.IsActive() {
		t.Fatalf("halo_5 devrait être en pause")
	}
	// halo_infinite reste actif → re-pause de halo_5 OK ; pause halo_infinite maintenant interdit ? non, halo_5 est en pause donc halo_infinite est le dernier actif.
	if err := st.SetSyncEnabled("halo_infinite", "JGtm", false); !errors.Is(err, ErrLastActiveTitle) {
		t.Fatalf("halo_infinite est le dernier actif, pause devrait échouer, reçu %v", err)
	}
}

func TestRemoveEntry_RefusesLastActive_AllowsPaused(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `{"version":"3.0","profiles":{"halo_infinite":{"JGtm":{"db_path":"d","xuid":"1"}},"halo_5":{"JGtm":{"db_path":"d5","xuid":"1","sync_enabled":false}}}}`)
	st := NewStore(p)
	// halo_5 est en pause → le purger ne touche pas au dernier actif → autorisé.
	if err := st.RemoveEntry("halo_5", "JGtm"); err != nil {
		t.Fatalf("purge d'un titre en pause devrait passer: %v", err)
	}
	// Reste halo_infinite (seul actif) → purge refusée.
	if err := st.RemoveEntry("halo_infinite", "JGtm"); !errors.Is(err, ErrLastActiveTitle) {
		t.Fatalf("purge du dernier titre actif devrait échouer, reçu %v", err)
	}
}

func TestRemoveEntry_NotFound(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `{"version":"3.0","profiles":{"halo_infinite":{"JGtm":{"db_path":"d"}}}}`)
	st := NewStore(p)
	if err := st.RemoveEntry("halo_5", "JGtm"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("attendu ErrEntryNotFound, reçu %v", err)
	}
}

func TestActiveTitlesForGamertag_ExcludeAndPause(t *testing.T) {
	f := &File{Profiles: map[string]map[string]Entry{
		"halo_infinite": {"JGtm": {SyncEnabled: boolPtr(true)}},
		"halo_5":        {"JGtm": {SyncEnabled: boolPtr(false)}},
		"halo_2":        {"JGtm": {}}, // nil → actif
	}}
	got := f.ActiveTitlesForGamertag("jgtm") // insensible à la casse
	if len(got) != 2 {
		t.Fatalf("attendu 2 titres actifs (halo_infinite, halo_2), reçu %v", got)
	}
	got2 := f.ActiveTitlesForGamertag("JGtm", "halo_infinite")
	if len(got2) != 1 || got2[0] != "halo_2" {
		t.Fatalf("attendu [halo_2] après exclusion, reçu %v", got2)
	}
}

func TestEntry_MarshalOmitsEmpty(t *testing.T) {
	raw, _ := json.Marshal(Entry{DBPath: "d"})
	s := string(raw)
	if strings.Contains(s, "sync_enabled") || strings.Contains(s, "initial_max_matches") || strings.Contains(s, "xuid") {
		t.Fatalf("champs vides devraient être omis: %s", s)
	}
}
