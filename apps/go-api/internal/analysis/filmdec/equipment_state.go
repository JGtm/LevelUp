package filmdec

// equipment_state.go — L'ÉTAT DES ÉQUIPEMENTS DÉPLOYÉS, lu dans les paquets delta de ti=37.
//
// CE QUE CE BALAYAGE FAIT SORTIR, ET POURQUOI IL N'EXISTAIT PAS. Le film porte une ENTITÉ DU
// MONDE dédiée à l'équipement, dont les composants sont nommés en clair dans le registre
// (chunk_00) : `equipment-deployed-component`, `equipment-activated-component`,
// `equipment-creator-component`, `equipment-energy-component`. Leurs grammaires étaient
// PORTÉES depuis le portage ECS — mais les quatre désers CONSOMMAIENT leurs bits pour rester
// alignés et JETAIENT les valeurs, exactement le défaut d'i48 corrigé le 2026-08-14
// (cf. ability_rank.go). Ce fichier ne change AUCUNE largeur : il fait publier ce que les
// désers lisaient déjà.
//
// LA RÈGLE QUI GOUVERNE : c'est le DÉSERIALISEUR qui publie, jamais un second lecteur posé à
// côté de lui. Deux lecteurs du même champ divergent le jour où l'un des deux est corrigé.
//
// CE QU'IL NE DIT PAS. ti=37 est l'archétype des OBJETS d'équipement — ce qui se pose au sol
// et vit dans le monde. Un équipement sans objet (une impulsion instantanée) n'y apparaît
// pas, et rien ici ne permet de supposer le contraire : ce que le balayage compte, ce sont
// des entités, pas des utilisations d'équipement. L'identité de l'équipement (quel objet ?)
// n'est PAS dans ces quatre champs.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.

import "fmt"

// Étiquettes de registre des composants de ti=37 dont ce fichier publie la valeur. Elles
// servent DEUX fois chacune (le routage de consumeByName et la résolution d'index par nom
// ci-dessous) : les nommer évite qu'une correction de l'un oublie l'autre.
const (
	compEquipmentDeployed  = "equipment-deployed-component"
	compEquipmentActivated = "equipment-activated-component"
	compEquipmentCreator   = "equipment-creator-component"
	compEquipmentEnergy    = "equipment-energy-component"
)

// EquipmentField désigne l'un des quatre champs publiés. L'ordre est celui du flux.
type EquipmentField int

// Les quatre champs, et EquipmentFieldCount qui les compte (dimension des tableaux publiés).
const (
	EquipDeployed EquipmentField = iota
	EquipActivated
	EquipCreator
	EquipEnergy
	EquipmentFieldCount = 4
)

// String rend l'étiquette de registre du champ — la seule façon honnête de le nommer.
func (f EquipmentField) String() string {
	switch f {
	case EquipDeployed:
		return compEquipmentDeployed
	case EquipActivated:
		return compEquipmentActivated
	case EquipCreator:
		return compEquipmentCreator
	case EquipEnergy:
		return compEquipmentEnergy
	}
	return fmt.Sprintf("champ inconnu (%d)", int(f))
}

// equipmentStateHook, si non nil, reçoit CHAQUE lecture d'un des quatre composants : le
// champ, la valeur, et `present` — faux quand la porte du composant s'est fermée sans
// transmettre de valeur. Une porte fermée n'est PAS une valeur nulle, et les confondre
// fabriquerait des transitions qui n'existent pas.
var equipmentStateHook func(f EquipmentField, value uint64, present bool)

// SetEquipmentStateHook installe (ou retire, avec nil) la sonde des composants d'équipement.
func SetEquipmentStateHook(h func(f EquipmentField, value uint64, present bool)) {
	equipmentStateHook = h
}

func publishEquipment(f EquipmentField, value uint64, present bool) {
	if equipmentStateHook != nil {
		equipmentStateHook(f, value, present)
	}
}

// consumeEquipmentDeployed mirroite FUN_142ed4618 (ti=37 i20) : R(1), sans porte.
func consumeEquipmentDeployed(br *BitReader) {
	publishEquipment(EquipDeployed, br.ReadBits(1), true)
}

// consumeEquipmentActivated mirroite le déser de ti=37 i21 : R(1) porte de polarité INVERSÉE
// (la valeur R(3) n'est présente que si le bit vaut 0) ; sinon le corps FUN_1408f0ac4.
// Coût : 4 bits (porte à 0) ou 1 + celui de FUN_1408f0ac4 (porte à 1) — inchangé.
func consumeEquipmentActivated(br *BitReader) {
	if !br.ReadBit() {
		publishEquipment(EquipActivated, br.ReadBits(3), true)
		return
	}
	consume1408f0ac4(br)
	publishEquipment(EquipActivated, 0, false)
}

