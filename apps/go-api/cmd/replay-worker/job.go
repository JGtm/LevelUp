package main

// job.go — le TRAVAIL de l'ouvrier : télécharger, décoder, rendre.
//
// Le pont disque (filmcache.Write) est réutilisé tel quel : le décodeur lit un
// dossier de morceaux, et il n'existe qu'UNE disposition de cache film dans le
// dépôt (garde-rail filmcache_guard_test.go). L'ouvrier écrit donc dans cette
// disposition-là, pas dans une mise en page à lui.

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
)

// heartbeatInterval : cadence des signes de vie pendant un travail long. Trois
// battements manqués valent « hors ligne » côté serveur (domain.WorkerOfflineAfter).
const heartbeatInterval = 30 * time.Second

// chunkDownloadTimeout borne le téléchargement d'un morceau (CDN Azure).
const chunkDownloadTimeout = 60 * time.Second

// chunkParallelism : nombre maximal de téléchargements CDN simultanés pour UN film.
//
// MÊME VALEUR ET MÊME RAISON QUE `haloclient.filmChunkParallelism` : le CDN Azure sert des
// blobs indépendants, la latence domine, et huit connexions saturent la bande passante d'une
// liaison ordinaire sans jamais faire de l'ouvrier un client abusif. Les monter n'achète plus
// de temps (la limite devient le débit) et multiplie la mémoire tenue simultanément —
// l'ouvrier garde les morceaux décompressés en RAM jusqu'à l'écriture du cache.
const chunkParallelism = 8

// genericBuildFailedErrorCode : le motif d'échec ORDINAIRE (décodage, réseau, artefact non
// transmis...). Nommé pour rester DISTINCT, par construction, de
// domain.BuildJobErrorCodeMemoryExceeded — un film-bombe isolé ne doit jamais se confondre
// avec une vraie erreur de décodage (cf. TestProcessJob_EchecOrdinaire_GardeSonMotif, memlimit_test.go).
const genericBuildFailedErrorCode = "replay_build_failed"

// outilOuvrier : le nom sous lequel l'ouvrier tient le verrou de décodage de la machine. C'est
// ce mot que lira l'opérateur à qui le verrou est refusé (cf. internal/filmproc/solo.go).
const outilOuvrier = "replay-worker"

// attenteVerrouOuvrier : combien de temps l'ouvrier attend son tour avant de rendre le job en
// échec. Même borne que l'enfant de la passe de backfill, pour la même raison : c'est plus long
// que toute cuisson connue, donc une attente qui expire signale une machine vraiment occupée et
// non un chevauchement ordinaire.
const attenteVerrouOuvrier = 10 * time.Minute

// exitCodeMemoryExceeded : code de sortie quand la sentinelle mémoire (internal/filmproc.Arm,
// armée ci-dessous dans processJob) a isolé le job en cours. Distinct de 1 (arrêt sur erreur
// de la boucle) et 2 (configuration manquante, cf. main.go) : un opérateur qui lit les
// journaux du superviseur du processus doit pouvoir distinguer un dépassement mémoire
// volontaire d'un simple plantage.
const exitCodeMemoryExceeded = 3

// worker : l'état LOCAL de l'ouvrier. Rien d'autre que des compteurs — l'état de
// la file vit côté web, et c'est ce qui rend le travail distant observable.
type worker struct {
	identity workerIdentity
	client   *protocolClient
	repoRoot string
	workDir  string
	// keepsFilms : le dossier de travail EST le cache film du dépôt (archive
	// perpétuelle) — les morceaux ne sont alors jamais supprimés. Cf. cleanupFilm.
	keepsFilms bool
	// memLimitGiB : le plafond mémoire dur de CHAQUE job (0 = désarmé). Cf. internal/filmproc.Arm,
	// armé ci-dessous dans processJob.
	memLimitGiB int
	jobsDone    int64
	jobsFailed  int64
}

