package filmdec

// DÉCOUPAGE BINAIRE DU COMPOSANT i0 (object-position-dynamic-precision) — LU DANS LE FILM.
//
// Les largeurs d'axe NE SONT PAS une constante du décodeur : ce sont des CONSTANTES PAR
// CARTE. Le moteur (FUN_140be9a14, appelé en fin de chargement de carte) parcourt le bloc
// structure-BSP du tag scenario et précalcule, pour chaque région de compression r et chaque
// niveau de précision L :
//
//	W[r][L][axe] = min(26, ceilLog2(ceil(extent_axe / (2*q(L)))))   avec q(L) = 2^(16-L)/120
//
// La position d'objet utilise L = 16 câblé au site d'appel (MOV R9D,0x10 en 1406d008a), donc
// 2*q = 1/60 et W = min(26, ceilLog2(ceil(60*extent))). Ni les largeurs ni les bornes ne
// transitent par le bitstream, et le REGISTRE chunk_00 n'y contribue pas : il est
// BIT-À-BIT IDENTIQUE d'un film à l'autre (vérifié Cliffhanger vs Catalyst, FNV des 1067
// slots noms+flags = a413610cd08e4355 des deux côtés). Le flags=0 du composant i0 de
// l'archétype biped n'est donc PAS un niveau de précision exploitable.
//
// CE QUI EST DISPONIBLE HORS LIGNE : la LONGUEUR des champs se lit directement dans le
// bitstream. Pour un champ quantifié de largeur W dont la valeur bouge peu d'une frame à la
// suivante, le TAUX DE BASCULE par position de bit vaut ~50 % sur le LSB et DOUBLE du MSB
// vers le LSB. Le profil sur trois champs contigus est une DENT DE SCIE : montée
// géométrique puis effondrement au MSB du champ suivant. DetectI0Layout lit les trois
// frontières sur ce profil, sans aucun a priori de largeur.
//
// Preuve de chaînage (le critère qui départage les lectures rivales — une statistique de pas
// médian ne le fait PAS) : après le vec3, la suite du record est une structure CONNUE et
// indépendante — 2 bits de queue i0 (handleSel + regionPresent, toujours nuls) puis le
// composant i1 object-translational-velocity = [1 outer][1 present][19 dir][10 scale], dont
// les largeurs 19/10 viennent de l'asm du site d'appel (FUN_14076d4d0). Sur Cliffhanger
// (fin i0 = 45) et Catalyst (fin i0 = 50) les profils de bascule de cette structure se
// SUPERPOSENT point par point une fois décalés de 5 bits, effondrement compris à l'offset
// +23 = fin du champ direction de 19 bits. Un i0 trop court ou trop long désaligne tout.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/filmsource"
)

// Découpage de l'en-tête d'i0, seule partie NON dérivée du film (source : Ghidra).
//
//	i0SpineBits      : les 3 bits de contrôle lus par FUN_1406cfe44 avant l'aiguillage.
//	i0UseDefaultBits : le bit useDefault de FUN_14076e524 (1 => table par défaut, boîte monde).
//	i0RegionIndexBits: DAT_144632be0 = ceilLog2(nb de BSP valides), 1 quand la carte en a 2.
//
// La valeur 1 de l'index est CONFIRMÉE DANS LE FILM sur les deux cartes de référence : le bit
// 4 d'i0 ne bascule JAMAIS (0 bascule sur 170 518 / 291 288 paires consécutives) mais prend la
// valeur 1 sur une poignée d'enregistrements (3 sur Cliffhanger, 2 sur Catalyst) — signature
// d'un champ d'index d'un bit, et non d'un bit de données.
const (
	i0SpineBits       = 3
	i0UseDefaultBits  = 1
	i0RegionIndexBits = 1
)

// I0Layout est le découpage binaire du vec3 absolu d'i0 pour UNE carte.
type I0Layout struct {
	// GateBits est le nombre de bits qui précèdent l'axe X (spine + useDefault + index).
	GateBits int
	// AxisW sont les largeurs de quantification de X, Y, Z, en bits.
	AxisW [3]uint
	// Region est la VALEUR d'index de région attendue sur les records décodés (l'index
	// occupe GateBits-4 bits). Zéro partout sauf sur les cartes dont la région jouée n'est
	// pas la première du bloc structure-BSP (Live Fire : région 1 sur 2 bits — lot C
	// catalogues, 2026-08-27). Un record d'une AUTRE région est écarté : ses quanta sont
	// exprimés dans une autre AABB, les déquantifier avec ces bornes produirait une
	// coordonnée fausse silencieuse.
	Region uint32
}

// DefaultI0GateBits est la longueur d'en-tête d'i0 sur le chemin dominant (région explicite).
const DefaultI0GateBits = i0SpineBits + i0UseDefaultBits + i0RegionIndexBits

