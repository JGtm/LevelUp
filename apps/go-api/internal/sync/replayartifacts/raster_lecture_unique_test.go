package replayartifacts

// raster_lecture_unique_test.go — LE COMPTEUR QUI PROUVE LA DOUBLE LECTURE (lot M6,
// `.ai/PLAN_ORCHESTRATION_2026-09-07.md`, decouverte du registre `DECOUVERTES_TACTIQUE`
// section « Fusion origin/feat/v75 -> feat/tactique »).
//
// # LE DEFAUT MESURE ICI
//
// `lireArtefacts` (derivations.go) lit et deserialise CHAQUE artefact du cycle UNE fois et
// pose le document sur `artefactLu.doc`. Avant ce lot, `projeterRastersTactiques`
// (raster.go) rappelait `ProjeterRasterTactique(r.path)` — qui rouvre et redeserialise le
// MEME fichier — au lieu de reutiliser `r.doc` deja en memoire : une seconde lecture/parse
// par artefact et par cycle, pour rien.
//
// # COMMENT LE TEST COMPTE
//
// `ouvrirArtefact` (document.go) est le seul point d'ouverture du fichier. Ce test
// substitue une variante qui incremente un compteur, puis appelle `projeterRastersTactiques`
// avec UN artefact dont `doc` est deja renseigne (l'etat que `lireArtefacts` produit
// toujours en production). Le nombre de lectures DECLENCHEES PAR CET APPEL doit etre nul :
// le document est deja en main, `projeterRastersTactiques` ne doit plus rouvrir le fichier.
//
// AVANT LE CORRECTIF : ce test est ROUGE (`lectures == 1`, la relecture de
// `ProjeterRasterTactique`). APRES : il est VERT (`lectures == 0`).
import (
	"context"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

func TestProjeterRastersTactiques_NeRelitPasLeDocumentDejaEnMain(t *testing.T) {
	root := t.TempDir()
	path := ecrireFichier(t, "artefact.json", artefactImmobile)

	// La lecture DU CYCLE, deja faite par lireArtefacts avant que projeterRastersTactiques
	// ne soit jamais appelee — exactement ce qui se passe dans Deriver (derivations.go).
	doc, octets, err := lireDocumentRange(path)
	if err != nil {
		t.Fatalf("lecture initiale (celle du cycle): %v", err)
	}

	original := ouvrirArtefact
	lectures := 0
	ouvrirArtefact = func(p string) ([]byte, error) {
		lectures++
		return original(p)
	}
	defer func() { ouvrirArtefact = original }()

	d := Deps{RepoRoot: root, TitleSlug: titlePkg.DefaultSlug, Gamertag: "TestGT"}
	projeterRastersTactiques(context.Background(), d, []artefactLu{
		{matchID: doc.MatchID, path: path, doc: doc, octets: octets},
	})

	if lectures != 0 {
		t.Fatalf("projeterRastersTactiques a rouvert le fichier %d fois alors que le "+
			"document etait deja en memoire (artefactLu.doc) — double lecture/parse de "+
			"l'artefact par cycle (cf. DECOUVERTES_TACTIQUE, fusion feat/v75)", lectures)
	}
}
