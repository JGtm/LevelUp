package objectiveevents

import "sort"

// slotidentity.go — QUI est le joueur derriere un slot d'entite du statborg.
//
// # Le probleme, et il bloquait le collage au rejeu 2D
//
// Les evenements nommes portent un slot d'ENTITE STATBORG (10..24 pairs). Le rejeu 2D, lui,
// indexe ses trajectoires par slot de BIPED et identifie ses vies par XUID. **Ce sont deux
// espaces de slots differents** : rien ne garantit qu'ils coincident, et le supposer
// collerait des evenements sur les mauvais joueurs sans que rien ne le signale.
//
// Le pont ne peut donc pas etre le numero de slot. C'est le XUID.
//
// # La methode : un triplet, pas une coincidence temporelle
//
// Le statborg replique les statistiques de base de chaque joueur, et trois d'entre elles
// sont dans `match_participants` : les frags, les morts et les assistances. Le triplet
// (frags, morts, assistances) est apparie a la ligne de match du joueur.
//
// Ce que cela remplace : l'appariement par COINCIDENCE TEMPORELLE (apparier les increments
// de +100 aux instants de frag, etat de l'art §16.2). Celui-ci marchait, mais il exigeait
// une fenetre de tolerance et une source d'instants de frag. Le triplet n'exige ni l'un ni
// l'autre — il compare trois entiers exacts.
//
// # Ce qui est mesure
//
// L'appariement est UNIQUE sur les 8 joueurs des deux films de reference (`696a9d7c` et
// `1bc77d2e`), et les slots qu'il rend concordent avec les identites etablies par ailleurs
// (§16.2). Les trois emplacements employes sont eux-memes confirmes nominativement :
// `comp 2 A` = frags, `comp 2 B` = morts (8/8 exacts contre `match_participants`),
// `comp 3 A` = assistances. Le binaire les corrobore : `CoreStats_` declare Kills, Deaths et
// Assists dans cet ordre (etat de l'art §19.5).
//
// # La regle de prudence
//
// Un slot dont le triplet ne designe pas UNE seule ligne n'est PAS apparie, et un xuid que
// deux slots se disputent n'est attribue a aucun des deux. Se taire vaut mieux qu'attribuer
// les frags d'un joueur a un autre — sur une carte, l'erreur serait invisible et credible.

// Emplacements des statistiques de base employees par l'appariement. Confirmes
// nominativement (cf. en-tete) et corrobores par l'ordre de `CoreStats_` dans le binaire.
const (
	// coreKillsComp porte les frags en valeur A et les morts en valeur B.
	coreKillsComp = 2
	// coreAssistsComp porte les assistances en valeur A.
	coreAssistsComp = 3
)

// PlayerLine est la ligne de match d'un joueur, telle que `match_participants` la donne.
// L'appelant la fournit : ce paquet ne touche jamais la base.
type PlayerLine struct {
	// XUID en decimal, meme forme que la base et que le rejeu 2D.
	XUID string
	// Kills, Deaths, Assists sont les compteurs du match.
	Kills, Deaths, Assists int
}

// SlotIdentity resout, pour un film, le slot d'entite statborg de chaque joueur.
//
// Le resultat s'utilise pour traduire le Slot d'un [NamedEvent] en XUID, seule cle sur
// laquelle le rejeu 2D peut joindre. Un slot absent de la table n'a pas pu etre apparie
// sans ambiguite : ses evenements ne doivent PAS etre attribues.
//
// Ne depend d'AUCUN mode : les frags, morts et assistances sont des statistiques de base,
// repliquees quel que soit le type de partie. L'appariement fonctionne donc aussi en
// Slayer, KOTH ou Oddball, ou aucun emplacement d'objectif n'est nomme.
func SlotIdentity(src FilmSource, lines []PlayerLine) map[int]string {
	return SlotIdentityFrom(StatRecords(src), lines)
}

// SlotIdentityFrom est le coeur pur : il travaille sur des enregistrements deja decodes.
//
// EXPORTE POUR LA PRODUCTION (meme raison que [NamedEventsFrom]) : le constructeur
// d'artefact decode le film une seule fois et fait servir les memes enregistrements a la
// courbe de score, a l'identite des slots et aux evenements nommes.
func SlotIdentityFrom(recs []StatRecord, lines []PlayerLine) map[int]string {
	kills := countsOf(recs, statSlotKey{coreKillsComp, sideA})
	deaths := countsOf(recs, statSlotKey{coreKillsComp, sideB})
	assists := countsOf(recs, statSlotKey{coreAssistsComp, sideA})

	// Premiere passe : les slots dont le triplet designe UNE seule ligne de match.
	claim := map[int]string{}
	for slot := range kills {
		var found string
		n := 0
		for _, l := range lines {
			if l.Kills == kills[slot] && l.Deaths == deaths[slot] && l.Assists == assists[slot] {
				found, n = l.XUID, n+1
			}
		}
		if n == 1 {
			claim[slot] = found
		}
	}

	// Seconde passe : un xuid revendique par plusieurs slots n'est attribue a aucun. Deux
	// joueurs peuvent finir un match sur le meme triplet ; dans ce cas l'appariement ne
	// prouve rien, et le mauvais choix serait indetectable a l'ecran.
	byXUID := map[string][]int{}
	for slot, xuid := range claim {
		byXUID[xuid] = append(byXUID[xuid], slot)
	}
	out := make(map[int]string, len(claim))
	for xuid, slots := range byXUID {
		if len(slots) == 1 {
			out[slots[0]] = xuid
		}
	}
	return out
}

// IdentifiedEvent est un evenement nomme dont l'auteur est resolu en XUID — la forme
// directement superposable aux positions du rejeu 2D, qui joint par xuid.
type IdentifiedEvent struct {
	NamedEvent
	// XUID du joueur auteur de l'action, en decimal.
	XUID string
}

// IdentifyNamedEvents traduit le slot de chaque evenement en XUID et ECARTE ceux dont le
// slot n'a pas ete apparie sans ambiguite.
//
// Le nombre d'evenements ecartes est volontairement observable par difference avec
// [NamedEvents] : publier des evenements attribues sans dire combien ne l'ont pas ete
// laisserait croire a l'exhaustivite (meme regle que le champ Coverage du rejeu 2D).
func IdentifyNamedEvents(evs []NamedEvent, identity map[int]string) []IdentifiedEvent {
	out := make([]IdentifiedEvent, 0, len(evs))
	for _, e := range evs {
		xuid, ok := identity[e.Slot]
		if !ok {
			continue
		}
		out = append(out, IdentifiedEvent{NamedEvent: e, XUID: xuid})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		if out[i].XUID != out[j].XUID {
			return out[i].XUID < out[j].XUID
		}
		return out[i].Stat < out[j].Stat
	})
	return out
}
