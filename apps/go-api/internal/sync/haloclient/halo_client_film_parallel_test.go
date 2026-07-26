// Package sync — halo_client_film_parallel_test.go : tests TDD pour la
// parallélisation du download des chunks REPLICATION_DATA dans GetMatchFilm
// (plan stabilisation 2026-05-22 §3.1 — opportunité ratée initialement,
// reprise sur demande utilisateur le 2026-05-23).
//
// Contexte : un film Halo contient 10-30 chunks (~200-500ms latence CDN
// chacun). La boucle for séquentielle dans halo_client.go::GetMatchFilm
// téléchargeait les chunks un par un → 3-15s wall-time par film. Avec
// errgroup parallèle (limit configurable), gain attendu 3-5× (1 vague au
// lieu de N vagues séquentielles).
//
// **Mode TDD strict** (directive utilisateur explicite, opération critique
// sur le path sync) : les tests définissent le contrat AVANT toute modif
// du code de production. Critères couverts :
//
//  1. ParallelFasterThanSequential : test perf qui ÉCHOUE sur le code
//     séquentiel actuel (10 chunks × 100ms = 1s) et PASSE post-impl
//     (~200ms avec parallelism=8).
//  2. PreservesAllChunks : map[int]FilmChunkData identique au séquentiel
//     (clés == chunk.Index, data == blob CDN décompressé).
//  3. CompletesAllBeforeReturn : GetMatchFilm ne retourne QUE quand tous
//     les chunks sont téléchargés (le caller collectWeaponKillsForMatch
//     traite ensuite — pas de race avec le download).
//  4. OneChunkFails_ReturnsError : errgroup propage la 1ère erreur, les
//     autres goroutines abortent via ctx cancel.
//  5. CancelMidDownload : ctx.Cancel à mi-parcours → ctx.Err retournée
//     proprement, pas de fuite goroutine.
//  6. NoRace : 30 chunks parallèles, -race clean (preuve que le write au
//     map résultat est sync-safe).

package haloclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// filmManyChunks renvoie un manifest JSON avec n chunks REPLICATION_DATA
// indexés 1..n (l'index 0 = header conventionnellement skip dans la prod).
func filmManyChunks(prefix string, n int) map[string]any {
	chunks := make([]map[string]any, 0, n+1)
	chunks = append(chunks, filmChunkEntry(0, filmChunkTypeHeader, "header.bin"))
	for i := 1; i <= n; i++ {
		chunks = append(chunks, filmChunkEntry(i, filmChunkTypeReplicationData,
			fmt.Sprintf("chunk%d.bin", i)))
	}
	return filmManifestJSON(prefix, chunks)
}

// newLatencyFilmServer démarre un httptest server qui :
//   - répond au manifest avec `filmManyChunks(prefix, n)`
//   - répond aux blobs avec un payload zlib-compressé "chunk-{path}" après
//     `latency` de sleep (simule RTT CDN)
//   - track le nombre d'appels par URL (atomic counter) pour détecter les
//     doublons ou les appels manquants.
func newLatencyFilmServer(t *testing.T, n int, latency time.Duration) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var blobCalls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			// Manifest. Le prefix doit pointer vers le SAME server pour que
			// downloadBlob soit redirigé via redirectTransport.
			_ = json.NewEncoder(w).Encode(filmManyChunks("http://blobs.test/", n))
			return
		}
		// Blob chunk. Sleep latency, retourne zlib(<chunk-name>).
		if latency > 0 {
			select {
			case <-time.After(latency):
			case <-r.Context().Done():
				w.WriteHeader(499)
				return
			}
		}
		blobCalls.Add(1)
		// Payload distinct par chunk pour détecter les swaps : le chemin URL
		// contient `chunkN.bin`, on garde N comme tag.
		payload := zlibCompress(t, []byte("data-"+r.URL.Path))
		_, _ = w.Write(payload)
	}))
	return srv, &blobCalls
}

