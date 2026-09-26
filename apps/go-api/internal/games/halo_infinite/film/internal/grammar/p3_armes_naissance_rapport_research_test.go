//go:build research

package grammar

// p3_armes_naissance_rapport_research_test.go — le test et les tableaux de la sonde P3 (voir
// l en-tete de `p3_armes_naissance_research_test.go`).

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

// p3Tolerance est l ecart maximal (pas de 100 ms) entre un NEW et le debut de sa vie.
const p3Tolerance = 30

func TestP3ArmesNaissance(t *testing.T) {
	doc := os.Getenv("P3_DOC")
	if doc == "" {
		t.Skip("P3_DOC absent : instrument de recherche")
	}
	tc := t516Cadre(t)
	vies, originMs := p3Vies(t, doc)
	origine := p3Origine(t, tc, originMs)
	p := p3Marcher(tc)
	p3Recensement(t, p)
	p3Rattacher(p, vies, origine)
	marche := func(v *p3Vie) *p3Naissance { return v.naissance }
	p3Couverture(t, "MARCHE vue B", vies, marche)
	p3Accords(t, "MARCHE vue B", vies, marche)
	cat := p3Catalogue(t, doc)
	entree := m511Entree(t)
	r := p3Rechercher(t, tc, vies, origine, cat, tcgCtxDeCarte(entree.Min, entree.Max))
	p3RapportRecherche(t, r)
	trouvee := func(v *p3Vie) *p3Naissance { return v.trouvee }
	p3Couverture(t, "RECORD NEW, lecture de production", vies, trouvee)
	p3Accords(t, "RECORD NEW, lecture de production", vies, trouvee)
	p3Localiser(t, tc, vies, origine)
	p3Chainer(t, tc, vies, origine)
	p3LireParCatalogue(t, tc, vies, origine, cat)
	p3Couverture(t, "RECORD NEW, lecture par catalogue", vies, trouvee)
	p3Accords(t, "RECORD NEW, lecture par catalogue", vies, trouvee)
	p3Detail(t, vies)
}

// p3Origine rend l origine de la frame 0 en horloge de film : premier paquet du chunk 1 +
// `originMs` du document (la formule de `resolveOriginMs`).
func p3Origine(t *testing.T, tc t516Temoin, originMs int64) int64 {
	t.Helper()
	_, pks, ok := tc.fc.ChunkAt(1)
	if !ok || len(pks) == 0 {
		t.Fatalf("chunk 1 absent")
	}
	o := int64(pks[0].TimestampUS) + originMs*1000 //nolint:gosec // horodatage de film
	t.Logf("== P3.1 ORIGINE : premier paquet du chunk 1 %d us + originMs %d = %d us", pks[0].TimestampUS,
		originMs, o)
	return o
}

// p3Recensement publie la marche.
func p3Recensement(t *testing.T, p *p3Passe) {
	t.Helper()
	t.Logf("== P3.0 MARCHE : %d paquets delta, %d non localises ; NEW par ti %v", p.paquets,
		p.nonLocalises, p.news)
	t.Logf("   NEW ti=35 : %d ; desync (index:nombre) %v", len(p.naissances), p.desyncNew35)
	var ann, lus, pres, cadre, crochet int
	parIdx := map[int][2]int{}
	for _, n := range p.naissances {
		ann += n.annoncees
		for _, a := range n.armes {
			lus++
			c := parIdx[a.idx]
			c[0]++
			if a.present {
				pres++
				c[1]++
			}
			parIdx[a.idx] = c
			if a.cadreOK {
				cadre++
			}
			if a.crochet {
				crochet++
			}
		}
	}
	t.Logf("   emplacements d arme : annonces au masque %d, lus %d, presents %d ; moitie basse relue "+
		"== Variant %d/%d ; paire vue par le crochet %d/%d (crochet : %d appels, %d presents)", ann, lus,
		pres, cadre, lus, crochet, lus, p.crochetAppels, p.crochetPresents)
	t.Logf("   par index (lus, presents) : %v", parIdx)
	for _, n := range p.naissances {
		t.Logf("   NEW35 ts=%d c%d slot=%d gen=%d desync=%d annonces=%d lus=%d familles=%08X", n.ts, n.chunk,
			n.slot, n.gen, n.desync, n.annoncees, len(n.armes), n.familles())
	}
}

// p3Rattacher attache a chaque vie le NEW ti=35 du meme slot le plus proche de son debut.
func p3Rattacher(p *p3Passe, vies []*p3Vie, origine int64) {
	for i := range p.naissances {
		n := &p.naissances[i]
		n.trame = int((int64(n.ts) - origine) / tcgPasUS) //nolint:gosec // horodatage de film
	}
	for _, v := range vies {
		best := -1
		for i := range p.naissances {
			n := &p.naissances[i]
			if n.slot != v.slot || n.rattachee {
				continue
			}
			d := n.trame - v.start
			if d < -p3Tolerance || d > p3Tolerance && v.debutConnu || !v.debutConnu && n.trame > p3Tolerance {
				continue
			}
			if best < 0 || p3Abs(d) < p3Abs(p.naissances[best].trame-v.start) {
				best = i
			}
		}
		if best >= 0 {
			p.naissances[best].rattachee = true
			v.naissance = &p.naissances[best]
		}
	}
}

