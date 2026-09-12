package replay

// identity_registry_scoreboard.go — LE TABLEAU DE L'API NOMME CE QUE LE FILM NE TABLE PAS.
//
// # LE TROU QU'IL FERME, MESURE (lot 4.3, film `4f77afc1`, Grand combat, 24 joueurs + bots)
//
// Le registre nomme 357 vies par lien direct et en laisse 18 `non_resolu`, cause
// `index_hors_table` : l'index de participant est LU dans le record de creation du bipede, mais
// `PlayerIndexTable` ne le nomme pas. Le balayage des records donne la ventilation exacte de ces
// 18 vies — trois index, et trois situations differentes :
//
//	index 25   2 vies   un bot que `BOT_METADATA` DECLARE (« 343 Donos »), qu'aucun humain ne
//	                    tient. Le film sait qui c'est ; il n'a simplement pas de xuid a poser.
//	index 26   8 vies   un SIEGE PARTAGE : `BOT_METADATA` y declare « 343 Doomfruit » ET la
//	                    table d'index y place l'humain « Narotlcs », arrive en cours a 2:46. Les
//	                    records de creation d'index 26 courent jusqu'a 4:35 du match : les
//	                    premieres vies sont celles du bot, les dernieres celles de l'humain qui
//	                    a pris son siege.
//	index 24   8 vies   un participant que NI la table NI `BOT_METADATA` ne nomme.
//
// # LA REGLE, ET CE QU'ELLE REFUSE
//
// Le tableau de l'API (`match_participants`) porte les participants du match — bots compris,
// sous leur identifiant `bid(N.0)` — et leurs BORNES DE PARTICIPATION. Il ne porte AUCUN index
// de participant du film : c'est le film qui indexe. Le tableau ne peut donc rien dire d'un
// index que le film n'attribue a personne, et la regle s'y arrete :
//
//	1. l'index est celui d'un bot que le film DECLARE, et le tableau porte son `bid` — la vie
//	   prend ce `bid` ; deux lectures concordantes, un identifiant publiable ;
//	2. le meme index est AUSSI tenu par un humain de la table d'index — la fenetre de
//	   participation du tableau tranche PAR VIE : entierement avant l'arrivee, c'est le bot ;
//	   entierement apres, c'est l'humain. A cheval, on se tait et on COMPTE ;
//	3. l'index n'est declare par personne (le cas 24) — le tableau ne dit pas QUEL index occupe
//	   un participant qu'il est seul a connaitre. Rien n'est pose, la cause `index_hors_table`
//	   reste, et le compte se publie. C'est le refus qui manque encore d'un temoin : il faudrait
//	   que le film porte l'index des bots qu'il ne declare pas (ou que `BOT_METADATA` les
//	   declare tous), et cela ne se devine pas depuis le tableau.
//
// # POURQUOI CETTE ETAPE EST APRES LE PONT PAR MORTS, ET PAS DANS LA LECTURE DIRECTE
//
// La fenetre de participation est datee sur l'axe du MATCH ; les vies vivent sur l'horloge du
// FILM. Le seul calage entre les deux est `DeathOffsetMS`, que `bestDeathOffset` mesure APRES le
// nommage direct. Sans calage apparie, aucune fenetre n'est exprimable : la regle 2 se tait
// entierement, et c'est la degradation voulue.

import (
	"log/slog"
	"strconv"

	"levelup/go-api/internal/games/canonical"
)

// NomParTableauAPI : la vie est nommee par le TABLEAU DE L'API — l'identifiant du participant
// (`bid(N.0)` d'un bot) et, sur un siege partage, la fenetre qui le departage. Cinquieme valeur
// de l'axe `nomPar` (cf. lives.go). Ce n'est ni une lecture du film, ni une deduction : c'est
// une SOURCE EXTERNE, et la section la publie comme telle (`canonical.LinkExternal`).
const NomParTableauAPI = "tableau_api"

