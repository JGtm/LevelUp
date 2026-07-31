package killsource

// feed.go — LA VERITE KILL-FEED : ce que le JEU affiche.
//
// Le chunk HIGHLIGHT (celui qui produit le plus d evenements `kill`) porte les kills et les
// morts horodates, avec XUID et gamertag. Il est deja decode par `analysis.ParseHighlightEvents`
// — il n y avait rien a craquer, et le parseur n a AUCUN correctif a recevoir.
//
// IL EST HUMAIN SEUL, ET C EST MESURE SANS AUCUNE ANCRE : une enumeration exhaustive des
// end-markers rend EXACTEMENT les memes events que le parseur, et la comptabilite des chaines
// UTF-16LE se ferme au bit pres (194 gamertags pour 194 events). Il n y a pas la place d un
// neuvieme nom. Un bot n a pas de XUID : sa mort ne produit aucun event. (RE_LOG 7ter.59.)
//
// CONSEQUENCE, ET C EST LA PLUS COUTEUSE DU CHANTIER : quand un humain tue un bot, le film
// porte le KILL mais pas la MORT. La reconstruction de couples recolle alors le kill au `death`
// du voisin et FABRIQUE un couple qui n existe pas. Ce couple doit sortir du denominateur, pas
// etre compte comme une mort manquee (RE_LOG 7ter.66, verifie en Theater).

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis"
)

// feedEvent : un instant du kill-feed. `killer` et `victim` peuvent etre vides separement
// (deux morts a la meme seconde ne portent pas toujours les deux champs).
type feedEvent struct {
	timeMS int
	killer string
	victim string
}

// killFeed : la decomposition HONNETE du kill-feed. Chaque champ est un denominateur potentiel,
// et c est pour cela qu ils sont separes au lieu d etre additionnes.
type killFeed struct {
	events []feedEvent // instants, tries
	pairs  []feedEvent // couples reconstruits (reels + recolles)
	names  []string    // roster HUMAIN, trie

	real  []feedEvent // couples portant kill ET death au meme instant
	fab   []feedEvent // couples RECOLLES sur le voisin : candidats a la mort de bot
	orphK []feedEvent // kills sans aucun voisin a consommer : la victime n est pas humaine
	// orphD : morts que la reconstruction n a JAMAIS consommees — ni au meme instant, ni
	// recollees sur un kill voisin. Le kill-feed porte la MORT, il ne porte AUCUN kill en face :
	// LE TUEUR N EST PAS HUMAIN. C est la population symetrique de `orphK`, et elle n avait
	// jamais ete isolee (RE_LOG 7ter.79).
	orphD []feedEvent

	nKills, nDeaths int
}

// loadKillFeed : localise le chunk HIGHLIGHT PAR SON CONTENU (celui qui produit le plus de
// kills) et en tire les instants. Aucune borne de chunk : un BTB a son HIGHLIGHT en n62.
func loadKillFeed(f *film) (*killFeed, error) {
	var best []analysis.HighlightEvent
	bestN := 0
	for ch := range f.chunks {
		evs, err := analysis.ParseHighlightEvents(f.chunks[ch], 0)
		if err != nil {
			continue // un chunk de replication n est pas un chunk HIGHLIGHT : ce n est pas une erreur
		}
		nk := 0
		for _, e := range evs {
			if e.EventType == analysis.EventTypeKill {
				nk++
			}
		}
		if nk > bestN {
			bestN, best = nk, evs
		}
	}
	if bestN == 0 {
		return nil, ErrNoKillFeed
	}
	kf := buildFeed(best)
	kf.pairs = reconstructPairs(kf.events)
	kf.names = rosterNames(kf.events)
	kf.split()
	return kf, nil
}

// buildFeed : regroupe les events par instant et resout les XUID en gamertags.
func buildFeed(evs []analysis.HighlightEvent) *killFeed {
	gt := map[uint64]string{}
	for _, e := range evs {
		if e.Gamertag != "" {
			gt[e.XUID] = e.Gamertag
		}
	}
	kf := &killFeed{}
	byTime := map[int]*feedEvent{}
	at := func(ms int) *feedEvent {
		if byTime[ms] == nil {
			byTime[ms] = &feedEvent{timeMS: ms}
		}
		return byTime[ms]
	}
	for _, e := range evs {
		name := gt[e.XUID]
		if name == "" {
			name = fmt.Sprintf("xuid:%d", e.XUID)
		}
		switch e.EventType {
		case analysis.EventTypeKill:
			kf.nKills++
			at(e.TimeMS).killer = name
		case analysis.EventTypeDeath:
			kf.nDeaths++
			at(e.TimeMS).victim = name
		}
	}
	for _, v := range byTime {
		kf.events = append(kf.events, *v)
	}
	sort.Slice(kf.events, func(i, j int) bool { return kf.events[i].timeMS < kf.events[j].timeMS })
	return kf
}

