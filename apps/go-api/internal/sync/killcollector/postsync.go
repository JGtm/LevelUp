package killcollector

// postsync.go — L ETAPE 1.57 DU POST-SYNC : la source du kill, AU FIL DE L EAU.
//
// # POURQUOI ELLE EXISTE, ET POURQUOI ELLE N EST PAS UN OUTIL A PART
//
// Le decodage de kill-source n avait qu UN declencheur : la sous-commande manuelle
// `levelup backfill-killsource`, hors ligne, qui ne lisait que les films deja en cache. Or ce
// cache etait alimente par le projet Python supprime a la migration : il s est arrete le
// 2026-04-07. Depuis, AUCUN match synchronise n avait de passe de film — donc `assist_known`
// y valait FALSE (« on ne sait pas ») sur 100 % des morts, et les deux blocs d assistances de
// l app se retiraient. Cinq mois, sans un log. Mesure complete :
// `.ai/V7.5/REGISTRE_ASSISTANCES_2026-08-29.md`.
//
// UN RATTRAPAGE MANUEL AURAIT REPRODUIT LE DEFAUT. Une donnee qui ne se remplit que si
// quelqu un pense a lancer une commande finit toujours par ne plus se remplir. L etape vit
// donc dans le pipeline, a la place qu occupait l etape 1.55 (supprimee le 2026-09-01), qui
// exploitait deja le film du meme match, au meme moment, pour la meme raison : c est la que
// le film est disponible ET recent.
//
// # ELLE N EST PAS BRIDEE A UNE MACHINE, ET C EST DELIBERE
//
// Le rejeu 2D (etape 1.58) est interdit en production — son pic memoire vaut ~800 fois les
// octets du film. Ce n est PAS le cas ici : le pic du decodage kill-source vaut le film brut
// plus le film decompresse, et le plus gros film du corpus pese 88 Mio (mesure du 2026-08-24,
// cf. l en-tete de `cmd/levelup/cmd_backfill_killsource.go`). L etape tourne donc partout ou
// l etape 1.55 — qui telechargeait deja des films en production — tournait. Elle est bornee
// par cycle, pas par machine.
//
// # ELLE ARCHIVE, ET CELA FAIT BAISSER LE TRAFIC DU RESTE DU PIPELINE
//
// La source est [RemoteFilms] : disque d abord, reseau ensuite, et le reseau ecrit au cache.
// L etape 1.58 (artefacts de rejeu), qui telechargeait le film complet pour son propre pont
// disque, le trouve donc DEJA sur disque.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/killscope"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/sync/matchflags"
)

// DefaultPostSyncPerCycle borne le nombre de films decodes par cycle.
//
// MEME RAISON QUE `replayartifacts.maxPerCycle` : un sync initial insere des centaines de
// matchs, et decoder des centaines de films dans le post-sync bloquerait le cycle pendant des
// heures. Le solde est repris au cycle suivant (l etape est idempotente, `decoder_rev` fait
// foi) ou par `levelup backfill-killsource --online`.
const DefaultPostSyncPerCycle = 8

// PostSyncBudget : LE GARDE-FOU DE DUREE DU CYCLE, et il n est pas negociable.
//
// La borne `perCycle` compte des matchs, pas des secondes — or le cout d un film varie d un
// facteur dix. Sans budget, huit gros films suffiraient a rallonger un cycle de sync de
// plusieurs dizaines de minutes, et le reste du post-sync (PSA, agregats, medias) attendrait
// derriere. Le budget arrete la passe ENTRE deux matchs : ce qui reste est repris au cycle
// suivant, l etape etant idempotente (`decoder_rev` fait foi).
const PostSyncBudget = 5 * time.Minute

// PostSyncMatchTimeout : la limite PAR MATCH de cette etape.
//
// Le defaut du collecteur (45 min) est celui d une passe de backfill qu on laisse tourner la
// nuit. Dans un cycle de sync il serait absurde : un seul film pathologique tiendrait le
// pipeline trois quarts d heure. Un film qui ne se decode pas en trois minutes releve de
// `levelup backfill-killsource`, pas du fil de l eau.
const PostSyncMatchTimeout = 3 * time.Minute

