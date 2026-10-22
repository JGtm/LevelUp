package anonymisegamertags

// remplacement_test.go — LA SUBSTITUTION DANS UN TAMPON CLAIR, A TOUT DECALAGE DE BIT.
//
// Un flux de film se lit MSB d abord : le i-eme bit est le bit (7 - i mod 8) de l octet i div 8
// (`source.Bits`). Une chaine posee au bit b se lit donc, alignee, dans la VUE du tampon decalee
// de b mod 8 bits. On cherche dans chacune des huit vues, et on reecrit les bits dans le tampon.

import (
	"bytes"
	"unicode/utf16"
)

// encodage : une forme sous laquelle un nom s ecrit.
type encodage struct {
	nom        string
	aligneSeul bool // UTF-8 : ne se cherche qu aligne sur l octet (texte, faits serialises)
	coder      func(string) []byte
}

var encodages = []encodage{
	{nom: "utf16le", coder: func(s string) []byte { return coderUTF16(s, false) }},
	{nom: "utf16be", coder: func(s string) []byte { return coderUTF16(s, true) }},
	{nom: "utf8", aligneSeul: true, coder: func(s string) []byte { return []byte(s) }},
}

func coderUTF16(s string, grosBoutiste bool) []byte {
	unites := utf16.Encode([]rune(s))
	out := make([]byte, 0, 2*len(unites))
	for _, u := range unites {
		if grosBoutiste {
			out = append(out, byte(u>>8), byte(u))
		} else {
			out = append(out, byte(u), byte(u>>8))
		}
	}
	return out
}

// vueDecalee rend le tampon lu a partir de son bit `d` (1..7) : un octet de moins, le dernier
// etant incomplet.
func vueDecalee(tampon []byte, d int) []byte {
	if len(tampon) < 2 {
		return nil
	}
	out := make([]byte, len(tampon)-1)
	for i := range out {
		out[i] = tampon[i]<<d | tampon[i+1]>>(8-d)
	}
	return out
}

// ecrireOctetsAuBit pose `motif` dans `tampon` a partir du bit `bit0`, MSB d abord.
func ecrireOctetsAuBit(tampon []byte, bit0 int, motif []byte) {
	for j := range len(motif) * 8 {
		i := bit0 + j
		masque := byte(1) << (7 - i%8)
		if motif[j/8]&(1<<(7-j%8)) != 0 {
			tampon[i/8] |= masque
		} else {
			tampon[i/8] &^= masque
		}
	}
}

// remplacer substitue dans `tampon`, EN PLACE, chaque original par son pseudonyme, sous les trois
// encodages et aux huit decalages. Rend le nombre d occurrences remplacees par original.
//
// Une occurrence remplacee n est plus un original : elle ne se recompte pas sous une autre forme
// (une chaine UTF-16LE precedee d un octet nul se lit aussi en UTF-16BE un octet plus tot).
func remplacer(tampon []byte, ps []paire) map[string]int {
	compte := map[string]int{}
	for d := range 8 {
		vue := tampon
		if d > 0 {
			vue = vueDecalee(tampon, d)
		}
		for _, p := range ps {
			for _, e := range encodages {
				if e.aligneSeul && d > 0 {
					continue
				}
				n := remplacerDansVue(tampon, vue, d, e.coder(p.origine), e.coder(p.pseudonyme))
				if n > 0 {
					compte[p.origine] += n
				}
			}
		}
	}
	return compte
}

// remplacerDansVue remplace chaque occurrence de `avant` dans la vue decalee de `d` bits, dans la
// vue (pour les recherches suivantes) et dans le tampon.
func remplacerDansVue(tampon, vue []byte, d int, avant, apres []byte) int {
	n := 0
	for debut := 0; ; {
		k := bytes.Index(vue[debut:], avant)
		if k < 0 {
			return n
		}
		pos := debut + k
		if d == 0 {
			copy(tampon[pos:], apres) // la vue EST le tampon
		} else {
			copy(vue[pos:], apres)
			ecrireOctetsAuBit(tampon, pos*8+d, apres)
		}
		n++
		debut = pos + len(apres)
	}
}
