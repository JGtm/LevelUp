package filmdec

// lot1_visee_ghidra_research_test.go — LOT 1 : RECALER LA VISEE SUR LA GRAMMAIRE GHIDRA.
//
// L'agent Ghidra « percer le cadrage de la visee type 105 » a trace FUN_14080C1F8 (le lecteur
// de charge du record de tir/degat) INSTRUCTION PAR INSTRUCTION. Sa grammaire de charge, dans
// l'ordre reel :
//
//	a variant        R(1)
//	b bloc-sup       R(1)
//	c attaquant      R(7)+R(1)                     (FUN_141fcf670)
//	d                R(1) + [si 0] R(5)            (FUN_1407f2034)   <- POLARITE
//	e                R(1) + [si 0] R(2)            (FUN_1406d00ec)
//	f arme famille   R(1) + [si 1] R(32)           (FUN_14080d69c)
//	g arme variante  R(32)                          (FUN_14080dec4)
//	i,j              R(1),R(1)
//	k                si bloc-sup : R(1)+R(1)
//	[si variant==0]  boucles composantes/cibles, puis visee R(30)
//
// RESULTAT (2 temoins, confirme) : DEUX corrections cumulees percent la visee bien au-dela des
// 19 % que fire_events couvre.
//  1. POLARITE d : saute R(5) si garde==0 (Ghidra), pas ==1. Avec elle, mon decodage d'en-tete
//     atterrit a une position POST-COMPTES stable de 111 sur 100 % des paquets a visee vide.
//  2. LES DEUX COMPOSITES SONT PARASITES dans le chemin modal : fire_events place la visee a
//     113 = post-comptes(111) + 2. Les 2 bits = les 2 derniers drapeaux ; PAS de lecture
//     cd5b8/eff64 avant la visee. La vraie visee est donc a post-comptes + 2.
//
// PREUVE par l'oracle de concentration (une vraie visee unitaire sature UN axe : part<0.3 tres
// haute ; le bruit uniforme reste ~26 %). Sur les paquets que fire_events NE couvre PAS (177 et
// 348 selon le temoin), la visee a post-comptes+2 sature un axe a 97 % — au-dessus meme de
// fire@113 (76-88 %), tres au-dessus du controle (12-24 %). COUVERTURE : de 33 -> 210 paquets
// (x6,4) et 143 -> 491 (x3,4) sur les deux temoins. Le plafond 19 % TOMBE pour le cas modal.
//
// L'agent Ghidra avait raison sur la structure ET sur sa reserve (les boucles cibles/composantes
// non vides restent runtime-width) : on ne perce QUE le cas modal (0 cible, 0 composante), qui
// est justement le tir "propre". CET INSTRUMENT mesure les deux corrections cote a cote (polarite
// buggee vs Ghidra ; visee post-comptes +1/+2/+3) et prouve l'extension sur le sous-ensemble
// hors-fire_events.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule.

import (
	"os"
	"testing"
)

// lot1HeaderPostCounts decode l'en-tete du record type 36/105 (framing modele-M : 2 bits
// config/continuation puis R(7) type) jusqu'a la position APRES les comptes cibles/composantes,
// pour le cas modal (0 cible, 0 composante) — c.-a-d. AVANT les deux lecteurs composites. flipD
// choisit la polarite du champ d : false = mon modele actuel (saute R(5) si garde==1) ; true =
// grammaire Ghidra (saute R(5) si garde==0). Rend la position de bit, ou ok=false si le paquet
// n'est pas un type 36 modal. C'est le point de mesure qui isole les composites : fire_events
// place la visee 5 bits (les drapeaux) apres un point equivalent, sans aucun composite.
func lot1HeaderPostCounts(pay []byte, flipD bool) (int, bool) {
	br := NewBitReader(pay)
	br.Skip(2)
	if br.ReadBits(7) != 36 {
		return 0, false
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
	br.Skip(8) // c : R(7)+R(1)
	// d : R(1) + [si 0] R(5) (Ghidra) vs mon modele [si 1] R(5).
	if flipD {
		if !br.ReadBit() {
			br.Skip(5)
		}
	} else {
		if br.ReadBit() {
			br.Skip(5)
		}
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
			return 0, false // horodatage bloc non resolu
		}
	}
	if estCourt {
		return 0, false
	}
	var nCibles, nComp uint64
	if !br.ReadBit() {
		if br.ReadBit() {
			nCibles = 1
		} else {
			nCibles = br.ReadBits(4)
		}
		if !br.ReadBit() {
			if br.ReadBit() {
				nComp = 1
			} else {
				nComp = br.ReadBits(4)
			}
		}
	}
	if nCibles != 0 || nComp != 0 {
		return 0, false
	}
	return br.BitPos(), true // position APRES les comptes, AVANT les composites
}

