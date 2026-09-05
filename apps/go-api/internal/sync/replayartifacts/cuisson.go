package replayartifacts

// cuisson.go — CE QUE LE CYCLE FAIT D'UN LOT : archiver le film, puis construire l'artefact,
// ou bien le confier à la file durable d'un ouvrier.
//
// Extrait d'artifacts.go le 2026-09-01, quand le rattrapage a poussé le fichier d'orchestration
// au-delà du seuil projet. Le découpage suit les responsabilités : artifacts.go décide QUOI
// faire, backlog.go dit SUR QUOI, ce fichier-ci fait le travail.

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"
)

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
	debut, budget := time.Now(), budgetDuCycle(d)
	queued, skipped, appauvris, refuses := 0, 0, 0, 0
	for _, w := range work {
		if ctx.Err() != nil {
			break
		}
		// Enfiler coûte une résolution de manifeste — donc un aller-retour RÉSEAU par match.
		// Le même budget s'applique ici : la file est faite pour s'allonger, pas le cycle.
		if time.Since(debut) >= budget {
			slog.InfoContext(ctx, "post-sync: rejeu 2D — budget de cycle épuisé, solde au cycle suivant",
				"gamertag", d.Gamertag, "budget", budget, "enfiles", queued)
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
			// dans la fenêtre — cf. l'en-tête de `replaybuild.Digest.HasPlayerCounters`). La re-cuisson rendra
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
	observability.AddIntT(ctxkeys.TitleSlug(ctx), CompteurEnfiles, int64(queued))
	observability.AddIntT(ctxkeys.TitleSlug(ctx), CompteurDejaAJour, int64(skipped))
	observability.AddIntT(ctxkeys.TitleSlug(ctx), CompteurAppauvrisReEnfiles, int64(appauvris))
	// Le résumé sort dès qu'il y a eu du travail à examiner. Le conditionner à
	// `queued > 0 || skipped > 0` rendait MUET le cas où tout a été refusé (film expiré sur
	// tout le lot) : un cycle entier sans la moindre trace au niveau INFO.
	if len(work) > 0 {
		slog.InfoContext(ctx, "post-sync: rejeu 2D mis en file (construction déléguée à un ouvrier)",
			"gamertag", d.Gamertag, "queued", queued, "deja_a_jour", skipped,
			"appauvris_re_enfiles", appauvris, "refuses", refuses, "selected", len(work))
	}
}

// bilanCuisson : ce que le cycle a fait, et POURQUOI il n'a rien fait quand c'est le cas.
//
// Un simple couple (construits, films) ne distingue pas « tout était déjà à jour » de « tous
// les films sont expirés » ni de « la cuisson a échoué cinq fois » — trois situations qui
// appellent trois actions différentes et qui s'écrivaient toutes « 0 ».
type bilanCuisson struct {
	construits   int
	filmsSauves  int
	dejaAJour    int
	sansFilm     int
	echecs       int
	budgetEpuise bool
	// t0Film : les coups d'envoi mesurés par les artefacts CUITS DANS CE CYCLE, à reporter au
	// registre une fois toute cuisson terminée (cf. t0film.go). Ils voyagent dans le bilan
	// plutôt que d'être écrits ici : un burst writer au milieu d'une boucle de décodage est
	// exactement ce que le découpage du paquet interdit.
	t0Film []rapportT0Film
	// usage : les artefacts cuits dans ce cycle, à projeter (résumé d'usage, statistiques
	// d'Assaut) puis à écrire une fois TOUTE cuisson terminée (cf. usage.go, bombstats.go)
	// — même règle de voyage que t0Film.
	usage []artefactCuit
}

// DeadlineParFilm : la borne DURE de la cuisson d'UN film, quand le budget du cycle en laisse
// autant.
//
// POURQUOI ELLE EXISTE (PLAN_CUISSON_PERF item 5.5). Le budget de cycle s'applique ENTRE deux
// matchs : il ne peut rien contre un enfant qui ne rend jamais la main. Un enfant bloqué —
// spirale GC sous le plafond souple, disque qui ne répond plus, processus suspendu par l'OS —
// tenait donc le cycle de synchronisation INDÉFINIMENT, et avec lui tout ce qui vient après
// (PSA, agrégats, médias). La deadline coupe l'enfant, le film compte en échec, et le cycle
// continue : c'est la même doctrine que le protocole de codes de sortie — la santé de la passe
// ne dépend jamais de la santé d'un film.
//
// QUINZE MINUTES, ET C'EST LARGE À DESSEIN : le film le plus cher du corpus se cuit en moins de
// deux minutes (mesures §6 du plan). Cette borne n'est pas un réglage de performance, c'est un
// dernier rempart — la couper court transformerait un film lent en échec, ce que personne ne
// demande.
const DeadlineParFilm = 15 * time.Minute

// PlancherCuisson : le solde de budget SOUS LEQUEL AUCUNE CUISSON NE SE LANCE.
//
// POURQUOI IL EXISTE (revue du lot 6, constat 6.3). La borne d'un film vaut
// `min(solde, DeadlineParFilm)` et n'avait pas de plancher : un solde de deux secondes — ou nul —
// donnait à la cuisson un contexte DÉJÀ EXPIRÉ. L'enfant était tué aussitôt né, le film comptait
// en ÉCHEC, et un WARN « artefact rejeu non construit » accusait le décodage d'une panne qui n'en
// était pas une. C'est exactement ce que la doctrine de ce fichier interdit : LE BUDGET S'APPLIQUE
// ENTRE DEUX MATCHS, JAMAIS AU MILIEU D'UN — et une cuisson lancée pour être tuée trois lignes
// plus loin, c'est le milieu d'un match. Sous le plancher, le cycle REPORTE (Info, pas Warn : un
// report est nominal) et le match revient au cycle suivant, l'étape étant idempotente.
//
// TRENTE SECONDES, ET C'EST MESURÉ : après le lot 4, un film témoin se cuit en 15 à 19 s
// (`MESURES_CUISSON_PERF.md` §1 : 15,7 / 18,6 / 18,2 s). Trente secondes laissent donc passer la
// cuisson médiane et refusent celles qui n'ont plus la place de finir. Le pire film sain du corpus
// (BTB à 26 joueurs, ~1 min 40) ne rentre dans aucun plancher raisonnable : le reporter au cycle
// suivant EST le bon comportement, et sa borne dure reste [DeadlineParFilm]. La valeur est un
// dixième de [BudgetParCycle] — le cycle ne se prive que de sa dernière tranche.
const PlancherCuisson = 30 * time.Second

// deadlineDuFilm rend la borne effective d'UNE cuisson : le minimum du solde de budget et de
// [DeadlineParFilm]. Rien ne sert de laisser un enfant courir vingt minutes quand le cycle
// s'arrête dans trois.
//
// LE SOLDE REÇU N'EST JAMAIS SOUS [PlancherCuisson] : l'appelant a déjà refusé de lancer la
// cuisson en dessous (cf. `buildAll`). Cette fonction ne rend donc plus jamais un délai
// dérisoire — un contexte expiré à la naissance était un échec compté pour rien.
func deadlineDuFilm(d Deps, restant time.Duration) time.Duration {
	max := DeadlineParFilm
	if d.DeadlineParFilm > 0 {
		max = d.DeadlineParFilm
	}
	if restant < max {
		return restant
	}
	return max
}

// buildAll persiste le film puis construit l'artefact de chaque match du lot.
func buildAll(ctx context.Context, d Deps, work []buildWork) bilanCuisson {
	var b bilanCuisson
	debut, budget := time.Now(), budgetDuCycle(d)
	// LE PONT DISQUE PRECHARGE LE FILM SUIVANT pendant la cuisson du courant (cf. prefetch.go).
	// `fermer` est différé : aucune goroutine de téléchargement ne survit au cycle, quelle que
	// soit la sortie de la boucle (budget, contexte annulé, lot épuisé).
	pont := &pontDisque{d: d}
	defer pont.fermer()
	// LE CAS « PAS DE CONSTRUCTION CÂBLÉE » SE DIT UNE FOIS PAR CYCLE, pas une fois par film :
	// c'est un état de configuration, pas un incident de match. Le pont disque, lui, continue —
	// un film persisté est irremplaçable (ils EXPIRENT côté serveur Halo).
	if d.BuildOne == nil {
		slog.WarnContext(ctx, "post-sync: aucune construction hors processus câblée — films persistés, cuisson SAUTÉE",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug, "selectionnes", len(work))
	}
	for i, w := range work {
		if ctx.Err() != nil {
			break
		}
		// LE BUDGET S'APPLIQUE ENTRE DEUX MATCHS, jamais au milieu d'un : couper une cuisson
		// en cours ne rendrait qu'un artefact tronqué. Le solde repart au cycle suivant.
		// (Le solde RÉSIDUEL, lui, est jugé plus bas contre [PlancherCuisson] : à zéro on
		// s'arrête ici, sous le plancher on s'arrête juste avant de cuire.)
		if time.Since(debut) >= budget {
			b.budgetEpuise = true
			slog.InfoContext(ctx, "post-sync: rejeu 2D — budget de cycle épuisé, solde au cycle suivant",
				"gamertag", d.Gamertag, "budget", budget, "traites", b.construits+b.dejaAJour+b.sansFilm+b.echecs)
			break
		}
		film := pont.film(ctx, w.matchID)
		// LE PRECHARGEMENT PART AVANT LA CUISSON, ET AVANT TOUT `continue` : c'est pendant les
		// dizaines de secondes de décodage que le lien est libre. Un match sans film ou déjà à
		// jour ne doit pas priver le SUIVANT de son avance.
		if i+1 < len(work) {
			pont.precharger(ctx, work[i+1].matchID, budget-time.Since(debut))
		}
		if !film.dispo {
			b.sansFilm++
			continue // film absent/expiré côté serveur : rien à construire (débité en debug)
		}
		if film.sauve {
			b.filmsSauves++
		}
		// SOUS LE PLANCHER, ON REPORTE — ON NE CUIT PAS POUR SE FAIRE TUER. Le solde se mesure
		// ICI, après le pont disque : télécharger a pris du temps, et ce qu'il reste ne suffit
		// peut-être plus. Lancer quand même donnerait un contexte expiré à la naissance, donc un
		// ÉCHEC compté et un WARN pour une panne qui n'existe pas (cf. [PlancherCuisson]).
		//
		// LE FILM, LUI, EST DÉJÀ PERSISTÉ, et c'est l'essentiel : il EXPIRE côté serveur Halo,
		// l'artefact non. Le cycle suivant reprendra ce match avec son film déjà en cache.
		//
		// LE PLANCHER NE S'APPLIQUE QUE SI UNE CUISSON EST CÂBLÉE. Sans `BuildOne`, cette boucle
		// n'est plus qu'un pont disque : il n'y a AUCUNE cuisson à protéger d'une deadline
		// dérisoire, et s'arrêter tôt ne ferait que perdre des films qui expirent. Le budget
		// PLEIN (garde d'entrée de boucle) reste alors la seule borne.
		solde := budget - time.Since(debut)
		if d.BuildOne != nil && solde < PlancherCuisson {
			b.budgetEpuise = true
			slog.InfoContext(ctx, "post-sync: rejeu 2D — solde de budget sous le plancher de cuisson, match reporté au cycle suivant",
				"gamertag", d.Gamertag, "match_id", w.matchID, "solde", solde,
				"plancher", PlancherCuisson,
				"traites", b.construits+b.dejaAJour+b.sansFilm+b.echecs)
			break
		}
		cuireUnMatch(ctx, d, w, &b, solde)
	}
	return b
}

// cuireUnMatch décide si CE match se re-cuit, le cuit sous deadline, et compte le résultat.
//
// EXTRAITE DE `buildAll` LE 2026-09-03 (items 5.5/5.6) : la boucle porte désormais le pont
// disque et son préchargement, la cuisson d'un match porte sa deadline. Les deux tenaient
// ensemble au-delà des 80 lignes du dépôt, et elles ne répondent pas à la même question.
//
// `restant` VAUT AU MOINS [PlancherCuisson] : c'est la boucle qui refuse d'appeler ici quand le
// solde ne suffit plus (constat 6.3). Cette fonction n'a donc jamais à se demander si sa deadline
// est tenable — elle l'est par construction.
func cuireUnMatch(ctx context.Context, d Deps, w buildWork, b *bilanCuisson, restant time.Duration) {
	// MÊME RÈGLE QUE LA MISE EN FILE : la version de schéma ne suffit pas. Un artefact
	// appauvri déposé par un ouvrier d'avant le transport des faits porte le bon numéro ;
	// le sauter ici le figerait, sur le chemin même qui a les faits sous la main pour le
	// réparer (ils sont passés à la cuisson quelques lignes plus bas).
	if d.BuildOne == nil {
		return // avertissement deja emis une fois pour le cycle
	}
	paths := titlePkg.NewPathResolver(d.RepoRoot)
	aJour, complet := etatArtefact(paths.ReplayArtifactPath(d.TitleSlug, w.matchID), w.facts)
	if aJour && complet {
		b.dejaAJour++
		return
	}
	if aJour {
		slog.InfoContext(ctx, "post-sync: rejeu 2D — artefact au bon schéma mais SANS compteurs de joueur, reconstruit",
			"gamertag", d.Gamertag, "match_id", w.matchID, "lignes_de_match", len(w.facts.Players))
	}
	short := titlePkg.FilmShortMatchID(w.matchID)
	// LA CUISSON PART HORS DU PROCESSUS (lot BUILDALL, 2026-08-26) : l'enfant décode et rend
	// les OCTETS, le serveur les range. Le garde anti-régression et la notification restent
	// dans `StoreArtifact`, donc exactement où ils étaient.
	//
	// SOUS DEADLINE (item 5.5) : passé cette borne, l'enfant est TUÉ (le contexte est celui de
	// `exec.CommandContext`), le film compte en échec, et le cycle passe au suivant.
	cctx, annuler := context.WithTimeout(ctx, deadlineDuFilm(d, restant))
	defer annuler()
	out, berr := buildAndStoreOne(cctx, d, w, filmcache.ChunkDir(d.CacheRoot, short))
	if berr != nil {
		// Carte hors catalogue = échec voulu (Forge) ; le reste = erreur réelle.
		// Les deux sont best-effort, mais seuls les seconds méritent un WARN.
		logFn := slog.WarnContext
		if strings.Contains(berr.Error(), replaybuild.ErrMapNotInCatalog.Error()) {
			logFn = slog.DebugContext
		}
		logFn(ctx, "post-sync: artefact rejeu non construit",
			"gamertag", d.Gamertag, "match_id", w.matchID, "err", berr)
		b.echecs++
		return
	}
	b.construits++
	// Le coup d'envoi est lu sur l'artefact TEL QU'IL EST SUR DISQUE après rangement, et
	// mis de côté : l'écriture en base attend la fin du lot (cf. t0film.go).
	if t0 := lireT0FilmArtefact(out.stored.Path); t0 != nil {
		b.t0Film = append(b.t0Film, rapportT0Film{matchID: w.matchID, t0FilmMs: *t0})
	}
	// Le résumé d'usage suit la même règle : projeté depuis le FICHIER RANGÉ (jamais les
	// octets candidats, que `StoreArtifact` peut refuser), écrit en base une fois toute
	// cuisson terminée (cf. usage.go).
	b.usage = append(b.usage, artefactCuit{matchID: w.matchID, path: out.stored.Path})
	// LA DUREE ET LE PIC VIENNENT DE L'ENFANT (cf. buildone.go) : c'est la seule ligne du
	// cycle qui dit ce qu'a coute un film. Un pic a zero signifie « non mesure » — un enfant
	// mort avant de se mesurer —, jamais « aucune memoire ».
	slog.InfoContext(ctx, "post-sync: artefact rejeu construit",
		"gamertag", d.Gamertag, "match_id", w.matchID,
		"tracks", out.stored.Tracks, "bytes", out.stored.Bytes,
		"duration", out.dur, "pic_octets", out.peak)
}

// persistFilmToCache télécharge les chunks COMPLETS du film et les persiste au cache
// (pont disque). Rend (persisté, film disponible). Un film déjà entièrement en cache ne
// re-télécharge rien (GetFilmChunks est cache-first chunk par chunk).
func persistFilmToCache(ctx context.Context, d Deps, matchID string) (saved, available bool) {
	chunks, found, err := d.Fetcher.GetFilmChunks(ctx, matchID)
	if err != nil {
		// UN PRECHARGEMENT ABANDONNE N'EST PAS UN FILM ILLISIBLE. Le pont disque tourne aussi en
		// avance de phase (cf. prefetch.go) : quand le cycle se termine, son contexte est annulé
		// et le téléchargement en vol échoue — au niveau WARN, ce serait une fausse alerte à
		// chaque fin de cycle. Jamais muet pour autant : le renoncement se dit en DEBUG.
		if ctx.Err() != nil {
			slog.DebugContext(ctx, "post-sync: rejeu 2D — téléchargement de film abandonné (cycle terminé)",
				"gamertag", d.Gamertag, "match_id", matchID, "err", err)
			return false, false
		}
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
