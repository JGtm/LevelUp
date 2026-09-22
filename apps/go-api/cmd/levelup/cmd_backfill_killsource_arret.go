package main

// cmd_backfill_killsource_arret.go — S ARRETER SANS RIEN PERDRE, ET LE DIRE (lot 5.24.4).
//
// # CE QU UN CTRL-C DOIT FAIRE, ET CE QU IL NE DOIT PAS FAIRE
//
// Une passe de backfill dure des heures. L interrompre est NORMAL — on rend la machine, on
// repart le lendemain. Ce qui ne doit pas arriver :
//
//	PERDRE LE TRAVAIL EN VOL   avec N ouvriers, un arret brutal jette N decodages entames,
//	                           jusqu a une minute de calcul chacun sur les gros films.
//	MOURIR AU MILIEU D UNE ECRITURE   la doctrine anti-corruption (ADR 0013/0019/0026) existe
//	                           precisement pour que cela n arrive pas.
//	SORTIR EN SILENCE          une passe interrompue qui rend le meme code qu une passe finie
//	                           ne se distingue pas dans un script.
//
// # LE PROCEDE, EN TROIS TEMPS
//
//	1er SIGNAL   `signal.NotifyContext` annule le contexte d ARRET DOUX. La distribution des
//	             films s arrete ; ceux qui sont en vol vont au bout, ECRITURE COMPRISE
//	             (`KillSourceCollector.AvecArretDoux`). Le contexte de TRAVAIL, lui, n est pas
//	             annule : aucun decodage n est coupe en deux.
//	APRES        le releveur de signaux est RENDU au systeme (`stop()`), donc un SECOND Ctrl-C
//	             tue le processus comme d habitude. Une passe qui refuserait de mourir au
//	             deuxieme signal serait pire que celle qui meurt au premier.
//	A LA FIN     l etat est ecrit avec sa cause, un message dit quoi faire, et le processus sort
//	             avec [CodeSortieInterrompue].
//
// # RIEN DE TOUT CELA N EST LA REPRISE
//
// La reprise ne depend d AUCUNE de ces pieces : elle se decide en base, sur `decoder_rev`, au
// demarrage de la passe suivante. Un `kill -9` en plein decodage se reprend exactement pareil —
// simplement, le film en cours sera redecode. Ce fichier ne rend pas la reprise possible, il
// rend l arret PROPRE et LISIBLE.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// CodeSortieInterrompue : le code de sortie d une passe arretee par un signal.
//
// 130 = 128 + SIGINT, la convention POSIX que tout shell comprend. Il se distingue donc du 0
// d une passe finie ET du 1 d une panne, ce qui est le seul point : un script qui rejoue la
// commande doit pouvoir faire la difference entre « interrompue, relance-moi » et « cassee ».
const CodeSortieInterrompue = 130

// interruptionDeLaPasse : l erreur qui porte le code de sortie dedie.
//
// C EST UNE ERREUR ET PAS UN RETOUR NORMAL, parce que la passe n a PAS fait ce qu on lui a
// demande : il reste des films. Ce n est simplement pas une PANNE, et le message le dit.
type interruptionDeLaPasse struct{ cause string }

func (e *interruptionDeLaPasse) Error() string {
	return fmt.Sprintf("passe interrompue (%s) — les films en cours ont ete termines et ecrits ; "+
		"REPRISE : relancer LA MEME commande, elle repart au dernier etat "+
		"(la reprise se decide en base sur `decoder_rev`, pas sur le fichier d etat)", e.cause)
}

// contexteDArret installe le releveur de signaux et rend le contexte d ARRET DOUX.
//
// `stop` est a appeler en `defer` : il rend le releveur au systeme. Un goroutine l appelle DEJA
// des le premier signal, pour qu un second Ctrl-C tue le processus — l appel differe reste
// necessaire pour le cas nominal (aucun signal recu).
func contexteDArret() (ctx context.Context, stop func()) {
	ctx, stop = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		fmt.Fprintln(os.Stderr,
			"\narret demande : plus aucun film n est distribue, les decodages en cours vont au "+
				"bout et sont ecrits. Un SECOND Ctrl-C tue le processus immediatement.")
		// LE RELEVEUR EST RENDU ICI, pas a la fin : sans cela le second signal serait avale et le
		// processus paraitrait refuser de mourir.
		stop()
	}()
	return ctx, stop
}

// causeDArret : pourquoi la passe s est arretee, ou "" si elle est allee au bout.
//
// ELLE SE LIT SUR LE CONTEXTE ET PAS SUR UNE VARIABLE PARTAGEE : le releveur de signaux vit dans
// un autre goroutine, et une chaine ecrite la-bas et lue ici serait une course — sur une donnee
// dont le contexte porte deja toute l information.
func causeDArret(ctx context.Context) string {
	if ctx == nil || ctx.Err() == nil {
		return ""
	}
	return "signal recu (SIGINT/SIGTERM)"
}

// sortirSur : le code de sortie d une erreur de sous-commande, et son message.
//
// Extrait pour que `main` n ait pas a connaitre le type d erreur d une sous-commande : il
// demande un code, il l obtient.
func sortirSur(err error) int {
	var interruption *interruptionDeLaPasse
	if errors.As(err, &interruption) {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return CodeSortieInterrompue
	}
	fmt.Fprintf(os.Stderr, "erreur: %v\n", err)
	return 1
}
