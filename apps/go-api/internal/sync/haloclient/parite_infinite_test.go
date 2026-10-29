// Package haloclient — parite_infinite_test.go : parité entre les DTO et valeurs de ce paquet et
// leur COPIE dans la brique `internal/haloapi/infinite` (plan D3, lot 3). Le test vit ici parce
// que c'est le seul paquet qui voit les deux côtés (les structures d'origine ne sont pas exportées).
// Il part avec ce paquet au lot 4 : à partir de là, la brique est la seule source et les tests de
// conversion de l'adaptateur et de `sync/filmfetch` prennent le relais.
package haloclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/haloapi/infinite"
)

// formeJSON rend la forme d'un type telle que `encoding/json` la lit : noms de champs, étiquettes et
// formes des champs, récursivement, SANS les noms de types. Un type à `UnmarshalJSON` est marqué.
// (Seconde copie, avec `platform/halo/parite_infinite_test.go` ; les deux partent au lot 4.)
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

// TestPariteInfinite_FormesDesDTO : chaque structure décodée ou rendue ici a la même forme que sa
// copie.
func TestPariteInfinite_FormesDesDTO(t *testing.T) {
	paires := []struct {
		nom         string
		ici, brique any
	}{
		{"matchSkillResponse", matchSkillResponse{}, infinite.MatchSkillDTO{}},
		{"playerCSRResponse", playerCSRResponse{}, infinite.PlayerCSRsDTO{}},
		{"playlistCSRResponse", playlistCSRResponse{}, infinite.PlaylistCSRDTO{}},
		{"csrRankRaw", csrRankRaw{}, infinite.CSRRankDTO{}},
		{"filmManifest", filmManifest{}, infinite.FilmManifestDTO{}},
		{"filmChunk", filmChunk{}, infinite.FilmChunkMetaDTO{}},
		{"FilmChunkRef", FilmChunkRef{}, infinite.FilmChunkRef{}},
		{"playlistConfigRaw", playlistConfigRaw{}, infinite.PlaylistConfigDTO{}},
		{"PlaylistConfig", PlaylistConfig{}, infinite.PlaylistConfig{}},
		{"customizationAppearance", customizationAppearance{}, infinite.CustomizationAppearanceDTO{}},
		{"AppearanceInputs", AppearanceInputs{}, infinite.AppearanceInputs{}},
		{"emblemMappingEntry", emblemMappingEntry{}, infinite.EmblemMappingEntry{}},
		{"BlobHTTPError", BlobHTTPError{}, infinite.BlobHTTPError{}},
		{"ErrChunkIncomplet", ErrChunkIncomplet{}, infinite.ErrChunkIncomplet{}},
	}
	for _, p := range paires {
		ici, brique := formeJSON(reflect.TypeOf(p.ici)), formeJSON(reflect.TypeOf(p.brique))
		if ici != brique {
			t.Errorf("%s : formes différentes\n  ici    : %s\n  brique : %s", p.nom, ici, brique)
		}
	}
	// HTTPError : la brique ajoute Body (corps de la réponse en erreur) ; les quatre champs d'ici
	// sont repris à l'identique, en tête et dans l'ordre.
	ici, brique := reflect.TypeOf(HTTPError{}), reflect.TypeOf(infinite.HTTPError{})
	for i := range ici.NumField() {
		if a, b := ici.Field(i), brique.Field(i); a.Name != b.Name || a.Type != b.Type {
			t.Errorf("HTTPError champ %d : ici %s %s, brique %s %s", i, a.Name, a.Type, b.Name, b.Type)
		}
	}
}

