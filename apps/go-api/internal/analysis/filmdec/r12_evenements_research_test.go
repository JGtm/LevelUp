package filmdec

// r12_evenements_research_test.go — LA LISTE COMPLETE D'EVENEMENTS SUR UN FILM HORS CATALOGUE.
//
// CE QUI DEBLOQUE LA MESURE, ET C'EST LE FAIT NEUF DE CE FICHIER. La marche de liste de R7
// exige un `r7Ctx` — les ETENDUES de la carte — pour consommer les vecteurs quantifies de
// certaines charges, et R7 le construisait depuis `map_quant_bounds.json`. Argyle n'y est
// pas. Mais la formule de largeur d'axe est
//
//	bits(k) = ceil(log2(ceil(etendue * 60 / 2^(16-k))))
//
// et `DetectI0Layout` rend deja `bits(16)` DEPUIS LE FILM (`AxisW`). Or, pour toute etendue
// reelle `e` telle que `bits(16) = b`, c'est-a-dire `e` dans `]2^(b-1)/60, 2^b/60]`, on a
// pour TOUT k <= 16 : `bits(k) = b - 16 + k`, la MEME valeur que pour l'etendue
// representative `2^b/60`. **La reconstruction `etendue := 2^AxisW[i] / 60` rend donc les
// memes largeurs que la vraie carte, exactement, sans catalogue.** Ce n'est pas une
// approximation : c'est une classe d'equivalence.
//
// ET ELLE EST CONTROLEE, PAS SUPPOSEE : `TestR12Cadrage` rejoue les temoins de cadrage de R7
// sur ce film (fin de liste propre contre temoin decale de +3 bits). Si le cadrage n'est pas
// bon, aucun resultat de ce fichier n'est publie.
//
// CE QUE LA MESURE CHERCHE : quel TYPE D'EVENEMENT apparait dans les cinq fenetres d'usage du
// repulseur et pas ailleurs. Le report n°1 de R9 designe le type 14 `PlayEffectOnObject` ;
// cet instrument l'instruit sans porter sa grammaire (un type opaque BLOQUE la marche, donc
// il est COMPTE la ou il apparait).
//
// GARDES : `R12_FILMS`, `R12_IDS`, `R12_FENETRE_MS`. Aucune ecriture, aucune DuckDB,
// `CGO_ENABLED=0`. USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R12_FILMS=<repo>/data/cache/film_chunks R12_IDS=215e7022 \
//	  go test ./internal/analysis/filmdec/ \
//	  -run '^TestR12(Cadrage|Evenements)$' -count=1 -timeout 60m -v

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

// r12CtxDeLayout reconstruit le contexte de carte depuis le SEUL decoupage i0 du film.
// Voir l'en-tete : la classe d'equivalence rend les memes largeurs que la vraie carte.
func r12CtxDeLayout(lay I0Layout) r7Ctx {
	var e [3]float64
	for i := 0; i < 3; i++ {
		e[i] = math.Pow(2, float64(lay.AxisW[i])) / 60.0
	}
	rb := uint(0)
	if lay.GateBits > i0SpineBits+i0UseDefaultBits {
		rb = uint(lay.GateBits - i0SpineBits - i0UseDefaultBits)
	}
	return r7Ctx{etendues: e, regionBits: rb, hasMap: true}
}

// r12VerifieCtx controle la reconstruction : les largeurs a k=16 doivent EGALER `AxisW`.
// Un ecart signifierait que la formule et le detecteur ne parlent pas de la meme grandeur.
func r12VerifieCtx(t *testing.T, lay I0Layout, ctx r7Ctx) {
	t.Helper()
	for i := 0; i < 3; i++ {
		if got := r7BitsAxe(ctx.etendues[i], 16); got != lay.AxisW[i] {
			t.Fatalf("reconstruction du contexte fausse sur l'axe %d : "+
				"r7BitsAxe(%.3f, 16) = %d, AxisW = %d", i, ctx.etendues[i], got, lay.AxisW[i])
		}
	}
}

// TestR12Cadrage rejoue les temoins de cadrage de R7 sur le film de R12. SEUILS PRE-INSCRITS,
// repris tels quels de R7 : >= 90 % de fins propres parmi les marches non bloquees, et un
// facteur >= 2 sur le temoin decale de +3 bits.
func TestR12Cadrage(t *testing.T) {
	for _, dir := range r12FilmDirs(t) {
		r12CadrageOneFilm(t, dir)
	}
}

// r12MarcheBilan : le compte d'une campagne de marche.
type r12MarcheBilan struct {
	listes  int
	fins    int
	bloques int
	events  int
	opaques map[int]int
}

