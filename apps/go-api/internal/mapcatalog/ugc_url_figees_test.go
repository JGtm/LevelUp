package mapcatalog

// Témoin T2 « URL figées » du client Discovery UGC (plan PUBLICATION, lot 1.2) : méthode, URL absolue
// et en-têtes triés de chaque requête, pour des identifiants sains, en littéral. Le client n'avait
// aucun test avant ce fichier. Il est coupé en protocole / exploitation au lot 2.2 sans changement
// d'octet sur le fil.

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/haloapi/haloapitest"
	"levelup/go-api/internal/haloapi/infinite"
)

const (
	ugcFigeesAsset   = "5c6f1e8a-3b2d-4f7a-9d10-8a4e2c7b1f03"
	ugcFigeesVersion = "9e0b7d42-61c8-4a35-b8f2-0d3a5c9e7f14"
	ugcFigeesBlob    = "https://blobs-infiniteugc.svc.halowaypoint.com/ugcstorage/map/" + ugcFigeesAsset + "/" + ugcFigeesVersion + "/"
	ugcFigeesAgent   = "LevelUp/1.0 (dashboard stats Halo, usage personnel)"

	ugcEntetesAvecAuth = " [343-Clearance=CLR; Accept=application/json; Origin=https://www.halowaypoint.com; User-Agent=" + ugcFigeesAgent + "; X-343-Authorization-Spartan=v4=SPARTAN]"
	ugcEntetesSansAuth = " [Accept=application/json; User-Agent=" + ugcFigeesAgent + "]"
)

const ugcFigeesCorps = `{"AssetId":"` + ugcFigeesAsset + `","VersionId":"` + ugcFigeesVersion + `","PublicName":"Carte de test",` +
	`"Files":{"Prefix":"` + ugcFigeesBlob + `","FileRelativePaths":["images/thumbnail.jpg","MapVariant.mvar"]},` +
	`"CustomData":{"NumOfObjectsOnMap":12,"TagLevelId":34,"IsBaked":true}}`

func ugcClientEnregistreur(tokens *domain.HaloTokens, statut int, corps string) (*Client, *haloapitest.Enregistreur) {
	e := &haloapitest.Enregistreur{Repondre: func(int, *http.Request) *http.Response {
		return haloapitest.Reponse(statut, corps)
	}}
	c := NewClient(tokens).WithHTTPClient(&http.Client{Transport: e})
	return c, e
}

func ugcJetons() *domain.HaloTokens {
	return &domain.HaloTokens{SpartanToken: "v4=SPARTAN", ClearanceToken: "CLR"}
}

// TestURLFigees_UGC_FetchAsset : Discovery, dernière version (sans segment `versions`) ou version
// précise ; segment `Maps` en majuscule, préfixe de jeu `hi` en dur ; les quatre en-têtes d'auth et
// d'identité ; l'identifiant est échappé en segment de chemin.
func TestURLFigees_UGC_FetchAsset(t *testing.T) {
	const hote = "https://discovery-infiniteugc.svc.halowaypoint.com/hi/Maps/"
	cas := []struct {
		nom, asset, version, attendu string
	}{
		{"derniere version", ugcFigeesAsset, "", "GET " + hote + ugcFigeesAsset + ugcEntetesAvecAuth},
		{"version precise", ugcFigeesAsset, ugcFigeesVersion, "GET " + hote + ugcFigeesAsset + "/versions/" + ugcFigeesVersion + ugcEntetesAvecAuth},
		{"identifiants echappes", "a/b c", "v/1", "GET " + hote + "a%2Fb%20c/versions/v%2F1" + ugcEntetesAvecAuth},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			client, e := ugcClientEnregistreur(ugcJetons(), http.StatusOK, ugcFigeesCorps)
			a, err := client.FetchAsset(context.Background(), c.asset, c.version)
			if err != nil {
				t.Fatal(err)
			}
			if len(e.Requetes()) != 1 || e.Requetes()[0] != c.attendu {
				t.Errorf("requêtes = %q\nattendu    %q", e.Requetes(), c.attendu)
			}
			if a.VersionID != ugcFigeesVersion || a.CustomData.NumOfObjectsOnMap != 12 || !a.CustomData.IsBaked ||
				len(MvarPaths(a)) != 1 || MvarPaths(a)[0] != "MapVariant.mvar" {
				t.Errorf("DTO mal décodé : %+v", a)
			}
		})
	}
}

// TestURLFigees_UGC_SansClearance : sans jeton de clearance, `343-clearance` n'est pas posé, les
// autres en-têtes d'auth le sont.
func TestURLFigees_UGC_SansClearance(t *testing.T) {
	client, e := ugcClientEnregistreur(&domain.HaloTokens{SpartanToken: "v4=SPARTAN"}, http.StatusOK, ugcFigeesCorps)
	if _, err := client.FetchAsset(context.Background(), ugcFigeesAsset, ""); err != nil {
		t.Fatal(err)
	}
	attendu := "GET https://discovery-infiniteugc.svc.halowaypoint.com/hi/Maps/" + ugcFigeesAsset +
		" [Accept=application/json; Origin=https://www.halowaypoint.com; User-Agent=" + ugcFigeesAgent + "; X-343-Authorization-Spartan=v4=SPARTAN]"
	if len(e.Requetes()) != 1 || e.Requetes()[0] != attendu {
		t.Errorf("requêtes = %q\nattendu    %q", e.Requetes(), attendu)
	}
}

