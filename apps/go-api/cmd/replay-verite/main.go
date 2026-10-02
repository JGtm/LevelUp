// Command replay-verite juge un artefact de rejeu APRES contre un artefact AVANT du meme temoin,
// avec le banc de verite (`internal/replayverite`).
//
// C'est l'appel AUTONOME du banc. Le gate de corpus (`cmd/replay-corpus-gate`) l'a integre (D-5) : il
// juge avec le meme paquet `internal/replayverite`, et compile CET outil a la base pour en lire le
// registre des replis (`-registre`). Il ne cuit rien, ne decode aucun film et n'ouvre aucune base. Il lit
// deux artefacts deja cuits, les faits du match qui les ont cuits, et — en option — le registre des
// replis de la revision d'AVANT (produit par ce meme outil, compile a cette revision).
//
// Usage (depuis apps/go-api) :
//
//	go run ./cmd/replay-verite -avant <a.json> -apres <b.json> -faits <id>.facts.json [-temoin id] [-registre-avant r.json]
//	go run ./cmd/replay-verite -registre > registre.json
//
// Code de sortie : 0 ok, 1 FAUX ou MANQUE, 2 usage ou lecture impossible.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/replayverite"
)

const (
	codeOK      = 0
	codeVerdict = 1
	codeUsage   = 2
)

func main() {
	os.Exit(lancer(os.Args[1:], os.Stdout, os.Stderr))
}

// options regroupe les drapeaux de l'outil.
type options struct {
	avant, apres, faits, oracle, temoin, registreAvant string
	registre                                           bool
}

func lancer(args []string, out, errw io.Writer) int {
	var o options
	fs := flag.NewFlagSet("replay-verite", flag.ContinueOnError)
	fs.SetOutput(errw)
	fs.StringVar(&o.avant, "avant", "", "artefact de reference (revision d'avant)")
	fs.StringVar(&o.apres, "apres", "", "artefact juge (revision d'apres)")
	fs.StringVar(&o.faits, "faits", "", "faits du match (<short8>.facts.json, ceux de la cuisson)")
	fs.StringVar(&o.oracle, "oracle", "", "oracle officiel (<short8>.oracle.json, replay-facts-export --oracle) ; vide = scores O-S3 absents")
	fs.StringVar(&o.temoin, "temoin", "", "nom du temoin dans le rapport (defaut : matchId)")
	fs.StringVar(&o.registreAvant, "registre-avant", "", "registre des replis de la revision d'avant (-registre)")
	fs.BoolVar(&o.registre, "registre", false, "ecrire le registre des replis de CETTE revision (JSON) et sortir")
	if err := fs.Parse(args); err != nil {
		return codeUsage
	}
	if o.registre {
		return ecrireRegistre(out, errw)
	}
	c, temoin, err := juger(o)
	if err != nil {
		_, _ = fmt.Fprintf(errw, "replay-verite : %v\n", err)
		return codeUsage
	}
	if err := replayverite.Rendre(out, temoin, c); err != nil {
		_, _ = fmt.Fprintf(errw, "replay-verite : rendu : %v\n", err)
		return codeUsage
	}
	if c.Statut != replayverite.StatutOK {
		return codeVerdict
	}
	return codeOK
}

// juger lit les entrees et rend la comparaison.
func juger(o options) (replayverite.Comparaison, string, error) {
	if o.avant == "" || o.apres == "" || o.faits == "" {
		return replayverite.Comparaison{}, "", errors.New("-avant, -apres et -faits sont obligatoires")
	}
	faits, err := replaybuild.ReadFactsFile(o.faits)
	if err != nil {
		return replayverite.Comparaison{}, "", err
	}
	avant, err := lireAvec(o.avant, replayverite.LireDocument)
	if err != nil {
		return replayverite.Comparaison{}, "", err
	}
	apres, err := lireAvec(o.apres, replayverite.LireDocument)
	if err != nil {
		return replayverite.Comparaison{}, "", err
	}
	var reg replayverite.RegistreReplis
	if o.registreAvant != "" {
		if reg, err = lireAvec(o.registreAvant, replayverite.LireRegistre); err != nil {
			return replayverite.Comparaison{}, "", err
		}
	}
	var oracle *domain.MatchOracle
	if o.oracle != "" {
		if oracle, err = lireAvec(o.oracle, replayverite.LireOracle); err != nil {
			return replayverite.Comparaison{}, "", err
		}
	}
	temoin := o.temoin
	if temoin == "" {
		temoin = apres.MatchID
	}
	c := replayverite.Comparer(replayverite.Noter(avant, faits.MatchFacts, oracle),
		replayverite.Noter(apres, faits.MatchFacts, oracle), reg)
	return c, temoin, nil
}

// lireAvec lit un fichier et le decode par le lecteur du banc.
func lireAvec[T any](path string, lire func([]byte) (T, error)) (T, error) {
	var zero T
	blob, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'operateur
	if err != nil {
		return zero, fmt.Errorf("%s : %w", path, err)
	}
	v, err := lire(blob)
	if err != nil {
		return zero, fmt.Errorf("%s : %w", path, err)
	}
	return v, nil
}

// ecrireRegistre ecrit, pour CETTE revision, si le compteur de chaque repli est branche : l'entree
// `-registre-avant` d'une comparaison ulterieure (decision D-6).
func ecrireRegistre(out, errw io.Writer) int {
	reg := replayverite.RegistreReplis{}
	for _, r := range decfilm.Table() {
		reg[string(r.Nom)] = r.CompteurBranche
	}
	blob, err := json.MarshalIndent(reg, "", "  ")
	if err == nil {
		_, err = fmt.Fprintf(out, "%s\n", blob)
	}
	if err != nil {
		_, _ = fmt.Fprintf(errw, "replay-verite : registre : %v\n", err)
		return codeUsage
	}
	return codeOK
}
