//go:build research

// Package divergence : instrument jetable (plan marche complete, A0.7) qui confronte NOTRE lecture des
// paquets de tick a celle du temoin libfilmshell, paquet par paquet, et dit OU les deux lectures
// cessent de coincider.
//
// Entrees (le test est SAUTE sans elles) :
//
//	DIVERGENCE_RACINE  racine des films (`<parc>/data/cache/film_chunks`), lecture seule
//	DIVERGENCE_FILMS   fichier des films, une ligne `id<TAB>build<TAB>...` (format de temoins_duree.txt)
//	DIVERGENCE_TEMOIN  dossier des TSV du temoin (`<id8>.tsv`, sonde oracle-filmshell)
//	DIVERGENCE_SORTIE  dossier de sortie, HORS de data/ (refuse sinon)
//	DIVERGENCE_ID      optionnel : un seul film
//	DIVERGENCE_CARTE   optionnel : catalogue map_quant_bounds.json, la liste des films porte le niveau en colonne 4 ; voir ouvrir.go
//
// Sorties : `<sortie>/<id>/divergence_paquets.tsv` et `divergence_resume.md` par film,
// `<sortie>/resume_global.md` pour l ensemble.
//
//	cd libs/film && DIVERGENCE_RACINE=... DIVERGENCE_FILMS=... DIVERGENCE_TEMOIN=C:\t\marche\A0\oracle \
//	  DIVERGENCE_SORTIE=C:\t\marche\A0\divergence \
//	  go test -tags=research ./research/divergence/ -run TestDivergence -v -timeout 60m
//
// LE CONTROLE DE L INSTRUMENT est la porte du test : sur 100 paquets tires au hasard parmi ceux que
// les deux lectures ferment, zero desaccord de position. Un desaccord y serait d abord une erreur de
// convention de bit de l instrument, pas du decodeur.
package divergence

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/memguard"
)

const plafondGiB = 6

type filmDeTest struct{ id, build, niveau string }

func lireLesFilms(chemin string) ([]filmDeTest, error) {
	f, err := os.Open(chemin) //nolint:gosec // chemin de l operateur
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []filmDeTest
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		c := strings.Split(l, "\t")
		if len(c) < 2 {
			return nil, fmt.Errorf("ligne sans build : %q", l)
		}
		f := filmDeTest{id: c[0], build: c[1]}
		if len(c) > 3 {
			f.niveau = c[3]
		}
		out = append(out, f)
	}
	return out, sc.Err()
}

// sousData dit si un chemin passe par un dossier `data`.
func sousData(chemin string) bool {
	abs, err := filepath.Abs(chemin)
	if err != nil {
		return true
	}
	for _, s := range strings.FieldsFunc(abs, func(r rune) bool { return r == '/' || r == '\\' }) {
		if strings.EqualFold(s, "data") {
			return true
		}
	}
	return false
}

// refuserSortieSousData arrete le test quand le fichier de sortie d un instrument tomberait sous un
// dossier `data` (jamais d ecriture d un rapport dans les donnees).
func refuserSortieSousData(t *testing.T, sortie string) {
	t.Helper()
	if sousData(sortie) {
		t.Fatalf("sortie refusee : %s est sous un dossier data", sortie)
	}
}

func TestDivergence(t *testing.T) {
	cfg := config{racine: os.Getenv("DIVERGENCE_RACINE"), temoin: os.Getenv("DIVERGENCE_TEMOIN"), sortie: os.Getenv("DIVERGENCE_SORTIE")}
	if chemin := os.Getenv("DIVERGENCE_CARTE"); chemin != "" {
		cat, err := profile.LoadMapQuantCatalog(chemin)
		if err != nil {
			t.Fatal(err)
		}
		cfg.carte = cat
	}
	liste := os.Getenv("DIVERGENCE_FILMS")
	if cfg.racine == "" || cfg.temoin == "" || cfg.sortie == "" || liste == "" {
		t.Skip("DIVERGENCE_RACINE, DIVERGENCE_FILMS, DIVERGENCE_TEMOIN et DIVERGENCE_SORTIE requis")
	}
	refuserSortieSousData(t, cfg.sortie)
	films, err := lireLesFilms(liste)
	if err != nil {
		t.Fatal(err)
	}
	g := nouvelAgregat()
	for _, f := range films {
		if seul := os.Getenv("DIVERGENCE_ID"); seul != "" && seul != f.id {
			continue
		}
		garde := memguard.Arm(context.Background(), "film_re/divergence", plafondGiB, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets) - arret\n", f.id, pic)
			os.Exit(3)
		})
		cfg.niveau = f.niveau
		if err := traiterUnFilm(cfg, f.id, f.build, g); err != nil {
			garde.Disarm()
			t.Fatalf("film %s : %v", f.id, err)
		}
		t.Logf("%s (%s) : pic %d Mio", f.id, f.build, garde.Peak()>>20)
		garde.Disarm()
		runtime.GC()
	}
	if err := g.ecrireGlobal(cfg.sortie); err != nil {
		t.Fatal(err)
	}
	enDesaccord := 0
	for _, c := range g.controle {
		if c.desaccord {
			enDesaccord++
			t.Errorf("controle de l'instrument : desaccord sur %s %d/%d : %s", c.film, c.chunk, c.paquet, c.detail)
		}
	}
	t.Logf("controle : %d desaccords sur %d paquets tires (population %d, en desaccord %d)", enDesaccord, len(g.controle), g.candidats, g.candidatsEnDesaccord)
}
