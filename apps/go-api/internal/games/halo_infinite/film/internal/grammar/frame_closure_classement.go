package grammar

// frame_closure_classement.go — LE CLASSEMENT D UN PAQUET MARCHE (lot J4.0, `frame_closure.go`).
//
// Tout ce qui est ici est une fonction de ce que les marcheurs ont DEJA rendu : les records de la
// vue B, le nombre de rangs lus, le verdict de la vue C. Aucune lecture de bits.
//
// # LA PREMIERE CAUSE D ARRET D UN PAQUET, ET POURQUOI C EST ELLE QUI COMPTE
//
// Une trame se lit en sequence : la premiere chose qui arrete la marche arrete tout ce qui suit.
// Le bloquant d un paquet est donc, dans l ordre de la marche : la liste d evenements non
// localisee, le message de la vue A non porte, le composant sans lecteur (ou le slot non lie) du
// record de la vue B qui a desynchronise, la fin de payload atteinte dans la vue B, la cause
// d arret de la vue C, et enfin le terminateur de la vue C qui ne ferme pas le paquet — ce
// dernier dit qu une largeur est fausse QUELQUE PART devant, sans dire ou.

import "fmt"

// Les causes d arret qui ne sont pas un composant. Elles nourrissent un golden : leur texte est
// un contrat.
const (
	causeListeNonLocalisee    = "liste d evenements non localisee"
	causeMessageNonPorte      = "vue A : message non porte"
	causeFinDePayloadVueB     = "vue B : fin de payload"
	causeSlotNonLie           = "vue B : slot non lie (archetype non resolu)"
	causeTerminateurHorsCadre = "vue C : terminateur hors cadre"
)

// nomArretVueC nomme une cause d arret de la vue C ([ArretVueC]).
func nomArretVueC(a ArretVueC) string {
	switch a {
	case ArretVueCDebordement:
		return "vue C : debordement"
	case ArretVueCKindNonPorte:
		return "vue C : kind non porte"
	case ArretVueCBlocBC:
		return "vue C : bloc 0xbc non porte"
	case ArretVueCPlafond:
		return "vue C : plafond de tours"
	}
	return fmt.Sprintf("vue C : arret %d", int(a))
}

// paquetMarche est ce que la marche a rendu pour UN paquet.
type paquetMarche struct {
	// enTete : la marche est partie de la tete du paquet, donc a lu la vue A.
	enTete bool
	recs   []FrameRecord
	// rangs : les vues lues jusqu a leur terminateur, comptees depuis le point de depart.
	rangs int
	// vueC : le verdict publie au crochet ; `Atteinte` dit que la vue B s est terminee.
	vueC LectureVueC
}

// vueBAtteinte : la marche est entree dans la vue B — la vue A s est terminee, ou n etait pas a lire.
func (p paquetMarche) vueBAtteinte() bool { return !p.enTete || p.rangs >= 1 }

// bloquant est une cause d arret, nommee, avec le composant qu elle designe s il y en a un.
type bloquant struct {
	nom       string
	ti, index int
	composant string
}

// causeSimple rend une cause qui ne designe aucun composant.
func causeSimple(nom string) bloquant { return bloquant{nom: nom, ti: -1, index: -1} }

// classer range UN paquet marche dans la carte.
func (m *mesureDesTrames) classer(p paquetMarche) {
	m.rep.Paquets++
	ferme := p.vueC.Fermee
	var cause bloquant
	if ferme {
		m.rep.PaquetsFermes++
		m.rep.Utiles.EntreesDeControleFermees += len(p.vueC.Entrees)
	} else {
		cause = m.bloquantDuPaquet(p)
	}
	m.classerLesVues(p, cause.nom)
	enJeu := m.classerLesRecords(p.recs, ferme, cause)
	if !ferme {
		m.compterBloquant(cause, enJeu)
	}
}

// listeNonLocalisee compte un paquet a liste d evenements dont aucune vue n a ete lue.
func (m *mesureDesTrames) listeNonLocalisee() {
	m.rep.Paquets++
	m.rep.ListesNonLocalisees++
	m.compterBloquant(causeSimple(causeListeNonLocalisee), 0)
}

// bloquantDuPaquet rend la PREMIERE cause d arret d un paquet qui ne ferme pas.
func (m *mesureDesTrames) bloquantDuPaquet(p paquetMarche) bloquant {
	switch {
	case !p.vueBAtteinte():
		return causeSimple(causeMessageNonPorte)
	case !p.vueC.Atteinte:
		if n := len(p.recs); n > 0 {
			if b, ok := m.bloquantDuRecord(p.recs[n-1]); ok {
				return b
			}
		}
		return causeSimple(causeFinDePayloadVueB)
	case p.vueC.Arret != ArretVueCAucun:
		return causeSimple(nomArretVueC(p.vueC.Arret))
	}
	return causeSimple(causeTerminateurHorsCadre)
}

// classerLesVues compte le paquet dans chaque vue atteinte. Les vues A et B ne comptent un arret
// que si elles ne se terminent pas ; la vue C, dernier rang, en compte un des qu elle ne ferme pas.
func (m *mesureDesTrames) classerLesVues(p paquetMarche, cause string) {
	ferme := p.vueC.Fermee
	if p.enTete {
		terminee := p.rangs >= 1
		m.compterVue(VueMessages, terminee, ferme, cause)
	}
	if p.vueBAtteinte() {
		m.compterVue(VueEntites, p.vueC.Atteinte, ferme, cause)
	}
	if p.vueC.Atteinte {
		s := &m.rep.Vues[VueControle]
		s.Atteints++
		if p.vueC.Arret == ArretVueCAucun {
			s.Terminees++
		}
		if ferme {
			s.Fermes++
			return
		}
		compterArret(s, cause)
	}
}

