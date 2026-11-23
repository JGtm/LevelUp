package replaybuild

// artifact_kickoff_test.go — la lecture du coup d'envoi par le fichier d'en-tete rend la valeur de la lecture complete,
// et n'ouvre plus l'artefact quand l'en-tete suffit.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replayartifact"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// kickoffTemoin : ce que rendait la lecture complete avant l'en-tete (lecture et decodage du document entier).
func kickoffTemoin(t *testing.T, path string) (*int64, error) {
	t.Helper()
	c, err := replayartifact.Read(path)
	if err != nil {
		return nil, err
	}
	var head struct {
		T0FilmMs *int64 `json:"t0FilmMs"`
	}
	if err := json.Unmarshal(c.JSON, &head); err != nil {
		return nil, err
	}
	return head.T0FilmMs, nil
}

func lecturesCompletes() int64 { return observability.LoadCounter(CompteurKickoffLecturesCompletes) }

func egal(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func docAvecCoupDEnvoi(t *testing.T, t0 *int64) []byte {
	t.Helper()
	blob, err := json.Marshal(replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: "000d5950", TitleSlug: title.DefaultSlug,
		FrameIntervalMS: CookFrameIntervalMS(), Tracks: []replay.Track{{XUID: "2533491259474651"}}, T0FilmMs: t0,
	})
	if err != nil {
		t.Fatal(err)
	}
	return blob
}

// lireMetaTest relit l'en-tete tel que range, sans exiger qu'il fasse foi.
func lireMetaTest(t *testing.T, path string) (artifactMeta, bool) {
	t.Helper()
	raw, err := os.ReadFile(MetaPath(path))
	if err != nil {
		return artifactMeta{}, false
	}
	var m artifactMeta
	if json.Unmarshal(raw, &m) != nil {
		return artifactMeta{}, false
	}
	return m, true
}

// Un artefact rangé par l'ecrivain (en-tete complet) : meme valeur que la lecture complete, aucune lecture complete,
// pour une valeur, un zero (le piege omitempty) et un document sans le champ.
func TestArtifactKickoff_EnteteDeLEcrivain_MemeValeurSansLireLArtefact(t *testing.T) {
	zero, valeur := int64(0), int64(26304)
	for nom, t0 := range map[string]*int64{"valeur": &valeur, "zero": &zero, "absent": nil} {
		t.Run(nom, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "000d5950.json.gz")
			if _, err := StoreArtifactAt(context.Background(), path, title.DefaultSlug, "000d5950", docAvecCoupDEnvoi(t, t0)); err != nil {
				t.Fatal(err)
			}
			avant := lecturesCompletes()
			got, found, err := ArtifactKickoff(context.Background(), path, true)
			temoin, errTemoin := kickoffTemoin(t, path)
			if err != nil || errTemoin != nil || !found || !egal(got, temoin) || !egal(got, t0) {
				t.Fatalf("got=%v found=%v err=%v, temoin=%v (%v), attendu %v", got, found, err, temoin, errTemoin, t0)
			}
			if lecturesCompletes() != avant {
				t.Fatal("l'en-tete suffit : l'artefact n'est pas lu en entier")
			}
		})
	}
}

