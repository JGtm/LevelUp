package filmdec

// equipment_creation_offset_test.go — OÙ SE TROUVE le record de création de ti=37, mesuré
// contre un oracle au lieu d'être supposé.
//
// POURQUOI CET INSTRUMENT EXISTE. Le balayage par l'en-tête ([0][01][slot][gen][ti=37]) suivi
// du default-state porté rend 274 records sur `000d5950` et ZÉRO sur `00ba2e1c` — le même code,
// deux films. Une explication au jugé (« l'identifiant de record est plus large sur les gros
// films ») n'en est pas une : il faut mesurer.
//
// LA MÉTHODE, ORACLE D'ABORD. On ne cherche PAS l'en-tête : on cherche la POSITION. Les vies
// d'objet ti=37 sont déjà décodées par les paquets delta (ScanFilmWorldObjects), et le premier
// point de chaque vie est un triplet de quanta connu. On balaye alors chaque position de bit
// à la recherche d'un masque de présence valide suivi d'une position i0 EXACTEMENT égale à ce
// premier point. Cette coïncidence-là ne s'obtient pas par hasard (quelques milliers de cibles
// dans un espace de 2^45), et elle localise le corps du record sans rien supposer de son
// en-tête. On regarde ENSUITE ce qui précède, et à quelle distance.
//
// CE QUE L'HISTOGRAMME TRANCHE. Un pic de distance en-tête -> masque à la largeur prédite du
// default-state (60 bits sur le chemin minimal) dit que le déserialiseur est juste. Un pic
// ailleurs dit que la grammaire n'est pas celle du flux. Aucun en-tête trouvé dit que le record
// de création n'a pas la forme supposée sur ce film.
//
// LECTURE SEULE, gardé par EQUIP_CREATION_FILM. UN SEUL décodage filmdec par process.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 EQUIP_CREATION_FILM=<repo>/data/cache/film_chunks/06dfe6d9 \
//	  go test ./internal/analysis/filmdec/ -run '^TestEquipmentCreationOffset$' -timeout 60m -v

import (
	"os"
	"testing"
)

// equipCreationMaxBack borne la recherche de l'en-tête EN AMONT du corps trouvé. 512 bits
// couvre huit fois la largeur prédite du default-state : au-delà, on ne chercherait plus
// l'en-tête de CE record mais celui d'un autre.
const equipCreationMaxBack = 512

func TestEquipmentCreationOffset(t *testing.T) {
	dir := os.Getenv(equipCreationFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", equipCreationFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	prev := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prev })
	SetWorldObjectPrecisionFromLayout(lay)

	n := CountFilmChunks(dir)
	band := worldObjectSlotBand(dir, n, EquipmentTypeIndex)
	tracks, err := ScanFilmWorldObjects(dir, &equipCreationUnitRange, EquipmentTypeIndex)
	if err != nil {
		t.Fatalf("trajectoires ti=%d illisibles : %v", EquipmentTypeIndex, err)
	}
	arch, err := EquipmentArchetype(dir)
	if err != nil {
		t.Fatalf("archétype ti=%d illisible : %v", EquipmentTypeIndex, err)
	}
	pr := equipOffsetProbe{
		comps: len(arch.Components), band: band, want37: EquipmentTypeIndex,
		want: map[[3]int32][]equipCreationWanted{},
		gap:  map[uint32]int{}, ti: map[uint32]int{}, hdr: map[uint32]int{},
		bits: map[uint32]*[equipProfileBits]int{},
	}
	for _, tr := range tracks {
		p := [3]float32{tr.Pts[0].X, tr.Pts[0].Y, tr.Pts[0].Z}
		c := equipCell(p)
		pr.want[c] = append(pr.want[c],
			equipCreationWanted{pos: p, life: equipCreationLifeKey{tr.Slot, tr.Gen}})
	}
	t.Logf("FILM %s · largeurs %v · bande %d slots · %d vies delta · %d premiers points distincts",
		dir, lay.AxisW, len(band), len(tracks), len(pr.want))

	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type == PacketTypeDelta {
				pr.scan(pk.Payload(data))
			}
		}
	}
	t.Logf("== CORPS TROUVÉS PAR L'ORACLE %d · dont précédés d'un en-tête NEW ti=%d %d ==",
		pr.bodies, EquipmentTypeIndex, pr.withHeader)
	t.Logf("   distance en-tête -> masque, en bits (prédiction du déser porté : 60 + 24 = 84) :%s",
		equipCreationLine(pr.gap, 24))
	t.Logf("   en-têtes NEW quelconques en amont, par typeIndex lu :%s", equipCreationLine(pr.ti, 24))
	equipLogBitProfile(t, pr)
}

