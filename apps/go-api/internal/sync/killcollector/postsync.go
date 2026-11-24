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
// l app se retiraient. Cinq mois, sans un log.
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
//
// # UNE SEULE PASSE PAR (PROCESSUS, TITRE) A LA FOIS
//
// Le cycle v2 appelle [RunPostSync] une fois PAR JOUEUR, en parallele, et l arriere est GLOBAL.
// C est l exclusivite de `postsync_exclusivite.go` qui garde la boucle en serie a l echelle du
// processus : le premier appel d un titre prend la passe, les autres se retirent (rendent 0,
// [CompteurPostSyncPasseDejaEnCours]). Sans elle, N joueurs decodaient N fois les memes films.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmarchive"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/filmbacklog"
	"levelup/go-api/internal/sync/filmfetch"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
)

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
	// CompteurPostSyncRetard : le backlog restant apres le cycle, matchs sans carte COMPRIS ; le
	// retard que le decodeur peut resorber est ce nombre moins [CompteurPostSyncSansCarte].
	CompteurPostSyncRetard = "killsource_postsync_backlog_restant"
	// CompteurPostSyncClientSansFilm : le client injecte ne porte pas GetFilmChunks, donc
	// l etape ne peut rien faire. C EST UN DEFAUT DE CABLAGE, PAS UN ETAT NORMAL.
	CompteurPostSyncClientSansFilm = "killsource_postsync_client_sans_film"
	// CompteurPostSyncPasseDejaEnCours : un appel a trouve une passe du MEME titre en cours
	// dans le process et s est retire (cf. postsync_exclusivite.go). CE N EST PAS UN DEFAUT :
	// c est le cas normal d un cycle qui synchronise plusieurs joueurs a la fois.
	CompteurPostSyncPasseDejaEnCours = "killsource_postsync_passe_deja_en_cours"
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
	// archiveRoot rend la racine de l'archive des films, relue à chaque passe : une archive vide
	// peut changer d'emplacement à chaud (title.SetFilmArchiveRuntime). Nil : `cacheRoot` seul.
	archiveRoot func() string
	perCycle    int
	// archive : le reglage « archiver les films », relu a chaque film (WithArchive). Nil = oui.
	archive func(context.Context) bool
	// budget : la duree maximale d une passe ([PostSyncBudget]). Un champ et non la constante en dur,
	// pour que les tests prouvent l arret sur le temps sans attendre cinq minutes.
	budget time.Duration
	// horizon : la taille d une PAGE de backlog lue par cycle ([PostSyncBacklogHorizon]). Un champ
	// et non la constante en dur, pour que les tests exercent la pagination sans inscrire 64 matchs.
	horizon int

	caps    games.CapabilityMap
	capsErr error
	capsOK  bool

	cache       *filmfetch.LocalFilmCache
	cacheRacine string
	cacheVoulue string
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
		repoRoot:    repoRoot,
		archiveRoot: titlePkg.NewPathResolver(repoRoot).FilmArchiveDir,
		perCycle:    perCycle,
		budget:      PostSyncBudget,
		horizon:     PostSyncBacklogHorizon,
	}
}

// WithArchive branche l'ecriture des films lus au reseau sur le reglage « archiver les films » : la
// fonction est relue a chaque film, un reglage change a chaud prend effet a la passe en cours. Sans
// elle (nil), la source archive tout ce qu'elle lit. Le moteur de synchronisation l'appelle depuis le
// hook des rejeux, qui porte le reglage. Coupe, rien n'est ecrit, dans aucun profil.
func (h *PostSyncHook) WithArchive(archive func(context.Context) bool) *PostSyncHook {
	h.archive = archive
	return h
}

// ArchiveFilms dit si la source des frags ecrit a l'archive le film qu'elle lit au reseau (oui sans
// [PostSyncHook.WithArchive]).
func (h *PostSyncHook) ArchiveFilms(ctx context.Context) bool {
	return h.archive == nil || h.archive(ctx)
}

// capabilities resout `capabilities.toml` UNE fois, puis rend le resultat memorise — succes
// comme echec. Reessayer a chaque cycle un fichier absent ne ferait que repeter le meme WARN.
// La recette de lecture vit dans games.LoadCapabilityMap (centralisee le 2026-09-04, regle
// des <= 2 copies) ; seule la MEMORISATION reste ici, propre a la vie du hook.
func (h *PostSyncHook) capabilities(slug string) (games.CapabilityMap, error) {
	if h.capsOK {
		return h.caps, h.capsErr
	}
	h.capsOK = true
	h.caps, h.capsErr = games.LoadCapabilityMap(h.repoRoot, slug)
	return h.caps, h.capsErr
}

