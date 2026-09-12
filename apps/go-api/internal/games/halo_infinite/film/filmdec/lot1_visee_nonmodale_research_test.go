package filmdec

// lot1_visee_nonmodale_research_test.go — LOT 1 : LA VISEE DES TIRS NON-MODAUX (ceux qui
// TOUCHENT : >= 1 cible ou >= 1 composante de degat). Le decodeur modal (fire_aim_modal.go)
// ne lit la visee QUE sur le cas modal (0 cible, 0 composante), a post-comptes + 2. Le
// non-modal insere, AVANT la visee, deux boucles de longueur variable puis deux composites.
//
// GRAMMAIRE DES DEUX BOUCLES + COMPOSITES — tracee au desassemblage de FUN_14080c1f8 (le
// lecteur de charge du record 0xD2), instruction par instruction, param_5 CONFIRME == 0 en
// mode film (le chemin modal appelle FUN_1406cd5b8, branche param_5==0). Ordre reel :
//
//	comptes (FUN_14080cc68 : cibles P @0xf8 lu en premier, composantes N @0x34 en second)
//	BOUCLE COMPOSANTES (N iterations) : par composante R(2) + R(1) + R(32)   = 35 bits FIXES
//	   (le R(2) est memorise : component[i].R2, relu par la boucle cibles)
//	BOUCLE CIBLES (P iterations), mode = 12 si P==1 sinon 4 :
//	   par cible R(4) + R(1)[hit] ; si hit==1 :
//	      R(3)(FUN_1406d310c(6)=3) + (N<3 ? R(1) : R(4))->idx + R(16) + 3*W bits
//	      W = FUN_14102bd24(mode, component[idx].R2) : min(mode,6) si R2==1, sinon mode
//	      (FUN_140c1e924 -> FUN_140c1e9d4 lit 3 champs de W bits)
//	COMPOSITES FUN_1406cd5b8 + FUN_1408eff64 (grammaires bit-exactes : lot1SkipCd5b8 /
//	   lot1SkipEff64, verifiees au desassemblage : cd5b8 = A(1)+B(1)+[si B: c9eabc, si A:
//	   R(4)+R(4), flags R(3), si flags&2: R(1)+[si0: R(20)+R(14)]]+[si A: R(1)+[si1: R(5)]])
//	VISEE R(30).
//
// AMBIGUITES DU RELEVE INITIAL, TRANCHEES ICI (Ghidra) : param_5 = 0 (film) -> R(32) et non
// une ref de domaine ; W = 3 * table PURE (FUN_14102bd24), AUCUNE dependance runtime. Les
// deux boucles ET les composites sont donc ENTIEREMENT decodables hors ligne — ce qui REFUTE
// la reserve de la note modale (« boucles cibles/composantes restent runtime-width »).
//
// RESULTAT DE L'ORACLE (garde-fou : ne rien survendre). Malgre le decode bit-exact, la visee
// non-modale n'est PAS validable par l'oracle de concentration :
//   - TOUS les tirs qui touchent arrivent en (1 cible, 1 composante) : impossible d'isoler
//     une boucle empiriquement (0 record composante-seule / cible-seule).
//   - validite cubemap non discriminante a 30 bits (face<6 quasi toujours) : seule la
//     concentration juge ; or elle produit des FAUX POSITIFS sur les champs a faible entropie
//     du record (un R(32) de petite valeur -> face 0 -> (1, petit, petit) -> axe x sature)
//     tandis que la vraie visee R(30) qui suit reste au bruit (43-55 %, vs modal 83-95 %).
//   - le tir qui TOUCHE vise a des elevations variees (pas le biais horizontal du tir rate) :
//     un vecteur unitaire uniforme donne E|x|=E|y|=E|z|=0.5, indiscernable du bruit.
// CONCLUSION : le decode non-modal est desormais possible (grammaire complete), mais sa
// CORRECTION reste non prouvable hors ligne avec l'oracle disponible.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule.

import (
	"os"
	"testing"
)

const (
	nmDLo       = -6 // fenetre balayee autour d'apres-boucles
	nmDHi       = 6
	nmAimD      = 2   // decalage de la visee validee par le modal (post-comptes + 2)
	nmCtrlBit   = 250 // controle bruit : offset profond fixe (comme l'instrument modal)
	nmClassMinN = 30
)

