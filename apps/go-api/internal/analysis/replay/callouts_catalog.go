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
// AUCUNE CARTE FORGE N'EST AU CATALOGUE AUJOURD'HUI, ET C'EST UNE LACUNE, PAS UN FAIT DE
// CONSTRUCTION. Cet en-tête a longtemps affirmé le contraire ; la mesure du 2026-08-27 l'a
// renversé. Ce qui est vrai : un CANEVAS ne porte aucune zone nommée (mesuré sur les 8 canevas
// installés). Ce qui est faux : en déduire qu'une CARTE Forge n'en a pas. Ses zones vivent
// dans son map.mvar, chacune un objet de type himap.TypeIDZoneNommee portant un StringId qui
// se résout contre le tag global locs — 18 zones sur Isolation, dont « cave » et « top mid ».
// Les alimenter demande une clé map_id à côté des clés-module, et l'extraction des libellés
// manquants : 274 des 434 StringId employés n'ont pas encore de texte joueur.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// MapCalloutsSchemaVersion est la version de forme attendue du catalogue.
const MapCalloutsSchemaVersion = 1

// ErrCalloutsUnknownMap signale un module absent du catalogue de callouts. Cas COURANT : le
// catalogue couvre les 22 cartes intégrées et aucune carte Forge — non parce qu'elles n'ont
// pas de zones (elles en ont, cf. l'en-tête), mais parce que leur alimentation reste à
// écrire. L'appelant dégrade, il n'échoue pas.
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
}

// CalloutBrutZone est le polygone d'origine d'une zone, conservé après découpage.
type CalloutBrutZone struct {
	VolumeIndex int          `json:"volume_index"`
	Polygon     [][2]float64 `json:"polygon"`
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
