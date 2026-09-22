package main

// cmd_backfill_killsource.go — sous-commande `levelup backfill-killsource`.
//
// ELLE REMPLIT `shared.match_kill_events` ET `shared.match_weapon_shots`, par les DEUX
// producteurs. Depuis l inversion de preseance (2026-08-03), AUCUN DES DEUX NE REFUSE PLUS RIEN :
// la base est toujours le CREDIT, le film ENRICHIT.
//
//	1. LE FILM      — decodage hors ligne des films en cache. Sa sortie est FUSIONNEE sur la base
//	                  credit du match (couples de `killer_victim_pairs`) avant d etre ecrite :
//	                  elle apporte la source du degat, l assistant nomme et les parts, plus la
//	                  ventilation des tirs par arme.
//	2. LE CREDIT    — depuis `highlight_events`, EN BASE, sans film. Il apporte les matchs dont
//	                  le film a EXPIRE cote serveur (29,3 % du corpus), et sur les autres il
//	                  RECOMPOSE : sa base + l enrichissement de la passe de film deja en base.
//
// L ORDRE RESTE FILM PUIS CREDIT, et il n est plus une question de preseance mais de cout : la
// passe credit est une transformation SQL -> SQL de quelques secondes, la repasser apres le
// decodage ne coute rien et rattrape les matchs dont le film vient d arriver.
//
// # ELLE EST 100 % HORS LIGNE — NI RESEAU, NI TOKENS, NI CDN
//
// Mesure du 2026-08-01 : les 949 films utilisables du cache portent sur disque leur en-tete
// (type 1), toutes leurs replications (type 2) ET leur kill-feed (type 3). La source de chunks
// est donc [killcollector.LocalCacheFilms], qui n a aucun client HTTP derriere elle : il est
// IMPOSSIBLE que cette commande parte sur le reseau, et donc impossible qu elle echoue sur une
// authentification expiree au bout de trois heures de decodage.
//
// # ELLE EST REPRENABLE, ET LA CLE EST `decoder_rev`
//
// Un match dont TOUTES les passes courantes (vues `_latest`) portent leur revision de decodeur
// courante est saute. Interrompre la passe et la relancer reprend donc ou elle en etait, sans
// re-decoder ce qui est deja fait. `--force` redecode tout — c est ce qu il faut le jour ou une
// revision change.
//
// DEUX REVISIONS, DEUX UNITES DE FRAICHEUR (lot 7C) : `facts.Rev` pour le journal des
// morts, `IsolationDecoderRev` pour les faits d isolement (`match_lives`,
// `match_death_context`). Elles evoluent separement — un changement de la regle de visibilite
// doit refaire les faits d isolement SANS refaire le journal, qui n a pas bouge. La commande
// reprend donc un match dont le journal est a jour mais dont les faits d isolement manquent ou
// datent : c est ce qui permet au corpus DEJA collecte de recevoir les deux nouvelles tables.
//
// # ELLE PASSE LES GROS FILMS EN DERNIER, ET CE N EST PAS UNE PREFERENCE
//
// Le cout par film reste croissant avec sa taille (0,13 s/chunk a 8 chunks, 0,68 s/chunk a 69
// apres le correctif du 2026-08-01). Trier par taille croissante fait que la passe produit la
// quasi-totalite de sa valeur AVANT de s attaquer aux quelques films qui coutent le plus : une
// interruption au bout d une heure laisse alors un resultat presque complet, pas un tiers de
// corpus.
//
// # UN SEUL PROCESSUS, UN SEUL HANDLE RW, N GOROUTINES DE DECODAGE
//
// (Examen du 2026-08-24, AMENDE le 2026-09-22 par le lot 5.24.2 — le raisonnement d origine est
// intact, il gagne une troisieme ligne.)
//
// `backfill-replay` a ete decoupee en parent/enfant (un film = un processus) apres avoir sature
// la machine le 2026-08-20. Cette passe-ci a ete examinee pour le meme traitement et NE L A PAS
// RECU. Ce n est pas un oubli, et voici de quoi refaire le raisonnement :
//
//	1. AUCUNE RETENTION INTER-MATCHS. `KillSourceCollector` ne porte que des poignees sans
//	   etat (client, roster, lease, capabilities, delai) et `CollectMatches` n accumule que
//	   des compteurs plus le `[]string` des identifiants. Son etat apres 950 matchs est celui
//	   qu il avait apres un seul. `replaybuild` n en avait pas davantage — mais SON APPELANT
//	   chargeait les faits de TOUT le lot dans une map vivante toute la passe (supprime).
//	2. LE PIC EST UNE FONCTION DES OCTETS DU FILM, BORNEE ET MESUREE. Le pic vaut le film brut
//	   (garde vivant, `FilmOf` l aliase) plus le film decompresse (`source.Load`, un tampon
//	   par chunk). Mesure du cache au 2026-08-24, 951 films : le PLUS GROS
//	   du corpus est `1c4c63c2` a 88 Mio sur disque (69 chunks), la moyenne est a 24 Mio. Le
//	   pire cas tient donc largement sous le gibioctet.
//	3. LE REJEU 2D, LUI, N A AUCUN RAPPORT AVEC LA TAILLE DU FILM. `51101d1d` pese 9,1 Mio sur
//	   disque (13 chunks) et a fait monter `backfill-replay` a 7,36 Gio en 2,6 s — pres de 800
//	   fois ses octets — parce que son pic est fait d une quinzaine de tranches a l echelle du
//	   RECORD decode, toutes vivantes ensemble. C est cette amplification-la qui exigeait un
//	   processus par film ; elle n existe pas ici.
//	4. LE DECOUPAGE COUTERAIT PLUS CHER QU IL NE RAPPORTERAIT. Cette commande tient un handle
//	   RW sur le shared pendant TOUTE la passe. Un enfant par match devrait rouvrir cette base
//	   en ecriture et rejouer les migrations a chaque film — 950 cycles ouverture/checkpoint —
//	   et multiplierait d autant les occasions de mourir au milieu d une transaction, ce que
//	   la doctrine anti-corruption (ADR 0013/0019/0026) existe precisement pour eviter.
//
// COROLLAIRE : aucune sentinelle memoire ici non plus. Celle de l enfant du rejeu fait
// `os.Exit` — un procede acceptable dans un processus qui ne tient AUCUNE base en ecriture, et
// interdit dans celui-ci. Poser un plafond souple a 3 Gio sur une passe qui plafonne sous le
// gibioctet ne serait qu un reglage mort.
//
// A REEXAMINER SI : un film du cache depasse ~500 Mio sur disque, ou si le decodage killsource
// se met a produire des structures a l echelle du record (comme le fait le rejeu 2D).
//
// ─── CE QUE LE LOT 5.24.2 AJOUTE, ET POURQUOI IL NE CONTREDIT RIEN DE CE QUI PRECEDE ────────
//
// TOUJOURS UN PROCESSUS, TOUJOURS UN HANDLE RW, TOUJOURS ZERO CYCLE D OUVERTURE PAR FILM : les
// quatre points ci-dessus restent vrais mot pour mot. Ce qui change est le nombre de goroutines
// qui DECODENT : `--workers` (defaut 3).
//
//	POURQUOI      la decomposition du cout (5.24.1, `backfill_cout_integration_test.go`) mesure
//	              93 a 99 % du temps d un film en CPU HORS BASE. Une passe qui ne sature qu un
//	              coeur laisse les autres vides pendant quatre heures.
//	POURQUOI C EST POSSIBLE MAINTENANT   le decodeur ne porte plus d etat de paquet : la cloture
//	              M3 d ADR 0034 a DEPENSE le profil (il voyage en argument) et le dernier reglage
//	              global a disparu au lot E.2 du 2026-09-05. L avertissement contraire qui vivait
//	              dans `collector.go` etait vrai a son epoque et ne l est plus.
//	CE QUI TIENT ADR 0013   la PORTE DE LA BASE (`killcollector.PorteDeLaBase`) : un jeton
//	              unique, pris par le lease RW, par la resolution d identites et par la resolution
//	              de carte. A tout instant AU PLUS UN goroutine parle a la base.
//	LA PREUVE     `TestOuvriers_MemesLignesQuUnSeulOuvrier` : les memes films ecrits par 1 puis
//	              par N ouvriers rendent des vues `_latest` IDENTIQUES ligne a ligne.
//	LE PLAFOND    le pic mesure du pire film du corpus (`1c4c63c2`) est de 422 Mio ; la commande
//	              REFUSE au demarrage un `--workers` dont le produit depasserait 4 Gio.
//
// Usage (SERVEUR ARRETE — `OpenReadWrite` echoue si le lock est tenu) :
//
//	levelup backfill-killsource --dry-run              # ce qui SERAIT fait, aucune ecriture
//	levelup backfill-killsource --limit 20             # les 20 films les moins chers
//	levelup backfill-killsource                        # tout : films puis credit
//	levelup backfill-killsource --workers 1            # la boucle en serie d avant le lot 5.24
//	levelup backfill-killsource --credit-only          # la passe SQL -> SQL seule (secondes)
//	levelup backfill-killsource --force                # redecode meme ce qui est a jour
//
// Le cache de films se resout par `--cache`, sinon `LEVELUP_LEGACY_FILM_CACHE_DIR`, sinon
// `<repo>/data/cache`.

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/sync/killcollector"
)

