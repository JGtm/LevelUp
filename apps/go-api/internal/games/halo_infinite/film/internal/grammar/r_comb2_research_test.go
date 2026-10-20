//go:build research && campagne_overlay

package grammar

// r_comb2_research_test.go — RECHERCHE R-COMB-2 DE LA CAMPAGNE DE GRAMMAIRE (2026-10-02, PLAN
// §6.0 « Mesures ouvertes », critique `CRITIQUE_COMPLETUDE_R.md` A4, B5 a B9, C10, C11, E21, E22) :
// TOUS les leviers mesures de la campagne joues ENSEMBLE sur la surcouche unique post-J12, puis la
// combinaison privee d UN levier a la fois. Instrument de recherche : aucun fichier de production
// n est touche, aucune sortie ne change, `grammar.Rev` ne bouge pas.
//
// CE FICHIER EXIGE LA SURCOUCHE UNIQUE (`surcouche_unique_postj12/`, tag `campagne_overlay`), y
// compris ses deux ajouts R-COMB-2 (`frame_infer.go` : bascule L7 ; `rcomb2_leviers.go`).
//
// Les leviers (chacun est la MEME copie de recherche que sa mesure separee) :
//
//	L8   `ti=3` : low-frequency porte, high-frequency a 26 bits   ([b3CrochetTi3], BIS_3 §6)
//	L2   grammaire `ti=43` de T7                                    ([b2vLireTi43], BIS_2 §5)
//	LM   decoupage MPP 8/3 sur les formats 24-25, flux delta        (R_VEH §3.3, `r_veh_delta`)
//	L3a  moteur `ti=0/1/2` (i11, i13 a i17)                          ([rl3Crochet] « moteur », R_COMP §2.3)
//	L6a  largeurs de la ligne de l index de plage (Live Fire)       ([bis2C3], BIS_2 §3.3)
//	L6b  sites flock-position et tacmap-displayasset au jeu         ([bis2SitesJeu], BIS_2/BIS_4)
//	L4a  16 composants `ti=40`, porte `+0x818` posee                ([b2vLireTi40], R_VEH §3.1)
//	LS   localisateur `ls2` : 123 strict, fermeture par NEW de tete, signature high-frequency
//	     ([rlocLocaliser], R_LOC §3.1)
//	L1a  localisateur `tete-bloc+inv` sous condition CAUSALE : score d allocateur cumule des
//	     chunks ANTERIEURS de la marche elle-meme (>= 30 NEW, >= 50 %) ([cmTeteInv], R_NAIS §3)
//	LP   desaveu a l ouverture du chunk de la liaison d image-cle d un slot que le bloc de type 1
//	     du meme chunk dit non vivant ([rp3Diag] `desaveuKF`, R_COMP §4.4)
//	L7   NEW sur slot occupe LIE au lieu d etre refuse (bascule `rc2L7` de la surcouche)
//	L9   oracle P1-pont, derive de la marche qui porte tous les autres leviers ([b3Diag])
//
// Variantes de verification : WO (site world-object-i0 au jeu, extension de L6a), L6b-flock
// (flock-position seul), L1a-ref (condition calculee sur la marche de reference, forme de R_NAIS).

