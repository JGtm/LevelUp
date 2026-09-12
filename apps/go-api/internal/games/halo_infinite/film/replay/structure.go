package replay

// STRUCTURE DE CARTE — fichier de référence figé par carte.
//
// Le fond de carte du rejeu venait jusqu'ici des props de la variante Forge : 382 objets,
// 0,25 m² d'emprise médiane, 3,4 % de la surface de jeu couverts. Ce sont des accessoires,
// pas la carte. Les sols, plateformes, rampes et murs vivent dans le bloc
// `instanced geometry instances` du tag scenario_structure_bsp du module de la carte
// (cf. internal/himap). Ce bloc est lu HORS LIGNE par cmd/mapstruct-build, qui fige ici
// l'emprise (AABB monde projetée) de chaque instance.
//
// RÈGLE : ce fichier exige les fichiers du jeu installé, donc il est VERSIONNÉ comme
// donnée de référence (au même titre que map_quant_bounds.json), jamais régénéré à la
// volée. Son absence n'est PAS fatale au rejeu : le document sort sans champ Structure.

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// MapStructureSchemaVersion est la version de forme du fichier de structure.
const MapStructureSchemaVersion = 1

// MapStructure est le fichier figé d'une carte (un par module, cf.
// PathResolver.MapStructurePath).
type MapStructure struct {
	SchemaVersion int `json:"schemaVersion"`
	// Module est le dossier du .module d'où vient la structure (clé du fichier).
	Module string `json:"module"`
	// MapNames sont les noms de carte affichés qui partagent ce module (traçabilité).
	MapNames []string `json:"mapNames,omitempty"`
	// Source documente la chaîne d'extraction.
	Source      string    `json:"source"`
	GeneratedAt time.Time `json:"generatedAt"`
	// BSPIndex est l'index du tag sbsp retenu dans le module (le BSP de jeu : celui dont
	// les bornes monde ont le plus petit volume parmi ceux qui portent des instances —
	// le BSP de skybox couvre ±3300 m et serait retenu par un critère de taille brute).
	BSPIndex int `json:"bspIndex"`
	// WorldMin / WorldMax sont les bornes monde du BSP retenu.
	WorldMin [3]float32 `json:"worldMin"`
	WorldMax [3]float32 `json:"worldMax"`
	// Excluded compte les instances écartées par motif (traçabilité de la sélection).
	Excluded map[string]int `json:"excluded,omitempty"`
	// Stats résume les emprises publiées — de quoi CONSTATER, sans relire tout le fichier,
	// si la structure d'une carte est exploitable ou non.
	Stats MapStructureStats `json:"stats"`
	// Surfaces sont les emprises publiées, dans l'ordre des instances du bloc.
	Surfaces []Surface `json:"surfaces"`
}

// MapStructureStats résume la distribution des emprises d'une carte.
//
// AVERTISSEMENT porté par ces chiffres : le contenu du bloc varie fortement d'une carte à
// l'autre. Sur Cliffhanger (module ridgeline) et Streets la structure couvre 100 % des
// bornes monde ; sur Catalyst, Forest et Aquarius elle plafonne à 40-49 %, le reste vivant
// dans le maillage de rendu NON instancié (bloc `meshes`, non lu). CoveragePct dit lequel
// des deux cas s'applique : ne pas publier une carte comme « décor complet » sans l'avoir
// regardé.
type MapStructureStats struct {
	Count int `json:"count"`
	// AreaMedian / AreaMax sont en m².
	AreaMedian float32 `json:"areaMedian"`
	AreaMax    float32 `json:"areaMax"`
	// CoveragePct est la part des bornes monde du BSP couverte par au moins une emprise,
	// rastérisée au pas CoverCell.
	CoveragePct float32 `json:"coveragePct"`
	// CoveragePctCapped est la MÊME couverture en n'utilisant que les emprises d'aire
	// <= AreaCap. Contrôle ANTI-TAUTOLOGIE : une seule dalle géante suffit à « couvrir »
	// une carte. Un écart massif entre les deux chiffres (Streets : 100 % contre 41 %)
	// dit que la couverture brute est portée par des boîtes englobantes, pas par du sol.
	CoveragePctCapped float32 `json:"coveragePctCapped"`
	// AreaCap est le plafond d'aire de CoveragePctCapped, en m².
	AreaCap float32 `json:"areaCap"`
	// CoverCell est le pas de rastérisation des couvertures, en mètres.
	CoverCell float32 `json:"coverCell"`
}

// LoadMapStructure lit le fichier de structure d'une carte.
func LoadMapStructure(path string) (*MapStructure, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("structure de carte illisible (%s) : %w", path, err)
	}
	var s MapStructure
	if err := json.Unmarshal(blob, &s); err != nil {
		return nil, fmt.Errorf("structure de carte invalide (%s) : %w", path, err)
	}
	if s.SchemaVersion != MapStructureSchemaVersion {
		return nil, fmt.Errorf("structure de carte en version %d, attendu %d (%s)",
			s.SchemaVersion, MapStructureSchemaVersion, path)
	}
	return &s, nil
}

// surfaceBounds calcule l'étendue XY des emprises (nil si aucune).
func surfaceBounds(surf []Surface) *Bounds {
	if len(surf) == 0 {
		return nil
	}
	b := Bounds{MinX: surf[0].X0, MinY: surf[0].Y0, MaxX: surf[0].X1, MaxY: surf[0].Y1}
	for _, s := range surf[1:] {
		b.MinX, b.MaxX = minf(b.MinX, s.X0), maxf(b.MaxX, s.X1)
		b.MinY, b.MaxY = minf(b.MinY, s.Y0), maxf(b.MaxY, s.Y1)
	}
	return &b
}
