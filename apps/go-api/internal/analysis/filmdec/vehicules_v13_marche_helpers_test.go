package filmdec

// vehicules_v13_marche_helpers_test.go — LA MARCHE, portee sur la branche vehicules (lot V13).
//
// POURQUOI. Le lot V10 a etabli que la grammaire du dead-state de `ti=40` est PROUVEE au
// deserialiseur (FUN_140c1dce0 : corps lourd commun a 0x23 et 0x28), et que le verrou est
// ailleurs : le balayage ANCRE (`ScanFilmBipedPositionsForBand`) n'accepte qu'un record dont le
// masque commence par un `i0` ABSOLU, donc il n'atteint JAMAIS `i11` — y compris sur le bipede,
// dont les morts sont un fait du corpus (temoin `v10ControlBiped`).
//
// Il existe un lecteur qui ne passe PAS par l'ancre : la marche du decodeur de source de degat
// (`internal/games/halo_infinite/film/killsource`, gate (b) 98,2 %). Elle a fait passer un film
// de 5 a 26 morts detectees. Ce fichier la porte ici, avec les primitives EXPORTEES de filmdec —
// c'est le MEME algorithme, cablant les memes fonctions (World.Snapshot/Restore/
// GenerationMatches, TryDeltaAt, DecodeFrameRecords). Quatre elements font sa robustesse :
//
//  1. TIMELINE CHRONOLOGIQUE : un monde unique, amorce par la PREMIERE declaration de chaque
//     slot, puis les images-cles appliquees DANS L'ORDRE DU TEMPS (un slot recycle n'est pas
//     ecrase par la derniere image-cle).
//  2. LOCALISATEUR D'EVENTS : dans un paquet a events la boucle de records ne commence pas a
//     l'amorce ; signature stricte = premier delta slot 123 de 35 bits a un bit precede d'un 0,
//     puis repli largeur libre. C'est le correctif qui debloque les morts.
//  3. HUIT VUES DE REPLICATION par paquet : une mort peut vivre dans une vue > 0.
//  4. SNAPSHOT/RESTORE + filtre `DesyncAt == -1` : une marche desynchronisee ne laisse pas de
//     liaison derriere elle, et seuls les records PROPRES sont retenus.
//
// DIFFERENCE AVEC LE PORTAGE DE `feat/v75` : aucun filtre de bande n'est applique ici. La marche
// recolte TOUS les dead-states propres et les range par archetype APRES coup — c'est ce qui rend
// la non-regression bipede structurelle (gate G1 du GATE_V13).
//
// Lecture seule, aucun code de production touche.

import (
	"sort"
)

// v13BipedTI / v13VehicleTI : les deux archetypes compares. VehicleTypeIndex vaut deja 40.
const v13BipedTI uint32 = 35

// v13Views : vues de replication marchees par paquet. Defaut killsource = 8.
const v13Views = 8

// v13KF : une image-cle decodee, avec son horodatage.
type v13KF struct {
	ts   uint64
	recs []KeyframeRec
}

// v13Timeline : suite chronologique des images-cles (preload + curseur).
type v13Timeline struct {
	events []v13KF
	cursor int
	w      *World
}

// newV13Timeline amorce le monde par la PREMIERE declaration de chaque slot, images-cles triees.
func newV13Timeline(reg *Registry, kfs []v13KF) *v13Timeline {
	tl := &v13Timeline{w: NewWorld(reg), events: kfs}
	sort.Slice(tl.events, func(i, j int) bool { return tl.events[i].ts < tl.events[j].ts })
	seen := map[int]bool{}
	for _, e := range tl.events {
		for _, r := range e.recs {
			if seen[r.Slot] {
				continue
			}
			seen[r.Slot] = true
			tl.w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
		}
	}
	return tl
}

// advanceTo applique toutes les images-cles d'horodatage <= ts. LES APPELS DOIVENT CROITRE.
func (tl *v13Timeline) advanceTo(ts uint64) *World {
	for tl.cursor < len(tl.events) && tl.events[tl.cursor].ts <= ts {
		for _, r := range tl.events[tl.cursor].recs {
			tl.w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
		}
		tl.cursor++
	}
	return tl.w
}

// v13DeclaredSlots : les slots que les images-cles declarent avec l'archetype `ti`, et la bande
// [lo, hi] qu'ils occupent. La bande est DERIVEE du film, jamais une constante en dur.
func v13DeclaredSlots(kfs []v13KF, ti uint32) (slots map[int]bool, lo, hi int) {
	slots, lo, hi = map[int]bool{}, 1<<30, -1
	for _, e := range kfs {
		for _, r := range e.recs {
			if uint32(r.TI) != ti {
				continue
			}
			slots[r.Slot] = true
			if r.Slot < lo {
				lo = r.Slot
			}
			if r.Slot > hi {
				hi = r.Slot
			}
		}
	}
	return slots, lo, hi
}

