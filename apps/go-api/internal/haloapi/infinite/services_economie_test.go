package infinite

import (
	"context"
	"strings"
	"testing"
)

// TestCustomization : apparence à portée joueur et vue publique, champs lus par leurs chemins,
// entrées de diagnostic.
func TestCustomization(t *testing.T) {
	corps := `{"Appearance":{"ServiceTag":"ABC1","BackdropImagePath":"Inv/Backdrop.json",
"Emblem":{"EmblemPath":"Inventory/Spartan/Emblems/104-001-x.json","ConfigurationId":-809699482},
"Nameplate":{"ImagePath":"Inv/Nameplate.json"}}}`
	srv, cap := serveurCorps(t, []byte(corps))
	c := clientSur(srv)
	a, err := c.GetCustomizationAppearance(context.Background(), authTest, "5")
	if err != nil {
		t.Fatal(err)
	}
	if a.ServiceTag != "ABC1" || a.BannerImagePath != "Inv/Nameplate.json" || a.BackdropImagePath != "Inv/Backdrop.json" ||
		a.EmblemPath != "Inventory/Spartan/Emblems/104-001-x.json" || a.EmblemConfigurationID != "-809699482" {
		t.Errorf("DTO = %+v", a)
	}
	if u, _ := cap.derniere(t); u.Path != "/hi/players/xuid(5)/customization/appearance" {
		t.Errorf("URL = %s", u)
	}
	if _, err := c.GetCustomizationPublic(context.Background(), authTest, "5"); err != nil {
		t.Fatal(err)
	}
	if u, _ := cap.derniere(t); u.Path != "/hi/players/xuid(5)/customization" || u.Query().Get("view") != "public" {
		t.Errorf("URL = %s", u)
	}
	in := AppearanceInputsDe(a)
	if in.EmblemConfigID != -809699482 || in.ServiceTag != "ABC1" || AppearanceInputsDe(nil) != nil {
		t.Errorf("AppearanceInputsDe = %+v", in)
	}
	if _, err := c.GetCustomizationAppearance(context.Background(), authTest, " "); err == nil {
		t.Error("xuid vide : erreur attendue")
	}
}

// TestResolveCustomizationImageURL : descripteur GameCMS -> URL d'image sous /{jeu}/images/file/.
func TestResolveCustomizationImageURL(t *testing.T) {
	srv, cap := serveurCorps(t, []byte(`{"CommonData":{"DisplayPath":{"Media":{"MediaUrl":{"Path":"progression/x.png"}}}}}`))
	got, err := clientSur(srv).ResolveCustomizationImageURL(context.Background(), authTest, "/Inventory/Spartan/Emblems/e.json")
	if err != nil || got != srv.URL+"/hi/images/file/progression/x.png" {
		t.Fatalf("URL = %q, %v", got, err)
	}
	if u, _ := cap.derniere(t); u.Path != "/hi/progression/file/Inventory/Spartan/Emblems/e.json" {
		t.Errorf("URL = %s", u)
	}
	for in, veut := range map[string]string{
		"https://cdn/x.png":    "https://cdn/x.png",
		"hi/images/file/a.png": "B/hi/images/file/a.png",
		"/images/file/a.png":   "B/hi/images/file/a.png",
		"a.png":                "B/hi/images/file/a.png",
		"":                     "",
	} {
		if got := buildCustomizationImageURL("B/", "hi", in); got != veut {
			t.Errorf("build(%q) = %q, veut %q", in, got, veut)
		}
	}
}

