package replayartifacts

// document_unique_test.go — GARDE-RAIL : UNE SEULE LECTURE D'ARTEFACT DANS CE PAQUET
// (constat C7 de la revue de la phase 6).
//
// # CE QU'IL EMPECHE
//
// Les quatre projections post-cuisson lisent le meme fichier de la meme facon. Le motif
// `os.ReadFile` + `json.Unmarshal(&replay.ReplayDocument)` existait en TROIS exemplaires
// (usage.go, bombstats.go, raster.go) : a la troisieme copie, la regle du depot impose un
// helper ET un garde-rail (CLAUDE.md n 6). Sans lui, la quatrieme copie arrive — et une
// correction (un cas d'erreur, une garde de taille, un compteur de lectures) n'en touche
// qu'une.
//
// Le garde interdit la DESERIALISATION vers `ReplayDocument` hors du helper. Il ne dit rien
// de `os.ReadFile` seul : ce paquet lit legitimement d'autres fichiers (chunks de film,
// capabilities), et interdire la primitive au lieu du motif aurait ete un garde qui gene
// sans proteger.
//
// LE MOTIF SURVEILLE EST `json.Unmarshal`, PAS LE NOM DU TYPE (fusion feat/v75 du
// 2026-09-07) : depuis que [Deriver] (derivations.go) lit CHAQUE artefact une seule fois et
// fait CIRCULER le document deja parse vers les projections (`doc *replay.ReplayDocument` en
// parametre — bombstats.go, derivations.go, positions.go), le simple NOM du type apparait
// legitimement hors de document.go. Ce n'est pas de la deserialisation, c'est le type qui
// voyage — exactement ce que ce garde-rail est cense laisser passer (cf. son intention
// ci-dessus). Le motif precedent (`replay.ReplayDocument`) confondait les deux.

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// motifDeserialisationDocument : la forme surveillee.
const motifDeserialisationDocument = "json.Unmarshal"

// proprietaireDeLaLecture : le seul fichier ou le type peut etre deserialise.
const proprietaireDeLaLecture = "document.go"

// TestUneSeuleLectureDArtefact — le ratchet.
//
// SELF-CHECK POSITIF : le proprietaire doit porter le motif, sinon le garde ne verifie plus
// rien (le helper a-t-il ete renomme ou deplace ?).
func TestUneSeuleLectureDArtefact(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	dir := filepath.Dir(thisFile)
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", dir, err)
	}

	var violations []string
	vuChezLeProprietaire := false
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, nom))
		if readErr != nil {
			t.Fatalf("lecture de %s: %v", nom, readErr)
		}
		for i, line := range strings.Split(string(data), "\n") {
			// Les en-tetes PARLENT du document et du motif : c'est le code qu'on garde,
			// pas la prose qui l'explique.
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if !strings.Contains(line, motifDeserialisationDocument) {
				continue
			}
			if nom == proprietaireDeLaLecture {
				vuChezLeProprietaire = true
				continue
			}
			violations = append(violations, nom+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
		}
	}
	if !vuChezLeProprietaire {
		t.Fatalf("le motif %q est introuvable dans %s : le helper de lecture a-t-il ete "+
			"renomme ou deplace ? Le garde ne verifie plus rien",
			motifDeserialisationDocument, proprietaireDeLaLecture)
	}
	if len(violations) > 0 {
		t.Errorf("deserialisation d'un artefact hors de %s (%d) — appeler lireDocumentRange : "+
			"le motif a deja existe en trois exemplaires, et une correction n'en touchait "+
			"qu'un :\n  %s", proprietaireDeLaLecture, len(violations), strings.Join(violations, "\n  "))
	}
}
