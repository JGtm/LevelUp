package filmdec

// equipment_recovery.go — LA RÉCUPÉRATION GATÉE DES ÉMISSIONS i48 MANQUÉES (décision D1 du
// PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03, mesures R2 + P1.0).
//
// CE QUE C'EST. Le balayage strict (walkAbilityEmissions) manque ~5 % des émissions du canal
// d'équipement, et le compteur de rotation R(3) les DÉNONCE : un pas k > 1 entre deux
// émissions d'une même vie annonce k−1 émissions manquées, aux compteurs PRÉDITS. La mesure
// R2 (13 sauts, 8 films) a prouvé que dans ~3 cas sur 4 les octets EXISTENT dans le film,
// sous deux formes de record que le balayage de production rejette par construction :
//
//   - SANS i0 : l'en-tête de production est intact (tag=1, bits 16-17 nuls, comptage 2..7,
//     indices strictement croissants) mais le masque ne commence PAS au composant 0 — la
//     position n'a pas changé dans ce record, l'équipement si. Garde fautive :
//     `ascendingFromZero` (premier index = 0).
//   - MASQUE DENSE R(64) : bit de porte du masque à 1 (grammaire FUN_1406d7610 — la seule
//     forme possible à 8 composants et plus), ordre de bits `bit k = composant 63−k`, FIGÉ
//     par P1.0 sur 44 témoins (4 de R2 + 40 supplémentaires sur 12 films, unanimes au profil
//     de production ; l'unique contre-signal msb inverse est illisible par cette forme).
//     Ces records portent un i0 absolu de la bonne région : l'ancre anti-bruit demeure.
//
// POURQUOI LA RÉCUPÉRATION EST GATÉE, ET PAR QUOI. Le relâchement inconditionnel est RÉFUTÉ
// par contrôle négatif (R2 §5 : +800 fausses acceptations sur 10 films, chaînes de compteur
// détruites). La seule politique sûre, mesurée : re-balayer LA SEULE fenêtre d'un saut
// annoncé, et n'accepter un candidat QUE si son compteur comble le saut — au plus k−1
// candidats, dans l'ordre des prédictions, sans créer ni répétition ni nouveau saut. Le
// risque résiduel mesuré est de l'ordre de 1-2 % par saut (R2 §6).
//
// AUCUNE GARDE DU BALAYAGE STRICT N'EST AFFAIBLIE : la récupération est une POST-PASSE, qui
// ne touche ni matchBipedHeader ni ascendingFromZero, et ne lit que des fenêtres bornées.

import "sort"

// equipRecoveryMaxDense borne le nombre de composants d'un masque dense candidat — enveloppe
// mesurée (R2/P1.0 : 8 à 32 composants observés sur les vrais manques ; au-delà de 40, le
// motif est du bruit).
const equipRecoveryMaxDense = 40

// equipRecoveryHeadCounter est le compteur VIRTUEL qui précède la première émission d'une
// vie : la première émission attendue porte equipmentFirstCounter (5), donc la tête de vie
// se traite comme une fenêtre de saut dont l'émission d'avant porterait 4 — la même
// arithmétique de prédiction sert aux deux fenêtres.
const equipRecoveryHeadCounter = equipmentFirstCounter - 1

// equipRecoveryWindow est UNE fenêtre de re-balayage : un saut de compteur entre deux
// émissions d'une même vie, ou la tête d'une vie dont la première émission n'a pas le
// compteur attendu (fenêtre [naissance, première émission]).
type equipRecoveryWindow struct {
	slot  uint32
	fromC uint32 // compteur d'avant (equipRecoveryHeadCounter pour une tête de vie)
	toC   uint32 // compteur d'après (l'émission qui ferme la fenêtre)
	miss  int    // émissions manquées annoncées : (toC − fromC − 1) modulo 8
	// tsMin/tsMax bornent la fenêtre sur l'horloge des paquets ; chunkMin/chunkMax bornent
	// la lecture disque (le balayage ne lit JAMAIS le film entier — leçon R2 §5).
	tsMin, tsMax       uint64
	chunkMin, chunkMax int
	head               bool
	cands              []equipRecovered
}

