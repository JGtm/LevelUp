package grammar

// tir_continu.go — LE TIR CONTINU LU DANS LA VUE DE CONTROLE : des verdicts de paquet aux RAFALES
// (lot M4b de la campagne « retours rejeu », 2026-09-24).
//
// # LA REGLE, ECRITE AVANT LA MESURE, ET ELLE EST CELLE DE THEATER
//
// Le lecteur du jeu (`FUN_14076b838`, code 0xd) COPIE l entree d un participant dans sa case
// (`vue + 0x25f0 + i * 0x68`) : l etat de ses gachettes PERSISTE jusqu a l entree suivante, et
// le barillet tire a sa cadence tant que le bit est pose. Une rafale est donc :
//
//	DEBUT   la premiere entree LUE qui pose le bit, apres une entree lue qui ne le posait pas ;
//	FIN     la premiere entree LUE qui ne le pose plus.
//
// Entre deux entrees lues, un paquet lu (vue C fermee) qui ne porte pas l entree du joueur ne
// change rien : l etat persiste, comme chez le lecteur du jeu.
//
// # LE TROU, ET CE QU IL COUTE (decision de l utilisateur du 2026-09-24)
//
// Un paquet dont la vue C n est pas lue — liste d evenements non localisee, vue B ouverte, vue C
// qui ne ferme pas — peut porter un lacher que nous ne voyons pas : de lui a la prochaine entree
// lue du joueur, l etat de sa gachette est INCONNU. Rien n autorise a le supposer tenu. Le trou
// est donc PORTE par la rafale en cours, et le rejeu s y TAIT :
//
//	l entree lue apres le trou TIENT la gachette  -> la rafale continue ; le passage
//	                                                [premier paquet non lu, cette entree) est un
//	                                                TROU INTERIEUR, muet ;
//	elle ne la tient PLUS                          -> le lacher est DANS le trou : la rafale finit
//	                                                au premier paquet non lu (borne « trou ») ;
//	un bit qui n etait pas tenu avant le trou      -> s il est pose apres, la rafale commence a
//	                                                cette entree (borne « trou »).
//
// Aucun balayage, aucune lecture de repli, aucune duree supposee : le compte le dit
// (`Holes`, `HoleRuns`, `BurstsWithHole`, `InnerHoles`, `HeldHoleMS`).
//
// # UNE ENTREE SANS BLOC NE DIT RIEN
//
// Le bloc de 0x68 octets (`a = R(1)` de `FUN_1406d0388`) est facultatif. Une entree qui ne le porte
// pas ne transmet aucune gachette : elle ne pose ni ne lache rien, et ne leve pas l inconnu d un
// trou. Un bloc PRESENT dont la garde d action est fermee, lui, dit « aucune gachette » : c est un
// lacher.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// joueursDeControle est la plage de l index de controle : R(5).
const joueursDeControle = 1 << largeurIndexControle

// cleBitDeTir designe UN bit de tir d un joueur : sa main, sa nature (gachette ou barillet) et
// son rang.
type cleBitDeTir struct {
	joueur, main int
	barillet     bool
	rang         int
}

// etatBitDeTir est l etat courant d un bit tenu : la rafale en cours, et le trou ouvert s il y
// en a un (`troue`, depuis `trou` : l instant du premier paquet non lu).
type etatBitDeTir struct {
	debut      uint64
	debutBorne string
	entrees    int
	arme       int
	trous      []types.ContinuousFireHole
	troue      bool
	trou       uint64
}

// collecteurTirContinu recoit le verdict de vue C de chaque paquet et plie les entrees en rafales.
type collecteurTirContinu struct {
	st  *types.ContinuousFireStats
	out []types.ContinuousFireBurst
	// ts : l instant du paquet en cours ; recu / lecture : son verdict, quand la marche l a publie.
	ts      uint64
	recu    bool
	lecture LectureVueC
	dernier uint64
	enTrou  bool
	// tenus : les bits TENUS, avec leur rafale en cours.
	tenus map[cleBitDeTir]*etatBitDeTir
	// inconnu : l etat du joueur depuis un trou, jusqu a sa prochaine entree lue.
	inconnu [joueursDeControle]bool
}

func nouveauCollecteurTirContinu(st *types.ContinuousFireStats) *collecteurTirContinu {
	return &collecteurTirContinu{st: st, tenus: map[cleBitDeTir]*etatBitDeTir{}}
}

// ouvrir annonce un paquet delta a l instant `ts`.
func (c *collecteurTirContinu) ouvrir(ts uint64) {
	c.ts, c.recu, c.lecture = ts, false, LectureVueC{}
}

// recevoir est le crochet de la marche ([Observation.VueControleHook]) : UN verdict par paquet.
func (c *collecteurTirContinu) recevoir(l LectureVueC) {
	if c.recu {
		return // un seul verdict par paquet : la marche n en publie qu un
	}
	c.recu, c.lecture = true, l
}

// fermer applique le verdict du paquet ouvert. `nonLocalise` : le paquet porte une liste
// d evenements que le localisateur n a pas su sauter — la marche n a rien lu.
func (c *collecteurTirContinu) fermer(nonLocalise bool) {
	c.st.Packets++
	c.dernier = c.ts
	l := c.lecture
	if l.Atteinte {
		c.st.Reached++
	}
	if nonLocalise || !l.Fermee {
		c.ventilerTrou(nonLocalise, l)
		c.trou()
		return
	}
	c.st.Closed++
	c.enTrou = false
	for _, e := range l.Entrees {
		c.st.Entries++
		if !e.Bloc {
			continue // aucune information sur les gachettes
		}
		c.st.WithAction++
		if e.Action.Tire() {
			c.st.Firing++
		}
		c.entree(e)
	}
}

