// Package mapvar — lecture des variantes de carte Halo Infinite (.mvar).
//
// Format : Microsoft Bond CompactBinary v2 (corroboré par InfiniteMITM qui associe
// le content-type alias `:ct-bond` aux réponses `/ugcstorage/map/.../*.mvar`).
//
// cb2.go implémente le lecteur bas niveau générique : varint, zigzag, tags de champ,
// conteneurs, structs longueur-préfixées. Aucune connaissance de la grammaire .mvar
// ici — voir mapvar.go.
//
// Règle de robustesse : tout tag/type inconnu remonte une erreur. On ne saute JAMAIS
// silencieusement un octet : un décalage d'un octet produirait des positions d'objectif
// plausibles mais fausses, ce qui est pire que pas d'objectif du tout.
package mapvar

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Types Bond (BondDataType).
const (
	btStop     byte = 0
	btStopBase byte = 1
	btBool     byte = 2
	btUint8    byte = 3
	btUint16   byte = 4
	btUint32   byte = 5
	btUint64   byte = 6
	btFloat    byte = 7
	btDouble   byte = 8
	btString   byte = 9
	btStruct   byte = 10
	btList     byte = 11
	btSet      byte = 12
	btMap      byte = 13
	btInt8     byte = 14
	btInt16    byte = 15
	btInt32    byte = 16
	btInt64    byte = 17
	btWString  byte = 18
)

// Value est un nœud générique de l'arbre Bond décodé.
type Value struct {
	Type   byte
	Int    int64            // entiers signés
	Uint   uint64           // entiers non signés + bool
	Float  float64          // float/double
	Str    string           // string/wstring
	Fields map[uint16]Value // struct
	Items  []Value          // list/set
	Pairs  []KeyValue       // map
}

// KeyValue est une entrée de conteneur BT_MAP.
type KeyValue struct {
	Key Value
	Val Value
}

// Field retourne le champ d'id donné et un booléen de présence.
func (v Value) Field(id uint16) (Value, bool) {
	if v.Fields == nil {
		return Value{}, false
	}
	f, ok := v.Fields[id]
	return f, ok
}

type decoder struct {
	buf []byte
	pos int
}

func (d *decoder) readByte() (byte, error) {
	if d.pos >= len(d.buf) {
		return 0, fmt.Errorf("cb2: fin de tampon à %d", d.pos)
	}
	b := d.buf[d.pos]
	d.pos++
	return b, nil
}

func (d *decoder) readBytes(n int) ([]byte, error) {
	if n < 0 || d.pos+n > len(d.buf) {
		return nil, fmt.Errorf("cb2: lecture de %d octets hors bornes à %d", n, d.pos)
	}
	out := d.buf[d.pos : d.pos+n]
	d.pos += n
	return out, nil
}

func (d *decoder) readVarint() (uint64, error) {
	var out uint64
	var shift uint
	for {
		b, err := d.readByte()
		if err != nil {
			return 0, err
		}
		out |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return out, nil
		}
		shift += 7
		if shift > 63 {
			return 0, fmt.Errorf("cb2: varint trop long à %d", d.pos)
		}
	}
}

func zigzag(u uint64) int64 { return int64(u>>1) ^ -int64(u&1) }

// readTag lit un octet de tag : type = tag&0x1F, field_id = tag>>5 avec
// extension 1 octet (id nibble 6) ou 2 octets little-endian (id nibble 7).
func (d *decoder) readTag() (typ byte, id uint16, err error) {
	tag, err := d.readByte()
	if err != nil {
		return 0, 0, err
	}
	typ = tag & 0x1F
	switch tag >> 5 {
	case 6:
		b, e := d.readByte()
		if e != nil {
			return 0, 0, e
		}
		id = uint16(b)
	case 7:
		raw, e := d.readBytes(2)
		if e != nil {
			return 0, 0, e
		}
		id = binary.LittleEndian.Uint16(raw)
	default:
		id = uint16(tag >> 5)
	}
	return typ, id, nil
}

// readContainerHeader lit l'en-tête d'un BT_LIST/BT_SET.
// Encodage CompactBinary : si count < 7, un octet = ((count+1)<<5)|elemType ;
// sinon un octet = elemType (3 bits hauts nuls) suivi d'un varint count.
func (d *decoder) readContainerHeader() (elemType byte, count int, err error) {
	b, err := d.readByte()
	if err != nil {
		return 0, 0, err
	}
	elemType = b & 0x1F
	if n := b >> 5; n != 0 {
		return elemType, int(n) - 1, nil
	}
	c, err := d.readVarint()
	if err != nil {
		return 0, 0, err
	}
	return elemType, int(c), nil
}

