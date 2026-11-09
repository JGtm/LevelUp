package infinite

// identifiants_url_test.go — les identifiants reçus par la brique (identifiant de match, gamertag,
// xuid, saison et playlist du classement) entrent dans l'URL échappés : un caractère réservé reste
// dans son segment, il ne coupe ni le chemin (`/`) ni la requête (`#`, `?`). Un identifiant de match
// qui n'est pas un UUID est refusé avant tout appel. Pour un identifiant sain, la cible de requête
// reçue par le serveur est celle d'avant l'échappement, octet pour octet.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// cibles : les cibles de requête (request-target, telles qu'écrites sur le fil) reçues par un
// serveur de test qui répond `{}`.
type cibles struct {
	mu  sync.Mutex
	vus []string
}

func (c *cibles) liste() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.vus...)
}

func serveurCibles(t *testing.T) (*httptest.Server, *cibles) {
	t.Helper()
	c := &cibles{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.mu.Lock()
		c.vus = append(c.vus, r.RequestURI)
		c.mu.Unlock()
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(srv.Close)
	return srv, c
}

const xuidTest = "2533274800000000"

// appelIdentifiant : un appel de la brique avec un identifiant donné, contre le serveur de test.
type appelIdentifiant func(ctx context.Context, srv *httptest.Server, id string) error

var appelsParSite = map[string]appelIdentifiant{
	"FetchMatchStats": func(ctx context.Context, srv *httptest.Server, id string) error {
		_, err := clientSur(srv).FetchMatchStats(ctx, authTest, id)
		return err
	},
	"GetServiceRecord": func(ctx context.Context, srv *httptest.Server, id string) error {
		_, err := clientSur(srv).GetServiceRecord(ctx, authTest, id)
		return err
	},
	"GetSeasonServiceRecord": func(ctx context.Context, srv *httptest.Server, id string) error {
		classe := true
		_, err := clientSur(srv).GetSeasonServiceRecord(ctx, authTest, id, "Seasons/Season7.json", &classe)
		return err
	},
	"GetOperations": func(ctx context.Context, srv *httptest.Server, id string) error {
		_, _, err := clientSur(srv).GetOperations(ctx, authTest, id)
		return err
	},
	"GetDecks": func(ctx context.Context, srv *httptest.Server, id string) error {
		_, _, err := clientSur(srv).GetDecks(ctx, authTest, id)
		return err
	},
	"GetMatchPrivacy": func(ctx context.Context, srv *httptest.Server, id string) error {
		_, err := clientSur(srv).GetMatchPrivacy(ctx, authTest, id)
		return err
	},
	"FetchPageBytes(saison)": func(ctx context.Context, srv *httptest.Server, id string) error {
		_, err := classementSur(srv, nil).FetchPageBytes(ctx, id, matchTest, 2, "")
		return err
	},
	"FetchPageBytes(playlist)": func(ctx context.Context, srv *httptest.Server, id string) error {
		_, err := classementSur(srv, nil).FetchPageBytes(ctx, "csrseason13-3", id, 2, "")
		return err
	},
}

type casIdentifiant struct {
	site, id, cible string
}

// TestIdentifiantsSains_CibleInchangee : les cibles de requête d'AVANT l'échappement, figées avant
// la modification, pour des identifiants sains (UUID, gamertag alphanumérique, xuid numérique,
// forme `xuid(NNN)` du service record, saison CSR).
func TestIdentifiantsSains_CibleInchangee(t *testing.T) {
	cas := []casIdentifiant{
		{"FetchMatchStats", matchTest, "/hi/players/xuid(0)/matches/" + matchTest + "/stats"},
		{"GetServiceRecord", "JGtm", "/hi/players/JGtm/Matchmade/servicerecord"},
		{"GetServiceRecord", "xuid(" + xuidTest + ")", "/hi/players/xuid(" + xuidTest + ")/Matchmade/servicerecord"},
		{"GetSeasonServiceRecord", "JGtm", "/hi/players/JGtm/Matchmade/servicerecord?isRanked=true&seasonId=Seasons%2FSeason7.json"},
		{"GetOperations", xuidTest, "/hi/players/xuid(" + xuidTest + ")/rewardtracks/operations"},
		{"GetDecks", xuidTest, "/hi/players/xuid(" + xuidTest + ")/decks"},
		{"GetMatchPrivacy", xuidTest, "/hi/players/xuid(" + xuidTest + ")/matches-privacy"},
		{"FetchPageBytes(saison)", "csrseason13-3", "/halo-infinite/leaderboards/csrseason13-3/" + matchTest + "?page=2"},
		{"FetchPageBytes(playlist)", matchTest, "/halo-infinite/leaderboards/csrseason13-3/" + matchTest + "?page=2"},
	}
	verifierCibles(t, cas)
}

// TestIdentifiantsReserves_RestentDansLeurSegment : un espace, une barre, un dièse ou un point
// d'interrogation dans un identifiant reste DANS son segment (échappé), jamais une URL coupée.
func TestIdentifiantsReserves_RestentDansLeurSegment(t *testing.T) {
	cas := []casIdentifiant{
		{"GetServiceRecord", "Foo Bar/Baz", "/hi/players/Foo%20Bar%2FBaz/Matchmade/servicerecord"},
		{"GetServiceRecord", "Nom#1234", "/hi/players/Nom%231234/Matchmade/servicerecord"},
		{"GetSeasonServiceRecord", "Foo Bar/Baz", "/hi/players/Foo%20Bar%2FBaz/Matchmade/servicerecord?isRanked=true&seasonId=Seasons%2FSeason7.json"},
		{"GetOperations", "1/2", "/hi/players/xuid(1%2F2)/rewardtracks/operations"},
		{"GetDecks", "1?a=b", "/hi/players/xuid(1%3Fa=b)/decks"},
		{"GetMatchPrivacy", "1#2", "/hi/players/xuid(1%232)/matches-privacy"},
		{"FetchPageBytes(saison)", "csr/season", "/halo-infinite/leaderboards/csr%2Fseason/" + matchTest + "?page=2"},
		{"FetchPageBytes(playlist)", "a b?c", "/halo-infinite/leaderboards/csrseason13-3/a%20b%3Fc?page=2"},
	}
	verifierCibles(t, cas)
}

// TestFetchMatchStats_RefuseUnIdentifiantQuiNEstPasUnUUID : refus AVANT l'appel (le serveur ne
// reçoit rien), comme GetMatchStats, la confidentialité et le film.
func TestFetchMatchStats_RefuseUnIdentifiantQuiNEstPasUnUUID(t *testing.T) {
	for _, id := range []string{"../x", "", "abc", matchTest + "/../y", matchTest + "?a=b"} {
		srv, vus := serveurCibles(t)
		if err := appelsParSite["FetchMatchStats"](context.Background(), srv, id); err == nil {
			t.Errorf("FetchMatchStats(%q) : pas d'erreur", id)
		}
		if got := vus.liste(); len(got) != 0 {
			t.Errorf("FetchMatchStats(%q) : %d requête(s) envoyée(s) %v, attendu 0", id, len(got), got)
		}
	}
}

func verifierCibles(t *testing.T, cas []casIdentifiant) {
	t.Helper()
	for _, c := range cas {
		srv, vus := serveurCibles(t)
		if err := appelsParSite[c.site](context.Background(), srv, c.id); err != nil {
			t.Errorf("%s(%q) : %v", c.site, c.id, err)
			continue
		}
		got := vus.liste()
		if len(got) != 1 || got[0] != c.cible {
			t.Errorf("%s(%q) : cible %q, attendu %q", c.site, c.id, got, c.cible)
		}
	}
}
