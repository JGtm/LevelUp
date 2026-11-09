package hinavmesh

// enveloppe.go — L'EMBALLAGE COMMUN DES BLOBS PUBLIES A COTE D'UN `.mvar`.
//
// Les trois blobs d'un asset UGC de carte Forge (`navmesh.blob`, `audioocclusion.blob`,
// `lightprobes.blob`) portent le MEME emballage, et c'est MESURE sur Isolation
// (01af558d) : les trois commencent par le meme en-tete gros-boutiste de 12 octets
// [u32 version=2][u32 taille-12][u32 0x001FFFFF], suivi du meme conteneur Bond
// CompactBinary v2 dont le champ 1 est un flux zlib d'en-tete `58 09`. Le deballage
// vit donc ICI, une seule fois, et non dans chaque lecteur de blob.
//
// POURQUOI UN LECTEUR BOND DEDIE PLUTOT QUE `mapvar.DecodeRoot`. Le decodeur generique
// materialise un `mapvar.Value` (~104 octets) PAR OCTET de charge comprimee : les
// 9,1 Mo de `lightprobes.blob` demanderaient pres d'un gigaoctet de tas, et c'est ce
// cout qui imposait le plafond `tailleComprimeeMax` de 8 Mo — un plafond qui interdisait
// purement et simplement de lire ce blob-la. Le lecteur ci-dessous ne parcourt que
// l'EN-TETE des champs de la racine et tranche la charge sans la recopier : cout
// constant, plus de plafond arbitraire. Il reste STRICT au sens de cb2.go — tout champ
// ou type inattendu est une erreur, jamais un octet saute en silence.
//
// La forme de la racine est celle deja documentee par navmesh.go, re-verifiee ici :
//
//	champ 0 : BT_INT32  = 1        (role inconnu, non utilise)
//	champ 1 : BT_LIST<BT_INT8>     — le flux zlib
//	champ 2 : BT_UINT32            — la taille attendue apres inflate

import (
	"encoding/binary"
	"fmt"
)

// Types Bond utilises par la racine d'un blob. Ce sont les memes valeurs que cb2.go ;
// on ne re-decode pas un document Bond quelconque, seulement cette racine-la.
const (
	bondStop     byte = 0
	bondStopBase byte = 1
	bondUint32   byte = 5
	bondList     byte = 11
	bondInt8     byte = 14
	bondInt32    byte = 16
)

// champRoleInconnu est le champ 0 de la racine : un i32 valant 1 sur les trois blobs
// mesures. On le lit pour ne pas le sauter en aveugle, on n'en fait rien.
const champRoleInconnu = 0

// enveloppe est ce qu'un blob UGC porte autour de sa charge.
type enveloppe struct {
	comprime       []byte // le flux zlib, TRANCHE du blob d'origine (aucune copie)
	tailleInflatee int
}

// decompresse deroule l'emballage d'un blob publie a cote d'un `.mvar`
// (`navmesh.blob`, `audioocclusion.blob`, `lightprobes.blob`) et rend sa charge inflatee.
//
// La taille inflatee declaree par le conteneur est VERIFIEE contre la taille obtenue : un
// ecart signifie que le flux n'est pas celui que le conteneur annonce, et il vaut mieux
// echouer que rendre une charge tronquee dont le parcours produirait des coordonnees
// plausibles et fausses.
func decompresse(blob []byte) ([]byte, error) {
	env, err := lisEnveloppe(blob)
	if err != nil {
		return nil, err
	}
	charge, err := inflate(env.comprime, env.tailleInflatee)
	if err != nil {
		return nil, err
	}
	if len(charge) != env.tailleInflatee {
		return nil, fmt.Errorf("hinavmesh: inflate rend %d octets, le conteneur en declare %d",
			len(charge), env.tailleInflatee)
	}
	return charge, nil
}

// lisEnveloppe verifie l'en-tete proprietaire puis extrait le flux zlib du conteneur Bond.
func lisEnveloppe(blob []byte) (enveloppe, error) {
	var env enveloppe
	if len(blob) < tailleEntete {
		return env, fmt.Errorf("hinavmesh: blob de %d octets, en-tete de %d attendue", len(blob), tailleEntete)
	}
	if version := binary.BigEndian.Uint32(blob[0:]); version != versionAttendue {
		return env, fmt.Errorf("hinavmesh: version de blob %d inconnue (attendu %d)", version, versionAttendue)
	}
	if declaree := int(binary.BigEndian.Uint32(blob[4:])); declaree != len(blob)-tailleEntete {
		return env, fmt.Errorf("hinavmesh: l'en-tete annonce %d octets de charge, le fichier en porte %d",
			declaree, len(blob)-tailleEntete)
	}
	// Le troisieme champ (0x001FFFFF) est CONSTANT sur les blobs mesures, et sa
	// signification est inconnue : on ne s'en sert pas, et on ne le suppose pas non plus
	// invariant en echouant dessus.
	return lisRacineBond(blob[tailleEntete:])
}

