package replay

// zone_state_p2a_tag4_test.go — CB.2a.2 : LA SEMANTIQUE DU TAG 4.
//
// CE QUE LA PHASE 1 A LAISSE. Le tag 4 est le SEUL canal enumerable du corpus (100 % des slots
// juges portent au plus 8 valeurs distinctes, 5 films, 4 modes). Mais sa clause temporelle etait
// VIDE PAR CONSTRUCTION : avec 135 a 138 changements sur dix minutes, une fenetre de +/- 2 s
// couvre 93 a 97 % du match, et « 100 % des captures couvertes » ne dit alors rien.
//
// LA METRIQUE ADAPTEE A UN CANAL BAVARD, et pourquoi elle a deux faces :
//
//	RAPPEL     part des captures DE LA ZONE d'un slot suivies d'un changement de tag 4 de CE
//	           slot dans la fenetre. C'est la face que la phase 1 mesurait — elle sature.
//	PRECISION  part des changements de tag 4 d'un slot qui tombent dans la fenetre d'une capture
//	           de SA zone. C'est la face qui manquait : un canal qui change tout le temps a une
//	           precision faible, quel que soit son rappel.
//	HASARD     la couverture temporelle des fenetres, mesuree film par film. Chaque taux est lu
//	           CONTRE lui, jamais dans l'absolu.
//
// PUIS LA VALEUR, qui est la vraie question produit : le tag 4 designe-t-il le PROPRIETAIRE ?
// L'equipe du capteur vient du ROSTER (xuid -> team_id, corpus gele) et jamais du film — la
// phase 1 a etabli que `game-engine-team-mapping` lit ses bits sans les publier.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// p2aTag4Slot porte la mesure d'UN slot : ses changements, les captures de sa zone, et les
// quatre nombres qui en sortent.
type p2aTag4Slot struct {
	slot uint32
	zone int
	// source dit d'ou vient la zone : « tag3 » (la carte de CB.2a.1, sans circularite) ou
	// « vote » (les captures proches des changements — partiellement circulaire).
	source      string
	changements []p2aEch
	// captures de SA zone ; toutes = toutes les captures attribuees du film (denominateur
	// NON circulaire, publie a cote).
	captures  []p2aCapture
	toutes    []p2aCapture
	precisNum int
	rappelNum int
}

// p2aVoletTag4 mesure CB.2a.2 : precision / rappel / hasard, puis la valeur contre l'equipe.
func p2aVoletTag4(t *testing.T, sb *strings.Builder, e p2aEntree, app p2aAppariement) {
	t.Helper()
	t.Logf("")
	t.Logf("=== CB.2a.2 SEMANTIQUE DU TAG 4")
	if len(app.zoneParSlot) == 0 {
		t.Logf("  aucune carte slot -> zone : volet CB.2a.2 SANS OBJET sur ce film")
		fmt.Fprintf(sb, "# a2 %s : sans objet (aucune carte slot -> zone)\n", e.short)
		return
	}
	dureeMS := e.sc.t1MS - e.sc.t0MS
	p2aTag4Inventaire(t, sb, e, app)
	slots := p2aTag4Slots(e, app)
	if len(slots) == 0 {
		t.Logf("  aucun slot ne porte a la fois un tag 4 et une zone jugeable : NON MESURABLE")
		fmt.Fprintf(sb, "# a2 %s : non mesurable (aucun slot avec tag 4 et zone)\n", e.short)
		return
	}
	p2aTag4Taux(t, sb, e, slots, dureeMS)
	p2aTag4Proprietaire(t, sb, e, slots, app)
}

