package infinite

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/haloapi"
)

// TestGetFilmManifest : chemin, DTO, références triées ; absence typée.
func TestGetFilmManifest(t *testing.T) {
	corps := `{"BlobStoragePathPrefix":"https://blobs/f/","CustomData":{"FilmMajorVersion":41,"Chunks":[
{"Index":2,"ChunkType":2,"ChunkSize":10,"ChunkStartTimeOffsetMilliseconds":5000,"DurationMilliseconds":5000,"FileRelativePath":"/filmChunk2"},
{"Index":0,"ChunkType":1,"ChunkSize":7,"FileRelativePath":"filmChunk0"},
{"Index":3,"ChunkType":3,"ChunkSize":9,"FileRelativePath":"filmChunk3"}]}}`
	srv, cap := serveurCorps(t, []byte(corps))
	m, err := clientSur(srv).GetFilmManifest(context.Background(), authTest, matchTest)
	if err != nil || m.CustomData.FilmMajorVersion != 41 || len(m.CustomData.Chunks) != 3 || m.CustomData.Chunks[0].ChunkSize != 10 {
		t.Fatalf("manifeste = %+v, %v", m, err)
	}
	if u, _ := cap.derniere(t); u.Path != "/hi/films/matches/"+matchTest+"/spectate" {
		t.Errorf("URL = %s", u)
	}
	refs := m.Refs()
	if len(refs) != 3 || refs[0].Index != 0 || refs[2].URL != "https://blobs/f/filmChunk3" || refs[1].StartMS != 5000 {
		t.Errorf("refs = %+v", refs)
	}
	sansPrefixe := &FilmManifestDTO{}
	sansPrefixe.CustomData.Chunks = []FilmChunkMetaDTO{{Index: 1}}
	if len(sansPrefixe.Refs()) != 0 {
		t.Error("fragment sans adresse : omis")
	}
	absent, _ := serveurScripte(t, "", http.StatusNotFound)
	if _, err := clientSur(absent).GetFilmManifest(context.Background(), authTest, matchTest); !IsFilmGoneErr(err) {
		t.Errorf("404 : IsFilmGoneErr faux (%v)", err)
	}
	if BuildChunkURL("p", "/x") != "p/x" || BuildChunkURL("p/", "x") != "p/x" || BuildChunkURL("p", "") != "p" {
		t.Error("BuildChunkURL")
	}
}

// compteursCrochets compte les crochets appelés.
type compteursCrochets struct{ reprise, epuise, incomplet, journal atomic.Int32 }

func (k *compteursCrochets) crochets() Crochets {
	return Crochets{
		BlobRepriseReussie: func() { k.reprise.Add(1) },
		BlobEpuise:         func() { k.epuise.Add(1) },
		BlobIncomplet:      func() { k.incomplet.Add(1) },
		JournalBlobReseau:  func(context.Context, string, int, error) { k.journal.Add(1) },
	}
}

// TestDownloadBlob_304PuisSucces : un 304 est retenté avec revalidation, le crochet de reprise est
// appelé, les octets rendus sont ceux du CDN, sans aucun jeton envoyé.
func TestDownloadBlob_304PuisSucces(t *testing.T) {
	var n atomic.Int32
	var revalide, jeton atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(enteteSpartan) != "" || r.Header.Get(enteteClearance) != "" {
			jeton.Store(true)
		}
		if n.Add(1) == 1 {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		revalide.Store(r.Header.Get("Cache-Control") == "no-cache")
		_, _ = w.Write([]byte("brut-compresse"))
	}))
	defer srv.Close()
	k := &compteursCrochets{}
	c := New(srv.Client(), sansDelai(RetryHaloClient())).WithCrochets(k.crochets())
	b, err := c.DownloadBlob(context.Background(), srv.URL+"/c", len("brut-compresse"))
	if err != nil || string(b) != "brut-compresse" || n.Load() != 2 || !revalide.Load() || jeton.Load() || k.reprise.Load() != 1 {
		t.Fatalf("b=%q err=%v n=%d revalide=%v jeton=%v reprise=%d", b, err, n.Load(), revalide.Load(), jeton.Load(), k.reprise.Load())
	}
}