// v13Dead : un dead-state PROPRE atteint par la marche, avec l'archetype de son porteur.
type v13Dead struct {
	ts     uint64
	slot   uint32
	ti     uint32
	enumA  int32  // victime (index roster) chez le bipede
	enumB  int32  // tueur (index roster) chez le bipede
	val0c  uint8  // categorie
	gid    uint32 // global-id de la source (0xFFFFFFFF = absent)
	hasRef bool
	// tailDesync : le record a rompu APRES le dead-state (queue inconnue, dead-state lu au bon
	// endroit). Compte a part pour que la mesure ne melange jamais les deux qualites.
	tailDesync bool
}

// v13Stats : les denominateurs du gate G3. AUCUN compte de dead-state ne se publie sans eux.
type v13Stats struct {
	recTotal  map[uint32]int          // archetype -> records marches
	recClean  map[uint32]int          // archetype -> records dont DesyncAt == -1
	slotsSeen map[uint32]map[int]bool // archetype -> slots atteints proprement
	pkTotal   int                     // paquets delta marches
	pkEvents  int                     // dont paquets a liste d'evenements
	pkLocated int                     // dont paquets a events effectivement localises
	// maskDead / maskDeadDirty : records dont le MASQUE declare le composant dead-state, tous
	// et parmi les DESYNCHRONISES. C'est le controle qui separe « le vehicule ne meurt pas dans
	// le film » de « ses morts sont dans la fraction desynchronisee qu'on jette ».
	maskDead      map[uint32]int
	maskDeadDirty map[uint32]int
	// desyncAt : histogramme de la position de rupture, desyncMask : masques concernes.
	desyncAt   map[uint32]map[int]int
	desyncMask map[uint32]map[uint64]int
	// deadIdx : index du composant dead-state dans chaque archetype (-1 s'il n'en porte pas).
	deadIdx map[uint32]int
	reg     *Registry
}

func newV13Stats(reg *Registry) *v13Stats {
	return &v13Stats{recTotal: map[uint32]int{}, recClean: map[uint32]int{},
		slotsSeen: map[uint32]map[int]bool{}, maskDead: map[uint32]int{},
		maskDeadDirty: map[uint32]int{}, deadIdx: map[uint32]int{}, reg: reg,
		desyncAt: map[uint32]map[int]int{}, desyncMask: map[uint32]map[uint64]int{}}
}

// deadStateIndex rend l'index du composant `object-dead-state-component` dans l'archetype `ti`,
// resolu PAR LE NOM du registre (jamais un indice en dur), ou -1.
func (s *v13Stats) deadStateIndex(ti uint32) int {
	if i, ok := s.deadIdx[ti]; ok {
		return i
	}
	idx := -1
	if a, ok := s.reg.Archetype(int(ti)); ok {
		for i, c := range a.Components {
			if c == "object-dead-state-component" {
				idx = i
				break
			}
		}
	}
	s.deadIdx[ti] = idx
	return idx
}

func (s *v13Stats) note(r *FrameRecord) {
	s.recTotal[r.TypeIndex]++
	if di := s.deadStateIndex(r.TypeIndex); di >= 0 && di < 64 && r.Trace.Mask&(1<<uint(di)) != 0 {
		s.maskDead[r.TypeIndex]++
		if r.DesyncAt != -1 {
			s.maskDeadDirty[r.TypeIndex]++
			if s.desyncAt[r.TypeIndex] == nil {
				s.desyncAt[r.TypeIndex] = map[int]int{}
			}
			s.desyncAt[r.TypeIndex][r.DesyncAt]++
			if s.desyncMask[r.TypeIndex] == nil {
				s.desyncMask[r.TypeIndex] = map[uint64]int{}
			}
			s.desyncMask[r.TypeIndex][r.Trace.Mask]++
		}
	}
	if r.DesyncAt != -1 {
		return
	}
	s.recClean[r.TypeIndex]++
	if s.slotsSeen[r.TypeIndex] == nil {
		s.slotsSeen[r.TypeIndex] = map[int]bool{}
	}
	s.slotsSeen[r.TypeIndex][int(r.Slot)] = true
}

// v13HasEvents : le paquet porte-t-il une liste d'evenements ? Bit 1 du payload.
func v13HasEvents(pay []byte) bool { return kfBitAt(pay, 1) != 0 }

