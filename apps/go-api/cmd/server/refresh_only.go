// refresh_only.go — le mode -mode refresh-only (ADR 0045 D-10, A10) : un processus du MÊME binaire,
// lancé par une tâche planifiée du poste, qui rafraîchit les jetons des profils suivis puis sort.
//
// Il prend le verrou d'instance (<DataRoot>/levelup.lock), n'ouvre AUCUNE base, ne lie aucun port et
// n'écrit aucun événement sur stdout (ce n'est pas le mode `desktop` de la coquille) : le journal va
// sur stderr et sous la racine des journaux. Un verrou déjà tenu veut dire que l'application est
// ouverte, et c'est elle qui rafraîchit : le processus sort en 0 sans rien faire, parce qu'un refresh
// token ne se présente qu'à UN processus à la fois (il tourne à chaque usage, ADR 0023).
//
// La vérification des jetons est celle de la séquence de réveil (internal/wakeup) : un jeton vivant
// est rafraîchi et le jeton tourné écrit, un jeton mort pose reauth_required (la bannière de
// reconnexion de l'application), un réseau absent ne marque rien. Aucun de ces résultats ne change
// le code de sortie : la tâche planifiée repasse, et seul un échec à poser le verrou ou à lire la
// configuration est une panne.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/instancelock"
	"levelup/go-api/internal/platform/settings"
)

// refreshOnlyTimeout borne tout le processus : une tâche planifiée qui reste pendue bloquerait le
// verrou d'instance et empêcherait l'ouverture de l'application.
const refreshOnlyTimeout = 10 * time.Minute

// refreshOnlyLockError : code de sortie quand le verrou d'instance n'a pu être posé pour une autre
// raison qu'une instance déjà lancée (la racine de données est inutilisable).
const refreshOnlyLockError = 1

// runRefreshOnly est le point d'entrée du mode : journaux, configuration, puis le rafraîchissement.
// Il rend le code de sortie du processus, journaux fermés.
func runRefreshOnly(st *bootState) int {
	initProcessHooks()
	setupLogging(st)
	defer func() {
		if st.multiHandler != nil {
			if err := st.multiHandler.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "fermeture logging multi-module: %v\n", err)
			}
		}
	}()
	loadBootConfig(st)
	ctx, stop := signal.NotifyContext(st.ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	provider := buildTokenProvider(settings.NewStore(st.cfg.AppSettingsPath), title.DefaultHaloAuthDescriptor())
	return refreshOnly(ctx, st.cfg, provider)
}

// refreshOnly prend le verrou d'instance puis vérifie les jetons de chaque profil suivi. Rend le code
// de sortie : 0 (fait, ou verrou tenu par l'application), refreshOnlyLockError si le verrou n'a pu
// être posé.
func refreshOnly(ctx context.Context, cfg *config.AppConfig, provider auth.TokenProvider) int {
	ctx, cancel := context.WithTimeout(ctx, refreshOnlyTimeout)
	defer cancel()

	lockPath := cfg.PathResolver().InstanceLockPath()
	lock, err := instancelock.Acquire(lockPath)
	switch {
	case errors.Is(err, instancelock.ErrInstanceDejaLancee):
		slog.InfoContext(ctx, "refresh-only: une instance tient le dossier de données — rien à faire, "+
			"elle rafraîchit elle-même (une seule rotation à la fois, ADR 0023)", "verrou", lockPath)
		return 0
	case err != nil:
		slog.ErrorContext(ctx, "refresh-only: verrou d'instance non pris — jetons non rafraîchis",
			"verrou", lockPath, "err", err)
		return refreshOnlyLockError
	}
	defer func() {
		if rerr := lock.Release(); rerr != nil {
			slog.WarnContext(ctx, "refresh-only: libération du verrou d'instance échouée", "err", rerr)
		}
	}()

	report := newTokenCheck(cfg, provider).Check(ctx, nil)
	slog.InfoContext(ctx, "refresh-only: terminé", "profils", len(report.Results),
		"reseau_absent", report.NetworkAbsent())
	return 0
}
