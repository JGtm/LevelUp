package replayview

import "strconv"

// appendColumn écrit la colonne `c` d'un tableau de n lignes : un tableau nu de valeurs quand tout
// est présent et brut, sinon `{"v":[...],"s":...,"d":1,"a":...}`.
func (e *colEncoder) appendColumn(dst []byte, c *column, n int, blobs []byte) []byte {
	sparse := len(c.cells) < n
	if scale, ok := e.scaleInts(c); ok {
		return e.appendScaled(dst, c, n, scale)
	}
	if !sparse {
		return e.appendRaw(dst, c, blobs)
	}
	dst = append(dst, `{"v":`...)
	dst = e.appendRaw(dst, c, blobs)
	dst = append(dst, `,"a":`...)
	dst = appendAbsence(dst, c.cells, n)
	return append(dst, '}')
}

// appendRaw écrit les valeurs présentes de la colonne telles quelles.
func (e *colEncoder) appendRaw(dst []byte, c *column, blobs []byte) []byte {
	dst = append(dst, '[')
	for i, cl := range c.cells {
		if i > 0 {
			dst = append(dst, ',')
		}
		if cl.kind == columnsKindBlob {
			dst = append(dst, blobs[cl.lo:cl.hi]...)
		} else {
			dst = append(dst, e.src[cl.lo:cl.hi]...)
		}
	}
	return append(dst, ']')
}

// scaleInts range dans e.ints les valeurs de la colonne mises à l'échelle de son plus grand nombre
// de décimales, et rend cette échelle. Faux (colonne brute) dès qu'une valeur n'est pas un nombre
// décimal sans exposant, que l'échelle dépasse columnsMaxDecimal ou qu'un entier dépasse la borne
// des entiers sûrs. Aucun calcul flottant : l'échelle est lue dans le jeton.
func (e *colEncoder) scaleInts(c *column) (int, bool) {
	if len(c.cells) == 0 {
		return 0, false
	}
	scale := 0
	for _, cl := range c.cells {
		dec, ok := e.decimals(cl)
		if !ok {
			return 0, false
		}
		scale = max(scale, dec)
	}
	if scale > columnsMaxDecimal {
		return 0, false
	}
	e.ints = e.ints[:0]
	for _, cl := range c.cells {
		v, ok := parseScaled(e.src[cl.lo:cl.hi], scale)
		if !ok {
			return 0, false
		}
		e.ints = append(e.ints, v)
	}
	return scale, true
}

// decimals rend le nombre de décimales du jeton, faux si ce n'est pas un nombre sans exposant.
func (e *colEncoder) decimals(cl cell) (int, bool) {
	if cl.kind != columnsKindScalar {
		return 0, false
	}
	lit := e.src[cl.lo:cl.hi]
	if c := lit[0]; c != '-' && (c < '0' || c > '9') {
		return 0, false
	}
	dot := -1
	for i, ch := range lit {
		switch ch {
		case '.':
			dot = i
		case 'e', 'E':
			return 0, false
		}
	}
	if dot < 0 {
		return 0, true
	}
	return len(lit) - dot - 1, true
}

// parseScaled lit le jeton comme entier à l'échelle `scale` ; faux au-delà des entiers sûrs.
func parseScaled(lit []byte, scale int) (int64, bool) {
	var mant uint64
	dec := -1
	for _, ch := range lit {
		switch {
		case ch == '-':
		case ch == '.':
			dec = 0
		case ch >= '0' && ch <= '9':
			mant = mant*10 + uint64(ch-'0')
			if mant > columnsSafeInt {
				return 0, false
			}
			if dec >= 0 {
				dec++
			}
		default:
			return 0, false
		}
	}
	dec = max(dec, 0)
	for i := dec; i < scale; i++ {
		mant *= 10
		if mant > columnsSafeInt {
			return 0, false
		}
	}
	if lit[0] == '-' {
		return -int64(mant), true
	}
	return int64(mant), true
}

// appendScaled écrit la colonne numérique dont les entiers sont dans e.ints : en différences
// successives si le texte en est strictement plus court.
func (e *colEncoder) appendScaled(dst []byte, c *column, n, scale int) []byte {
	vals := e.ints
	delta := false
	if len(vals) > 1 && e.fillDeltas(vals) && intsTextLen(e.deltas) < intsTextLen(vals) {
		vals, delta = e.deltas, true
	}
	sparse := len(c.cells) < n
	if !sparse && scale == 0 && !delta {
		return appendInts(dst, vals)
	}
	dst = append(dst, `{"v":`...)
	dst = appendInts(dst, vals)
	if scale > 0 {
		dst = append(dst, `,"s":`...)
		dst = strconv.AppendInt(dst, int64(scale), 10)
	}
	if delta {
		dst = append(dst, `,"d":1`...)
	}
	if sparse {
		dst = append(dst, `,"a":`...)
		dst = appendAbsence(dst, c.cells, n)
	}
	return append(dst, '}')
}

// fillDeltas remplit e.deltas des différences successives de vals ; faux si l'une dépasse la
// borne des entiers sûrs (le décodeur ne les relirait pas exactement).
func (e *colEncoder) fillDeltas(vals []int64) bool {
	e.deltas = e.deltas[:0]
	e.deltas = append(e.deltas, vals[0])
	for i := 1; i < len(vals); i++ {
		d := vals[i] - vals[i-1]
		if d > columnsSafeInt || d < -columnsSafeInt {
			return false
		}
		e.deltas = append(e.deltas, d)
	}
	return true
}

func appendInts(dst []byte, vals []int64) []byte {
	dst = append(dst, '[')
	for i, v := range vals {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = strconv.AppendInt(dst, v, 10)
	}
	return append(dst, ']')
}

// intsTextLen est la longueur du texte JSON de la liste.
func intsTextLen(vals []int64) int {
	n := 1
	for _, v := range vals {
		n += 1 + digitsLen(v)
	}
	return n
}

func digitsLen(v int64) int {
	n := 1
	if v < 0 {
		n++
		v = -v
	}
	for v >= 10 {
		v /= 10
		n++
	}
	return n
}

// appendAbsence écrit où la valeur manque : longueurs alternées (présent d'abord, la première
// pouvant valoir 0) ou chaîne de 0 et de 1, la plus courte des deux, la liste à égalité.
func appendAbsence(dst []byte, cells []cell, n int) []byte {
	runs := absenceRuns(cells, n)
	runsLen := 1
	for _, r := range runs {
		runsLen += 1 + digitsLen(r)
	}
	if runsLen <= n+2 {
		return appendInts(dst, runs)
	}
	dst = append(dst, '"')
	next := 0
	for _, cl := range cells {
		for ; next < int(cl.row); next++ {
			dst = append(dst, '0')
		}
		dst = append(dst, '1')
		next = int(cl.row) + 1
	}
	for ; next < n; next++ {
		dst = append(dst, '0')
	}
	return append(dst, '"')
}

// absenceRuns rend les longueurs alternées présent / absent / présent... des n lignes.
func absenceRuns(cells []cell, n int) []int64 {
	var runs []int64
	on, length := true, int64(0)
	step := func(present bool, count int64) {
		if count == 0 {
			return
		}
		if present != on {
			runs = append(runs, length)
			on, length = present, 0
		}
		length += count
	}
	next := 0
	for _, cl := range cells {
		step(false, int64(int(cl.row)-next))
		step(true, 1)
		next = int(cl.row) + 1
	}
	step(false, int64(n-next))
	return append(runs, length)
}
