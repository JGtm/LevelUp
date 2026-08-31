package filmdec

// equipment_creation.go — LE RECORD DE CRÉATION d'un objet d'équipement (ti=37).
//
// CE QUE CE BALAYAGE FAIT SORTIR, ET POURQUOI IL N'EXISTAIT PAS. Les quatre composants delta
// de ti=37 (`deployed`, `activated`, `creator`, `energy` — cf. equipment_state.go) ne portent
// AUCUNE identité : mesuré le 2026-08-15, aucun d'eux ne distingue un mur d'un capteur ni d'un
// bonus au sol. Mais le DEFAULT-STATE de l'archétype, lui, en porte deux — et son déserialiseur
// les consommait pour rester aligné, exactement le défaut d'i48 (2026-08-14) et des quatre
// champs delta (2026-08-16) :
//
//	FUN_1407f105c = V ; deser ti36 ; ECS_ReadEntityRefIndex5 [R(1) ; si 0 -> R(5)] ;
//	                R(1) porte ; si 1 -> R(32) « ability-enabled-id »
//
// LA RÈGLE QUI GOUVERNE, la même qu'ailleurs : c'est le DÉSERIALISEUR qui publie
// (consumeDefaultStateTI37, default_state_arch.go), jamais un second lecteur posé à côté de lui.
//
// OÙ CE RECORD SE TROUVE. Un record de CRÉATION (type NEW) vit dans les paquets DELTA, à côté
// des records delta ordinaires, et son en-tête diffère du leur :
//
//	DELTA   [1] [slot:13] [gen:2] [baseline:1] [masque…]          (cf. matchWorldObjectRecord)
//	NEW     [0] [type:2 = 1] [slot:13] [gen:2] [ti:6] [default-state] [porte:1] [masque…]
//
// C'est le R(6) `typeIndex` qui rend ce balayage sélectif là où celui des deltas ne l'est pas :
// un record NEW DIT son archétype, on ne le déduit pas d'une bande de slots.
//
// CE QU'IL NE DIT PAS. Ni ce que valent les deux champs, ni s'ils nomment quoi que ce soit :
// ce fichier les fait SORTIR, l'instrument de mesure (equipment_creation_test.go) les juge.
// VERDICT DE LA MESURE, à lire avant de bâtir sur ces deux champs : leur porte est FERMÉE sur
// 503 records de création sur 503 (2026-08-17). L'identité de l'objet est dans le bloc
// `object-multiplayer-properties` du MÊME record — son mot de 32 bits inconditionnel est le
// GlobalID du tag `eqip` de l'objet (11 valeurs observées sur 11 résolues dans les 105 tags
// `eqip` du jeu). C'est ce champ-là que le lecteur veut, et il voyage dans MPPVal[MPPWord32].
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.

import "fmt"

// EquipmentCreationField désigne l'un des deux champs que le default-state de ti=37 lisait et
// jetait. L'ordre est celui du flux.
type EquipmentCreationField int

// Les deux champs, et EquipmentCreationFieldCount qui les compte.
const (
	// EquipCreationRef est la référence d'entité sur 5 bits (ECS_ReadEntityRefIndex5,
	// FUN_1407f2058). Le nom du déserialiseur dit « référence d'entité » ; il ne dit PAS que
	// c'est un créateur, et rien ici ne le suppose.
	EquipCreationRef EquipmentCreationField = iota
	// EquipCreationAbilityID est l'identifiant 32 bits nommé « ability-enabled-id » par le
	// registre de champs de FUN_14080dec4.
	EquipCreationAbilityID
	EquipmentCreationFieldCount = 2
)

// String rend l'étiquette du champ — celle du déserialiseur, la seule façon honnête de le nommer.
func (f EquipmentCreationField) String() string {
	switch f {
	case EquipCreationRef:
		return "entity-ref-index5"
	case EquipCreationAbilityID:
		return "ability-enabled-id"
	}
	return fmt.Sprintf("champ inconnu (%d)", int(f))
}

// equipmentCreationHook, si non nil, reçoit CHAQUE lecture des deux champs du default-state de
// ti=37 : le champ, la valeur, et `present` — faux quand la porte s'est fermée sans transmettre
// de valeur. Une porte fermée n'est PAS une valeur nulle.
var equipmentCreationHook func(f EquipmentCreationField, value uint64, present bool)

// SetEquipmentCreationHook installe (ou retire, avec nil) la sonde du default-state de ti=37.
func SetEquipmentCreationHook(h func(f EquipmentCreationField, value uint64, present bool)) {
	equipmentCreationHook = h
}

