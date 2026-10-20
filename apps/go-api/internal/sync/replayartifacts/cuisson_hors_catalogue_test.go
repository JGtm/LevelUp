package replayartifacts

// cuisson_hors_catalogue_test.go — une carte hors catalogue de bornes bloque TOUS ses films : elle
// se dit en WARN, une fois par carte et par cycle, avec le nombre de films concernés ; chaque film,
// lui, reste en DEBUG (un WARN par film noierait le journal).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
)

// lignesDuJournal : les lignes JSON du tampon, décodées.
func lignesDuJournal(t *testing.T, brut string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, ligne := range strings.Split(strings.TrimSpace(brut), "\n") {
		if ligne == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(ligne), &rec); err != nil {
			t.Fatalf("ligne de journal illisible %q : %v", ligne, err)
		}
		out = append(out, rec)
	}
	return out
}

func TestBuildAll_CarteHorsCatalogue_UnWarnParCarteEtParCycle(t *testing.T) {
	buf := capturerJournal(t)
	horsCatalogue := func(noms []string) error {
		return fmt.Errorf("%w (candidats: %v)", replaybuild.ErrMapNotInCatalog, noms)
	}
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
		Fetcher: &fetcherFilms{}, Budget: time.Minute, Gamertag: "JGtm",
		BuildOne: func(_ context.Context, req BuildOneRequest) (BuildOneResult, error) {
			if req.MatchID == "0000d0a1" {
				return BuildOneResult{}, errors.New("echec ordinaire")
			}
			return BuildOneResult{}, horsCatalogue(req.MapNames)
		},
	}
	b := buildAll(context.Background(), d, []buildWork{
		{matchID: "0000d001", mapNames: []string{"Detachment"}},
		{matchID: "0000f001", mapNames: []string{"Forge Arena"}},
		{matchID: "0000d002", mapNames: []string{"Detachment"}},
		{matchID: "0000d0a1", mapNames: []string{"Streets"}},
		{matchID: "0000d003", mapNames: []string{"Detachment"}},
	})

	if b.echecs != 5 {
		t.Errorf("echecs = %d, attendu 5 (une carte hors catalogue reste un échec compté)", b.echecs)
	}
	parCarte := map[string]float64{}
	warnsParFilm := 0
	for _, rec := range lignesDuJournal(t, buf.String()) {
		msg, _ := rec["msg"].(string)
		if rec["level"] == "WARN" && strings.Contains(msg, "carte hors catalogue de bornes") {
			carte, _ := rec["carte"].(string)
			if _, deja := parCarte[carte]; deja {
				t.Errorf("carte %q signalée deux fois dans le même cycle", carte)
			}
			parCarte[carte], _ = rec["films"].(float64)
		}
		if rec["level"] == "WARN" && strings.Contains(msg, "artefact rejeu non construit") {
			warnsParFilm++
		}
	}
	attendu := map[string]float64{"[Detachment]": 3, "[Forge Arena]": 1}
	if fmt.Sprint(parCarte) != fmt.Sprint(attendu) {
		t.Errorf("WARN par carte = %v, attendu %v", parCarte, attendu)
	}
	if warnsParFilm != 1 {
		t.Errorf("%d WARN par film, attendu 1 (l'échec ordinaire seul ; les films hors catalogue restent en DEBUG)", warnsParFilm)
	}
}

func TestBuildAll_SansCarteHorsCatalogue_AucunWarnDeCarte(t *testing.T) {
	buf := capturerJournal(t)
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
		Fetcher: &fetcherFilms{}, Budget: time.Minute,
		BuildOne: func(context.Context, BuildOneRequest) (BuildOneResult, error) {
			return BuildOneResult{}, errors.New("echec ordinaire")
		},
	}
	buildAll(context.Background(), d, []buildWork{{matchID: "0000a001", mapNames: []string{"Streets"}}})
	if aDit(t, buf, "WARN", "carte hors catalogue de bornes") {
		t.Error("aucune carte hors catalogue : aucun WARN de carte attendu")
	}
}