// TestGetMatchFilm_ParallelDownloadFasterThanSequential : 10 chunks × 100ms
// latence simulée. Séquentiel théorique : 1000ms. Parallèle (limit=8 attendu) :
// 2 vagues × 100ms ≈ 200-300ms + overhead.
//
// **Test perf qui ÉCHOUE sur le code séquentiel** (~1s) et passe post-impl
// (<500ms attendu, seuil 500ms pour absorber variance CI Windows).
func TestGetMatchFilm_ParallelDownloadFasterThanSequential(t *testing.T) {
	const nChunks = 10
	const latency = 100 * time.Millisecond
	srv, _ := newLatencyFilmServer(t, nChunks, latency)
	defer srv.Close()

	c := newFilmTestClient(srv)

	start := time.Now()
	chunks, found, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("GetMatchFilm: %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if len(chunks) != nChunks {
		t.Errorf("len(chunks) = %d, want %d", len(chunks), nChunks)
	}

	// Séquentiel : 10 × 100ms = 1000ms.
	// Parallèle (limit=8) : 2 vagues × 100ms + overhead = ~250ms.
	// Seuil 500ms (2× speedup mini) pour absorber variance Windows CI.
	const maxAcceptable = 500 * time.Millisecond
	if elapsed > maxAcceptable {
		t.Errorf("wall-time %v > %v (parallélisation absente ou défaillante — "+
			"séquentiel théorique = 1000ms pour 10×100ms, parallèle attendu < 500ms)",
			elapsed, maxAcceptable)
	} else {
		t.Logf("wall-time = %v (gain parallélisation confirmé)", elapsed)
	}
}

// TestGetMatchFilm_PreservesAllChunks : 10 chunks → map a 10 entries
// indexées 1..10, chaque data correspond bien au chunk demandé (no swap
// inter-goroutines).
func TestGetMatchFilm_PreservesAllChunks(t *testing.T) {
	const nChunks = 10
	srv, _ := newLatencyFilmServer(t, nChunks, 5*time.Millisecond)
	defer srv.Close()

	c := newFilmTestClient(srv)
	chunks, found, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatalf("GetMatchFilm: %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if len(chunks) != nChunks {
		t.Fatalf("len(chunks) = %d, want %d", len(chunks), nChunks)
	}

	// Vérifier que chaque chunk a la data du bon chunk (no swap entre
	// goroutines : si chunk 5 reçoit la data du chunk 3, c'est un bug
	// classique de fermeture sur loop variable).
	for i := 1; i <= nChunks; i++ {
		c, ok := chunks[i]
		if !ok {
			t.Errorf("chunk %d absent du résultat", i)
			continue
		}
		// L'URL contient "chunk{i}.bin", la data du test est "data-/<path>".
		wantPart := fmt.Sprintf("chunk%d.bin", i)
		if !strings.Contains(string(c.Data), wantPart) {
			t.Errorf("chunk %d : data ne contient pas %q\ndata = %q",
				i, wantPart, c.Data)
		}
		// Métadonnées préservées (StartMS = index × 500, DurationMS = 500).
		wantStart := i * 500
		if c.StartMS != wantStart {
			t.Errorf("chunk %d : StartMS = %d, want %d", i, c.StartMS, wantStart)
		}
		if c.DurationMS != 500 {
			t.Errorf("chunk %d : DurationMS = %d, want 500", i, c.DurationMS)
		}
	}
}