// TestOperationsEtDecks : DTO et corps brut rendus ; corps illisible rendu avec l'erreur.
func TestOperationsEtDecks(t *testing.T) {
	ops := `{"ActiveOperationRewardTrackPath":"RT/op2.json","OperationRewardTracks":[
{"RewardTrackPath":"RT/op1.json","CurrentProgress":{"Rank":3,"PartialProgress":10}},
{"RewardTrackPath":"RT/op2.json","CurrentProgress":{"Rank":7,"PartialProgress":250}}]}`
	srv, cap := serveurCorps(t, []byte(ops))
	c := clientSur(srv)
	dto, raw, err := c.GetOperations(context.Background(), authTest, "11")
	if err != nil || string(raw) != ops || dto.OperationRewardTracks[1].CurrentProgress.Rank != 7 ||
		dto.ActiveOperationRewardTrackPath != "RT/op2.json" {
		t.Fatalf("GetOperations = %+v, %v", dto, err)
	}
	if u, _ := cap.derniere(t); u.Path != "/hi/players/xuid(11)/rewardtracks/operations" {
		t.Errorf("URL = %s", u)
	}
	decks := `{"AssignedDecks":[{"Expiration":{"ISO8601Date":"2026-11-01T00:00:00Z"},
"ActiveChallenges":[{"Path":"C/a.json","TrackingId":"t","XPReward":500,"Threshold":3,"Progress":1,"CanReroll":true}],
"CompletedChallenges":[{"Path":"C/b.json"}]}]}`
	srv2, cap2 := serveurCorps(t, []byte(decks))
	d, raw2, err := clientSur(srv2).GetDecks(context.Background(), authTest, "11")
	if err != nil || string(raw2) != decks || *d.AssignedDecks[0].ActiveChallenges[0].XPReward != 500 ||
		len(d.AssignedDecks[0].CompletedChallenges) != 1 || d.AssignedDecks[0].Expiration.ISO8601Date == "" {
		t.Fatalf("GetDecks = %+v, %v", d, err)
	}
	if u, _ := cap2.derniere(t); u.Path != "/hi/players/xuid(11)/decks" {
		t.Errorf("URL = %s", u)
	}
	srv3, _ := serveurCorps(t, []byte(`pas du json`))
	if _, raw3, err := clientSur(srv3).GetDecks(context.Background(), authTest, "11"); err == nil || string(raw3) != "pas du json" {
		t.Errorf("corps illisible : raw=%q err=%v", raw3, err)
	}
}

// TestFetchAsset : Accept-Language de l'appelant, segment du type, nom localisé, miniature.
func TestFetchAsset(t *testing.T) {
	corps := `{"AssetId":"a1","VersionId":"v1","PublicName":{"value":"Cliffhanger"},"Description":"desc",
"Files":{"Prefix":"https://blobs/x","FileRelativePaths":["map.mvar","images/hero.jpg","images/thumbnail.png"]}}`
	srv, cap := serveurCorps(t, []byte(corps))
	c := New(srv.Client(), sansDelai(RetryDiscovery())).WithEndpoints(endpointsVers(srv.URL))
	a, err := c.FetchAsset(context.Background(), authTest, AssetTypeMap, "a1", "v1", "xx-YY")
	if err != nil {
		t.Fatal(err)
	}
	if a.PublicName != "Cliffhanger" || a.Description != "desc" || a.ImageURL != "https://blobs/x/images/thumbnail.png" {
		t.Errorf("actif = %+v", a)
	}
	u, h := cap.derniere(t)
	if u.Path != "/hi/maps/a1/versions/v1" || h.Get(enteteLangue) != "xx-YY" {
		t.Errorf("URL = %s, langue = %q", u, h.Get(enteteLangue))
	}
	if _, err := c.FetchAsset(context.Background(), authTest, "inconnu", "a", "v", ""); err == nil {
		t.Error("type invalide : erreur attendue")
	}
	if _, err := c.FetchAsset(context.Background(), authTest, AssetTypeMap, "a", "", ""); err == nil {
		t.Error("version vide : erreur attendue")
	}
	for _, typ := range []AssetType{AssetTypeMap, AssetTypePlaylist, AssetTypePair, AssetTypeGameVariant} {
		if s, ok := SegmentDAsset(typ); !ok || s == "" {
			t.Errorf("SegmentDAsset(%s)", typ)
		}
		if k, ok := CleMatchInfo(typ); !ok || k == "" {
			t.Errorf("CleMatchInfo(%s)", typ)
		}
	}
	if BuildAssetImageURL("p", []string{"a.txt"}) != "" || BuildAssetImageURL("p/", []string{"x/HERO.PNG"}) != "p/x/HERO.PNG" {
		t.Error("BuildAssetImageURL")
	}
}

