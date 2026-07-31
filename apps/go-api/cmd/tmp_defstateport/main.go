// tmp_defstateport — THROWAWAY (TRACK B). NE COMMIT JAMAIS. Lecture seule.
//
// OBJET : port DRAFT + diagnostic de la "default-state" consommée par le deser
// vtable[0x60] du descripteur d'archétype runtime (biped #35) dans le keyframe type-2.
//
// CE QUE GHIDRA A ÉTABLI (assembleur de FUN_141f86704 @141f868c2) :
//
//	R11 = param_1 = contexte de décode runtime.
//	typeIndex = EDI = R(6) lu juste avant (channel-type 1, cf. FUN_1406cd128).
//	plVar1 = *(*(R11+0x18) + 8 + typeIndex*8)         // table de PTRS d'archétypes, stride 8
//	size1  = (*(plVar1->vt + 0x20))(plVar1)           // taille buffer default-state (bytes)
//	size2  = (*(plVar1->vt + 0x10))(plVar1)           // 2e taille
//	_Dst   = alloc(size1); memset(_Dst,0,size1)
//	ok     = (*(plVar1->vt + 0x60))(plVar1, size1, _Dst, bitreader, 1)   // <== DESER DEFAULT-STATE
//	ok2    = (*(plVar1->vt + 0x88))(plVar1, size1, _Dst, size2, lVar7)   // deser2
//	(*(plVar1->vt + 0x30))(plVar1, ...)               // finalize
//
// => vtable[0x60] est BIEN le deser-default-state, 5 args, qui consomme un nombre
//
//	de bits VARIABLE et auto-délimité (la "default-state" 380-bits du record Hydra
//	n'est PAS un Skip de largeur fixe).
//
// BLOCAGE STATIQUE (cf. rapport) : la table `param_1+0x18` est construite À L'EXÉCUTION
// (relocs non tracées par Ghidra). Impossible de résoudre statiquement QUEL pointeur de
// descripteur correspond à typeIndex=35, donc impossible de lire vtable[0x60] concret.
// La vtable de DÉFINITION du tag "biped" (143737138 : get_type=35, get_name="biped")
// existe mais n'est PAS la vtable du descripteur de réplication (son [0x20] retourne un
// byte-flag, pas une taille de buffer ; son [0x60] = MOV EAX,0x98;RET = getter, pas deser).
//
// PREUVE EMPIRIQUE (ce programme) : à defaultBits FIXE, seul le record déjà calibré (Hydra,
// 380) traverse jusqu'au bout. Chaque autre biped exige une largeur DIFFÉRENTE. => la
// default-state est content-dependent : un Skip(380) ne peut pas la porter en général.
//
// Ce port NE PEUT PAS être bit-exact tant que la grammaire de vtable[0x60] n'est pas
// décompilée. Il fournit : (1) la confirmation que 380 n'est pas constant, (2) la mesure
// de la largeur réelle consommée par TraverseEntity pour chaque biped porteur, (3) le
// squelette d'appel attendu pour brancher le futur deser bit-exact.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

// Calibration de référence (acquis du brief).
const (
	hydraStart       = 194126
	hydraDefaultBits = 380
	hydraRSP         = 2
	hydraEndBit      = 195892
	bipedTypeIndex   = 35
)

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func extractType2(d []byte) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == 2 {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func knownHigh32(v uint32) bool {
	for id := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return true
		}
	}
	return false
}

type lit struct {
	bit  int
	id64 uint64
	name string
}

func collectLits(payload []byte) []lit {
	var lits []lit
	total := len(payload) * 8
	for bp := 0; bp+64 <= total; bp++ {
		v := uint32(bitsAt(payload, bp, 32))
		if !knownHigh32(v) {
			continue
		}
		low := uint32(bitsAt(payload, bp+32, 32))
		id64 := (uint64(v) << 32) | uint64(low)
		if nm, ok := analysis.WeaponIDToName[id64]; ok {
			lits = append(lits, lit{bp, id64, nm})
		}
	}
	sort.Slice(lits, func(i, j int) bool { return lits[i].bit < lits[j].bit })
	return lits
}

// bestCarrierWidth balaie (start #35, d) dans une large fenêtre amont du gateBit et
// retourne le couple (start, d) qui place un WST EXACT sur gateBit (id64 attendu) avec
// la traversée la plus saine. C'est la mesure FIABLE de la largeur réelle de default-state
// de CE biped (à la différence de "le start #35 le plus proche", souvent un faux positif
// R(6)==35 fortuit dans les bits de données).
func bestCarrierWidth(reg *filmdec.Registry, payload []byte, gateBit int, id64 uint64) (bestStart, bestD, bestDesync int, found bool) {
	bestEff := -1
	lo := gateBit - 3200
	if lo < 0 {
		lo = 0
	}
	for start := lo; start <= gateBit-20; start++ {
		if uint32(bitsAt(payload, start, 6)) != bipedTypeIndex {
			continue
		}
		for d := 1; d <= 420; d++ {
			filmdec.SetRecordStateParam(hydraRSP)
			br := filmdec.NewBitReader(payload)
			br.Skip(start)
			t := filmdec.TraverseEntity(br, reg, d)
			if t.TypeIndex != bipedTypeIndex {
				continue
			}
			for _, c := range t.Comps {
				if c.Name != "weapon-state-type-info" || c.StartBit != gateBit {
					continue
				}
				h := uint32(bitsAt(payload, gateBit+1, 32))
				v := uint32(bitsAt(payload, gateBit+33, 32))
				if (uint64(h)<<32)|uint64(v) != id64 {
					continue
				}
				eff := t.DesyncAt
				if eff == -1 {
					eff = 1 << 30
				}
				if eff > bestEff {
					bestEff, bestStart, bestD, bestDesync, found = eff, start, d, t.DesyncAt, true
				}
			}
		}
	}
	return
}

