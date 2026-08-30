package main

// cmd_archive_films.go — sous-commande `levelup archive-films`.
//
// ELLE CONSERVE LES FILMS, ELLE N EN DECODE AUCUN. C est toute la difference avec
// `backfill-killsource --online`, et elle justifie une commande a part :
//
//	                        archive-films              backfill-killsource --online
//	but                     SAUVER les octets          produire assist_known / les tirs
//	perimetre               tout match sans film        les matchs a redecoder
//	                        en cache local              (marqueur terminal exclu)
//	cout par match          un manifeste + N chunks     idem PLUS ~19 s de decodage
//	base partagee           lecture seule               ecriture
//	ordre                   du PLUS VIEUX au plus       du plus RECENT au plus vieux
//	                        recent
//
// # POURQUOI LE PERIMETRE EST PLUS LARGE QUE CELUI DU DECODAGE
//
// Le decodage saute les matchs marques `MBitWeaponKillsNoFilm` : leur film a repondu 404 a une
// passe precedente, les redecoder ne produirait rien. L archivage, lui, ne les saute PAS par
// defaut. Un manifeste coute UNE petite requete, et c est la seule facon de transformer un
// marqueur pose il y a des mois en fait mesure aujourd hui. Mesure du 2026-08-30 : 917 matchs
// sur 1 948 n ont aucun film en local, dont 579 anterieurs a 2025 qui n ont JAMAIS ete sondes.
// `--sauter-marques` les exclut quand on veut economiser les requetes.
//
// # POURQUOI L ORDRE EST INVERSE PAR RAPPORT AU DECODAGE
//
// Ce n est pas une incoherence, c est le meme raisonnement applique a deux buts differents.
// Le decodage prend les RECENTS d abord parce que les vieux sont deja perdus et que l ecran
// montre les recents. L archivage prend les VIEUX d abord parce qu il court apres l expiration :
// parmi les films ENCORE servis, les plus vieux sont ceux a qui il reste le moins de temps, et
// un film expire ne se retelecharge jamais (`internal/scheduler/replay_purge_cron.go`).
//
// # SERVEUR ARRETE, COMME LES AUTRES PASSES — ET LA RAISON N EST PAS CELLE QU ON CROIT
//
// Cette commande n ECRIT rien sur la base : elle ne lit que la liste des matchs, par
// `OpenReadForQuery` (jamais `OpenReadOnly` force). On pourrait croire qu elle tourne donc
// serveur allume. **C est faux, et c est mesure** : DuckDB n autorise qu UN SEUL processus par
// fichier (CLAUDE.md, regle des ecritures n°4). Un handle RW tenu ailleurs fait echouer
// l ouverture en lecture avec « File is already open in ... (PID …) » — verifie le 2026-08-30
// pendant qu une passe de backfill tenait la base.
//
// Ce que la lecture seule apporte tout de meme : la passe ne peut RIEN corrompre, et
// l interrompre a n importe quel instant ne laisse aucun etat a moitie ecrit cote base. Cote
// cache, `filmcache.Write` ecrit les chunks d abord et le manifeste EN DERNIER : une coupure
// laisse des chunks orphelins invisibles, jamais un film a moitie lisible.
//
// Usage (SERVEUR ARRETE) :
//
//	levelup archive-films --gamertag JGtm --dry-run          # ce qui manque, aucune requete
//	levelup archive-films --gamertag JGtm --limit 50         # un lot
//	levelup archive-films --gamertag JGtm                    # tout ce qui manque
//	levelup archive-films --gamertag JGtm --sauter-marques   # sans les films declares perdus

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/matchflags"
)

// Compteurs de la passe, publies en expvar (ADR 0009) et restitues en fin de commande.
const (
	compteurArchiveSauves  = "archive_films_sauves"
	compteurArchiveExpires = "archive_films_expires"
	compteurArchiveErreurs = "archive_films_erreurs"
	compteurArchiveChunks  = "archive_films_chunks_ecrits"
)

type archiveOptions struct {
	titleSlug     string
	cacheDir      string
	gamertag      string
	limit         int
	rps           int
	dryRun        bool
	sauterMarques bool
}

