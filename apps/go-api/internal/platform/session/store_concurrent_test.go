// Package session — store_concurrent_test.go : garde-rail anti-« torn read ».
package session_test

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/platform/session"
)

// TestStore_ConcurrentLoadDuringSave garantit qu'un Load concurrent d'une session
// VIVANTE ne renvoie jamais nil (ni une session amputée de son Username) pendant
// qu'une rafale de Touch/Save réécrit le même fichier.
//
// C'était la cause racine de la boucle /login : os.WriteFile (truncate puis write,
// non atomique) exposait un fichier vide/tronqué aux Load concurrents déclenchés
// par la rafale refetchOnWindowFocus → Load nil → session anonyme transitoire →
// éjection vers /login. ROUGE avec os.WriteFile, VERT avec l'écriture atomique
// (fichier temporaire du même répertoire + os.Rename).
func TestStore_ConcurrentLoadDuringSave(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32-bytesXXXXXXXXXX")

	sess := store.New()
	username := "alice"
	sess.Username = &username // session authentifiée → meaningful → persistée
	if err := store.Save(sess); err != nil {
		t.Fatalf("initial Save: %v", err)
	}
	id := sess.SessionID

	var (
		wg       sync.WaitGroup
		stop     = make(chan struct{})
		loadNil  atomic.Int64
		loadBad  atomic.Int64
		saveErrs atomic.Int64
	)

	// Writers : rechargent la session puis la re-touchent, sur une COPIE fraîche
	// par itération (évite une data-race Go sur un pointeur *SessionData partagé).
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				cur := store.Load(context.Background(), id)
				if cur == nil {
					continue // l'ancienne impl pouvait déjà renvoyer nil ici
				}
				if err := store.Touch(cur); err != nil {
					saveErrs.Add(1)
				}
			}
		}()
	}

	// Readers : Load en boucle. Une session vivante DOIT toujours revenir non nil
	// avec son Username intact.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				got := store.Load(context.Background(), id)
				if got == nil {
					loadNil.Add(1)
					continue
				}
				if got.Username == nil || *got.Username != username {
					loadBad.Add(1)
				}
			}
		}()
	}

	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()

	if n := loadNil.Load(); n > 0 {
		t.Errorf("Load a renvoyé nil %d fois pour une session vivante (torn read)", n)
	}
	if n := loadBad.Load(); n > 0 {
		t.Errorf("Load a renvoyé une session sans Username %d fois (torn read partiel)", n)
	}
	if n := saveErrs.Load(); n > 0 {
		t.Errorf("Touch/Save a échoué %d fois", n)
	}
}

// TestStore_PurgeExpiredDuringLiveSession garantit qu'une session VIVANTE (Touch/Save
// en rafale, TTL loin devant) n'est JAMAIS supprimée par des PurgeExpired concurrents,
// et reste toujours chargeable. Le bug fermé ici était double :
//   - PurgeExpired ne prenait pas s.mu : son os.ReadFile courait avec le os.Rename de
//     Save → sharing violation sous Windows.
//   - Sur cette erreur de lecture, il supprimait le fichier inconditionnellement → une
//     session vivante déconnectée en permanence.
//
// ROUGE avec l'ancienne impl (sous Windows), VERT avec le verrou par fichier + le
// skip-on-read-error. À exécuter sous -race.
func TestStore_PurgeExpiredDuringLiveSession(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32-bytesXXXXXXXXXX")

	sess := store.New()
	username := "alice"
	sess.Username = &username // session authentifiée → meaningful → persistée
	if err := store.Save(sess); err != nil {
		t.Fatalf("initial Save: %v", err)
	}
	id := sess.SessionID

	var (
		wg      sync.WaitGroup
		stop    = make(chan struct{})
		loadNil atomic.Int64
	)

	// Writer : recharge puis re-touche la session vivante, sur une COPIE fraîche par
	// itération (évite une data-race Go sur un *SessionData partagé).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if cur := store.Load(context.Background(), id); cur != nil {
				_ = store.Touch(cur)
			}
		}
	}()

	// Purger en boucle serrée pendant que la session vit.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			store.PurgeExpired()
		}
	}()

	// Reader : une session vivante DOIT toujours revenir non nil.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if store.Load(context.Background(), id) == nil {
				loadNil.Add(1)
			}
		}
	}()

	time.Sleep(300 * time.Millisecond)
	close(stop)
	wg.Wait()

	if n := loadNil.Load(); n > 0 {
		t.Errorf("Load a renvoyé nil %d fois : PurgeExpired a supprimé une session vivante", n)
	}
	// Assertion finale : la session a survécu au barrage de purges.
	if got := store.Load(context.Background(), id); got == nil {
		t.Fatal("la session vivante a été supprimée par PurgeExpired (régression)")
	}
}