// readValue décode une valeur du type donné à la position courante.
func (d *decoder) readValue(typ byte) (Value, error) {
	switch typ {
	case btBool, btUint8:
		b, err := d.readByte()
		return Value{Type: typ, Uint: uint64(b)}, err
	case btInt8:
		b, err := d.readByte()
		return Value{Type: typ, Int: int64(int8(b))}, err
	case btUint16, btUint32, btUint64:
		u, err := d.readVarint()
		return Value{Type: typ, Uint: u}, err
	case btInt16, btInt32, btInt64:
		u, err := d.readVarint()
		return Value{Type: typ, Int: zigzag(u)}, err
	case btFloat:
		raw, err := d.readBytes(4)
		if err != nil {
			return Value{}, err
		}
		return Value{Type: typ, Float: float64(math.Float32frombits(binary.LittleEndian.Uint32(raw)))}, nil
	case btDouble:
		raw, err := d.readBytes(8)
		if err != nil {
			return Value{}, err
		}
		return Value{Type: typ, Float: math.Float64frombits(binary.LittleEndian.Uint64(raw))}, nil
	case btString, btWString:
		return d.readStr(typ)
	case btStruct:
		return d.readStruct()
	case btList, btSet:
		return d.readList(typ)
	case btMap:
		return d.readMap()
	default:
		return Value{}, fmt.Errorf("cb2: type Bond inconnu %d à %d", typ, d.pos)
	}
}

func (d *decoder) readStr(typ byte) (Value, error) {
	n, err := d.readVarint()
	if err != nil {
		return Value{}, err
	}
	width := 1
	if typ == btWString {
		width = 2
	}
	raw, err := d.readBytes(int(n) * width)
	if err != nil {
		return Value{}, err
	}
	if typ == btString {
		return Value{Type: typ, Str: string(raw)}, nil
	}
	runes := make([]rune, 0, n)
	for i := 0; i+1 < len(raw); i += 2 {
		runes = append(runes, rune(binary.LittleEndian.Uint16(raw[i:i+2])))
	}
	return Value{Type: typ, Str: string(runes)}, nil
}

func (d *decoder) readList(typ byte) (Value, error) {
	elemType, count, err := d.readContainerHeader()
	if err != nil {
		return Value{}, err
	}
	out := Value{Type: typ, Items: make([]Value, 0, count)}
	for i := 0; i < count; i++ {
		item, err := d.readValue(elemType)
		if err != nil {
			return Value{}, fmt.Errorf("cb2: item %d/%d: %w", i, count, err)
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

func (d *decoder) readMap() (Value, error) {
	keyType, err := d.readByte()
	if err != nil {
		return Value{}, err
	}
	valType, err := d.readByte()
	if err != nil {
		return Value{}, err
	}
	count, err := d.readVarint()
	if err != nil {
		return Value{}, err
	}
	out := Value{Type: btMap, Pairs: make([]KeyValue, 0, count)}
	for i := uint64(0); i < count; i++ {
		k, err := d.readValue(keyType & 0x1F)
		if err != nil {
			return Value{}, err
		}
		v, err := d.readValue(valType & 0x1F)
		if err != nil {
			return Value{}, err
		}
		out.Pairs = append(out.Pairs, KeyValue{Key: k, Val: v})
	}
	return out, nil
}

// readStruct décode un struct CompactBinary v2 : longueur varint puis champs
// jusqu'à BT_STOP. La longueur est vérifiée — un écart est une erreur dure.
func (d *decoder) readStruct() (Value, error) {
	length, err := d.readVarint()
	if err != nil {
		return Value{}, err
	}
	end := d.pos + int(length)
	if end > len(d.buf) {
		return Value{}, fmt.Errorf("cb2: struct de %d octets déborde à %d", length, d.pos)
	}
	out := Value{Type: btStruct, Fields: make(map[uint16]Value)}
	for d.pos < end {
		typ, id, err := d.readTag()
		if err != nil {
			return Value{}, err
		}
		if typ == btStop {
			break
		}
		if typ == btStopBase {
			continue
		}
		val, err := d.readValue(typ)
		if err != nil {
			return Value{}, fmt.Errorf("cb2: champ %d (type %d): %w", id, typ, err)
		}
		out.Fields[id] = val
	}
	if d.pos != end {
		return Value{}, fmt.Errorf("cb2: struct mal terminé (pos %d, fin attendue %d)", d.pos, end)
	}
	return out, nil
}

// DecodeRoot décode un document Bond CompactBinary v2 complet (struct racine
// longueur-préfixée). Erreur si des octets résiduels subsistent.
func DecodeRoot(buf []byte) (Value, error) {
	d := &decoder{buf: buf}
	root, err := d.readStruct()
	if err != nil {
		return Value{}, err
	}
	if d.pos != len(buf) {
		return Value{}, fmt.Errorf("cb2: %d octets résiduels après la racine", len(buf)-d.pos)
	}
	return root, nil
}
