package replay

// CATALOGUE DES EMPLACEMENTS DE SOCLE — la FORME du fichier de référence figé par titre
// (data/titles/{slug}/reference/map_weapon_pads.json, cf. PathResolver.MapWeaponPadsPath),
// et son lecteur.
//
// D'OU IL VIENT. Les trois type_id de socle du `.mvar` d'une carte, extraits hors ligne par
// cmd/mapopads-build depuis le dépôt local de variantes. Mesure du plan
// `.ai/V7.5/replay2d/PLAN_SOCLES_MVAR.md` : 32 positions d'oracle sur trois cartes, 32
// appariées à moins d'un mètre, médiane 0,01 m.
//
// CE QU'IL DIT, ET CE QU'IL NE DIT PAS. Il dit OU sont TOUS les socles d'une carte — y
// compris les 22 emplacements que le film ne montre jamais (7 sur Cliffhanger, 15 sur
// Smallhalla). Il ne dit NI ce qui y apparaît (un même objet porte l'épée ou le marteau
// selon le match), NI s'ils sont allumés : le fichier de carte POSE, le mode ALLUME —
// Cliffhanger porte 17 socles, en rend 10 en CTF et ZÉRO en Super Fiesta.
//
// D'OU LA RÈGLE, NON NÉGOCIABLE : ce catalogue ne se sert JAMAIS seul au client. Le
// croisement avec les socles du match vit dans map_weapon_pads.go, et lui seul décide de ce
// qui part à l'écran. Publier ce fichier brut afficherait 17 socles fantômes sur un rejeu
// de Fiesta.
//
// MÊME PATRON QUE map_objectives.json (objectives_catalog.go) : production hors ligne,
// résultat VERSIONNÉ, clé = map_id, jointure par map_id SEUL, rejeu 100 % hors ligne.

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// MapWeaponPadsSchemaVersion est la version de forme attendue du catalogue.
//
//	v1 (2026-08-19) : par carte, la liste des emplacements — position monde, type_id brut
//	en hexadécimal, famille DÉRIVÉE publiée à côté du brut, et le nombre d'objets du
//	fichier fusionnés dans l'emplacement.
const MapWeaponPadsSchemaVersion = 1

// MapWeaponPadsCatalog est le catalogue figé, tel qu'il est sur disque.
type MapWeaponPadsCatalog struct {
	SchemaVersion int                           `json:"schema_version"`
	TitleSlug     string                        `json:"title_slug"`
	GeneratedAt   time.Time                     `json:"generated_at"`
	Maps          map[string]MapWeaponPadsEntry `json:"maps"` // clé = map_id (asset UGC)
	Notes         map[string]string             `json:"notes,omitempty"`
}

// MapWeaponPadsEntry est l'entrée d'une carte.
type MapWeaponPadsEntry struct {
	MapID      string `json:"map_id"`
	PublicName string `json:"public_name,omitempty"`
	// Module et MvarFile disent DE QUOI cette entrée est faite : le dossier de niveau et le
	// fichier de variante lu. Ils sont la trace de production, et le seul moyen de rejouer
	// l'extraction sur le même fichier — sur une carte Forge, deux fichiers cohabitent
	// (canevas et rack) et un seul porte les socles.
	Module   string             `json:"module,omitempty"`
	MvarFile string             `json:"mvar_file"`
	LevelID  int32              `json:"level_id"`
	ObjectsN int                `json:"objects_n"`
	Pads     []MapWeaponPadSpot `json:"pads"`
}

// MapWeaponPadSpot est UN emplacement de socle de la carte.
//
// LE TYPE BRUT RESTE À CÔTÉ DE LA FAMILLE, jamais remplacé par elle (règle du dépôt : on ne
// stocke jamais une résolution qui peut s'améliorer). La famille est une INFÉRENCE mesurée
// par corrélation avec les armes observées ; le jour où elle est infirmée, on la recalcule
// depuis `type_id` sans ré-extraire un seul `.mvar`.
type MapWeaponPadSpot struct {
	Pos mapvar.Vec3 `json:"pos"`
	// TypeID est le type_id BRUT du fichier, en hexadécimal (ex. `0x5F379533`) — l'écriture
	// sous laquelle il se reconnaît d'un dump à l'autre.
	TypeID string `json:"type_id"`
	// Family : `power` (arme de pouvoir), `rack` (arme de râtelier) ou `powerup`.
	Family string `json:"family"`
	// Objects est le nombre d'objets du fichier FUSIONNÉS dans cet emplacement (deux
	// déclarations à quelques centimètres = un seul socle). Un partout sauf sur les
	// doublons mesurés de Catalyst.
	Objects int `json:"objects"`
}

// LoadMapWeaponPads lit le catalogue figé d'un titre.
func LoadMapWeaponPads(path string) (*MapWeaponPadsCatalog, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("catalogue des socles illisible (%s) : %w", path, err)
	}
	var c MapWeaponPadsCatalog
	if err := json.Unmarshal(blob, &c); err != nil {
		return nil, fmt.Errorf("catalogue des socles invalide (%s) : %w", path, err)
	}
	if c.SchemaVersion != MapWeaponPadsSchemaVersion {
		return nil, fmt.Errorf("catalogue des socles en version %d, attendu %d (%s)",
			c.SchemaVersion, MapWeaponPadsSchemaVersion, path)
	}
	return &c, nil
}

// Lookup rend l'entrée d'une carte par son map_id (asset UGC). ErrUnknownMap est le cas
// NOMINAL : le catalogue ne couvre que les cartes dont un `.mvar` est au dépôt local.
func (c *MapWeaponPadsCatalog) Lookup(mapID string) (MapWeaponPadsEntry, error) {
	if c == nil {
		return MapWeaponPadsEntry{}, ErrUnknownMap
	}
	e, ok := c.Maps[mapID]
	if !ok {
		return MapWeaponPadsEntry{}, fmt.Errorf("%w : %q", ErrUnknownMap, mapID)
	}
	return e, nil
}