// TestGameCMS : médailles et calendriers, corps brut rendu.
func TestGameCMS(t *testing.T) {
	cas := []struct {
		corps, chemin string
		appel         func(c *Client) (int, []byte, error)
	}{
		{`{"Medals":[{"NameId":"Killjoy","DescriptionId":"d","SpriteIndex":4,"Difficulty":"normal","Type":"spree","PersonalScore":5,"MedalId":2063152177}]}`,
			"/hi/Progression/file/medals/metadata.json",
			func(c *Client) (int, []byte, error) {
				d, b, err := c.FetchMedalsMetadata(context.Background(), authTest)
				if err != nil {
					return 0, b, err
				}
				if d.Medals[0].MedalID != 2063152177 || d.Medals[0].SpriteIndex != 4 {
					t.Errorf("médaille = %+v", d.Medals[0])
				}
				return len(d.Medals), b, nil
			}},
		{`{"Seasons":[{"SeasonMetadataPath":"Seasons/S1.json","StartDate":"2021-11-15T00:00:00Z"},{"SeasonMetadataPath":"Seasons/S2.json","StartDate":"2022-05-03T00:00:00Z","EndDate":"2022-11-08T00:00:00Z"}]}`,
			"/hi/Progression/file/SeasonCalendar.json",
			func(c *Client) (int, []byte, error) {
				d, b, err := c.FetchSeasonCalendar(context.Background(), authTest)
				if err != nil {
					return 0, b, err
				}
				if d.Seasons[1].EndDate == "" {
					t.Error("EndDate absent")
				}
				return len(d.Seasons), b, nil
			}},
		{`{"Seasons":[{"CsrSeasonFilePath":"Csr/S1.json","StartDate":"2021-11-15T00:00:00Z"}]}`,
			"/hi/Progression/file/CsrSeasonCalendar.json",
			func(c *Client) (int, []byte, error) {
				d, b, err := c.FetchCSRSeasonCalendar(context.Background(), authTest)
				if err != nil {
					return 0, b, err
				}
				return len(d.Seasons), b, nil
			}},
	}
	for _, k := range cas {
		srv, cap := serveurCorps(t, []byte(k.corps))
		n, raw, err := k.appel(clientSur(srv))
		if err != nil || n == 0 || string(raw) != k.corps {
			t.Errorf("%s : n=%d err=%v", k.chemin, n, err)
		}
		if u, _ := cap.derniere(t); u.Path != k.chemin {
			t.Errorf("URL = %s, veut %s", u.Path, k.chemin)
		}
	}
	if !strings.HasSuffix(SeasonCalendarURL(DefaultEndpoints()), "/hi/Progression/file/SeasonCalendar.json") {
		t.Error("SeasonCalendarURL")
	}
}

// TestGetMatchPrivacy : chemin et DTO.
func TestGetMatchPrivacy(t *testing.T) {
	srv, cap := serveurCorps(t, []byte(`{"AllMatchesPrivacy":"Show","PublicMatchesPrivacy":"Private","RankedMatchesPrivacy":"Show","CustomMatchesPrivacy":"Show"}`))
	p, err := clientSur(srv).GetMatchPrivacy(context.Background(), authTest, "3")
	if err != nil || p.PublicMatchesPrivacy != "Private" || p.AllMatchesPrivacy != "Show" {
		t.Fatalf("privacy = %+v, %v", p, err)
	}
	if u, _ := cap.derniere(t); u.Path != "/hi/players/xuid(3)/matches-privacy" {
		t.Errorf("URL = %s", u)
	}
}

// TestGetPlaylistConfig : couples de RotationEntries, repli CustomData, version requise.
func TestGetPlaylistConfig(t *testing.T) {
	srv, cap := serveurCorps(t, []byte(`{"PublicName":"Ranked Arena","RotationEntries":[
{"AssetId":"p1","VersionId":"v1","Metadata":{"Weight":2.5}},{"AssetId":"","VersionId":"x"}]}`))
	c := clientSur(srv)
	cfg, err := c.GetPlaylistConfig(context.Background(), authTest, "pl", "ver")
	if err != nil || cfg.PublicName != "Ranked Arena" || len(cfg.Entries) != 1 || cfg.Entries[0].Weight != 2.5 {
		t.Fatalf("cfg = %+v, %v", cfg, err)
	}
	if u, _ := cap.derniere(t); u.Path != "/hi/playlists/pl/versions/ver" {
		t.Errorf("URL = %s", u)
	}
	if _, err := c.GetPlaylistConfig(context.Background(), authTest, "pl", ""); err == nil {
		t.Error("version vide : erreur attendue")
	}
	repli, err := ParsePlaylistConfig([]byte(`{"PublicName":{"value":"Quick Play"},"CustomData":{"PlaylistEntries":[{"MapModePairAssetId":"mm1"}]}}`))
	if err != nil || repli.PublicName != "Quick Play" || len(repli.Entries) != 1 || repli.Entries[0].MapModePairAssetID != "mm1" {
		t.Errorf("repli = %+v, %v", repli, err)
	}
}
