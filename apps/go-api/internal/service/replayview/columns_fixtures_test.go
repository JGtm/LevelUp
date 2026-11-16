package replayview_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/service/replayview"
	"levelup/go-api/internal/testutil"
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

func TestColumnsFixturesMatchCommitted(t *testing.T) {
	files := webFixtureFiles(t)
	regen := os.Getenv(envRegenColumnsFixtures) == "1"
	for _, film := range columnsFixtureFilms {
		src, ok := files[film]
		if !ok {
			t.Fatalf("le film %s n'est plus dans les fixtures du web : mettre à jour columnsFixtureFilms", film)
		}
		enc, err := replayview.EncodeColumns(readGzipFile(t, src))
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
