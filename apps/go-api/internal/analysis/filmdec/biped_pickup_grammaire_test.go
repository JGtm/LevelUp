package filmdec

import "testing"

// LA GRAMMAIRE DU TYPE 9, LUE DANS L'EXE (chaine independante du film).
//
// Table des descripteurs : le marcheur de liste FUN_14080a9d4 fait
// `desc = *(ctx+0x18) + 0x210 + type*8`. Le constructeur FUN_140e453b4 remplit cette table :
// +0x210 (type 0) = 0x144724f80 (damage_aftermath, concorde avec le chantier trame),
// +0x2b8 (type 21) = 0x144724e80 (unit_zoom, concorde aussi), et
// **+0x258 (type 9) = 0x144724e18** — le descripteur de biped_pickup.
//
// IDENTIFICATION DE PREMIERE MAIN : la vtable du type 9 est 0x143d0d758 ; son entree +0x08
// vaut 0x141164e10, l'UNIQUE fonction qui reference la chaine "biped_pickup" (0x143c97f98).
// Le nom n'est donc pas repris d'une note : il est lu dans le binaire.
//
// DOMAINES DES 3 REFERENCES (vtable+0x58 = 0x1410f92bc) :
//
//	index 0 -> `LEA EAX,[RDX+2]` = domaine 2  -> R(8)
//	index != 0 -> bloc froid partage 0x14232a4ba : index 1 -> domaine 8 (R(13)),
//	              index 2 -> domaine 7 (R(13))
//
// (Controle : la meme lecture sur le type 21 donne 4 / 8 / 7, exactement ce que le chantier
// trame avait etabli — la methode est validee sur un cas connu avant d'etre appliquee ici.)
//
// CHARGE (vtable+0x68 = FUN_141037828) :
//
//	R(3)                                   -> out[0]
//	R(1) porte ; si 1 : R(32) (FUN_14080d6f0, lecture 32 bits nue) -> out+4
//	                    sinon out+4 = 0xFFFFFFFF (sentinelle « absent »)
//	[0 bit] si out+4 est un handle valide (FUN_1405838f0 : ni 0 ni 0xFFFFFFFF), une
//	        RESOLUTION runtime (FUN_1407f21b4, table DAT_144eae7b8) ecrit out+8. Hors ligne
//	        on garde le handle brut : la resolution est une table de jeu, pas des bits.
//
// LONGUEUR MODALE ATTENDUE, en bits apres le champ de type :
// ref0 presente 1+8+2 = 11 · ref1 absente 1 · ref2 absente 1 · charge 3+1+32 = 36 ·
// bit de fin de liste 1  =  **50**. C'est EXACTEMENT le pic du scan empirique (etape 2),
// obtenu sans aucun ajustement : deux chaines independantes se ferment.

// bpkEvent est un evenement biped_pickup decode.
type bpkEvent struct {
	TimestampUS  uint64
	Ref0         uint64 // domaine 2, R(8) — le ramasseur presume
	Ref0Present  bool
	Ref1Present  bool
	Ref2Present  bool
	Kind         uint64 // R(3) de tete de charge
	Objet        uint32 // R(32) : le handle de l'objet ramasse (0xFFFFFFFF = absent)
	ObjetPresent bool
	// FinBit : position du bit qui suit le bit de fin de liste, donc le debut de la trame
	// quand l'evenement est seul dans sa liste.
	FinBit int
	// Suite : un autre evenement suit dans la meme liste.
	Suite bool
}

// bpkDecode consomme UN evenement type 9 a partir d'un lecteur place juste apres le champ de
// type, puis le bit de fin de liste. La grammaire est celle lue dans l'exe (ci-dessus).
func bpkDecode(br *BitReader) bpkEvent {
	var e bpkEvent
	e.Ref0, e.Ref0Present = bpkRef(br, 2)
	_, e.Ref1Present = bpkRef(br, 8)
	_, e.Ref2Present = bpkRef(br, 7)
	e.Kind = br.ReadBits(3)
	if br.ReadBit() {
		e.Objet = uint32(br.ReadBits(32))
		e.ObjetPresent = true
	} else {
		e.Objet = 0xFFFFFFFF
	}
	e.Suite = br.ReadBit()
	e.FinBit = br.BitPos()
	return e
}

