package replayartifacts

// derivations_marque_test.go — LA MARQUE DE DERIVATION NE MENT JAMAIS (constat C1, revue A-R1).
//
// # Le defaut que ces tests ferment
//
// `Deriver` posait la marque `<artefact>.derived.json` EN FIN DE PASSE, sans condition. Quand le
// writer shared etait indisponible — aucun writer cable sur le chemin (depot d'ouvrier sans
// provider) ou acquisition en echec (lease tenu, B-swap en cours) — les quatre familles
// journalisaient leur degradation, ecrivaient ZERO ligne... et la marque se posait quand meme.
// `DerivationsUpToDate` rendait alors `true` a jamais, et `candidatsDerivations` excluait ce
// match DEFINITIVEMENT : `real_start_time`, `match_usage_*`, `match_bomb_stats` et
// `match_player_positions` restaient vides, sans autre reprise que la suppression manuelle du
// fichier de marque.
//
// # La regle, en une phrase
//
// ON NE MARQUE QUE CE QU'ON A PU ECRIRE. Writer indisponible = aucune marque (le rattrapage
// rejouera tout le lot) ; writer disponible = une marque PAR MATCH dont aucune famille n'a
// echoue. « Rien a ecrire » reste une derivation JOUEE, et se marque : c'est le cas d'un titre
// sans les capabilities film et d'un document sans trajectoire ni coup d'envoi.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/replaybuild"
)

// artefactADeriver ecrit un artefact qui a QUELQUE CHOSE a deriver dans les quatre familles :
// un coup d'envoi mesure et une trajectoire. Sans cela, une passe sans writer n'aurait rien a
// ecrire et la marque serait legitime — le test ne mordrait plus.
func artefactADeriver(t *testing.T, dir, matchID string) string {
	t.Helper()
	t0 := int64(26_304)
	xuid := "111"
	doc := replay.ReplayDocument{
		SchemaVersion:   replay.SchemaVersion,
		MatchID:         matchID,
		FrameIntervalMS: 100,
		FrameCount:      1000,
		DurationMS:      100_000,
		T0FilmMs:        &t0,
		Roster:          []replay.RosterEntry{{XUID: xuid, FilmIndex: 0}},
		Tracks: []replay.Track{{
			Slot: 1, XUID: xuid, Team: -1, StartFrame: 0, EndFrame: 900,
			Points: []replay.Point{{T: 0, X: 1, Y: 2, Z: 3}},
		}},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal artefact: %v", err)
	}
	path := filepath.Join(dir, matchID+".json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write artefact: %v", err)
	}
	return path
}

// TestDeriver_WriterIndisponible_NeMarquePas — LE test du constat C1, dans ses deux
// sous-cas : aucun writer cable, et acquisition en erreur.
func TestDeriver_WriterIndisponible_NeMarquePas(t *testing.T) {
	cas := map[string]func(context.Context) (*sql.DB, func(), error){
		"aucun writer cable": nil,
		"acquisition en erreur": func(context.Context) (*sql.DB, func(), error) {
			return nil, nil, errors.New("lease shared tenu au-dela du delai")
		},
	}
	for nom, acquerir := range cas {
		t.Run(nom, func(t *testing.T) {
			dir := t.TempDir()
			chemin := artefactADeriver(t, dir, "marqueA")
			Deriver(context.Background(), DerivationsDeps{
				RepoRoot: racineDepot(t), TitleSlug: "halo_infinite", Gamertag: "testeur",
				AcquireWriter: acquerir,
			}, []ArtefactRange{{MatchID: "marqueA", Path: chemin}})

			if _, ok := replaybuild.ReadDerivationsMark(chemin); ok {
				t.Errorf("fichier de marque present alors qu'AUCUNE ligne n'a ete ecrite")
			}
			if replaybuild.DerivationsUpToDate(chemin) {
				t.Fatalf("marque posee sans ecriture : le match sort du rattrapage a jamais " +
					"(constat C1)")
			}
		})
	}
}

// TestDeriver_FamilleEnEchec_NeMarquePas — la granularite PAR MATCH, prouvee sans writer du
// tout : un titre dont les capabilities sont illisibles (racine sans `config/titles/`) fait
// ecarter le lot par les familles usage et Assaut. C'est un ECHEC, pas un silence — et un match
// en echec ne se marque pas, meme quand le segment d'ecriture, lui, allait bien.
func TestDeriver_FamilleEnEchec_NeMarquePas(t *testing.T) {
	dir := t.TempDir()
	// Ni coup d'envoi ni trajectoire : les deux familles qui ecrivent sans capability (T0 et
	// positions) n'ont rien a faire, et le writer n'est donc jamais sollicite.
	doc := replay.ReplayDocument{SchemaVersion: replay.SchemaVersion, MatchID: "marqueC", FrameIntervalMS: 100}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	chemin := filepath.Join(dir, "marqueC.json")
	if err := os.WriteFile(chemin, raw, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	Deriver(context.Background(), DerivationsDeps{
		RepoRoot: t.TempDir(), TitleSlug: "halo_infinite", Gamertag: "testeur",
	}, []ArtefactRange{{MatchID: "marqueC", Path: chemin}})

	if replaybuild.DerivationsUpToDate(chemin) {
		t.Fatalf("marque posee alors qu'une famille a ECARTE ce match (capabilities illisibles) : " +
			"il sortirait du rattrapage sans que ses derives existent")
	}
}

// TestDeriver_RienAEcrire_MarqueQuandMeme — la contre-epreuve, et elle compte autant : un titre
// sans capability film et un document sans coup d'envoi ni trajectoire n'ont RIEN a ecrire.
// C'est une derivation JOUEE ; la rejouer a chaque cycle serait du travail pur perte.
func TestDeriver_RienAEcrire_MarqueQuandMeme(t *testing.T) {
	dir := t.TempDir()
	doc := replay.ReplayDocument{SchemaVersion: replay.SchemaVersion, MatchID: "marqueB", FrameIntervalMS: 100}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	chemin := filepath.Join(dir, "marqueB.json")
	if err := os.WriteFile(chemin, raw, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	// halo_5 ne declare aucune clef `film.*` : les familles usage et Assaut se taisent
	// proprement, et le document ne porte ni t0FilmMs ni trajectoire.
	Deriver(context.Background(), DerivationsDeps{
		RepoRoot: racineDepot(t), TitleSlug: "halo_5", Gamertag: "testeur",
	}, []ArtefactRange{{MatchID: "marqueB", Path: chemin}})

	if !replaybuild.DerivationsUpToDate(chemin) {
		t.Fatalf("marque ABSENTE alors qu'il n'y avait rien a ecrire : le rattrapage rejouerait " +
			"ce match a chaque cycle, indefiniment")
	}
}
