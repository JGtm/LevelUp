//go:build research

package grammar

// mouvement_5_15_position_research_test.go — LA POSITION DU RESTE (lot 5.15.1).
//
// Le lot 5.14 a referme les vues A et C et laisse UN reste dominant : sur `bfecd02b`,
// 23 852 paquets ont leurs TROIS rangs lus jusqu a leur terminateur et laissent pourtant
// environ 1 076 bits chacun (D1 du 5.14). Cet instrument NE CONCLUT RIEN : il LOCALISE.
//
// Il repond a quatre questions, et a elles seules :
//
//	1. OU commence le reste : apres le terminateur de la vue C (donc apres les trois rangs),
//	   ou plus tot ?
//	2. COMMENT la vue B a clos sa liste : sur son terminateur `recEnd` (trois bits `000`) ou
//	   sur le REJET de table de vue (`rejetDeVue`, qui laisse le curseur a la fin d un en-tete
//	   de 1 + idLow + 2 bits) ?
//	3. QUELLE EST LA LARGEUR du reste, classe par classe (1 076 constant, ou 1 076 plus ou
//	   moins un multiple d une largeur nommee) ?
//	4. QUE PORTE le reste, en bits bruts.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestTrou515Position$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// t515Marche est ce qu un paquet a rendu rang par rang, avec les CURSEURS intermediaires.
type t515Marche struct {
	Debut    int  // le bit de depart de la marche (amorce ou debut localise)
	FinVueA  int  // curseur apres la vue A (== Debut quand la marche part d un debut localise)
	FinVueB  int  // curseur apres la vue B
	FinVueC  int  // curseur apres la vue C
	PorteA   bool // la vue A a lu jusqu a son terminateur
	HitEndB  bool // la vue B a clos sa liste (terminateur OU rejet de table de vue)
	PorteC   bool // la vue C a lu jusqu a son terminateur
	Records  int  // records rendus par la vue B
	DernSlot uint32
	DernTI   uint32
	DernType int
	KindsC   []int
}

// t515Marcher rejoue un paquet rang par rang, exactement comme `decodeFrameParRangs`, et rend
// les curseurs intermediaires. C est la MEME decoupe que `c514Cause` du gate 5.14 ; elle est
// reecrite ici parce que `c514Cause` ne rend qu une cause et un reste, pas les positions.
func t515Marcher(pay []byte, w *World, cfg FrameConfig, debut int) t515Marche {
	frameLen := len(pay) * 8
	m := t515Marche{Debut: debut}
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	if debut == DefaultPacketPreambleBits && cfg.PacketPreambleBits >= 1 {
		br.Skip(cfg.PacketPreambleBits - 1)
		a := consumeVueA(br, frameLen)
		m.PorteA = a.Porte
		m.FinVueA = br.BitPos()
		if !a.Porte {
			m.FinVueB, m.FinVueC = br.BitPos(), br.BitPos()
			return m
		}
	} else {
		br.Skip(debut)
		m.PorteA, m.FinVueA = true, br.BitPos()
	}
	w.PoserVueCourante(int(vueDeLImageCle))
	recs, _, hitEnd := decodeInferLoop(br, pay, w, cfg)
	m.HitEndB, m.FinVueB, m.Records = hitEnd, br.BitPos(), len(recs)
	if n := len(recs); n > 0 {
		m.DernSlot, m.DernTI, m.DernType = recs[n-1].Slot, recs[n-1].TypeIndex, recs[n-1].Type
	}
	if !hitEnd {
		m.FinVueC = br.BitPos()
		return m
	}
	c := consumeVueC(br, frameLen)
	m.PorteC, m.KindsC, m.FinVueC = c.Porte, c.Kinds, br.BitPos()
	return m
}