// equipLogBitProfile publie, pour la distance DOMINANTE, la part de records dont chaque bit du
// record vaut 1. Un bit constant est une CONSTANTE du format ; c'est en comparant ces profils
// d'un film à l'autre qu'on localise les trois bits d'écart, au lieu de les deviner.
func equipLogBitProfile(t *testing.T, pr equipOffsetProbe) {
	t.Helper()
	top := equipCreationTop(pr.gap, 1)
	if len(top) == 0 {
		return
	}
	g := top[0].key
	prof := pr.bits[g]
	if prof == nil {
		return
	}
	n := top[0].count
	line := ""
	for i := 0; i < equipProfileBits; i++ {
		switch {
		case prof[i] == 0:
			line += "0"
		case prof[i] == n:
			line += "1"
		default:
			line += "."
		}
		if i%8 == 7 {
			line += " "
		}
	}
	t.Logf("   PROFIL DE BITS à la distance %d (%d records) — 0/1 = bit constant, . = variable :",
		g, n)
	t.Logf("     %s", line)
}

// equipOffsetProbe localise les corps de record par la position, puis cherche l'en-tête.
type equipOffsetProbe struct {
	comps int
	band  map[uint32]bool
	// want37 est l'ARCHÉTYPE cherché en amont du corps. Il est un CHAMP et non la constante
	// `EquipmentTypeIndex` parce que le même instrument sert de contrôle POSITIF à la mesure de
	// `ti=42` (ground_weapon_creation_offset_test.go) : le déserialiseur de `ti=37` est validé
	// par ailleurs, donc son pic de distance est la référence à laquelle celui de `ti=42` se
	// compare — et une comparaison n'a de valeur que si les deux passent par le MÊME code.
	want37 uint32
	// want indexe le PREMIER point de chaque vie delta — l'oracle — par CELLULE d'une grille
	// au pas d'equipCreationPosEps : la position de création n'est pas identique au premier
	// point transmis en delta (mesuré : écart médian de deux quanta), une égalité stricte ne
	// retrouverait donc presque aucun record.
	want map[[3]int32][]equipCreationWanted
	// gap : distance (bits) entre un en-tête NEW ti=37 au bon slot et le masque du corps.
	gap map[uint32]int
	// ti : typeIndex lu sur les en-têtes NEW trouvés en amont, quel que soit leur slot.
	ti map[uint32]int
	// hdr : distance du corps au premier en-tête NEW quelconque trouvé en amont.
	hdr map[uint32]int
	// bits[g][i] compte les records de distance g dont le i-ème bit (depuis l'en-tête) vaut 1.
	bits               map[uint32]*[equipProfileBits]int
	bodies, withHeader int
	// onMatch, si non nil, reçoit CHAQUE couple (en-tête, masque) apparié — le payload, le bit
	// de l'en-tête et le bit du masque. C'est ce qui permet à un appelant de faire l'ESSAI
	// D'ATTERRISSAGE (dérouler un déserialiseur candidat depuis l'en-tête et vérifier qu'il
	// tombe AU BIT PRÈS sur le masque) sans redécoder le film une seconde fois. Nil ici : la
	// mesure de `ti=37` n'en a pas besoin, son pic est déjà la preuve.
	onMatch func(pay []byte, hdr, mask int)
}

// equipProfileBits borne le profil de bits : 128 couvre l'en-tête (24), le default-state du
// chemin court (60) et la porte + le masque qui suivent.
const equipProfileBits = 128

func (pr *equipOffsetProbe) scan(pay []byte) {
	total := len(pay) * 8
	for b := 0; b+16 <= total; b++ {
		life, ok := pr.body(pay, b, total)
		if !ok {
			continue
		}
		pr.bodies++
		pr.back(pay, b, life)
	}
}

