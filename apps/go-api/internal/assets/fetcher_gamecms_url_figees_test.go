package assets

// Témoin T2 « URL figées » du fetcher GameCMS (plan PUBLICATION, lot 1.2) : pour chacune des sept
// fonctions `fetch*`, l'URL absolue et les en-têtes tels que le client les construit, avec et sans
// fournisseur de jetons, en littéral. Le fetcher est coupé en protocole / exploitation au lot 2.3
// sans changement d'octet sur le fil.
//
// Le RoundTripper enregistre `req.Header` tel que le client l'a posé : le `User-Agent` par défaut de
// Go est ajouté par le transport, pas par le client — il n'apparaît donc pas ici, et c'est le fait
// figé (aucun `Accept` ni `User-Agent` explicite).

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/haloapi/haloapitest"
	"levelup/go-api/internal/haloapi/infinite"
)

const (
	cmsHote       = "https://gamecms-hacs.svc.halowaypoint.com"
	cmsEntetesJWT = " [343-Clearance=clearance-test; X-343-Authorization-Spartan=spartan-test]"
	cmsSansEntete = " []"
)

// cmsEnregistreur : un enregistreur qui rend `statut` et le corps « {} » à chaque requête.
func cmsEnregistreur(statut int) *haloapitest.Enregistreur {
	return &haloapitest.Enregistreur{Repondre: func(int, *http.Request) *http.Response {
		return haloapitest.Reponse(statut, "{}")
	}}
}

// cmsSansResolveur retire le résolveur d'adresses partagé le temps du test : l'hôte et le préfixe de
// jeu sont alors ceux par défaut (Halo Infinite), quel que soit l'ordre des tests du paquet.
func cmsSansResolveur(t *testing.T) {
	t.Helper()
	precedent := games.DefaultEndpointResolver()
	games.SetDefaultEndpointResolver(nil)
	t.Cleanup(func() { games.SetDefaultEndpointResolver(precedent) })
}

// TestURLFigees_GameCMS_Fetch : les sept fonctions `fetch*` via `Fetch`, sans fournisseur de jetons
// (`f.tokens == nil` : la requête part sans en-tête) puis avec (jetons posés en
// `x-343-authorization-spartan` et `343-clearance`). Médaille et métadonnées de médailles n'envoient
// jamais de jeton, même avec un fournisseur ; l'image générique en envoie s'il y en a un.
func TestURLFigees_GameCMS_Fetch(t *testing.T) {
	cmsSansResolveur(t)
	p := cmsHote + "/hi/"
	cas := []struct {
		nom                string
		ref                Ref
		sansJetons, jetons string
	}{
		{"fetchMedalImage", Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "42"},
			"GET " + p + "Progression/file/medals/halo_infinite/42.png" + cmsSansEntete,
			"GET " + p + "Progression/file/medals/halo_infinite/42.png" + cmsSansEntete},
		{"fetchChallengeBadge", Ref{Kind: KindChallengeBadge, TitleID: "halo_infinite", ID: "combat/EnemiesKilledMelee"},
			"GET " + p + "waypoint/file/images/combat/EnemiesKilledMelee.png" + cmsSansEntete,
			"GET " + p + "waypoint/file/images/combat/EnemiesKilledMelee.png" + cmsEntetesJWT},
		{"fetchGameCMSImage", Ref{Kind: KindBPTrackImage, TitleID: "halo_infinite", ID: "progression/file/RewardTracks/Season1.png"},
			"GET " + p + "progression/file/RewardTracks/Season1.png" + cmsSansEntete,
			"GET " + p + "progression/file/RewardTracks/Season1.png" + cmsEntetesJWT},
		{"fetchMedalMetadata", Ref{Kind: KindMedalMetadata, TitleID: "halo_infinite", ID: "metadata"},
			"GET " + p + "Progression/file/Metadata/Metadata.json" + cmsSansEntete,
			"GET " + p + "Progression/file/Metadata/Metadata.json" + cmsSansEntete},
		{"fetchChallengeDefinition", Ref{Kind: KindChallengeDefinition, TitleID: "halo_infinite", ID: "/Challenges/Weekly/Win3.json"},
			"GET " + p + "Progression/file/Challenges/Weekly/Win3.json" + cmsSansEntete,
			"GET " + p + "Progression/file/Challenges/Weekly/Win3.json" + cmsEntetesJWT},
		{"fetchRewardTrackDefinition", Ref{Kind: KindRewardTrackDefinition, TitleID: "halo_infinite", ID: "RewardTracks/Season1.json"},
			"GET " + p + "Progression/file/RewardTracks/Season1.json" + cmsSansEntete,
			"GET " + p + "Progression/file/RewardTracks/Season1.json" + cmsEntetesJWT},
		{"fetchBPItemDefinition", Ref{Kind: KindBPItemDefinition, TitleID: "halo_infinite", ID: "//Inventory/Armor/Coatings/Gold.json"},
			"GET " + p + "Progression/file/Inventory/Armor/Coatings/Gold.json" + cmsSansEntete,
			"GET " + p + "Progression/file/Inventory/Armor/Coatings/Gold.json" + cmsEntetesJWT},
	}
	for _, c := range cas {
		for _, variante := range []struct {
			nom     string
			tokens  TokenProvider
			attendu string
		}{{"sans fournisseur", nil, c.sansJetons}, {"avec fournisseur", stubTokens, c.jetons}} {
			t.Run(c.nom+"/"+variante.nom, func(t *testing.T) {
				e := cmsEnregistreur(http.StatusOK)
				f := NewGameCMSFetcher(&http.Client{Transport: e}, variante.tokens, "")
				if _, err := f.Fetch(context.Background(), c.ref); err != nil {
					t.Fatal(err)
				}
				if len(e.Requetes()) != 1 || e.Requetes()[0] != variante.attendu {
					t.Errorf("requêtes = %q\nattendu    %q", e.Requetes(), variante.attendu)
				}
			})
		}
	}
}

