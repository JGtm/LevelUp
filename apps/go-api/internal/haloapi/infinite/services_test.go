package infinite

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// capture : les requêtes reçues par un serveur de test.
type capture struct {
	mu      sync.Mutex
	urls    []*url.URL
	entetes []http.Header
}

func (c *capture) noter(r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()
	u := *r.URL
	c.urls = append(c.urls, &u)
	c.entetes = append(c.entetes, r.Header.Clone())
}

func (c *capture) derniere(t *testing.T) (*url.URL, http.Header) {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.urls) == 0 {
		t.Fatal("aucune requête reçue")
	}
	return c.urls[len(c.urls)-1], c.entetes[len(c.entetes)-1]
}

func (c *capture) nombre() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.urls)
}

// serveurCorps sert `corps` en 200 à toute requête et capture les requêtes.
func serveurCorps(t *testing.T, corps []byte) (*httptest.Server, *capture) {
	t.Helper()
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.noter(r)
		_, _ = w.Write(corps)
	}))
	t.Cleanup(srv.Close)
	return srv, cap
}

// endpointsVers rend des adresses dont TOUTES les clés visent `base`.
func endpointsVers(base string) Endpoints {
	e := DefaultEndpoints()
	for k := range hotesParDefaut() {
		e = e.Avec(k, base)
	}
	return e
}

// clientSur rend un client (politique du client de synchronisation, sans délai) dont toutes les
// adresses visent le serveur de test.
func clientSur(srv *httptest.Server) *Client {
	return New(srv.Client(), sansDelai(RetryHaloClient())).WithEndpoints(endpointsVers(srv.URL))
}

var authTest = Auth{Spartan: "sp", Clearance: "cl"}

const matchTest = "b8c1b220-5ef4-4dee-9e92-77d3ff55d6d3"

func lireFixture(t *testing.T, nom string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", nom))
	if err != nil {
		t.Fatalf("fixture %s : %v", nom, err)
	}
	return b
}

// TestGetMatchHistory_Fixture : la fixture d'historique décode le DTO ; chemin et requête exacts.
func TestGetMatchHistory_Fixture(t *testing.T) {
	srv, cap := serveurCorps(t, lireFixture(t, "halo/match_history_response.json"))
	dto, err := clientSur(srv).GetMatchHistory(context.Background(), authTest, "xuid(123)", "matchmaking", 0, 25)
	if err != nil {
		t.Fatal(err)
	}
	if len(dto.Results) != 4 || dto.Results[0].MatchID != "11111111-1111-1111-1111-000000000001" ||
		dto.Results[3].MatchInfo.StartTime != "2026-05-06T18:01:25.539Z" {
		t.Errorf("DTO = %+v", dto)
	}
	u, h := cap.derniere(t)
	if u.Path != "/hi/players/xuid(123)/matches" || u.Query().Get("start") != "0" ||
		u.Query().Get("count") != "25" || u.Query().Get("type") != "matchmaking" {
		t.Errorf("URL = %s", u)
	}
	if h.Get(enteteSpartan) != "sp" || h.Get(enteteClearance) != "cl" {
		t.Errorf("jetons absents : %v", h)
	}
	for _, args := range []struct {
		j, typ string
		s, n   int
	}{{"", "all", 0, 1}, {"x", "bof", 0, 1}, {"x", "all", -1, 1}, {"x", "all", 0, 26}} {
		if _, err := clientSur(srv).GetMatchHistory(context.Background(), authTest, args.j, args.typ, args.s, args.n); err == nil {
			t.Errorf("%+v : erreur attendue", args)
		}
	}
	if cap.nombre() != 1 {
		t.Errorf("%d requêtes : une entrée invalide ne doit rien envoyer", cap.nombre())
	}
}

// TestStats_DeuxHotes : GetMatchStats vise CleStats, FetchMatchStats CleStatsSansPort par xuid(0).
func TestStats_DeuxHotes(t *testing.T) {
	srvA, capA := serveurCorps(t, []byte(`{"MatchId":"m","MatchInfo":{"Playlist":{"AssetId":"p"}}}`))
	srvB, capB := serveurCorps(t, []byte(`{"MatchId":"m2"}`))
	e := DefaultEndpoints().Avec(CleStats, srvA.URL).Avec(CleStatsSansPort, srvB.URL)
	c := New(srvA.Client(), sansDelai(RetryProvider())).WithEndpoints(e)
	m, err := c.GetMatchStats(context.Background(), authTest, matchTest)
	if err != nil || m["MatchId"] != "m" {
		t.Fatalf("GetMatchStats = %v, %v", m, err)
	}
	if u, _ := capA.derniere(t); u.Path != "/hi/matches/"+matchTest+"/stats" {
		t.Errorf("GetMatchStats URL = %s", u)
	}
	m2, err := c.FetchMatchStats(context.Background(), authTest, matchTest)
	if err != nil || m2["MatchId"] != "m2" {
		t.Fatalf("FetchMatchStats = %v, %v", m2, err)
	}
	if u, _ := capB.derniere(t); u.Path != "/hi/players/xuid(0)/matches/"+matchTest+"/stats" {
		t.Errorf("FetchMatchStats URL = %s", u)
	}
	if _, err := c.GetMatchStats(context.Background(), authTest, "pas-un-uuid"); err == nil {
		t.Error("identifiant invalide : erreur attendue")
	}
}