// body dit si un masque valide ouvrant sur i0 commence au bit b et si la position qui le suit
// tombe à moins d'equipCreationPosEps du premier point d'une vie delta connue.
func (pr *equipOffsetProbe) body(pay []byte, b, total int) (equipCreationLifeKey, bool) {
	br := NewBitReader(pay)
	br.SetBitPos(b)
	idx, _, valid := readMaskIndices(br, pr.comps)
	if !valid || br.BitPos() > total || idx[0] != 0 {
		return equipCreationLifeKey{}, false
	}
	v, valid := decodeWorldObjectPos(pay, br.BitPos(), &equipCreationUnitRange)
	if !valid {
		return equipCreationLifeKey{}, false
	}
	return pr.nearest(v)
}

// equipCreationWanted est un premier point de vie, avec la vie qu'il désigne.
type equipCreationWanted struct {
	pos  [3]float32
	life equipCreationLifeKey
}

// equipCreationPosEps est le rayon de recherche, en coordonnées normalisées de l'AABB.
// 0,002 vaut environ deux quanta sur l'axe le plus fin — assez pour absorber le déplacement
// entre la création et le premier delta, trop peu pour qu'un tirage au hasard tombe dedans
// (quelques centaines de cibles pour un volume relatif de l'ordre de 1e-5).
const equipCreationPosEps = 0.002

func equipCell(p [3]float32) [3]int32 {
	return [3]int32{
		int32(p[0] / equipCreationPosEps),
		int32(p[1] / equipCreationPosEps),
		int32(p[2] / equipCreationPosEps),
	}
}

// nearest cherche la cible la plus proche dans les 27 cellules voisines.
func (pr *equipOffsetProbe) nearest(v [3]float32) (equipCreationLifeKey, bool) {
	c := equipCell(v)
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			for dz := int32(-1); dz <= 1; dz++ {
				for _, w := range pr.want[[3]int32{c[0] + dx, c[1] + dy, c[2] + dz}] {
					if equipNear(v, w.pos) {
						return w.life, true
					}
				}
			}
		}
	}
	return equipCreationLifeKey{}, false
}

func equipNear(a, b [3]float32) bool {
	return absF(a[0]-b[0]) < equipCreationPosEps &&
		absF(a[1]-b[1]) < equipCreationPosEps &&
		absF(a[2]-b[2]) < equipCreationPosEps
}

func absF(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

// back cherche, en amont du corps, un en-tête de record NEW — d'abord celui qui porte ti=37 ET
// le slot de la vie (la preuve), puis n'importe lequel (le diagnostic).
func (pr *equipOffsetProbe) back(pay []byte, b int, life equipCreationLifeKey) {
	firstAny := -1
	for d := 1; d <= equipCreationMaxBack && b-d >= 0; d++ {
		p := b - d
		if PeekBits(pay, p, 1) != 0 || PeekBits(pay, p+1, 2) != 1 {
			continue
		}
		ti := uint32(PeekBits(pay, p+woNewTypeBits+woNewSlotBits+woNewGenBits, woNewTIBits))
		slot := uint32(PeekBits(pay, p+woNewTypeBits, woNewSlotBits))
		gen := uint32(PeekBits(pay, p+woNewTypeBits+woNewSlotBits, woNewGenBits))
		if firstAny < 0 {
			firstAny = d
			pr.ti[ti]++
		}
		if ti == pr.want37 && slot == life.slot && gen == life.gen {
			pr.withHeader++
			pr.gap[uint32(d)]++
			pr.profile(pay, p, uint32(d))
			if pr.onMatch != nil {
				pr.onMatch(pay, p, b)
			}
			break
		}
	}
	if firstAny >= 0 {
		pr.hdr[uint32(firstAny)]++
	}
}

// profile accumule le profil de bits du record commençant au bit p, pour la distance d.
func (pr *equipOffsetProbe) profile(pay []byte, p int, d uint32) {
	if d >= equipProfileBits {
		return
	}
	prof := pr.bits[d]
	if prof == nil {
		prof = &[equipProfileBits]int{}
		pr.bits[d] = prof
	}
	for i := 0; i < equipProfileBits; i++ {
		if PeekBits(pay, p+i, 1) == 1 {
			prof[i]++
		}
	}
}
