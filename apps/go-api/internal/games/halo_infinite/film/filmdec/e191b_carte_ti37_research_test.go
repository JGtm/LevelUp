package filmdec

// e191b_carte_ti37_research_test.go — LOT 1.9.1 bis, PAS 1 : LA CARTE DU TRAVAIL, MESUREE
// AVANT DE CODER (regle du chantier : mesure d abord, tableaux colles depuis l instrument).
//
// # LA QUESTION
//
// L archetype 37 (« equipment / item, objet du monde ») FERME 0 ou 1 record d image-cle sur
// 220 a 762 par bobine (golden `keyframe_closure.golden`), et la colonne « bloquant » de ce
// golden est VIDE : AUCUN composant ne desynchronise. Autrement dit les 31 composants sont
// tous consommes par un deserialiseur, mais la marche n atterrit pas sur la frontiere : une
// largeur AU MOINS est fausse, et il faut savoir laquelle avant d ouvrir Ghidra.
//
// # CE QUE L INSTRUMENT MESURE, ET CE QU IL NE PROUVE PAS
//
//	[1] fermeture ti=37 par bobine : fermes / bornes, desynchronisations, l HISTOGRAMME du
//	    RESIDU (`EndBit - Want`), et DEUX histogrammes qui bornent le defaut :
//	      — le premier composant qui COMMENCE deja au-dela de la frontiere du record (preuve :
//	        la marche s est trompee a cet index OU AVANT) ;
//	      — le composant qui FAIT franchir cette frontiere (celui d avant), avec la largeur
//	        moyenne qu il y consomme : c est le suspect numero un, et il se lit directement.
//	    Un residu CONSTANT designerait une largeur fixe fausse ; un residu qui varie designe
//	    un composant a largeur VARIABLE, donc une marche qui derive et lit du bruit.
//	[2] par composant i0..i30 : combien de fois consomme, et la distribution des largeurs
//	    reellement consommees (min, max, nombre de valeurs distinctes, somme). Un composant
//	    que la table ECS donne a largeur FIXE et qui rend ici plusieurs largeurs est un
//	    suspect ; un composant a largeur fixe est un candidat au residu constant.
//	[3] LE CONTROLE QUI DIT SI LE DEFAUT EST PROPRE A L EQUIPEMENT : la fermeture de TOUS les
//	    archetypes, separes en deux populations — ceux qui portent `object-position-component`
//	    (le prefixe « objet du monde », dont ti=37 fait partie) et les autres. Si les deux
//	    populations se comportent de la meme facon, le defaut est dans l equipement ; si seule
//	    la premiere echoue, il est dans le prefixe objet et ti=37 n en est qu une victime.
//	[4] la liste ORDONNEE des composants de ti=37 au registre, bobine par bobine : c est elle
//	    qui dit si les 31 composants sont les memes sur les 8 builds.
//
// CE QUE CA NE PROUVE PAS : une largeur consommee n est pas une largeur JUSTE. La fermeture
// est le seul oracle de justesse, et elle est globale au record. Cet instrument NOMME les
// suspects ; c est l ecrivain (Ghidra) qui tranche (D3).
//
// LECTURE SEULE, sans garde d environnement : les 7 bobines par build sont VERSIONNEES
// (`../replay/testdata/minifilm_*`), comme pour le ratchet 0.A.3.
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191bCarteTI37$' -v -count=1

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// e191bTI est l archetype mesure par ce pas : « equipment / item, objet du monde ».
const e191bTI = 37

// e191bLargeurs accumule, pour un composant, les largeurs consommees et leur frequence.
type e191bLargeurs struct {
	Nom   string
	Vues  int
	ParW  map[int]int
	Somme int
	MinW  int
	MaxW  int
}

