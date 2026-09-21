//go:build research

package grammar

// mouvement_5_3_2d_trame_research_test.go — LA VENTILATION PAR LE VRAI DECODEUR DE TRAME.
//
// # POURQUOI CE TROISIEME INSTRUMENT
//
// 5.3.2 a ventile `ti=35` avec `walkDeltaBipedPayload`, qui est un CHERCHEUR D ANCRES : il
// balaie le payload bit a bit et retient ce qui RESSEMBLE a un en-tete de bipede. Il a conclu
// `i29 unit-crouch` a 0,0 % des records delta — ce qui contredit trois documents du depot
// (« per-tick biped state … crouch »). Quand une mesure contredit le depot, c est l instrument
// qu on change.
//
// CE FICHIER UTILISE `DecodeFrameRecords` — le decodeur de trame du depot : preambule de
// paquet, puis les records DANS L ORDRE, contre un `World` amorce par les images-cles du chunk.
// C est la recette de `biped_pickup_research_test.go`, reprise telle quelle.
//
// # LA LIMITE, ECRITE
//
// `DecodeFrameRecords` saute la liste d evenements : il ne cadre donc que les paquets dont la
// liste est VIDE (bit de continuation a 0, soit `pay[0]&0x40 == 0`). Sur `bfecd02b` c est
// 83,1 % des paquets. Les 16,9 % restants portent un evenement et ne sont PAS ventiles ici.
//
// # L ETALON, ET IL EST EXIGE AVANT TOUTE CONCLUSION
//
// `i21 unit-desired-aiming-vector` doit etre vu sur une large part des records : le depot le
// dit lu par image. S il ne l est pas, l instrument est faux et le rapport le DIT au lieu de
// publier une cadence.
//
//	MOUV532D_FILM=<repertoire de chunks> \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	    -run '^TestMouvement532Trame$' -count=1 -v -timeout 60m

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m532dStat porte les denominateurs de la passe.
type m532dStat struct {
	paquets, trames, records int
	tmin, tmax               uint64
}

// m532dEchec est UN record sur lequel la marche a desynchronise : son masque et l index du
// composant fautif. C EST LA POPULATION INTERESSANTE — un delta ne porte `i29` QUE quand
// l accroupi CHANGE, donc ces records sont rares, et s ils sont precisement ceux qui echouent,
// l accroupi par instant est derriere une largeur fausse.
type m532dEchec struct {
	ti     uint32
	masque uint64
	fautif int
}

// TestMouvement532Trame — LA VENTILATION ETALONNEE.
func TestMouvement532Trame(t *testing.T) {
	dir := os.Getenv("MOUV532D_FILM")
	if dir == "" {
		t.Skip("MOUV532D_FILM absent : chemin du repertoire de chunks du film attendu")
	}
	ech, echecs, st := m532dLire(t, dir)
	t.Logf("FILM %s", dir)
	t.Logf("  paquets delta a liste VIDE : %d cadres, %d trames decodees sans erreur (%.1f %%)",
		st.paquets, st.trames, m532Pct(st.trames, st.paquets))
	t.Logf("  records de trame : %d au total, dont %d de ti=35 (%.1f %%)",
		st.records, len(ech), m532Pct(len(ech), st.records))
	if len(ech) == 0 {
		t.Fatalf("aucun record ti=35 : l instrument ne mesure rien")
	}
	m532dHistoEchecs(t, ech, echecs)
	if !m532dEtalon(t, ech) {
		return
	}
	m532dCadence(t, ech, st)
	m532dIntervalles(t, ech, st)
}

// m532dEtalon exige `i21` sur une large part des records AVANT de publier quoi que ce soit.
func m532dEtalon(t *testing.T, ech []m532Ech) bool {
	t.Helper()
	var i0, i1, i21, i25 int
	for _, e := range ech {
		if e.masque&1 != 0 {
			i0++
		}
		if e.masque&(1<<1) != 0 {
			i1++
		}
		if e.masque&(1<<21) != 0 {
			i21++
		}
		if e.masque&(1<<25) != 0 {
			i25++
		}
	}
	n := len(ech)
	t.Logf("ETALON : i0 %.1f %% · i1 %.1f %% · i21 %.1f %% · i25 %.1f %% (sur %d records ti=35)",
		m532Pct(i0, n), m532Pct(i1, n), m532Pct(i21, n), m532Pct(i25, n), n)
	if m532Pct(i21, n) < 50 {
		t.Logf("ETALON REFUSE : `i21` n est vu que sur %.1f %% des records alors que le depot le "+
			"dit lu PAR IMAGE. L instrument ne cadre donc pas la trame comme il faut, et AUCUNE "+
			"cadence n est publiee — publier un chiffre ici serait refaire l erreur de 5.3.2.",
			m532Pct(i21, n))
		return false
	}
	t.Logf("ETALON TENU : la ventilation qui suit porte sur une trame correctement cadree.")
	return true
}