// v13Signature123 : un delta sur le slot 123 decode-t-il en `s`, finit 35 bits plus loin, avec un
// unique composant ?
func v13Signature123(pay []byte, s int, w *World, cfg FrameConfig) bool {
	rec, end, ok := TryDeltaAt(pay, s, w, cfg)
	return ok && rec.Slot == 123 && end == s+35 && len(rec.Trace.Comps) == 1
}

// v13LocateStrict : premiere position S >= 2, bit S-1 == 0, signature stricte. -1 si aucune.
func v13LocateStrict(pay []byte, w *World, cfg FrameConfig) int {
	nb := len(pay) * 8
	for s := 2; s+35 < nb; s++ {
		if kfBitAt(pay, s-1) != 0 {
			continue
		}
		if v13Signature123(pay, s, w, cfg) {
			return s
		}
	}
	return -1
}

// v13LocateFallback : meme condition, LARGEUR LIBRE, essaye seulement apres l'echec strict.
func v13LocateFallback(pay []byte, w *World, cfg FrameConfig) int {
	nb := len(pay) * 8
	for s := 2; s+16 < nb; s++ {
		if kfBitAt(pay, s-1) != 0 {
			continue
		}
		rec, _, ok := TryDeltaAt(pay, s, w, cfg)
		if !ok || rec.Slot != 123 || !w.GenerationMatches(rec.ID) {
			continue
		}
		return s
	}
	return -1
}

// v13Locate : localisateur complet — signature stricte (avec controle de generation), puis repli.
func v13Locate(pay []byte, w *World, cfg FrameConfig) int {
	if s := v13LocateStrict(pay, w, cfg); s >= 0 {
		if rec, _, ok := TryDeltaAt(pay, s, w, cfg); ok && w.GenerationMatches(rec.ID) {
			return s
		}
	}
	return v13LocateFallback(pay, w, cfg)
}

// v13HarvestPacket : snapshot, marche jusqu'a v13Views vues, restore, puis recolte TOUS les
// dead-states propres — SANS filtre de bande (c'est le point du lot). Alimente aussi les
// denominateurs de couverture.
func v13HarvestPacket(pay []byte, w *World, cfg FrameConfig, start int, ts uint64,
	st *v13Stats, out []v13Dead) []v13Dead {
	snap := w.Snapshot()
	br := NewBitReader(pay)
	br.Skip(start)
	var recs []FrameRecord
	for v := 0; v < v13Views && br.Remaining() >= 8; v++ {
		r2, err := DecodeFrameRecords(br, w, cfg)
		recs = append(recs, r2...)
		if err != nil {
			break
		}
	}
	w.Restore(snap)
	for i := range recs {
		r := &recs[i]
		st.note(r)
		if r.Trace.Dead == nil || !r.Trace.Dead.Mort {
			continue
		}
		// LE POINT DE BASCULE DU LOT. Le filtre historique `DesyncAt == -1` jette tout record
		// desynchronise. Or `DesyncAt` est l'index du PREMIER composant present NON PORTE : tout
		// ce qui precede a ete consomme dans l'ordre. Si la rupture est APRES le dead-state, les
		// bits du dead-state ont ete lus au bon endroit — seule la QUEUE du record est inconnue.
		// Mesure qui l'impose : sur `ti=40`, 65 des 69 records qui declarent le dead-state
		// rompent a i30..i35 (`vehicle-auto-turret-*`, `vehicle-type-state`), TOUS apres i11.
		// Le filtre strict jetait donc des morts de vehicule PARFAITEMENT LUES.
		di := st.deadStateIndex(r.TypeIndex)
		tail := r.DesyncAt != -1 && di >= 0 && r.DesyncAt > di
		if r.DesyncAt != -1 && !tail {
			continue
		}
		d := r.Trace.Dead
		out = append(out, v13Dead{ts: ts, slot: r.Slot, ti: r.TypeIndex, enumA: d.EnumA,
			enumB: d.EnumB, val0c: d.Val0c, gid: d.GlobalID, hasRef: d.HasRef, tailDesync: tail})
	}
	return out
}

// v13Dedup : une entite ne meurt qu'une fois a un instant donne. Plusieurs vues de replication
// peuvent republier le MEME dead-state.
func v13Dedup(in []v13Dead) []v13Dead {
	type key struct {
		slot uint32
		ts   uint64
	}
	seen := map[key]bool{}
	out := in[:0]
	for _, d := range in {
		k := key{d.slot, d.ts}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, d)
	}
	return out
}
