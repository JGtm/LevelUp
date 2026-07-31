package handlers

import (
	"net"
	"net/http"
	"os"
	"strings"
)

// replay_local_gate.go — LE REJEU 2D N'EST SERVI QU'EN LOCAL, POUR L'INSTANT.
//
// POURQUOI CE GARDE EXISTE, dans les termes de l'utilisateur (2026-07-28) : « on a des écarts
// entre la réalité et ce qu'on sait sortir. Je sais pas si ça dépasse le stade du POC mais en
// tout cas c'est pas productionnalisable. Sinon ok pour sortir une version activée que sur le
// localhost. »
//
// CE QUI EST EN CAUSE, chiffré : le rejeu publie 475 tirs sur les 519 que le film porte et 70
// lancers sur 70, mais 15 vies sur 105 restent sans identité (survivants de fin de match,
// entités d'avant le coup d'envoi, trois morts hors fenêtre d'appariement). Ces écarts sont
// mesurés et publiés dans `coverage` — ils ne sont pas cachés — mais ils n'ont pas encore été
// confrontés à autre chose qu'un seul film.
//
// CE GARDE N'EST PAS UN INTERRUPTEUR « POUR PLUS TARD » (règle du dépôt sur les kill-switches).
// Il porte les trois éléments exigés :
//
//	BASCULÉ LE          2026-07-28, à l'ouverture du rejeu en local.
//	RETRAIT CIBLE       à la première confrontation réussie sur un SECOND film.
//	CRITÈRE MESURABLE   couverture des tirs > 85 % et `coverage.verdict.bridge` nominal sur au
//	                    moins deux films de cartes différentes, sans collision de trace.
//
// Le critère est celui du plan de fiabilisation, et il se lit dans l'artefact lui-même : nul
// besoin d'un jugement pour décider du retrait.
//
// `LEVELUP_REPLAY_PUBLIC=1` lève le garde — utile à qui veut exposer le rejeu en connaissance
// de cause, sans avoir à recompiler.

// replayPublic lève le garde local. Lu une seule fois : ce réglage ne change pas en cours de
// vie du processus, et le relire à chaque requête coûterait un appel système par appel d'API.
var replayPublic = os.Getenv("LEVELUP_REPLAY_PUBLIC") == "1"

// isLocalRequest dit si la requête vient de la machine qui héberge l'API.
//
// ON NE FAIT PAS CONFIANCE AUX EN-TÊTES. `X-Forwarded-For` est fourni par le client ; s'en
// servir ici transformerait le garde en simple suggestion. Seule l'adresse de la connexion
// TCP compte, et elle n'est pas falsifiable par le demandeur.
func isLocalRequest(r *http.Request) bool {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = h
	}
	// Un hôte vide arrive sur certains transports de test ; le traiter comme local évite de
	// rendre les tests dépendants d'une pile réseau.
	if host == "" {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// allowReplay dit si la requête a le droit d'obtenir un rejeu.
func allowReplay(r *http.Request) bool {
	return replayPublic || isLocalRequest(r)
}