// filmCandidat : un film du cache, avec le nombre de chunks qui donne son cout.
type filmCandidat struct {
	matchID string
	chunks  int
}

// killsourceOptions : les reglages de la passe.
type killsourceOptions struct {
	titleSlug  string
	cacheDir   string
	limit      int
	force      bool
	dryRun     bool
	filmsOnly  bool
	creditOnly bool
	// online : va CHERCHER le film quand il n est pas en cache, et l archive au passage.
	// Sans ce drapeau la passe reste 100 % hors ligne — c est le defaut, et il est
	// deliberement conserve : une passe qui part sur le reseau sans qu on l ait demande
	// exigerait une authentification vivante la ou l ancienne n en avait pas besoin.
	online   bool
	gamertag string
	rps      int
	// workers : le nombre de films decodes EN PARALLELE (lot 5.24.2). 1 = la boucle d avant.
	workers int
	// status : LIT le fichier d etat et sort. N ouvre aucune base (lot 5.24.3).
	status bool
	// workersExplicite : `--workers` a ete ecrit sur la ligne de commande (par opposition au
	// defaut). Sert UNIQUEMENT a refuser `--online --workers N` sans refuser `--online`.
	workersExplicite bool
}

func runBackfillKillSource(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-killsource", flag.ExitOnError)
	o := killsourceOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.StringVar(&o.cacheDir, "cache", "", "racine du cache de films (defaut : LEVELUP_LEGACY_FILM_CACHE_DIR puis <repo>/data/cache)")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre de films decodes (0 = tous) — les moins chers d abord")
	fs.BoolVar(&o.force, "force", false, "redecoder meme les matchs deja a jour pour la revision de decodeur courante")
	fs.BoolVar(&o.dryRun, "dry-run", false, "afficher le plan de passe sans rien ecrire")
	fs.BoolVar(&o.filmsOnly, "films-only", false, "ne jouer que la passe de decodage des films")
	fs.BoolVar(&o.creditOnly, "credit-only", false, "ne jouer que la passe credit-seul (SQL -> SQL)")
	fs.BoolVar(&o.online, "online", false, "telecharger les films absents du cache (et les y archiver) au lieu de s en tenir au cache")
	fs.StringVar(&o.gamertag, "gamertag", "", "joueur dont les films sont traités, les plus récents d abord (obligatoire avec --online) ; les jetons viennent du pool")
	fs.IntVar(&o.rps, "rps", 4, "debit maximal des requetes Halo de la passe --online")
	fs.BoolVar(&o.status, "status", false,
		"LIRE le fichier d etat de la derniere passe et l afficher, une fois, SANS ouvrir "+
			"aucune base — c est ce qu on tape dans un second terminal pendant que la passe tourne")
	fs.IntVar(&o.workers, "workers", killcollector.OuvriersParDefaut, fmt.Sprintf(
		"films decodes EN PARALLELE (1 = la boucle en serie). Le decodage vaut 93 a 99 %% du "+
			"temps d un film (mesure 5.24.1) et il ne porte plus aucun etat de paquet : N "+
			"ouvriers decodent, UN SEUL touche la base. Pic mesure du pire film du corpus : "+
			"%d Mio — le plafond de la passe est de %d Mio, soit %d ouvriers au maximum",
		killcollector.PicMemoireParFilm>>20, killcollector.PlafondMemoireDeLaPasse>>20,
		killcollector.OuvriersMaximum()))
	if err := fs.Parse(args); err != nil {
		return err
	}
	// `--workers` A-T-IL ETE DEMANDE, ou est-ce le defaut ? La question n a qu un usage —
	// refuser `--online --workers N` sans refuser `--online` tout court (cf. validerLesOptions).
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "workers" {
			o.workersExplicite = true
		}
	})
	// LE CONTEXTE D ARRET (lot 5.24.4) : son annulation par SIGINT/SIGTERM arrete la
	// DISTRIBUTION des films ; ceux qui sont en vol vont au bout, ecriture comprise. Le contexte
	// de TRAVAIL en est derive par `context.WithoutCancel` la ou il faut aller au bout.
	ctx, arreterLeReleveur := contexteDArret()
	defer arreterLeReleveur()
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	// `--status` SORT AVANT TOUT LE RESTE, et avant meme la validation des autres drapeaux :
	// c est une LECTURE de fichier, elle n a rien a valider et surtout rien a ouvrir. La passe
	// qu on interroge tient le shared en ecriture (ADR 0013) — une sous-commande d etat qui
	// ouvrirait la base echouerait exactement quand on en a besoin.
	if o.status {
		return afficherEtatDeLaPasse(pr.BackfillKillSourceStatePath(o.titleSlug))
	}
	if err := validerLesOptions(o); err != nil {
		return err
	}

	sharedPath := pr.SharedDBPath(o.titleSlug)
	if _, err := os.Stat(sharedPath); err != nil {
		return fmt.Errorf("shared_matches introuvable (%s): %w", sharedPath, err)
	}
	handle, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared RW (%s): %w (serveur arrete ?)", sharedPath, err)
	}
	defer handle.Close()
	db := handle.SQLDb()

	// LES DEUX TABLES DOIVENT EXISTER AVANT D ECRIRE. Elles sont creees par les migrations du
	// shared — que le serveur joue a son demarrage, mais cette commande tourne SERVEUR ARRETE.
	// Sans cela la premiere passe echouerait sur « table inconnue » apres avoir decode un film,
	// c est-a-dire au pire moment. `RunForDB` est idempotent (`schema_migrations`).
	if err := migrerSchemaPartage(db, o.titleSlug); err != nil {
		return err
	}

	suivi, err := jouerLesDeuxPasses(ctx, cfg, db, o, pr.BackfillKillSourceStatePath(o.titleSlug))
	if err != nil {
		return err
	}
	cause := causeDArret(ctx)
	if suivi != nil {
		suivi.Terminee(cause)
	}
	if !o.dryRun {
		afficherSante(o.online)
	}
	if cause != "" {
		return &interruptionDeLaPasse{cause: cause}
	}
	return nil
}

