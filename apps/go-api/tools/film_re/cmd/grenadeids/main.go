//go:build research

// Command grenadeids est le BANC DU LOT 3.3 (volet recherche) : il mesure, film par film, ou
// les quatre identifiants de grenade ACTUELS apparaissent dans les films anciens, et — sur
// demande — apparie ce qui suit le marqueur aux decrements unitaires du compteur i22.
//
// Il ne compile QUE sous le tag `research` et n ecrit aucun fichier. Il lit les films EN PLACE,
// UN A LA FOIS, dans l ordre donne.
//
//	cd apps/go-api
//	go run -tags=research ./tools/film_re/cmd/grenadeids \
//	  -racine <parc>/data/cache/film_chunks \
//	  -films 111fa685,e5adf7b2,60ae07c4,a349fea8,fb1a1a72
//
// Le verrou de decodage de `internal/filmproc` n est PAS pris : il ecrirait un fichier sous la
// racine du cache, et le cadre de ce lot interdit toute ecriture sous `data/`. La sentinelle
// memoire, elle, est armee — elle ne fait qu abaisser un plafond dans ce processus.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/tools/film_re/grenadeids"
)

// plafondParDefautGiB : le plafond memoire du banc. Un balayage de bits ne construit aucun
// artefact ; ce plafond est une ceinture, pas un dimensionnement.
const plafondParDefautGiB = 4

func main() {
	racine := flag.String("racine", "", "racine portant les repertoires de films (lecture seule)")
	films := flag.String("films", "", "identifiants de films, separes par des virgules")
	fenetre := flag.Int("fenetre", 64, "demi-largeur, en bits, du balayage de decalage (passe A)")
	voisinage := flag.Int("voisinage", 256, "ecart maximal, en bits, pour rattacher une occurrence absolue a un marqueur")
	top := flag.Int("top", 12, "nombre d entrees detaillees par histogramme")
	apparier := flag.Bool("apparier", false, "jouer la passe D : appariement aux decrements i22 (question 2)")
	toleranceMS := flag.Int("tolerance-ms", 0, "tolerance temporelle, en millisecondes, autour de chaque intervalle de decrement")
	temoinMS := flag.Int("temoin-ms", 0, "temoin de hasard : rejoue la passe D avec les instants decales de N millisecondes ; 0 = pas de temoin")
	dump := flag.String("dump", "", "identifiant (hexadecimal, ex. 0x764ACFA8) dont on releve les bits autour du marqueur")
	fenetreIndex := flag.Int("fenetre-index", 0, "demi-largeur, en bits, du balayage de la position du champ d index auteur ; 0 = pas de balayage")
	dumpMax := flag.Int("dump-max", 4, "nombre maximal de tranches relevees par film")
	ti := flag.Int("ti", -1, "restreint la passe D aux naissances de cet archetype (41 = projectile, 9 = managed-player) ; -1 = toutes")
	plafond := flag.Int("plafond-gib", plafondParDefautGiB, "plafond memoire du banc ; 0 desarme")
	flag.Parse()

	ids := decouper(*films)
	if *racine == "" || len(ids) == 0 {
		fmt.Fprintln(os.Stderr, "usage : -racine <dir> -films <id,id,...> [-fenetre N] [-apparier]")
		os.Exit(2)
	}
	if err := grenadeids.VerifierMarqueurDeProduction(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	garde := filmproc.Arm("film_re/grenadeids", *plafond, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "plafond memoire franchi (%d octets) — arret\n", pic)
		os.Exit(3)
	})
	defer garde.Disarm()

	cible, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimPrefix(*dump, "0x"), "0X"), 16, 32)
	if *dump != "" && err != nil {
		fmt.Fprintf(os.Stderr, "-dump %q : %v\n", *dump, err)
		os.Exit(2)
	}
	opt := grenadeids.Options{
		Fenetre:      *fenetre,
		Voisinage:    *voisinage,
		Dump:         uint32(cible),
		DumpMax:      *dumpMax,
		FenetreIndex: *fenetreIndex,
	}
	echecs := 0
	for _, id := range ids {
		if err := traiter(*racine, id, opt, reglages{
			top:         *top,
			apparier:    *apparier,
			toleranceUS: uint64(*toleranceMS) * 1000,
			temoinUS:    uint64(*temoinMS) * 1000,
			ti:          *ti,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "%s : %v\n", id, err)
			echecs++
		}
		runtime.GC()
	}
	if echecs > 0 {
		os.Exit(1)
	}
}

// reglages porte ce qui ne tient pas dans [grenadeids.Options], pour respecter la regle des
// cinq parametres.
type reglages struct {
	top         int
	apparier    bool
	toleranceUS uint64
	temoinUS    uint64
	ti          int
}

// traiter ouvre UN film, le mesure, publie, et le laisse partir avant le suivant.
func traiter(racine, id string, opt grenadeids.Options, rg reglages) error {
	b, err := grenadeids.Ouvrir(racine, id)
	if err != nil {
		return err
	}
	grenadeids.EcrireEntete(os.Stdout, b)
	rel := grenadeids.Balayer(b, opt)
	grenadeids.EcrireReleve(os.Stdout, rel, rg.top)
	grenadeids.EcrireTranches(os.Stdout, rel)
	grenadeids.EcrireIndexAuteur(os.Stdout, rel, rg.top)
	if !rg.apparier {
		return nil
	}
	dec, st, err := grenadeids.LireDecrements(racine, id)
	if err != nil {
		return fmt.Errorf("inventaire i22 : %w", err)
	}
	occ := grenadeids.FiltrerParTi(rel.Occurrences, rg.ti)
	app := grenadeids.Apparier(occ, dec, rg.toleranceUS)
	grenadeids.EcrireAppariement(os.Stdout, st, app, rg.top)
	if rg.temoinUS == 0 {
		return nil
	}
	decales := grenadeids.Decaler(occ, rg.temoinUS)
	fmt.Printf("   passe D  TEMOIN DE HASARD (instants decales de %d us)\n", rg.temoinUS)
	grenadeids.EcrireAppariement(os.Stdout, st,
		grenadeids.Apparier(decales, dec, rg.toleranceUS), rg.top)
	return nil
}

// decouper rend les identifiants non vides de la liste.
func decouper(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
