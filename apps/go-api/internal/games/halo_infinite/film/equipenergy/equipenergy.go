// Package equipenergy porte le catalogue de l ENERGIE DES EQUIPEMENTS d Halo Infinite : sa forme
// sur disque (data/titles/halo_infinite/reference/equipment_energy.json, chemin que l hote
// resout pour le titre), son lecteur, et la regle du jeu qui compte les usages
// restants d un equipement porte.
//
// D OU IL VIENT. De `cmd/equipenergy-build`, qui le lit dans les tags `eqip` et `sofa` du jeu
// installe (`himap.LireLesEnergiesDEquipement`) : jamais saisi a la main. Chaque entree est une fiche
// `sofa` qui relie l objet RAMASSABLE (nomme par le film au sol) a l objet PORTE (cree attache au
// joueur qui le prend), avec l energie initiale des deux et la capacite du second.
package equipenergy

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// SchemaVersion est la version de forme du catalogue.
//
//	v1 (2026-10-14) : par fiche, les deux GlobalID, les deux energies initiales, le seuil et la
//	variation par usage de la capacite de l objet porte, et le module lu.
const SchemaVersion = 1

// Catalog est le catalogue, tel qu il est sur disque.
type Catalog struct {
	SchemaVersion int    `json:"schema_version"`
	TitleSlug     string `json:"title_slug"`
	// Tool est la commande qui le produit ; About dit ce qu il porte.
	Tool  string `json:"tool"`
	About string `json:"about"`
	// Offsets : ou chaque constante se lit dans un tag, et quel lecteur du jeu la lit.
	Offsets map[string]string `json:"offsets"`
	Entries []Entry           `json:"entries"`
}

// Entry est une fiche `sofa` et ses deux objets.
type Entry struct {
	// Sofa, Pickup, Carried : GlobalID en hexadecimal (0x%08x) de la fiche, de l objet ramassable et
	// de l objet porte.
	Sofa    string `json:"sofa"`
	Pickup  string `json:"pickup"`
	Carried string `json:"carried"`
	// PickupEnergy et CarriedEnergy : energies initiales des deux objets (`eqip` +0x8ac).
	PickupEnergy  float32 `json:"pickup_energy"`
	CarriedEnergy float32 `json:"carried_energy"`
	// Threshold et Delta : la capacite de l objet porte (`eqip` +0x904 → +0x1c, +0x24).
	Threshold float32 `json:"threshold"`
	Delta     float32 `json:"delta"`
	// Module : le module du jeu ou la fiche a ete lue.
	Module string `json:"module"`
}

// Load lit le catalogue et refuse une version de forme inconnue.
func Load(path string) (*Catalog, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // chemin resolu par le PathResolver
	if err != nil {
		return nil, err
	}
	var c Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("catalogue de l energie des equipements %s : %w", path, err)
	}
	if c.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("catalogue de l energie des equipements %s : version %d, %d attendue", path,
			c.SchemaVersion, SchemaVersion)
	}
	return &c, nil
}

// PickupID rend le GlobalID de l objet ramassable d une entree, ou faux s il est illisible.
func (e Entry) PickupID() (uint32, bool) {
	v, err := strconv.ParseUint(e.Pickup, 0, 32)
	return uint32(v), err == nil
}

// epsilonDesUsages est la constante que `FUN_1407e9384` ajoute a l energie avant la division
// (`DAT_143cd837c` = 0x38d1b717 = 1e-4).
const epsilonDesUsages float32 = 1e-4

// UsagesRestants applique la regle du HUD (`FUN_1407e9384`) a une energie : seuil > 0 et variation
// < 0 → trunc((energie + 1e-4) / -variation). Faux quand la capacite ne compte pas par l energie
// (seuil <= 0 : le jeu lit alors un compteur que le film n ecrit pas ; variation >= 0 : illimite).
func UsagesRestants(energie, seuil, variation float32) (int, bool) {
	if seuil <= 0 || -variation <= 0 {
		return 0, false
	}
	return int((energie + epsilonDesUsages) / -variation), true
}
