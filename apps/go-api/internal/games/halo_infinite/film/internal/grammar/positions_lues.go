package grammar

// positions_lues.go — LES POSITIONS BIPEDES : LA MARCHE DES TRAMES D ABORD, L ANCRAGE DERRIERE ELLE
// (plan de l etape 2 de la representation intermediaire, 2.7.d2 ; ADR 0037 IR-6).
//
// # LA MARCHE DESIGNE LES RECORDS, LE LECTEUR DE POSITION LES LIT
//
// Le canal des lectures bipedes ([canalDesLecturesBipedes]) retient de chaque record bipede delta
// que la marche a lu — corps vivant, generation acceptee — le bit de son composant i0 quand cet i0
// est ABSOLU dans la region jouee ([i0AbsoluDeLaRegion], la grammaire d i0 des positions) ; puis
// chaque record que l ancrage d en-tete rend derriere la marche, selon la regle des huit lecteurs
// ([rendParLAncrage]). La position se lit ensuite a ce bit, par le lecteur de toujours
// ([lireLaPosition]) : aux records que les deux sources designent, la valeur est la meme par
// construction.
//
// # CE QUE LA REGLE CORRIGE
//
// Un slot de la bande bipede que la marche lit comme un autre objet, dans une trame que sa fermeture
// prouve, ne se lit plus comme un joueur ; un en-tete fortuit dans l etendue que la fermeture de la
// trame prouve n est plus une position ; un cadavre ne se deplace plus (mesure 2 de 2.7.d0,
// decouverte 38 du plan). Dans une trame qu elle ne prouve pas, l archetype que la marche donne au
// slot ne l est pas non plus : seul un record bipede de la marche y ecarte l en-tete ancre.
//
// # SANS REGISTRE, L ANCRAGE SEUL
//
// La marche des trames exige le registre du film. Un film sans `chunk_00` (bobines de test) ne la
// joue pas : ses positions viennent de l ancrage seul ([positionsDesAncres]), comme la phase des
// images-cles sans registre rend ses ancres (decision 4 du lot 2.2). La cuisson lit toujours un film
// complet.