func runArchiveFilms(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("archive-films", flag.ExitOnError)
	o := archiveOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.StringVar(&o.cacheDir, "cache", "", "racine du cache de films (defaut : LEVELUP_LEGACY_FILM_CACHE_DIR puis <repo>/data/cache)")
	fs.StringVar(&o.gamertag, "gamertag", "", "joueur dont les tokens servent la passe (obligatoire hors --dry-run)")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre de films traites (0 = tous)")
	fs.IntVar(&o.rps, "rps", 4, "debit maximal des requetes Halo")
	fs.BoolVar(&o.dryRun, "dry-run", false, "lister ce qui manque, sans aucune requete ni ecriture")
	fs.BoolVar(&o.sauterMarques, "sauter-marques", false,
		"exclure les matchs deja marques « film absent » par le pipeline (economise des requetes, "+
			"au prix de ne jamais reverifier un marqueur ancien)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !o.dryRun && o.gamertag == "" {
		return fmt.Errorf("archive-films exige --gamertag hors --dry-run : les films se telechargent " +
			"avec les tokens d un joueur declare dans db_profiles.json (n importe lequel — le film " +
			"s obtient par identifiant de MATCH, pas par joueur)")
	}

	ctx := context.Background()
	cacheRoot := resoudreCacheFilms(cfg, o.cacheDir)
	if err := filmcache.EnsureDirs(cacheRoot); err != nil {
		return err
	}

	manquants, err := filmsManquants(ctx, cfg, o, cacheRoot)
	if err != nil {
		return err
	}
	fmt.Printf("films manquants : %d (cache %s)\n", len(manquants), cacheRoot)
	if len(manquants) == 0 || o.dryRun {
		afficherPlanEnLigne(manquants, ordreArchivage)
		return nil
	}

	tokens, err := haloTokensForPlayer(ctx, cfg.RepoRoot, o.gamertag)
	if err != nil {
		return fmt.Errorf("archive-films (%s): %w", o.gamertag, err)
	}
	client := go_sync.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, o.rps)

	debut := time.Now()
	var sauves, expires, erreurs, chunks int
	for i, id := range manquants {
		if ctx.Err() != nil {
			break
		}
		n, etat := archiverUnFilm(ctx, client, cacheRoot, id)
		switch etat {
		case "sauve":
			sauves++
			chunks += n
			observability.IncCounter(compteurArchiveSauves)
			observability.AddInt(compteurArchiveChunks, int64(n))
		case "expire":
			expires++
			observability.IncCounter(compteurArchiveExpires)
		default:
			erreurs++
			observability.IncCounter(compteurArchiveErreurs)
		}
		if (i+1)%25 == 0 {
			fmt.Printf("  [%d/%d] %d sauves, %d expires, %d erreurs — %s\n",
				i+1, len(manquants), sauves, expires, erreurs, time.Since(debut).Round(time.Second))
		}
	}
	fmt.Printf("archive-films : %d films sauves (%d chunks), %d expires cote serveur, %d erreurs — %s\n",
		sauves, chunks, expires, erreurs, time.Since(debut).Round(time.Second))
	if expires > 0 {
		fmt.Printf("  %d films sont DEFINITIVEMENT perdus : 343 ne les sert plus, et un film expire "+
			"ne se retelecharge jamais.\n", expires)
	}
	return nil
}

// archiverUnFilm telecharge la sequence COMPLETE (en-tete + replication + kill-feed) et l ecrit
// au cache. Rend le nombre de chunks et l etat.
//
// ⚠ `GetFilmChunks` et non `GetMatchFilm` : le second ne rend que la REPLICATION_DATA. Archiver
// un film sans son kill-feed le rendrait inutilisable pour la source du kill — c est-a-dire
// archiver a moitie une donnee irremplacable.
func archiverUnFilm(ctx context.Context, client *go_sync.HaloAPIClient, cacheRoot, matchID string) (int, string) {
	chunks, found, err := client.GetFilmChunks(ctx, matchID)
	if err != nil {
		slog.ErrorContext(ctx, "archive-films: telechargement echoue", "match_id", matchID, "err", err)
		return 0, "erreur"
	}
	if !found || len(chunks) == 0 {
		return 0, "expire"
	}
	wc := make([]filmcache.WriteChunk, 0, len(chunks))
	for _, c := range chunks {
		wc = append(wc, filmcache.WriteChunk{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS, DurationMS: c.DurationMS,
			Data: c.Data,
		})
	}
	if err := filmcache.Write(cacheRoot, titlePkg.FilmShortMatchID(matchID), wc); err != nil {
		slog.ErrorContext(ctx, "archive-films: ecriture au cache echouee", "match_id", matchID, "err", err)
		return 0, "erreur"
	}
	return len(chunks), "sauve"
}

// filmsManquants : les matchs du registre dont AUCUN film n est en cache, du plus vieux au plus
// recent (cf. l en-tete : l archivage court apres l expiration).
//
// La presence se juge sur le MANIFESTE, qui est le marqueur de commit de `filmcache.Write` :
// des chunks orphelins sans manifeste ne sont pas lisibles, donc ne comptent pas comme archives.
//
// LECTURE SEULE sur la base partagee — `OpenReadForQuery`, jamais `OpenReadOnly` force : la base
// peut etre tenue en ecriture par le serveur, et cette commande doit pouvoir tourner quand meme.
func filmsManquants(ctx context.Context, cfg *config.AppConfig, o archiveOptions, cacheRoot string) ([]string, error) {
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	db, release, err := duckdb.OpenReadForQuery(pr.SharedDBPath(o.titleSlug))
	if err != nil {
		return nil, fmt.Errorf("ouverture en lecture de la base partagee: %w", err)
	}
	defer release()

	filtre := int64(0)
	if o.sauterMarques {
		filtre = int64(matchflags.MBitWeaponKillsNoFilm)
	}
	rows, err := db.QueryContext(ctx, `
		SELECT match_id
		FROM match_registry
		WHERE COALESCE(backfill_completed, 0) & ? = 0
		ORDER BY `+analysis.SQLStartTimeCanonical("")+`, match_id`, filtre)
	if err != nil {
		return nil, fmt.Errorf("registre des matchs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]string, 0, 256)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("registre des matchs (scan): %w", err)
		}
		if filmDejaEnCache(cacheRoot, id) {
			continue
		}
		out = append(out, id)
		if o.limit > 0 && len(out) >= o.limit {
			break
		}
	}
	return out, rows.Err()
}

// filmDejaEnCache : le manifeste existe-t-il ? Le chemin vient de `filmcache`, jamais recopie.
func filmDejaEnCache(cacheRoot, matchID string) bool {
	_, err := os.Stat(filmcache.ManifestPath(cacheRoot, titlePkg.FilmShortMatchID(matchID)))
	return err == nil
}
