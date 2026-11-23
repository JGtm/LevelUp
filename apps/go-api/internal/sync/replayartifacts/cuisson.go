package replayartifacts

// cuisson.go — CE QUE LE CYCLE FAIT D'UN LOT : archiver le film, puis construire l'artefact,
// ou bien le confier à la file durable d'un ouvrier.
//
// Extrait d'artifacts.go le 2026-09-01, quand le rattrapage a poussé le fichier d'orchestration
// au-delà du seuil projet. Le découpage suit les responsabilités : artifacts.go décide QUOI
// faire, backlog.go dit SUR QUOI, ce fichier-ci fait le travail.

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmarchive"
	"levelup/go-api/internal/filmrun"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
	"github.com/JGtm/halo-infinite-film-decoder/finalise"
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
//
// Rend le nombre de matchs du lot que la passe n'a PAS examinés (budget de temps, annulation) : ils
// restent à rattraper au cycle suivant.
func enqueueAll(ctx context.Context, d Deps, work []buildWork) (nonExamines int) {
	if d.Enqueue == nil {
		return 0
	}
	paths := titlePkg.NewPathResolver(d.RepoRoot)
	debut, budget := time.Now(), budgetDuCycle(d)
	queued, skipped, appauvris, refuses := 0, 0, 0, 0
	for i, w := range work {
		if ctx.Err() != nil {
			nonExamines = len(work) - i
			break
		}
		// Enfiler coûte une résolution de manifeste — donc un aller-retour RÉSEAU par match.
		// Le même budget s'applique ici : la file est faite pour s'allonger, pas le cycle.
		if time.Since(debut) >= budget {
			slog.InfoContext(ctx, "post-sync: rejeu 2D — budget de cycle épuisé, solde au cycle suivant",
				"gamertag", d.Gamertag, "budget", budget, "enfiles", queued)
			nonExamines = len(work) - i
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
		if err := d.Enqueue(ctx, d.TitleSlug, w.matchID, w.priorite); err != nil {
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
	return nonExamines
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
				"gamertag", d.Gamertag, "budget", budget, "traites", b.traites())
			break
		}
		film := pont.film(ctx, w.matchID)
		// LE PRECHARGEMENT PART AVANT LA CUISSON, ET AVANT TOUT `continue` : c'est pendant les
		// dizaines de secondes de décodage que le lien est libre. Un match sans film ou déjà à
		// jour ne doit pas priver le SUIVANT de son avance.
		if i+1 < len(work) {
			pont.precharger(ctx, work[i+1].matchID, budget-time.Since(debut))
		}
		if film.reporte {
			b.reportes++
			continue // film pas encore finalisé : compté et journalisé par reporterNonFinalise
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
				"traites", b.traites())
			break
		}
		if reporterEnModeJeu(ctx, d, &b) {
			continue
		}
		cuireUnMatch(ctx, d, w, &b, solde)
	}
	signalerCartesHorsCatalogue(ctx, d, b.horsCatalogue)
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
	observability.ReportMatchStep(d.TitleSlug, w.matchID, domain.MatchStepReplayBuildStarted, "")
	cctx, annuler := context.WithTimeout(ctx, deadlineDuFilm(d, restant))
	defer annuler()
	out, berr := buildAndStoreOne(cctx, d, w, filmcache.ChunkDir(d.FilmRoot, short))
	if errors.Is(berr, filmrun.ErrDecodeBusy) {
		reporterVerrouTenu(ctx, d, b, berr)
		return
	}
	if berr != nil {
		classerRefus(ctx, d, w, b, berr)
		return
	}
	b.construits++
	observability.ReportMatchStep(d.TitleSlug, w.matchID, domain.MatchStepArtifactStored, "local")
	retirerFilmNonArchive(ctx, d, short)
	// L'ARTEFACT EST RANGÉ : c'est le seul fait qui déclenche les dérivations (cf.
	// derivations.go). Il est mis de côté, jamais dérivé ici — les projections lisent le
	// FICHIER RANGÉ (jamais les octets candidats, que `StoreArtifact` peut refuser) et
	// écrivent en base une fois TOUTE cuisson terminée.
	b.ranges = append(b.ranges, ArtefactRange{MatchID: w.matchID, Path: out.stored.Path})
	// LA DUREE ET LE PIC VIENNENT DE L'ENFANT (cf. buildone.go) : c'est la seule ligne du
	// cycle qui dit ce qu'a coute un film. Un pic a zero signifie « non mesure » — un enfant
	// mort avant de se mesurer —, jamais « aucune memoire ».
	slog.InfoContext(ctx, "post-sync: artefact rejeu construit",
		"gamertag", d.Gamertag, "match_id", w.matchID,
		"tracks", out.stored.Tracks, "bytes", out.stored.Bytes,
		"duration", out.dur, "pic_octets", out.peak)
}

