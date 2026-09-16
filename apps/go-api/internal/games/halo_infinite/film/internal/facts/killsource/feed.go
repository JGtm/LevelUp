package killsource

// feed.go — LA VERITE KILL-FEED : ce que le JEU affiche.
//
// Le chunk HIGHLIGHT (celui qui produit le plus d evenements `kill`) porte les kills et les
// morts horodates, avec XUID et gamertag. Il est deja decode par `grammar.ParseHighlightEvents`
// — il n y avait rien a craquer, et le parseur n a AUCUN correctif a recevoir.
//
// IL EST HUMAIN SEUL, ET C EST MESURE SANS AUCUNE ANCRE : une enumeration exhaustive des
// end-markers rend EXACTEMENT les memes events que le parseur, et la comptabilite des chaines
// UTF-16LE se ferme au bit pres (194 gamertags pour 194 events). Il n y a pas la place d un
// neuvieme nom. Un bot n a pas de XUID : sa mort ne produit aucun event. (RE_LOG 7ter.59.)
//
// CONSEQUENCE, ET C EST LA PLUS COUTEUSE DU CHANTIER : quand un humain tue un bot, le film
// porte le KILL mais pas la MORT.
//
// DEPUIS LE LOT 1.9.3, CE N EST PLUS LE VOISINAGE QUI TRANCHE : le kill-event de code 85 ecrit
// victime ET tueur dans le meme enregistrement, et c est LUI qui decide le couple d un kill sans
// mort en face ([killFeed.resoudreCouples], `feed_couples.go`). Le recollage sur le voisin — qui
// FABRIQUAIT un couple quand la vraie victime etait un bot — n est plus qu un REPLI NOMME, sur
// les seuls instants ou le film se tait. Les couples qu il produit encore (`fab`) restent des
// candidats a la mort de bot, et sortent du denominateur comme avant (RE_LOG 7ter.66, verifie en
// Theater).

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// feedEvent : un instant du kill-feed. `killer` et `victim` peuvent etre vides separement
// (deux morts a la meme seconde ne portent pas toujours les deux champs).
type feedEvent struct {
	timeMS int
	killer string
	victim string
	// victimXUID : le XUID que le kill-feed porte pour la victime. Il ne sert a AUCUN
	// appariement — ceux-ci se font par nom, comme partout ailleurs dans ce paquet — mais a la
	// PUBLICATION : un consommateur qui joint des pistes de film joint par xuid, jamais par
	// pseudo (le pseudo change, le xuid non). Zero quand l instant ne porte pas de mort.
	victimXUID uint64
	// paquet : L IDENTITE DE PAQUET DE CET INSTANT (lot 1.9.7), prise au KILL-EVENT 85 que
	// [killFeed.resoudreCouples] lui a associe. C est elle qui apparie le dead-state, la fenetre
	// de 2,5 s n etant plus qu un repli. Absente quand aucun kill-event ne s est attache : le
	// kill-feed ne localise rien par lui-meme, il n horodate.
	paquet paquetID
}

// killFeed : la decomposition HONNETE du kill-feed. Chaque champ est un denominateur potentiel,
// et c est pour cela qu ils sont separes au lieu d etre additionnes.
//
// TOUT CE QUI SUIT `names` EST REMPLI PAR [killFeed.resoudreCouples], jamais par [loadKillFeed] :
// la decomposition exige le ROSTER EPINGLE et les KILL-EVENTS, que l appelant construit apres le
// chargement (cf. `feed_couples.go` et [decodeCtx.prepare]).
type killFeed struct {
	events []feedEvent // instants, tries
	pairs  []feedEvent // couples publies (meme instant + lus au kill-event + recolles)
	names  []string    // roster HUMAIN, trie
	// xuidDe : le xuid que le kill-feed porte pour chaque gamertag. Il sert quand un couple LU
	// au kill-event nomme une victime dont aucun instant voisin ne porte la mort.
	xuidDe map[string]uint64

	real []feedEvent // couples portant kill ET death au meme instant
	// lus : couples dont le KILL-EVENT 85 a decide la victime — la part LUE de la publication.
	lus []feedEvent
	fab []feedEvent // couples RECOLLES sur le voisin (REPLI) : candidats a la mort de bot
	// botLus : kills dont le film NOMME un bot en victime. Ils n entrent dans AUCUN couple.
	botLus []killDeBot
	orphK  []feedEvent // kills sans aucun voisin a consommer : la victime n est pas humaine
	// orphD : morts que la reconstruction n a JAMAIS consommees — ni au meme instant, ni
	// recollees sur un kill voisin. Le kill-feed porte la MORT, il ne porte AUCUN kill en face :
	// LE TUEUR N EST PAS HUMAIN. C est la population symetrique de `orphK`, et elle n avait
	// jamais ete isolee (RE_LOG 7ter.79).
	orphD []feedEvent

	nKills, nDeaths int
}

