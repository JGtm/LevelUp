package main

// cmd_backfill_replay_child.go — L'ENFANT de `backfill-replay` : UN film, puis il meurt.
//
// Ce processus est lance par le parent (cf. backfill_child.go) et n'est pas destine a la
// main de l'operateur : il ne planifie rien, ne saute rien, ne compte rien. Il cuit LE film
// qu'on lui nomme et rend un CODE DE SORTIE que le parent traduit en categorie de recap.
//
// # POURQUOI L'ENFANT RELIT SES PROPRES FAITS DE MATCH
//
// Le parent les chargeait autrefois TOUS, d'un coup, dans une map indexee par match — et la
// gardait vivante pendant toute la passe. C'etait la seule structure du processus qui
// croissait avec le CORPUS et non avec le film : exactement ce qu'un blindage memoire doit
// supprimer. Chaque enfant ouvre donc la base en LECTURE, prend SES faits, et RELACHE le
// handle AVANT de decoder. Les enfants sont sequentiels : il n'y a jamais deux lecteurs.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
)

// runBackfillReplayUn cuit UN film et rend le code de sortie du protocole parent/enfant.
//
// IL NE REND JAMAIS D'ERREUR A `main` : le code de sortie EST le canal de retour. Une erreur
// rendue a main sortirait en 1, que le protocole reserve aux morts hors categorie.
func runBackfillReplayUn(cfg *config.AppConfig, o replayBackfillOptions, cacheRoot string) int {
	sentinelle := armerPlafondMemoire(o.memLimitGiB)
	// Le pic part sur TOUTES les sorties ordinaires. La sortie par la sentinelle, elle,
	// l'emet elle-meme : `os.Exit` ne joue pas les differes.
	defer func() { emettrePicMemoire(sentinelle.picObserve()) }()

	ctx := context.Background()
	builder, err := replaybuild.NewBuilder(cfg.RepoRoot, o.titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "backfill-replay (enfant): builder indisponible",
			"err", err, "match_id", o.one, "title", o.titleSlug)
		return codeEnfantPreparation
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	faits := chargerFaitsUnMatch(ctx, pr, o.titleSlug, o.one)
	filmDir := filmcache.ChunkDir(cacheRoot, titlePkg.FilmShortMatchID(o.one))

	// Le PARENT passe les identites de carte via `--map-name` ; elles gagnent toujours. Sans
	// elles (forme `--one` TAPEE A LA MAIN), l'enfant les resout lui-meme depuis le registre
	// plutot que d'echouer « carte hors catalogue ([]) » sur une carte pourtant au catalogue.
	mapNames := o.mapNames
	if len(mapNames) == 0 {
		mapNames = mapNamesForOne(ctx, pr, o.titleSlug, o.one)
	}

	out, berr := builder.BuildMatch(o.one, mapNames, filmDir, faits)
	switch {
	case berr == nil:
		fmt.Printf("  %s : %d tracks, %d octets (%s)\n", o.one, out.Tracks, out.Bytes, out.Module)
		return codeEnfantOK
	case errors.Is(berr, replaybuild.ErrMapNotInCatalog):
		fmt.Printf("  %s : carte hors catalogue (%v) — echec voulu\n", o.one, mapNames)
		return codeEnfantHorsCatalogue
	default:
		slog.ErrorContext(ctx, "backfill-replay (enfant): decodage en echec",
			"err", berr, "match_id", o.one)
		fmt.Printf("  %s : ERREUR %v\n", o.one, berr)
		return codeEnfantErreurDecodage
	}
}

// chargerFaitsUnMatch lit, en UNE ouverture RO RELACHEE AVANT LE DECODAGE, ce que la base
// sait de CE match : lignes de match (pont d'identite des joueurs), scores des deux camps
// (identite des camps) et nom de variante (famille d'objectif).
//
// Une base indisponible (serveur en ecriture) DEGRADE l'artefact — il sort sans compteurs de
// joueur ni actions d'objectif — plutot que de faire echouer le film.
func chargerFaitsUnMatch(
	ctx context.Context, pr *titlePkg.PathResolver, titleSlug, matchID string,
) port.MatchFacts {
	sharedPath := pr.SharedDBPath(titleSlug)
	db, release, err := duckdb.OpenReadForQuery(sharedPath)
	if err != nil {
		slog.WarnContext(ctx, "backfill-replay (enfant): base indisponible — artefact sans "+
			"compteurs de joueur ni actions d'objectif", "err", err, "match_id", matchID)
		return port.MatchFacts{}
	}
	defer release()

	var repo port.ReplayFactsRepo = duckdb.NewReplayFactsRepo(db)
	facts, ferr := repo.FactsForMatch(ctx, matchID)
	if ferr != nil {
		slog.WarnContext(ctx, "backfill-replay (enfant): faits de match illisibles",
			"err", ferr, "match_id", matchID)
		return port.MatchFacts{}
	}
	return facts
}