// Participant est UNE LIGNE DU TABLEAU DE L'API, reduite a ce que le registre consomme.
//
// Le paquet `replay` n'ouvre aucune base : l'appelant (replaybuild) projette
// `domain.MatchPlayerFact` vers ce type. Une liste VIDE est legitime — un producteur sans base
// ne resout rien par cette voie, et il le publie.
type Participant struct {
	// ID est l'identifiant que la base emploie : un xuid decimal pour un humain, `bid(N.0)`
	// pour un bot. C'est la MEME forme que `RosterEntry.XUID` / `.Bid`, donc joignable sans
	// conversion.
	ID string
	// JoinedInProgress dit que la ligne declare une arrivee EN COURS de partie.
	JoinedInProgress bool
	// JoinMatchMS est l'instant de cette arrivee sur l'axe du MATCH (le meme que
	// `Death.TimeMS`). Nil = la base ne le porte pas : la fenetre n'est pas exprimable.
	JoinMatchMS *int64
}

// scoreboardReport compte ce que le tableau a nomme et ce qu'il a refuse. Les refus sont
// ventiles par CAUSE : un refus muet est exactement ce que la doctrine §0.2 interdit.
type scoreboardReport struct {
	// Lignes est le denominateur : le nombre de participants recus.
	Lignes int
	// Bots / Humains : les vies nommees, selon que le tableau y pose un `bid` ou un xuid.
	Bots, Humains int
	// Conflits : vies d'un siege PARTAGE que la fenetre ne departage pas — a cheval sur
	// l'arrivee, ou arrivee inconnue (pas de calage, pas d'instant au tableau).
	Conflits int
	// SansCandidat : vies dont l'index n'est declare par PERSONNE, ou dont le bot declare ne
	// figure pas au tableau. Le cas 3 de l'en-tete.
	SansCandidat int
}

// Nommees rend le nombre de vies que le tableau a nommees, toutes formes d'identifiant.
func (s scoreboardReport) Nommees() int { return s.Bots + s.Humains }

// resolveByScoreboard applique les trois regles de l'en-tete aux vies que la lecture directe a
// refusees faute de table. Elle ne touche AUCUNE vie deja nommee.
func (r *IdentityRegistry) resolveByScoreboard(in IdentityInput) {
	r.tableau.Lignes = len(in.Participants)
	if len(in.Participants) == 0 || len(r.creation.indexLu) == 0 {
		return
	}
	tab := lireTableau(in.Participants)
	bots, humains := bidsParIndex(in.Bots), indexToXUIDOf(in.PlayerIndices.ByXUID)
	// LES VIES SE LISENT PAR L'ACCESSEUR, jamais par les tables brutes du pont : ce fichier
	// DECIDE, il ne mute pas (garde-rail `no_identity_bridge_outside_registry_test.go`, meme
	// regle que `_elimination.go` et `_exclusion.go`). Les mutations passent par les poseurs.
	for i, l := range r.Vies() {
		if l.xuid != 0 || l.bid != "" {
			continue
		}
		if r.CauseNonResolue(i) != canonical.MethodIndexOutOfTable {
			continue
		}
		pi, lu := r.creation.indexLu[i]
		if !lu {
			continue
		}
		r.nommerParLeTableau(i, l, int(pi), tab, candidatsDIndex{bots: bots, humains: humains})
	}
	r.tableau.alarmer(in.MatchID)
}

// candidatsDIndex porte les deux tables qui peuvent revendiquer un index de participant : le
// `bid` du bot que le film declare, et le xuid que la table d'index y place. Une structure
// plutot que deux parametres de plus : le depot borne a cinq, et les deux vont toujours ensemble.
type candidatsDIndex struct {
	bots    map[int]string
	humains map[int]uint64
}

// nommerParLeTableau tranche le sort d'UNE vie. Les trois issues sont celles de l'en-tete : le
// bot, l'humain qui a pris son siege, ou le silence compte.
func (r *IdentityRegistry) nommerParLeTableau(i int, l lifeSpan, pi int, tab tableauLu,
	c candidatsDIndex) {
	bid, declare := c.bots[pi]
	if !declare || !tab.porte[bid] {
		// L'index n'est declare par personne, ou le bot du film ne figure pas au tableau : le
		// tableau n'indexe pas, il ne peut donc pas dire QUI occupe ce corps.
		r.tableau.SansCandidat++
		return
	}
	x, partage := c.humains[pi]
	if !partage {
		if r.poserBidDeVie(i, bid) {
			r.tableau.Bots++
		}
		return
	}
	arrivee, connue := tab.arriveeFilmUS(x, r.DeathOffsetMS(), r.DeathOffsetMatches())
	if !connue {
		r.tableau.Conflits++
		return
	}
	switch {
	case l.to < arrivee:
		if r.poserBidDeVie(i, bid) {
			r.tableau.Bots++
		}
	case l.from >= arrivee:
		if r.poserIdentiteDeVie(i, x, pi, true, NomParTableauAPI) {
			r.tableau.Humains++
		}
	default:
		// A CHEVAL SUR L'ARRIVEE : les deux candidats couvrent la vie. On se tait, et le compte
		// dit qu'on s'est tu — c'est le refus, pas l'absence de question.
		r.tableau.Conflits++
	}
}

