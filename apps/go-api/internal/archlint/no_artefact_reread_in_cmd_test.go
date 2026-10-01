// Package archlint — no_artefact_reread_in_cmd_test.go : garde-rail de la lecture d'un artefact
// de rejeu par les commandes hors ligne (plan `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md`,
// item L7.5.1 RV3, CLAUDE.md regle 6).
//
// La lecture « `os.ReadFile` + `json.Unmarshal` vers un `replay.ReplayDocument` » etait copiee
// dans cinq commandes de `cmd/levelup` (bomb_stats, flag_grabs_net, pad_tiers, usage_summary,
// vehicle_takes) alors que `sync/replayartifacts` la possede deja (`lireDocumentRange`). Elle
// est desormais exportee en UN point, `replayartifacts.LireArtefactRange`
// (`internal/sync/replayartifacts/document.go`) ; une correction (cas d'erreur, garde de
// taille, compteur) la touche une fois. Une commande qui lit un artefact appelle ce helper.
//
// Empreinte : un meme fichier de `cmd/` qui deserialise par `json.Unmarshal` ET nomme
// `replay.ReplayDocument`. Le paquet `sync/replayartifacts` a son propre garde
// (`document_unique_test.go`). Tests de `cmd/` exclus (un temoin peut fabriquer un artefact).
// Le test du motif prouve que l'empreinte attrape l'ancienne copie.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var (
	deserialiseArtefactRE = regexp.MustCompile(`json\.Unmarshal\(`)
	typeDocumentRE        = regexp.MustCompile(`replay\.ReplayDocument`)
)

// relitUnArtefactALaMain dit si un texte source copie la lecture d'un document de rejeu.
func relitUnArtefactALaMain(source string) bool {
	return deserialiseArtefactRE.MatchString(source) && typeDocumentRE.MatchString(source)
}

func TestNoArtefactRereadInCmd_ReconnaitLesCopies(t *testing.T) {
	ancienneCopie := "\traw, err := os.ReadFile(path)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n" +
		"\tvar doc replay.ReplayDocument\n\tif err := json.Unmarshal(raw, &doc); err != nil {\n\t\treturn nil, err\n\t}\n"
	if !relitUnArtefactALaMain(ancienneCopie) {
		t.Error("le garde-rail ne reconnait pas l'ancienne copie de la lecture d'un artefact")
	}
	// Ni l'appel du helper, ni une deserialisation sans rapport avec le document ne comptent.
	if relitUnArtefactALaMain("doc, err := replayartifacts.LireArtefactRange(path)") {
		t.Error("faux positif sur l'appel du helper")
	}
	if relitUnArtefactALaMain("var cfg Config\n_ = json.Unmarshal(raw, &cfg)") {
		t.Error("faux positif sur une deserialisation sans rapport")
	}
}

func TestNoArtefactRereadInCmd(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	var violations []string
	err := filepath.WalkDir(filepath.Join(goAPIRoot, "cmd"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if relitUnArtefactALaMain(string(data)) {
			rel, _ := filepath.Rel(goAPIRoot, path)
			violations = append(violations, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de cmd/ : %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("lecture d'un artefact de rejeu copiee dans cmd/ — appeler "+
			"replayartifacts.LireArtefactRange (internal/sync/replayartifacts/document.go) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