// t515SortieVueB dit COMMENT la vue B a clos sa liste, en relisant les bits qui precedent son
// curseur de sortie. Les deux sorties `hitEnd` de `decodeInferLoop` sont indiscernables de
// l exterieur et ne portent pas la meme grammaire :
//
//	"terminateur" : `readRecordType` a rendu `recEnd`, soit un bit 0 puis `R(2) = 0` — les
//	                trois bits `000` qui precedent le curseur (plus les 32 bits de
//	                `HasExtraFields` quand il est leve).
//	"rejet"       : `rejetDeVue` a remis le curseur a la FIN DE L EN-TETE d un delta, soit
//	                `[1][idLow][tag]` — le bit a `curseur - (1 + idLow + 2)` vaut 1.
func t515SortieVueB(pay []byte, cfg FrameConfig, m t515Marche) string {
	if !m.HitEndB {
		return "desynchronisation"
	}
	enTete := 1 + cfg.IDLowBits + 2
	if cfg.HasExtraFields {
		enTete += 32
	}
	if m.FinVueB-3 >= m.FinVueA {
		if v := t515LireBits(pay, m.FinVueB-3, 3); v == 0 {
			return "terminateur"
		}
	}
	if m.FinVueB-enTete >= m.FinVueA {
		if t515LireBits(pay, m.FinVueB-enTete, 1) == 1 {
			return "rejet de table de vue"
		}
	}
	return "indetermine"
}

// t515LireBits rend `n` bits de `pay` a partir de `at` (MSB-first), ou -1 hors bornes.
func t515LireBits(pay []byte, at, n int) int {
	if at < 0 || n <= 0 || at+n > len(pay)*8 {
		return -1
	}
	v := 0
	for i := 0; i < n; i++ {
		v <<= 1
		if pay[(at+i)/8]&(1<<uint(7-(at+i)%8)) != 0 {
			v |= 1
		}
	}
	return v
}

// TestTrou515Position LOCALISE le reste des paquets dont les trois rangs sont portes.
func TestTrou515Position(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	bal := fc.ProfilDeBalayage()
	bal.Grammaire.ClassesDeVue = true
	fc.PoserProfilDeBalayage(bal)
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)

	var fautifs []t515Fautif
	var paquets, fermes int
	parSortie := map[string]int{}
	parCause := map[string]int{}
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
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			paquets++
			m := t515Marcher(pay, w, cfg, debut)
			reste := len(pay)*8 - m.FinVueC
			if reste >= 0 && reste <= m5116GateOctet && c514ResteNul(pay, m.FinVueC) {
				fermes++
				continue
			}
			switch {
			case !m.PorteA:
				parCause["rang 0 vue A non portee"]++
			case !m.HitEndB:
				parCause["rang 1 vue B desynchronisee"]++
			case !m.PorteC:
				parCause["rang 2 vue C non portee"]++
			default:
				parCause["les trois rangs portes, reste hors bourrage"]++
				parSortie[t515SortieVueB(pay, cfg, m)]++
				fautifs = append(fautifs, t515Fautif{Chunk: c, Marche: m,
					Reste: reste, Long: len(pay) * 8,
					Sortie: t515SortieVueB(pay, cfg, m),
					Bits:   m511Bits(pay, m.FinVueC, 64)})
			}
		}
	}

	t515Publier(t, paquets, fermes, parCause, parSortie, fautifs)
}

// t515Fautif est un paquet dont les trois rangs sont portes et qui laisse du reste.
type t515Fautif struct {
	Chunk  int
	Marche t515Marche
	Reste  int
	Long   int
	Sortie string
	Bits   string
}