// lisRacineBond parcourt la racine Bond du conteneur et rend le flux zlib avec la taille
// inflatee declaree. Cout constant : la charge est TRANCHEE, jamais recopiee.
func lisRacineBond(bond []byte) (enveloppe, error) {
	var env enveloppe
	d := &lecteurBond{buf: bond}
	longueur, err := d.varint()
	if err != nil {
		return env, fmt.Errorf("hinavmesh: longueur de la racine Bond: %w", err)
	}
	fin := d.pos + int(longueur)
	if fin != len(bond) {
		return env, fmt.Errorf("hinavmesh: la racine Bond declare %d octets, le conteneur en porte %d",
			longueur, len(bond)-d.pos)
	}
	vus := map[uint16]bool{}
	for d.pos < fin {
		typ, id, err := d.tag()
		if err != nil {
			return env, err
		}
		if typ == bondStop {
			break
		}
		if typ == bondStopBase {
			continue
		}
		if err := d.litChampRacine(&env, id, typ); err != nil {
			return env, err
		}
		vus[id] = true
	}
	if !vus[champCharge] {
		return env, fmt.Errorf("hinavmesh: le conteneur Bond ne porte pas le champ %d (charge)", champCharge)
	}
	if !vus[champTailleInflatee] {
		return env, fmt.Errorf("hinavmesh: le conteneur Bond ne porte pas le champ %d (taille inflatee)", champTailleInflatee)
	}
	if len(env.comprime) == 0 {
		return env, fmt.Errorf("hinavmesh: charge vide")
	}
	return env, nil
}

// litChampRacine consomme UN champ de la racine et le range dans l'enveloppe.
//
// Doctrine cb2.go : on ne saute JAMAIS un octet en silence. Un couple (id, type)
// inattendu, c'est un emballage qu'on ne connait pas — donc une erreur, pas un champ
// ignore.
func (d *lecteurBond) litChampRacine(env *enveloppe, id uint16, typ byte) error {
	switch {
	case id == champRoleInconnu && typ == bondInt32:
		_, err := d.varint()
		return err
	case id == champCharge && typ == bondList:
		comprime, err := d.listeInt8()
		if err != nil {
			return err
		}
		env.comprime = comprime
		return nil
	case id == champTailleInflatee && typ == bondUint32:
		n, err := d.varint()
		if err != nil {
			return err
		}
		if n > tailleInflateeMax {
			return fmt.Errorf("hinavmesh: taille inflatee declaree %d au-dela du plafond %d",
				n, tailleInflateeMax)
		}
		env.tailleInflatee = int(n)
		return nil
	default:
		return fmt.Errorf("hinavmesh: champ Bond inattendu (id %d, type %d) a %d", id, typ, d.pos)
	}
}

// lecteurBond est le strict necessaire de CompactBinary v2 pour cette racine-la.
type lecteurBond struct {
	buf []byte
	pos int
}

func (d *lecteurBond) octet() (byte, error) {
	if d.pos >= len(d.buf) {
		return 0, fmt.Errorf("hinavmesh: fin de conteneur a %d", d.pos)
	}
	b := d.buf[d.pos]
	d.pos++
	return b, nil
}

func (d *lecteurBond) varint() (uint64, error) {
	var out uint64
	var shift uint
	for {
		b, err := d.octet()
		if err != nil {
			return 0, err
		}
		out |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return out, nil
		}
		if shift += 7; shift > 63 {
			return 0, fmt.Errorf("hinavmesh: varint trop long a %d", d.pos)
		}
	}
}

// tag lit un octet de tag : type = tag&0x1F, id = tag>>5 avec extension 1 octet (nibble 6)
// ou 2 octets petit-boutistes (nibble 7).
func (d *lecteurBond) tag() (typ byte, id uint16, err error) {
	t, err := d.octet()
	if err != nil {
		return 0, 0, err
	}
	typ = t & 0x1F
	switch t >> 5 {
	case 6:
		b, e := d.octet()
		if e != nil {
			return 0, 0, e
		}
		id = uint16(b)
	case 7:
		if d.pos+2 > len(d.buf) {
			return 0, 0, fmt.Errorf("hinavmesh: id de champ tronque a %d", d.pos)
		}
		id = binary.LittleEndian.Uint16(d.buf[d.pos:])
		d.pos += 2
	default:
		id = uint16(t >> 5)
	}
	return typ, id, nil
}

// listeInt8 lit un BT_LIST<BT_INT8> et rend ses octets SANS copie. C'est tout l'interet du
// lecteur dedie : la charge comprimee est une tranche du blob, pas un million de Value.
func (d *lecteurBond) listeInt8() ([]byte, error) {
	b, err := d.octet()
	if err != nil {
		return nil, err
	}
	elemType := b & 0x1F
	n := int(b >> 5)
	if n != 0 {
		n--
	} else {
		c, err := d.varint()
		if err != nil {
			return nil, err
		}
		n = int(c)
	}
	if elemType != bondInt8 {
		return nil, fmt.Errorf("hinavmesh: la charge est une liste de type %d, int8 attendu", elemType)
	}
	if n < 0 || d.pos+n > len(d.buf) {
		return nil, fmt.Errorf("hinavmesh: charge de %d octets hors bornes a %d", n, d.pos)
	}
	out := d.buf[d.pos : d.pos+n]
	d.pos += n
	return out, nil
}