// Compteurs de l etape, publies en expvar (ADR 0009).
const (
	CompteurPostSyncTraites = "killsource_postsync_matchs_traites"
	CompteurPostSyncRetard  = "killsource_postsync_backlog_restant"
	// CompteurPostSyncClientSansFilm : le client injecte ne porte pas GetFilmChunks, donc
	// l etape ne peut rien faire. C EST UN DEFAUT DE CABLAGE, PAS UN ETAT NORMAL.
	CompteurPostSyncClientSansFilm = "killsource_postsync_client_sans_film"
)

// FilmChunkFetcher : la SEULE capacite que l etape demande au client Halo.
//
// Elle est exportee pour que le cablage (paquet `sync`) et son garde-rail assertent la MEME
// interface — une copie du littéral d'interface cote appelant peut deriver d un parametre et
// desarmer l etape sans que rien ne le dise. C est exactement ce qui s est produit.
type FilmChunkFetcher = filmChunkFetcher

// PostSyncHook : le reglage de l etape, injecte au wiring.
//
// Il porte les capabilities parce qu elles se lisent sur DISQUE (`capabilities.toml`) et
// qu une lecture par cycle serait du gaspillage : elles sont resolues a la premiere passe
// puis conservees. Le collecteur se branche sur `film.kill_source`, JAMAIS sur un slug
// (ratchet no_slug_comparison_test.go) : un titre qui ne la declare pas fait une etape vide.
type PostSyncHook struct {
	repoRoot  string
	cacheRoot string
	perCycle  int

	caps    games.CapabilityMap
	capsErr error
	capsOK  bool

	cache       *haloclient.LocalFilmCache
	cacheRacine string
	cacheOK     bool
}

// NewPostSyncHook construit le reglage.
//
// La racine du cache vient du PathResolver, comme partout ailleurs (jamais un
// `filepath.Join(..., "data", ...)` a la main). `perCycle <= 0` = [DefaultPostSyncPerCycle].
func NewPostSyncHook(repoRoot string, perCycle int) *PostSyncHook {
	if perCycle <= 0 {
		perCycle = DefaultPostSyncPerCycle
	}
	return &PostSyncHook{
		repoRoot:  repoRoot,
		cacheRoot: titlePkg.NewPathResolver(repoRoot).CacheRootDir(),
		perCycle:  perCycle,
	}
}

// capabilities resout `capabilities.toml` UNE fois, puis rend le resultat memorise — succes
// comme echec. Reessayer a chaque cycle un fichier absent ne ferait que repeter le meme WARN.
func (h *PostSyncHook) capabilities(slug string) (games.CapabilityMap, error) {
	if h.capsOK {
		return h.caps, h.capsErr
	}
	h.capsOK = true
	reg := mappings.NewRegistry()
	for _, err := range reg.LoadFromConfigDir(h.repoRoot, []string{slug}, nil) {
		h.capsErr = fmt.Errorf("mappings du titre %s: %w", slug, err)
		return nil, h.capsErr
	}
	set, ok := reg.GetCapabilities(slug)
	if !ok {
		h.capsErr = fmt.Errorf("capabilities.toml absent pour le titre %s", slug)
		return nil, h.capsErr
	}
	caps, err := games.CapabilityMapFromMappings(set)
	if err != nil {
		h.capsErr = fmt.Errorf("capabilities du titre %s: %w", slug, err)
		return nil, h.capsErr
	}
	h.caps = caps
	return caps, nil
}

// racineDuCache : LA racine, celle qu on lit ET celle qu on ecrit.
//
// ⚠ DEUX RACINES ETAIENT UN PIEGE, ET TROIS RELECTEURS L ONT TROUVE LE MEME JOUR. Le moteur
// peut porter un cache herite (LEVELUP_LEGACY_FILM_CACHE_DIR) ; archiver dans la racine par
// defaut du depot pendant qu on lit celle-la revient a ecrire dans un repertoire que la
// lecture ne consulte jamais — le repli disque ne prend plus et chaque cycle repaie le reseau
// en entier. La racine du moteur gagne, et elle sert aux DEUX sens.
//
// Resolue UNE fois : les dossiers sont crees au premier passage (sans quoi
// `NewLocalFilmCache` rend nil pour toute la vie du process), puis le resultat est memorise.
func (h *PostSyncHook) racineDuCache(ctx context.Context, duMoteur *haloclient.LocalFilmCache) (string, *haloclient.LocalFilmCache) {
	if h.cacheOK {
		return h.cacheRacine, h.cache
	}
	h.cacheOK = true
	h.cacheRacine = duMoteur.RootDir()
	if h.cacheRacine == "" {
		h.cacheRacine = h.cacheRoot
	}
	if h.cacheRacine == "" {
		slog.WarnContext(ctx, "post-sync: killsource sans racine de cache — aucun film ne sera archive")
		return "", nil
	}
	if err := filmcache.EnsureDirs(h.cacheRacine); err != nil {
		slog.WarnContext(ctx, "post-sync: killsource cache film non preparable",
			"racine", h.cacheRacine, "err", err)
		h.cacheRacine = ""
		return "", nil
	}
	if duMoteur != nil {
		h.cache = duMoteur
	} else {
		h.cache = haloclient.NewLocalFilmCache(h.cacheRacine)
	}
	return h.cacheRacine, h.cache
}

