package replayview_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"testing"
)

// Le DÉCODEUR de la forme en colonnes, côté Go, n'existe que dans ce fichier de test : il prouve
// l'aller-retour de `EncodeColumns`. Le décodeur de production est celui du web
// (`apps/web/src/lib/replay/replayColumns.ts`), dont un test lit les mêmes fixtures.

// ordObj est un objet JSON qui garde l'ordre de ses clés.
type ordObj struct {
	keys []string
	vals []any
}

func (o *ordObj) get(k string) (any, bool) {
	for i, key := range o.keys {
		if key == k {
			return o.vals[i], true
		}
	}
	return nil, false
}

// parseOrdered lit du JSON en arbre ordonné ; les nombres restent des json.Number.
func parseOrdered(t testing.TB, raw []byte) any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	v, err := readOrdered(dec)
	if err != nil {
		t.Fatalf("JSON illisible : %v", err)
	}
	return v
}

func readOrdered(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	d, isDelim := tok.(json.Delim)
	if !isDelim {
		return tok, nil
	}
	if d == '[' {
		out := []any{}
		for dec.More() {
			v, err := readOrdered(dec)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		_, err := dec.Token()
		return out, err
	}
	o := &ordObj{}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, err
		}
		v, err := readOrdered(dec)
		if err != nil {
			return nil, err
		}
		o.keys = append(o.keys, kt.(string))
		o.vals = append(o.vals, v)
	}
	_, err = dec.Token()
	return o, err
}

// decodeColumnsDoc défait la forme en colonnes d'un arbre ordonné ; un document sans `$enc` est
// rendu tel quel.
func decodeColumnsDoc(t testing.TB, root any) any {
	t.Helper()
	o, ok := root.(*ordObj)
	if !ok {
		return root
	}
	if v, has := o.get("$enc"); !has || fmt.Sprint(v) != "1" {
		return root
	}
	out := &ordObj{}
	for i, k := range o.keys {
		if k != "$enc" {
			out.keys = append(out.keys, k)
			out.vals = append(out.vals, o.vals[i])
		}
	}
	return decodeValue(t, out)
}

func isContainer(v any) bool {
	switch v.(type) {
	case *ordObj, []any:
		return true
	}
	return false
}

func decodeValue(t testing.TB, v any) any {
	switch x := v.(type) {
	case []any:
		if len(x) == 0 || !isContainer(x[0]) {
			return x
		}
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = decodeValue(t, e)
		}
		return out
	case *ordObj:
		if _, ok := x.get("$c"); ok {
			return decodeTable(t, x)
		}
		out := &ordObj{keys: x.keys, vals: make([]any, len(x.vals))}
		for i, e := range x.vals {
			out.vals[i] = decodeValue(t, e)
		}
		return out
	}
	return v
}

func mustInt(t testing.TB, v any) int64 {
	t.Helper()
	n, ok := v.(json.Number)
	if !ok {
		t.Fatalf("entier attendu, reçu %T", v)
	}
	i, err := n.Int64()
	if err != nil {
		t.Fatalf("entier attendu : %v", err)
	}
	return i
}

// decodedColumn rend les valeurs présentes de la colonne et, si elle est creuse, sa présence.
func decodedColumn(t testing.TB, col any, n int) (vals []any, present []bool) {
	t.Helper()
	switch c := col.(type) {
	case []any:
		out := make([]any, len(c))
		for i, e := range c {
			out[i] = decodeValue(t, e)
		}
		return out, nil
	case *ordObj:
		return decodedColumnObject(t, c, n)
	}
	t.Fatalf("colonne de type inattendu %T", col)
	return nil, nil
}

