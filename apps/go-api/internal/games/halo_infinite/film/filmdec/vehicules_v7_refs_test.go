package filmdec

// vehicules_v7_refs_test.go — INSTRUMENT (lot V7) : QUEL TYPE D'EVENEMENT DE TETE DATE LA
// DESTRUCTION D'UN VEHICULE ? Balayage aveugle des references portees par les types JAMAIS
// DECODES du registre des 30 types (V6 § 1.3).
//
// L'ORACLE, ECRIT AVANT LA MESURE.
//
// Le lot V6 a etabli la grammaire de tete au bit pres :
//
//	[1 bit config] [ 1 [R(7) type] [3 refs gardees] [charge] ]* 0 [trame ECS]
//
// Le corps d'un evenement commence donc TOUJOURS au bit 9 (`eventPayloadStartBit`), et les trois
// premieres choses qu'il porte sont des REFERENCES GARDEES : [garde(1)] ([sonde(1)] si domaine 1)
// [index(w)] [generation(2)]. Ce qui MANQUE pour les 28 types non decodes, c'est le DOMAINE de
// chaque ref, donc la largeur `w` — et Ghidra est mort.
//
// LA SUBSTITUTION : au lieu de LIRE la largeur, on la CHERCHE. Pour chaque type, chaque decalage
// de bit `b` et chaque largeur `w` plausible (7, 8, 9, 11, 13 — celles portees par le chantier),
// on lit une valeur brute et on la teste comme un slot de vehicule.
//
// LE CRITERE, ET POURQUOI IL N'EST PAS « LE SLOT EST VIVANT ». Un premier balayage (2 films) a
// montre que « la valeur designe un vehicule VIVANT a cet instant » ne discrimine RIEN : une vie
// de vehicule dure presque tout le film, donc le critere degenere en « la valeur est petite », et
// n'importe quel champ de petite valeur le passe (type 36, le tir, l'a passe a 99,0 % — avec un
// temoin temporel a 98,9 %, c'est-a-dire aucune information temporelle). LE CRITERE RETENU EST
// DONC LA FENETRE DE FIN DE VIE :
//
//	la valeur, lue comme un slot (brut, ou rapportee a la base de la bande `ti=40`), designe une
//	vie de vehicule dont la DISPARITION est bornee par l'instant du paquet :
//	   dernier recensement <= t <= premiere image-cle qui ne recense plus.
//
// Cette fenetre fait ~20 s (l'intervalle d'image-cle) et elle est PROPRE A CHAQUE VIE : un champ
// de petite valeur ne la passe que par hasard, et le temoin de chance chiffre ce hasard.
//
// LES TROIS TEMOINS, chacun neutralisant une facon de se tromper :
//
//  1. TEMOIN DE CHANCE (analytique, sans biais) : sous l'hypothese « ces bits sont du bruit
//     uniforme », la part attendue vaut exactement |{v < 2^w : v designe une vie en train de
//     disparaitre a t}| / 2^w, calculee A CHAQUE EVENEMENT sur la bande reelle. C'est le
//     denominateur honnete : une bande large et beaucoup de vies rendent la resolution facile.
//  2. TEMOIN TEMPOREL : la meme valeur, testee contre l'etat de la bande 60 s PLUS TARD. Il
//     conserve la valeur et la bande, et ne change QUE l'instant.
//  3. TEMOIN DE BANDE : la meme valeur testee contre les ARMES AU SOL (`ti=42`), meme regle de
//     fenetre de fin. Il conserve la valeur et l'instant, et ne change QUE la cible.
//
// CE QUI SERAIT UNE TROUVAILLE : un type dont un couple (b, w) tombe dans la fenetre de fin d'une
// vie de vehicule dans >= 90 % de ses instances alors que les trois temoins restent au plancher.
//
// Garde d'environnement V7_ROOT / V7_FILMS : sans elle, tout SKIP.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// v7CensusTolUS est la tolerance de fenetre d'une vie recensee, DE PART ET D'AUTRE. C'est la
// constante du calque (`replay.vehicleCensusTolUS`), reprise a l'identique : les images-cles sont
// espacees de ~20 s, une vie a donc pu naitre 20 s avant son premier recensement.
const v7CensusTolUS = uint64(20_000_000)

// v7ShiftUS est le decalage du TEMOIN TEMPOREL : 60 s, celui du chantier (rattachement V6 § 2.4).
const v7ShiftUS = uint64(60_000_000)

// v7MaxBit / v7Widths bornent le balayage. 120 bits couvrent trois references ET une charge (la
// sortie finit a 42 bits, l'embarquement a 52, le record de tir lit jusqu'au bit 143) ; les
// largeurs sont celles que le chantier a portees (domaines 2/3/4/6 et 1/7/8, sonde comprise).
const v7MaxBit = 120

var v7Widths = []int{7, 8, 9, 11, 13}

// v7Samples borne le nombre d'instances echantillonnees par type et par film : au-dela, la part
// mesuree ne bouge plus et le balayage coute pour rien.
const v7Samples = 400

