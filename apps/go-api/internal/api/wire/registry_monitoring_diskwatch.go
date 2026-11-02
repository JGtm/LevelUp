// Package api — registry_monitoring_diskwatch.go : surveillance périodique du
// disque du volume data (lot ops 2026-07-13, suite incident disque-plein VPS).
//
// Pourquoi côté serveur Go et pas côté hôte : le volume data est un bind mount
// du filesystem hôte → l'espace libre mesuré dans le conteneur EST celui de
// l'hôte. L'alerte hôte historique (levelup-disk-check.sh → journald) n'était
// lue par personne ; ici le dépassement de seuil produit :
//   - un log WARN/ERROR à message STABLE → détection persistée du monitoring
//     (ErrorCollector → detections_latest, badge admin) ;
//   - une notification Discord si le webhook est configuré (transition de
//     statut + rétablissement, sans rappel périodique — politique
//     ops.ShouldNotifyDisk, V72-31) ;
//   - des gauges expvar (levelup.disk_data_*).
//
// Mesure = r.resourceDisk() (même source que l'onglet Ressources admin, seuils
// nommés A5.3 : absolus 2 Go/500 Mo + pourcentage 80/90).
package wire

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/adminstate"
)

// diskWatchInterval : période de vérification. Un disque se remplit en heures
// (sync + médias) — 15 min suffit largement et le statfs est trivial.
const diskWatchInterval = 15 * time.Minute

// RunDiskWatchLoop boucle la surveillance disque jusqu'à ctx.Done(). Premier
// check immédiat (un boot sur disque déjà plein alerte tout de suite). Câblée
// au boot sur schedulerCtx/schedulerWG comme RunDetectionFlushLoop.
//
// L'état anti-spam (ops.DiskWatchState) est PERSISTÉ hors DuckDB et réhydraté ici
// au boot : sans persistance, un redémarrage du conteneur en warn/critical (deploy
// à chaque push main, crash-loop disque plein) re-notifiait via le chemin « boot »
// à CHAQUE boot → rafale d'alertes Discord identiques observée en prod.
//
// L'ARCHIVE DES FILMS, quand elle vit sur un AUTRE VOLUME que les données (réglage `film_archive_dir`,
// lot L14), est surveillée par la MÊME boucle, aux mêmes seuils, avec son propre état anti-spam :
// un disque d'archive qui se remplit ne doit ni masquer ni être masqué par celui des données.
func (r *ServiceRegistry) RunDiskWatchLoop(ctx context.Context) {
	pr := r.cfg.PathResolver()
	data := &volumeWatch{
		name: "data", store: adminstate.NewFileStore(pr.DiskWatchStatePath()),
		measure: r.resourceDisk, gauge: "disk_data",
		msgCritical: "disk_watch: espace disque CRITIQUE sur le volume data",
		msgWarn:     "disk_watch: espace disque faible sur le volume data",
	}
	watches := []*volumeWatch{data}
	if archive := r.filmArchiveWatch(); archive != nil {
		watches = append(watches, archive)
	}
	for _, w := range watches {
		w.state = loadDiskWatchState(ctx, w.store)
		w.state = r.diskWatchTick(ctx, w)
	}
	t := time.NewTicker(diskWatchInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for _, w := range watches {
				w.state = r.diskWatchTick(ctx, w)
			}
		}
	}
}

// volumeWatch : un volume surveillé, avec son état anti-spam persisté et ses messages (STABLES :
// ils forment l'empreinte de détection).
type volumeWatch struct {
	name        string
	store       *adminstate.FileStore
	state       ops.DiskWatchState
	measure     func() domain.ResourceDisk
	gauge       string // préfixe des gauges expvar : <gauge>_free_bytes, <gauge>_used_percent
	msgCritical string
	msgWarn     string
}

// filmArchiveWatch rend la surveillance du volume de l'archive des films, ou nil quand elle vit
// sur le même volume que les données (déjà surveillé, une seule surveillance par volume).
func (r *ServiceRegistry) filmArchiveWatch() *volumeWatch {
	return filmArchiveWatchFor(r.cfg.PathResolver())
}

// filmArchiveWatchFor est filmArchiveWatch pour un résolveur donné.
func filmArchiveWatchFor(pr *titlePkg.PathResolver) *volumeWatch {
	dir := pr.FilmArchiveDir()
	// UN VOLUME, UNE SURVEILLANCE : on compare les volumes et non les chemins. Une archive hors de
	// la racine des donnees mais sur le meme disque doublerait journaux et notifications.
	if same, err := ops.SameVolume(dir, pr.DataRoot()); err == nil && same {
		return nil
	}
	return &volumeWatch{
		name:  "film_archive",
		store: adminstate.NewFileStore(filmArchiveWatchStatePath(pr.DiskWatchStatePath())),
		measure: func() domain.ResourceDisk {
			out := domain.ResourceDisk{Path: dir}
			vol, err := ops.MeasureVolume(dir)
			if err != nil {
				out.Status, out.Error = domain.FreshnessStatusUnknown, err.Error()
				return out
			}
			out.FreeBytes, out.TotalBytes, out.Status = vol.FreeBytes, vol.TotalBytes, vol.Status
			return out
		},
		gauge:       "disk_film_archive",
		msgCritical: "disk_watch: espace disque CRITIQUE sur le volume de l'archive des films",
		msgWarn:     "disk_watch: espace disque faible sur le volume de l'archive des films",
	}
}