func publishEquipmentCreation(f EquipmentCreationField, value uint64, present bool) {
	if equipmentCreationHook != nil {
		equipmentCreationHook(f, value, present)
	}
}

// Découpage de l'en-tête d'un record de CRÉATION (type NEW) d'objet du monde dans un paquet
// delta. Chaque largeur vient d'un lecteur porté et vérifié : readRecordType (R(1) ; si 0 ->
// R(2)), readRecordID (R(IDLowBits=13) puis R(2) de génération, cf. DefaultFrameConfig) et
// TraverseEntity (R(6) typeIndex).
const (
	woNewTypeBits   = 3  // R(1)=0 « pas un delta » + R(2)=1 « recNew »
	woNewSlotBits   = 13 // FrameConfig.IDLowBits
	woNewGenBits    = 2
	woNewTIBits     = 6
	woNewHeaderBits = woNewTypeBits + woNewSlotBits + woNewGenBits + woNewTIBits
)

// EquipmentCreation est UN record de création d'objet d'équipement, lu jusqu'à sa position.
type EquipmentCreation struct {
	// Slot et Gen identifient la vie de l'objet — LA PAIRE, comme partout ailleurs : le pool de
	// slots reboucle et la génération ne fait que 2 bits.
	Slot, Gen uint32
	// Chunk / PacketIndex / TimestampUS localisent la lecture (même horloge que BipedPosition).
	Chunk, PacketIndex int
	TimestampUS        uint64
	// BitPos est la position de l'en-tête dans le payload, en bits (traçabilité).
	BitPos int
	// HasRef / Ref : la référence d'entité 5 bits, si la porte l'a transmise.
	HasRef bool
	Ref    uint32
	// HasID / AbilityID : l'identifiant 32 bits « ability-enabled-id », si transmis.
	HasID     bool
	AbilityID uint32
	// MPP porte les quatre champs du bloc `object-multiplayer-properties` du MÊME record
	// (cf. MPPField). Ils sont là parce que le default-state de ti=37 les contient : chercher
	// l'identité de l'objet ailleurs que dans son propre record de création n'aurait pas de sens.
	MPPPresent [MPPFieldCount]bool
	MPPVal     [MPPFieldCount]uint64
	// X, Y, Z est la position du composant i0 du MÊME record : le lieu de la pose.
	X, Y, Z float32
	// Mask est la liste des index de composant du masque, strictement croissante.
	Mask []int
	// MaskFull dit que le masque était la branche PLEINE (R(64)) et non la branche éparse.
	MaskFull bool
	// MaskHasI0 dit que le masque annonçait i0 explicitement. Faux = i0 a été décodé quand
	// même, parce que le masque par défaut {i0} est OR'd en amont par le moteur
	// (vtable[0xa0], cf. consumeBipedDefaultStateMovement) : la position est alors le PREMIER
	// composant décodé sans figurer au masque du flux.
	MaskHasI0 bool
	// DefaultStateBits est le nombre de bits qu'a consommés consumeDefaultStateTI37 sur CE
	// record. Publié parce qu'un déserialiseur mal porté se voit à une largeur qui s'éparpille.
	DefaultStateBits int
	// AfterBit est la position du premier bit après le composant i0 (traçabilité du balayage).
	AfterBit int
}

// EquipmentCreationStats compte ce que le balayage a rencontré. Sans ces dénominateurs, une
// distribution de valeurs ne se juge pas — et sans le détail des rejets, on ne sait pas si le
// balayage rate des records ou en invente.
type EquipmentCreationStats struct {
	// Slots est le nombre de slots de la bande passée au balayage.
	Slots int
	// Anchors est le nombre d'en-têtes NEW ti=37 reconnus (les quatre constantes + la bande).
	Anchors int
	// Overflow : le default-state déborde du payload — le curseur n'est plus digne de confiance.
	Overflow int
	// MaskBad : compte hors bornes, index non croissants, ou index >= nombre de composants.
	MaskBad int
	// PosBad : position rejetée (porte non nulle, ou quantum saturé — cf. decodeWorldObjectPos).
	PosBad int
	// Accepted est le nombre de records rendus.
	Accepted int
	// MaskSparse / MaskFull : branche du masque des records acceptés.
	MaskSparse, MaskFull int
	// NoI0 : records acceptés dont le masque du flux n'annonçait PAS i0 (position décodée par
	// le masque par défaut OR'd en amont). Comptés à part : ce sont les moins sûrs.
	NoI0 int
	// WithRef / WithID : records acceptés dont la porte a transmis la valeur du champ.
	WithRef, WithID int
}

