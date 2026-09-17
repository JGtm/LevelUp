package objectives

// flag_grabs_net.go — LES PRISES NETTES DE DRAPEAU : le compteur officiel, le jonglage replie.
//
// # LE DEFAUT QUE CETTE FONCTION CORRIGE
//
// `flag_grabs` (StatFlagGrabs, la table DRAPEAU du statborg — et la colonne du meme nom de
// l'API) compte CHAQUE ramassage. Or un porteur qui LANCE le drapeau devant lui pour courir plus
// vite, puis le reprend une seconde plus tard, se voit crediter une prise de plus a chaque
// aller-retour : le compteur classe le JONGLAGE, pas la PRISE. Mesure du 2026-09-13 sur les
// treize films CTF du parc local (rapport `.ai/V7.5/RAPPORT_PRISES_NETTES_2026-09-13.md`) :
// 617 prises brutes, 370 nettes a 1,5 s — 40 % du compteur officiel est du jonglage, et sur un
// film le premier au brut (27 prises) tombe TROISIEME au net (4).
//
// # LA DEFINITION, ET LES QUATRE CAS QU ELLE TRANCHE
//
// Une prise est NETTE si le portage PRECEDENT DU MEME DRAPEAU :
//
//	n existe pas                    premiere prise du drapeau                      -> NETTE
//	est d un AUTRE joueur           passe de main, vol, reprise adverse            -> NETTE
//	est du MEME joueur, ferme il y  le drapeau a vecu sa vie entre-temps           -> NETTE
//	  a plus de `window`
//	est du MEME joueur, ferme il y  JONGLAGE : le meme geste, compte une fois      -> repliee
//	  a `window` ou moins
//
// DEUX EXCEPTIONS, ET ELLES VIENNENT DE LA DONNEE :
//
//	LE DRAPEAU RENTRE CHEZ LUI ENTRE LES DEUX. Un etat `home` entre la fin du portage precedent
//	et la prise dit que le drapeau est retourne a son socle : le reprendre est une VRAIE prise,
//	quelle que soit la duree. La mesure dit que ce cas ne se produit pas sous 1,5 s (aucun retour
//	ne s effectue en une seconde et demie), mais la regle ne repose pas sur cette rarete — elle
//	lit l etat.
//
//	UN PORTAGE QUE RIEN NE FERME NE REPLIE RIEN. Un portage [FlagSpanCarriedOpen] court jusqu a
//	la fin de l axe : sa fin est une BORNE HAUTE, pas une mesure (cf. `replay.FlagStateCarriedOpen`).
//	Replier une prise sur un ecart calcule contre une borne inventee ferait disparaitre une prise
//	sur un film TRONQUE — exactement ceux que le pont par instants de mort vient de rendre
//	exploitables. Un tel portage laisse donc la prise suivante NETTE.
//
// # LA FENETRE EST UNE DONNEE DU TITRE, JAMAIS UNE CONSTANTE
//
// Elle vit dans `config/titles/{slug}/mappings/regulation.toml`
// (`[flag_grabs_net] flag_juggle_window_s`), au meme titre que le temps reglementaire ou la
// cible de victoire : c est une regle du JEU, et un titre qui ne la declare pas ne publie pas
// la grandeur. Cette fonction ne connait aucune valeur par defaut — une fenetre <= 0 rend un
// resultat NON MESURE (`Measured` faux), jamais un repli silencieux.
//
// # UNITE : LA MILLISECONDE, ET C EST L APPELANT QUI CONVERTIT
//
// Les bornes des portages arrivent en millisecondes. L artefact de rejeu, lui, publie des
// FRAMES : son lecteur multiplie par `frameIntervalMs` avant d appeler ici. Faire voyager des
// frames dans une fonction title-agnostic y ferait entrer la cadence d un format de film.

import (
	"levelup/go-api/internal/games/halo_infinite/film/types"
	"time"
)

// Les QUATRE etats d un drapeau, dans le vocabulaire de cette fonction. Ce sont les MEMES
// chaines que `replay.FlagState*` — une recopie volontaire (faire dependre `analysis` du
// decodeur de film pour quatre chaines serait un couplage disproportionne) et TENUE par un
// garde-rail cote `replay` (flag_grabs_net_bridge_test.go, TestFlagStates_SentinellesObjectiveEvents),
// qui echoue le jour ou l une des quatre diverge de sa source.
const (
	// FlagSpanCarried : un joueur le porte, et un FAIT DATE a mis fin a ce portage.
	FlagSpanCarried = "carried"
	// FlagSpanCarriedOpen : un joueur l a pris, et RIEN dans le film ne dit qu il l a lache.
	FlagSpanCarriedOpen = "carried_open"
	// FlagSpanDropped : il est au sol, la ou son dernier porteur l a laisse.
	FlagSpanDropped = "dropped"
	// FlagSpanHome : il est a sa base.
	FlagSpanHome = "home"
)