const serviceRecordJSON = `{"MatchesCompleted":10,"Wins":6,"Losses":4,"TimePlayed":"PT1H",
"CoreStats":{"Kills":120,"Deaths":80,"Assists":30,"ShotsFired":1000,"ShotsHit":500,"DamageDealt":9000.5,"DamageTaken":7000,
"Medals":[{"NameId":1512363953,"Count":3}]},
"Subqueries":{"SeasonIds":["Seasons/Season7.json"],"IsRanked":[true,false],"PlaylistAssetIds":["p1"]}}`

// TestServiceRecord : cumulatif et par saison (requête seasonId/isRanked), DTO décodé.
func TestServiceRecord(t *testing.T) {
	srv, cap := serveurCorps(t, []byte(serviceRecordJSON))
	c := clientSur(srv)
	dto, err := c.GetServiceRecord(context.Background(), authTest, "JGtm")
	if err != nil {
		t.Fatal(err)
	}
	if dto.MatchesCompleted != 10 || dto.CoreStats.Kills != 120 || dto.CoreStats.DamageDealt != 9000.5 ||
		len(dto.CoreStats.Medals) != 1 || dto.CoreStats.Medals[0].NameID != 1512363953 ||
		dto.Subqueries.SeasonIDs[0] != "Seasons/Season7.json" || len(dto.Subqueries.IsRanked) != 2 {
		t.Errorf("DTO = %+v", dto)
	}
	if u, _ := cap.derniere(t); u.Path != "/hi/players/JGtm/Matchmade/servicerecord" || u.RawQuery != "" {
		t.Errorf("URL = %s", u)
	}
	classe := true
	if _, err := c.GetSeasonServiceRecord(context.Background(), authTest, "JGtm", "Seasons/Season7.json", &classe); err != nil {
		t.Fatal(err)
	}
	if u, _ := cap.derniere(t); u.Query().Get("seasonId") != "Seasons/Season7.json" || u.Query().Get("isRanked") != "true" {
		t.Errorf("URL = %s", u)
	}
	if _, err := c.GetSeasonServiceRecord(context.Background(), authTest, "JGtm", "", nil); err == nil {
		t.Error("seasonID vide : erreur attendue")
	}
	want := DefaultEndpoints().Hote(CleStatsSansPort) + "/hi/players/g/Matchmade/servicerecord?seasonId=S%2F1.json"
	if got := BuildSeasonServiceRecordURL(DefaultEndpoints(), "g", "S/1.json", nil); got != want {
		t.Errorf("BuildSeasonServiceRecordURL = %s", got)
	}
}

const skillJSON = `{"Value":[
{"Id":"xuid(1)","ResultCode":0,"Result":{"TeamMmr":1500.5,"TeamId":0,"TeamMmrs":{"0":1500.5,"1":"1450"},
 "StatPerformances":{"Kills":{"Count":12,"Expected":"1.5e-3","StdDev":2.3},"Deaths":{"Count":8,"Expected":9.1,"StdDev":null}},
 "RankRecap":{"PreMatchCsr":{"Value":1200,"Tier":"Diamond","SubTier":2,"MeasurementMatchesRemaining":0,"InitialMeasurementMatches":5},
              "PostMatchCsr":{"Value":1210,"Tier":"Diamond","SubTier":3}}}},
{"Id":"xuid(2)","ResultCode":1,"Result":{}}]}`