// TestURLFigees_UGC_Blob : le stockage blob se lit sans aucun en-tête d'auth ni `Origin` (seulement
// `Accept` et `User-Agent`), avec un client à jetons nil ; le préfixe perd son `/` final, le chemin
// relatif n'est NI échappé NI nettoyé (les `/` internes restent, l'espace n'est encodé qu'à
// l'émission) ; `FetchMvar` prend le préfixe de l'asset.
func TestURLFigees_UGC_Blob(t *testing.T) {
	asset := &infinite.CarteUGC{AssetID: ugcFigeesAsset}
	asset.Files.Prefix = ugcFigeesBlob
	cas := []struct {
		nom     string
		appeler func(c *Client) ([]byte, error)
		attendu string
	}{
		{"FetchFileAt prefixe avec barre", func(c *Client) ([]byte, error) {
			return c.FetchFileAt(context.Background(), ugcFigeesBlob, "MapVariant.mvar")
		}, "GET " + ugcFigeesBlob + "MapVariant.mvar" + ugcEntetesSansAuth},
		{"FetchFileAt prefixe sans barre", func(c *Client) ([]byte, error) {
			return c.FetchFileAt(context.Background(), strings.TrimSuffix(ugcFigeesBlob, "/"), "images/thumbnail.jpg")
		}, "GET " + ugcFigeesBlob + "images/thumbnail.jpg" + ugcEntetesSansAuth},
		{"FetchFileAt chemin non echappe", func(c *Client) ([]byte, error) {
			return c.FetchFileAt(context.Background(), ugcFigeesBlob, "Variants/Mon Terrain.mvar")
		}, "GET " + ugcFigeesBlob + "Variants/Mon%20Terrain.mvar" + ugcEntetesSansAuth},
		{"FetchMvar", func(c *Client) ([]byte, error) {
			return c.FetchMvar(context.Background(), asset, "MapVariant.mvar")
		}, "GET " + ugcFigeesBlob + "MapVariant.mvar" + ugcEntetesSansAuth},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			client, e := ugcClientEnregistreur(nil, http.StatusOK, "OCTETS")
			octets, err := c.appeler(client)
			if err != nil || string(octets) != "OCTETS" {
				t.Fatalf("octets %q, erreur %v", octets, err)
			}
			if len(e.Requetes()) != 1 || e.Requetes()[0] != c.attendu {
				t.Errorf("requêtes = %q\nattendu    %q", e.Requetes(), c.attendu)
			}
		})
	}
}

// TestURLFigees_UGC_Erreurs : sans jeton Spartan (client nil ou jeton vide) Discovery est refusé
// AVANT toute requête, avec l'URL dans le message ; un statut hors 200 rend l'URL, le statut et le
// début du corps ; un préfixe ou un chemin vide est refusé avant toute requête.
func TestURLFigees_UGC_Erreurs(t *testing.T) {
	url := "https://discovery-infiniteugc.svc.halowaypoint.com/hi/Maps/" + ugcFigeesAsset
	for nom, tokens := range map[string]*domain.HaloTokens{"jetons nil": nil, "jeton vide": {ClearanceToken: "CLR"}} {
		client, e := ugcClientEnregistreur(tokens, http.StatusOK, ugcFigeesCorps)
		_, err := client.FetchAsset(context.Background(), ugcFigeesAsset, "")
		if want := "aucun jeton Spartan disponible pour " + url; err == nil || err.Error() != want || len(e.Requetes()) != 0 {
			t.Errorf("%s : erreur %v, %d requête(s), attendu %q et aucune requête", nom, err, len(e.Requetes()), want)
		}
	}

	client, _ := ugcClientEnregistreur(ugcJetons(), http.StatusNotFound, "  introuvable \n")
	_, err := client.FetchAsset(context.Background(), ugcFigeesAsset, "")
	if want := "GET " + url + ": HTTP 404 (introuvable)"; err == nil || err.Error() != want {
		t.Errorf("statut 404 : erreur %v, attendu %q", err, want)
	}

	client, e := ugcClientEnregistreur(nil, http.StatusOK, "")
	if _, err := client.FetchFileAt(context.Background(), "", "x"); err == nil || len(e.Requetes()) != 0 {
		t.Errorf("préfixe vide : erreur %v, %d requête(s), attendu un refus sans requête", err, len(e.Requetes()))
	}
	if _, err := client.FetchFileAt(context.Background(), ugcFigeesBlob, ""); err == nil || len(e.Requetes()) != 0 {
		t.Errorf("chemin vide : erreur %v, %d requête(s), attendu un refus sans requête", err, len(e.Requetes()))
	}
}
