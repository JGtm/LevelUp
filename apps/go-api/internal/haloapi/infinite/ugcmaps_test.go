package infinite

// ugcmaps_test.go — le service UGC des cartes : URL (avec et sans version), en-têtes (identité
// fournie par l'appelant, jetons), refus sans jeton Spartan, erreur typée d'un non-200, blob sans
// authentification.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/haloapi"
	"levelup/go-api/internal/haloapi/haloapitest"
)

var identiteTest = Identite{UserAgent: "App/1.0", Origin: "https://origine.test"}

const carteJSON = `{"AssetId":"a","VersionId":"v","PublicName":"Carte","Files":{"Prefix":"https://blob.test/m/","FileRelativePaths":["x.mvar"]},"CustomData":{"NumOfObjectsOnMap":3,"TagLevelId":7,"IsBaked":true}}`

func clientEnregistreur(statut int, corps string) (*Client, *haloapitest.Enregistreur) {
	e := &haloapitest.Enregistreur{Repondre: func(int, *http.Request) *http.Response {
		return haloapitest.Reponse(statut, corps)
	}}
	return New(&http.Client{Transport: e}, RetryAucune()), e
}

func TestFetchCarteUGC_URLEtEntetes(t *testing.T) {
	const hote = "https://discovery-infiniteugc.svc.halowaypoint.com/hi/Maps/"
	const entetes = " [343-Clearance=C; Accept=application/json; Origin=https://origine.test; User-Agent=App/1.0; X-343-Authorization-Spartan=S]"
	cas := map[string]struct{ version, attendu string }{
		"sans version": {"", "GET " + hote + "a%2Fb" + entetes},
		"avec version": {"v 1", "GET " + hote + "a%2Fb/versions/v%201" + entetes},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			client, e := clientEnregistreur(http.StatusOK, carteJSON)
			carte, err := client.FetchCarteUGC(context.Background(), Auth{Spartan: "S", Clearance: "C"}, "a/b", c.version, identiteTest)
			if err != nil || carte.PublicName != "Carte" || carte.CustomData.TagLevelID != 7 || !carte.CustomData.IsBaked ||
				carte.Files.Prefix != "https://blob.test/m/" || len(carte.Files.FileRelativePaths) != 1 {
				t.Fatalf("carte %+v, erreur %v", carte, err)
			}
			if got := e.Requetes(); len(got) != 1 || got[0] != c.attendu {
				t.Errorf("requêtes = %q\nattendu    %q", got, c.attendu)
			}
		})
	}
}

func TestFetchCarteUGC_SansJetonSpartan(t *testing.T) {
	client, e := clientEnregistreur(http.StatusOK, carteJSON)
	_, err := client.FetchCarteUGC(context.Background(), Auth{Clearance: "C"}, "a", "", identiteTest)
	const attendu = "aucun jeton Spartan disponible pour https://discovery-infiniteugc.svc.halowaypoint.com/hi/Maps/a"
	if err == nil || err.Error() != attendu || len(e.Requetes()) != 0 {
		t.Errorf("erreur %v, %d requête(s), attendu %q et aucune requête", err, len(e.Requetes()), attendu)
	}
}

func TestUGC_Non200EstUneErreurTypeeAuCorpsTronque(t *testing.T) {
	long := strings.Repeat("x", 600)
	client, _ := clientEnregistreur(http.StatusNotFound, long)
	for nom, appel := range map[string]func() error{
		"discovery": func() error {
			_, err := client.FetchCarteUGC(context.Background(), Auth{Spartan: "S"}, "a", "", identiteTest)
			return err
		},
		"blob": func() error {
			_, err := client.TelechargerFichierUGC(context.Background(), "https://blob.test/m", "x.mvar", identiteTest)
			return err
		},
	} {
		var he *HTTPError
		err := appel()
		if !errors.As(err, &he) || he.StatusCode != http.StatusNotFound || len(he.Body) != 512 || !IsNotFound(err) {
			t.Errorf("%s : erreur %v, attendu *HTTPError 404 au corps de 512 octets", nom, err)
		}
	}
}