// t515Publier ecrit le rapport : les causes, la sortie de la vue B, la distribution EXACTE de la
// largeur du reste, et les quarante echantillons demandes par l item 5.15.1.
func t515Publier(t *testing.T, paquets, fermes int, parCause, parSortie map[string]int,
	fautifs []t515Fautif) {
	t.Helper()
	t.Logf("PAQUETS MARCHES : %d · FERMES A RESTE NUL : %d", paquets, fermes)
	t.Logf("CAUSES :")
	for _, n := range t515Cles(parCause) {
		t.Logf("  %-46s %6d", n, parCause[n])
	}
	t.Logf("SORTIE DE LA VUE B sur les paquets a trois rangs portes :")
	for _, n := range t515Cles(parSortie) {
		t.Logf("  %-30s %6d", n, parSortie[n])
	}

	largeurs := map[int]int{}
	modulo := map[int]int{}
	records := map[int]int{}
	var minR, maxR, somme int
	minR = 1 << 30
	for _, f := range fautifs {
		largeurs[f.Reste]++
		modulo[f.Reste%16]++
		records[f.Marche.Records]++
		somme += f.Reste
		if f.Reste < minR {
			minR = f.Reste
		}
		if f.Reste > maxR {
			maxR = f.Reste
		}
	}
	if len(fautifs) == 0 {
		t.Logf("AUCUN PAQUET A TROIS RANGS PORTES NE LAISSE DE RESTE.")
		return
	}
	t.Logf("LARGEUR DU RESTE : %d paquets · min %d · max %d · moyenne %.1f · %d classes",
		len(fautifs), minR, maxR, float64(somme)/float64(len(fautifs)), len(largeurs))
	t.Logf("  distribution (les 24 plus petites classes) : %s", c514Hist(largeurs))
	t.Logf("  largeur modulo 16 (l en-tete de record fait 1+13+2) : %s", c514Hist(modulo))
	t.Logf("  records rendus par la vue B : %s", c514Hist(records))

	idx := t515Echantillons(len(fautifs))
	t.Logf("ECHANTILLONS (20 premiers + 20 du milieu) :")
	for _, i := range idx {
		f := fautifs[i]
		m := f.Marche
		t.Logf("  #%-6d chunk %2d · payload %6d bits · A %5d · B %6d · C %6d · reste %5d · "+
			"%2d records (dernier slot %5d ti %2d type %d) · kindsC %v · %s",
			i, f.Chunk, f.Long, m.FinVueA, m.FinVueB, m.FinVueC, f.Reste, m.Records,
			m.DernSlot, m.DernTI, m.DernType, m.KindsC, f.Sortie)
	}
	t.Logf("CONTENU BRUT du reste (les 64 premiers bits, MSB-first) :")
	for _, i := range idx[:5] {
		t.Logf("  #%-6d a partir du bit %6d : %s", i, fautifs[i].Marche.FinVueC, fautifs[i].Bits)
	}
}

// t515Echantillons rend les index des 20 premiers et des 20 du milieu.
func t515Echantillons(n int) []int {
	var out []int
	for i := 0; i < 20 && i < n; i++ {
		out = append(out, i)
	}
	for i := n / 2; i < n/2+20 && i < n; i++ {
		out = append(out, i)
	}
	return out
}

