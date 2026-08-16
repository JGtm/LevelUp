package filmdec

// equipment_identity.go — L'IDENTITÉ DE L'OBJET `ti=37` : les quatre entiers de 32 bits que
// son ÉTAT PAR DÉFAUT lit déjà, et qu'il jetait.
//
// CE QUE CE FICHIER ATTAQUE. Le lot du 2026-08-15 (PLAN_EQUIPEMENT_TI37, verdict 0.6) a
// établi que les QUATRE composants delta de `ti=37` — deployed / activated / creator /
// energy — ne portent AUCUNE identité : rien n'y distingue une entité d'une autre. Le
// report écrit ce jour-là au registre désigne la seule voie restante, mot pour mot :
// « trouver dans le record de CRÉATION de l'entité la référence de définition de l'objet
// — même voie que la famille high-32 des armes au sol ».
//
// OÙ ELLE SE TROUVE. L'état par défaut de `ti=37` (`consumeDefaultStateTI37`,
// default_state_arch.go) est porté depuis le portage des vtable[0x60], et il lit QUATRE
// entiers de 32 bits :
//
//	consumeDefaultStateTI37 = V ; consumeDefaultStateTI36 ; ECS_ReadEntityRefIndex5 ;
//	                          porte -> R(32) « ability-enabled-id »
//	consumeDefaultStateTI36 = V ; consumeMultiplayerPropertiesBlock
//	consumeMultiplayerPropertiesBlock = R(9) ; R(32) ; [porte inversée] R(32) « variant-name »
//	                                    ... ; [porte G3] R(32) + ...
//
// Les quatre étaient consommés pour rester alignés et JETÉS — exactement le défaut d'`i48`
// corrigé le 2026-08-14 (ability_rank.go) puis des quatre champs delta le 2026-08-15
// (equipment_state.go). Ce fichier ne change AUCUNE largeur : il fait publier ce que le
// désérialiseur lisait déjà.
//
// LA RÈGLE QUI GOUVERNE : c'est le DÉSERIALISEUR qui publie, jamais un second lecteur posé
// à côté de lui. Deux lecteurs du même champ divergent le jour où l'un des deux est corrigé.
//
// PAR OÙ ON LIT, ET AVEC QUEL ORACLE. La voie la moins chère est la table KEYFRAME :
// `WalkKeyframeWorld` rend déjà, pour chaque image-clé, le bit de début et le `ti` de chaque
// record (walker durci, validé 249/250 entités). Le corps d'un record y suit l'en-tête de
// 64 bits `[id:32][field:26][ti:6]`, et c'est le MÊME lecteur de record NEW que partout
// ailleurs — on le rejoue donc par `TraverseEntity`, positionné sur les 6 bits de `ti`.
//
// L'ORACLE est alors gratuit et sans appel : la marche complète du record doit atterrir
// EXACTEMENT sur le premier bit du record SUIVANT, que le walker connaît indépendamment. Un
// état par défaut lu au mauvais endroit décale tout ce qui suit ; la probabilité qu'une
// chaîne de plusieurs centaines de bits retombe malgré tout pile sur la frontière voisine est
// nulle. Une valeur extraite d'un record qui ne retombe pas juste N'EST PAS une mesure.
//
// TROIS BASCULES DE GRAMMAIRE gouvernent cette marche et n'ont jamais été calibrées pour la
// table keyframe (leurs défauts viennent du chemin delta) : le corruption-check per-composant
// du mode film, le tail terminal de record NEW, et le routage vers le déserialiseur d'état
// par défaut de l'archétype. `EquipmentIdentityLayout` les expose pour que la MESURE les
// tranche, au lieu de les supposer.
//
// CE QUE CE FICHIER NE DIT PAS, et il ne faut pas le lui demander : il ne NOMME rien. Il
// rend des entiers et leurs dénominateurs. Décider qu'une valeur est « le mur » ou « le
// capteur » exige un témoin INDÉPENDANT (cf. PLAN_R3_IDENTITE_TI37 §5, phase 3) ; sans lui,
// une classe garde son numéro.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.

import (
	"fmt"
	"sort"
)

// EquipIDField désigne l'un des quatre entiers de 32 bits de l'état par défaut. L'ordre est
// celui du FLUX, pas celui de l'intérêt supposé.
type EquipIDField int

