// Package analysis — highlight_medal_rawjson.go : LE DOCUMENT QUI PORTE L IDENTITE
// D UNE MEDAILLE.
//
// `highlight_events.raw_json` est le seul endroit ou vit le nom d une medaille du
// film. Le lecteur (platform/duckdb.medalNameFromRawJSON) n en tire qu un champ,
// `medal_name`, et tolere les champs surnumeraires ; les documents de l ere Python
// en portaient trois (type_hint, medal_value, medal_name). Cote Go on n ecrit que
// le champ lu — le couple d octets, lui, a ses propres colonnes.
//
// Ce fichier existe pour que les DEUX ecrivains (le collecteur du live sync et la
// passe de rattrapage hors ligne) composent le meme document a partir du meme code.
// Un contrat de serialisation recopie diverge ; celui-ci ne peut pas.
package analysis

import (
	"encoding/json"
	"fmt"
	"strings"
)

// documentMedaille est la forme ecrite dans highlight_events.raw_json.
type documentMedaille struct {
	MedalName string `json:"medal_name"`
}

// MedalRawJSON compose le document `raw_json` d un event medal a partir du nom
// anglais de la medaille (clef de referentiel, pas un libelle localise).
//
// Un nom vide est une ERREUR et non un document vide : ecrire `{"medal_name":""}`
// produirait une ligne qui a l air renseignee et que le lecteur rejette quand meme.
// L appelant qui n a pas de nom doit laisser raw_json a NULL.
func MedalRawJSON(medalName string) (string, error) {
	nom := strings.TrimSpace(medalName)
	if nom == "" {
		return "", fmt.Errorf("MedalRawJSON: nom de medaille vide")
	}
	b, err := json.Marshal(documentMedaille{MedalName: nom})
	if err != nil {
		return "", fmt.Errorf("MedalRawJSON %q: %w", nom, err)
	}
	return string(b), nil
}
