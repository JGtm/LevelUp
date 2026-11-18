package replayartifacts

// coupdenvoi_test.go — la famille du coup d'envoi SANS BASE : ses dégradations, et la lecture du
// champ `t0FilmMs` par la chaîne réelle (`lireArtefacts`). Le report en base est éprouvé dans
// coupdenvoi_integration_test.go.

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replayartifact"
	"levelup/go-api/internal/sync/kickoff"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// ecrireFichier range un artefact de test (contenu quelconque) par la porte, sous le nom canonique `.json.gz` de `nom`.
func ecrireFichier(t *testing.T, nom, contenu string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), strings.TrimSuffix(nom, ".json")+titlePkg.ReplayArtifactExt)
	if _, _, err := replayartifact.Write(context.Background(), path, []byte(contenu)); err != nil {
		t.Fatalf("write %s: %v", nom, err)
	}
	return path
}

// Le champ se lit tel que le document le publie : absent et zéro se distinguent (pointeur), et un
// artefact illisible ne donne rien.
func TestLireArtefacts_CoupDEnvoiDuDocument(t *testing.T) {
	zero, mesure := int64(0), int64(26304)
	cas := map[string]struct {
		contenu string
		attendu *int64
		lu      bool
	}{
		"coup d envoi publie":        {`{"schemaVersion":36,"t0FilmMs":26304}`, &mesure, true},
		"coup d envoi a zero":        {`{"schemaVersion":36,"t0FilmMs":0}`, &zero, true},
		"champ absent":               {`{"schemaVersion":36,"matchId":"abc"}`, nil, true},
		"champ explicitement a null": {`{"t0FilmMs":null}`, nil, true},
		"artefact illisible":         {`{"schemaVersion":`, nil, false},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			lus := lireArtefacts(context.Background(), Deps{Gamertag: "t"},
				[]ArtefactRange{{MatchID: "m1", Path: ecrireFichier(t, "artefact.json", c.contenu)}})
			if (len(lus) == 1) != c.lu {
				t.Fatalf("%d artefact(s) lu(s), attendu lu=%v", len(lus), c.lu)
			}
			if !c.lu {
				return
			}
			got := lus[0].doc.T0FilmMs
			switch {
			case c.attendu == nil && got != nil:
				t.Fatalf("attendu absent, got %d", *got)
			case c.attendu != nil && (got == nil || *got != *c.attendu):
				t.Fatalf("t0FilmMs = %v, attendu %d", got, *c.attendu)
			}
		})
	}
}

// Sans segment de lecture ni writer câblés, le match échoue faute de writer (aucune marque du lot :
// le rattrapage le rejouera).
func TestCoupDEnvoi_SansLectureNiWriterLeMatchEchoue(t *testing.T) {
	d := Deps{TitleSlug: "halo_infinite"}
	b := &bilanDerivations{}
	ecrireCoupsDEnvoi(context.Background(), d, b, preparerCoupsDEnvoi(context.Background(), d, b, []artefactLu{lu0("m1")}))
	if !b.aEchoue("m1") || !b.sansWriter {
		t.Fatalf("echec=%v sansWriter=%v ; attendu les deux", b.aEchoue("m1"), b.sansWriter)
	}
}

// Un lot vide ou sans changement ne prend pas le writer partagé : c'est le régime stationnaire.
func TestCoupDEnvoi_RienAEcrireAucunWriter(t *testing.T) {
	acquis := 0
	d := Deps{AcquireWriter: func(context.Context) (*sql.DB, func(), error) {
		acquis++
		return nil, func() {}, nil
	}}
	b := &bilanDerivations{}
	ecrireCoupsDEnvoi(context.Background(), d, b, coupsDEnvoi{})
	ecrireCoupsDEnvoi(context.Background(), d, b, coupsDEnvoi{plans: []kickoff.Plan{{MatchID: "m1"}}})
	if acquis != 0 || b.aEchoue("m1") {
		t.Fatalf("writer=%d echec=%v ; attendu aucun writer, aucun echec", acquis, b.aEchoue("m1"))
	}
}

// Un plan à écrire sans writer câblé : échec faute de writer (aucune marque du lot).
func TestCoupDEnvoi_SansWriterLeMatchEchoue(t *testing.T) {
	b := &bilanDerivations{}
	ecrireCoupsDEnvoi(context.Background(), Deps{}, b, coupsDEnvoi{plans: []kickoff.Plan{{MatchID: "m1", BeforeSource: "api"}}})
	if !b.aEchoue("m1") || !b.sansWriter {
		t.Fatalf("echec=%v sansWriter=%v ; attendu les deux", b.aEchoue("m1"), b.sansWriter)
	}
}

// lu0 : un artefact lu sans coup d'envoi publié.
func lu0(matchID string) artefactLu {
	return artefactLu{matchID: matchID, doc: &replay.ReplayDocument{MatchID: matchID}}
}