// ScanFilmEquipmentCreations décode les records de création des objets d'équipement du film de
// dir, sur la bande de slots de ti=37 lue dans les images-clés.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe
// `equipmentCreationHook`, qui est un global de paquet. Le hook est restauré à la sortie.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
func ScanFilmEquipmentCreations(dir string, wr *Vec3Range) ([]EquipmentCreation, EquipmentCreationStats, error) {
	var st EquipmentCreationStats
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, st, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	band := worldObjectSlotBand(dir, n, EquipmentTypeIndex)
	if len(band) == 0 {
		return nil, st, fmt.Errorf("aucun slot d'archétype ti=%d dans les keyframes de %s",
			EquipmentTypeIndex, dir)
	}
	return ScanFilmEquipmentCreationsForBand(dir, wr, band)
}

// ScanFilmEquipmentCreationsForBand balaye une BANDE DE SLOTS donnée. La bande est un paramètre
// pour que le témoin de contrôle (une bande FANTÔME de même cardinalité, faite de slots jamais
// vus porter cet archétype) passe par le MÊME code que la mesure — sans quoi le contrôle ne
// contrôlerait pas le décodeur mais une variante de lui (règle établie par
// WorldObjectPositionsForBand).
func ScanFilmEquipmentCreationsForBand(
	dir string, wr *Vec3Range, band map[uint32]bool,
) ([]EquipmentCreation, EquipmentCreationStats, error) {
	var st EquipmentCreationStats
	if wr == nil {
		return nil, st, fmt.Errorf("bornes monde absentes : sans elles le décodeur ne rend que des quanta")
	}
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, st, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	st.Slots = len(band)
	arch, err := EquipmentArchetype(dir)
	if err != nil {
		return nil, st, err
	}

	var cur equipCreationRead
	defer installCreationHooks(&cur)()

	w := equipCreationWalk{comps: len(arch.Components), wr: wr, band: band, cur: &cur}
	var out []EquipmentCreation
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			out = append(out, w.scanPayload(pk.Payload(data), &st, pk, c)...)
		}
	}
	return out, st, nil
}

// installCreationHooks branche les DEUX sondes du record de création — le default-state de
// ti=37 et le bloc `object-multiplayer-properties` qu'il contient — sur un même tampon, et rend
// la fonction qui restaure les sondes précédentes.
//
// Les deux vont ENSEMBLE et se posent d'un seul geste : le balayage lit un record entier, et
// n'installer que l'une des deux rendrait un record à moitié observé sans que rien ne le dise.
func installCreationHooks(cur *equipCreationRead) func() {
	prev, prevMPP := equipmentCreationHook, mppHook
	SetEquipmentCreationHook(func(f EquipmentCreationField, v uint64, present bool) {
		cur.present[f], cur.val[f] = present, v
	})
	SetMultiplayerPropertiesHook(func(f MPPField, v uint64, present bool) {
		cur.mppPresent[f], cur.mppVal[f] = present, v
	})
	return func() {
		SetEquipmentCreationHook(prev)
		SetMultiplayerPropertiesHook(prevMPP)
	}
}

// equipCreationRead porte ce que les hooks ont publié pour UN record.
type equipCreationRead struct {
	present    [EquipmentCreationFieldCount]bool
	val        [EquipmentCreationFieldCount]uint64
	mppPresent [MPPFieldCount]bool
	mppVal     [MPPFieldCount]uint64
}

// equipCreationWalk porte ce que la marche d'un record doit connaître (règle des 5 paramètres).
//
// ELLE EST PARAMÉTRÉE PAR L'ARCHÉTYPE, et il le faut : la marche d'un record de création est la
// MÊME pour tous les objets du monde (en-tête NEW, default-state, porte, masque, i0) — seuls
// changent le `typeIndex` que l'en-tête doit porter et le déserialiseur du default-state. Les
// ARMES AU SOL (`ti=42`, ground_weapon_creation.go) empruntent donc ce code au lieu d'en
// recopier une seconde version qui re-divergerait au premier correctif.
type equipCreationWalk struct {
	comps int
	wr    *Vec3Range
	band  map[uint32]bool
	cur   *equipCreationRead
	// ti est le typeIndex exigé de l'en-tête NEW ; zéro vaut `EquipmentTypeIndex`.
	ti uint32
	// deser est le déserialiseur du default-state de cet archétype ; nil vaut celui de `ti=37`.
	deser func(*BitReader)
	// posDecode décode et VALIDE le composant i0 à l'offset donné (gate de sélectivité). nil vaut
	// le chemin OBJET DU MONDE (decodeWorldObjectPos, porte 3 bits). L'archétype VÉHICULE (`ti=40`)
	// porte un i0 en PRÉCISION-DYNAMIQUE (porte 5 bits, biped) : il passe ici decodeBipedI0Pos
	// (vehicle_creation.go). Sans ce paramètre le gate lit i0 avec la mauvaise grammaire et n'est
	// plus sélectif — mesuré : le témoin fantôme rendait alors PLUS de records que la vraie bande.
	posDecode func([]byte, int) ([3]float32, bool)
	// posBits est la largeur du composant i0, pour avancer le curseur après un record accepté.
	// Zéro vaut projPosBits() (chemin objet du monde) ; `ti=40` passe lay.TotalBits().
	posBits int
}

