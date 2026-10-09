// Package api — registry_replay_notify.go : la NOTIFICATION GROUPÉE « rejeux 2D prêts »
// (lot B v7.5, point 5 de l'encadré Notion « REPLAY 2D »).
//
// TROIS GESTES, ET RIEN D'AUTRE :
//
//  1. au boot, câbler le puits d'artefacts de `internal/replaybuild` — le point d'écriture
//     unique par lequel passent la construction locale (fil de l'eau post-sync), la
//     livraison d'un ouvrier distant et l'action admin ;
//  2. battre la mesure : à chaque tick, demander au groupeur les lots échus ;
//  3. pour chaque lot, résoudre les liens (UNE lecture shared courte, via le repo) et
//     envoyer UN message.
//
// AUCUNE LOGIQUE MÉTIER ICI : le groupement vit dans `internal/replaynotify` (pur, horloge
// injectée), la lecture dans le repo `platform/duckdb`, le rendu du message dans
// `internal/notify`. Ce fichier est le câblage — c'est le rôle de la racine de `api/`.
//
// BEST-EFFORT DE BOUT EN BOUT : aucune erreur ne remonte, aucune n'est avalée non plus
// (chaque dégradation a son log et son compteur).
package wire

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/replaynotify"
)

// replayNotifyTick : pas de la boucle de flush. La fenêtre de groupement (10 min) est la
// grandeur qui compte ; ce tick n'en est que la résolution. Conséquence assumée : un lot
// sort entre T+10 et T+11 min après son premier artefact.
const replayNotifyTick = time.Minute

// replayNotifyLog route ces logs vers logs/notifications.log, comme le relais externe et
// l'émission « rival croisé » : diagnostiquer « pourquoi ma notif n'est pas partie » doit
// se faire d'un seul fichier.
var replayNotifyLog = slog.With("module", logging.ModuleNotif)

// replayGrouper : l'accumulateur du process. Singleton de paquet, comme `replayBuildMu` et
// `notifServicesByXUID` — la boucle et le puits doivent voir le MÊME état, et le puits est
// appelé depuis n'importe quel goroutine d'écriture.
var replayGrouper = replaynotify.New(0)

// InstallReplayNotify câble le puits d'artefacts. Appelé UNE SEULE FOIS au boot, avant le
// montage des routes ouvrier (un artefact peut arriver dès la première seconde).
func (r *ServiceRegistry) InstallReplayNotify() {
	replaybuild.SetArtifactStoredSink(func(ev replaybuild.ArtifactStored) {
		if ev.TitleSlug == "" || ev.MatchID == "" {
			// Jamais muet : un artefact sans identité ne peut être ni listé ni lié.
			// C'est un défaut d'appelant, pas un cas nominal.
			observability.IncCounter("replay_notify_events_invalid_total")
			replayNotifyLog.Warn("rejeu 2D : artefact rangé sans identité exploitable — non notifié",
				"title", ev.TitleSlug, "match_id", ev.MatchID, "path", ev.Path)
			return
		}
		observability.IncCounter("replay_notify_events_total")
		replayGrouper.Add(time.Now(), replaynotify.Event{
			TitleSlug: ev.TitleSlug, MatchID: ev.MatchID,
		})
	})
	replayNotifyLog.Info("rejeu 2D : notification groupée armée",
		"fenetre", replaynotify.DefaultWindow.String(), "tick", replayNotifyTick.String())
}

// RunReplayNotifyLoop boucle le flush jusqu'à ctx.Done(). Câblée au boot sur
// schedulerCtx/schedulerWG, comme RunDiskWatchLoop.
//
// PAS DE FLUSH FINAL À L'ARRÊT, ET C'EST LE CHOIX DOCUMENTÉ DU PLAN : le groupe en cours
// est perdu au redémarrage (cf. l'en-tête de internal/replaynotify). Vider la fenêtre à
// l'arrêt enverrait un message « prêt » à chaque déploiement, soit exactement le bruit
// qu'on cherche à éviter.
func (r *ServiceRegistry) RunReplayNotifyLoop(ctx context.Context) {
	t := time.NewTicker(replayNotifyTick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			r.flushReplayNotify(ctx, now)
		}
	}
}

// flushReplayNotify envoie les lots échus et met à jour la jauge d'attente.
func (r *ServiceRegistry) flushReplayNotify(ctx context.Context, now time.Time) {
	for _, b := range replayGrouper.Due(now) {
		r.sendReplayBatch(ctx, b)
	}
	titles, artifacts := replayGrouper.Pending()
	observability.SetInt("replay_notify_pending_titles", int64(titles))
	observability.SetInt("replay_notify_pending_artifacts", int64(artifacts))
}

