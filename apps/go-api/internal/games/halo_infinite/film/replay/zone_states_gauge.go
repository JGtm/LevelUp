package replay

// zone_states_gauge.go — LA JAUGE EN DIRECT (schema 18) : comment la serie brute du tag 3 devient
// `ZoneState.gauge`, c'est-a-dire une jauge qui se remplit a l'image, et pas seulement son sommet.
//
// CE QUE v16 MONTRAIT, ET POURQUOI CE N'ETAIT PAS UNE JAUGE. `ZoneSpan.progress` porte le SOMMET
// de la jauge sur l'intervalle de propriete : une valeur par intervalle, souvent 1,0, tenue
// pendant toute la duree de la propriete. Dessinee en arc, elle restait pleine des minutes durant
// et se lisait comme « la zone est en cours de capture » — le contraire de ce qu'elle disait. La
// rampe pas a pas, elle, est dans le film (~40 emissions par montee, un pas de ~1 199 quanta), et
// le lecteur de production la lit deja pour apparier les slots : il ne manquait que de la PUBLIER.
//
// LES TROIS REGLES DE LA SERIE, ecrites AVANT la mesure des temoins (plan lot C-ter volet 3) :
//
//	RIEN HORS RAMPE      la serie ne porte que les montees monotones de la jauge — les captures
//	                     en cours, abouties ou non (findZoneRamps, memes seuils que l'appariement)
//	                     — et le RETOUR A ZERO qui ferme chacune quand le film le porte
//	                     (appendGaugeReset : la jauge ne redescend jamais autrement). Entre deux
//	                     rampes la jauge n'a rien a montrer.
//	ALLEGEE              dans une rampe, un point n'est publie que si la jauge a bouge d'au moins
//	                     zoneGaugeMinDelta depuis le dernier point publie, OU si une seconde s'est
//	                     ecoulee sans point (zoneGaugeMaxGapMS) : c'est ce qui borne le poids
//	                     (<= +2 % de l'artefact, mesure au journal). Le premier et le dernier
//	                     point de chaque rampe sont toujours publies : le depart et le sommet sont
//	                     ce que l'oeil lit. Cote client, l'escalier TIENT la derniere valeur
//	                     jusqu'au point suivant (une jauge figee — zone contestee — reste
//	                     affichee), et efface l'arc une seconde apres le DERNIER point de la serie.
//	MEME ECHELLE         `v` est la fraction de capture sur l'echelle du JEU — 0 = jauge au repos,
//	                     1 = pleine — exactement l'echelle de `progress` (cf. gaugeProgressOf dans
//	                     zone_states.go, et pourquoi ce n'est plus l'excursion du match) — arrondie
//	                     a trois decimales, et T est STRICTEMENT croissant.
//
// EN KOTH, RIEN : la serie n'est publiee que sur les modes a zones SIMULTANEES (Bastion), la ou le
// tag 3 est la vraie rampe de capture (97 % des captures precedees d'une rampe, lot C-bis). Sur une
// colline, le meme tag est un compteur de transfert d'environ une seconde (volet 1 du lot C-ter),
// pas la progression de garde : `buildHillStates` ne pose aucune serie, et le dit.

