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
// Un match dont la passe COURANTE (vue `_latest`) porte la revision de decodeur courante est
// saute. Interrompre la passe et la relancer reprend donc ou elle en etait, sans re-decoder ce
// qui est deja fait. `--force` redecode tout — c est ce qu il faut le jour ou la revision change.
//
// # ELLE PASSE LES GROS FILMS EN DERNIER, ET CE N EST PAS UNE PREFERENCE
//
// Le cout par film reste croissant avec sa taille (0,13 s/chunk a 8 chunks, 0,68 s/chunk a 69
// apres le correctif du 2026-08-01). Trier par taille croissante fait que la passe produit la
// quasi-totalite de sa valeur AVANT de s attaquer aux quelques films qui coutent le plus : une
// interruption au bout d une heure laisse alors un resultat presque complet, pas un tiers de
// corpus.
//
// # ELLE BOUCLE DANS UN SEUL PROCESSUS, ET ON L Y LAISSE (examen du 2026-08-24)
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
//	   (garde vivant, `ChunkSourceOf` l aliase) plus le film decompresse (`loadFilm` :
//	   `f.chunks = make([][]byte, n)`). Mesure du cache au 2026-08-24, 951 films : le PLUS GROS
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
// Usage (SERVEUR ARRETE — `OpenReadWrite` echoue si le lock est tenu) :
//
//	levelup backfill-killsource --dry-run              # ce qui SERAIT fait, aucune ecriture
//	levelup backfill-killsource --limit 20             # les 20 films les moins chers
//	levelup backfill-killsource                        # tout : films puis credit
//	levelup backfill-killsource --credit-only          # la passe SQL -> SQL seule (secondes)
//	levelup backfill-killsource --force                # redecode meme ce qui est a jour
//
// Le cache de films se resout par `--cache`, sinon `LEVELUP_LEGACY_FILM_CACHE_DIR`, sinon
// `<repo>/data/cache`.

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/killscope"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
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
	fs.StringVar(&o.gamertag, "gamertag", "", "joueur dont les tokens servent la passe --online (obligatoire avec --online)")
	fs.IntVar(&o.rps, "rps", 4, "debit maximal des requetes Halo de la passe --online")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if o.filmsOnly && o.creditOnly {
		return fmt.Errorf("--films-only et --credit-only s excluent")
	}
	if o.online && o.gamertag == "" {
		return fmt.Errorf("--online exige --gamertag : les films se telechargent avec les tokens " +
			"d un joueur declare dans db_profiles.json")
	}
	if !o.online && o.gamertag != "" {
		return fmt.Errorf("--gamertag n a de sens qu avec --online (la passe hors ligne n emet aucune requete)")
	}

	ctx := context.Background()
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
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

	if !o.creditOnly {
		passe := passeDesFilms
		if o.online {
			passe = passeDesFilmsEnLigne
		}
		if err := passe(ctx, cfg, db, o); err != nil {
			return err
		}
	}
	if !o.filmsOnly {
		if err := passeDuCredit(ctx, db, o); err != nil {
			return err
		}
	}
	if !o.dryRun {
		afficherSante(o.online)
	}
	return nil
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
func passeDesFilms(ctx context.Context, cfg *config.AppConfig, db *sql.DB, o killsourceOptions) error {
	cacheRoot := resoudreCacheFilms(cfg, o.cacheDir)
	cache := haloclient.NewLocalFilmCache(cacheRoot)
	if cache == nil {
		return fmt.Errorf("cache de films introuvable sous %s — cette passe est HORS LIGNE, "+
			"elle n a pas d autre source", cacheRoot)
	}

	candidats, err := filmsACollecter(ctx, db, cacheRoot, o)
	if err != nil {
		return err
	}
	fmt.Printf("films a decoder : %d (cache %s)\n", len(candidats), cacheRoot)
	if len(candidats) == 0 {
		return nil
	}
	if o.dryRun {
		afficherPlan(candidats)
		return nil
	}

	caps, err := capabilitesDuTitre(cfg, o.titleSlug)
	if err != nil {
		return err
	}
	collecteur := killcollector.NewKillSourceCollector(
		killcollector.NewLocalCacheFilms(cache),
		killcollector.NewSharedRoster(db),
		writerDeja(db),
		caps,
		0, // limite par match : le defaut du collecteur (45 min)
	)
	// CAPTURE DES POSITIONS (G.2bis), BEST-EFFORT : catalogue de bornes illisible ou metadata
	// indisponible degrade en « positions desactivees » (le collecteur continue sans elles, la
	// passe des morts/tirs n en depend pas). ACTIVEE PAR DEFAUT ici, PAS derriere un flag CLI —
	// c est la seule commande de backfill de ce producteur, et une feature OFF « pour plus tard »
	// est l anti-pattern que CLAUDE.md interdit (regle 11) : la capture est prete, elle capture.
	mapNames, mapBounds, cleanupPositions := positionCaptureDeps(cfg, o.titleSlug, db)
	defer cleanupPositions()
	if mapNames != nil && mapBounds != nil {
		collecteur = collecteur.WithPositionCapture(mapNames, mapBounds)
	}

	ids := make([]string, 0, len(candidats))
	for _, c := range candidats {
		ids = append(ids, c.matchID)
	}
	debut := time.Now()
	sum := collecteur.CollectMatches(ctx, ids)
	fmt.Printf("films : %d ecrits (%d morts), %d absents, %d sans kill-feed, "+
		"%d abandons sur delai, %d erreurs, %d sans capability — %s\n",
		sum.Written, sum.Deaths, sum.NoFilm, sum.NoKillFeed, sum.Timeouts, sum.Errors,
		sum.NotSupport, time.Since(debut).Round(time.Second))
	return nil
}