// nmHeaderCounts decode l'en-tete du record type 36 (polarite Ghidra du champ d, cf. le
// decodeur modal de production) jusqu'a la position APRES les comptes, et rend cette
// position, N (composantes) et P (cibles). ok=false si non-type-36, variante courte, ou
// horodatage bloc non resolu hors ligne. Contrairement a modalPostCountsBit, N'ECARTE PAS
// les records non-modaux : on en a besoin ici.
func nmHeaderCounts(pay []byte) (pos, nComp, nTargets int, ok bool) {
	br := NewBitReader(pay)
	br.Skip(2)
	if br.ReadBits(7) != 36 {
		return 0, 0, 0, false
	}
	if br.ReadBit() { // ref0 dom1 sonde
		w := 13
		if br.ReadBit() {
			w = 9
		}
		br.Skip(w + 2)
	}
	for range 2 { // ref1 dom8, ref2 dom7
		if br.ReadBit() {
			br.Skip(15)
		}
	}
	estCourt := br.ReadBit()
	estBloc := br.ReadBit()
	br.Skip(8)         // c : R(7)+R(1)
	if !br.ReadBit() { // d : R(1) + [si 0] R(5) — polarite Ghidra
		br.Skip(5)
	}
	if !br.ReadBit() { // e : R(1) + [si 0] R(2)
		br.Skip(2)
	}
	if br.ReadBit() { // f : R(1) + [si 1] R(32)
		br.Skip(32)
	}
	br.Skip(32) // g : variant_name
	br.Skip(2)  // i, j
	if estBloc {
		br.Skip(1)
		if br.ReadBit() {
			return 0, 0, 0, false
		}
	}
	if estCourt {
		return 0, 0, 0, false
	}
	var nCibles, nComps uint64
	if !br.ReadBit() { // toutVide ?
		if br.ReadBit() {
			nCibles = 1
		} else {
			nCibles = br.ReadBits(4)
		}
		if !br.ReadBit() {
			if br.ReadBit() {
				nComps = 1
			} else {
				nComps = br.ReadBits(4)
			}
		}
	}
	return br.BitPos(), int(nComps), int(nCibles), true
}

// nmLoopBits place un BitReader a postCounts et le fait avancer, au bit pres, a travers la
// boucle composantes (nComp * 35 bits) puis la boucle cibles (grammaire Ghidra ci-dessus),
// et rend la position de bit APRES les deux boucles.
func nmLoopBits(pay []byte, postCounts, nComp, nTargets int) int {
	br := NewBitReader(pay)
	br.Skip(postCounts)
	var comp [16]uint64 // R(2) memorise par composante (idx 0..15 : R(4) max)
	for i := 0; i < nComp; i++ {
		r2 := br.ReadBits(2)
		if i < len(comp) {
			comp[i] = r2
		}
		br.Skip(1)  // R(1)
		br.Skip(32) // R(32) (param_5==0)
	}
	mode := 4
	if nTargets == 1 {
		mode = 12
	}
	for i := 0; i < nTargets; i++ {
		br.Skip(4)        // R(4)
		if br.ReadBit() { // R(1) hit
			br.Skip(3) // R(3) (FUN_1406d310c(6)=3)
			var idx uint64
			if nComp < 3 {
				idx = br.ReadBits(1)
			} else {
				idx = br.ReadBits(4)
			}
			br.Skip(16) // R(16)
			cb := uint64(0)
			if idx < uint64(len(comp)) {
				cb = comp[idx]
			}
			w := mode
			if cb == 1 && mode > 6 {
				w = 6
			}
			br.Skip(3 * w) // FUN_140c1e924 : 3 champs de W bits
		}
	}
	return br.BitPos()
}

// nmAfterComposites part de la position apres les boucles et consomme les deux composites
// (cd5b8 + eff64, bit-exact) pour rendre le debut de la visee.
func nmAfterComposites(pay []byte, afterLoops int) int {
	br := NewBitReader(pay)
	br.Skip(afterLoops)
	lot1SkipCd5b8(br)
	lot1SkipEff64(br)
	return br.BitPos()
}

// nmClasse accumule la concentration de la visee lue au decalage d autour d'apres-composites.
type nmClasse struct {
	nom  string
	d    [nmDHi - nmDLo + 1]lot1AimConc
	ctrl lot1AimConc
	n    int
}

func (c *nmClasse) at(d int) *lot1AimConc { return &c.d[d-nmDLo] }

