package replay

// CATALOGUE DE CALLOUTS — lecteur du fichier de référence figé par titre
// (data/titles/{slug}/reference/map_callouts.json, cf. PathResolver.MapCalloutsPath).
//
// CE QUE C'EST : les ZONES NOMMÉES officielles des cartes intégrées (tag levl du jeu,
// libellés FR/EN résolus hors ligne depuis le tag uslg — 816 zones sur 22 cartes,
// liaison nom<->volume 816/816). Produit par cmd/mapcallouts-build, qui exige le jeu
// installé : le résultat est VERSIONNÉ comme donnée de référence — même règle que
// map_structure, map_quant_bounds et map_objectives — et le rejeu d'un match reste
// 100 % hors ligne.
//
// LA CLÉ EST LE MODULE INSTALLÉ (« ridgeline », « ctf_bazaar »), celui que porte déjà
// map_quant_bounds.json : le lien nom affiché -> module reste déclaré à UN seul endroit.
// Une carte Forge ne résout jamais vers une entrée : son canevas ne porte AUCUNE zone
// nommée (mesuré sur les 8 canevas installés) et n'est donc pas au catalogue — l'absence
// est un fait de construction, pas une lacune.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// MapCalloutsSchemaVersion est la version de forme attendue du catalogue.
const MapCalloutsSchemaVersion = 1

// ErrCalloutsUnknownMap signale un module absent du catalogue de callouts. Cas NOMINAL :
// le catalogue couvre les 22 cartes intégrées, jamais les cartes Forge (leur canevas ne
// porte aucune zone nommée — par construction). L'appelant dégrade, il n'échoue pas.
var ErrCalloutsUnknownMap = errors.New("replay: carte absente du catalogue de callouts")

// Provenance du polygone livré pour une carte.
const (
	// CalloutsProvenanceBrut : le pavé du designer, lu tel quel dans le tag levl.
	CalloutsProvenanceBrut = "brut"
	// CalloutsProvenanceDecoupe : découpé sur le sol praticable (Ridgeline seule — le
	// découpage universel dépend du chantier cartes, au registre des reports).
	CalloutsProvenanceDecoupe = "decoupe"
)

// MapCalloutsCatalog est le catalogue figé, tel qu'il est sur disque.
type MapCalloutsCatalog struct {
	SchemaVersion int                         `json:"schema_version"`
	TitleSlug     string                      `json:"title_slug"`
	Source        string                      `json:"source"`
	Maps          map[string]MapCalloutsEntry `json:"maps"` // clé = module installé
}

// MapCalloutsEntry est l'entrée d'une carte — c'est aussi la charge utile servie au
// rejeu 2D (le service la rend telle quelle, résolue par module comme le fond de carte).
type MapCalloutsEntry struct {
	Module string `json:"module"`
	// Provenance dit d'où viennent les polygones : CalloutsProvenanceBrut ou
	// CalloutsProvenanceDecoupe. Une valeur par carte — le producteur ne mélange pas.
	Provenance string        `json:"provenance"`
	Zones      []CalloutZone `json:"zones"`
}

// CalloutZone est une zone nommée, en mètres monde (le repère des trajectoires).
type CalloutZone struct {
	// VolumeIndex est l'indice du volume dans le tag levl — la clé de traçabilité vers
	// callouts_i18n.csv et les dumps de recherche.
	VolumeIndex int `json:"volume_index"`
	// Name est le nom de CONCEPTION (possiblement tronqué à 32 octets par le format).
	Name string `json:"name"`
	// EN et FR sont les libellés joueur officiels (816/816 résolus par string_id).
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

// Lookup rend l'entrée d'une carte par son module installé.
//
// ErrCalloutsUnknownMap est le cas NOMINAL des cartes Forge : l'appelant dégrade — pas
// de calque zones — et n'échoue jamais.
func (c *MapCalloutsCatalog) Lookup(module string) (MapCalloutsEntry, error) {
	if c == nil {
		return MapCalloutsEntry{}, ErrCalloutsUnknownMap
	}
	e, ok := c.Maps[module]
	if !ok {
		return MapCalloutsEntry{}, fmt.Errorf("%w : %q", ErrCalloutsUnknownMap, module)
	}
	return e, nil
}
