package filmdec

// playergameevent_0xe9_research_test.go — LOT 1 : decodage de PlayerGameEventSmall (0xE9,
// type 82 ; ~923 k evenements sur le corpus, jamais decode). QUESTION : cet evenement
// porte-t-il, COTE TIREUR, des confirmations de touche / degat inflige attribuees au
// JOUEUR (+ arme), qui couvriraient les armes explosives que damage_aftermath rate ?
//
// GRAMMAIRE, LUE DANS L'EXE (HaloInfinite.exe, image base 140000000). La table de handlers
// (0x144724xxx) est indexee par un enum INTERNE, pas par le type filaire : le descripteur
// PlayerGameEventSmall est l'objet 0x143d0ec18 (nom a +0x08 -> thunk 0x14119e930 = LEA
// "PlayerGameEventSmall"). Ses domaines de refs d'en-tete (vtable+0x58 = 0x142ef7f6c) :
// index0 -> dom0, index1 -> dom8, index2 -> dom7 (calibre contre damage {1,1,7} et fire
// {1,...} dont les fonctions de domaine voisines rendent exactement les domaines connus).
// Son lecteur de charge (vtable+0x68 = FUN_14080add8 -> FUN_14080ae70 + FUN_14080ae28) :
//
//	R(32) A   — identifiant/type de l'evenement (out[0])
//	R(8)  B   — champ court (out+8)
//	liste de proprietes : R(3) compte (0..7), puis compte x [ nom R(32) + selecteur R(3) +
//	            valeur typee ] ; valeurs : 0 -> 0 bit, {1,2,3,6} -> R(32), 4 -> R(1),
//	            5 -> chaine (<=16 octets), 7 -> palette runtime (rare)
//	bloc "text" optionnel : R(1) porte ; si 1 : nom R(32), R(3) compte, compte x element
//	            [ sous-type R(3) ; 1 -> R(1)+[si 0: R(5) index de participant], 3 -> R(32),
//	            2 -> quantifie runtime, autres -> R(32)/0 bit ]
//	R(32) masque final
//
// C'EST UN SAC DE PROPRIETES NOMMEES TYPEES. Il N'Y A, dans la charge, NI champ WeaponID
// (aucun R(64) arme), NI magnitude de degat, NI reference de victime structuree, NI paire
// tireur+cible. L'arme et la cible, si elles existent, ne pourraient etre que dans des
// valeurs opaques hachees — ce qui n'est PAS un enregistrement de touche attribue.
// Le present instrument MESURE cette lecture (chiffre vs temoin) plutot que de l'affirmer.
//
// MESURES (temoins ecrits avant) :
//
//	M1 CENSUS — compte type82/type83 ; A categoriel ? (distinct/total, contre le temoin arme
//	   ~11 %) ; distribution de B ; compte et types des proprietes ; presence du bloc "text".
//	M2 EMETTEUR — refs d'en-tete presentes et resolues en bipede (base 512) : l'evenement est-il
//	   emis par / rattache a un joueur ?
//	M3 ORACLE DE TRAME — apres la charge (exacte) + continuation=0, la trame de records doit
//	   aller LOIN (>= 1 record/paquet ET >= 3x un temoin decale de +3 bits), preuve du cadrage
//	   bit-exact (meme juge que damage_aftermath).
//	M4 ARME / TOUCHE — A ou une valeur de propriete intersecte-t-elle l'ensemble des WeaponID
//	   des tirs (bas/haut 32 bits) ? Les evenements coincident-ils avec un tir (±250 ms) au-dela
//	   d'un temoin decale de +3 s ? Une couverture arme exigerait au moins l'un des deux.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule, borne a
// deltaWitnessChunks (12). Lancer une fois par film (000d5950, 01e1f945, 00502e52).

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

const (
	pgesMatchW  = uint64(250_000)   // 250 ms : coincidence evenement <-> tir
	pgesWitOff  = uint64(3_000_000) // 3 s : temoin decale
	pgesRefBase = lot1chReferenceBase
)

// pgesCensus agrege les mesures M1/M2 sur une tranche de records type 82.
type pgesCensus struct {
	n82, n83                int
	aVals                   map[uint64]int
	bVals                   map[uint64]int
	propCount               map[int]int
	selHist                 map[int]int
	propNames               map[uint64]int
	ref0Pres, ref1, ref2    int
	ref0Bip, ref1Bip, r2Bip int
	hasText, participants   int
	inexact                 int
}

func newPgesCensus() *pgesCensus {
	return &pgesCensus{
		aVals: map[uint64]int{}, bVals: map[uint64]int{}, propCount: map[int]int{},
		selHist: map[int]int{}, propNames: map[uint64]int{},
	}
}