// equipRecovered est un candidat retrouvé dans une fenêtre : une émission plus son offset de
// bit (dédoublonnage) et sa forme (journal).
type equipRecovered struct {
	abilityEmission
	off   int
	dense bool
}

// buildEquipRecoveryWindows dresse les fenêtres de re-balayage d'un film : une par saut de
// compteur, plus une par tête de vie hors norme quand le témoin de naissance existe.
// `bySlot` doit être trié par instant croissant à l'intérieur de chaque slot.
func buildEquipRecoveryWindows(
	bySlot map[uint32][]abilityEmission, bornAt func(uint32) (uint64, bool),
) []equipRecoveryWindow {
	var out []equipRecoveryWindow
	for slot, list := range bySlot {
		if len(list) == 0 {
			continue
		}
		if first := list[0]; first.Counter != equipmentFirstCounter && bornAt != nil {
			if birth, ok := bornAt(slot); ok && birth < first.TimestampUS {
				out = append(out, equipRecoveryWindow{
					slot: slot, fromC: equipRecoveryHeadCounter, toC: first.Counter,
					miss:  counterStep(equipmentFirstCounter, first.Counter),
					tsMin: birth, tsMax: first.TimestampUS,
					chunkMin: 1, chunkMax: first.Chunk, head: true,
				})
			}
		}
		for i := 1; i < len(list); i++ {
			step := counterStep(list[i-1].Counter, list[i].Counter)
			if step <= 1 {
				continue
			}
			out = append(out, equipRecoveryWindow{
				slot: slot, fromC: list[i-1].Counter, toC: list[i].Counter, miss: step - 1,
				tsMin: list[i-1].TimestampUS, tsMax: list[i].TimestampUS,
				chunkMin: list[i-1].Chunk, chunkMax: list[i].Chunk,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].tsMin != out[j].tsMin {
			return out[i].tsMin < out[j].tsMin
		}
		return out[i].slot < out[j].slot
	})
	return out
}

// scanEquipmentRecovery re-balaye les fenêtres et rend, pour chacune, les émissions
// ACCEPTÉES par le témoin de compteur — avec leur offset de bit : c'est lui qui départage
// deux émissions du même paquet à la fusion (revue ronde 1, F3). Lecture disque bornée aux
// chunks des fenêtres.
func scanEquipmentRecovery(s abilityScanSetup, wins []equipRecoveryWindow) []equipRecovered {
	if len(wins) == 0 {
		return nil
	}
	cMin, cMax := wins[0].chunkMin, wins[0].chunkMax
	for _, w := range wins {
		if w.chunkMin < cMin {
			cMin = w.chunkMin
		}
		if w.chunkMax > cMax {
			cMax = w.chunkMax
		}
	}
	last, restore := equipRecoveryHook()
	defer restore()
	for c := cMin; c <= cMax; c++ {
		active := windowsOfChunk(wins, c)
		if len(active) == 0 {
			continue
		}
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			scanEquipRecoveryPacket(s, pk.Payload(data), active, c, pk, last)
		}
	}
	var out []equipRecovered
	for i := range wins {
		out = append(out, acceptEquipRecovery(&wins[i])...)
	}
	return out
}

// windowsOfChunk rend les fenêtres dont la plage de chunks couvre c.
func windowsOfChunk(wins []equipRecoveryWindow, c int) []*equipRecoveryWindow {
	var out []*equipRecoveryWindow
	for i := range wins {
		if wins[i].chunkMin <= c && c <= wins[i].chunkMax {
			out = append(out, &wins[i])
		}
	}
	return out
}

// equipRecoveryHook installe la sonde i48 du déserialiseur de production et rend (capture,
// restauration) — même geste que walkAbilityEmissionsWith : le hook EST la grammaire, on ne
// relit pas les bits à côté de lui.
func equipRecoveryHook() (*struct {
	counter uint32
	rank    int
	got     bool
}, func()) {
	last := &struct {
		counter uint32
		rank    int
		got     bool
	}{}
	prev := abilitySetHook
	SetAbilitySetHook(func(counter uint64, rank, _ int) {
		last.counter, last.rank, last.got = uint32(counter), rank, true
	})
	return last, func() { SetAbilitySetHook(prev) }
}