// sendReplayBatch rend et envoie UN lot. Config Discord relue à chaque envoi (réactive aux
// PATCH /settings, comme disk_watch) et résolue POUR LE TITRE du lot : c'est ce qui rend le
// message title-agnostic (libellés et langue du titre, jamais de slug comparé).
func (r *ServiceRegistry) sendReplayBatch(ctx context.Context, b replaynotify.Batch) {
	cfg := notify.LoadNotifyConfigForTitle(
		r.cfg.AppSettingsPath,
		titlePkg.NewPathResolver(r.cfg.RepoRoot).TitleSettingsPath(b.TitleSlug))
	cfg.Labels = notify.LabelsForSlug(b.TitleSlug)
	observability.AddInt("replay_notify_artifacts_total", int64(b.Total))

	// Le lot est CONSOMMÉ quoi qu'il arrive (le groupeur l'a déjà retiré) : sans webhook,
	// on ne construit même pas les liens — une lecture base pour un message qui ne partira
	// pas serait du travail pur perte, à chaque fenêtre, sur une instance sans Discord.
	if cfg.WebhookURL == "" || !cfg.NotifyReplay {
		replayNotifyLog.InfoContext(ctx, "rejeu 2D : lot prêt, aucune notification émise",
			"title", b.TitleSlug, "artefacts", b.Total,
			"webhook", cfg.WebhookURL != "", "categorie_active", cfg.NotifyReplay)
		return
	}

	items := r.replayReadyItems(ctx, b)
	if notify.NotifyReplayBatch(cfg, items, b.Omitted) {
		observability.IncCounter("replay_notify_batches_sent_total")
		replayNotifyLog.InfoContext(ctx, "rejeu 2D : notification groupée envoyée",
			"title", b.TitleSlug, "artefacts", b.Total, "listes", len(items), "omis", b.Omitted)
		return
	}
	// Aucune retransmission : le rejeu reste disponible dans l'application, il ne vaut pas
	// une file de reprise. L'échec est compté et journalisé, jamais avalé.
	observability.IncCounter("replay_notify_failed_total")
	replayNotifyLog.WarnContext(ctx, "rejeu 2D : notification groupée non reçue par Discord — lot abandonné",
		"title", b.TitleSlug, "artefacts", b.Total)
}

// replayReadyItems construit les lignes du message : identifiant court, nom de carte quand
// la base le donne, lien quand un joueur connu du match ET une base publique existent.
func (r *ServiceRegistry) replayReadyItems(ctx context.Context, b replaynotify.Batch) []notify.ReplayReadyItem {
	targets, slugByXUID := r.replayLinkTargets(ctx, b)
	base := publicBaseURL()
	items := make([]notify.ReplayReadyItem, 0, len(b.MatchIDs))
	sansLien := 0
	for _, id := range b.MatchIDs {
		it := notify.ReplayReadyItem{MatchID: titlePkg.FilmShortMatchID(id)}
		if t, ok := targets[id]; ok {
			it.Label = t.MapName
			if slug := slugByXUID[t.XUID]; t.XUID != "" && slug != "" && base != "" {
				it.URL = base + notifications.PlayerTargetRoute(b.TitleSlug, slug, "matches/"+id+"/replay")
			}
		}
		if it.URL == "" {
			sansLien++
		}
		items = append(items, it)
	}
	if sansLien > 0 {
		// Dégradation NOMINALE (base publique absente en local, match sans joueur connu) :
		// la ligne s'affiche sans lien. Debug + compteur, jamais un échec.
		observability.AddInt("replay_notify_links_unresolved_total", int64(sansLien))
		replayNotifyLog.DebugContext(ctx, "rejeu 2D : lignes sans lien profond",
			"title", b.TitleSlug, "sans_lien", sansLien, "total", len(items),
			"base_publique", base != "")
	}
	return items
}

// replayLinkTargets lit, EN UNE FOIS pour tout le lot, les cibles de lien (joueur connu +
// nom de carte) et rend aussi la table xuid -> player_slug des joueurs de l'instance.
//
// OpenReadForQuery et JAMAIS OpenReadOnly : la shared du titre peut être tenue en écriture
// par le cycle de sync de ce même process (B-swap) — une ouverture RO forcée échouerait sur
// « different configuration ». Le handle est relâché au retour ; aucune écriture ici.
func (r *ServiceRegistry) replayLinkTargets(
	ctx context.Context, b replaynotify.Batch,
) (map[string]port.ReplayLinkTarget, map[string]string) {
	slugByXUID := map[string]string{}
	var xuids []string
	players, err := r.cfg.LoadPlayers(b.TitleSlug)
	if err != nil {
		replayNotifyLog.WarnContext(ctx, "rejeu 2D : joueurs connus illisibles — message sans liens",
			"title", b.TitleSlug, "err", err)
	}
	for _, p := range players {
		if p.XUID == "" || p.PlayerSlug == "" {
			continue
		}
		if _, dup := slugByXUID[p.XUID]; dup {
			continue
		}
		slugByXUID[p.XUID] = p.PlayerSlug
		xuids = append(xuids, p.XUID)
	}

	sharedPath := titlePkg.NewPathResolver(r.cfg.RepoRoot).SharedDBPath(b.TitleSlug)
	sqlDB, release, err := duckdb.OpenReadForQuery(sharedPath)
	if err != nil {
		observability.IncCounter("replay_notify_link_read_failed_total")
		replayNotifyLog.WarnContext(ctx, "rejeu 2D : shared indisponible — message sans liens ni cartes",
			"title", b.TitleSlug, "err", err)
		return nil, slugByXUID
	}
	defer release()

	var repo port.ReplayLinkRepo = duckdb.NewReplayFactsRepo(sqlDB)
	targets, err := repo.LinkTargetsForMatches(ctx, b.MatchIDs, xuids)
	if err != nil {
		observability.IncCounter("replay_notify_link_read_failed_total")
		replayNotifyLog.WarnContext(ctx, "rejeu 2D : cibles de lien illisibles — message sans liens ni cartes",
			"title", b.TitleSlug, "err", err)
		return nil, slugByXUID
	}
	return targets, slugByXUID
}

// publicBaseURL rend la base publique de l'application, sans « / » final. Vide = aucune
// base configurée (cas nominal en local) : les messages sortent alors sans lien profond.
//
// SOURCE UNIQUE dans ce paquet : le relais externe des notifications lit la même variable
// (registry_notifications.go). Deux lectures littérales auraient divergé au premier
// changement de nom.
func publicBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv("LEVELUP_PUBLIC_BASE_URL")), "/")
}
