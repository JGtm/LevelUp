package main

// cmd_backfill_replay.go — sous-commande `levelup backfill-replay`.
//
// ELLE CONSTRUIT LES ARTEFACTS DE REJEU 2D de tous les films du cache local
// (data/cache/replays/{title}/{short8}.json), par la LIBRAIRIE internal/replaybuild.
//
// # UN FILM = UN PROCESSUS (depuis le 2026-08-20)
//
// Le parent PLANIFIE (enumeration, tri par cout, sauts, `--dry-run`) et ne decode RIEN :
// pour chaque film retenu il RE-EXECUTE SON PROPRE BINAIRE (`backfill-replay --one <id>`),
// sequentiellement, et traduit le code de sortie de l'enfant en categorie de recap.
//
// Cette commande bouclait auparavant sur tout le corpus dans un seul processus, et la note
// qui tient ici disait « jamais en executant un CLI ». Le 2026-08-20 cette boucle a SATURE
// LA MACHINE : quatre petits films cuits, puis six heures de spirale GC sur le cinquieme, et
// `runtime.preemptM: duplicatehandle failed; errno=1450` — le runtime n'obtenait plus un
// handle de thread de Windows. Les deux objections de l'ancienne note tombent, et il faut
// dire pourquoi plutot que de les laisser contredire le code :
//
//   - « conflit DuckDB mono-process » : les enfants sont STRICTEMENT SEQUENTIELS et chacun
//     RELACHE son handle de lecture AVANT de decoder. Il n'existe jamais deux lecteurs, et
//     jamais un lecteur pendant un decodage.
//   - « un exec perdrait le verrou process filmdec » : ce verrou
//     (filmdec.LockProcessDecode) protege des GLOBAUX DE PAQUET contre deux decodages
//     simultanes DANS UN MEME PROCESSUS. Deux processus ne partagent pas ces globaux : un
//     film par processus est une isolation STRICTEMENT PLUS FORTE que le verrou, pas sa
//     perte. Elle remet meme a zero, a chaque film, la table d'observation compWidthObs que
//     le paquet n'efface jamais.
//
// Le decodage passe donc toujours par la librairie, jamais par un autre binaire : le seul
// processus lance est CELUI-CI, sur un seul film.
//
// # ELLE NE PASSE PAS PAR LE REGLAGE « OU SE CONSTRUIT UN REJEU », ET C'EST VOULU
//
// Le reglage `replay_build_location` (local / worker / off) arbitre les chemins de SERVICE :
// le fil de l'eau post-sync et l'action admin. Cette commande est un outil d'OPERATEUR —
// celui qui la tape a deja decide ou il construit, sur SA machine, avec SES films en cache.
//
// # ELLE EST 100 % HORS LIGNE — NI RESEAU, NI TOKENS, NI CDN
//
// La source est le cache film sur disque (filmcache) ; la seule base touchee est le registre
// partage, en LECTURE COURTE au demarrage cote parent, puis une lecture par enfant pour SES
// seuls faits de match — relachee, elle aussi, avant tout decodage.
//
// # ELLE EST REPRENABLE, ET LA CLE EST replay.SchemaVersion
//
// Un artefact present qui porte la version de schema courante est saute ; toute version
// anterieure se lit « a re-cuire » (cf. document.go). `--force` re-cuit tout. L'ecriture
// d'un artefact est atomique : un enfant tue ne laisse jamais un artefact a moitie ecrit.
//
// # ELLE PASSE LES GROS FILMS EN DERNIER (meme regle que backfill-killsource)
//
// Le cout croit avec le nombre de chunks : trier par cout croissant produit la quasi-totalite
// de la valeur avant les films les plus chers.
//
// # UN FILM QUI ECHOUE N'EMPORTE QUE LUI
//
// « Carte hors catalogue » est un echec VOULU (cartes Forge sans bornes de dequantification),
// compte a part. Une MORT MEMOIRE (plafond depasse) et une MORT SUBITE (crash, tue par l'OS)
// sont comptees a part elles aussi — et dans les trois cas LA PASSE CONTINUE.
//
// Usage :
//
//	levelup backfill-replay --dry-run              # le plan de passe, aucune ecriture
//	levelup backfill-replay --limit 25             # pilote : les 25 films les moins chers
//	levelup backfill-replay                        # tout (~8 h, artefacts ~2 Go)
//	levelup backfill-replay --force                # re-cuit meme les artefacts a jour
//	levelup backfill-replay --only-existing        # re-cuit UNIQUEMENT les artefacts deja la
//	levelup backfill-replay --repair-impoverished  # re-cuit les artefacts appauvris reparables
//	levelup backfill-replay --mem-limit-gib 6      # remonte le plafond memoire des enfants
//
// `--one <matchID>` est la forme INTERNE : c'est ainsi que le parent appelle l'enfant. La
// taper a la main est licite (cuire un film precis), mais elle ne planifie rien et ne saute
// rien — elle cuit.
//
// # `--only-existing` EST LA PASSE D'APRES UN BUMP DE SCHEMA
//
// Une montee de `replay.SchemaVersion` perime d'un coup TOUS les artefacts : une passe
// ordinaire repartirait alors sur les 951 films du cache (des heures) la ou l'on veut
// seulement remettre a niveau ce qui est deja servi.
//
// # `--repair-impoverished` EST LA REMEDIATION DU CACHE APPAUVRI
//
// A NE PAS CONFONDRE AVEC `--only-existing`. Un artefact APPAUVRI (cuit sans les faits de match ->
// `scoreTimeline.players` vide) porte le BON numero de schema : `--only-existing` le SAUTE comme
// « deja a jour ». Ce mode-ci le RE-CUIT — mais SEULEMENT si la base porte des lignes de match, ce
// que l'enfant transportera au constructeur. Il saute l'artefact deja riche, la vacuite LEGITIME
// (base sans joueur : re-cuire ne donnerait rien) et l'artefact hors schema courant. Le predicat et
// sa mecanique vivent dans cmd_backfill_replay_repair.go ; c'est une SELECTION, pas un second chemin
// de cuisson. Passe A LANCER AVANT la premiere activation prod de l'ouvrier (dette
// PLAN_OUVRIER_DISTANT.md §5ter). S'EXCLUT de `--force`.
//
// Le cache de films se resout par `--cache`, sinon `LEVELUP_LEGACY_FILM_CACHE_DIR`, sinon
// `<repo>/data/cache` (meme regle que backfill-killsource). La racine du depot se resout par
// `LEVELUP_REPO_ROOT` — le parent l'IMPOSE a ses enfants pour que toute la passe ecrive dans
// la meme arborescence.

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/replaybuild"
)

