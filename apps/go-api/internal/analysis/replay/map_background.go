package replay

// FOND DE CARTE — le sidecar de calage d'une image de carte figée.
//
// POURQUOI CE FICHIER EST ICI ET PAS DANS `internal/himap`. Le fond de carte est PRODUIT hors
// ligne par cmd/mapfond-build, qui lit les fichiers du jeu via internal/himap → internal/ooz,
// **licencié GPLv3**. L'application, elle, doit pouvoir LIRE le calage sans jamais linker cette
// chaîne : le type et son lecteur vivent donc du côté consommateur, exactement comme
// MapStructure. Un garde-rail (cmd/mapfond-build/licence_test.go) interdit à cmd/server de
// dépendre d'ooz.
//
// CE QUE RÉSOUT CE SIDECAR. `carte_validee_v1.png`, la première carte reconstruite, a été
// produite sans que son calage soit écrit nulle part : il a fallu le retrouver à la main sur
// les trajectoires de joueur (0,0920 m/px, X0 -43,5, Y1 61,0). Une image dont on ignore où elle
// se pose dans le monde ne se superpose à rien. Le calage voyage désormais AVEC l'image.
//
// RÈGLE, la même que map_structure et map_quant_bounds : la production exige les fichiers du
// jeu installé, donc c'est une donnée de RÉFÉRENCE versionnée, jamais régénérée à la volée.

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// MapBackgroundSchemaVersion est la version de forme du sidecar de fond de carte.
const MapBackgroundSchemaVersion = 1

// MapBackground est le sidecar JSON qui accompagne `{module}.png`.
type MapBackground struct {
	SchemaVersion int `json:"schemaVersion"`
	// Module est le dossier du .module d'où vient la carte (clé du fichier, et nom du PNG).
	Module string `json:"module"`
	// MapNames sont les noms de carte affichés qui partagent ce module (traçabilité).
	MapNames []string `json:"mapNames,omitempty"`
	// Image est le nom de fichier du PNG, à côté de ce sidecar.
	Image string `json:"image"`
	// Source documente la chaîne de production.
	Source      string    `json:"source"`
	GeneratedAt time.Time `json:"generatedAt"`
	// Style est l'habillage appliqué (himap.StyleFond). Un fond qu'on ne sait pas colorier
	// de la même façon n'est pas reproductible.
	Style string `json:"style"`
	// Calibration place l'image dans le monde.
	Calibration MapBackgroundCalibration `json:"calibration"`
	// Stats chiffrent la cuisson — de quoi CONSTATER qu'une carte est exploitable sans
	// ouvrir l'image.
	Stats MapBackgroundStats `json:"stats"`
	// Degradations liste en clair ce qui a manqué à la cuisson (pas de tag sddt, pas d'eau,
	// frontière refusée...). Vide = chaîne complète.
	Degradations []string `json:"degradations,omitempty"`
}

// MapBackgroundCalibration place l'image dans le repère monde.
//
// Convention (également publiée en clair dans le champ Convention, pour que le fichier se
// suffise à lui-même) :
//
//	xMonde = originX + (px + 0.5) * metersPerPixel
//	yMonde = originY - (py + 0.5) * metersPerPixel
//
// OriginY est l'ordonnée du bord HAUT : l'image descend en Y quand le monde monte.
type MapBackgroundCalibration struct {
	MetersPerPixel float64 `json:"metersPerPixel"`
	OriginX        float64 `json:"originX"`
	OriginY        float64 `json:"originY"`
	WidthPx        int     `json:"widthPx"`
	HeightPx       int     `json:"heightPx"`
	Convention     string  `json:"convention"`
}

