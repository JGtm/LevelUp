// Package replayartifacts — étape post-sync « artefacts de rejeu 2D » (fil de l'eau,
// lot 6 v7.5), extraite de la racine de internal/sync/ le 2026-08-14.
//
// POURQUOI UN SOUS-PACKAGE. Le ratchet archlint.TestSyncRootPackageFrozen (K3c, ADR 0027)
// gèle le nombre de fichiers à la racine de internal/sync/ : le neuf va dans un
// sous-package cohésif, jamais dans un nouveau fichier racine. Ce package est ce
// sous-package : il porte TOUTE la logique de l'étape (sélection du travail, résolution du
// nom EN de la carte, pont disque filmcache, boucle de construction) et prend ses
// dépendances en paramètres — il ne connaît ni SyncEngine ni aucun type privé du paquet
// sync.
//
// CE QUE FAIT L'ÉTAPE. Après runWeaponKills (le film vient d'être exploité pour les kills :
// c'est le moment où il est disponible ET récent), pour chaque match inséré dans la fenêtre
// replay_retention_months :
//
//  1. LE PONT DISQUE — les chunks COMPLETS du film (en-tête + réplication + kill-feed)
//     sont téléchargés et persistés au cache via filmcache.Write. C'est plus qu'un
//     préalable au décodage : les films EXPIRENT côté serveur Halo (~29 % du corpus déjà
//     perdus), et un film persisté est une archive IRREMPLAÇABLE.
//  2. L'ARTEFACT — replaybuild.BuildMatch décode le film du disque et écrit
//     data/cache/replays/{title}/{short8}.json. Le décodage est sérialisé par le verrou
//     process filmdec (partagé avec killsource).
//
// CONDITIONNÉ LOCAL : l'étape n'existe que si le wiring installe le Hook, et le wiring ne
// l'installe qu'en environnement non-production — « le VPS web ne décode JAMAIS » (le
// garde de service replay_local_gate protège la lecture ; ce gate-ci protège le CPU du
// VPS). Best-effort : aucun échec ne casse le pipeline post-sync.
package replayartifacts

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	settingsPkg "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/sync/haloclient"
)

// maxPerCycle borne le nombre d'artefacts construits par cycle : un sync initial insère
// des centaines de matchs, et décoder des centaines de films dans le post-sync bloquerait
// le cycle pendant des heures. Le solde relève du backfill CLI (levelup backfill-replay),
// idempotent et repris par SchemaVersion.
const maxPerCycle = 5

// EnqueueFunc met la construction du rejeu d'un match dans la FILE DURABLE (le
// travail part alors à un ouvrier, éventuellement sur une autre machine).
// Implémentée par wire.ServiceRegistry.EnqueueReplayBuild.
type EnqueueFunc func(ctx context.Context, titleSlug, matchID string) error

// Hook : réglage du fil de l'eau replay, injecté au wiring.
//
// LE HOOK S'INSTALLE TOUJOURS ; C'EST LUI QUI DÉCIDE. Avant, chaque site de
// wiring portait un `if !IsProduction()` — trois copies d'une règle qui a
// désormais trois valeurs. Le lieu de construction se relit à CHAQUE cycle
// (patron scheduler : un PATCH /settings prend effet sans redémarrage), et la
// décision elle-même vit à UN seul endroit : replaybuild.DecidePlacement.
type Hook struct {
	// RetentionMonths relit la fenêtre à CHAQUE cycle. 0 = illimité.
	RetentionMonths func() int
	// Location relit le réglage brut « où se construit un rejeu » à chaque cycle.
	Location func() string
	// Env : ce que l'instance sait d'elle-même (production, ouvrier ouvert).
	Env replaybuild.PlacementEnv
	// Enqueue : la mise en file, pour le placement « worker ». Nil = pas de file
	// câblée sur ce chemin (le placement dégrade alors en « off », journalisé).
	Enqueue EnqueueFunc
	// CacheRoot : racine du cache film (PathResolver.CacheRootDir).
	CacheRoot string
	// BuildOne : la strategie de cuisson, cablee par NewHook (cf. buildone.go). Elle voyage
	// avec le hook pour que les DEUX sites de wiring la recoivent sans la redeclarer.
	BuildOne BuildOneFunc
}

// SettingsReader : la lecture des réglages vivants dont le hook a besoin.
// Satisfaite par *settings.Store — interface plutôt que type concret pour que ce
// sous-package reste testable sans fichier de configuration.
type SettingsReader interface {
	Load() (*settingsPkg.AppSettings, error)
}

