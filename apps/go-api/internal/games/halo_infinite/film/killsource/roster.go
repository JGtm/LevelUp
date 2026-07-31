package killsource

// roster.go — LE ROSTER ETENDU : les humains du kill-feed, PUIS les bots de BOT_METADATA.
//
// Le dead-state identifie un participant par son INDICE ABSOLU sur 5 bits (espace 0..31), pas
// par un gamertag. Il faut donc une bijection indice -> joueur, et un roster a mettre en face.
//
// CE QUI MANQUAIT AVANT N ETAIT PAS UN DECODAGE MAIS UN GATE : l outil de RE figeait
// `indice <= 7`, et l indice 8 — le bot — etait CORRECTEMENT DECODE PUIS JETE A L AFFICHAGE.
// BOT_METADATA donne le slot, donc la borne se DERIVE du film au lieu d etre codee en dur.
//
// REGLE D EPINGLAGE, structurelle et qui ne regarde AUCUN resultat : un bot dont le slot tombe
// AU-DELA de l espace des humains est ajoute au roster et son indice est EPINGLE (il sort de
// l espace de recherche de la bijection) ; un bot dont le slot tombe DANS cet espace contredit
// le modele et n est PAS epingle — il est signale, pas dissimule.
//
// POURQUOI EPINGLER PLUTOT QUE LAISSER LIBRE : un bot n apparait JAMAIS au kill-feed. Laisser
// son indice libre offrirait au solveur une case qui score zero, et lui permettrait d y ranger
// un HUMAIN — ce qui deplacerait toutes les autres attributions.

import "fmt"

// Roster : les noms retenus et la bijection indice -> joueur, publies pour que le consommateur
// puisse verifier a qui les indices ont ete attribues.
type Roster struct {
	// Names : les joueurs, humains d abord (ordre du kill-feed, trie), puis les bots.
	Names []string
	// Humans : combien des `Names` viennent du kill-feed.
	Humans int
	// Bots : les bots declares par le film, avec leur slot.
	Bots []BotEntry
	// IndexToName : bijection resolue. `IndexToName[i]` est le joueur porte par l indice i.
	IndexToName []string
	// UnpinnedBots : bots dont le slot contredit l espace des humains. Non vide = anomalie a
	// regarder, pas a ignorer.
	UnpinnedBots []BotEntry
}

// BotEntry : un bot tel que le film le declare.
type BotEntry struct {
	Slot  int
	BotID int
	Name  string
}

// roster : l etat interne. `pin` associe un indice absolu a une position de `names`.
type roster struct {
	names    []string
	pin      map[int]int
	nPlay    int // borne du gate des indices, derivee du film
	nHumans  int
	bots     botMeta
	unpinned []bot
	perm     []int
}

// botSuffix : marqueur ajoute au nom d un bot. Il doit rester visible : un consommateur ne doit
// jamais confondre un bot avec un joueur, et le kill-feed ne les distingue pas pour lui.
const botSuffix = " [bot]"

// buildRoster : roster etendu + epinglage des slots de bot + borne des indices.
func buildRoster(kf *killFeed, bm botMeta, useBots bool) *roster {
	r := &roster{
		names:   append([]string(nil), kf.names...),
		pin:     map[int]int{},
		nPlay:   len(kf.names),
		nHumans: len(kf.names),
		bots:    bm,
	}
	if !useBots {
		return r
	}
	for _, b := range bm.Bots {
		if b.Slot < r.nHumans || b.Slot >= 32 {
			r.unpinned = append(r.unpinned, b)
			continue
		}
		r.names = append(r.names, b.Name+botSuffix)
		r.pin[b.Slot] = len(r.names) - 1
		if b.Slot+1 > r.nPlay {
			r.nPlay = b.Slot + 1
		}
	}
	// Les indices libres qui ne recoivent aucun nom (trous entre le dernier humain et un slot
	// de bot eleve) recoivent un nom de remplissage : le probleme d affectation doit rester
	// CARRE (autant d indices libres que de noms libres), sinon le hongrois n est pas defini.
	for len(r.names) < r.nPlay {
		r.names = append(r.names, fmt.Sprintf("?%d", len(r.names)))
	}
	return r
}

// freeSlots : indices NON epingles, et positions de noms NON epinglees. Les deux listes ont la
// meme longueur par construction.
func (r *roster) freeSlots() (free, freeNames []int) {
	used := map[int]bool{}
	for _, p := range r.pin {
		used[p] = true
	}
	for i := 0; i < r.nPlay; i++ {
		if _, ok := r.pin[i]; !ok {
			free = append(free, i)
		}
	}
	for p := range r.names {
		if !used[p] {
			freeNames = append(freeNames, p)
		}
	}
	return free, freeNames
}

// nameOf : nom porte par un indice absolu sous la bijection resolue, ou "?" hors roster.
func (r *roster) nameOf(i int) string {
	if i < 0 || i >= len(r.perm) || r.perm[i] < 0 || r.perm[i] >= len(r.names) {
		return "?"
	}
	return r.names[r.perm[i]]
}

// isBotIndex : l indice est-il epingle sur un bot ?
func (r *roster) isBotIndex(i int) bool { _, ok := r.pin[i]; return ok }

// public : la vue exportee.
func (r *roster) public() Roster {
	out := Roster{Names: append([]string(nil), r.names...), Humans: r.nHumans}
	for _, b := range r.bots.Bots {
		out.Bots = append(out.Bots, BotEntry{Slot: b.Slot, BotID: b.BotID, Name: b.Name})
	}
	for _, b := range r.unpinned {
		out.UnpinnedBots = append(out.UnpinnedBots, BotEntry{Slot: b.Slot, BotID: b.BotID, Name: b.Name})
	}
	out.IndexToName = make([]string, r.nPlay)
	for i := 0; i < r.nPlay; i++ {
		out.IndexToName[i] = r.nameOf(i)
	}
	return out
}
