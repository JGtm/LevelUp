package filmdec

// biped_pickup_research_test.go — CHANTIER RAMASSAGE : decoder l'evenement natif
// `biped_pickup` (type 9) de la liste d'evenements d'un paquet delta.
//
// LE MODELE DE PAQUET (prouve par le chantier trame, 30-31/08) :
//
//	[1 bit configuration][liste : ( 1 [R(7) type][3 refs gardees][charge] )* 0][trame de records]
//
// L'octet de tete vaut 0xC0 | (type>>1) : le type 9 partage l'octet 0xC4 avec le type 8
// (biped_board_vehicle). bit8 = type & 1 les departage (8 -> 0, 9 -> 1).
//
// UNE REFERENCE GARDEE : R(1) porte ; si 1 : [R(1) sonde si domaine 1] R(w) index R(2)
// generation. Largeurs par domaine (table exe 0x1451f98d0) : dom 0/1/7/8 = 13, dom 2/3/5 = 8,
// dom 4/6 = 9 (dom 1 tombe a 9 si la sonde vaut 1).
//
// LE JUGE n'est pas geometrique : c'est l'ORACLE DE TRAME. Apres avoir consomme l'evenement
// EN ENTIER puis le bit de fin de liste, la trame de records qui suit doit se decoder — et,
// critere DUR retenu ici, CONSOMMER LE PAYLOAD JUSQU'AU BOUT (le bourrage vaut moins d'un
// octet). Ce critere est calibre AVANT usage sur les trames pures du film
// (TestBipedPickupCalibration) : il vaut 93 % au cadrage vrai contre 0,0-0,1 % a +/-1, 2 ou
// 3 bits. C'est lui qui tranche, pas la profondeur seule.
//
// GARDE : BIPED_PICKUP_FILM porte le repertoire d'UN film (celui qui contient chunk_00).
// Sans elle tous les tests de ce fichier se sautent — les films ne sont pas versionnes.
//
//	CGO_ENABLED=0 BIPED_PICKUP_FILM=<depot>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run BipedPickup -v -timeout 30m

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const (
	bpkFilmEnv = "BIPED_PICKUP_FILM"
	// bpkOctet est l'octet de tete des paquets dont la liste s'ouvre sur un type 8 ou 9.
	bpkOctet = 0xC4
	// bpkTypePickup / bpkTypeBoard : les deux types que 0xC4 porte.
	bpkTypePickup = 9
	bpkTypeBoard  = 8
	// bpkHeaderBits : config(1) + continuation(1) + type R(7) = le premier bit d'une reference.
	bpkHeaderBits = 9
	// bpkCalibPackets borne le nombre de paquets lus pour calibrer (assez pour un taux stable).
	bpkCalibPackets = 3000
)

// bpkDomWidths : largeur de l'index R(w) par domaine (table 0x1451f98d0).
var bpkDomWidths = map[int]int{0: 13, 1: 13, 2: 8, 3: 8, 4: 9, 5: 8, 6: 9, 7: 13, 8: 13}

// bpkRef consomme une reference gardee du domaine dom. Le domaine 1 porte une sonde R(1)
// qui ramene la largeur a 9. Rend (index, presente).
func bpkRef(br *BitReader, dom int) (uint64, bool) {
	if !br.ReadBit() {
		return 0, false
	}
	w := bpkDomWidths[dom]
	if dom == 1 && br.ReadBit() {
		w = 9
	}
	idx := br.ReadBits(uint(w))
	br.Skip(2) // generation
	return idx, true
}

func bpkPct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}

func bpkVerdict(ok bool) string {
	if ok {
		return "TENU"
	}
	return "NON TENU"
}

// bpkTop rend les k entrees les plus frequentes d'un histogramme a cle entiere.
func bpkTop(m map[uint64]int, k int) string {
	type kv struct {
		k uint64
		v int
	}
	s := make([]kv, 0, len(m))
	for key, v := range m {
		s = append(s, kv{key, v})
	}
	sort.Slice(s, func(i, j int) bool {
		if s[i].v != s[j].v {
			return s[i].v > s[j].v
		}
		return s[i].k < s[j].k
	})
	if len(s) > k {
		s = s[:k]
	}
	out := ""
	for i, e := range s {
		if i > 0 {
			out += " · "
		}
		out += fmt.Sprintf("%d x%d", e.k, e.v)
	}
	return out
}

// bpkFilm porte ce qu'un film rend une fois ouvert.
type bpkFilm struct {
	dir    string
	reg    *Registry
	chunks int
	// idLow est la largeur d'id bas CALIBREE sur ce film (valeur de runtime, cf.
	// FrameConfig.IDLowBits : le defaut 13 n'est pas une constante du format).
	idLow int
}