// e191bBobine est la mesure d une bobine.
type e191bBobine struct {
	Court   string
	Fermes  int
	Bornes  int
	Desync  int
	Residus map[int]int
	// Depassement : index du PREMIER composant dont le bit de depart est DEJA au-dela de la
	// frontiere du record (-1 = la marche n a jamais depasse). C est une PREUVE, pas un
	// indice : un composant qui commence apres la fin du record signifie que la marche s est
	// trompee A CET INDEX OU AVANT. Elle borne donc le defaut par le haut.
	Depassement map[int]int
	// Coupable : le composant qui a FAIT franchir la frontiere (celui d avant le premier
	// depassement), et la somme des largeurs qu il a consommees dans ces records. Il NOMME le
	// suspect a relire chez l ecrivain en premier.
	Coupable  map[int]int
	CoupableW map[int]int
	Comps     map[int]*e191bLargeurs
	NbComps   int
	Registre  []string
}

func TestE191bCarteTI37(t *testing.T) {
	t.Logf("######## PAS 1 — CARTE DE L ARCHETYPE %d, MESUREE SUR LES 7 BOBINES PAR BUILD ########", e191bTI)
	total := &e191bBobine{Court: "TOTAL", Residus: map[int]int{}, Depassement: map[int]int{}, Coupable: map[int]int{}, CoupableW: map[int]int{}, Comps: map[int]*e191bLargeurs{}}
	for _, court := range closureMiniFilms() {
		b := e191bMesurerBobine(t, court)
		if b == nil {
			continue
		}
		e191bLogFermeture(t, b)
		e191bCumuler(total, b)
	}
	t.Logf("")
	t.Logf("==== [1] FERMETURE ti=%d — CUMUL DES 7 BOBINES ====", e191bTI)
	e191bLogFermeture(t, total)
	t.Logf("")
	t.Logf("==== [2] LARGEURS CONSOMMEES PAR COMPOSANT (cumul des 7 bobines) ====")
	e191bLogLargeurs(t, total)
	t.Logf("")
	t.Logf("==== [3] CONTROLE : fermeture des archetypes AVEC et SANS le prefixe objet ====")
	e191bControlePrefixeObjet(t)
	t.Logf("")
	t.Logf("==== [4] COMPOSANTS DE ti=%d AU REGISTRE, BOBINE PAR BOBINE ====", e191bTI)
	for _, court := range closureMiniFilms() {
		e191bLogRegistre(t, court)
	}
}

// e191bPrefixeObjet est le composant qui SIGNE le prefixe « objet du monde » : sa largeur
// depend des largeurs d axe de la CARTE (`WorldObjectPrecision`), installees par
// `replay.BuildFromFilm` et ABSENTES de cette mesure — c est le premier suspect.
const e191bPrefixeObjet = "object-position-component"

// e191bControlePrefixeObjet separe la fermeture de TOUS les archetypes en deux populations,
// selon qu ils portent ou non `object-position-component`.
func e191bControlePrefixeObjet(t *testing.T) {
	t.Helper()
	avecF, avecT, sansF, sansT := 0, 0, 0, 0
	detail := map[int][2]int{}
	for _, court := range closureMiniFilms() {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
		film, err := filmsource.LoadDir(dir, nil)
		if err != nil {
			t.Fatalf("LoadDir %s : %v", dir, err)
		}
		fc := NewFilmContext(film)
		reg, err := fc.Registry()
		if err != nil {
			t.Fatalf("registre %s : %v", court, err)
		}
		stats, err := KeyframeClosure(fc)
		if err != nil {
			t.Fatalf("KeyframeClosure %s : %v", court, err)
		}
		for ti, s := range stats {
			if e191bPorteLePrefixe(reg, int(ti)) { //nolint:gosec // ti est un index d archetype
				avecF, avecT = avecF+s.Closed, avecT+s.Total
				d := detail[int(ti)] //nolint:gosec // idem
				detail[int(ti)] = [2]int{d[0] + s.Closed, d[1] + s.Total}
				continue
			}
			sansF, sansT = sansF+s.Closed, sansT+s.Total
		}
	}
	t.Logf("  AVEC %-30s : %5d fermes / %5d bornes (%.2f %%)", e191bPrefixeObjet,
		avecF, avecT, 100*float64(avecF)/float64(max(avecT, 1)))
	t.Logf("  SANS %-30s : %5d fermes / %5d bornes (%.2f %%)", e191bPrefixeObjet,
		sansF, sansT, 100*float64(sansF)/float64(max(sansT, 1)))
	tis := make([]int, 0, len(detail))
	for ti := range detail {
		tis = append(tis, ti)
	}
	sort.Ints(tis)
	for _, ti := range tis {
		t.Logf("     ti=%-3d %5d / %5d", ti, detail[ti][0], detail[ti][1])
	}
}

