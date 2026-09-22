//go:build research

package grammar

// mouvement_5_19_temoin_research_test.go — LE TEMOIN DE LA CLASSE QUE LA DIFFERENTIELLE NOMME
// (lot 5.19.2).
//
// La differentielle (5.19.1) nomme une classe dont TOUT le corps est lu par des feuilles deja
// prouvees chez l ecrivain : un record de `ti=40` dont le masque vaut `0x10`, c est-a-dire le
// SEUL composant `i4 object-body-vitality-component`, dont `FUN_140fb8978` lit exactement
// `FUN_1406d84b4(..., 8, 1, 1)` plus trois `FUN_1406cf008` — **R(8) + 3 x R(1) = 11 bits**,
// exactement ce que le port consomme. Le record entier vaut donc
// `[1][idLow 13][tag 2][selecteur de baseline][masque][11 bits]`, et il faute 87,2 % du temps.
//
// Si aucune de ces largeurs n est fausse, alors ce qui manque n est pas DANS le record : c est
// ce qui vient APRES. Cet instrument le LIT, comme le temoin de 96 bits du 5.15.1 (j) et
// l anatomie du 5.16.1 : les positions de chaque record du paquet, puis les bits bruts depuis la
// fin du dernier record, et la ventilation des premiers bits qui suivent.
//
// Il NE CONCLUT RIEN et n essaie AUCUNE largeur.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  MOUV519_TI=40 MOUV519_MASQUE=0x10 \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestTemoin519$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// t519Classe est la classe de record a isoler : archetype et masque exact du DERNIER record.
type t519Classe struct {
	ti     uint32
	masque uint64
}

// t519ClasseDemandee lit la classe dans l environnement (defaut : `ti=40`, masque `0x10`).
func t519ClasseDemandee(t *testing.T) t519Classe {
	t.Helper()
	c := t519Classe{ti: 40, masque: 0x10}
	if v := os.Getenv("MOUV519_TI"); v != "" {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			t.Fatalf("MOUV519_TI %q : %v", v, err)
		}
		c.ti = uint32(n)
	}
	if v := os.Getenv("MOUV519_MASQUE"); v != "" {
		n, err := strconv.ParseUint(strings.TrimPrefix(v, "0x"), 16, 64)
		if err != nil {
			t.Fatalf("MOUV519_MASQUE %q : %v", v, err)
		}
		c.masque = n
	}
	return c
}

// TestTemoin519 isole les paquets fautifs dont le DERNIER record est de la classe demandee, et
// publie leur anatomie : chaque record avec ses bornes, puis la queue en bits bruts.
func TestTemoin519(t *testing.T) {
	tc := t516Cadre(t)
	cl := t519ClasseDemandee(t)
	w := NewWorld(tc.reg)
	suite := map[string]int{}
	tetes, restes := map[string]int{}, map[int]int{}
	vus := 0
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
				w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
			}
		}
		LierTableDeDatums(w, data, pks)
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
			mar := t519Marcher(pay, w, tc.cfg, debut)
			if !t519EstDeLaClasse(pay, tc.cfg, mar, cl) {
				continue
			}
			vus++
			t519Cumuler(pay, tc.cfg, mar, suite, tetes, restes)
			if vus <= 6 {
				t519Dumper(t, vus, pay, tc.cfg, mar)
			}
		}
	}
	t.Logf("TEMOIN ti=%d masque %#x : %d paquets fautifs", cl.ti, cl.masque, vus)
	t.Logf("  SUITE DES RECORDS (archetypes, du premier au dernier) :")
	t519Top(t, suite, 12)
	t.Logf("  LES 16 PREMIERS BITS APRES LA FIN DU DERNIER RECORD :")
	t519Top(t, tetes, 16)
	t.Logf("  LARGEUR DE LA QUEUE (bits entre la fin du dernier record et la fin du payload) :")
	t519Restes(t, restes)
}

