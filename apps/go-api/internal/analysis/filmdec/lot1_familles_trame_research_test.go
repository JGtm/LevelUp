package filmdec

// lot1_familles_trame_research_test.go — LOT 1 DU PLAN « PERCER LA TRAME » (2026-08-30) :
// LES PAQUETS ECARTES (0xD3, 528 262 SUR LE CORPUS) SONT-ILS DES TRAMES ORDINAIRES ?
//
// CE QUE LE LOT D A LAISSE OUVERT (point 6 de sa note) : le modele « un paquet delta = une
// trame de records » n'a pas de confirmation POSITIVE — S3/S4 disent ce que le premier octet
// n'est PAS. La confirmation proposee : rejouer le decodeur de production sur les familles
// de premier octet et verifier que la chaine de records s'y ferme proprement. C'est CET
// instrument.
//
// MECANIQUE : identique au temoin de marche (delta_walk_witness_test.go) — monde amorce par
// les images-cles du chunk courant, marche sequentielle DecodeFrameRecords, memes chunks,
// meme config — mais VENTILEE PAR PREMIER OCTET du payload. La reference interne est 0xA0
// (la trame de tick, 80 % du corpus) : ce qui compte n'est pas le taux absolu (la marche
// sequentielle s'arrete au premier composant non porte, ~20 % des records) mais l'ECART
// entre une famille et la reference SUR LE MEME FILM avec le MEME decodeur.
//
// CRITERES ECRITS AVANT LA MESURE :
//
//	L1-C1 (confirmation positive) — une famille est une TRAME ORDINAIRE si sa part de
//	      paquets fermes proprement (fin de chaine atteinte, sans desync) vaut au moins
//	      50 % de celle de 0xA0 sur le meme film. Vise : 0xD2 et 0xD3.
//	L1-C2 (recoupement lot D par une voie differente) — le nombre de SLOTS DISTINCTS du
//	      premier record decode : au plus 10 pour 0xD2, entre 30 et 70 pour 0xD3 (le lot D
//	      a mesure 7 et 50 par fenetre de bits fixe ; ici on DECODE l'en-tete du record).
//	L1-C3 (recensement, sans seuil) — les archetypes (ti) des premiers records par famille,
//	      publies : c'est la carte qui dira OU vivent l'arme, la visee et la victime.
//
// Garde LOT1_TRAME_FILM : UN repertoire de film par process (memoire du depot : un film =
// un process). Lecture seule, verrou de decodage pris, aucun code de production touche.