// TestGetMatchFilm_CompletesAllBeforeReturn : garantit la propriété critique
// "GetMatchFilm ne retourne qu'une fois tous les chunks DL". Compte les
// requêtes HTTP servies — doit ÉGALER le nombre de chunks à la fin du
// GetMatchFilm. Test du contrat de synchronisation (errgroup.Wait()).
func TestGetMatchFilm_CompletesAllBeforeReturn(t *testing.T) {
	const nChunks = 8
	srv, blobCalls := newLatencyFilmServer(t, nChunks, 20*time.Millisecond)
	defer srv.Close()

	c := newFilmTestClient(srv)
	chunks, _, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatalf("GetMatchFilm: %v", err)
	}
	// Au moment du retour, TOUS les blobs doivent avoir été servis.
	// Si la fonction retournait avant la fin (race), blobCalls < nChunks.
	got := int(blobCalls.Load())
	if got != nChunks {
		t.Errorf("blobCalls = %d à la fin, want %d (retour précoce ? race ?)",
			got, nChunks)
	}
	if len(chunks) != nChunks {
		t.Errorf("len(chunks) = %d, want %d", len(chunks), nChunks)
	}
}

// TestGetMatchFilm_OneChunkFails_ReturnsError : 5 chunks où le chunk 3 retourne
// 500 → errgroup retourne l'erreur, les autres goroutines doivent aborter
// proprement (via ctx cancel propagé par errgroup.WithContext).
func TestGetMatchFilm_OneChunkFails_ReturnsError(t *testing.T) {
	const nChunks = 5
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManyChunks("http://blobs.test/", nChunks))
			return
		}
		// Chunk 3 retourne 500, les autres OK.
		if strings.Contains(r.URL.Path, "chunk3.bin") {
			w.WriteHeader(500)
			return
		}
		// Petit sleep pour que les goroutines se lancent ensemble avant que
		// chunk3 ne fail.
		time.Sleep(20 * time.Millisecond)
		_, _ = w.Write(zlibCompress(t, []byte("ok-"+r.URL.Path)))
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	_, _, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err == nil {
		t.Fatal("attendu erreur (chunk 3 retourne 500), got nil")
	}
	if !strings.Contains(err.Error(), "chunk 3") &&
		!strings.Contains(err.Error(), "GetMatchFilm chunk") {
		// Au moins l'erreur doit mentionner le chunk fautif pour traçabilité.
		t.Errorf("err = %v, attendu mention 'chunk 3' ou 'GetMatchFilm chunk'", err)
	}
}

// TestGetMatchFilm_CancelMidDownload : 20 chunks × 100ms, ctx.Cancel après
// 75ms → la fonction doit retourner une erreur ctx.Err() (ou wrapping),
// sans deadlock ni fuite goroutine.
func TestGetMatchFilm_CancelMidDownload(t *testing.T) {
	const nChunks = 20
	const latency = 100 * time.Millisecond
	srv, _ := newLatencyFilmServer(t, nChunks, latency)
	defer srv.Close()

	c := newFilmTestClient(srv)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(75 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, _, err := c.GetMatchFilm(ctx, testFilmMatchUUID)
	elapsed := time.Since(start)

	if err == nil {
		t.Errorf("attendu erreur ctx.Err après cancel, got nil")
	}
	// Sanité : le cancel doit court-circuiter, pas attendre tous les sleeps.
	// Séquentiel théorique 2s, parallèle théorique 250ms — avec cancel à 75ms
	// on doit revenir bien avant 1s.
	if elapsed > 1*time.Second {
		t.Errorf("wall-time après cancel = %v, attendu < 1s (cancel pas propagé ?)",
			elapsed)
	}
}

// TestGetMatchFilm_NoRace : 30 chunks parallèles, le test doit passer avec
// `go test -race`. Garantit que les écritures dans le map result sont
// sync-safe (pas de race condition sur la collection des résultats).
//
// Sans synchronisation correcte, `go test -race` détecterait une data race
// sur l'accès concurrent au `map[int]FilmChunkData`.
func TestGetMatchFilm_NoRace(t *testing.T) {
	const nChunks = 30
	srv, _ := newLatencyFilmServer(t, nChunks, 2*time.Millisecond)
	defer srv.Close()

	c := newFilmTestClient(srv)
	chunks, found, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatalf("GetMatchFilm: %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if len(chunks) != nChunks {
		t.Errorf("len(chunks) = %d, want %d", len(chunks), nChunks)
	}
}