// TestTelechargerFichierUGC_GardeDAbord : une garde qui refuse arrête l'appel avant toute émission,
// même sur une URL qui ne se construit pas, et son erreur est rendue telle quelle.
func TestTelechargerFichierUGC_GardeDAbord(t *testing.T) {
	cause := errors.New("sortie refusée")
	d := &doerGarde{refus: haloapi.NonRetryable(cause)}
	_, err := New(d, RetryAucune()).TelechargerFichierUGC(context.Background(), "https://h/%zz", "x.mvar", identiteTest)
	if !errors.Is(err, haloapi.ErrDefinitif) || !errors.Is(err, cause) || d.n.Load() != 0 || d.verifs.Load() != 1 {
		t.Errorf("erreur %v, émissions=%d gardes=%d, attendu la cause définitive, 0 émission, 1 garde", err, d.n.Load(), d.verifs.Load())
	}
}

func TestTelechargerFichierUGC_SansAuthNiOrigine(t *testing.T) {
	client, e := clientEnregistreur(http.StatusOK, "OCTETS")
	octets, err := client.TelechargerFichierUGC(context.Background(), "https://blob.test/m/", "dir/a b.mvar", identiteTest)
	const attendu = "GET https://blob.test/m/dir/a%20b.mvar [Accept=application/json; User-Agent=App/1.0]"
	if err != nil || string(octets) != "OCTETS" {
		t.Fatalf("octets %q, erreur %v", octets, err)
	}
	if got := e.Requetes(); len(got) != 1 || got[0] != attendu {
		t.Errorf("requêtes = %q\nattendu    %q", got, attendu)
	}
}

// TestFetchCarteUGC_GardeDAbord : une garde qui refuse arrête l'appel AVANT la vérification du jeton
// Spartan : l'erreur est celle de la garde (définitive), pas « aucun jeton Spartan disponible », sans
// émission.
func TestFetchCarteUGC_GardeDAbord(t *testing.T) {
	cause := errors.New("sortie refusée")
	d := &doerGarde{refus: haloapi.NonRetryable(cause)}
	_, err := New(d, RetryAucune()).FetchCarteUGC(context.Background(), Auth{}, "a", "", identiteTest)
	if !errors.Is(err, haloapi.ErrDefinitif) || !errors.Is(err, cause) || d.n.Load() != 0 || d.verifs.Load() != 1 {
		t.Errorf("erreur %v, émissions=%d gardes=%d, attendu la cause définitive, 0 émission, 1 garde", err, d.n.Load(), d.verifs.Load())
	}
	if err != nil && strings.Contains(err.Error(), "jeton Spartan") {
		t.Errorf("erreur %q : le refus de la garde doit précéder le contrôle du jeton", err)
	}
}

// repeteur : un lecteur sans fin d'octets 'x' (le corps n'est pas alloué d'avance).
type repeteur struct{}

func (repeteur) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

// TestTelechargerFichierUGC_LectureBornee : un fichier de variante de taille réaliste (200 Ko) est
// lu en entier ; un corps plus long que [lectureMaxUGC] est lu jusqu'à cette borne exactement, sans
// erreur (le surplus est ignoré).
func TestTelechargerFichierUGC_LectureBornee(t *testing.T) {
	for nom, c := range map[string]struct{ servi, attendu int64 }{
		"200 Ko revient entier":  {200_000, 200_000},
		"au-delà de la borne":    {lectureMaxUGC + 100, lectureMaxUGC},
		"juste à la borne":       {lectureMaxUGC, lectureMaxUGC},
		"un octet sous la borne": {lectureMaxUGC - 1, lectureMaxUGC - 1},
	} {
		t.Run(nom, func(t *testing.T) {
			e := &haloapitest.Enregistreur{Repondre: func(int, *http.Request) *http.Response {
				rep := haloapitest.Reponse(http.StatusOK, "")
				rep.Body = io.NopCloser(io.LimitReader(repeteur{}, c.servi))
				return rep
			}}
			client := New(&http.Client{Transport: e}, RetryAucune())
			octets, err := client.TelechargerFichierUGC(context.Background(), "https://blob.test/m", "x.mvar", identiteTest)
			if err != nil || int64(len(octets)) != c.attendu {
				t.Errorf("%d octet(s) lus (erreur %v), attendu %d sans erreur", len(octets), err, c.attendu)
			}
		})
	}
}
