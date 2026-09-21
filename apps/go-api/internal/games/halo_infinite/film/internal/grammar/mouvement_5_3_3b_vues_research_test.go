//go:build research

package grammar

// mouvement_5_3_3b_vues_research_test.go — CE QUE L ECRIVAIN DE TRAME IMPOSE, MESURE (lot 5.3.3-b).
//
// # CE QUE LA LECTURE DU JEU A ETABLI, ET QUE CET INSTRUMENT VERIFIE
//
// `FUN_142987460` — le frame-processor — consomme un paquet de trame ainsi :
//
//	DAT_144706104 = R(1)                               // le drapeau de configuration
//	pour vue dans 0..2 :                               // TROIS vues, dans l ordre
//	    vtable[0x60](vue, capacite, sortie, &n)        // passe HORS BANDE : aucun bit lu
//	    vtable[0x40](vue, etat, LECTEUR, capacite, sortie, &n)   // FUN_1406cd128 : la boucle
//	pour vue dans 0..2 : pour chaque record de sa plage : vtable[0x48](vue, record)  // APPLIQUER
//
// LA BOUCLE EST APPELEE TROIS FOIS SUR LE MEME LECTEUR, et `DecodeFrameRecords` s arrete au
// PREMIER record de type 0 : il ne lit donc que la vue 0. Deux questions mesurables en decoulent,
// et ce fichier ne fait que les poser :
//
//	(1) reste-t-il des bits APRES la fin de la vue 0 sur un paquet sain ? (donc y a-t-il
//	    quelque chose a lire pour les vues 1 et 2 ?)
//	(2) les slots rejetes au PREMIER record sont-ils du BRUIT, ou de vraies entites que le
//	    monde de l instrument n a jamais liees ?
//
// (2) se tranche par la FREQUENCE : un identifiant lu dans du bruit ne revient pas, une entite
// reelle revient a chaque trame ou elle bouge.
//
// Rejouable :
//
//	MOUV533B_FILM=<dir du film> MOUV533B_CARTE=snowbound \
//	MOUV533B_BORNES=<catalogue> go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestMouvement533BVues$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m533bStat : les comptes de la passe.
type m533bStat struct {
	deltaTotal, listeVide, listePleine int
	sains, rejets, desyncs             int
	// bitsRestants : distribution du reste apres la fin de la vue 0, en classes.
	resteNul, resteCourt, resteLong int
	// vue1Sains / vue1Rejets : ce que donne la POURSUITE de la lecture apres la fin de la vue 0.
	vue1Records, vue1Sains, vue1Rejets int
	// slotsRejetes : combien de fois chaque slot est rejete au premier record.
	slotsRejetes map[uint32]int
	// slotsNew : les slots vus dans un record NEW, quelle que soit l issue de sa traversee.
	slotsNew map[uint32]bool
	// slotsSains : les slots vus dans un record sain.
	slotsSains map[uint32]bool
}

func TestMouvement533BVues(t *testing.T) {
	dir := os.Getenv("MOUV533B_FILM")
	if dir == "" {
		t.Skip("MOUV533B_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := m533bContexte(t, film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	cfg := fc.CadreDeBalayage()
	st := m533bStat{slotsRejetes: map[uint32]int{}, slotsNew: map[uint32]bool{},
		slotsSains: map[uint32]bool{}}
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			m533bPaquet(pk, data, w, cfg, &st)
		}
	}
	m533bRendre(t, st)
}