// e191bPorteLePrefixe dit si l archetype porte `object-position-component`.
func e191bPorteLePrefixe(reg *Registry, ti int) bool {
	arch, ok := reg.Archetype(ti)
	if !ok {
		return false
	}
	for _, n := range arch.Components {
		if n == e191bPrefixeObjet {
			return true
		}
	}
	return false
}

// e191bLogRegistre imprime la liste ordonnee des composants de ti=37 pour une bobine.
func e191bLogRegistre(t *testing.T, court string) {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	reg, err := NewFilmContext(film).Registry()
	if err != nil {
		t.Fatalf("registre %s : %v", court, err)
	}
	arch, ok := reg.Archetype(e191bTI)
	if !ok {
		t.Logf("  %-10s : ti=%d ABSENT du registre", court, e191bTI)
		return
	}
	var parts []string
	for i, n := range arch.Components {
		parts = append(parts, fmt.Sprintf("i%d=%s", i, n))
	}
	t.Logf("  %-10s (%2d) %s", court, len(arch.Components), strings.Join(parts, " "))
}

// e191bMesurerBobine charge une bobine et mesure la fermeture ti=37, record par record.
func e191bMesurerBobine(t *testing.T, court string) *e191bBobine {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre %s : %v", court, err)
	}
	b := &e191bBobine{Court: court, Residus: map[int]int{}, Depassement: map[int]int{}, Coupable: map[int]int{}, CoupableW: map[int]int{}, Comps: map[int]*e191bLargeurs{}}
	if arch, ok := reg.Archetype(e191bTI); ok {
		b.NbComps = len(arch.Components)
		b.Registre = append(b.Registre, arch.Components...)
	}
	for _, num := range fc.ChunkNumbers() {
		data, packets, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range packets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			e191bAccumulerPayload(b, pk.Payload(data), reg)
		}
	}
	return b
}

// e191bAccumulerPayload classe les records ti=37 BORNES d un payload d image-cle.
func e191bAccumulerPayload(b *e191bBobine, pay []byte, reg *Registry) {
	for _, borne := range keyframeBornes(pay) {
		if borne.TI != e191bTI {
			continue
		}
		tr := WalkKeyframeFullState(pay, borne.Bit, reg)
		b.Bornes++
		if tr.DesyncAt >= 0 {
			b.Desync++
			continue
		}
		if tr.EndBit == borne.Want {
			b.Fermes++
		}
		b.Residus[tr.EndBit-borne.Want]++
		b.Depassement[e191bPremierDepassement(tr, borne.Want)]++
		if ci, cw := e191bCoupable(tr, borne.Want); ci >= 0 {
			b.Coupable[ci]++
			b.CoupableW[ci] += cw
		}
		e191bAccumulerLargeurs(b, tr)
	}
}

// e191bPremierDepassement rend l index du PREMIER composant qui COMMENCE deja au-dela de la
// frontiere du record, ou -1 si la marche n a jamais depasse.
//
// C EST UNE PREUVE, ET C EST LA SEULE DE CET INSTRUMENT. Un composant dont le bit de depart
// est superieur a la frontiere du record suivant ne peut pas appartenir a ce record : la
// marche s est donc trompee A CET INDEX OU AVANT. L histogramme de cet index BORNE le defaut
// par le haut, sans rien supposer des grammaires.
func e191bPremierDepassement(tr EntityTrace, want int) int {
	for k := range tr.Comps {
		if tr.Comps[k].StartBit > want {
			return tr.Comps[k].Index
		}
	}
	return -1
}