// TotalBits est la longueur du composant i0 jusqu'à la fin du vec3 (queue exclue).
func (l I0Layout) TotalBits() int {
	return l.GateBits + int(l.AxisW[0]+l.AxisW[1]+l.AxisW[2])
}

// AxisOffset est l'offset bit de l'axe ax depuis le début d'i0.
func (l I0Layout) AxisOffset(ax int) int {
	off := l.GateBits
	for i := 0; i < ax; i++ {
		off += int(l.AxisW[i])
	}
	return off
}

// Valid signale un découpage exploitable (largeurs dans la plage physique du moteur).
func (l I0Layout) Valid() bool {
	if l.GateBits < i0SpineBits+i0UseDefaultBits {
		return false
	}
	for _, w := range l.AxisW {
		if w < 8 || w > 26 { // cap moteur = 26 ; sous 8 bits aucune carte ne descend
			return false
		}
	}
	return true
}

func (l I0Layout) String() string {
	if l.Region != 0 {
		return fmt.Sprintf("gate=%d region=%d %d/%d/%d (i0=%d bits)",
			l.GateBits, l.Region, l.AxisW[0], l.AxisW[1], l.AxisW[2], l.TotalBits())
	}
	return fmt.Sprintf("gate=%d %d/%d/%d (i0=%d bits)", l.GateBits, l.AxisW[0], l.AxisW[1], l.AxisW[2], l.TotalBits())
}

// I0LayoutReport porte les mesures qui ONT servi à établir le découpage : c'est la pièce
// justificative, pas un log. FlipRate[k] est la fraction de paires d'enregistrements
// consécutifs du même slot dont le bit k d'i0 diffère.
type I0LayoutReport struct {
	Records    int
	Pairs      int
	FlipRate   []float64
	Boundaries []int
	// IndexBitOnes est le nombre d'enregistrements dont le bit d'index de région vaut 1.
	IndexBitOnes int
}

// detectWindow est la fenêtre de bits profilée depuis le début d'i0. Elle doit couvrir
// gate + 3 axes au maximum moteur observé (18/18/17 sur Illusion/Chasm = 58 bits) avec de la
// marge pour voir le retour à zéro de la queue.
const detectWindow = 72

// detectMaxChunks borne le coût de la détection : 6 chunks (~2 min de film) donnent déjà
// plusieurs dizaines de milliers de paires, largement de quoi séparer 0 % de 50 %.
const detectMaxChunks = 6

// i0Sample est un enregistrement candidat réduit à sa fenêtre de bits.
type i0Sample struct {
	slot     uint32
	chunk    int
	pkt      int
	bits     [2]uint64 // fenêtre MSB-first : bit k = bits[k/64] >> (63 - k%64)
	indexBit uint8
}

func (s i0Sample) bit(k int) uint64 { return (s.bits[k>>6] >> (63 - uint(k&63))) & 1 }

// DetectI0Layout lit le découpage d'i0 DANS le film de dir. Retourne le découpage, le
// rapport de mesure, et une erreur si le profil ne fait pas apparaître trois frontières
// nettes (film trop court, ou grammaire de record différente).
// DetectI0Layout est l'ENVELOPPE D2, HORS PRODUCTION : elle charge le film puis appelle
// [DetectI0LayoutOf]. La cuisson passe un film deja charge.
func DetectI0Layout(dir string) (I0Layout, I0LayoutReport, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return I0Layout{}, I0LayoutReport{}, err
	}
	return DetectI0LayoutOf(film)
}

// DetectI0LayoutOf lit le découpage d'i0 DANS un film DEJA CHARGE. Cf. [DetectI0Layout].
func DetectI0LayoutOf(film *filmsource.Film) (I0Layout, I0LayoutReport, error) {
	nums := FilmChunkNumbers(film)
	if len(nums) == 0 {
		return I0Layout{}, I0LayoutReport{}, ErrNoFilmChunk
	}
	scanned := nums
	if len(scanned) > detectMaxChunks {
		scanned = scanned[:detectMaxChunks]
	}
	slots := bipedSlotBand(film, scanned)
	if len(slots) == 0 {
		return I0Layout{}, I0LayoutReport{}, fmt.Errorf("aucun slot biped (ti=%d) dans le film", BipedTypeIndex)
	}
	var samples []i0Sample
	for _, c := range scanned {
		data, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			samples = collectI0Samples(pk.Payload(data), slots, c, pk.Index, samples)
		}
	}
	rep := profileI0(samples)
	rep.Boundaries = i0Boundaries(rep.FlipRate)
	if len(rep.Boundaries) < 3 {
		return I0Layout{}, rep, fmt.Errorf("profil i0 non concluant : %d frontière(s) détectée(s) sur %d paires",
			len(rep.Boundaries), rep.Pairs)
	}
	b := rep.Boundaries
	lay := I0Layout{
		GateBits: DefaultI0GateBits,
		AxisW:    [3]uint{uint(b[0] - DefaultI0GateBits), uint(b[1] - b[0]), uint(b[2] - b[1])},
	}
	if !lay.Valid() {
		return lay, rep, fmt.Errorf("découpage i0 implausible : %s", lay)
	}
	return lay, rep, nil
}

