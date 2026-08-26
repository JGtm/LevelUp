package main

// reglages.go — LES RÉGLAGES DE CUISSON PAR CARTE.
//
// LA RÈGLE, ET ELLE EST LA RAISON D'ÊTRE DE CE FICHIER. `internal/himap` ne contient AUCUNE
// branche par carte : c'est ce qui rend la chaîne transférable, et une seule carte possède
// l'oracle fort des positions de joueur, donc régler dans le code serait invérifiable.
// Mais le gate utilisateur du 2026-08-26 a établi, images à l'appui, que le meilleur rendu
// n'est pas le même partout — `encre` sur Cliffhanger, la cible du témoin sur Catalyst.
//
// Le choix vit donc en DONNÉE, ici, avec sa raison écrite et la date de son gate. La chaîne
// reçoit une ENTRÉE ; elle ne sait toujours pas quelle carte elle cuit.
//
// UNE ENTRÉE SANS RAISON NI DATE EST UN RÉGLAGE ORPHELIN : dans six mois personne ne saura
// s'il tient encore. `TestReglagesFondJustifies` les refuse.

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/himap"
)

// reglageCarte : ce qui peut varier d'une carte à l'autre. Un champ vide ou nul vaut « la
// valeur de production » — un réglage ne se déclare que là où il a été jugé.
type reglageCarte struct {
	Carte   string  `json:"carte,omitempty"`
	Style   string  `json:"style,omitempty"`
	Echelle float64 `json:"echelle,omitempty"`
	// EcreteToits : vider les pixels dont aucune surface n est a hauteur de jeu
	// (himap/ecretage_toits.go). Jamais un defaut — il efface les rochers hauts d une carte
	// qui en fait son identite.
	EcreteToits bool `json:"ecreteToits,omitempty"`
	// RogneAuxZones : effacer la matiere hors des zones nommees dilatees
	// (himap/masque_zones.go). A ne poser qu apres avoir regarde le taux mesure.
	RogneAuxZones bool   `json:"rogneAuxZones,omitempty"`
	Raison        string `json:"raison"`
	GateLe        string `json:"gateLe"`
}

type reglagesFond struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Note          string                  `json:"note,omitempty"`
	Cartes        map[string]reglageCarte `json:"cartes"`
}

// chargeReglages lit les réglages par carte. ABSENT n'est pas une erreur : c'est le cas
// nominal d'un titre qui n'a encore rien fait juger.
func chargeReglages(chemin string) (*reglagesFond, error) {
	blob, err := os.ReadFile(chemin)
	if errors.Is(err, os.ErrNotExist) {
		return &reglagesFond{Cartes: map[string]reglageCarte{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var r reglagesFond
	if err := json.Unmarshal(blob, &r); err != nil {
		return nil, fmt.Errorf("réglages de fond illisibles : %w", err)
	}
	if r.Cartes == nil {
		r.Cartes = map[string]reglageCarte{}
	}
	// Un habillage inconnu est REFUSÉ ici et pas silencieusement ignoré : une carte cuite
	// dans l'habillage par défaut alors que la donnée en demandait un autre passerait le
	// gate sous une fausse identité.
	for cle, c := range r.Cartes {
		if c.Style != "" && !himap.StyleFondValide(himap.StyleFond(c.Style)) {
			return nil, fmt.Errorf("réglage %q : habillage inconnu %q", cle, c.Style)
		}
	}
	return &r, nil
}

// styleDe rend l'habillage de cette carte : le sien s'il est déclaré, sinon celui de la
// cuisson. Le choix retenu est JOURNALISÉ — un fond publié dans un habillage qu'on n'a pas vu
// passer est un fond qu'on ne saura pas expliquer.
func (e *environnement) styleDe(cle string) himap.StyleFond {
	if e.reglages == nil {
		return e.style
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.Style == "" {
		return e.style
	}
	s := himap.StyleFond(c.Style)
	if s != e.style {
		slog.Info("mapfond: habillage propre a la carte", "carte", cle, "style", string(s),
			"gateLe", c.GateLe)
	}
	return s
}

// echelleDe rend l'échelle de cette carte : la sienne si elle est déclarée, sinon celle de la
// cuisson (elle-même à la valeur de production si aucune n'est demandée).
func (e *environnement) echelleDe(cle string) float64 {
	if e.reglages == nil {
		return e.echelle
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.Echelle <= 0 {
		return e.echelle
	}
	slog.Info("mapfond: echelle propre a la carte", "carte", cle, "mpp", c.Echelle,
		"gateLe", c.GateLe)
	return c.Echelle
}

// ecreteToitsDe dit si cette carte demande l'écrêtage des toits. Le choix est JOURNALISÉ :
// c'est la seule voie qui SUPPRIME de la matière, elle ne doit jamais passer inaperçue.
func (e *environnement) ecreteToitsDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.EcreteToits {
		return false
	}
	slog.Info("mapfond: ecretage des toits arme pour cette carte", "carte", cle, "gateLe", c.GateLe)
	return true
}

// zonesNommeesDe rend les polygones des callouts d'une carte : le contour principal de chaque
// zone, plus ses PARTIES (provenance « decoupe »).
//
// LES TROUS NE SONT PAS SOUSTRAITS, et c'est délibéré : le masque sert à décider ce qu'on
// GARDE. Un trou rempli garde un peu plus de matière — l'erreur va dans le sens prudent. Les
// soustraire demanderait une règle pair-impair globale et ferait disparaître, au moindre défaut
// de découpe, du terrain que personne n'a jugé.
func (e *environnement) zonesNommeesDe(cle string) [][][2]float64 {
	if e.callouts == nil {
		return nil
	}
	entree, ok := e.callouts.Maps[cle]
	if !ok {
		return nil
	}
	var out [][][2]float64
	for _, z := range entree.Zones {
		if len(z.Polygon) >= 3 {
			out = append(out, z.Polygon)
		}
		for _, p := range z.Parts {
			if len(p) >= 3 {
				out = append(out, p)
			}
		}
	}
	return out
}

// rogneAuxZonesDe dit si cette carte demande le rognage sur ses zones nommées. Journalisé :
// c'est, avec l'écrêtage, l'une des deux voies qui SUPPRIMENT de la matière.
func (e *environnement) rogneAuxZonesDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.RogneAuxZones {
		return false
	}
	slog.Info("mapfond: rognage aux zones nommees arme pour cette carte", "carte", cle, "gateLe", c.GateLe)
	return true
}