// positionCaptureDeps construit les dependances de la capture des positions (G.2bis) :
// resolution de carte (`port.ReplayMapNameRepo`, implementee par `duckdb.ReplayMapRepo`) et
// catalogue de bornes de dequantification (le meme que `replaybuild`, DONNEE DE REFERENCE
// VERSIONNEE — data/titles/{slug}/reference/map_quant_bounds.json, pas une sortie de sync).
//
// BEST-EFFORT PAR CONCEPTION : un catalogue illisible ou une metadata indisponible degrade en
// « positions desactivees », jamais une erreur fatale — le backfill des morts et des tirs, la
// raison d etre de cette commande, ne doit pas dependre d une brique tierce. `cleanup` ferme le
// handle metadata ouvert ici ; elle est TOUJOURS non-nil (no-op si rien n a ete ouvert), donc
// l appelant peut la `defer` inconditionnellement.
func positionCaptureDeps(
	cfg *config.AppConfig, titleSlug string, sharedDB *sql.DB,
) (port.ReplayMapNameRepo, *filmdec.MapQuantCatalog, func()) {
	noop := func() {}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)

	catalog, err := filmdec.LoadMapQuantCatalog(pr.MapQuantBoundsPath(titleSlug))
	if err != nil {
		fmt.Printf("catalogue de bornes indisponible (%v) — positions desactivees pour cette passe\n", err)
		return nil, nil, noop
	}

	// OpenReadOnly (pas OpenReadForQuery) : la precondition de CETTE commande est le SERVEUR
	// ARRETE (comme pour le handle RW de shared), donc aucun autre process ne tient metadata en
	// ecriture pendant la passe — la garde « different configuration » d OpenReadForQuery n a
	// rien a proteger ici, et NewReplayMapRepo demande le type *duckdb.DB, pas *sql.DB brut.
	metaDB, err := duckdb.OpenReadOnly(pr.MetadataDBPath(titleSlug))
	if err != nil {
		fmt.Printf("metadata illisible (%v) — positions desactivees pour cette passe\n", err)
		return nil, nil, noop
	}

	repo := duckdb.NewReplayMapRepo(staticSharedReader{db: sharedDB}, metaDB)
	return repo, catalog, func() { _ = metaDB.Close() }
}

// staticSharedReader adapte le handle shared DEJA OUVERT (le lease de cette commande) en
// duckdb.SharedReader. Meme raison que writerDeja : le process est seul (serveur arrete), donc
// release est un no-op — il n y a pas de second lease a poser par-dessus celui deja tenu.
type staticSharedReader struct{ db *sql.DB }

func (s staticSharedReader) Get(context.Context) (*sql.DB, func(), error) {
	return s.db, func() {}, nil
}

// passeDuCredit : la transformation SQL -> SQL, sur TOUS les matchs du registre.
//
// Elle passe sur tous les matchs et pas seulement sur ceux sans film : c est le producteur
// lui-meme qui applique la preseance (il refuse un match qu une passe de film couvre deja), et
// centraliser cette regle a UN endroit vaut mieux que de la recopier dans la selection.
func passeDuCredit(ctx context.Context, db *sql.DB, o killsourceOptions) error {
	ids, err := matchsDuRegistre(ctx, db, o.limit)
	if err != nil {
		return err
	}
	fmt.Printf("credit-seul : %d matchs a examiner\n", len(ids))
	if o.dryRun || len(ids) == 0 {
		return nil
	}
	debut := time.Now()
	sum := killcollector.NewCreditCollector(db, writerDeja(db)).CollectMatches(ctx, ids)
	fmt.Printf("credit : %d ecrits + %d enrichis par un film (%d morts), "+
		"%d sans evenement, %d erreurs — %s\n",
		sum.Written, sum.Enriched, sum.Deaths, sum.NoEvents, sum.Errors,
		time.Since(debut).Round(time.Second))
	return nil
}