// TestURLFigees_GameCMS_FournisseurEnEchec : l'erreur du fournisseur de jetons est fatale (aucune
// requête) pour le badge et les trois définitions, ignorée sans journal pour l'image générique, qui
// part alors sans en-tête ; médaille et métadonnées n'appellent pas le fournisseur.
func TestURLFigees_GameCMS_FournisseurEnEchec(t *testing.T) {
	cmsSansResolveur(t)
	for _, k := range []Kind{KindChallengeBadge, KindChallengeDefinition, KindRewardTrackDefinition, KindBPItemDefinition} {
		e := cmsEnregistreur(http.StatusOK)
		_, err := NewGameCMSFetcher(&http.Client{Transport: e}, failingTokens, "").Fetch(context.Background(), Ref{Kind: k, TitleID: "halo_infinite", ID: "x"})
		if !errors.Is(err, ErrUpstreamUnavailable) || len(e.Requetes()) != 0 {
			t.Errorf("%s : erreur %v, %d requête(s), attendu ErrUpstreamUnavailable sans requête", k, err, len(e.Requetes()))
		}
	}
	e := cmsEnregistreur(http.StatusOK)
	f := NewGameCMSFetcher(&http.Client{Transport: e}, failingTokens, "")
	if _, err := f.Fetch(context.Background(), Ref{Kind: KindCareerRankImage, TitleID: "halo_infinite", ID: "career_rank/rank1.png"}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []Kind{KindMedalImage, KindMedalMetadata} {
		if _, err := f.Fetch(context.Background(), Ref{Kind: k, TitleID: "halo_infinite", ID: "1"}); err != nil {
			t.Fatalf("%s : %v", k, err)
		}
	}
	attendu := []string{
		"GET " + cmsHote + "/hi/images/file/career_rank/rank1.png" + cmsSansEntete,
		"GET " + cmsHote + "/hi/Progression/file/medals/halo_infinite/1.png" + cmsSansEntete,
		"GET " + cmsHote + "/hi/Progression/file/Metadata/Metadata.json" + cmsSansEntete,
	}
	if strings.Join(e.Requetes(), "\n") != strings.Join(attendu, "\n") {
		t.Errorf("requêtes =\n%s\nattendu\n%s", strings.Join(e.Requetes(), "\n"), strings.Join(attendu, "\n"))
	}
}

// TestURLFigees_GameCMS_BaseSurchargee : l'hôte passé à la construction remplace celui par défaut et
// perd ses `/` finaux.
func TestURLFigees_GameCMS_BaseSurchargee(t *testing.T) {
	cmsSansResolveur(t)
	e := cmsEnregistreur(http.StatusOK)
	f := NewGameCMSFetcher(&http.Client{Transport: e}, nil, "https://cms.exemple.test//")
	if _, err := f.Fetch(context.Background(), Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "7"}); err != nil {
		t.Fatal(err)
	}
	attendu := "GET https://cms.exemple.test/hi/Progression/file/medals/halo_infinite/7.png" + cmsSansEntete
	if len(e.Requetes()) != 1 || e.Requetes()[0] != attendu {
		t.Errorf("requêtes = %q\nattendu    %q", e.Requetes(), attendu)
	}
}

