// tmp_liveinv — PRODUIRE l'inventaire VIVANT par joueur et par image, pour la colonne d'etat
// du POC (item 6 de la liste Notion).
//
// CE QUE L'ITEM 6 DEMANDE : « live player status (including shield, health, weapon/grenade in
// hand/live inventory) », plus un compteur de reapparition, plus les capacites (6.1) et les
// munitions (6.2). Le bouclier et la vie sont deja dans le POC ; le reste manquait parce que
// les composants n'etaient pas decodes. Ils le sont depuis le 2026-07-27.
//
// LE PONT ENTITE -> SLOT, ET POURQUOI IL EST DIRECT. La capture journalise l'entite (0x40000200
// par exemple) ; le POC indexe par slot. La grammaire de record donne `slot = id & 0x3fffffff`,
// donc 0x40000200 -> 512, et 512 est precisement un slot present dans les loadouts du POC.
// Aucune heuristique n'est necessaire.
//
// CE QUE CE PROGRAMME LIT, ET D'OU VIENT CHAQUE GRAMMAIRE :
//
//	i22  comptes de grenades   R(3)=4 puis 4xR(8), valeurs 0/1/2   (relu aux 249 positions exactes)
//	i47  jeu de grenades       [6 bits masque][3 bits selection]   (12 valeurs, selection toujours
//	                                                                dans le masque, 12/12)
//	i48  capacite equipee      10 bits                              (218 lectures, 13 valeurs)
//	i57  capacite ACTIVE       2 bits, bit 0 = interrupteur         (990 lectures, bit 0 a 48 %)
//	i43  arme emplacement 1    identifiant 64 bits a +1, suffixe 0x42C9679F a +33
//	i44  arme emplacement 2    idem
//	i42  selecteur d'arme      7 bits, deux valeurs dominantes separees d'un bit
//
// RESERVE A DIRE EN CLAIR : cette extraction depend de la CAPTURE Cheat Engine, donc elle ne
// vaut que pour le film capture. Les 948 autres films demandent le scan hors ligne, qui n'est
// pas encore assez precis. Ce livrable est un POC, pas une fonctionnalite.
package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	recSize      = 40
	weaponSuffix = 0x42C9679F
	minDist      = 12
)

type hit struct {
	EID, TypeIndex, CompIndex, BitCursor uint32
	Sig                                  [16]byte
}

// state est l'etat d'un joueur a un instant, tel qu'on sait le lire.
type state struct {
	Frame int      `json:"f"`
	Slot  uint32   `json:"s"`
	Gren  []int    `json:"g,omitempty"`  // comptes par type, 4 entrees
	GSel  int      `json:"gs,omitempty"` // type de grenade selectionne, 1..4 (0 = aucun)
	GMask int      `json:"gm,omitempty"` // masque des types possedes
	Abil  int      `json:"a,omitempty"`  // identifiant de capacite equipee (index de palette)
	Act   int      `json:"ac,omitempty"` // capacite active : 1 = oui
	Weap  []string `json:"w,omitempty"`  // armes des deux emplacements, en hexadecimal
	Sel   int      `json:"ws,omitempty"` // valeur du selecteur i42
}

func main() {
	in := flag.String("in", "", "dump binaire de la capture CE")
	dir := flag.String("film", "", "dossier des chunks du film")
	out := flag.String("out", "", "fichier JSON de sortie")
	hz := flag.Float64("hz", 8, "images par seconde du document de rejeu")
	flag.Parse()
	if *in == "" || *dir == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_liveinv -in <capture.bin> -film <dir> [-out inv.json]")
		os.Exit(2)
	}

	hits, err := readHits(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// On n'indexe que les composants utiles a la colonne d'etat.
	want := map[uint32]bool{22: true, 42: true, 43: true, 44: true, 47: true, 48: true, 57: true}
	sigIdx := map[uint64][]hit{}
	for _, h := range hits {
		if h.TypeIndex != 35 || !want[h.CompIndex] || !usable(h.Sig) {
			continue
		}
		k := binary.LittleEndian.Uint64(h.Sig[:8])
		sigIdx[k] = append(sigIdx[k], h)
	}
	fmt.Printf("INVENTAIRE VIVANT — %d signatures indexees\n\n", len(sigIdx))

	type reading struct {
		slot  uint32
		comp  uint32
		frame int
		pay   []byte
		cur   int
	}
	var rs []reading
	var originUS uint64
	nc := filmdec.CountFilmChunks(*dir)
	for c := 1; c <= nc; c++ {
		chunk, err := filmdec.ReadFilmChunk(*dir, c)
		if err != nil {
			continue
		}
		type pkt struct {
			start, size int
			ts          uint64
		}
		var pkts []pkt
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			pkts = append(pkts, pkt{p.Start, p.Size, p.TimestampUS})
			if originUS == 0 || p.TimestampUS < originUS {
				originUS = p.TimestampUS
			}
		}
		for off := 0; off+16 <= len(chunk); off++ {
			cands, ok := sigIdx[binary.LittleEndian.Uint64(chunk[off:off+8])]
			if !ok {
				continue
			}
			for _, h := range cands {
				same := true
				for j := 0; j < 16; j++ {
					if chunk[off+j] != h.Sig[j] {
						same = false
						break
					}
				}
				if !same {
					continue
				}
				for _, p := range pkts {
					if off < p.start || off >= p.start+p.size {
						continue
					}
					rs = append(rs, reading{h.EID & 0x3fffffff, h.CompIndex, int(p.ts),
						chunk[p.start : p.start+p.size], int(h.BitCursor)})
					break
				}
			}
		}
	}
	fmt.Printf("  %d lectures localisees dans le film\n", len(rs))

	// Regroupement par (slot, instant) : un record ne porte qu'une lecture par composant.
	type key struct {
		slot  uint32
		frame int
	}
	byKey := map[key]*state{}
	for _, r := range rs {
		f := int(float64(uint64(r.frame)-originUS) / 1e6 * *hz)
		k := key{r.slot, f}
		st := byKey[k]
		if st == nil {
			st = &state{Frame: f, Slot: r.slot}
			byKey[k] = st
		}
		applyComponent(st, r.comp, r.pay, r.cur)
	}

	list := make([]*state, 0, len(byKey))
	for _, s := range byKey {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Frame != list[j].Frame {
			return list[i].Frame < list[j].Frame
		}
		return list[i].Slot < list[j].Slot
	})

	nG, nA, nW := 0, 0, 0
	slots := map[uint32]bool{}
	for _, s := range list {
		slots[s.Slot] = true
		if s.GMask > 0 || len(s.Gren) > 0 {
			nG++
		}
		if s.Abil > 0 {
			nA++
		}
		if len(s.Weap) > 0 {
			nW++
		}
	}
	fmt.Printf("  %d etats sur %d slots distincts\n", len(list), len(slots))
	fmt.Printf("    dont %d avec grenades · %d avec capacite · %d avec arme\n\n", nG, nA, nW)

	for i, s := range list {
		if i >= 10 {
			break
		}
		fmt.Printf("    f=%-5d slot=%-5d gren=%v sel=%d abil=%d actif=%d armes=%v\n",
			s.Frame, s.Slot, s.Gren, s.GSel, s.Abil, s.Act, s.Weap)
	}

	if *out != "" {
		b, err := json.Marshal(list)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(*out, b, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("\n  -> %s (%d octets)\n", *out, len(b))
	}
}