// TestGetMatchSkill : bots retirés de la requête, chaînes numériques décodées, RankRecap lu.
func TestGetMatchSkill(t *testing.T) {
	srv, cap := serveurCorps(t, []byte(skillJSON))
	c := clientSur(srv)
	dto, err := c.GetMatchSkill(context.Background(), authTest, matchTest, []string{"1", " 2 ", "bid(3.0)", ""})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := cap.derniere(t)
	if u.Path != "/hi/matches/"+matchTest+"/skill" || len(u.Query()["players"]) != 2 || u.Query()["players"][1] != "xuid(2)" {
		t.Errorf("URL = %s", u)
	}
	v := dto.Value[0]
	if v.Result.TeamMMR != 1500.5 || v.Result.TeamMMRs["1"] != 1450 || v.Result.StatPerformances.Kills.Expected != 0.0015 ||
		v.Result.StatPerformances.Deaths.StdDev != 0 || v.Result.RankRecap.PreMatchCsr.InitialMeasurementMatches != 5 ||
		v.Result.RankRecap.PostMatchCsr.SubTier != 3 || dto.Value[1].ResultCode != 1 {
		t.Errorf("DTO = %+v", v)
	}
	vide, err := c.GetMatchSkill(context.Background(), authTest, matchTest, []string{"bid(1.0)"})
	if err != nil || len(vide.Value) != 0 || cap.nombre() != 1 {
		t.Errorf("sans humain : %+v, %v, %d requêtes", vide, err, cap.nombre())
	}
	if d, err := DecodeMatchSkill(nil); err != nil || len(d.Value) != 0 {
		t.Errorf("corps vide : %+v, %v", d, err)
	}
	if UnwrapXUID(" xuid(42) ") != "42" || UnwrapXUID("bid(1)") != "bid(1)" {
		t.Error("UnwrapXUID")
	}
}

// TestCSR : CSR d'un joueur (par saison) et d'une playlist (saison facultative).
func TestCSR(t *testing.T) {
	corps := `{"Value":[{"Id":"xuid(7)","ResultCode":0,"Result":{"Current":{"Value":1300,"Tier":"Onyx"},
"SeasonMax":{"Value":1400},"AllTimeMax":{"Value":1500},"PlaylistId":"pl","PlaylistName":"Ranked Arena","Queue":"Open","Input":"Crossplay"}}]}`
	srv, cap := serveurCorps(t, []byte(corps))
	c := clientSur(srv)
	p, err := c.GetPlayerCSRs(context.Background(), authTest, "7", "CsrSeason8")
	if err != nil || p.Value[0].Result.Current.Tier != "Onyx" || p.Value[0].Result.Queue != "Open" || p.Value[0].Result.AllTimeMax.Value != 1500 {
		t.Fatalf("GetPlayerCSRs = %+v, %v", p, err)
	}
	if u, _ := cap.derniere(t); u.Path != "/hi/players/xuid(7)/csrs" || u.Query().Get("Season") != "CsrSeason8" {
		t.Errorf("URL = %s", u)
	}
	pl, err := c.GetPlaylistCsr(context.Background(), authTest, "pl", "7", "")
	if err != nil || pl.Value[0].Result.SeasonMax.Value != 1400 {
		t.Fatalf("GetPlaylistCsr = %+v, %v", pl, err)
	}
	if u, _ := cap.derniere(t); u.Path != "/hi/playlist/pl/csrs" || u.Query().Get("players") != "xuid(7)" || u.Query().Has("season") {
		t.Errorf("URL = %s", u)
	}
	if _, err := c.GetPlaylistCsr(context.Background(), authTest, "pl", "7", "S9"); err != nil {
		t.Fatal(err)
	}
	if u, _ := cap.derniere(t); u.Query().Get("season") != "S9" {
		t.Errorf("URL = %s", u)
	}
}

// TestGetCareerProgress : les deux formes de la réponse rendent la même progression.
func TestGetCareerProgress(t *testing.T) {
	for _, corps := range []string{
		`{"CurrentProgress":{"Rank":42,"PartialProgress":1500,"HasReachedMaxRank":false}}`,
		`{"RewardTracks":[{"Result":{}},{"Result":{"CurrentProgress":{"Rank":42,"PartialProgress":1500}}}]}`,
	} {
		srv, cap := serveurCorps(t, []byte(corps))
		dto, err := clientSur(srv).GetCareerProgress(context.Background(), authTest, "9")
		if err != nil {
			t.Fatal(err)
		}
		if p := dto.Progress(); p == nil || p.Rank != 42 || p.PartialProgress != 1500 {
			t.Errorf("Progress = %+v", p)
		}
		if u, _ := cap.derniere(t); u.Path != "/hi/careerranks/careerRank1" || u.Query().Get("players") != "xuid(9)" {
			t.Errorf("URL = %s", u)
		}
	}
	if (&CareerRankDTO{}).Progress() != nil {
		t.Error("sans progression : nil attendu")
	}
}
