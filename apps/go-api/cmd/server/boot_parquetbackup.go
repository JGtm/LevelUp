// boot_parquetbackup.go — la sauvegarde périodique en Parquet du serveur desktop (lot L22, ADR 0045) :
// relie le service internal/ops/parquetbackup aux réglages vivants (`backup_dir`, `backup_interval`,
// `backup_keep`) et lance la boucle qui écrit un jeu quand il est dû. Ne tourne que sous la capacité
// ParquetBackup (profil desktop) ; la mécanique d'export et de rotation est dans le paquet.
package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ops/parquetbackup"
	"levelup/go-api/internal/platform/gameproc"
	"levelup/go-api/internal/platform/settings"
)

const (
	// parquetBackupPoll : la boucle relit les réglages et l'état toutes les minutes ; l'intervalle de sauvegarde
	// (heures) se juge sur ce rythme.
	parquetBackupPoll = time.Minute
	// parquetBackupRetryAfter : après un échec, nouveau passage au bout d'une heure plutôt que d'un intervalle plein.
	parquetBackupRetryAfter = time.Hour
)

// backupOptionsFrom construit les options d'un passage depuis les réglages vivants : ErrDisabled tant qu'aucun
// dossier n'est choisi.
func backupOptionsFrom(cfg *config.AppConfig, store settingsLoader) func() (parquetbackup.Options, error) {
	return func() (parquetbackup.Options, error) {
		s, err := store.Load()
		if err != nil {
			return parquetbackup.Options{}, err
		}
		dir := strings.TrimSpace(s.BackupDir)
		if dir == "" {
			return parquetbackup.Options{}, parquetbackup.ErrDisabled
		}
		return parquetbackup.Options{
			Paths: cfg.PathResolver(), AuthDir: cfg.AuthDir, Dest: dir, Keep: s.BackupKeep, AppVersion: cfg.AppVersion,
		}, nil
	}
}

// launchParquetBackup relie le service du processus et lance la boucle périodique sous la capacité
// ParquetBackup. Elle tourne sous le contexte des tâches de fond et dans leur groupe d'attente : l'arrêt
// annule un passage en cours puis l'attend avant la fermeture des bases.
func launchParquetBackup(st *bootState) {
	if !st.cfg.Capabilities.ParquetBackup || st.settingsStore == nil {
		return
	}
	svc := parquetbackup.Default()
	svc.Bind(st.schedulerCtx, &st.schedulerWG, backupOptionsFrom(st.cfg, st.settingsStore))
	ctx := st.schedulerCtx
	st.schedulerWG.Add(1)
	go func() {
		defer st.schedulerWG.Done()
		runParquetBackupLoop(ctx, svc, st.settingsStore, nil, parquetBackupPoll)
	}()
	slog.InfoContext(ctx, "backup parquet: boucle armée (réglage backup_dir)", "sondage", parquetBackupPoll)
}

// backupRunner : ce que la boucle demande au service (satisfait par *parquetbackup.Service).
type backupRunner interface {
	Due(dest string, interval, retryAfter time.Duration, now time.Time) bool
	RunNow(ctx context.Context) (*parquetbackup.Manifest, error)
}

// runParquetBackupLoop relit les réglages à chaque tic (ticks nil : un ticker de every, arrêté avec la boucle)
// et écrit un jeu quand il est dû. Un passage ne démarre jamais pendant une partie (porte du mode jeu,
// consultée AVANT le lancement) ; l'échec est journalisé par le service, la boucle continue.
func runParquetBackupLoop(ctx context.Context, svc backupRunner, store settingsLoader, ticks <-chan time.Time, every time.Duration) {
	if ticks == nil {
		t := time.NewTicker(every)
		defer t.Stop()
		ticks = t.C
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			backupTick(ctx, svc, store)
		}
	}
}

// backupTick : un tic de la boucle.
func backupTick(ctx context.Context, svc backupRunner, store settingsLoader) {
	s, err := store.Load()
	if err != nil {
		slog.WarnContext(ctx, "backup parquet: réglages illisibles, tic ignoré", "err", err)
		return
	}
	dir := strings.TrimSpace(s.BackupDir)
	if dir == "" || !svc.Due(dir, settings.BackupIntervalDuration(s.BackupInterval), parquetBackupRetryAfter, time.Now()) {
		return
	}
	err = gameproc.Default().WaitHeavyAllowed(ctx, func() {
		slog.InfoContext(ctx, "game_mode: sauvegarde différée jusqu'à la fin de la partie")
	})
	if err != nil {
		return
	}
	if _, err := svc.RunNow(ctx); err != nil && ctx.Err() == nil {
		slog.WarnContext(ctx, "backup parquet: passage périodique sans jeu", "err", err)
	}
}
