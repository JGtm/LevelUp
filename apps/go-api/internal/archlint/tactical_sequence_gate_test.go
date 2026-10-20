// Package archlint — tactical_sequence_gate_test.go : ratchet du MONTAGE de la route de sequence d'une
// mort (onglet Tactique, vue D) sous EXACTEMENT les gardes du rejeu 2D.
//
// La sequence lit les trajectoires decodees du film, comme `/replay`. Decision du user du 2026-10-14 :
// elle suit le meme interrupteur que le rejeu, pour qu'activer le rejeu en production active la vue D
// sans autre changement. Concretement, son site de montage doit avoir pour ancetres a la fois
// `RequireCapability(..., CapReplay)` (porte de titre) et `LocalOnlyReplay` (garde local), les deux
// que porte le groupe de `NewReplayHandler` (replay_routes_capability_gate_test.go,
// tactical_background_local_gate_test.go : meme extracteur de sites).
//
// UN CHIFFRE NE SUFFIT PAS : on exige aussi que le montage du rejeu soit encore sous ces deux gardes
// (sinon « memes gardes que le rejeu » ne veut plus rien dire) — c'est la paire qui dit la regle.
package archlint

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const sequenceHandlerCtor = "NewTacticalSequenceHandler"

// TestTacticalSequenceSousLesGardesDuRejeu — chaque montage de la sequence est sous la porte de titre
// ET le garde local.
func TestTacticalSequenceSousLesGardesDuRejeu(t *testing.T) {
	apiRoot := archlintAPIRoot(t)
	sequence := montagesAvecGardes(t, apiRoot, sequenceHandlerCtor)
	rejeu := montagesAvecGardes(t, apiRoot, replayHandlerCtor)

	if sequence.sites == 0 || rejeu.sites == 0 {
		t.Fatalf("sites de montage trouves : sequence %d, rejeu %d — le ratchet ne mord plus sur rien "+
			"(constructeur renomme ? montage deplace ?)", sequence.sites, rejeu.sites)
	}
	if len(rejeu.sansPorte) > 0 || len(rejeu.sansGardeLocal) > 0 {
		t.Fatalf("le montage du rejeu a perdu l'un de ses gardes (%v / %v) : « memes gardes que le "+
			"rejeu » ne veut plus rien dire — corriger d'abord le rejeu",
			rejeu.sansPorte, rejeu.sansGardeLocal)
	}
	for _, s := range sequence.sansPorte {
		t.Errorf("la sequence d'une mort est montee SANS %s(..., %s) (%s) : un titre sans rejeu la "+
			"servirait", replayGateCall, replayGateCap, s)
	}
	for _, s := range sequence.sansGardeLocal {
		t.Errorf("la sequence d'une mort est montee SANS %s (%s) : elle exposerait les trajectoires "+
			"du film hors du garde local du rejeu", gardeLocalRejeu, s)
	}
}

type verdictGardes struct {
	sites          int
	sansPorte      []string
	sansGardeLocal []string
}

// montagesAvecGardes rend, pour un constructeur, ses sites de montage et ceux auxquels il manque
// la porte de titre ou le garde local.
func montagesAvecGardes(t *testing.T, apiRoot, ctor string) verdictGardes {
	t.Helper()
	var out verdictGardes
	err := filepath.WalkDir(apiRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", ".git", "node_modules", "tmp", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		octets, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel := filepath.ToSlash(mustRel(apiRoot, path))
		if !strings.Contains(string(octets), ctor+"(") || strings.HasPrefix(rel, "internal/api/handlers/") {
			return nil
		}
		fset := token.NewFileSet()
		fichier, parseErr := parser.ParseFile(fset, path, octets, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatalf("parser.ParseFile(%s): %v", rel, parseErr)
		}
		recenser(&out, fset, rel, sitesDeMontage(fichier, ctor))
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s: %v", apiRoot, err)
	}
	return out
}

func recenser(out *verdictGardes, fset *token.FileSet, rel string, sites []siteMontage) {
	for _, site := range sites {
		out.sites++
		ou := rel + ":" + strconv.Itoa(fset.Position(site.appel.Pos()).Line)
		if !unAncetreInstalleLaPorte(site.ancetres) {
			out.sansPorte = append(out.sansPorte, ou)
		}
		if !unAncetreInstalleLeGardeLocal(site.ancetres) {
			out.sansGardeLocal = append(out.sansGardeLocal, ou)
		}
	}
}
