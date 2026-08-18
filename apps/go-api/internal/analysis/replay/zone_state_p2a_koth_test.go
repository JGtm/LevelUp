package replay

// zone_state_p2a_koth_test.go — CB.2a.3 : LA COLLINE ACTIVE, APPARIEE PAR LA GRAPPE.
//
// CE QUE LA PHASE 1 A ETABLI, ET CE QUI MANQUAIT. En KOTH, une seule zone ti=13 rampe a la fois,
// 100,0 % du temps sur 60 rampes (`01e1f945`) — l'unicite est acquise. Mais QUELLE zone : le
// catalogue de formes ne connait AUCUN role de colline (mesure du 2026-08-18 : `strongholds_zone`,
// `extraction_zone`, `flag_*`, `oddball_spawn`, rien d'autre). La colline se pose donc sur des
// emplacements que le fichier de carte declare sous d'autres roles, et le seul temoin disponible
// est la GRAPPE DES POSITIONS : la ou les joueurs s'agglutinent pendant qu'une jauge monte.
//
// LA PERIODE : une par RAMPE, fondue avec ses voisines quand la grappe designe la meme zone, et
// etendue jusqu'a la garde suivante. La premiere definition essayee segmentait par SLOT ACTIF ;
// elle a ete refutee par la mesure (un seul slot porte la jauge de tout un match KOTH) — le
// detail, et la raison de garder la trace de l'essai, sont dans `p2aPeriodesParRampe`.
//
// LE TEMOIN est le meme que celui du croisement de zones : les memes positions contre des zones
// TRANSLATEES de 12 m. Si la grappe designait une zone par simple densite de circulation, le
// temoin la designerait aussi bien.

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// p2aPeriode est un intervalle pendant lequel UN slot ti=13 est le seul a monter.
type p2aPeriode struct {
	slot     uint32
	t0, t1   int
	zone     int
	inst     int32
	dedans   int
	deuxieme int
	echants  int
	temoin   int
	hasZone  bool
}

// p2aVoletKOTH mesure CB.2a.3.
func p2aVoletKOTH(t *testing.T, sb *strings.Builder, e p2aEntree) {
	t.Helper()
	t.Logf("")
	t.Logf("=== CB.2a.3 KOTH — zone active appariee par la grappe des positions")
	if len(e.zones) == 0 {
		t.Logf("  aucune zone au catalogue : volet CB.2a.3 NON MESURABLE sur cette carte")
		fmt.Fprintf(sb, "# a3 %s : non mesurable (aucune zone au catalogue)\n", e.short)
		return
	}
	ramps := p2aRampesParDepart(e.sc)
	if len(ramps) == 0 {
		t.Logf("  aucune rampe du tag 3 : volet CB.2a.3 SANS OBJET sur ce film")
		fmt.Fprintf(sb, "# a3 %s : sans objet (aucune rampe)\n", e.short)
		return
	}
	brutes := p2aPeriodesParRampe(ramps)
	pts := p2aPointsParFrame(e.doc)
	loin := TranslateZones(e.zones, mapvar.Vec3{X: p2aTemoinTranslationM, Y: p2aTemoinTranslationM})
	for i := range brutes {
		p2aAttribuePeriode(&brutes[i], e, pts, loin)
	}
	t.Logf("  %d rampes -> %d periodes brutes, attribuees par la grappe puis fusionnees",
		len(ramps), len(brutes))
	p2aLogPeriodes(t, sb, e, p2aFusionne(brutes, e.sc.t1MS))
}

// p2aRampesParDepart rend les rampes triees par instant de DEPART (la segmentation en periodes
// se fait sur le depart, pas sur le sommet).
func p2aRampesParDepart(sc *p2aScan) []p2aRamp {
	out := p2aRampes(sc)
	sort.SliceStable(out, func(i, j int) bool { return out[i].t0 < out[j].t0 })
	return out
}

