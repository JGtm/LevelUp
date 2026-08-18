package filmdec

// game_entities_hunt.go — CHERCHER LE SLOT D UN ARCHETYPE QUE LES IMAGES-CLES NE PORTENT PAS.
//
// POURQUOI CE SECOND CHEMIN EXISTE. La premiere mesure du lot B a trouve la bande de ti=0
// VIDE sur `000d5950` : le recensement des images-cles porte 28 archetypes (ti=2, 4, 5, 6, 9,
// 12 ... 47) et pas celui du moteur de partie. L'archetype est pourtant declare au registre du
// film, avec ses 27 composants nommes. Conclusion possible seulement apres mesure : soit
// l'entite du moteur n'entre pas dans la table des images-cles, soit son handle y est rejete.
// Dans les deux cas, la bande observee ne peut pas la donner — et un lot qui s'arreterait la
// conclurait « l'horloge officielle n'est pas dans le film », ce qui serait faux.
//
// LA SIGNATURE UTILISEE EST CELLE DE LA GRAMMAIRE, PAS CELLE DES VALEURS. On compte, PAR SLOT,
// les en-tetes delta dont le masque tient entierement dans la grammaire de l'archetype ET
// annonce un composant impose (le round-timer pour ti=0). Aucune valeur n'entre dans le
// critere : chercher un slot par ce que ses valeurs devraient valoir serait construire le
// resultat. Le slot du moteur, s'il existe, se detache par son DEBIT ; les autres restent au
// plancher de bruit, que la fonction rend avec le compte pour que la comparaison soit possible.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requete.

import (
	"fmt"
	"sort"
)

// SlotHuntRow est le releve d'UN slot pendant une chasse.
type SlotHuntRow struct {
	Slot uint32
	// Records : en-tetes reconnus sur ce slot, tous masques confondus. InGrammar : ceux dont
	// le masque tient dans la grammaire. WithMust : ceux qui annoncent en plus le composant
	// impose — c'est ce compte qui designe un candidat.
	Records, InGrammar, WithMust int
}

// HuntArchetypeSlots compte, pour CHAQUE slot de l'espace, les en-tetes delta compatibles avec
// la grammaire de l'archetype `ti`, et parmi eux ceux qui annoncent le composant `must`.
// Rend les lignes triees par `WithMust` decroissant. PAS de marche de composants : la chasse
// ne lit aucune valeur, elle localise un slot.
//
// HORS LIGNE (I/O disque sur tout le film).
func HuntArchetypeSlots(dir string, ti int, must string) ([]SlotHuntRow, error) {
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	reg, err := gameEntityRegistry(dir)
	if err != nil {
		return nil, err
	}
	arch, ok := reg.Archetype(ti)
	if !ok {
		return nil, fmt.Errorf("archetype ti=%d absent du registre de %s", ti, dir)
	}
	mustIdx := -1
	if ids := arch.indicesOf(must); len(ids) > 0 {
		mustIdx = ids[0]
	}
	if mustIdx < 0 {
		return nil, fmt.Errorf("composant %q absent de l'archetype ti=%d de %s", must, ti, dir)
	}
	// Bande OUVERTE : tout slot est accepte par l'ancrage, c'est le principe meme de la chasse.
	all := make(map[uint32]bool, kfTableCap)
	for s := uint32(0); s < kfTableCap; s++ {
		all[s] = true
	}
	rows := map[uint32]*SlotHuntRow{}
	for ch := 1; ch <= n; ch++ {
		data, err := ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			huntPayload(pk.Payload(data), all, len(arch.Components), mustIdx, rows)
		}
	}
	out := make([]SlotHuntRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].WithMust != out[j].WithMust {
			return out[i].WithMust > out[j].WithMust
		}
		return out[i].Slot < out[j].Slot
	})
	return out, nil
}

// huntPayload compte les en-tetes d'UN payload delta.
func huntPayload(
	pay []byte, all map[uint32]bool, grammar, mustIdx int, rows map[uint32]*SlotHuntRow,
) {
	total := len(pay) * 8
	limit := total - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, all)
		if !ok {
			continue
		}
		r := rows[rec.Slot]
		if r == nil {
			r = &SlotHuntRow{Slot: rec.Slot}
			rows[rec.Slot] = r
		}
		r.Records++
		inGrammar, hasMust := true, false
		for _, i := range rec.Idx {
			if i >= grammar {
				inGrammar = false
			}
			if i == mustIdx {
				hasMust = true
			}
		}
		if inGrammar {
			r.InGrammar++
			if hasMust {
				r.WithMust++
			}
		}
		p = rec.After - 1
	}
}

// ClockCandidate est le releve d'UN slot pendant la chasse a l'horloge de manche.
type ClockCandidate struct {
	Slot uint32
	// Samples : lectures d'horloge abouties sur ce slot. Down / Up : transitions ou la
	// valeur A decroit / croit d'un echantillon au suivant, a slot egal.
	Samples, Down, Up int
	// MinA / MaxA : bornes de la valeur A dequantifiee (secondes).
	MinA, MaxA float32
	// Slope : pente de A contre l'horloge du film, en secondes par seconde, par moindres
	// carres sur tous les echantillons du slot.
	Slope float64
	// FirstUS / LastUS : bornes temporelles des echantillons de ce slot.
	FirstUS, LastUS uint64
}