// m533bContexte ouvre le contexte du film, sous la carte quand elle est fournie.
func m533bContexte(t *testing.T, film *source.Film) *FilmContext {
	t.Helper()
	nom := os.Getenv("MOUV533B_CARTE")
	if nom == "" {
		return NewFilmContext(film)
	}
	chemin := os.Getenv("MOUV533B_BORNES")
	if chemin == "" {
		t.Fatalf("MOUV533B_BORNES attendu avec MOUV533B_CARTE")
	}
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	entry, err := cat.Lookup(nom)
	if err != nil {
		t.Fatalf("carte %q : %v", nom, err)
	}
	t.Logf("CARTE : %s (axes %v)", nom, entry.AxisWidths)
	fc := NewFilmContextForMap(film, &entry, nil)
	// LES LARGEURS D AXE DE LA CARTE S INSTALLENT ICI (cf. `m534Contexte` : sans elles le chemin
	// absolu d `i0` lit ses axes aux largeurs de `cliffhanger`, et tout le record derriere est du
	// bruit — cause mesuree au lot 5.3.5).
	bal := fc.ProfilDeBalayage()
	bal.PoserLargeursObjetDuMondeDepuisDecoupage(entry.Layout())
	fc.PoserProfilDeBalayage(bal)
	t.Logf("LARGEURS WORLD-OBJECT INSTALLEES : %v", fc.LargeursObjetDuMonde().AxisW)
	return fc
}

// m533bLierMonde lie les slots portes par les images-cles d un chunk (meme geste que la marche
// de reference de 5.3.2).
func m533bLierMonde(w *World, data []byte, pks []FilmPacket) {
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
			//nolint:gosec // slot, TI et Gen viennent du walker d image-cle, bornes par construction
			w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
		}
	}
}

// m533bPaquet traite UN paquet : il compte la population, puis lit la vue 0 et tente la suite.
func m533bPaquet(pk FilmPacket, data []byte, w *World, cfg FrameConfig, st *m533bStat) {
	if pk.Type != PacketTypeDelta || pk.Size < 1 {
		return
	}
	st.deltaTotal++
	pay := pk.Payload(data)
	if _, present := PacketHeadEventType(pay); present {
		st.listePleine++
		return // le decodeur de records ne sait pas sauter la liste d evenements
	}
	st.listeVide++
	br := LecteurSur(pay)
	recs, errD := DecodeFrameRecords(br, w, cfg)
	for _, r := range recs {
		if r.Type == recNew {
			st.slotsNew[r.Slot] = true
		}
	}
	if errD != nil {
		m533bEchec(recs, st)
		return
	}
	st.sains++
	for _, r := range recs {
		st.slotsSains[r.Slot] = true
	}
	m533bReste(br, pay, w, cfg, st)
}

// m533bEchec ventile un echec : rejet de generation au premier record, ou desync reelle.
func m533bEchec(recs []FrameRecord, st *m533bStat) {
	n := len(recs)
	if n == 0 {
		st.desyncs++
		return
	}
	l := recs[n-1]
	if l.TypeIndex == 0 && l.DesyncAt == 0 && len(l.Trace.Comps) == 0 {
		st.rejets++
		st.slotsRejetes[l.Slot]++
		return
	}
	st.desyncs++
}

// m533bReste mesure ce qui suit la fin de la vue 0, et tente la vue suivante sur le MEME lecteur.
func m533bReste(br *Lecteur, pay []byte, w *World, cfg FrameConfig, st *m533bStat) {
	reste := len(pay)*8 - br.BitPos()
	switch {
	case reste < 8:
		st.resteNul++
		return
	case reste < 24:
		st.resteCourt++
		return
	default:
		st.resteLong++
	}
	// LA VUE SUIVANTE, LUE SUR LE MEME LECTEUR — c est exactement ce que le frame-processor
	// fait (`vtable[0x40]` appele trois fois d affilee). L amorce de paquet ne se rejoue pas :
	// `DecodeFrameRecords` ne la consomme que si le lecteur est au bit 0.
	recs, errD := DecodeFrameRecords(br, w, cfg)
	st.vue1Records += len(recs)
	if errD == nil {
		st.vue1Sains++
		return
	}
	st.vue1Rejets++
}