// FlagGrabsNetResult est le resultat d un match.
type FlagGrabsNetResult struct {
	// Measured dit si la grandeur est PUBLIABLE. Faux quand la fenetre n est pas declaree :
	// le titre ne connait pas la regle, donc il n y a pas de grandeur — jamais des zeros.
	Measured bool
	// WindowMS est la fenetre appliquee, en millisecondes. Elle voyage AVEC le resultat :
	// « 4 prises nettes » ne veut rien dire sans elle, et deux parcs cuits sous deux fenetres
	// differentes seraient autrement indistinguables.
	WindowMS int
	// Players : une entree par joueur ayant au moins une prise brute, triee par xuid.
	//
	// UN JOUEUR SANS PRISE N A PAS D ENTREE ICI, et ce n est pas une omission : cette
	// fonction ne connait pas le roster du match. C est a l appelant, qui le connait, de
	// completer a ZERO les joueurs presents — un zero mesure est une mesure, mais elle ne
	// peut pas se deduire de pistes ou le joueur n apparait pas.
	Players []types.FlagGrabsNetPlayer
}

// NetFlagGrabs compte, par joueur, les prises brutes et les prises nettes d un match.
//
// `tracks` porte la mesure ; `window` <= 0 rend un resultat NON MESURE.
//
// LE DENOMINATEUR DU BRUT N EST PAS ICI, ET C EST DELIBERE. Le compte des OUVERTURES de
// l oracle (`flag_grabs` + `flag_steals` fusionnes) est un fait de COUVERTURE du document
// (`coverage.flagCarries.openings`), pas une lecture des pistes : le recompter ici a partir
// d evenements que l artefact ne publie pas aurait fait un parametre toujours nil, donc un
// denominateur toujours zero. Il voyage avec la projection (`replayartifacts`), qui le lit ou
// il vit.
//
// Fonction PURE : aucune horloge, aucune base, aucune chaine de langue.
func NetFlagGrabs(tracks []types.FlagTrack, window time.Duration) FlagGrabsNetResult {
	out := FlagGrabsNetResult{}
	if window <= 0 {
		return out
	}
	out.Measured = true
	out.WindowMS = int(window / time.Millisecond)

	counts := map[string]*types.FlagGrabsNetPlayer{}
	for _, tr := range tracks {
		accumulateFlagTrack(tr, out.WindowMS, counts)
	}
	out.Players = sortedNetPlayers(counts)
	return out
}

// accumulateFlagTrack applique la regle a UNE piste de drapeau.
func accumulateFlagTrack(tr types.FlagTrack, windowMS int, counts map[string]*types.FlagGrabsNetPlayer) {
	spans := sortedFlagSpans(tr.Spans)
	var prev *types.FlagSpan // dernier PORTAGE rencontre sur cette piste
	for i := range spans {
		sp := &spans[i]
		if !isFlagCarry(sp.State) {
			continue
		}
		if sp.XUID == "" {
			// Un portage que le pont n a pas nomme n est la prise de PERSONNE. Il reste
			// neanmoins le « portage precedent » : le drapeau a bel et bien change de mains,
			// et l ignorer ferait replier une prise sur un portage qui n est plus le dernier.
			prev = sp
			continue
		}
		c := counts[sp.XUID]
		if c == nil {
			c = &types.FlagGrabsNetPlayer{XUID: sp.XUID}
			counts[sp.XUID] = c
		}
		c.Raw++
		if flagGrabEstNette(prev, sp, spans, windowMS) {
			c.Net++
		}
		prev = sp
	}
}

// flagGrabEstNette applique la definition (cf. l en-tete) a UNE prise.
func flagGrabEstNette(prev, cur *types.FlagSpan, spans []types.FlagSpan, windowMS int) bool {
	if prev == nil || prev.XUID != cur.XUID {
		return true
	}
	if prev.State == FlagSpanCarriedOpen {
		// La fin du portage precedent est une BORNE HAUTE, pas une mesure : aucun ecart ne
		// s en deduit, donc rien ne se replie.
		return true
	}
	if cur.StartMS-prev.EndMS > windowMS {
		return true
	}
	return flagRentreEntre(spans, prev.EndMS, cur.StartMS)
}

// flagRentreEntre dit si le drapeau est repasse par SON SOCLE entre deux instants. Un etat
// `home` qui CHEVAUCHE l intervalle suffit : le drapeau y etait, la reprise qui suit est une
// vraie prise.
func flagRentreEntre(spans []types.FlagSpan, deMS, aMS int) bool {
	for i := range spans {
		if spans[i].State != FlagSpanHome {
			continue
		}
		if spans[i].EndMS >= deMS && spans[i].StartMS <= aMS {
			return true
		}
	}
	return false
}

// isFlagCarry dit si l etat est un PORTAGE (les deux etats portes, fermes ou non).
func isFlagCarry(state string) bool {
	return state == FlagSpanCarried || state == FlagSpanCarriedOpen
}

// sortedFlagSpans rend une COPIE triee par instant de debut : l appelant garde son ordre, et
// deux artefacts ranges differemment rendent le meme compte.
func sortedFlagSpans(in []types.FlagSpan) []types.FlagSpan {
	out := make([]types.FlagSpan, len(in))
	copy(out, in)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].StartMS < out[j-1].StartMS; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// sortedNetPlayers rend les joueurs tries par xuid — un contrat stable, jamais l ordre
// d arrivee d une map.
func sortedNetPlayers(counts map[string]*types.FlagGrabsNetPlayer) []types.FlagGrabsNetPlayer {
	out := make([]types.FlagGrabsNetPlayer, 0, len(counts))
	for _, c := range counts {
		out = append(out, *c)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].XUID < out[j-1].XUID; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
