package infinite

// gamecmsfichiers_test.go — les fichiers GameCMS : les sept formes d'entrée de l'URL d'image, l'URL
// d'un fichier, un GET sans en-tête sans jetons, avec les deux jetons quand ils sont donnés, la garde
// du Doer consultée avant tout, et l'erreur typée d'un 404.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"levelup/go-api/internal/haloapi"
)

func TestURLImageGameCMS_SeptFormes(t *testing.T) {
	const h = "https://cms.test"
	e := DefaultEndpoints().Avec(CleGameCMS, h).AvecPrefixeJeu("hi")
	for entree, attendu := range map[string]string{
		"  ":                               "",
		"hi/Waypoint/file/Defi.JSON":       "",
		" HTTPS://cdn.test/A.png ":         "HTTPS://cdn.test/A.png",
		"/HI/Progression/file/a.png":       h + "/HI/Progression/file/a.png",
		"hi/images/file/a.png":             h + "/hi/images/file/a.png",
		"hi/waypoint/file/images/c.png":    h + "/hi/waypoint/file/images/c.png",
		"Waypoint/file/images/d.png":       h + "/hi/Waypoint/file/images/d.png",
		"progression/file/RewardTrack.png": h + "/hi/progression/file/RewardTrack.png",
		"images/file/b.png":                h + "/hi/images/file/b.png",
		"//career_rank/rank1.png":          h + "/hi/images/file/career_rank/rank1.png",
	} {
		if got := URLImageGameCMS(e, entree); got != attendu {
			t.Errorf("URLImageGameCMS(%q) = %q, attendu %q", entree, got, attendu)
		}
	}
	if got, want := URLImageGameCMS(e.AvecPrefixeJeu("h5"), "images/file/a.png"), h+"/h5/images/file/a.png"; got != want {
		t.Errorf("segment h5 : %q, attendu %q", got, want)
	}
}

func TestURLFichierGameCMS(t *testing.T) {
	if got, want := URLFichierGameCMS(DefaultEndpoints(), "Progression/file/Metadata/Metadata.json"),
		"https://gamecms-hacs.svc.halowaypoint.com/hi/Progression/file/Metadata/Metadata.json"; got != want {
		t.Errorf("défaut : %q, attendu %q", got, want)
	}
	e := DefaultEndpoints().Avec(CleGameCMS, "https://cms.test/").AvecPrefixeJeu("h5")
	if got, want := URLFichierGameCMS(e, "waypoint/file/images/x.png"), "https://cms.test/h5/waypoint/file/images/x.png"; got != want {
		t.Errorf("surchargé : %q, attendu %q", got, want)
	}
}

func TestTelechargerFichierGameCMS_Entetes(t *testing.T) {
	const url = "https://cms.test/hi/waypoint/file/images/x.png"
	for nom, c := range map[string]struct {
		auth    Auth
		attendu string
	}{
		"sans jetons": {Auth{}, "GET " + url + " []"},
		"avec jetons": {Auth{Spartan: "S", Clearance: "C"}, "GET " + url + " [343-Clearance=C; X-343-Authorization-Spartan=S]"},
	} {
		t.Run(nom, func(t *testing.T) {
			client, e := clientEnregistreur(http.StatusOK, "PNG")
			octets, err := client.TelechargerFichierGameCMS(context.Background(), c.auth, url)
			if err != nil || string(octets) != "PNG" {
				t.Fatalf("octets %q, erreur %v", octets, err)
			}
			if got := e.Requetes(); len(got) != 1 || got[0] != c.attendu {
				t.Errorf("requêtes = %q\nattendu    %q", got, c.attendu)
			}
		})
	}
}

// TestTelechargerFichierGameCMS_GardeDAbord : une garde qui refuse arrête l'appel avant toute
// émission, même sur une URL qui ne se construit pas, et son erreur est rendue telle quelle.
func TestTelechargerFichierGameCMS_GardeDAbord(t *testing.T) {
	cause := errors.New("sortie refusée")
	d := &doerGarde{refus: haloapi.NonRetryable(cause)}
	_, err := New(d, RetryAucune()).TelechargerFichierGameCMS(context.Background(), Auth{Spartan: "S"}, "https://cms.test/%zz")
	if !errors.Is(err, haloapi.ErrDefinitif) || !errors.Is(err, cause) || d.n.Load() != 0 || d.verifs.Load() != 1 {
		t.Errorf("erreur %v, émissions=%d gardes=%d, attendu la cause définitive, 0 émission, 1 garde", err, d.n.Load(), d.verifs.Load())
	}
}

func TestTelechargerFichierGameCMS_404EstUneErreurTypee(t *testing.T) {
	client, _ := clientEnregistreur(http.StatusNotFound, "absent")
	_, err := client.TelechargerFichierGameCMS(context.Background(), Auth{}, "https://cms.test/hi/x.png")
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusNotFound || string(he.Body) != "absent" {
		t.Errorf("erreur %v, attendu *HTTPError 404 au corps « absent »", err)
	}
}