// TestPariteInfinite_Valeurs : adresses et politique de reprise identiques.
func TestPariteInfinite_Valeurs(t *testing.T) {
	e := infinite.DefaultEndpoints()
	hotes := map[infinite.Cle]string{
		infinite.CleStats:     haloStatsHost,
		infinite.CleGameCMS:   haloGameCMSHost,
		infinite.CleEconomy:   defaultEconomyHostSync,
		infinite.CleSkill:     haloSkillHost,
		infinite.CleUGCFilm:   haloUGCHost,
		infinite.CleNameplate: nameplateHost,
	}
	for k, v := range hotes {
		if e.Hote(k) != v {
			t.Errorf("hôte %s : brique %q, ici %q", k, e.Hote(k), v)
		}
	}
	r := infinite.RetryHaloClient()
	if r.Essais != maxRetries || r.DelaiBase != retryBaseDelay || r.Plafond != backoffCeiling {
		t.Errorf("RetryHaloClient = %+v", r)
	}
	if infinite.FilmChunkParallelism != filmChunkParallelism || infinite.DelaiNameplate != nameplateResolverTimeout ||
		infinite.MatchCountMax != matchCountMax {
		t.Error("constantes de parallélisme, d'échéance ou de page différentes")
	}
}

// TestPariteInfinite_FonctionsPures : mêmes sorties pour les fonctions copiées.
func TestPariteInfinite_FonctionsPures(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	for _, h := range []string{"", "5", "-1", "99999", "x", now.Add(time.Minute).Format(http.TimeFormat)} {
		if a, b := parseRetryAfter(h, now), infinite.ParseRetryAfter(h, now); a != b {
			t.Errorf("Retry-After %q : ici %v, brique %v", h, a, b)
		}
	}
	for _, c := range [][2]string{{"p", "/x"}, {"p/", "x"}, {"p", ""}, {"", "x"}} {
		if a, b := buildChunkURL(c[0], c[1]), infinite.BuildChunkURL(c[0], c[1]); a != b {
			t.Errorf("buildChunkURL%v : ici %q, brique %q", c, a, b)
		}
	}
	xuids := []string{" 1 ", "", "bid(2.0)", "xuid(3)"}
	if a, b := filterHumanXUIDs(xuids), infinite.FilterHumanXUIDs(xuids); !reflect.DeepEqual(a, b) {
		t.Errorf("filterHumanXUIDs : ici %v, brique %v", a, b)
	}
	for _, x := range []string{"xuid(4)", " xuid(5) ", "6", "xuid(7"} {
		if unwrapXUID(x) != infinite.UnwrapXUID(x) {
			t.Errorf("unwrapXUID(%q)", x)
		}
	}
	for _, p := range []string{"Inventory/Spartan/Emblems/a-b.json", "x/Spartan/Emblems/d/e.json", "autre"} {
		if extractEmblemStem(p) != infinite.ExtractEmblemStem(p) {
			t.Errorf("extractEmblemStem(%q)", p)
		}
	}
	for _, body := range []string{`{"AvailableConfigurations":[{"ConfigurationId":-1},{"ConfigurationId":9}]}`, `{}`, `{`} {
		c1, p1 := firstPositiveEmblemCfg([]byte(body))
		c2, p2 := infinite.FirstPositiveEmblemCfg([]byte(body))
		if c1 != c2 || p1 != p2 {
			t.Errorf("firstPositiveEmblemCfg(%s) : ici (%d,%v), brique (%d,%v)", body, c1, p1, c2, p2)
		}
	}
	playlist := `{"PublicName":{"value":"P"},"RotationEntries":[{"AssetId":"a","VersionId":"v","Metadata":{"Weight":1.5}}]}`
	pa, errA := parsePlaylistConfig([]byte(playlist))
	pb, errB := infinite.ParsePlaylistConfig([]byte(playlist))
	if errA != nil || errB != nil || formeEtValeur(t, pa) != formeEtValeur(t, pb) {
		t.Errorf("parsePlaylistConfig : ici %+v (%v), brique %+v (%v)", pa, errA, pb, errB)
	}
	apparence := `{"Appearance":{"ServiceTag":"T","Emblem":{"EmblemPath":"E","ConfigurationId":12},"Banner":{"Path":"B"}}}`
	ca, errA := parseCustomizationAppearance([]byte(apparence))
	cb, errB := infinite.DecodeCustomizationAppearance([]byte(apparence))
	if errA != nil || errB != nil || formeEtValeur(t, ca) != formeEtValeur(t, cb) {
		t.Errorf("parseCustomizationAppearance : ici %+v, brique %+v", ca, cb)
	}
}

// formeEtValeur sérialise une valeur (mêmes champs : mêmes octets si et seulement si mêmes valeurs).
func formeEtValeur(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal : %v", err)
	}
	return string(b)
}