// jouerLesDeuxPasses : LES FILMS PUIS LE CREDIT, dans cet ordre, sous UN SEUL suivi.
//
// Extraite de `runBackfillKillSource` au tour de revue du 2026-09-22 : la fonction depassait le
// plafond de complexite du depot, et surtout cet enchainement etait la piece la moins couverte
// du lot alors que c est lui qui porte le contrat de l arret (le constat P1 de la revue vivait
// ICI). La coupure suit une frontiere nette — LA-BAS ce qui ouvre, valide et ferme, ICI ce qui
// se joue entre les deux.
//
// L ORDRE N EST PAS INTERCHANGEABLE (cf. l en-tete du fichier) : la passe credit repassee APRES
// le decodage ne coute rien et rattrape les matchs dont le film vient d arriver.
func jouerLesDeuxPasses(
	ctx context.Context, cfg *config.AppConfig, db *sql.DB, o killsourceOptions, cheminEtat string,
) (*suiviDeLaPasse, error) {
	var suivi *suiviDeLaPasse
	var err error
	if !o.creditOnly {
		passe := passeDesFilms
		if o.online {
			passe = passeDesFilmsEnLigne
		}
		if suivi, err = passe(ctx, cfg, db, o, cheminEtat); err != nil {
			return nil, err
		}
	}
	if o.filmsOnly {
		return suivi, nil
	}
	// `--credit-only` A DROIT A SON ETAT, ET C EST LA PASSE QUI EN A LE PLUS BESOIN (constat de
	// revue, 2026-09-22) : celle du 2026-09-21 a tourne PLUS DE 22 HEURES en silence, et c est
	// elle que l en-tete de `cmd_backfill_killsource_etat.go` cite comme la douleur a corriger.
	// Sans ce bloc, `--credit-only` n ecrivait aucun fichier et `--status` servait l etat d une
	// passe PRECEDENTE pendant toute sa duree.
	if suivi == nil && !o.dryRun {
		suivi = suiviDuCreditSeul(cheminEtat, o)
	}
	if err := passeDuCredit(ctx, db, o, suivi); err != nil {
		return suivi, err
	}
	return suivi, nil
}