// NewHook construit le hook du fil de l'eau replay — LA fabrique unique des sites de
// wiring (BuildEngine scheduler, factory V2, handler HTTP legacy), pour que ni le cache
// root, ni la lecture des réglages, ni l'environnement ne se résolvent à trois endroits.
//
// enqueue peut être nil : un chemin de sync qui n'a pas accès à la file dégrade en
// « aucune construction » plutôt qu'en construction locale silencieuse.
func NewHook(cfg *config.AppConfig, store SettingsReader, enqueue EnqueueFunc) *Hook {
	load := func() *settingsPkg.AppSettings {
		if store == nil {
			return nil
		}
		s, _ := store.Load()
		return s
	}
	return &Hook{
		RetentionMonths: func() int {
			if s := load(); s != nil {
				return s.ReplayRetentionMonths
			}
			return 0
		},
		Location: func() string {
			if s := load(); s != nil {
				return s.ReplayBuildLocation
			}
			return ""
		},
		Env: replaybuild.PlacementEnv{
			Production:       cfg.IsProduction(),
			WorkerConfigured: strings.TrimSpace(cfg.BuildWorkerToken) != "",
		},
		Enqueue:   enqueue,
		CacheRoot: titlePkg.NewPathResolver(cfg.RepoRoot).CacheRootDir(),
		// LA CUISSON PART DANS UN ENFANT BORNE, TOUJOURS (lot BUILDALL) : le serveur ne
		// decode plus un film dans son propre processus.
		BuildOne: SpawnBuildOne,
	}
}

// Months rend la fenêtre de rétention du cycle courant (0 = illimité), hook ou closure nils
// compris.
func (h *Hook) Months() int {
	if h == nil || h.RetentionMonths == nil {
		return 0
	}
	return h.RetentionMonths()
}

// Placement rend la décision du cycle courant. Un hook sans mise en file câblée ne
// peut pas tenir « worker » : il dégrade en « off », jamais en construction locale
// (le VPS web ne décode jamais, et un repli silencieux le lui ferait faire).
func (h *Hook) Placement() replaybuild.Placement {
	if h == nil {
		return replaybuild.PlacementOff
	}
	setting := ""
	if h.Location != nil {
		setting = strings.TrimSpace(h.Location())
	}
	p, err := replaybuild.DecidePlacement(setting, h.Env)
	replaybuild.LogPlacement("post-sync", p, err)
	if p == replaybuild.PlacementWorker && h.Enqueue == nil {
		slog.Warn("rejeu 2D : mise en file demandée mais aucune file câblée sur ce chemin de sync — aucune construction")
		return replaybuild.PlacementOff
	}
	return p
}

// ChunksFetcher : capacité OPTIONNELLE du client de sync (assertion côté appelant, pas
// extension de l'interface HaloClient — les mocks des autres étapes n'ont pas à la porter).
type ChunksFetcher interface {
	GetFilmChunks(ctx context.Context, matchID string) ([]haloclient.FilmChunk, bool, error)
}

// Deps : tout ce dont l'étape a besoin, passé à plat par le paquet sync (le sous-package ne
// voit aucun type privé du moteur).
type Deps struct {
	// Fetcher lit les chunks du film (cache-first chunk par chunk).
	Fetcher ChunksFetcher
	// WithRead ouvre un segment de LECTURE shared court — c'est le paquet sync qui porte
	// le lease et sa dégradation best-effort ; ici on ne fait que l'emprunter.
	WithRead func(ctx context.Context, step string, fn func(sharedDB *sql.DB))
	// MetaDB (optionnel) résout le nom EN de la carte. Nil = map_name brut seul.
	MetaDB *sql.DB
	// RepoRoot, TitleSlug : résolution des chemins d'artefact et du catalogue du titre.
	RepoRoot  string
	TitleSlug string
	// Gamertag : identité journalisée (aucun rôle fonctionnel).
	Gamertag string
	// CacheRoot : racine du cache film (Hook.CacheRoot).
	CacheRoot string
	// RetentionMonths : fenêtre de sélection, <= 0 = illimitée (Hook.Months()).
	RetentionMonths int
	// Placement : la décision du cycle (Hook.Placement()).
	Placement replaybuild.Placement
	// Enqueue : la mise en file, utilisée seulement si Placement == worker.
	Enqueue EnqueueFunc
	// BuildOne construit l'artefact d'un film HORS DU PROCESSUS et rend ses octets (cf.
	// buildone.go). Le serveur la câble au boot.
	//
	// NIL = AUCUNE CONSTRUCTION, ET C'EST UNE DÉGRADATION VOULUE, PAS UN REPLI SILENCIEUX. Le
	// pont disque continue (les films sont persistés — ils EXPIRENT côté serveur Halo, c'est
	// l'irremplaçable) ; seule la cuisson est sautée, et elle est journalisée. Décoder
	// in-process « pour ne pas perdre le cycle » est exactement ce que le lot BUILDALL
	// supprime : c'est ainsi que la machine est morte trois fois.
	BuildOne BuildOneFunc
}