func (c *pgesCensus) add(r pgesRecord, w *World) {
	c.n82++
	c.aVals[r.payload.fieldA]++
	c.bVals[r.payload.fieldB]++
	c.propCount[len(r.payload.props)]++
	for _, pr := range r.payload.props {
		c.selHist[pr.sel]++
		c.propNames[pr.name]++
	}
	if r.payload.hasText {
		c.hasText++
	}
	c.participants += len(r.payload.participants)
	if !r.payload.exact {
		c.inexact++
	}
	if r.ref0 >= 0 {
		c.ref0Pres++
		if pgesResolveBiped(w, pgesRefBase, r.ref0) {
			c.ref0Bip++
		}
	}
	if r.ref1 >= 0 {
		c.ref1++
		if pgesResolveBiped(w, pgesRefBase, r.ref1) {
			c.ref1Bip++
		}
	}
	if r.ref2 >= 0 {
		c.ref2++
		if pgesResolveBiped(w, pgesRefBase, r.ref2) {
			c.r2Bip++
		}
	}
}

// pgesOracle porte les compteurs de l'oracle de trame (M3).
type pgesOracle struct {
	nEvt, deltasReel, fermees, deltasTemoin, fermTemoin int
}

// pgesRunFrame juge la trame apres un evenement type 82 a charge exacte et continuation=0.
func pgesRunFrame(o *pgesOracle, pay []byte, r pgesRecord, reg *Registry, snap WorldSnapshot) {
	if !r.payload.exact {
		return
	}
	br := NewBitReader(pay)
	br.SetBitPos(r.bitAfter)
	if br.ReadBit() { // continuation : un autre evenement suit -> on ne juge pas
		return
	}
	pos := br.BitPos()
	o.nEvt++
	w := NewWorld(reg)
	w.Restore(snap)
	recs, err := DecodeFrameRecords(br, w, DefaultFrameConfig())
	o.deltasReel += len(recs)
	if err == nil {
		o.fermees++
	}
	if p := pos + 3; p+16 < len(pay)*8 { // temoin negatif : +3 bits
		w2 := NewWorld(reg)
		w2.Restore(snap)
		cbr := NewBitReader(pay)
		cbr.Skip(p)
		crecs, cerr := DecodeFrameRecords(cbr, w2, DefaultFrameConfig())
		o.deltasTemoin += len(crecs)
		if cerr == nil {
			o.fermTemoin++
		}
	}
}

func TestPlayerGameEventSmall(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	t.Logf("== film %s · %d chunks · base bipede %d ==", filepath.Base(dir), n, pgesRefBase)

	cen := newPgesCensus()
	var ora pgesOracle
	var evtTs []uint64
	var aValsAll []uint64
	cfg := DefaultFrameConfig()

	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		wBase := NewWorld(reg)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, wBase, cfg)
			}
		}
		snap := wBase.Snapshot()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			switch {
			case pay[0] == 0xE9:
				r, ok := pgesDecodePacket(pay, pk.TimestampUS)
				if !ok { // type 83 (TeamGameEvent) ou liste vide
					if pgesIsType83(pay) {
						cen.n83++
					}
					continue
				}
				cen.add(r, wBase)
				evtTs = append(evtTs, r.ts)
				aValsAll = append(aValsAll, r.payload.fieldA)
				pgesRunFrame(&ora, pay, r, reg, snap)
			}
		}
	}
	sort.Slice(evtTs, func(a, b int) bool { return evtTs[a] < evtTs[b] })

	pgesLogCensus(t, cen)
	pgesLogOracle(t, &ora)
	pgesLogWeaponTouch(t, dir, n, cen, aValsAll, evtTs)
}

// pgesIsType83 rend vrai si le paquet 0xE9 porte le type 83 (TeamGameEvent).
func pgesIsType83(pay []byte) bool {
	br := NewBitReader(pay)
	br.Skip(1)
	return br.ReadBit() && br.ReadBits(7) == 83
}

