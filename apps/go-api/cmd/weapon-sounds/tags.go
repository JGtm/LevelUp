package main

// tags.go — en-tete `ucsh` d'un fichier de tag et sa TABLE DE DEPENDANCES.
//
// La table de dependances est la piece maitresse de la voie B : elle donne, pour un tag,
// la liste des tags qu'il reference AVEC LEUR GROUPE (fourCC). On peut donc repondre a
// « quel `weap` depend de ce `snd!` ? » sans connaitre un seul nom.
//
// En-tete (0x50 fixes) : +0x00 magic `ucsh` | +0x18 nombre de dependances | ...
// Dependance (0x18) : +0x00 groupe fourCC | +0x08 assetID u64 | +0x10 idGlobal u32
//
//	+0x14 index parent i32

import "encoding/binary"

const (
	magicUCSH     = 0x68736375 // 'ucsh' en little-endian
	tailleEnteteT = 0x50
	tailleDep     = 0x18
	offNbDeps     = 0x18
)

// dependance est une reference sortante d'un tag vers un autre.
type dependance struct {
	Groupe   string
	IDGlobal uint32
}

// fourCCInverse decode un fourCC stocke a l'envers (comme dans les modules).
func fourCCInverse(v uint32) string {
	b := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return ""
		}
	}
	return string(b)
}

// dependances rend les references sortantes d'un tag, ou nil si l'en-tete n'est pas un tag.
func dependances(b []byte) []dependance {
	if len(b) < tailleEnteteT || binary.LittleEndian.Uint32(b) != magicUCSH {
		return nil
	}
	n := int(int32(binary.LittleEndian.Uint32(b[offNbDeps:])))
	if n <= 0 || n > 1<<16 {
		return nil
	}
	out := make([]dependance, 0, n)
	for i := 0; i < n; i++ {
		o := tailleEnteteT + i*tailleDep
		if o+tailleDep > len(b) {
			break
		}
		out = append(out, dependance{
			Groupe:   fourCCInverse(binary.LittleEndian.Uint32(b[o:])),
			IDGlobal: binary.LittleEndian.Uint32(b[o+0x10:]),
		})
	}
	return out
}
