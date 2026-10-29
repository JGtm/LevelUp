// Package halo — parite_infinite_test.go : parité entre les DTO et valeurs de ce paquet et leur
// COPIE dans la brique `internal/haloapi/infinite` (plan D3, lot 3). Le test vit ici parce que
// c'est le seul paquet qui voit les deux côtés (les structures d'origine ne sont pas exportées).
// Il part avec ce paquet au lot 4 : à partir de là, la brique est la seule source et les tests de
// conversion de l'adaptateur prennent le relais.
package halo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/haloapi/infinite"
)

// formeJSON rend la forme d'un type telle que `encoding/json` la lit : noms de champs, étiquettes et
// formes des champs, récursivement, SANS les noms de types (un DTO nommé dans la brique est la même
// forme qu'une struct anonyme ici). Un type à `UnmarshalJSON` est marqué.
func formeJSON(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + formeJSON(t.Elem())
	case reflect.Slice:
		return "[]" + formeJSON(t.Elem())
	case reflect.Map:
		return "map[" + formeJSON(t.Key()) + "]" + formeJSON(t.Elem())
	case reflect.Struct:
		var b strings.Builder
		b.WriteString("{")
		for i := range t.NumField() {
			f := t.Field(i)
			fmt.Fprintf(&b, "%s %s `%s`; ", f.Name, formeJSON(f.Type), f.Tag)
		}
		b.WriteString("}")
		return b.String()
	}
	s := t.Kind().String()
	if reflect.PointerTo(t).Implements(reflect.TypeFor[json.Unmarshaler]()) {
		s += "+UnmarshalJSON"
	}
	return s
}

// TestPariteInfinite_FormesDesDTO : chaque structure décodée ici a la même forme JSON que sa copie.
func TestPariteInfinite_FormesDesDTO(t *testing.T) {
	paires := []struct {
		nom         string
		ici, brique any
	}{
		{"serviceRecordResponse", serviceRecordResponse{}, infinite.ServiceRecordDTO{}},
		{"privacyResponse", privacyResponse{}, infinite.PrivacyDTO{}},
		{"medalMetadataRaw", medalMetadataRaw{}, infinite.MedalsMetadataDTO{}},
		{"seasonCalendarRaw", seasonCalendarRaw{}, infinite.SeasonCalendarDTO{}},
		{"csrSeasonRaw", csrSeasonRaw{}, infinite.CSRSeasonCalendarDTO{}},
		{"battlePassTrack", battlePassTrack{}, infinite.BattlePassTrackDTO{}},
		{"challengeDeckRaw", challengeDeckRaw{}, infinite.ChallengeDeckDTO{}},
		{"challengeDeckItemRaw", challengeDeckItemRaw{}, infinite.ChallengeDeckItemDTO{}},
		{"DiscoveryAsset", DiscoveryAsset{}, infinite.DiscoveryAsset{}},
		{"MatchInfoAsset", MatchInfoAsset{}, infinite.MatchInfoAsset{}},
		{"waypointPageProps", waypointPageProps{}, infinite.LeaderboardPageDTO{}},
		{"waypointLBEntry", waypointLBEntry{}, infinite.LeaderboardEntryDTO{}},
		{"waypointSeasonRef", waypointSeasonRef{}, infinite.WaypointSeasonDTO{}},
		{"waypointPlaylistRef", waypointPlaylistRef{}, infinite.WaypointPlaylistDTO{}},
		{"WaypointRef", WaypointRef{}, infinite.WaypointRef{}},
	}
	for _, p := range paires {
		ici, brique := formeJSON(reflect.TypeOf(p.ici)), formeJSON(reflect.TypeOf(p.brique))
		if ici != brique {
			t.Errorf("%s : formes différentes\n  ici    : %s\n  brique : %s", p.nom, ici, brique)
		}
	}
}