import (
	"strconv"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rc2L : les leviers d une marche ; comparable, c est la cle du cache des marches d un film.
type rc2L struct {
	l8, l2, lm, l3a, l6a, l6b, l4a bool
	ls, l1a, lp, l7, l9            bool
	wo, flock, l1aRef              bool
}

// rc2Noms : nom et acces de chaque levier ; les rc2NLots premiers forment la combinaison complete.
var rc2Noms = []struct {
	nom string
	p   func(*rc2L) *bool
}{
	{"L8", func(k *rc2L) *bool { return &k.l8 }}, {"L2", func(k *rc2L) *bool { return &k.l2 }},
	{"LM", func(k *rc2L) *bool { return &k.lm }}, {"L3a", func(k *rc2L) *bool { return &k.l3a }},
	{"L6a", func(k *rc2L) *bool { return &k.l6a }}, {"L6b", func(k *rc2L) *bool { return &k.l6b }},
	{"L4a", func(k *rc2L) *bool { return &k.l4a }}, {"LS", func(k *rc2L) *bool { return &k.ls }},
	{"L1a", func(k *rc2L) *bool { return &k.l1a }}, {"LP", func(k *rc2L) *bool { return &k.lp }},
	{"L7", func(k *rc2L) *bool { return &k.l7 }}, {"L9", func(k *rc2L) *bool { return &k.l9 }},
	{"WO", func(k *rc2L) *bool { return &k.wo }}, {"L6b-flock", func(k *rc2L) *bool { return &k.flock }},
	{"L1a-ref", func(k *rc2L) *bool { return &k.l1aRef }},
}

const rc2NLots = 12

// rc2Avec construit un jeu de leviers par leurs noms.
func rc2Avec(noms ...string) rc2L {
	var k rc2L
	for _, n := range noms {
		for _, x := range rc2Noms {
			if x.nom == n {
				*x.p(&k) = true
			}
		}
	}
	return k
}

// rc2Tous : la combinaison complete.
func rc2Tous() rc2L {
	var k rc2L
	for _, x := range rc2Noms[:rc2NLots] {
		*x.p(&k) = true
	}
	return k
}

// sans retire un levier.
func (k rc2L) sans(nom string) rc2L {
	for _, x := range rc2Noms {
		if x.nom == nom {
			*x.p(&k) = false
		}
	}
	return k
}

// nom : les leviers poses, joints par « + ».
func (k rc2L) nom() string {
	var n []string
	for _, x := range rc2Noms {
		if *x.p(&k) {
			n = append(n, x.nom)
		}
	}
	if len(n) == 0 {
		return "aucun"
	}
	return strings.Join(n, "+")
}

// rc2Film : un film ouvert, ses donnees, ses marches.
type rc2Film struct {
	f         *cmFilm
	b         *cmBlocs
	contexte  string
	format    int
	bornes    map[int][3][2]float32
	hf        map[uint32]bool
	refActifs map[int]bool
	cache     map[rc2L]*rc2Res
	oracles   map[rc2L]map[[2]int][]cmLiaison
}

// effectif : les leviers qui agissent sur ce film (LM : formats 24-25 ; L6a : bornes connues).
func (x *rc2Film) effectif(k rc2L) rc2L {
	if x.format != 24 && x.format != 25 {
		k.lm = false
	}
	if len(x.bornes) == 0 {
		k.l6a = false
	}
	return k
}

// filmDe : le film sous les leviers de profil (LM pose le decoupage 8/3 sur le flux delta).
func (x *rc2Film) filmDe(k rc2L) *cmFilm {
	fv := *x.f
	if k.lm {
		fv.cfg.Profil.MPP = profile.MPPWidths{Lead: 8, Index: 3}
	}
	return &fv
}

// poser installe les leviers de composant dans la surcouche ; [rc2Lever] les retire.
func (x *rc2Film) poser(k rc2L) {
	rc2Lever()
	b2vPorte, rc2L7 = k.l4a, k.l7
	ti3 := b3CrochetTi3(true, true)
	mot := rl3Crochet(rl3Variante{nom: "moteur", bassin: true, moteur: true})
	if k.l8 || k.l2 || k.l4a || k.l3a {
		bis2Intercepteur = func(br *Lecteur, name string, typeIndex, level uint32) (bool, bool) {
			switch {
			case k.l8 && typeIndex == 3:
				return ti3(br, name, typeIndex, level)
			case k.l2 && typeIndex == 43:
				return b2vLireTi43(br, name)
			case k.l4a && typeIndex == 40:
				return b2vLireTi40(br, name)
			case k.l3a:
				return mot(br, name, typeIndex, level)
			}
			return false, false
		}
	}
	bis2SitesJeu[bis2SiteFlockPosition] = k.l6b || k.flock
	bis2SitesJeu[bis2SiteDisplayAsset] = k.l6b
	bis2SitesJeu[bis2SiteWorldObject] = k.wo
	if k.l6a {
		bis2C3 = &bis2EtatC3{parIndex: true, bornes: x.bornes}
	}
}

func rc2Lever() {
	rcLever()
	b3Ti4Temoin, b2vPorte, rc2L7 = false, false, false
	rl3Masques = rl3Masques[:0]
}

// rc2Obs ecoute une marche : statut de chaque paquet delta, dans l ordre de la marche (0 non
// ferme, 1 ferme contredit, 2 ferme sain), records utiles fermes, score d allocateur de L1a,
// desaveu de LP, signatures du localisateur.
type rc2Obs struct {
	chk     *cmCollecteur
	b       *cmBlocs
	lp      bool
	st      []uint8
	ut      []int32
	debut   []int32
	a       rcAgg
	pred    map[uint32]int
	score   map[int][2]int
	acc     [2]int
	desav   int
	sig     map[string]int
	tr      *rlocTrace
	extra   int
	actifs  int
	garder  bool
	chunkDe int
}

func (o *rc2Obs) debutDeChunk(c int, _ []byte, _ []FilmPacket, w *World) {
	o.chk.chunk, o.chunkDe = c, c
	o.pred = o.b.predictions(c)
	if bis2C3 != nil {
		bis2C3.evenements = bis2C3.evenements[:0]
	}
	if !o.lp {
		return
	}
	for s, ti := range o.b.declares[c] {
		if e, ok := o.b.entree(c, s); ok && !e.Vivante() {
			if x, lie := w.slots[s]; lie && x.TypeIndex == ti && x.GenAny && !x.Soft {
				w.Unbind(s)
				o.desav++
			}
		}
	}
}

func (o *rc2Obs) finDeFilm() {}

func (o *rc2Obs) paquet(c int, p *cmPaquet, _ *World) {
	if bis2C3 != nil {
		bis2C3.evenements = bis2C3.evenements[:0]
	}
	rl3Masques = rl3Masques[:0]
	o.compterScore(c, p)
	o.signatures(p)
	if o.garder {
		d := int32(p.debut) //nolint:gosec // bit d un paquet
		if p.strict == -2 {
			d = -2
		}
		o.debut = append(o.debut, d)
	}
	if !p.d.Fermee {
		o.st, o.ut = append(o.st, 0), append(o.ut, 0)
		return
	}
	o.a.fermes++
	o.a.utilesFermes += p.utilesFermes
	o.ut = append(o.ut, int32(p.utilesFermes)) //nolint:gosec // compte borne
	if len(cmContredit(o.chk, p)) > 0 {
		o.st = append(o.st, 1)
		return
	}
	o.st = append(o.st, 2)
	o.a.sains++
	o.a.utilesSains += p.utilesFermes
}

// compterScore : [rnP1.compterScore], tous paquets (NEW lus proprement, predits par l allocateur).
func (o *rc2Obs) compterScore(c int, p *cmPaquet) {
	refuses := map[uint32]bool{}
	for _, rf := range p.refuses {
		refuses[rf.slot] = true
	}
	s := o.score[c]
	for _, r := range p.recs {
		if r.Type != recNew || r.DesyncAt >= 0 || refuses[r.Slot] {
			continue
		}
		predit := false
		if rang, ok := o.pred[r.Slot]; ok && rang < cmProfondeurPrediction {
			e, _ := o.b.entree(c, r.Slot)
			predit = (e.Gen+1)&3 == uint8(r.ID>>30) //nolint:gosec // deux bits
		}
		s[0]++
		o.acc[0]++
		if predit {
			s[1]++
			o.acc[1]++
		}
	}
	o.score[c] = s
}

// signatures : l archetype du record de la signature stricte du slot 123 (localisateur de
// production, `p.strict`) et la passe du localisateur LS.
func (o *rc2Obs) signatures(p *cmPaquet) {
	if p.strict == -2 {
		return
	}
	if p.strict >= 0 {
		cle := "123 strict : record non retrouve"
		for _, r := range p.recs {
			if r.HeaderBit == p.strict || r.HeaderBit == p.strict+o.extra {
				cle = "123 strict : ti=" + strconv.Itoa(int(r.TypeIndex))
				break
			}
		}
		o.sig[cle]++
	}
	if o.tr != nil {
		o.sig["passe LS : "+o.tr.passe]++
		if strings.HasPrefix(o.tr.passe, "strict-hf") {
			o.sig["slot HF : "+strconv.Itoa(int(o.tr.slotHF))]++
		}
	}
}

// rc2Actif : la condition causale de L1a (R_NAIS §3.2) sur un score cumule.
func rc2Actif(acc [2]int) bool {
	return acc[0] >= 30 && float64(acc[1]) >= rnSeuilScore*float64(acc[0])
}

// tete : le localisateur de debut de liste des leviers de marche LS et L1a (nil : production).
func (x *rc2Film) tete(k rc2L, fv *cmFilm, o *rc2Obs) func(c int) func([]byte, *World, FrameConfig) (int, bool) {
	if !k.ls && !k.l1a && !k.l1aRef {
		return nil
	}
	inv := cmTeteInv(fv, x.b, true, false)
	hf := x.hf
	return func(c int) func([]byte, *World, FrameConfig) (int, bool) {
		l1a := (k.l1a && rc2Actif(o.acc)) || (k.l1aRef && x.refActifs[c])
		if l1a {
			o.actifs++
		}
		switch {
		case !k.ls && !l1a:
			return nil
		case !k.ls:
			return inv(c)
		case !l1a:
			return func(pay []byte, w *World, cfg FrameConfig) (int, bool) {
				return rlocLocaliser("ls2", hf, pay, w, cfg, o.tr, nil)
			}
		}
		loc := inv(c)
		return func(pay []byte, w *World, cfg FrameConfig) (int, bool) {
			*o.tr = rlocTrace{passe: "l1a"}
			if d, ok := loc(pay, w, cfg); d >= 0 {
				return d, ok
			}
			_, sHF, slotHF := rlocSignatures(pay, w, cfg, hf)
			if sHF < 0 {
				o.tr.passe = "aucune"
				return -1, false
			}
			o.tr.slotHF = slotHF
			d, ok := debutParChaine(pay, sHF, candidatsDeTete(pay, sHF, w), w, cfg)
			d, ok, o.tr.passe = rlocSiChaine(d, ok, "strict-hf")
			return d, ok
		}
	}
}