// collectI0Samples balaie un payload de paquet delta et retient la fenêtre de bits d'i0 de
// chaque en-tête biped reconnu. L'en-tête (préfixe/slot/tag/masque) est la seule hypothèse ;
// à partir d'i0 rien n'est supposé, hormis les 4 bits spine+useDefault nuls qui identifient
// le chemin absolu à région explicite.
func collectI0Samples(pay []byte, slots map[uint32]bool, chunk, pkt int, out []i0Sample) []i0Sample {
	total := len(pay) * 8
	const preGate = i0SpineBits + i0UseDefaultBits
	for p := 0; p+bipedHeaderBits+bipedIndexBits*bipedMinMaskCnt+detectWindow <= total; {
		i0, slot, _, ok := matchBipedHeaderRaw(pay, p, total, slots, true, detectWindow)
		if !ok || readBitsAt(pay, i0, preGate) != 0 {
			p++
			continue
		}
		var s i0Sample
		s.slot, s.chunk, s.pkt = slot, chunk, pkt
		s.indexBit = uint8(readBitsAt(pay, i0+preGate, 1))
		for k := 0; k < detectWindow; k++ {
			if readBitsAt(pay, i0+k, 1) == 1 {
				s.bits[k>>6] |= 1 << (63 - uint(k&63))
			}
		}
		out = append(out, s)
		p = i0 + detectWindow/2 // pas de re-scan chevauchant
	}
	return out
}

// profileI0 calcule le taux de bascule par position de bit entre enregistrements
// TEMPORELLEMENT CONSÉCUTIFS DU MÊME SLOT (paquets voisins du même chunk) : c'est la
// continuité du mouvement d'un joueur qui fait apparaître la dent de scie.
func profileI0(samples []i0Sample) I0LayoutReport {
	rep := I0LayoutReport{Records: len(samples), FlipRate: make([]float64, detectWindow)}
	flips := make([]int, detectWindow)
	bySlot := map[uint32][]i0Sample{}
	for _, s := range samples {
		bySlot[s.slot] = append(bySlot[s.slot], s)
		if s.indexBit == 1 {
			rep.IndexBitOnes++
		}
	}
	for _, ss := range bySlot {
		sort.Slice(ss, func(i, j int) bool {
			if ss[i].chunk != ss[j].chunk {
				return ss[i].chunk < ss[j].chunk
			}
			return ss[i].pkt < ss[j].pkt
		})
		for i := 1; i < len(ss); i++ {
			a, b := ss[i-1], ss[i]
			if a.chunk != b.chunk || b.pkt <= a.pkt || b.pkt-a.pkt > detectMaxPktGap {
				continue
			}
			rep.Pairs++
			for k := 0; k < detectWindow; k++ {
				if a.bit(k) != b.bit(k) {
					flips[k]++
				}
			}
		}
	}
	if rep.Pairs > 0 {
		for k := range flips {
			rep.FlipRate[k] = float64(flips[k]) / float64(rep.Pairs)
		}
	}
	return rep
}

// detectMaxPktGap borne l'écart de paquets d'une paire : au-delà le joueur a eu le temps de
// bouger de plusieurs quanta de poids fort et la dent de scie s'aplatit.
const detectMaxPktGap = 3

// i0BoundarySaturation / i0BoundaryDrop : une position k est le MSB d'un champ si le bit
// précédent était SATURÉ (taux de bascule >= 10 %, donc bit de poids faible d'un champ) et
// que le taux s'effondre d'au moins un facteur 8. Sur les 12 cartes mesurées l'écart réel est
// d'au moins trois ordres de grandeur (ex. Catalyst : 48,94 % au bit 19 puis 0,00 % au bit 20).
const (
	i0BoundarySaturation = 0.10
	i0BoundaryDrop       = 8.0
)

// i0Boundaries lit les MSB de champ sur le profil de bascule.
func i0Boundaries(flip []float64) []int {
	var out []int
	for k := 1; k < len(flip); k++ {
		if flip[k-1] >= i0BoundarySaturation && flip[k]*i0BoundaryDrop < flip[k-1] {
			out = append(out, k)
		}
	}
	return out
}