// tableauLu est le tableau de l'API mis en forme pour les deux questions que le registre pose :
// « ce participant est-il au tableau ? » et « quand est-il arrive ? ».
type tableauLu struct {
	porte   map[string]bool
	arrivee map[string]int64
}

// lireTableau met le tableau en forme. Une ligne sans arrivee EN COURS n'entre pas dans
// `arrivee` : un joueur present au coup d'envoi ne departage aucun siege.
func lireTableau(lignes []Participant) tableauLu {
	t := tableauLu{porte: make(map[string]bool, len(lignes)), arrivee: map[string]int64{}}
	for _, p := range lignes {
		if p.ID == "" {
			continue
		}
		t.porte[p.ID] = true
		if p.JoinedInProgress && p.JoinMatchMS != nil {
			t.arrivee[p.ID] = *p.JoinMatchMS
		}
	}
	return t
}

// arriveeFilmUS rend l'instant d'arrivee de ce xuid sur l'horloge du FILM, en microsecondes.
//
// `connue` est faux des que l'un des deux maillons manque : l'instant au tableau, ou le calage
// du fil des morts (`horlogeFilm = horlogeMatch + DeathOffsetMS`, temoin `DeathOffsetMatches`).
// Un calage a zero EXACT est valide ; son ABSENCE ne l'est pas — c'est la meme distinction que
// [IdentityRegistry.CalageSiConnu].
func (t tableauLu) arriveeFilmUS(xuid uint64, offsetMS int64, apparies int) (int64, bool) {
	if apparies <= 0 {
		return 0, false
	}
	ms, ok := t.arrivee[strconv.FormatUint(xuid, 10)]
	if !ok {
		return 0, false
	}
	return (ms + offsetMS) * 1000, true
}

// bidsParIndex rend l'identifiant stable de chaque siege de bot du film. Un siege que PLUSIEURS
// bots declarent n'entre pas : le nom les differencie, l'index dit le siege, et servir l'un des
// deux publierait un identifiant arbitraire — la meme abstention que `botNamesBySeat`.
func bidsParIndex(bots []BotIdentity) map[int]string {
	vus := map[int][]string{}
	for _, b := range bots {
		if bid := b.Bid(); bid != "" && !contientBid(vus[b.FilmIndex], bid) {
			vus[b.FilmIndex] = append(vus[b.FilmIndex], bid)
		}
	}
	out := make(map[int]string, len(vus))
	for idx, bids := range vus {
		if len(bids) == 1 {
			out[idx] = bids[0]
		}
	}
	return out
}

func contientBid(l []string, bid string) bool {
	for _, v := range l {
		if v == bid {
			return true
		}
	}
	return false
}

// alarmer journalise ce que le tableau a pose et ce qu'il n'a pas su poser. Un refus sans cause
// ne se corrige pas ; un refus muet ne se voit meme pas.
func (s scoreboardReport) alarmer(matchID string) {
	slog.Info("rejeu : nommage par le tableau de l'API",
		"match_id", matchID, "lignes", s.Lignes, "bots", s.Bots, "humains", s.Humains,
		"conflits", s.Conflits, "sansCandidat", s.SansCandidat)
	if s.Conflits > 0 {
		slog.Warn("rejeu : siege d'index partage que la fenetre de participation ne departage "+
			"pas — vies laissees non resolues",
			"match_id", matchID, "vies", s.Conflits)
	}
	if s.SansCandidat > 0 {
		slog.Warn("rejeu : index de participant que NI la table d'index NI BOT_METADATA ne "+
			"nomme — le tableau n'indexe pas, il ne peut pas le rattacher",
			"match_id", matchID, "vies", s.SansCandidat)
	}
}
