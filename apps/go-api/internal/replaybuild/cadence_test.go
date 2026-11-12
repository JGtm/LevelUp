package replaybuild

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// digestAuPas : un artefact au schéma courant, décodé par les révisions courantes, au pas donné (0 = absent).
func digestAuPas(pas int) Digest {
	layers := map[string]string{}
	for fam, rev := range replay.RevisionsCourantesDesCouches() {
		layers[fam] = rev
	}
	return Digest{SchemaVersion: replay.SchemaVersion, FrameIntervalMS: pas, Layers: layers}
}

// TestVerdict_LePasEntreDansLeCritere : au schéma courant, un rejeu à 100 ms est « à recuire » quand la
// cadence courante est 33 ms, à jour quand elle est 100 ms ; un document sans pas se lit à 100 ms.
func TestVerdict_LePasEntreDansLeCritere(t *testing.T) {
	cas := []struct {
		nom       string
		courant   int
		pas       int
		faits     bool
		attendu   Verdict
		aJourVrai bool
	}{
		{"100 ms, cadence courante 33, faits présents", 33, 100, true, VerdictRepublier, false},
		{"100 ms, cadence courante 33, faits absents", 33, 100, false, VerdictRedecoder, false},
		{"100 ms, cadence courante 100", 100, 100, true, VerdictAJour, true},
		{"33 ms, cadence courante 33", 33, 33, true, VerdictAJour, true},
		{"33 ms, cadence courante 100 (retour en arrière)", 100, 33, true, VerdictRepublier, false},
		{"pas absent = 100 ms, cadence courante 100", 100, 0, true, VerdictAJour, true},
		{"pas absent = 100 ms, cadence courante 33", 33, 0, true, VerdictRepublier, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			defer ForceCookFrameIntervalMS(c.courant)()
			d := digestAuPas(c.pas)
			if got := d.Verdict(c.faits); got != c.attendu {
				t.Fatalf("verdict = %q, attendu %q", got, c.attendu)
			}
			if d.UpToDate() != c.aJourVrai {
				t.Fatalf("UpToDate = %v, attendu %v", d.UpToDate(), c.aJourVrai)
			}
		})
	}
}

// TestMeta_PorteLePasEtUneMetaSansPasSeLitA100ms : l'en-tête écrit avec l'artefact porte son pas, sans ouvrir
// l'artefact ; une méta d'avant le champ décrit une grille de 100 ms.
func TestMeta_PorteLePasEtUneMetaSansPasSeLitA100ms(t *testing.T) {
	defer ForceCookFrameIntervalMS(33)()
	path := filepath.Join(t.TempDir(), "000d5950.json")
	if _, err := StoreArtifactAt(context.Background(), path, title.DefaultSlug, "000d5950", docJSON(t, "000d5950", true)); err != nil {
		t.Fatal(err)
	}
	h, ok := ArtifactHeader(context.Background(), path)
	if !ok || h.FrameIntervalMS != 33 {
		t.Fatalf("en-tête = %+v, %v : le pas écrit doit être 33", h, ok)
	}

	// La même méta, d'avant le champ : on retire `frameIntervalMs`, la taille et la date restent celles de l'artefact.
	raw, err := os.ReadFile(MetaPath(path))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	delete(m, "frameIntervalMs")
	sans, _ := json.Marshal(m)
	if err := os.WriteFile(MetaPath(path), sans, 0o644); err != nil {
		t.Fatal(err)
	}
	h, ok = ArtifactHeader(context.Background(), path)
	if !ok || h.FrameIntervalMS != 0 {
		t.Fatalf("en-tête sans pas = %+v, %v", h, ok)
	}
	if FrameIntervalOrHistorical(h.FrameIntervalMS) != 100 || h.UpToDate() {
		t.Fatalf("une méta sans pas se lit à 100 ms, donc périmée face à 33 ms (UpToDate = %v)", h.UpToDate())
	}
}