// retirerFilmNonArchive retire les chunks d'un film cuit quand l'archive est désactivée
// (ADR 0045 D-12) : l'artefact est rangé et les faits persistés par la cuisson, le film n'a plus
// d'usage. Jamais avant : un film non cuit (budget, carte hors catalogue) reste, c'est ce qui
// permet de le cuire au cycle suivant. Un échec de retrait se journalise et ne casse rien : le
// film reste, il sera retiré à la prochaine cuisson ou par l'utilisateur.
func retirerFilmNonArchive(ctx context.Context, d Deps, short string) {
	if !d.SansArchive {
		return
	}
	if err := filmarchive.Supprimer(d.FilmRoot, short); err != nil {
		slog.WarnContext(ctx, "post-sync: film cuit non retiré (archive désactivée)",
			"gamertag", d.Gamertag, "short_id", short, "err", err)
		return
	}
	observability.IncCounter(CompteurFilmsRetiresSansArchive)
}

// classerRefus range le refus d'une cuisson : écarté (clé de profil inconnue, film non finalisé),
// carte hors catalogue (échec compté par carte), ou échec ordinaire (WARN par film).
func classerRefus(ctx context.Context, d Deps, w buildWork, b *bilanCuisson, berr error) {
	// CLÉ INCONNUE = ÉCARTÉ, PAS UN ÉCHEC (lot 3.1.1). Le film est là et lisible ; ce qui
	// manque est une ligne de table côté dépôt. Le compter en échec ferait chercher une
	// panne, et noierait dans le bruit le seul signal qui compte : le jeu a changé.
	//
	// LE CLASSEMENT PORTE SUR LE TYPE, JAMAIS SUR LE TEXTE (lot J2.12, décision DT-5) :
	// l'erreur traverse une frontière de processus, mais la RAISON du refus la traverse
	// aussi, en jeton de protocole (`filmrun.EmitRaison`), et `replaychild` rend l'erreur
	// typée enveloppée — `errors.Is` a donc de quoi mordre.
	if errors.Is(berr, replaybuild.ErrUnknownFilmKey) {
		slog.WarnContext(ctx, "post-sync: artefact rejeu ECARTE — clé du film absente de la "+
			"table de profil (ajouter la ligne : docs/RUNBOOK_FILM_PROFILES.md)",
			"gamertag", d.Gamertag, "match_id", w.matchID, "err", berr)
		b.ecartes++
		return
	}
	// FILM NON FINALISÉ = ÉCARTÉ, PAS UN ÉCHEC (lot L3, 2026-09-23). La cuisson refuse un
	// film dont le manifeste ne porte pas le morceau des temps forts, ou dont des morceaux ne
	// sont pas décrits par le manifeste : cuire un film TRONQUÉ publierait un document faux
	// (`ab526724`). Même frontière de processus, donc même classement par le type.
	if errors.Is(berr, finalise.ErrFilmNonFinalise) {
		slog.InfoContext(ctx, "post-sync: artefact rejeu ÉCARTÉ — film non finalisé au cache",
			"gamertag", d.Gamertag, "match_id", w.matchID, "err", berr)
		b.ecartes++
		return
	}
	// CARTE HORS CATALOGUE DE BORNES : échec voulu (Forge, ou module absent de l'installation),
	// compté en échec. Le film se dit en DEBUG ; la CARTE se dit en WARN, une fois par cycle
	// ([signalerCartesHorsCatalogue]) : elle bloque tous ses films, et un WARN par film
	// noierait le journal quand un DEBUG seul la cachait.
	if errors.Is(berr, replaybuild.ErrMapNotInCatalog) {
		slog.DebugContext(ctx, "post-sync: artefact rejeu non construit — carte hors catalogue",
			"gamertag", d.Gamertag, "match_id", w.matchID, "err", berr)
		b.noterHorsCatalogue(w.mapNames)
		b.echecs++
		return
	}
	slog.WarnContext(ctx, "post-sync: artefact rejeu non construit",
		"gamertag", d.Gamertag, "match_id", w.matchID, "err", berr)
	b.echecs++
}