// buildWork : un match à construire, avec ses identités de carte candidates et les faits que
// seule la base connaît (lignes de match, scores des deux camps, nom de variante).
type buildWork struct {
	matchID  string
	mapNames []string
	facts    port.MatchFacts
}

// attachMatchFacts lit, DANS LE MÊME SEGMENT DE LECTURE que la sélection, ce que la base sait
// de chaque match du lot — le pont d'identité des joueurs et celui des camps.
//
// POURQUOI ICI ET PAS AU MOMENT DE CONSTRUIRE : le décodage d'un film dure des secondes à des
// minutes, et rien ne doit tenir un handle partagé pendant ce temps-là (même règle que l'action
// admin). Deux SELECT courts par match, puis la base est relâchée.
//
// Une lecture qui échoue DÉGRADE ce match seul, journalisée : l'artefact sortira sans compteurs
// de joueur ni actions d'objectif, ce qui vaut mieux que pas d'artefact du tout.
func attachMatchFacts(ctx context.Context, sharedDB *sql.DB, work []buildWork) {
	if sharedDB == nil {
		return
	}
	var repo port.ReplayFactsRepo = duckdbpkg.NewReplayFactsRepo(sharedDB)
	for i := range work {
		facts, err := repo.FactsForMatch(ctx, work[i].matchID)
		if err != nil {
			slog.WarnContext(ctx, "post-sync: faits de match illisibles — rejeu sans courbe de score complète",
				"err", err, "match_id", work[i].matchID)
			continue
		}
		work[i].facts = facts
	}
}

// etatArtefact dit ce que vaut l'artefact déjà sur disque pour ce match : est-il à la version
// COURANTE, et porte-t-il ce que les faits permettent d'y mettre.
//
// UNE SEULE ÉCRITURE DE LA RÈGLE POUR LES DEUX CHEMINS post-sync (mise en file et construction
// locale). En deux exemplaires elle aurait divergé au premier ajustement, et un chemin serait
// resté à figer des artefacts appauvris pendant que l'autre les répare.
//
// `complet` est une PRÉSOMPTION, pas une preuve : sans lignes de match, il n'y a rien de mieux
// à espérer d'une reconstruction, donc l'artefact est réputé complet ; avec des lignes, l'absence
// de compteurs de joueur le fait présumer appauvri (cf. l'en-tête d'ArtifactHasPlayerCounters
// pour les trois vacuités légitimes que cette présomption peut confondre).
func etatArtefact(path string, facts port.MatchFacts) (aJour, complet bool) {
	if !replaybuild.ArtifactUpToDate(path) {
		return false, false
	}
	return true, len(facts.Players) == 0 || replaybuild.ArtifactHasPlayerCounters(path)
}

