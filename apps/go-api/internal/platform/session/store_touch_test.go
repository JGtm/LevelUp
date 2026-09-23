// Package session — store_touch_test.go : Touch n'écrit une session inchangée qu'au plus
// toutes les 5 min (plan perf du 2026-09-23, D3.5). Horloge injectée (WithClock) : le
// last_seen_at relu sur disque dit si une écriture a eu lieu.
package session_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
)

// touchClock : horloge de test avancée à la main.
type touchClock struct{ now time.Time }

func (c *touchClock) Now() time.Time { return c.now }

func newClockStore(t *testing.T) (*session.Store, *touchClock, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "sessions")
	clock := &touchClock{now: time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)}
	store := session.NewStore(dir, session.DefaultTTL, "test-secret-32-bytesXXXXXXXXXX", session.WithClock(clock.Now))
	return store, clock, dir
}

// onDisk relit le fichier de la session tel qu'il est persisté.
func onDisk(t *testing.T, dir string, sess *domain.SessionData) domain.SessionData {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, sess.SessionID+".json"))
	if err != nil {
		t.Fatalf("lecture du fichier de session : %v", err)
	}
	var got domain.SessionData
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("fichier de session illisible : %v", err)
	}
	return got
}

func mustTouch(t *testing.T, store *session.Store, sess *domain.SessionData) {
	t.Helper()
	if err := store.Touch(sess); err != nil {
		t.Fatalf("Touch : %v", err)
	}
}

func TestTouch_InchangeeDansLIntervalle_AucuneEcriture(t *testing.T) {
	store, clock, dir := newClockStore(t)
	sess := store.New()
	mustTouch(t, store, sess) // jamais écrite par ce process : écrite
	written := clock.now.Unix()

	for _, d := range []time.Duration{time.Second, 4*time.Minute + 58*time.Second} {
		clock.now = clock.now.Add(d)
		mustTouch(t, store, sess)
		if got := onDisk(t, dir, sess).LastSeenAt; got != written {
			t.Fatalf("après +%v : last_seen_at sur disque = %d, attendu %d (session inchangée réécrite)", d, got, written)
		}
	}
}

func TestTouch_ApresLIntervalle_Reecrit(t *testing.T) {
	store, clock, dir := newClockStore(t)
	sess := store.New()
	mustTouch(t, store, sess)

	clock.now = clock.now.Add(5*time.Minute + time.Second)
	mustTouch(t, store, sess)
	if got := onDisk(t, dir, sess).LastSeenAt; got != clock.now.Unix() {
		t.Errorf("last_seen_at sur disque = %d, attendu %d : le TTL glissant n'est plus rafraîchi", got, clock.now.Unix())
	}
}

func TestTouch_SessionModifiee_EcriteAussitot(t *testing.T) {
	store, clock, dir := newClockStore(t)
	sess := store.New()
	mustTouch(t, store, sess)

	clock.now = clock.now.Add(time.Second)
	sess.Locale = "en"
	mustTouch(t, store, sess)
	got := onDisk(t, dir, sess)
	if got.Locale != "en" || got.LastSeenAt != clock.now.Unix() {
		t.Errorf("session modifiée non écrite : locale %q, last_seen_at %d", got.Locale, got.LastSeenAt)
	}
}

// TestTouch_ApresSaveDUnLastSeenAncien_Reecrit : un handler qui Save une session chargée
// réécrit son last_seen_at ANCIEN ; le Touch de fin de requête doit quand même rafraîchir
// la présence sur disque (sinon une session active pourrait expirer).
func TestTouch_ApresSaveDUnLastSeenAncien_Reecrit(t *testing.T) {
	store, clock, dir := newClockStore(t)
	sess := store.New()
	sess.LastSeenAt = clock.now.Add(-6 * 24 * time.Hour).Unix()
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save : %v", err)
	}

	clock.now = clock.now.Add(time.Second)
	mustTouch(t, store, sess)
	if got := onDisk(t, dir, sess).LastSeenAt; got != clock.now.Unix() {
		t.Errorf("last_seen_at sur disque = %d, attendu %d", got, clock.now.Unix())
	}
}

// TestTouch_NouveauProcess_EcritLaPremiereFois : la marque d'écriture vit en mémoire ; un
// autre process (redémarrage) écrit à sa première Touch.
func TestTouch_NouveauProcess_EcritLaPremiereFois(t *testing.T) {
	store, clock, dir := newClockStore(t)
	sess := store.New()
	mustTouch(t, store, sess)

	restarted := session.NewStore(dir, session.DefaultTTL, "test-secret-32-bytesXXXXXXXXXX", session.WithClock(clock.Now))
	clock.now = clock.now.Add(time.Second)
	mustTouch(t, restarted, sess)
	if got := onDisk(t, dir, sess).LastSeenAt; got != clock.now.Unix() {
		t.Errorf("last_seen_at sur disque = %d, attendu %d", got, clock.now.Unix())
	}
}

func TestTouch_HorlogeEnArriere_Ecrit(t *testing.T) {
	store, clock, dir := newClockStore(t)
	sess := store.New()
	mustTouch(t, store, sess)

	clock.now = clock.now.Add(-time.Minute)
	mustTouch(t, store, sess)
	if got := onDisk(t, dir, sess).LastSeenAt; got != clock.now.Unix() {
		t.Errorf("last_seen_at sur disque = %d, attendu %d", got, clock.now.Unix())
	}
}