// archetype et defaultState rendent les réglages effectifs de la marche (défauts `ti=37`).
func (w equipCreationWalk) archetype() uint32 {
	if w.ti == 0 {
		return EquipmentTypeIndex
	}
	return w.ti
}

func (w equipCreationWalk) defaultState() func(*BitReader) {
	if w.deser == nil {
		return consumeDefaultStateTI37
	}
	return w.deser
}

// decodePos VALIDE et décode i0 à l'offset at : le décodeur paramétré (dyn.-préc. pour `ti=40`)
// s'il est fourni, sinon le chemin objet du monde (porte 3 bits).
func (w equipCreationWalk) decodePos(pay []byte, at int) ([3]float32, bool) {
	if w.posDecode != nil {
		return w.posDecode(pay, at)
	}
	return decodeWorldObjectPos(pay, at, w.wr)
}

// posAdvance rend la largeur d'i0 pour avancer le curseur après un record accepté.
func (w equipCreationWalk) posAdvance() int {
	if w.posBits > 0 {
		return w.posBits
	}
	return projPosBits()
}

// scanPayload balaye UN payload delta et rend les records de création reconnus.
func (w equipCreationWalk) scanPayload(
	pay []byte, st *EquipmentCreationStats, pk FilmPacket, chunk int,
) []EquipmentCreation {
	var out []EquipmentCreation
	total := len(pay) * 8
	limit := total - woNewHeaderBits
	for p := 0; p <= limit; p++ {
		slot, gen, ok := matchWorldObjectNewHeader(pay, p, w.band, w.archetype())
		if !ok {
			continue
		}
		st.Anchors++
		cre, ok := w.readCreation(pay, p, total, st)
		if !ok {
			continue
		}
		cre.Slot, cre.Gen, cre.BitPos = slot, gen, p
		cre.Chunk, cre.PacketIndex, cre.TimestampUS = chunk, pk.Index, pk.TimestampUS
		st.Accepted++
		st.MaskSparse, st.MaskFull = st.MaskSparse+b2i(!cre.MaskFull), st.MaskFull+b2i(cre.MaskFull)
		st.NoI0 += b2i(!cre.MaskHasI0)
		if cre.HasRef {
			st.WithRef++
		}
		if cre.HasID {
			st.WithID++
		}
		out = append(out, cre)
		p = cre.AfterBit - 1 // un record accepté n'est pas re-balayé
	}
	return out
}

// matchEquipmentNewHeader reconnaît un en-tête de record de CRÉATION d'objet d'ÉQUIPEMENT.
func matchEquipmentNewHeader(pay []byte, p int, band map[uint32]bool) (slot, gen uint32, ok bool) {
	return matchWorldObjectNewHeader(pay, p, band, EquipmentTypeIndex)
}

// matchWorldObjectNewHeader reconnaît un en-tête de record de CRÉATION d'objet du monde de
// l'archétype `ti` à la position de bit p. PUR (aucune I/O).
//
// Quatre contraintes, dont trois sont des CONSTANTES du format : le préfixe de type
// (`0` puis `01`), le typeIndex R(6) == ti, et l'appartenance du slot à la bande de l'archétype.
// La génération est libre : ses quatre valeurs sont légitimes (même règle que les deltas
// d'objet du monde, cf. matchWorldObjectRecord).
func matchWorldObjectNewHeader(
	pay []byte, p int, band map[uint32]bool, ti uint32,
) (slot, gen uint32, ok bool) {
	if PeekBits(pay, p, 1) != 0 { // un record DELTA ouvre sur 1
		return 0, 0, false
	}
	if PeekBits(pay, p+1, 2) != 1 { // type de record : 1 = NEW
		return 0, 0, false
	}
	if uint32(PeekBits(pay, p+woNewTypeBits+woNewSlotBits+woNewGenBits, woNewTIBits)) != ti {
		return 0, 0, false
	}
	slot = uint32(PeekBits(pay, p+woNewTypeBits, woNewSlotBits))
	if !band[slot] {
		return 0, 0, false
	}
	return slot, uint32(PeekBits(pay, p+woNewTypeBits+woNewSlotBits, woNewGenBits)), true
}

