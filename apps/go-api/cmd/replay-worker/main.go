// cmd/replay-worker — l'OUVRIER de la file de construction, dans sa forme minimale.
//
// CE BINAIRE EXISTE POUR PROUVER LE PROTOCOLE, pas pour être déployé tel quel.
// Le plan (piste F §1) est explicite : l'ouvrier est un RÔLE, pas une machine —
// il peut être tenu par un second VPS, par le poste de développement, ou par
// personne. Ce qu'il fait tient en une boucle :
//
//	prendre un job → télécharger les morceaux (URL pré-signées) → décoder →
//	POUSSER l'artefact → rendre le résultat, en battant pendant tout le trajet.
//
// L'artefact part AVANT le compte rendu, et le compte rendu n'est envoyé que si
// l'artefact est arrivé : sans fichier chez le web, le travail n'a pas eu lieu.
//
// CE QU'IL N'A PAS, ET C'EST TOUT L'INTÉRÊT : aucun token Halo, aucun accès à la
// base, aucun port entrant. Il présente un jeton d'ouvrier, il tire son travail
// déjà résolu. On peut le faire tourner n'importe où sans lui confier un secret.
//
// CE QU'IL LUI FAUT QUAND MÊME : le dépôt (le catalogue de bornes de carte
// data/titles/{slug}/reference/ et les libellés vivent en fichiers versionnés) et
// de quoi écrire un dossier de travail. Pas de DuckDB.
//
// Usage :
//
//	replay-worker --url http://127.0.0.1:8000/api/v1/internal --token <jeton> \
//	              [--id ouvrier-1] [--once] [--poll 5s] [--work <dir>] [--mem-limit-gib 3]
//
// --once prend UN job, le traite, et sort : c'est le mode de la preuve de bout en
// bout (et celui d'un test manuel). Sans --once, il boucle jusqu'à Ctrl-C.
//
// --mem-limit-gib POSE LE MÊME BLINDAGE QUE cmd/levelup backfill-replay (lot 2026-08-20/24,
// cf. memlimit.go) : un film-bombe (empreinte hors norme, ex. 51101d1d à 7,9 Go en 2,6 s) fait
// arrêter CE PROCESSUS plutôt que de le laisser spiraler en GC pendant des heures — mais
// contrairement à l'enfant de la passe hors ligne, l'ouvrier RAPPORTE d'abord au serveur un
// échec explicite (error_code=memory_exceeded) avant de s'arrêter : sans ce compte rendu, le
// job resterait `running` jusqu'à l'expiration de son bail et repartirait avec un motif
// générique qui ne dit rien du film-bombe. 0 désarme (mesure seule, aucune coupure).
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
)

// defaultPollInterval : cadence d'interrogation de la file quand elle est vide.
// Un ouvrier au repos doit être discret ; le travail n'est jamais urgent à la
// seconde près (un rejeu se consulte des heures après le match).
const defaultPollInterval = 5 * time.Second

// workerVersion identifie la version de protocole que cet ouvrier parle. Affiché
// dans le tableau des ouvriers du dashboard admin.
const workerVersion = "replay-worker/1"

// workerIdentity : ce que l'ouvrier dit de lui-même à chaque appel.
type workerIdentity struct {
	workerID string
	hostname string
	version  string
}