// TestPariteInfinite_Valeurs : adresses, politiques et tables d'actifs identiques.
func TestPariteInfinite_Valeurs(t *testing.T) {
	e := infinite.DefaultEndpoints()
	hotes := map[infinite.Cle]string{
		infinite.CleEconomy:       defaultEconomyHost,
		infinite.CleChallenges:    defaultChallengesHost,
		infinite.CleGameCMS:       defaultGameCMSHost,
		infinite.CleDiscoveryUGC:  discoveryUGCHost,
		infinite.CleStatsSansPort: defaultStatsHost,
		infinite.CleWaypointWeb:   waypointLeaderboardHost,
	}
	for k, v := range hotes {
		if e.Hote(k) != v {
			t.Errorf("hôte %s : brique %q, ici %q", k, e.Hote(k), v)
		}
	}
	if p := infinite.RetryProvider(); p.Essais != providerMaxRetries || p.DelaiBase != providerRetryBase || p.Plafond != 10*time.Second {
		t.Errorf("RetryProvider = %+v", p)
	}
	if d := infinite.RetryDiscovery(); d.Essais != providerMaxRetries || d.DelaiBase != providerRetryBase {
		t.Errorf("RetryDiscovery = %+v", d)
	}
	for typ, seg := range AssetTypeToEndpoint {
		if s, ok := infinite.SegmentDAsset(infinite.AssetType(typ)); !ok || s != seg {
			t.Errorf("segment %s : brique %q, ici %q", typ, s, seg)
		}
	}
	for typ, cle := range AssetTypeToMatchInfoKey {
		if s, ok := infinite.CleMatchInfo(infinite.AssetType(typ)); !ok || s != cle {
			t.Errorf("clé MatchInfo %s : brique %q, ici %q", typ, s, cle)
		}
	}
	if infinite.ScraperUserAgent != scraperUserAgent {
		t.Error("agent du lecteur de classement différent")
	}
	if got := infinite.MedalsMetadataURL(e); got != defaultGameCMSHost+"/hi"+medalsMetadataPathSuffix {
		t.Errorf("MedalsMetadataURL = %s", got)
	}
	if got := infinite.SeasonCalendarURL(e); got != defaultGameCMSHost+"/hi"+seasonCalendarPathSuffix {
		t.Errorf("SeasonCalendarURL = %s", got)
	}
	if got := infinite.CSRSeasonCalendarURL(e); got != defaultGameCMSHost+"/hi"+csrSeasonCalendarPathSuffix {
		t.Errorf("CSRSeasonCalendarURL = %s", got)
	}
}

// TestPariteInfinite_FonctionsPures : mêmes sorties pour les fonctions copiées.
func TestPariteInfinite_FonctionsPures(t *testing.T) {
	classe := false
	for _, r := range []*bool{nil, &classe} {
		if a, b := buildSeasonServiceRecordURL("hi", "g t", "Seasons/S7.json", r),
			infinite.BuildSeasonServiceRecordURL(infinite.DefaultEndpoints(), "g t", "Seasons/S7.json", r); a != b {
			t.Errorf("URL service record : ici %s, brique %s", a, b)
		}
	}
	for _, raw := range []string{`"Nom"`, `{"value":"Nom"}`, `42`, ``} {
		if a, b := decodeLocalizedAssetName(json.RawMessage(raw)), infinite.DecodeLocalizedName(json.RawMessage(raw)); a != b {
			t.Errorf("nom %s : ici %q, brique %q", raw, a, b)
		}
	}
	for _, chemins := range [][]string{{"a.mvar", "x/hero.jpg", "y/thumbnail.webp"}, {"z/screenshot.PNG"}, {"a.txt"}, nil} {
		if a, b := buildAssetImageURL("https://p", chemins), infinite.BuildAssetImageURL("https://p", chemins); a != b {
			t.Errorf("image %v : ici %q, brique %q", chemins, a, b)
		}
	}
	body, err := os.ReadFile(filepath.Join("testdata", "leaderboard_sample.html"))
	if err != nil {
		t.Fatal(err)
	}
	ici, err := parseLeaderboardPage(body)
	if err != nil {
		t.Fatal(err)
	}
	brique, err := infinite.ParseLeaderboardPage(body)
	if err != nil {
		t.Fatal(err)
	}
	// Mêmes étiquettes : mêmes octets si et seulement si mêmes valeurs.
	a, errA := json.Marshal(ici)
	b, errB := json.Marshal(brique)
	if errA != nil || errB != nil {
		t.Fatalf("marshal : %v / %v", errA, errB)
	}
	if string(a) != string(b) {
		t.Error("la page du classement ne décode pas les mêmes valeurs")
	}
}