// consumeEquipmentCreator mirroite FUN_142ed45f4 (ti=37 i23) : R(1) porte INVERSÉE, puis R(5).
func consumeEquipmentCreator(br *BitReader) {
	if !br.ReadBit() {
		publishEquipment(EquipCreator, br.ReadBits(5), true)
		return
	}
	publishEquipment(EquipCreator, 0, false)
}

// consumeEquipmentEnergy mirroite FUN_141087bec (ti=37 i24) : R(14), sans porte.
func consumeEquipmentEnergy(br *BitReader) {
	publishEquipment(EquipEnergy, br.ReadBits(14), true)
}

// EquipmentStateSample est UN record delta de ti=37 dont la marche des composants a abouti
// jusqu'au dernier champ demandé. Val[f] n'a de sens que si Present[f].
type EquipmentStateSample struct {
	// Slot et Gen identifient la vie de l'objet — LA PAIRE, comme pour les projectiles : le
	// pool de slots reboucle et la génération ne fait que 2 bits.
	Slot, Gen uint32
	// Chunk / PacketIndex / TimestampUS localisent la lecture (même horloge que BipedPosition).
	Chunk, PacketIndex int
	TimestampUS        uint64
	// Seen[f] : le composant f était au masque ET la marche l'a consommé.
	Seen [EquipmentFieldCount]bool
	// Present[f] : le composant a transmis une valeur (porte ouverte). Val[f] la porte.
	Present [EquipmentFieldCount]bool
	Val     [EquipmentFieldCount]uint64
}

// EquipmentStateStats compte ce que la marche a rencontré. Sans ces dénominateurs, un
// histogramme de valeurs ne se juge pas.
type EquipmentStateStats struct {
	// Records est le nombre de records delta ti=37 reconnus.
	Records int
	// WithAny est le nombre de ces records dont le masque annonce au moins un des quatre.
	WithAny int
	// Walked / Broken : marches abouties, et marches interrompues (composant non porté ou
	// débordement du payload).
	Walked, Broken int
	// WithField[f] : records dont le masque annonce f. Read[f] : lectures abouties.
	// Gated[f] : lectures abouties dont la porte était fermée (aucune valeur transmise).
	WithField, Read, Gated [EquipmentFieldCount]int
	// Slots est le nombre de slots ti=37 retenus par la bande de slots.
	Slots int
	// MaskCensus[i] compte les records dont le masque annonce le composant d'index i, POUR
	// TOUS les composants de l'archétype — pas seulement les quatre publiés. C'est le
	// recensement qui dit OÙ le signal se trouve : un composant jamais annoncé ne portera
	// jamais rien, quel que soit le soin mis à décoder sa valeur.
	MaskCensus [worldObjectMaxComponent]int
}

// worldObjectMaxComponent borne le recensement : l'index de composant du masque tient sur
// 6 bits (cf. worldObjectIndexBits), donc 64 valeurs possibles.
const worldObjectMaxComponent = 1 << worldObjectIndexBits

// equipmentFieldIndices résout, DANS LE REGISTRE DU FILM, l'index d'itérateur de chacun des
// quatre composants. Résolu par NOM et jamais câblé : l'index est un numéro de build.
// Un composant absent du registre rend -1, et le balayage le déclare non mesurable.
func equipmentFieldIndices(arch Archetype) [EquipmentFieldCount]int {
	var out [EquipmentFieldCount]int
	for f := 0; f < EquipmentFieldCount; f++ {
		out[f] = -1
		if ids := arch.indicesOf(EquipmentField(f).String()); len(ids) > 0 {
			out[f] = ids[0]
		}
	}
	return out
}

// EquipmentArchetype charge l'archétype des objets d'équipement (ti=37) du registre du film.
func EquipmentArchetype(dir string) (Archetype, error) {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return Archetype{}, fmt.Errorf("chunk_00 (registre) illisible dans %s : %w", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return Archetype{}, fmt.Errorf("registre illisible dans %s : %w", dir, err)
	}
	arch, ok := reg.Archetype(EquipmentTypeIndex)
	if !ok {
		return Archetype{}, fmt.Errorf("archétype équipement %d absent du registre de %s",
			EquipmentTypeIndex, dir)
	}
	return arch, nil
}

