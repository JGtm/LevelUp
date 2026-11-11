//go:build research

package divergence

// fermeture_carte_research_test.go : la carte de fermeture d un film sous les DEUX contextes de
// l instrument (plan marche complete, A1) : sans carte (`grammar.ContexteDeFilm`, celui de A0.1) et sous
// carte (la voie de la cuisson, voir ouvrir.go). Pas de temoin : c est la fermeture de NOTRE marche.
//
//	FERMETURE_RACINE=<film_chunks> FERMETURE_FILMS=<corpus_151.tsv> FERMETURE_CARTE=<map_quant_bounds.json> \
//	  FERMETURE_SORTIE=C:\t\marche\A1\fermeture_carte \
//	  go test -tags=research ./research/divergence/ -run TestFermetureCarte -v -timeout 60m
//
// Sorties : `fermeture_carte_films.tsv` (une ligne par film : paquets, fermes sans carte, fermes sous
// carte) et `fermeture_carte_causes.tsv` (build, contexte, cause d arret, paquets).

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/memguard"
)

// fermetureDuFilm : paquets, fermes et causes d arret de la marche d un film sous un contexte.
func fermetureDuFilm(racine, id, niveau string, cat *profile.MapQuantCatalog) (paquets, fermes int, causes map[string]int, err error) {
	fc, err := ouvrirFilm(racine, id, niveau, cat)
	if err != nil {
		return 0, 0, nil, err
	}
	causes = map[string]int{}
	_, err = grammar.FrameClosureDetaillee(fc, grammar.UsagesProduit{}, func(d grammar.PaquetDeCarte) {
		paquets++
		if d.Fermee {
			fermes++
		} else {
			causes[d.Cause]++
		}
	})
	return paquets, fermes, causes, err
}

func TestFermetureCarte(t *testing.T) {
	racine, liste, chemin, sortie := os.Getenv("FERMETURE_RACINE"), os.Getenv("FERMETURE_FILMS"), os.Getenv("FERMETURE_CARTE"), os.Getenv("FERMETURE_SORTIE")
	if racine == "" || liste == "" || chemin == "" || sortie == "" {
		t.Skip("FERMETURE_RACINE, FERMETURE_FILMS, FERMETURE_CARTE et FERMETURE_SORTIE requis")
	}
	refuserSortieSousData(t, sortie)
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sortie, 0o755); err != nil { //nolint:gosec // dossier de sortie de l operateur
		t.Fatal(err)
	}
	f, err := os.Open(liste) //nolint:gosec // chemin de l operateur
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	films, err := os.Create(filepath.Join(sortie, "fermeture_carte_films.tsv")) //nolint:gosec // idem
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = films.Close() }()
	fmt.Fprintln(films, "film\tbuild\tniveau\tpaquets\tfermes_sans_carte\tfermes_sous_carte")
	type cle struct{ build, contexte, cause string }
	total := map[cle]int{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		if len(c) < 7 || c[0] == "film" {
			continue // ligne d en-tete ; les films que le temoin refuse sont mesures aussi
		}
		id, build, niveau := c[0], c[4], c[6]
		garde := memguard.Arm(context.Background(), "film_re/fermeture_carte", plafondGiB, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets) - arret\n", id, pic)
			os.Exit(3)
		})
		p0, f0, c0, err0 := fermetureDuFilm(racine, id, niveau, nil)
		p1, f1, c1, err1 := fermetureDuFilm(racine, id, niveau, cat)
		garde.Disarm()
		runtime.GC()
		if err0 != nil || err1 != nil {
			t.Errorf("%s : sans carte %v ; sous carte %v", id, err0, err1)
			continue
		}
		if p0 != p1 {
			t.Errorf("%s : %d paquets sans carte, %d sous carte", id, p0, p1)
		}
		fmt.Fprintf(films, "%s\t%s\t%s\t%d\t%d\t%d\n", id, build, niveau, p1, f0, f1)
		for k, n := range c0 {
			total[cle{build, "sans_carte", k}] += n
		}
		for k, n := range c1 {
			total[cle{build, "sous_carte", k}] += n
		}
		total[cle{build, "sans_carte", "(fermes)"}] += f0
		total[cle{build, "sous_carte", "(fermes)"}] += f1
		total[cle{build, "tous", "(paquets)"}] += p1
	}
	causes, err := os.Create(filepath.Join(sortie, "fermeture_carte_causes.tsv")) //nolint:gosec // idem
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = causes.Close() }()
	fmt.Fprintln(causes, "build\tcontexte\tcause\tpaquets")
	cles := make([]cle, 0, len(total))
	for k := range total {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		a, b := cles[i], cles[j]
		if a.build != b.build {
			return a.build < b.build
		}
		if a.contexte != b.contexte {
			return a.contexte < b.contexte
		}
		return total[a] > total[b]
	})
	for _, k := range cles {
		fmt.Fprintf(causes, "%s\t%s\t%s\t%d\n", k.build, k.contexte, k.cause, total[k])
	}
}
