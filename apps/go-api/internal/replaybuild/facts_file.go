package replaybuild

// facts_file.go — LE FICHIER DE FAITS D'UN MATCH : UNE SEULE FORME, ECRITE ET LUE ICI.
//
// # CE QUE C'EST
//
// Les faits que la base sait et que le film ne dit pas (lignes de match, scores des deux camps,
// nom de variante, map_id), A PLAT dans un JSON, plus deux champs que le film ne porte pas
// davantage : l'identite COMPLETE du match et ses identites de carte candidates.
//
// # POURQUOI CE TYPE VIT ICI, ET PAS DANS CHAQUE OUTIL QUI LE LIT
//
// Trois programmes le manipulent : `levelup replay-facts-export` l'ECRIT, `cmd/replay-equiv`
// et l'operateur de `cmd/replay-build --facts` le LISENT. Une copie du type par programme
// derive au premier champ ajoute — et la derive serait SILENCIEUSE, puisque `encoding/json`
// ignore les champs qu'il ne connait pas : le harnais d'equivalence cuirait alors sans cartes
// candidates, rendrait un artefact appauvri, et son digest passerait pour une regression du
// decodeur. Le type est donc UNIQUE, dans le paquet qui consomme les faits (PLAN_CUISSON_PERF
// §4, items 0.2 et 0.4).
//
// # POURQUOI L'EMBARQUEMENT
//
// `port.MatchFacts` est EMBARQUE, pas imbrique : le JSON reste PLAT, exactement la forme que
// `cmd/replay-build --facts` desserialise deja en `port.MatchFacts` seul (il ignore `matchId`
// et `mapNames`). Un champ imbrique casserait ce lecteur sans que rien ne le dise.

import (
	"encoding/json"
	"fmt"
	"os"

	"levelup/go-api/internal/port"
)

// FactsFile est la forme d'un `<short8>.facts.json`.
type FactsFile struct {
	// MatchFacts est EMBARQUE : ses champs sont au premier niveau du JSON.
	port.MatchFacts
	// MatchID est l'identite complete du match (le nom du fichier n'en porte que les huit
	// premiers caracteres, et la construction a besoin de l'identite entiere).
	MatchID string `json:"matchId"`
	// MapNames sont les identites de carte candidates, DANS L'ORDRE que ResolveMapEntry
	// essaie. Une carte devinee donnerait des coordonnees fausses d'un facteur d'echelle
	// arbitraire, sans que rien ne le signale (cf. ErrMapNotInCatalog).
	MapNames []string `json:"mapNames"`
}

// ReadFactsFile lit un fichier de faits.
//
// L'ECHEC EST FRANC, ET C'EST VOULU : un lecteur qui degraderait en faits vides rendrait un
// artefact sans compteurs de joueur ni actions d'objectif — un artefact qui se compare
// tranquillement a un autre artefact appauvri, et une equivalence VACUANTE. Ici, l'appelant
// demande explicitement des faits : ne pas les avoir est une erreur.
func ReadFactsFile(path string) (FactsFile, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'appelant (CLI, harnais)
	if err != nil {
		return FactsFile{}, fmt.Errorf("faits du match illisibles (%s) : %w", path, err)
	}
	var f FactsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return FactsFile{}, fmt.Errorf("faits du match invalides (%s) : %w", path, err)
	}
	return f, nil
}
