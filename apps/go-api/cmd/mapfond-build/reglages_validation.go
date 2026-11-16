package main

// reglages_validation.go — LE CHARGEMENT DES RÉGLAGES ET CE QU'IL REFUSE.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

// sansLevier dit si l'entrée ne déclare AUCUN réglage : elle ne porte alors que son identité
// (carte, raison, date de gate) et n'a strictement aucun effet sur la cuisson.
//
// LE CONTRÔLE EST GÉNÉRIQUE, ET C'EST LE POINT. `TestReglagesFondJustifies` énumérait trois
// champs à la main — habillage, échelle, écrêtage — et cette liste avait cessé de suivre la
// structure : une entrée qui ne déclare que `moduleGeometrie` était refusée comme « sans
// effet » alors qu'elle change le module d'où la géométrie est lue. C'est exactement le cas de
// Live Fire. Une liste écrite à la main re-divergera ; la comparaison à la valeur nulle, non.
func (c reglageCarte) sansLevier() bool {
	c.Carte, c.Raison, c.GateLe = "", "", ""
	return reflect.DeepEqual(c, reglageCarte{})
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
	// Un champ inconnu est REFUSÉ : un levier retiré qu'une entrée déclarerait encore serait
	// sinon ignoré en silence, et la carte cuite sans lui.
	dec := json.NewDecoder(bytes.NewReader(blob))
	dec.DisallowUnknownFields()
	var r reglagesFond
	if err := dec.Decode(&r); err != nil {
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
		if !modeNavmeshValide(c.NavmeshMode) {
			return nil, fmt.Errorf("réglage %q : navmeshMode inconnu %q (faces ou silhouette)", cle, c.NavmeshMode)
		}
	}
	return &r, nil
}