// v7Life est une vie recensee : sa fenetre de presence, et la FENETRE DE SA DISPARITION.
type v7Life struct {
	lo, hi     uint64 // presence, tolerance de recensement comprise
	last, gone uint64 // dernier recensement, premiere preuve d'absence (0 = aucune)
}

// v7Bande porte une bande de slots et, par slot, les fenetres de ses vies.
type v7Bande struct {
	band  map[uint32]bool
	lives map[uint32][]v7Life
	slots []uint32 // bande triee (pour le temoin de chance)
	min   uint32
	ends  int // vies dont la disparition est bornee (denominateur du temoin de chance)
}

// v7BandeFrom construit la bande et les fenetres de vie d'un archetype.
func v7BandeFrom(k WorldObjectKeyframes) v7Bande {
	out := v7Bande{band: k.Band, lives: map[uint32][]v7Life{}, min: ^uint32(0)}
	for key, seen := range k.SeenUS {
		if len(seen) == 0 {
			continue
		}
		l := v7Life{lo: 0, last: seen[len(seen)-1]}
		if seen[0] > v7CensusTolUS {
			l.lo = seen[0] - v7CensusTolUS
		}
		l.gone = v7FirstAfter(k.TimesUS, l.last)
		l.hi = l.gone
		if l.hi == 0 {
			l.hi = l.last + v7CensusTolUS
		} else {
			out.ends++
		}
		out.lives[key.Slot] = append(out.lives[key.Slot], l)
	}
	for s := range k.Band {
		out.slots = append(out.slots, s)
		if s < out.min {
			out.min = s
		}
	}
	sort.Slice(out.slots, func(i, j int) bool { return out.slots[i] < out.slots[j] })
	if len(out.slots) == 0 {
		out.min = 0
	}
	return out
}

// alive dit si le slot est DANS la bande et si une de ses vies couvre l'instant.
func (b v7Bande) alive(slot uint32, at uint64) bool {
	if !b.band[slot] {
		return false
	}
	for _, l := range b.lives[slot] {
		if at >= l.lo && at <= l.hi {
			return true
		}
	}
	return false
}

// ending dit si l'instant tombe dans la FENETRE DE DISPARITION d'une vie de ce slot : entre son
// dernier recensement et la premiere image-cle qui ne la recense plus. C'est LE CRITERE du lot.
func (b v7Bande) ending(slot uint32, at uint64) bool {
	if !b.band[slot] {
		return false
	}
	for _, l := range b.lives[slot] {
		if l.gone > 0 && at >= l.last && at <= l.gone {
			return true
		}
	}
	return false
}

// lifeAt rend la vie de ce slot qui couvre l'instant, et false s'il n'y en a aucune.
func (b v7Bande) lifeAt(slot uint32, at uint64) (v7Life, bool) {
	for _, l := range b.lives[slot] {
		if at >= l.lo && at <= l.hi {
			return l, true
		}
	}
	return v7Life{}, false
}

// chance rend la part du domaine de valeurs [0, 2^w) qui designe une vie en train de disparaitre
// a `at`. C'est le TEMOIN 1, analytique — il n'y a rien a echantillonner.
func (b v7Bande) chance(w int, base uint32, at uint64) float64 {
	span := uint64(1) << uint(w)
	n := 0
	for _, s := range b.slots {
		if s < base || uint64(s-base) >= span {
			continue
		}
		if b.ending(s, at) {
			n++
		}
	}
	return float64(n) / float64(span)
}

// v7Cell est un compteur de couple (decalage, largeur, variante).
type v7Cell struct{ end, shift, other, alive int }

// v7Type accumule tout ce qu'on releve d'un type de tete.
type v7Type struct {
	n     int
	cells [][]v7Cell // [variante*len(v7Widths)+iw][bit]
	exp   []float64  // TEMOIN 1 cumule, par variante*largeur
}

func newV7Type() *v7Type {
	k := 2 * len(v7Widths)
	t := &v7Type{cells: make([][]v7Cell, k), exp: make([]float64, k)}
	for i := range t.cells {
		t.cells[i] = make([]v7Cell, v7MaxBit+1)
	}
	return t
}

// v7Scan porte le releve d'un balayage.
type v7Scan struct {
	types map[int]*v7Type
	seen  map[int]int // instances vues (avant plafond d'echantillonnage)
}

// scanFilm balaie un film et alimente le releve.
func (sc *v7Scan) scanFilm(dir string) (v7Bande, v7Bande) {
	veh := v7BandeFrom(ScanFilmWorldObjectKeyframes(dir, v0VehiculeTI))
	gnd := v7BandeFrom(ScanFilmWorldObjectKeyframes(dir, GroundWeaponTypeIndex))
	perFilm := map[int]int{}
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta || p.Size < 20 {
				continue
			}
			pay := p.Payload(data)
			ty, present := PacketHeadEventType(pay)
			if !present {
				continue
			}
			sc.seen[ty]++
			if perFilm[ty] >= v7Samples {
				continue
			}
			perFilm[ty]++
			sc.sample(ty, pay, p.TimestampUS, veh, gnd)
		}
	}
	return veh, gnd
}