// writerDeja : le handle RW est deja ouvert et le process est seul (serveur arrete). La
// fonction de lease se contente donc de le rendre — ADR 0013 est satisfaite par le fait qu il
// n existe qu UN writer, pas par un verrou supplementaire.
func writerDeja(db *sql.DB) func(context.Context) (*sql.DB, func(), error) {
	return func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil }
}

// filmsACollecter : les films du cache qui correspondent a un match du registre, tries par
// COUT CROISSANT, moins ceux qui sont deja a jour.
func filmsACollecter(
	ctx context.Context, db *sql.DB, cacheRoot string, o killsourceOptions,
) ([]filmCandidat, error) {
	registre, err := matchsDuRegistre(ctx, db, 0)
	if err != nil {
		return nil, err
	}
	dejaFaits := map[string]bool{}
	if !o.force {
		if dejaFaits, err = matchsAJour(ctx, db); err != nil {
			return nil, err
		}
	}

	out := make([]filmCandidat, 0, len(registre))
	for _, id := range registre {
		if dejaFaits[id] {
			continue
		}
		n, ok := compterChunks(cacheRoot, id)
		if !ok {
			continue // pas de film en cache : ce match releve du producteur credit-seul
		}
		out = append(out, filmCandidat{matchID: id, chunks: n})
	}
	// LES GROS EN DERNIER. A cout egal, l identifiant departage — une passe doit etre
	// reproductible, y compris dans son ordre.
	sort.Slice(out, func(i, j int) bool {
		if out[i].chunks != out[j].chunks {
			return out[i].chunks < out[j].chunks
		}
		return out[i].matchID < out[j].matchID
	})
	if o.limit > 0 && len(out) > o.limit {
		out = out[:o.limit]
	}
	return out, nil
}

// matchsAJour : les matchs dont la passe COURANTE porte la revision de decodeur courante.
//
// La lecture passe par la VUE `_latest` (ADR 0026) : une passe ancienne, deja supplantee, ne
// doit pas faire sauter un match. `read_path` distingue les deux producteurs — un match couvert
// par le credit-seul reste candidat au decodage de son film, et c est voulu : le film apporte la
// source du degat, que le credit ne peut pas connaitre.
func matchsAJour(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT match_id FROM match_kill_events_latest
		WHERE decoder_rev = ? AND read_path <> ?`,
		killcollector.KillSourceDecoderRev, killscope.ReadPathCreditBackfill)
	if err != nil {
		return nil, fmt.Errorf("matchs deja a jour: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("matchs deja a jour (scan): %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// matchsDuRegistre : les matchs du registre, dans un ordre stable.
func matchsDuRegistre(ctx context.Context, db *sql.DB, limit int) ([]string, error) {
	q := `SELECT match_id FROM match_registry ORDER BY match_id`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("registre des matchs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("registre des matchs (scan): %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// compterChunks : le nombre de chunks declares au manifeste cache. C est le PROXY DE COUT, et
// il est lu sans ouvrir un seul chunk.
func compterChunks(cacheRoot, matchID string) (int, bool) {
	court := matchID
	if i := strings.IndexByte(matchID, '-'); i > 0 {
		court = matchID[:i]
	}
	raw, err := os.ReadFile(filepath.Join(cacheRoot, "film_manifests", court+".json"))
	if err != nil {
		return 0, false
	}
	var m struct {
		Chunks []struct{} `json:"chunks"`
	}
	if err := json.Unmarshal(raw, &m); err != nil || len(m.Chunks) == 0 {
		return 0, false
	}
	return len(m.Chunks), true
}

// afficherPlan : le plan de passe, avec sa queue de films chers en evidence.
func afficherPlan(candidats []filmCandidat) {
	total := 0
	gros := 0
	for _, c := range candidats {
		total += c.chunks
		if c.chunks > 50 {
			gros++
		}
	}
	fmt.Printf("  chunks a decoder : %d au total, %d film(s) au-dela de 50 chunks (passes en dernier)\n",
		total, gros)
	for i, c := range candidats {
		if i >= 5 && i < len(candidats)-3 {
			continue
		}
		if i == 5 {
			fmt.Println("  ...")
		}
		fmt.Printf("  %-40s %3d chunks\n", c.matchID, c.chunks)
	}
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