// t519EstDeLaClasse dit si ce paquet est un fautif « rejet de slot inconnu » dont le DERNIER
// record porte l archetype et le masque demandes.
func t519EstDeLaClasse(pay []byte, cfg FrameConfig, mar d519Marche, cl t519Classe) bool {
	reste := len(pay)*8 - mar.m.FinVueC
	if reste < 0 || (reste <= m5116GateOctet && c514ResteNul(pay, mar.m.FinVueC)) {
		return false
	}
	if !mar.m.PorteA || !mar.m.HitEndB || !mar.m.PorteC {
		return false
	}
	if t515SortieVueB(pay, cfg, mar.m) != "rejet de table de vue" {
		return false
	}
	if len(mar.recs) == 0 {
		return false
	}
	last := mar.recs[len(mar.recs)-1]
	return last.TypeIndex == cl.ti && last.Trace.Mask == cl.masque
}

// t519Cumuler ventile la suite des archetypes, les bits de tete de la queue et sa largeur.
func t519Cumuler(pay []byte, cfg FrameConfig, mar d519Marche,
	suite, tetes map[string]int, restes map[int]int) {
	var tis []string
	for _, r := range mar.recs {
		tis = append(tis, fmt.Sprintf("%d", r.TypeIndex))
	}
	suite[strings.Join(tis, "-")]++
	fin := mar.recs[len(mar.recs)-1].Trace.EndBit
	restes[len(pay)*8-fin]++
	// LE CURSEUR DE SORTIE EST A LA FIN DE L EN-TETE REJETE, donc l en-tete commence
	// `1 + IDLowBits + 2` bits plus tot — c est-a-dire a la fin du dernier record.
	enTete := 1 + cfg.IDLowBits + 2
	if cfg.HasExtraFields {
		enTete += 32
	}
	tetes[fmt.Sprintf("%016b (en-tete rejete a %+d du record)",
		uint16(t519Bits16(pay, fin)), mar.m.FinVueB-enTete-fin)]++
}

// t519Bits16 rend les seize bits de `pay` a partir de `at`, MSB-first, ou -1 hors bornes.
func t519Bits16(pay []byte, at int) int {
	if v := t515LireBits(pay, at, 16); v >= 0 {
		return v
	}
	return 0
}

// t519Dumper publie l anatomie complete d UN paquet temoin.
func t519Dumper(t *testing.T, n int, pay []byte, cfg FrameConfig, mar d519Marche) {
	t.Helper()
	t.Logf("PAQUET TEMOIN %d — %d octets (%d bits) · debut %d · fin vue A %d · fin vue B %d · "+
		"fin vue C %d", n, len(pay), len(pay)*8, mar.m.Debut, mar.m.FinVueA, mar.m.FinVueB,
		mar.m.FinVueC)
	for i, r := range mar.recs {
		cs := d519Largeurs(r)
		var noms []string
		for _, c := range cs {
			noms = append(noms, fmt.Sprintf("i%d@%d(%db)", c.Index, c.Debut, c.Largeur))
		}
		t.Logf("  record %d : type %d ti=%d slot %d masque %#x fin %d · %s",
			i, r.Type, r.TypeIndex, r.Slot, r.Trace.Mask, r.Trace.EndBit,
			strings.Join(noms, " "))
	}
	fin := mar.recs[len(mar.recs)-1].Trace.EndBit
	enTete := 1 + cfg.IDLowBits + 2
	if cfg.HasExtraFields {
		enTete += 32
	}
	t.Logf("  QUEUE depuis le bit %d (%d bits) : %s", fin, len(pay)*8-fin,
		m511Bits(pay, fin, len(pay)*8-fin))
	t.Logf("  EN-TETE REJETE lu a %d (soit %+d du record) : prefixe %d · idLow %d · tag %d",
		mar.m.FinVueB-enTete, mar.m.FinVueB-enTete-fin,
		t515LireBits(pay, mar.m.FinVueB-enTete, 1),
		t515LireBits(pay, mar.m.FinVueB-enTete+1, cfg.IDLowBits),
		t515LireBits(pay, mar.m.FinVueB-2, 2))
}

