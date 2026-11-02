// Command fake-sidecar : le FAUX SERVEUR du contrat coquille <-> serveur (ADR 0045 D-21, A1).
//
// Il joue, sans aucune dependance du serveur reel, le serveur que la coquille Tauri lance
// (`-mode desktop -arret-sur-stdin`) : il lie un port, ecrit les evenements du contrat sur
// stdout, sert l'entree de la coquille et s'arrete a la fermeture de stdin. Les tests `cargo test`
// de la coquille le compilent et le lancent pour exercer chaque cas.
//
// Environnement lu (les trois racines et le secret sont ceux que pose la coquille) :
//
//	LEVELUP_DESKTOP_SECRET   secret d'entree ; absent ou present dans les arguments = echec
//	LEVELUP_DATA_DIR         racine de donnees : <racine>/desktop/port est lu puis ecrit comme le
//	                         fait le serveur reel ; le marqueur d'arret propre y est ecrit
//	LEVELUP_RESOURCES_DIR, LEVELUP_LOGS_DIR   reproduits sur stderr avec le repertoire courant
//
// Variantes et delais (variables FAKE_*, voir readOptions) :
//
//	FAKE_VARIANT = ok (defaut) | never_ready | failed | failed_no_port | late_stopped | crash |
//	               crash_after_ready
//	FAKE_CHATTY_BYTES      octets de lignes non JSON ecrites sur stdout avant `ready` (bavard)
//	FAKE_LISTEN_DELAY_MS, FAKE_READY_DELAY_MS, FAKE_STOP_DELAY_MS, FAKE_CRASH_AFTER_MS
//	FAKE_FAILED_STATE      etat de l'evenement `failed` (defaut migration_failed)
package main

import (
	"fmt"
	"os"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fake: repertoire courant illisible : %v\n", err)
		os.Exit(exitSetup)
	}
	os.Exit(run(os.Args[1:], os.Getenv, cwd, os.Stdin, os.Stdout, os.Stderr))
}

// Codes de sortie propres au faux serveur ; 3 est le « crash » du contrat de test.
const (
	exitOK       = 0
	exitFailed   = 1 // le demarrage est refuse (comme un serveur reel apres `failed`)
	exitSetup    = 2 // arguments ou variantes inconnus
	exitCrash    = 3
	exitSecret   = 4 // le secret apparait dans les arguments
	exitNoSecret = 5 // le secret manque dans l'environnement
	exitBind     = 6
)

// expectedArgs : la ligne de commande exacte du contrat (D-1), sans le secret.
var expectedArgs = []string{"-mode", "desktop", "-arret-sur-stdin"}
