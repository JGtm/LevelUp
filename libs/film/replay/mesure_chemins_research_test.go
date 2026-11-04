//go:build research

package replay

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// mesure_chemins_research_test.go — LES CHEMINS DE DONNEES QUE LES MESURES DE POSITION RECOIVENT.
//
// Les mesures `*_research_test.go` qui relisent le parc d'un poste (faits cuits, artefacts de rejeu,
// bornes de quantification des cartes) ne devinent aucun chemin : la brique ne connait pas la
// disposition d'un depot hote. Chaque dossier arrive par sa variable, propre a ces instruments :
//
//	L1_BORNES      le fichier des bornes de quantification (`map_quant_bounds.json`)
//	L1_FAITS       le dossier des faits de film (`<id>.filmfacts.bin`)
//	L1_ARTEFACTS   le dossier des artefacts de rejeu (`<id>.json`)
//
// Une mesure qui a besoin d'un chemin et ne le trouve pas s'arrete en le disant (aucun repli).
const (
	envMesureBornes    = "L1_BORNES"
	envMesureFaits     = "L1_FAITS"
	envMesureArtefacts = "L1_ARTEFACTS"
)

// cheminMesureRequis rend la valeur de la variable `env`, ou arrete le test en nommant la variable.
func cheminMesureRequis(t *testing.T, env string) string {
	t.Helper()
	v := os.Getenv(env)
	if v == "" {
		t.Fatalf("%s absente : cette mesure lit le parc d'un poste, poser le chemin (voir l'en-tete du fichier de mesure)", env)
	}
	return v
}

// lireArtefactMesure lit `nom` dans le dossier de [envMesureArtefacts]. L artefact est FACULTATIF pour
// les mesures qui l utilisent : variable absente ou fichier absent rendent une erreur que l appelant
// traite comme « pas d artefact ».
func lireArtefactMesure(nom string) ([]byte, error) {
	dir := os.Getenv(envMesureArtefacts)
	if dir == "" {
		return nil, fmt.Errorf("%s absente", envMesureArtefacts)
	}
	return os.ReadFile(filepath.Join(dir, nom)) //nolint:gosec // mesure
}
