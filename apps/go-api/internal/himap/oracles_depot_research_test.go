//go:build research

package himap

// oracles_depot_research_test.go — les ORACLES du depot hote que lisent les instruments de
// recherche de la brique, sans importer aucun paquet de LevelUp.
//
// DEPUIS LE LOT 2c DU PLAN D1 (2026-10-28, decision utilisateur), ces sondes sont des
// INSTRUMENTS (`*_research_test.go`) : elles lisent des donnees que la brique ne possede pas,
// donc ne font partie ni du build par defaut ni de la suite `gamefiles` (le jeu seul). Elles
// exercent des fonctions NON exportees de la brique (tags sbsp/instances/forge, champs internes
// de `Rendu`) : leur place est ici, pas dans `internal/gamefilescorpus`. Les oracles se lisent
// par un parseur LOCAL (mapvar.Objective suffit) et la remontee depuis le repertoire de test
// (cheminDepuisDepot) ; fichier absent = instrument saute. Les fichiers lus sont ceux du depot
// hote, jamais copies dans la brique :
//
//   - data/titles/halo_infinite/reference/map_objectives.json (ancres d'objectifs) ;
//   - data/cache/replays/halo_infinite/*.json (positions jouees) ;
//   - data/titles/halo_infinite/reference/callouts_i18n.csv (libelles, lus par sonde_locs).
//
// Instruments concernes : cloture, filtres_reclaimer, physique_sonde, sddt, sonde_bouillie,
// sonde_forge_saut, sonde_locs.

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
)

// moduleCliffhanger : le module du banc de reference (Cliffhanger, variante Ridgeline).
const moduleCliffhanger = "cliffhanger_ridgeline"

// entreeCatalogue : ce que les sondes lisent d'une carte du catalogue d'objectifs.
type entreeCatalogue struct {
	Module     string             `json:"module"`
	ObjectsN   int                `json:"objects_n"`
	Objectives []mapvar.Objective `json:"objectives"`
}

// catalogueObjectifs : le fichier, indexe par map_id.
type catalogueObjectifs struct {
	Maps map[string]entreeCatalogue `json:"maps"`
}

// cheminCatalogue trouve le catalogue d'objectifs par remontee depuis le repertoire de test.
func cheminCatalogue() (string, error) {
	return cheminDepuisDepot("data/titles/halo_infinite/reference/map_objectives.json")
}

// chargeCatalogue lit le catalogue ; absent = test saute.
func chargeCatalogue(t *testing.T) catalogueObjectifs {
	t.Helper()
	p, err := cheminCatalogue()
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(p) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Fatal(err)
	}
	var c catalogueObjectifs
	if err := json.Unmarshal(brut, &c); err != nil {
		t.Fatalf("catalogue d'objectifs invalide (%s) : %v", p, err)
	}
	return c
}

// ancresDuModule rend les entrees du catalogue dont le module est `module`.
func ancresDuModule(t *testing.T, module string) []entreeCatalogue {
	t.Helper()
	var out []entreeCatalogue
	for _, e := range chargeCatalogue(t).Maps {
		if e.Module == module {
			out = append(out, e)
		}
	}
	return out
}

func ancresCliffhanger(t *testing.T) [][3]float64 {
	t.Helper()
	var pts [][3]float64
	for _, e := range ancresDuModule(t, moduleCliffhanger) {
		for _, o := range e.Objectives {
			pts = append(pts, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
		}
	}
	return pts
}

// chargeCarteForge lit le `.mvar` d'une carte Forge et son entree au catalogue, et PROUVE leur
// jointure par le compte d'objets — sans quoi on cuirait la carte d'une autre.
func chargeCarteForge(t *testing.T, c CarteForge) (*mapvar.Variant, entreeCatalogue) {
	t.Helper()
	brut := lireVarianteDuDepot(t, c.FichierMvar)
	v, err := mapvar.Parse(brut)
	if err != nil {
		t.Fatal(err)
	}
	entree, ok := chargeCatalogue(t).Maps[c.MapID]
	if !ok {
		t.Fatalf("carte %q absente du catalogue d'objectifs", c.MapID)
	}
	if entree.ObjectsN != len(v.Objects) {
		t.Fatalf("jointure catalogue<->mvar cassee : %d objets au catalogue, %d dans le .mvar",
			entree.ObjectsN, len(v.Objects))
	}
	return v, entree
}

// pointRejeu : une position de joueur du document de rejeu.
type pointRejeu struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type pisteRejeu struct {
	Points []pointRejeu `json:"points"`
}

type documentRejeu struct {
	Tracks []pisteRejeu `json:"tracks"`
}

// cheminRejeu cherche le document de rejeu : variable dediee, puis remontee depuis le
// repertoire de test.
func cheminRejeu() (string, error) {
	if p := os.Getenv("ORACLE_REPLAY"); p != "" {
		return p, nil
	}
	p, err := cheminDepuisDepot("data/cache/replays/halo_infinite/000d5950.json")
	if err != nil {
		return "", fmt.Errorf("document de rejeu introuvable (poser ORACLE_REPLAY) : %w", err)
	}
	return p, nil
}

func chargePositions(t *testing.T) []pointRejeu {
	t.Helper()
	p, err := cheminRejeu()
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(p) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Skip(err)
	}
	var doc documentRejeu
	if err := json.Unmarshal(brut, &doc); err != nil {
		t.Fatal(err)
	}
	var out []pointRejeu
	for _, piste := range doc.Tracks {
		out = append(out, piste.Points...)
	}
	if len(out) == 0 {
		t.Fatalf("%s ne porte aucune position", p)
	}
	return out
}

// cheminCalloutsI18n trouve `callouts_i18n.csv` (le catalogue de libelles versionne du depot
// hote, jamais copie dans la brique) ; absent = instrument saute.
func cheminCalloutsI18n(t *testing.T) string {
	t.Helper()
	p, err := cheminDepuisDepot("data/titles/halo_infinite/reference/callouts_i18n.csv")
	if err != nil {
		t.Skipf("catalogue i18n absent : %v", err)
	}
	return p
}
