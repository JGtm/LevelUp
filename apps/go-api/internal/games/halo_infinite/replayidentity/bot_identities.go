// Package replayidentity — LE ROSTER DE BOTS DECODE PAR killsource, PROJETE VERS LE REGISTRE
// D'IDENTITE DU REJEU (`internal/games/halo_infinite/film/replay`).
//
// # POURQUOI CE PAQUET EXISTE, ET PAS UN APPEL DIRECT ENTRE SES DEUX VOISINS
//
// La projection a DEUX consommateurs : `internal/replaybuild` (la cuisson) et
// `internal/sync/killcollector` (le collecteur de sync) construisent tous deux
// `replay.IdentityInput.Bots` a partir du MEME roster BOT_METADATA
// (`decfilm.Result.Roster`). Avant ce paquet, seul `replaybuild` la projetait — le
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
	"levelup/go-api/internal/observability"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// Les deux parts du compteur expvar des bots NON EPINGLES (`UnpinnedBots` : leur slot tombe dans
// l'espace des humains de la table du film) : ceux que [BotIdentities] passe au registre
// d'identite du decodeur, et ceux qu'elle refuse.
const (
	MetricBotsNonEpinglesPasses  = "killsource_bots_non_epingles_passes"
	MetricBotsNonEpinglesRefuses = "killsource_bots_non_epingles_refuses"
)

// BotIdentities projette le roster de bots declares par BOT_METADATA vers ce que le registre
// d'identite consomme (`replay.IdentityInput.Bots`). PURE — aucune I/O, aucun re-decodage : elle
// lit `res.Roster`, deja construit par `decfilm.Decode`.
//
// # CE QUE LE FILTRE REFUSE
//
// Un bot dont le film ne donne AUCUN nom (`Name == ""`) n'a rien a publier — un consommateur ne
// doit jamais confondre son absence de nom avec une identite.
//
// # CE QU'IL NE REFUSE PLUS : LE BOT NON EPINGLE NOMME
//
// Un bot non epingle dont le film donne un nom est PASSE, avec ses declarations. Qu'il tombe
// dans l'espace des humains ne dit pas que ses vies n'ont pas de nom : le registre decide, par
// les listes de session du film, si un humain tient ce siege a l'instant de la creation du
// corps (aucun humain liste = la declaration du bot nomme la vie ; un humain liste = elle ne la
// nomme pas). Le retrait en amont laissait sans nom des vies que le film nomme (`343 Byrontron`,
// siege 1 des 1 101 s de `4f77afc1`) ; la copie decalee d'un nom (`43 Ritzy`, slot 0) reste sans
// nom parce qu'un humain est liste a ce siege.
//
// Chaque bot non epingle est compte, chez les deux consommateurs (cuisson et collecteur) :
// [MetricBotsNonEpinglesPasses] pour ceux qui voyagent jusqu'au registre,
// [MetricBotsNonEpinglesRefuses] pour ceux que l'absence de nom retient. Le JOURNAL vit a la
// source, la ou le contexte existe : `killsource` avertit par film au decodage et publie le
// compte dans sa couverture (`Coverage.BotsNonEpingles`).
//
// UN BOT NON EPINGLE SE RECONNAIT A SON ENTREE, PAS A SON SEUL `bid` : la copie decalee d'un nom
// (`43 Ritzy` au slot 0, `bid` lu 0) partage le `bid` 0 du vrai bot (`343 Ritzy`, `bid(0.0)` en
// base) ; une cle par `bid` les confondrait. Le `bid` projete est marque LU
// (`replay.BotIdentity.BotIDLu`) : toute entree BOT_METADATA l'ecrit, zero compris.
func BotIdentities(res *decfilm.Result) []replay.BotIdentity {
	if res == nil || len(res.Roster.Bots) == 0 {
		return nil
	}
	unpinned := make(map[entreeDeBot]bool, len(res.Roster.UnpinnedBots))
	for _, b := range res.Roster.UnpinnedBots {
		unpinned[entreeDeBot{b.Slot, b.BotID, b.Name}] = true
	}
	out := make([]replay.BotIdentity, 0, len(res.Roster.Bots))
	passes, refuses := 0, 0
	for _, b := range res.Roster.Bots {
		nonEpingle := unpinned[entreeDeBot{b.Slot, b.BotID, b.Name}]
		if b.Name == "" {
			if nonEpingle {
				refuses++
			}
			continue
		}
		if nonEpingle {
			passes++
		}
		// `BotID` VOYAGE (lot 4.3) : c'est la cle EXACTE que `BotIdentity.Bid()` publie
		// (`bid(N.0)`), la meme forme que `RosterEntry.Bid` / `IdentityPlayer.Bid`. Les
		// DECLARATIONS voyagent aussi (lot M2.1) : elles lient le bot a son entite `ti=9`.
		id := replay.BotIdentity{FilmIndex: b.Slot, Name: b.Name + decfilm.BotSuffix, BotID: b.BotID, BotIDLu: true}
		for _, d := range b.Declarations {
			id.Declarations = append(id.Declarations, [2]uint64{d.FromUS, d.ToUS})
		}
		// L'ÉQUIPE VOYAGE AUSSI (2026-10-06) : celle que l'entrée BOT_METADATA du bot écrit, ou
		// rien quand le décodage ne l'a pas lue.
		if b.Team != nil {
			t := *b.Team
			id.Team = &t
		}
		out = append(out, id)
	}
	if passes > 0 {
		observability.AddInt(MetricBotsNonEpinglesPasses, int64(passes))
	}
	if refuses > 0 {
		observability.AddInt(MetricBotsNonEpinglesRefuses, int64(refuses))
	}
	return out
}

// entreeDeBot identifie une entree du roster de bots : slot, `bid` et nom lus ensemble.
type entreeDeBot struct {
	slot, bid int
	nom       string
}