// PostSyncDeps : ce dont l etape a besoin du moteur, et rien de plus.
//
// Les dependances sont passees en parametres (patron `replayartifacts`) : ce paquet ne
// connait ni `SyncEngine` ni aucun type prive du paquet `sync`.
type PostSyncDeps struct {
	// Fetcher : le client Halo. nil = etape vide (un mock de test qui ne porte pas la
	// capacite film ne doit pas faire echouer le cycle).
	Fetcher filmChunkFetcher
	// LocalCache : le cache disque deja instancie par le moteur. Peut etre nil.
	LocalCache *haloclient.LocalFilmCache
	// WithRead ouvre un segment de LECTURE shared court (jamais le writer).
	WithRead func(ctx context.Context, step string, fn func(sharedDB *sql.DB))
	// AcquireWriter : le burst writer, relache par l appelant du collecteur.
	AcquireWriter persist.SharedWriterFn
	TitleSlug     string
	Gamertag      string
}

// RunPostSync decode la source du kill des matchs inseres, puis rattrape le backlog.
//
// BEST-EFFORT SIGNALE : aucun echec ne casse le pipeline post-sync, mais aucun n est avale
// non plus — c est le silence qui a coute cinq mois.
//
// Elle REND le nombre de matchs ecrits, et ce n est pas cosmetique : la trace de cycle du
// post-sync (`clock.lap`) codait un zero en dur, de sorte que « a tourne et ecrit 8 » et
// « n a jamais tourne » etaient indistinguables sur la surface meme construite pour lever
// l ambiguite.
func RunPostSync(ctx context.Context, h *PostSyncHook, d PostSyncDeps, insertedIDs []string) (ecrits int) {
	if h == nil || d.Fetcher == nil || d.WithRead == nil || d.AcquireWriter == nil {
		return 0
	}
	caps, err := h.capabilities(d.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: killsource capabilities illisibles — etape sautee",
			"title", d.TitleSlug, "err", err)
		return 0
	}
	if !caps.Has(games.CapFilmKillSource) {
		return 0 // titre sans decodeur : etape vide, proprement.
	}

	var (
		backlog []string
		total   int
	)
	d.WithRead(ctx, "killsource_select", func(sharedDB *sql.DB) {
		backlog, total = backlogAJour(ctx, sharedDB, PostSyncBacklogHorizon)
	})
	travail, _ := ordonnancer(backlog, insertedIDs, h.perCycle)
	restant := max(0, total-len(travail))
	// Le backlog est PUBLIE meme a zero : une cle absente de /debug/vars ne se distingue pas
	// d une etape qui ne tourne pas, et c est exactement l ambiguite qui a dure cinq mois.
	observability.SetInt(CompteurPostSyncRetard, int64(restant))
	if len(travail) == 0 {
		return 0
	}

	racine, cache := h.racineDuCache(ctx, d.LocalCache)
	source := NewRemoteFilms(NewLocalCacheFilms(cache), d.Fetcher, racine)
	debut := time.Now()
	sum := NewKillSourceCollector(
		source, rosterParSegment{withRead: d.WithRead}, d.AcquireWriter, caps, PostSyncMatchTimeout,
	).WithBudget(PostSyncBudget).CollectMatches(ctx, travail)
	observability.AddInt(CompteurPostSyncTraites, int64(sum.Written))

	slog.InfoContext(ctx, "post-sync: kill source",
		"gamertag", d.Gamertag, "demandes", len(travail), "ecrits", sum.Written,
		"morts", sum.Deaths, "films_absents", sum.NoFilm, "sans_killfeed", sum.NoKillFeed,
		"erreurs", sum.Errors, "backlog_restant", restant,
		"duree", time.Since(debut).Round(time.Second))
	return sum.Written
}