// t519Top publie les `n` classes les plus peuplees d une ventilation.
func t519Top(t *testing.T, m map[string]int, n int) {
	t.Helper()
	cles := make([]string, 0, len(m))
	tot := 0
	for k, v := range m {
		cles = append(cles, k)
		tot += v
	}
	sort.Slice(cles, func(i, j int) bool {
		if m[cles[i]] != m[cles[j]] {
			return m[cles[i]] > m[cles[j]]
		}
		return cles[i] < cles[j]
	})
	for i := 0; i < n && i < len(cles); i++ {
		t.Logf("    %6d (%5.1f %%)  %s", m[cles[i]], m533bPart(m[cles[i]], tot), cles[i])
	}
	t.Logf("    TOTAL %d · %d classes", tot, len(cles))
}

// t519Restes publie la ventilation des largeurs de queue.
func t519Restes(t *testing.T, m map[int]int) {
	t.Helper()
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return m[cles[i]] > m[cles[j]] })
	var sb []string
	for i, k := range cles {
		if i >= 16 {
			sb = append(sb, fmt.Sprintf("... %d classes de plus", len(cles)-16))
			break
		}
		sb = append(sb, fmt.Sprintf("%d bits x%d", k, m[k]))
	}
	t.Logf("    %s", strings.Join(sb, " · "))
}

// TestRejets519 ventile LE SLOT REJETE PAR VOLUME, et confronte chaque slot aux sources
// d archetype du film. Le 5.16.2 avait mesure les slots rejetes DISTINCTS (632, etendue quasi
// uniforme sur les treize bits) et en avait conclu « ce ne sont pas des slots » ; cette mesure
// les pondere par leur NOMBRE D OCCURRENCES, ce que personne n avait fait.
//
//	MOUV511_FILM=... go test -tags=research -run '^TestRejets519$' ...
func TestRejets519(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	parSlot := map[uint32]int{}
	balaye, datum := map[uint32]uint32{}, map[uint32]uint32{}
	marche := map[uint32]uint32{}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, r := range WalkKeyframeWorld(pay) {
				//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
				w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
				balaye[uint32(r.Slot)] = uint32(r.TI) //nolint:gosec // bornes par le walker
			}
			table, _ := TableDeDatums(pay)
			for s, ti := range table {
				datum[s] = ti
			}
			recs, _ := WalkKeyframeRecords(pay, tc.reg, tc.fc.ContexteDeLecture())
			for _, r := range recs {
				marche[uint32(r.Slot)] = uint32(r.TI) //nolint:gosec // bornes par le walker
			}
		}
		LierTableDeDatums(w, data, pks)
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
			mar := t519Marcher(pay, w, tc.cfg, debut)
			if s, ok := t519SlotRejete(pay, tc.cfg, mar); ok {
				parSlot[s]++
			}
		}
	}
	t.Logf("SLOTS REJETES PAR VOLUME — %d slots distincts, %d rejets lisibles",
		len(parSlot), t519Somme(parSlot))
	t519Slots(t, parSlot, balaye, datum, marche, 40)
	t.Logf("SOURCES D ARCHETYPE DU FILM : balayeur d image-cle %d slots · table de datums %d · "+
		"marcheur deterministe %d", len(balaye), len(datum), len(marche))
}

// t519SlotRejete rend le slot de l en-tete que la vue B a rejete, quand le paquet est un fautif
// de cette classe. La lecture reproduit `readRecordID` : `low = R(IDLowBits) + IDBase`.
func t519SlotRejete(pay []byte, cfg FrameConfig, mar d519Marche) (uint32, bool) {
	reste := len(pay)*8 - mar.m.FinVueC
	if reste < 0 || (reste <= m5116GateOctet && c514ResteNul(pay, mar.m.FinVueC)) {
		return 0, false
	}
	if !mar.m.PorteA || !mar.m.HitEndB || !mar.m.PorteC {
		return 0, false
	}
	if t515SortieVueB(pay, cfg, mar.m) != "rejet de table de vue" {
		return 0, false
	}
	enTete := 1 + cfg.IDLowBits + 2
	if cfg.HasExtraFields {
		enTete += 32
	}
	low := t515LireBits(pay, mar.m.FinVueB-enTete+1, cfg.IDLowBits)
	if low < 0 {
		return 0, false
	}
	//nolint:gosec // low est borne par IDLowBits <= 15
	return (uint32(low) + cfg.IDBase) & 0x3fffffff, true
}

