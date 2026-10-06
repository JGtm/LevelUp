package grammar

// canal_des_morts.go — LES MORTS D OBJET ET L OCCUPATION, UN CANAL DE LA MARCHE DES TRAMES (ADR 0037
// IR-2 et IR-6).
//
// Le canal recolte, trame par trame, les records de la vue B que la marche a lus : les dead-states
// lisibles de tous les archetypes et les lectures d `object-parent-state` de la bande bipede, sous la
// regle de [objectDeathHarvest] (record entierement porte, ou rupture apres le dead-state). Il lit
// la trace de capture des records — les valeurs que la couche de capture rend, jamais un octet.
//
// # LES LISTES QUE LA MARCHE NE LOCALISE PAS
//
// Une trame dont la liste d evenements n est pas localisee n est pas lue par la marche
// ([localiserLaListe]). Pour ce canal seul, sa vue B est RECUPEREE : le localisateur unique lui
// cherche un debut dans l ordre des marches qui lisent les morts ([SignaturePuisLargeurLibre] : la
// signature stricte, que la marche vient deja de chercher en vain, puis le repli a largeur libre),
// et la vue B est lue de la sous le monde de la marche, rendu intact. Les autres canaux ne voient
// pas ces records ; la marche ne les lie pas. Chaque liste recuperee par le repli se compte au
// repli `repli_localisation_largeur_libre` (registre des replis, ordre « apres la lecture »).

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// canalDesMorts est le canal des morts d objet et de l occupation ([CanalDesTrames]).
type canalDesMorts struct {
	reg *Registry
	m   *MarcheDistribuee
	st  ObjectDeathStats
	h   objectDeathHarvest
	// largeurLibre : les listes recuperees par le repli a largeur libre.
	largeurLibre int
	// cadrePose : [ObjectDeathStats.Config] a ete pris de la marche.
	cadrePose bool
}

// nouveauCanalDesMorts prepare le canal des morts d un film de registre `reg`.
func nouveauCanalDesMorts(reg *Registry) *canalDesMorts {
	c := &canalDesMorts{reg: reg, st: newObjectDeathStats()}
	c.h = objectDeathHarvest{reg: reg, idx: map[uint32]int{}, st: &c.st}
	return c
}

// ImageCle compte les paquets d image-cle que la marche a lies au monde : le premier denominateur
// de la lecture ([ObjectDeathStats.Keyframes]).
func (c *canalDesMorts) ImageCle(*lecture.Paquet, *MarcheDistribuee) { c.st.Keyframes++ }

// Interets : le dead-state de chaque archetype du registre qui le declare, et l etat de parente du
// bipede.
func (c *canalDesMorts) Interets() []Interet {
	var out []Interet
	if c.reg != nil {
		for ti, a := range c.reg.Archetypes {
			if componentIndexOfAny(a, deadStateComponentName) >= 0 {
				out = append(out, Interet{Phase: PhaseTrames, TI: ti, Composant: deadStateComponentName})
			}
		}
	}
	return append(out, Interet{Phase: PhaseTrames, TI: BipedTypeIndex, Composant: compObjectParentState})
}

// Brancher garde ce que la marche expose ; le canal ne pose aucun crochet.
func (c *canalDesMorts) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }

// Trame recolte UNE trame delta : ses records lus, ou, pour une liste que la marche n a pas
// localisee, ceux de la vue B recuperee.
func (c *canalDesMorts) Trame(p *lecture.Paquet) {
	if !c.cadrePose {
		c.st.Config, c.cadrePose = c.m.cadreDeLaMarche(), true
	}
	c.st.Deltas++
	avecEvenements := listeAnnoncee(&p.VueA)
	if avecEvenements {
		c.st.EventPackets++
	}
	recs, lus := c.m.recordsDeLaTrame()
	if !lus && avecEvenements {
		var libre bool
		recs, lus, libre = c.m.recupererLaListe()
		c.largeurLibre += unSi(libre)
	}
	if !lus {
		return
	}
	if avecEvenements {
		c.st.LocatedPackets++
	}
	c.st.Packets++
	c.h.harvest(recs, p.TS)
}

// Clore : rien — le canal se lit par [canalDesMorts.resultat].
func (c *canalDesMorts) Clore(BilanDeMarche) {}

// resultat rend les morts et l occupation, triees et dedoublonnees, et les denominateurs.
func (c *canalDesMorts) resultat() ([]types.ObjectDeath, []types.VehicleOccupancy, ObjectDeathStats) {
	return dedupObjectDeaths(c.h.out), dedupOccupancy(c.h.rides), c.st
}

// cadreDeLaMarche rend le cadre de la marche des trames, sans son observation : un cadre se
// publie, des crochets non.
func (m *MarcheDistribuee) cadreDeLaMarche() FrameConfig {
	cfg := m.marche.cfg
	cfg.Obs = nil
	return cfg
}

// recordsDeLaTrame rend les records de la vue B que la marche vient de lire, et faux pour une liste
// d evenements qu elle n a pas localisee.
func (m *MarcheDistribuee) recordsDeLaTrame() ([]FrameRecord, bool) {
	t := &m.marche.trame
	return t.lecture.recs, t.debut >= 0
}

// debutRecupere rend le debut de la vue B d une liste que la cuisson n a pas localisee, dans
// l ordre des sites qui lisent les morts, et si ce debut vient du repli a largeur libre ; -1 sans
// debut.
func debutRecupere(pay []byte, w *World, cfg FrameConfig) (int, bool) {
	return LocaliserBoucleDeRecords(pay, w, cfg, SignaturePuisLargeurLibre)
}

// recupererLaListe lit la vue B de la trame en cours, dont la marche n a pas localise la liste
// d evenements, depuis [debutRecupere]. Le monde de la marche est rendu intact, et l observation
// n est pas posee : aucun crochet ne publie ces records. Rend les records, s ils ont ete lus, et si
// le debut vient du repli a largeur libre.
func (m *MarcheDistribuee) recupererLaListe() (recs []FrameRecord, lus, libre bool) {
	mt := m.marche
	cfg := mt.cfg
	cfg.Obs = nil
	pay := m.Paquet.Payload
	s, libre := debutRecupere(pay, mt.monde, cfg)
	if s < 0 {
		return nil, false, false
	}
	defer mt.monde.Restore(mt.monde.Snapshot())
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	br.Skip(s)
	recs, _ = DecodeFrameRecords(br, mt.monde, cfg)
	return recs, true, libre
}
