// cmd/statnames-sweep — INVENTAIRE DES EMPLACEMENTS DU STATBORG d'un film, en CLI durable.
//
// Il remplace `cmd/tmp_statnames` (l'outil jetable du balayage de 2026-08-05, disparu du
// depot — c'est lui que cite l'en-tete d'`objectiveevents/named.go`). La methode est la
// meme : la valeur FINALE de chaque emplacement (composant 0..27, cotes A et B), par slot
// de joueur, l'identite des slots par le pont des INSTANTS DE MORT — puis la confrontation
// a un oracle par joueur sur MOITIES DISJOINTES de films (une moitie pour chercher, une
// moitie pour verifier).
//
// # Trois modes, un seul decodage borne
//
//	-films a,b,c            PARENT : un processus enfant PAR FILM via internal/filmproc
//	                        (plafond de mesure 2 Gio, priorite basse, codes de protocole).
//	-child -match a         ENFANT : balaye UN film sous la sentinelle et imprime le TSV.
//	-confront ...           CONFRONTATION : aucune lecture de film — parse un TSV de
//	                        balayage, un oracle JSON, et rend candidats puis verdicts.
//
// AUCUNE base n'est ouverte, dans aucun mode : le film vient du cache disque
// (`filmcache`), l'oracle vient d'un JSON fige par l'operateur (releve de
// `match_objective_stats_latest`, la vue — jamais la table brute).
//
// # La garde de mode du drapeau ne s'applique pas ici, et c'est mesure
//
// `SlotIdentityByDeaths` est le pont que la garde de mode du calque CTF refuse de payer
// hors CTF (19-22 Go avant le correctif). Le correctif reel est le plafond
// `objectiveevents.maxDeathsPerSlot`, qui borne le deroulage SUR TOUT FILM ; les
// instruments D4-D9 l'appellent deja sur Oddball sous filmproc. Ce CLI fait de meme :
// enfant borne + plafond de deroulage — les deux moities de la correction.
//
// Usage (depuis apps/go-api, binaire compile — jamais `go run` en boucle) :
//
//	go build -o /tmp/statnames-sweep.exe ./cmd/statnames-sweep
//	/tmp/statnames-sweep.exe -cache <repo>/data/cache -films 24dbb67d,92f18088
//	/tmp/statnames-sweep.exe -confront -sweep sweep.log -oracle oracle.json \
//	    -search 24dbb67d,92f18088 -verify 43716616,d9781168
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/domain/title"
)

func main() {
	cache := flag.String("cache", "", "racine du cache film (defaut : <repo>/data/cache)")
	films := flag.String("films", "", "PARENT : ids courts des films a balayer, separes par des virgules")
	match := flag.String("match", "", "ENFANT : l'unique film balaye par ce processus")
	child := flag.Bool("child", false, "ENFANT d'une passe (pose par le parent, pas par l'operateur)")
	confront := flag.Bool("confront", false, "confrontation TSV x oracle (aucun film decode)")
	vip := flag.Bool("vip", false, "confrontation VIP (avec -confront) : gate 2/3 + stabilite + somme-film")
	sweepPath := flag.String("sweep", "", "confrontation : fichier TSV produit par le balayage")
	oraclePath := flag.String("oracle", "", "confrontation : oracle JSON film -> xuid -> colonne -> valeur")
	search := flag.String("search", "", "confrontation : films de la moitie de RECHERCHE")
	verify := flag.String("verify", "", "confrontation : films de la moitie de VERIFICATION")
	flag.Parse()

	code, err := run(runArgs{
		cache: *cache, films: splitIDs(*films), match: *match, child: *child,
		confront: *confront, vip: *vip, sweepPath: *sweepPath, oraclePath: *oraclePath,
		search: splitIDs(*search), verify: splitIDs(*verify),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "statnames-sweep: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

// runArgs regroupe la ligne de commande (regle des 5 parametres).
type runArgs struct {
	cache, match          string
	films, search, verify []string
	child, confront, vip  bool
	sweepPath, oraclePath string
}

func run(a runArgs) (int, error) {
	if a.confront {
		if a.vip {
			return 0, runConfrontVIP(a)
		}
		return 0, runConfront(a)
	}
	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		return 0, err
	}
	if a.cache == "" {
		a.cache = repoRoot + "/data/cache"
	}
	if a.child {
		return runChild(a.cache, a.match), nil
	}
	if len(a.films) == 0 {
		return 0, fmt.Errorf("aucun film : -films (parent), -child -match (enfant) ou -confront")
	}
	return 0, runParent(context.Background(), repoRoot, a)
}

// splitIDs decoupe une liste d'ids courts, sans entree vide.
func splitIDs(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
