//go:build research

package grammar

// mouvement_5_19_differentielle_research_test.go — LA DIFFERENTIELLE DU RECORD AVANT LE REJET
// (lot 5.19.1).
//
// Quatre lots (5.15 a 5.18) ont instruit le residu de `bfecd02b` sans le reduire, chacun en
// refutant le suspect du precedent. Ils ont tous cherche le suspect d abord et mesure ensuite.
// Cet instrument fait l inverse : il NE CONCLUT RIEN et ne cherche RIEN — il compare, composant
// par composant, ce que la marche lit juste AVANT un rejet de slot inconnu (23 452 paquets) a ce
// qu elle lit dans les paquets qui FERMENT (2 884). Le composant dont la lecture est fausse est
// celui qui distingue les deux populations ; il est alors lu chez l ecrivain (5.19.2).
//
// Ce qu il publie, en UNE passe :
//
//	(0) le gate du 5.16/5.18 reproduit AU PAQUET, pour que l instrument soit le meme que celui
//	    qui a mesure le residu (fermes a reste NUL, debordements, causes de sortie) ;
//	(a) l archetype du DERNIER record lu avant le rejet ;
//	(b) son dernier composant A CORPS NON VIDE (i25 est un simple bit de masque quand il ne
//	    consomme rien : l exclure par la LARGEUR, pas par le nom — 5.15.1 (i)) ;
//	(c) son masque complet, les trente classes les plus frequentes ;
//	(d) la MEME ventilation sur les paquets qui ferment, plus la table des suspects : presence
//	    par paquet (fautif contre ferme) et LARGEURS lues par composant, avec les largeurs que
//	    seuls les paquets fautifs portent ;
//	(e) les deux records qui PRECEDENT celui du rejet.
//
// Rejouable (un film a la fois) :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestDiff519$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// d519Comp est un composant lu, avec sa position et la LARGEUR reellement consommee.
type d519Comp struct {
	Index   int
	Name    string
	Debut   int
	Largeur int
	Porte   bool
}

// d519Marche est une marche de paquet : les curseurs de [t515Marcher] et les records de la vue B,
// obtenus en UN SEUL decodage (les deux instruments du 5.15 decodaient le paquet deux fois, ce
// qui muait le monde deux fois).
type d519Marche struct {
	m    t515Marche
	recs []FrameRecord
}

// t519Marcher rejoue un paquet rang par rang, comme `decodeFrameParRangs`, et rend les curseurs
// intermediaires AVEC les records de la vue B.
func t519Marcher(pay []byte, w *World, cfg FrameConfig, debut int) d519Marche {
	frameLen := len(pay) * 8
	out := d519Marche{m: t515Marche{Debut: debut}}
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	if debut == DefaultPacketPreambleBits && cfg.PacketPreambleBits >= 1 {
		br.Skip(cfg.PacketPreambleBits - 1)
		a := consumeVueA(br, frameLen)
		out.m.PorteA = a.Porte
		out.m.FinVueA = br.BitPos()
		if !a.Porte {
			out.m.FinVueB, out.m.FinVueC = br.BitPos(), br.BitPos()
			return out
		}
	} else {
		br.Skip(debut)
		out.m.PorteA, out.m.FinVueA = true, br.BitPos()
	}
	w.PoserVueCourante(int(vueDeLImageCle))
	recs, _, hitEnd := decodeInferLoop(br, pay, w, cfg)
	out.recs = recs
	out.m.HitEndB, out.m.FinVueB, out.m.Records = hitEnd, br.BitPos(), len(recs)
	if n := len(recs); n > 0 {
		out.m.DernSlot, out.m.DernTI, out.m.DernType = recs[n-1].Slot, recs[n-1].TypeIndex, recs[n-1].Type
	}
	if !hitEnd {
		out.m.FinVueC = br.BitPos()
		return out
	}
	c := consumeVueC(br, frameLen)
	out.m.PorteC, out.m.KindsC, out.m.FinVueC = c.Porte, c.Kinds, br.BitPos()
	return out
}

// d519Largeurs rend les composants d un record avec leur largeur : la fin d un composant est le
// debut du suivant, celle du dernier est la fin du corps. Aucune largeur n est supposee.
func d519Largeurs(r FrameRecord) []d519Comp {
	out := make([]d519Comp, 0, len(r.Trace.Comps))
	for i, c := range r.Trace.Comps {
		fin := r.Trace.EndBit
		if i+1 < len(r.Trace.Comps) {
			fin = r.Trace.Comps[i+1].StartBit
		}
		out = append(out, d519Comp{Index: c.Index, Name: c.Name, Debut: c.StartBit,
			Largeur: fin - c.StartBit, Porte: c.Ported})
	}
	return out
}

// d519DernierNonVide nomme le dernier composant du record dont le CORPS a consomme des bits.
// Un composant a largeur nulle n est qu un bit de masque leve : il ne peut pas etre la faute.
func d519DernierNonVide(cs []d519Comp) string {
	for i := len(cs) - 1; i >= 0; i-- {
		if cs[i].Largeur > 0 {
			return fmt.Sprintf("i%-2d %s", cs[i].Index, cs[i].Name)
		}
	}
	return "aucun composant a corps non vide"
}

