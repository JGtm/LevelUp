package replay

// CATALOGUE DE CALLOUTS — lecteur du fichier de référence figé par titre
// (data/titles/{slug}/reference/map_callouts.json, cf. PathResolver.MapCalloutsPath).
//
// CE QUE C'EST : les ZONES NOMMÉES officielles des cartes, DEUX SOURCES SOUS UN SEUL
// FICHIER, et une clé par source :
//
//	Maps      clé = MODULE INSTALLÉ (« ridgeline », « ctf_bazaar ») — cartes intégrées, tag
//	          levl du jeu, libellés FR/EN figés dans callouts_i18n.csv (816 zones sur
//	          22 cartes, liaison nom<->volume 816/816).
//	MapsByID  clé = MAP_ID (asset UGC) — cartes FORGE, objets `himap.TypeIDZoneNommee` du
//	          map.mvar de la carte. Même vocabulaire de StringId que les natives.
//
// Le tout est produit par cmd/mapcallouts-build et VERSIONNÉ comme donnée de référence —
// même règle que map_structure, map_quant_bounds et map_objectives — le rejeu d'un match
// reste 100 % hors ligne à la lecture.
//
// POURQUOI DEUX ESPACES DE CLÉS ET PAS UN. Une carte Forge n'a PAS de module : le jeu ne
// range sous `levels/multi/` que les cartes intégrées et les CANEVAS, et un canevas ne
// porte aucune zone nommée (mesuré sur les 8 installés — c'est le zéro qui avait fait
// conclure, à tort, que les cartes Forge n'en avaient pas). Son identité est son asset UGC.
// Mélanger les deux dans une seule table ferait dépendre la lecture d'une devinette sur la
// forme de la clé.
//
// LE LIBELLÉ PEUT MANQUER, ET LA ZONE EST PUBLIÉE QUAND MÊME (EN/FR vides). Le vocabulaire
// des lieux Forge dépasse celui des 22 cartes natives : `callouts_i18n.csv` ne couvre qu'une
// partie des StringId employés. Une zone muette garde sa géométrie MESURÉE ; lui coller un
// libellé de repli (« Zone 7 », le hash) afficherait un nom que le jeu ne prononce pas —
// la règle du chantier est « aucun nom deviné ». Le rendu saute les libellés vides sans
// rien changer (calloutsLayer.ts, drawLabels).

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// MapCalloutsSchemaVersion est la version de forme attendue du catalogue.
//
// ELLE NE BOUGE PAS AVEC L'ARRIVÉE DES CARTES FORGE, ET C'EST DÉLIBÉRÉ : `maps_by_id` est
// une section OPTIONNELLE et purement additive — un lecteur ancien l'ignore, un lecteur neuf
// devant un fichier sans elle dégrade proprement (pas de zones pour les cartes Forge, ce qui
// était déjà l'état du monde). La bumper forcerait une reconstruction complète de la partie
// NATIVE, qui exige le jeu installé ET les fonds publiés, pour une modification qui ne touche
// pas une seule de ses 816 zones.
const MapCalloutsSchemaVersion = 1

// ErrCalloutsUnknownMap signale une carte absente du catalogue de callouts — module inconnu
// (Lookup) ou map_id inconnu (LookupByID). Cas COURANT : toutes les cartes Forge ne sont pas
// extraites, et une carte hors rotation n'a jamais été vue. L'appelant dégrade, il n'échoue
// pas.
var ErrCalloutsUnknownMap = errors.New("replay: carte absente du catalogue de callouts")