// Les quatre champs, et EquipIDFieldCount qui les compte (dimension des tableaux publiés).
const (
	// EquipIDMppDefinition est le R(32) sec du bloc multiplayer-properties (FUN_14080d6f0),
	// premier entier de 32 bits que rencontre un objet du monde.
	EquipIDMppDefinition EquipIDField = iota
	// EquipIDVariantName est le R(32) « variant-name » du même bloc (FUN_14080dec4), derrière
	// une porte de polarité INVERSÉE : présent seulement quand le bit vaut 0.
	EquipIDVariantName
	// EquipIDMppTail est le R(32) de la queue G3 du même bloc, derrière la porte G3.
	EquipIDMppTail
	// EquipIDAbilityEnabled est le R(32) « ability-enabled-id » propre à `ti=37`, derrière sa
	// porte. C'est le seul des quatre dont le nom de la fonction porteuse évoque une capacité.
	EquipIDAbilityEnabled
	// EquipIDFieldCount compte les champs publiés.
	EquipIDFieldCount = 4
)

// String rend l'étiquette du champ — le nom de son porteur dans le décompilé, la seule
// façon honnête de le nommer tant que rien n'est mesuré.
func (f EquipIDField) String() string {
	switch f {
	case EquipIDMppDefinition:
		return "mpp-r32 (FUN_14080d6f0)"
	case EquipIDVariantName:
		return "mpp-variant-name (FUN_14080dec4)"
	case EquipIDMppTail:
		return "mpp-tail-g3-r32"
	case EquipIDAbilityEnabled:
		return "ability-enabled-id"
	}
	return fmt.Sprintf("champ inconnu (%d)", int(f))
}

// equipmentIdentityHook, si non nil, reçoit CHAQUE lecture d'un des quatre entiers : le
// champ, la valeur, et `present` — faux quand la porte s'est fermée sans transmettre de
// valeur. Une porte fermée n'est PAS une valeur nulle, et les confondre fabriquerait une
// identité là où le flux n'en transmet aucune.
//
// ATTENTION — LE HOOK EST GLOBAL AU PAQUET, comme `equipmentStateHook` et `abilitySetHook`.
// Trois des quatre points de publication vivent dans `consumeMultiplayerPropertiesBlock`,
// PARTAGÉ par les archétypes 36/37/38/39/43 : le hook s'allume donc pour tout objet du
// monde traversé. C'est à l'APPELANT de ne faire traverser que des records `ti=37` — ce que
// `ScanFilmEquipmentIdentity` fait, record par record.
var equipmentIdentityHook func(f EquipIDField, value uint64, present bool)

// SetEquipmentIdentityHook installe (ou retire, avec nil) la sonde des entiers d'identité.
func SetEquipmentIdentityHook(h func(f EquipIDField, value uint64, present bool)) {
	equipmentIdentityHook = h
}

// publishEquipID transmet une lecture au hook, s'il y en a un.
func publishEquipID(f EquipIDField, value uint64, present bool) {
	if equipmentIdentityHook != nil {
		equipmentIdentityHook(f, value, present)
	}
}

// EquipmentIdentityLayout porte les trois bascules de grammaire du record NEW dont les
// défauts viennent du chemin DELTA et n'ont jamais été calibrés sur la table keyframe.
// Les exposer permet à la mesure de trancher — c'est l'oracle qui choisit, pas le lecteur.
type EquipmentIdentityLayout struct {
	// Corruption : corruption-check per-composant du mode film (R(1) après chaque composant
	// présent, cf. filmComponentCorruptionCheck).
	Corruption bool
	// TailBits : bits terminaux consommés après la boucle de composants d'un record NEW.
	TailBits int
	// ArchDefaultState : router `ti=37` vers `consumeDefaultStateTI37` (true) ou vers le
	// Skip(0) historique (false). À false, AUCUNE valeur n'est publiée — c'est le témoin
	// négatif de la voie elle-même.
	ArchDefaultState bool
}

// EquipmentIdentityLayouts est la matrice PROBÉE par le balayage : les huit combinaisons des
// trois bascules. Le premier élément est le défaut de production.
var EquipmentIdentityLayouts = []EquipmentIdentityLayout{
	{Corruption: false, TailBits: 0, ArchDefaultState: true},
	{Corruption: false, TailBits: 1, ArchDefaultState: true},
	{Corruption: true, TailBits: 0, ArchDefaultState: true},
	{Corruption: true, TailBits: 1, ArchDefaultState: true},
	{Corruption: false, TailBits: 0, ArchDefaultState: false},
	{Corruption: false, TailBits: 1, ArchDefaultState: false},
	{Corruption: true, TailBits: 0, ArchDefaultState: false},
	{Corruption: true, TailBits: 1, ArchDefaultState: false},
}