// p2aTag4Inventaire publie QUI porte le tag 4 et QUI porte une zone, avant tout verdict.
//
// POURQUOI CET INVENTAIRE EST UNE MESURE ET NON UN JOURNAL DE DEBOGAGE : si les slots porteurs du
// tag 4 ne sont PAS ceux que la jauge du tag 3 designe, alors les deux canaux ne parlent pas des
// memes objets, et « le tag 4 est l'etat de la zone » tombe avant meme la clause temporelle. Le
// dire chiffres a l'appui vaut mieux qu'un « non mesurable » sans cause.
func p2aTag4Inventaire(t *testing.T, sb *strings.Builder, e p2aEntree, app p2aAppariement) {
	t.Helper()
	series := p2aSeries(e.sc.scal, p2aTagU32)
	communs := 0
	for _, s := range p2aSlotsTries(series) {
		_, aZone := app.zoneParSlot[s]
		if aZone {
			communs++
		}
		t.Logf("  slot %-5d : %d valeurs de tag 4, %d distinctes, %d changements · zone du tag 3 :"+
			" %v · valeurs %s", s, len(series[s]), p2aDistinctes(series[s]),
			len(p2aChangements(series[s])), aZone, p2aTopValeurs(series[s]))
		fmt.Fprintf(sb, "a2_inventaire\t%s\t%d\t%d\t%d\t%d\t%v\t%s\n", e.short, s, len(series[s]),
			p2aDistinctes(series[s]), len(p2aChangements(series[s])), aZone,
			p2aTopValeurs(series[s]))
	}
	t.Logf("  %d slots portent le tag 4, %d portent aussi une zone du tag 3 (carte de CB.2a.1 :"+
		" %d slots)", len(series), communs, len(app.zoneParSlot))
}

// p2aTopValeurs rend les trois valeurs les plus frequentes d'une serie, avec leur compte.
//
// POURQUOI LES PUBLIER PLUTOT QUE DE LES COMPTER. Un canal enumerable ne se caracterise pas par
// le NOMBRE de ses valeurs mais par LESQUELLES : `0xFFFFFFFF` dominant les deux equipes ne dit
// pas la meme chose que deux valeurs qui se partagent le film. Sans la liste, le negatif serait
// invérifiable.
func p2aTopValeurs(ss []p2aEch) string {
	compte := map[uint64]int{}
	for _, s := range ss {
		compte[s.pay]++
	}
	vals := make([]uint64, 0, len(compte))
	for v := range compte {
		vals = append(vals, v)
	}
	sort.Slice(vals, func(i, j int) bool {
		if compte[vals[i]] != compte[vals[j]] {
			return compte[vals[i]] > compte[vals[j]]
		}
		return vals[i] < vals[j]
	})
	var parts []string
	for i, v := range vals {
		if i == 3 {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, fmt.Sprintf("0x%08X x%d", v, compte[v]))
	}
	return strings.Join(parts, " ")
}

// p2aDistinctes compte les valeurs distinctes d'une serie — la clause d'ENUMERABILITE de la
// phase 1, republiee ici parce qu'elle conditionne la lecture du reste.
func p2aDistinctes(ss []p2aEch) int {
	vus := map[uint64]bool{}
	for _, s := range ss {
		vus[s.pay] = true
	}
	return len(vus)
}

// p2aTag4Slots assemble, pour chaque slot PORTEUR DU TAG 4, ses changements et les captures de
// SA zone.
//
// D'OU VIENT « SA ZONE », ET POURQUOI IL A FALLU DEUX SOURCES. La mesure du 2026-08-18 a etabli
// que les slots bavards en tag 4 ne sont PRESQUE JAMAIS ceux que la jauge du tag 3 designe (sur
// `7344d24f` : 10 slots portent le tag 4, 2 seulement portent aussi une zone, et ces deux-la
// n'ont qu'UNE valeur, donc zero changement). S'en tenir a la carte de CB.2a.1 rendrait « non
// mesurable » sans rien dire. La zone d'un slot de tag 4 est donc etablie, a defaut, PAR VOTE :
// la zone des captures qui tombent dans la fenetre de ses changements.
//
// CE QUE CE VOTE COUTE, ecrit ici et republie avec le verdict : la precision par zone devient
// PARTIELLEMENT CIRCULAIRE (on choisit la zone qui maximise la coincidence, puis on mesure la
// coincidence). C'est pourquoi le rapport publie AUSSI la precision et le rappel contre TOUTES
// les captures, qui eux ne doivent rien au vote, et le niveau du hasard dans les deux cas.
func p2aTag4Slots(e p2aEntree, app p2aAppariement) []p2aTag4Slot {
	series := p2aSeries(e.sc.scal, p2aTagU32)
	capsParZone := map[int][]p2aCapture{}
	var toutes []p2aCapture
	for _, c := range app.captures {
		if c.hasZone {
			capsParZone[c.rank] = append(capsParZone[c.rank], c)
			toutes = append(toutes, c)
		}
	}
	var out []p2aTag4Slot
	for _, s := range p2aSlotsTries(series) {
		ch := p2aChangements(series[s])
		if len(ch) < p2aMinParSlot {
			continue
		}
		z, source := p2aZoneDuSlotTag4(s, ch, app, capsParZone)
		if source == "aucune" || len(capsParZone[z]) < p2aMinParSlot {
			continue
		}
		out = append(out, p2aTag4Slot{slot: s, zone: z, source: source, changements: ch,
			captures: capsParZone[z], toutes: toutes})
	}
	return out
}