// best rend la meilleure concentration et le decalage qui l'atteint.
func (c *nmClasse) best() (int, float64) {
	bd, bv := nmDLo, -1.0
	for d := nmDLo; d <= nmDHi; d++ {
		if v := c.at(d).maxSousSeuil(); v > bv {
			bd, bv = d, v
		}
	}
	return bd, bv
}

func TestLot1ViseeNonModale(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	modal := &nmClasse{nom: "modal (0,0)"}
	compS := &nmClasse{nom: "composante-seule"}
	cibleS := &nmClasse{nom: "cible-seule"}
	deux := &nmClasse{nom: "cible+composante"}
	hist := map[[2]int]int{}
	var fauxPos lot1AimConc // FAUX POSITIF : le R(32) de la 1re composante (faible entropie)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 {
				continue
			}
			pc, nComp, nTargets, ok := nmHeaderCounts(pay)
			if !ok {
				continue
			}
			hist[[2]int{nComp, nTargets}]++
			cl := deux
			switch {
			case nComp == 0 && nTargets == 0:
				cl = modal
			case nComp > 0 && nTargets == 0:
				cl = compS
			case nComp == 0 && nTargets > 0:
				cl = cibleS
			}
			cl.n++
			// Origine = APRES-BOUCLES + d. Le modal (boucles vides) retrouve ainsi la visee
			// validee a post-comptes + 2 (afterLoops == post-comptes) : l'oracle DOIT y saturer
			// ~0.8. Le non-modal insere les boucles ; la queue composites->visee est la meme.
			afterL := nmLoopBits(pay, pc, nComp, nTargets)
			for d := nmDLo; d <= nmDHi; d++ {
				cl.at(d).add(pay, afterL+d)
			}
			cl.ctrl.add(pay, nmCtrlBit)
			if nComp > 0 { // R(32) de la 1re composante @ postCounts+3 : le faux positif de l'oracle
				fauxPos.add(pay, pc+3)
			}
		}
	}
	nmReport(t, []*nmClasse{modal, compS, cibleS, deux}, hist, &fauxPos)
}

func nmReport(t *testing.T, classes []*nmClasse, hist map[[2]int]int, fauxPos *lot1AimConc) {
	t.Helper()
	t.Logf("== visee des tirs NON-MODAUX : en-tete + 2 boucles + 2 composites decodes bit-exact (Ghidra) ==")
	t.Logf("distribution (N composantes, P cibles) : %s", nmTopHist(hist, 10))
	for _, cl := range classes {
		if cl.n == 0 {
			t.Logf("-- classe %-16s : 0 record", cl.nom)
			continue
		}
		bd, bv := cl.best()
		t.Logf("-- classe %-16s : %d records — visee@d=2 conc %s ; max fenetre d=%d conc %s ; controle %s",
			cl.nom, cl.n, nmPct(cl.at(nmAimD).maxSousSeuil()), bd, nmPct(bv), nmPct(cl.ctrl.maxSousSeuil()))
		t.Logf("     profil d %s", nmDProfil(cl))
		cl.at(nmAimD).log(t, "  visee@d=2")
	}
	fauxPos.log(t, "FAUX-POS R32")
	t.Logf("  ^ FAUX POSITIFS : le modal a des pics parasites hors d=2 (d=-6/-2 : champs de direction de l'en-tete)")
	t.Logf("    et le R(32) de composante concentre un axe au-dessus du controle : l'oracle N'EST PAS fiable dans cette zone dense")
	nmVerdict(t, classes)
}

func nmVerdict(t *testing.T, classes []*nmClasse) {
	t.Helper()
	modal := classes[0]
	ctrl := modal.ctrl.maxSousSeuil()
	mv := modal.at(nmAimD).maxSousSeuil()
	t.Logf("VERDICT (oracle : visee@d=2 conc >=0.7 ET >= 1.8x controle %s) — modal reference @d=2 conc %s (l'oracle FONCTIONNE sur le modal) :",
		nmPct(ctrl), nmPct(mv))
	totalNM, lisibleNM := 0, 0
	for _, cl := range classes[1:] {
		totalNM += cl.n
		if cl.n < nmClassMinN {
			t.Logf("  %-16s : %d records — effectif insuffisant (<%d), non statue", cl.nom, cl.n, nmClassMinN)
			continue
		}
		v := cl.at(nmAimD).maxSousSeuil()
		lisible := v >= 0.7 && v >= 1.8*ctrl
		if lisible {
			lisibleNM += cl.n
		}
		t.Logf("  %-16s : %d records — visee@d=2 conc %s vs controle %s — %s",
			cl.nom, cl.n, nmPct(v), nmPct(cl.ctrl.maxSousSeuil()), lot1Verdict(lisible))
	}
	t.Logf("COUVERTURE : %d / %d tirs non-modaux rendus lisibles par l'oracle (%.1f %%)",
		lisibleNM, totalNM, lot1Pct(lisibleNM, totalNM))
}

