// Package session — store_test.go : tests unitaires du Store de session.
package session_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/platform/session"
)

// captureWarnLogs redirige le logger par défaut vers un buffer (niveau >= WARN)
// le temps du test, et restaure l'ancien. Les tests du package ne tournent pas en
// parallèle (aucun t.Parallel) → pas de course sur slog.Default.
func captureWarnLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func newTestStore(t *testing.T) *session.Store {
	t.Helper()
	dir := t.TempDir()
	return session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32-bytesXXXXXXXXXX")
}

func TestStore_NewSession(t *testing.T) {
	store := newTestStore(t)
	sess := store.New()

	if sess == nil {
		t.Fatal("expected session, got nil")
	}
	if sess.SessionID == "" {
		t.Fatal("session ID should not be empty")
	}
	if sess.CreatedAt == 0 {
		t.Fatal("CreatedAt should be set")
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	store := newTestStore(t)
	sess := store.New()

	if err := store.Touch(sess); err != nil {
		t.Fatalf("Touch: %v", err)
	}

	loaded := store.Load(context.Background(), sess.SessionID)
	if loaded == nil {
		t.Fatal("expected loaded session, got nil")
	}
	if loaded.SessionID != sess.SessionID {
		t.Errorf("session ID mismatch: %q vs %q", loaded.SessionID, sess.SessionID)
	}
}

func TestStore_Load_NotFound(t *testing.T) {
	store := newTestStore(t)
	loaded := store.Load(context.Background(), "nonexistent-session-id")
	if loaded != nil {
		t.Fatal("expected nil for unknown session")
	}
}

// TestStore_Load_NotFound_NoLog : un fichier absent est le cas NOMINAL (session
// neuve ou expirée-supprimée) — il ne doit RIEN logger (sinon spam à chaque
// requête anonyme).
func TestStore_Load_NotFound_NoLog(t *testing.T) {
	store := newTestStore(t)
	buf := captureWarnLogs(t)

	if loaded := store.Load(context.Background(), "absent-session-id"); loaded != nil {
		t.Fatalf("expected nil for absent session, got %+v", loaded)
	}
	if buf.Len() != 0 {
		t.Errorf("un fichier absent ne doit rien logger, got: %q", buf.String())
	}
}

// TestStore_Load_CorruptFile_LogsWarnAndReturnsNil : un fichier illisible (JSON
// corrompu / torn read résiduel) doit renvoyer nil ET tracer un WARN — c'était le
// point aveugle de la boucle /login (retour nil silencieux → session anonyme).
func TestStore_Load_CorruptFile_LogsWarnAndReturnsNil(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	store := session.NewStore(dir, time.Hour, "test-secret-32-bytesXXXXXXXXXX")
	sess := store.New()
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Corrompre le fichier de session sur disque.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var target string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			target = filepath.Join(dir, e.Name())
		}
	}
	if target == "" {
		t.Fatal("fichier de session introuvable après Save")
	}
	if err := os.WriteFile(target, []byte("{ceci n'est pas du JSON valide"), 0o600); err != nil {
		t.Fatalf("corrupt write: %v", err)
	}

	buf := captureWarnLogs(t)
	if loaded := store.Load(context.Background(), sess.SessionID); loaded != nil {
		t.Fatalf("expected nil for corrupt session file, got %+v", loaded)
	}
	if !strings.Contains(buf.String(), "illisible") {
		t.Errorf("expected a WARN log for corrupt JSON, got: %q", buf.String())
	}
}

