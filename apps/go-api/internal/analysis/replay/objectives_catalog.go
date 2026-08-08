package replay

// CATALOGUE D'OBJECTIFS — lecteur du fichier de référence figé par titre
// (data/titles/{slug}/reference/map_objectives.json, cf. PathResolver.MapObjectivesPath).
//
// Ce fichier était jusqu'ici PRODUIT (cmd/mapobj-build) et jamais LU : 34 cartes,
// 63 zones de Bastion et 161 zones d'Extraction versionnées sans aucun consommateur Go.
// C'est ce lecteur qui les branche.
//
// RÈGLE, la même que map_structure et map_quant_bounds : la production exige un accès
// réseau aux variantes UGC, donc le résultat est VERSIONNÉ comme donnée de référence et
// jamais régénéré à la volée. Le rejeu d'un match reste 100 % hors ligne.
//
// CE QUE LE CATALOGUE NE CONTIENT PAS : le nom des zones. Les trois zones de Bastion
// d'une même carte portent le même type_id, les mêmes labels et le même hachage ; elles
// ne diffèrent que par position, dimensions et team_index (mapvar/objectives.go). La
// lettre A/B/C affichée en jeu n'est donc PAS dans le fichier, et ce lecteur ne
// l'invente pas — voir ZoneSet.Zones et le champ SpatialRank.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// MapObjectivesSchemaVersion est la version de forme attendue du catalogue. Elle doit
// suivre catalogSchemaVersion de cmd/mapobj-build : en v2 chaque objectif surfacique
// porte son champ `shape`.
const MapObjectivesSchemaVersion = 2

// ErrUnknownMap signale une carte absente du catalogue. C'est le cas NOMINAL et
// fréquent : le catalogue couvre 34 cartes sur la centaine jouée. L'appelant doit
// dégrader (rejeu sans zones), jamais échouer.
var ErrUnknownMap = errors.New("replay: carte absente du catalogue d'objectifs")

// MapObjectivesCatalog est le catalogue figé, tel qu'il est sur disque.
type MapObjectivesCatalog struct {
	SchemaVersion int                           `json:"schema_version"`
	TitleSlug     string                        `json:"title_slug"`
	GeneratedAt   time.Time                     `json:"generated_at"`
	Maps          map[string]MapObjectivesEntry `json:"maps"` // clé = map_id (asset UGC)
}

// MapObjectivesEntry est l'entrée d'une carte.
type MapObjectivesEntry struct {
	MapID      string             `json:"map_id"`
	VersionID  string             `json:"version_id"`
	PublicName string             `json:"public_name"`
	Module     string             `json:"module"`
	ObjectsN   int                `json:"objects_n"`
	Objectives []mapvar.Objective `json:"objectives"`
	// CarriedFromSchema : la carte a été REPORTÉE d'un schéma antérieur sans re-parse.
	// PIÈGE documenté par le producteur : en v2 une zone non migrée n'a pas de `shape`,
	// exactement comme un objectif ponctuel. Confondre les deux ferait passer une
	// absence de migration pour une mesure de ponctualité — d'où le comptage séparé
	// dans ZoneSet.Carried.
	CarriedFromSchema int `json:"carried_from_schema,omitempty"`
}

// LoadMapObjectives lit le catalogue figé d'un titre.
func LoadMapObjectives(path string) (*MapObjectivesCatalog, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("catalogue d'objectifs illisible (%s) : %w", path, err)
	}
	var c MapObjectivesCatalog
	if err := json.Unmarshal(blob, &c); err != nil {
		return nil, fmt.Errorf("catalogue d'objectifs invalide (%s) : %w", path, err)
	}
	if c.SchemaVersion != MapObjectivesSchemaVersion {
		return nil, fmt.Errorf("catalogue d'objectifs en version %d, attendu %d (%s)",
			c.SchemaVersion, MapObjectivesSchemaVersion, path)
	}
	return &c, nil
}

// Lookup rend l'entrée d'une carte par son map_id (asset UGC).
func (c *MapObjectivesCatalog) Lookup(mapID string) (MapObjectivesEntry, error) {
	if c == nil {
		return MapObjectivesEntry{}, ErrUnknownMap
	}
	e, ok := c.Maps[mapID]
	if !ok {
		return MapObjectivesEntry{}, fmt.Errorf("%w : %q", ErrUnknownMap, mapID)
	}
	return e, nil
}

