package filmdec

// BORNES DE DÉQUANTIFICATION PAR CARTE.
//
// Le film ne porte QUE des indices de quantum. La conversion en coordonnées monde exige
// l'AABB du BSP de la carte (`world bounds x/y/z` du tag scenario_structure_bsp), qui ne
// vit que dans le module de la carte. Ces bornes sont figées dans un catalogue JSON
// versionné (produit hors ligne par cmd/mapquant-build depuis les .module) et chargé ici.
//
// RÈGLE : pas de bornes -> PAS de coordonnée monde. Une valeur fausse silencieuse est pire
// qu'une erreur : jusqu'ici toutes les cartes étaient déquantifiées avec les bornes de
// Cliffhanger, ce qui multipliait l'échelle par un facteur arbitraire (0,38 sur Catalyst)
// et décalibrait d'autant le filtre de téléportation en m/s.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrUnknownMapBounds signale que la carte n'a pas de bornes connues : aucune coordonnée
// monde ne peut être produite.
var ErrUnknownMapBounds = errors.New("filmdec: bornes de quantification inconnues pour cette carte")

// MapQuantSchemaVersion est la version du catalogue. Incrémentée si la forme change.
const MapQuantSchemaVersion = 1

// MapQuantEntry est l'AABB monde du BSP principal d'une carte, plus la largeur d'axe que
// la loi du moteur en déduit (contrôle de cohérence, pas une donnée d'entrée).
type MapQuantEntry struct {
	// Module est le dossier du .module d'où viennent les bornes (traçabilité).
	Module string `json:"module"`
	// Min / Max sont les bornes monde par axe (X, Y, Z).
	Min [3]float32 `json:"min"`
	Max [3]float32 `json:"max"`
	// AxisWidths est W = min(26, ceilLog2(ceil(60*extent))) par axe, DÉDUIT des bornes par
	// la loi du moteur. C'est une valeur de contrôle : elle doit égaler le découpage lu
	// dans le film par DetectI0Layout, sans quoi les bornes ne sont pas celles de la carte.
	AxisWidths [3]uint `json:"axisWidths"`
}

// Range convertit l'entrée en plage de déquantification.
func (e MapQuantEntry) Range() Vec3Range {
	var r Vec3Range
	for ax := 0; ax < 3; ax++ {
		r[ax] = AxisRange{Min: e.Min[ax], Max: e.Max[ax]}
	}
	return r
}

// MapQuantCatalog est le catalogue des bornes, indexé par nom de carte normalisé.
type MapQuantCatalog struct {
	SchemaVersion int `json:"schemaVersion"`
	// Source documente d'où viennent les bornes (build du jeu, chaîne d'extraction).
	Source string `json:"source"`
	// Maps est indexé par NormalizeMapName(nom affiché).
	Maps map[string]MapQuantEntry `json:"maps"`
}

// LoadMapQuantCatalog lit le catalogue JSON.
func LoadMapQuantCatalog(path string) (*MapQuantCatalog, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("catalogue de bornes illisible (%s) : %w", path, err)
	}
	var c MapQuantCatalog
	if err := json.Unmarshal(blob, &c); err != nil {
		return nil, fmt.Errorf("catalogue de bornes invalide (%s) : %w", path, err)
	}
	if c.SchemaVersion != MapQuantSchemaVersion {
		return nil, fmt.Errorf("catalogue de bornes en version %d, attendu %d (%s)",
			c.SchemaVersion, MapQuantSchemaVersion, path)
	}
	return &c, nil
}

// Lookup renvoie l'entrée d'une carte à partir de son nom affiché.
func (c *MapQuantCatalog) Lookup(mapName string) (MapQuantEntry, error) {
	if c == nil {
		return MapQuantEntry{}, ErrUnknownMapBounds
	}
	e, ok := c.Maps[NormalizeMapName(mapName)]
	if !ok {
		return MapQuantEntry{}, fmt.Errorf("%w : %q", ErrUnknownMapBounds, mapName)
	}
	return e, nil
}

// variantSuffixes : suffixes accolés au nom de carte côté API pour désigner une variante
// de PLAYLIST ou de SANDBOX. Aucun ne change la géométrie, donc aucun ne change les
// bornes monde. Retirés dans cet ordre (le second peut suivre le premier).
//
//   - « - Ranked » : playlist classée, même carte.
//   - « Heavies » : réglage d'armes et de véhicules. Vérifié sur pièces le 2026-08-08 —
//     dans map_objectives.json, Fragmentation / Highpower / Breaker et leur variante
//     Heavies portent le MÊME level_id (l'identifiant de niveau moteur, root[1][0][0] du
//     .mvar), et les trois zones de Bastion de Fragmentation Heavies sont au centimètre
//     sur celles de Fragmentation. Même niveau => même BSP => mêmes bornes.
//     Couverture regagnée sur le registre : 43 matchs (22 Fragmentation, 11 Highpower,
//     10 Breaker).
var variantSuffixes = []string{" - ranked", " heavies"}

// NormalizeMapName met le nom de carte sous sa forme de clé : minuscules, espaces
// resserrés, suffixes de variante retirés.
func NormalizeMapName(s string) string {
	n := strings.ToLower(strings.Join(strings.Fields(s), " "))
	for _, suf := range variantSuffixes {
		n = strings.TrimSuffix(n, suf)
	}
	return n
}