// t519Slots publie les slots rejetes les plus frequents, avec ce que chaque source declare.
func t519Slots(t *testing.T, parSlot map[uint32]int, balaye, datum, marche map[uint32]uint32,
	n int) {
	t.Helper()
	cles := make([]uint32, 0, len(parSlot))
	for k := range parSlot {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		if parSlot[cles[i]] != parSlot[cles[j]] {
			return parSlot[cles[i]] > parSlot[cles[j]]
		}
		return cles[i] < cles[j]
	})
	tot := t519Somme(parSlot)
	cumul := 0
	for i, s := range cles {
		if i >= n {
			break
		}
		cumul += parSlot[s]
		t.Logf("    slot %6d · %6d rejets (%5.2f %% · cumul %5.1f %%) · balayeur %s · "+
			"datums %s · marcheur %s", s, parSlot[s], m533bPart(parSlot[s], tot),
			m533bPart(cumul, tot), t519Dit(balaye, s), t519Dit(datum, s),
			t519Dit(marche, s))
	}
}

// t519Dit rend l archetype qu une source declare pour un slot, ou « absent ».
func t519Dit(m map[uint32]uint32, s uint32) string {
	if ti, ok := m[s]; ok {
		return fmt.Sprintf("ti=%d", ti)
	}
	return "absent"
}

// t519Somme additionne les occurrences d une ventilation par slot.
func t519Somme(m map[uint32]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

// TestDelies519 dit POURQUOI un slot que DEUX sources d image-cle declarent `ti=35` n est pas
// lie a l instant du rejet. Trois etats possibles, et un seul est une absence :
//
//	jamais lie          aucune source ne l a pose avant ce paquet
//	DELIE PAR UN `DEL`  il etait lie, un record de type 2 l a retire (`World.Unbind`)
//	lie                 impossible par construction (le rejet exige l absence)
//
// `TestRejets519` a montre que les 27 slots les plus rejetes (72 % du volume) sont TOUS declares
// `ti=35` par le balayeur d image-cle ET par la table de datums : ce ne sont donc pas des
// lectures a une position fausse, contrairement a ce que le 5.16.2 concluait sur les slots
// DISTINCTS. Reste a savoir qui les delie.
func TestDelies519(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	delies := map[uint32]int{} // slot -> nombre de `DEL` lus sur lui
	liesUnJour := map[uint32]bool{}
	etats := map[string]int{}
	parSlot := map[uint32]int{}
	apresDel := map[uint32]int{}
	dels := 0
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
				w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
				liesUnJour[uint32(r.Slot)] = true //nolint:gosec // borne par le walker
			}
		}
		LierTableDeDatums(w, data, pks)
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
			mar := t519Marcher(pay, w, tc.cfg, debut)
			if s, ok := t519SlotRejete(pay, tc.cfg, mar); ok {
				parSlot[s]++
				switch {
				case delies[s] > 0:
					etats["DELIE par un record de type 2 (DEL)"]++
					apresDel[s]++
				case liesUnJour[s]:
					etats["lie un jour par une image-cle, plus lie maintenant"]++
				default:
					etats["jamais lie par aucune source avant ce paquet"]++
				}
			}
			for _, r := range mar.recs {
				if r.Type == recDel {
					delies[r.Slot]++
					dels++
				}
				liesUnJour[r.Slot] = true
			}
		}
	}
	t.Logf("RECORDS `DEL` LUS : %d, sur %d slots distincts", dels, len(delies))
	t.Logf("ETAT DU SLOT REJETE A L INSTANT DU REJET :")
	t519Top(t, etats, 6)
	t.Logf("LES VINGT SLOTS LES PLUS REJETES — dont rejets APRES un `DEL` :")
	cles := make([]uint32, 0, len(parSlot))
	for k := range parSlot {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return parSlot[cles[i]] > parSlot[cles[j]] })
	for i, s := range cles {
		if i >= 20 {
			break
		}
		t.Logf("    slot %6d · %6d rejets · dont apres un DEL %6d · DEL lus sur lui %d",
			s, parSlot[s], apresDel[s], delies[s])
	}
}
