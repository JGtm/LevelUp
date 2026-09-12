// Package replayidentity — LE ROSTER DE BOTS DECODE PAR killsource, PROJETE VERS LE REGISTRE
// D'IDENTITE DU REJEU (`internal/games/halo_infinite/film/replay`).
//
// # POURQUOI CE PAQUET EXISTE, ET PAS UN APPEL DIRECT ENTRE SES DEUX VOISINS
//
// La projection a DEUX consommateurs : `internal/replaybuild` (la cuisson) et
// `internal/sync/killcollector` (le collecteur de sync) construisent tous deux
// `replay.IdentityInput.Bots` a partir du MEME roster BOT_METADATA
// (`killsource.Result.Roster`). Avant ce paquet, seul `replaybuild` la projetait — le
// collecteur ne passait ni `Bots` ni `Participants` au registre d'identite (revue
// adversariale de la vague 4, 2026-09-10, constat P2 sur
// `sync/killcollector/positions.go:484-489`). Consequence mesuree : sur un siege d'index
// partage entre un bot et un humain arrive en cours, `PlayerIndexTable` resout l'index vers
// l'humain (ses morts APRES l'arrivee suffisent a la bijection du collecteur), et SANS `Bots`
// pour dire que cet index est AUSSI un bot, le lien direct
// (`identity_registry_creation.go`) attribue TOUT le siege — y compris les vies D'AVANT
// l'arrivee, qui sont celles du bot — au xuid de l'humain.
//
// ELLE NE PEUT VIVRE NI DANS `killsource` NI DANS `replay` :
//
//   - `killsource` -> `replay` cree un cycle DE TEST : `replay` a un instrument de mesure
//     (`visee_lunette_research_test.go`, `package replay`) qui importe deja `killsource` pour
//     decoder un film. Un `killsource` qui importerait `replay` en production fermerait la
//     boucle au moment de compiler les tests de `replay`.
//   - `replay` -> `killsource` inverserait la couche : `replay` est l'analyse GENERIQUE du
//     rejeu (parametree par `titleSlug`, cf. l'en-tete de `identity_registry.go` — « les DEUX
//     producteurs »), `killsource` est un decodeur SPECIFIQUE a Halo Infinite. Le sens de
//     dependance attendu est celui-ci : le code specifique consomme l'analyse generique,
//     jamais l'inverse.
//
// Un paquet tiers, qui ne depend QUE des deux, ferme le probleme sans creer de cycle et sans
// abimer la couche : `replaybuild` et `killcollector` l'importent tous deux, et aucun des deux
// paquets qu'il relie ne le connait.
package replayidentity

import (
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// BotIdentities projette le roster de bots declares par BOT_METADATA vers ce que le registre
// d'identite consomme (`replay.IdentityInput.Bots`). PURE — aucune I/O, aucun re-decodage : elle
// lit `res.Roster`, deja construit par `killsource.Decode`.
//
// # CE QUE LE FILTRE REFUSE
//
// Un bot dont le film ne donne AUCUN nom (`Name == ""`) n'a rien a publier — un consommateur ne
// doit jamais confondre son absence de nom avec une identite. Un bot NON EPINGLE
// (`UnpinnedBots` — son slot contredit l'espace des humains) est une anomalie DECLAREE, pas une
// identite : le publier ferait porter un identifiant a un corps que le modele ne sait pas
// placer.
func BotIdentities(res *killsource.Result) []replay.BotIdentity {
	if res == nil || len(res.Roster.Bots) == 0 {
		return nil
	}
	unpinned := make(map[int]bool, len(res.Roster.UnpinnedBots))
	for _, b := range res.Roster.UnpinnedBots {
		unpinned[b.BotID] = true
	}
	out := make([]replay.BotIdentity, 0, len(res.Roster.Bots))
	for _, b := range res.Roster.Bots {
		if b.Name == "" || unpinned[b.BotID] {
			continue
		}
		// `BotID` VOYAGE (lot 4.3) : c'est la cle EXACTE que `BotIdentity.Bid()` publie
		// (`bid(N.0)`), la meme forme que `RosterEntry.Bid` / `IdentityPlayer.Bid`.
		out = append(out, replay.BotIdentity{
			FilmIndex: b.Slot, Name: b.Name + killsource.BotSuffix, BotID: b.BotID})
	}
	return out
}
