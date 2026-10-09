package mapcatalog

// callouts_resolution.go — LA CASCADE QUI DONNE SES ZONES NOMMÉES À UNE CARTE JOUÉE, écrite
// une fois.
//
// Trois lecteurs la partagent : le rejeu 2D (par match) et l'onglet Tactique (par carte), au
// service, et le rattrapage des zones au fetch de film, qui l'interroge pour savoir si une
// carte a DÉJÀ ses zones avant d'aller chercher sa variante. En deux exemplaires, une carte
// pourrait être servie sans zones et jugée couverte par le rattrapage — elle ne serait alors
// jamais rattrapée.
//
// LES ESSAIS, DANS CET ORDRE :
//
//	1. module (carte INTÉGRÉE)      map_id, puis noms candidats -> catalogue des bornes
//	                                -> module installé -> `maps` du catalogue versionné.
//	                                Le map_id passe d'abord : le catalogue des bornes en porte
//	                                quelques-uns (cartes dont le nom ne suffit pas).
//	2. map_id, catalogue VERSIONNÉ  `maps_by_id` (cartes Forge relues en revue).
//	3. map_id, catalogue GÉNÉRÉ     cartes Forge rattrapées par le runtime.
//	4. identité DÉCLARÉE            nom -> index des identités des fonds publiés (les noms de
//	                                variante qu'une cuisson a déclarés pour sa carte : « Highpower
//	                                Sentry Defense » pour `btb_highpower`) -> `maps` versionné.
//
// L'essai 1 passe d'abord parce qu'une carte intégrée a une entrée de MEILLEURE qualité
// (polygones du designer découpés sur le décor praticable). Le versionné prime sur le généré :
// une carte relue en revue n'est jamais remplacée par une carte rattrapée.
//
// AUCUN RAPPROCHEMENT PAR NOM ENTRE DEUX map_id. Les essais 2 et 3 ne connaissent que le map_id
// EXACT : deux assets qui portent le même nom sont souvent deux versions différentes d'une carte
// Forge, et leurs zones ne se superposent pas. L'essai 4 ne rend que des MODULES (cartes
// intégrées) : la clé qu'il trouve pour une carte Forge est un map_id, absent par construction
// de la section `maps`.

import (
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// IdentitesDeCarte est ce que le registre sait d'une carte jouée : son asset UGC et ses noms
// candidats, du plus fiable au moins fiable.
type IdentitesDeCarte struct {
	MapID string
	Noms  []string
}

// IndexIdentites résout un nom de carte vers la clé de son fond publié (module installé pour
// une carte intégrée, map_id pour une carte Forge). L'index des fonds en est l'implémentation.
type IndexIdentites interface {
	Lookup(nom string) (string, bool)
}

// SourcesDeZones porte les catalogues que la cascade lit. Chaque champ peut être nil (catalogue
// absent ou illisible — c'est à l'appelant de l'avoir journalisé) : l'essai correspondant rend
// alors une absence.
type SourcesDeZones struct {
	Versionne *replay.MapCalloutsCatalog
	Genere    *replay.MapCalloutsCatalog
	Bornes    *decfilm.MapQuantCatalog
	Identites IndexIdentites
}

// OrigineZones dit quel essai de la cascade a donné ses zones à une carte.
type OrigineZones string

const (
	// OrigineModule : carte intégrée, module trouvé au catalogue des bornes.
	OrigineModule OrigineZones = "module"
	// OrigineVersionne : carte Forge du catalogue versionné.
	OrigineVersionne OrigineZones = "versionne"
	// OrigineGenere : carte Forge rattrapée par le runtime.
	OrigineGenere OrigineZones = "genere"
	// OrigineIdentiteDeclaree : carte intégrée, nom de variante déclaré par son fond publié.
	OrigineIdentiteDeclaree OrigineZones = "identite_declaree"
)

// Resoudre rend les zones nommées d'une carte, et l'essai qui les a trouvées. `false` quand
// aucun essai n'aboutit — l'absence est un cas NORMAL (carte pas encore rattrapée, carte sans
// zone), jamais une erreur.
func (s SourcesDeZones) Resoudre(id IdentitesDeCarte) (replay.MapCalloutsEntry, OrigineZones, bool) {
	if e, ok := s.parModule(id); ok {
		return e, OrigineModule, true
	}
	if e, err := s.Versionne.LookupByID(id.MapID); err == nil {
		return e, OrigineVersionne, true
	}
	if e, err := s.Genere.LookupByID(id.MapID); err == nil {
		return e, OrigineGenere, true
	}
	if e, ok := s.parIdentiteDeclaree(id); ok {
		return e, OrigineIdentiteDeclaree, true
	}
	return replay.MapCalloutsEntry{}, "", false
}

// parModule tente l'essai 1. Une carte Forge y passe sans rien trouver : le catalogue des
// bornes l'envoie vers son CANEVAS, qui ne porte aucune zone et n'a pas d'entrée `maps`.
func (s SourcesDeZones) parModule(id IdentitesDeCarte) (replay.MapCalloutsEntry, bool) {
	if s.Bornes == nil {
		return replay.MapCalloutsEntry{}, false
	}
	cles := make([]string, 0, len(id.Noms)+1)
	if id.MapID != "" {
		cles = append(cles, id.MapID)
	}
	cles = append(cles, id.Noms...)
	for _, cle := range cles {
		bornes, err := s.Bornes.Lookup(cle)
		if err != nil || bornes.Module == "" {
			continue
		}
		if e, err := s.Versionne.Lookup(bornes.Module); err == nil {
			return e, true
		}
	}
	return replay.MapCalloutsEntry{}, false
}

// parIdentiteDeclaree tente l'essai 4 : seuls les MODULES qu'il trouve comptent (cf. l'en-tête).
func (s SourcesDeZones) parIdentiteDeclaree(id IdentitesDeCarte) (replay.MapCalloutsEntry, bool) {
	if s.Identites == nil {
		return replay.MapCalloutsEntry{}, false
	}
	for _, nom := range id.Noms {
		cle, ok := s.Identites.Lookup(nom)
		if !ok {
			continue
		}
		if e, err := s.Versionne.Lookup(cle); err == nil {
			return e, true
		}
	}
	return replay.MapCalloutsEntry{}, false
}