// replayCandidat : un film du cache joint a son match du registre.
type replayCandidat struct {
	matchID  string
	mapNames []string // candidats du plus fiable au moins fiable (asset EN, puis brut)
	chunks   int
}

// replayBackfillOptions : les reglages de la passe.
type replayBackfillOptions struct {
	titleSlug string
	cacheDir  string
	limit     int
	force     bool
	dryRun    bool
	// onlyExisting borne la passe aux matchs qui portent DEJA un artefact sur disque.
	onlyExisting bool
	// repairImpoverished re-cuit les artefacts a jour MAIS sans compteurs de joueur alors que la
	// base a des lignes (remediation du cache appauvri, cf. cmd_backfill_replay_repair.go).
	repairImpoverished bool
	// one : forme INTERNE. Non vide = ce processus est un ENFANT et cuit ce seul film.
	one string
	// mapNames : les identites de carte candidates, passees par le parent a l'enfant.
	mapNames listeDrapeau
	// memLimitGiB : le plafond memoire de CHAQUE enfant (0 = desarme).
	memLimitGiB int
}

func runBackfillReplay(cfg *config.AppConfig, args []string) error {
	o, err := parserOptionsReplay(args)
	if err != nil {
		return err
	}
	// `--repair-impoverished` EST une selection ciblee (les seuls appauvris-reparables) ; `--force`
	// re-cuirait TOUT, ce que ce mode existe precisement pour eviter. Les deux s'excluent.
	if o.repairImpoverished && o.force {
		return fmt.Errorf("--repair-impoverished et --force s'excluent : le mode reparation est deja une selection ciblee")
	}
	ctx := context.Background()
	cacheRoot := resoudreCacheFilms(cfg, o.cacheDir)

	// FORME ENFANT : un film, puis ce processus meurt. Le code de sortie EST le canal de
	// retour vers le parent (cf. backfill_child.go) — d'ou l'`os.Exit` plutot qu'un `return`.
	if strings.TrimSpace(o.one) != "" {
		os.Exit(runBackfillReplayUn(cfg, o, cacheRoot))
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	candidats, horsRegistre, err := replaysACuire(ctx, pr, cacheRoot, o)
	if err != nil {
		return err
	}
	report := replayBackfillReport{horsRegistre: horsRegistre}
	var aFaire []replayCandidat
	if o.repairImpoverished {
		if aFaire, err = passeReparation(ctx, cfg, candidats, o, &report); err != nil {
			return err
		}
		fmt.Printf("artefacts appauvris a reparer : %d (cache %s, %d deja complets, %d vacuites legitimes, %d hors schema courant, %d hors registre, %d sans artefact)\n",
			len(aFaire), cacheRoot, report.dejaComplets, report.vacuitesLegitimes, report.horsSchemaCourant, report.horsRegistre, report.sansArtefact)
	} else {
		aFaire = filtrerEtTrierReplay(candidats, pr, o, &report)
		fmt.Printf("films a construire : %d (cache %s, %d deja a jour, %d hors registre, %d sans artefact)\n",
			len(aFaire), cacheRoot, report.dejaAJour, report.horsRegistre, report.sansArtefact)
	}
	if o.dryRun {
		afficherPlanReplay(aFaire)
		return nil
	}
	if len(aFaire) == 0 {
		afficherRapportReplay(report)
		return nil
	}
	return executerPasseReplay(ctx, cfg.RepoRoot, o, cacheRoot, aFaire, &report)
}

// parserOptionsReplay : les drapeaux de la commande.
func parserOptionsReplay(args []string) (replayBackfillOptions, error) {
	fs := flag.NewFlagSet("backfill-replay", flag.ExitOnError)
	o := replayBackfillOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.StringVar(&o.cacheDir, "cache", "", "racine du cache de films (defaut : LEVELUP_LEGACY_FILM_CACHE_DIR puis <repo>/data/cache)")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre de films construits (0 = tous) — les moins chers d abord")
	fs.BoolVar(&o.force, "force", false, "re-cuire meme les artefacts deja a la version de schema courante")
	fs.BoolVar(&o.dryRun, "dry-run", false, "afficher le plan de passe sans rien construire")
	fs.BoolVar(&o.onlyExisting, "only-existing", false,
		"ne traiter que les matchs qui ont deja un artefact sur disque (passe d apres un bump de schema)")
	fs.BoolVar(&o.repairImpoverished, "repair-impoverished", false,
		"re-cuire les artefacts a jour MAIS sans compteurs de joueur alors que la base a des lignes "+
			"(remediation du cache appauvri — passe A LANCER AVANT activation ouvrier)")
	fs.StringVar(&o.one, "one", "", "INTERNE : cuire CE seul match dans ce processus (forme appelee par le parent)")
	fs.Var(&o.mapNames, "map-name", "INTERNE : identite de carte candidate (repetable, du plus fiable au moins fiable)")
	fs.IntVar(&o.memLimitGiB, "mem-limit-gib", plafondMemoireDefautGiB,
		"plafond memoire de chaque enfant, en GiB (0 = desarme)")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	return o, nil
}

// filtrerEtTrierReplay ecarte ce qui n'est pas a cuire, puis trie par cout croissant.
//
// LE FILTRE « DEJA A JOUR » SE FAIT AVANT LE TRI ET LE `--limit` : un pilote `--limit 25`
// doit livrer 25 constructions REELLES, pas 25 sauts.
func filtrerEtTrierReplay(
	candidats []replayCandidat, pr *titlePkg.PathResolver, o replayBackfillOptions,
	report *replayBackfillReport,
) []replayCandidat {
	aFaire := make([]replayCandidat, 0, len(candidats))
	for _, c := range candidats {
		path := pr.ReplayArtifactPath(o.titleSlug, c.matchID)
		if o.onlyExisting {
			if _, err := os.Stat(path); err != nil {
				report.sansArtefact++
				continue
			}
		}
		if !o.force && replaybuild.ArtifactUpToDate(path) {
			report.dejaAJour++
			continue
		}
		aFaire = append(aFaire, c)
	}
	return trierEtBornerReplay(aFaire, o.limit)
}

// replaysACuire joint le cache film au registre : pour chaque film du cache, le match et ses
// identites de carte candidates. LA LECTURE DE BASE EST COURTE : les deux handles (shared RO,
// metadata RO) sont relaches au retour, AVANT tout lancement d'enfant.
func replaysACuire(
	ctx context.Context, pr *titlePkg.PathResolver, cacheRoot string, o replayBackfillOptions,
) ([]replayCandidat, int, error) {
	shorts, err := filmcache.ListShortIDs(cacheRoot)
	if err != nil {
		return nil, 0, err
	}
	if len(shorts) == 0 {
		return nil, 0, fmt.Errorf("aucun film dans le cache %s — cette passe est HORS LIGNE, elle n'a pas d'autre source", cacheRoot)
	}

	registre, err := registreParShort(ctx, pr, o.titleSlug)
	if err != nil {
		return nil, 0, err
	}

	horsRegistre := 0
	out := make([]replayCandidat, 0, len(shorts))
	for _, short := range shorts {
		entry, ok := registre[short]
		if !ok {
			horsRegistre++
			continue
		}
		n, ok := compterChunks(cacheRoot, short)
		if !ok {
			continue // manifeste illisible/vide : rien a decoder
		}
		out = append(out, replayCandidat{matchID: entry.matchID, mapNames: entry.mapNames, chunks: n})
	}
	return out, horsRegistre, nil
}

// registreEntry : un match du registre, avec ses identites de carte candidates.
type registreEntry struct {
	matchID  string
	mapNames []string
}

// registreParShort lit match_registry (RO) et metadata.asset_translations (RO, best-effort)
// et indexe par forme courte de match_id.
//
// L'ordre des candidats est celui de ReplayMapRepo : nom d'asset EN (resolu, fiable meme
// quand map_name porte un UUID brut) PUIS map_name brut. metadata peut etre tenue RW par un
// serveur qui tourne : son ouverture en echec DEGRADE (candidat brut seul), jamais ne bloque.
func registreParShort(ctx context.Context, pr *titlePkg.PathResolver, titleSlug string) (map[string]registreEntry, error) {
	sharedPath := pr.SharedDBPath(titleSlug)
	db, release, err := duckdb.OpenReadForQuery(sharedPath)
	if err != nil {
		return nil, fmt.Errorf("open shared RO (%s): %w (serveur en ecriture ? reessayer)", sharedPath, err)
	}
	defer release()

	enParMapID := nomsAssetsEN(ctx, pr, titleSlug)

	rows, err := db.QueryContext(ctx, `SELECT match_id, map_name, map_id FROM match_registry ORDER BY match_id`)
	if err != nil {
		return nil, fmt.Errorf("registre des matchs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]registreEntry{}
	for rows.Next() {
		var id string
		var rawName, mapID sql.NullString
		if err := rows.Scan(&id, &rawName, &mapID); err != nil {
			return nil, fmt.Errorf("registre des matchs (scan): %w", err)
		}
		names := candidatsCarte(enParMapID[strings.TrimSpace(mapID.String)], rawName.String)
		out[titlePkg.FilmShortMatchID(id)] = registreEntry{matchID: id, mapNames: names}
	}
	return out, rows.Err()
}

// candidatsCarte ordonne les identites de carte du plus fiable au moins fiable : nom d'asset
// EN (resolu, fiable meme quand map_name porte un UUID brut) PUIS map_name brut. C'est l'ordre
// que ResolveMapEntry essaie.
//
// UN SEUL point de verite pour cet ordre — le PARENT de masse (registreParShort) et l'ENFANT
// tape a la main (mapNamesForOne) doivent produire EXACTEMENT les memes candidats, sans quoi le
// meme film se resoudrait differemment selon la forme d'appel.
func candidatsCarte(enName, rawName string) []string {
	var names []string
	if en := strings.TrimSpace(enName); en != "" {
		names = append(names, en)
	}
	if raw := strings.TrimSpace(rawName); raw != "" {
		names = append(names, raw)
	}
	return names
}

// mapNamesForOne resout les identites de carte candidates d'UN match depuis le registre, pour
// la forme `--one` TAPEE A LA MAIN.
//
// POURQUOI ELLE EXISTE. La resolution de carte vit dans le PARENT (registreParShort), qui la
// passe a l'enfant via `--map-name`. Un operateur qui tape `backfill-replay --one <id>` — usage
// documente comme licite — n'a pas ce drapeau : sans elle, `o.mapNames` est vide et le film
// echoue « carte hors catalogue ([]) » ALORS QUE sa carte est au catalogue. Elle rend a l'enfant
// la meme auto-suffisance que pour ses faits (chargerFaitsUnMatch).
//
// UNE ouverture RO relachee AVANT tout decodage, comme chargerFaitsUnMatch — jamais deux
// lecteurs, jamais un lecteur pendant un decodage. Base indisponible (serveur en ecriture) ou
// match hors registre = candidats vides, JOURNALISE : l'echec de carte reste alors possible,
// mais il n'est plus DU a l'absence de resolution.
//
// La masse (parent -> enfant) passe toujours `--map-name` : cette lecture ne se declenche QUE
// pour le `--one` a la main, jamais dans une passe de masse (aucun surcout de corpus).
func mapNamesForOne(ctx context.Context, pr *titlePkg.PathResolver, titleSlug, matchID string) []string {
	sharedPath := pr.SharedDBPath(titleSlug)
	db, release, err := duckdb.OpenReadForQuery(sharedPath)
	if err != nil {
		slog.WarnContext(ctx, "backfill-replay (enfant): registre indisponible — identite de carte "+
			"non resolue (passer --map-name)", "err", err, "match_id", matchID)
		return nil
	}
	defer release()
	var rawName, mapID sql.NullString
	row := db.QueryRowContext(ctx,
		`SELECT map_name, map_id FROM match_registry WHERE match_id = ?`, matchID)
	if err := row.Scan(&rawName, &mapID); err != nil {
		slog.WarnContext(ctx, "backfill-replay (enfant): match absent du registre — identite de carte "+
			"non resolue (passer --map-name)", "err", err, "match_id", matchID)
		return nil
	}
	en := nomsAssetsEN(ctx, pr, titleSlug)[strings.TrimSpace(mapID.String)]
	return candidatsCarte(en, rawName.String)
}

// nomsAssetsEN charge la table map_id -> nom EN depuis metadata.asset_translations.
// BEST-EFFORT : metadata absente ou tenue RW par un autre process → map vide + message, la
// passe continue sur les map_name bruts du registre.
func nomsAssetsEN(ctx context.Context, pr *titlePkg.PathResolver, titleSlug string) map[string]string {
	metaPath := pr.MetadataDBPath(titleSlug)
	db, release, err := duckdb.OpenReadForQuery(metaPath)
	if err != nil {
		fmt.Printf("metadata illisible (%v) — resolution des noms EN degradee (map_name brut seul)\n", err)
		return nil
	}
	defer release()
	rows, err := db.QueryContext(ctx,
		`SELECT asset_id, name FROM asset_translations WHERE asset_type = 'map' AND lang = 'en-US'`)
	if err != nil {
		fmt.Printf("asset_translations illisible (%v) — resolution des noms EN degradee\n", err)
		return nil
	}
	defer func() { _ = rows.Close() }()
	out := map[string]string{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		out[strings.TrimSpace(id)] = strings.TrimSpace(name)
	}
	return out
}