// String rend une étiquette lisible d'une combinaison.
func (l EquipmentIdentityLayout) String() string {
	return fmt.Sprintf("corruption=%-5v tail=%d etatParDefaut=%-5v",
		l.Corruption, l.TailBits, l.ArchDefaultState)
}

// apply installe la combinaison et rend la fonction qui restaure l'état précédent.
func (l EquipmentIdentityLayout) apply() func() {
	oc, ot, oa := filmComponentCorruptionCheck, newRecordTailBits, useArchDefaultStateDeser
	SetFilmComponentCorruptionCheck(l.Corruption)
	SetNewRecordTailBits(l.TailBits)
	SetUseArchDefaultStateDeser(l.ArchDefaultState)
	return func() {
		SetFilmComponentCorruptionCheck(oc)
		SetNewRecordTailBits(ot)
		SetUseArchDefaultStateDeser(oa)
	}
}

// EquipmentIdentityRead est UN record `ti=37` d'image-clé dont la marche a été rejouée sous
// la combinaison retenue. Val[f] n'a de sens que si Present[f], et AUCUNE valeur n'est une
// mesure tant qu'Exact est faux.
type EquipmentIdentityRead struct {
	// Slot et Gen identifient la vie de l'objet — LA PAIRE, comme partout ailleurs : le pool
	// de slots reboucle et la génération ne fait que 2 bits.
	Slot, Gen uint32
	// Chunk / PacketIndex / TimestampUS localisent la lecture (même horloge que BipedPosition).
	Chunk, PacketIndex int
	TimestampUS        uint64
	// Bit est la position du début du record dans le payload (traçabilité).
	Bit int
	// Present[f] : le champ a transmis une valeur (porte ouverte). Val[f] la porte.
	Present [EquipIDFieldCount]bool
	Val     [EquipIDFieldCount]uint64
	// Exact dit que la marche complète a atterri EXACTEMENT sur le début du record suivant.
	// C'est la seule condition sous laquelle les valeurs ci-dessus sont des mesures.
	Exact bool
	// Gap est l'écart final de la marche (0 si Exact). GapKnown est faux pour le DERNIER
	// record du payload, dont aucun voisin ne borne la marche.
	Gap      int
	GapKnown bool
}

// EquipmentIdentityStats compte ce que le balayage a rencontré. Sans ces dénominateurs, un
// histogramme de valeurs ne se juge pas.
type EquipmentIdentityStats struct {
	// Chunks / Keyframes : chunks lus, paquets d'image-clé traversés.
	Chunks, Keyframes int
	// AllRecords est le nombre total de records reconstruits par WalkKeyframeWorld, tous
	// archétypes confondus — le dénominateur qui dit si `ti=37` est rare ou courant.
	AllRecords int
	// Records est le nombre de ces records dont l'archétype est `ti=37`.
	Records int
	// Bounded est le nombre de records `ti=37` bornés par un voisin (donc contrôlables).
	Bounded int
	// Exact / Inexact / Desync, sous la combinaison RETENUE : marches bit-exactes, marches
	// ratées, marches interrompues par un composant non porté (DesyncAt >= 0).
	Exact, Inexact, Desync int
	// GapHist est la distribution des écarts finaux sous la combinaison retenue (0 = exact).
	GapHist map[int]int
	// LayoutExact / LayoutDesync : résultat de la MATRICE, index parallèle à
	// EquipmentIdentityLayouts. C'est cette ligne de chiffres qui tranche la grammaire.
	LayoutExact, LayoutDesync []int
}

// ScanFilmEquipmentIdentity rejoue le lecteur de record NEW sur chaque record `ti=37` des
// images-clés du film de dir, sous la combinaison `lay`, et rend les quatre entiers de
// 32 bits que son état par défaut transporte. Il PROBE au passage les huit combinaisons de
// `EquipmentIdentityLayouts` et publie leurs taux de marche bit-exacte.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe
// `equipmentIdentityHook` ET manipule les bascules globales de grammaire. Les deux sont
// restaurés à la sortie, y compris en cas d'erreur. L'appelant doit détenir
// `LockProcessDecode`.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
func ScanFilmEquipmentIdentity(dir string, lay EquipmentIdentityLayout) (
	[]EquipmentIdentityRead, EquipmentIdentityStats, error,
) {
	st := EquipmentIdentityStats{
		GapHist:      map[int]int{},
		LayoutExact:  make([]int, len(EquipmentIdentityLayouts)),
		LayoutDesync: make([]int, len(EquipmentIdentityLayouts)),
	}
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, st, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return nil, st, fmt.Errorf("chunk_00 (registre) illisible dans %s : %w", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return nil, st, fmt.Errorf("registre illisible dans %s : %w", dir, err)
	}

	var cur EquipmentIdentityRead
	prevHook := equipmentIdentityHook
	SetEquipmentIdentityHook(func(f EquipIDField, value uint64, present bool) {
		cur.Present[f], cur.Val[f] = present, value
	})
	defer SetEquipmentIdentityHook(prevHook)

	w := equipIDWalk{reg: reg, lay: lay}
	var out []EquipmentIdentityRead
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		st.Chunks++
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			st.Keyframes++
			out = append(out, w.scanKeyframe(pk.Payload(data), &st, &cur, pk, c)...)
		}
	}
	return out, st, nil
}

