//go:build research

// Command cmd_fermeture mesure un corpus de films, UN A LA FOIS, sous l un ou l autre de ses trois
// MODES (`-mode`, liste separee par des virgules ; defaut `fermeture`) :
//
//   - `fermeture` : la CARTE DE FERMETURE DES TRAMES DELTA (lot J4.0 du plan de suite de l audit
//     du decodeur, item J4.0.4) : par build, la part des paquets fermes au bit pres par vue, la
//     part des RECORDS UTILES fermes — le declencheur du chantier de representation
//     intermediaire — et le classement des causes d arret, c est-a-dire la liste courte de ce qui
//     merite Ghidra (ou une largeur mesuree presumee, decision DU-9) ;
//   - `gb1` : la MESURE PREALABLE DU CONSTAT GB-1 (lot J5.0) : les vies (slot, generation) du
//     bipede, celles que le filtre de production laisse sans position (la generation 1 seule
//     avant le lot J5.2, les generations vivantes depuis), `durationMs` contre la duree
//     du film, et les en-tetes dont (slot, tag) n est aucune vie connue (cf. gb1.go) ;
//   - `v2` : la CARTE DE FERMETURE V2 (campagne de recherche sur la grammaire, phase 1, etape 1,
//     2026-10-01) : la carte de `fermeture` A L IDENTIQUE (memes TSV, memes valeurs), plus la
//     ventilation de « vue C : terminateur hors cadre », les rejets de la vue B contre le bloc de
//     type 1, le denominateur des entrees de controle utiles, le compte declare du chunk des temps
//     forts et le mode borne (cf. v2.go). `v2` implique `fermeture`.
//
// Il ne compile QUE sous le tag `research`. Il lit les films EN PLACE, UN A LA FOIS, dans l ordre
// donne, sous la sentinelle memoire de `filmproc` armee film par film ; il n ecrit que dans le
// repertoire `-sortie`, qui doit etre HORS de `data/`.
//
//	cd apps/go-api
//	go run -tags=research ./internal/games/halo_infinite/film/research/cmd_fermeture \
//	  -racine <parc>/data/cache/film_chunks -films 0797ce72,bfecd02b -sortie <dossier hors data> \
//	  [-mode fermeture,gb1,v2] [-mpp-declare]
//
// `-mpp-declare` pose sur chaque film le decoupage MPP que la grammaire resout — celui de la cuisson
// (cf. mpp_declare.go) — au lieu du profil par defaut, et l ecrit par film dans `mpp_declare.tsv`.
//
// La mesure de fermeture est `grammar.FrameClosure` (la marche de production, aucune lecture de
// bits de plus) — `grammar.FrameClosureDetaillee` en mode `v2`, qui rend la meme carte — sous le contexte des instruments (`grammar.ContexteDeFilm` : largeurs d axe lues
// dans le film, profil par defaut). Le verrou de decodage de `filmproc` n est PAS pris : il
// ecrirait un fichier sous la racine du cache ; la serialisation des decodages sur la machine est
// celle de l operateur, comme pour `cmd_grenadeids`.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
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

// Les noms de mode reconnus par `-mode`.
const (
	modeFermeture = "fermeture"
	modeGB1       = "gb1"
	modeV2        = "v2"
)

// modes dit quelles mesures l outil fait sur chaque film. `v2` implique `fermeture`.
type modes struct{ fermeture, gb1, v2 bool }

