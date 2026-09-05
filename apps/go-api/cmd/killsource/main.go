// Commande killsource — exercer le decodeur de la SOURCE DE DEGAT sur un film Theater, seul.
//
// A QUOI ELLE SERT. AUCUN CHEMIN D EXECUTION DE L APPLICATION ne passe encore par le paquet
// `internal/games/halo_infinite/film/killsource` : son unique importeur cote application est le
// pont `internal/sync/killsource_bridge.go`, ecrit d avance et lui-meme sans appelant. Le paquet
// est donc autonome, et il doit pouvoir etre teste a fond avant d etre branche. Cette commande est
// ce moyen. Elle ne touche ni la base, ni le reseau, ni les fichiers du jeu : elle lit des chunks
// sur le disque et affiche ce que le decodeur en tire.
//
// LES DEUX VERITES SONT TOUJOURS AFFICHEES COTE A COTE, et c est le point de toute la sortie :
// le kill-feed dit QUI RECOIT LE CREDIT, le dead-state dit D OU VIENT LE DEGAT FATAL. Les deux
// sont vraies, elles repondent a des questions differentes, et quand elles divergent la
// divergence est elle-meme une information. Aucune n est presentee comme corrigeant l autre.
//
// USAGE
//
//	go run ./cmd/killsource kills    <film>            la table des morts, lisible
//	go run ./cmd/killsource json     <film>            la meme chose, en JSON
//	go run ./cmd/killsource sante    <film>            la metrique de sante et son verdict
//	go run ./cmd/killsource comparer <film> <film>     ce qui change entre deux films
//
// `<film>` est soit un identifiant court de 8 caracteres (resolu sous `-cache`), soit un chemin
// de repertoire contenant `chunk_NN.bin`. Pour lister les films en cache :
// `ls data/cache/film_chunks`.
//
// SORTIE ET JOURNALISATION — la distinction est volontaire : le RAPPORT est le livrable de cette
// commande, il part sur la sortie standard via `fmt` ; les erreurs et les diagnostics partent sur
// l erreur standard via `slog`, comme partout ailleurs dans le depot.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
)

// defaultCacheDir : la racine du cache de films, relative a `apps/go-api/`. Meme valeur que
// `cmd/fetch_film_chunks`, qui est ce qui remplit ce cache.
const defaultCacheDir = "../../data/cache"

// shortIDLen : longueur d un identifiant court de film (prefixe hexadecimal du match).
const shortIDLen = 8

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	o, films, err := analyserArgs(os.Args[1:])
	if err != nil {
		usage()
		slog.Error("killsource", "err", err)
		os.Exit(2)
	}
	if err := run(films, o); err != nil {
		slog.Error("killsource", "err", err)
		os.Exit(1)
	}
}

// analyserArgs : les options peuvent se placer N IMPORTE OU sur la ligne de commande.
//
// `flag` de la bibliotheque standard s arrete a la premiere valeur qui n est pas une option :
// `killsource kills 000d5950 -limite 3` lui rend TROIS arguments positionnels et zero option.
// C est un piege classique, et il rendrait fausses les commandes ecrites dans le guide d usage.
// On separe donc soi-meme les options des positionnels avant de laisser `flag` faire son travail.
func analyserArgs(args []string) (options, []string, error) {
	opts, pos := separer(args)
	fs := flag.NewFlagSet("killsource", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = usage
	cacheDir := fs.String("cache", defaultCacheDir,
		"Racine du cache de films (contient film_chunks/<id>/chunk_NN.bin)")
	limit := fs.Int("limite", 0,
		"N afficher que les N premieres morts de la table (0 = toutes)")
	full := fs.Bool("tout", false,
		"Table : ajouter les colonnes techniques (tag brut, voie de lecture, statut)")
	profil := fs.String("profil", "",
		"Ecrire un profil CPU pprof dans ce fichier (go tool pprof <binaire> <fichier>)")
	if err := fs.Parse(opts); err != nil {
		return options{}, nil, err
	}
	return options{cache: *cacheDir, limit: *limit, full: *full, profil: *profil}, pos, nil
}

// optionsAvecValeur : les options qui consomment l argument suivant. Une option booleenne n en
// consomme pas — la confondre avalerait un nom de film.
var optionsAvecValeur = map[string]bool{"cache": true, "limite": true, "profil": true}

// separer : partage les arguments entre options et positionnels, dans l ordre d apparition.
func separer(args []string) (opts, pos []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") || a == "-" {
			pos = append(pos, a)
			continue
		}
		opts = append(opts, a)
		nom := strings.TrimLeft(a, "-")
		if strings.Contains(nom, "=") || !optionsAvecValeur[nom] {
			continue
		}
		if i+1 < len(args) {
			i++
			opts = append(opts, args[i])
		}
	}
	return opts, pos
}

// options : les reglages d AFFICHAGE de la commande. Ils ne touchent JAMAIS au decodage — la
// configuration du decodeur est gelee dans le paquet et n est pas pilotable d ici, deliberement.
type options struct {
	cache string
	limit int
	full  bool
	// profil : fichier de sortie d un profil CPU pprof. Il existe parce que le cout du decodage
	// est VIOLEMMENT superlineaire (0,20 s par chunk a 8 chunks, 16,6 s a 69 — facteur 83) et
	// qu une telle pente s instruit par la mesure, jamais au jugement.
	profil string
}

