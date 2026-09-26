//go:build research

package grammar

// mouvement_5_23_table_research_test.go — LA TABLE ANTICIPEE : SA CLE, SES CONFLITS, SA
// COUVERTURE (lot 5.23.1).
//
// Une passe, un decodage. L instrument repond a trois questions et a rien d autre :
//
//	LA CLE        les deux bits de tete que les images-cles portent, et ceux que les en-tetes
//	              REJETES presentent. La cle du jeu est le mot de 32 bits entier
//	              (`FUN_1406caad8` : `*(uint *)(slot * 200 + base) != eid` -> return 3) ; si
//	              les deux cotes ne portent qu une seule valeur de tete, la cle `(slot, tete)`
//	              et la cle `slot` coincident sur ce film, et il faut le DIRE.
//	LES CONFLITS  combien de cles `(slot, tete)` sont portees par plus d un archetype — la
//	              reutilisation de slot, celle que la datation par chunk arbitre.
//	LA COUVERTURE sur les rejets mesures, combien la table resout (attendu ~74,7 %, 5.20.2).
//
// Rejouable (un film a la fois) :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestTable523$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// a523Bilan cumule la mesure de couverture.
type a523Bilan struct {
	rejets      int
	resolus     int            // la table donne un archetype declare par un chunk POSTERIEUR
	horsTable   int            // aucune image-cle du film ne declare cette cle, jamais
	dejaDeclare int            // declaree, mais seulement par un chunk ANTERIEUR ou COURANT
	parTI       map[uint32]int // archetype anticipe -> rejets resolus
	ecart       map[int]int    // distance en chunks entre le rejet et le declarant
	tetesRejet  map[uint8]int  // les deux bits de tete des en-tetes REJETES
	sansTete    int            // rejets qu une cle SANS tete resoudrait et que la cle avec tete manque
}

// TestTable523 construit la table du film et la confronte aux rejets mesures.
func TestTable523(t *testing.T) {
	tc := t516Cadre(t)
	tab := NouvelleTableAnticipee()
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		tab.AjouterChunk(c, data, pks)
	}
	tab.Clore()
	a523PublierTable(t, tab)
	b := a523Bilan{parTI: map[uint32]int{}, ecart: map[int]int{}, tetesRejet: map[uint8]int{}}
	a523Marcher(t, tc, tab, &b)
	a523PublierCouverture(t, &b)
}

// a523PublierTable publie la table elle-meme : volume, cles, conflits, tetes.
func a523PublierTable(t *testing.T, tab *TableAnticipee) {
	t.Helper()
	t.Logf("TABLE ANTICIPEE : %d declarations d image-cle · %d cles (slot, tete) distinctes · "+
		"%d cles a PLUS D UN archetype", tab.Declarations(), tab.Entrees(), tab.Conflits())
	tetes := make([]int, 0, len(tab.Tetes()))
	for k := range tab.Tetes() {
		tetes = append(tetes, int(k))
	}
	sort.Ints(tetes)
	for _, k := range tetes {
		t.Logf("  tete %d (bits 30-31 de l eid d image-cle) : %d declarations",
			k, tab.Tetes()[uint8(k)]) //nolint:gosec // k vient d un uint8
	}
}

// a523Marcher rejoue la marche du gate et interroge la table a chaque rejet.
func a523Marcher(t *testing.T, tc t516Temoin, tab *TableAnticipee, b *a523Bilan) {
	t.Helper()
	w := NewWorld(tc.reg)
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
			if id, ok := a523IdRejete(pay, tc.cfg, mar); ok {
				a523Compter(tab, id, c, b)
			}
		}
	}
}

// a523Compter classe UN rejet.
func a523Compter(tab *TableAnticipee, id uint32, chunk int, b *a523Bilan) {
	b.rejets++
	b.tetesRejet[uint8(id>>30)]++
	if ti, declarant, ok := tab.ArchetypeApres(id, chunk); ok {
		b.resolus++
		b.parTI[ti]++
		b.ecart[declarant-chunk]++
		return
	}
	// La cle SANS tete : ce que la table resoudrait si les deux bits de tete etaient ignores.
	// L ecart entre les deux chiffres est le PRIX de la cle du jeu — s il est nul, les deux
	// cles coincident sur ce film.
	for tete := uint8(0); tete < 4; tete++ {
		if tete == uint8(id>>30) {
			continue
		}
		if _, _, ok := tab.ArchetypeApres(id&0x3fffffff|uint32(tete)<<30, chunk); ok {
			b.sansTete++
			break
		}
	}
	if _, _, ok := tab.ArchetypeApres(id, -1); ok {
		b.dejaDeclare++
		return
	}
	b.horsTable++
}