// migrerSchemaPartage : joue les migrations du shared sur le handle deja ouvert.
//
// Le provider de steps title-owned est cable comme partout ailleurs dans `cmd/`
// (`apply_shared_migrations`, `h5-backfill`, ...) : les migrations propres a un titre ne sont
// pas dans le jeu canonique, et sans ce cablage la commande creerait les tables communes en
// manquant SILENCIEUSEMENT celles du titre.
func migrerSchemaPartage(db *sql.DB, slug string) error {
	_ = migration.All()
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		return fmt.Errorf("migrations shared (%s): %w", slug, err)
	}
	return nil
}

// passeDesFilms : le decodage hors ligne, du film le moins cher au plus cher.
//
// Elle rend le SUIVI de la passe (lot 5.24.3) pour que la passe credit qui suit ecrive dans le
// MEME fichier d etat : une commande, un etat. nil quand il n y a rien eu a suivre.
func passeDesFilms(
	ctx context.Context, cfg *config.AppConfig, db *sql.DB, o killsourceOptions, cheminEtat string,
) (*suiviDeLaPasse, error) {
	cacheRoot := resoudreCacheFilms(cfg, o.cacheDir)
	cache := haloclient.NewLocalFilmCache(cacheRoot)
	if cache == nil {
		return nil, fmt.Errorf("cache de films introuvable sous %s — cette passe est HORS LIGNE, "+
			"elle n a pas d autre source", cacheRoot)
	}

	// LE CONTEXTE DE TRAVAIL NE S ANNULE PAS AVEC LE SIGNAL. `ctx` est le contexte d ARRET :
	// l annuler doit arreter la DISTRIBUTION, pas couper un decodage en deux ni une ecriture au
	// milieu. `WithoutCancel` garde les valeurs du contexte et lui retire son annulation ; c est
	// `AvecArretDoux` qui porte l arret, et il l applique ENTRE deux films.
	ctxTravail := context.WithoutCancel(ctx)

	candidats, bilan, err := filmsACollecter(ctxTravail, db, cacheRoot, o)
	if err != nil {
		return nil, err
	}
	fmt.Printf("films a decoder : %d (cache %s)\n", len(candidats), cacheRoot)
	fmt.Println(bilanInitial(candidats, bilan.TotalRegistre, bilan.DejaAJour, o.workers))
	if len(candidats) == 0 {
		return nil, nil
	}
	if o.dryRun {
		afficherPlan(candidats)
		return nil, nil
	}

	caps, err := capabilitesDuTitre(cfg, o.titleSlug)
	if err != nil {
		return nil, err
	}
	// LA PORTE DE LA BASE (lot 5.24.2) : le jeton unique qui donne le droit de toucher le shared.
	// Elle enveloppe le lease RW, la resolution d identites et la resolution de carte — les trois
	// seuls chemins par lesquels cette passe parle a la base. A tout instant, au plus un ouvrier
	// y parle : ADR 0013 est tenue par une piece, plus par la forme de la boucle.
	porte := killcollector.NouvellePorteDeLaBase()
	// CAPTURE DES POSITIONS (G.2bis), BEST-EFFORT : catalogue de bornes illisible ou metadata
	// indisponible degrade en « positions desactivees » (le collecteur continue sans elles, la
	// passe des morts/tirs n en depend pas). ACTIVEE PAR DEFAUT ici, PAS derriere un flag CLI —
	// c est la seule commande de backfill de ce producteur, et une feature OFF « pour plus tard »
	// est l anti-pattern que CLAUDE.md interdit (regle 11) : la capture est prete, elle capture.
	capture, cleanupPositions := positionCaptureDeps(cfg, o.titleSlug, db, porte)
	defer cleanupPositions()

	// LE SUIVI (lot 5.24.3) : il ecrit le fichier d etat APRES CHAQUE FILM et journalise une
	// ligne de progression tous les 25 films OU toutes les 60 s. Il ne decide rien — ni ce qui
	// est decode, ni ce qui est ecrit.
	suivi := nouveauSuivi(cheminEtat, o.titleSlug, candidats, bilan, o)

	collecteur := killcollector.NewKillSourceCollector(
		killcollector.NewLocalCacheFilms(cache),
		// L ANNUAIRE DE PASSE (lot 5.24.2) : `v_gamertag_lookup` lue UNE FOIS, pas par match.
		// C est la seule lecture repetee que la decomposition 5.24.1 ait trouvee — et depuis que
		// la passe a des ouvriers, elle serait SERIALISEE derriere la porte, donc un plafond.
		porte.GarderLeRoster(killcollector.NewSharedRoster(db).AvecAnnuaireDePasse()),
		porte.GarderLeWriter(writerDeja(db)),
		caps,
		0, // limite par match : le defaut du collecteur (45 min)
	).AvecCapture(capture).AvecObservateur(suivi).AvecArretDoux(ctx)

	ids := make([]string, 0, len(candidats))
	for _, c := range candidats {
		ids = append(ids, c.matchID)
	}
	debut := time.Now()
	sum := collecteur.CollectMatchesOuvriers(ctxTravail, ids, o.workers)
	fmt.Printf("films : %d ecrits (%d morts), %d absents, %d sans kill-feed, "+
		"%d abandons sur delai, %d erreurs, %d sans capability — %s\n",
		sum.Written, sum.Deaths, sum.NoFilm, sum.NoKillFeed, sum.Timeouts, sum.Errors,
		sum.NotSupport, time.Since(debut).Round(time.Second))
	return suivi, nil
}

