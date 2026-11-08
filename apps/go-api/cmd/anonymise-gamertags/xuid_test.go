package anonymisegamertags

// xuid_test.go — LES XUIDS : la liste des originaux, le fictif de chacun, et leurs formes.
//
// Un xuid est un entier de 64 bits. Un film l ecrit sous deux formes binaires, a un decalage de bit
// quelconque : petit-boutiste (paquets d identite, evenements, temps forts) ou gros-boutiste, MSB
// d abord (table des joueurs du registre). Les derives textuels le portent en decimal a seize
// chiffres, ou en hexadecimal a seize chiffres dans l ordre de lecture ou octet par octet inverse
// (prose des commentaires qui citent un champ du film). Toutes ces formes ont la meme longueur
// avant et apres : seuls des bits changent.

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/identitefictive"
)

// Les formes d un xuid, telles que le bilan les nomme.
const (
	formeXuidLE      = "xuid-le"
	formeXuidMSB     = "xuid-msb"
	formeXuidDecimal = "xuid-decimal"
	formeXuidHexa    = "xuid-hexa"
	formeXuidHexaLE  = "xuid-hexa-octets-inverses"
)

// paireXuid : un xuid reel et son fictif.
type paireXuid struct {
	reel, fictif uint64
}

// jeu : tout ce que l outil remplace. `vus` collecte les xuids d origine effectivement trouves.
type jeu struct {
	noms  []paire
	xuids []paireXuid
	vus   map[uint64]bool
}

func nouveauJeu(noms []paire, xuids []paireXuid) *jeu {
	return &jeu{noms: noms, xuids: xuids, vus: map[uint64]bool{}}
}

// motifXuid : une forme d un xuid a chercher, et ce qui la remplace.
type motifXuid struct {
	forme        string
	reel         uint64
	avant, apres []byte
	texte        bool // aligne sur l octet, a bornes nettes
	borne        func(vue []byte, pos, n int) bool
}

// motifsXuid developpe chaque paire en ses formes.
func (j *jeu) motifsXuid() []motifXuid {
	var out []motifXuid
	for _, p := range j.xuids {
		out = append(out,
			motifXuid{forme: formeXuidLE, reel: p.reel, avant: octetsLE(p.reel), apres: octetsLE(p.fictif)},
			motifXuid{forme: formeXuidMSB, reel: p.reel, avant: octetsBE(p.reel), apres: octetsBE(p.fictif)},
			texteXuid(formeXuidDecimal, p, strconv.FormatUint(p.reel, 10), strconv.FormatUint(p.fictif, 10), estChiffre),
		)
		for _, maj := range []bool{false, true} {
			out = append(out,
				texteXuid(formeXuidHexa, p, hexa(p.reel, maj), hexa(p.fictif, maj), estChiffreHexa),
				texteXuid(formeXuidHexaLE, p, hexa(bits.ReverseBytes64(p.reel), maj),
					hexa(bits.ReverseBytes64(p.fictif), maj), estChiffreHexa))
		}
	}
	return out
}

func texteXuid(forme string, p paireXuid, avant, apres string, alphabet func(byte) bool) motifXuid {
	return motifXuid{forme: forme, reel: p.reel, avant: []byte(avant), apres: []byte(apres), texte: true,
		borne: func(vue []byte, pos, n int) bool {
			return (pos == 0 || !alphabet(vue[pos-1])) && (pos+n >= len(vue) || !alphabet(vue[pos+n]))
		}}
}

func hexa(v uint64, majuscules bool) string {
	if majuscules {
		return fmt.Sprintf("%016X", v)
	}
	return fmt.Sprintf("%016x", v)
}

func estChiffre(b byte) bool { return b >= '0' && b <= '9' }

func estChiffreHexa(b byte) bool {
	return estChiffre(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func octetsLE(v uint64) []byte { return binary.LittleEndian.AppendUint64(nil, v) }
func octetsBE(v uint64) []byte { return binary.BigEndian.AppendUint64(nil, v) }

// lireXuidsOriginaux lit la liste des xuids d origine (un par ligne, decimal ; lignes vides et `#`
// ignorees) et rend la table original -> fictif, validee.
func lireXuidsOriginaux(r io.Reader) ([]paireXuid, error) {
	var reels []uint64
	sc := bufio.NewScanner(r)
	for ligne := 1; sc.Scan(); ligne++ {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		v, err := strconv.ParseUint(l, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("ligne %d : %q n est pas un xuid decimal", ligne, l)
		}
		reels = append(reels, v)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("lecture des xuids : %w", err)
	}
	return tableXuids(reels)
}

// tableXuids derive le fictif de chaque xuid et refuse ce qui rendrait la substitution ambigue :
// un original hors plage ou en double, deux originaux de meme fictif, un fictif qui est aussi un
// original (il serait reecrit une seconde fois).
func tableXuids(reels []uint64) ([]paireXuid, error) {
	if len(reels) == 0 {
		return nil, fmt.Errorf("liste de xuids vide")
	}
	sort.Slice(reels, func(i, j int) bool { return reels[i] < reels[j] })
	originaux := map[uint64]bool{}
	for _, r := range reels {
		if !identitefictive.DansLaPlage(r) {
			return nil, fmt.Errorf("%d : hors de la plage des xuids", r)
		}
		if originaux[r] {
			return nil, fmt.Errorf("%d : xuid en double", r)
		}
		originaux[r] = true
	}
	fictifs := map[uint64]uint64{}
	out := make([]paireXuid, 0, len(reels))
	for _, r := range reels {
		f := identitefictive.Xuid(r)
		if autre, dup := fictifs[f]; dup {
			return nil, fmt.Errorf("%d et %d rendent le meme fictif %d", r, autre, f)
		}
		if originaux[f] {
			return nil, fmt.Errorf("%d : son fictif %d est aussi un xuid d origine", r, f)
		}
		fictifs[f] = r
		out = append(out, paireXuid{reel: r, fictif: f})
	}
	return out, nil
}