// bpkGramStats accumule ce que le decodage rend, et le publie.
type bpkGramStats struct {
	n, seuls, exact                       int
	ref0Abs, ref1, ref2, suites, objetAbs int
	ref0Hist, kindHist, objHist, finBits  map[uint64]int
	temoins                               map[int]int
}

func (g *bpkGramStats) init() {
	g.ref0Hist, g.kindHist = map[uint64]int{}, map[uint64]int{}
	g.objHist, g.finBits = map[uint64]int{}, map[uint64]int{}
	g.temoins = map[int]int{-3: 0, -2: 0, -1: 0, 1: 0, 2: 0, 3: 0}
}

func (g *bpkGramStats) compte(e bpkEvent) {
	g.n++
	if e.Ref0Present {
		g.ref0Hist[e.Ref0]++
	} else {
		g.ref0Abs++
	}
	if e.Ref1Present {
		g.ref1++
	}
	if e.Ref2Present {
		g.ref2++
	}
	g.kindHist[e.Kind]++
	if e.ObjetPresent {
		g.objHist[uint64(e.Objet)]++
	} else {
		g.objetAbs++
	}
	if e.Suite {
		g.suites++
	}
}

// rapport publie les histogrammes et le juge ; rend le PIRE taux des temoins decales.
func (g *bpkGramStats) rapport(t *testing.T) float64 {
	t.Helper()
	t.Logf("ref0 (domaine 2, R(8)) : absente x%d · %d index distincts : %s",
		g.ref0Abs, len(g.ref0Hist), bpkTop(g.ref0Hist, 12))
	t.Logf("ref1 (domaine 8) presente x%d · ref2 (domaine 7) presente x%d · listes multiples x%d",
		g.ref1, g.ref2, g.suites)
	t.Logf("charge R(3) : %s", bpkTop(g.kindHist, 8))
	t.Logf("charge R(32) (identifiant d'objet) : absente x%d · %d valeurs distinctes sur %d presentes : %s",
		g.objetAbs, len(g.objHist), g.n-g.objetAbs, bpkTop(g.objHist, 10))
	t.Logf("longueur totale de l'evenement (bits apres le type, bit de fin de liste compris) : %s",
		bpkTop(g.finBits, 8))
	t.Logf("JUGE : trames EXACTES %d / %d evenements seuls (%.1f %%)",
		g.exact, g.seuls, bpkPct(g.exact, g.seuls))
	pire := 0.0
	for _, d := range []int{-3, -2, -1, 1, 2, 3} {
		p := bpkPct(g.temoins[d], g.seuls)
		t.Logf("  TEMOIN %+d bit(s) : %.1f %%", d, p)
		if p > pire {
			pire = p
		}
	}
	return pire
}