// processJob traite un job pris de bout en bout et en rend le résultat. Ne
// retourne jamais d'erreur : un travail raté est un job `failed` rendu au
// serveur, pas un ouvrier qui tombe.
//
// L'ORDRE EST L'ARTEFACT PUIS LE COMPTE RENDU, et c'est le cœur du transport :
// tant que le fichier n'est pas arrivé chez le web, il n'y a rien à annoncer. Si
// l'envoi échoue, l'ouvrier ne marque RIEN — le job reste `running`, son bail
// expire, et il repart en file. Un compte rendu de succès sans artefact serait un
// mensonge que le serveur refuserait de toute façon.
func (w *worker) processJob(ctx context.Context, job *domain.BuildQueueJob) {
	slog.InfoContext(ctx, "replay-worker: job pris",
		"job_id", job.JobID, "match_id", job.MatchID, "attempt", job.Attempt)

	beatCtx, stopBeat := context.WithCancel(ctx)
	go w.beatUntil(beatCtx, job.JobID)

	// LA SENTINELLE EST ARMÉE POUR CE JOB SEUL, via la sentinelle canonique
	// internal/filmproc.Arm (memes deux plafonds : 3 GiB souple, +25 % dur, échantillonnage
	// 250 ms — le calcul vit desormais dans un seul endroit, memguard.go) : un pic mesuré
	// depuis le démarrage de l'ouvrier confondrait plusieurs films. onExceeded RAPPORTE
	// l'échec au serveur PUIS arrête ce processus — l'OS récupère la RAM par construction,
	// même doctrine que l'enfant de la passe hors ligne (cmd/levelup, blindage 2026-08-20/24).
	// LE TEXTE DE LA LIGNE D'ARMEMENT EST CELUI D'AVANT LA CENTRALISATION, mot pour mot
	// (constat C5 de la revue R1) : un filtre de journal cale sur « replay-worker: plafond
	// memoire arme pour ce job » ne matchait plus le libelle unifie.
	guard := filmproc.Arm(outilOuvrier, w.memLimitGiB, func(peakBytes uint64) {
		w.reportMemoryExceeded(ctx, job, peakBytes)
	}, filmproc.WithArmMessage("replay-worker: plafond memoire arme pour ce job"))
	result, err := w.buildAndSend(ctx, job)
	guard.Disarm()
	stopBeat()
	w.cleanupFilm(ctx, job)

	if errors.Is(err, errArtifactNotDelivered) {
		// Rien à rendre : le serveur n'a pas le fichier, le bail tranchera.
		slog.ErrorContext(ctx, "replay-worker: artefact non transmis — job laissé au bail",
			"job_id", job.JobID, "match_id", job.MatchID, "err", err)
		w.jobsFailed++
		return
	}

	req := handlers.BuildQueueCompleteRequest{JobID: job.JobID, WorkerID: w.identity.workerID}
	if err != nil {
		w.jobsFailed++
		req.Succeeded = false
		req.ErrorCode = genericBuildFailedErrorCode
		req.ErrorMessage = err.Error()
		slog.WarnContext(ctx, "replay-worker: job échoué",
			"job_id", job.JobID, "match_id", job.MatchID, "err", err)
	} else {
		w.jobsDone++
		req.Succeeded = true
		req.ResultJSON = result
		slog.InfoContext(ctx, "replay-worker: job réussi",
			"job_id", job.JobID, "match_id", job.MatchID, "result", result)
	}
	// Le rendu part sur un contexte frais : un Ctrl-C pendant le décodage ne doit
	// pas empêcher de dire au serveur ce qui s'est passé. Sans ça, le job
	// resterait `running` jusqu'à l'expiration de son bail — vrai, mais lent.
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), httpTimeout)
	defer cancel()
	if cerr := w.client.complete(rctx, req); cerr != nil {
		slog.ErrorContext(ctx, "replay-worker: rendu du résultat échoué",
			"job_id", job.JobID, "err", cerr)
	}
}

// reportMemoryExceeded rend compte au serveur d'un dépassement du plafond mémoire dur PUIS
// arrête ce processus (le plafond lui-même est armé par internal/filmproc.Arm dans
// processJob ; le pourquoi des deux plafonds vit dans filmproc/doc.go). Le compte rendu est
// ISOLÉ dans completeMemoryExceeded (testable) : seule la ligne os.Exit ne l'est pas — même
// parti pris que memlimit_test.go (on teste les pièces, jamais la coupure elle-même, pour ne
// pas donner à un test le pouvoir de tuer le binaire de test).
//
// APPELÉE DEPUIS LA GOROUTINE DE LA SENTINELLE, PENDANT QUE LE DÉCODAGE EST ENCORE EN VOL :
// c'est voulu, c'est le seul moment où ce processus est encore vivant pour parler au serveur.
func (w *worker) reportMemoryExceeded(ctx context.Context, job *domain.BuildQueueJob, peakBytes uint64) {
	w.completeMemoryExceeded(ctx, job, peakBytes)
	os.Exit(exitCodeMemoryExceeded)
}