func main() {
	racine := flag.String("racine", "", "racine portant les repertoires de films (lecture seule)")
	films := flag.String("films", "", "identifiants de films, separes par des virgules, mesures dans cet ordre")
	limite := flag.Int("limite", 0, "nombre maximal de films mesures ; 0 = tous")
	sortie := flag.String("sortie", "", "repertoire du rapport, HORS de data/ (cree s il manque)")
	table := flag.String("table", tableParDefaut, "chemin de ecs_table.tsv (usage produit et statuts)")
	top := flag.Int("top", 30, "nombre de causes d arret classees dans le resume ; 0 = toutes")
	plafond := flag.Int("plafond-gib", plafondParDefautGiB, "plafond memoire par film ; 0 desarme")
	mode := flag.String("mode", modeFermeture, "mesures par film, separees par des virgules : fermeture, gb1, v2 (v2 implique fermeture)")
	fixe := flag.String("denominateur-fixe", "", "v2 : TSV du denominateur fixe consolide (colonnes film et fixe)")
	paquets := flag.Bool("paquets", false, "v2 : ecrire fermeture_paquets.tsv, une ligne par paquet delta")
	mppDeclare := flag.Bool("mpp-declare", false, "poser sur chaque film le decoupage MPP que la grammaire resout "+
		"(format, ou taille declaree n1) ; journal mpp_declare.tsv")
	flag.Parse()

	ids := borner(decouper(*films), *limite)
	md, errMode := lireModes(*mode)
	if *racine == "" || len(ids) == 0 || *sortie == "" || errMode != nil {
		fmt.Fprintln(os.Stderr, "usage : -racine <dir> -films <id,id,...> -sortie <dir hors data> "+
			"[-limite N] [-mode fermeture,gb1,v2]")
		if errMode != nil {
			fmt.Fprintln(os.Stderr, errMode)
		}
		os.Exit(2)
	}
	opts, errFixe := lireOptionsV2(*fixe, *paquets)
	if errFixe != nil {
		fmt.Fprintln(os.Stderr, errFixe)
		os.Exit(2)
	}
	rap, err := preparer(*sortie, *table, md, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	jm, err := ouvrirJournalMPP(*sortie, *mppDeclare)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, id := range ids {
		if err := mesurerUnFilm(*racine, id, *plafond, rap, jm); err != nil {
			fmt.Fprintf(os.Stderr, "%s : %v\n", id, err)
			rap.echecs++
		}
		runtime.GC()
	}
	if err := jm.fermer(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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

// preparer verifie et cree le repertoire du rapport, relit la table ECS si le mode `fermeture`
// la demande, et ouvre le rapport.
func preparer(sortie, table string, md modes, opts optionsV2) (*rapport, error) {
	if err := preparerSortie(sortie); err != nil {
		return nil, err
	}
	var tab tableECS
	if md.fermeture {
		var err error
		if tab, err = lireTable(table); err != nil {
			return nil, err
		}
	}
	return ouvrirRapport(sortie, tab, md, opts)
}

// lireModes lit la valeur de `-mode`.
func lireModes(v string) (modes, error) {
	var md modes
	for _, m := range decouper(v) {
		switch m {
		case modeFermeture:
			md.fermeture = true
		case modeGB1:
			md.gb1 = true
		case modeV2:
			md.fermeture, md.v2 = true, true
		default:
			return modes{}, fmt.Errorf("-mode : %q inconnu (fermeture, gb1, v2)", m)
		}
	}
	if !md.fermeture && !md.gb1 {
		return modes{}, errors.New("-mode : aucun mode demande")
	}
	return md, nil
}

// mesurerUnFilm ouvre UN film, le mesure sous sa propre sentinelle dans chacun des modes
// demandes, ecrit ses lignes, et le laisse partir avant le suivant.
func mesurerUnFilm(racine, id string, plafondGiB int, rap *rapport, jm *journalMPP) error {
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
	if err := jm.poser(fc, id); err != nil {
		return fmt.Errorf("journal MPP : %w", err)
	}
	build := buildDuFilm(fc)
	if rap.modes.fermeture {
		carte, v2, err := mesurerLaCarte(fc, id, filepath.Join(racine, id), rap)
		if err != nil {
			return err
		}
		m := mesureFilm{id: id, build: build, carte: carte, pic: garde.Peak(),
			duree: time.Since(debut), largeursLues: errLargeurs == nil}
		fmt.Printf("%s  build=%s  paquets=%d/%d  utiles=%d/%d  pic=%s  %s\n", id, m.build,
			carte.PaquetsFermes, carte.Paquets, carte.Utiles.RecordsFermes, carte.Utiles.Records,
			mio(m.pic), m.duree.Round(time.Millisecond))
		if err := rap.ajouter(m); err != nil {
			return err
		}
		if v2 != nil {
			if err := rap.v2.ajouter(id, build, v2); err != nil {
				return err
			}
		}
	}
	if rap.modes.gb1 {
		if err := mesurerEtEcrireGB1(fc, id, build, garde, rap); err != nil {
			return err
		}
	}
	rap.mesures++
	return nil
}

// mesurerLaCarte mesure la carte de fermeture d un film deja ouvert : `grammar.FrameClosure` en
// mode `fermeture`, `grammar.FrameClosureDetaillee` (la meme carte, plus le detail de chaque
// paquet, et le chunk des temps forts) en mode `v2`. La mesure v2 est nil hors de ce mode.
func mesurerLaCarte(fc *grammar.FilmContext, id, dir string, rap *rapport) (grammar.FrameClosureReport, *mesureV2, error) {
	if !rap.modes.v2 {
		carte, err := grammar.FrameClosure(fc, rap.tab.utiles)
		return carte, nil, err
	}
	var paquets io.Writer
	if rap.v2.paquets != nil {
		paquets = rap.v2.paquets
	}
	col := nouveauCollecteurV2(fc, id, paquets)
	carte, err := grammar.FrameClosureDetaillee(fc, rap.tab.utiles, col.voir)
	if err == nil {
		err = col.err
	}
	if err != nil {
		return carte, nil, err
	}
	col.m.chunk3 = mesurerChunk3(dir)
	return carte, col.m, nil
}

// mesurerEtEcrireGB1 fait la mesure GB-1 d un film deja ouvert et l ajoute au rapport.
func mesurerEtEcrireGB1(fc *grammar.FilmContext, id, build string, garde *filmproc.Guard, rap *rapport) error {
	debut := time.Now()
	g, err := mesurerGB1(fc)
	if err != nil {
		return fmt.Errorf("gb1 : %w", err)
	}
	g.id, g.build, g.pic, g.duree = id, build, garde.Peak(), time.Since(debut)
	fmt.Printf("%s  gb1  vies=%d  sans_position_prod=%d  sans_position_vivante=%d  orphelins=%d  pic=%s  %s\n",
		id, len(g.vies), g.viesSansPositionProd(), g.viesSansPositionVivante(), g.totalOrphelins(),
		mio(g.pic), g.duree.Round(time.Millisecond))
	return rap.gb1.ajouter(g)
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
	for seg := range strings.SplitSeq(filepath.ToSlash(filepath.Clean(abs)), "/") {
		if strings.EqualFold(seg, repertoireInterdit) {
			return errors.New("-sortie : le rapport s ecrit HORS de data/ (" + abs + ")")
		}
	}
	return os.MkdirAll(abs, 0o750)
}

// decouper rend les identifiants non vides de la liste.
func decouper(v string) []string {
	var out []string
	for p := range strings.SplitSeq(v, ",") {
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
