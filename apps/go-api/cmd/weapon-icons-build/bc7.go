package main

// bc7.go — DECODEUR BC7 (BPTC).
//
// PERIMETRE, ET POURQUOI IL EST CE QU IL EST. Les modes 4, 5 et 6 (un seul sous-ensemble) se
// decodent EXACTEMENT : ils ne demandent aucune table de partition. Ils representent 99 % des
// blocs mesures sur ces icones.
//
// Le mode 7 est RECONSTRUIT (bc7_mode7.go) : c est le seul des modes multi-sous-ensembles qui
// degradait le livrable, puisque les modes 0 a 3 n ont pas de canal alpha — leur alpha vaut
// 255 partout et le repli le rend exactement, or seul l alpha est conserve dans le glyphe.
//
// Les tables de partition ne sont volontairement PAS recopiees ici : une table fausse rendrait
// une image plausible mais fausse, et c est le pire cas. Ce qui manque est retrouve par
// ajustement sur le niveau de mip inferieur.
//
// Reference : specification D3D11 BC7 (BPTC). Les poids d interpolation et les largeurs de
// champs sont ceux de la norme ; rien n est devine.

import "image"

// poids d interpolation par nombre de bits d index.
var bc7W2 = [4]int{0, 21, 43, 64}
var bc7W3 = [8]int{0, 9, 18, 27, 37, 46, 55, 64}
var bc7W4 = [16]int{0, 4, 9, 13, 17, 21, 26, 30, 34, 38, 43, 47, 51, 55, 60, 64}

// bc7Mode : le mode d un bloc est code en UNAIRE dans les bits de poids faible du premier
// octet — m zeros suivis d un 1. Retourne -1 pour un bloc invalide (8 bits bas a zero).
func bc7Mode(b0 byte) int {
	for m := 0; m < 8; m++ {
		if b0&(1<<uint(m)) != 0 {
			return m
		}
	}
	return -1
}

// bitReader lit un bloc de 128 bits, bit de poids faible du premier octet en tete.
type bitReader struct {
	b   []byte
	pos int
}

func (r *bitReader) read(n int) int {
	v := 0
	for i := 0; i < n; i++ {
		byteIdx := r.pos >> 3
		if byteIdx >= len(r.b) {
			r.pos++
			continue
		}
		bit := (r.b[byteIdx] >> uint(r.pos&7)) & 1
		v |= int(bit) << uint(i)
		r.pos++
	}
	return v
}

// interp : ((64-w)*e0 + w*e1 + 32) >> 6, la formule de la norme.
func interp(e0, e1, w int) uint8 {
	return uint8((((64-w)*e0 + w*e1 + 32) >> 6) & 0xff)
}

// expand porte une valeur de `bits` bits sur 8 bits en repliquant les bits de tete.
func expand(v, bits int) int {
	if bits >= 8 {
		return v & 0xff
	}
	return ((v << uint(8-bits)) | (v >> uint(2*bits-8))) & 0xff
}

// DecodeBC7Block ecrit un bloc 4x4 dans img a la position (bx*4, by*4).
// Retourne true si le bloc a ete decode exactement, false s il a subi le repli.
func decodeBC7Block(block []byte, img *image.NRGBA, x0, y0 int, guide *image.NRGBA) bool {
	mode := bc7Mode(block[0])
	r := &bitReader{b: block}
	switch mode {
	case 4:
		r.read(5)
		rot := r.read(2)
		idxMode := r.read(1)
		var c [2][3]int
		for ch := 0; ch < 3; ch++ {
			c[0][ch] = expand(r.read(5), 5)
			c[1][ch] = expand(r.read(5), 5)
		}
		a0 := expand(r.read(6), 6)
		a1 := expand(r.read(6), 6)
		i2 := readIdx(r, 2)
		i3 := readIdx(r, 3)
		cw, aw := i2, i3
		cbits, abits := 2, 3
		if idxMode == 1 {
			cw, aw = i3, i2
			cbits, abits = 3, 2
		}
		writeBlock(img, x0, y0, c, a0, a1, cw, aw, cbits, abits, rot)
		return true
	case 5:
		r.read(6)
		rot := r.read(2)
		var c [2][3]int
		for ch := 0; ch < 3; ch++ {
			c[0][ch] = expand(r.read(7), 7)
			c[1][ch] = expand(r.read(7), 7)
		}
		a0 := r.read(8)
		a1 := r.read(8)
		cw := readIdx(r, 2)
		aw := readIdx(r, 2)
		writeBlock(img, x0, y0, c, a0, a1, cw, aw, 2, 2, rot)
		return true
	case 6:
		r.read(7)
		var c [2][3]int
		raw := [2][3]int{}
		for ch := 0; ch < 3; ch++ {
			raw[0][ch] = r.read(7)
			raw[1][ch] = r.read(7)
		}
		ra0 := r.read(7)
		ra1 := r.read(7)
		p0 := r.read(1)
		p1 := r.read(1)
		for ch := 0; ch < 3; ch++ {
			c[0][ch] = (raw[0][ch] << 1) | p0
			c[1][ch] = (raw[1][ch] << 1) | p1
		}
		a0 := (ra0 << 1) | p0
		a1 := (ra1 << 1) | p1
		w := readIdx(r, 4)
		writeBlock(img, x0, y0, c, a0, a1, w, w, 4, 4, 0)
		return true
	case 7:
		if decodeMode7(block, img, x0, y0, guide) {
			return true
		}
		fallbackBlock(block, img, x0, y0, mode)
		return false
	default:
		fallbackBlock(block, img, x0, y0, mode)
		return false
	}
}