// Run — étape post-sync 1.58 : selon le lieu de construction réglé, ou bien pont
// disque + construction locale des artefacts, ou bien MISE EN FILE des matchs
// insérés (l'ouvrier fera les deux chez lui), ou bien rien.
//
// LA FENÊTRE DE RÉTENTION S'APPLIQUE AVANT LES DEUX : on n'enfile pas ce que la
// purge effacera, et un travail hors fenêtre coûterait un décodage pour rien.
// Best-effort de bout en bout : aucun retour, aucune erreur ne remonte au cycle.
func Run(ctx context.Context, d Deps, insertedIDs []string) {
	if d.Placement == replaybuild.PlacementOff || d.WithRead == nil || len(insertedIDs) == 0 {
		return
	}
	if d.Placement == replaybuild.PlacementLocal && d.Fetcher == nil {
		return // sans client film, rien à télécharger ni à décoder ici
	}
	var work []buildWork
	d.WithRead(ctx, "replay_select", func(sharedDB *sql.DB) {
		work = selectBuildWork(ctx, sharedDB, d.MetaDB, insertedIDs, d.RetentionMonths)
		attachMatchFacts(ctx, sharedDB, work)
	})
	if len(work) == 0 {
		return
	}
	if d.Placement == replaybuild.PlacementWorker {
		enqueueAll(ctx, d, work)
		return
	}
	if len(work) > maxPerCycle {
		slog.InfoContext(ctx, "post-sync: rejeu 2D — lot borné, solde au backfill CLI",
			"gamertag", d.Gamertag, "selected", len(work), "cap", maxPerCycle)
		work = work[:maxPerCycle]
	}
	// SONDE DE TITRE, PAS UN CONSTRUCTEUR : depuis le lot BUILDALL la cuisson se fait dans un
	// ENFANT, qui monte le sien. Ce que cet appel garde, c'est la degradation par ABSENCE de
	// donnee — un titre sans catalogue de bornes ni libelles ne doit pas faire naitre sept
	// processus pour sept echecs de preparation.
	if _, err := replaybuild.NewBuilder(d.RepoRoot, d.TitleSlug); err != nil {
		// Titre sans catalogue de bornes / labels : dégradation par absence de donnée
		// (title-agnostic), journalisée une fois par cycle — jamais un échec de sync.
		slog.DebugContext(ctx, "post-sync: rejeu 2D indisponible pour ce titre",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug, "err", err)
		return
	}
	built, filmsSaved := buildAll(ctx, d, work)
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "postsync_replay_artifacts_built_total", int64(built))
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "postsync_replay_films_persisted_total", int64(filmsSaved))
	if built > 0 || filmsSaved > 0 {
		slog.InfoContext(ctx, "post-sync: rejeu 2D",
			"gamertag", d.Gamertag, "built", built, "films_persisted", filmsSaved, "selected", len(work))
	}
}

// enqueueAll met les matchs du lot dans la file durable — le chemin du VPS web,
// qui ne décode JAMAIS.
//
// AUCUN PONT DISQUE ICI, et c'est la différence de fond avec le chemin local :
// c'est la MISE EN FILE qui résout le manifeste et dépose les URL pré-signées
// (wire.EnqueueReplayBuild), et c'est l'ouvrier qui téléchargera les morceaux. Le
// web ne fait donc transiter aucun film.
//
// Le lot n'est pas borné comme le local (maxPerCycle) : enfiler coûte une
// résolution de manifeste, pas 50 s de CPU. La file, elle, est faite pour
// s'allonger — un ouvrier absent ne casse rien.
func enqueueAll(ctx context.Context, d Deps, work []buildWork) {
	if d.Enqueue == nil {
		return
	}
	paths := titlePkg.NewPathResolver(d.RepoRoot)
	queued, skipped, appauvris, refuses := 0, 0, 0, 0
	for _, w := range work {
		if ctx.Err() != nil {
			break
		}
		pourAppauvrissement := false
		// Idempotence : un artefact déjà à jour ne se reconstruit pas (même règle
		// que le chemin local ; la mise en file absorbe de son côté les doublons).
		//
		// « À JOUR » NE SE RÉSUME PAS À LA VERSION DE SCHÉMA : un artefact construit sans les
		// faits porte le bon numéro tout en étant APPAUVRI. Le sauter sur le seul critère de
		// version le figerait à demeure — c'est ainsi qu'un ouvrier sans faits empoisonnerait
		// le cache. La règle, et le pourquoi de son critère, vivent dans `etatArtefact`.
		artefact := paths.ReplayArtifactPath(d.TitleSlug, w.matchID)
		aJour, complet := etatArtefact(artefact, w.facts)
		if aJour {
			if complet {
				skipped++
				continue
			}
			// Jamais muet : un artefact re-enfilé alors qu'il paraît à jour doit s'expliquer.
			//
			// PRÉSOMPTION, PAS PREUVE : l'artefact peut être vide de compteurs pour une raison
			// légitime (film sans enregistrement d'entité, appariement ambigu, aucun compteur
			// dans la fenêtre — cf. l'en-tête d'ArtifactHasPlayerCounters). La re-cuisson rendra
			// alors le même document. Le résidu est BORNÉ des deux côtés : en fréquence, parce
			// que la sélection ne voit que les matchs INSÉRÉS du cycle (un match ne repasse pas
			// ici à chaque sync) ; en dégâts, parce que `StoreArtifact` refuse toute régression.
			// Le pire cas est donc UN cycle d'ouvrier gâché, jamais un artefact rétrogradé.
			slog.InfoContext(ctx, "post-sync: rejeu 2D — artefact au bon schéma mais SANS compteurs de joueur, remis en file",
				"gamertag", d.Gamertag, "match_id", w.matchID, "lignes_de_match", len(w.facts.Players))
			pourAppauvrissement = true
		}
		if err := d.Enqueue(ctx, d.TitleSlug, w.matchID); err != nil {
			// Cas nominal du refus : film absent ou expiré côté serveur (~29 % du
			// corpus). Journalisé en debug, jamais avalé.
			slog.DebugContext(ctx, "post-sync: rejeu 2D non mis en file",
				"gamertag", d.Gamertag, "match_id", w.matchID, "err", err)
			refuses++
			continue
		}
		queued++
		// Le compteur ne s'incrémente qu'APRÈS la mise en file effective : un film expiré
		// (~29 % du corpus) est refusé ici même, et le compter comme « re-enfilé » ferait
		// sur-déclarer la métrique de tout ce que la file n'a jamais reçu.
		if pourAppauvrissement {
			appauvris++
		}
	}
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "postsync_replay_jobs_enqueued_total", int64(queued))
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "postsync_replay_artifacts_factless_requeued_total", int64(appauvris))
	// Le résumé sort dès qu'il y a eu du travail à examiner. Le conditionner à
	// `queued > 0 || skipped > 0` rendait MUET le cas où tout a été refusé (film expiré sur
	// tout le lot) : un cycle entier sans la moindre trace au niveau INFO.
	if len(work) > 0 {
		slog.InfoContext(ctx, "post-sync: rejeu 2D mis en file (construction déléguée à un ouvrier)",
			"gamertag", d.Gamertag, "queued", queued, "deja_a_jour", skipped,
			"appauvris_re_enfiles", appauvris, "refuses", refuses, "selected", len(work))
	}
}