import (
	"fmt"
	"math/bits"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const lot1TrameFilmEnv = "LOT1_TRAME_FILM"

// lot1Famille agrege la mesure d'une famille de premier octet.
type lot1Famille struct {
	packets, cleanEnd, records, walked int
	firstSlots                         map[uint32]int // slot du 1er record -> paquets
	firstTIs                           map[string]int // "ti=N"/"del"/"new"/"desync" du 1er record -> paquets
	// delTIs : pour un 1er record DEL, l'archetype auquel le slot etait LIE au moment de la
	// suppression (interroge AVANT le decode, qui delie) — c'est l'identite de ce qui meurt.
	delTIs map[string]int
	// suiteTIs : les records qui SUIVENT le 1er (recs[1:]) — ou vit la charge utile reelle.
	suiteTIs map[string]int
}

// TestLot1FamillesTrame ventile la marche de production par premier octet du payload et rend
// les verdicts L1-C1/C2 (et le recensement C3) pour le film de la garde.
func TestLot1FamillesTrame(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible dans %s : %v", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible dans %s : %v", dir, err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks // meme borne que le temoin de marche : cout et RAM contenus
	}
	cfg := DefaultFrameConfig()
	fams := map[byte]*lot1Famille{}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		w := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			f := fams[pay[0]]
			if f == nil {
				f = &lot1Famille{firstSlots: map[uint32]int{}, firstTIs: map[string]int{},
					delTIs: map[string]int{}, suiteTIs: map[string]int{}}
				fams[pay[0]] = f
			}
			f.packets++
			// Pre-lecture du 1er en-tete AVANT le decode : un DEL delie son slot, il faut
			// interroger le monde avant pour savoir ce qui meurt.
			peek := NewBitReader(pay)
			peek.Skip(cfg.PacketPreambleBits)
			if readRecordType(peek) == recDel {
				slot := readRecordID(peek, cfg.IDLowBits, cfg.IDBase) & 0x3fffffff
				if ti, bound := w.ArchetypeForSlot(slot); bound {
					f.delTIs[fmt.Sprintf("ti=%d", ti)]++
				} else {
					f.delTIs["non-lie"]++
				}
			}
			br := NewBitReader(pay)
			recs, decErr := DecodeFrameRecords(br, w, cfg)
			if decErr == nil {
				f.cleanEnd++
			}
			for i := range recs {
				f.records++
				if recs[i].DesyncAt == -1 {
					f.walked++
				}
			}
			if len(recs) > 0 {
				r0 := recs[0]
				f.firstSlots[r0.Slot]++
				switch {
				case r0.Type == recDel:
					f.firstTIs["del"]++
				case r0.DesyncAt != -1 && r0.TypeIndex == 0 && len(r0.Trace.Comps) == 0:
					f.firstTIs["desync-avant-ti"]++
				case r0.Type == recNew:
					f.firstTIs[fmt.Sprintf("new ti=%d", r0.TypeIndex)]++
				default:
					f.firstTIs[fmt.Sprintf("ti=%d", r0.TypeIndex)]++
				}
				for _, r := range recs[1:] {
					switch {
					case r.Type == recDel:
						f.suiteTIs["del"]++
					case r.DesyncAt != -1 && r.TypeIndex == 0 && len(r.Trace.Comps) == 0:
						f.suiteTIs["desync-avant-ti"]++
					case r.Type == recNew:
						f.suiteTIs[fmt.Sprintf("new ti=%d", r.TypeIndex)]++
					default:
						f.suiteTIs[fmt.Sprintf("ti=%d", r.TypeIndex)]++
					}
				}
			} else {
				f.firstTIs["trame-vide"]++
			}
		}
	}

	id := filepath.Base(filepath.Clean(dir))
	t.Logf("== FILM %s (%d chunk(s) de replication) ==", id, n)
	ref := fams[0xA0]
	if ref == nil || ref.packets == 0 {
		t.Fatalf("aucun paquet 0xA0 : pas de reference interne, film inhabituel")
	}
	refClean := lot1Pct(ref.cleanEnd, ref.packets)
	var octets []int
	for o := range fams {
		octets = append(octets, int(o))
	}
	sort.Slice(octets, func(i, j int) bool { return fams[byte(octets[i])].packets > fams[byte(octets[j])].packets })
	for _, o := range octets {
		f := fams[byte(o)]
		if f.packets < 20 {
			continue // trop peu pour publier un taux
		}
		t.Logf("  0x%02X : %6d paquets · fermes proprement %6d (%.1f %%) · records %7d "+
			"(aboutis %.1f %%) · slots distincts du 1er record : %d",
			o, f.packets, f.cleanEnd, lot1Pct(f.cleanEnd, f.packets), f.records,
			lot1Pct(f.walked, f.records), len(f.firstSlots))
		t.Logf("         1er record : %s", lot1Top(f.firstTIs, 6))
		if len(f.delTIs) > 0 {
			t.Logf("         ce que le DEL supprime (archetype lie avant decode) : %s",
				lot1Top(f.delTIs, 8))
		}
		if len(f.suiteTIs) > 0 {
			t.Logf("         records APRES le 1er : %s", lot1Top(f.suiteTIs, 8))
		}
		if o == 0xD2 || o == 0xD3 {
			var slots []int
			for s := range f.firstSlots {
				slots = append(slots, int(s))
			}
			sort.Ints(slots)
			parts := make([]string, 0, len(slots))
			for _, s := range slots {
				parts = append(parts, fmt.Sprintf("%d(x%d)", s, f.firstSlots[uint32(s)]))
			}
			t.Logf("         slots du 1er record : %s", strings.Join(parts, " "))
		}
	}
	verdictFamille := func(o byte, minSlots, maxSlots int) {
		f := fams[o]
		if f == nil || f.packets == 0 {
			t.Logf("L1 [0x%02X] : famille absente du film — rien a conclure", o)
			return
		}
		clean := lot1Pct(f.cleanEnd, f.packets)
		c1 := clean >= 0.5*refClean
		c2 := len(f.firstSlots) >= minSlots && len(f.firstSlots) <= maxSlots
		t.Logf("L1-C1 [0x%02X] : %.1f %% fermes vs reference 0xA0 %.1f %% (seuil >= %.1f %%) — %s",
			o, clean, refClean, 0.5*refClean, lot1Verdict(c1))
		t.Logf("L1-C2 [0x%02X] : %d slots distincts du 1er record (attendu %d..%d) — %s",
			o, len(f.firstSlots), minSlots, maxSlots, lot1Verdict(c2))
	}
	verdictFamille(0xD2, 1, 10)
	verdictFamille(0xD3, 30, 70)
}