// Zone est une zone d'objectif POSÉE, prête pour le test d'appartenance.
type Zone struct {
	Role mapvar.Role
	// InstanceID est l'identifiant que le JEU donne à l'objet. C'est la seule identité
	// de zone qui ne soit pas de notre invention — la clé à porter partout.
	InstanceID int32
	// ObjectIdx est le rang de l'objet dans le fichier. Conservé pour la traçabilité,
	// JAMAIS pour nommer la zone : l'ordre du fichier n'est pas l'ordre du jeu.
	ObjectIdx int
	// TeamIndex est l'index d'équipe brut du fichier ; -1 = neutre (cas de toutes les
	// zones de Bastion, qui n'appartiennent à personne au départ).
	TeamIndex int
	// SpatialRank est un rang STABLE dérivé de la position (tri x, puis y, puis z),
	// 0-based.
	//
	// CE N'EST PAS LA LETTRE DU JEU. Le catalogue ne porte aucun nom de zone, et
	// l'ordre du fichier n'en est pas un non plus. Ce rang sert à désigner la même
	// zone de la même façon d'une exécution à l'autre — rien de plus. Le jour où la
	// lettre A/B/C sera établie par un témoin (relevé Theater), elle se posera dans un
	// champ DISTINCT, sans écraser celui-ci.
	SpatialRank int
	Center      mapvar.Vec3
	Volume      mapvar.Volume
}

// ZoneSet est l'ensemble des zones exploitables d'une carte, avec ce qui a été écarté.
//
// Les compteurs ne sont pas décoratifs : publier des zones sans dire combien ont été
// laissées de côté laisserait croire à l'exhaustivité (même règle que le champ Coverage
// du rejeu 2D).
type ZoneSet struct {
	Zones []Zone
	// Pointless compte les objectifs du rôle demandé qui n'ont PAS de forme. Cas normal
	// pour un rôle ponctuel (apparition de drapeau) ; anormal pour un rôle surfacique.
	Pointless int
	// Degenerate compte les formes dont l'orientation ne permet pas de construire un
	// repère. Mesuré : 0 sur les 224 formes de zone du catalogue.
	Degenerate int
	// Carried signale que la carte vient d'un schéma antérieur : ses objectifs sans
	// forme sont NON MIGRÉS, pas ponctuels (cf. CarriedFromSchema).
	Carried bool
}

// ZonesOfRole pose les objectifs d'un rôle donné en volumes testables.
//
// L'ordre de sortie est celui de SpatialRank, donc déterministe et indépendant de
// l'ordre du fichier.
func (e MapObjectivesEntry) ZonesOfRole(role mapvar.Role) ZoneSet {
	set := ZoneSet{Carried: e.CarriedFromSchema != 0}
	for _, o := range e.Objectives {
		if o.Role != role {
			continue
		}
		vol, err := mapvar.NewVolume(o.Pos, o.Shape)
		switch {
		case errors.Is(err, mapvar.ErrNoShape):
			set.Pointless++
			continue
		case err != nil:
			set.Degenerate++
			continue
		}
		set.Zones = append(set.Zones, Zone{
			Role:       o.Role,
			InstanceID: o.InstanceID,
			ObjectIdx:  o.ObjectIdx,
			TeamIndex:  o.TeamIndex,
			Center:     o.Pos,
			Volume:     vol,
		})
	}
	sortZonesSpatially(set.Zones)
	return set
}

// sortZonesSpatially trie par position puis fige SpatialRank.
//
// Le tri retombe sur InstanceID en dernier recours : deux zones exactement superposées
// resteraient sinon dans un ordre dépendant de l'ordre d'entrée, et le rang cesserait
// d'être stable — ce qui est précisément ce qu'il promet.
func sortZonesSpatially(zs []Zone) {
	sort.SliceStable(zs, func(i, j int) bool {
		a, b := zs[i], zs[j]
		if a.Center.X != b.Center.X {
			return a.Center.X < b.Center.X
		}
		if a.Center.Y != b.Center.Y {
			return a.Center.Y < b.Center.Y
		}
		if a.Center.Z != b.Center.Z {
			return a.Center.Z < b.Center.Z
		}
		return a.InstanceID < b.InstanceID
	})
	for i := range zs {
		zs[i].SpatialRank = i
	}
}