// scanEquipRecoveryPacket balaye un paquet position de bit par position de bit, SANS saut
// post-match (les records des deux formes s'imbriquent dans le flux que le strict a déjà
// parcouru autrement), et range chaque candidat marché dans sa fenêtre.
func scanEquipRecoveryPacket(
	s abilityScanSetup, pay []byte, wins []*equipRecoveryWindow, chunk int, pk FilmPacket,
	last *struct {
		counter uint32
		rank    int
		got     bool
	},
) {
	var active []*equipRecoveryWindow
	for _, w := range wins {
		if w.tsMin <= pk.TimestampUS && pk.TimestampUS <= w.tsMax {
			active = append(active, w)
		}
	}
	if len(active) == 0 {
		return
	}
	total := len(pay) * 8
	for p := 0; p+bipedHeaderBits+bipedIndexBits <= total; p++ {
		if readBitsAt(pay, p, 1) != 1 {
			continue
		}
		slot := readBitsAt(pay, p+1, bipedSlotBits)
		var w *equipRecoveryWindow
		for _, cand := range active {
			if cand.slot == slot {
				w = cand
				break
			}
		}
		if w == nil {
			continue
		}
		// EN-TÊTE DE PRODUCTION INTACT (R2 §4) : tag=1 et bit 16 nul. Seule la PORTE du
		// masque (bit 17) distingue les deux formes récupérables.
		if readBitsAt(pay, p+14, 2) != 1 || readBitsAt(pay, p+16, 1) != 0 {
			continue
		}
		counter, rank, ok := walkEquipRecoveryAt(s, pay, p, total, last)
		if !ok {
			continue
		}
		w.cands = append(w.cands, equipRecovered{
			abilityEmission: abilityEmission{
				Slot: slot, Chunk: chunk, PacketIndex: pk.Index,
				TimestampUS: pk.TimestampUS, Counter: counter, Rank: rank,
			},
			off: p, dense: readBitsAt(pay, p+17, 1) == 1,
		})
	}
}

// walkEquipRecoveryAt tente les deux formes récupérables à la position p (en-tête déjà
// contrôlé) et marche le record jusqu'à i48 avec les désers de PRODUCTION. Rend (compteur,
// rang, ok) — ok seulement si la marche a consommé i48.
func walkEquipRecoveryAt(
	s abilityScanSetup, pay []byte, p, total int,
	last *struct {
		counter uint32
		rank    int
		got     bool
	},
) (uint32, int, bool) {
	last.got = false
	reached := false
	stop := func(id int) bool {
		if id == i48Index {
			reached = true
			return false
		}
		return true
	}
	if readBitsAt(pay, p+17, 1) == 0 {
		// FORME SANS i0 : comptage 2..7, indices strictement croissants, premier != 0 (un
		// premier index à 0 est un record standard, déjà jugé par le balayage strict). Les
		// composants commencent juste après les indices — il n'y a pas de vec3 devant.
		mc := int(readBitsAt(pay, p+18, 3))
		if mc < bipedMinMaskCnt || mc > bipedMaxMaskCnt {
			return 0, 0, false
		}
		// GARDE DE BORNE (revue ronde 1, F1 — panic reproduite) : la boucle d'appel ne
		// garantit que l'en-tête et UN index ; les mc indices lisent jusqu'à p+21+6·mc, et
		// readBitsAt indexe le tampon SANS filet. Même patron que le balayage strict
		// (matchBipedHeaderRaw : needBits > total -> rejet) ; la forme dense se borne déjà.
		if p+bipedHeaderBits+bipedIndexBits*mc > total {
			return 0, 0, false
		}
		idx, ok := ascendingIndices(pay, p+bipedHeaderBits, mc)
		if !ok || idx[0] == 0 || !maskHas(idx, i48Index) {
			return 0, 0, false
		}
		walkComponentsAt(pay, p+bipedHeaderBits+bipedIndexBits*mc, total, idx, s.arch, stop)
	} else {
		// FORME DENSE R(64), ordre FIGÉ par P1.0 : bit k du flux = composant 63−k. Le record
		// porte un i0 absolu de la bonne région — l'ancre anti-bruit du balayage strict.
		i0 := p + 18 + 64 // [1 préfixe][14 id][2 tag][1 porte=1] puis R(64), i0 ensuite
		if i0+s.lay.TotalBits() > total {
			return 0, 0, false
		}
		idx := denseMaskIndices(pay, p+18)
		if len(idx) < bipedMinMaskCnt || len(idx) > equipRecoveryMaxDense ||
			idx[0] != 0 || !maskHas(idx, i48Index) {
			return 0, 0, false
		}
		const preGate = i0SpineBits + i0UseDefaultBits
		if readBitsAt(pay, i0, preGate) != 0 ||
			readBitsAt(pay, i0+preGate, s.lay.GateBits-preGate) != s.lay.Region {
			return 0, 0, false
		}
		walkRecordComponents(pay, i0, total, idx, s.lay, s.arch, stop)
	}
	if !reached || !last.got {
		return 0, 0, false
	}
	return last.counter, last.rank, true
}