// TestBipedPickupGrammaire — ETAPE 3. Decode le type 9 avec la grammaire de l'exe et la JUGE
// par l'oracle calibre, contre des temoins decales.
//
// SEUILS ECRITS AVANT LA MESURE (l'oracle vaut 93 % au cadrage vrai et 0,0 % a +/-1..3 sur ce
// film, cf. TestBipedPickupCalibration) :
//
//	G1 — trames EXACTES apres l'evenement, sur les paquets a evenement unique : >= 70 %.
//	G2 — chaque temoin (-3, -2, -1, +1, +2, +3 bits) : <= 10 %.
//	G3 — le verdict est G1 ET G2. En dessous, la grammaire est REFUTEE sur ce film.
func TestBipedPickupGrammaire(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()
	cfg := bpkCfg(f.idLow)

	var g bpkGramStats
	g.init()
	bpkEachEvent(t, f, func(typ int, pay []byte, snap WorldSnapshot, tsUS uint64) {
		if typ != bpkTypePickup {
			return
		}
		br := NewBitReader(pay)
		br.Skip(bpkHeaderBits)
		e := bpkDecode(br)
		e.TimestampUS = tsUS
		g.compte(e)
		if e.Suite {
			return // liste multiple : la trame ne commence pas ici
		}
		g.seuls++
		g.finBits[uint64(e.FinBit-bpkHeaderBits)]++
		if ok, _ := bpkTrameExacte(f.reg, snap, pay, e.FinBit, cfg); ok {
			g.exact++
		}
		for d := range g.temoins {
			p := e.FinBit + d
			if p < 0 || p+8 > len(pay)*8 {
				continue
			}
			if ok, _ := bpkTrameExacte(f.reg, snap, pay, p, cfg); ok {
				g.temoins[d]++
			}
		}
	})
	if g.n == 0 {
		t.Skip("aucun type 9 : rien a juger")
	}
	t.Logf("== GRAMMAIRE DE L'EXE sur %d evenements type 9 (%s) ==", g.n, f.dir)
	pire := g.rapport(t)
	exact, seuls := g.exact, g.seuls
	t.Logf("VERDICT G1 (>= 70 %%) : %s · G2 (temoins <= 10 %%) : %s · G3 : %s",
		bpkVerdict(bpkPct(exact, seuls) >= 70), bpkVerdict(pire <= 10),
		bpkVerdict(bpkPct(exact, seuls) >= 70 && pire <= 10))
}

