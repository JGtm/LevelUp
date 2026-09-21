//go:build research

package grammar

// mouvement_5_3_2b_research_test.go — LA SUITE DE 5.3.2 : QUI PORTE L ACCROUPISSEMENT PAR
// IMAGE, ET LE SPRINT EN METRES PAR SECONDE.
//
// # LA CONSEQUENCE QUI COMMANDE CETTE MESURE
//
// Theater rejoue l accroupissement A CHAQUE IMAGE. Or 5.3.2 a mesure qu `i29 unit-crouch` ne
// voyage QU A L IMAGE-CLE (0,0 % des 162 444 records delta de `bfecd02b`, 99 % des records
// d image-cle). Donc l etat par image EXISTE et il n est PAS dans `i29`. Il est ailleurs, dans
// l un des composants qui, eux, voyagent en delta.
//
// # LES TROIS MESURES
//
//	(1) PORTEURS   frequence de chaque bit du masque sur les records delta, AVEC LE NOM du
//	               composant : la liste exacte de ce qui voyage par image.
//	(2) ORACLE     pour chaque composant porteur, bit par bit, l accord avec l accroupissement
//	               lu a l IMAGE-CLE LA PLUS PROCHE du meme slot. Un bit qui vaut 1 quand i29 dit
//	               accroupi et 0 sinon sur plus de 95 % des paires EST le porteur par image.
//	(3) VITESSE    la magnitude d `i1` DEQUANTIFIEE en m/s par `DecodeVelocityMagnitude` (loi
//	               log/exp entre 0,03 et 350, lue dans l exe et deja portee), par slot et en
//	               histogramme : deux bosses si le sprint se voit par la vitesse.
//
// # LA FENETRE DE L ORACLE EST ETROITE, ET C EST LA CONDITION DE SA VALIDITE
//
// Les images-cles sont espacees d environ 18 s ; un accroupissement dure quelques secondes. Un
// record delta pris a 9 s d une image-cle ne dit donc RIEN de l etat a cet instant. Seules les
// paires dont l ecart est inferieur a `MOUV532B_FENETRE_MS` (500 ms par defaut) sont retenues,
// et leur NOMBRE est publie : un accord de 100 % sur trois paires ne vaut rien, et le rapport
// doit le montrer plutot que le cacher.
//
//	MOUV532B_FILM=<repertoire de chunks> [MOUV532B_FENETRE_MS=500] \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	    -run '^TestMouvement532Porteurs$' -count=1 -v -timeout 60m

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m532bIgnores : les composants que la question EXCLUT — position, vitesse, orientation et
// visee. Ils voyagent par image et on sait deja ce qu ils portent ; les ventiler noierait le
// rapport sous des bits de coordonnees.
var m532bIgnores = map[string]bool{
	"object-position-dynamic-precision":               true,
	"object-translational-velocity-dynamic-precision": true,
	"object-forward-and-up":                           true,
	"object-angular-velocity":                         true,
	"unit-desired-aiming-vector":                      true,
}

// m532bBitsMax borne le nombre de bits ventiles par composant : au-dela, ce ne sont plus des
// drapeaux mais des charges (coordonnees, identifiants), et la question porte sur des drapeaux.
const m532bBitsMax = 24

// m532bSpan est UN composant lu dans UN record : son nom et ses bits.
type m532bSpan struct {
	nom      string
	startBit int
	largeur  int
}

// m532bRec est UN record delta reduit a ce que la mesure demande.
type m532bRec struct {
	slot  uint32
	tUS   uint64
	spans []m532bSpan
	pay   []byte
	vit   bool
	mag   uint32
}

// m532bKF est UN echantillon d accroupissement lu a l image-cle.
type m532bKF struct {
	tUS      uint64
	accroupi bool
}

// TestMouvement532Porteurs — LES TROIS MESURES, UN SEUL FILM.
func TestMouvement532Porteurs(t *testing.T) {
	dir := os.Getenv("MOUV532B_FILM")
	if dir == "" {
		t.Skip("MOUV532B_FILM absent : chemin du repertoire de chunks du film attendu")
	}
	recs, kf, noms := m532bLire(t, dir)
	if len(recs) == 0 {
		t.Fatalf("aucun record bipede delta lu dans %s", dir)
	}
	t.Logf("FILM %s : %d records delta · %d slots avec au moins un echantillon d image-cle",
		dir, len(recs), len(kf))
	m532bPorteurs(t, recs, noms)
	m532bOracle(t, recs, kf)
	m532bVitesse(t, recs)
}

