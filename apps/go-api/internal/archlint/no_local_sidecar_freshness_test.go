// Package archlint — no_local_sidecar_freshness_test.go : ratchet « UNE SEULE DEFINITION
// DE LA FRAICHEUR D'UN SIDECAR DE RASTER » (constat C3 de la revue de la phase 6).
//
// # CE QU'IL EMPECHE, ET CE QUE CA A COUTE
//
// Le service et le rattrapage n'appliquaient pas la meme regle. Le service ecartait un
// sidecar dont `pas_m` ou `pas_echantillon_ms` n'etaient plus courants, en prescrivant
// `levelup tactical-rasters --backfill` dans son avertissement ; la CLI, elle, ne
// regardait que `schema_version` et `artifact_schema_version`. Sur un sidecar au bon
// schema mais a une autre unite de temps, le remede prescrit etait donc un NO-OP, et le
// match restait non mesure a demeure — une promesse de reparation que rien n'honorait.
//
// La regle vit desormais dans `domain.SidecarRasterCourant`. Ce ratchet interdit d'en
// reecrire une seconde ailleurs, sous la forme d'une comparaison directe des champs.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// comparaisonsDeFraicheur : les formes qui refont le predicat a la main.
var comparaisonsDeFraicheur = []string{
	"SchemaVersion ==", "SchemaVersion !=",
	"PasM ==", "PasM !=",
	"PasEchantillonMs ==", "PasEchantillonMs !=",
}

// paquetProprietaireDeLaFraicheur : le seul endroit ou ces comparaisons sont legitimes.
const paquetProprietaireDeLaFraicheur = "internal/domain/"

// typeDuSidecar BORNE le balayage aux seuls consommateurs du sidecar.
//
// SANS CETTE BORNE, LE RATCHET EST FAUX : `SchemaVersion ==` est une comparaison banale
// dans un depot qui versionne une dizaine de documents (catalogue de hoppers, artefact de
// rejeu, adaptateur de titre...). Mesure faite : le motif nu remontait cinq comparaisons
// parfaitement legitimes, sans le moindre rapport avec un raster. Seul un fichier qui
// NOMME le type peut en reecrire la fraicheur.
const typeDuSidecar = "TacticalRasterSidecar"

// TestUneSeuleDefinitionDeLaFraicheurDuSidecar — le ratchet.
//
// `ArtifactSchemaVersion` n'y figure PAS : elle ne se compare qu'a un artefact, que seul
// le rattrapage tient en main, et c'est une comparaison legitimement locale.
//
// SELF-CHECK POSITIF : le proprietaire doit porter au moins une de ces comparaisons, sinon
// le garde ne verifie plus rien (le predicat a-t-il ete renomme ou vide ?).
func TestUneSeuleDefinitionDeLaFraicheurDuSidecar(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	vuChezLeProprietaire := false
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
		rel := filepath.ToSlash(mustRel(apiRoot, path))
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if !strings.Contains(string(data), typeDuSidecar) {
			return nil
		}
		proprietaire := strings.HasPrefix(rel, paquetProprietaireDeLaFraicheur)
		for i, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			// `ArtifactSchemaVersion` est RETIREE avant l'examen : elle ne se compare
			// qu'a un artefact, que seul le rattrapage tient en main, et c'est une
			// comparaison legitimement locale. Sans ce retrait, le substring
			// « SchemaVersion == » la capturerait par accident.
			nue := strings.ReplaceAll(line, "ArtifactSchemaVersion", "")
			for _, motif := range comparaisonsDeFraicheur {
				if !strings.Contains(nue, motif) {
					continue
				}
				if proprietaire {
					vuChezLeProprietaire = true
					continue
				}
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours apps/go-api: %v", err)
	}
	if !vuChezLeProprietaire {
		t.Fatalf("aucune comparaison de fraicheur dans %s : le predicat canonique a-t-il ete "+
			"renomme ou vide ? Le garde ne verifie plus rien", paquetProprietaireDeLaFraicheur)
	}
	if len(violations) > 0 {
		t.Errorf("fraicheur d'un sidecar de raster recopiee hors de %s (%d) — appeler "+
			"domain.SidecarRasterCourant : deux definitions font que le remede prescrit par "+
			"l'une ne repare pas ce que l'autre refuse :\n  %s",
			paquetProprietaireDeLaFraicheur, len(violations), strings.Join(violations, "\n  "))
	}
}
