// Package archlint — no_cuisson_depuis_tactique_test.go : ratchet « L'ONGLET TACTIQUE NE
// CUIT RIEN » (plan tactique, phase 6).
//
// # CE QU'IL INTERDIT, ET POURQUOI C'EST LA REGLE LA PLUS DURE DU CHANTIER
//
// Cuire un artefact, c'est decoder un film : des minutes de CPU, des gigaoctets de RAM, et
// un verrou process (`filmproc.AcquireSolo`). La machine est morte trois fois pour l'avoir
// fait au mauvais endroit — d'ou la regle utilisateur « JAMAIS de cuisson d'artefacts en
// lot, demander avant ». Le chantier Tactique repose entierement sur l'inverse : les
// rasters sont calcules UNE FOIS a la cuisson, et tout le reste — la page, le service, le
// rattrapage — ne fait plus que LIRE ce qui est deja sur le disque.
//
// Cette regle est une propriete du CODE, pas une intention : une page qui pourrait
// declencher une cuisson la declencherait un jour, sur la requete d'un visiteur. Le
// ratchet la rend impossible a reintroduire par inadvertance.
//
// # LES DEUX SURFACES GARDEES
//
//	cmd/levelup/cmd_tactical_rasters.go   le rattrapage : il LIT les artefacts existants ;
//	internal/service/tactical*.go         le service de la page.
//
// Aucun des deux ne doit mentionner `replaybuild` ni `BuildFromFilm` — ni en import, ni en
// appel. La detection est TEXTUELLE et volontairement large : un fichier qui nomme le
// paquet de cuisson est deja suspect, meme s'il ne l'appelle pas encore.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// motifsDeCuisson : ce qu'aucun fichier tactique ne doit nommer.
//
//	replaybuild      le paquet de cuisson (construction d'artefact, spawn d'enfant) ;
//	BuildFromFilm    l'entree de decodage d'un film ;
//	filmcache        le pont disque des chunks — telecharger un film est le prealable
//	                 d'une cuisson, et n'a rien a faire dans une lecture ;
//	SpawnBuildOne    le spawn de l'enfant decodeur (`replayartifacts/buildone.go`) ;
//	replayartifacts.Run(   le cycle post-sync COMPLET, qui cuit ;
//	replaychild      le binaire enfant de cuisson ;
//	filmproc         le verrou process du decodage (`filmproc.AcquireSolo`).
//
// # POURQUOI LES QUATRE DERNIERS ONT ETE AJOUTES (constat C2 de la revue)
//
// Le rattrapage IMPORTE deja `internal/sync/replayartifacts`, pour `ProjeterRasterTactique`
// et `EcrireSidecarRaster` — et c'est legitime : ce sont des projections pures d'un fichier
// deja range. Mais ce meme paquet EXPORTE `SpawnBuildOne` et `Run`. Remplacer la branche
// d'echec de `projeterCorpusRasters` par un `replayartifacts.SpawnBuildOne(...)` laissait
// donc le ratchet VERT tout en faisant de la CLI une cuisson EN LOT : bombe RAM x4 et
// verrou `filmproc.AcquireSolo` pris sur toute la passe. Le nom de paquet nu reste
// autorise ; ce sont les points d'entree de cuisson qui sont fermes.
var motifsDeCuisson = []string{
	"replaybuild", "BuildFromFilm", "filmcache",
	"SpawnBuildOne", "replayartifacts.Run(", "replaychild", "filmproc",
}

// fichiersTactiquesSansCuisson : les fichiers gardes, relatifs a apps/go-api.
// `internal/service/tactical` est un PREFIXE — tout fichier tactique du service y entre,
// y compris ceux qui n'existent pas encore (c'est le point d'un ratchet).
var fichiersTactiquesSansCuisson = []string{
	"cmd/levelup/cmd_tactical_rasters.go",
	"internal/service/tactical",
}

// TestAucuneCuissonDepuisLeTactique — le ratchet lui-meme.
//
// SELF-CHECK POSITIF : le test exige d'avoir VU les fichiers qu'il garde. Sans lui, un
// renommage rendrait le garde vert en ne gardant plus rien — le defaut exact qui a fait
// tomber la premiere version du ratchet du motif XUID (revue ronde 2 de la phase 4 bis).
func TestAucuneCuissonDepuisLeTactique(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // internal/archlint -> internal -> apps/go-api

	var violations []string
	// SELF-CHECK PAR CIBLE, pas un simple compte : `internal/service/tactical` matche
	// PLUSIEURS fichiers, si bien qu'un total suffisant aurait pu masquer la disparition
	// complete de l'autre cible.
	vues := make(map[string]bool, len(fichiersTactiquesSansCuisson))
	// SELF-CHECK PAR MOTIF : chaque motif interdit doit EXISTER quelque part dans le
	// depot, hors des fichiers gardes. Un motif mort — une fonction renommee, un paquet
	// deplace — ne garde plus rien, et le ratchet resterait vert en n'interdisant que du
	// vocabulaire disparu.
	motifsVivants := make(map[string]bool, len(motifsDeCuisson))
	err := filepath.WalkDir(apiRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", ".git", "node_modules", "tmp":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// `mustRel` est le helper deja pose par tactical_background_local_gate_test.go :
		// un troisieme chemin relatif recopie dans ce paquet serait la copie de trop.
		rel := filepath.ToSlash(mustRel(apiRoot, path))
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		cible, garde := cibleDuFichierTactique(rel)
		if !garde {
			// Fichier HORS perimetre : il ne sert qu'au self-check des motifs (le
			// vocabulaire interdit doit exister quelque part, sans quoi il ne garde rien).
			for _, motif := range motifsDeCuisson {
				if strings.Contains(string(data), motif) {
					motifsVivants[motif] = true
				}
			}
			return nil
		}
		vues[cible] = true
		for i, line := range strings.Split(string(data), "\n") {
			// Les commentaires PARLENT de la regle (« ne cuit rien, aucun appel a
			// replaybuild ») : c'est le code qu'on garde, pas la prose qui l'explique.
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			for _, motif := range motifsDeCuisson {
				if strings.Contains(line, motif) {
					violations = append(violations,
						rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours apps/go-api: %v", err)
	}
	for _, cible := range fichiersTactiquesSansCuisson {
		if !vues[cible] {
			t.Fatalf("aucun fichier ne correspond a la cible %q : elle a ete renommee ou "+
				"deplacee, et le ratchet ne garde plus rien de ce cote-la", cible)
		}
	}
	for _, motif := range motifsDeCuisson {
		if !motifsVivants[motif] {
			t.Fatalf("le motif interdit %q n'existe plus nulle part dans le depot : il a ete "+
				"renomme ou supprime, et le ratchet n'interdit plus qu'un vocabulaire mort", motif)
		}
	}
	if len(violations) > 0 {
		t.Errorf("cuisson d'artefact atteignable depuis le chantier Tactique (%d) — la page "+
			"et son rattrapage LISENT les artefacts, ils n'en construisent JAMAIS :\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// cibleDuFichierTactique rend la cible du ratchet qu'un chemin relatif satisfait.
func cibleDuFichierTactique(rel string) (string, bool) {
	for _, cible := range fichiersTactiquesSansCuisson {
		if rel == cible || strings.HasPrefix(rel, cible) {
			return cible, true
		}
	}
	return "", false
}
