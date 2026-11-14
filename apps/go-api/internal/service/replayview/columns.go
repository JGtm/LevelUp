package replayview

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"strconv"
)

// Encodage en colonnes du document de rejeu servi : un ENCODAGE DE TRANSPORT, jamais une seconde
// forme logique. Le document reste `replaydoc.ReplayDocument` ; `EncodeColumns` réécrit seulement
// son JSON déjà produit (même assainissement des flottants) en regroupant les valeurs par champ,
// ce qui fait gagner un facteur 4 à la compression sans qu'aucune table par champ ne le sache.
//
// # Forme
//
//   - la racine porte `"$enc":1` en première clé ;
//   - toute liste d'au moins 16 éléments qui ne contient que des objets devient
//     `{"$c":N,"k":[clés],"c":[colonnes]}` ; une liste d'au moins 16 n-uplets numériques de même
//     arité (1 à 8) devient `{"$c":N,"a":arité,"c":[colonnes]}` ; l'opération est récursive ;
//   - une liste dont le premier élément n'est ni objet ni liste, une liste plus courte, ou une liste
//     d'objets et de listes mêlés restent des listes (leurs éléments sont encodés un à un) ;
//   - une colonne est un tableau nu de valeurs, ou `{"v":[...],"s":décimales,"d":1,"a":absences}` :
//     `s` est le nombre de décimales (les valeurs de `v` sont des entiers à diviser par 10^s),
//     `d` vaut 1 quand `v` porte les différences successives, `a` dit où la valeur manque ;
//   - l'absence est une propriété à part, distincte de la valeur 0 et de `null` : `a` est soit une
//     liste de longueurs alternées (présent, absent, présent, ...), soit une chaîne de 0 et de 1,
//     la plus courte des deux ; `v` ne porte alors que les valeurs présentes.
//
// # Échelle
//
// L'échelle d'une colonne se lit dans la représentation décimale des jetons, sans calcul flottant :
// c'est le plus grand nombre de décimales des valeurs. Une colonne qui n'est pas entièrement
// numérique, qui contient une notation à exposant, qui dépasse 9 décimales ou dont un entier mis à
// l'échelle dépasse 2^53-1 (borne des entiers sûrs de JavaScript) reste brute, de même qu'une
// colonne dont une différence dépasserait cette borne reste en valeurs absolues. Le delta n'est
// retenu que s'il est strictement plus court.
//
// # Aller-retour
//
// Toutes les valeurs reviennent à l'identique, à une exception près, décidée : `-0` devient `0`
// dans une colonne à échelle (une colonne brute le garde). L'ordre des clés d'un objet est
// restitué quand les objets de la liste suivent un ordre commun, dont les clés absentes sont
// sautées (c'est le cas des structures Go, des cartes triées). Un ordre contradictoire entre deux
// objets d'une même liste donne l'ordre de première apparition, jamais une erreur.
//
// # Refus
//
// L'encodeur refuse (`ErrColumnsKeyCollision`) toute clé d'objet qui commence par `$` dans une
// partie qu'il parcourt : le décodeur la prendrait pour un marqueur. Il refuse aussi un document
// qui n'est pas un objet JSON bien formé, ou un objet qui répète une clé dans une liste
// d'objets (`ErrColumnsUnsupported`). L'appelant sert alors la forme normale.
//
// Le décodeur vit côté web (`apps/web/src/lib/replay/replayColumns.ts`) ; côté Go il n'existe que
// dans un test, qui prouve l'aller-retour.
const (
	columnsMinRows    = 16
	columnsMaxArity   = 8
	columnsMaxDecimal = 9
	columnsSafeInt    = 1<<53 - 1
	columnsMaxColumns = 1<<16 - 1
	columnsKindScalar = 0
	columnsKindBlob   = 1
)

var (
	// ErrColumnsKeyCollision : une clé d'objet commence par `$`.
	ErrColumnsKeyCollision = errors.New("replayview: reserved object key (prefix $): column encoding refused")
	// ErrColumnsUnsupported : le corps n'est pas un objet JSON, est mal formé, ou sort du modèle.
	ErrColumnsUnsupported = errors.New("replayview: body cannot be encoded in columns")
)