// cmsURLImage : la construction de l'URL d'image par la brique, sur l'hôte `h` et le segment de jeu
// `p` (l'ancienne `buildGameCMSImageFetchURL(h, p, entree)` du fetcher).
func cmsURLImage(h, p, entree string) string {
	return infinite.URLImageGameCMS(infinite.DefaultEndpoints().Avec(infinite.CleGameCMS, h).AvecPrefixeJeu(p), entree)
}

// TestURLFigees_GameCMS_ConstructionDeLImage : la fonction pure `infinite.URLImageGameCMS`, une
// ligne par forme d'entrée. Vide et `.json` -> aucune URL ; URL absolue -> telle quelle ; chemin déjà
// préfixé du jeu -> sous l'hôte seul ; chemin `images|progression|waypoint/file/` -> préfixe du jeu
// ajouté ; tout autre -> `images/file/` ajouté. Les comparaisons sont insensibles à la casse, le
// chemin garde la sienne.
func TestURLFigees_GameCMS_ConstructionDeLImage(t *testing.T) {
	const h = "https://cms.exemple.test"
	for entree, attendu := range map[string]string{
		"":                                       "",
		"   ":                                    "",
		"hi/Waypoint/file/Defi.JSON":             "",
		"https://cdn.exemple.test/a.png":         "https://cdn.exemple.test/a.png",
		"  HTTP://cdn.exemple.test/B.png ":       "HTTP://cdn.exemple.test/B.png",
		"hi/images/file/a/b.png":                 h + "/hi/images/file/a/b.png",
		"/HI/Progression/file/a/b.png":           h + "/HI/Progression/file/a/b.png",
		"hi/waypoint/file/images/c.png":          h + "/hi/waypoint/file/images/c.png",
		"images/file/a/b.png":                    h + "/hi/images/file/a/b.png",
		"Progression/file/medals/12.png":         h + "/hi/Progression/file/medals/12.png",
		"/Waypoint/file/images/d.png":            h + "/hi/Waypoint/file/images/d.png",
		"career_rank/rank1.png":                  h + "/hi/images/file/career_rank/rank1.png",
		"//Progression/x.png":                    h + "/hi/images/file/Progression/x.png",
		" Inventory/Armor/Emblems/Un Nom.png":    h + "/hi/images/file/Inventory/Armor/Emblems/Un Nom.png",
		"h5/images/file/sans-effet-pour-hi.png":  h + "/hi/images/file/h5/images/file/sans-effet-pour-hi.png",
		"hi/images/file/a.png?v=2#fragment-brut": h + "/hi/images/file/a.png?v=2#fragment-brut",
	} {
		if got := cmsURLImage(h, "hi", entree); got != attendu {
			t.Errorf("URLImageGameCMS(%q) = %q, attendu %q", entree, got, attendu)
		}
	}
	if got, want := cmsURLImage(h, "h5", "h5/images/file/a.png"), h+"/h5/images/file/a.png"; got != want {
		t.Errorf("préfixe de jeu h5 : %q, attendu %q", got, want)
	}
	if got, want := cmsURLImage(h, "h5", "images/file/a.png"), h+"/h5/images/file/a.png"; got != want {
		t.Errorf("préfixe de jeu h5 ajouté : %q, attendu %q", got, want)
	}
}

// primaireAbsent : une source primaire qui ne trouve jamais l'actif.
type primaireAbsent struct{}

func (primaireAbsent) Supports(Kind) bool { return true }
func (primaireAbsent) Fetch(context.Context, Ref) (Payload, error) {
	return nil, ErrNotFound
}

// TestURLFigees_GameCMS_SpritesheetDeRepli : sans réseau, l'adresse de la feuille de sprites que
// `fetcher_chain.go` construit avec les mêmes fonctions d'hôte et de préfixe.
func TestURLFigees_GameCMS_SpritesheetDeRepli(t *testing.T) {
	cmsSansResolveur(t)
	for base, attendu := range map[string]string{
		"":                           cmsHote + "/hi/Progression/file/medals/sprites/halo_infinite.png",
		"https://cms.exemple.test/x": "https://cms.exemple.test/x/hi/Progression/file/medals/sprites/halo_infinite.png",
	} {
		f := NewSpritesheetFallbackFetcher(primaireAbsent{}, base)
		p, err := f.Fetch(context.Background(), Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "42"})
		u, ok := p.(URLPayload)
		if err != nil || !ok || u.URL != attendu || u.ContentType != MimeImagePNG {
			t.Errorf("base %q : %+v, erreur %v, attendu l'URL %q", base, p, err, attendu)
		}
	}
}