// readIdx lit 16 index de `bits` bits ; le PREMIER (ancre) en porte un de moins, son bit de
// tete etant implicitement nul.
func readIdx(r *bitReader, bits int) [16]int {
	var out [16]int
	out[0] = r.read(bits - 1)
	for i := 1; i < 16; i++ {
		out[i] = r.read(bits)
	}
	return out
}

func weightOf(bits, i int) int {
	switch bits {
	case 2:
		return bc7W2[i&3]
	case 3:
		return bc7W3[i&7]
	default:
		return bc7W4[i&15]
	}
}

func writeBlock(img *image.NRGBA, x0, y0 int, c [2][3]int, a0, a1 int,
	cw, aw [16]int, cbits, abits, rot int) {
	b := img.Bounds()
	for i := 0; i < 16; i++ {
		x, y := x0+i%4, y0+i/4
		if x >= b.Dx() || y >= b.Dy() {
			continue
		}
		wc := weightOf(cbits, cw[i])
		wa := weightOf(abits, aw[i])
		px := [4]uint8{
			interp(c[0][0], c[1][0], wc),
			interp(c[0][1], c[1][1], wc),
			interp(c[0][2], c[1][2], wc),
			interp(a0, a1, wa),
		}
		// rotation : le canal alpha est echange avec R, G ou B.
		switch rot {
		case 1:
			px[0], px[3] = px[3], px[0]
		case 2:
			px[1], px[3] = px[3], px[1]
		case 3:
			px[2], px[3] = px[3], px[2]
		}
		o := img.PixOffset(x, y)
		copy(img.Pix[o:o+4], px[:])
	}
}

// fallbackBlock — modes a plusieurs sous-ensembles (0-3, 7). Les tables de partition ne sont
// PAS reproduites ici ; le bloc est rempli d une couleur unique lue dans ses premiers champs
// de points extremes. Approximation ASSUMEE et comptee, jamais silencieuse.
func fallbackBlock(block []byte, img *image.NRGBA, x0, y0, mode int) {
	r := &bitReader{b: block}
	if mode < 0 {
		mode = 0
	}
	r.read(mode + 1)
	var cb, ab, pb int
	switch mode {
	case 0:
		cb, ab, pb = 4, 0, 4
	case 1:
		cb, ab, pb = 6, 0, 6
	case 2:
		cb, ab, pb = 5, 0, 6
	case 3:
		cb, ab, pb = 7, 0, 6
	case 7:
		cb, ab, pb = 5, 5, 6
	default:
		cb, ab, pb = 7, 7, 0
	}
	r.read(pb) // partition — non exploitee (voir en-tete)
	nEnd := 4
	if mode == 0 || mode == 2 {
		nEnd = 6
	}
	sum := [3]int{}
	for ch := 0; ch < 3; ch++ {
		for e := 0; e < nEnd; e++ {
			sum[ch] += expand(r.read(cb), cb)
		}
	}
	alpha := 255
	if ab > 0 {
		as := 0
		for e := 0; e < nEnd; e++ {
			as += expand(r.read(ab), ab)
		}
		alpha = as / nEnd
	}
	b := img.Bounds()
	for i := 0; i < 16; i++ {
		x, y := x0+i%4, y0+i/4
		if x >= b.Dx() || y >= b.Dy() {
			continue
		}
		o := img.PixOffset(x, y)
		img.Pix[o] = uint8(sum[0] / nEnd)
		img.Pix[o+1] = uint8(sum[1] / nEnd)
		img.Pix[o+2] = uint8(sum[2] / nEnd)
		img.Pix[o+3] = uint8(alpha)
	}
}

// decodeBC7 rend l image w*h portee par `data` (blocs 4x4, 16 octets, ordre ligne par ligne).
//
// `guide` est le niveau de mip inferieur agrandi, ou nil. Il ne sert QU AUX BLOCS DE MODE 7,
// dont la partition et la position d ancre ne sont pas reproduites ici : il permet de les
// reconstruire par ajustement au lieu de les rendre en aplat (cf. bc7_mode7.go).
//
// Les trois compteurs rendus distinguent ce qui DEGRADE le livrable de ce qui ne le degrade
// pas — la distinction compte, parce que seul l alpha est conserve dans le glyphe final :
//
//	rebuilt  blocs de mode 7 rebatis sur le temoin : alpha RECONSTRUIT ;
//	opaque   blocs de modes 0 a 3 : ces modes n ont PAS de canal alpha, leur alpha vaut donc
//	         255 partout et le repli le rend EXACTEMENT — seul le RGB est approche, et le RGB
//	         est jete. Ce compteur ne signale aucune degradation du livrable ;
//	degraded blocs de mode 7 SANS temoin : alpha rendu en aplat. La seule vraie degradation.
func decodeBC7(data []byte, w, h int, guide *image.NRGBA) (img *image.NRGBA, rebuilt, opaque, degraded int) {
	bw, bh := (w+3)/4, (h+3)/4
	img = image.NewNRGBA(image.Rect(0, 0, w, h))
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			o := (by*bw + bx) * 16
			if o+16 > len(data) {
				continue
			}
			block := data[o : o+16]
			mode := bc7Mode(block[0])
			if decodeBC7Block(block, img, bx*4, by*4, guide) {
				if mode == 7 {
					rebuilt++
				}
				continue
			}
			if mode == 7 {
				degraded++
				continue
			}
			opaque++
		}
	}
	return img, rebuilt, opaque, degraded
}
