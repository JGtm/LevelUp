package main

// job_test.go — le TRAVAIL de l'ouvrier exercé SANS décoder un vrai film : téléchargement
// des morceaux (URL pré-signées), décompression zlib du CDN, préservation des métadonnées et
// de l'ordre, et le nettoyage qui protège le cache-archive du dépôt. Le décodage réel est
// prouvé au volet B (E2E avec mini-film versionné) — ici, aucun octet de film.

import (
	"bytes"
	"compress/zlib"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// servirZlib rend un handler qui compresse en zlib ce que fabrique produce(nom-de-fichier) —
// exactement ce que fait le CDN Azure des films, et ce que l'ouvrier doit savoir défaire.
func servirZlib(produce func(nom string) []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		_, _ = zw.Write(produce(filepath.Base(r.URL.Path)))
		_ = zw.Close()
		_, _ = w.Write(buf.Bytes())
	}
}

func TestDownloadChunk_DecompresseLeZlib(t *testing.T) {
	original := []byte("REPLICATION_DATA brut du morceau de film")
	srv := httptest.NewServer(servirZlib(func(string) []byte { return original }))
	defer srv.Close()

	got, err := downloadChunk(context.Background(), srv.Client(), srv.URL+"/chunk_00.bin")
	if err != nil {
		t.Fatalf("downloadChunk: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("morceau décompressé = %q, veut %q", got, original)
	}
}

func TestDownloadChunk_HTTPErreur_URLExpiree(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "expiré", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := downloadChunk(context.Background(), srv.Client(), srv.URL+"/x")
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("err = %v, veut la mention HTTP 404 (URL pré-signée expirée)", err)
	}
}

func TestDownloadChunk_ZlibInvalide(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ceci n'est pas du zlib")) // pas d'en-tête zlib valide
	}))
	defer srv.Close()

	_, err := downloadChunk(context.Background(), srv.Client(), srv.URL+"/x")
	if err == nil || !strings.Contains(err.Error(), "zlib") {
		t.Fatalf("err = %v, veut la mention zlib", err)
	}
}

func TestFetchChunks_TelechargeDecompresseEtPreserve(t *testing.T) {
	// CDN : /<nom> -> zlib("data-<nom>"). L'ouvrier doit rendre les morceaux DANS L'ORDRE,
	// métadonnées (index, type, bornes) intactes, données décompressées.
	srv := httptest.NewServer(servirZlib(func(nom string) []byte { return []byte("data-" + nom) }))
	defer srv.Close()

	p := &domain.BuildQueuePayload{Chunks: []domain.BuildQueueChunk{
		{Index: 0, ChunkType: 2, StartMS: 0, DurationMS: 100, URL: srv.URL + "/c0"},
		{Index: 1, ChunkType: 6, StartMS: 100, DurationMS: 250, URL: srv.URL + "/c1"},
	}}
	got, err := (&worker{}).fetchChunks(context.Background(), p)
	if err != nil {
		t.Fatalf("fetchChunks: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got = %d morceaux, veut 2", len(got))
	}
	if got[0].Index != 0 || got[1].Index != 1 || got[1].ChunkType != 6 || got[1].DurationMS != 250 {
		t.Fatalf("métadonnées non préservées: %+v", got)
	}
	if string(got[0].Data) != "data-c0" || string(got[1].Data) != "data-c1" {
		t.Fatalf("données décompressées = %q / %q", got[0].Data, got[1].Data)
	}
}

// TestFetchChunks_ParalleleBorneOrdreRestitue — LE TÉLÉCHARGEMENT EST PARALLÈLE, BORNÉ, ET
// L'ORDRE DU JOB EST RENDU INTACT (item 5.1 de PLAN_CUISSON_PERF).
//
// Trente morceaux, le volume d'un film ordinaire. Le serveur compte les requêtes SIMULTANÉES :
// le maximum observé doit dépasser 1 (sans quoi rien n'est parallèle) sans jamais dépasser
// [chunkParallelism] (sans quoi la borne ne borne rien). Chaque réponse porte le nom demandé,
// donc un slot mal attribué se voit immédiatement dans la comparaison finale.
func TestFetchChunks_ParalleleBorneOrdreRestitue(t *testing.T) {
	const total = 30
	var mu sync.Mutex
	enCours, maxEnCours := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		enCours++
		if enCours > maxEnCours {
			maxEnCours = enCours
		}
		mu.Unlock()
		// Une pause courte donne aux requêtes le temps de se chevaucher : sans elle, un
		// serveur local peut servir séquentiellement et le maximum resterait à 1.
		time.Sleep(10 * time.Millisecond)
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		_, _ = zw.Write([]byte("data-" + filepath.Base(r.URL.Path)))
		_ = zw.Close()
		mu.Lock()
		enCours--
		mu.Unlock()
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	p := &domain.BuildQueuePayload{}
	for i := range total {
		p.Chunks = append(p.Chunks, domain.BuildQueueChunk{
			Index: i, ChunkType: 2, StartMS: i * 100, DurationMS: 100,
			URL: srv.URL + "/c" + strconv.Itoa(i),
		})
	}

	avant := runtime.NumGoroutine()
	got, err := (&worker{}).fetchChunks(context.Background(), p)
	if err != nil {
		t.Fatalf("fetchChunks: %v", err)
	}
	if len(got) != total {
		t.Fatalf("got = %d morceaux, veut %d", len(got), total)
	}
	for i, c := range got {
		if c.Index != i || c.StartMS != i*100 {
			t.Fatalf("morceau %d : index=%d startMS=%d — l'ordre du job n'est pas restitué",
				i, c.Index, c.StartMS)
		}
		if veut := "data-c" + strconv.Itoa(i); string(c.Data) != veut {
			t.Fatalf("morceau %d : données = %q, veut %q — slot mal attribué", i, c.Data, veut)
		}
	}
	mu.Lock()
	vu := maxEnCours
	mu.Unlock()
	// LE SERVEUR SE FERME AVANT LE COMPTAGE : ses connexions persistantes tiennent des
	// goroutines des DEUX cotes (handler et transport) tant qu'il vit. Les compter comme des
	// fuites rendrait le garde-fou inutilisable ; les fermer d'abord rend le comptage exact.
	srv.Close()
	if vu <= 1 {
		t.Errorf("simultanéité maximale observée = %d : les téléchargements restent séquentiels", vu)
	}
	if vu > chunkParallelism {
		t.Errorf("simultanéité maximale observée = %d, borne = %d : la borne ne borne rien",
			vu, chunkParallelism)
	}
	verifierAucuneFuite(t, avant)
}