func main() {
	url := flag.String("url", "http://127.0.0.1:8000/api/v1/internal", "racine des routes internes du serveur web")
	token := flag.String("token", os.Getenv("LEVELUP_BUILD_WORKER_TOKEN"), "jeton d'ouvrier (défaut : LEVELUP_BUILD_WORKER_TOKEN)")
	id := flag.String("id", "", "identifiant de cet ouvrier (défaut : nom de la machine)")
	repoRoot := flag.String("repo", os.Getenv("LEVELUP_REPO_ROOT"), "racine du dépôt (catalogue de cartes ; défaut : LEVELUP_REPO_ROOT)")
	work := flag.String("work", "", "dossier de travail pour les morceaux téléchargés, effacés après chaque job (défaut : <repo>/data/cache, cache film du dépôt — jamais effacé)")
	poll := flag.Duration("poll", defaultPollInterval, "cadence d'interrogation quand la file est vide")
	once := flag.Bool("once", false, "prendre un seul job puis sortir")
	memLimit := flag.Int("mem-limit-gib", memGuardDefaultGiB,
		"plafond mémoire dur du décodage de CHAQUE job, en GiB (0 = désarmé ; cf. memlimit.go)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if *token == "" {
		slog.Error("replay-worker: jeton d'ouvrier absent — passer --token ou LEVELUP_BUILD_WORKER_TOKEN")
		os.Exit(2)
	}
	if *repoRoot == "" {
		slog.Error("replay-worker: racine du dépôt absente — passer --repo ou LEVELUP_REPO_ROOT")
		os.Exit(2)
	}
	host, _ := os.Hostname()
	identity := workerIdentity{workerID: *id, hostname: host, version: workerVersion}
	if identity.workerID == "" {
		identity.workerID = host
	}
	if identity.workerID == "" {
		identity.workerID = "replay-worker"
	}
	// Dossier de travail par défaut : le cache film du dépôt. C'est le bon défaut
	// sur un poste de développement (les films y sont déjà, rien à re-télécharger)
	// et c'est précisément pour ça que l'ouvrier n'y supprime RIEN — ce cache est
	// une archive irremplaçable (cf. cleanupFilm). Un ouvrier distant passe --work.
	workDir := *work
	repoCache := titlePkg.NewPathResolver(*repoRoot).CacheRootDir()
	if workDir == "" {
		workDir = repoCache
	}
	keepsFilms := sameDir(workDir, repoCache)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	w := &worker{
		identity:    identity,
		client:      newProtocolClient(*url, *token),
		repoRoot:    *repoRoot,
		workDir:     workDir,
		keepsFilms:  keepsFilms,
		memLimitGiB: *memLimit,
	}
	slog.InfoContext(ctx, "replay-worker: démarré",
		"worker_id", identity.workerID, "url", *url, "work_dir", workDir,
		"films_conserves", keepsFilms, "once", *once, "mem_limit_gib", *memLimit)

	if err := w.run(ctx, *poll, *once); err != nil {
		slog.ErrorContext(ctx, "replay-worker: arrêt sur erreur", "err", err)
		os.Exit(1)
	}
	slog.InfoContext(ctx, "replay-worker: arrêté",
		"jobs_done", w.jobsDone, "jobs_failed", w.jobsFailed)
}

// sameDir dit si deux chemins désignent le même répertoire, à la casse et aux
// séparateurs près (Windows comme Linux). Sert l'unique décision « ai-je le droit
// d'effacer les morceaux de film ? » — mieux vaut conclure « c'est l'archive » à
// tort que l'inverse.
func sameDir(a, b string) bool {
	ca, err := filepath.Abs(filepath.Clean(a))
	if err != nil {
		return true
	}
	cb, err := filepath.Abs(filepath.Clean(b))
	if err != nil {
		return true
	}
	return strings.EqualFold(ca, cb)
}

// run : la boucle. Une erreur de PROTOCOLE (serveur injoignable, jeton refusé)
// arrête l'ouvrier ; une erreur de TRAVAIL (film illisible, carte hors
// catalogue) est rendue au serveur comme un échec de job, et la boucle continue —
// un ouvrier ne se suicide pas parce qu'un match est mauvais.
func (w *worker) run(ctx context.Context, poll time.Duration, once bool) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		claimed, err := w.client.claim(ctx, w.identity)
		if err != nil {
			return err
		}
		if claimed.Job == nil {
			if once {
				slog.InfoContext(ctx, "replay-worker: file vide, rien à faire")
				return nil
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(poll):
			}
			continue
		}
		w.processJob(ctx, claimed.Job)
		if once {
			return nil
		}
	}
}
