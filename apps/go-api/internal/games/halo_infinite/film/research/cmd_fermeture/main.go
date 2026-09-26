//go:build research

// Command cmd_fermeture mesure la CARTE DE FERMETURE DES TRAMES DELTA sur un corpus de films (lot
// J4.0 du plan de suite de l audit du decodeur, item J4.0.4) : par build, la part des paquets
// fermes au bit pres par vue, la part des RECORDS UTILES fermes — le declencheur du chantier de
// representation intermediaire — et le classement des causes d arret, c est-a-dire la liste
// courte de ce qui merite Ghidra (ou une largeur mesuree presumee, decision DU-9).
//
// Il ne compile QUE sous le tag `research`. Il lit les films EN PLACE, UN A LA FOIS, dans l ordre
// donne, sous la sentinelle memoire de `filmproc` armee film par film ; il n ecrit que dans le
// repertoire `-sortie`, qui doit etre HORS de `data/`.
//
//	cd apps/go-api
//	go run -tags=research ./internal/games/halo_infinite/film/research/cmd_fermeture \
//	  -racine <parc>/data/cache/film_chunks -films 0797ce72,bfecd02b -sortie <dossier hors data>
//
// La mesure elle-meme est `grammar.FrameClosure` (la marche de production, aucune lecture de bits
// de plus), sous le contexte des instruments (`grammar.ContexteDeFilm` : largeurs d axe lues dans
// le film, profil par defaut). Le verrou de decodage de `filmproc` n est PAS pris : il ecrirait
// un fichier sous la racine du cache ; la serialisation des decodages sur la machine est celle de
// l operateur, comme pour `cmd_grenadeids`.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// plafondParDefautGiB : le plafond memoire de l outil, par film. La marche des trames d un film
// complet est celle d un balayage de cuisson, sans l assemblage du document ; ce plafond est une
// ceinture, pas un dimensionnement.
const plafondParDefautGiB = 4

// nomOutil : le nom que la sentinelle journalise.
const nomOutil = "film_re/fermeture"

// tableParDefaut : la table ECS du depot, relative a `apps/go-api`.
const tableParDefaut = "internal/games/halo_infinite/film/internal/grammar/testdata/ecs_table.tsv"

// repertoireInterdit : le rapport ne s ecrit jamais sous un repertoire de ce nom.
const repertoireInterdit = "data"

func main() {
	racine := flag.String("racine", "", "racine portant les repertoires de films (lecture seule)")
	films := flag.String("films", "", "identifiants de films, separes par des virgules, mesures dans cet ordre")
	limite := flag.Int("limite", 0, "nombre maximal de films mesures ; 0 = tous")
	sortie := flag.String("sortie", "", "repertoire du rapport, HORS de data/ (cree s il manque)")
	table := flag.String("table", tableParDefaut, "chemin de ecs_table.tsv (usage produit et statuts)")
	top := flag.Int("top", 30, "nombre de causes d arret classees dans le resume ; 0 = toutes")
	plafond := flag.Int("plafond-gib", plafondParDefautGiB, "plafond memoire par film ; 0 desarme")
	flag.Parse()

	ids := borner(decouper(*films), *limite)
	if *racine == "" || len(ids) == 0 || *sortie == "" {
		fmt.Fprintln(os.Stderr, "usage : -racine <dir> -films <id,id,...> -sortie <dir hors data> [-limite N]")
		os.Exit(2)
	}
	if err := preparerSortie(*sortie); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	tab, err := lireTable(*table)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	rap, err := ouvrirRapport(*sortie, tab)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, id := range ids {
		if err := mesurerUnFilm(*racine, id, *plafond, rap); err != nil {
			fmt.Fprintf(os.Stderr, "%s : %v\n", id, err)
			rap.echecs++
		}
		runtime.GC()
	}
	if err := rap.terminer(*top); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%d film(s) mesure(s), %d echec(s) ; rapport dans %s\n", rap.mesures, rap.echecs, *sortie)
	if rap.echecs > 0 {
		os.Exit(1)
	}
}

// mesurerUnFilm ouvre UN film, le mesure sous sa propre sentinelle, ecrit ses lignes, et le
// laisse partir avant le suivant.
func mesurerUnFilm(racine, id string, plafondGiB int, rap *rapport) error {
	garde := filmproc.Arm(nomOutil, plafondGiB, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets) — arret\n", id, pic)
		os.Exit(3)
	})
	defer garde.Disarm()
	debut := time.Now()
	fc, _, errLargeurs := grammar.ContexteDeFilm(filepath.Join(racine, id))
	if fc == nil {
		return fmt.Errorf("film illisible : %w", errLargeurs)
	}
	carte, err := grammar.FrameClosure(fc, rap.tab.utiles)
	if err != nil {
		return err
	}
	m := mesureFilm{id: id, build: buildDuFilm(fc), carte: carte, pic: garde.Peak(),
		duree: time.Since(debut), largeursLues: errLargeurs == nil}
	fmt.Printf("%s  build=%s  paquets=%d/%d  utiles=%d/%d  pic=%s  %s\n", id, m.build,
		carte.PaquetsFermes, carte.Paquets, carte.Utiles.RecordsFermes, carte.Utiles.Records,
		mio(m.pic), m.duree.Round(time.Millisecond))
	return rap.ajouter(m)
}

// buildDuFilm rend le build en clair de la section d identification de `chunk_00`, ou
// `version-<n>` pour un film anterieur a cette section.
func buildDuFilm(fc *grammar.FilmContext) string {
	film := fc.Film()
	if raw, ok := grammar.FilmRegistryChunk(film); ok {
		if ident, err := grammar.ReadFilmIdentity(raw); err == nil && ident.Build != "" {
			return ident.Build
		}
	}
	if v, ok := grammar.FilmMajorVersion(film); ok {
		return fmt.Sprintf("version-%d", v)
	}
	return "inconnu"
}

// preparerSortie refuse un repertoire de rapport situe sous `data/`, puis le cree.
func preparerSortie(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("-sortie %q : %w", dir, err)
	}
	for _, seg := range strings.Split(filepath.ToSlash(filepath.Clean(abs)), "/") {
		if strings.EqualFold(seg, repertoireInterdit) {
			return errors.New("-sortie : le rapport s ecrit HORS de data/ (" + abs + ")")
		}
	}
	return os.MkdirAll(abs, 0o750)
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

// borner garde les `limite` premiers identifiants (tous quand `limite` <= 0).
func borner(ids []string, limite int) []string {
	if limite > 0 && len(ids) > limite {
		return ids[:limite]
	}
	return ids
}