// TestLot1ViseeNonModaleScan — DIAGNOSTIC : scanne l'offset depuis APRES-BOUCLES (ote la
// variance bimodale 3*W) sur tous les records non-modaux. Si la visee etait a un offset
// constant apres les boucles, un pic net apparaitrait ; le composite variable + l'oracle
// aveugle font qu'AUCUN offset ne depasse le niveau du controle. Publie aussi la longueur
// decodee des composites (tres variable : 3 a ~44 bits selon le degat).
func TestLot1ViseeNonModaleScan(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	const oMax = 220
	var scan [oMax + 1]lot1AimConc
	var ctrl lot1AimConc
	compLen := map[int]int{}
	nm := 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 {
				continue
			}
			pc, nComp, nTargets, ok := nmHeaderCounts(pay)
			if !ok || (nComp == 0 && nTargets == 0) {
				continue
			}
			nm++
			afterLoops := nmLoopBits(pay, pc, nComp, nTargets)
			for o := 0; o <= oMax; o++ {
				scan[o].add(pay, afterLoops+o)
			}
			ctrl.add(pay, nmCtrlBit)
			compLen[nmAfterComposites(pay, afterLoops)-afterLoops]++
		}
	}
	t.Logf("== SCAN visee non-modale : %d records, offset depuis APRES-BOUCLES 0..%d ==", nm, oMax)
	best, bestV := 0, 0.0
	for o := 0; o <= oMax; o++ {
		if v := scan[o].maxSousSeuil(); v > bestV {
			best, bestV = o, v
		}
	}
	t.Logf("  MEILLEUR offset = %d : conc %s (controle %s) — %s", best, nmPct(bestV),
		nmPct(ctrl.maxSousSeuil()), lot1Verdict(bestV >= 0.7 && bestV >= 1.8*ctrl.maxSousSeuil()))
	t.Logf("  longueur composites decodee (apres-composites - apres-boucles) : %s", nmTopOff(compLen, 8))
}

// nmDProfil rend la ligne "d:concentration" du decalage nmDLo..nmDHi.
func nmDProfil(cl *nmClasse) string {
	out := ""
	for d := nmDLo; d <= nmDHi; d++ {
		if d > nmDLo {
			out += " "
		}
		out += itoa(d) + ":" + nmPct(cl.at(d).maxSousSeuil())
	}
	return out
}

// nmTopHist rend les k paires (N,P) les plus frequentes.
func nmTopHist(m map[[2]int]int, k int) string {
	type e struct {
		np [2]int
		n  int
	}
	var s []e
	for np, c := range m {
		s = append(s, e{np, c})
	}
	nmSortDesc(len(s), func(i, j int) bool { return s[j].n > s[i].n }, func(i, j int) { s[i], s[j] = s[j], s[i] })
	out := ""
	for i := 0; i < k && i < len(s); i++ {
		if i > 0 {
			out += " "
		}
		out += "(" + itoa(s[i].np[0]) + "," + itoa(s[i].np[1]) + "):" + itoa(s[i].n)
	}
	return out
}

// nmTopOff rend les k offsets les plus frequents d'un histogramme offset->compte.
func nmTopOff(m map[int]int, k int) string {
	type e struct{ o, n int }
	var s []e
	for o, c := range m {
		s = append(s, e{o, c})
	}
	nmSortDesc(len(s), func(i, j int) bool { return s[j].n > s[i].n }, func(i, j int) { s[i], s[j] = s[j], s[i] })
	out := ""
	for i := 0; i < k && i < len(s); i++ {
		if i > 0 {
			out += " "
		}
		out += itoa(s[i].o) + ":" + itoa(s[i].n)
	}
	return out
}

// nmSortDesc : tri par insertion generique (petit N ; evite d'importer sort pour deux usages).
func nmSortDesc(n int, less func(i, j int) bool, swap func(i, j int)) {
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if less(i, j) {
				swap(i, j)
			}
		}
	}
}

// nmPct formate une part 0..1 en pourcentage entier sans importer strconv/fmt.
func nmPct(v float64) string { return itoa(int(v*100+0.5)) + "%" }