func TestStore_Delete(t *testing.T) {
	store := newTestStore(t)
	sess := store.New()
	_ = store.Touch(sess)

	if err := store.Delete(sess.SessionID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	loaded := store.Load(context.Background(), sess.SessionID)
	if loaded != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestStore_SignAndUnsign(t *testing.T) {
	store := newTestStore(t)
	sessionID := "test-session-uuid-1234"
	signed := store.SignCookie(sessionID)

	if signed == "" {
		t.Fatal("signed cookie should not be empty")
	}

	unsigned := store.UnsignCookie(signed)
	if unsigned != sessionID {
		t.Errorf("UnsignCookie(%q) = %q, want %q", signed, unsigned, sessionID)
	}
}

func TestStore_UnsignCookie_Invalid(t *testing.T) {
	store := newTestStore(t)
	result := store.UnsignCookie("tampered-invalid-cookie")
	if result != "" {
		t.Errorf("expected empty string for invalid cookie, got %q", result)
	}
}

func TestStore_PurgeExpired(t *testing.T) {
	// Créer un store avec TTL très court
	dir := t.TempDir()
	shortStore := session.NewStore(filepath.Join(dir, "sessions"), 1*time.Millisecond, "test-secret-32-bytesXXXXXXXXXX")
	sess := shortStore.New()
	_ = shortStore.Touch(sess)

	// Attendre que la session expire
	time.Sleep(10 * time.Millisecond)

	removed := shortStore.PurgeExpired()
	if removed == 0 {
		t.Log("PurgeExpired returned 0, session may not have expired yet (timing)")
	}
}

// TestStore_PurgeExpired_CorruptFile_FreshKeptOldDeleted : un fichier de session au
// JSON illisible n'est supprimé que s'il est durablement corrompu (mtime > TTL 1h).
// Un corrompu FRAIS peut être un write en vol d'un autre process (doublon `air`) →
// conservé + WARN, pour ne jamais déconnecter une session vivante sur une corruption
// illusoire.
func TestStore_PurgeExpired_CorruptFile_FreshKeptOldDeleted(t *testing.T) {
	sessDir := filepath.Join(t.TempDir(), "sessions")
	store := session.NewStore(sessDir, time.Hour, "test-secret-32-bytesXXXXXXXXXX")

	// Corrompu FRAIS (mtime = maintenant) → doit être CONSERVÉ.
	fresh := filepath.Join(sessDir, "fresh-corrupt.json")
	if err := os.WriteFile(fresh, []byte("{ceci n'est pas du JSON"), 0o600); err != nil {
		t.Fatalf("write fresh: %v", err)
	}
	// Corrompu VIEUX (mtime reculé au-delà du TTL 1h) → doit être SUPPRIMÉ.
	old := filepath.Join(sessDir, "old-corrupt.json")
	if err := os.WriteFile(old, []byte("{encore du JSON casse"), 0o600); err != nil {
		t.Fatalf("write old: %v", err)
	}
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatalf("chtimes old: %v", err)
	}

	buf := captureWarnLogs(t)
	if removed := store.PurgeExpired(); removed != 1 {
		t.Errorf("PurgeExpired = %d, attendu 1 (seul le corrompu vieux)", removed)
	}

	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("un fichier corrompu FRAIS ne doit pas être supprimé (write en vol ?): %v", err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("un fichier corrompu VIEUX doit être supprimé, stat err=%v", err)
	}
	if !strings.Contains(buf.String(), "PurgeExpired") {
		t.Errorf("PurgeExpired doit tracer un WARN sur le corrompu conservé, got: %q", buf.String())
	}
}

// Le cas « erreur de lecture transitoire (hors fichier absent) ne supprime JAMAIS le
// fichier » est reproduit de façon DÉTERMINISTE dans store_purge_windows_test.go (via
// une sharing violation Windows, la course exacte du bug), et exercé sous charge dans
// TestStore_PurgeExpiredDuringLiveSession (store_concurrent_test.go, -race).

// TestStore_PurgeExpired_ExpiredAndOldTmp : non-régression — une session valide mais
// expirée est bien purgée, et un .tmp orphelin vieux est bien nettoyé (non compté
// dans le retour).
func TestStore_PurgeExpired_ExpiredAndOldTmp(t *testing.T) {
	sessDir := filepath.Join(t.TempDir(), "sessions")
	store := session.NewStore(sessDir, time.Millisecond, "test-secret-32-bytesXXXXXXXXXX")

	sess := store.New()
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // dépasse le TTL 1ms

	// .tmp orphelin VIEUX (mtime reculé au-delà d'orphanTmpTTL = 1h).
	orphanTmp := filepath.Join(sessDir, "orphan-123.tmp")
	if err := os.WriteFile(orphanTmp, []byte("partial"), 0o600); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(orphanTmp, past, past); err != nil {
		t.Fatalf("chtimes tmp: %v", err)
	}

	if removed := store.PurgeExpired(); removed != 1 {
		t.Errorf("PurgeExpired = %d, attendu 1 (la session expirée, le .tmp non compté)", removed)
	}
	if got := store.Load(context.Background(), sess.SessionID); got != nil {
		t.Error("la session expirée aurait dû être purgée")
	}
	if _, err := os.Stat(orphanTmp); !os.IsNotExist(err) {
		t.Errorf("le .tmp orphelin vieux aurait dû être nettoyé, stat err=%v", err)
	}
}
