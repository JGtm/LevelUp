package sync

// Témoin T2 « URL figées » du client Xbox Achievements v2 (plan PUBLICATION, lot 1.2) : méthode,
// URL absolue et en-têtes triés de chaque page demandée, en littéral. Le chemin HTTP de ce client
// n'avait aucun test avant ce fichier. Il a été coupé en protocole (`haloapi/xboxlive`) et exploitation
// (ce paquet) au lot 2.4 sans changement d'octet sur le fil : ces attendus n'ont pas bougé.

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/platform/auth"

	"github.com/JGtm/halo-api/haloapitest"
	"github.com/JGtm/halo-api/xboxlive"
)

const (
	xboxFigeesXuid   = "2535428000000001"
	xboxFigeesTitre  = "1144039928"
	xboxFigeesHote   = "https://achievements.xboxlive.com/users/xuid(" + xboxFigeesXuid + ")/achievements"
	xboxFigeesAuth   = "XBL3.0 x=9876543210;jeton-xsts"
	xboxFigeesPage1  = `{"achievements":[{"id":"1","name":"Un"},{"id":"2","name":"Deux"}],"pagingInfo":{"totalRecords":3,"continuationToken":"suite"}}`
	xboxFigeesPage2  = `{"achievements":[{"id":"3","name":"Trois"}],"pagingInfo":{"totalRecords":3,"continuationToken":""}}`
	xboxFigeesEntete = " [Accept=application/json; Accept-Language=en-US; Authorization=" + xboxFigeesAuth + "; X-Xbl-Contract-Version=2]"
)

// xboxFigeesEnregistreur : rend la page n de `pages` (200 sauf `statuts[n-1]`, indexé depuis 0).
func xboxFigeesEnregistreur(pages []string, statuts map[int]int) *haloapitest.Enregistreur {
	return &haloapitest.Enregistreur{Repondre: func(n int, _ *http.Request) *http.Response {
		statut := http.StatusOK
		if s, ok := statuts[n-1]; ok {
			statut = s
		}
		return haloapitest.Reponse(statut, pages[n-1])
	}}
}

func xboxFigeesClient(t *testing.T, titre string, e *haloapitest.Enregistreur) XboxAchievementsClient {
	t.Helper()
	c := NewXboxHTTPClient(&auth.XSTSResult{UserHash: "9876543210", Token: "jeton-xsts"}, titre)
	hc, ok := c.(*xboxHTTPClient)
	if !ok {
		t.Fatalf("NewXboxHTTPClient rend %T, attendu *xboxHTTPClient", c)
	}
	hc.api = xboxlive.New(&http.Client{Transport: e})
	return c
}

// TestURLFigees_Xbox_Pagination : deux pages. Première page sans `skipItems`, seconde avec le nombre
// d'éléments déjà reçus, la clé `skipItems` avant `titleId` (ordre alphabétique de l'encodage), le
// `xuid(...)` du chemin ni échappé ni encodé, les quatre en-têtes ; la boucle s'arrête sur un
// `continuationToken` vide et rend les trois éléments.
func TestURLFigees_Xbox_Pagination(t *testing.T) {
	e := xboxFigeesEnregistreur([]string{xboxFigeesPage1, xboxFigeesPage2}, nil)
	liste, err := xboxFigeesClient(t, xboxFigeesTitre, e).GetPlayerAchievements(context.Background(), xboxFigeesXuid, "en-US")
	if err != nil {
		t.Fatal(err)
	}
	attendu := []string{
		"GET " + xboxFigeesHote + "?titleId=" + xboxFigeesTitre + xboxFigeesEntete,
		"GET " + xboxFigeesHote + "?skipItems=2&titleId=" + xboxFigeesTitre + xboxFigeesEntete,
	}
	if strings.Join(e.Requetes(), "\n") != strings.Join(attendu, "\n") {
		t.Errorf("requêtes =\n%s\nattendu\n%s", strings.Join(e.Requetes(), "\n"), strings.Join(attendu, "\n"))
	}
	if len(liste) != 3 || liste[0].ID != "1" || liste[1].Name != "Deux" || liste[2].ID != "3" {
		t.Errorf("liste = %+v, attendu les trois éléments des deux pages dans l'ordre", liste)
	}
}

// TestURLFigees_Xbox_SansTitre : sans identifiant de titre, aucun `titleId` n'est posé (ni `?` sur la
// première page) ; la langue vide est quand même posée en `Accept-Language` vide.
func TestURLFigees_Xbox_SansTitre(t *testing.T) {
	e := xboxFigeesEnregistreur([]string{xboxFigeesPage1, xboxFigeesPage2}, nil)
	if _, err := xboxFigeesClient(t, "", e).GetPlayerAchievements(context.Background(), xboxFigeesXuid, ""); err != nil {
		t.Fatal(err)
	}
	const entetes = " [Accept=application/json; Accept-Language=; Authorization=" + xboxFigeesAuth + "; X-Xbl-Contract-Version=2]"
	attendu := []string{
		"GET " + xboxFigeesHote + entetes,
		"GET " + xboxFigeesHote + "?skipItems=2" + entetes,
	}
	if strings.Join(e.Requetes(), "\n") != strings.Join(attendu, "\n") {
		t.Errorf("requêtes =\n%s\nattendu\n%s", strings.Join(e.Requetes(), "\n"), strings.Join(attendu, "\n"))
	}
}

// TestURLFigees_Xbox_PageEnErreur : un statut hors 200 abandonne toute la lecture (aucune liste
// partielle) avec l'erreur qui porte le statut et le corps ENTIER ; aucune reprise.
func TestURLFigees_Xbox_PageEnErreur(t *testing.T) {
	for nom, c := range map[string]struct {
		pages   []string
		statuts map[int]int
		requete int
	}{
		"première page": {[]string{"corps-de-refus"}, map[int]int{0: http.StatusForbidden}, 1},
		"seconde page":  {[]string{xboxFigeesPage1, "corps-de-refus"}, map[int]int{1: http.StatusForbidden}, 2},
	} {
		e := xboxFigeesEnregistreur(c.pages, c.statuts)
		liste, err := xboxFigeesClient(t, xboxFigeesTitre, e).GetPlayerAchievements(context.Background(), xboxFigeesXuid, "en-US")
		if want := "xbox_client: HTTP 403: corps-de-refus"; err == nil || err.Error() != want {
			t.Errorf("%s : erreur %v, attendu %q", nom, err, want)
		}
		if liste != nil || len(e.Requetes()) != c.requete {
			t.Errorf("%s : liste %v, %d requête(s), attendu aucune liste et %d requête(s)", nom, liste, len(e.Requetes()), c.requete)
		}
	}
}