// completeMemoryExceeded envoie le compte rendu explicite du dépassement mémoire. Séparée de
// reportMemoryExceeded UNIQUEMENT pour rester testable (aucun os.Exit ici) — cf.
// memlimit_test.go, TestWorker_CompleteMemoryExceeded_MotifExplicite.
func (w *worker) completeMemoryExceeded(ctx context.Context, job *domain.BuildQueueJob, peakBytes uint64) {
	slog.ErrorContext(ctx, "replay-worker: PLAFOND MEMOIRE DEPASSE — film isolé, arrêt du processus",
		"job_id", job.JobID, "match_id", job.MatchID, "pic_octets", peakBytes)
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), httpTimeout)
	defer cancel()
	req := memoryExceededRequest(w.identity.workerID, job, peakBytes)
	if cerr := w.client.complete(rctx, req); cerr != nil {
		slog.ErrorContext(ctx, "replay-worker: rendu du dépassement mémoire échoué",
			"job_id", job.JobID, "match_id", job.MatchID, "err", cerr)
	}
}

// memoryExceededRequest construit le compte rendu d'échec pour UN dépassement du plafond
// mémoire dur. Fonction PURE (aucun appel réseau, aucun os.Exit) : c'est elle que les tests
// exercent pour vérifier le motif, jamais le chemin qui arrête le processus.
func memoryExceededRequest(workerID string, job *domain.BuildQueueJob, peakBytes uint64) handlers.BuildQueueCompleteRequest {
	return handlers.BuildQueueCompleteRequest{
		JobID:        job.JobID,
		WorkerID:     workerID,
		Succeeded:    false,
		ErrorCode:    domain.BuildJobErrorCodeMemoryExceeded,
		ErrorMessage: memoryExceededMessage(job.MatchID, peakBytes),
	}
}

// memoryExceededMessage : le texte lisible par un opérateur qui consulte le tableau de bord
// admin. Porte les trois informations demandées par le lot : LE MATCH, LE PIC (quand mesuré),
// et le sort du film — jamais un "failed" anonyme qui obligerait à rouvrir les journaux pour
// distinguer un film-bombe connu d'une vraie erreur de décodage.
func memoryExceededMessage(matchID string, peakBytes uint64) string {
	pic := "pic inconnu"
	if peakBytes > 0 {
		pic = "pic " + formatMemGuardBytes(peakBytes)
	}
	return fmt.Sprintf("dépassement mémoire (film-bombe) sur le match %s, %s — film isolé, passe poursuivie",
		matchID, pic)
}

// memGuardOctetsParGiB : la conversion, nommée pour ne pas semer des 1<<30 dans le code. Sert
// uniquement à la mise en forme humaine (formatMemGuardBytes, ci-dessous) — la sentinelle
// mémoire elle-même est armée par internal/filmproc.Arm (cf. processJob), qui porte sa propre
// constante interne.
const memGuardOctetsParGiB = 1 << 30

// formatMemGuardBytes rend une taille lisible ("7.90 GiB" / "512 MiB"). Même forme que
// libelleOctets de cmd/levelup — paquets main distincts, duplication triviale (une conversion
// et un printf) sous le seuil de centralisation de CLAUDE.md règle 6.
func formatMemGuardBytes(n uint64) string {
	if n >= memGuardOctetsParGiB {
		return fmt.Sprintf("%.2f GiB", float64(n)/float64(memGuardOctetsParGiB))
	}
	return fmt.Sprintf("%.0f MiB", float64(n)/(1024*1024))
}

// errArtifactNotDelivered : l'artefact a été construit mais n'est pas arrivé chez
// le web. Distinct d'un échec de travail : il ne se rend PAS au serveur (rien à
// annoncer), il laisse le bail expirer.
var errArtifactNotDelivered = errors.New("artefact non transmis")