// TestDownloadBlob_Verdicts : 404 définitif (absent, pas auth), 401 jamais auth, 503 épuisé,
// taille refusée sans nouvelle tentative.
func TestDownloadBlob_Verdicts(t *testing.T) {
	k := &compteursCrochets{}
	for _, c := range []struct {
		statut, requetes, attempts int
		absent                     bool
	}{{404, 1, 1, true}, {401, 1, 1, false}, {503, 4, 4, false}} {
		srv, n := serveurScripte(t, "", c.statut)
		_, err := New(srv.Client(), sansDelai(RetryHaloClient())).WithCrochets(k.crochets()).DownloadBlob(context.Background(), srv.URL, 0)
		var be *BlobHTTPError
		if !errors.As(err, &be) || be.Attempts != c.attempts || int(n.Load()) != c.requetes {
			t.Errorf("statut %d : err=%v requêtes=%d", c.statut, err, n.Load())
		}
		if IsFilmGoneErr(err) != c.absent || IsAuthError(err) {
			t.Errorf("statut %d : IsFilmGoneErr=%v IsAuthError=%v", c.statut, IsFilmGoneErr(err), IsAuthError(err))
		}
	}
	if k.epuise.Load() != 1 {
		t.Errorf("BlobEpuise = %d, veut 1 (le 503)", k.epuise.Load())
	}
	srv, n := serveurScripte(t, "", 200)
	_, err := New(srv.Client(), sansDelai(RetryHaloClient())).WithCrochets(k.crochets()).DownloadBlob(context.Background(), srv.URL, 999)
	var inc *ErrChunkIncomplet
	if !errors.As(err, &inc) || inc.Annonce != 999 || n.Load() != 1 || k.incomplet.Load() != 1 || IsFilmGoneErr(err) {
		t.Errorf("taille : err=%v requêtes=%d incomplet=%d", err, n.Load(), k.incomplet.Load())
	}
}

// TestDownloadBlob_ReseauEtDefinitif : échec réseau épuisé (crochet de journal), erreur définitive
// au premier essai.
func TestDownloadBlob_ReseauEtDefinitif(t *testing.T) {
	k := &compteursCrochets{}
	d := &doerCompteur{err: errors.New("réseau")}
	_, err := New(d, sansDelai(RetryHaloClient())).WithCrochets(k.crochets()).DownloadBlob(context.Background(), "http://x.invalid/b", 0)
	if d.n.Load() != 4 || k.journal.Load() != 1 || k.epuise.Load() != 1 || err == nil {
		t.Errorf("réseau : appels=%d journal=%d err=%v", d.n.Load(), k.journal.Load(), err)
	}
	d2 := &doerCompteur{err: haloapi.NonRetryable(errors.New("hors ligne"))}
	_, err = New(d2, sansDelai(RetryHaloClient())).DownloadBlob(context.Background(), "http://x.invalid/b", 0)
	if d2.n.Load() != 1 || !errors.Is(err, haloapi.ErrDefinitif) {
		t.Errorf("définitif : appels=%d err=%v", d2.n.Load(), err)
	}
}

// TestDownloadChunks : ordre de la liste conservé, erreur préfixée par l'appelant et le fragment.
func TestDownloadChunks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/mort") {
			w.WriteHeader(http.StatusGone)
			return
		}
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	defer srv.Close()
	c := New(srv.Client(), sansDelai(RetryHaloClient()))
	var chunks []FilmChunkMetaDTO
	for i, nom := range []string{"c5", "c1", "c3", "c0", "c9", "c2", "c7", "c4", "c8", "c6"} {
		chunks = append(chunks, FilmChunkMetaDTO{Index: 10 - i, ChunkType: 2, FileRelativePath: nom, ChunkStartTimeOffsetMilliseconds: i})
	}
	out, err := c.DownloadChunks(context.Background(), srv.URL+"/f", chunks, "GetFilmChunks", matchTest)
	if err != nil || len(out) != len(chunks) {
		t.Fatalf("out=%d err=%v", len(out), err)
	}
	for i, fr := range out {
		if string(fr.Brut) != "/f/"+chunks[i].FileRelativePath || fr.Index != chunks[i].Index || fr.StartMS != i {
			t.Errorf("fragment %d = %+v", i, fr)
		}
	}
	chunks[4].FileRelativePath = "mort"
	_, err = c.DownloadChunks(context.Background(), srv.URL+"/f", chunks, "GetMatchFilm", matchTest)
	if !IsFilmGoneErr(err) || !strings.HasPrefix(err.Error(), "GetMatchFilm chunk 6("+matchTest+")") {
		t.Errorf("err = %v", err)
	}
}