// lot1HeaderAimStart ajoute les deux lecteurs composites (cd5b8, eff64) a la position
// post-comptes : c'est la position de visee du modele-M complet.
func lot1HeaderAimStart(pay []byte, flipD bool) (int, bool) {
	pos, ok := lot1HeaderPostCounts(pay, flipD)
	if !ok {
		return 0, false
	}
	br := NewBitReader(pay)
	br.Skip(pos)
	lot1SkipCd5b8(br)
	lot1SkipEff64(br)
	return br.BitPos(), true
}

func TestLot1ViseeGhidra(t *testing.T) {
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
	var (
		modalBug, modalFix              int
		at113Bug, at113Fix              int
		feEmpty, feEmptyFixModal        int
		feEmptyFixAt113                 int
		concBug, concFix                lot1AimConc
		concFE, concCtrl                lot1AimConc
		concPost1, concPost2, concPost3 lot1AimConc // visee lue a post-comptes +1/+2/+3
		concPost2FE, concPost2NonFE     lot1AimConc // post-comptes+2 : couverts / NON couverts par fire_events
		gainAim                         int         // paquets a visee gagnee (modaux hors fire_events)
		startBug                        = map[int]int{}
		startFix                        = map[int]int{}
		postFE                          = map[int]int{} // position post-comptes des paquets a visee vide
	)
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
			// Ancre fire_events : paquet a visee lisible (chemin record vide).
			isFEEmpty := false
			if len(pay)*8 >= fireAimBit+int(FireAimBits) {
				var fl [5]uint8
				for i := 0; i < 5; i++ {
					fl[i] = uint8(readBitsAt(pay, fireFlagsBit+i, 1))
				}
				if fl[2] == 1 && fl[3] == 0 && fl[4] == 0 {
					isFEEmpty = true
					feEmpty++
					concFE.add(pay, fireAimBit)
					concCtrl.add(pay, 250)
				}
			}
			if s, ok := lot1HeaderAimStart(pay, false); ok { // polarite buggee
				modalBug++
				startBug[s]++
				if s == fireAimBit {
					at113Bug++
				}
				concBug.add(pay, s)
			}
			if s, ok := lot1HeaderAimStart(pay, true); ok { // polarite Ghidra
				modalFix++
				startFix[s]++
				if s == fireAimBit {
					at113Fix++
				}
				concFix.add(pay, s)
				if isFEEmpty {
					feEmptyFixModal++
					if s == fireAimBit {
						feEmptyFixAt113++
					}
				}
			}
			// POSITION POST-COMPTES (avant les composites) : isole la sur-consommation des
			// lecteurs composites. Sur les paquets a visee vide, fire_events place la visee a 113 ;
			// si mon post-comptes est ~108, la visee est juste apres (drapeaux), sans composites.
			if s, ok := lot1HeaderPostCounts(pay, true); ok {
				concPost1.add(pay, s+1)
				concPost2.add(pay, s+2)
				concPost3.add(pay, s+3)
				if isFEEmpty {
					postFE[s]++
					concPost2FE.add(pay, s+2)
				} else {
					gainAim++
					concPost2NonFE.add(pay, s+2)
				}
			}
		}
	}
	t.Logf("== recalage visee sur la grammaire Ghidra (champ d) ==")
	t.Logf("cas modaux : polarite BUGGEE %d (aimStart==113 : %d, %.1f %%) · polarite GHIDRA %d (aimStart==113 : %d, %.1f %%)",
		modalBug, at113Bug, lot1Pct(at113Bug, modalBug), modalFix, at113Fix, lot1Pct(at113Fix, modalFix))
	t.Logf("ANCRE fire_events (paquets a visee vide) : %d ; dont decodes modaux par la polarite Ghidra : %d ; dont aimStart==113 : %d (%.1f %%)",
		feEmpty, feEmptyFixModal, feEmptyFixAt113, lot1Pct(feEmptyFixAt113, feEmptyFixModal))
	t.Logf("  positions de debut de visee — BUGGEE : %s", lot1TopPos(startBug, 6))
	t.Logf("  positions de debut de visee — GHIDRA : %s", lot1TopPos(startFix, 6))
	t.Logf("CONCENTRATION (vraie visee : UN axe horizontal, part<0.3 tres haute ; bruit uniforme ~26 %%) :")
	concBug.log(t, "d buggee")
	concFix.log(t, "d Ghidra")
	concPost1.log(t, "post-cnt+1")
	concPost2.log(t, "post-cnt+2")
	concPost3.log(t, "post-cnt+3")
	concPost2FE.log(t, "+2 fire")    // sous-ensemble deja couvert par fire_events
	concPost2NonFE.log(t, "+2 GAIN") // sous-ensemble NON couvert : la vraie preuve d'extension
	concFE.log(t, "fire@113")
	concCtrl.log(t, "controle")
	t.Logf("EXTENSION VISEE : fire_events couvre %d paquets (visee vide) ; post-comptes+2 en couvre %d (tout le modal) ; GAIN sur des paquets NON couverts par fire_events : %d",
		feEmpty, modalFix, gainAim)
	t.Logf("  position POST-COMPTES des paquets a visee vide (attendu ~108 si composites parasites) : %s",
		lot1TopPos(postFE, 6))
	// DISCRIMINANT ROBUSTE AU TEMOIN : l'axe le plus concentre (max part<0.3). fire@113, vraie
	// visee, sature un axe (76-88 %) ; le controle plafonne au niveau uniforme. LE JUGE DE
	// L'EXTENSION est concPost2NonFE : la visee lue a post-comptes+2 sur les paquets que
	// fire_events NE couvre PAS. Si cet axe sature au niveau de fire@113, la visee est acquise
	// bien au-dela des 19 % — les deux composites (cd5b8/eff64) etaient parasites dans le chemin
	// modal, la vraie visee est 2 bits (les 2 derniers drapeaux) apres les comptes.
	oracleFE := concFE.maxSousSeuil()
	gainConc := concPost2NonFE.maxSousSeuil()
	t.Logf("axe le plus concentre (max part<0.3) — fire@113 : %.0f%% · +2 GAIN (hors fire) : %.0f%% · post+1 : %.0f%% · post+3 : %.0f%% · controle : %.0f%%",
		100*oracleFE, 100*gainConc, 100*concPost1.maxSousSeuil(), 100*concPost3.maxSousSeuil(), 100*concCtrl.maxSousSeuil())
	extensionAcquise := concPost2NonFE.n > 50 && gainConc >= 0.7 && gainConc >= 1.8*concCtrl.maxSousSeuil()
	t.Logf("VERDICT EXTENSION (visee a post-comptes+2 sur les paquets NON couverts par fire_events atteint la concentration de la vraie visee) : %s",
		lot1Verdict(extensionAcquise))
}

// maxSousSeuil rend la part<0.3 de l'axe le PLUS concentre. Discriminant robuste au temoin : une
// vraie visee sature un axe (l'axe vertical du repere du film, qui varie selon la carte), le bruit
// uniforme plafonne partout au niveau ~26 %.
func (a *lot1AimConc) maxSousSeuil() float64 {
	if a.n == 0 {
		return 0
	}
	m := 0
	for i := 0; i < 3; i++ {
		if a.sousSeuil[i] > m {
			m = a.sousSeuil[i]
		}
	}
	return float64(m) / float64(a.n)
}

// lot1TopPos rend les k positions les plus frequentes d'un histogramme position->compte.
func lot1TopPos(m map[int]int, k int) string {
	type pc struct{ p, n int }
	var s []pc
	for p, n := range m {
		s = append(s, pc{p, n})
	}
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j].n > s[i].n {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
	out := ""
	for i := 0; i < k && i < len(s); i++ {
		if i > 0 {
			out += " "
		}
		out += itoa(s[i].p) + ":" + itoa(s[i].n)
	}
	return out
}
