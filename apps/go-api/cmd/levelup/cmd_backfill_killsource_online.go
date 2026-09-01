package main

// cmd_backfill_killsource_online.go — LA PASSE `--online` de `levelup backfill-killsource`.
//
// Fichier dedie plutot qu un ajout a `cmd_backfill_killsource.go` (472 lignes, seuil a 500) :
// la passe hors ligne reste ce qu elle etait, mot pour mot.
//
// # POURQUOI ELLE EXISTE
//
// La passe hors ligne ne decode QUE les films deja en cache. Or ce cache etait alimente par le
// projet Python supprime a la migration : aucun code Go ne creait de nouveau manifeste, et le
// cache s est arrete le 2026-04-07. Depuis cette date, aucun match synchronise n a de passe de
// film, donc `assist_known` y vaut FALSE — « on ne sait pas » — sur 100 % des morts, et les
// deux blocs d assistances de l app se retirent. Mesure complete et datee :
// `.ai/V7.5/REGISTRE_ASSISTANCES_2026-08-29.md`.
//
// # CE QU ELLE FAIT DE PLUS, ET RIEN D AUTRE
//
// Elle change UNE chose : la source de films. [killcollector.RemoteFilms] lit le disque
// d abord, va au reseau ensuite, et ARCHIVE au cache ce qu elle telecharge. Toute la suite —
// decodage, fusion sur la base credit, ecriture — est la chaine existante, inchangee.
//
// # ELLE PREND LES PLUS RECENTS D ABORD
//
// La passe hors ligne trie par cout croissant : une interruption y laisse un resultat presque
// complet. Ce critere ne vaut pas ici, le cout n etant pas connu avant d avoir lu le manifeste
// (deja un aller-retour reseau). Le critere est donc la DATE, et dans le sens des RECENTS :
// un film expire ne se retelecharge jamais, et le pipeline sait deja lesquels le sont
// (`MBitWeaponKillsNoFilm`, exclu par defaut). Ce qui reste de recuperable est recent, et
// c est aussi ce que l utilisateur regarde.
//
// Usage (SERVEUR ARRETE — la commande tient un handle RW sur le shared) :
//
//	levelup backfill-killsource --online --dry-run                 # la liste, aucun octet lu
//	levelup backfill-killsource --online --limit 20 --gamertag JGtm
//	levelup backfill-killsource --online --films-only              # sans la repasse credit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	go_sync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/sync/killcollector"
	"levelup/go-api/internal/sync/matchflags"
)

// compteursEnLigne : ce que la passe en ligne ajoute a la restitution de fin de passe.
// ordreDecodage / ordreArchivage : les deux libelles de tri, chacun a cote de sa raison. Ils
// sont OPPOSES et c est voulu (cf. l en-tete de cmd_archive_films.go).
const (
	ordreDecodage  = "du plus recent au plus vieux (les films recuperables d abord)"
	ordreArchivage = "du plus VIEUX au plus recent (ceux a qui il reste le moins de temps)"
)

var compteursEnLigne = []string{
	killcollector.CompteurFilmsDepuisCache, killcollector.CompteurFilmsTelecharges,
	killcollector.CompteurFilmsArchives, killcollector.CompteurArchiveErreurs,
}

// passeDesFilmsEnLigne : le decodage avec repli reseau et archivage.
func passeDesFilmsEnLigne(ctx context.Context, cfg *config.AppConfig, db *sql.DB, o killsourceOptions) error {
	cacheRoot := resoudreCacheFilms(cfg, o.cacheDir)

	candidats, err := matchsSansPasseDeFilm(ctx, db, o)
	if err != nil {
		return err
	}
	fmt.Printf("films a traiter en ligne : %d (cache d archivage %s, joueur %s)\n",
		len(candidats), cacheRoot, o.gamertag)
	if len(candidats) == 0 {
		return nil
	}
	if o.dryRun {
		afficherPlanEnLigne(candidats, ordreDecodage)
		return nil
	}

	caps, err := capabilitesDuTitre(cfg, o.titleSlug)
	if err != nil {
		return err
	}
	// Les tokens sont resolus APRES le dry-run et APRES les capabilities : une passe qui
	// n aurait rien a faire, ou un titre sans `film.kill_source`, ne doit pas exiger une
	// authentification vivante pour dire qu elle n a rien a faire.
	tokens, err := haloTokensForPlayer(ctx, cfg.RepoRoot, o.gamertag)
	if err != nil {
		return fmt.Errorf("passe en ligne (%s): %w", o.gamertag, err)
	}
	// LE CACHE DOIT EXISTER AVANT D ETRE LU. `NewLocalFilmCache` rend nil quand
	// `film_manifests/` est absent, et ce nil vaut pour TOUT le process : sur une machine
	// neuve, la passe archiverait sans jamais relire, et la passe SUIVANTE repaierait le
	// reseau en entier. Creer les deux dossiers d abord supprime le piege.
	if err := filmcache.EnsureDirs(cacheRoot); err != nil {
		return err
	}
	source := killcollector.NewRemoteFilms(
		killcollector.NewLocalCacheFilms(haloclient.NewLocalFilmCache(cacheRoot)),
		go_sync.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, o.rps),
		cacheRoot,
	)

	collecteur := killcollector.NewKillSourceCollector(
		source, killcollector.NewSharedRoster(db), writerDeja(db), caps,
		0, // limite par match : le defaut du collecteur (45 min)
	)
	debut := time.Now()
	sum := collecteur.CollectMatches(ctx, candidats)
	fmt.Printf("films (en ligne) : %d ecrits (%d morts), %d absents/expires, %d sans kill-feed, "+
		"%d abandons sur delai, %d erreurs, %d sans capability — %s\n",
		sum.Written, sum.Deaths, sum.NoFilm, sum.NoKillFeed, sum.Timeouts, sum.Errors,
		sum.NotSupport, time.Since(debut).Round(time.Second))
	return nil
}