// p2aPeriodesParRampe rend une periode brute par rampe : son support temporel.
//
// DEUX DEFINITIONS ONT ETE ESSAYEES, ET C'EST LA MESURE QUI A TRANCHE — il faut le dire, parce
// que la seconde n'a pas ete choisie pour arranger un chiffre :
//
//	(1) SEGMENTER PAR SLOT ACTIF. Elle supposait qu'une colline = un slot ti=13, comme une zone
//	    de Strongholds = un slot. REFUTEE sur `01e1f945` : UN SEUL slot (1474) porte la jauge de
//	    tout le match KOTH, donc cette lecture rend UNE periode et n'apparie rien.
//	(2) SEGMENTER PAR RAMPE. Chaque montee de la jauge est une session de garde ; la zone se lit
//	    dans la grappe des positions PENDANT la montee, et les rampes voisines qui designent la
//	    MEME zone se fondent en une periode de colline.
//
// Le seuil de couverture (80 %) et la methode d'attribution (la grappe, jugee contre des zones
// translatees) sont ceux de l'arbitrage, inchanges.
func p2aPeriodesParRampe(ramps []p2aRamp) []p2aPeriode {
	out := make([]p2aPeriode, 0, len(ramps))
	for _, r := range ramps {
		out = append(out, p2aPeriode{slot: r.slot, t0: r.t0, t1: r.tMax})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].t0 < out[j].t0 })
	return out
}

// p2aFusionne fond les periodes voisines qui designent la MEME zone et etend chacune jusqu'au
// debut de la suivante : entre deux gardes de la meme colline, la colline n'a pas change.
//
// Les periodes NON attribuees (aucune zone ne ressort) sont conservees telles quelles : elles
// comptent dans le temps couvert mais PAS dans le temps attribue, ce qui est exactement ce que
// le seuil de 80 % doit trancher.
func p2aFusionne(ps []p2aPeriode, finMS int) []p2aPeriode {
	if len(ps) == 0 {
		return nil
	}
	var out []p2aPeriode
	cur := ps[0]
	for _, p := range ps[1:] {
		if p.hasZone && cur.hasZone && p.zone == cur.zone {
			cur.t1, cur.dedans, cur.echants = p.t1, cur.dedans+p.dedans, cur.echants+p.echants
			cur.temoin, cur.deuxieme = cur.temoin+p.temoin, cur.deuxieme+p.deuxieme
			continue
		}
		if p.t0 > cur.t1 {
			cur.t1 = p.t0 // la colline reste la meme jusqu'a la garde suivante
		}
		out = append(out, cur)
		cur = p
	}
	if finMS > cur.t1 {
		cur.t1 = finMS
	}
	return append(out, cur)
}

// p2aPointsParFrame indexe toutes les positions publiees par frame — la grappe se lit par
// tranche de temps, pas par joueur.
func p2aPointsParFrame(doc ReplayDocument) map[int][]Point {
	out := map[int][]Point{}
	for _, tr := range doc.Tracks {
		for _, p := range tr.Points {
			out[p.T] = append(out[p.T], p)
		}
	}
	return out
}

// p2aAttribuePeriode compte, pour une periode, les positions tombant dans chaque zone, et retient
// la plus peuplee. `deuxieme` est publie a cote : sans lui, « la zone gagnante » ne dit pas si
// elle gagne d'une tete ou d'une longueur.
func p2aAttribuePeriode(p *p2aPeriode, e p2aEntree, pts map[int][]Point, loin []Zone) {
	f0, ok0 := p2aFrameOf(e.doc, p.t0)
	f1, ok1 := p2aFrameOf(e.doc, p.t1)
	if !ok0 {
		f0 = 0
	}
	if !ok1 {
		f1 = e.doc.FrameCount - 1
	}
	parZone, parTemoin := map[int]int{}, map[int]int{}
	for f := f0; f <= f1; f++ {
		for _, pt := range pts[f] {
			p.echants++
			if r, ok := p2aZoneDe(e.zones, pt); ok {
				parZone[r]++
			}
			if r, ok := p2aZoneDe(loin, pt); ok {
				parTemoin[r]++
			}
		}
	}
	for r, n := range parZone {
		switch {
		case n > p.dedans:
			p.deuxieme, p.dedans, p.zone, p.hasZone = p.dedans, n, r, true
		case n > p.deuxieme:
			p.deuxieme = n
		}
	}
	// Le temoin est la MEME mesure sur les zones translatees : sa zone la plus peuplee. Compter
	// les positions tombant dans n'importe quelle zone translatee comparerait un maximum a une
	// somme, et le temoin paraitrait au niveau du signal sans l'etre.
	for _, n := range parTemoin {
		if n > p.temoin {
			p.temoin = n
		}
	}
	if p.hasZone {
		for _, z := range e.zones {
			if z.SpatialRank == p.zone {
				p.inst = z.InstanceID
			}
		}
	}
}