// buildAndSend construit l'artefact PUIS le pousse au web. Rend le résumé JSON du
// travail (ce que l'admin verra dans la colonne résultat) — l'accusé du serveur y
// figure, parce que c'est lui, et pas la construction locale, qui prouve la
// livraison.
func (w *worker) buildAndSend(ctx context.Context, job *domain.BuildQueueJob) (string, error) {
	p := job.Payload
	if p == nil || len(p.Chunks) == 0 {
		return "", fmt.Errorf("job sans travail résolu (aucune URL de morceau)")
	}
	chunks, err := w.fetchChunks(ctx, p)
	if err != nil {
		return "", err
	}
	if err := filmcache.Write(w.workDir, p.ShortID, chunks); err != nil {
		return "", fmt.Errorf("écriture du cache film : %w", err)
	}

	builder, err := replaybuild.NewBuilder(w.repoRoot, p.TitleSlug)
	if err != nil {
		return "", fmt.Errorf("catalogue de titre indisponible : %w", err)
	}
	// L'OUVRIER N'A PAS DE BASE : les faits du match lui arrivent DANS LE JOB, résolus par le
	// web au moment de la mise en file (cf. domain.BuildQueuePayload.Facts). C'est ce qui lui
	// permet de rendre un artefact COMPLET sans jamais toucher une DuckDB.
	var facts port.MatchFacts
	if p.Facts != nil {
		facts = *p.Facts
	}
	if facts.Empty() {
		// DÉGRADATION ANNONCÉE, JAMAIS MUETTE. Cas réel : un match hors registre, ou un job
		// enfilé par une version antérieure du serveur (le payload est stocké tel quel dans la
		// file, il ne se met pas à jour tout seul). L'artefact reste VALIDE, seulement appauvri —
		// et la liste ci-dessous est MESURÉE (témoin 7344d24f, 2026-08-24), pas supposée.
		slog.WarnContext(ctx, "replay-worker: aucun fait de match dans le job — artefact sans "+
			"actions d'objectif, sans zones de mode, sans socles de drapeau et sans compteurs de "+
			"joueur, identité des camps au mieux par les frags",
			"job_id", job.JobID, "match_id", p.MatchID)
	} else {
		slog.InfoContext(ctx, "replay-worker: faits du match reçus dans le job",
			"job_id", job.JobID, "match_id", p.MatchID, "joueurs", len(facts.Players),
			"variante", facts.GameVariantName, "carte", facts.MapID)
	}
	// LE VERROU SOLO, EN ATTENTE BORNÉE (PLAN_CUISSON_PERF §3 D7). L'ouvrier ne prend qu'un job
	// à la fois, mais RIEN n'empêche de le lancer sur une machine qui décode déjà — un poste de
	// développement qui fait tourner le serveur (post-sync), une passe `backfill-replay`, ou
	// simplement deux ouvriers. C'est exactement le trou par lequel le quatrième sinistre est
	// passé (cf. internal/filmproc/solo.go) : « un film à la fois DANS ce processus » ne dit
	// rien du nombre de processus. Il ATTEND son tour plutôt que d'échouer (le job est déjà pris,
	// son bail court, et le rendre en échec pour un chevauchement gâcherait le téléchargement
	// qu'on vient de payer) ; passé la borne, c'est un échec ordinaire, détenteur nommé.
	//
	// PRIS APRÈS LE TÉLÉCHARGEMENT ET RENDU AVANT L'ENVOI : le verrou protège la RAM du
	// DÉCODAGE, pas le réseau. L'étendre au transfert immobiliserait la machine pour rien.
	//
	// ENRACINÉ SUR LE CACHE DU DÉPÔT, PAS SUR `w.workDir` (lot 6, constat 7). Le verrou vit à
	// `<racine>/data/cache/film_decode.lock` : c'est le contrat de `filmproc.AcquireSolo`, et
	// c'est là que le prennent les TROIS autres points d'entrée (post-sync, passe de backfill,
	// `replay-build`). Tant que `--work` n'est pas passé, `w.workDir` VAUT ce cache et l'exclusion
	// tenait par coïncidence ; dès qu'un ouvrier distant passe `--work`, il posait son verrou dans
	// un dossier que personne d'autre ne regarde — et l'exclusion annoncée juste au-dessus
	// n'existait plus. `w.repoRoot` est garanti non vide (le binaire sort en 2 sans `--repo`).
	verrouRoot := titlePkg.NewPathResolver(w.repoRoot).CacheRootDir()
	lock, err := filmproc.AcquireSoloWait(ctx, verrouRoot, outilOuvrier, p.MatchID, attenteVerrouOuvrier)
	if err != nil {
		return "", fmt.Errorf("decodage refuse : %w", err)
	}
	// LE DIFFÉRÉ EST UN FILET, PAS LE RENDU NOMINAL (`Release` est idempotent) : le verrou tombe
	// explicitement dès le décodage fini, quelques lignes plus bas. Ce `defer` couvre le jour où
	// quelqu'un ajoutera un retour anticipé entre les deux.
	defer lock.Release()
	// LES OCTETS NE TOUCHENT PAS LE DISQUE DE L'OUVRIER (PLAN_CUISSON_PERF §3 D8). `BuildMatch`
	// écrivait l'artefact à sa place canonique LOCALE, puis cette fonction le RELISAIT pour
	// l'envoyer : deux entrées/sorties de plusieurs mégaoctets pour un fichier dont personne
	// ici n'a l'usage — l'artefact qui fait foi est celui que le SERVEUR range
	// (`replaybuild.StoreArtifact`), avec son garde anti-régression et sa notification. Un
	// ouvrier distant écrivait en prime dans une arborescence de dépôt qu'il n'a pas.
	built, err := builder.BuildBytes(p.MatchID, p.MapNames, filmcache.ChunkDir(w.workDir, p.ShortID), facts)
	lock.Release()
	if err != nil {
		return "", err
	}
	// L'EMPREINTE SE CALCULE ICI, SUR LES OCTETS ENCORE EN MÉMOIRE — jamais par une relecture
	// du disque (l'ouvrier n'en garde pas, cf. D8 et TestOuvrier_NeComposeJamaisLEcritureDArtefact).
	// C'est la SEULE trace indépendante de ce que l'ouvrier a construit qui survit à son
	// processus : le compte rendu (ci-dessous) est journalisé et lu par la file durable, alors
	// que `built.Blob` disparaît avec l'ouvrier. Elle prouve, sans copie locale, que l'artefact
	// rangé par le serveur est bien celui-ci — cf. build_queue_worker_binary_integration_test.go,
	// assertArtefactLivreEtComplet.
	empreinte := sha256.Sum256(built.Blob)
	// Contexte frais : un Ctrl-C pendant le décodage ne doit pas jeter un artefact
	// qui a coûté 50 s de CPU alors qu'il ne reste qu'à l'envoyer.
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), artifactUploadTimeout)
	defer cancel()
	receipt, err := w.client.sendArtifact(sctx, job.JobID, w.identity.workerID, built.Blob)
	if err != nil {
		return "", fmt.Errorf("%w : %v", errArtifactNotDelivered, err)
	}
	slog.InfoContext(ctx, "replay-worker: artefact transmis",
		"job_id", job.JobID, "match_id", p.MatchID, "bytes", receipt.Bytes,
		"schema", receipt.SchemaVersion)

	// PAS DE `artifact_path` DANS LE RÉSUMÉ, ET C'EST LA CONSÉQUENCE DIRECTE DE D8 : l'ouvrier
	// n'écrit plus d'artefact, il n'a donc plus de chemin à annoncer. Le champ n'avait aucun
	// lecteur (ni serveur, ni web) — il décrivait un fichier de la machine de l'ouvrier, que
	// personne d'autre ne pouvait ouvrir.
	summary, err := json.Marshal(map[string]any{
		"match_id": p.MatchID,
		"module":   built.Module,
		"tracks":   built.Tracks,
		"bytes":    receipt.Bytes,
		"chunks":   len(chunks),
		// sha256 : empreinte de `built.Blob` (avant envoi), pas de l'artefact rangé — ils ne
		// peuvent différer que si le serveur a REFUSÉ l'écriture (garde anti-régression,
		// cf. writeArtifactBytes) ; le vérifier alors est un faux positif attendu, pas un bug.
		"sha256": hex.EncodeToString(empreinte[:]),
	})
	if err != nil {
		return "", fmt.Errorf("sérialisation du résultat : %w", err)
	}
	return string(summary), nil
}