// persistFilmToCache télécharge les chunks COMPLETS du film et les persiste au cache
// (pont disque). Rend ce que le cycle doit en savoir : persisté, disponible, ou REPORTÉ. Un film
// déjà entièrement en cache ne re-télécharge rien (GetFilmChunks est cache-first chunk par chunk).
//
// UN FILM NON FINALISÉ N'EST NI ARCHIVÉ NI DISPONIBLE (lot L3) : refusé par le client ou par le
// writer, il est REPORTÉ ([reporterNonFinalise]) — le match revient au cycle suivant par le
// rattrapage, avec un film complet. Le report est un poste DISTINCT du bilan : le ranger avec
// les films absents dirait « expiré » d'un film qui existe (constat L3-R3 de la revue adverse).
func persistFilmToCache(ctx context.Context, d Deps, matchID string) resultatFilm {
	chunks, found, err := d.Fetcher.GetFilmChunks(ctx, matchID)
	if err != nil {
		// UN PRECHARGEMENT ABANDONNE N'EST PAS UN FILM ILLISIBLE. Le pont disque tourne aussi en
		// avance de phase (cf. prefetch.go) : quand le cycle se termine, son contexte est annulé
		// et le téléchargement en vol échoue — au niveau WARN, ce serait une fausse alerte à
		// chaque fin de cycle. Jamais muet pour autant : le renoncement se dit en DEBUG.
		if ctx.Err() != nil {
			slog.DebugContext(ctx, "post-sync: rejeu 2D — téléchargement de film abandonné (cycle terminé)",
				"gamertag", d.Gamertag, "match_id", matchID, "err", err)
			return resultatFilm{}
		}
		if reporterNonFinalise(ctx, d, matchID, err) {
			return resultatFilm{reporte: true}
		}
		slog.WarnContext(ctx, "post-sync: film illisible — rejeu non construit",
			"gamertag", d.Gamertag, "match_id", matchID, "err", err)
		return resultatFilm{}
	}
	if !found || len(chunks) == 0 {
		slog.DebugContext(ctx, "post-sync: film absent côté serveur — rejeu non construit",
			"match_id", matchID)
		return resultatFilm{}
	}
	wc := make([]filmarchive.WriteChunk, 0, len(chunks))
	for _, c := range chunks {
		wc = append(wc, filmarchive.WriteChunk{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS,
			DurationMS: c.DurationMS, Data: c.Data,
		})
	}
	if err := filmarchive.Write(ctx, d.FilmRoot, titlePkg.FilmShortMatchID(matchID), wc); err != nil {
		// Un client qui n'aurait pas vérifié la finalisation : le writer la vérifie, et c'est le
		// même report.
		if reporterNonFinalise(ctx, d, matchID, err) {
			return resultatFilm{reporte: true}
		}
		slog.WarnContext(ctx, "post-sync: persistance du film au cache échouée",
			"gamertag", d.Gamertag, "match_id", matchID, "err", err)
		return resultatFilm{}
	}
	// LE FILM EST ARCHIVÉ, FINALISÉ : son attente (s'il en avait une) est close.
	attentes.oublier(cleDAttente(ctx, matchID))
	return resultatFilm{sauve: true, dispo: true}
}
