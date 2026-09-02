package main

// cmd_backfill_medailles_feed.go — sous-commande `levelup backfill-medailles-feed`.
//
// ELLE REND LEUR NOM AUX MEDAILLES DEJA EN BASE. Depuis la bascule Collect->Persist
// (avril 2026) le collecteur ecrivait les events `medal` de `highlight_events` sans
// `raw_json` : 415 matchs, 22 031 events, aucune identite. Le fil des eliminations
// lit `raw_json.medal_name` — sans lui il n affiche AUCUNE medaille.
//
// LES DEUX ECRIVAINS SONT REPARES depuis le 2026-09-02 — le flux primaire
// (sync/collect.go) ET la voie completion/convergence
// (sync/engine_highlight_events.go), qui reecrit les events des matchs dont le film
// n etait pas publie au sync primaire. Cette passe rattrape l existant.
//
// # ELLE EST 100 % HORS LIGNE — NI RESEAU, NI TOKENS, NI CDN
//
// La seule source est le cache de films sur disque (`film_manifests/` +
// `film_chunks/`), lu par [haloclient.LocalFilmCache]. Un match dont le film n est
// pas en cache est CONSIGNE ET SAUTE : il n y a aucun chemin qui parte sur le
// reseau, donc aucune facon pour cette passe d echouer sur une authentification.
//
// # ELLE EXIGE LE SERVEUR ARRETE
//
// Elle tient un handle EN ECRITURE sur `shared_matches_v2.duckdb` pendant toute la
// passe (mono-process, ADR 0013) : `OpenReadWrite` echoue si le serveur tient le
// fichier. C est la meme precondition que `backfill-killsource`.
//
// # ELLE EST REPRENABLE, ET LA CLE EST `raw_json IS NULL`
//
// Le lot est fait des matchs auxquels il manque au moins une identite de medaille.
// Un match entierement rattrape en sort. Un match dont des events restent sans nom
// (couple jamais mesure) ou sans paire (film desaccorde de la base) y revient a
// chaque run — c est le signal, pas un bug.
//
// Usage (SERVEUR ARRETE) :
//
//	levelup backfill-medailles-feed --dry-run          # ce qui SERAIT ecrit, aucune ecriture
//	levelup backfill-medailles-feed --limit 20         # les 20 premiers matchs du lot
//	levelup backfill-medailles-feed                    # tout le lot
//
// Le cache de films se resout par `--cache`, sinon `LEVELUP_LEGACY_FILM_CACHE_DIR`,
// sinon `<repo>/data/cache` — exactement comme `backfill-killsource`.

import (
	"context"
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/medalname"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/haloclient"
)

// medaillesFeedOptions : les reglages de la passe.
type medaillesFeedOptions struct {
	titleSlug string
	cacheDir  string
	limit     int
	dryRun    bool
}

func runBackfillMedaillesFeed(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-medailles-feed", flag.ExitOnError)
	o := medaillesFeedOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.StringVar(&o.cacheDir, "cache", "",
		"racine du cache de films (defaut : LEVELUP_LEGACY_FILM_CACHE_DIR puis <repo>/data/cache)")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre de matchs traites (0 = tous)")
	fs.BoolVar(&o.dryRun, "dry-run", false, "afficher le bilan sans rien ecrire")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(o.titleSlug)
	if _, err := os.Stat(sharedPath); err != nil {
		return fmt.Errorf("shared_matches introuvable (%s): %w", sharedPath, err)
	}

	cacheRoot := resoudreCacheFilms(cfg, o.cacheDir)
	cache := haloclient.NewLocalFilmCache(cacheRoot)
	if cache == nil {
		return fmt.Errorf("cache de films introuvable sous %s — cette passe est HORS LIGNE, "+
			"elle n a pas d autre source", cacheRoot)
	}

	handle, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared RW (%s): %w (serveur arrete ?)", sharedPath, err)
	}
	defer handle.Close()
	db := handle.SQLDb()

	// LES TABLES DOIVENT EXISTER AVANT D ECRIRE : le serveur joue les migrations a
	// son demarrage, cette commande tourne serveur ARRETE. `RunForDB` est idempotent.
	if err := migrerSchemaPartage(db, o.titleSlug); err != nil {
		return err
	}

	mention := ""
	if o.dryRun {
		mention = " (dry-run)"
	}
	fmt.Printf("backfill medailles : cache %s, titre %s%s\n", cacheRoot, o.titleSlug, mention)
	bilan, err := ops.BackfillIdentiteMedailles(ctx, db, chunksHighlightDuCache{cache: cache},
		medalname.Lookup, ops.OptionsBackfillMedailles{Plafond: o.limit, DryRun: o.dryRun})
	if err != nil {
		return err
	}
	fmt.Printf("matchs : %d candidats, %d rattrapes, %d sans film, %d illisibles\n",
		bilan.MatchsCandidats, bilan.MatchsTraites, bilan.MatchsSansFilm, bilan.MatchsIllisibles)
	fmt.Printf("events : %d candidats, %d identifies, %d sans nom (couple inconnu), %d sans paire\n",
		bilan.EventsCandidats, bilan.EventsIdentifies, bilan.EventsSansNom, bilan.EventsSansPaire)
	return nil
}

// chunksHighlightDuCache lit LE SEUL chunk dont la passe a besoin.
//
// Elle ne passe PAS par `killcollector.LocalCacheFilms` : celle-ci charge TOUS les
// chunks d un film (le plus gros du corpus pese 88 Mio sur disque) alors que
// l identite des medailles tient dans le seul chunk highlight. Charger le reste
// serait payer un ordre de grandeur de memoire pour rien.
type chunksHighlightDuCache struct {
	cache *haloclient.LocalFilmCache
}

// ChunkHighlight rend le chunk de type highlight (3) du film en cache.
// `trouve=false` = manifeste absent, aucun chunk de ce type, ou fichier de chunk
// manquant — dans les trois cas le match n a pas de film exploitable ici.
func (c chunksHighlightDuCache) ChunkHighlight(_ context.Context, matchID string) ([]byte, bool, error) {
	manifest, err := c.cache.LoadManifest(matchID)
	if err != nil {
		return nil, false, fmt.Errorf("manifeste du film %s: %w", matchID, err)
	}
	if manifest == nil {
		return nil, false, nil
	}
	for _, chunk := range manifest.Chunks {
		if chunk.ChunkType != haloclient.FilmChunkTypeHighlightEvents {
			continue
		}
		data, err := c.cache.LoadChunk(matchID, chunk.Index)
		if err != nil {
			return nil, false, fmt.Errorf("chunk highlight du film %s: %w", matchID, err)
		}
		if len(data) == 0 {
			return nil, false, nil
		}
		return data, true, nil
	}
	return nil, false, nil
}