// m532dCadence rend, pour chaque composant suivi, sa frequence et sa cadence en records par
// seconde et par slot — la reponse a « i29 est-il par tick ? ».
func m532dCadence(t *testing.T, ech []m532Ech, st m532dStat) {
	t.Helper()
	secondes := float64(st.tmax-st.tmin) / 1e6
	slots := map[uint32]bool{}
	for _, e := range ech {
		slots[e.slot] = true
	}
	var ctl, cr, mo, po, sl, vi int
	for _, e := range ech {
		if e.aControle {
			ctl++
		}
		if e.aCrouch {
			cr++
		}
		if e.aMobilite {
			mo++
		}
		if e.aPosture {
			po++
		}
		if e.aSlide {
			sl++
		}
		if e.aVitesse {
			vi++
		}
	}
	t.Logf("CADENCE (%.1f s de film, %d slots) — frequence, puis records par seconde et par slot",
		secondes, len(slots))
	type suivi struct {
		nom string
		n   int
	}
	for _, s := range []suivi{
		{"i1  object-translational-velocity", vi},
		{"i18 unit-control", ctl},
		{"i29 unit-crouch", cr},
		{"i54 biped-mobility-action", mo},
		{"i55 biped-posture-physics", po},
		{"i62 biped-slide", sl},
	} {
		par := 0.0
		if secondes > 0 && len(slots) > 0 {
			par = float64(s.n) / secondes / float64(len(slots))
		}
		t.Logf("    %-36s %7d  %5.1f %%  %6.2f rec/s/slot", s.nom, s.n,
			m532Pct(s.n, len(ech)), par)
	}
}

// m532dInter est UN intervalle d etat reconstruit par front du booleen.
type m532dInter struct {
	slot   uint32
	t0, t1 uint64
}

// m532dIntervalles reconstruit les intervalles d accroupi par slot et publie les tags d i55.
func m532dIntervalles(t *testing.T, ech []m532Ech, st m532dStat) {
	t.Helper()
	parSlot := map[uint32][]m532Ech{}
	for _, e := range ech {
		parSlot[e.slot] = append(parSlot[e.slot], e)
	}
	slots := make([]int, 0, len(parSlot))
	for s := range parSlot {
		slots = append(slots, int(s)) //nolint:gosec // slot d entite
	}
	sort.Ints(slots)
	var accroupis []m532dInter
	tags := map[uint32]int{}
	for _, s := range slots {
		serie := parSlot[uint32(s)] //nolint:gosec // clef issue d un uint32
		sort.Slice(serie, func(a, b int) bool { return serie[a].tUS < serie[b].tUS })
		var debut uint64
		dedans := false
		for _, e := range serie {
			if e.aPosture {
				tags[e.tag]++
			}
			if !e.aCrouch {
				continue
			}
			bas := e.crouch && e.crouchQ > 512
			switch {
			case bas && !dedans:
				debut, dedans = e.tUS, true
			case !bas && dedans:
				accroupis = append(accroupis, m532dInter{uint32(s), debut, e.tUS}) //nolint:gosec // clef
				dedans = false
			}
		}
	}
	m532dPublierIntervalles(t, accroupis, len(slots), st)
	if len(tags) > 0 {
		cles := make([]int, 0, len(tags))
		for k := range tags {
			cles = append(cles, int(k)) //nolint:gosec // tag sur 2 bits
		}
		sort.Ints(cles)
		var p []string
		for _, k := range cles {
			p = append(p, fmt.Sprintf("%d:%d", k, tags[uint32(k)])) //nolint:gosec // clef
		}
		t.Logf("I55 SUR LA TRAME : tags %s", strings.Join(p, " "))
	}
}