// TestLot1VuesMultiples — HYPOTHESE (ecrite avant la mesure) : les familles a « trame vide »
// (0xC0/0xC2/0xC3/0xC4/0xC7 : fermeture immediate, payload entier NON LU) sont des paquets
// dont la PREMIERE VUE de replication est vide, le contenu vivant dans les vues 2/3 — le
// frame-processor de l'exe (FUN_142987460) lit TROIS vues par paquet, et le decodeur
// sequentiel n'en lit qu'une. PREDICTIONS : (a) sur ces familles, DecodeFrameViews(3) rend
// viewsDone >= 2 et des records la ou la vue 1 n'en rendait aucun, pour au moins la moitie
// des paquets ; (b) la couverture du payload (dernier bit atteint / taille) monte nettement.
// Recensement egalement publie pour 0xA0 (reference) et 0xD2/0xD3 (cibles du lot).
func TestLot1VuesMultiples(t *testing.T) {
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
	cfg := DefaultFrameConfig()
	prev := inferChain
	SetInferChain(true)
	defer SetInferChain(prev)
	type agg struct {
		packets, recs, viewsSum, ge2, avecRecs int
		covSum                                 float64
	}
	fams := map[byte]*agg{}
	cibles := map[byte]bool{0xA0: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xC4: true,
		0xC7: true, 0xD2: true, 0xD3: true}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		w := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			if !cibles[pay[0]] {
				continue
			}
			f := fams[pay[0]]
			if f == nil {
				f = &agg{}
				fams[pay[0]] = f
			}
			f.packets++
			recs, views := DecodeFrameViews(pay, w, cfg, 3, cfg.PacketPreambleBits)
			f.recs += len(recs)
			f.viewsSum += views
			if views >= 2 {
				f.ge2++
			}
			if len(recs) > 0 {
				f.avecRecs++
			}
			end := 0
			for i := range recs {
				if recs[i].Trace.EndBit > end {
					end = recs[i].Trace.EndBit
				}
			}
			f.covSum += float64(end) / float64(len(pay)*8)
		}
	}
	var octets []int
	for o := range fams {
		octets = append(octets, int(o))
	}
	sort.Ints(octets)
	for _, o := range octets {
		f := fams[byte(o)]
		t.Logf("  0x%02X : %6d paquets · records %7d · vues moyennes %.2f · >=2 vues %5d (%.1f %%) · "+
			"paquets avec records %5d (%.1f %%) · couverture moyenne %.1f %%",
			o, f.packets, f.recs, float64(f.viewsSum)/float64(max(1, f.packets)),
			f.ge2, lot1Pct(f.ge2, f.packets), f.avecRecs, lot1Pct(f.avecRecs, f.packets),
			100*f.covSum/float64(max(1, f.packets)))
	}
}