// ventilerTrou range un paquet non lu sous sa cause.
func (c *collecteurTirContinu) ventilerTrou(nonLocalise bool, l LectureVueC) {
	c.st.Holes++
	switch {
	case nonLocalise:
		c.st.Unlocated++
	case !l.Atteinte:
		c.st.OpenViewB++
	case l.Arret == ArretVueCDebordement:
		c.st.StopOverflow++
	case l.Arret == ArretVueCKindNonPorte:
		c.st.StopKind++
	case l.Arret == ArretVueCBlocBC:
		c.st.StopBlockBC++
	case l.Arret == ArretVueCPlafond:
		c.st.StopCap++
	default:
		c.st.NotClosing++
	}
}

// trou ouvre, sur chaque bit tenu qui n en a pas deja un, un trou a l instant du paquet non lu,
// et rend inconnu l etat de tous les joueurs.
func (c *collecteurTirContinu) trou() {
	if !c.enTrou {
		c.st.HoleRuns++
		c.enTrou = true
	}
	for _, s := range c.tenus {
		if !s.troue {
			s.troue, s.trou = true, c.ts
		}
	}
	for i := range c.inconnu {
		c.inconnu[i] = true
	}
}

// entree applique UNE entree lue qui porte le bloc : ses dix bits de tir.
func (c *collecteurTirContinu) entree(e EntreeDeControle) {
	if e.Index < 0 || e.Index >= joueursDeControle {
		return
	}
	for main := 0; main < 2; main++ {
		arme := e.Action.Arme[main]
		for rang := 0; rang < entreesDeGachetteParMain; rang++ {
			k := cleBitDeTir{joueur: e.Index, main: main, rang: rang}
			c.bit(k, e.Action.Gachettes[main]&(1<<rang) != 0, arme)
		}
		for rang := 0; rang < barilletsParMain; rang++ {
			k := cleBitDeTir{joueur: e.Index, main: main, barillet: true, rang: rang}
			c.bit(k, e.Action.Barillets[main]&(1<<rang) != 0, arme)
		}
	}
	c.inconnu[e.Index] = false
}

// bit applique la valeur lue d UN bit de tir.
func (c *collecteurTirContinu) bit(k cleBitDeTir, pose bool, arme int) {
	s := c.tenus[k]
	switch {
	case s == nil && pose:
		borne := types.ContinuousFireBoundPressed
		if c.inconnu[k.joueur] {
			borne = types.ContinuousFireBoundHole
		}
		c.tenus[k] = &etatBitDeTir{debut: c.ts, debutBorne: borne, entrees: 1, arme: arme}
	case s == nil:
		return
	case pose && s.troue: // tenu avant ET apres le trou : un passage interieur, muet
		s.trous = append(s.trous, types.ContinuousFireHole{StartUS: s.trou, EndUS: c.ts})
		c.st.HeldHoleMS += msEntre(s.trou, c.ts)
		s.troue = false
		s.entrees++
	case pose:
		s.entrees++
	case s.troue: // lache DANS le trou : la rafale finit a son debut
		c.st.HeldHoleMS += msEntre(s.trou, c.ts)
		c.clore(k, s, s.trou, types.ContinuousFireBoundHole)
	default:
		c.clore(k, s, c.ts, types.ContinuousFireBoundReleased)
	}
}

// msEntre rend la duree en millisecondes entre deux instants croissants du film.
func msEntre(a, b uint64) int64 {
	if b <= a {
		return 0
	}
	return int64((b - a) / 1000) //nolint:gosec // duree d un film, bien sous 2^63 µs
}

// clore rend la rafale d un bit, finie a `fin` pour la cause `borne`, et libere le bit.
func (c *collecteurTirContinu) clore(k cleBitDeTir, s *etatBitDeTir, fin uint64, borne string) {
	r := types.ContinuousFireBurst{FilmIndex: k.joueur, Hand: k.main, Barrel: k.barillet,
		Input: k.rang, Weapon: s.arme, StartUS: s.debut, EndUS: fin, StartBound: s.debutBorne,
		EndBound: borne, Entries: s.entrees, Holes: s.trous}
	c.out = append(c.out, r)
	c.st.Bursts++
	c.st.InnerHoles += len(r.Holes)
	if r.StartBound == types.ContinuousFireBoundHole || r.EndBound == types.ContinuousFireBoundHole ||
		len(r.Holes) > 0 {
		c.st.BurstsWithHole++
	}
	delete(c.tenus, k)
}

// terminer clot les rafales encore tenues au dernier paquet et rend la liste TRIEE. Une rafale
// dont un trou est ouvert finit au debut du trou : rien ne dit qu elle a dure au-dela.
func (c *collecteurTirContinu) terminer() []types.ContinuousFireBurst {
	for k, s := range c.tenus {
		if s.troue {
			c.st.HeldHoleMS += msEntre(s.trou, c.dernier)
			c.clore(k, s, s.trou, types.ContinuousFireBoundHole)
			continue
		}
		c.clore(k, s, c.dernier, types.ContinuousFireBoundFilmEnd)
	}
	sortContinuousFire(c.out)
	return c.out
}

// sortContinuousFire ordonne les rafales sur un ordre TOTAL (debut, joueur, main, nature, rang) :
// le document ne doit pas dependre de l iteration d une table.
func sortContinuousFire(out []types.ContinuousFireBurst) {
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch {
		case a.StartUS != b.StartUS:
			return a.StartUS < b.StartUS
		case a.FilmIndex != b.FilmIndex:
			return a.FilmIndex < b.FilmIndex
		case a.Hand != b.Hand:
			return a.Hand < b.Hand
		case a.Barrel != b.Barrel:
			return !a.Barrel
		default:
			return a.Input < b.Input
		}
	})
}
