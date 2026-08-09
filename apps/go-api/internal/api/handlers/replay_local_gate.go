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
// CE QUI EST EN CAUSE, chiffré. Le rejeu ne place pas tout ce que le film porte, et la perte se
// concentre là où ça se voit le plus : le DERNIER DÉCILE d'un match perd de 40 à 74 % de ses
// tirs, dans TOUS les modes. Cause établie le 2026-08-08 (`.ai/V7.5/RECHERCHE_CTF_TIRS_PERDUS.md`)
// : le pont nomme une vie par la mort qui la termine, donc un joueur qui cesse de mourir cesse
// d'être localisable. Les FERMETURES (`analysis/replay/closures.go`) ont depuis porté les sept
// films de 79,7-93,4 % à 88,7-96,4 %.
//
// CORPUS REMESURÉ LE 2026-08-09, après la ronde de correction des fermetures (revue adversariale :
// deux déductions infondées retirées). Trois films reculent — le plus fort de 95,0 à 91,6 % — sans
// que la borne basse bouge. **La marge sur le critère est mince : 88,68 % sur `829abef9`, pour un
// plancher à 88 %.** Tout changement du pont doit donc rejouer le corpus AVANT de conclure, jamais
// après (cf. §7.5bis de la recherche).
//
// ATTENTION AU DÉNOMINATEUR, ET C'EST LA PREMIÈRE CHOSE À COMPRENDRE AVANT DE RETIRER CE GARDE :
// ces taux portent sur les tirs QUE LE FILM CONTIENT, pas sur les tirs du match. Le film n'en
// porte que 69 à 87 % en arène (23 % en Fiesta), si bien que la part des tirs RÉELS posés sur la
// carte vaut 61 à 83 %. La complétude du flux de tirs est un AUTRE chantier (piste E), et aucun
// critère de ce fichier ne la couvre.
//
// CE GARDE N'EST PAS UN INTERRUPTEUR « POUR PLUS TARD » (règle du dépôt sur les kill-switches).
// Il porte les trois éléments exigés :
//
//	BASCULÉ LE          2026-07-28, à l'ouverture du rejeu en local.
//	                    Critère RÉÉCRIT le 2026-08-08 : le précédent (« > 85 % sur au moins deux
//	                    films de cartes différentes ») était satisfait à la lettre par deux films
//	                    CHOISIS, alors qu'un troisième tombait à 80,3 %. Un critère qu'on satisfait
//	                    en choisissant ses films ne protège de rien.
//	RETRAIT CIBLE       à la reconstruction des artefacts du corpus nommé avec le pont fermé,
//	                    puis décision utilisateur. À RÉEXAMINER AU PLUS TARD LE 2026-11-08.
//	CRITÈRE MESURABLE   couverture des tirs >= 88 % et `coverage.verdict.bridge` nominal sur
//	                    TOUS les films du corpus nommé ci-dessous, sans collision de trace.
//
// LE CORPUS EST NOMMÉ, et c'est ce qui rend le critère opposable — sept films, sept cartes ou
// modes distincts, à rejouer à chaque changement du pont :
//
//	000d5950 Cliffhanger Fiesta Slayer · 0edb8512 Aquarius Team Slayer
//	9aeca4b3 Catalyst Team Slayer      · 01e1f945 Catalyst KOTH
//	64e8adfa Catalyst CTF              · db7b8c3c Aquarius CTF · 829abef9 Behemoth CTF
//
// POURQUOI 88 ET NON 90. Mesuré : à 88 % le corpus passe 7 sur 7 avec le pont fermé ; à 90 % il
// passe 5 sur 7, ce qui obligerait à excepter deux films — et une exception négociée est
// exactement le défaut que la réécriture corrige.
//
// Le critère se lit dans l'artefact lui-même : nul besoin d'un jugement pour décider du retrait.
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

// LocalOnlyReplay applique le garde local devant les routes de rejeu.
//
// C'EST UN MIDDLEWARE, PAS UNE BRANCHE DE HANDLER, et ce n'est pas un détail de style :
// le garde repose sur l'adresse de la connexion TCP (`r.RemoteAddr`), la seule donnée
// que le demandeur ne peut pas falsifier. La couche Huma ne l'expose pas à ses handlers
// typés — elle ne voit que le contexte et les paramètres de chemin. Le garde reste donc
// à l'étage transport, où l'information existe.
func LocalOnlyReplay(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowReplay(r) {
			writeError(r.Context(), w, http.StatusNotFound, "replay_not_available",
				"le rejeu 2D n'est servi qu'en local")
			return
		}
		next.ServeHTTP(w, r)
	})
}