func p3Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// p3Couverture publie le taux de naissances lues.
func p3Couverture(t *testing.T, nom string, vies []*p3Vie, sel func(*p3Vie) *p3Naissance) {
	t.Helper()
	type cpt struct{ vies, sansNew, desyncAvant, sansArme, lues int }
	var tout, connues cpt
	for _, v := range vies {
		for _, c := range []*cpt{&tout, &connues} {
			if c == &connues && !v.debutConnu {
				continue
			}
			c.vies++
			switch n := sel(v); {
			case n == nil:
				c.sansNew++
			case len(n.armes) < n.annoncees:
				c.desyncAvant++
			case len(n.familles()) == 0:
				c.sansArme++
			default:
				c.lues++
			}
		}
	}
	for _, x := range []struct {
		nom string
		c   cpt
	}{{"toutes les vies", tout}, {"vies nees apres l origine", connues}} {
		t.Logf("== P3.2 %s COUVERTURE (%s) : %d vies ; LUES %d (%.1f %%) ; sans NEW rattache %d ; NEW "+
			"desynchronise avant ses armes %d ; NEW sans arme presente %d", nom, x.nom, x.c.vies, x.c.lues,
			tcgPct(x.c.lues, x.c.vies), x.c.sansNew, x.c.desyncAvant, x.c.sansArme)
	}
}

// p3Egal compare deux listes.
func p3Egal(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func p3Contient(a []uint32, x uint32) bool {
	for _, v := range a {
		if v == x {
			return true
		}
	}
	return false
}

// p3Accords publie les accords avec O1 et O2, le temoin (vie suivante) et le hasard (toutes paires).
func p3Accords(t *testing.T, nom string, vies []*p3Vie, sel func(*p3Vie) *p3Naissance) {
	t.Helper()
	var lues []*p3Vie
	for _, v := range vies {
		if n := sel(v); n != nil && len(n.familles()) > 0 {
			lues = append(lues, v)
		}
	}
	sort.SliceStable(lues, func(i, j int) bool { return lues[i].start < lues[j].start })
	o1 := func(b *p3Naissance, v *p3Vie) (bool, bool) {
		return v.aO1, v.aO1 && p3Egal(b.familles(), v.o1)
	}
	o1Ordre := func(b *p3Naissance, v *p3Vie) (bool, bool) {
		return v.aO1, v.aO1 && p3Egal(b.ordonnees(), v.o1Ordre)
	}
	o2 := func(b *p3Naissance, v *p3Vie) (bool, bool) {
		return v.aO2, v.aO2 && p3Contient(b.familles(), v.o2)
	}
	for _, o := range []struct {
		nom string
		f   func(*p3Naissance, *p3Vie) (bool, bool)
	}{{"O1 image-cle (ensemble)", o1}, {"O1 image-cle (ordre des emplacements)", o1Ordre},
		{"O2 premier tir", o2}} {
		var n, ok, tn, tok, hn, hok int
		for i, v := range lues {
			if a, b := o.f(sel(v), v); a {
				n++
				if b {
					ok++
				}
			}
			for k := 1; k < len(lues); k++ { // temoin : la vie suivante d un AUTRE slot
				w := lues[(i+k)%len(lues)]
				if w.slot == v.slot {
					continue
				}
				if a, b := o.f(sel(v), w); a {
					tn++
					if b {
						tok++
					}
				}
				break
			}
			for _, w := range lues {
				if w.slot == v.slot {
					continue
				}
				if a, b := o.f(sel(v), w); a {
					hn++
					if b {
						hok++
					}
				}
			}
		}
		t.Logf("== P3.3 %s %-40s : accord %d/%d (%.1f %%) ; TEMOIN vie suivante %d/%d (%.1f %%) ; HASARD "+
			"toutes paires %d/%d (%.1f %%)", nom, o.nom, ok, n, tcgPct(ok, n), tok, tn, tcgPct(tok, tn), hok, hn,
			tcgPct(hok, hn))
	}
}

// p3Detail publie chaque vie : naissance lue, oracles, desaccords.
func p3Detail(t *testing.T, vies []*p3Vie) {
	t.Helper()
	fam := func(v []uint32) string {
		s := ""
		for _, x := range v {
			s += fmt.Sprintf(" %08X", x)
		}
		return "[" + s + " ]"
	}
	t.Logf("== P3.4 DETAIL PAR VIE (slot debut..fin | marche | record trouve : trame chunk bit gen desync " +
		"annonces/lus emplacements | O1 | O2)")
	for _, v := range vies {
		s := fmt.Sprintf("slot %d %d..%d (connu %v, prises a la naissance %d)", v.slot, v.start, v.end,
			v.debutConnu, v.prisesNaissance)
		if v.naissance != nil {
			s += " | MARCHE oui"
		}
		if n := v.trouvee; n != nil {
			var idx string
			for _, a := range n.armes {
				idx += fmt.Sprintf(" i%d:%v:%08X/%08X", a.idx, a.present, a.hi, a.lo)
			}
			s += fmt.Sprintf(" | TROUVE t=%d c%d bit %d (fin vue B %d, close %v) g%d desync=%d %d/%d%s",
				n.trame, n.chunk, n.bit, n.finMarche, n.hitEndB, n.gen, n.desync, n.annoncees,
				len(n.armes), idx)
		} else {
			s += " | RIEN TROUVE"
		}
		marque := ""
		if n := v.trouvee; n != nil && len(n.familles()) > 0 {
			if v.aO1 && !p3Egal(n.familles(), v.o1) {
				marque += " DESACCORD-O1"
			}
			if v.aO2 && !p3Contient(n.familles(), v.o2) {
				marque += " DESACCORD-O2"
			}
		}
		if v.aO1 {
			s += fmt.Sprintf(" | O1 t=%d %s", v.o1T, fam(v.o1Ordre))
		}
		if v.aO2 {
			s += fmt.Sprintf(" | O2 t=%d %08X", v.o2T, v.o2)
		}
		t.Logf("   %s%s", s, marque)
	}
}