// compterVue compte un paquet dans la vue A ou B.
func (m *mesureDesTrames) compterVue(v VueDeTrame, terminee, ferme bool, cause string) {
	s := &m.rep.Vues[v]
	s.Atteints++
	switch {
	case !terminee:
		compterArret(s, cause)
	case ferme:
		s.Terminees++
		s.Fermes++
	default:
		s.Terminees++
	}
}

// compterArret ajoute un arret a une vue.
func compterArret(s *FrameViewStat, cause string) {
	if s.Arrets == nil {
		s.Arrets = map[string]int{}
	}
	s.Arrets[cause]++
}

// classerLesRecords range les records NEW et DELTA de la vue B par archetype et rend le nombre de
// records utiles non fermes. Un record ne ferme que dans un paquet ferme ; un record non ferme est
// impute a SA desynchronisation s il en a une, a la cause du paquet sinon.
func (m *mesureDesTrames) classerLesRecords(recs []FrameRecord, ferme bool, cause bloquant) int {
	enJeu := 0
	for _, r := range recs {
		if r.Type != recNew && r.Type != recDelta {
			continue
		}
		ti := cleArchetype(r)
		s := m.rep.Archetypes[ti]
		fermeR := ferme && r.DesyncAt < 0
		utile := m.recordUtile(r, ti)
		compterRecord(&s, r.Type == recNew, fermeR)
		if utile {
			s.Utiles++
			m.rep.Utiles.Records++
			if fermeR {
				s.UtilesFermes++
				m.rep.Utiles.RecordsFermes++
			} else {
				enJeu++
			}
		}
		m.rep.Archetypes[ti] = s
		if !fermeR {
			c := cause
			if b, ok := m.bloquantDuRecord(r); ok {
				c = b
			}
			m.noterBloquantDArchetype(ti, c.nom)
		}
	}
	return enJeu
}

// compterRecord compte un record NEW ou DELTA.
func compterRecord(s *FrameArchetypeStat, neuf, ferme bool) {
	switch {
	case neuf && ferme:
		s.Neufs++
		s.NeufsFermes++
	case neuf:
		s.Neufs++
	case ferme:
		s.Deltas++
		s.DeltasFermes++
	default:
		s.Deltas++
	}
}

// noterBloquantDArchetype compte une cause contre un archetype.
func (m *mesureDesTrames) noterBloquantDArchetype(ti int, nom string) {
	if m.bloquantsParTI[ti] == nil {
		m.bloquantsParTI[ti] = map[string]int{}
	}
	m.bloquantsParTI[ti][nom]++
}

// compterBloquant impute un paquet non ferme a sa premiere cause d arret.
func (m *mesureDesTrames) compterBloquant(b bloquant, enJeu int) {
	s, ok := m.rep.Bloquants[b.nom]
	if !ok {
		s = FrameBlockerStat{TI: b.ti, Index: b.index, Composant: b.composant}
	}
	s.Paquets++
	s.UtilesEnJeu += enJeu
	m.rep.Bloquants[b.nom] = s
}

// cleArchetype rend l archetype d un record, ou [ArchetypeNonResolu] pour un delta dont le slot
// n est pas lie et dont l inference a echoue : son `TypeIndex` vaut alors zero PAR DEFAUT, et le
// ranger sous `ti=0` le confondrait avec le moteur de partie. Un tel record n a pas de trace du
// tout (aucun bit de corps lu), la ou un delta lie en a toujours une.
func cleArchetype(r FrameRecord) int {
	if r.Type == recDelta && r.DesyncAt >= 0 && r.Trace.EndBit == 0 && len(r.Trace.Comps) == 0 {
		return ArchetypeNonResolu
	}
	return int(r.TypeIndex)
}

// bloquantDuRecord nomme ce qui a arrete un record desynchronise : son composant sans lecteur
// (`ti=<a> i<idx> <nom>`), un slot non lie, ou un archetype absent du registre.
func (m *mesureDesTrames) bloquantDuRecord(r FrameRecord) (bloquant, bool) {
	if r.DesyncAt < 0 {
		return bloquant{}, false
	}
	ti := cleArchetype(r)
	if ti == ArchetypeNonResolu {
		return causeSimple(causeSlotNonLie), true
	}
	if n := len(r.Trace.Comps); n > 0 && !r.Trace.Comps[n-1].Ported {
		c := r.Trace.Comps[n-1]
		return bloquant{nom: fmt.Sprintf("ti=%d %s", ti, nomComposantBloquant(m.reg, ti, c.Index)),
			ti: ti, index: c.Index, composant: c.Name}, true
	}
	return bloquant{nom: fmt.Sprintf("ti=%d archetype hors registre", ti), ti: ti, index: -1}, true
}

// recordUtile dit si le masque d un record annonce un composant a usage produit. Un record dont
// le masque n a pas ete lu (corps saute par l inference d archetype) n est pas classe utile : on
// ne sait pas ce qu il portait.
func (m *mesureDesTrames) recordUtile(r FrameRecord, ti int) bool {
	if ti < 0 || len(m.utiles) == 0 || m.reg == nil {
		return false
	}
	arch, ok := m.reg.Archetype(ti)
	if !ok {
		return false
	}
	for i, nom := range arch.Components {
		if r.Trace.Mask&(uint64(1)<<(uint(i)&63)) != 0 && m.utiles[CleComposant(ti, nom)] {
			return true
		}
	}
	return false
}