// validerLesOptions : les incompatibilites de drapeaux, AVANT d ouvrir quoi que ce soit.
//
// Extraite de `runBackfillKillSource` au lot 5.24.2 : le drapeau `--workers` y ajoutait une
// cinquieme condition et poussait la fonction a une complexite de 18, au-dela du plafond du
// depot. La coupure suit une frontiere nette — ICI ce qui refuse, LA-BAS ce qui fait.
func validerLesOptions(o killsourceOptions) error {
	if o.filmsOnly && o.creditOnly {
		return fmt.Errorf("--films-only et --credit-only s excluent")
	}
	if o.online && o.gamertag == "" {
		return fmt.Errorf("--online exige --gamertag : il nomme le joueur DONT les films sont " +
			"traites (profil declare dans db_profiles.json) ; les jetons viennent du pool")
	}
	if !o.online && o.gamertag != "" {
		return fmt.Errorf("--gamertag n a de sens qu avec --online (la passe hors ligne n emet aucune requete)")
	}
	// LE DRAPEAU ACCEPTE EN SILENCE ETAIT UN MENSONGE (constat de revue, 2026-09-22) :
	// `--online` decode EN SERIE — son cout est le RESEAU, borne par `--rps`, pas le decodage —
	// et il ne lit jamais `o.workers`. Accepter la valeur, la valider contre le plafond memoire,
	// puis l ignorer laissait croire a N ouvriers pour une passe qui en a UN.
	if o.online && o.workersExplicite && o.workers > 1 {
		return fmt.Errorf("--online --workers %d : la passe en ligne reste EN SERIE — son cout "+
			"est le RESEAU (plafonne par --rps), pas le decodage, et paralleliser ne ferait "+
			"qu attendre plus vite en depassant le debit qu on s est donne. Relancer sans "+
			"--workers, ou avec --workers 1", o.workers)
	}
	return verifierLesOuvriers(o.workers)
}

