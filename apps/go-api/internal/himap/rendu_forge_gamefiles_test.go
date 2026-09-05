//go:build gamefiles

package himap

// LE RENDU D'UNE CARTE FORGE — Vagabond, posee depuis les objets de son `.mvar`.
//
// La chaine et ses mesures fondatrices vivent desormais dans `cuisson_forge.go`, et elle
// produit un asset (`cmd/mapfond-build`). Ce test l'APPELLE et verifie ce qu'elle seule ne peut
// pas verifier : la jointure du catalogue d'objectifs au fichier de variante, l'exclusion des
// volumes de mort, et le detail par ancre.
//
// CE QUE LE DETAIL PAR ANCRE APPREND, et pourquoi la mediane ne suffit pas ici : Vagabond n'a
// que 4 ancres. Un ecart tres negatif sur un CENTRE de zone Bastion dit « il y a un toit
// au-dessus », pas « le sol est faux » — le z-buffer retient la surface la plus haute.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

func TestRenduForgeVagabond(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	carte := CartesForge[0]
	v, entree := chargeCarteForge(t, carte)

	var ancres [][3]float64
	for _, o := range entree.Objectives {
		ancres = append(ancres, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
	}
	if len(ancres) == 0 {
		t.Fatal("aucune ancre pour Vagabond — l'oracle faible est vide")
	}
	t.Logf("%s (%s) : %d objets, %d ancres", carte.Nom, carte.MapID, len(v.Objects), len(ancres))

	rendu, bilan, err := CuitCarteForge(context.Background(), OptionsCuissonForge{
		RacineDeploy:        root,
		Objets:              v.Objects,
		Ancres:              ancres,
		CheminModuleCanevas: CheminCanevasForge(carte),
		Cle:                 carte.MapID,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("objets : %d dessines · %d sans modele rtgo (sautes) · %d volumes de mort EXCLUS",
		bilan.ObjetsDessines, bilan.ObjetsSansModele, bilan.VolumesDeMort)
	if bilan.VolumesDeMort == 0 {
		t.Error("Vagabond devrait porter au moins un volume de mort a exclure")
	}
	journaliseBilan(t, rendu, bilan)

	for i, o := range entree.Objectives {
		ci := int((o.Pos.X - rendu.Min[0]) / rendu.Cell)
		cj := int((o.Pos.Y - rendu.Min[1]) / rendu.Cell)
		if z, ok := rendu.Altitude(ci, cj); ok {
			t.Logf("  ancre %d (%s) z=%.2f : surface dessinee %.2f (ecart %+.2f)",
				i, o.Role, o.Pos.Z, z, o.Pos.Z-AncrageDecalageSol-z)
		}
	}

	if sortie := os.Getenv("RENDU_PNG_FORGE"); sortie != "" {
		ecritPNG(t, sortie, FondPNG(rendu, rendu.NX, rendu.NY, nil, styleDemande()))
		fmt.Println("carte forge ecrite:", sortie)
	}
}

// chargeCarteForge lit le `.mvar` d'une carte Forge et son entree au catalogue, et PROUVE leur
// jointure par le compte d'objets — sans quoi on cuirait la carte d'une autre.
func chargeCarteForge(t *testing.T, c CarteForge) (*mapvar.Variant, replay.MapObjectivesEntry) {
	t.Helper()
	chemin, err := cheminDepuisDepot(filepath.Join(DepotVariantesCarte, c.FichierMvar))
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Fatal(err)
	}
	v, err := mapvar.Parse(brut)
	if err != nil {
		t.Fatal(err)
	}
	p, err := cheminCatalogue()
	if err != nil {
		t.Skip(err)
	}
	cat, err := replay.LoadMapObjectives(p)
	if err != nil {
		t.Fatal(err)
	}
	entree, err := cat.Lookup(c.MapID)
	if err != nil {
		t.Fatal(err)
	}
	if entree.ObjectsN != len(v.Objects) {
		t.Fatalf("jointure catalogue<->mvar cassee : %d objets au catalogue, %d dans le .mvar",
			entree.ObjectsN, len(v.Objects))
	}
	return v, entree
}
