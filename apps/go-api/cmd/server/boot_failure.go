// boot_failure.go — un démarrage qui ne peut pas aboutir reste visible, puis sort en erreur.
//
// Un échec de migration ne se dégrade jamais en « démarrage sans migration » : le serveur
// refuserait alors de servir des données dont le schéma n'est pas celui que son code attend.
// Il reste debout sur la porte de démarrage (boot_gate.go), qui sert l'état d'échec en 503 non
// réessayable pour que la page l'affiche au lieu d'attendre, garde le port jusqu'au signal
// d'arrêt, puis le processus sort avec un code non nul.
//
// bootFailureState est aussi le vocabulaire de l'événement stdout `failed` du contrat
// coquille <-> serveur (desktop_events.go) : refuseStart l'annonce avec l'état posé sur la porte.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/instancelock"
)

// bootFailureExitCode : code de sortie d'un démarrage refusé, rendu une fois l'arrêt demandé.
const bootFailureExitCode = 1

// bootFailureShutdownTimeout borne la fermeture du serveur HTTP d'un démarrage refusé : aucune
// base n'est ouverte, seules les requêtes de la porte sont en vol.
const bootFailureShutdownTimeout = 5 * time.Second

// classifyBootFailure range une erreur de migration dans un état d'échec : schéma plus récent,
// base tenue par un autre processus, sinon échec de migration.
func classifyBootFailure(err error) bootFailureState {
	switch {
	case errors.Is(err, migration.ErrSchemaNewerThanBinary):
		return bootFailureSchemaNewer
	case isBootFileLock(err):
		return bootFailureLocked
	default:
		return bootFailureMigrationFailed
	}
}

// isBootFileLock : l'erreur vient d'un verrou de fichier tenu par un autre processus.
func isBootFileLock(err error) bool {
	return errors.Is(err, duckdb.ErrBaseTenueEnEcriture) || duckdb.IsFileLockError(err)
}

// fail pose l'état d'échec : dès lors la porte répond 503 server_start_failed non réessayable.
// Appelée par main seul, une fois.
func (g *bootGate) fail(state bootFailureState) { g.failure.Store(&state) }

// failureState rend l'état d'échec posé par fail, et false si le démarrage n'a pas échoué.
func (g *bootGate) failureState() (bootFailureState, bool) {
	if s := g.failure.Load(); s != nil {
		return *s, true
	}
	return "", false
}

// refuseStart journalise le refus de démarrer (Error), pose l'état d'échec sur la porte, annonce
// l'événement `failed` (mode desktop), garde le port jusqu'au signal d'arrêt, ferme le serveur HTTP
// et rend le code de sortie non nul. L'événement vient APRÈS l'état posé : la coquille qui le lit
// charge la page de la porte, qui doit déjà répondre l'état.
func (b *bootServer) refuseStart(ctx context.Context, state bootFailureState, err error) int {
	slog.ErrorContext(ctx, "démarrage refusé", "err", err, "state", state)
	// Pas d'initialisation à finir : le premier signal d'arrêt suffit, sans second signal.
	b.initDone.Store(true)
	b.gate.fail(state)
	b.events.Failed(state)
	<-b.stopRequested()
	shutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), bootFailureShutdownTimeout)
	defer cancel()
	if shutErr := b.srv.Shutdown(shutCtx); shutErr != nil {
		slog.ErrorContext(ctx, "démarrage refusé : fermeture du serveur HTTP échouée", "err", shutErr)
	}
	return bootFailureExitCode
}

// exitAfterRefusal refuse le démarrage (refuseStart : porte debout jusqu'au signal d'arrêt), ferme
// les journaux fichier puis termine le processus avec le code non nul. Elle ne rend jamais la main.
func exitAfterRefusal(st *bootState, state bootFailureState, err error) {
	code := st.bootSrv.refuseStart(context.Background(), state, err)
	if st.multiHandler != nil {
		if cerr := st.multiHandler.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "fermeture logging multi-module: %v\n", cerr)
		}
	}
	os.Exit(code)
}

// refuseIfInstanceLockFailed refuse le démarrage si le verrou d'instance n'a pas été pris : une
// autre instance le tient (state locked, comme un verrou de base : la coquille ne distingue pas les
// deux), ou la racine de données n'a pu le recevoir (state migration_failed : la préparation du
// démarrage a échoué). Sans effet quand le verrou est tenu par ce processus.
func refuseIfInstanceLockFailed(st *bootState) {
	if st.instanceLockErr == nil {
		return
	}
	state := bootFailureMigrationFailed
	if errors.Is(st.instanceLockErr, instancelock.ErrInstanceDejaLancee) {
		state = bootFailureLocked
	}
	exitAfterRefusal(st, state, st.instanceLockErr)
}
