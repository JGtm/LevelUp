package main

// uslg.go — LES TEXTES DES REPLIQUES : listes de chaînes localisées (`uslg`) du jeu.
//
// Le tag `uslg` ne porte que ses langues ; son blob de ressources empile un sous-fichier `ucsh` par langue,
// dans l'ordre du bloc de langues du tag (format détaillé dans `himap/uslg.go`, brique des cartes). Chaque
// sous-fichier est une table d'index {empreinte, offset} et un bloc de chaînes UTF-8 terminées par zéro : la
// clé de texte d'une réplique est l'empreinte.
//
// Les index de langue du jeu utilisés ici (0 anglais, 3 français) sont ceux que `himap` a vérifiés ; l'ordre
// des slots n'est croissant que jusqu'au dixième, ce qui suffit pour ces deux langues. Une langue de plus
// s'ajoute dans `languesDuJeu` après avoir lu son index dans le bloc de langues du tag.

import (
	"bytes"
	"encoding/binary"
)

const (
	magicUcsh        = "ucsh"
	ucshEnteteFixe   = 0x50
	ucshOffDeps      = 0x18
	ucshOffEntete    = 0x38
	ucshDepStride    = 0x18
	ucshBlocStride   = 0x10
	ucshOffBlocTaill = 0
	ucshOffBlocPos   = 8
	ucshTailleMini   = 0x80
	ucshIndexStride  = 8
)

// sousFichiersUcsh coupe le blob de ressources d'un `uslg` à chaque `ucsh`.
func sousFichiersUcsh(d []byte) [][]byte {
	var debuts []int
	for i := 0; ; {
		j := bytes.Index(d[i:], []byte(magicUcsh))
		if j < 0 {
			break
		}
		debuts = append(debuts, i+j)
		i += j + len(magicUcsh)
	}
	out := make([][]byte, 0, len(debuts))
	for k, a := range debuts {
		e := len(d)
		if k+1 < len(debuts) {
			e = debuts[k+1]
		}
		out = append(out, d[a:e])
	}
	return out
}

// tableUcsh lit la table clé -> texte d'un sous-fichier `ucsh`. Un sous-fichier illisible rend une table
// vide : l'appelant constate l'absence de texte au lieu de recevoir une erreur par langue.
func tableUcsh(b []byte) map[uint32]string {
	out := map[uint32]string{}
	if len(b) < ucshTailleMini {
		return out
	}
	deps := int(binary.LittleEndian.Uint32(b[ucshOffDeps:]))
	entete := int(binary.LittleEndian.Uint32(b[ucshOffEntete:]))
	base := ucshEnteteFixe + deps*ucshDepStride
	if deps < 0 || base+3*ucshBlocStride > len(b) {
		return out
	}
	var taille, pos [3]int
	for k := 0; k < 3; k++ {
		taille[k] = int(binary.LittleEndian.Uint32(b[base+k*ucshBlocStride+ucshOffBlocTaill:]))
		pos[k] = int(binary.LittleEndian.Uint32(b[base+k*ucshBlocStride+ucshOffBlocPos:]))
	}
	if entete+pos[2]+taille[2] > len(b) || entete+pos[1]+taille[1] > len(b) {
		return out
	}
	index := b[entete+pos[1] : entete+pos[1]+taille[1]]
	chaines := b[entete+pos[2] : entete+pos[2]+taille[2]]
	for i := 0; i+ucshIndexStride <= len(index); i += ucshIndexStride {
		cle := binary.LittleEndian.Uint32(index[i:])
		o := int(binary.LittleEndian.Uint32(index[i+4:]))
		if o >= len(chaines) {
			continue
		}
		fin := bytes.IndexByte(chaines[o:], 0)
		if fin < 0 {
			continue
		}
		out[cle] = string(chaines[o : o+fin])
	}
	return out
}

// Le tag `snd!` : un en-tête `ucsh` de 0x50 octets, une table de dépendances (groupe, identifiant) de
// 0x18 octets par entrée, puis des mots de quatre octets où les couples (évènement, clé de texte) se lisent.
const (
	sndTailleEntete = 0x50
	sndOffNbDeps    = 0x18
	sndOffDepID     = 0x10
)

// dependancesSnd rend les groupes (`sbnk`, `sndc`, `uslg`) et identifiants des dépendances d'un tag `snd!`.
func dependancesSnd(d []byte) (groupes []string, ids []uint32) {
	if len(d) < sndTailleEntete || string(d[:4]) != magicUcsh {
		return nil, nil
	}
	n := int(int32(binary.LittleEndian.Uint32(d[sndOffNbDeps:])))
	for i := 0; i < n && sndTailleEntete+i*ucshDepStride+ucshDepStride <= len(d); i++ {
		o := sndTailleEntete + i*ucshDepStride
		v := binary.LittleEndian.Uint32(d[o:])
		groupes = append(groupes, string([]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}))
		ids = append(ids, binary.LittleEndian.Uint32(d[o+sndOffDepID:]))
	}
	return groupes, ids
}

// motsSnd rend les mots de quatre octets qui suivent la table de dépendances d'un tag `snd!`.
func motsSnd(d []byte, nbDeps int) []uint32 {
	var mots []uint32
	for o := sndTailleEntete + nbDeps*ucshDepStride; o+4 <= len(d); o += 4 {
		mots = append(mots, binary.LittleEndian.Uint32(d[o:]))
	}
	return mots
}