// cleanupFilm supprime les morceaux de film que CET ouvrier a téléchargés : il ne
// conserve rien (piste F §1), et 24 Mo par match rempliraient vite une machine de
// calcul.
//
// SAUF SI SON DOSSIER DE TRAVAIL EST LE CACHE FILM DU DÉPÔT. C'est le cas par
// défaut quand l'ouvrier tourne sur le poste de développement, et ce cache est une
// ARCHIVE IRREMPLAÇABLE : les films expirent côté serveur Halo (29,3 % du corpus
// déjà perdus), un film effacé ne se re-télécharge pas. Un ouvrier ne détruit
// jamais l'archive de la machine qui l'héberge ; pour un ouvrier distant qui doit
// nettoyer, --work désigne un dossier de travail à lui.
func (w *worker) cleanupFilm(ctx context.Context, job *domain.BuildQueueJob) {
	if job.Payload == nil || job.Payload.ShortID == "" {
		return
	}
	if w.keepsFilms {
		slog.DebugContext(ctx, "replay-worker: morceaux conservés (cache film du dépôt, archive perpétuelle)",
			"match_id", job.MatchID)
		return
	}
	dir := filmcache.ChunkDir(w.workDir, job.Payload.ShortID)
	if err := os.RemoveAll(dir); err != nil {
		// Non fatal : un morceau qui traîne coûte du disque, pas de la justesse.
		slog.WarnContext(ctx, "replay-worker: morceaux de film non supprimés",
			"dir", dir, "err", err)
		return
	}
	slog.InfoContext(ctx, "replay-worker: morceaux de film supprimés", "dir", dir)
}