func m532dPublierIntervalles(t *testing.T, inters []m532dInter, nbSlots int, st m532dStat) {
	t.Helper()
	t.Logf("INTERVALLES D ACCROUPI (progression > 0,5) : %d sur %d slots", len(inters), nbSlots)
	if len(inters) == 0 {
		return
	}
	durees := make([]float64, 0, len(inters))
	for _, i := range inters {
		durees = append(durees, float64(i.t1-i.t0)/1e6)
	}
	sort.Float64s(durees)
	t.Logf("    duree : mediane %.2f s · p90 %.2f s · max %.2f s",
		durees[len(durees)/2], durees[(len(durees)*9)/10], durees[len(durees)-1])
	var parts []string
	var dernier uint64
	for _, i := range inters {
		if len(parts) > 0 && i.t0 < dernier+10_000_000 {
			continue
		}
		dernier = i.t0
		parts = append(parts, fmt.Sprintf("%s->%s (slot %d)",
			m532Barre(i.t0, st.tmin), m532Barre(i.t1, st.tmin), i.slot))
		if len(parts) == 5 {
			break
		}
	}
	t.Logf("    CINQ INSTANTS ACCROUPI (temps de barre Theater) : %s", strings.Join(parts, " · "))
}

// m532dLire decode le film par `DecodeFrameRecords`, chunk par chunk.
//
//nolint:gocyclo // instrument de recherche : une seule marche, lisible de haut en bas
func m532dLire(t *testing.T, dir string) ([]m532Ech, []m532dEchec, m532dStat) {
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
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	cfg := fc.CadreDeBalayage()
	var ech []m532Ech
	var echecs []m532dEchec
	var st m532dStat
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		w := m532dMonde(reg, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0]&0x40 != 0 {
				continue // la liste d evenements n est pas vide : hors du domaine de ce decodeur
			}
			st.paquets++
			if st.tmin == 0 || pk.TimestampUS < st.tmin {
				st.tmin = pk.TimestampUS
			}
			if pk.TimestampUS > st.tmax {
				st.tmax = pk.TimestampUS
			}
			br := LecteurSur(pay)
			recs, errD := DecodeFrameRecords(br, w, cfg)
			if errD != nil {
				// LES RECORDS PARTIELS SONT RENDUS AVEC L ERREUR : le dernier porte le masque et
				// l index du composant sur lequel la marche s est arretee. C est la matiere de
				// l histogramme des echecs.
				if n := len(recs); n > 0 {
					last := recs[n-1]
					echecs = append(echecs, m532dEchec{ti: last.TypeIndex, masque: last.Trace.Mask,
						fautif: last.DesyncAt})
				}
				continue
			}
			st.trames++
			st.records += len(recs)
			for _, r := range recs {
				if r.TypeIndex != BipedTypeIndex {
					continue
				}
				ech = append(ech, m532dEch(pay, r, pk.TimestampUS))
			}
		}
	}
	return ech, echecs, st
}

// m532dMonde amorce le monde du chunk par ses images-cles : sans lui, le decodeur de trame ne
// sait pas quel archetype porte un slot, et chaque record delta est illisible.
func m532dMonde(reg *Registry, data []byte, pks []FilmPacket) *World {
	w := NewWorld(reg)
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
			w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI)) //nolint:gosec // valeurs de registre
		}
	}
	return w
}

// m532dEch relit les champs suivis aux positions que la trame publie.
func m532dEch(pay []byte, r FrameRecord, ts uint64) m532Ech {
	e := m532Ech{slot: r.Slot, tUS: ts, masque: r.Trace.Mask}
	total := len(pay) * 8
	e.marcheComplete = r.Trace.DesyncAt == -1
	for i, comp := range r.Trace.Comps {
		fin := r.Trace.EndBit
		if i+1 < len(r.Trace.Comps) {
			fin = r.Trace.Comps[i+1].StartBit
		}
		if comp.StartBit < 0 || fin > total || fin < comp.StartBit {
			continue
		}
		m532Champ(pay, comp, fin, &e)
	}
	return e
}

