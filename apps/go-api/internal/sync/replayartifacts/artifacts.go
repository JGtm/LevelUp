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
// LE PAQUET PAR RESPONSABILITÉ (découpage du 2026-09-01) : artifacts.go décide QUOI faire
// (le hook, les dépendances, l'orchestration d'un cycle), backlog.go dit SUR QUOI (les insérés
// puis le rattrapage), cuisson.go fait le travail (pont disque, construction, mise en file),
// buildone.go délègue la construction hors du processus, mvar_rattrapage.go comble le catalogue
// de cartes, t0film.go reporte au registre le coup d'envoi mesuré dans le film, et journal.go
// dit ce que le cycle a fait.
//
// CE QUE FAIT L'ÉTAPE. Après runKillSource, l'étape 1.57 (le film vient d'être exploité pour
// la source des kills : c'est le moment où il est disponible ET récent — et l'étape 1.57 l'a
// au passage persisté au cache disque, donc celle-ci le trouve sans le retélécharger). Elle
// venait auparavant après runWeaponKills, l'étape 1.55, supprimée le 2026-09-01 avec la
// chaîne d'attribution par corrélation des tirs. Pour chaque match retenu dans la fenêtre
// replay_retention_months — les matchs INSÉRÉS du cycle d'abord, puis le RATTRAPAGE de la
// queue récente, parce que le film Theater se publie après la partie et qu'une tentative
// unique à l'instant de l'insertion ne rattrape jamais rien (cf. backlog.go) :
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
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
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
	// MvarFetcher est la capacite OPTIONNELLE qui rapatrie la variante d'une carte absente du
	// catalogue. Meme regime que `Fetcher` : nil desarme le rattrapage et ne casse rien — les
	// films sont recuperes et les artefacts construits comme avant, simplement sans que les
	// cartes inconnues entrent au catalogue.
	MvarFetcher MvarFetcher
	// WithRead ouvre un segment de LECTURE shared court — c'est le paquet sync qui porte
	// le lease et sa dégradation best-effort ; ici on ne fait que l'emprunter.
	WithRead func(ctx context.Context, step string, fn func(sharedDB *sql.DB))
	// AcquireWriter ouvre un segment d'ÉCRITURE shared COURT, acquis APRÈS la cuisson et
	// relâché aussitôt (cf. t0film.go) — jamais pendant un décodage. Même patron que le
	// burst du collecteur de kills (`killcollector.PostSyncDeps.AcquireWriter`).
	//
	// NIL = LE COUP D'ENVOI MESURÉ N'EST PAS REPORTÉ AU REGISTRE, et c'est journalisé : un
	// chemin de sync sans writer câblé cuit ses artefacts mais laisse `real_start_time` sur
	// le T0 estimé de l'API. Aucune autre partie de l'étape n'en dépend.
	AcquireWriter func(ctx context.Context) (*sql.DB, func(), error)
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
	// Budget : durée maximale d'une passe de cycle. 0 = BudgetParCycle (le contrat de
	// production) ; négatif = « déjà épuisé ». N'est renseigné que par les tests.
	Budget time.Duration
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
	if !armee(ctx, d) {
		return
	}
	titre := ctxkeys.TitleSlug(ctx)
	observability.IncCounterT(titre, CompteurCycles)
	work, retard := selectionnerLeTravail(ctx, d, insertedIDs)
	observability.AddIntT(titre, CompteurSelectionnes, int64(len(work)))
	// JAUGE, PAS COMPTEUR : ce qui reste à rattraper APRÈS ce cycle. Publiée MÊME À ZÉRO —
	// une clé absente de /debug/vars ne se distingue pas d'une étape qui ne tourne pas, et
	// c'est précisément l'ambiguïté que ce lot ferme.
	observability.SetIntT(titre, CompteurRetard, int64(retard))
	if len(work) == 0 {
		// SÉLECTION VIDE ≠ ÉTAPE MUETTE. Sans cette ligne, « la fenêtre de rétention a tout
		// écarté » et « l'étape n'a jamais tourné » s'écrivent pareil dans le journal : rien.
		slog.DebugContext(ctx, "post-sync: rejeu 2D — aucun match à traiter ce cycle",
			"gamertag", d.Gamertag, "inseres", len(insertedIDs), "retention_mois", d.RetentionMonths)
		return
	}
	// LE RATTRAPAGE DU CATALOGUE DE CARTES, AVANT TOUTE CUISSON et pour les DEUX placements.
	//
	// Il vit ici parce que c'est le dernier point de la chaine qui soit EN LIGNE : la cuisson,
	// qu'elle ait lieu tout de suite ou chez un ouvrier, est offline-pure. Mettre en file sans
	// avoir comble le catalogue produirait un artefact ampute que rien ne recuirait.
	//
	// Il ne peut pas faire echouer le cycle : voir mvar_rattrapage.go.
	rattraperCartesAbsentes(ctx, d, work, d.MvarFetcher)
	if d.Placement == replaybuild.PlacementWorker {
		enqueueAll(ctx, d, work)
		return
	}
	if len(work) > maxPerCycle {
		slog.InfoContext(ctx, "post-sync: rejeu 2D — lot borné, solde au cycle suivant",
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
	b := buildAll(ctx, d, work)
	// LE REPORT DU COUP D'ENVOI VIENT APRÈS TOUTE CUISSON, jamais entre deux : c'est ce qui
	// garantit que le burst writer ne recouvre aucun décodage (cf. t0film.go).
	reporterT0Film(ctx, d, b.t0Film)
	publierBilan(ctx, d, b, len(work))
}

// selectionnerLeTravail compose le lot du cycle et rend le retard qui subsiste après lui.
//
// DEUX ÉTAGES, ET L'ORDRE COMPTE.
//
//  1. LES MATCHS INSÉRÉS. Leur film vient d'être publié : c'est le moment le plus sûr pour le
//     télécharger, et le seul étage qui dispose des FAITS du match — d'où la réparation des
//     artefacts appauvris, qui lui reste attachée.
//  2. LE RATTRAPAGE, en complément et seulement s'il reste de la place dans le lot. Le film
//     Theater se publie APRÈS la partie : sans cet étage, un film arrivé en retard n'était
//     jamais repris (cf. backlog.go).
//
// LES DEUX SEGMENTS DE LECTURE N'EN FONT QU'UN : la base partagée est empruntée une fois, puis
// relâchée avant la moindre seconde de décodage (même règle que l'action admin).
func selectionnerLeTravail(ctx context.Context, d Deps, insertedIDs []string) (work []buildWork, retard int) {
	d.WithRead(ctx, "replay_select", func(sharedDB *sql.DB) {
		work = selectBuildWork(ctx, sharedDB, d.MetaDB, insertedIDs, d.RetentionMonths)
		deja := make(map[string]bool, len(work))
		for _, w := range work {
			deja[w.matchID] = true
		}
		// Le retard est TOUJOURS mesuré, même quand le lot est déjà plein : une jauge qu'on
		// ne rafraîchit que les bons jours ne décrit plus rien.
		rattrapage, restant := candidatsARattraper(ctx, sharedDB, d.MetaDB, d, deja, maxPerCycle-len(work))
		work = append(work, rattrapage...)
		retard = restant
		attachMatchFacts(ctx, sharedDB, work)
	})
	return work, retard
}

// armee dit si l'étape peut travailler, et DIT POURQUOI quand elle ne le peut pas.
//
// Les trois refus étaient muets. « Le réglage dit off », « aucun segment de lecture n'est
// câblé » et « le client ne sait pas lire un film » produisaient le même journal que « l'étape
// n'existe pas » : rien. Le niveau suit la nature du refus — une configuration se dit en
// DEBUG, un câblage manquant est un WARN.
func armee(ctx context.Context, d Deps) bool {
	switch {
	case d.Placement == replaybuild.PlacementOff:
		// Choix de configuration assumé. Les DÉGRADATIONS (« local » refusé en production,
		// « worker » sans file câblée) ont déjà averti dans Hook.Placement — les répéter ici
		// doublerait la ligne.
		slog.DebugContext(ctx, "post-sync: rejeu 2D éteint (replay_build_location)",
			"gamertag", d.Gamertag)
		return false
	case d.WithRead == nil:
		slog.WarnContext(ctx, "post-sync: rejeu 2D désarmée — aucun segment de lecture câblé",
			"gamertag", d.Gamertag)
		return false
	case d.Placement == replaybuild.PlacementLocal && d.Fetcher == nil:
		// Le câblage a déjà émis le WARN nominatif (SignalerClientSansChunks, qui connaît le
		// type du client) : ici on ne fait que tracer la sortie.
		slog.DebugContext(ctx, "post-sync: rejeu 2D — sans client film, rien à archiver ni à décoder",
			"gamertag", d.Gamertag)
		return false
	}
	return true
}

// publierBilan publie les compteurs du cycle et le résumé, MÊME QUAND RIEN N'A ÉTÉ CONSTRUIT.
//
// Le résumé était conditionné à `built > 0 || filmsSaved > 0`. Un cycle où les cinq films
// étaient expirés, ou où la cuisson échouait cinq fois, ne laissait donc aucune trace — le
// cas qu'il est le plus utile de voir. Même correction que celle déjà appliquée au chemin de
// mise en file.
func publierBilan(ctx context.Context, d Deps, b bilanCuisson, selectionnes int) {
	titre := ctxkeys.TitleSlug(ctx)
	observability.AddIntT(titre, CompteurConstruits, int64(b.construits))
	observability.AddIntT(titre, CompteurFilmsPersistes, int64(b.filmsSauves))
	observability.AddIntT(titre, CompteurDejaAJour, int64(b.dejaAJour))
	slog.InfoContext(ctx, "post-sync: rejeu 2D",
		"gamertag", d.Gamertag, "built", b.construits, "films_persisted", b.filmsSauves,
		"deja_a_jour", b.dejaAJour, "sans_film", b.sansFilm, "echecs", b.echecs,
		"budget_epuise", b.budgetEpuise, "selected", selectionnes)
}