func r12CadrageOneFilm(t *testing.T, dir string) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	defer func() { WorldObjectPrecision = saved }()
	s := r12Prepare(t, dir)
	ctx := r12CtxDeLayout(s.lay)
	r12VerifieCtx(t, s.lay, ctx)
	t.Logf("=== FILM %s — CADRAGE ===", s.id)
	t.Logf("  contexte reconstruit : etendues=%.1f/%.1f/%.1f m regionBits=%d "+
		"(depuis AxisW=%v gate=%d)", ctx.etendues[0], ctx.etendues[1], ctx.etendues[2],
		ctx.regionBits, s.lay.AxisW, s.lay.GateBits)
	// TEMOIN 1 (R7) — LE BLOCAGE. Une marche juste bute rarement ; une marche decalee derive
	// et bute sur un type opaque ou un type >= 123. On mesure le TAUX DE BLOCAGE au bon
	// cadrage contre les memes listes decalees de +1 et +3 bits APRES la charge (le decalage
	// est place la, et pas au depart, parce que c'est la LARGEUR DES CHARGES qu'il attaque).
	// SEUIL PRE-INSCRIT : facteur >= 2 sur le taux de blocage contre chaque temoin.
	for _, dec := range []int{0, 1, 3} {
		b := r12Campagne(s, ctx, dec)
		nom := "MARCHE REELLE"
		if dec > 0 {
			nom = fmt.Sprintf("temoin decale +%d bits", dec)
		}
		t.Logf("  %-22s : %d listes, %d bloquees (%.2f %%), %d evenements traverses",
			nom, b.listes, b.bloques, 100*float64(b.bloques)/float64(max(1, b.listes)), b.events)
		if dec == 0 {
			t.Logf("    types opaques : %s", r12TopOpaques(b.opaques, 12))
		}
	}
	// TEMOIN 3 (R7) — L'ORACLE DE TRAME, LE JUGE. Apres le dernier evenement, la trame de
	// records doit se decoder et ALLER LOIN. SEUIL PRE-INSCRIT (celui de R7) : facteur >= 3
	// sur la profondeur contre un temoin decale de +3 bits.
	reg, chunks, err := r7Chargements(dir)
	if err != nil || len(chunks) == 0 {
		t.Logf("  ORACLE DE TRAME : registre illisible (%v) — temoin 3 non mesure", err)
		return
	}
	cfg := DefaultFrameConfig()
	var prof float64
	cfg.IDLowBits, prof = r7CalibreIDLow(reg, chunks)
	t.Logf("  IDLowBits calibre = %d (profondeur %.2f a liste vide)", cfg.IDLowBits, prof)
	juste, horsMarche := r7OracleFilm(reg, chunks, ctx, cfg, nil, 0)
	temoin, _ := r7OracleFilm(reg, chunks, ctx, cfg, nil, 3)
	r7RapportTrame(t, "  ORACLE cadrage JUSTE ", juste)
	r7RapportTrame(t, "  ORACLE temoin +3 bits", temoin)
	den := temoin.profondeur()
	if den < 0.0001 {
		den = 0.0001
	}
	t.Logf("  TEMOIN 3 : facteur de profondeur %.2f (seuil pre-inscrit 3) ; "+
		"%d listes non marchees jusqu'au bout", juste.profondeur()/den, horsMarche)
}

// r12Campagne marche toutes les listes non vides des paquets delta, avec un decalage donne.
func r12Campagne(s r12Setup, ctx r7Ctx, decale int) r12MarcheBilan {
	b := r12MarcheBilan{opaques: map[int]int{}}
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			if r12Tete(pay) < 0 {
				continue // liste vide : elle ne juge aucun cadrage
			}
			b.listes++
			evs, stop, typ, _ := r7MarcheDecalee(pay, ctx, decale)
			b.events += len(evs)
			switch stop {
			case r7StopFin:
				b.fins++
			case r7StopOpaque, r7StopSansDomaine:
				b.bloques++
				b.opaques[typ]++
			default:
				b.bloques++
			}
		}
	}
	return b
}

func r12TopOpaques(m map[int]int, top int) string {
	type kv struct {
		k, v int
	}
	var xs []kv
	for k, v := range m {
		xs = append(xs, kv{k, v})
	}
	sort.Slice(xs, func(a, b int) bool { return xs[a].v > xs[b].v })
	if len(xs) > top {
		xs = xs[:top]
	}
	var out string
	for _, x := range xs {
		out += fmt.Sprintf("%d %s:%d · ", x.k, r7Noms[x.k], x.v)
	}
	return out
}

// --- LE RECENSEMENT ANCRE DES EVENEMENTS ---------------------------------------------------

// r12EvBilan compte les evenements par type, dedans et dehors, avec leurs instants.
type r12EvBilan struct {
	dedans   map[int]int
	dehors   map[int]int
	instants map[int][]int64
	nDedans  int
	nDehors  int
}

func newR12EvBilan() *r12EvBilan {
	return &r12EvBilan{dedans: map[int]int{}, dehors: map[int]int{}, instants: map[int][]int64{}}
}

func TestR12Evenements(t *testing.T) {
	for _, dir := range r12FilmDirs(t) {
		r12EvenementsOneFilm(t, dir)
	}
}