// p2aZoneDe rend le rang spatial de la zone qui contient le point a la tolerance du verdict.
// Meme regle d'ambiguite qu'`AttributeZones` : deux zones ex aequo ne tranchent pas.
func p2aZoneDe(zones []Zone, pt Point) (int, bool) {
	best, hits := nearestZones(zones, pt)
	if len(hits) != 1 || best > p2aVerdictDistanceM {
		return 0, false
	}
	return hits[0].SpatialRank, true
}

// p2aLogPeriodes publie les periodes, leur couverture et le verdict.
func p2aLogPeriodes(t *testing.T, sb *strings.Builder, e p2aEntree, ps []p2aPeriode) {
	t.Helper()
	dureeMS := e.sc.t1MS - e.sc.t0MS
	var couvert, couvertAttribue, echants, dedans, temoin int
	for _, p := range ps {
		d := p.t1 - p.t0
		couvert += d
		echants, dedans, temoin = echants+p.echants, dedans+p.dedans, temoin+p.temoin
		if p.hasZone {
			couvertAttribue += d
		}
		t.Logf("  periode slot %-5d [%7d ; %7d] %6d ms -> zone %s · %d/%d positions dedans"+
			" (2e zone %d) · temoin translate %d", p.slot, p.t0, p.t1, d, p2aZoneLbl(p),
			p.dedans, p.echants, p.deuxieme, p.temoin)
		fmt.Fprintf(sb, "a3_periode\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%v\n", e.short, p.slot,
			p.t0, p.t1, p.zone, p.inst, p.dedans, p.deuxieme, p.echants, p.hasZone)
	}
	nettes, nettesMS := p2aNettes(ps)
	part := p2aRatio(float64(couvertAttribue), float64(dureeMS))
	v := "NON TENU"
	if part >= p2aSeuilCouvertureKOTH {
		v = "TENU"
	}
	// LA COUVERTURE SEULE EST UNE CLAUSE FAIBLE, et il faut l'ecrire : des que des rampes sont
	// reparties sur le match, les periodes etendues couvrent presque tout. Ce qui porte
	// reellement le resultat, c'est la NETTETE de chaque attribution — la zone retenue doit
	// devancer nettement la deuxieme ET le temoin translate. Publiee a cote du verdict, jamais
	// a sa place : le seuil ecrit dans l'arbitrage n'est pas rehausse apres coup.
	t.Logf("  NETTETE : %d/%d periodes ou la zone retenue devance la 2e d'un facteur 2 ET bat le"+
		" temoin translate — %d ms sur %d ms attribues (%.1f %%)", nettes, len(ps), nettesMS,
		couvertAttribue, 100*p2aRatio(float64(nettesMS), float64(couvertAttribue)))
	fmt.Fprintf(sb, "a3_nettete\t%s\t%d\t%d\t%d\t%d\n", e.short, nettes, len(ps), nettesMS,
		couvertAttribue)
	t.Logf("  %d periodes · temps couvert %d ms dont %d ms ATTRIBUES a une zone = %.1f %% du film"+
		" (%d ms) — seuil %.0f %% : %s", len(ps), couvert, couvertAttribue, 100*part, dureeMS,
		100*p2aSeuilCouvertureKOTH, v)
	t.Logf("  GRAPPE : %d/%d positions dans la zone retenue (%.1f %%) · TEMOIN zones a %.0f m :"+
		" %d (%.1f %%)", dedans, echants, 100*p2aRate(dedans, echants), p2aTemoinTranslationM,
		temoin, 100*p2aRate(temoin, echants))
	fmt.Fprintf(sb, "a3_verdict\t%s\t%d\t%d\t%d\t%d\t%.4f\t%d\t%d\t%d\t%s\n", e.short, len(ps),
		couvert, couvertAttribue, dureeMS, part, dedans, echants, temoin, v)
}

// p2aNettes compte les periodes dont l'attribution est NETTE, et le temps qu'elles couvrent.
func p2aNettes(ps []p2aPeriode) (int, int) {
	n, ms := 0, 0
	for _, p := range ps {
		if p.hasZone && p.dedans > 2*p.deuxieme && p.dedans > p.temoin {
			n++
			ms += p.t1 - p.t0
		}
	}
	return n, ms
}

func p2aZoneLbl(p p2aPeriode) string {
	if !p.hasZone {
		return "AUCUNE"
	}
	return fmt.Sprintf("rang %d (InstanceID %d)", p.zone, p.inst)
}
