package replaydoc

// map.go — LES CALQUES DE CARTE servis avec le rejeu : objectifs statiques, emplacements de
// socle, structure, et les deux corps de route qui vivent a cote du document
// (`MapBackground` pour le calage du fond, `MapCalloutsEntry` pour les zones nommees).

import (
	"time"

	"levelup/go-api/internal/i18n"
)

// MapObjectives est le calque statique servi avec le rejeu.
type MapObjectives struct {
	Zones   []ObjectiveZoneDTO   `json:"zones,omitempty"`
	Markers []ObjectiveMarkerDTO `json:"markers,omitempty"`
}

// ObjectiveZoneDTO est une zone d'objectif prête à dessiner : centre monde, forme en
// demi-extents (la conversion tailles-pleines -> demi est déjà faite côté producteur) et
// orientation projetée au plan.
type ObjectiveZoneDTO struct {
	Role   string  `json:"role"`
	Team   int     `json:"team"`
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Z      float32 `json:"z"`
	Family string  `json:"family"`
	HalfX  float32 `json:"halfX,omitempty"`
	HalfY  float32 `json:"halfY,omitempty"`
	Radius float32 `json:"radius,omitempty"`
	FwdX   float32 `json:"fwdX"`
	FwdY   float32 `json:"fwdY"`
}

// ObjectiveMarkerDTO est un objectif ponctuel prêt à dessiner.
type ObjectiveMarkerDTO struct {
	Role string  `json:"role"`
	Team int     `json:"team"`
	X    float32 `json:"x"`
	Y    float32 `json:"y"`
	Z    float32 `json:"z"`
}

// MapWeaponPads est le calque des emplacements de socle servi avec le rejeu.
type MapWeaponPads struct {
	Pads     []MapWeaponPadDTO `json:"pads"`
	CatalogN int               `json:"catalogN"`
}

// MapWeaponPadDTO est UN emplacement allumé : la position du fichier de carte, le socle du
// match qui l'a confirmé, et LA NATURE DE L'EMPLACEMENT.
type MapWeaponPadDTO struct {
	X   float32 `json:"x"`
	Y   float32 `json:"y"`
	Z   float32 `json:"z,omitempty"`
	Pad int     `json:"pad"`
	// Family : `rack` (arme de terrain), `power` (arme de puissance) ou `powerup` (bonus).
	// Elle vient du fichier de carte Forge, JAMAIS du nom de l'arme : la même Hydra est de
	// terrain sur une carte et de puissance sur une autre. Un socle du match sans entrée ici
	// n'est confirmé par aucun emplacement — son niveau est « non classé ».
	Family string `json:"family"`
}

// MapObject est un prop Forge projeté en 2D : centre orienté + emprise de sa bounding box.
// Ce sont de PETITS objets (0,25 m² en moyenne) — décor et repères, pas les sols/murs.
type MapObject struct {
	TypeID int64   `json:"typeId"`
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Z      float32 `json:"z,omitempty"`
	DX     float32 `json:"dx,omitempty"`
	DY     float32 `json:"dy,omitempty"`
	Yaw    float32 `json:"yaw,omitempty"`
}

// Surface est l'emprise au sol d'un élément de structure : la projection sur (x, y) de
// l'AABB MONDE d'une instance de géométrie instanciée, plus les altitudes de ses faces
// haute et basse. Rectangle aligné sur les axes — PAS le maillage : le lien instance ->
// géométrie n'est pas résolu (layout du champ meshRef inconnu), on publie donc la boîte
// englobante et rien de plus. Une boîte de plateforme ou de mur suffit à une carte
// reconnaissable en vue de dessus ; elle ne rend pas les formes courbes.
type Surface struct {
	X0   float32      `json:"x0"`
	Y0   float32      `json:"y0"`
	X1   float32      `json:"x1"`
	Y1   float32      `json:"y1"`
	Z    float32      `json:"z"`
	ZB   float32      `json:"zb"`
	Poly [][2]float32 `json:"poly,omitempty"`
}

// MapBackground est le CALAGE du fond de carte servi par
// `GET /players/{slug}/matches/{id}/replay/background` : ou l'image se pose dans le repere
// monde, celui-la meme ou vivent les trajectoires, plus ce que sa fabrication a mesure. Il
// va toujours avec l'image PNG servie par la route soeur `.../background.png`.
type MapBackground struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Module        string                   `json:"module"`
	MapNames      []string                 `json:"mapNames,omitempty"`
	Image         string                   `json:"image"`
	Source        string                   `json:"source"`
	GeneratedAt   time.Time                `json:"generatedAt"`
	Style         string                   `json:"style"`
	Calibration   MapBackgroundCalibration `json:"calibration"`
	Stats         MapBackgroundStats       `json:"stats"`
	Degradations  []string                 `json:"degradations,omitempty"`
	// Plan est le plan des murs tiré du maillage de navigation de la carte : absent quand la
	// carte n'a pas de maillage ou que son fond a été cuit sans.
	Plan *MapBackgroundPlan `json:"plan,omitempty"`
	// PlanImage : nom du fichier de l'image plan de la carte (même cadre et même calage que
	// l'image du fond), servie par la route de l'image du fond avec `?variante=plan`. Vide quand
	// la carte n'a pas de plan : la route répond alors comme pour un fond absent.
	PlanImage string `json:"planImage,omitempty"`
}