// ascendingIndices lit une liste d'indices STRICTEMENT croissants sans exiger que le premier
// soit 0 — c'est la seule garde que la forme « sans i0 » relâche ; la croissance stricte,
// elle, est conservée (ancre anti-bruit, R2 §4).
func ascendingIndices(pay []byte, at, count int) ([]int, bool) {
	out := make([]int, 0, count)
	prev := -1
	for k := 0; k < count; k++ {
		idx := int(readBitsAt(pay, at+bipedIndexBits*k, bipedIndexBits))
		if idx <= prev {
			return nil, false
		}
		prev = idx
		out = append(out, idx)
	}
	return out, true
}

// denseMaskIndices lit un masque dense R(64) à la position bit at, dans l'ordre FIGÉ par
// P1.0 : bit k du flux = composant 63−k. Rend les index levés, croissants.
func denseMaskIndices(pay []byte, at int) []int {
	var idx []int
	for k := 63; k >= 0; k-- {
		if readBitsAt(pay, at+k, 1) == 1 {
			idx = append(idx, 63-k)
		}
	}
	return idx
}

// acceptEquipRecovery applique le TÉMOIN DE COMPTEUR à une fenêtre : n'accepte que des
// candidats aux compteurs PRÉDITS, sans doublon de compteur (l'ambiguïté rejette la fenêtre
// entière — départager serait un vote), dans l'ordre des prédictions, et SEULEMENT si leur
// insertion ne crée ni répétition ni nouveau saut : la chaîne fromC → candidats → toC ne
// doit pas porter plus d'un pas différent de 1 (le trou résiduel d'une récupération
// partielle est permis, un éclatement du saut en deux ne l'est pas).
func acceptEquipRecovery(w *equipRecoveryWindow) []equipRecovered {
	if len(w.cands) == 0 || w.miss == 0 {
		return nil
	}
	predIdx := map[uint32]int{}
	for j := 0; j < w.miss; j++ {
		predIdx[(w.fromC+1+uint32(j))%8] = j
	}
	var kept []equipRecovered
	seen := map[uint32]bool{}
	for _, c := range w.cands {
		if _, predicted := predIdx[c.Counter%8]; !predicted {
			continue // hors prédiction : plancher de bruit, jamais accepté (R2 §5)
		}
		if seen[c.Counter%8] {
			return nil // deux candidats pour le même compteur prédit : fenêtre rejetée
		}
		seen[c.Counter%8] = true
		kept = append(kept, c)
	}
	if len(kept) == 0 {
		return nil
	}
	sort.Slice(kept, func(i, j int) bool {
		if kept[i].TimestampUS != kept[j].TimestampUS {
			return kept[i].TimestampUS < kept[j].TimestampUS
		}
		return kept[i].off < kept[j].off
	})
	holes, prev := 0, w.fromC
	for i, c := range kept {
		if i > 0 && predIdx[c.Counter%8] <= predIdx[kept[i-1].Counter%8] {
			return nil // l'ordre temporel contredit l'ordre des compteurs : bruit
		}
		if counterStep(prev, c.Counter) != 1 {
			holes++
		}
		prev = c.Counter
	}
	if counterStep(prev, w.toC) != 1 {
		holes++
	}
	if holes > 1 {
		return nil // l'insertion éclaterait le saut en deux : refusé
	}
	return kept
}
