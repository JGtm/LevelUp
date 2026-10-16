package service

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// bytesOpener ouvre un fichier téléversé dont le contenu est s.
func bytesOpener(s string) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(s)), nil }
}

// failingReader livre n octets puis échoue : un téléversement coupé en cours de route.
type failingReader struct {
	n   int
	err error
}

func (r *failingReader) Read(p []byte) (int, error) {
	if r.n <= 0 {
		return 0, r.err
	}
	k := min(len(p), r.n)
	for i := range k {
		p[i] = 'x'
	}
	r.n -= k
	return k, nil
}

// Le contenu est écrit en flux sur disque : un gros fichier arrive intact sans être
// chargé en mémoire par le service (le lecteur n'est lu qu'à travers un tampon).
func TestMediaService_UploadMedia_StreamsLargeFileToDisk(t *testing.T) {
	dir := t.TempDir()
	const size = 20 << 20
	svc := NewMediaService(&mockMediaRepo{}, "")
	res, err := svc.UploadMedia(context.Background(), domain.UploadRequest{
		Files: []domain.UploadedFile{{
			OriginalName: "big.mp4",
			Open: func() (io.ReadCloser, error) {
				return io.NopCloser(io.LimitReader(zeroReader{}, size)), nil
			},
		}},
		CapturesDir: dir,
		DBPath:      filepath.Join(dir, "stats.duckdb"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Saved != 1 {
		t.Fatalf("Saved = %d, attendu 1 (errors: %v)", res.Saved, res.Errors)
	}
	st, err := os.Stat(filepath.Join(dir, "big.mp4"))
	if err != nil || st.Size() != size {
		t.Fatalf("fichier écrit: %v taille=%v, attendu %d", err, st, size)
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

// Un téléversement coupé ne laisse ni fichier final ni reste temporaire dans le dossier
// des captures, et l'erreur est rapportée.
func TestMediaService_UploadMedia_InterruptedUploadLeavesNothing(t *testing.T) {
	dir := t.TempDir()
	svc := NewMediaService(&mockMediaRepo{}, "")
	res, err := svc.UploadMedia(context.Background(), domain.UploadRequest{
		Files: []domain.UploadedFile{{
			OriginalName: "cut.mp4",
			Open: func() (io.ReadCloser, error) {
				return io.NopCloser(&failingReader{n: 1 << 20, err: errors.New("connexion coupée")}), nil
			},
		}},
		CapturesDir: dir,
		DBPath:      filepath.Join(dir, "stats.duckdb"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Saved != 0 || len(res.Errors) == 0 {
		t.Fatalf("Saved=%d Errors=%v, attendu 0 sauvé et une erreur", res.Saved, res.Errors)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mp4") || strings.HasSuffix(e.Name(), ".part") {
			t.Errorf("reste d'un téléversement coupé: %s", e.Name())
		}
	}
}

// Un fichier qui n'a pu être ouvert est rapporté sans interrompre les autres.
func TestMediaService_UploadMedia_OpenFailureIsReportedAndOthersProceed(t *testing.T) {
	dir := t.TempDir()
	svc := NewMediaService(&mockMediaRepo{}, "")
	res, err := svc.UploadMedia(context.Background(), domain.UploadRequest{
		Files: []domain.UploadedFile{
			{OriginalName: "bad.mp4", Open: func() (io.ReadCloser, error) { return nil, errors.New("illisible") }},
			{OriginalName: "ok.mp4", Open: bytesOpener("contenu")},
		},
		CapturesDir: dir,
		DBPath:      filepath.Join(dir, "stats.duckdb"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Saved != 1 || len(res.Errors) == 0 {
		t.Fatalf("Saved=%d Errors=%v, attendu 1 sauvé et une erreur", res.Saved, res.Errors)
	}
	if _, err := os.Stat(filepath.Join(dir, "ok.mp4")); err != nil {
		t.Fatalf("ok.mp4 absent: %v", err)
	}
}