// d519CleRecord nomme un record : type, archetype, dernier composant non vide, nombre de comps.
func d519CleRecord(r FrameRecord) string {
	cs := d519Largeurs(r)
	if len(cs) == 0 {
		return fmt.Sprintf("type %d ti=%2d masque vide", r.Type, r.TypeIndex)
	}
	return fmt.Sprintf("type %d ti=%2d %-42s (%d comps)", r.Type, r.TypeIndex,
		d519DernierNonVide(cs), len(cs))
}

// d519Bilan porte les deux populations : les paquets qui ferment et ceux qui sortent sur un rejet.
type d519Bilan struct {
	paquets, nonLocalise, debordements int
	fermes, rejets, autres             int
	causes                             map[string]int

	// (a) (b) (c) (e) : le dernier record et les deux qui le precedent.
	rejTI, fermeTI             map[int]int
	rejDernier, fermeDernier   map[string]int
	rejMasque, fermeMasque     map[string]int
	rejAvant1, rejAvant2       map[string]int
	rejChaine                  map[string]int
	rejRecords, fermeRecords   int
	rejSansRecord, fermeSansRe int

	// (d) : la table des suspects — presence PAR PAQUET et largeurs lues PAR COMPOSANT.
	presRej, presFerme     map[string]int
	largRej, largFerme     map[string]map[int]int
	tiPresRej, tiPresFerme map[int]int

	// (f) : LA DOSE. Un paquet ferme-t-il moins souvent parce qu il porte PLUS de deltas de
	// bipede ? Si la fermeture suit une loi geometrique en ce nombre, la faute est PAR RECORD
	// et le temoin a isoler est le paquet qui n en porte QU UN.
	doseRej, doseFerme     map[int]int
	unRejMasq, unFermeMasq map[string]int

	// (g) : LA DIFFERENTIELLE INTERNE, celle qui n a AUCUN confondant de densite. Dans le MEME
	// paquet fautif, le DERNIER record est celui dont la lecture laisse le curseur faux ; tous
	// ceux qui le precedent ont ete lus JUSTE (le record suivant s est decode). Comparer les
	// deux, c est comparer deux populations du meme film, de la meme trame, du meme paquet.
	dernMasq, avantMasq map[string]int
	dernLarg, avantLarg map[string]map[int]int
	dernN, avantN       int
}

func d519Nouveau() *d519Bilan {
	return &d519Bilan{
		causes: map[string]int{}, rejTI: map[int]int{}, fermeTI: map[int]int{},
		rejDernier: map[string]int{}, fermeDernier: map[string]int{},
		rejMasque: map[string]int{}, fermeMasque: map[string]int{},
		rejAvant1: map[string]int{}, rejAvant2: map[string]int{}, rejChaine: map[string]int{},
		presRej: map[string]int{}, presFerme: map[string]int{},
		largRej: map[string]map[int]int{}, largFerme: map[string]map[int]int{},
		tiPresRej: map[int]int{}, tiPresFerme: map[int]int{},
		doseRej: map[int]int{}, doseFerme: map[int]int{},
		unRejMasq: map[string]int{}, unFermeMasq: map[string]int{},
		dernMasq: map[string]int{}, avantMasq: map[string]int{},
		dernLarg: map[string]map[int]int{}, avantLarg: map[string]map[int]int{},
	}
}

// d519Largeur enregistre une largeur lue pour un composant dans une des deux populations.
func d519Largeur(m map[string]map[int]int, nom string, largeur int) {
	if m[nom] == nil {
		m[nom] = map[int]int{}
	}
	m[nom][largeur]++
}

// d519Classer ventile UN paquet dans l une des deux populations et cumule tout ce que la
// differentielle compare.
func d519Classer(pay []byte, cfg FrameConfig, mar d519Marche, w *World, b *d519Bilan) {
	b.paquets++
	reste := len(pay)*8 - mar.m.FinVueC
	switch {
	case reste < 0:
		b.debordements++
		b.causes["debordement"]++
		return
	case reste <= m5116GateOctet && c514ResteNul(pay, mar.m.FinVueC):
		b.fermes++
		d519Cumuler(mar.recs, w, b, true)
		return
	}
	sortie := t515SortieVueB(pay, cfg, mar.m)
	if !mar.m.PorteA || !mar.m.HitEndB || !mar.m.PorteC || sortie != "rejet de table de vue" {
		b.autres++
		b.causes["reste hors bourrage · "+sortie]++
		return
	}
	b.rejets++
	d519Cumuler(mar.recs, w, b, false)
}

