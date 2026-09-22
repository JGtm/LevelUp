//go:build research

package grammar

// mouvement_5_16_temoin_research_test.go — LE TEMOIN DE 96 BITS, FERME AU BIT (lot 5.16.1).
//
// # CE QUE CET INSTRUMENT ETABLIT
//
// Le lot 5.15 a nomme le trou du rang 1 : la vue B sort sur le REJET d un slot que le monde hors
// ligne n a jamais lie, et la branche VIVE de `FUN_1406cd128` lit le corps d un delta depuis la
// table de datums du DECODEUR PARTAGE (`FUN_1406cbaa0`, type 3 : garde
// `*(uint *)(slot * 200 + *(*(vue+0x20) + 0x20)) != eid`, archetype a `+0x04`). La question du
// lot 5.16 est donc : D OU VIENT L ARCHETYPE d un slot que ni une image-cle LUE ni un NEW LU ne
// declare ?
//
// L ORACLE D ARCHETYPE Y REPOND SANS INVENTER UNE LARGEUR. L ensemble des candidats est le
// REGISTRE du film (`chunk_00`, 50 archetypes, un artefact LU), et le critere d acceptation est
// la FERMETURE DU PAQUET au bit : corps propre, terminateur de la vue B (`000`), vue C portee,
// reste a bourrage NUL. Sur les douze paquets de 96 bits de `dad793c7` (temoin D5 du 5.15), DEUX
// archetypes ferment les douze — `ti=30` et `ti=47` — et l image-cle du film tranche : le slot
// rejete 1298 y est declare `ti=47`, c est-a-dire
// `managed-object-networked-splash-message-*`, dont le composant `i1` est un `R(24)`.
//
// ANATOMIE DU TEMOIN, MESUREE (les douze paquets sont identiques a partir du bit 37 sauf un
// compteur de 16 bits a pas constant de 4 584) :
//
//	bit  0        1     bit de configuration du frame-processeur
//	bit  1        0     terminateur de la vue A vide
//	bit  2        1     prefixe DELTA
//	bits 3..15    123   idLow, 13 bits          } record 1 : ti=4, masque 0x1, i0
//	bits 16..17   1     tag de generation       } `high-frequency` = R(8) (FUN_14076d034)
//	bit  18       0     selecteur de baseline ferme
//	bits 19..28   0x1   masque (1 + 3 + 6 bits)
//	bits 29..36         i0 high-frequency, 8 bits -> fin du record au bit 37
//	bit  37       1     prefixe DELTA           <- l en-tete que la marche REJETAIT
//	bits 38..50   1298  idLow  (le 5.15 lisait « 1 314 » : c est 1 298, id 0x40000512)
//	bits 51..52   1     tag
//	bit  53       0     selecteur de baseline ferme
//	bits 54..63   0x2   masque = le composant i1
//	bits 64..87         i1 `splash-message-dynamic` = R(24) (le compteur y est)
//	bits 88..90   000   terminateur de la vue B
//	bit  91       0     terminateur de la vue C (vide)
//	bits 92..95   0     bourrage d octet, PROUVE a zero
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> //	  go test -tags=research -count=1 -v -timeout 30m //	  -run '^TestTemoin516(Anatomie|Archetype|Generalise)$' //	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"testing"
)

// t516Temoin est le cadre commun des deux tests : film, contexte, registre, cadre de balayage.
type t516Temoin struct {
	fc  *FilmContext
	reg *Registry
	cfg FrameConfig
}

// t516Cadre monte le cadre de balayage sous la grammaire des trois classes de vue (lot 5.14),
// c est-a-dire celle du gate que le lot 5.16 doit deplacer.
func t516Cadre(t *testing.T) t516Temoin {
	t.Helper()
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		t.Cleanup(restore)
	}
	bal := fc.ProfilDeBalayage()
	bal.Grammaire.ClassesDeVue = true
	fc.PoserProfilDeBalayage(bal)
	return t516Temoin{fc: fc, reg: reg, cfg: fc.CadreDeBalayage()}
}