// capture construit les dependances de la capture des positions, BEST-EFFORT.
//
// Un catalogue de bornes illisible ou un resolveur absent degrade en « positions desactivees »
// pour ce cycle — jamais une erreur : le journal des morts, raison d'etre de l'etape, ne doit
// pas dependre d'une brique tierce. La degradation se JOURNALISE (le silence a deja coute cinq
// mois sur cette etape).
func (h *PostSyncHook) capture(ctx context.Context, d PostSyncDeps) DepsCapture {
	if d.MapNames == nil {
		slog.InfoContext(ctx, "post-sync: killsource — positions desactivees (aucun resolveur "+
			"de carte cable)", "title", d.TitleSlug)
		return DepsCapture{}
	}
	deps, err := CaptureDepuisCatalogue(ctx, h.repoRoot, d.TitleSlug, d.MapNames)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: killsource — positions desactivees",
			"title", d.TitleSlug, "err", err)
		return DepsCapture{}
	}
	return deps
}

// racineDuCache : LA racine, celle qu on lit ET celle qu on ecrit.
//
// ⚠ DEUX RACINES ETAIENT UN PIEGE, ET TROIS RELECTEURS L ONT TROUVE LE MEME JOUR. Le moteur
// peut porter un cache herite (LEVELUP_LEGACY_FILM_CACHE_DIR) ; archiver dans la racine par
// defaut du depot pendant qu on lit celle-la revient a ecrire dans un repertoire que la
// lecture ne consulte jamais — le repli disque ne prend plus et chaque cycle repaie le reseau
// en entier. La racine du moteur gagne, et elle sert aux DEUX sens.
//
// Resolue au premier passage : les dossiers sont crees alors (sans quoi `NewLocalFilmCache` rend
// nil), puis le resultat est memorise TANT QUE LA RACINE VOULUE NE CHANGE PAS — une archive vide
// qui change d'emplacement a chaud est reprise a la passe suivante.
func (h *PostSyncHook) racineDuCache(ctx context.Context, duMoteur *filmfetch.LocalFilmCache) (string, *filmfetch.LocalFilmCache) {
	voulue := h.racineVoulue(duMoteur)
	if h.cacheOK && h.cacheVoulue == voulue {
		return h.cacheRacine, h.cache
	}
	h.cacheOK, h.cacheVoulue, h.cache = true, voulue, nil
	h.cacheRacine = voulue
	if h.cacheRacine == "" {
		slog.WarnContext(ctx, "post-sync: killsource sans racine de cache — aucun film ne sera archive")
		return "", nil
	}
	if err := filmarchive.EnsureDirs(h.cacheRacine); err != nil {
		slog.WarnContext(ctx, "post-sync: killsource cache film non preparable",
			"racine", h.cacheRacine, "err", err)
		h.cacheRacine = ""
		return "", nil
	}
	if duMoteur != nil {
		h.cache = duMoteur
	} else {
		h.cache = filmfetch.NewLocalFilmCache(h.cacheRacine)
	}
	return h.cacheRacine, h.cache
}

// racineVoulue rend la racine de l archive des films que la passe doit lire, SANS rien creer : celle du
// cache du moteur quand il en porte une, sinon celle du reglage (relue a chaque passe).
func (h *PostSyncHook) racineVoulue(duMoteur *filmfetch.LocalFilmCache) string {
	if racine := duMoteur.RootDir(); racine != "" {
		return racine
	}
	if h.archiveRoot != nil {
		return h.archiveRoot()
	}
	return h.cacheRoot
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
	LocalCache *filmfetch.LocalFilmCache
	// WithRead ouvre un segment de LECTURE shared court (jamais le writer).
	WithRead func(ctx context.Context, step string, fn func(sharedDB *sql.DB))
	// AcquireWriter : le burst writer, relache par l appelant du collecteur.
	AcquireWriter persist.SharedWriterFn
	TitleSlug     string
	Gamertag      string
	// MapNames resout l'identite de carte d'un match. NON NIL = la capture des positions
	// (et donc les faits d'isolement) s'active pour ce cycle.
	//
	// ELLE MANQUAIT, ET C'EST TOUT LE DEFAUT P0-1 : `WithPositionCapture` n'etait appele que
	// par le backfill hors ligne, si bien qu'aucune position n'etait jamais produite AU FIL
	// DU SYNC en production. L'appelant la fournit parce que lui seul sait quel handle
	// metadata il a le droit d'ouvrir (modele mono-process, ADR 0013).
	MapNames port.ReplayMapNameRepo
}