// TestLot1AmorceParFamille — LE BALAYAGE QUI TRANCHE LE CADRAGE PAR FAMILLE.
//
// QUESTION : le deuxieme bit d'amorce (bit 1 du payload) vaut 0 sur 0xA0/0x80 et 1 sur
// 0xC0/0xD2/0xD3/0xE9 — les familles que la marche lit mal. Le desassemblage du
// frame-processor (FUN_142987460) n'etablit qu'UN bit d'amorce ; le second est empirique.
// Si les familles a bit 1 different d'amorce, tout leur decodage est decale.
//
// METHODE : decoder chaque famille sous des amorces candidates k, et departager par le
// discriminant deja etabli du depot (frame_records.go) : la part de records DELTA sur slot
// lie dont le masque compte 1..7 composants — 84,8 % sous la bonne grammaire, 10,7 % au
// niveau du hasard. CRITERES ECRITS AVANT LA MESURE :
//
//	A1 — sur 0xA0, k=2 doit gagner (sinon l'instrument est faux : c'est le temoin).
//	A2 — pour chaque famille a >= 50 records delta lies, le k gagnant est publie ; un
//	     gagnant NET (>= 15 points d'ecart sur le suivant) vaut cadrage etabli pour la
//	     famille ; un palmares serre vaut NON CONCLUANT, publie tel quel.
func TestLot1AmorceParFamille(t *testing.T) {
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
	amorces := []int{1, 2, 3, 4, 5, 6, 8, 10, 12, 16}
	cibles := map[byte]bool{0xA0: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xC7: true,
		0xD2: true, 0xD3: true, 0xE9: true, 0xE5: true}
	type cell struct{ deltasLies, masquesOK, clean, packets int }
	mesure := map[byte]map[int]*cell{}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		wBase := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		snap := wBase.Snapshot()
		for _, k := range amorces {
			cfg := DefaultFrameConfig()
			cfg.PacketPreambleBits = k
			w := NewWorld(reg)
			w.Restore(snap)
			for _, pk := range pks {
				if pk.Type != PacketTypeDelta || pk.Size < 1 {
					continue
				}
				pay := pk.Payload(data)
				if !cibles[pay[0]] {
					continue
				}
				if mesure[pay[0]] == nil {
					mesure[pay[0]] = map[int]*cell{}
				}
				cl := mesure[pay[0]][k]
				if cl == nil {
					cl = &cell{}
					mesure[pay[0]][k] = cl
				}
				cl.packets++
				br := NewBitReader(pay)
				recs, decErr := DecodeFrameRecords(br, w, cfg)
				if decErr == nil {
					cl.clean++
				}
				for i := range recs {
					r := &recs[i]
					nm := bits.OnesCount64(r.Trace.Mask)
					if r.Type == recDelta && r.DesyncAt == -1 && nm > 0 {
						cl.deltasLies++
						if nm <= 7 {
							cl.masquesOK++
						}
					}
				}
			}
		}
	}
	var octets []int
	for o := range mesure {
		octets = append(octets, int(o))
	}
	sort.Ints(octets)
	for _, o := range octets {
		type score struct {
			k    int
			pct  float64
			nrec int
		}
		var sc []score
		for _, k := range amorces {
			cl := mesure[byte(o)][k]
			if cl == nil {
				continue
			}
			sc = append(sc, score{k, lot1Pct(cl.masquesOK, cl.deltasLies), cl.deltasLies})
		}
		sort.Slice(sc, func(i, j int) bool { return sc[i].pct > sc[j].pct })
		parts := make([]string, 0, len(sc))
		for _, s := range sc {
			parts = append(parts, fmt.Sprintf("k=%d %.1f%%(n=%d)", s.k, s.pct, s.nrec))
		}
		t.Logf("  0x%02X : %s", o, strings.Join(parts, " · "))
		// Le palmares du verdict ne retient que les amorces a effectif >= 50 (critere A2 :
		// « pour chaque famille a >= 50 records delta lies ») — sans ce plancher, un k qui ne
		// rend que 2 records a 100 % masquerait le vrai gagnant.
		var el []score
		for _, s := range sc {
			if s.nrec >= 50 {
				el = append(el, s)
			}
		}
		switch {
		case len(el) == 0:
			t.Logf("         aucune amorce a >= 50 deltas lies : NON CONCLUANT")
		case len(el) == 1:
			t.Logf("         GAGNANT k=%d (%.1f %% sur %d deltas lies) — SEUL au-dessus du plancher (pas de rival a departager)",
				el[0].k, el[0].pct, el[0].nrec)
		default:
			net := el[0].pct-el[1].pct >= 15
			t.Logf("         GAGNANT k=%d (%.1f %% sur %d deltas lies, suivant k=%d %.1f %%) — %s",
				el[0].k, el[0].pct, el[0].nrec, el[1].k, el[1].pct,
				map[bool]string{true: "NET", false: "NON CONCLUANT (palmares serre)"}[net])
		}
	}
}

func lot1Pct(num, den int) float64 {
	if den <= 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}

// lot1Top rend les `k` entrees les plus frequentes d'un histogramme, formatees.
func lot1Top(m map[string]int, k int) string {
	type kv struct {
		k string
		v int
	}
	var s []kv
	for key, v := range m {
		s = append(s, kv{key, v})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].v > s[j].v })
	if len(s) > k {
		s = s[:k]
	}
	parts := make([]string, 0, len(s))
	for _, e := range s {
		parts = append(parts, fmt.Sprintf("%s x%d", e.k, e.v))
	}
	return strings.Join(parts, " · ")
}

func lot1Verdict(ok bool) string {
	if ok {
		return "TENU"
	}
	return "RATE"
}
