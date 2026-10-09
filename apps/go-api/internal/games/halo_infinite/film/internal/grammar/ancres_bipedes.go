package grammar

// ancres_bipedes.go — L ANCRAGE BIPEDE DU FILM, FAIT UNE FOIS PAR CONTEXTE (ADR 0037 IR-6 ; lot 2.4
// du plan de l etape 2).
//
// # DEUX LECTEURS, UN ANCRAGE
//
// La recuperation qui passe DERRIERE la marche des trames, pour les huit lecteurs de composants et
// pour les positions ([lecturesBipedes] : changements d arme, deltas d inventaire, rangs, impulsions
// et charges de capacite, changements d equipement, camouflage, grappin ; [positionsBipedes]), et
// les positions d un film sans registre ([positionsDesAncres]), ancrent les records bipedes des
// trames delta avec les MEMES parametres : ceux du contexte — ses chunks, sa
// bande bipede, son decoupage d i0, ses generations vivantes datees a l instant du paquet. Le curseur
// bit a bit de l ancrage ([walkDeltaBipedRecords]) est l essentiel de leur cout : il est donc fait
// une fois, au premier lecteur, et range ici ; chaque lecteur parcourt ce qui est range, dans
// l ordre du flux.
//
// # CE QUI EST RANGE
//
// Par paquet porteur : son chunk et son en-tete ; par record : le bit de son i0, son slot, la
// generation de son handle et les index de son masque (au plus [bipedMaxMaskCnt], chacun sous 64).
// Le payload n est pas copie : il se relit dans le chunk, que le film garde decompresse.

// ancresBipedes est l ancrage bipede d un film : ses paquets porteurs, puis leurs records, dans
// l ordre du flux.
type ancresBipedes struct {
	paquets []paquetAncre
	records []recordAncre
}

// paquetAncre est un paquet delta qui porte au moins un record ancre : son chunk, son en-tete, et le
// rang de son premier record dans [ancresBipedes.records] — ses records vont jusqu au premier du
// paquet suivant.
type paquetAncre struct {
	chunk   int
	paquet  FilmPacket
	premier int
}

// recordAncre est un record bipede ancre, range compact.
type recordAncre struct {
	i0, slot uint32
	gen      uint8
	n        uint8
	masque   [bipedMaxMaskCnt]uint8
}

// ancresBipedes rend l ancrage bipede du film, fait au premier appel sous les parametres du contexte.
// Un film sans chunk, sans bande bipede ou au decoupage illisible n ancre rien : ses lecteurs le
// refusent avant de le parcourir, chacun avec son erreur.
func (c *FilmContext) ancresBipedes() *ancresBipedes {
	if c.recup.ancres != nil {
		return c.recup.ancres
	}
	c.recup.ancres = &ancresBipedes{}
	chunks, slots := c.ChunkNumbers(), c.BipedSlots()
	lay, err := c.I0Layout()
	if len(chunks) == 0 || slots.Count() == 0 || err != nil {
		return c.recup.ancres
	}
	a := c.recup.ancres
	walkDeltaBipedRecords(c, chunks, slots, lay, func(r deltaBipedRecord) {
		if n := len(a.paquets); n == 0 || a.paquets[n-1].chunk != r.Chunk ||
			a.paquets[n-1].paquet.Index != r.Packet.Index {
			a.paquets = append(a.paquets, paquetAncre{chunk: r.Chunk, paquet: r.Packet, premier: len(a.records)})
		}
		a.records = append(a.records, recordAncreDe(r))
	})
	return c.recup.ancres
}

// recordAncreDe range un record du marcheur.
func recordAncreDe(r deltaBipedRecord) recordAncre {
	ra := recordAncre{i0: uint32(r.I0), slot: r.Slot, gen: uint8(r.Gen), n: uint8(len(r.Mask))} //nolint:gosec // position dans un payload, generation sur 2 bits, au plus 7 index
	for i, idx := range r.Mask {
		ra.masque[i] = uint8(idx) //nolint:gosec // index de composant sous 64
	}
	return ra
}

// parcourirLesAncresBipedes rend a `visit` chaque record bipede ancre du film, dans l ordre du flux,
// sous la forme du marcheur ([deltaBipedRecord]) : ce que [walkDeltaBipedRecords] rendrait sous les
// parametres du contexte. Le masque de chaque record est une tranche neuve, a son lecteur.
func (c *FilmContext) parcourirLesAncresBipedes(visit func(deltaBipedRecord)) {
	a := c.ancresBipedes()
	chunk, data := -1, []byte(nil)
	for k, p := range a.paquets {
		if p.chunk != chunk {
			chunk = p.chunk
			data, _, _ = c.ChunkAt(chunk)
		}
		pay := p.paquet.Payload(data)
		fin := len(a.records)
		if k+1 < len(a.paquets) {
			fin = a.paquets[k+1].premier
		}
		for _, ra := range a.records[p.premier:fin] {
			visit(ra.enRecord(pay, p.chunk, p.paquet))
		}
	}
}

// enRecord rend le record range sous la forme du marcheur, dans le payload `pay` de son paquet.
func (ra recordAncre) enRecord(pay []byte, chunk int, pk FilmPacket) deltaBipedRecord {
	masque := make([]int, ra.n)
	for i := range masque {
		masque[i] = int(ra.masque[i])
	}
	return deltaBipedRecord{Payload: pay, Total: len(pay) * 8, I0: int(ra.i0), Slot: ra.slot, Mask: masque,
		Gen: uint32(ra.gen), Chunk: chunk, Packet: pk}
}