// t516SlotsImageCle rend, pour les paquets d image-cle d un chunk, ce que les DEUX lectures
// declarent : le balayeur filtre (`WalkKeyframeWorld`, celui que la marche emprunte) et le
// marcheur deterministe (`WalkKeyframeRecords`, sans filtre sur `Field26`).
func t516SlotsImageCle(data []byte, pks []FilmPacket, reg *Registry,
	ctx ContexteDeLecture) (balaye, marche map[uint32]uint32, stops []string) {
	balaye, marche = map[uint32]uint32{}, map[uint32]uint32{}
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		pay := pk.Payload(data)
		for _, r := range WalkKeyframeWorld(pay) {
			balaye[uint32(r.Slot)] = uint32(r.TI) //nolint:gosec // slot/TI bornes par le walker
		}
		recs, stop := WalkKeyframeRecords(pay, reg, ctx)
		for _, r := range recs {
			marche[uint32(r.Slot)] = uint32(r.TI) //nolint:gosec // slot/TI bornes par le walker
		}
		stops = append(stops, fmt.Sprintf("%d records, arret %s", len(recs), stop))
	}
	return balaye, marche, stops
}

// TestTemoin516Anatomie dump les paquets fautifs de 96 bits et INTERROGE les deux lectures de
// l image-cle sur le slot que la marche rejette. C est le plan de travail du lot : douze octets
// au lieu de 25 millions de bits.
func TestTemoin516Anatomie(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	ctx := tc.fc.ContexteDeLecture()
	vus := 0
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		balaye, marche, stops := t516SlotsImageCle(data, pks, tc.reg, ctx)
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			if _, present := PacketHeadEventType(pay); present {
				continue
			}
			m := t515Marcher(pay, w, tc.cfg, DefaultPacketPreambleBits)
			reste := len(pay)*8 - m.FinVueC
			if reste >= 0 && reste <= m5116GateOctet && c514ResteNul(pay, m.FinVueC) {
				continue
			}
			if !m.PorteA || !m.HitEndB || !m.PorteC || len(pay)*8 != 96 {
				continue
			}
			vus++
			if vus > 2 {
				return
			}
			t.Logf("CHUNK %d — IMAGE-CLE : balayeur filtre %d slots · marcheur deterministe "+
				"%d slots · %v", c, len(balaye), len(marche), stops)
			t516Dump(t, pay, w, tc.cfg, m, reste, balaye, marche)
		}
	}
	if vus == 0 {
		t.Logf("AUCUN paquet fautif de 96 bits sur ce film")
	}
}

// t516Dump publie l anatomie d un paquet temoin et le sort du slot rejete.
func t516Dump(t *testing.T, pay []byte, w *World, cfg FrameConfig, m t515Marche, reste int,
	balaye, marche map[uint32]uint32) {
	t.Helper()
	recs, _, _ := t515Records(pay, w, cfg, DefaultPacketPreambleBits)
	t.Logf("PAQUET de %d bits · A %d · B %d · C %d · reste %d · sortie B %s",
		len(pay)*8, m.FinVueA, m.FinVueB, m.FinVueC, reste,
		t515SortieVueB(pay, cfg, m))
	t.Logf("  TOTALITE : %s", m511Bits(pay, 0, len(pay)*8))
	for _, r := range recs {
		t.Logf("  record type %d slot %d ti %d masque %#x comps %d fin %d",
			r.Type, r.Slot, r.TypeIndex, r.Trace.Mask, len(r.Trace.Comps), r.Trace.EndBit)
	}
	enTete := m.FinVueB - (1 + cfg.IDLowBits + 2)
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	br.Skip(enTete)
	typ := readRecordType(br)
	id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
	slot := id & 0x3fffffff
	t.Logf("  EN-TETE REJETE au bit %d : type %d id %#x slot %d tag %d", enTete, typ, id,
		slot, id>>30)
	tiB, okB := balaye[slot]
	tiM, okM := marche[slot]
	_, lie := w.ArchetypeForSlot(slot)
	t.Logf("  LE SLOT %d : lie dans le monde %v · image-cle balayee %v (ti %d) · "+
		"image-cle marchee %v (ti %d)", slot, lie, okB, tiB, okM, tiM)
	t.Logf("  RESTE (%d bits) : %s", reste, m511Bits(pay, m.FinVueC, reste))
}