// rosterParSegment : le roster du collecteur, servi par un segment de LECTURE court.
//
// LE MOTEUR DE SYNC N A PAS DE HANDLE PERMANENT, ET C EST VOULU (ADR 0013 / 0016 B-swap) :
// garder un `*sql.DB` shared vivant pendant toute la passe rendrait le B-swap RO<->RW
// impossible. Le collecteur resout le roster AVANT d acquerir le writer, donc les deux
// segments ne se chevauchent jamais — c est la meme garde anti-deadlock que l ex-etape 1.55
// (« le segment est relache AVANT le burst »).
type rosterParSegment struct {
	withRead func(ctx context.Context, step string, fn func(sharedDB *sql.DB))
}

func (r rosterParSegment) IdentitiesForMatch(
	ctx context.Context, matchID string,
) (MatchIdentities, error) {
	var (
		out    MatchIdentities
		errLec error
		ouvert bool
	)
	r.withRead(ctx, "killsource_roster", func(sharedDB *sql.DB) {
		ouvert = true
		out, errLec = NewSharedRoster(sharedDB).IdentitiesForMatch(ctx, matchID)
	})
	if !ouvert {
		// Segment indisponible : deja logge et compte par `withRead`. On rend une ERREUR
		// plutot qu un roster vide — un roster vide ferait ecrire des morts sans xuid,
		// c est-a-dire des lignes qu aucun agregat ne peut joindre (mesure du chantier :
		// 16 908 morts dont 10 seulement portaient un xuid de victime).
		return MatchIdentities{}, fmt.Errorf("killsource: segment de lecture shared indisponible (%s)", matchID)
	}
	return out, errLec
}

// ordonnancer : les matchs a decoder ce cycle, et la taille du backlog restant.
//
// Fonction PURE — c est ce qui la rend testable sans base, et c est voulu : l ordre est la
// moitie du sens de l etape.
//
// L ORDRE : les matchs inseres d abord (leur film vient d etre publie, c est le moment le plus
// sur), puis le backlog DANS L ORDRE OU LA REQUETE LE REND — du plus recent au plus vieux, cf.
// `requeteBacklog`. Cette fonction ne retrie RIEN : elle fusionne et deduplique.
//
// Un match insere qui n est PAS au backlog est ignore : il est deja decode a la revision
// courante, le redecoder serait payer le reseau pour rien.
//
// Le second rendu est le reliquat DE CETTE LISTE ; la jauge publiee, elle, part de la taille
// TOTALE du backlog (cf. `backlogAJour`) — une jauge calculee sur une liste bornee decrirait
// l horizon, pas le retard.
func ordonnancer(backlog, insertedIDs []string, perCycle int) (travail []string, restant int) {
	auBacklog := make(map[string]bool, len(backlog))
	for _, id := range backlog {
		auBacklog[id] = true
	}
	vus := make(map[string]bool, perCycle)
	ajouter := func(id string) {
		if id == "" || vus[id] {
			return
		}
		vus[id] = true
		travail = append(travail, id)
	}
	for _, id := range insertedIDs {
		if len(travail) >= perCycle {
			break
		}
		if auBacklog[id] {
			ajouter(id)
		}
	}
	for _, id := range backlog {
		if len(travail) >= perCycle {
			break
		}
		ajouter(id)
	}
	return travail, max(0, len(backlog)-len(travail))
}

// matchsSansPasseAJour : les matchs du registre dont la passe COURANTE ne porte pas la
// revision de decodeur courante sur une voie de FILM, du plus vieux au plus recent.
//
// ⚠ LECTURE PAR LA VUE `_latest` (ADR 0026) : une lecture brute servirait des passes
// perimees et ferait sauter des matchs qui ont besoin d etre redecodes.
//
// Le filtre `read_path <> credit-backfill` est ce qui rend l etape UTILE : un match couvert
// par le producteur credit-seul reste candidat, parce que c est precisement le credit qui ne
// sait rien de l assistant. Sans ce filtre, l etape se croirait a jour sur toute la base et
// l attribution ne repartirait jamais.
//
// `start_time_utc` par le COALESCE canonique (regle 8) — un `start_time` brut trierait faux.
// PostSyncBacklogHorizon borne la liste de travail lue par cycle.
//
// Elle n est pas la borne de TRAVAIL (c est `perCycle`) mais celle de la LECTURE : ramener
// 999 identifiants pour en traiter 8 est du gaspillage. La TAILLE du backlog reste comptee
// separement, sans borne — un compteur tronque mentirait sur le retard.
const PostSyncBacklogHorizon = 64