// sample balaie UN evenement : toutes les positions de bit, toutes les largeurs, deux variantes
// (valeur brute, valeur rapportee a la base de la bande).
func (sc *v7Scan) sample(ty int, pay []byte, at uint64, veh, gnd v7Bande) {
	t := sc.types[ty]
	if t == nil {
		t = newV7Type()
		sc.types[ty] = t
	}
	t.n++
	bits := len(pay) * 8
	for iw, w := range v7Widths {
		for variant := 0; variant < 2; variant++ {
			vBase, gBase := uint32(0), uint32(0)
			if variant == 1 {
				vBase, gBase = veh.min, gnd.min
			}
			k := variant*len(v7Widths) + iw
			t.exp[k] += veh.chance(w, vBase, at)
			row := t.cells[k]
			for b := eventPayloadStartBit; b <= v7MaxBit && b+w <= bits; b++ {
				raw := readBitsAt(pay, b, w)
				if veh.ending(vBase+raw, at) {
					row[b].end++
				}
				if veh.ending(vBase+raw, at+v7ShiftUS) {
					row[b].shift++
				}
				if gnd.ending(gBase+raw, at) {
					row[b].other++
				}
				if veh.alive(vBase+raw, at) {
					row[b].alive++
				}
			}
		}
	}
}

// v7Res est le meilleur couple d'un type et le detail de ses temoins.
type v7Res struct {
	variant, w, bit          int
	end, shift, other, alive int
	exp                      float64
}

// best rend le couple (variante, largeur, bit) de plus fort EXCES sur le temoin de chance.
func (t *v7Type) best() v7Res {
	out := v7Res{}
	bestScore := -1.0
	for k := range t.cells {
		e := t.exp[k] / float64(v7Max(t.n, 1))
		for b, c := range t.cells[k] {
			if c.end == 0 {
				continue
			}
			if score := float64(c.end)/float64(t.n) - e; score > bestScore {
				bestScore = score
				out = v7Res{
					variant: k / len(v7Widths), w: v7Widths[k%len(v7Widths)], bit: b,
					end: c.end, shift: c.shift, other: c.other, alive: c.alive, exp: e,
				}
			}
		}
	}
	return out
}

func v7Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// v7FirstAfter rend le premier instant de `times` (TRIE) strictement posterieur a `at`, ou zero.
// C'est la PREMIERE PREUVE D'ABSENCE d'une vie recensee — meme regle que `replay.firstTimeAfter`.
func v7FirstAfter(times []uint64, at uint64) uint64 {
	i := sort.Search(len(times), func(k int) bool { return times[k] > at })
	if i >= len(times) {
		return 0
	}
	return times[i]
}

// v7FilmDirs : garde d'environnement.
func v7FilmDirs(t *testing.T) []string {
	t.Helper()
	root := os.Getenv("V7_ROOT")
	if root == "" {
		t.Skip("V7_ROOT non defini")
	}
	if l := os.Getenv("V7_FILMS"); l != "" {
		var out []string
		for _, s := range splitComma(l) {
			out = append(out, filepath.Join(root, s))
		}
		return out
	}
	return evtListFilmDirs(root)
}

// TestV7Refs — LE BALAYAGE. Une ligne par type de tete : le meilleur couple (bit, largeur), sa
// part de coincidence avec une fin de vie de vehicule, et les trois temoins.
func TestV7Refs(t *testing.T) {
	dirs := v7FilmDirs(t)
	sc := &v7Scan{types: map[int]*v7Type{}, seen: map[int]int{}}
	for _, d := range dirs {
		veh, gnd := sc.scanFilm(d)
		t.Logf("film %-10s vehicules : bande %3d slots (min %d) %3d vies dont %3d a fin bornee · "+
			"armes : bande %3d slots %4d vies",
			filepath.Base(filepath.Clean(d)), len(veh.band), veh.min, len(veh.lives), veh.ends,
			len(gnd.band), len(gnd.lives))
	}
	var tys []int
	for ty := range sc.types {
		tys = append(tys, ty)
	}
	sort.Ints(tys)
	t.Logf("== V7 REFS — %d films · balayage bits %d..%d · largeurs %v · critere FENETRE DE FIN ==",
		len(dirs), eventPayloadStartBit, v7MaxBit, v7Widths)
	t.Logf("%-5s %-7s %-6s %-4s %-4s %-5s %-9s %-11s %-11s %-9s %-8s",
		"type", "corpus", "echant", "var", "bit", "larg", "FIN", "temoin t+60", "temoin ti42",
		"chance", "vivant")
	for _, ty := range tys {
		p := sc.types[ty]
		if p.n == 0 {
			continue
		}
		r := p.best()
		pc := func(n int) float64 { return 100 * float64(n) / float64(p.n) }
		t.Logf("%-5d %-7d %-6d %-4d %-4d %-5d %6.1f %%  %8.1f %%  %8.1f %%  %6.2f %% %6.1f %%",
			ty, sc.seen[ty], p.n, r.variant, r.bit, r.w,
			pc(r.end), pc(r.shift), pc(r.other), 100*r.exp, pc(r.alive))
	}
}