// verifierLesOuvriers : le plafond memoire REFUSE au demarrage, avec le chiffre.
//
// Il refuse AVANT d ouvrir quoi que ce soit, et il le refuse en NOMMANT la mesure : un plafond
// qui dit seulement « trop » oblige a relire le code pour savoir pourquoi. Le pic vient de la
// mesure 5.24.1 sur le pire film du corpus, et il est re-mesurable par le meme test.
func verifierLesOuvriers(n int) error {
	if n < 1 {
		return fmt.Errorf("--workers %d : il en faut au moins un (1 = la boucle en serie)", n)
	}
	if max := killcollector.OuvriersMaximum(); n > max {
		return fmt.Errorf("--workers %d : le pic mesure du pire film du corpus est de %d Mio "+
			"(mesure 5.24.1, film 1c4c63c2 a 69 chunks) et la passe se plafonne a %d Mio, soit "+
			"%d ouvriers au maximum", n, killcollector.PicMemoireParFilm>>20,
			killcollector.PlafondMemoireDeLaPasse>>20, max)
	}
	return nil
}

// writerDeja : le handle RW est deja ouvert et le process est seul (serveur arrete). La
// fonction de lease se contente donc de le rendre — ADR 0013 est satisfaite par le fait qu il
// n existe qu UN writer, pas par un verrou supplementaire.
func writerDeja(db *sql.DB) func(context.Context) (*sql.DB, func(), error) {
	return func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil }
}

// resoudreCacheFilms : `--cache`, puis l environnement, puis le defaut du depot.
func resoudreCacheFilms(cfg *config.AppConfig, flagValue string) string {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("LEVELUP_LEGACY_FILM_CACHE_DIR")); v != "" {
		return v
	}
	return filepath.Join(cfg.RepoRoot, "data", "cache")
}

// capabilitesDuTitre : les capabilities lues dans `capabilities.toml`.
//
// LE COLLECTEUR SE BRANCHE SUR UNE CAPABILITY, JAMAIS SUR UN SLUG (ratchet
// no_slug_comparison_test.go). Un titre qui ne declare pas `film.kill_source` fait donc une
// passe VIDE, proprement — pas une erreur, pas un panic. La recette de lecture vit dans
// games.LoadCapabilityMap (centralisee le 2026-09-04, regle des <= 2 copies — ce fichier
// en portait une des trois copies d origine).
func capabilitesDuTitre(cfg *config.AppConfig, slug string) (games.CapabilityMap, error) {
	return games.LoadCapabilityMap(cfg.RepoRoot, slug)
}