// ScanFilmEquipmentState décode l'état des objets d'équipement transmis dans les paquets
// delta du film de dir.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe `equipmentStateHook`,
// qui est un global de paquet. Le hook est restauré à la sortie, y compris en cas d'erreur.
func ScanFilmEquipmentState(dir string) ([]EquipmentStateSample, EquipmentStateStats, error) {
	var st EquipmentStateStats
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, st, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	band := worldObjectSlotBand(dir, n, EquipmentTypeIndex)
	if len(band) == 0 {
		return nil, st, fmt.Errorf("aucun slot d'archétype ti=%d dans les keyframes de %s",
			EquipmentTypeIndex, dir)
	}
	st.Slots = len(band)
	arch, err := EquipmentArchetype(dir)
	if err != nil {
		return nil, st, err
	}
	w := equipmentWalk{arch: arch, want: equipmentFieldIndices(arch)}

	var cur EquipmentStateSample
	prev := equipmentStateHook
	SetEquipmentStateHook(func(f EquipmentField, value uint64, present bool) {
		cur.Seen[f], cur.Present[f], cur.Val[f] = true, present, value
	})
	defer SetEquipmentStateHook(prev)

	var out []EquipmentStateSample
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			out = append(out, w.scanPayload(pay, band, &st, &cur, pk, c)...)
		}
	}
	return out, st, nil
}

// equipmentWalk porte ce que la marche d'un record doit connaître (règle des 5 paramètres).
type equipmentWalk struct {
	arch Archetype
	// want[f] est l'index d'itérateur du champ f dans l'archétype, ou -1 s'il en est absent.
	want [EquipmentFieldCount]int
}

// scanPayload balaye UN payload delta et rend les records de ti=37 dont la marche a abouti.
func (w equipmentWalk) scanPayload(
	pay []byte, band map[uint32]bool, st *EquipmentStateStats,
	cur *EquipmentStateSample, pk FilmPacket, chunk int,
) []EquipmentStateSample {
	var out []EquipmentStateSample
	total := len(pay) * 8
	limit := total - (worldObjectHeaderBits + worldObjectIndexBits + projPosBits())
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || rec.Idx[0] != 0 { // i0 doit ouvrir le masque : c'est la position
			continue
		}
		st.Records++
		for _, id := range rec.Idx {
			if id >= 0 && id < worldObjectMaxComponent {
				st.MaskCensus[id]++
			}
		}
		last := w.lastWanted(rec.Idx)
		if last < 0 {
			p = rec.After - 1 // un record reconnu n'est pas re-balayé
			continue
		}
		st.WithAny++
		w.countMask(rec.Idx, st)
		*cur = EquipmentStateSample{
			Slot: rec.Slot, Gen: rec.Gen, Chunk: chunk,
			PacketIndex: pk.Index, TimestampUS: pk.TimestampUS,
		}
		if w.walk(pay, rec.After, total, rec.Idx, last) {
			st.Walked++
			w.countRead(cur, st)
			out = append(out, *cur)
		} else {
			st.Broken++
		}
		p = rec.After - 1
	}
	return out
}

// lastWanted rend le plus grand index d'itérateur du masque qui soit l'un des quatre champs,
// ou -1 si le masque n'en annonce aucun. La marche s'arrête là : pousser plus loin exposerait
// à un composant non porté sans rien apporter.
func (w equipmentWalk) lastWanted(idx []int) int {
	last := -1
	for _, id := range idx {
		for f := 0; f < EquipmentFieldCount; f++ {
			if w.want[f] >= 0 && id == w.want[f] && id > last {
				last = id
			}
		}
	}
	return last
}

func (w equipmentWalk) countMask(idx []int, st *EquipmentStateStats) {
	for _, id := range idx {
		for f := 0; f < EquipmentFieldCount; f++ {
			if w.want[f] >= 0 && id == w.want[f] {
				st.WithField[f]++
			}
		}
	}
}

func (w equipmentWalk) countRead(cur *EquipmentStateSample, st *EquipmentStateStats) {
	for f := 0; f < EquipmentFieldCount; f++ {
		if !cur.Seen[f] {
			continue
		}
		st.Read[f]++
		if !cur.Present[f] {
			st.Gated[f]++
		}
	}
}

// walk marche les composants du masque avec les désers de PRODUCTION jusqu'à consommer celui
// d'index `last` — c'est cette consommation qui déclenche le hook. Rend false dès qu'un
// composant intermédiaire n'est pas porté ou que la marche déborde du payload : au-delà, la
// position du curseur ne serait plus digne de confiance, et lire du bruit vaut moins que ne
// rien lire.
func (w equipmentWalk) walk(pay []byte, at, total int, idx []int, last int) bool {
	for _, id := range idx {
		if at > total {
			return false
		}
		name := w.arch.component(id)
		if name == "" {
			return false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(EquipmentTypeIndex), w.arch.Level(id))
		if !ported || br.BitPos() > total {
			return false
		}
		at = br.BitPos()
		if id == last {
			return true
		}
	}
	return false
}