// a523PublierCouverture publie le tableau de couverture.
func a523PublierCouverture(t *testing.T, b *a523Bilan) {
	t.Helper()
	t.Logf("REJETS %d · RESOLUS PAR LA TABLE %d (%.1f %%) · declares seulement AVANT %d (%.1f %%) ·"+
		" hors table %d (%.1f %%)",
		b.rejets, b.resolus, m533bPart(b.resolus, b.rejets),
		b.dejaDeclare, m533bPart(b.dejaDeclare, b.rejets),
		b.horsTable, m533bPart(b.horsTable, b.rejets))
	t.Logf("  ce qu une cle SANS les deux bits de tete resoudrait EN PLUS : %d", b.sansTete)
	tetes := make([]int, 0, len(b.tetesRejet))
	for k := range b.tetesRejet {
		tetes = append(tetes, int(k))
	}
	sort.Ints(tetes)
	for _, k := range tetes {
		t.Logf("  tete %d des en-tetes REJETES : %d", k, b.tetesRejet[uint8(k)]) //nolint:gosec // k vient d un uint8
	}
	ecarts := make([]int, 0, len(b.ecart))
	for k := range b.ecart {
		ecarts = append(ecarts, k)
	}
	sort.Ints(ecarts)
	for _, k := range ecarts {
		t.Logf("  declarant a +%d chunk(s) : %d rejets", k, b.ecart[k])
	}
	tis := make([]int, 0, len(b.parTI))
	for k := range b.parTI {
		tis = append(tis, int(k))
	}
	sort.Slice(tis, func(i, j int) bool {
		return b.parTI[uint32(tis[i])] > b.parTI[uint32(tis[j])] //nolint:gosec // index d archetype
	})
	for i, k := range tis {
		if i >= 15 {
			break
		}
		t.Logf("  ti=%d : %d rejets resolus", k, b.parTI[uint32(k)]) //nolint:gosec // index d archetype
	}
}

// a523IdRejete rend l eid COMPLET (tete comprise) de l en-tete que la vue B a rejete.
//
// C est `t519SlotRejete` (lot 5.19) dont il ne jette PAS les deux bits de tete : la cle que
// `FUN_1406caad8` compare est le mot de 32 bits entier, et l instrument doit pouvoir dire si
// la tete discrimine. L instrument du 5.19 n est pas touche (regle 7).
func a523IdRejete(pay []byte, cfg FrameConfig, mar d519Marche) (uint32, bool) {
	if _, ok := t519SlotRejete(pay, cfg, mar); !ok {
		return 0, false
	}
	enTete := 1 + cfg.IDLowBits + 2
	if cfg.HasExtraFields {
		enTete += 32
	}
	debut := mar.m.FinVueB - enTete + 1
	low := t515LireBits(pay, debut, cfg.IDLowBits)
	tete := t515LireBits(pay, debut+cfg.IDLowBits, 2)
	if low < 0 || tete < 0 {
		return 0, false
	}
	//nolint:gosec // low est borne par IDLowBits <= 15, tete par 2 bits
	return ((uint32(low)+cfg.IDBase)&0x3fffffff | uint32(tete)<<30), true
}

// a523Filtrer rend une COPIE de la table restreinte aux archetypes nommes. C est l instrument
// qui cherche la cause d un oracle qui bouge : quel archetype anticipe fait deborder un paquet ?
// Rien de ceci n entre en production — la table de production n a pas de filtre.
func a523Filtrer(tab *TableAnticipee, tis map[uint32]bool) *TableAnticipee {
	out := NouvelleTableAnticipee()
	for cle, decls := range tab.entrees {
		var garde []declarationAnticipee
		for _, d := range decls {
			if tis[d.ti] {
				garde = append(garde, d)
			}
		}
		if len(garde) > 0 {
			out.entrees[cle] = garde
			out.declarations += len(garde)
		}
	}
	return out
}

// a523TIDemandes lit `MOUV523_TI` (liste d archetypes separes par des virgules). Vide = tous.
func a523TIDemandes() map[uint32]bool {
	v := os.Getenv("MOUV523_TI")
	if v == "" {
		return nil
	}
	out := map[uint32]bool{}
	for _, s := range strings.Split(v, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n < 0 {
			continue
		}
		out[uint32(n)] = true //nolint:gosec // n >= 0, borne par le registre
	}
	return out
}