func pgesLogCensus(t *testing.T, c *pgesCensus) {
	t.Helper()
	t.Logf("M1 CENSUS : type 82 (PlayerGameEventSmall) x%d · type 83 (TeamGameEvent) x%d · charge inexacte (selecteur runtime) %d",
		c.n82, c.n83, c.inexact)
	t.Logf("   champ A (R32) : %d valeurs distinctes / %d (%.1f %%) — %s",
		len(c.aVals), c.n82, lot1Pct(len(c.aVals), c.n82), lot1TopU64(c.aVals, 6))
	t.Logf("   champ B (R8) : %s", lot1TopU64(c.bVals, 6))
	t.Logf("   proprietes/evenement : %s · selecteurs de valeur : %s",
		pgesTopInt(c.propCount, 8), pgesTopInt(c.selHist, 8))
	t.Logf("   noms de propriete distincts : %d · top : %s", len(c.propNames), lot1TopU64(c.propNames, 6))
	t.Logf("   bloc text present : %d/%d (%.1f %%) · participants (text #1, index R5) : %d",
		c.hasText, c.n82, lot1Pct(c.hasText, c.n82), c.participants)
	t.Logf("M2 EMETTEUR (refs d'en-tete, base %d) :", pgesRefBase)
	t.Logf("   ref0 (dom0) presente %d/%d (%.1f %%) · resolue bipede %d (%.1f %%)",
		c.ref0Pres, c.n82, lot1Pct(c.ref0Pres, c.n82), c.ref0Bip, lot1Pct(c.ref0Bip, c.ref0Pres))
	t.Logf("   ref1 (dom8) presente %d · bipede %d · ref2 (dom7) presente %d · bipede %d",
		c.ref1, c.ref1Bip, c.ref2, c.r2Bip)
}

func pgesLogOracle(t *testing.T, o *pgesOracle) {
	t.Helper()
	profReel := float64(o.deltasReel) / float64(max(1, o.nEvt))
	profTem := float64(o.deltasTemoin) / float64(max(1, o.nEvt))
	t.Logf("M3 ORACLE DE TRAME (charge exacte, evenement unique) : %d records / %d paquets = %.2f/paquet · fermee %.1f %%",
		o.deltasReel, o.nEvt, profReel, lot1Pct(o.fermees, o.nEvt))
	t.Logf("   TEMOIN NEGATIF (+3 bits) : %d records / %d = %.2f/paquet · fermee %.1f %%",
		o.deltasTemoin, o.nEvt, profTem, lot1Pct(o.fermTemoin, o.nEvt))
	ok := o.nEvt >= 30 && profReel >= 1.0 && o.deltasReel >= 3*o.deltasTemoin
	t.Logf("   VERDICT cadrage bit-exact (profondeur >= 1/paquet ET >= 3x le temoin) : %s", lot1Verdict(ok))
}

// pgesLogWeaponTouch mesure M4 : intersection A/proprietes avec les WeaponID, et coincidence
// temporelle evenement <-> tir contre un temoin decale.
func pgesLogWeaponTouch(t *testing.T, dir string, n int, c *pgesCensus, aVals, evtTs []uint64) {
	t.Helper()
	widLo, widHi, shotTs := pgesCollectShotSets(t, dir, n)
	interA := 0
	for _, a := range aVals {
		if widLo[a] || widHi[a] || widLo[a&0xFFFFFFFF] {
			interA++
		}
	}
	interName := 0
	for name := range c.propNames {
		if widLo[name] || widHi[name] {
			interName++
		}
	}
	t.Logf("M4 ARME / TOUCHE :")
	t.Logf("   champ A intersecte un WeaponID (bas/haut 32b) : %d/%d evenements · noms de propriete = WeaponID : %d/%d",
		interA, len(aVals), interName, len(c.propNames))
	near, wit := 0, 0
	for _, ts := range evtTs {
		if lot1mtNear(shotTs, ts, pgesMatchW) {
			near++
		}
		if lot1mtNear(shotTs, ts+pgesWitOff, pgesMatchW) {
			wit++
		}
	}
	nn := len(evtTs)
	t.Logf("   coincidence evenement <-> tir (n'importe lequel) ±%dms : %d/%d (%.1f %%) · temoin +%ds %d (%.1f %%)",
		pgesMatchW/1000, near, nn, lot1Pct(near, nn), pgesWitOff/1_000_000, wit, lot1Pct(wit, nn))
	couvre := interA > 0 || interName > 0
	t.Logf("   COUVERTURE ARME (A ou nom de propriete = WeaponID) : %s — sinon l'evenement ne porte AUCUNE attribution d'arme",
		lot1Verdict(couvre))
}

// pgesTopInt rend les k entrees les plus frequentes d'un histogramme a cle int, "cle:n".
func pgesTopInt(m map[int]int, k int) string {
	type kv struct{ key, v int }
	var s []kv
	for key, v := range m {
		s = append(s, kv{key, v})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].v > s[j].v })
	if len(s) > k {
		s = s[:k]
	}
	out := ""
	for i, e := range s {
		if i > 0 {
			out += " · "
		}
		out += itoa(e.key) + ":" + itoa(e.v)
	}
	if out == "" {
		return "(aucun)"
	}
	return out
}