// d519Cumuler cumule les ventilations d UN paquet dans la population demandee.
func d519Cumuler(recs []FrameRecord, w *World, b *d519Bilan, ferme bool) {
	ti, dern, masq := b.rejTI, b.rejDernier, b.rejMasque
	pres, larg, tiPres := b.presRej, b.largRej, b.tiPresRej
	if ferme {
		ti, dern, masq = b.fermeTI, b.fermeDernier, b.fermeMasque
		pres, larg, tiPres = b.presFerme, b.largFerme, b.tiPresFerme
	}
	// Presence PAR PAQUET : un composant lu dix fois dans un paquet compte pour un.
	vus, vusTI := map[string]bool{}, map[int]bool{}
	bipedes := 0
	var unique FrameRecord
	for _, r := range recs {
		vusTI[int(r.TypeIndex)] = true
		if r.TypeIndex == BipedTypeIndex {
			bipedes++
			unique = r
		}
		for _, c := range d519Largeurs(r) {
			vus[fmt.Sprintf("i%-2d %s", c.Index, c.Name)] = true
		}
	}
	dose, masqUn := b.doseRej, b.unRejMasq
	if ferme {
		dose, masqUn = b.doseFerme, b.unFermeMasq
	}
	dose[bipedes]++
	if bipedes == 1 {
		masqUn[fmt.Sprintf("masque %#x (%d comps) fin %s", unique.Trace.Mask,
			len(unique.Trace.Comps), d519DernierNonVide(d519Largeurs(unique)))]++
	}
	for k := range vus {
		pres[k]++
	}
	for k := range vusTI {
		tiPres[k]++
	}
	if ferme {
		b.fermeRecords += len(recs)
	} else {
		b.rejRecords += len(recs)
	}
	if len(recs) == 0 {
		if ferme {
			b.fermeSansRe++
		} else {
			b.rejSansRecord++
		}
		return
	}
	// Le DERNIER record : archetype, dernier composant non vide, masque, et les largeurs de
	// CHACUN de ses composants — c est lui « le record avant le rejet ».
	last := recs[len(recs)-1]
	cs := d519Largeurs(last)
	ti[int(last.TypeIndex)]++
	dern[d519CleRecord(last)]++
	masq[fmt.Sprintf("ti=%2d masque %#x (%d comps) fin %s",
		last.TypeIndex, last.Trace.Mask, len(cs), d519DernierNonVide(cs))]++
	for _, c := range cs {
		d519Largeur(larg, fmt.Sprintf("i%-2d %s", c.Index, c.Name), c.Largeur)
	}
	if ferme {
		return
	}
	// (g) LA DIFFERENTIELLE INTERNE : le dernier record contre tous ceux qui le precedent DANS
	// LE MEME PAQUET. Ceux-la ont ete lus juste — le record suivant s est decode derriere eux.
	for i, r := range recs {
		masq, larg := b.avantMasq, b.avantLarg
		if i == len(recs)-1 {
			masq, larg = b.dernMasq, b.dernLarg
			b.dernN++
		} else {
			b.avantN++
		}
		cs := d519Largeurs(r)
		masq[fmt.Sprintf("ti=%2d masque %#x (%d comps)", r.TypeIndex, r.Trace.Mask, len(cs))]++
		for _, c := range cs {
			d519Largeur(larg, fmt.Sprintf("ti=%2d i%-2d %s", r.TypeIndex, c.Index, c.Name),
				c.Largeur)
		}
		// LA PROVENANCE DE L ARCHETYPE, et c est la seule chose qui, dans cette boucle, ne
		// vienne pas du flux : `HardBound` distingue une liaison de VERITE TERRAIN (image-cle,
		// table de datums, record NEW propre) d une liaison DEVINEE par l inference de chaine
		// (`BindSoft`). Un archetype devine faux lit tous les composants du slot avec la
		// mauvaise liste — meme largeur portee, mauvaise verite.
		prov := "MOU (inference de chaine)"
		if w.HardBound(r.Slot) {
			prov = "dur (image-cle / datum / NEW)"
		}
		masq[fmt.Sprintf("ti=%2d PROVENANCE %s", r.TypeIndex, prov)]++
	}
	// (e) LES DEUX RECORDS QUI PRECEDENT, et la chaine des trois archetypes.
	chaine := []string{fmt.Sprintf("ti=%d", last.TypeIndex)}
	if n := len(recs); n >= 2 {
		b.rejAvant1[d519CleRecord(recs[n-2])]++
		chaine = append([]string{fmt.Sprintf("ti=%d", recs[n-2].TypeIndex)}, chaine...)
	}
	if n := len(recs); n >= 3 {
		b.rejAvant2[d519CleRecord(recs[n-3])]++
		chaine = append([]string{fmt.Sprintf("ti=%d", recs[n-3].TypeIndex)}, chaine...)
	}
	b.rejChaine[strings.Join(chaine, " -> ")]++
}

// TestDiff519 joue la differentielle sur le film courant.
func TestDiff519(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	b := d519Nouveau()
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
				w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
			}
		}
		if os.Getenv("MOUV516_DATUMS") != "0" {
			LierTableDeDatums(w, data, pks)
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, tc.cfg); debut < 0 {
					b.nonLocalise++
					continue
				}
			}
			d519Classer(pay, tc.cfg, t519Marcher(pay, w, tc.cfg, debut), w, b)
		}
	}
	d519Publier(t, b)
}
