// Command devlaunch : lanceur de développement du serveur, pour air (apps/go-api/.air.toml).
//
// Problème : sous Windows air arrête le binaire qu'il a lancé par TASKKILL /F /T (force-kill de
// l'arbre de processus) ; le serveur ne passe jamais par sa séquence d'arrêt et laisse des WAL.
//
// Contrat : air lance devlaunch (processus A), jamais le serveur. A tient le bout ÉCRIVAIN du
// stdin du serveur ; le serveur, lancé avec -arret-sur-stdin, voit EOF quand A meurt (tué par
// air, Ctrl+C, fermeture du terminal : l'OS ferme le tube) et s'arrête proprement. C'est le
// contrat de la coquille desktop, joué tous les jours par le poste de dev.
//
// Pour que TASKKILL /T sur A n'atteigne pas le serveur, celui-ci ne doit pas être un descendant
// de A : A relance une copie de lui-même (B, -spawn) avec le lecteur du tube en stdin ; B lance
// le serveur avec ce même stdin hérité puis sort aussitôt. Le parent du serveur n'existe plus,
// A bloque jusqu'à sa mort.
//
//	devlaunch <chemin du serveur> [args du serveur...]   ajoute -arret-sur-stdin
//	devlaunch -spawn <chemin du serveur> [args...]       interne : rôle B
//
// Avant de lancer, A attend que le port du serveur (LEVELUP_API_PORT, défaut 8000) soit libre :
// le serveur précédent s'arrête parce que son lanceur vient de mourir. Borne : portWaitTimeout.
package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const (
	spawnFlag       = "-spawn"
	stdinStopFlag   = "-arret-sur-stdin"
	defaultPort     = 8000
	portWaitTimeout = 45 * time.Second
	portPollEvery   = 200 * time.Millisecond
)

// writerKeepAlive retient le bout écrivain du tube : s'il était collecté, son finaliseur le
// fermerait et le serveur s'arrêterait aussitôt.
var writerKeepAlive *os.File

func main() { os.Exit(run(os.Args[1:])) }

// run rend le code de sortie ; le rôle A ne rend jamais la main hors erreur.
func run(args []string) int {
	if len(args) >= 2 && args[0] == spawnFlag {
		return spawnServer(args[1], args[2:])
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage : devlaunch <chemin du serveur> [args du serveur...]")
		return 2
	}
	return launch(args[0], args[1:])
}

// launch est le rôle A : attend le port, crée le tube, confie le lecteur à B, garde l'écrivain.
func launch(server string, serverArgs []string) int {
	port := apiPort()
	if err := waitPortFree(port, portWaitTimeout, portPollEvery); err != nil {
		fmt.Fprintf(os.Stderr, "devlaunch : %v\n", err)
		return 1
	}
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "devlaunch : exécutable introuvable : %v\n", err)
		return 1
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "devlaunch : création du tube : %v\n", err)
		return 1
	}
	b := exec.Command(self, append([]string{spawnFlag, server, stdinStopFlag}, serverArgs...)...)
	b.Stdin, b.Stdout, b.Stderr = reader, os.Stdout, os.Stderr
	if err := b.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "devlaunch : lancement du serveur %s : %v\n", server, err)
		return 1
	}
	_ = reader.Close() // le serveur tient le seul lecteur ; A garde l'écrivain
	writerKeepAlive = writer
	fmt.Fprintf(os.Stderr, "devlaunch : serveur lancé (%s), port %d ; il s'arrêtera proprement à la mort de ce processus\n", server, port)
	for {
		time.Sleep(time.Hour) // meurt tué par air ; pas de select{} : le détecteur d'interblocage le refuserait
	}
}

// spawnServer est le rôle B : lance le serveur avec le stdin hérité (le lecteur du tube) puis
// sort sans l'attendre, pour que le serveur n'ait plus de parent à tuer.
func spawnServer(server string, serverArgs []string) int {
	cmd := exec.Command(server, serverArgs...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "devlaunch : %v\n", err)
		return 1
	}
	return 0
}

// apiPort lit LEVELUP_API_PORT (défaut 8000 ; valeur illisible = défaut). Seul l'environnement
// est lu : un port posé seulement dans .env.local n'est pas vu, et l'attente ne porte alors que
// sur le port par défaut.
func apiPort() int {
	if p, err := strconv.Atoi(os.Getenv("LEVELUP_API_PORT")); err == nil && p > 0 && p < 65536 {
		return p
	}
	return defaultPort
}

// waitPortFree attend que plus rien n'écoute sur 127.0.0.1:port (connexion refusée), au plus
// timeout. Le sondage est une simple tentative de connexion : il ne prend jamais le port.
func waitPortFree(port int, timeout, every time.Duration) error {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("tcp", addr, every)
		if err != nil {
			return nil
		}
		_ = conn.Close()
		if time.Now().After(deadline) {
			return fmt.Errorf("le port %d est toujours occupé après %s : l'ancien serveur ne s'arrête pas", port, timeout)
		}
		time.Sleep(every)
	}
}