// buildAll persiste le film puis construit l'artefact de chaque match du lot. Rend
// (artefacts construits, films persistés).
func buildAll(ctx context.Context, d Deps, work []buildWork) (built, filmsSaved int) {
	paths := titlePkg.NewPathResolver(d.RepoRoot)
	// LE CAS « PAS DE CONSTRUCTION CÂBLÉE » SE DIT UNE FOIS PAR CYCLE, pas une fois par film :
	// c'est un état de configuration, pas un incident de match. Le pont disque, lui, continue —
	// un film persisté est irremplaçable (ils EXPIRENT côté serveur Halo).
	if d.BuildOne == nil {
		slog.WarnContext(ctx, "post-sync: aucune construction hors processus câblée — films persistés, cuisson SAUTÉE",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug, "selectionnes", len(work))
	}
	for _, w := range work {
		if ctx.Err() != nil {
			break
		}
		saved, ok := persistFilmToCache(ctx, d, w.matchID)
		if !ok {
			continue // film absent/expiré côté serveur : rien à construire (débité en debug)
		}
		if saved {
			filmsSaved++
		}
		// MÊME RÈGLE QUE LA MISE EN FILE : la version de schéma ne suffit pas. Un artefact
		// appauvri déposé par un ouvrier d'avant le transport des faits porte le bon numéro ;
		// le sauter ici le figerait, sur le chemin même qui a les faits sous la main pour le
		// réparer (ils sont passés à BuildMatch quinze lignes plus bas).
		if d.BuildOne == nil {
			continue // avertissement deja emis une fois pour le cycle
		}
		aJour, complet := etatArtefact(paths.ReplayArtifactPath(d.TitleSlug, w.matchID), w.facts)
		if aJour && complet {
			continue
		}
		if aJour {
			slog.InfoContext(ctx, "post-sync: rejeu 2D — artefact au bon schéma mais SANS compteurs de joueur, reconstruit",
				"gamertag", d.Gamertag, "match_id", w.matchID, "lignes_de_match", len(w.facts.Players))
		}
		short := titlePkg.FilmShortMatchID(w.matchID)
		// LA CUISSON PART HORS DU PROCESSUS (lot BUILDALL, 2026-08-26) : l'enfant décode et rend
		// les OCTETS, le serveur les range. Le garde anti-régression et la notification restent
		// dans `StoreArtifact`, donc exactement où ils étaient.
		out, berr := buildAndStoreOne(ctx, d, w, filmcache.ChunkDir(d.CacheRoot, short))
		if berr != nil {
			// Carte hors catalogue = échec voulu (Forge) ; le reste = erreur réelle.
			// Les deux sont best-effort, mais seuls les seconds méritent un WARN.
			logFn := slog.WarnContext
			if strings.Contains(berr.Error(), replaybuild.ErrMapNotInCatalog.Error()) {
				logFn = slog.DebugContext
			}
			logFn(ctx, "post-sync: artefact rejeu non construit",
				"gamertag", d.Gamertag, "match_id", w.matchID, "err", berr)
			continue
		}
		built++
		slog.InfoContext(ctx, "post-sync: artefact rejeu construit",
			"gamertag", d.Gamertag, "match_id", w.matchID, "tracks", out.Tracks, "bytes", out.Bytes)
	}
	return built, filmsSaved
}