// RunPostSync decode la source du kill des matchs inseres, puis rattrape le backlog.
//
// BEST-EFFORT SIGNALE : aucun echec ne casse le pipeline post-sync, mais aucun n est avale
// non plus — c est le silence qui a coute cinq mois.
//
// Elle REND le nombre de matchs ecrits, et ce n est pas cosmetique : la trace de cycle du
// post-sync (`clock.lap`) codait un zero en dur, de sorte que « a tourne et ecrit 8 » et
// « n a jamais tourne » etaient indistinguables sur la surface meme construite pour lever
// l ambiguite. Un appel retire parce qu une passe du meme titre tourne deja rend aussi 0 :
// c est [CompteurPostSyncPasseDejaEnCours] qui le distingue.
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
	// UNE PASSE PAR (PROCESSUS, TITRE) : prise ICI, apres les gardes et avant la lecture de
	// l arriere global (OPS-3, cf. postsync_exclusivite.go).
	passe := verrouDePasse(d.TitleSlug)
	if !passe.TryLock() {
		signalerPasseDejaEnCours(ctx, d)
		return 0
	}
	defer passe.Unlock()

	// Le backlog se lit dans UN segment ; la meme lecture rend, UNE fois par (processus, titre), les
	// marques « film absent » a reconcilier avec l archive (film_absent_reconcile.go).
	backlog, total, marques := h.lireArriere(ctx, d)
	reconcilierFilmsAbsents(ctx, d, h.racineVoulue(d.LocalCache), marques)
	// Le backlog est PUBLIE meme a zero : une cle absente de /debug/vars ne se distingue pas
	// d une etape qui ne tourne pas, et c est exactement l ambiguite qui a dure cinq mois.
	if len(backlog) == 0 {
		publierArriere(d.TitleSlug, total, 0)
		return 0
	}

	racine, cache := h.racineDuCache(ctx, d.LocalCache)
	source := NewRemoteFilms(NewLocalCacheFilms(cache), d.Fetcher, racine).AvecArchivage(h.ArchiveFilms)
	debut := time.Now()
	capture := h.capture(ctx, d)
	col := NewKillSourceCollector(
		source, rosterParSegment{withRead: d.WithRead}, d.AcquireWriter, caps, PostSyncMatchTimeout,
	).WithBudget(h.budget).AvecCapture(capture).AvecFilmComplet(func(matchID string) bool {
		ok, err := filmcache.FilmComplet(racine, titlePkg.FilmShortMatchID(matchID))
		return err == nil && ok
	})
	// LA CARTE AVANT LA BORNE ET AVANT LE TELECHARGEMENT (2026-09-27, cf. postsync_travail.go) :
	// un match sans carte ne prend ni une place du cycle ni un telechargement, et un match DEJA
	// constate sans carte n est pas relu (cf. postsync_sans_carte.go).
	reg := registreDuCycle(ctx, d.TitleSlug, capture)
	travail := h.travailDuCycle(ctx, col, reg,
		sourceDuBacklog{premiere: backlog, lire: lectureDesPages(ctx, d, h.horizon)}, insertedIDs)
	horsTravail := max(0, total-len(travail))
	if len(travail) == 0 {
		publierArriere(d.TitleSlug, horsTravail, nbSansCarte(reg, horsTravail))
		return 0
	}
	// LE VERROU SOLO AVANT TOUT DECODAGE (postsync_verrou_decodage.go) : occupe = la passe se reporte au cycle suivant.
	liberer, ok := h.prendreLeVerrouDeDecodage(ctx, d)
	if !ok {
		publierArriere(d.TitleSlug, horsTravail+len(travail), nbSansCarte(reg, horsTravail+len(travail)))
		return 0
	}
	defer liberer()
	sum := col.CollectMatches(ctx, travail)
	observability.AddInt(CompteurPostSyncTraites, int64(sum.Written))
	filmbacklog.AjouterEcrits(d.TitleSlug, sum.Written)
	// CE QUI RESTE APRES LA PASSE : le hors-travail, plus ce que la passe n a pas atteint (budget de
	// temps, arret) — la jauge de retard decrit l etat de la base une fois la passe finie, pas celui
	// d avant.
	restant := horsTravail + max(0, len(travail)-sum.Traites())
	publierArriere(d.TitleSlug, restant, nbSansCarte(reg, restant))

	slog.InfoContext(ctx, "post-sync: kill source",
		"gamertag", d.Gamertag, "positions", col.CaptureCablee(),
		"demandes", len(travail), "ecrits", sum.Written,
		"morts", sum.Deaths, "films_absents", sum.NoFilm, "sans_killfeed", sum.NoKillFeed,
		"erreurs", sum.Errors, "ecartes_carte_non_resolue", sum.CarteNonResolue,
		"backlog_restant", restant,
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
