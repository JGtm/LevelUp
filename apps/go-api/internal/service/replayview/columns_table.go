package replayview

import (
	"bytes"
	"strconv"
)

// cell est une valeur de colonne : un jeton scalaire du corps (lo, hi dans `src`) ou une valeur
// imbriquée déjà encodée (lo, hi dans les `blobs` du tableau).
type cell struct {
	lo, hi int32
	row    int32
	kind   uint8
}

// column regroupe les valeurs d'un même champ ; `name` est la clé telle qu'écrite (guillemets
// compris), vide pour une colonne de n-uplet.
type column struct {
	name  []byte
	cells []cell
}

// objTable accumule une liste d'objets en colonnes.
type objTable struct {
	n      int
	cols   []column
	byName map[string]int
	blobs  []byte
	sigs   map[string]struct{}
	prev   []byte
	cur    []byte
}

func newObjTable(n int) *objTable {
	return &objTable{n: n, byName: map[string]int{}, sigs: map[string]struct{}{}}
}

// column rend l'indice de la colonne du champ `key`, lue en j-ième position de la ligne. La
// colonne de la même position à la ligne précédente est essayée d'abord : les lignes d'une liste
// partagent presque toujours leur suite de champs.
func (t *objTable) column(key []byte, j int) (int, error) {
	if 2*j+1 < len(t.prev) {
		ci := int(t.prev[2*j]) | int(t.prev[2*j+1])<<8
		if bytes.Equal(t.cols[ci].name, key) {
			return ci, nil
		}
	}
	if ci, ok := t.byName[string(key)]; ok {
		return ci, nil
	}
	if reservedKey(key) {
		return 0, ErrColumnsKeyCollision
	}
	if len(t.cols) >= columnsMaxColumns {
		return 0, ErrColumnsUnsupported
	}
	t.cols = append(t.cols, column{name: key, cells: make([]cell, 0, t.n)})
	t.byName[string(key)] = len(t.cols) - 1
	return len(t.cols) - 1, nil
}

// noteRow retient la suite de champs de la ligne qui vient d'être lue.
func (t *objTable) noteRow() {
	if !bytes.Equal(t.cur, t.prev) {
		if _, ok := t.sigs[string(t.cur)]; !ok {
			t.sigs[string(t.cur)] = struct{}{}
		}
	}
	t.prev, t.cur = t.cur, t.prev[:0]
}

// objectTable encode la liste d'objets lue en pos en `{"$c":N,"k":[...],"c":[...]}`.
func (e *colEncoder) objectTable(dst []byte, pos int, sv listSurvey) ([]byte, error) {
	t := newObjTable(sv.n)
	p := e.ws(pos + 1)
	for row := 0; row < sv.n; row++ {
		end, err := e.readRow(t, row, e.ws(p))
		if err != nil {
			return nil, err
		}
		p = e.ws(end)
		if e.at(p) == ',' {
			p++
		}
	}
	order := t.keyOrder()
	dst = append(dst, `{"$c":`...)
	dst = appendJSONInt(dst, int64(sv.n))
	dst = append(dst, `,"k":[`...)
	for i, ci := range order {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = append(dst, t.cols[ci].name...)
	}
	dst = append(dst, `],"c":[`...)
	for i, ci := range order {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = e.appendColumn(dst, &t.cols[ci], sv.n, t.blobs)
	}
	return append(dst, ']', '}'), nil
}

// readRow lit l'objet de la ligne `row` lu en pos et range chacune de ses valeurs dans sa colonne.
func (e *colEncoder) readRow(t *objTable, row, pos int) (int, error) {
	p := e.ws(pos + 1)
	if e.at(pos) != '{' {
		return 0, ErrColumnsUnsupported
	}
	if e.at(p) == '}' {
		t.noteRow()
		return p + 1, nil
	}
	for j := 0; ; j++ {
		p = e.ws(p)
		if e.at(p) != '"' {
			return 0, ErrColumnsUnsupported
		}
		kend, err := e.str(p)
		if err != nil {
			return 0, err
		}
		ci, err := t.column(e.src[p:kend], j)
		if err != nil {
			return 0, err
		}
		c := &t.cols[ci]
		if n := len(c.cells); n > 0 && int(c.cells[n-1].row) == row {
			return 0, ErrColumnsUnsupported
		}
		p = e.ws(kend)
		if e.at(p) != ':' {
			return 0, ErrColumnsUnsupported
		}
		cl, end, err := e.readCell(t, e.ws(p+1))
		if err != nil {
			return 0, err
		}
		cl.row = int32(row)
		c.cells = append(c.cells, cl)
		t.cur = append(t.cur, byte(ci), byte(ci>>8))
		p = e.ws(end)
		switch e.at(p) {
		case ',':
			p++
		case '}':
			t.noteRow()
			return p + 1, nil
		default:
			return 0, ErrColumnsUnsupported
		}
	}
}