// fetchChunks télécharge les morceaux depuis les URL PRÉ-SIGNÉES du job et les
// décompresse (le CDN Azure des films rend du zlib brut). Aucune authentification
// n'est présentée : c'est exactement la propriété qui permet à cet ouvrier de
// n'avoir aucun secret Halo.
//
// LES TÉLÉCHARGEMENTS SONT PARALLÈLES, BORNÉS À [chunkParallelism], SUR LE MODÈLE EXACT DU
// CLIENT DE SYNC (`haloclient.fetchFilmChunks`) : errgroup.WithContext + SetLimit, chaque
// goroutine écrivant dans un SLOT PRÉ-ALLOUÉ. Aucun mutex, aucun tri après coup —
// l'assemblage est fait par l'indice de boucle, donc L'ORDRE DU JOB EST L'ORDRE RENDU, et il
// l'est par construction plutôt que par une convention que la prochaine relecture pourrait
// perdre. Un film de 30 morceaux tenait la latence CDN trente fois de suite ; il la tient
// désormais quatre fois. Une erreur annule le contexte du groupe : les téléchargements en vol
// s'arrêtent, et le JOB ÉCHOUE — un film à trous ne se cuit pas.
func (w *worker) fetchChunks(ctx context.Context, p *domain.BuildQueuePayload) ([]filmcache.WriteChunk, error) {
	// LE CONTEXTE DÉJÀ ANNULÉ NE LANCE RIEN. errgroup exécuterait quand même chaque tâche
	// (pour la voir échouer sur son propre contexte) : le dire ici garde la propriété
	// « annulé = aucun appel réseau », que le protocole d'arrêt de l'ouvrier suppose.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: chunkDownloadTimeout}
	out := make([]filmcache.WriteChunk, len(p.Chunks))
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(chunkParallelism)
	for i, c := range p.Chunks {
		eg.Go(func() error {
			data, err := downloadChunk(egCtx, client, c.URL)
			if err != nil {
				return fmt.Errorf("morceau %d : %w", c.Index, err)
			}
			out[i] = filmcache.WriteChunk{
				Index: c.Index, ChunkType: c.ChunkType,
				StartMS: c.StartMS, DurationMS: c.DurationMS, Data: data,
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// downloadChunk lit un blob CDN et le décompresse.
func downloadChunk(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d (URL pré-signée expirée ?)", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("en-tête zlib : %w", err)
	}
	defer func() { _ = zr.Close() }()
	return io.ReadAll(zr)
}

// beatUntil bat tant que le job est en cours. C'est ce battement qui prolonge le
// bail : sans lui, un décodage plus long que le bail verrait son job repris par
// la file alors qu'il avance très bien.
func (w *worker) beatUntil(ctx context.Context, jobID string) {
	t := time.NewTicker(heartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			bctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), httpTimeout)
			err := w.client.heartbeat(bctx, w.identity, heartbeat{
				jobID: jobID, note: "décodage en cours",
				done: w.jobsDone, failed: w.jobsFailed,
			})
			cancel()
			if err != nil {
				// Un battement perdu n'est pas fatal : le bail tient 5 min, il en
				// reste plusieurs à venir. Loguer, jamais avaler (règle n°3).
				slog.WarnContext(ctx, "replay-worker: battement non transmis", "job_id", jobID, "err", err)
			}
		}
	}
}