// TestValidateArtifact_RefuseUnPasQuiNEstPasLeCourant : le dépôt d'un ouvrier d'une autre cadence est refusé,
// comme celui d'un autre schéma.
func TestValidateArtifact_RefuseUnPasQuiNEstPasLeCourant(t *testing.T) {
	blob := func(pas int) []byte {
		doc := replay.ReplayDocument{
			SchemaVersion: replay.SchemaVersion, MatchID: "000d5950", TitleSlug: title.DefaultSlug,
			FrameIntervalMS: pas, Tracks: []replay.Track{{XUID: "2533491259474651"}},
		}
		b, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	defer ForceCookFrameIntervalMS(33)()
	if err := validateArtifact(title.DefaultSlug, "000d5950", blob(33)); err != nil {
		t.Fatalf("au pas courant : %v", err)
	}
	for _, pas := range []int{100, 0} {
		if err := validateArtifact(title.DefaultSlug, "000d5950", blob(pas)); err == nil {
			t.Fatalf("un document à %d ms (0 = absent, lu à 100) doit être refusé face à 33 ms", pas)
		}
	}
}

// TestCuissonEtCritereLisentLeMemePas : le pas que la cuisson passe au décodeur est celui que le critère exige.
// Si l'un des deux se met à lire autre chose (le défaut du décodeur, une valeur écrite en dur), ce test échoue.
func TestCuissonEtCritereLisentLeMemePas(t *testing.T) {
	options := func(b *Builder) replay.Options {
		t.Helper()
		b.geometryOverride = []replay.MapObject{}
		b.structures = map[string][]replay.Surface{}
		b.energieTried = true
		return b.buildReplayOptions(context.Background(), decfilm.MapQuantEntry{}, port.MatchFacts{}, entreesCatalogue{}, &filmStats{})
	}
	for _, courant := range []int{33, 100, 50} {
		restore := ForceCookFrameIntervalMS(courant)
		got := options(&Builder{}).FrameIntervalMS
		restore()
		if got != courant {
			t.Fatalf("la cuisson passe %d ms au décodeur, le critère exige %d ms", got, courant)
		}
	}
	if got := options((&Builder{}).WithFrameInterval(77)).FrameIntervalMS; got != 77 {
		t.Fatalf("un réglage explicite (outil de recherche) gagne : %d", got)
	}
	// Sans forçage, c'est la valeur courante de LevelUp, dont on ne présume rien ici sinon qu'elle est celle du critère.
	if got, want := options(&Builder{}).FrameIntervalMS, CookFrameIntervalMS(); got != want {
		t.Fatalf("pas de cuisson = %d, critère = %d", got, want)
	}
}

// TestAucuneLectureDuPasDuDecodeurHorsDeCadenceGo : le défaut de cadence du décodeur n'est lu qu'au point unique
// (cadence.go) ; sinon la cuisson ou le critère pourrait diverger sans qu'aucun test ne le voie.
func TestAucuneLectureDuPasDuDecodeurHorsDeCadenceGo(t *testing.T) {
	racine, err := filepath.Abs("../..") // apps/go-api
	if err != nil {
		t.Fatal(err)
	}
	motif := regexp.MustCompile(`\bDefaultFrameIntervalMS\b`)
	var hors []string
	err = filepath.WalkDir(racine, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") || filepath.Base(p) == "cadence.go" {
			return nil
		}
		raw, rerr := os.ReadFile(p) //nolint:gosec // parcours du depot par un test
		if rerr != nil {
			return rerr
		}
		for _, ligne := range bytes.Split(raw, []byte("\n")) {
			code := strings.TrimSpace(string(ligne))
			if strings.HasPrefix(code, "//") {
				continue
			}
			if motif.MatchString(code) {
				hors = append(hors, p)
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hors) > 0 {
		t.Fatalf("`replay.DefaultFrameIntervalMS` lu hors de replaybuild/cadence.go : %v — passer par CookFrameIntervalMS", hors)
	}
}
