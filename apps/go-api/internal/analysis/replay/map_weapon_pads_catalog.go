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
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
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
	// SpawnPoints est la SECONDE liste : les points d'apparition d'objet ramassable NON-ARME
	// que la recette (`himap.EstPointDApparition`) reconnait dans le catalogue Forge du jeu.
	//
	// LISTE SEPAREE, ET C'EST LA GARANTIE DE NON-REGRESSION. `Pads` alimente des chemins
	// livres — datation des occupations de socle, tableau de la page match. Ajouter les
	// points d'apparition DEDANS aurait mele deux natures dans un tableau dont des clients
	// lisent deja chaque element. Ici, un lecteur qui ignore ce champ voit exactement ce
	// qu'il voyait.
	//
	// C'EST UN POINTEUR, ET C'EST LA CORRECTION D'UN PIEGE PAYE. Avec une tranche nue et
	// `omitempty`, une carte ACCEPTEE mais sans aucun point n'ecrivait PAS la cle — donc elle
	// devenait indiscernable d'une carte SAUTEE pour derive de source. Les SEIZE cartes sautees
	// (Deadlock, Fragmentation, Highpower, Oasis, Breaker, Scarr...) se lisaient alors « carte
	// connue, aucun point », c'est-a-dire un mensonge : leur catalogue de points n'est PAS
	// ETABLI, et `spawner` y est impossible.
	//
	// LES TROIS ETATS SE LISENT DONC DANS LA FORME MEME DU JSON :
	//
	//	entree absente du fichier   la carte n'est pas au catalogue
	//	cle `spawn_points` ABSENTE  carte connue, points NON ETABLIS (sautee pour derive)
	//	cle presente, meme `[]`     carte connue, points ETABLIS — `[]` veut dire « aucun »
	//
	// `omitempty` sur un POINTEUR n'omet que le nil : une tranche vide non-nil s'ecrit `[]`.
	// C'est exactement la distinction voulue, et c'est pourquoi la tranche nue ne convenait pas.
	SpawnPoints *[]MapSpawnPointSpot `json:"spawn_points,omitempty"`
}

// MapSpawnPointSpot est UN point d'apparition d'objet ramassable de la carte.
//
// MEME REGLE QUE POUR LES SOCLES : le type brut reste a cote de la nature, jamais remplace par
// elle. La nature vient d'une MESURE (canal natif des ramassages sur un film), pas d'une
// lecture des fichiers du jeu — le jour ou une seconde carte l'infirme, on la recalcule depuis
// `type_id` sans re-extraire un seul `.mvar`.
type MapSpawnPointSpot struct {
	Pos mapvar.Vec3 `json:"pos"`
	// TypeID est le type_id BRUT du fichier, en hexadecimal.
	TypeID string `json:"type_id"`
	// Kind : `grenade`, `equipment`, ou `unknown` quand aucune mesure ne dit ce que le point
	// fait naitre. `unknown` est une valeur PLEINE, pas un trou : la recette affirme que c'est
	// un point d'apparition, et seule sa nature manque.
	Kind string `json:"kind"`
	// Objects est le nombre d'objets du fichier FUSIONNES dans ce point.
	Objects int `json:"objects"`
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

// LoadMapWeaponPadsMerged lit le catalogue VERSIONNÉ puis y superpose l'OVERLAY NON VERSIONNÉ
// des cartes rattrapées par le runtime (cf. PathResolver.MapWeaponPadsOverlayPath).
//
// L'ENTRÉE VERSIONNÉE PRIME, TOUJOURS. Une carte présente des deux côtés garde la version du
// fichier suivi par git : elle a été produite à la main par `cmd/mapopads-build` et relue, alors
// que l'overlay est une sortie de runtime que personne n'a regardée. Le sens de la préséance
// n'est pas un détail — l'inverser ferait remplacer une donnée relue par une donnée automatique.
//
// LES TROIS ÉTATS DE L'OVERLAY, ET CE QU'ILS VALENT :
//
//	ABSENT      cas NOMINAL — rien n'a encore été rattrapé sur ce titre. Le catalogue versionné
//	            est rendu tel quel, sans un mot.
//	ILLISIBLE   on JOURNALISE puis on dégrade au seul catalogue versionné. Un overlay corrompu
//	            est un fichier jetable : il ne doit jamais faire perdre les 72+ cartes relues.
//	LISIBLE     ses cartes ABSENTES du versionné sont ajoutées, et ça se journalise en Debug.
//
// Le catalogue VERSIONNÉ, lui, reste obligatoire : son absence est une installation incomplète,
// et l'erreur remonte à l'appelant comme avant (les deux appelants la dégradent, chacun avec sa
// trace).
func LoadMapWeaponPadsMerged(versionne, overlay string) (*MapWeaponPadsCatalog, error) {
	cat, err := LoadMapWeaponPads(versionne)
	if err != nil {
		return nil, err
	}
	sur, err := LoadMapWeaponPads(overlay)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return cat, nil
	case err != nil:
		slog.Warn("catalogue des socles : overlay illisible — seules les cartes versionnees "+
			"sont servies ; le rattrapage au fetch de film les rajoutera",
			"err", err, "overlay", overlay)
		return cat, nil
	}
	if cat.Maps == nil {
		cat.Maps = map[string]MapWeaponPadsEntry{}
	}
	ajoutees := 0
	for id, e := range sur.Maps {
		if _, deja := cat.Maps[id]; deja {
			// L'entrée versionnée prime : on ne la remplace pas, et ce n'est pas une anomalie
			// (la carte a pu être promue au fichier versionné après son rattrapage).
			continue
		}
		cat.Maps[id] = e
		ajoutees++
	}
	if ajoutees > 0 {
		slog.Debug("catalogue des socles : cartes rattrapees superposees au catalogue versionne",
			"ajoutees", ajoutees, "versionnees", len(cat.Maps)-ajoutees, "overlay", overlay)
	}
	return cat, nil
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
