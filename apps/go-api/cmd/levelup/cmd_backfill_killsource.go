package main

// cmd_backfill_killsource.go — sous-commande `levelup backfill-killsource`.
//
// ELLE REMPLIT `shared.match_kill_events` ET `shared.match_weapon_shots`, par les DEUX
// producteurs, dans l ordre qui garantit la preseance :
//
//	1. LE FILM      — decodage hors ligne des films en cache. Les deux verites (credit du
//	                  kill-feed + source du degat), plus la ventilation des tirs par arme.
//	2. LE CREDIT    — depuis `highlight_events`, EN BASE, sans film. Il apporte les matchs dont
//	                  le film a EXPIRE cote serveur (29,3 % du corpus), et il REFUSE tout match
//	                  qu une passe de film couvre deja.
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

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	if o.filmsOnly && o.creditOnly {
		return fmt.Errorf("--films-only et --credit-only s excluent")
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
		if err := passeDesFilms(ctx, cfg, db, o); err != nil {
			return err
		}
	}
	if !o.filmsOnly {
		if err := passeDuCredit(ctx, db, o); err != nil {
			return err
		}
	}
	if !o.dryRun {
		afficherSante()
	}
	return nil
}

// santeCompteurs : les compteurs a rendre en fin de passe.
//
// ILS SONT PUBLIES EN `expvar` (ADR 0009), ce qui les rend interrogeables sur un SERVEUR — mais
// cette commande est un process qui s arrete : sans cette restitution, ses compteurs mourraient
// avec lui. Un compteur qu on ne peut lire qu apres coup n alerte personne, et c est
// precisement ce que le gate de cette phase demande de constater.
var santeCompteurs = []string{
	"killsource_matchs_collectes", "killsource_morts_ecrites",
	"killsource_films_absents", "killsource_sans_killfeed",
	"killsource_erreurs_decodage", "killsource_abandons_delai", "killsource_erreurs_ecriture",
	"killsource_passes_non_publiables", "killsource_assist_extra_count",
	"killsource_tirs_matchs_ventiles", "killsource_tirs_lignes_ecrites",
	"killsource_tirs_indices_non_resolus", "killsource_tirs_erreurs_ecriture",
	"killsource_credit_matchs_ecrits", "killsource_credit_morts_ecrites",
	"killsource_credit_matchs_deja_couverts_par_un_film", "killsource_credit_matchs_sans_evenement",
}

// afficherSante restitue les compteurs de sante de la passe.
func afficherSante() {
	fmt.Println("compteurs de sante de la passe :")
	for _, nom := range santeCompteurs {
		fmt.Printf("  %-52s %d\n", nom, observability.LoadCounter(nom))
	}
	fmt.Println("  ⚠ killsource_assist_extra_count > 0 = l hypothese de schema « un seul " +
		"assistant » est en defaut : migrer vers une table fille (ADR 0026).")
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
	fmt.Printf("credit-seul : %d ecrits (%d morts), %d couverts par un film, "+
		"%d sans evenement, %d erreurs — %s\n",
		sum.Written, sum.Deaths, sum.SkippedFilm, sum.NoEvents, sum.Errors,
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
		killcollector.KillSourceDecoderRev, killcollector.CreditReadPath)
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
// passe VIDE, proprement — pas une erreur, pas un panic.
func capabilitesDuTitre(cfg *config.AppConfig, slug string) (games.CapabilityMap, error) {
	reg := mappings.NewRegistry()
	for _, err := range reg.LoadFromConfigDir(cfg.RepoRoot, []string{slug}, nil) {
		return nil, fmt.Errorf("mappings du titre %s: %w", slug, err)
	}
	set, ok := reg.GetCapabilities(slug)
	if !ok {
		return nil, fmt.Errorf("capabilities.toml absent pour le titre %s", slug)
	}
	caps, err := games.CapabilityMapFromMappings(set)
	if err != nil {
		return nil, fmt.Errorf("capabilities du titre %s: %w", slug, err)
	}
	return caps, nil
}
