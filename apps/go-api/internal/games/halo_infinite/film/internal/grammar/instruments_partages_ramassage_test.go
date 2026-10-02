package grammar

// instruments_partages_ramassage_test.go — l'instrument du ramassage natif (`bpk*`) et le
// catalogue des armes tenues (`hwCatalogue`), dont se sert la garde `biped_pickups_test.go`.
// Deplaces tels quels au J12.7 bis depuis les fichiers tagues `research` (decision DU-5 : le
// tag cache les instruments, jamais une garde) ; chaque declaration garde le corps et le
// commentaire de son fichier d'origine.

import (
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
)

// bpkCollecte rend tous les biped_pickup du film, decodes et tries par horodatage.
func bpkCollecte(t *testing.T, f bpkFilm) []bpkEvent {
	t.Helper()
	var evs []bpkEvent
	bpkEachEvent(t, f, func(typ int, pay []byte, _ WorldSnapshot, tsUS uint64) {
		if typ != bpkTypePickup {
			return
		}
		br := LecteurSur(pay)
		br.Skip(bpkHeaderBits)
		e := bpkDecode(br)
		e.TimestampUS = tsUS
		evs = append(evs, e)
	})
	sort.Slice(evs, func(i, j int) bool { return evs[i].TimestampUS < evs[j].TimestampUS })
	return evs
}

func bpkPct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
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

// hwCatalogue rend le predicat d'appartenance au catalogue de production.
func hwCatalogue() map[uint32]bool {
	connues := weaponv3.KnownWeaponHigh32Copie()
	m := make(map[uint32]bool, len(connues))
	for f := range connues {
		m[f] = true
	}
	return m
}

// bpkEvent est un evenement biped_pickup decode.
type bpkEvent struct {
	TimestampUS  uint64
	Ref0         uint64 // domaine 2, R(8) — le ramasseur presume
	Ref0Present  bool
	Ref1Present  bool
	Ref2Present  bool
	Kind         uint64 // R(3) de tete de charge
	Objet        uint32 // R(32) : le handle de l'objet ramasse (0xFFFFFFFF = absent)
	ObjetPresent bool
	// FinBit : position du bit qui suit le bit de fin de liste, donc le debut de la trame
	// quand l'evenement est seul dans sa liste.
	FinBit int
	// Suite : un autre evenement suit dans la meme liste.
	Suite bool
}

// bpkDecode consomme UN evenement type 9 a partir d'un lecteur place juste apres le champ de
// type, puis le bit de fin de liste. La grammaire est celle lue dans l'exe (ci-dessus).
func bpkDecode(br *Lecteur) bpkEvent {
	var e bpkEvent
	e.Ref0, e.Ref0Present = bpkRef(br, 2)
	_, e.Ref1Present = bpkRef(br, 8)
	_, e.Ref2Present = bpkRef(br, 7)
	e.Kind = br.ReadBits(3)
	if br.ReadBit() {
		e.Objet = uint32(br.ReadBits(32))
		e.ObjetPresent = true
	} else {
		e.Objet = 0xFFFFFFFF
	}
	e.Suite = br.ReadBit()
	e.FinBit = br.BitPos()
	return e
}

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

// bpkFilm porte ce qu'un film rend une fois ouvert.
type bpkFilm struct {
	dir    string
	reg    *Registry
	chunks int
	// idLow est la largeur d'id bas CALIBREE sur ce film (valeur de runtime, cf.
	// FrameConfig.IDLowBits : le defaut 13 n'est pas une constante du format).
	idLow int
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
			br := LecteurSur(pay)
			br.Skip(1)
			if !br.ReadBit() {
				continue // liste vide : impossible pour 0xC4, mais on ne le suppose pas
			}
			fn(int(br.ReadBits(7)), pay, snap, pk.TimestampUS)
		}
	}
}

// bpkRef consomme une reference gardee du domaine dom. Le domaine 1 porte une sonde R(1)
// qui ramene la largeur a 9. Rend (index, presente).
func bpkRef(br *Lecteur, dom int) (uint64, bool) {
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
			br := LecteurSur(pay)
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
	br := LecteurSur(pay)
	br.Skip(bit)
	recs, err := DecodeFrameRecords(br, w, cfg)
	return err == nil && len(pay)*8-br.BitPos() < 8, len(recs)
}

// bpkDomWidths : largeur de l'index R(w) par domaine (table 0x1451f98d0).
var bpkDomWidths = map[int]int{0: 13, 1: 13, 2: 8, 3: 8, 4: 9, 5: 8, 6: 9, 7: 13, 8: 13}
