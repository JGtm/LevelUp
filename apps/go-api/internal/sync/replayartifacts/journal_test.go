package replayartifacts

// journal_test.go — L ETAPE 1.58 NE SORT PLUS SANS LE DIRE.
//
// Chaque cas ci-dessous correspond a une sortie qui, avant ce lot, s ecrivait exactement
// comme « l etape n a jamais tourne » : rien du tout. Mesure du 2026-09-01 : 1 artefact sur
// 222 matchs et zero ligne « rejeu 2D » au journal de synchronisation.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/sync/haloclient"
)

// capturerJournal redirige le logger par defaut vers un tampon JSON pour la duree du test
// (motif de sync/citations_terminal_state_test.go).
func capturerJournal(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// aDit dit si le tampon porte une ligne du niveau donne dont le message contient `motif`.
func aDit(t *testing.T, buf *bytes.Buffer, niveau, motif string) bool {
	t.Helper()
	for _, ligne := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if ligne == "" {
			continue
		}
		var rec struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
		}
		if err := json.Unmarshal([]byte(ligne), &rec); err != nil {
			continue
		}
		if rec.Level == niveau && strings.Contains(rec.Msg, motif) {
			return true
		}
	}
	return false
}

// TestArmee_ChaqueRefusSeDit : les trois refus de `armee`, chacun avec son niveau.
func TestArmee_ChaqueRefusSeDit(t *testing.T) {
	lecture := func(context.Context, string, func(*sql.DB)) {}
	cas := []struct {
		nom    string
		deps   Deps
		niveau string
		motif  string
	}{
		{"placement eteint", Deps{Placement: replaybuild.PlacementOff, WithRead: lecture},
			"DEBUG", "rejeu 2D éteint"},
		{"aucun segment de lecture", Deps{Placement: replaybuild.PlacementLocal},
			"WARN", "aucun segment de lecture câblé"},
		{"client sans chunks", Deps{Placement: replaybuild.PlacementLocal, WithRead: lecture},
			"DEBUG", "sans client film"},
	}
	for _, c := range cas {
		buf := capturerJournal(t)
		if armee(context.Background(), c.deps) {
			t.Errorf("%s : l etape se croit armee", c.nom)
		}
		if !aDit(t, buf, c.niveau, c.motif) {
			t.Errorf("%s : aucune ligne %s contenant %q — le refus est muet.\nJournal :\n%s",
				c.nom, c.niveau, c.motif, buf.String())
		}
	}
}

// TestRun_SelectionVide_LeDitEtPublieLesCompteurs — LE CAS QUI S ECRIVAIT « RIEN ».
//
// La fenetre de retention ecarte tout : avant ce lot, `Run` sortait sans un mot ET sans
// publier un compteur, donc indistinguable d une etape non cablee.
func TestRun_SelectionVide_LeDitEtPublieLesCompteurs(t *testing.T) {
	buf := capturerJournal(t)
	avant := observability.LoadCounterT("", CompteurCycles)
	// RepoRoot/TitleSlug RENSEIGNES : depuis le 2026-09-05, Run passe d'abord la porte de
	// production `film.replay_artifact` (capability.go) — un titre non resolu la ferme, et
	// ce test-ci parle du cas OU L'ETAPE TOURNE.
	Run(context.Background(), Deps{
		RepoRoot:  racineDepot(t),
		TitleSlug: titlePkg.DefaultSlug,
		Placement: replaybuild.PlacementLocal,
		Fetcher:   fetcherMuet{},
		WithRead:  func(context.Context, string, func(*sql.DB)) {}, // ne rend aucun travail
	}, []string{"m1"})

	if got := observability.LoadCounterT("", CompteurCycles); got != avant+1 {
		t.Errorf("%s = %d, attendu %d : le compteur de cycles doit prouver que l etape a tourne",
			CompteurCycles, got, avant+1)
	}
	if !aDit(t, buf, "DEBUG", "aucun match à traiter") {
		t.Errorf("selection vide : aucune trace.\nJournal :\n%s", buf.String())
	}
}

// TestPublierBilan_ParleMemeQuandRienNEstConstruit : le resume etait conditionne a
// `built > 0 || filmsSaved > 0`. Un cycle ou les cinq films sont expires — le cas le plus
// utile a voir — ne laissait donc aucune trace.
func TestPublierBilan_ParleMemeQuandRienNEstConstruit(t *testing.T) {
	buf := capturerJournal(t)
	publierBilan(context.Background(), Deps{Gamertag: "GT"}, bilanCuisson{sansFilm: 5}, 5)
	if !aDit(t, buf, "INFO", "post-sync: rejeu 2D") {
		t.Errorf("bilan vide : aucune ligne INFO.\nJournal :\n%s", buf.String())
	}
}

// TestSignalerClientSansChunks_NiveauSelonPlacement : un WARN la ou la capacite est
// indispensable, un DEBUG la ou elle ne sert a rien — un avertissement par cycle sur le
// chemin ouvrier finirait par masquer les vrais.
func TestSignalerClientSansChunks_NiveauSelonPlacement(t *testing.T) {
	buf := capturerJournal(t)
	avant := observability.LoadCounterT("", CompteurClientSansChunks)
	SignalerClientSansChunks(context.Background(), replaybuild.PlacementLocal, "GT", "*sync.mockClient")
	if !aDit(t, buf, "WARN", "ne porte pas GetFilmChunks") {
		t.Errorf("placement local : le defaut de cablage doit etre un WARN.\nJournal :\n%s", buf.String())
	}
	if got := observability.LoadCounterT("", CompteurClientSansChunks); got != avant+1 {
		t.Errorf("%s = %d, attendu %d", CompteurClientSansChunks, got, avant+1)
	}

	buf2 := capturerJournal(t)
	SignalerClientSansChunks(context.Background(), replaybuild.PlacementWorker, "GT", "*sync.mockClient")
	if aDit(t, buf2, "WARN", "ne porte pas GetFilmChunks") {
		t.Error("placement worker : mettre en file ne telecharge aucun film, un WARN par cycle y serait du bruit")
	}
	if !aDit(t, buf2, "DEBUG", "ne porte pas GetFilmChunks") {
		t.Errorf("placement worker : le fait reste tracable en DEBUG.\nJournal :\n%s", buf2.String())
	}
}

// fetcherMuet : un ChunksFetcher qui ne rend jamais de film. Il n est la que pour franchir la
// garde « pas de client film » sans toucher le reseau.
type fetcherMuet struct{}

func (fetcherMuet) GetFilmChunks(context.Context, string) ([]haloclient.FilmChunk, bool, error) {
	return nil, false, nil
}