// ---------------------------------------------------------------------------
// (1) LES PORTEURS
// ---------------------------------------------------------------------------

func m532bPorteurs(t *testing.T, recs []m532bRec, noms []string) {
	t.Helper()
	par := map[string]int{}
	largeurs := map[string]map[int]int{}
	for _, r := range recs {
		for _, sp := range r.spans {
			par[sp.nom]++
			if largeurs[sp.nom] == nil {
				largeurs[sp.nom] = map[int]int{}
			}
			largeurs[sp.nom][sp.largeur]++
		}
	}
	type ligne struct {
		nom string
		n   int
	}
	var lignes []ligne
	for n, c := range par {
		lignes = append(lignes, ligne{n, c})
	}
	sort.Slice(lignes, func(a, b int) bool { return lignes[a].n > lignes[b].n })
	t.Logf("(1) PORTEURS PAR IMAGE — %d composants distincts sur %d records delta", len(lignes), len(recs))
	for _, l := range lignes {
		marque := ""
		if m532bIgnores[l.nom] {
			marque = "  (exclu de l oracle)"
		}
		t.Logf("    %-52s %7d  %5.1f %%  largeurs %s%s",
			l.nom, l.n, m532Pct(l.n, len(recs)), m532bLargeurs(largeurs[l.nom]), marque)
	}
	_ = noms
}

func m532bLargeurs(m map[int]int) string {
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var parts []string
	for i, k := range cles {
		if i >= 4 {
			parts = append(parts, fmt.Sprintf("+%d autres", len(cles)-4))
			break
		}
		parts = append(parts, fmt.Sprintf("%db:%d", k, m[k]))
	}
	return strings.Join(parts, " ")
}

// ---------------------------------------------------------------------------
// (2) L ORACLE D ACCROUPISSEMENT
// ---------------------------------------------------------------------------

