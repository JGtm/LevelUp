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
	// usage : les artefacts cuits dans ce cycle, à projeter en résumé d'usage puis à écrire
	// une fois toute cuisson terminée (cf. usage.go) — même règle de voyage que t0Film.
	usage []rapportUsage
}

// buildAll persiste le film puis construit l'artefact de chaque match du lot.
func buildAll(ctx context.Context, d Deps, work []buildWork) bilanCuisson {
	var b bilanCuisson
	debut, budget := time.Now(), budgetDuCycle(d)
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
		// LE BUDGET S'APPLIQUE ENTRE DEUX MATCHS, jamais au milieu d'un : couper une cuisson
		// en cours ne rendrait qu'un artefact tronqué. Le solde repart au cycle suivant.
		if time.Since(debut) >= budget {
			b.budgetEpuise = true
			slog.InfoContext(ctx, "post-sync: rejeu 2D — budget de cycle épuisé, solde au cycle suivant",
				"gamertag", d.Gamertag, "budget", budget, "traites", b.construits+b.dejaAJour+b.sansFilm+b.echecs)
			break
		}
		saved, ok := persistFilmToCache(ctx, d, w.matchID)
		if !ok {
			b.sansFilm++
			continue // film absent/expiré côté serveur : rien à construire (débité en debug)
		}
		if saved {
			b.filmsSauves++
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
			b.dejaAJour++
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
			b.echecs++
			continue
		}
		b.construits++
		// Le coup d'envoi est lu sur l'artefact TEL QU'IL EST SUR DISQUE après rangement, et
		// mis de côté : l'écriture en base attend la fin du lot (cf. t0film.go).
		if t0 := lireT0FilmArtefact(out.Path); t0 != nil {
			b.t0Film = append(b.t0Film, rapportT0Film{matchID: w.matchID, t0FilmMs: *t0})
		}
		// Le résumé d'usage suit la même règle : projeté depuis le disque, écrit en base
		// une fois toute cuisson terminée (cf. usage.go).
		b.usage = append(b.usage, rapportUsage{matchID: w.matchID, path: out.Path})
		slog.InfoContext(ctx, "post-sync: artefact rejeu construit",
			"gamertag", d.Gamertag, "match_id", w.matchID, "tracks", out.Tracks, "bytes", out.Bytes)
	}
	return b
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