// keyframeRecordTIBit est le décalage, depuis le début d'un record de la table keyframe, des
// 6 bits de `typeIndex` : l'en-tête vaut `[id:32][field:26][ti:6]` et `ti` en occupe la
// queue (cf. keyframe_world.go). C'est là que `TraverseEntity` doit être positionné, parce
// que c'est par ces 6 bits qu'il commence.
const keyframeRecordTIBit = 58

// equipIDWalk porte ce que la marche d'un record doit connaître (règle des 5 paramètres).
type equipIDWalk struct {
	reg *Registry
	lay EquipmentIdentityLayout
}

// scanKeyframe rejoue les records `ti=37` d'UN payload d'image-clé.
func (w equipIDWalk) scanKeyframe(
	pay []byte, st *EquipmentIdentityStats,
	cur *EquipmentIdentityRead, pk FilmPacket, chunk int,
) []EquipmentIdentityRead {
	recs := WalkKeyframeWorld(pay)
	if len(recs) == 0 {
		return nil
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	st.AllRecords += len(recs)

	var out []EquipmentIdentityRead
	for i, r := range recs {
		if r.TI != EquipmentTypeIndex {
			continue
		}
		st.Records++
		if i+1 >= len(recs) {
			continue // dernier record du payload : aucun voisin ne borne la marche
		}
		st.Bounded++
		want := recs[i+1].Bit
		w.probeLayouts(pay, r.Bit, want, st)

		*cur = EquipmentIdentityRead{
			Slot: uint32(r.Slot), Gen: uint32(r.Gen), Chunk: chunk,
			PacketIndex: pk.Index, TimestampUS: pk.TimestampUS, Bit: r.Bit,
		}
		restore := w.lay.apply()
		trace := traverseKeyframeRecord(pay, r.Bit, w.reg)
		restore()
		if trace.DesyncAt >= 0 {
			st.Desync++
			continue
		}
		cur.Gap, cur.GapKnown = want-trace.EndBit, true
		cur.Exact = cur.Gap == 0
		st.GapHist[cur.Gap]++
		if cur.Exact {
			st.Exact++
		} else {
			st.Inexact++
		}
		out = append(out, *cur)
	}
	return out
}

// traverseKeyframeRecord rejoue le lecteur de record NEW de PRODUCTION sur un record de la
// table keyframe, positionné sur ses 6 bits de `typeIndex`.
func traverseKeyframeRecord(pay []byte, recBit int, reg *Registry) EntityTrace {
	br := NewBitReader(pay)
	br.SetBitPos(recBit + keyframeRecordTIBit)
	return TraverseEntity(br, reg, 0)
}

// probeEquipIDLayouts rejoue le MÊME record sous les huit combinaisons de grammaire et
// compte, pour chacune, les marches bit-exactes. C'est la mesure qui tranche : aucune
// combinaison n'est supposée juste, elles sont toutes essayées et comparées sur le même
// dénominateur.
func (w equipIDWalk) probeLayouts(pay []byte, recBit, want int, st *EquipmentIdentityStats) {
	// Le hook est neutralisé pendant la matrice : ces marches sont des essais, leurs valeurs
	// ne doivent pas polluer la lecture retenue.
	prev := equipmentIdentityHook
	SetEquipmentIdentityHook(nil)
	defer SetEquipmentIdentityHook(prev)

	for k, l := range EquipmentIdentityLayouts {
		restore := l.apply()
		trace := traverseKeyframeRecord(pay, recBit, w.reg)
		restore()
		switch {
		case trace.DesyncAt >= 0:
			st.LayoutDesync[k]++
		case trace.EndBit == want:
			st.LayoutExact[k]++
		}
	}
}