// p2aZoneDuSlotTag4 rend la zone d'un slot porteur du tag 4 : celle de la carte du tag 3 quand
// elle existe, sinon celle que ses changements designent le plus souvent.
func p2aZoneDuSlotTag4(s uint32, ch []p2aEch, app p2aAppariement,
	capsParZone map[int][]p2aCapture,
) (int, string) {
	if z, ok := app.zoneParSlot[s]; ok {
		return z, "tag3"
	}
	votes := map[int]int{}
	for _, e := range ch {
		for z, cs := range capsParZone {
			for _, c := range cs {
				if d := c.tMS - e.tMS; d <= p2aFenetreMS && d >= -p2aFenetreMS {
					votes[z]++
					break
				}
			}
		}
	}
	best, bestN := 0, 0
	for z, n := range votes {
		if n > bestN {
			best, bestN = z, n
		}
	}
	if bestN == 0 {
		return 0, "aucune"
	}
	return best, "vote"
}

// p2aChangements rend les echantillons dont la valeur DIFFERE de la precedente. Le premier
// echantillon n'en est pas un : il n'a pas de precedent, le compter gonflerait le denominateur.
func p2aChangements(ss []p2aEch) []p2aEch {
	var out []p2aEch
	for i := 1; i < len(ss); i++ {
		if ss[i].pay != ss[i-1].pay {
			out = append(out, ss[i])
		}
	}
	return out
}

// p2aTag4Taux mesure precision, rappel et leurs niveaux de hasard, slot par slot puis en somme.
func p2aTag4Taux(t *testing.T, sb *strings.Builder, e p2aEntree, slots []p2aTag4Slot, dureeMS int) {
	t.Helper()
	var precisN, precisD, rappelN, rappelD int
	var tousN, tousD int
	var hasardP, hasardR, poidsP, poidsR, hasardT, poidsT float64
	for i := range slots {
		s := &slots[i]
		capT := p2aTemps(s.captures)
		chT := p2aTempsEch(s.changements)
		toutT := p2aTemps(s.toutes)
		s.precisNum = p2aDansFenetre(chT, capT)
		s.rappelNum = p2aDansFenetre(capT, chT)
		precisTous := p2aDansFenetre(chT, toutT)
		hp := p2aCouverture(capT, e.sc.t0MS, e.sc.t1MS)
		hr := p2aCouverture(chT, e.sc.t0MS, e.sc.t1MS)
		ht := p2aCouverture(toutT, e.sc.t0MS, e.sc.t1MS)
		precisN, precisD = precisN+s.precisNum, precisD+len(chT)
		rappelN, rappelD = rappelN+s.rappelNum, rappelD+len(capT)
		tousN, tousD = tousN+precisTous, tousD+len(chT)
		hasardP, poidsP = hasardP+hp*float64(len(chT)), poidsP+float64(len(chT))
		hasardR, poidsR = hasardR+hr*float64(len(capT)), poidsR+float64(len(capT))
		hasardT, poidsT = hasardT+ht*float64(len(chT)), poidsT+float64(len(chT))
		t.Logf("  slot %-5d zone %d (%s) : %d changements, %d captures de la zone · precision"+
			" %d/%d = %.1f %% (hasard %.1f %%) · rappel %d/%d = %.1f %% (hasard %.1f %%) ·"+
			" precision TOUTES captures %d/%d = %.1f %% (hasard %.1f %%)", s.slot, s.zone,
			s.source, len(chT), len(capT), s.precisNum, len(chT),
			100*p2aRate(s.precisNum, len(chT)), 100*hp, s.rappelNum, len(capT),
			100*p2aRate(s.rappelNum, len(capT)), 100*hr, precisTous, len(chT),
			100*p2aRate(precisTous, len(chT)), 100*ht)
		fmt.Fprintf(sb, "a2_slot\t%s\t%d\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%.4f\t%.4f\t%.4f\n",
			e.short, s.slot, s.zone, s.source, len(chT), len(capT), s.precisNum, s.rappelNum,
			precisTous, hp, hr, ht)
	}
	t.Logf("  SANS CIRCULARITE (toutes captures, aucun vote de zone) : precision %d/%d = %.1f %%"+
		" pour un hasard de %.1f %% (facteur %.2fx)", tousN, tousD, 100*p2aRate(tousN, tousD),
		100*p2aRatio(hasardT, poidsT), p2aRatio(p2aRate(tousN, tousD), p2aRatio(hasardT, poidsT)))
	fmt.Fprintf(sb, "a2_sanscirc\t%s\t%d\t%d\t%.4f\t%.4f\n", e.short, tousN, tousD,
		p2aRate(tousN, tousD), p2aRatio(hasardT, poidsT))
	p2aTag4Verdict(t, sb, e, [4]int{precisN, precisD, rappelN, rappelD},
		[2]float64{p2aRatio(hasardP, poidsP), p2aRatio(hasardR, poidsR)}, dureeMS)
}