// TestTemoin516ImageCle compare, sur tout le film, ce que les DEUX lectures de la table
// d image-cle declarent — et croise le manque avec les slots que la vue B REJETTE.
//
// C EST LA MESURE DE VERIFICATION DU MAILLON LU : si le filtre `Field26 == 0` du balayeur cache
// des records, les slots rejetes par la vue B doivent se retrouver dans le marcheur deterministe
// et pas dans le balayeur.
// (ou un petit groupe) ferme les douze paquets au bit.
func TestTemoin516Archetype(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	vus, ferme := 0, 0
	compte := map[string]int{}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
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
				continue
			}
			m := t515Marcher(pay, w, tc.cfg, DefaultPacketPreambleBits)
			reste := len(pay)*8 - m.FinVueC
			if reste >= 0 && reste <= m5116GateOctet && c514ResteNul(pay, m.FinVueC) {
				continue
			}
			if !m.PorteA || !m.HitEndB || !m.PorteC || len(pay)*8 != 96 {
				continue
			}
			vus++
			gagnants := t516Oracle(pay, tc.reg, tc.cfg, m.FinVueB)
			if len(gagnants) > 0 {
				ferme++
			}
			compte[fmt.Sprint(gagnants)]++
		}
	}
	t.Logf("TEMOINS DE 96 BITS : %d · FERMES par au moins un archetype du registre : %d",
		vus, ferme)
	noms := make([]string, 0, len(compte))
	for k := range compte {
		noms = append(noms, k)
	}
	sort.Strings(noms)
	for _, n := range noms {
		t.Logf("  archetypes gagnants %s : %d paquets", n, compte[n])
	}
}

// t516Gagnant est un archetype candidat et le bilan de la fermeture qu il produit.
type t516Gagnant struct {
	TI     int
	FinB   int // fin du corps du record
	FinVue int // fin de la vue C
	Reste  int
}

// String rend « ti=N corps->B vueC->C reste R ».
func (g t516Gagnant) String() string {
	return fmt.Sprintf("ti=%d corps->%d vueC->%d reste %d", g.TI, g.FinB, g.FinVue, g.Reste)
}

// t516Oracle essaie chaque archetype du registre sur le corps du record rejete (qui commence au
// curseur `finEntete`, la fin de l en-tete) et rend ceux qui ferment le paquet : corps propre,
// terminateur de la vue B, vue C portee, reste a bourrage NUL.
func t516Oracle(pay []byte, reg *Registry, cfg FrameConfig, finEntete int) []t516Gagnant {
	frameLen := len(pay) * 8
	var out []t516Gagnant
	for ti := range reg.Archetypes {
		br := LecteurSur(pay)
		br.poserCadre(cfg)
		br.Skip(finEntete)
		if br.ReadBit() { // selecteur de baseline
			br.Skip(7)
		}
		tr := decodeDeltaWithArch(br, reg.Archetypes[ti], uint32(ti)) //nolint:gosec // ti borne
		if tr.DesyncAt != -1 || br.BitPos() > frameLen {
			continue
		}
		finCorps := br.BitPos()
		if readRecordType(br) != recEnd {
			continue
		}
		c := consumeVueC(br, frameLen)
		if !c.Porte {
			continue
		}
		reste := frameLen - br.BitPos()
		if reste < 0 || reste > m5116GateOctet || !c514ResteNul(pay, br.BitPos()) {
			continue
		}
		out = append(out, t516Gagnant{TI: ti, FinB: finCorps, FinVue: br.BitPos(), Reste: reste})
	}
	return out
}

// TestTemoin516Generalise porte l ORACLE D ARCHETYPE a TOUS les paquets fautifs du film, et non
// aux seuls temoins de 96 bits : a chaque rejet, il essaie les archetypes du REGISTRE sur le
// corps du record, lie le slot au vainqueur et RELANCE la marche — jusqu a ce que le paquet
// ferme ou qu aucun archetype ne tienne.
//
// Ce n est pas un balayage de largeur : l ensemble des candidats est le registre du film
// (chunk_00), et le critere d acceptation est la FERMETURE DU PAQUET, pas une ressemblance.
// C est la mesure qui dit combien du trou du rang 1 le MODELE de la table de datums referme.
func TestTemoin516Generalise(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	var paquets, dejaFermes, fermes, echecs int
	parTI := map[int]int{}
	toursHist := map[int]int{}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
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
				if debut = marchLocateStrict(pay, w, tc.cfg); debut < 0 {
					continue
				}
			}
			paquets++
			m := t515Marcher(pay, w, tc.cfg, debut)
			if r := len(pay)*8 - m.FinVueC; r >= 0 && r <= m5116GateOctet &&
				c514ResteNul(pay, m.FinVueC) {
				dejaFermes++
				continue
			}
			tours, ti, ok := t516Relancer(pay, w, tc.reg, tc.cfg, debut)
			toursHist[tours]++
			if !ok {
				echecs++
				continue
			}
			fermes++
			for _, v := range ti {
				parTI[v]++
			}
		}
	}
	t.Logf("PAQUETS %d · deja fermes %d · REFERMES PAR L ORACLE %d · echecs %d",
		paquets, dejaFermes, fermes, echecs)
	t.Logf("  tours d oracle par paquet : %s", c514Hist(toursHist))
	t.Logf("  archetypes elus : %s", t516HistTI(parTI))
}