// m533bRendre publie les tableaux de la passe.
func m533bRendre(t *testing.T, st m533bStat) {
	t.Helper()
	t.Logf("POPULATION : %d paquets delta · %d a liste VIDE (%.1f %%) · %d a liste PLEINE (%.1f %%)",
		st.deltaTotal, st.listeVide, m533bPart(st.listeVide, st.deltaTotal),
		st.listePleine, m533bPart(st.listePleine, st.deltaTotal))
	t.Logf("VUE 0 : %d saines · %d rejets au premier record · %d desyncs reelles",
		st.sains, st.rejets, st.desyncs)
	t.Logf("RESTE APRES LA FIN DE LA VUE 0 (sur %d paquets sains) : %d < 8 bits · %d < 24 bits · "+
		"%d >= 24 bits", st.sains, st.resteNul, st.resteCourt, st.resteLong)
	t.Logf("VUE SUIVANTE, LUE SUR LE MEME LECTEUR : %d paquets tentes · %d fins propres · "+
		"%d echecs · %d records lus", st.resteLong, st.vue1Sains, st.vue1Rejets, st.vue1Records)
	m533bRendreSlots(t, st)
}

// m533bRendreSlots tranche la question du BRUIT : un identifiant lu dans du bruit ne revient pas.
func m533bRendreSlots(t *testing.T, st m533bStat) {
	t.Helper()
	type paire struct {
		slot uint32
		n    int
	}
	ps := make([]paire, 0, len(st.slotsRejetes))
	var total, uniques, vusNew, vusSains int
	for s, n := range st.slotsRejetes {
		ps = append(ps, paire{s, n})
		total += n
		if n == 1 {
			uniques++
		}
		if st.slotsNew[s] {
			vusNew++
		}
		if st.slotsSains[s] {
			vusSains++
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].n > ps[j].n })
	t.Logf("SLOTS REJETES : %d distincts pour %d rejets · %d vus UNE SEULE fois (%.1f %%) · "+
		"%d vus dans un record NEW (%.1f %%) · %d vus dans un record SAIN (%.1f %%)",
		len(ps), total, uniques, m533bPart(uniques, len(ps)),
		vusNew, m533bPart(vusNew, len(ps)), vusSains, m533bPart(vusSains, len(ps)))
	var tete []string
	for i, p := range ps {
		if i == 12 {
			break
		}
		tete = append(tete, fmt.Sprintf("%d:%d", p.slot, p.n))
	}
	t.Logf("  les plus rejetes : %s", strings.Join(tete, " "))
	// LA LOI DE PUISSANCE EST LE TEMOIN : du bruit uniforme sur 2^13 slots donnerait une
	// ecrasante majorite de slots vus UNE fois ; des entites reelles donnent des queues epaisses.
	if len(ps) > 0 && ps[0].n > 10 {
		t.Logf("  VERDICT : le slot le plus rejete revient %d fois — ce n est PAS du bruit "+
			"uniforme, c est une entite reelle que le monde n a jamais liee", ps[0].n)
	}
}

// m533bPart rend un pourcentage, denominateur nul compris.
func m533bPart(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) * 100 / float64(d)
}

// m533bVuesStat : ce que rend la marche PAR VUES, celle du frame-processor.
type m533bVuesStat struct {
	paquets, vuesPleines, records, ti35 int
	pleins, pleinsLocalises             int
	i0, i1, i21, i25                    int
	i29, i62                            int
}

