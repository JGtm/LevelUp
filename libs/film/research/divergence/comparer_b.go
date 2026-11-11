//go:build research

package divergence

// comparer_b.go : la vue B (les records d entites), record par record puis composant par composant.
// Voir comparer.go pour la definition du lecteur en cause.

import (
	"fmt"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// nomDuGenre rend le genre du record dans les mots du temoin (create, update, delete).
func nomDuGenre(g lecture.Genre) string {
	switch g {
	case lecture.GenreNeuf:
		return "create"
	case lecture.GenreDelta:
		return "update"
	case lecture.GenreSuppression:
		return "delete"
	}
	return fmt.Sprintf("genre%d", g)
}

// longueurEnTete : bits d un en-tete jusqu a la fin de l eid (operation, index, generation).
func longueurEnTete(op string, idLowBits int) int {
	if op == "update" {
		return 1 + idLowBits + 2
	}
	return 3 + idLowBits + 2
}

func luRecordNous(r *lecture.Record) string {
	return fmt.Sprintf("%s slot=%d gen=%d ti=%d liaison=%d", nomDuGenre(r.Genre), r.Vie.Slot, r.Vie.Gen, r.TI, r.Liaison)
}

func luOpTemoin(o *opTemoin) string {
	return fmt.Sprintf("%s index=%d gen=%d type=%s", o.op, o.index, o.gen, o.typeEntite)
}

// typeDuRecord : le type d entite pour nommer un lecteur (celui du temoin quand il le donne).
func typeDuRecord(r *lecture.Record, o *opTemoin) string {
	if o != nil && o.typeEntite != "" {
		return o.typeEntite
	}
	if r != nil && r.TI >= 0 {
		return fmt.Sprint(r.TI)
	}
	return "?"
}

// lecteurApres nomme le lecteur de la LARGEUR du record i-1 (le dernier composant lu, ou son etat de
// creation), ou le debut de la vue quand i vaut 0.
func lecteurApres(p *lecture.Paquet, ops []opTemoin, i int) string {
	if i == 0 {
		return "B : debut de la vue (premier record)"
	}
	r := &p.Records[i-1]
	var o *opTemoin
	if i-1 < len(ops) {
		o = &ops[i-1]
	}
	if r.Comps[1] > r.Comps[0] {
		c := p.Comps[r.Comps[1]-1]
		return fmt.Sprintf("B : ti=%s composant %d (%s)", typeDuRecord(r, o), c.Index, nomDuGenre(r.Genre))
	}
	return fmt.Sprintf("B : ti=%s etat de %s", typeDuRecord(r, o), nomDuGenre(r.Genre))
}

// comparerVueB : les records dans l ordre, puis la sortie de la vue.
// comparerVueB confronte les records a partir du record `depuis`.
func comparerVueB(n paquetNous, t *paquetTemoin, cx *contexteFilm, depuis int) desaccord {
	p := n.p
	recs, ops := p.Records, t.ops
	m := min(len(recs), len(ops))
	for i := depuis; i < m; i++ {
		if d := comparerRecord(p, t, i); !d.aucun() {
			d.record = i
			return d
		}
	}
	switch {
	case len(recs) < len(ops):
		return nousSArretons(p, t, cx, len(recs))
	case len(recs) > len(ops):
		return leTemoinSArrete(p, t, len(ops))
	}
	return sortieDeLaVueB(p, t, cx)
}

// comparerRecord : l en-tete du record i (debut, genre, eid, type), puis ses composants.
func comparerRecord(p *lecture.Paquet, t *paquetTemoin, i int) desaccord {
	r, o := &p.Records[i], &t.ops[i]
	lu := func() (string, string) { return luRecordNous(r), luOpTemoin(o) }
	if int(r.Debut) != o.opBit {
		a, b := lu()
		return desaccord{vue: "B", lecteur: lecteurApres(p, t.ops, i), bitNous: int(r.Debut), bitTemoin: o.opBit, luNous: a, luTemoin: b}
	}
	if nomDuGenre(r.Genre) != o.op || int(r.Vie.Slot) != o.index || int(r.Vie.Gen) != o.gen {
		a, b := lu()
		return desaccord{vue: "B", lecteur: fmt.Sprintf("B : en-tete du record (%s)", o.op), bitNous: int(r.Debut), bitTemoin: o.opBit, luNous: a, luTemoin: b}
	}
	if o.op != "delete" && o.typeEntite != "" && o.typeEntite != "-" && r.TI >= 0 && fmt.Sprint(r.TI) != o.typeEntite {
		a, b := lu()
		return desaccord{vue: "B", lecteur: fmt.Sprintf("B : type de l'entite (%s)", o.op), bitNous: int(r.Debut), bitTemoin: o.opBit, luNous: a, luTemoin: b}
	}
	return comparerComposants(p, r, o)
}

// comparerComposants confronte la liste des composants du record et le bit de debut de chacun.
func comparerComposants(p *lecture.Paquet, r *lecture.Record, o *opTemoin) desaccord {
	if o.op == "delete" || o.initAbsent || len(o.compo) == 0 {
		return desaccord{}
	}
	cs := p.Comps[r.Comps[0]:r.Comps[1]]
	ti := typeDuRecord(r, o)
	liste := func() string { return fmt.Sprintf("B : ti=%s liste de composants (%s)", ti, o.op) }
	for k := 0; k < min(len(cs), len(o.compo)); k++ {
		if int(cs[k].Index) != o.compo[k] {
			return desaccord{vue: "B", lecteur: liste(), bitNous: int(cs[k].Debut), bitTemoin: o.blocBit[k],
				luNous: fmt.Sprintf("composant %d au bit %d", cs[k].Index, cs[k].Debut), luTemoin: fmt.Sprintf("composant %d au bit %d", o.compo[k], o.blocBit[k])}
		}
		if o.blocBit[k] >= 0 && int(cs[k].Debut) != o.blocBit[k] {
			lect := liste()
			if k > 0 {
				lect = fmt.Sprintf("B : ti=%s composant %d (%s)", ti, o.compo[k-1], o.op)
			}
			return desaccord{vue: "B", lecteur: lect, decalage: true, bitNous: int(cs[k].Debut), bitTemoin: o.blocBit[k],
				luNous: fmt.Sprintf("composant %d au bit %d", cs[k].Index, cs[k].Debut), luTemoin: fmt.Sprintf("composant %d au bit %d", o.compo[k], o.blocBit[k])}
		}
	}
	if len(cs) < len(o.compo) {
		k := len(cs)
		lect := liste()
		if k > 0 {
			c := cs[k-1]
			lect = fmt.Sprintf("B : ti=%s composant %d : lecture arretee (%s, etat %d)", ti, c.Index, o.op, c.Etat)
		}
		return desaccord{vue: "B", lecteur: lect, bitNous: sansBit, bitTemoin: o.blocBit[k],
			luNous:   fmt.Sprintf("%d composants lus, desync=%d", len(cs), r.Desync),
			luTemoin: fmt.Sprintf("composant %d au bit %d", o.compo[k], o.blocBit[k])}
	}
	if len(cs) > len(o.compo) {
		return desaccord{vue: "B", lecteur: liste(), bitNous: int(cs[len(o.compo)].Debut), bitTemoin: sansBit,
			luNous: fmt.Sprintf("%d composants", len(cs)), luTemoin: fmt.Sprintf("%d composants", len(o.compo))}
	}
	return desaccord{}
}

// sourceDuType dit d ou le temoin tient le type d une entite non creee dans ce paquet.
func sourceDuType(t *paquetTemoin, cx *contexteFilm, i int) string {
	o := &t.ops[i]
	cle := [2]int{o.index, o.gen}
	for k := 0; k < i; k++ {
		if t.ops[k].op == "create" && t.ops[k].index == o.index && t.ops[k].gen == o.gen {
			return "creee plus haut dans ce paquet"
		}
	}
	if c, ok := cx.crees[cle]; ok {
		return fmt.Sprintf("creee dans le film (chunk %d paquet %d)", c[0], c[1])
	}
	return "jamais creee dans le film vu : instantane du chunk"
}

// nousSArretons : nos records s arretent avant ceux du temoin. Le temoin lit encore l op `i`.
func nousSArretons(p *lecture.Paquet, t *paquetTemoin, cx *contexteFilm, i int) desaccord {
	o := &t.ops[i]
	fin := int(p.VueB.Debut + p.VueB.Bits)
	lt := luOpTemoin(o)
	if o.op == "update" {
		lt += fmt.Sprintf(" (type tenu de : %s)", sourceDuType(t, cx, i))
	}
	switch p.VueB.Sortie {
	case lecture.SortieRejetHorsDatum, lecture.SortieRejetDeVue:
		nom := "hors datum"
		if p.VueB.Sortie == lecture.SortieRejetDeVue {
			nom = "de vue"
		}
		attendu := o.opBit + longueurEnTete(o.op, cx.idLowBits)
		ln := fmt.Sprintf("rejet %s : eid=%#x (slot %d, gen %d) ; %s", nom, p.VueB.EIDRejete, p.VueB.EIDRejete&0x3fffffff, p.VueB.EIDRejete>>30, notreTable(p, p.VueB.EIDRejete&0x3fffffff))
		if fin != attendu {
			return desaccord{vue: "B", lecteur: lecteurApres(p, t.ops, i), bitNous: fin - longueurEnTete(o.op, cx.idLowBits),
				bitTemoin: o.opBit, luNous: ln + " (en-tete a un autre bit)", luTemoin: lt}
		}
		return desaccord{vue: "B", lecteur: "B : sortie par rejet (" + nom + ")", bitNous: o.opBit, bitTemoin: o.opBit, luNous: ln, luTemoin: lt}
	}
	lect := lecteurApres(p, t.ops, i)
	if n := len(p.Records); n > 0 && p.Records[n-1].Desync >= 0 {
		lect = fmt.Sprintf("B : ti=%s composant %d non lu (desync, %s)", typeDuRecord(&p.Records[n-1], &t.ops[n-1]), p.Records[n-1].Desync, nomDuGenre(p.Records[n-1].Genre))
	}
	return desaccord{vue: "B", lecteur: lect + " / sortie " + fmt.Sprint(p.VueB.Sortie), bitNous: fin, bitTemoin: o.opBit,
		luNous: fmt.Sprintf("vue B arretee au bit %d (sortie %d)", fin, p.VueB.Sortie), luTemoin: lt}
}

// leTemoinSArrete : le temoin n a pas d op `i` (arret ou reprise de sa marche), nous en avons un.
func leTemoinSArrete(p *lecture.Paquet, t *paquetTemoin, i int) desaccord {
	r := &p.Records[i]
	bt, lt := t.end, "arret "+t.stop
	if t.repriseDepart >= 0 {
		bt, lt = t.repriseDepart, "reprise "+t.repriseCause
	}
	return desaccord{vue: "B", lecteur: "B : le temoin s'arrete ou reprend (" + lt + ")", bitNous: int(r.Debut), bitTemoin: bt,
		luNous: luRecordNous(r), luTemoin: lt}
}

// sortieDeLaVueB : tous les records coincident ; la sortie de la vue B aussi ?
func sortieDeLaVueB(p *lecture.Paquet, t *paquetTemoin, cx *contexteFilm) desaccord {
	fin := int(p.VueB.Debut + p.VueB.Bits)
	i := len(p.Records)
	switch {
	case t.stop != "aucun":
		return desaccord{}
	case p.VueB.Sortie == lecture.SortieTerminateur && fin == t.end:
		return desaccord{}
	case p.VueB.Sortie == lecture.SortieTerminateur:
		return desaccord{vue: "B", lecteur: lecteurApres(p, t.ops, i), bitNous: fin - 3, bitTemoin: t.end - 3,
			luNous: "terminateur de la vue B", luTemoin: "terminateur de la vue B"}
	case p.VueB.Sortie == lecture.SortieRejetHorsDatum || p.VueB.Sortie == lecture.SortieRejetDeVue:
		// Notre en-tete rejete finit a `fin` ; le temoin lit ici le terminateur (3 bits). L en-tete d une mise a jour
		// tient 1 + idLowBits + 2 bits : debut suppose. Les deux debuts different : c est la largeur du record precedent.
		return desaccord{vue: "B", lecteur: lecteurApres(p, t.ops, i) + " / nous rejetons un en-tete ou le temoin lit le terminateur", bitNous: fin - longueurEnTete("update", cx.idLowBits), bitTemoin: t.end - 3,
			luNous: fmt.Sprintf("rejet eid=%#x", p.VueB.EIDRejete), luTemoin: "terminateur de la vue B"}
	}
	return desaccord{vue: "B", lecteur: lecteurApres(p, t.ops, i) + " / sortie " + fmt.Sprint(p.VueB.Sortie), bitNous: fin, bitTemoin: t.end,
		luNous: fmt.Sprintf("sortie %d", p.VueB.Sortie), luTemoin: "terminateur de la vue B"}
}

// notreTable dit ce que NOTRE table d entites sait du slot rejete (type, vue, provenance de la liaison).
func notreTable(p *lecture.Paquet, slot uint32) string {
	if p.Entites == nil {
		return "table d'entites absente"
	}
	e, ok := p.Entites.Entite(slot)
	if !ok {
		return "slot absent de notre table"
	}
	return fmt.Sprintf("notre table : ti=%d vue=%d liaison=%d generation_connue=%t", e.TI, e.Vue, e.Liaison, e.GenerationConnue)
}