// Provenance du polygone livré pour une carte.
const (
	// CalloutsProvenanceBrut : le pavé du designer, lu tel quel dans le tag levl.
	CalloutsProvenanceBrut = "brut"
	// CalloutsProvenanceDecoupe : découpé sur le décor praticable. Ridgeline vient du dump
	// du POC (découpage par ÉTAGE, sur les emprises de map_structure) ; les autres cartes à
	// fond publié viennent de la chaîne universelle (internal/mapdecoupe, masque alpha du
	// fond). Une carte sans fond publié reste `brut` — jamais un découpage deviné.
	CalloutsProvenanceDecoupe = "decoupe"
	// CalloutsProvenanceMvar : le volume posé par le créateur dans Forge, lu dans les objets
	// du map.mvar (boîte ORIENTÉE par son vecteur avant, ou cylindre approché). Jamais
	// découpé : une carte Forge n'a pas de tag levl à confronter, et son fond publié ne borne
	// pas ses zones — mesuré sur Isolation le 2026-08-27, le rognage aux zones coûte une ancre
	// d'objectif, donc les zones ne couvrent pas tout le terrain joué.
	CalloutsProvenanceMvar = "mvar"
)

// MapCalloutsCatalog est le catalogue figé, tel qu'il est sur disque.
type MapCalloutsCatalog struct {
	SchemaVersion int                         `json:"schema_version"`
	TitleSlug     string                      `json:"title_slug"`
	Source        string                      `json:"source"`
	Maps          map[string]MapCalloutsEntry `json:"maps"` // clé = module installé
	// Brut conserve, par module découpé, le PAVÉ DU DESIGNER de chaque zone dont le
	// polygone servi a été rogné sur le décor. Le découpage n'est pas une perte de donnée :
	// `Maps` dit ce qu'on sert, `Brut` garde ce que le jeu déclare.
	//
	// POURQUOI ICI ET PAS DANS LA ZONE. `MapCalloutsEntry` est la charge utile SERVIE au
	// rejeu (contrat OpenAPI) : y ajouter le brut le mettrait sur le réseau à chaque match
	// pour un besoin de traçabilité hors ligne. Le catalogue, lui, n'est jamais servi.
	Brut map[string][]CalloutBrutZone `json:"brut,omitempty"`
	// MapsByID porte les cartes FORGE, clé = map_id (asset UGC). Optionnelle : son absence
	// est l'état d'un catalogue produit avant l'extraction Forge, pas une erreur.
	MapsByID map[string]MapCalloutsEntry `json:"maps_by_id,omitempty"`
}

// CalloutBrutZone est le polygone d'origine d'une zone, conservé après découpage.
type CalloutBrutZone struct {
	VolumeIndex int          `json:"volume_index"`
	Polygon     [][2]float64 `json:"polygon"`
}

// MapCalloutsEntry est l'entrée d'une carte — c'est aussi la charge utile servie au
// rejeu 2D (le service la rend telle quelle, résolue par module comme le fond de carte).
type MapCalloutsEntry struct {
	// Module est le module installé pour une carte intégrée, et RESTE VIDE pour une carte
	// Forge : elle n'en a pas, son identité est la clé de MapsByID (le map_id).
	Module string `json:"module"`
	// Provenance dit d'où viennent les polygones : CalloutsProvenanceBrut,
	// CalloutsProvenanceDecoupe ou CalloutsProvenanceMvar. Une valeur par carte — le
	// producteur ne mélange pas.
	Provenance string        `json:"provenance"`
	Zones      []CalloutZone `json:"zones"`
}