// HuntGameEngineClock cherche le slot qui porte l'HORLOGE DE MANCHE, en marchant les records
// compatibles avec la grammaire de ti=0 et en relevant la valeur capturee.
//
// POURQUOI CETTE CHASSE N'EST PAS CIRCULAIRE, alors qu'elle utilise la grandeur meme que le
// lot B veut mesurer. Le critere d'IDENTIFICATION est qualitatif — un slot dont la valeur
// decroit de facon monotone sur toute la partie — et il ne fixe ni la pente, ni la valeur de
// depart, ni l'instant du premier decompte. Ce sont ces trois-la que B.0.2 mesure, contre
// l'horloge du film, `regulation.toml` et `originMs` : trois oracles EXTERIEURS au film ou
// exterieurs a ti=0. Un slot choisi parce qu'il decroit ne peut pas, par construction, avoir
// la bonne valeur initiale ni le bon instant de depart.
//
// LE PIEGE MESURE QUI JUSTIFIE CE CHEMIN : la chasse par grammaire seule (`HuntArchetypeSlots`)
// est dominee par le trafic des bipedes (slots 512-620), dont la densite de records fabrique
// assez d'en-tetes compatibles pour ecraser un archetype a une seule entite.
//
// HORS LIGNE (I/O disque sur tout le film). Installe et restaure les hooks de paquet.
func HuntGameEngineClock(dir string) ([]ClockCandidate, error) {
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	reg, err := gameEntityRegistry(dir)
	if err != nil {
		return nil, err
	}
	arch, ok := reg.Archetype(GameEngineTypeIndex)
	if !ok {
		return nil, fmt.Errorf("archetype ti=%d absent du registre de %s", GameEngineTypeIndex, dir)
	}
	ids := arch.indicesOf(compGameEngineRoundTimer)
	if len(ids) == 0 {
		return nil, fmt.Errorf("%s absent de l'archetype ti=%d de %s", compGameEngineRoundTimer,
			GameEngineTypeIndex, dir)
	}
	h := clockHunt{arch: arch, timerIdx: ids[0], acc: map[uint32]*clockAcc{}}
	all := make(map[uint32]bool, kfTableCap)
	for s := uint32(0); s < kfTableCap; s++ {
		all[s] = true
	}
	for ch := 1; ch <= n; ch++ {
		data, err := ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type == PacketTypeDelta {
				h.scanPayload(pk.Payload(data), all, pk.TimestampUS)
			}
		}
	}
	return h.rows(), nil
}

// clockAcc accumule les echantillons d'horloge d'un slot (regression incluse, en une passe).
type clockAcc struct {
	n, down, up            int
	minA, maxA             float32
	first, last            uint64
	prevA                  float32
	hasPrev                bool
	sx, sy, sxx, sxy, base float64
}

type clockHunt struct {
	arch     Archetype
	timerIdx int
	acc      map[uint32]*clockAcc
}

// scanPayload balaye UN payload delta et releve les horloges des records compatibles.
func (h *clockHunt) scanPayload(pay []byte, all map[uint32]bool, tUS uint64) {
	total := len(pay) * 8
	limit := total - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, all)
		if !ok {
			continue
		}
		p = rec.After - 1
		if !h.eligible(rec.Idx) {
			continue
		}
		if rt, ok := h.walk(pay, rec.After, total, rec.Idx); ok {
			h.add(rec.Slot, rt, tUS)
		}
	}
}

// eligible exige que le masque tienne dans la grammaire de ti=0 ET annonce l'horloge.
func (h *clockHunt) eligible(idx []int) bool {
	has := false
	for _, i := range idx {
		if i >= len(h.arch.Components) {
			return false
		}
		if i == h.timerIdx {
			has = true
		}
	}
	return has
}

// walk consomme les composants du masque jusqu'a l'horloge et rend la valeur capturee.
func (h *clockHunt) walk(pay []byte, at, total int, idx []int) (RoundTimer, bool) {
	for _, id := range idx {
		if at > total {
			return RoundTimer{}, false
		}
		name := h.arch.component(id)
		if name == "" {
			return RoundTimer{}, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, payload, ported := consumeByNameCapturing(br, name, GameEngineTypeIndex, h.arch.Level(id))
		if !ported || br.BitPos() > total {
			return RoundTimer{}, false
		}
		at = br.BitPos()
		if id == h.timerIdx {
			rt, ok := payload.(RoundTimer)
			return rt, ok
		}
	}
	return RoundTimer{}, false
}

// add range un echantillon dans l'accumulateur de son slot.
func (h *clockHunt) add(slot uint32, rt RoundTimer, tUS uint64) {
	a := h.acc[slot]
	if a == nil {
		a = &clockAcc{minA: rt.A, maxA: rt.A, first: tUS, base: float64(tUS) / 1e6}
		h.acc[slot] = a
	}
	a.n++
	a.last = tUS
	if rt.A < a.minA {
		a.minA = rt.A
	}
	if rt.A > a.maxA {
		a.maxA = rt.A
	}
	if a.hasPrev {
		switch {
		case rt.A < a.prevA:
			a.down++
		case rt.A > a.prevA:
			a.up++
		}
	}
	a.prevA, a.hasPrev = rt.A, true
	x := float64(tUS)/1e6 - a.base
	y := float64(rt.A)
	a.sx, a.sy, a.sxx, a.sxy = a.sx+x, a.sy+y, a.sxx+x*x, a.sxy+x*y
}

// rows rend les candidats tries par nombre d'echantillons decroissant.
func (h *clockHunt) rows() []ClockCandidate {
	out := make([]ClockCandidate, 0, len(h.acc))
	for slot, a := range h.acc {
		c := ClockCandidate{
			Slot: slot, Samples: a.n, Down: a.down, Up: a.up,
			MinA: a.minA, MaxA: a.maxA, FirstUS: a.first, LastUS: a.last,
		}
		if fn := float64(a.n); fn >= 2 {
			if den := fn*a.sxx - a.sx*a.sx; den != 0 {
				c.Slope = (fn*a.sxy - a.sx*a.sy) / den
			}
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Samples != out[j].Samples {
			return out[i].Samples > out[j].Samples
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}