// applyComponent lit UN composant a sa position exacte et remplit l'etat. Chaque grammaire est
// celle mesuree le 2026-07-27, pas une hypothese.
func applyComponent(st *state, comp uint32, pay []byte, cur int) {
	total := len(pay) * 8
	switch comp {
	case 22: // comptes de grenades : R(3)=4 puis 4 x R(8)
		if cur+35 > total || filmdec.PeekBits(pay, cur, 3) != 4 {
			return
		}
		g := make([]int, 4)
		for i := 0; i < 4; i++ {
			v := filmdec.PeekBits(pay, cur+3+i*8, 8)
			if v > 2 {
				return // hors bornes du jeu : lecture douteuse, on n'ecrit rien
			}
			g[i] = int(v)
		}
		st.Gren = g
	case 47: // [6 bits masque][3 bits selection], la selection appartient au masque
		if cur+9 > total {
			return
		}
		v := filmdec.PeekBits(pay, cur, 9)
		mask, sel := int((v>>3)&0x3F), int(v&7)
		if mask&0x30 != 0 || sel > 4 {
			return
		}
		if sel > 0 && mask&(1<<(sel-1)) == 0 {
			return // selection hors masque : incoherent
		}
		st.GMask, st.GSel = mask, sel
	case 48: // capacite equipee : index de palette sur 10 bits
		if cur+10 <= total {
			st.Abil = int(filmdec.PeekBits(pay, cur, 10))
		}
	case 57: // capacite active : bit 0 = interrupteur
		if cur+1 <= total {
			st.Act = int(filmdec.PeekBits(pay, cur, 1))
		}
	case 42: // selecteur d'arme
		if cur+7 <= total {
			st.Sel = int(filmdec.PeekBits(pay, cur, 7))
		}
	case 43, 44: // identifiant d'arme 64 bits a +1, suffixe verifie a +33
		b := cur + 1
		if b+64 > total || uint32(filmdec.PeekBits(pay, b+32, 32)) != weaponSuffix {
			return
		}
		id := fmt.Sprintf("0x%08X", uint32(filmdec.PeekBits(pay, b, 64)>>32))
		for len(st.Weap) < 2 {
			st.Weap = append(st.Weap, "")
		}
		if comp == 43 {
			st.Weap[0] = id
		} else {
			st.Weap[1] = id
		}
	}
}

func usable(s [16]byte) bool {
	seen := map[byte]bool{}
	for _, b := range s {
		seen[b] = true
	}
	return len(seen) >= minDist
}

func readHits(path string) ([]hit, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture %s : %w", path, err)
	}
	out := make([]hit, 0, len(raw)/recSize)
	for i := 0; i+recSize <= len(raw); i += recSize {
		b := raw[i : i+recSize]
		var h hit
		h.EID = binary.LittleEndian.Uint32(b[0:])
		h.TypeIndex = binary.LittleEndian.Uint32(b[4:])
		h.CompIndex = binary.LittleEndian.Uint32(b[8:])
		h.BitCursor = binary.LittleEndian.Uint32(b[16:])
		copy(h.Sig[:], b[24:40])
		if h.EID == 0 && h.TypeIndex == 0 && h.BitCursor == 0 && h.CompIndex == 0 {
			break
		}
		out = append(out, h)
	}
	return out, nil
}