// loadKillFeed : localise le chunk HIGHLIGHT PAR SON CONTENU (celui qui produit le plus de
// kills) et en tire les instants. Aucune borne de chunk : un BTB a son HIGHLIGHT en n62.
//
// LA VERSION DU FILM EST LUE, PLUS DEVINEE (2026-09-12). Elle vient du PROFIL du film depuis le
// lot 2.1.4 (`grammar.HighlightProfileOfFilm`, pose par `loadFilm` : meme lecture de l en-tete du
// registre, mais c est le profil qui en est la source unique) et commande le decoupage du
// gamertag dans le bloc d event. Le 0 qui trainait ici designait « gamertag en tete » pour TOUS les films, y
// compris les versions 39-40 ou il vit douze octets plus loin — d ou 68 a 96 % de morts sans
// source de degat sur les films de mars a novembre 2025
// (.ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md). Film sans registre : `f.versionLue` est faux,
// `f.majorVersion` vaut 0 et le comportement historique tient — l appelant l a consigne.
func loadKillFeed(f *film) (*killFeed, error) {
	var best []highlightevent.HighlightEvent
	bestN := 0
	for ch := 0; ch < f.src.NumChunks(); ch++ {
		evs, err := grammar.ParseHighlightEvents(f.src.Chunk(ch), f.majorVersion)
		if err != nil {
			continue // un chunk de replication n est pas un chunk HIGHLIGHT : ce n est pas une erreur
		}
		nk := 0
		for _, e := range evs {
			if e.EventType == highlightevent.EventTypeKill {
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
	kf.names = rosterNames(kf.events)
	return kf, nil
}

// XUIDNamePrefix : le prefixe du nom de REPLI, quand le kill-feed d un film ne porte pas de
// gamertag pour un joueur (il porte alors son XUID, et lui seul).
//
// EXPORTE PARCE QUE L APPELANT DOIT SAVOIR LE LIRE. Un nom `xuid:2533...` n est pas un pseudo :
// c est l identite la plus forte qui soit, et un collecteur qui le traiterait comme un gamertag
// chercherait dans son roster une cle qui n y sera jamais — il ecrirait alors des lignes SANS
// xuid, qu aucun agregat carriere ne peut joindre. C est exactement le defaut mesure le
// 2026-08-01 : 16 908 morts ecrites, 10 avec un xuid de victime.
const XUIDNamePrefix = "xuid:"

// buildFeed : regroupe les events par instant et resout les XUID en gamertags.
func buildFeed(evs []highlightevent.HighlightEvent) *killFeed {
	gt := map[uint64]string{}
	for _, e := range evs {
		if e.Gamertag != "" {
			gt[e.XUID] = e.Gamertag
		}
	}
	kf := &killFeed{xuidDe: map[string]uint64{}}
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
			name = fmt.Sprintf("%s%d", XUIDNamePrefix, e.XUID)
		}
		switch e.EventType {
		case highlightevent.EventTypeKill:
			kf.nKills++
			at(e.TimeMS).killer = name
		case highlightevent.EventTypeDeath:
			kf.nDeaths++
			ev := at(e.TimeMS)
			ev.victim = name
			ev.victimXUID = e.XUID
			kf.xuidDe[name] = e.XUID
		}
	}
	for _, v := range byTime {
		kf.events = append(kf.events, *v)
	}
	sort.Slice(kf.events, func(i, j int) bool { return kf.events[i].timeMS < kf.events[j].timeMS })
	return kf
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