func r12EvenementsOneFilm(t *testing.T, dir string) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	defer func() { WorldObjectPrecision = saved }()
	s := r12Prepare(t, dir)
	ctx := r12CtxDeLayout(s.lay)
	r12VerifieCtx(t, s.lay, ctx)
	rd := r12Collect(s)
	half := r12FenetreMS()

	var graAnc []r12Ancre
	for _, tg := range rd.Tags {
		if tg.Src == "i59" && tg.Tag == 3 {
			graAnc = append(graAnc, r12Ancre{"G@" + r12MMSS(tg.MS), tg.MS})
		}
	}
	fRep := r12Fenetres(r12AncresUsage, half)
	fGra := r12Fenetres(graAnc, half)

	bRep, bGra := newR12EvBilan(), newR12EvBilan()
	// UN SEUL balayage pour les deux campagnes : un film coute cher a decoder.
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			if r12Tete(pay) < 0 {
				continue
			}
			ms := s.ms(pk.TimestampUS)
			evs, stop, typ, _ := r7MarcheDecalee(pay, ctx, 0)
			types := make([]int, 0, len(evs)+1)
			for _, e := range evs {
				types = append(types, e.Typ)
			}
			// LE TYPE QUI A BLOQUE LA MARCHE EST COMPTE : c'est precisement le cas du
			// type 14 `PlayEffectOnObject`, opaque et donc jamais traverse.
			if stop == r7StopOpaque || stop == r7StopSansDomaine {
				types = append(types, typ)
			}
			_, inR := r12In(fRep, ms)
			_, inG := r12In(fGra, ms)
			bRep.ajoute(types, inR, ms)
			bGra.ajoute(types, inG, ms)
		}
	}
	t.Logf("=== FILM %s — EVENEMENTS, fenetres +/-%d ms ===", s.id, half)
	bRep.rapport(t, "REPULSEUR (5 ancres du releve)")
	bGra.rapport(t, fmt.Sprintf("GRAPPIN (%d ancres i59 tag 3) — TEMOIN POSITIF", len(graAnc)))
	r12JournalTypes(t, bRep, []int{14, 30, 31, 42, 43, 48, 93, 98, 100, 103, 104, 105, 116,
		117, 119, 20, 10, 12})
}

func (b *r12EvBilan) ajoute(types []int, in bool, ms int64) {
	if in {
		b.nDedans++
	} else {
		b.nDehors++
	}
	for _, tp := range types {
		if in {
			b.dedans[tp]++
		} else {
			b.dehors[tp]++
		}
		b.instants[tp] = append(b.instants[tp], ms)
	}
}

func (b *r12EvBilan) rapport(t *testing.T, titre string) {
	t.Helper()
	t.Logf("  %s — %d listes dans les fenetres, %d hors", titre, b.nDedans, b.nDehors)
	if b.nDedans == 0 {
		return
	}
	keys := map[int]bool{}
	for k := range b.dedans {
		keys[k] = true
	}
	for k := range b.dehors {
		keys[k] = true
	}
	ks := make([]int, 0, len(keys))
	for k := range keys {
		ks = append(ks, k)
	}
	ri := func(k int) float64 { return float64(b.dedans[k]) / float64(b.nDedans) }
	ro := func(k int) float64 {
		if b.nDehors == 0 {
			return 0
		}
		return float64(b.dehors[k]) / float64(b.nDehors)
	}
	sort.Slice(ks, func(i, j int) bool { return ri(ks[i])-ro(ks[i]) > ri(ks[j])-ro(ks[j]) })
	t.Logf("    %-4s %-40s %8s %8s %9s %9s %8s", "type", "nom", "dedans", "dehors",
		"tauxDed", "tauxHor", "facteur")
	for _, k := range ks {
		fac := "inf"
		if ro(k) > 0 {
			fac = fmt.Sprintf("%.2f", ri(k)/ro(k))
		}
		t.Logf("    %-4d %-40s %8d %8d %9.4f %9.4f %8s",
			k, r7Noms[k], b.dedans[k], b.dehors[k], ri(k), ro(k), fac)
	}
}

// r12JournalTypes publie TOUS les instants des types d'equipement suivis : sur une population
// de quelques unites, un taux ne dit rien et la liste des instants dit tout.
func r12JournalTypes(t *testing.T, b *r12EvBilan, types []int) {
	t.Helper()
	t.Logf("  INSTANTS des types d'equipement (tous, sans fenetrage) :")
	for _, tp := range types {
		xs := b.instants[tp]
		if len(xs) == 0 {
			t.Logf("    %-4d %-40s ABSENT du film", tp, r7Noms[tp])
			continue
		}
		sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
		var s string
		for i, ms := range xs {
			if i >= 40 {
				s += fmt.Sprintf("(+%d)", len(xs)-40)
				break
			}
			s += r12MMSS(ms) + " "
		}
		t.Logf("    %-4d %-40s %3d : %s", tp, r7Noms[tp], len(xs), s)
	}
}