func decodedColumnObject(t testing.TB, c *ordObj, n int) ([]any, []bool) {
	t.Helper()
	vAny, _ := c.get("v")
	list, _ := vAny.([]any)
	scale, delta := int64(0), false
	if s, ok := c.get("s"); ok {
		scale = mustInt(t, s)
	}
	if d, ok := c.get("d"); ok {
		delta = mustInt(t, d) == 1
	}
	vals := make([]any, len(list))
	var acc int64
	for i, e := range list {
		if !delta && scale == 0 {
			vals[i] = decodeValue(t, e)
			continue
		}
		x := mustInt(t, e)
		if delta {
			acc += x
			x = acc
		}
		vals[i] = float64(x) / math.Pow10(int(scale))
	}
	if a, ok := c.get("a"); ok {
		return vals, decodedPresence(t, a, n)
	}
	return vals, nil
}

func decodedPresence(t testing.TB, a any, n int) []bool {
	t.Helper()
	present := make([]bool, n)
	switch x := a.(type) {
	case string:
		for i := 0; i < n; i++ {
			present[i] = x[i] == '1'
		}
	case []any:
		p, on := 0, true
		for _, r := range x {
			l := int(mustInt(t, r))
			for i := p; on && i < p+l; i++ {
				present[i] = true
			}
			p += l
			on = !on
		}
	default:
		t.Fatalf("absence de type inattendu %T", a)
	}
	return present
}

func decodeTable(t testing.TB, tab *ordObj) any {
	t.Helper()
	cAny, _ := tab.get("$c")
	n := int(mustInt(t, cAny))
	colsAny, _ := tab.get("c")
	colList := colsAny.([]any)
	type decoded struct {
		vals    []any
		present []bool
	}
	cols := make([]decoded, len(colList))
	for j, c := range colList {
		cols[j].vals, cols[j].present = decodedColumn(t, c, n)
	}
	rows := make([]any, n)
	if aAny, ok := tab.get("a"); ok {
		arity := int(mustInt(t, aAny))
		for i := range rows {
			row := make([]any, arity)
			for j := range row {
				row[j] = cols[j].vals[i]
			}
			rows[i] = row
		}
		return rows
	}
	kAny, _ := tab.get("k")
	keys := kAny.([]any)
	next := make([]int, len(cols))
	for i := range rows {
		o := &ordObj{}
		for j, k := range keys {
			c := cols[j]
			if c.present != nil && !c.present[i] {
				continue
			}
			var v any
			if c.present == nil {
				v = c.vals[i]
			} else {
				v = c.vals[next[j]]
				next[j]++
			}
			o.keys = append(o.keys, k.(string))
			o.vals = append(o.vals, v)
		}
		rows[i] = o
	}
	return rows
}

// equalOrdered compare deux arbres ordonnés : mêmes clés dans le même ordre, mêmes valeurs ; deux
// nombres sont égaux s'ils valent la même valeur flottante (`-0` et `0` sont la même valeur).
func equalOrdered(a, b any, path string) (string, bool) {
	switch x := a.(type) {
	case *ordObj:
		y, ok := b.(*ordObj)
		if !ok || len(x.keys) != len(y.keys) {
			return path + " (objet)", false
		}
		for i := range x.keys {
			if x.keys[i] != y.keys[i] {
				return path + " (ordre ou nom de clé " + x.keys[i] + " / " + y.keys[i] + ")", false
			}
			if p, ok := equalOrdered(x.vals[i], y.vals[i], path+"."+x.keys[i]); !ok {
				return p, false
			}
		}
		return "", true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return path + " (liste)", false
		}
		for i := range x {
			if p, ok := equalOrdered(x[i], y[i], path+"["+strconv.Itoa(i)+"]"); !ok {
				return p, false
			}
		}
		return "", true
	}
	fa, oka := asFloat(a)
	fb, okb := asFloat(b)
	if oka && okb {
		if fa == fb {
			return "", true
		}
		return fmt.Sprintf("%s (%v / %v)", path, a, b), false
	}
	if a == b {
		return "", true
	}
	return fmt.Sprintf("%s (%v / %v)", path, a, b), false
}

func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case json.Number:
		f, err := strconv.ParseFloat(x.String(), 64)
		return f, err == nil
	}
	return 0, false
}

// readAllString lit un lecteur jusqu'au bout.
func readAllString(t testing.TB, r io.Reader) []byte {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