// TestMouvement533BParVues compare LA MARCHE DU JEU a la marche de la reference 5.3.2.
//
// LA REFERENCE LIT UNE VUE ; LE JEU EN LIT PLUSIEURS. `DecodeFrameViews` est le port du
// frame-processor (`FUN_142987460`) et il existe DANS LE DEPOT depuis le lot d origine, avec sa
// valeur de production : `killsource` et la marche des morts d objet deroulent HUIT vues
// (`marchViews`). La marche de reference de 5.3.2, elle, appelle `DecodeFrameRecords` UNE fois
// par paquet — donc la vue 0 seule, et son monde n est lie que par les images-cles.
//
// CE TEST NE CHANGE RIEN : il mesure les deux cote a cote, sur le meme film et le meme monde.
func TestMouvement533BParVues(t *testing.T) {
	dir := os.Getenv("MOUV533B_FILM")
	if dir == "" {
		t.Skip("MOUV533B_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := m533bContexte(t, film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	cfg := fc.CadreDeBalayage()
	var st m533bVuesStat
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			if _, present := PacketHeadEventType(pay); present {
				// LES 16,9 % A LISTE PLEINE ONT UNE PORTE, ET ELLE EST DANS LE DEPOT :
				// `marchLocateStrict` cherche la SIGNATURE du premier record d un paquet a
				// evenements (un delta du slot 123, long de 35 bits, a composant unique —
				// candidat unique et vrai sur 690 paquets sur 690). Elle rend le bit de debut
				// de la trame, donc la fin de la liste d evenements, sans porter la grammaire
				// de charge d aucun type.
				st.pleins++
				s := marchLocateStrict(pay, w, cfg)
				if s < 0 {
					continue
				}
				st.pleinsLocalises++
				recs, vues := DecodeFrameViews(pay, w, cfg, m533bNbVues(), s)
				st.vuesPleines += vues
				st.records += len(recs)
				m533bVentiler(recs, &st)
				continue
			}
			st.paquets++
			// `skipLeadBits = 2` : l amorce du paquet telle que la reference la consomme
			// ([config][continuation=0]). `nViews = 8` : la valeur de production du depot.
			recs, vues := DecodeFrameViews(pay, w, cfg, m533bNbVues(), 2)
			st.vuesPleines += vues
			st.records += len(recs)
			m533bVentiler(recs, &st)
			continue
		}
	}
	t.Logf("MARCHE PAR VUES (%d vues, amorce 2) : %d paquets · %d vues franchies (%.2f par "+
		"paquet) · %d records · %d de ti=35 (%.1f %%)", m533bNbVues(), st.paquets, st.vuesPleines,
		float64(st.vuesPleines)/float64(max(st.paquets, 1)), st.records, st.ti35,
		m533bPart(st.ti35, st.records))
	t.Logf("  ETALON sur ti=35 : i0 %.1f %% · i1 %.1f %% · i21 %.1f %% · i25 %.1f %%",
		m533bPart(st.i0, st.ti35), m533bPart(st.i1, st.ti35),
		m533bPart(st.i21, st.ti35), m533bPart(st.i25, st.ti35))
	t.Logf("  ETATS DE MOUVEMENT lus : i29 unit-crouch %d · i62 biped-slide %d", st.i29, st.i62)
	t.Logf("  PAQUETS A LISTE PLEINE : %d, dont %d LOCALISES par la signature (%.1f %%) — leurs "+
		"records sont inclus ci-dessus", st.pleins, st.pleinsLocalises,
		m533bPart(st.pleinsLocalises, st.pleins))
}

// m533bVues est le nombre de vues deroulees. Defaut : la valeur de production du depot
// (`marchViews` = 8). `MOUV533B_VUES` la surcharge — le frame-processor n en deroule que TROIS
// (`do { ... } while (uVar7 < 3)`), et l ecart merite d etre mesure plutot que suppose.
func m533bNbVues() int {
	if v := os.Getenv("MOUV533B_VUES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 8
}

// m533bVentiler compte les records de bipede et les bits d etalon de leur masque.
func m533bVentiler(recs []FrameRecord, st *m533bVuesStat) {
	for _, r := range recs {
		if r.TypeIndex != BipedTypeIndex {
			continue
		}
		st.ti35++
		for _, b := range []struct {
			i int
			n *int
		}{{0, &st.i0}, {1, &st.i1}, {21, &st.i21}, {25, &st.i25}, {29, &st.i29}, {62, &st.i62}} {
			if r.Trace.Mask&(1<<uint(b.i)) != 0 {
				*b.n++
			}
		}
	}
}