func bpkOpen(t *testing.T) (bpkFilm, bool) {
	t.Helper()
	dir := os.Getenv(bpkFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de recherche saute", bpkFilmEnv)
		return bpkFilm{}, false
	}
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	// TOUT le film : la confrontation produit se fait contre des canaux (i43..i46, images-cles)
	// qui balayent le film entier ; se limiter a une fenetre temoin fausserait les taux.
	n := CountFilmChunks(dir)
	f := bpkFilm{dir: dir, reg: reg, chunks: n, idLow: DefaultFrameConfig().IDLowBits}
	f.idLow, _ = bpkCalibre(t, f, nil)
	return f, true
}

// bpkCfg rend la configuration de trame pour une largeur d'id bas donnee.
func bpkCfg(idLow int) FrameConfig {
	c := DefaultFrameConfig()
	c.IDLowBits = idLow
	return c
}

// bpkChunkWorld rejoue les images-cle puis les paquets sans evenement d'un chunk pour rendre
// un etat monde credible, et le fige. La trame post-evenement se decode contre cet etat.
func bpkChunkWorld(reg *Registry, data []byte, pks []FilmPacket, cfg FrameConfig) WorldSnapshot {
	w := NewWorld(reg)
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
		if pay := pk.Payload(data); pay[0]&0x40 == 0 {
			br := NewBitReader(pay)
			_, _ = DecodeFrameRecords(br, w, cfg)
		}
	}
	return w.Snapshot()
}

// bpkTrameExacte est LE JUGE : depuis le bit `bit`, la trame se decode-t-elle proprement ET
// consomme-t-elle le payload jusqu'a moins d'un octet de la fin ? Rend aussi sa profondeur.
func bpkTrameExacte(reg *Registry, snap WorldSnapshot, pay []byte, bit int, cfg FrameConfig) (bool, int) {
	w := NewWorld(reg)
	w.Restore(snap)
	br := NewBitReader(pay)
	br.Skip(bit)
	recs, err := DecodeFrameRecords(br, w, cfg)
	return err == nil && len(pay)*8-br.BitPos() < 8, len(recs)
}