// conditionBacklog : le predicat commun a la liste et a la jauge. UNE seule copie, parce que
// deux copies divergent et la jauge se met alors a decrire un autre ensemble que le travail.
//
// TROIS CONDITIONS, TROIS RAISONS — les retirer casse l etape en silence :
//
//	match_kill_events_latest        la VUE, jamais la table (ADR 0026) : une passe perimee
//	                                ferait sauter un match qui a besoin d etre redecode.
//	read_path <> credit-backfill    c est CE filtre qui rend l etape utile. Toute la base est
//	                                couverte par le producteur credit-seul, qui porte pourtant
//	                                la revision courante : sans lui, l etape se croirait a jour
//	                                partout et l assistant ne se resoudrait plus jamais.
//	backfill_completed & no_film    LE MARQUEUR TERMINAL, ET IL EXISTAIT DEJA. L etape 1.55
//	                                (weapon kills) tourne AVANT celle-ci sur le meme film et
//	                                pose `MBitWeaponKillsNoFilm` quand 343 ne le sert plus.
//	                                Sans cette exclusion, un film expire ne produit aucune
//	                                ligne, donc le match reste au backlog A VIE : mesure du
//	                                2026-08-29, 581 des 999 candidats sont dans ce cas, tous
//	                                anterieurs a 2026. Ils occupaient toute la liste de travail
//	                                et les 415 matchs de 2026 n etaient JAMAIS atteints.
const conditionBacklog = `
		FROM match_registry r
		WHERE COALESCE(r.backfill_completed, 0) & ? = 0
		  AND NOT EXISTS (
			SELECT 1 FROM match_kill_events_latest e
			WHERE e.match_id = r.match_id
			  AND e.decoder_rev = ?
			  AND e.read_path <> ?
		)`

// requeteBacklog : les identifiants a traiter, DU PLUS RECENT AU PLUS VIEUX.
//
// L ORDRE A ETE INVERSE LE 2026-08-29, ET LE RAISONNEMENT D ORIGINE ETAIT FAUX. Il disait
// « les films expirent, sauvons d abord les plus vieux ». Mais un film deja expire ne se sauve
// pas : il est perdu, et le depot le SAIT (bit no-film ci-dessus). Trier du plus vieux au plus
// recent revenait donc a offrir toute la liste de travail a des matchs irrecuperables. Les
// recents sont a la fois les seuls recuperables et ceux que l utilisateur regarde.
var requeteBacklog = `SELECT r.match_id` + conditionBacklog + `
		ORDER BY ` + analysis.SQLStartTimeCanonical("r") + ` DESC, r.match_id
		LIMIT ?`

// requeteBacklogTaille : la jauge, SANS borne. Elle repond a « combien reste-t-il », pas a
// « qu est-ce que je traite maintenant ».
var requeteBacklogTaille = `SELECT COUNT(*)` + conditionBacklog

// backlogAJour rend la liste de travail bornee et la taille TOTALE du backlog.
//
// ⚠ LECTURE PAR LA VUE `_latest` (ADR 0026) : une lecture brute servirait des passes perimees
// et ferait sauter des matchs a redecoder.
func backlogAJour(ctx context.Context, db *sql.DB, horizon int) (ids []string, total int) {
	args := []any{matchflags.MBitFilmAbsent, KillSourceDecoderRev, killscope.ReadPathCreditBackfill}

	if err := db.QueryRowContext(ctx, requeteBacklogTaille, args...).Scan(&total); err != nil {
		slog.WarnContext(ctx, "post-sync: killsource taille du backlog illisible", "err", err)
		// On continue : une jauge absente ne doit pas empecher le travail.
	}
	rows, err := db.QueryContext(ctx, requeteBacklog, append(args, horizon)...)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: killsource backlog illisible", "err", err)
		return nil, total
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			slog.WarnContext(ctx, "post-sync: killsource backlog (scan)", "err", err)
			return ids, total
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "post-sync: killsource backlog (rows)", "err", err)
	}
	return ids, total
}