// t515Cles rend les cles triees d une table de comptage.
func t515Cles(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// t515Rejet est l EN-TETE que la vue B a rejete : le record que le jeu aurait lu et que la
// marche hors ligne refuse.
type t515Rejet struct {
	Slot   uint32
	Tag    uint32
	Lie    bool // le monde hors ligne connait ce slot
	Vue    int8 // la vue de la liaison (-1 = inconnue), quand elle existe
	TI     uint32
	Sansg  int  // records lus depuis cet en-tete, garde de vue DESARMEE
	FinSg  int  // curseur atteint, garde desarmee
	FermSg bool // le paquet ferme a reste NUL, garde desarmee
}

// t515LireRejet relit l en-tete rejete a `FinVueB - (1 + idLow + 2)` et interroge le monde.
func t515LireRejet(pay []byte, w *World, cfg FrameConfig, m t515Marche) (t515Rejet, bool) {
	enTete := 1 + cfg.IDLowBits + 2
	if cfg.HasExtraFields {
		enTete += 32
	}
	at := m.FinVueB - enTete
	if at < m.FinVueA {
		return t515Rejet{}, false
	}
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	br.Skip(at)
	if readRecordType(br) != recDelta {
		return t515Rejet{}, false
	}
	id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
	r := t515Rejet{Slot: id & 0x3fffffff, Tag: id >> 30, Vue: vueInconnue}
	if s, ok := w.slots[r.Slot]; ok {
		r.Lie, r.Vue, r.TI = true, s.Vue, s.TypeIndex
	}
	// LA MEME MARCHE, GARDE DE VUE DESARMEE : elle dit si le reste est fait de records de la
	// vue B que la garde a fait perdre, ou d autre chose.
	cfgSg := cfg
	cfgSg.Profil.Grammaire.TablesParVue = false
	br2 := LecteurSur(pay)
	br2.poserCadre(cfgSg)
	br2.Skip(at)
	recs, _, _ := decodeInferLoop(br2, pay, w, cfgSg)
	r.Sansg, r.FinSg = len(recs), br2.BitPos()
	reste := len(pay)*8 - r.FinSg
	r.FermSg = reste >= 0 && reste <= m5116GateOctet && c514ResteNul(pay, r.FinSg)
	return r, true
}

// TestTrou515Rejet NOMME l en-tete que la vue B rejette sur les paquets fautifs : quel slot, lie
// ou non, dans quelle vue — et ce que la meme marche lit quand la garde de vue est DESARMEE.
func TestTrou515Rejet(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	bal := fc.ProfilDeBalayage()
	bal.Grammaire.ClassesDeVue = true
	fc.PoserProfilDeBalayage(bal)
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)

	var rejets, nonLie, lieAutreVue, lieMemeVue, lieVueInconnue int
	var fermeSansGarde, gainRecords int
	parSlot := map[uint32]int{}
	parTI := map[uint32]int{}
	parTag := map[uint32]int{}
	sansGarde := map[int]int{}
	var vus int
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
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			m := t515Marcher(pay, w, cfg, debut)
			reste := len(pay)*8 - m.FinVueC
			if reste >= 0 && reste <= m5116GateOctet && c514ResteNul(pay, m.FinVueC) {
				continue
			}
			if !m.PorteA || !m.HitEndB || !m.PorteC {
				continue
			}
			vus++
			r, ok := t515LireRejet(pay, w, cfg, m)
			if !ok {
				continue
			}
			rejets++
			parSlot[r.Slot]++
			parTag[r.Tag]++
			switch {
			case !r.Lie:
				nonLie++
			case r.Vue == vueInconnue:
				lieVueInconnue++
				parTI[r.TI]++
			case r.Vue == int8(vueDeLImageCle):
				lieMemeVue++
				parTI[r.TI]++
			default:
				lieAutreVue++
				parTI[r.TI]++
			}
			sansGarde[r.Sansg]++
			gainRecords += r.Sansg
			if r.FermSg {
				fermeSansGarde++
			}
		}
	}
	t.Logf("PAQUETS A TROIS RANGS PORTES ET RESTE : %d · dont clos sur un REJET lisible : %d",
		vus, rejets)
	t.Logf("L EN-TETE REJETE : slot NON LIE %d · lie en vue INCONNUE %d · lie en MEME vue %d · "+
		"lie en AUTRE vue %d", nonLie, lieVueInconnue, lieMemeVue, lieAutreVue)
	t.Logf("  tag de generation du rejet : %s", c514Hist32(parTag))
	t.Logf("  archetype des slots LIES rejetes : %s", c514Hist32(parTI))
	t.Logf("  slots distincts rejetes : %d", len(parSlot))
	t.Logf("  slots les plus rejetes : %s", t515Top(parSlot, 12))
	t.Logf("GARDE DE VUE DESARMEE, depuis l en-tete rejete : %d records lus en tout "+
		"(%.1f par paquet) · %d paquets ferment alors a reste NUL",
		gainRecords, float64(gainRecords)/float64(max(rejets, 1)), fermeSansGarde)
	t.Logf("  records lus par paquet, garde desarmee : %s", c514Hist(sansGarde))
}

// c514Hist32 est [c514Hist] pour une table indexee par uint32.
func c514Hist32(m map[uint32]int) string {
	c := map[int]int{}
	for k, v := range m {
		c[int(k)] = v
	}
	return c514Hist(c)
}

// t515Top rend les `n` cles les plus comptees.
func t515Top(m map[uint32]int, n int) string {
	type kv struct {
		k uint32
		v int
	}
	l := make([]kv, 0, len(m))
	for k, v := range m {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].v != l[j].v {
			return l[i].v > l[j].v
		}
		return l[i].k < l[j].k
	})
	var parts []string
	for i := 0; i < n && i < len(l); i++ {
		parts = append(parts, fmt.Sprintf("slot %d : %d", l[i].k, l[i].v))
	}
	return strings.Join(parts, " · ")
}