// matchsSansPasseDeFilm : les matchs du registre a decoder, DU PLUS RECENT AU PLUS VIEUX.
//
// # DEUX CORRECTIONS DU 2026-08-29, TOUTES DEUX MESUREES
//
//  1. L ORDRE ETAIT A L ENVERS. La version d origine prenait les plus vieux d abord, au nom
//     d une « course contre l expiration ». Mais un film deja expire ne se sauve pas, et la
//     mesure est sans appel : sur 999 candidats, 584 sont anterieurs a 2026 et 581 portent deja
//     le marqueur « film absent ». Un `--limit 20` partait donc sur des matchs de 2021 — un
//     operateur les voyait revenir en 404 et concluait a tort que le rattrapage etait mort.
//  2. LE MARQUEUR TERMINAL EXISTAIT DEJA et n etait pas lu. `MBitWeaponKillsNoFilm` est pose
//     par le pipeline quand 343 ne sert plus le film. L exclure retire 581 des 999 candidats,
//     tous irrecuperables. `--force` la leve, pour la passe qui veut quand meme retenter.
//
// `start_time_utc` par le COALESCE canonique (regle 8) — un `start_time` brut trierait faux.
func matchsSansPasseDeFilm(ctx context.Context, db *sql.DB, o killsourceOptions) ([]string, error) {
	dejaFaits := map[string]bool{}
	filtreNoFilm := int64(matchflags.MBitFilmAbsent)
	if o.force {
		filtreNoFilm = 0 // `--force` : on retente meme ce que le pipeline a declare perdu.
	} else {
		var err error
		if dejaFaits, err = matchsAJour(ctx, db); err != nil {
			return nil, err
		}
	}
	rows, err := db.QueryContext(ctx, `
		SELECT match_id
		FROM match_registry
		WHERE COALESCE(backfill_completed, 0) & ? = 0
		ORDER BY `+analysis.SQLStartTimeCanonical("")+` DESC, match_id`, filtreNoFilm)
	if err != nil {
		return nil, fmt.Errorf("registre par recence: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]string, 0, 128)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("registre par recence (scan): %w", err)
		}
		if dejaFaits[id] {
			continue
		}
		out = append(out, id)
		if o.limit > 0 && len(out) >= o.limit {
			break
		}
	}
	return out, rows.Err()
}

// afficherPlanEnLigne : la tete et la queue de la liste, sans cout par film — il n est
// connaissable qu apres un aller-retour reseau, et un dry-run ne doit en faire aucun.
//
// L elision se calcule AVANT la boucle. La version d origine testait `i >= tete && i < len-3`
// puis `i == tete` dans cet ordre : la ligne « ... » n etait donc JAMAIS imprimee des que la
// liste depassait 8 entrees (le seul cas ou elle sert), et affichait un compte NEGATIF pour
// 6 ou 7. Un operateur lisait un plan tronque comme s il etait complet.
// afficherPlanEnLigne prend le libelle de l ordre EN PARAMETRE, et ce n est pas cosmetique :
// les deux passes qui l utilisent trient a l INVERSE l une de l autre (decodage : les recents
// d abord ; archivage : les vieux d abord, cf. cmd_archive_films.go). Un libelle en dur
// affirmait le contraire de ce que la liste montrait — un plan qui ment sur son propre ordre
// est pire qu un plan sans libelle.
func afficherPlanEnLigne(ids []string, ordre string) {
	const tete, queue = 5, 3
	fmt.Printf("  ordre : %s\n", ordre)
	if len(ids) <= tete+queue {
		for _, id := range ids {
			fmt.Printf("  %s\n", id)
		}
		return
	}
	for _, id := range ids[:tete] {
		fmt.Printf("  %s\n", id)
	}
	fmt.Printf("  ... (%d autres)\n", len(ids)-tete-queue)
	for _, id := range ids[len(ids)-queue:] {
		fmt.Printf("  %s\n", id)
	}
}