// readCreation déroule le corps d'un record de création : le default-state (qui publie les deux
// champs par le hook), la porte du record NEW, le masque de présence, puis le composant i0.
//
// CHAQUE ÉCHEC EST COMPTÉ À PART. Un en-tête reconnu dont le corps ne se déroule pas est très
// probablement un faux positif du balayage bit à bit ; le dire par un compteur, plutôt que par
// un rejet silencieux, est ce qui permet de juger la sélectivité de l'ancre.
func (w equipCreationWalk) readCreation(
	pay []byte, p, total int, st *EquipmentCreationStats,
) (EquipmentCreation, bool) {
	var cre EquipmentCreation
	*w.cur = equipCreationRead{}
	br := NewBitReader(pay)
	start := p + woNewHeaderBits
	br.SetBitPos(start)
	w.defaultState()(br)
	if br.BitPos() > total {
		st.Overflow++
		return cre, false
	}
	cre.DefaultStateBits = br.BitPos() - start
	br.ReadBit() // porte has-components du record NEW (R(1) avant le masque, cf. TraverseEntity)
	idx, full, ok := readMaskIndices(br, w.comps)
	if !ok || br.BitPos() > total {
		st.MaskBad++
		return cre, false
	}
	v, ok := w.decodePos(pay, br.BitPos())
	if !ok {
		st.PosBad++
		return cre, false
	}
	cre.HasRef, cre.Ref = w.cur.present[EquipCreationRef], uint32(w.cur.val[EquipCreationRef])
	cre.HasID, cre.AbilityID = w.cur.present[EquipCreationAbilityID], uint32(w.cur.val[EquipCreationAbilityID])
	cre.MPPPresent, cre.MPPVal = w.cur.mppPresent, w.cur.mppVal
	cre.X, cre.Y, cre.Z = v[0], v[1], v[2]
	cre.Mask, cre.MaskFull, cre.MaskHasI0 = idx, full, idx[0] == 0
	cre.AfterBit = br.BitPos() + w.posAdvance()
	return cre, true
}

// readMaskIndices lit le masque de présence (FUN_1406d7610) et rend ses index, plus la branche
// empruntée. La contrainte commune aux deux branches — TOUT index annoncé doit exister dans
// l'archétype — est ce qui fait la sélectivité, et elle est bien plus sévère sur la branche
// pleine (33 bits de poids fort à zéro) que sur l'éparse.
//
// LA BRANCHE PLEINE EST ACCEPTÉE, et il le faut : un record de CRÉATION annonce l'état complet
// de l'entité, donc souvent plus de sept composants — au-delà, le compte de 3 bits de la
// branche éparse ne suffit plus et le moteur bascule sur le masque explicite. La refuser
// perdait la majorité des créations (mesuré : 58 records retenus contre 351 une fois acceptée).
func readMaskIndices(br *BitReader, comps int) (idx []int, full, ok bool) {
	if br.ReadBit() { // gate==1 : masque plein explicite R(64)
		// Le composant i est le bit i du mot rendu par ReadBits(64) — la convention de
		// consumeMask, dont la branche éparse pose `mask |= 1 << idx` et que
		// traverseComponentLoop interroge par `Mask & (1 << i)`.
		m := br.ReadBits(64)
		for i := 0; i < 64; i++ {
			if m&(uint64(1)<<uint(i)) == 0 {
				continue
			}
			if i >= comps {
				return nil, true, false
			}
			idx = append(idx, i)
		}
		return idx, true, len(idx) > 0
	}
	cnt := int(br.ReadBits(3))
	if cnt < 1 || cnt > worldObjectMaxMaskCnt {
		return nil, false, false
	}
	idx = make([]int, cnt)
	prev := -1
	for i := 0; i < cnt; i++ {
		v := int(br.ReadBits(6))
		if v <= prev || v >= comps {
			return nil, false, false
		}
		idx[i], prev = v, v
	}
	return idx, false, true
}