// e191bCoupable rend le composant qui a FAIT franchir la frontiere : celui qui precede
// immediatement le premier depassement, avec la largeur qu il a consommee. C est le suspect
// NUMERO UN de ce record — la marche etait encore dans le record avant lui et ne l est plus
// apres. (-1, 0 quand la marche n a jamais depasse.)
func e191bCoupable(tr EntityTrace, want int) (idx, largeur int) {
	for k := range tr.Comps {
		if tr.Comps[k].StartBit <= want {
			continue
		}
		if k == 0 {
			return -1, 0 // le depassement precede la boucle : en-tete ou etat par defaut
		}
		return tr.Comps[k-1].Index, tr.Comps[k].StartBit - tr.Comps[k-1].StartBit
	}
	return -1, 0
}

// e191bAccumulerLargeurs releve la largeur consommee par chaque composant de la marche.
func e191bAccumulerLargeurs(b *e191bBobine, tr EntityTrace) {
	for k := range tr.Comps {
		fin := tr.EndBit
		if k+1 < len(tr.Comps) {
			fin = tr.Comps[k+1].StartBit
		}
		w := fin - tr.Comps[k].StartBit
		c := b.Comps[tr.Comps[k].Index]
		if c == nil {
			c = &e191bLargeurs{Nom: tr.Comps[k].Name, ParW: map[int]int{}, MinW: w, MaxW: w}
			b.Comps[tr.Comps[k].Index] = c
		}
		c.Vues++
		c.ParW[w]++
		c.Somme += w
		if w < c.MinW {
			c.MinW = w
		}
		if w > c.MaxW {
			c.MaxW = w
		}
	}
}

// e191bCumuler additionne une bobine dans le total.
func e191bCumuler(total, b *e191bBobine) {
	total.Fermes += b.Fermes
	total.Bornes += b.Bornes
	total.Desync += b.Desync
	for r, n := range b.Residus {
		total.Residus[r] += n
	}
	for i, n := range b.Depassement {
		total.Depassement[i] += n
	}
	for i, n := range b.Coupable {
		total.Coupable[i] += n
		total.CoupableW[i] += b.CoupableW[i]
	}
	if len(total.Registre) == 0 {
		total.Registre, total.NbComps = b.Registre, b.NbComps
	}
	for i, c := range b.Comps {
		e191bFusionner(total, i, c)
	}
}

// e191bFusionner ajoute la mesure d un composant d une bobine au cumul.
func e191bFusionner(total *e191bBobine, i int, c *e191bLargeurs) {
	d := total.Comps[i]
	if d == nil {
		d = &e191bLargeurs{Nom: c.Nom, ParW: map[int]int{}, MinW: c.MinW, MaxW: c.MaxW}
		total.Comps[i] = d
	}
	d.Vues += c.Vues
	d.Somme += c.Somme
	for w, n := range c.ParW {
		d.ParW[w] += n
	}
	if c.MinW < d.MinW {
		d.MinW = c.MinW
	}
	if c.MaxW > d.MaxW {
		d.MaxW = c.MaxW
	}
}