// TestFetchChunks_UneErreurFaitEchouerLeJob — un morceau perdu = film à trous : le job échoue
// entier, et aucune goroutine ne survit à l'échec (le contexte du groupe coupe les autres).
func TestFetchChunks_UneErreurFaitEchouerLeJob(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/c7") {
			http.Error(w, "expiré", http.StatusNotFound)
			return
		}
		time.Sleep(5 * time.Millisecond)
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		_, _ = zw.Write([]byte("ok"))
		_ = zw.Close()
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	p := &domain.BuildQueuePayload{}
	for i := range 30 {
		p.Chunks = append(p.Chunks, domain.BuildQueueChunk{
			Index: i, URL: srv.URL + "/c" + strconv.Itoa(i),
		})
	}

	avant := runtime.NumGoroutine()
	got, err := (&worker{}).fetchChunks(context.Background(), p)
	if err == nil {
		t.Fatal("un morceau en échec doit faire échouer le job entier — un film à trous ne se cuit pas")
	}
	if !strings.Contains(err.Error(), "morceau 7") || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("err = %v, veut le numéro du morceau ET le motif HTTP", err)
	}
	if got != nil {
		t.Errorf("got = %v, veut nil : un lot partiel ne doit jamais remonter", got)
	}
	srv.Close() // cf. le commentaire du cas precedent : les connexions persistantes d'abord
	verifierAucuneFuite(t, avant)
}