func main() {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	lits := collectLits(payload)

	fmt.Printf("=== TRACK B — port DRAFT / diagnostic vtable[0x60] (deser default-state biped #35) ===\n")
	fmt.Printf("registre=%d archétypes ; keyframe type-2=%d octets ; %d littéraux d'arme\n\n",
		len(reg.Archetypes), len(payload), len(lits))

	// (1) VALIDATION DE RÉFÉRENCE : Skip(380) sur le record Hydra atteint-il le gate @194512
	//     et traverse-t-il jusqu'à -1 ?  (380 bits = de bit194132 après R(6) à bit194512.)
	fmt.Printf("--- (1) VALIDATION record Hydra : defaultBits=%d, rsp=%d, start=%d ---\n", hydraDefaultBits, hydraRSP, hydraStart)
	filmdec.SetRecordStateParam(hydraRSP)
	br := filmdec.NewBitReader(payload)
	br.Skip(hydraStart)
	t := filmdec.TraverseEntity(br, reg, hydraDefaultBits)
	gateExpected := hydraStart + 6 + hydraDefaultBits // 194132 + 380 = 194512
	fmt.Printf("    typeIndex=%d (attendu %d) desyncAt=i%d ncomp=%d endBit=%d (attendu %d)\n",
		t.TypeIndex, bipedTypeIndex, t.DesyncAt, len(t.Comps), t.EndBit, hydraEndBit)
	fmt.Printf("    gate par construction = start+6+380 = %d  (R(6) typeIndex @%d, default-state [%d..%d[)\n",
		gateExpected, hydraStart, hydraStart+6, gateExpected)
	hydraOK := t.TypeIndex == bipedTypeIndex && t.DesyncAt == -1 && t.EndBit == hydraEndBit
	fmt.Printf("    => 380 bits consomment EXACTEMENT la default-state du record Hydra : %v\n\n", boolMark(hydraOK))

	// (2) TEST DE CONSTANCE : 380 est-il une CONSTANTE d'archétype biped #35 ?
	//     On mesure, pour chaque autre biped porteur d'arme, la largeur réelle nécessaire.
	fmt.Printf("--- (2) La default-state est-elle de largeur CONSTANTE (=380) par archétype ? ---\n")
	fmt.Printf("    Pour chaque WST d'arme : balayage (start #35, d) sur la fenêtre amont, on retient\n")
	fmt.Printf("    le couple qui place le WST sur son gate avec la traversée la plus saine.\n")
	widths := map[int]int{}
	type row struct {
		name           string
		gateBit, start int
		d, desync      int
		found          bool
	}
	var rows []row
	for _, l := range lits {
		gateBit := l.bit - 1
		start, d, dsc, found := bestCarrierWidth(reg, payload, gateBit, l.id64)
		if found {
			widths[d]++
		}
		rows = append(rows, row{l.name, gateBit, start, d, dsc, found})
	}
	for _, r := range rows {
		if !r.found {
			fmt.Printf("    %-16s gate=%-7d : (pas de largeur saine trouvée depuis start=%d)\n", r.name, r.gateBit, r.start)
			continue
		}
		tag := ""
		if r.desync == -1 {
			tag = "  traversée COMPLÈTE"
		}
		fmt.Printf("    %-16s gate=%-7d start=%-7d : largeur réelle d=%-3d (desyncAt=i%d)%s\n",
			r.name, r.gateBit, r.start, r.d, r.desync, tag)
	}
	fmt.Printf("\n    largeurs distinctes mesurées : ")
	ws := make([]int, 0, len(widths))
	for w := range widths {
		ws = append(ws, w)
	}
	sort.Ints(ws)
	for _, w := range ws {
		fmt.Printf("%d(×%d) ", w, widths[w])
	}
	fmt.Printf("\n")
	verdictConstant := len(widths) == 1
	fmt.Printf("    => largeur de default-state CONSTANTE par archétype : %v\n", boolMark(verdictConstant))
	if !verdictConstant {
		fmt.Printf("    => CONFIRME le verdict Ghidra : vtable[0x60] consomme un nombre de bits VARIABLE\n")
		fmt.Printf("       et auto-délimité. 380 est SPÉCIFIQUE au record Hydra, pas une constante biped.\n")
	}

	// (3) SQUELETTE D'APPEL pour le futur deser bit-exact (à brancher quand vtable[0x60]
	//     sera décompilée). Documente l'interface attendue.
	fmt.Printf("\n--- (3) Interface attendue du futur deser-default-state (port cible) ---\n")
	fmt.Printf("    func deserDefaultStateBiped(br *BitReader, size1 int, dst []byte) (bitsConsumed int, ok bool)\n")
	fmt.Printf("      // size1 = vtable[0x20](descripteur) ; dst = make([]byte, size1) ; le retour\n")
	fmt.Printf("      //  bitsConsumed DOIT valoir 380 pour le record Hydra (start+6 -> gate %d).\n", gateExpected)
	fmt.Printf("      // Tant que la grammaire bit n'est pas décompilée, TraverseEntity(br, reg, d)\n")
	fmt.Printf("      //  avec d=largeur mesurée reste le seul substitut (NON bit-exact en général).\n")

	fmt.Printf("\n=== FIN. 380 bits sur record Hydra : %v ; default-state constante : %v ===\n",
		boolMark(hydraOK), boolMark(verdictConstant))
}

func boolMark(b bool) string {
	if b {
		return "OUI"
	}
	return "NON"
}