// bpkCalibre balaye IDLowBits sur les paquets a liste VIDE (trame pure, cadrage connu au
// bit 2 — verite terrain du depot) et rend la largeur qui maximise le taux de trames EXACTES.
// Si log est non nul, le tableau complet y est journalise.
func bpkCalibre(t *testing.T, f bpkFilm, log func(string, ...any)) (int, float64) {
	t.Helper()
	const wMin, wMax = 9, 16
	exact := make([]int, wMax+1)
	prof := make([]int, wMax+1)
	vus := 0
	for c := 1; c <= f.chunks && vus < bpkCalibPackets; c++ {
		data, err := ReadFilmChunk(f.dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		for w := wMin; w <= wMax; w++ {
			cfg := bpkCfg(w)
			snap := bpkChunkWorld(f.reg, data, pks, cfg)
			n := 0
			for _, pk := range pks {
				if pk.Type != PacketTypeDelta || pk.Size < 4 || vus+n >= bpkCalibPackets {
					continue
				}
				pay := pk.Payload(data)
				if pay[0]&0x40 != 0 {
					continue
				}
				n++
				ok, d := bpkTrameExacte(f.reg, snap, pay, 2, cfg)
				prof[w] += d
				if ok {
					exact[w]++
				}
			}
			if w == wMax {
				vus += n
			}
		}
	}
	best, bestPct := wMin, -1.0
	for w := wMin; w <= wMax; w++ {
		p := bpkPct(exact[w], vus)
		if log != nil {
			log("  IDLowBits=%2d : trames EXACTES %.1f %% · profondeur %.2f record/paquet",
				w, p, float64(prof[w])/float64(max(vus, 1)))
		}
		if p > bestPct {
			best, bestPct = w, p
		}
	}
	return best, bestPct
}

// bpkEachEvent parcourt les paquets delta dont l'octet de tete est 0xC4 et appelle fn avec
// le type lu, le payload, l'etat monde de reference et l'horodatage du paquet.
func bpkEachEvent(t *testing.T, f bpkFilm, fn func(typ int, pay []byte, snap WorldSnapshot, tsUS uint64)) {
	t.Helper()
	cfg := bpkCfg(f.idLow)
	for c := 1; c <= f.chunks; c++ {
		data, err := ReadFilmChunk(f.dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		snap := bpkChunkWorld(f.reg, data, pks, cfg)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != bpkOctet {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(1)
			if !br.ReadBit() {
				continue // liste vide : impossible pour 0xC4, mais on ne le suppose pas
			}
			fn(int(br.ReadBits(7)), pay, snap, pk.TimestampUS)
		}
	}
}

// TestBipedPickupCalibration — CALIBRAGE OBLIGATOIRE AVANT TOUT VERDICT DE CADRAGE.
//
//	(1) la largeur d'id bas du film (IDLowBits), balayee contre la verite terrain des trames
//	    PURES (paquets a liste vide, cadrage connu au bit 2) ;
//	(2) le pouvoir SEPARATEUR de l'oracle une fois calibre : trames pures au bon cadrage vs
//	    les MEMES a +1, +2, +3, +5 bits.
//
// SEUILS ECRITS AVANT LA MESURE : le critere est « trame FERMEE proprement ET payload
// consomme jusqu'a moins de 8 bits de la fin ». L'oracle est declare UTILISABLE si ce taux
// vaut >= 50 % au bon cadrage et <= 10 % aux temoins +1/+2/+3. Sinon aucune conclusion de
// cadrage ne sera tiree de la trame sur ce film.
func TestBipedPickupCalibration(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()
	t.Logf("== CALIBRAGE IDLowBits sur les trames PURES de %s ==", f.dir)
	best, pct := bpkCalibre(t, f, t.Logf)
	t.Logf("RETENU : IDLowBits=%d (%.1f %% de trames exactes)", best, pct)

	cfg := bpkCfg(best)
	decalages := []int{0, 1, 2, 3, 5}
	exact := make([]int, len(decalages))
	prof := make([]int, len(decalages))
	vus := 0
	for c := 1; c <= f.chunks && vus < bpkCalibPackets; c++ {
		data, err := ReadFilmChunk(f.dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		snap := bpkChunkWorld(f.reg, data, pks, cfg)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 4 || vus >= bpkCalibPackets {
				continue
			}
			pay := pk.Payload(data)
			if pay[0]&0x40 != 0 {
				continue
			}
			vus++
			for i, d := range decalages {
				ok, n := bpkTrameExacte(f.reg, snap, pay, 2+d, cfg)
				prof[i] += n
				if ok {
					exact[i]++
				}
			}
		}
	}
	pire := 0.0
	for i, d := range decalages {
		p := bpkPct(exact[i], vus)
		t.Logf("  decalage +%d bit(s) : trames EXACTES %.1f %% (n=%d) · profondeur %.2f",
			d, p, vus, float64(prof[i])/float64(max(vus, 1)))
		if d >= 1 && d <= 3 && p > pire {
			pire = p
		}
	}
	bon := bpkPct(exact[0], vus)
	t.Logf("VERDICT ORACLE (bon >= 50 %% et temoins +1/+2/+3 <= 10 %%) : %s — bon %.1f %%, pire temoin %.1f %%",
		bpkVerdict(bon >= 50 && pire <= 10), bon, pire)
}

// TestBipedPickupRecensement — ETAPE 1. Volume du type 9 sur le film, repartition 8/9, et
// lecture BRUTE des bits qui suivent le type (avant toute hypothese de domaine).
func TestBipedPickupRecensement(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()

	var (
		total, nPickup, nBoard, nAutre int
		tailles                        = map[uint64]int{}
		ref0Porte                      int
		echantillon                    []string
	)
	bpkEachEvent(t, f, func(typ int, pay []byte, _ WorldSnapshot, tsUS uint64) {
		total++
		switch typ {
		case bpkTypePickup:
			nPickup++
		case bpkTypeBoard:
			nBoard++
			return
		default:
			nAutre++
			return
		}
		tailles[uint64(len(pay))]++
		br := NewBitReader(pay)
		br.Skip(bpkHeaderBits)
		if br.ReadBit() {
			ref0Porte++
		}
		if len(echantillon) < 12 {
			br3 := NewBitReader(pay)
			br3.Skip(bpkHeaderBits)
			bits := ""
			for i := 0; i < 64 && bpkHeaderBits+i < len(pay)*8; i++ {
				if br3.ReadBit() {
					bits += "1"
				} else {
					bits += "0"
				}
				if i%8 == 7 {
					bits += " "
				}
			}
			echantillon = append(echantillon, fmt.Sprintf("ts=%d len=%d %s", tsUS, len(pay), bits))
		}
	})
	t.Logf("== film %s · %d chunk(s) · IDLowBits calibre = %d ==", f.dir, f.chunks, f.idLow)
	t.Logf("paquets 0xC4 : %d — type 9 (biped_pickup) x%d (%.1f %%) · type 8 (board_vehicle) x%d · autres x%d",
		total, nPickup, bpkPct(nPickup, total), nBoard, nAutre)
	if nPickup == 0 {
		t.Log("VERDICT : aucun type 9 sur ce film — rien a decoder ici.")
		return
	}
	t.Logf("taille des paquets porteurs (octets) : %s", bpkTop(tailles, 10))
	t.Logf("ref0 : bit de porte a 1 sur %d / %d (%.1f %%)", ref0Porte, nPickup, bpkPct(ref0Porte, nPickup))
	for _, e := range echantillon {
		t.Logf("  brut : %s", e)
	}
}

// TestBipedPickupCadrageScan — ETAPE 2, chaine EMPIRIQUE, sans aucune hypothese de grammaire.
// Pour chaque paquet type 9 on essaie tous les cadrages de fin d'evenement possibles et on
// juge chacun par l'oracle CALIBRE (trame exacte). Le taux de faux positifs est mesure sur le
// flux reel par la calibration (0,0-0,1 % a +/-1..3 bits) : un offset « exact » est donc
// quasi unique par paquet, et l'histogramme des offsets gagnants EST la distribution des
// longueurs de l'evenement.
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	S1 — au moins 80 % des paquets type 9 doivent avoir AU MOINS un offset exact (sinon
//	     l'evenement ne se termine pas dans la fenetre exploree, ou l'oracle est aveugle).
//	S2 — la mediane du nombre d'offsets exacts PAR PAQUET doit valoir 1 ou 2 (sinon l'oracle
//	     ne designe pas un cadrage unique et le scan ne tranche rien).
//	S3 — le mode de l'histogramme doit couvrir >= 20 % des paquets resolus : en dessous, la
//	     longueur de l'evenement est trop variable pour qu'un cadrage fixe existe.
func TestBipedPickupCadrageScan(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()

	const (
		offMin = 4
		offMax = 260
	)
	cfg := bpkCfg(f.idLow)
	hist := map[uint64]int{}
	var nPickup, resolus int
	cardinalites := []int{}
	premiers := map[uint64]int{}
	bpkEachEvent(t, f, func(typ int, pay []byte, snap WorldSnapshot, _ uint64) {
		if typ != bpkTypePickup {
			return
		}
		nPickup++
		total := len(pay) * 8
		gagnants := []int{}
		for off := offMin; off <= offMax; off++ {
			p := bpkHeaderBits + off
			if p+8 > total {
				break
			}
			if ok, _ := bpkTrameExacte(f.reg, snap, pay, p, cfg); ok {
				gagnants = append(gagnants, off)
			}
		}
		cardinalites = append(cardinalites, len(gagnants))
		if len(gagnants) == 0 {
			return
		}
		resolus++
		premiers[uint64(gagnants[0])]++
		for _, g := range gagnants {
			hist[uint64(g)]++
		}
	})
	if nPickup == 0 {
		t.Skip("aucun type 9 : scan sans objet")
	}
	sort.Ints(cardinalites)
	med := cardinalites[len(cardinalites)/2]
	t.Logf("== SCAN DE CADRAGE, %d evenements type 9, offsets %d..%d apres le champ de type ==",
		nPickup, offMin, offMax)
	t.Logf("paquets avec au moins un cadrage exact : %d / %d (%.1f %%) · offsets exacts par paquet : mediane %d, max %d",
		resolus, nPickup, bpkPct(resolus, nPickup), med, cardinalites[len(cardinalites)-1])
	t.Logf("histogramme des offsets exacts (tous)     : %s", bpkTop(hist, 14))
	t.Logf("histogramme du PREMIER offset exact       : %s", bpkTop(premiers, 14))
	mode, modeN := uint64(0), 0
	for k, v := range premiers {
		if v > modeN || (v == modeN && k < mode) {
			mode, modeN = k, v
		}
	}
	t.Logf("MODE du premier offset exact : %d bits apres le type, sur %d / %d paquets resolus (%.1f %%)",
		mode, modeN, resolus, bpkPct(modeN, resolus))
	t.Logf("VERDICT S1 (>= 80 %% resolus) : %s · S2 (mediane 1 ou 2) : %s · S3 (mode >= 20 %%) : %s",
		bpkVerdict(bpkPct(resolus, nPickup) >= 80), bpkVerdict(med == 1 || med == 2),
		bpkVerdict(bpkPct(modeN, resolus) >= 20))
}