// filmArchiveWatchStatePath : l'état anti-spam de l'archive, à côté de celui des données.
func filmArchiveWatchStatePath(dataStatePath string) string {
	ext := filepath.Ext(dataStatePath)
	return strings.TrimSuffix(dataStatePath, ext) + "_film_archive" + ext
}

// loadDiskWatchState réhydrate l'état persisté. Fichier absent (premier boot) →
// état zéro (le premier relevé sera confirmé d'emblée). Fichier corrompu → état
// zéro APRÈS un log (jamais avalé) — au pire une notification de plus, pas une
// rafale : on ne supprime pas le fichier (un opérateur peut l'inspecter).
func loadDiskWatchState(ctx context.Context, store *adminstate.FileStore) ops.DiskWatchState {
	var state ops.DiskWatchState
	found, err := store.Load(&state)
	if err != nil {
		monitoringLog.WarnContext(ctx, "disk_watch: état persisté illisible — démarrage à vide",
			"path", store.Path(), "err", err)
		return ops.DiskWatchState{}
	}
	if found {
		monitoringLog.InfoContext(ctx, "disk_watch: état réhydraté",
			"last_status", state.LastStatus, "last_notified_at", state.LastNotifiedAt)
	}
	return state
}

// saveDiskWatchState persiste l'état après chaque tick (best-effort : un échec
// d'écriture est LOGGÉ mais ne casse pas la boucle — l'état mémoire fait foi
// jusqu'au prochain tick). Écriture atomique via FileStore (survit à un kill).
func saveDiskWatchState(ctx context.Context, store *adminstate.FileStore, state ops.DiskWatchState) {
	if err := store.Save(state); err != nil {
		monitoringLog.WarnContext(ctx, "disk_watch: persistance de l'état échouée (état mémoire conservé)",
			"path", store.Path(), "err", err)
	}
}

// diskWatchTick mesure, logge, met à jour les gauges et notifie si la politique
// le décide. Persiste le nouvel état (anti-spam à travers les restarts) et le
// retourne.
func (r *ServiceRegistry) diskWatchTick(ctx context.Context, w *volumeWatch) ops.DiskWatchState {
	disk := w.measure()
	usedPct := ops.DiskUsedPercent(disk.FreeBytes, disk.TotalBytes)

	observability.SetInt(w.gauge+"_free_bytes", int64(disk.FreeBytes))
	observability.SetInt(w.gauge+"_used_percent", int64(usedPct))

	// Log à message STABLE (fingerprint de détection = title+level+module+message,
	// les valeurs vivent dans les attributs). warn/critical = niveaux distincts →
	// deux détections distinctes, chacune avec son cycle de vie.
	attrs := []any{
		"module", "monitoring", "volume", w.name, "path", disk.Path, "status", disk.Status,
		"free_bytes", disk.FreeBytes, "total_bytes", disk.TotalBytes,
		"used_percent", int64(usedPct),
	}
	switch disk.Status {
	case domain.FreshnessStatusCritical:
		slog.ErrorContext(ctx, w.msgCritical, attrs...)
	case domain.FreshnessStatusWarn:
		slog.WarnContext(ctx, w.msgWarn, attrs...)
	case domain.FreshnessStatusUnknown:
		// Mesure indisponible : visible mais pas une alerte (pas de détection ERROR).
		slog.WarnContext(ctx, "disk_watch: mesure disque indisponible", "module", "monitoring", "volume", w.name, "err", disk.Error)
	default:
		slog.DebugContext(ctx, "disk_watch: disque ok", attrs...)
	}

	shouldNotify, next := ops.ShouldNotifyDisk(w.state, disk.Status, time.Now())
	if shouldNotify {
		observability.IncCounter("disk_watch_notifications_total")
		cfg := notify.LoadNotifyConfig(r.cfg.AppSettingsPath)
		if sent := notify.NotifyDiskAlert(cfg, disk.Status, disk.Path, disk.FreeBytes, disk.TotalBytes, usedPct); sent {
			monitoringLog.InfoContext(ctx, "disk_watch: notification Discord envoyée", "status", disk.Status, "volume", w.name)
		} else if cfg.WebhookURL == "" {
			// Pas de webhook configuré : l'alerte reste visible via la détection
			// persistée + le badge admin. Log une info pour la traçabilité.
			monitoringLog.InfoContext(ctx, "disk_watch: alerte sans notification (webhook Discord non configuré)",
				"status", disk.Status, "volume", w.name)
		}
	}
	// Persister l'état APRÈS la décision (statut confirmé + timestamp de dernière
	// notification + débounce) pour que l'anti-spam et la garantie « rétablissement
	// notifié une seule fois » survivent aux redémarrages du conteneur.
	if next != w.state {
		saveDiskWatchState(ctx, w.store, next)
	}
	return next
}