// EncodeColumns réécrit le corps JSON `body` (objet racine) dans la forme en colonnes décrite plus
// haut. Un saut de ligne final est conservé. Le corps d'entrée n'est pas modifié.
func EncodeColumns(body []byte) ([]byte, error) {
	if len(body) >= math.MaxInt32 {
		return nil, ErrColumnsUnsupported
	}
	e := &colEncoder{src: body}
	pos := e.ws(0)
	if e.at(pos) != '{' {
		return nil, ErrColumnsUnsupported
	}
	out, end, err := e.object(make([]byte, 0, len(body)/3), pos, true)
	if err != nil {
		return nil, err
	}
	end = e.ws(end)
	if end != len(body) {
		return nil, ErrColumnsUnsupported
	}
	if bytes.HasSuffix(body, []byte("\n")) {
		out = append(out, '\n')
	}
	return out, nil
}

// colEncoder porte le corps lu et les tampons de travail réutilisés d'une colonne à l'autre.
type colEncoder struct {
	src    []byte
	ints   []int64
	deltas []int64
}

// at rend l'octet à pos, ou 0 hors du corps : un corps tronqué se détecte aux jetons attendus.
func (e *colEncoder) at(pos int) byte {
	if pos < 0 || pos >= len(e.src) {
		return 0
	}
	return e.src[pos]
}

func (e *colEncoder) ws(pos int) int {
	for pos < len(e.src) {
		switch e.src[pos] {
		case ' ', '\t', '\n', '\r':
			pos++
		default:
			return pos
		}
	}
	return pos
}

// str rend la position qui suit la chaîne ouverte en pos.
func (e *colEncoder) str(pos int) (int, error) {
	for i := pos + 1; i < len(e.src); i++ {
		switch e.src[i] {
		case '\\':
			i++
		case '"':
			return i + 1, nil
		}
	}
	return 0, ErrColumnsUnsupported
}

// scalar rend la position qui suit le jeton scalaire (chaîne, nombre, true, false, null) lu en pos.
func (e *colEncoder) scalar(pos int) (int, error) {
	if e.at(pos) == '"' {
		return e.str(pos)
	}
	i := pos
	for i < len(e.src) {
		switch e.src[i] {
		case ',', ']', '}', ':', ' ', '\t', '\n', '\r':
			if i == pos {
				return 0, ErrColumnsUnsupported
			}
			return i, nil
		}
		i++
	}
	return 0, ErrColumnsUnsupported
}

// skip rend la position qui suit la valeur lue en pos, sans l'interpréter.
func (e *colEncoder) skip(pos int) (int, error) {
	pos = e.ws(pos)
	if c := e.at(pos); c != '{' && c != '[' {
		return e.scalar(pos)
	}
	depth := 0
	for i := pos; i < len(e.src); i++ {
		switch e.src[i] {
		case '"':
			end, err := e.str(i)
			if err != nil {
				return 0, err
			}
			i = end - 1
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i + 1, nil
			}
		}
	}
	return 0, ErrColumnsUnsupported
}

// reservedKey dit si la clé (guillemets compris) commence par `$` une fois les échappements levés.
func reservedKey(raw []byte) bool {
	if len(raw) >= 2 && raw[1] == '$' {
		return true
	}
	if bytes.IndexByte(raw, '\\') < 0 {
		return false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return true
	}
	return len(s) > 0 && s[0] == '$'
}

// value encode la valeur lue en pos et rend le tampon prolongé et la position qui suit.
func (e *colEncoder) value(dst []byte, pos int) ([]byte, int, error) {
	pos = e.ws(pos)
	switch e.at(pos) {
	case '{':
		return e.object(dst, pos, false)
	case '[':
		return e.array(dst, pos)
	}
	end, err := e.scalar(pos)
	if err != nil {
		return nil, 0, err
	}
	return append(dst, e.src[pos:end]...), end, nil
}

// object encode l'objet lu en pos. Avec `root`, la clé `$enc` est posée en tête.
func (e *colEncoder) object(dst []byte, pos int, root bool) ([]byte, int, error) {
	first := true
	if root {
		dst = append(dst, `{"$enc":1`...)
		first = false
	} else {
		dst = append(dst, '{')
	}
	p := e.ws(pos + 1)
	if e.at(p) == '}' {
		return append(dst, '}'), p + 1, nil
	}
	for {
		p = e.ws(p)
		if e.at(p) != '"' {
			return nil, 0, ErrColumnsUnsupported
		}
		kend, err := e.str(p)
		if err != nil {
			return nil, 0, err
		}
		if reservedKey(e.src[p:kend]) {
			return nil, 0, ErrColumnsKeyCollision
		}
		if !first {
			dst = append(dst, ',')
		}
		first = false
		dst = append(dst, e.src[p:kend]...)
		dst = append(dst, ':')
		p = e.ws(kend)
		if e.at(p) != ':' {
			return nil, 0, ErrColumnsUnsupported
		}
		if dst, p, err = e.value(dst, p+1); err != nil {
			return nil, 0, err
		}
		p = e.ws(p)
		switch e.at(p) {
		case ',':
			p++
		case '}':
			return append(dst, '}'), p + 1, nil
		default:
			return nil, 0, ErrColumnsUnsupported
		}
	}
}