func m532bOracle(t *testing.T, recs []m532bRec, kf map[uint32][]m532bKF) {
	t.Helper()
	fenetre := uint64(500) * 1000
	if v := os.Getenv("MOUV532B_FENETRE_MS"); v != "" {
		if ms, err := strconv.ParseUint(v, 10, 64); err == nil {
			fenetre = ms * 1000
		}
	}
	type compteur struct{ accord, total [m532bBitsMax]int }
	par := map[string]*compteur{}
	paires := 0
	for _, r := range recs {
		etat, ok := m532bEtatProche(kf[r.slot], r.tUS, fenetre)
		if !ok {
			continue
		}
		paires++
		for _, sp := range r.spans {
			if m532bIgnores[sp.nom] {
				continue
			}
			c := par[sp.nom]
			if c == nil {
				c = &compteur{}
				par[sp.nom] = c
			}
			n := sp.largeur
			if n > m532bBitsMax {
				n = m532bBitsMax
			}
			for b := 0; b < n; b++ {
				v, okb := m532Lit(r.pay, sp.startBit+b, 1)
				if !okb {
					continue
				}
				c.total[b]++
				if (v == 1) == etat {
					c.accord[b]++
				}
			}
		}
	}
	t.Logf("(2) ORACLE D ACCROUPISSEMENT — fenetre %d ms · %d paires (record delta, image-cle du meme slot)",
		fenetre/1000, paires)
	if paires == 0 {
		t.Logf("    AUCUNE PAIRE : soit le film n a pas d image-cle exploitable, soit la fenetre est trop etroite")
		return
	}
	noms := make([]string, 0, len(par))
	for n := range par {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	trouve := false
	for _, nom := range noms {
		c := par[nom]
		meilleur, meilleurB, meilleurN := 0.0, -1, 0
		for b := 0; b < m532bBitsMax; b++ {
			if c.total[b] < 30 {
				continue
			}
			p := m532Pct(c.accord[b], c.total[b])
			if p > meilleur {
				meilleur, meilleurB, meilleurN = p, b, c.total[b]
			}
		}
		if meilleurB < 0 {
			continue
		}
		marque := ""
		if meilleur > 95 {
			marque = "   <-- PORTEUR PAR IMAGE"
			trouve = true
		}
		t.Logf("    %-52s meilleur bit %2d : accord %5.1f %% sur %d paires%s",
			nom, meilleurB, meilleur, meilleurN, marque)
	}
	if !trouve {
		t.Logf("    NEGATIF MESURE : aucun bit d aucun composant porte par image n atteint 95 %% "+
			"d accord avec l accroupissement de l image-cle la plus proche, sur %d paires.", paires)
	}
}

// m532bEtatProche rend l accroupissement de l image-cle la plus proche dans la fenetre.
func m532bEtatProche(ech []m532bKF, tUS, fenetre uint64) (bool, bool) {
	meilleur, ok := uint64(0), false
	var etat bool
	for _, e := range ech {
		d := e.tUS - tUS
		if e.tUS < tUS {
			d = tUS - e.tUS
		}
		if d > fenetre {
			continue
		}
		if !ok || d < meilleur {
			meilleur, etat, ok = d, e.accroupi, true
		}
	}
	return etat, ok
}

// ---------------------------------------------------------------------------
// (3) LA VITESSE EN METRES PAR SECONDE
// ---------------------------------------------------------------------------

func m532bVitesse(t *testing.T, recs []m532bRec) {
	t.Helper()
	var toutes []float32
	parSlot := map[uint32][]float32{}
	for _, r := range recs {
		if !r.vit {
			continue
		}
		v := DecodeVelocityMagnitude(uint64(r.mag), velScaleBitsDef)
		toutes = append(toutes, v)
		parSlot[r.slot] = append(parSlot[r.slot], v)
	}
	if len(toutes) == 0 {
		t.Logf("(3) VITESSE : aucun record ne porte la velocite")
		return
	}
	sort.Slice(toutes, func(a, b int) bool { return toutes[a] < toutes[b] })
	t.Logf("(3) VITESSE AU SOL, DEQUANTIFIEE (log/exp, 0,03 a 350 m/s, 10 bits) — %d echantillons",
		len(toutes))
	t.Logf("    p10 %.2f · p25 %.2f · median %.2f · p75 %.2f · p90 %.2f · p99 %.2f m/s",
		m532bQ(toutes, 0.10), m532bQ(toutes, 0.25), m532bQ(toutes, 0.50),
		m532bQ(toutes, 0.75), m532bQ(toutes, 0.90), m532bQ(toutes, 0.99))
	m532bHisto(t, toutes)
	m532bParSlot(t, parSlot)
}

// m532bHisto cherche LES DEUX BOSSES. Marche ~5,5 m/s, sprint environ 20 % plus vite : si le
// sprint se voit par la vitesse, l histogramme doit montrer deux maxima locaux separes.
func m532bHisto(t *testing.T, v []float32) {
	t.Helper()
	const pas = 0.5
	const hautMax = 12.0
	classes := make([]int, int(hautMax/pas)+1)
	for _, x := range v {
		i := int(x / pas)
		if i >= len(classes) {
			i = len(classes) - 1
		}
		classes[i]++
	}
	t.Logf("    HISTOGRAMME (pas de 0,5 m/s, tout ce qui depasse %.0f m/s dans la derniere classe) :", hautMax)
	for i, n := range classes {
		if n == 0 {
			continue
		}
		bosse := ""
		if i > 0 && i < len(classes)-1 && n > classes[i-1] && n > classes[i+1] && m532Pct(n, len(v)) > 2 {
			bosse = "   <-- maximum local"
		}
		t.Logf("      [%4.1f-%4.1f[ %7d  %5.1f %%%s", float64(i)*pas, float64(i+1)*pas, n,
			m532Pct(n, len(v)), bosse)
	}
}

func m532bParSlot(t *testing.T, parSlot map[uint32][]float32) {
	t.Helper()
	slots := make([]int, 0, len(parSlot))
	for s := range parSlot {
		slots = append(slots, int(s)) //nolint:gosec // slot d entite
	}
	sort.Ints(slots)
	t.Logf("    PAR SLOT (les 12 plus fournis) — mediane et p95, en m/s :")
	sort.Slice(slots, func(a, b int) bool {
		return len(parSlot[uint32(slots[a])]) > len(parSlot[uint32(slots[b])]) //nolint:gosec // clefs issues d uint32
	})
	for i, s := range slots {
		if i >= 12 {
			break
		}
		v := parSlot[uint32(s)] //nolint:gosec // clef issue d un uint32
		sort.Slice(v, func(a, b int) bool { return v[a] < v[b] })
		t.Logf("      slot %4d : n=%6d  median %.2f  p95 %.2f", s, len(v),
			m532bQ(v, 0.50), m532bQ(v, 0.95))
	}
}

func m532bQ(v []float32, q float64) float32 {
	if len(v) == 0 {
		return 0
	}
	return v[int(q*float64(len(v)-1))]
}

// ---------------------------------------------------------------------------
// LA LECTURE
// ---------------------------------------------------------------------------

// m532bLire ouvre UN film et rend ses records delta (avec les bornes de chaque composant) et,
// par slot, les echantillons d accroupissement lus aux images-cles.
//
//nolint:gocyclo // instrument de recherche : une seule marche, lisible de haut en bas
func m532bLire(t *testing.T, dir string) ([]m532bRec, map[uint32][]m532bKF, []string) {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	arch, ok := reg.Archetype(BipedTypeIndex)
	if !ok {
		t.Fatalf("archetype ti=%d absent", BipedTypeIndex)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	opt := ScanFilmOptions{RequireTag1: true, DropSaturated: true, QuantaOnly: true}
	chunks, err := bipedScanChunks(film, opt)
	if err != nil {
		t.Fatalf("chunks : %v", err)
	}
	band := bipedSlotBand(film, chunks)
	lay, err := bipedI0Layout(film, opt)
	if err != nil {
		t.Fatalf("layout i0 : %v", err)
	}
	ctx := fc.ContexteDeLecture()
	i0Bits := lay.TotalBits()

	var recs []m532bRec
	kf := map[uint32][]m532bKF{}
	for _, c := range chunks {
		data, pks, okc := FilmChunkAt(film, c)
		if !okc {
			continue
		}
		for _, pk := range pks {
			pay, ts := pk.Payload(data), pk.TimestampUS
			switch pk.Type {
			case PacketTypeDelta:
				walkDeltaBipedPayload(pay, band, lay, opt.RequireTag1, func(r deltaBipedRecord) {
					if rec, okr := m532bRecord(pay, r, arch, ctx, i0Bits, ts); okr {
						recs = append(recs, rec)
					}
				})
			case PacketTypeKeyframe:
				m532bKeyframe(pay, reg, ctx, ts, kf)
			}
		}
	}
	return recs, kf, arch.Components
}

// m532bRecord rejoue la boucle de composants de PRODUCTION et retient les bornes de chacun.
func m532bRecord(pay []byte, r deltaBipedRecord, arch Archetype, ctx ContexteDeLecture,
	i0Bits int, ts uint64) (m532bRec, bool) {
	br := LecteurSur(pay)
	br.PoserContexte(ctx)
	br.SetBitPos(r.I0 + i0Bits + i0TailBits)
	tr := EntityTrace{DesyncAt: -1, TypeIndex: BipedTypeIndex, Mask: m532Masque(r.Mask)}
	traverseComponentLoopFrom(br, arch, &tr, 1)
	tr.EndBit = br.BitPos()
	total := len(pay) * 8
	if tr.DesyncAt != -1 || tr.EndBit > total {
		return m532bRec{}, false
	}
	rec := m532bRec{slot: r.Slot, tUS: ts, pay: pay}
	for i, comp := range tr.Comps {
		fin := tr.EndBit
		if i+1 < len(tr.Comps) {
			fin = tr.Comps[i+1].StartBit
		}
		if comp.StartBit < 0 || fin > total || fin < comp.StartBit {
			continue
		}
		nom := m532Nom(comp.Name)
		rec.spans = append(rec.spans, m532bSpan{nom: nom, startBit: comp.StartBit, largeur: fin - comp.StartBit})
		if nom == m532Velocite {
			if b, okb := m532Lit(pay, comp.StartBit, 1); okb && b == 0 {
				if b2, okb2 := m532Lit(pay, comp.StartBit+1, 1); okb2 && b2 == 0 {
					if mag, okm := m532Lit(pay, comp.StartBit+2+int(aimDirBits), velScaleBits); okm {
						rec.vit, rec.mag = true, mag
					}
				}
			}
		}
	}
	return rec, true
}

// m532bKeyframe releve l accroupissement des records d image-cle, par slot.
func m532bKeyframe(pay []byte, reg *Registry, ctx ContexteDeLecture, ts uint64, out map[uint32][]m532bKF) {
	total := len(pay) * 8
	for _, b := range keyframeBornes(pay) {
		if b.TI != BipedTypeIndex {
			continue
		}
		tr := WalkKeyframeFullState(pay, b.Bit, reg, ctx)
		for i, comp := range tr.Comps {
			if m532Nom(comp.Name) != m532Crouch {
				continue
			}
			fin := tr.EndBit
			if i+1 < len(tr.Comps) {
				fin = tr.Comps[i+1].StartBit
			}
			if comp.StartBit < 0 || fin > total {
				continue
			}
			q, okq := m532Lit(pay, comp.StartBit+1, 10)
			if !okq {
				continue
			}
			slot := uint32(b.Slot) //nolint:gosec // slot d un record
			out[slot] = append(out[slot], m532bKF{tUS: ts, accroupi: q > 512})
		}
	}
}