// persistFilmToCache télécharge les chunks COMPLETS du film et les persiste au cache
// (pont disque). Rend (persisté, film disponible). Un film déjà entièrement en cache ne
// re-télécharge rien (GetFilmChunks est cache-first chunk par chunk).
func persistFilmToCache(ctx context.Context, d Deps, matchID string) (saved, available bool) {
	chunks, found, err := d.Fetcher.GetFilmChunks(ctx, matchID)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: film illisible — rejeu non construit",
			"gamertag", d.Gamertag, "match_id", matchID, "err", err)
		return false, false
	}
	if !found || len(chunks) == 0 {
		slog.DebugContext(ctx, "post-sync: film absent côté serveur — rejeu non construit",
			"match_id", matchID)
		return false, false
	}
	wc := make([]filmcache.WriteChunk, 0, len(chunks))
	for _, c := range chunks {
		wc = append(wc, filmcache.WriteChunk{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS,
			DurationMS: c.DurationMS, Data: c.Data,
		})
	}
	if err := filmcache.Write(d.CacheRoot, titlePkg.FilmShortMatchID(matchID), wc); err != nil {
		slog.WarnContext(ctx, "post-sync: persistance du film au cache échouée",
			"gamertag", d.Gamertag, "match_id", matchID, "err", err)
		return false, false
	}
	return true, true
}

// selectBuildWork lit les identités de carte des matchs insérés et applique la fenêtre de
// rétention (months <= 0 = illimité). metaDB peut être nil (pas de résolution EN : map_name
// brut seul, même dégradation que le backfill CLI).
func selectBuildWork(
	ctx context.Context, sharedDB, metaDB *sql.DB, insertedIDs []string, months int,
) []buildWork {
	if len(insertedIDs) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(insertedIDs)), ",")
	args := make([]any, 0, len(insertedIDs))
	for _, id := range insertedIDs {
		args = append(args, id)
	}
	q := fmt.Sprintf(`SELECT match_id, map_name, map_id, %s AS start_canonical
		FROM match_registry WHERE match_id IN (%s)`,
		analysis.SQLStartTimeCanonical("match_registry"), placeholders)
	rows, err := sharedDB.QueryContext(ctx, q, args...)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: sélection rejeu échouée", "err", err)
		return nil
	}
	defer func() { _ = rows.Close() }()

	var cutoff time.Time
	if months > 0 {
		cutoff = time.Now().UTC().AddDate(0, -months, 0)
	}
	var out []buildWork
	for rows.Next() {
		var id string
		var rawName, mapID sql.NullString
		var start sql.NullTime
		if err := rows.Scan(&id, &rawName, &mapID, &start); err != nil {
			slog.WarnContext(ctx, "post-sync: sélection rejeu (scan)", "err", err)
			return out
		}
		if months > 0 && start.Valid && start.Time.Before(cutoff) {
			continue // hors fenêtre de rétention : le backfill CLI reste libre de le faire
		}
		var names []string
		if en := resolveMapNameEN(ctx, metaDB, strings.TrimSpace(mapID.String)); en != "" {
			names = append(names, en)
		}
		if raw := strings.TrimSpace(rawName.String); raw != "" {
			names = append(names, raw)
		}
		out = append(out, buildWork{matchID: id, mapNames: names})
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "post-sync: sélection rejeu (rows)", "err", err)
	}
	return out
}

// resolveMapNameEN résout le nom EN d'une carte par son asset UGC (asset_translations).
// Best-effort : metaDB nil ou nom absent → "" (le candidat brut reste).
func resolveMapNameEN(ctx context.Context, metaDB *sql.DB, mapID string) string {
	if metaDB == nil || mapID == "" {
		return ""
	}
	var en string
	err := metaDB.QueryRowContext(ctx,
		`SELECT name FROM asset_translations WHERE asset_type = 'map' AND asset_id = ? AND lang = 'en-US'`,
		mapID).Scan(&en)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(en)
}