// verifierAucuneFuite attend que le compte de goroutines redescende à son niveau d'avant.
//
// LA TOLÉRANCE EST DANS L'ATTENTE, PAS DANS LE SEUIL : les goroutines du serveur de test et du
// transport HTTP se terminent de façon asynchrone, mais elles se terminent. Un seuil « à N près »
// laisserait passer une vraie fuite ; une attente bornée, non.
func verifierAucuneFuite(t *testing.T, avant int) {
	t.Helper()
	for range 100 {
		if runtime.NumGoroutine() <= avant {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("goroutines : %d avant, %d après — une goroutine de téléchargement a fui",
		avant, runtime.NumGoroutine())
}

func TestFetchChunks_ContexteAnnule_NeTelechargeRien(t *testing.T) {
	p := &domain.BuildQueuePayload{Chunks: []domain.BuildQueueChunk{
		{Index: 0, URL: "http://127.0.0.1:0/jamais-appele"},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // déjà annulé : fetchChunks doit rendre l'erreur AVANT tout appel réseau
	if _, err := (&worker{}).fetchChunks(ctx, p); err == nil {
		t.Fatal("fetchChunks devrait rendre l'erreur d'un contexte annulé")
	}
}

func TestCleanupFilm_SupprimeLesMorceaux(t *testing.T) {
	work := t.TempDir()
	const short = "abcd1234"
	dir := ecrireMorceauBidon(t, work, short)

	wk := &worker{workDir: work, keepsFilms: false}
	job := &domain.BuildQueueJob{MatchID: "m", Payload: &domain.BuildQueuePayload{ShortID: short}}
	wk.cleanupFilm(context.Background(), job)

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("le dossier de morceaux existe encore (%v) — cleanup devait le supprimer", err)
	}
}

func TestCleanupFilm_ConserveLeCacheDuDepot(t *testing.T) {
	// keepsFilms=true : le dossier de travail EST le cache film du dépôt, archive
	// IRREMPLAÇABLE (les films expirent côté serveur). L'ouvrier n'y touche jamais.
	work := t.TempDir()
	const short = "abcd1234"
	dir := ecrireMorceauBidon(t, work, short)

	wk := &worker{workDir: work, keepsFilms: true}
	job := &domain.BuildQueueJob{MatchID: "m", Payload: &domain.BuildQueuePayload{ShortID: short}}
	wk.cleanupFilm(context.Background(), job)

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("le cache du dépôt (archive) a été touché: %v", err)
	}
}

// ecrireMorceauBidon crée le dossier de morceaux d'un film et y dépose un fichier, puis rend
// le chemin du dossier.
func ecrireMorceauBidon(t *testing.T, work, short string) string {
	t.Helper()
	dir := filmcache.ChunkDir(work, short)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "chunk_00.bin"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return dir
}

// TestBuildAndSend_NEcritAucunArtefactLocal — L'OUVRIER NE RANGE RIEN (PLAN_CUISSON_PERF §3 D8).
//
// Il télécharge, décode et ENVOIE des octets ; l'artefact canonique est celui que le SERVEUR
// range (`replaybuild.StoreArtifact`), avec son garde anti-régression et sa notification. Le
// cas exerce le pont disque en entier (les morceaux arrivent bien au cache film) puis constate
// qu'AUCUN fichier n'est apparu sous une arborescence d'artefacts : la construction échoue ici
// faute de catalogue de titre, et c'est exactement le point — même sur le chemin nominal, la
// seule écriture locale de l'ouvrier est celle des MORCEAUX.
func TestBuildAndSend_NEcritAucunArtefactLocal(t *testing.T) {
	srv := httptest.NewServer(servirZlib(func(nom string) []byte { return []byte("data-" + nom) }))
	defer srv.Close()

	workDir, repoRoot := t.TempDir(), t.TempDir()
	const short = "64e8adfa"
	job := &domain.BuildQueueJob{
		JobID: "j1", MatchID: "64e8adfa-0000-0000-0000-000000000000",
		Payload: &domain.BuildQueuePayload{
			MatchID: "64e8adfa-0000-0000-0000-000000000000", ShortID: short,
			TitleSlug: "halo_infinite", MapNames: []string{"Catalyst"},
			Chunks: []domain.BuildQueueChunk{{Index: 0, ChunkType: 2, URL: srv.URL + "/c0"}},
		},
	}
	w := &worker{repoRoot: repoRoot, workDir: workDir,
		client: newProtocolClient("http://127.0.0.1:1", "jeton-de-test")}
	if _, err := w.buildAndSend(context.Background(), job); err == nil {
		t.Fatal("sans catalogue de titre, la construction doit échouer — le cas ne prouverait rien")
	}

	// Le pont disque, lui, a bien eu lieu : c'est la SEULE écriture locale de l'ouvrier.
	if _, err := os.Stat(filepath.Join(filmcache.ChunkDir(workDir, short), "chunk_00.bin")); err != nil {
		t.Fatalf("les morceaux ne sont pas au cache film (%v) — le pont disque a disparu", err)
	}
	for _, racine := range []string{repoRoot, workDir} {
		_ = filepath.WalkDir(racine, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Le cache film (morceaux ET manifeste) est le pont disque legitime ; tout ce qui
			// ressemble a un artefact de rejeu, non.
			if strings.Contains(filepath.ToSlash(p), "/replays") {
				t.Errorf("artefact écrit localement : %s — l'ouvrier ne range plus rien (D8)", p)
			}
			return nil
		})
	}
}

// TestOuvrier_NeComposeJamaisLEcritureDArtefact — LE GARDE-RAIL de D8.
//
// Le cas ci-dessus constate un comportement ; celui-ci ferme la porte par laquelle il
// reviendrait. `BuildMatch` compose la construction ET l'écriture à la place canonique : le
// jour où quelqu'un le remet ici « parce que c'est plus court », l'ouvrier se remet à écrire
// dans une arborescence de dépôt qu'il n'a pas, et à relire depuis le disque des octets qu'il
// tenait déjà en mémoire.
func TestOuvrier_NeComposeJamaisLEcritureDArtefact(t *testing.T) {
	noms, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	interdits := []string{"BuildMatch(", "ArtifactPath", "os.ReadFile("}
	vuBuildBytes := false
	for _, n := range noms {
		if strings.HasSuffix(n, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(n)
		if err != nil {
			t.Fatalf("lecture de %s: %v", n, err)
		}
		for i, ligne := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(strings.TrimSpace(ligne), "//") {
				continue // un motif CITÉ en commentaire explique la règle, il ne la viole pas
			}
			if strings.Contains(ligne, "BuildBytes(") {
				vuBuildBytes = true
			}
			for _, mot := range interdits {
				if strings.Contains(ligne, mot) {
					t.Errorf("%s:%d porte %q — l'ouvrier construit des OCTETS et les envoie ; "+
						"il n'écrit ni ne relit d'artefact local (PLAN_CUISSON_PERF §3 D8)", n, i+1, mot)
				}
			}
		}
	}
	if !vuBuildBytes {
		t.Error("aucun appel à BuildBytes dans l'ouvrier : le garde-rail garde un motif mort")
	}
}