// m532dHistoEchecs — L HISTOGRAMME DES ECHECS, ET LA COMPARAISON QUI DECIDE.
//
// LA QUESTION : un delta ne porte `i29` QUE quand l accroupi CHANGE. Ces records sont donc
// rares. S ils sont AUSSI ceux sur lesquels la marche desynchronise, alors l accroupi par
// instant est bien dans la trame, derriere une largeur fausse — et le composant fautif se
// nomme. Sinon, il n y est pas.
//
// LA COMPARAISON EST UN RAPPORT, PAS UN COMPTE : la part des masques portant `i29` parmi les
// records EN ECHEC, contre cette meme part parmi les records SAINS. Un compte seul ne dirait
// rien, les deux populations n ayant pas la meme taille.
func m532dHistoEchecs(t *testing.T, sains []m532Ech, echecs []m532dEchec) {
	t.Helper()
	suivis := []struct {
		nom string
		idx uint
	}{
		{"i18 unit-control", 18},
		{"i29 unit-crouch", 29},
		{"i54 biped-mobility-action", 54},
		{"i55 biped-posture-physics", 55},
		{"i62 biped-slide", 62},
	}
	var ech35 []m532dEchec
	parTI := map[uint32]int{}
	for _, e := range echecs {
		parTI[e.ti]++
		if e.ti == BipedTypeIndex {
			ech35 = append(ech35, e)
		}
	}
	t.Logf("ECHECS : %d records desynchronises, dont %d de ti=35 (%.1f %%)",
		len(echecs), len(ech35), m532Pct(len(ech35), len(echecs)))
	tis := make([]int, 0, len(parTI))
	for k := range parTI {
		tis = append(tis, int(k)) //nolint:gosec // index d archetype
	}
	sort.Slice(tis, func(a, b int) bool { return parTI[uint32(tis[a])] > parTI[uint32(tis[b])] }) //nolint:gosec // clefs
	var parts []string
	for i, ti := range tis {
		if i >= 6 {
			break
		}
		parts = append(parts, fmt.Sprintf("ti=%d:%d", ti, parTI[uint32(ti)])) //nolint:gosec // clef
	}
	t.Logf("    par archetype : %s", strings.Join(parts, " "))
	// LE COMPOSANT FAUTIF DES TROIS ARCHETYPES QUI ECHOUENT LE PLUS : quand la trame casse sur
	// un archetype, LE RESTE DU PAQUET EST PERDU — donc les records bipedes riches, qui viennent
	// apres, ne sont jamais atteints. Nommer ces composants, c est nommer le verrou.
	for i, ti := range tis {
		if i >= 3 {
			break
		}
		f := map[int]int{}
		for _, e := range echecs {
			if int(e.ti) == ti { //nolint:gosec // index d archetype
				f[e.fautif]++
			}
		}
		ks := make([]int, 0, len(f))
		for k := range f {
			ks = append(ks, k)
		}
		sort.Slice(ks, func(a, b int) bool { return f[ks[a]] > f[ks[b]] })
		var pp []string
		for j, k := range ks {
			if j >= 5 {
				break
			}
			pp = append(pp, fmt.Sprintf("i%d:%d", k, f[k]))
		}
		t.Logf("      ti=%-2d fautifs : %s", ti, strings.Join(pp, " "))
	}
	if len(ech35) == 0 {
		t.Logf("    AUCUN echec sur ti=35 : la desynchronisation ne vient pas du bipede")
		return
	}
	fautifs := map[int]int{}
	for _, e := range ech35 {
		fautifs[e.fautif]++
	}
	cles := make([]int, 0, len(fautifs))
	for k := range fautifs {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(a, b int) bool { return fautifs[cles[a]] > fautifs[cles[b]] })
	parts = parts[:0]
	for i, k := range cles {
		if i >= 8 {
			break
		}
		parts = append(parts, fmt.Sprintf("i%d:%d", k, fautifs[k]))
	}
	t.Logf("    COMPOSANT FAUTIF sur ti=35 (les 8 premiers) : %s", strings.Join(parts, " "))
	t.Logf("    SUR-REPRESENTATION DES MASQUES (part parmi les ECHECS ti=35 contre part parmi les SAINS) :")
	for _, s := range suivis {
		var e, sa int
		for _, x := range ech35 {
			if x.masque&(uint64(1)<<s.idx) != 0 {
				e++
			}
		}
		for _, x := range sains {
			if x.masque&(uint64(1)<<s.idx) != 0 {
				sa++
			}
		}
		pe, ps := m532Pct(e, len(ech35)), m532Pct(sa, len(sains))
		verdict := ""
		switch {
		case e == 0 && sa == 0:
			verdict = "  ABSENT DES DEUX POPULATIONS"
		case ps > 0 && pe > ps*3 && e >= 20:
			verdict = "  <-- SUR-REPRESENTE : piste de grammaire"
		case e == 0:
			verdict = "  jamais dans un masque en echec"
		}
		t.Logf("      %-30s echecs %5d (%5.2f %%) · sains %6d (%5.2f %%)%s",
			s.nom, e, pe, sa, ps, verdict)
	}
}