import (
	"cmp"
	"math/bits"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// indexDeLaPosition est l index du composant i0 de l archetype bipede : la position
// (`object-position-dynamic-precision-component`), le premier composant de tout record.
const indexDeLaPosition = 0

// positionsBipedes : les records dont la position se lit, groupes par paquet, dans l ordre du flux.
type positionsBipedes struct {
	paquets []paquetAncre
	records []positionLue
}

// positionLue est un record dont la position se lit : le bit de son i0, son slot, la generation de
// son handle, son masque de composants et sa source.
type positionLue struct {
	i0, slot uint32
	gen      uint8
	recupere bool
	masque   uint64
}

// noter ajoute un record du paquet `pk` du chunk `chunk` ; les paquets arrivent dans l ordre du flux,
// les records d un paquet dans l ordre de leur bit.
func (pb *positionsBipedes) noter(chunk int, pk FilmPacket, r positionLue) {
	if n := len(pb.paquets); n == 0 || pb.paquets[n-1].chunk != chunk || pb.paquets[n-1].paquet.Index != pk.Index {
		pb.paquets = append(pb.paquets, paquetAncre{chunk: chunk, paquet: pk, premier: len(pb.records)})
	}
	pb.records = append(pb.records, r)
}

// recordsDu rend les records du k-ieme paquet.
func (pb *positionsBipedes) recordsDu(k int) []positionLue {
	fin := len(pb.records)
	if k+1 < len(pb.paquets) {
		fin = pb.paquets[k+1].premier
	}
	return pb.records[pb.paquets[k].premier:fin]
}

// fondrePositions rend les records de `a` (la marche) et de `b` (l ancrage derriere elle), deux
// suites dans l ordre du flux, fondus dans l ordre du flux : rang du chunk dans le film, rang du
// paquet, bit d i0.
func fondrePositions(a, b *positionsBipedes, chunks []int) positionsBipedes {
	rang := make(map[int]int, len(chunks))
	for i, c := range chunks {
		rang[c] = i
	}
	var out positionsBipedes
	i, j := 0, 0
	for i < len(a.paquets) || j < len(b.paquets) {
		ordre := 1
		switch {
		case i == len(a.paquets):
		case j == len(b.paquets):
			ordre = -1
		default:
			pa, pb := a.paquets[i], b.paquets[j]
			ordre = cmp.Or(cmp.Compare(rang[pa.chunk], rang[pb.chunk]), cmp.Compare(pa.paquet.Index, pb.paquet.Index))
		}
		switch {
		case ordre < 0:
			out.ajouterLePaquet(a.paquets[i], a.recordsDu(i), nil)
			i++
		case ordre > 0:
			out.ajouterLePaquet(b.paquets[j], nil, b.recordsDu(j))
			j++
		default:
			out.ajouterLePaquet(a.paquets[i], a.recordsDu(i), b.recordsDu(j))
			i, j = i+1, j+1
		}
	}
	return out
}

// ajouterLePaquet ajoute un paquet et ses records des deux sources, fondus par bit d i0.
func (pb *positionsBipedes) ajouterLePaquet(p paquetAncre, a, b []positionLue) {
	pb.paquets = append(pb.paquets, paquetAncre{chunk: p.chunk, paquet: p.paquet, premier: len(pb.records)})
	for len(a) > 0 || len(b) > 0 {
		if len(b) == 0 || (len(a) > 0 && a[0].i0 <= b[0].i0) {
			pb.records, a = append(pb.records, a[0]), a[1:]
			continue
		}
		pb.records, b = append(pb.records, b[0]), b[1:]
	}
}

// i0AbsoluDeLaRegion dit si le composant i0 qui commence au bit `i0` est ABSOLU (spine et useDefault
// nuls) dans la REGION jouee (`lay.Region`, zero partout sauf sur les cartes dont la region jouee
// n est pas la premiere du bloc structure-BSP), avec ses trois axes dans le payload. Un record d une
// autre region a ses quanta dans une autre AABB : il ne donne pas de position.
func i0AbsoluDeLaRegion(pay []byte, i0 int, lay profile.I0Layout) bool {
	const preGate = profile.I0SpineBits + profile.I0UseDefaultBits
	if i0 < 0 || i0+lay.TotalBits() > len(pay)*8 {
		return false
	}
	if uint32(source.BitsStricts(pay, i0, preGate)) != 0 {
		return false
	}
	return uint32(source.BitsStricts(pay, i0+preGate, lay.GateBits-preGate)) == lay.Region
}

// indexDuMasque rend les index du masque `m`, dans l ordre croissant.
func indexDuMasque(m uint64) []int {
	out := make([]int, 0, bits.OnesCount64(m))
	for ; m != 0; m &= m - 1 {
		out = append(out, bits.TrailingZeros64(m))
	}
	return out
}

// positionsDuContexte lit les positions bipedes du film sous les parametres du contexte : celles des
// records que la marche des trames a lus, puis de ceux que l ancrage rend derriere elle
// ([FilmContext.lecturesBipedes]) ; sans registre, celles de l ancrage seul. Rend aussi le nombre de
// chunks LUS parmi `chunks`, comme [scanBipedChunks].
func positionsDuContexte(fc *FilmContext, chunks []int, lay profile.I0Layout, opt ScanFilmOptions) (
	[]BipedPosition, int) {
	lu, err := fc.lecturesBipedes()
	if err != nil {
		return positionsDesAncres(fc, chunks, lay, opt)
	}
	read := 0
	for _, c := range chunks {
		if _, _, ok := fc.ChunkAt(c); ok {
			read++
		}
	}
	var out []BipedPosition
	ctx, g := fc.ContexteDeLecture(), grammaireDOrientation(opt)
	chunk := -1
	var data []byte
	var pks []FilmPacket
	pb := &lu.positions
	for k, p := range pb.paquets {
		if p.chunk != chunk {
			chunk = p.chunk
			data, pks, _ = fc.ChunkAt(chunk)
		}
		pay := payloadDeRang(data, pks, p.paquet.Index)
		if pay == nil {
			continue
		}
		br := LecteurSur(pay)
		br.PoserContexte(ctx)
		for _, r := range pb.recordsDu(k) {
			dr := deltaBipedRecord{Payload: pay, Total: len(pay) * 8, I0: int(r.i0), Slot: r.slot,
				Mask: indexDuMasque(r.masque), Gen: uint32(r.gen), Chunk: p.chunk, Packet: p.paquet}
			if rec, ok := lireLaPosition(br, dr, lay, opt, g); ok {
				rec.Chunk, rec.PacketIndex, rec.TimestampUS = p.chunk, p.paquet.Index, p.paquet.TimestampUS
				out = append(out, rec)
			}
		}
	}
	return out, read
}

// payloadDeRang rend le payload du paquet de rang `index` parmi `pks`, les paquets du chunk `data` ;
// nil si le chunk ne le porte pas.
func payloadDeRang(data []byte, pks []FilmPacket, index int) []byte {
	if index >= 0 && index < len(pks) && pks[index].Index == index {
		return pks[index].Payload(data)
	}
	for _, pk := range pks {
		if pk.Index == index {
			return pk.Payload(data)
		}
	}
	return nil
}
