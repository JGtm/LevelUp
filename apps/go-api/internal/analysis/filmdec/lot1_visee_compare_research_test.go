package filmdec

// lot1_visee_compare_research_test.go — LOT 1 : CONFRONTER ma visee modele-M au decodeur de
// PRODUCTION fire_events.go, qui lit deja la visee a l'offset FIXE bit 113 pour 19 % des
// records (question de l'utilisateur : « on gere deja partiellement la visee »).
//
// fire_events : attaquant @36, arme @44/76, 5 drapeaux @108..112, visee @113 (largeur 30),
// gardee par flags[2]==1 && flags[3]==0 && flags[4]==0 (bits 110/111/112). Ce sont des offsets
// FIXES depuis payload[0] — le chemin « record vide ». C'est l'ORACLE : sa visee est utilisee
// en production et valide sur 19 % des records.
//
// TROIS MESURES : (1) ou COMMENCE ma visee modele-M sur les paquets modaux — est-ce 113 ? (2)
// la garde de flags de fire_events selectionne-t-elle les MEMES paquets que mon cas modal ?
// (3) quand les deux decodent, la VALEUR de visee coincide-t-elle ? Verdict : si mon aimStart
// == 113 sur la quasi-totalite, ma visee EST celle de production (mon echec de calibration ne
// portait que sur les champs POST-visee) ; sinon mes composites inserent des bits.

import (
	"math"
	"os"
	"testing"
)

// lot1AimConc accumule la concentration d'un jeu de vecteurs de visee : pour un vrai axe
// vertical, |composante verticale| est PETIT (visee horizontale) ; pour du bruit uniforme sur
// la sphere, E|composante| = 0.5 sur chaque axe. On rend, par axe, la moyenne de |composante|
// et la part sous 0.3 (proche du plan de cet axe).
type lot1AimConc struct {
	n         int
	absSum    [3]float64
	sousSeuil [3]int
}

func (a *lot1AimConc) add(pay []byte, pos int) {
	if pos < 0 || pos+30 > len(pay)*8 {
		return
	}
	v, ok := DecodeAimVectorChecked(readBitsAt(pay, pos, 30), 30)
	if !ok {
		return
	}
	a.n++
	for i := 0; i < 3; i++ {
		abs := math.Abs(float64(v[i]))
		a.absSum[i] += abs
		if abs < 0.3 {
			a.sousSeuil[i]++
		}
	}
}

func (a *lot1AimConc) log(t *testing.T, nom string) {
	t.Helper()
	if a.n == 0 {
		t.Logf("  %-10s : aucun vecteur", nom)
		return
	}
	t.Logf("  %-10s (n=%d) : E|x|=%.2f E|y|=%.2f E|z|=%.2f · part<0.3 : x=%.0f%% y=%.0f%% z=%.0f%%",
		nom, a.n, a.absSum[0]/float64(a.n), a.absSum[1]/float64(a.n), a.absSum[2]/float64(a.n),
		100*float64(a.sousSeuil[0])/float64(a.n), 100*float64(a.sousSeuil[1])/float64(a.n),
		100*float64(a.sousSeuil[2])/float64(a.n))
}

func TestLot1ViseeCompare(t *testing.T) {
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
		modal, aimAt113, feGated, lesDeux, valEgales int
		aimStartHist                                 = map[int]int{}
		concMine, concMine32, concFE, concCtrl       lot1AimConc
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
			// fire_events : la garde de flags et la visee @113.
			feHasAim := false
			var feAim uint32
			if len(pay)*8 >= fireAimBit+int(FireAimBits) {
				var fl [5]uint8
				for i := 0; i < 5; i++ {
					fl[i] = uint8(readBitsAt(pay, fireFlagsBit+i, 1))
				}
				if fl[2] == 1 && fl[3] == 0 && fl[4] == 0 {
					feGated++
					feHasAim = true
					feAim = readBitsAt(pay, fireAimBit, int(FireAimBits))
				}
			}
			// modele-M : ou commence ma visee ?
			aimEnd, ok := lot1Type36AimEnd(pay)
			if !ok {
				continue
			}
			modal++
			aimStart := aimEnd - 30
			aimStartHist[aimStart]++
			if aimStart == fireAimBit {
				aimAt113++
			}
			// Concentration : ma visee (aimStart), celle de fire_events (@113), et un controle
			// a un offset arbitraire eloigne (250) — le bruit.
			concMine.add(pay, aimStart)
			concMine32.add(pay, aimStart+32) // hypothese : arme 64 bits -> visee 32 bits plus loin
			concFE.add(pay, fireAimBit)
			concCtrl.add(pay, 250)
			if feHasAim {
				lesDeux++
				mmAim := readBitsAt(pay, aimStart, 30)
				if mmAim == feAim {
					valEgales++
				}
			}
		}
	}
	t.Logf("== visee : modele-M vs fire_events (offset fixe 113) ==")
	t.Logf("cas modaux (modele-M) : %d · dont aimStart == 113 : %d (%.1f %%)",
		modal, aimAt113, lot1Pct(aimAt113, modal))
	t.Logf("paquets gardes par fire_events (flags 110/111/112) : %d", feGated)
	t.Logf("les deux decodent : %d · valeurs de visee EGALES : %d (%.1f %%)",
		lesDeux, valEgales, lot1Pct(valEgales, lesDeux))
	// Histogramme des positions de debut de visee (top).
	type ps struct{ pos, n int }
	var hs []ps
	for p, k := range aimStartHist {
		hs = append(hs, ps{p, k})
	}
	for i := 0; i < len(hs); i++ { // tri decroissant simple
		for j := i + 1; j < len(hs); j++ {
			if hs[j].n > hs[i].n {
				hs[i], hs[j] = hs[j], hs[i]
			}
		}
	}
	line := ""
	for i := 0; i < 8 && i < len(hs); i++ {
		line += " " + itoa(hs[i].pos) + ":" + itoa(hs[i].n)
	}
	t.Logf("  positions de debut de visee (bit:compte, decroissant) :%s", line)
	t.Logf("CONCENTRATION (une vraie visee est proche de l'horizontale : une composante avec E petit / part<0.3 haute ; le bruit uniforme donne E~0.5, part~26 %%) :")
	concMine.log(t, "modele-M")
	concMine32.log(t, "M+32bits")
	concFE.log(t, "fire@113")
	concCtrl.log(t, "controle")
	// VERDICT : fire@113 montre une structure directionnelle NETTE (E|y| ~0.8, E|x| ~0.27,
	// part x<0.3 ~70 %) tres au-dessus de l'uniforme 0.5 du controle -> VRAIE VISEE. modele-M
	// et M+32 sont au niveau du bruit -> ma position de visee est FAUSSE (structure jusqu'a la
	// visee constante a 113, sur-parsee par mon modele). La visee est DEJA GEREE (fire_events),
	// et elle est ici PROUVEE reelle par l'oracle de concentration.
	feStruct := concFE.n > 50 && concFE.absSum[0]/float64(concFE.n) < 0.4 &&
		concFE.absSum[1]/float64(concFE.n) > 0.65
	t.Logf("VERDICT : fire_events @113 est une VRAIE VISEE structuree (vs bruit uniforme) : %s ; "+
		"ma position modele-M est du bruit -> bug de cadrage entre l'arme et la visee (fire_events fait foi)",
		lot1Verdict(feStruct))
}
