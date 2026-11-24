package replayview_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/service/replayview"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// Les documents de rejeu des fixtures du web (`apps/web/src/features/match-replay/test/fixtures/go/`,
// un par build du jeu) servent de matière réelle à l'encodage en colonnes. Ce fichier prouve deux
// choses :
//
//   - l'aller-retour exact de chacun d'eux (toutes les fixtures, un document à la fois) ;
//   - la FRAÎCHEUR des fixtures en colonnes que le web décode (`fixtures/go_columns/`) : elles sont
//     écrites par `EncodeColumns`, jamais à la main.
//
// Régénérer après un changement de l'encodeur ou des fixtures de rejeu :
//
//	LEVELUP_REGEN_COLUMNS_FIXTURES=1 go test ./internal/service/replayview/ -run TestColumnsFixturesMatchCommitted

const (
	webFixturesRel          = "apps/web/src/features/match-replay/test/fixtures"
	envRegenColumnsFixtures = "LEVELUP_REGEN_COLUMNS_FIXTURES"
)

// webFixturesRoot rend le dossier des fixtures du web, sous la racine du dépôt.
func webFixturesRoot(t *testing.T) string {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, filepath.FromSlash(webFixturesRel))
}

// columnsFixtureFilms : les films dont la forme en colonnes est publiée pour le web. `bcb6d393` est
// le plus léger ; `000d5950` est le seul à porter toutes ses positions (la densité qui justifie
// les différences).
var columnsFixtureFilms = []string{"bcb6d393", "000d5950"}

func readGzipFile(t *testing.T, path string) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("fixture illisible : %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("%s : %v", path, err)
	}
	defer zr.Close()
	return readAllString(t, zr)
}

func webFixtureFiles(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(webFixturesRoot(t), "go", "manifest.json"))
	if err != nil {
		t.Fatalf("manifeste des fixtures du web illisible : %v", err)
	}
	var m struct {
		Fixtures []struct{ File, Film string }
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range m.Fixtures {
		out[f.Film] = filepath.Join(webFixturesRoot(t), "go", f.File)
	}
	if len(out) == 0 {
		t.Fatal("manifeste des fixtures du web vide")
	}
	return out
}

func TestEncodeColumns_FixturesDuWeb(t *testing.T) {
	for film, path := range webFixtureFiles(t) {
		body := readGzipFile(t, path)
		enc, err := replayview.EncodeColumns(body)
		if err != nil {
			t.Fatalf("%s : %v", film, err)
		}
		if len(enc) >= len(body) {
			t.Fatalf("%s : l'encodage en colonnes ne réduit rien (%d -> %d octets)", film, len(body), len(enc))
		}
		want := parseOrdered(t, body)
		got := decodeColumnsDoc(t, parseOrdered(t, enc))
		if p, ok := equalOrdered(want, got, "$"); !ok {
			t.Fatalf("%s : aller-retour différent en %s", film, p)
		}
	}
}

func columnsFixturePath(t *testing.T, film string) string {
	t.Helper()
	return filepath.Join(webFixturesRoot(t), "go_columns", "columns_"+film+".json.gz")
}

func gzipBytes(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := zw.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// calquesNonServis rend les clés JSON de premier niveau que le document stocké porte et que l'API ne sert pas
// (`testutil.ReplayChampsNonServis`, entrées `ReplayDocument.<Champ>`).
func calquesNonServis(t *testing.T) map[string]bool {
	t.Helper()
	exclus := map[string]bool{}
	stocke := reflect.TypeOf(replay.ReplayDocument{})
	for cle := range testutil.ReplayChampsNonServis {
		champ, ok := strings.CutPrefix(cle, "ReplayDocument.")
		if !ok {
			continue
		}
		f, trouve := stocke.FieldByName(champ)
		if !trouve {
			t.Fatalf("%s : champ inconnu du document stocké", cle)
		}
		exclus[strings.Split(f.Tag.Get("json"), ",")[0]] = true
	}
	return exclus
}

// TestCalquesNonServisDuWebSontCeuxDeGo : la liste que le chargeur de fixtures du web retire
// (`STORED_ONLY_LAYERS`, `goFixtures.ts`) est celle des calques que la décision de non-publication écrit côté Go. Deux
// listes divergent ; celle-ci les tient égales.
func TestCalquesNonServisDuWebSontCeuxDeGo(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(webFixturesRel), "..", "goFixtures.ts"))
	if err != nil {
		t.Fatal(err)
	}
	bloc := regexp.MustCompile(`(?s)STORED_ONLY_LAYERS: readonly string\[\] = \[(.*?)\]`).FindSubmatch(src)
	if bloc == nil {
		t.Fatal("STORED_ONLY_LAYERS introuvable dans goFixtures.ts")
	}
	web := map[string]bool{}
	for _, m := range regexp.MustCompile(`'([A-Za-z]+)'`).FindAllSubmatch(bloc[1], -1) {
		web[string(m[1])] = true
	}
	if !reflect.DeepEqual(web, calquesNonServis(t)) {
		t.Errorf("STORED_ONLY_LAYERS (web) = %v, calques non servis côté Go = %v", web, calquesNonServis(t))
	}
}

// sansCalquesNonServis rend le document tel que le serveur l'enverrait au web : les calques de premier niveau que
// `testutil.ReplayChampsNonServis` exclut (`ReplayDocument.<Champ>`) en sont retires, l'ordre des autres cles gardé.
// Les fixtures sont les documents du décodeur, avec tout ce qu'il écrit ; la racine du contrat du web est stricte.
func sansCalquesNonServis(t *testing.T, body []byte) []byte {
	t.Helper()
	exclus := calquesNonServis(t)
	dec := json.NewDecoder(bytes.NewReader(body))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		t.Fatalf("le document ne s'ouvre pas par un objet : %v %v", tok, err)
	}
	var out bytes.Buffer
	out.WriteByte('{')
	first := true
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			t.Fatal(err)
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			t.Fatal(err)
		}
		key := keyTok.(string)
		if exclus[key] {
			continue
		}
		if !first {
			out.WriteByte(',')
		}
		first = false
		k, _ := json.Marshal(key)
		out.Write(k)
		out.WriteByte(':')
		out.Write(raw)
	}
	out.WriteByte('}')
	return out.Bytes()
}

func TestColumnsFixturesMatchCommitted(t *testing.T) {
	files := webFixtureFiles(t)
	regen := os.Getenv(envRegenColumnsFixtures) == "1"
	for _, film := range columnsFixtureFilms {
		src, ok := files[film]
		if !ok {
			t.Fatalf("le film %s n'est plus dans les fixtures du web : mettre à jour columnsFixtureFilms", film)
		}
		enc, err := replayview.EncodeColumns(sansCalquesNonServis(t, readGzipFile(t, src)))
		if err != nil {
			t.Fatalf("%s : %v", film, err)
		}
		dst := columnsFixturePath(t, film)
		if regen {
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dst, gzipBytes(t, enc), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if committed := readGzipFile(t, dst); !bytes.Equal(committed, enc) {
			t.Fatalf("%s : la fixture en colonnes n'est plus celle que produit l'encodeur ; régénérer avec %s=1 (voir l'en-tête du fichier)", film, envRegenColumnsFixtures)
		}
	}
}