import (
	"math"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

const (
	// zoneGaugeMinDeltaMilli est la variation MINIMALE de la jauge entre deux points publies
	// d'une meme rampe, en milliemes de l'echelle [0, 1] : 0,02.
	zoneGaugeMinDeltaMilli = 20
	// zoneGaugeMaxGapMS est la duree au bout de laquelle une rampe re-emet un point meme si la
	// jauge n'a pas bouge de zoneGaugeMinDeltaMilli : une seconde. C'est aussi la duree pendant
	// laquelle le client tient la derniere valeur de la SERIE avant d'effacer l'arc.
	zoneGaugeMaxGapMS = 1000
	// zoneGaugeMilli est le pas d'arrondi de `v` (trois decimales).
	zoneGaugeMilli = 1000
)

// zoneGaugeWindow est un intervalle de frames [t0, t1] pendant lequel la jauge est EN MOUVEMENT :
// une rampe du slot de jauge de la zone (cf. rampWindowsOf).
type zoneGaugeWindow struct {
	t0, t1 int
}

// zoneGaugeGapFrames rend l'ecart maximal entre deux points publies d'une rampe, en frames — au
// moins une.
func zoneGaugeGapFrames(intervalMS int) int {
	if intervalMS <= 0 {
		return 1
	}
	if g := zoneGaugeMaxGapMS / intervalMS; g > 1 {
		return g
	}
	return 1
}

// zoneGaugeSeriesOf rend la serie allegee des emissions `ss` d'un slot de jauge sur les fenetres
// `wins`, ramenee a [0, 1] sur l'echelle du jeu (gaugeProgressOf). Rend nil sans fenetre ni
// emission.
func zoneGaugeSeriesOf(ss []zoneSample, wins []zoneGaugeWindow, gap int) []GaugePoint {
	if len(wins) == 0 || len(ss) == 0 {
		return nil
	}
	sort.SliceStable(wins, func(i, j int) bool { return wins[i].t0 < wins[j].t0 })
	var out []GaugePoint
	for _, w := range wins {
		out = appendGaugeWindow(out, ss, w, gap)
	}
	return out
}

// appendGaugeWindow pousse les points d'UNE fenetre : le premier et le dernier toujours, et entre
// les deux ceux qui ont bouge d'au moins zoneGaugeMinDeltaMilli ou qui suivent d'au moins `gap`
// frames le dernier point publie — puis le RETOUR A ZERO qui ferme la rampe, quand le film le
// porte (cf. appendGaugeReset).
func appendGaugeWindow(out []GaugePoint, ss []zoneSample, w zoneGaugeWindow, gap int) []GaugePoint {
	out, i := appendGaugeThinned(out, ss, w.t0, w.t1, gap)
	return appendGaugeReset(out, ss, i)
}

// appendGaugeThinned est L ALLEGEMENT SEUL, sans le retour a zero : le premier et le dernier
// point de la fenetre toujours, et entre les deux ceux qui ont bouge d au moins
// `zoneGaugeMinDeltaMilli` ou qui suivent d au moins `gap` frames le dernier point publie. Rend
// aussi l index du PREMIER echantillon au-dela de la fenetre, pour que l appelant decide lui-meme
// ce qu il en fait.
//
// EXTRAIT DE `appendGaugeWindow` LE 2026-09-19 (montee 63) POUR SON SECOND APPELANT : la jauge de
// RETOUR DU DRAPEAU (`flag_return_gauge.go`) veut le meme allegement, la meme echelle et le meme
// escalier, mais PAS le retour a zero — sa serie est publiee DANS un intervalle de lacher, et le
// zero qui suit tombe apres la fin de cet intervalle (le drapeau est rentre ou repris). L y
// pousser publierait un point hors des bornes du span.
func appendGaugeThinned(out []GaugePoint, ss []zoneSample, t0, t1, gap int) ([]GaugePoint, int) {
	i := sort.Search(len(ss), func(k int) bool { return ss[k].t >= t0 })
	first, lastT, lastM := true, 0, 0
	for ; i < len(ss) && ss[i].t <= t1; i++ {
		s := ss[i]
		m := gaugeMilliOf(s.v)
		last := i+1 >= len(ss) || ss[i+1].t > t1
		if !first && !last && m-lastM < zoneGaugeMinDeltaMilli && s.t-lastT < gap {
			continue
		}
		out = pushGaugePoint(out, GaugePoint{T: s.t, V: float32(m) / zoneGaugeMilli})
		first, lastT, lastM = false, s.t, m
	}
	return out, i
}

// appendGaugeReset publie le RETOUR A ZERO qui suit une rampe : la premiere emission apres la
// fenetre, si elle vaut le zero du jeu (ou moins).
//
// POURQUOI CE POINT EST LA FIN DE LA RAMPE, ET PAS UN POINT « HORS RAMPE » (mesure du 2026-08-19,
// `echelle_7344d24f.log`) : la jauge ne REDESCEND JAMAIS pas a pas — sur les trois slots de jauge
// des zones de Bastion, TOUS les pas descendants (18, 18 et 16) sont des retours au zero exact, et
// il n'y a aucun pas nul. Le canal est une marche d'escalier : il monte tant qu'on capture, se TAIT
// tant que la capture est figee (zone contestee : 29 s a 0,92 sur `7344d24f`) ou abandonnee, et
// se remet a zero d'une seule emission — a la capture menee a terme (une frame apres le sommet)
// comme a l'abandon (1,4 s a 11 s apres le dernier pas). Sans ce point, le client ne peut pas
// distinguer « figee » de « finie » ; avec lui, il tient la derniere valeur jusqu'au point suivant
// et efface l'arc quand le film le dit. Un retour a zero deja publie comme DEPART de la rampe
// suivante n'entre pas deux fois : pushGaugePoint le fond sur la meme frame.
func appendGaugeReset(out []GaugePoint, ss []zoneSample, next int) []GaugePoint {
	if next >= len(ss) || ss[next].v > zoneGaugeQuantZero {
		return out
	}
	return pushGaugePoint(out, GaugePoint{T: ss[next].t, V: 0})
}

// gaugeMilliOf ramene un quantum a l'echelle du jeu, en milliemes de [0, 1].
func gaugeMilliOf(q uint64) int {
	return int(math.Round(float64(gaugeProgressOf(q)) * zoneGaugeMilli))
}

// pushGaugePoint ajoute un point en gardant T STRICTEMENT croissant : un point sur la MEME frame
// que le precedent le REMPLACE (la valeur que la frame porte a sa fin est celle que l'escalier
// doit tenir — c'est aussi ce qui fond un retour a zero avec le depart de la rampe suivante), un
// point ANTERIEUR est ecarte (la serie ne revient jamais en arriere).
func pushGaugePoint(out []GaugePoint, p GaugePoint) []GaugePoint {
	if n := len(out); n > 0 && out[n-1].T >= p.T {
		if out[n-1].T == p.T {
			out[n-1] = p
		}
		return out
	}
	return append(out, p)
}

// zoneGaugeRampComplete est LE SOMMET A PARTIR DUQUEL UNE RAMPE A ABOUTI, sur l'echelle du jeu.
//
// LA VALEUR VIENT D'UNE MESURE, PAS D'UN REGLAGE (2026-09-20, 8 documents a zones du cache,
// 241 rampes). Separees par ce que le canal de PROPRIETE fait apres leur sommet :
//
//	160 rampes  suivies d'une bascule de camp dans la fenetre — sommets de 0,976 a 0,999 ;
//	 81 rampes  aucune bascule — sommets de 0,060 a 0,986, mais deux seulement au-dessus de
//	            0,95 (0,983 et 0,986). Ces deux-la sont des RE-SECURISATIONS par le camp deja
//	            en place : le canal ne change pas de valeur, donc `mergeZoneRuns` n'ouvre pas
//	            d'intervalle — et pourtant la valeur qu'il porte EST celle du pousseur.
//
// Hors ces deux cas, le plus haut sommet SANS bascule vaut 0,938 : le seuil tombe dans une marge
// mesuree de 0,038. Une capture menee a terme culmine juste sous 1,0 (cf. l'en-tete de l'echelle
// dans zone_states.go), jamais a 1,0 exactement — d'ou un seuil et non une egalite.
const zoneGaugeRampComplete = 0.95

// zoneGaugeRampsOf rend les rampes PUBLIEES d'une zone : les memes que celles dont la serie de
// jauge est tiree (`findZoneRamps`), chacune portant le camp qui la pousse.
//
// LE CAMP EST LU DANS LE FILM, ET PLUS DEDUIT DE L'ISSUE (lot 5.6) : le film porte un second
// canal `tag 4` par zone — le POUSSEUR —, dont la valeur pendant la rampe nomme le camp qui la
// mene, qu'elle aboutisse ou non. Son election et sa mesure vivent dans
// `zone_states_capturer.go`. La deduction du schema 64 survit en REPLI NOMME pour les zones ou
// aucun canal n'est elu : sans elle, ces zones perdraient le camp qu'elles publient deja.
func zoneGaugeRampsOf(ramps []zoneRamp, owner, capt []zoneSample, c zoneRampsCtx,
) []ZoneGaugeRamp {
	if len(ramps) == 0 {
		return nil
	}
	out := make([]ZoneGaugeRamp, 0, len(ramps))
	for _, r := range ramps {
		out = append(out, ZoneGaugeRamp{
			T0: r.t0, T1: r.tPeak,
			CapturingTeam: rampCapturingTeam(r, owner, capt, c),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].T0 < out[j].T0 })
	return out
}

// zoneRampsCtx porte ce dont la publication des rampes a besoin (regle des 5 parametres).
type zoneRampsCtx struct {
	teams map[uint64]bool
	win   int
	fb    *fallback.Compteur
}

// rampCapturingTeam rend le camp qui pousse la rampe : LU d'abord, deduit ensuite.
//
// L'ORDRE EST LE POINT (`OrdreApresLecture`) : le canal pousseur elu a le dernier mot, MEME
// quand il nomme le neutre — « le film dit que personne ne pousse » est une reponse, et la
// deduire par-dessus publierait un camp que le film contredit. Le repli ne s'exerce donc que
// la ou la LECTURE N'A PAS EU LIEU : aucun canal elu pour cette zone, ou canal muet sur cette
// rampe.
func rampCapturingTeam(r zoneRamp, owner, capt []zoneSample, c zoneRampsCtx) *int {
	if team, lu := zoneRampCapturerRead(capt, r, c.teams); lu {
		return team
	}
	return zoneRampCapturerDeduit(r, owner, c.teams, c.win, c.fb)
}

// rampWindowsOf traduit des rampes en fenetres de jauge.
func rampWindowsOf(ramps []zoneRamp) []zoneGaugeWindow {
	out := make([]zoneGaugeWindow, 0, len(ramps))
	for _, r := range ramps {
		out = append(out, zoneGaugeWindow{t0: r.t0, t1: r.tPeak})
	}
	return out
}

// tallyZoneStates compte ce que les etats publient, toutes zones confondues : les intervalles et
// les points de jauge en direct.
func tallyZoneStates(states []ZoneState, cov *ZonesCoverage) {
	for _, s := range states {
		cov.Spans += len(s.Spans)
		cov.GaugePoints += len(s.Gauge)
		if s.LetterRank != nil {
			cov.Letters++
		}
	}
}