// MapBackgroundPlan est le plan d'une carte tiré de son maillage de navigation : les arêtes de
// bord, celles que UNE seule face porte, donc les limites du praticable (murs, vides, bords de
// plateforme). Les passages entre deux faces n'y figurent pas.
type MapBackgroundPlan struct {
	// Source nomme le maillage dont le plan est tiré (« navmesh module any » pour une carte
	// native, « navmesh.blob » pour une carte Forge).
	Source string `json:"source"`
	// Vertices et Faces : taille du maillage entier, pas du plan.
	Vertices int `json:"vertices"`
	Faces    int `json:"faces"`
	// Edges : une arête par ligne, `[x1, y1, x2, y2]`, en mètres dans le repère monde des
	// trajectoires (celui de `calibration`), au millimètre.
	Edges [][4]float64 `json:"edges"`
}

// MapBackgroundCalibration place l'image dans le repère monde.
type MapBackgroundCalibration struct {
	MetersPerPixel float64 `json:"metersPerPixel"`
	OriginX        float64 `json:"originX"`
	OriginY        float64 `json:"originY"`
	WidthPx        int     `json:"widthPx"`
	HeightPx       int     `json:"heightPx"`
	Convention     string  `json:"convention"`
}

// MapBackgroundStats chiffre la fabrication du fond : sur quoi le calage s'appuie, ce qui a
// ete substitue ou rogne. Publie pour que le client puisse dire « fond approximatif » plutot
// que d'afficher une superposition fausse en silence.
type MapBackgroundStats struct {
	Anchors                  int      `json:"anchors"`
	AnchorsInFrame           int      `json:"anchorsInFrame"`
	AnchorsWithGround        int      `json:"anchorsWithGround"`
	AnchorMedianGapM         *float64 `json:"anchorMedianGapM,omitempty"`
	InstancesDrawn           int      `json:"instancesDrawn"`
	InstancesScenery         int      `json:"instancesScenery"`
	PlayLevelZ               float64  `json:"playLevelZ"`
	BoundaryApplied          bool     `json:"boundaryApplied"`
	BoundaryPlanes           int      `json:"boundaryPlanes"`
	BoundaryCellsCleared     int      `json:"boundaryCellsCleared"`
	WaterVolumes             int      `json:"waterVolumes"`
	WaterCells               int      `json:"waterCells"`
	CoveredShare             float64  `json:"coveredShare"`
	Covered                  bool     `json:"covered"`
	CellsSubstituted         int      `json:"cellsSubstituted,omitempty"`
	CellsClipped             int      `json:"cellsClipped,omitempty"`
	CellsAssumedFloor        int      `json:"cellsAssumedFloor,omitempty"`
	ForgeObjects             int      `json:"forgeObjects,omitempty"`
	ForgeObjectsDrawn        int      `json:"forgeObjectsDrawn,omitempty"`
	ForgeObjectsWithoutModel int      `json:"forgeObjectsWithoutModel,omitempty"`
	ForgeDeathVolumes        int      `json:"forgeDeathVolumes,omitempty"`
}

// MapCalloutsEntry porte les ZONES NOMMEES d'une carte : le corps de
// `GET /players/{slug}/matches/{id}/replay/callouts`. Le service resout la carte du match
// (module, puis map_id au catalogue versionne et au catalogue genere) puis PROJETTE l'entree
// trouvee sur cette forme. 404 quand la carte n'en a pas au catalogue.
type MapCalloutsEntry struct {
	Module     string        `json:"module"`
	Provenance string        `json:"provenance"`
	Zones      []CalloutZone `json:"zones"`
}

// CalloutZone est une zone nommée, en mètres monde (le repère des trajectoires).
type CalloutZone struct {
	VolumeIndex int            `json:"volume_index"`
	Name        string         `json:"name"`
	Text        i18n.Text      `json:"i18n_text,omitempty" required:"true"`
	X           float64        `json:"x"`
	Y           float64        `json:"y"`
	Z           float64        `json:"z"`
	ZBottom     float64        `json:"z_bottom"`
	ZTop        float64        `json:"z_top"`
	Big         bool           `json:"big,omitempty"`
	Polygon     [][2]float64   `json:"polygon,omitempty"`
	Parts       [][][2]float64 `json:"parts,omitempty"`
	Holes       [][][2]float64 `json:"holes,omitempty"`
}

// WeaponTiersInfo — ce que le match dit des NIVEAUX D'ARME, résolu à la requête.
// Jumeau de `replay.WeaponTiersInfo`.
type WeaponTiersInfo struct {
	// RandomStarts : le mode distribue les équipements de début de vie AU HASARD. Le niveau
	// « arme de base » n'y est pas publié.
	RandomStarts bool `json:"randomStarts"`
}