// Un en-tete d'avant le champ, puis un artefact sans en-tete : une lecture complete, l'en-tete est complete ou ecrit, la
// seconde lecture n'ouvre plus l'artefact. Sans majEntete (simulation), rien n'est ecrit.
func TestArtifactKickoff_EnteteAbsentOuIncomplet_UneLectureCompletePuisPlus(t *testing.T) {
	ms := int64(26304)
	path := filepath.Join(t.TempDir(), "000d5950.json.gz")
	if _, err := StoreArtifactAt(context.Background(), path, title.DefaultSlug, "000d5950", docAvecCoupDEnvoi(t, &ms)); err != nil {
		t.Fatal(err)
	}
	m, ok := lireMetaTest(t, path)
	if !ok {
		t.Fatal("en-tete attendu")
	}
	m.Kickoff = nil // en-tete d'avant le champ
	ecrireMeta(context.Background(), path, m)

	avant := lecturesCompletes()
	got, found, err := ArtifactKickoff(context.Background(), path, false)
	if err != nil || !found || !egal(got, &ms) || lecturesCompletes() != avant+1 {
		t.Fatalf("simulation : got=%v found=%v err=%v lectures=%d", got, found, err, lecturesCompletes()-avant)
	}
	if m2, _ := lireMetaTest(t, path); m2.Kickoff != nil {
		t.Fatal("la simulation n'ecrit pas l'en-tete")
	}
	got, _, err = ArtifactKickoff(context.Background(), path, true)
	if err != nil || !egal(got, &ms) || lecturesCompletes() != avant+2 {
		t.Fatalf("application : got=%v err=%v lectures=%d", got, err, lecturesCompletes()-avant)
	}
	if m2, _ := lireMetaTest(t, path); m2.Kickoff == nil || m2.AppVersion != m.AppVersion || m2.Bytes != m.Bytes {
		t.Fatalf("en-tete complete sans perdre ses autres champs : %+v", m2)
	}
	if got, _, err = ArtifactKickoff(context.Background(), path, true); err != nil || !egal(got, &ms) || lecturesCompletes() != avant+2 {
		t.Fatal("seconde lecture : l'en-tete complete fait foi")
	}

	// Artefact sans aucun en-tete : lecture complete, en-tete ecrit.
	if err := os.Remove(MetaPath(path)); err != nil {
		t.Fatal(err)
	}
	if got, _, err = ArtifactKickoff(context.Background(), path, true); err != nil || !egal(got, &ms) || lecturesCompletes() != avant+3 {
		t.Fatalf("sans en-tete : got=%v err=%v", got, err)
	}
	if got, _, err = ArtifactKickoff(context.Background(), path, true); err != nil || !egal(got, &ms) || lecturesCompletes() != avant+3 {
		t.Fatal("en-tete ecrit par la lecture complete")
	}
}

// Une fixture reelle du depot (document sans `t0FilmMs`, schema anterieur au champ) : meme reponse que la lecture complete,
// par le repli puis par l'en-tete ecrit.
func TestArtifactKickoff_FixtureReelleSansLeChamp(t *testing.T) {
	brut, err := os.ReadFile(filepath.Join("..", "replayverite", "testdata", "bcb6d393.json.gz"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "bcb6d393.json.gz")
	if err := os.WriteFile(path, brut, 0o600); err != nil {
		t.Fatal(err)
	}
	temoin, err := kickoffTemoin(t, path)
	if err != nil || temoin != nil {
		t.Fatalf("temoin = %v, %v, attendu un document sans le champ", temoin, err)
	}
	for i := range 2 {
		got, found, err := ArtifactKickoff(context.Background(), path, true)
		if err != nil || !found || got != nil {
			t.Fatalf("lecture %d : got=%v found=%v err=%v", i, got, found, err)
		}
	}
}

// Absent : (nil, false, nil). Tronque ou non gzip : une erreur, comme la lecture complete, meme si l'en-tete de l'artefact
// entier subsiste.
func TestArtifactKickoff_AbsentTronqueIllisible(t *testing.T) {
	dir := t.TempDir()
	if got, found, err := ArtifactKickoff(context.Background(), filepath.Join(dir, "absent.json.gz"), true); got != nil || found || err != nil {
		t.Fatalf("absent : %v %v %v", got, found, err)
	}
	ms := int64(1000)
	path := filepath.Join(dir, "000d5950.json.gz")
	if _, err := StoreArtifactAt(context.Background(), path, title.DefaultSlug, "000d5950", docAvecCoupDEnvoi(t, &ms)); err != nil {
		t.Fatal(err)
	}
	entier, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, entier[:len(entier)/2], 0o600); err != nil {
		t.Fatal(err)
	}
	_, errTemoin := kickoffTemoin(t, path)
	got, found, err := ArtifactKickoff(context.Background(), path, true)
	if got != nil || !found || err == nil || errTemoin == nil {
		t.Fatalf("tronque : got=%v found=%v err=%v (temoin %v), attendu une erreur", got, found, err, errTemoin)
	}
	brut := filepath.Join(dir, "brut.json.gz")
	if err := os.WriteFile(brut, []byte("pas du gzip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := ArtifactKickoff(context.Background(), brut, true); !found || err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("illisible : found=%v err=%v", found, err)
	}
}