// e191bLogFermeture imprime la ligne de fermeture d une bobine et son histogramme de residus.
func e191bLogFermeture(t *testing.T, b *e191bBobine) {
	t.Helper()
	t.Logf("%-10s | composants au registre %2d | bornes %4d | fermes %3d | desync %3d",
		b.Court, b.NbComps, b.Bornes, b.Fermes, b.Desync)
	type rn struct{ r, n int }
	rs := make([]rn, 0, len(b.Residus))
	for r, n := range b.Residus {
		rs = append(rs, rn{r, n})
	}
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].n != rs[j].n {
			return rs[i].n > rs[j].n
		}
		return rs[i].r < rs[j].r
	})
	var parts []string
	for k, x := range rs {
		if k >= 12 {
			parts = append(parts, fmt.Sprintf("... (%d autres valeurs)", len(rs)-12))
			break
		}
		parts = append(parts, fmt.Sprintf("%+d:%d", x.r, x.n))
	}
	t.Logf("%-10s | residus (EndBit-Want) sur %d valeurs distinctes : %s",
		"", len(rs), strings.Join(parts, "  "))
	e191bLogDepassement(t, b)
	e191bLogCoupable(t, b)
}

// e191bLogDepassement imprime, index par index, combien de records ont VU leur marche
// commencer un composant au-dela de la frontiere du record.
func e191bLogDepassement(t *testing.T, b *e191bBobine) {
	t.Helper()
	idx := make([]int, 0, len(b.Depassement))
	for i := range b.Depassement {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	var parts []string
	for _, i := range idx {
		nom := fmt.Sprintf("i%d", i)
		if i < 0 {
			nom = "jamais"
		}
		parts = append(parts, fmt.Sprintf("%s:%d", nom, b.Depassement[i]))
	}
	t.Logf("%-10s | premier composant COMMENCANT au-dela de la frontiere : %s",
		"", strings.Join(parts, "  "))
}

// e191bLogLargeurs imprime, composant par composant, la distribution des largeurs consommees.
func e191bLogLargeurs(t *testing.T, b *e191bBobine) {
	t.Helper()
	t.Logf("  %-4s %-52s %7s %6s %6s %5s  %s", "i", "composant", "vues", "min", "max", "#w", "largeurs (w:n)")
	idx := make([]int, 0, len(b.Comps))
	for i := range b.Comps {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	for _, i := range idx {
		c := b.Comps[i]
		t.Logf("  i%-3d %-52s %7d %6d %6d %5d  %s", i, c.Nom, c.Vues, c.MinW, c.MaxW,
			len(c.ParW), e191bTopLargeurs(c.ParW, 6))
	}
}

// e191bTopLargeurs rend les `n` largeurs les plus frequentes, du plus frequent au moins.
func e191bTopLargeurs(m map[int]int, n int) string {
	type wn struct{ w, c int }
	xs := make([]wn, 0, len(m))
	for w, c := range m {
		xs = append(xs, wn{w, c})
	}
	sort.Slice(xs, func(i, j int) bool {
		if xs[i].c != xs[j].c {
			return xs[i].c > xs[j].c
		}
		return xs[i].w < xs[j].w
	})
	var parts []string
	for k, x := range xs {
		if k >= n {
			parts = append(parts, fmt.Sprintf("+%d autres", len(xs)-n))
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%d", x.w, x.c))
	}
	return strings.Join(parts, " ")
}

// e191bLogCoupable imprime le suspect numero un : le composant qui a fait franchir la
// frontiere, avec le nombre de records concernes et la largeur MOYENNE qu il y consommait.
func e191bLogCoupable(t *testing.T, b *e191bBobine) {
	t.Helper()
	idx := make([]int, 0, len(b.Coupable))
	for i := range b.Coupable {
		idx = append(idx, i)
	}
	sort.Slice(idx, func(i, j int) bool { return b.Coupable[idx[i]] > b.Coupable[idx[j]] })
	var parts []string
	for k, i := range idx {
		if k >= 8 {
			parts = append(parts, fmt.Sprintf("+%d autres", len(idx)-8))
			break
		}
		parts = append(parts, fmt.Sprintf("i%d:%d(w~%d)", i, b.Coupable[i],
			b.CoupableW[i]/max(b.Coupable[i], 1)))
	}
	t.Logf("%-10s | composant qui FAIT franchir la frontiere : %s", "", strings.Join(parts, "  "))
}