// t516RelanceMax borne le nombre de rejets successifs qu un paquet peut demander. Ce n est pas
// une largeur de grammaire : c est le nombre d entites inconnues qu un paquet peut porter.
const t516RelanceMax = 64

// t516Relancer marche un paquet en resolvant chaque rejet par l oracle du registre. Rend le
// nombre de tours, les archetypes elus et si le paquet a FERME (reste a bourrage nul).
// Les liaisons posees sont RETIREES avant de rendre : l instrument ne doit pas apprendre.
func t516Relancer(pay []byte, w *World, reg *Registry, cfg FrameConfig, debut int) (int, []int, bool) {
	var elus []int
	var poses []uint32
	defer func() {
		for _, s := range poses {
			w.Unbind(s)
		}
	}()
	for tours := 0; tours < t516RelanceMax; tours++ {
		m := t515Marcher(pay, w, cfg, debut)
		if r := len(pay)*8 - m.FinVueC; r >= 0 && r <= m5116GateOctet &&
			c514ResteNul(pay, m.FinVueC) {
			return tours, elus, true
		}
		if !m.HitEndB || t515SortieVueB(pay, cfg, m) != "rejet de table de vue" {
			return tours, elus, false
		}
		finEntete := m.FinVueB
		enTete := finEntete - (1 + cfg.IDLowBits + 2)
		if enTete < 0 {
			return tours, elus, false
		}
		br := LecteurSur(pay)
		br.poserCadre(cfg)
		br.Skip(enTete)
		if readRecordType(br) != recDelta {
			return tours, elus, false
		}
		id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
		g := t516Oracle(pay, reg, cfg, finEntete)
		if len(g) == 0 {
			g = t516OracleCorps(pay, reg, cfg, finEntete)
		}
		if len(g) == 0 {
			return tours, elus, false
		}
		w.BindSoft(id, uint32(g[0].TI)) //nolint:gosec // TI vient du registre
		poses = append(poses, id&0x3fffffff)
		elus = append(elus, g[0].TI)
	}
	return t516RelanceMax, elus, false
}

// t516OracleCorps est [t516Oracle] SANS l exigence de fermeture du paquet : il retient les
// archetypes dont le CORPS se lit proprement. Il sert au second et aux tours suivants, ou la
// fermeture depend des rejets encore a resoudre.
func t516OracleCorps(pay []byte, reg *Registry, cfg FrameConfig, finEntete int) []t516Gagnant {
	frameLen := len(pay) * 8
	var out []t516Gagnant
	for ti := range reg.Archetypes {
		br := LecteurSur(pay)
		br.poserCadre(cfg)
		br.Skip(finEntete)
		if br.ReadBit() {
			br.Skip(7)
		}
		tr := decodeDeltaWithArch(br, reg.Archetypes[ti], uint32(ti)) //nolint:gosec // ti borne
		if tr.DesyncAt != -1 || br.BitPos() > frameLen || len(tr.Comps) == 0 {
			continue
		}
		out = append(out, t516Gagnant{TI: ti, FinB: br.BitPos()})
	}
	return out
}

// t516HistTI rend un histogramme trie « ti=N : c » .
func t516HistTI(m map[int]int) string {
	if len(m) == 0 {
		return "vide"
	}
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var sb []string
	for _, k := range cles {
		sb = append(sb, fmt.Sprintf("ti=%d : %d", k, m[k]))
	}
	return joinVirgule(sb)
}

// joinVirgule joint des morceaux par « · ».
func joinVirgule(p []string) string {
	out := ""
	for i, s := range p {
		if i > 0 {
			out += " · "
		}
		out += s
	}
	return out
}

// TestTemoin516Population recense CE QUE LE FILM DECLARE : les types de paquet presents, les
// slots que les images-cles portent, ceux que les records NEW posent, et ceux que la vue B
// REJETTE — avec les noms de registre des archetypes elus par l oracle.
//
// C est la mesure de l item 5.16.2 : si aucune des sources LUES ne declare un slot rejete, le
