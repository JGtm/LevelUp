package haloclient

// halo_client_isfilmgone_test.go — teste IsFilmGoneErr : manifeste ET blobs, tous les
// deux typés (*HTTPError pour le manifeste via fetchFilmManifest/doGet, *BlobHTTPError
// pour un chunk via downloadBlob), 404 ET 410, plus les erreurs qui ne sont PAS un film
// définitivement perdu (transitoires, ou sans rapport). Idée reprise du fork
// ChaseWoodhams (`halo_client_isfilmgone_test.go`, commit 1b16dc20c) — bilan 2026-09-11
// point 4b, réimplémentée contre notre propre isNotFoundErr typé (pas de repli textuel
// depuis le 2026-09-05, cf. no_text_predicate_test.go).
import (
	"errors"
	"fmt"
	"testing"
)

func TestIsFilmGoneErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"manifest_404", &HTTPError{StatusCode: 404, URL: "u"}, true},
		{"manifest_410", &HTTPError{StatusCode: 410, URL: "u"}, true},
		{"manifest_503_transitoire", &HTTPError{StatusCode: 503, URL: "u"}, false},
		{"manifest_429_transitoire", &HTTPError{StatusCode: 429, URL: "u"}, false},
		{"blob_404", &BlobHTTPError{StatusCode: 404, URL: "u", Attempts: 1}, true},
		{"blob_410", &BlobHTTPError{StatusCode: 410, URL: "u", Attempts: 1}, true},
		{"blob_503_transitoire", &BlobHTTPError{StatusCode: 503, URL: "u", Attempts: 3}, false},
		{
			// Expiration PARTIELLE : le manifeste répond, un chunk isolé rend 404. Le
			// biais assumé (bilan 4b) : compter cette erreur comme définitive plutôt
			// que transitoire — un faux « transitoire » se corrige à la passe
			// suivante, un faux « expiré » serait définitif à tort, donc le biais va
			// du bon côté.
			name: "blob_404_enveloppee_par_fetchFilmChunks",
			err:  fmt.Errorf("GetFilmChunks chunk 3(abc): %w", &BlobHTTPError{StatusCode: 404, URL: "u", Attempts: 1}),
			want: true,
		},
		{"transport_error_sans_statut", errors.New("context deadline exceeded"), false},
		{"unrelated", errors.New("Catalog Error: Table with name foo does not exist"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsFilmGoneErr(tc.err); got != tc.want {
				t.Fatalf("IsFilmGoneErr(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