// readCell lit la valeur en pos : un scalaire reste désigné dans le corps, une valeur imbriquée est
// encodée tout de suite dans les `blobs` du tableau.
func (e *colEncoder) readCell(t *objTable, pos int) (cell, int, error) {
	if c := e.at(pos); c == '{' || c == '[' {
		lo := len(t.blobs)
		blobs, end, err := e.value(t.blobs, pos)
		if err != nil {
			return cell{}, 0, err
		}
		t.blobs = blobs
		return cell{lo: int32(lo), hi: int32(len(blobs)), kind: columnsKindBlob}, end, nil
	}
	end, err := e.scalar(pos)
	if err != nil {
		return cell{}, 0, err
	}
	return cell{lo: int32(pos), hi: int32(end), kind: columnsKindScalar}, end, nil
}

// keyOrder rend les indices de colonnes dans l'ordre des clés : la fusion topologique des suites
// de champs observées (les structures Go les écrivent toujours dans le même ordre, certains champs
// pouvant manquer), à égalité l'ordre de première apparition ; sans ordre commun, la première
// apparition seule.
func (t *objTable) keyOrder() []int {
	n := len(t.cols)
	succ := make([]map[int]struct{}, n)
	indeg := make([]int, n)
	for sig := range t.sigs {
		for i := 1; i < len(sig)/2; i++ {
			a := int(sig[2*(i-1)]) | int(sig[2*(i-1)+1])<<8
			b := int(sig[2*i]) | int(sig[2*i+1])<<8
			if succ[a] == nil {
				succ[a] = map[int]struct{}{}
			}
			if _, ok := succ[a][b]; !ok {
				succ[a][b] = struct{}{}
				indeg[b]++
			}
		}
	}
	order := make([]int, 0, n)
	done := make([]bool, n)
	for len(order) < n {
		next := -1
		for i := 0; i < n; i++ {
			if !done[i] && indeg[i] == 0 {
				next = i
				break
			}
		}
		if next < 0 {
			return firstSeenOrder(n)
		}
		done[next] = true
		order = append(order, next)
		for b := range succ[next] {
			indeg[b]--
		}
	}
	return order
}

func firstSeenOrder(n int) []int {
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	return order
}

// tupleTable encode la liste de n-uplets numériques lue en pos en `{"$c":N,"a":A,"c":[...]}`.
func (e *colEncoder) tupleTable(dst []byte, pos int, sv listSurvey) ([]byte, error) {
	cols := make([]column, sv.arity)
	for j := range cols {
		cols[j].cells = make([]cell, 0, sv.n)
	}
	p := e.ws(pos + 1)
	for row := 0; row < sv.n; row++ {
		p = e.ws(p) + 1
		for j := 0; j < sv.arity; j++ {
			p = e.ws(p)
			end, err := e.scalar(p)
			if err != nil {
				return nil, err
			}
			cols[j].cells = append(cols[j].cells, cell{lo: int32(p), hi: int32(end), row: int32(row)})
			p = e.ws(end)
			if e.at(p) == ',' {
				p++
			}
		}
		p = e.ws(p) + 1
		if e.at(p) == ',' {
			p++
		}
	}
	dst = append(dst, `{"$c":`...)
	dst = strconv.AppendInt(dst, int64(sv.n), 10)
	dst = append(dst, `,"a":`...)
	dst = strconv.AppendInt(dst, int64(sv.arity), 10)
	dst = append(dst, `,"c":[`...)
	for j := range cols {
		if j > 0 {
			dst = append(dst, ',')
		}
		dst = e.appendColumn(dst, &cols[j], sv.n, nil)
	}
	return append(dst, ']', '}'), nil
}