// TestBipedPickupPlafondEvenement — LE CONTROLE DE PLAFOND, indispensable pour lire le
// verdict de TestBipedPickupGrammaire.
//
// Un paquet qui PORTE un evenement n'est pas un paquet ordinaire : c'est un paquet ou il
// vient de se passer quelque chose, donc souvent un paquet a records NEW (creation
// d'entite), que le decodeur de trame traverse moins bien (default-state non bit-exact).
// Comparer le taux de trames exactes des paquets type 9 au 93 % des trames PURES serait donc
// injuste. On mesure ici le PLAFOND REEL sur une famille dont la grammaire est DEJA PROUVEE
// par le chantier trame : `unit_zoom` (type 21, octet 0xCA — refs domaines 4/8/7, charge
// R(2)). Meme oracle, meme film, meme preparation.
//
// SEUIL ECRIT AVANT : si le type 9 atteint le plafond du type 21 a moins de 10 points, sa
// grammaire est aussi bonne que celle du type 21 — c'est-a-dire juste. Si elle est LOIN en
// dessous, quelque chose manque a la grammaire du type 9.
func TestBipedPickupPlafondEvenement(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()
	cfg := bpkCfg(f.idLow)

	var seuls, exact int
	longueurs := map[uint64]int{}
	temoins := map[int]int{-3: 0, -2: 0, -1: 0, 1: 0, 2: 0, 3: 0}
	for c := 1; c <= f.chunks; c++ {
		data, err := ReadFilmChunk(f.dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		snap := bpkChunkWorld(f.reg, data, pks, cfg)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xCA {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(1)
			if !br.ReadBit() || br.ReadBits(7) != 21 {
				continue // type 20 (incident) : charge variable, hors controle
			}
			bpkRef(br, 4)
			bpkRef(br, 8)
			bpkRef(br, 7)
			br.Skip(2) // charge du type 21 : R(2), niveau de lunette + 1
			if br.ReadBit() {
				continue // liste multiple
			}
			fin := br.BitPos()
			seuls++
			longueurs[uint64(fin-bpkHeaderBits)]++
			if ok, _ := bpkTrameExacte(f.reg, snap, pay, fin, cfg); ok {
				exact++
			}
			for d := range temoins {
				p := fin + d
				if p < 0 || p+8 > len(pay)*8 {
					continue
				}
				if ok, _ := bpkTrameExacte(f.reg, snap, pay, p, cfg); ok {
					temoins[d]++
				}
			}
		}
	}
	if seuls == 0 {
		t.Skip("aucun unit_zoom seul : pas de plafond mesurable sur ce film")
	}
	t.Logf("== PLAFOND : unit_zoom (type 21, grammaire PROUVEE) sur %s ==", f.dir)
	t.Logf("evenements seuls : %d · longueur totale : %s", seuls, bpkTop(longueurs, 6))
	t.Logf("PLAFOND — trames EXACTES au cadrage vrai : %d / %d (%.1f %%)",
		exact, seuls, bpkPct(exact, seuls))
	for _, d := range []int{-3, -2, -1, 1, 2, 3} {
		t.Logf("  TEMOIN %+d bit(s) : %.1f %% (plancher de bruit de l'oracle sur paquets a evenement)",
			d, bpkPct(temoins[d], seuls))
	}
}

// TestBipedPickupEchecs — DIAGNOSTIC DES ECHECS. Un taux de trames exactes sous le plafond
// admet deux explications, et elles ne se valent pas :
//
//	(A) la grammaire est fausse pour ces paquets — alors un AUTRE cadrage, proche, marche ;
//	(B) c'est le decodeur de trame qui cale sur ce paquet — alors AUCUN cadrage ne marche.
//
// On tranche en balayant une fenetre large autour du cadrage retenu et en publiant, pour les
// paquets en echec, l'ensemble des offsets exacts trouves.
//
// SEUIL ECRIT AVANT : si >= 80 % des paquets en echec n'ont AUCUN offset exact dans la
// fenetre [10, 120], l'explication (B) est retenue et la grammaire n'est pas en cause.
func TestBipedPickupEchecs(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()
	cfg := bpkCfg(f.idLow)

	const fMin, fMax = 10, 120
	var seuls, echecs, echecsSansCadrage int
	autres := map[uint64]int{}
	bpkEachEvent(t, f, func(typ int, pay []byte, snap WorldSnapshot, _ uint64) {
		if typ != bpkTypePickup {
			return
		}
		br := NewBitReader(pay)
		br.Skip(bpkHeaderBits)
		e := bpkDecode(br)
		if e.Suite {
			return
		}
		seuls++
		if ok, _ := bpkTrameExacte(f.reg, snap, pay, e.FinBit, cfg); ok {
			return
		}
		echecs++
		trouve := false
		for off := fMin; off <= fMax; off++ {
			p := bpkHeaderBits + off
			if p+8 > len(pay)*8 {
				break
			}
			if ok, _ := bpkTrameExacte(f.reg, snap, pay, p, cfg); ok {
				trouve = true
				autres[uint64(off)]++
			}
		}
		if !trouve {
			echecsSansCadrage++
		}
	})
	if seuls == 0 {
		t.Skip("aucun type 9 seul")
	}
	t.Logf("== DIAGNOSTIC DES ECHECS sur %s ==", f.dir)
	t.Logf("evenements seuls %d · en echec au cadrage 50 : %d", seuls, echecs)
	t.Logf("parmi les echecs : AUCUN offset exact dans [%d,%d] : %d / %d (%.1f %%)",
		fMin, fMax, echecsSansCadrage, echecs, bpkPct(echecsSansCadrage, echecs))
	t.Logf("offsets exacts trouves chez les autres echecs : %s", bpkTop(autres, 12))
	t.Logf("VERDICT (>= 80 %% des echecs sans aucun cadrage => c'est le decodeur de trame, pas la grammaire) : %s",
		bpkVerdict(bpkPct(echecsSansCadrage, echecs) >= 80))
}

// TestBipedPickupLargeurRef0 — LA LARGEUR DU DOMAINE 2 EST UNE VALEUR DE RUNTIME, PAS UNE
// CONSTANTE DU FORMAT. Le lecteur de reference de l'exe (FUN_1406d3140) lit sa largeur dans
// la table DAT_1451f98d0/d4 indexee par le domaine, peuplee au chargement de carte :
//
//	si le bit de configuration du paquet vaut 1 : base = DAT_1451f98d0[dom*2] et
//	   largeur = FUN_1406d310c(DAT_1451f98d4[dom*2]) ; sinon largeur globale de repli.
//	seul le domaine 1 porte une sonde R(1) qui bascule sur un second couple (0x1451f98f0/f4).
//	Puis, TOUJOURS, R(2) de generation. La reference vaut (gen<<30) | (base + index).
//
// C'est la MEME table que celle qui donne FrameConfig.IDLowBits — et sur ce film IDLowBits
// se calibre a 9, pas a la valeur par defaut 13. Il faut donc calibrer la largeur du
// domaine 2 sur le film au lieu de la supposer.
//
// SEUIL ECRIT AVANT LA MESURE : la largeur retenue est celle qui maximise le taux de trames
// exactes ; elle n'est acceptee que si ce taux atteint au moins 80 % du PLAFOND mesure sur
// unit_zoom (grammaire prouvee) et si les largeurs voisines restent sous 20 %.
// bpkEssaieLargeur decode UN evenement type 9 en supposant la largeur w pour l'index de
// ref0, puis soumet le cadrage obtenu a l'oracle. Alimente les compteurs du balayage.
func bpkEssaieLargeur(f bpkFilm, snap WorldSnapshot, pay []byte, w int, cfg FrameConfig,
	exact, seuls []int, idx map[uint64]int) {
	br := NewBitReader(pay)
	br.Skip(bpkHeaderBits)
	if !br.ReadBit() {
		return // ref0 absente : la largeur ne change rien
	}
	i0 := br.ReadBits(uint(w))
	br.Skip(2) // generation
	if _, p1 := bpkRef(br, 8); p1 {
		return
	}
	if _, p2 := bpkRef(br, 7); p2 {
		return
	}
	br.Skip(3) // la classe R(3)
	if br.ReadBit() {
		br.Skip(32) // l'identifiant d'objet
	}
	if br.ReadBit() {
		return // liste multiple : la trame ne commence pas ici
	}
	seuls[w]++
	idx[i0]++
	if ok, _ := bpkTrameExacte(f.reg, snap, pay, br.BitPos(), cfg); ok {
		exact[w]++
	}
}

func TestBipedPickupLargeurRef0(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()
	cfg := bpkCfg(f.idLow)

	const wMin, wMax = 4, 14
	exact := make([]int, wMax+1)
	seuls := make([]int, wMax+1)
	idx := make([]map[uint64]int, wMax+1)
	for w := wMin; w <= wMax; w++ {
		idx[w] = map[uint64]int{}
	}
	bpkEachEvent(t, f, func(typ int, pay []byte, snap WorldSnapshot, _ uint64) {
		if typ != bpkTypePickup {
			return
		}
		for w := wMin; w <= wMax; w++ {
			bpkEssaieLargeur(f, snap, pay, w, cfg, exact, seuls, idx[w])
		}
	})
	t.Logf("== LARGEUR DU DOMAINE 2 (index de ref0), balayee contre l'oracle · %s ==", f.dir)
	best, bestPct := wMin, -1.0
	for w := wMin; w <= wMax; w++ {
		p := bpkPct(exact[w], seuls[w])
		t.Logf("  R(%2d) : longueur d'evenement %3d bits · trames EXACTES %5.1f %% (%d/%d) · %d index distincts : %s",
			w, 42+w, p, exact[w], seuls[w], len(idx[w]), bpkTop(idx[w], 6))
		if p > bestPct {
			best, bestPct = w, p
		}
	}
	voisin := 0.0
	for _, d := range []int{-2, -1, 1, 2} {
		if w := best + d; w >= wMin && w <= wMax {
			if p := bpkPct(exact[w], seuls[w]); p > voisin {
				voisin = p
			}
		}
	}
	t.Logf("RETENU : R(%d) — %.1f %% de trames exactes · meilleure largeur voisine : %.1f %%",
		best, bestPct, voisin)
	t.Logf("VERDICT (retenue >= 74 %% soit 80 %% du plafond unit_zoom 92,7 %%, voisines < 20 %%) : %s",
		bpkVerdict(bestPct >= 74 && voisin < 20))
}