// reconstructPairs : les couples (tueur, victime) horodates. Un instant portant les deux champs
// est un couple REEL ; un kill orphelin est recolle au `death` d un voisin immediat (deux morts
// a la meme seconde n arrivent pas toujours dans le meme instant).
//
// CE RECOLLAGE EST LEGITIME ET IL EST MESURE : les couples recolles echouent MOINS que les
// autres (5/64 = 7.8 % contre 42/372 = 11.3 %, p = 0.886, RE_LOG 7ter.52 A2). Mais il FABRIQUE
// un couple quand la vraie victime est un bot — d ou [killFeed.split].
func reconstructPairs(feed []feedEvent) []feedEvent {
	var out []feedEvent
	for i := range feed {
		if feed[i].killer != "" && feed[i].victim != "" {
			out = append(out, feed[i])
			continue
		}
		if feed[i].killer == "" {
			continue
		}
		for d := 1; d <= 2 && i+d < len(feed); d++ {
			o := feed[i+d]
			if o.victim != "" && o.killer == "" {
				out = append(out, feedEvent{feed[i].timeMS, feed[i].killer, o.victim})
				break
			}
		}
	}
	return out
}

// split : rejoue EXACTEMENT [reconstructPairs] pour savoir ce qu il consomme, et separe les
// couples reels des couples recolles, des kills perdus et des MORTS PERDUES.
//
// HYPOTHESE MESUREE COMME FAUSSE, a ne pas reintroduire : << un kill sans death au meme instant
// est une mort de bot >>. Elle rend 13 candidats pour 3 morts de bot reelles. Ce qui reste vrai,
// et c est ce qui est mesure ici : un kill qui ne trouve AUCUN voisin a consommer est un kill
// dont la victime n est pas au feed. C est le DEAD-STATE qui tranche, jamais cette structure.
//
// LA SYMETRIE VAUT DANS L AUTRE SENS, ET ELLE N AVAIT JAMAIS ETE TIREE : une MORT que personne ne
// consomme est une mort dont le TUEUR n est pas au feed. Ici encore la structure ne fait que
// DESIGNER une population — c est le dead-state, plus le kill-event, qui tranchent (voir
// [decodeCtx.resolveBotKillerDeaths]).
//
// LA MARQUE DE CONSOMMATION NE CHANGE RIEN AU RESULTAT DE [reconstructPairs] : elle est posee
// APRES coup et n intervient dans aucune condition de la boucle. Deux kills orphelins qui
// convoitent la meme mort la consomment tous les deux, exactement comme avant.
func (kf *killFeed) split() {
	feed := kf.events
	pris := make([]bool, len(feed))
	for i := range feed {
		if feed[i].killer != "" && feed[i].victim != "" {
			kf.real = append(kf.real, feed[i])
			pris[i] = true
			continue
		}
		if feed[i].killer == "" {
			continue
		}
		matched := false
		for d := 1; d <= 2 && i+d < len(feed); d++ {
			o := feed[i+d]
			if o.victim != "" && o.killer == "" {
				kf.fab = append(kf.fab, feedEvent{feed[i].timeMS, feed[i].killer, o.victim})
				pris[i+d] = true
				matched = true
				break
			}
		}
		if !matched {
			kf.orphK = append(kf.orphK, feed[i])
		}
	}
	for i := range feed {
		if feed[i].victim != "" && !pris[i] {
			kf.orphD = append(kf.orphD, feed[i])
		}
	}
}

// rosterNames : les joueurs distincts nommes par le kill-feed, tries.
//
// PIEGE DE PARSEUR, consigne parce qu il a coute 42 appariements sans le moindre message
// d erreur : UN GAMERTAG CONTIENT DES ESPACES (<< Zeus Herd >>). Toute serialisation de la
// bijection doit se decouper sur les marqueurs `<chiffre>=`, jamais sur les blancs.
func rosterNames(feed []feedEvent) []string {
	set := map[string]bool{}
	for _, e := range feed {
		if e.killer != "" {
			set[e.killer] = true
		}
		if e.victim != "" {
			set[e.victim] = true
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