func run(args []string, o options) error {
	if len(args) == 0 {
		usage()
		return errors.New("aucune commande")
	}
	if o.profil != "" {
		arret, err := demarrerProfil(o.profil)
		if err != nil {
			return err
		}
		defer arret()
	}
	switch args[0] {
	case "kills":
		return withFilm(args[1:], o, func(r *rapport) error { return afficherTable(r, o) })
	case "json":
		return withFilm(args[1:], o, afficherJSON)
	case "sante":
		return withFilm(args[1:], o, afficherSante)
	case "comparer":
		return comparer(args[1:], o)
	default:
		usage()
		return fmt.Errorf("commande inconnue : %q", args[0])
	}
}

// demarrerProfil : ouvre le fichier de profil et lance l enregistrement CPU. Rend la fonction
// d arret, qui ferme les deux — un profil non ferme est un fichier illisible, donc l erreur de
// fermeture se JOURNALISE au lieu d etre avalee.
func demarrerProfil(chemin string) (func(), error) {
	f, err := os.Create(chemin)
	if err != nil {
		return nil, fmt.Errorf("profil %s : %w", chemin, err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("profil %s : %w", chemin, err)
	}
	return func() {
		pprof.StopCPUProfile()
		if cerr := f.Close(); cerr != nil {
			slog.Error("killsource: fermeture du profil", "fichier", chemin, "err", cerr)
		}
	}, nil
}

// rapport : un film decode, plus de quoi le nommer dans la sortie.
type rapport struct {
	film   string
	dir    string
	duree  time.Duration
	result *killsource.Result
}

// withFilm : resout le film, le decode, et passe le resultat au rendu demande.
func withFilm(args []string, o options, render func(*rapport) error) error {
	if len(args) != 1 {
		return errors.New("il faut exactement un film en argument")
	}
	r, err := decoder(args[0], o.cache)
	if err != nil {
		return err
	}
	return render(r)
}

// decoder : LA SEULE FACON DE LIRE UN FILM ICI, et c est exactement le code que le brancheur
// ecrira — un film charge, un appel, un resultat.
//
// LE CHARGEMENT EST HORS DU CHRONOMETRE depuis le lot 1 de PLAN_CUISSON_PERF (item 1.4) :
// `killsource.Decode` ne lit plus le disque et ne decompresse plus rien, donc `duree` mesure
// le DECODAGE seul — la lecture et l inflate du film, eux, sont le cout de `filmsource.LoadDir`.
func decoder(film, cache string) (*rapport, error) {
	dir, name := resoudre(film, cache)
	src, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, fmt.Errorf("film %s : %w", name, err)
	}
	t0 := time.Now()
	res, err := killsource.Decode(context.Background(), name, src, nil)
	if err != nil {
		return nil, fmt.Errorf("film %s : %w", name, err)
	}
	return &rapport{film: name, dir: dir, duree: time.Since(t0), result: res}, nil
}

// resoudre : un identifiant court devient `<cache>/film_chunks/<id>` ; tout le reste est pris pour
// un chemin tel quel. Rend aussi le nom court a afficher.
func resoudre(film, cache string) (dir, name string) {
	if len(film) == shortIDLen && !strings.ContainsAny(film, `/\.`) {
		return filepath.Join(cache, "film_chunks", film), film
	}
	return film, filepath.Base(strings.TrimRight(filepath.Clean(film), `/\`))
}

func usage() {
	fmt.Fprint(os.Stderr, `killsource — decodeur de la SOURCE DE DEGAT par kill (films Theater Halo Infinite)

  Chaque mort porte DEUX VERITES, affichees cote a cote et jamais l une a la place de l autre :
    - le CREDIT du jeu, tel que le kill-feed l affiche (ce sur quoi les stats sont baties) ;
    - la SOURCE DU DEGAT FATAL, lue dans l etat de mort de la victime.
  Quand elles divergent, la divergence est signalee : les deux restent vraies.

COMMANDES
  kills    <film>          la table des morts, lisible
  json     <film>          la meme chose, en JSON (schema stable, denominateurs nommes)
  sante    <film>          la metrique de sante, son verdict et ses alertes
  comparer <film> <film>   les memes quantites sur deux films, cote a cote

ARGUMENT <film>
  un identifiant court de 8 caracteres (ex. 000d5950), resolu sous -cache ;
  ou un chemin de repertoire contenant des fichiers chunk_NN.bin.
  Pour lister les films disponibles : ls data/cache/film_chunks

OPTIONS  (elles se placent n importe ou sur la ligne de commande)
  -cache <dir>   racine du cache de films (defaut ../../data/cache depuis apps/go-api)
  -limite N      n afficher que les N premieres morts de la table (0 = toutes)
  -tout          table : ajouter le tag brut, la voie de lecture et le statut
`)
	fmt.Fprint(os.Stderr, `
EXEMPLES
  go run ./cmd/killsource kills 000d5950
  go run ./cmd/killsource kills 9b191a7f -tout -limite 20
  go run ./cmd/killsource sante 4f77afc1
  go run ./cmd/killsource comparer 000d5950 fccc61cd
  go run ./cmd/killsource json fccc61cd > fccc61cd.json

CE QU IL FAUT SAVOIR AVANT DE LIRE UNE SORTIE
  - le perimetre valide est le 4v4. En BTB la publication ligne par ligne est refusee
    (la bijection indice -> joueur y a une marge nulle) : seul l agregat vaut.
  - une source sans nom propre publiable s affiche << Autres >>. La NATURE reste affichee.
  - un decodage a la fois par processus : le decodeur serialise les passes lui-meme.
`)
}