// listSurvey est ce qu'un premier passage sur une liste apprend, avant de choisir sa forme.
type listSurvey struct {
	n      int
	end    int
	first  byte // premier octet du premier élément
	allObj bool
	arity  int // > 0 : n-uplets numériques de cette arité
}

// survey parcourt la liste ouverte en pos sans rien encoder.
func (e *colEncoder) survey(pos int) (listSurvey, error) {
	sv := listSurvey{allObj: true, arity: -1}
	p := e.ws(pos + 1)
	if e.at(p) == ']' {
		sv.end = p + 1
		return sv, nil
	}
	sv.first = e.at(p)
	for {
		p = e.ws(p)
		end, err := e.skip(p)
		if err != nil {
			return sv, err
		}
		if e.at(p) != '{' {
			sv.allObj = false
		}
		sv.noteTuple(e, p, end)
		sv.n++
		p = e.ws(end)
		switch e.at(p) {
		case ',':
			p++
		case ']':
			sv.end = p + 1
			return sv, nil
		default:
			return sv, ErrColumnsUnsupported
		}
	}
}

// noteTuple tient l'arité commune des éléments tant qu'ils sont tous des n-uplets numériques.
func (sv *listSurvey) noteTuple(e *colEncoder, lo, hi int) {
	if sv.arity == 0 {
		return
	}
	a := -1
	if e.at(lo) == '[' {
		a = e.tupleArity(lo, hi)
	}
	if a < 1 || a > columnsMaxArity || (sv.arity > 0 && a != sv.arity) {
		sv.arity = 0
		return
	}
	sv.arity = a
}

// tupleArity rend le nombre de nombres de la liste [lo, hi), ou -1 si elle contient autre chose.
func (e *colEncoder) tupleArity(lo, hi int) int {
	p := e.ws(lo + 1)
	count := 0
	for p < hi {
		if c := e.at(p); c != '-' && (c < '0' || c > '9') {
			return -1
		}
		end, err := e.scalar(p)
		if err != nil {
			return -1
		}
		count++
		p = e.ws(end)
		if e.at(p) == ',' {
			p = e.ws(p + 1)
			continue
		}
		break
	}
	if e.at(p) != ']' {
		return -1
	}
	return count
}

// array encode la liste lue en pos : tableau de colonnes si elle s'y prête, sinon élément par élément.
func (e *colEncoder) array(dst []byte, pos int) ([]byte, int, error) {
	sv, err := e.survey(pos)
	if err != nil {
		return nil, 0, err
	}
	switch {
	case sv.n == 0:
		return append(dst, '[', ']'), sv.end, nil
	case sv.first != '{' && sv.first != '[':
		return append(dst, e.src[pos:sv.end]...), sv.end, nil
	case sv.n >= columnsMinRows && sv.allObj:
		dst, err = e.objectTable(dst, pos, sv)
	case sv.n >= columnsMinRows && sv.arity > 0:
		dst, err = e.tupleTable(dst, pos, sv)
	default:
		dst, err = e.elements(dst, pos, sv)
	}
	return dst, sv.end, err
}

// elements encode chaque élément de la liste, sans en changer la forme.
func (e *colEncoder) elements(dst []byte, pos int, sv listSurvey) ([]byte, error) {
	dst = append(dst, '[')
	p := e.ws(pos + 1)
	for i := 0; i < sv.n; i++ {
		var err error
		if i > 0 {
			dst = append(dst, ',')
		}
		if dst, p, err = e.value(dst, p); err != nil {
			return nil, err
		}
		p = e.ws(p)
		if e.at(p) == ',' {
			p = e.ws(p + 1)
		}
	}
	return append(dst, ']'), nil
}

// appendJSONInt écrit n en décimal.
func appendJSONInt(dst []byte, n int64) []byte { return strconv.AppendInt(dst, n, 10) }
