package replay

// objectifs_phase0_motif_test.go — CE QU'EST LE MOTIF (item 0.3) ET COMBIEN DE MAINS LE
// TIENNENT A LA FOIS.
//
// DEUX QUESTIONS, DEUX MESURES SANS REGLAGE :
//
//	OU VIT-IL ? La position du motif dans le record, et les 32 bits qui l'encadrent. Un
//	identifiant d'arme vit vers +1950 bits du debut du record et son identifiant complet
//	porte le suffixe 0x42C9679F (cf. `weaponv3.CanonWeaponID` et `keyframe_loadout.go`).
//	Si le motif porte ce suffixe, c'est un `weap` et il a un nom de tag ; sinon, non — et
//	c'est ce constat-la qu'il faut ecrire, pas un nom suppose.
//
//	COMBIEN DE MAINS ? Le nombre de bipedes qui portent le motif au MEME instant. En CTF il
//	y a deux drapeaux, donc 0, 1 ou 2 ; en Oddball il y a UN crane, donc 0 ou 1. Cette
//	contrainte-la n'est imposee par rien dans la methode : elle se verifie ou elle se refute.

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// objContexteMotif porte ce que le motif a autour de lui.
type objContexte struct {
	Occurrences int
	// OffsetMedian est la position mediane du premier bit du motif, comptee depuis le debut
	// du record porteur.
	OffsetMedian int
	// Avant / Apres : les mots de 32 bits les plus frequents immediatement avant et apres.
	Avant, Apres []objMotFrequent
	// SuffixeArme compte les occurrences suivies du suffixe d'identifiant d'arme.
	SuffixeArme int
}

// objMotFrequent est un mot de 32 bits et son compte.
type objMotFrequent struct {
	Mot    uint32
	Compte int
}

// objSuffixeArme — le low-32 partage par les identifiants d'arme reels
// (`weaponv3.commonWeaponSuffix`, non exporte : la valeur est recopiee ici avec sa source).
const objSuffixeArme uint32 = 0x42c9679f

// objMotifContexte re-balaye les images-cles et rend le contexte binaire du motif.
func objMotifContexte(dir string, val uint32) (objContexte, error) {
	n := filmdec.CountFilmChunks(dir)
	var offsets []int
	avant, apres := map[uint32]int{}, map[uint32]int{}
	ctx := objContexte{}
	lus := 0
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		lus++
		for _, p := range filmdec.WalkPackets(data) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			objContexteDePayload(p.Payload(data), val, &ctx, &offsets, avant, apres)
		}
	}
	if lus == 0 {
		return ctx, errNoChunk
	}
	ctx.Occurrences = len(offsets)
	ctx.OffsetMedian = objMediane(offsets)
	ctx.Avant, ctx.Apres = objTopMots(avant, 3), objTopMots(apres, 3)
	return ctx, nil
}

// errNoChunk : aucun chunk lisible.
var errNoChunk = errChunk("aucun chunk lisible")

// errChunk est une erreur de lecture de film.
type errChunk string

func (e errChunk) Error() string { return string(e) }

// objContexteDePayload accumule le contexte du motif pour un payload d'image-cle.
func objContexteDePayload(pay []byte, val uint32, ctx *objContexte, offsets *[]int,
	avant, apres map[uint32]int) {
	recs := filmdec.WalkKeyframeWorld(pay)
	if len(recs) == 0 {
		return
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	total := len(pay) * 8
	for i, r := range recs {
		if r.TI != objBipedTI || r.Slot < 0 {
			continue
		}
		fin := total
		if i+1 < len(recs) {
			fin = recs[i+1].Bit
		}
		for _, at := range objPositionsDansRecord(pay, r.Bit, fin, val) {
			p := r.Bit + at
			*offsets = append(*offsets, at)
			if p >= 32 {
				avant[objMot32(pay, p-32)]++
			}
			if p+64 <= total {
				suiv := objMot32(pay, p+32)
				apres[suiv]++
				if suiv == objSuffixeArme {
					ctx.SuffixeArme++
				}
			}
		}
	}
}

// objMot32 lit 32 bits a la position bit p.
func objMot32(pay []byte, p int) uint32 {
	var w uint32
	for i := 0; i < 32; i++ {
		b := p + i
		if b>>3 >= len(pay) {
			return w << uint(32-i)
		}
		w = w<<1 | uint32(pay[b>>3]>>(7-uint(b&7))&1)
	}
	return w
}

// objTopMots rend les n mots les plus frequents.
func objTopMots(m map[uint32]int, n int) []objMotFrequent {
	out := make([]objMotFrequent, 0, len(m))
	for w, c := range m {
		out = append(out, objMotFrequent{w, c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Compte != out[j].Compte {
			return out[i].Compte > out[j].Compte
		}
		return out[i].Mot < out[j].Mot
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// objSimultaneite rend, par instant d'image-cle, le nombre de bipedes DISTINCTS qui portent
// le motif — et la distribution de ce nombre.
func objSimultaneite(recs []objRecord, val uint32) (map[int]int, int) {
	parInstant := map[uint64]map[uint32]bool{}
	instants := map[uint64]bool{}
	for _, r := range recs {
		instants[r.TS] = true
		if !objPorte(r, val) {
			continue
		}
		if parInstant[r.TS] == nil {
			parInstant[r.TS] = map[uint32]bool{}
		}
		parInstant[r.TS][r.Slot] = true
	}
	distrib := map[int]int{}
	for ts := range instants {
		distrib[len(parInstant[ts])]++
	}
	return distrib, len(instants)
}

// objDistribString met en forme une distribution « n porteurs -> k images ».
func objDistribString(d map[int]int) string {
	cles := make([]int, 0, len(d))
	for k := range d {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	s := ""
	for _, k := range cles {
		if s != "" {
			s += ", "
		}
		s += itoa(k) + " porteur(s) -> " + itoa(d[k]) + " images"
	}
	return s
}

// itoa evite un import de strconv pour un seul usage de mise en forme.
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
