package assets

// demo_offline_no_persist_test.go — en démo, une image GameCMS absente du cache ne part
// pas sur le réseau et n'écrit rien dans le cache (revue adversariale du lot B5, constat
// C4 ; lot B-C3 du backlog 2026-09-26).
//
// Le cache d'assets est enraciné sur le dépôt, y compris en démo (lecture seule autorisée,
// tableau B5.0 ligne 27). Seule source d'un payload BINAIRE (donc d'un PersistBinary) :
// GameCMSFetcher, dont l'émission passe par doGet → netguard.Check. En démo, le fetch rend
// ErrOffline AVANT toute requête, et fetchAndPersist s'arrête avant l'écriture. Ce test
// verrouille les deux faits au niveau du résolveur ; le ratchet
// platform/netguard/netguard_coverage_test.go verrouille la présence du garde dans le fichier.

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/platform/netguard"
)

// resolveurImageEspion : un GameCMS local qui compte ses requêtes et un cache vide.
func resolveurImageEspion(t *testing.T) (*DefaultResolver, *atomic.Int32, string) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("\x89PNG\r\n\x1a\nimage"))
	}))
	t.Cleanup(srv.Close)
	cache := t.TempDir()
	f := NewGameCMSFetcher(srv.Client(), nil, srv.URL)
	return NewDefaultResolver(NewLocalFSStore(cache), nil, f, nil, nil), &hits, cache
}

func fichiersDuCache(t *testing.T, cache string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(cache, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk cache : %v", err)
	}
	return files
}

var refEmbleme = Ref{Kind: KindSpartanEmblem, TitleID: "halo_infinite", ID: "Progression/Inventory/emblems/demo.png"}

func TestImageGameCMS_Demo_NiReseauNiEcriture(t *testing.T) {
	r, hits, cache := resolveurImageEspion(t)
	netguard.SetOffline(true)
	t.Cleanup(func() { netguard.SetOffline(false) })

	_, err := r.Get(context.Background(), refEmbleme)
	if !errors.Is(err, ErrUpstreamUnavailable) {
		t.Errorf("démo : erreur %v, attendu ErrUpstreamUnavailable (refus netguard, 502 propre côté API)", err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("démo : %d requête(s) sortie(s) vers GameCMS, attendu 0", n)
	}
	if files := fichiersDuCache(t, cache); len(files) != 0 {
		t.Errorf("démo : le cache d'assets a été écrit : %v", files)
	}
}

func TestImageGameCMS_HorsDemo_TelechargeEtPersiste(t *testing.T) {
	r, hits, cache := resolveurImageEspion(t)
	netguard.SetOffline(false)

	if _, err := r.Get(context.Background(), refEmbleme); err != nil {
		t.Fatalf("hors démo : Get : %v", err)
	}
	if hits.Load() != 1 {
		t.Errorf("hors démo : %d requête(s), attendu 1 (comportement inchangé)", hits.Load())
	}
	if files := fichiersDuCache(t, cache); len(files) != 1 {
		t.Errorf("hors démo : fichiers du cache %v, attendu l'image persistée", files)
	}
}