// MapBackgroundStats chiffre la cuisson.
//
// AnchorsWithGround/AnchorsInFrame est l'ORACLE FAIBLE, disponible sur toutes les cartes :
// chaque ancre d'objectif doit trouver du sol dessiné sous elle. Une ancre sans sol est un trou
// de reconstruction, pas une question de goût.
type MapBackgroundStats struct {
	Anchors           int `json:"anchors"`
	AnchorsInFrame    int `json:"anchorsInFrame"`
	AnchorsWithGround int `json:"anchorsWithGround"`
	// AnchorMedianGapM est l'écart médian, en mètres, entre l'ancre ramenée au sol et le sol
	// dessiné. Absent si aucune ancre n'a trouvé de sol.
	AnchorMedianGapM *float64 `json:"anchorMedianGapM,omitempty"`
	// InstancesDrawn / InstancesScenery : instances rendues, et instances écartées comme
	// décor par le grain de leur maillage.
	InstancesDrawn   int `json:"instancesDrawn"`
	InstancesScenery int `json:"instancesScenery"`
	// PlayLevelZ est l'altitude du sol joué, déduite des ancres.
	PlayLevelZ float64 `json:"playLevelZ"`
	// BoundaryApplied dit si la coquille de mort déclarée par la carte a été appliquée ;
	// BoundaryPlanes compte ses triangles, BoundaryCellsCleared les cellules effacées.
	BoundaryApplied      bool `json:"boundaryApplied"`
	BoundaryPlanes       int  `json:"boundaryPlanes"`
	BoundaryCellsCleared int  `json:"boundaryCellsCleared"`
	// WaterVolumes / WaterCells : l'habillage d'eau.
	WaterVolumes int `json:"waterVolumes"`
	WaterCells   int `json:"waterCells"`
	// CoveredShare / Covered / CellsSubstituted : la voie de référence contre les TOITS
	// (himap/rendu_reference.go). CoveredShare est la part de matière qui cache un sol
	// praticable ; au-delà d'un tiers la carte est couverte et ses pixels, dans la portée des
	// ancres, montrent la surface la plus proche du sol de référence des ancres.
	CoveredShare     float64 `json:"coveredShare"`
	Covered          bool    `json:"covered"`
	CellsSubstituted int     `json:"cellsSubstituted,omitempty"`
	// Champs FORGE — absents d'une carte native. ForgeObjectsWithoutModel n'est pas du bruit :
	// il dit quelle part de la carte manque à l'image (les objets qui passent par
	// bloc/scen/mach vers hlmt, saut non traité).
	ForgeObjects             int `json:"forgeObjects,omitempty"`
	ForgeObjectsDrawn        int `json:"forgeObjectsDrawn,omitempty"`
	ForgeObjectsWithoutModel int `json:"forgeObjectsWithoutModel,omitempty"`
	ForgeDeathVolumes        int `json:"forgeDeathVolumes,omitempty"`
}

// LoadMapBackground lit le sidecar de fond de carte.
func LoadMapBackground(path string) (*MapBackground, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fond de carte illisible (%s) : %w", path, err)
	}
	var b MapBackground
	if err := json.Unmarshal(blob, &b); err != nil {
		return nil, fmt.Errorf("fond de carte invalide (%s) : %w", path, err)
	}
	if b.SchemaVersion != MapBackgroundSchemaVersion {
		return nil, fmt.Errorf("fond de carte en version %d, attendu %d (%s)",
			b.SchemaVersion, MapBackgroundSchemaVersion, path)
	}
	return &b, nil
}

// MondeVersPixel applique la convention de calage : position monde -> pixel de l'image.
//
// C'est le seul endroit où un consommateur doit implémenter la formule — l'écrire une seconde
// fois ailleurs est exactement la faute que ce sidecar existe pour empêcher. Rend `false` si le
// point tombe hors de l'image.
func (c MapBackgroundCalibration) MondeVersPixel(x, y float64) (px, py int, ok bool) {
	if c.MetersPerPixel <= 0 {
		return 0, 0, false
	}
	px = int((x - c.OriginX) / c.MetersPerPixel)
	py = int((c.OriginY - y) / c.MetersPerPixel)
	if px < 0 || px >= c.WidthPx || py < 0 || py >= c.HeightPx {
		return px, py, false
	}
	return px, py, true
}