// p2aTag4Verdict tranche la clause temporelle contre ses deux seuils.
func p2aTag4Verdict(t *testing.T, sb *strings.Builder, e p2aEntree, n [4]int, h [2]float64, dureeMS int) {
	t.Helper()
	precision, rappel := p2aRate(n[0], n[1]), p2aRate(n[2], n[3])
	t.Logf("  ENSEMBLE (film de %d ms) : precision %d/%d = %.1f %% pour un hasard de %.1f %%"+
		" (facteur %.2fx, exige %.1fx) · rappel %d/%d = %.1f %% (hasard %.1f %%, seuil %.0f %%)",
		dureeMS, n[0], n[1], 100*precision, 100*h[0], p2aRatio(precision, h[0]),
		p2aFacteurPrecision, n[2], n[3], 100*rappel, 100*h[1], 100*p2aSeuilRappel)
	v := "NON TENU"
	if rappel >= p2aSeuilRappel && precision >= p2aFacteurPrecision*h[0] {
		v = "TENU"
	}
	t.Logf("  VERDICT CB.2a.2 (clause temporelle) : %s", v)
	fmt.Fprintf(sb, "a2_verdict\t%s\t%d\t%d\t%.4f\t%.4f\t%d\t%d\t%.4f\t%.4f\t%s\n", e.short,
		n[0], n[1], precision, h[0], n[2], n[3], rappel, h[1], v)
}

// p2aTemps / p2aTempsEch extraient les instants, tries.
func p2aTemps(cs []p2aCapture) []int {
	out := make([]int, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.tMS)
	}
	sort.Ints(out)
	return out
}

func p2aTempsEch(es []p2aEch) []int {
	out := make([]int, 0, len(es))
	for _, e := range es {
		out = append(out, e.tMS)
	}
	sort.Ints(out)
	return out
}

// p2aDansFenetre compte les instants de `xs` qui ont un instant de `ys` a moins de la fenetre.
func p2aDansFenetre(xs, ys []int) int {
	n := 0
	for _, x := range xs {
		if p2aProche(ys, x) {
			n++
		}
	}
	return n
}

// p2aProche dit si `ys` (trie) contient un instant a moins de la demi-fenetre de x.
func p2aProche(ys []int, x int) bool {
	i := sort.SearchInts(ys, x)
	if i < len(ys) && ys[i]-x <= p2aFenetreMS {
		return true
	}
	return i > 0 && x-ys[i-1] <= p2aFenetreMS
}

// p2aCouverture rend la part du film couverte par l'UNION des fenetres posees sur `ts` — le
// NIVEAU DU HASARD de la clause temporelle. Sans lui, un canal bavard tient n'importe quel seuil.
func p2aCouverture(ts []int, t0, t1 int) float64 {
	if len(ts) == 0 || t1 <= t0 {
		return 0
	}
	sorted := append([]int(nil), ts...)
	sort.Ints(sorted)
	total, curDeb, curFin := 0, sorted[0]-p2aFenetreMS, sorted[0]+p2aFenetreMS
	for _, x := range sorted[1:] {
		if x-p2aFenetreMS > curFin {
			total += curFin - curDeb
			curDeb, curFin = x-p2aFenetreMS, x+p2aFenetreMS
			continue
		}
		curFin = x + p2aFenetreMS
	}
	total += curFin - curDeb
	return p2aRatio(float64(total), float64(t1-t0))
}

// p2aRatio rend a/b, 0 quand b vaut 0.
func p2aRatio(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
