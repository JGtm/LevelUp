package main

// walkers_parent.go — LE PARENT DU MODE `-walkers` : 1 380 films, 1 380 processus.
//
// La mesure de divergence (PLAN_CUISSON_PERF §4, item 0.7) porte sur TOUT le cache, pas sur le
// corpus : c'est elle qui dit si le lot 1 est un refacto pur, et sur quels films il ne l'est
// pas. Meme regle que l'equivalence — un film par processus borne, jamais deux dans le meme.
// `-films` limite la mesure a quelques films, pour la mettre au point sans payer le cache.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// parentWalkers lance un enfant par film du cache et concatene leurs mesures.
func parentWalkers(o options) int {
	films, err := filmsDuCache(o)
	if err != nil {
		fmt.Println("enumeration du cache :", err)
		return 1
	}
	runner, err := filmproc.NewRunner(o.repoRoot, os.Stdout)
	if err != nil {
		fmt.Println("lanceur :", err)
		return 1
	}
	tmp, err := os.MkdirTemp("", "replay-walkers")
	if err != nil {
		fmt.Println("dossier temporaire :", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	fmt.Printf("%d film(s) a mesurer -> %s\n", len(films), o.walkersOut)
	lignes, c := mesurerFilms(o, runner, tmp, films)
	if err := ecrireMesure(o.walkersOut, lignes); err != nil {
		fmt.Println("ecriture de la mesure :", err)
		return 1
	}
	resumerMesure(lignes, c, len(films))
	if c.echecs > 0 {
		return 1
	}
	return 0
}

// comptesWalkers : ce qui n'a pas donne de ligne de mesure.
//
// LES ECARTES NE SONT PAS DES ECHECS : un repertoire de film sans `chunk_NN.bin` (telechargement
// interrompu, purge) n'a rien a mesurer — l'enfant rend CodeSkipped et la passe reste verte.
type comptesWalkers struct{ echecs, ecartes int }

// mesurerFilms lance les enfants et rend leurs lignes, plus ce qui n'a rien donne.
func mesurerFilms(o options, runner *filmproc.Runner, tmp string, films []string) ([]string, comptesWalkers) {
	var lignes []string
	var c comptesWalkers
	debut := time.Now()
	for i, film := range films {
		sortie := filepath.Join(tmp, film+".tsv")
		res := runner.Run(context.Background(), argsEnfant(o, film, sortie, []string{"-walkers"}))
		switch res.Issue {
		case filmproc.IssueOK:
		case filmproc.IssueSkipped:
			c.ecartes++
			fmt.Printf("%5d/%d %-9s ECARTE (aucun chunk au cache)\n", i+1, len(films), film)
			continue
		default:
			c.echecs++
			fmt.Printf("%5d/%d %-9s %s (code %d)\n", i+1, len(films), film, res.Issue, res.Code)
			continue
		}
		lues, err := lireLignes(sortie)
		if err != nil || len(lues) == 0 {
			c.echecs++
			fmt.Printf("%5d/%d %-9s mesure illisible : %v\n", i+1, len(films), film, err)
			continue
		}
		lignes = append(lignes, lues[0])
		fmt.Printf("%5d/%d %s\n", i+1, len(films), lues[0])
	}
	fmt.Printf("mesure terminee en %s\n", time.Since(debut).Round(time.Second))
	return lignes, c
}

// ecrireMesure ecrit le fichier de mesure, en-tete comprise.
func ecrireMesure(path string, lignes []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	contenu := enteteWalkers + "\n" + strings.Join(lignes, "\n") + "\n"
	return os.WriteFile(path, []byte(contenu), 0o600)
}

// resumerMesure imprime le verdict : combien de films sans divergence, lesquels divergent et
// par quel axe, combien portent un flux tronque.
func resumerMesure(lignes []string, c comptesWalkers, total int) {
	sains, tronques := 0, 0
	parAxe := map[string]int{}
	parConsommateur := map[string]int{}
	for _, l := range lignes {
		divFd, divKs, divObj := entier(champ(l, 3)), entier(champ(l, 4)), entier(champ(l, 5))
		if entier(champ(l, 2)) > 0 {
			tronques++
		}
		if divFd+divKs+divObj == 0 {
			sains++
			continue
		}
		parAxe[champ(l, 6)]++
		compterConsommateur(parConsommateur, divFd, divKs, divObj)
	}
	fmt.Printf("\nRESUME (%d film(s) demandes, %d mesures, %d ecarte(s), %d echec(s))\n",
		total, len(lignes), c.ecartes, c.echecs)
	fmt.Printf("  sans divergence      : %d\n", sains)
	fmt.Printf("  avec divergence      : %d\n", len(lignes)-sains)
	fmt.Printf("  a flux tronque       : %d\n", tronques)
	imprimerTable("  divergents par axe", parAxe)
	imprimerTable("  divergents par consommateur", parConsommateur)
}

// compterConsommateur cumule, par marcheur du depot, les films ou il ne voit pas ce que voit la
// grammaire unifiee — c'est ce que le verdict D3 doit nommer.
func compterConsommateur(m map[string]int, divFd, divKs, divObj int) {
	if divFd > 0 {
		m["filmdec"]++
	}
	if divKs > 0 {
		m["killsource"]++
	}
	if divObj > 0 {
		m["objectiveevents"]++
	}
}

// imprimerTable imprime un compte par cle, cles triees.
func imprimerTable(titre string, m map[string]int) {
	if len(m) == 0 {
		fmt.Printf("%s : aucun\n", titre)
		return
	}
	cles := make([]string, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	fmt.Printf("%s :\n", titre)
	for _, k := range cles {
		fmt.Printf("      %-16s %d\n", k, m[k])
	}
}

// entier lit un champ numerique, 0 si illisible (la ligne vient d'un enfant du meme binaire).
func entier(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// filmsDuCache enumere les films du cache, ou la liste de `-films` si elle est posee.
func filmsDuCache(o options) ([]string, error) {
	if films := filmsDemandes(o.films); len(films) > 0 {
		return films, nil
	}
	racine := filmcache.ChunksRoot(title.NewPathResolver(o.repoRoot).CacheRootDir())
	entrees, err := os.ReadDir(racine)
	if err != nil {
		return nil, fmt.Errorf("%s : %w", racine, err)
	}
	var out []string
	for _, e := range entrees {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucun film sous %s", racine)
	}
	sort.Strings(out)
	return out, nil
}