// CalloutZone est une zone nommée, en mètres monde (le repère des trajectoires).
type CalloutZone struct {
	// VolumeIndex est l'indice du volume dans le tag levl pour une carte intégrée — la clé
	// de traçabilité vers callouts_i18n.csv et les dumps de recherche. Pour une carte FORGE
	// c'est le RANG DE L'OBJET dans le map.mvar, qui joue le même rôle : retrouver la zone
	// dans sa source. Les deux espaces ne se croisent jamais (deux catalogues de clés).
	VolumeIndex int `json:"volume_index"`
	// Name est le nom de CONCEPTION (possiblement tronqué à 32 octets par le format). VIDE
	// sur une carte Forge : le map.mvar ne porte que le StringId du lieu, pas son texte.
	Name string `json:"name"`
	// EN et FR sont les libellés joueur officiels (816/816 résolus par string_id sur les
	// cartes intégrées). VIDES quand le StringId de la zone n'a pas encore de texte joueur —
	// cas fréquent sur les cartes Forge, dont le vocabulaire dépasse celui des 22 natives.
	// Le rendu dessine alors le contour SANS libellé : jamais de nom de repli inventé.
	EN string `json:"en"`
	FR string `json:"fr"`
	// X, Y, Z : le point de référence du volume. C'est le centre 3D qu'utilise zoneAt —
	// l'affectation d'un joueur à sa zone se fait en distance 3D, sinon les étages se
	// confondent (règle du POC).
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	// ZBottom et ZTop bornent la tranche verticale habitée du prisme.
	ZBottom float64 `json:"z_bottom"`
	ZTop    float64 `json:"z_top"`
	// Big distingue les GRANDES zones (elles pavent la carte : aplat pair-impair et
	// frontières) des zones FINES (étages imbriqués : contour pointillé sans
	// remplissage). Classement mesuré par recouvrement 2D, étalonné sur Ridgeline
	// contre le POC (cmd/mapcallouts-build).
	Big bool `json:"big,omitempty"`
	// Polygon est le contour principal au sol (sommets XY monde). Vide quand le volume
	// n'a pas de forme propre — le rendu ne dessine alors rien, la zone reste
	// interrogeable par zoneAt.
	Polygon [][2]float64 `json:"polygon,omitempty"`
	// Parts et Holes n'existent qu'en provenance « decoupe » : parties supplémentaires
	// et trous du contour découpé, remplis en règle pair-impair (rendu du POC).
	Parts [][][2]float64 `json:"parts,omitempty"`
	Holes [][][2]float64 `json:"holes,omitempty"`
}

// LoadMapCallouts lit le catalogue figé d'un titre.
func LoadMapCallouts(path string) (*MapCalloutsCatalog, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("catalogue de callouts illisible (%s) : %w", path, err)
	}
	var c MapCalloutsCatalog
	if err := json.Unmarshal(blob, &c); err != nil {
		return nil, fmt.Errorf("catalogue de callouts invalide (%s) : %w", path, err)
	}
	if c.SchemaVersion != MapCalloutsSchemaVersion {
		return nil, fmt.Errorf("catalogue de callouts en version %d, attendu %d (%s)",
			c.SchemaVersion, MapCalloutsSchemaVersion, path)
	}
	return &c, nil
}

// Lookup rend l'entrée d'une carte INTÉGRÉE par son module installé.
//
// ErrCalloutsUnknownMap est le cas NOMINAL d'une carte Forge, qui n'a pas de module :
// l'appelant enchaîne sur LookupByID, il n'échoue jamais.
func (c *MapCalloutsCatalog) Lookup(module string) (MapCalloutsEntry, error) {
	if c == nil || module == "" {
		return MapCalloutsEntry{}, ErrCalloutsUnknownMap
	}
	e, ok := c.Maps[module]
	if !ok {
		return MapCalloutsEntry{}, fmt.Errorf("%w : %q", ErrCalloutsUnknownMap, module)
	}
	return e, nil
}

// LookupByID rend l'entrée d'une carte FORGE par son map_id (asset UGC).
//
// UN map_id VIDE EST UNE ABSENCE, PAS UNE ERREUR : le registre ne nomme pas toujours la
// carte d'un vieux match. Une carte hors catalogue non plus — l'extraction ne couvre que les
// variantes téléchargées.
func (c *MapCalloutsCatalog) LookupByID(mapID string) (MapCalloutsEntry, error) {
	if c == nil || mapID == "" {
		return MapCalloutsEntry{}, ErrCalloutsUnknownMap
	}
	e, ok := c.MapsByID[mapID]
	if !ok {
		return MapCalloutsEntry{}, fmt.Errorf("%w : map_id %q", ErrCalloutsUnknownMap, mapID)
	}
	return e, nil
}
