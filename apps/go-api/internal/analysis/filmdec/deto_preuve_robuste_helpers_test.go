package filmdec

// deto_preuve_robuste_helpers_test.go — LE SCAN DE KILLS ROBUSTE, porte dans filmdec.
//
// POURQUOI CE FICHIER EXISTE. La verite terrain de la piste de detonation (M5) etait
// data-starved : elle recoltait les morts par `geoCollectDamageKills`, qui ne decode la boucle
// de records QUE pour les paquets SANS liste d'evenements (`pay[0]&0x40 == 0`). Or les morts
// (dead-states) sont TOUTES dans les paquets A EVENTS (mesure killsource : 93/93). Ce collecteur
// sautait donc exactement les paquets porteurs de kills — d'ou 0 a 5 morts par film, 0 kill
// explosif relie.
//
// CE QUE FAIT CE FICHIER. Il reproduit, avec les primitives EXPORTEES de filmdec, LA MARCHE du
// decodeur de source de degat valide (internal/games/halo_infinite/film/killsource, gate (b)
// 98.2 %, part localisee 95.8-97.6 %). killsource importe filmdec : il ne peut PAS etre importe
// ici (cycle). On porte donc son ALGORITHME — ce n'est pas une recolte maison, c'est le meme
// scan, cablant les memes fonctions (World.Snapshot/Restore/GenerationMatches, TryDeltaAt,
// DecodeFrameRecords) que killsource/walk.go et /world.go appellent. Quatre elements que
// `geoCollectDamageKills` n'a pas et qui font la robustesse :
//
//  1. TIMELINE CHRONOLOGIQUE (killsource/world.go). Un monde unique, amorce par la PREMIERE
//     declaration de chaque slot (preload), puis les keyframes appliques DANS L'ORDRE DU TEMPS
//     (un slot recycle biped->autre n'est pas ecrase par le dernier keyframe).
//  2. LOCALISATEUR D'EVENTS (killsource/walk.go). Dans un paquet a events la boucle de records
//     ne commence pas au bit 2 : signature stricte = premier delta slot 123 de 35 bits a un bit
//     precede d'un 0, puis repli largeur libre. C'est le correctif qui debloque les morts.
//  3. HUIT VUES DE REPLICATION par paquet (Views=8) : les morts peuvent vivre dans une vue > 0.
//  4. SNAPSHOT/RESTORE + filtre DesyncAt==-1 : une marche desynchronisee ne laisse pas de
//     liaison derriere elle, et seuls les records PROPRES (dead-state lu avant toute rupture)
//     sont retenus.
//
// FILTRE DE CREDIBILITE (killsource/walk.go selectCredible) : slot dans la plage bipede DERIVEE
// des keyframes, indices EnumA/EnumB dans une borne de roster, categorie Val0c <= 9. Le garde
// FORT reste DesyncAt==-1 ; la borne de roster n'ecarte que le residuel.
//
// Le resultat alimente le MEME detoM5GroundTruth que TestDetoAttribution — on ne change que la
// source des morts. Garde LOT1_TRAME_FILM, verrou process pris par l'appelant, lecture seule.

import (
	"sort"
	"testing"
)

// rbBipedArchetype : l'archetype du spartan (TI=35), le seul a porter la forme lourde du
// dead-state (victime, tueur, categorie). Identique a killsource/world.go:bipedArchetype.
const rbBipedArchetype = 35

// rbRosterCeiling : borne haute de roster pour le filtre de credibilite. Le lobby Halo plafonne
// a 24 (BTB) ; 32 laisse une marge. Ce n'est PAS le filtre fort — DesyncAt==-1 + plage bipede le
// sont ; cette borne n'ecarte que des indices residuels d'un record faussement propre.
const rbRosterCeiling = 32

// rbViews : vues de replication marchees par paquet. DEFAUT killsource = 8.
const rbViews = 8

// robustKF : un keyframe decode, avec son horodatage.
type robustKF struct {
	ts   uint64
	recs []KeyframeRec
}

// robustTimeline : suite chronologique des keyframes (preload + curseur), portee de
// killsource/world.go (mecanismes 1 seul ; gap windows et sweep complementaire volontairement
// omis — ils ajoutent +4/+8 morts marginales, pas necessaires a l'echantillon vise).
type robustTimeline struct {
	events       []robustKF
	cursor       int
	w            *World
	initSnap     WorldSnapshot
	bipLo, bipHi int
}

// newRobustTimeline construit la timeline : preload de la premiere declaration de chaque slot,
// puis snapshot initial. Les keyframes sont tries par ts.
func newRobustTimeline(reg *Registry, kfs []robustKF) *robustTimeline {
	tl := &robustTimeline{w: NewWorld(reg), events: kfs}
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
	tl.bipLo, tl.bipHi = rbBipedRange(tl.events)
	tl.initSnap = tl.w.Snapshot()
	return tl
}

// advanceTo applique tous les keyframes d'horodatage <= ts. LES APPELS DOIVENT ETRE CROISSANTS.
func (tl *robustTimeline) advanceTo(ts uint64) *World {
	for tl.cursor < len(tl.events) && tl.events[tl.cursor].ts <= ts {
		for _, r := range tl.events[tl.cursor].recs {
			tl.w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
		}
		tl.cursor++
	}
	return tl.w
}

// rbBipedRange : plage des slots que les keyframes declarent avec l'archetype biped (TI=35).
func rbBipedRange(events []robustKF) (lo, hi int) {
	lo, hi = 1<<30, -1
	for _, e := range events {
		for _, r := range e.recs {
			if r.TI != rbBipedArchetype {
				continue
			}
			if r.Slot < lo {
				lo = r.Slot
			}
			if r.Slot > hi {
				hi = r.Slot
			}
		}
	}
	return lo, hi
}

// rbChunkPackets : un chunk decompresse et ses paquets, garde en memoire pour la deuxieme passe.
type rbChunkPackets struct {
	data []byte
	pks  []FilmPacket
}

// robustCollectKills : LA passe robuste. Charge les n chunks, construit la timeline depuis TOUS
// les keyframes, puis marche les paquets type-0 dans l'ordre du temps avec localisateur + 8 vues,
// et recolte les dead-states credibles. Rend la liste de morts (victime slot, roster EnumA,
// tueur EnumB, ts) au MEME format geoKill que geoHarvestKills.
func robustCollectKills(t *testing.T, dir string, reg *Registry, n int) []geoKill {
	t.Helper()
	chunks := make([]rbChunkPackets, 0, n)
	var kfs []robustKF
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		chunks = append(chunks, rbChunkPackets{data: data, pks: pks})
		for _, pk := range pks {
			if pk.Type == PacketTypeKeyframe {
				kfs = append(kfs, robustKF{ts: pk.TimestampUS, recs: WalkKeyframeWorld(pk.Payload(data))})
			}
		}
	}
	tl := newRobustTimeline(reg, kfs)
	cfg := DefaultFrameConfig()

	// Rassembler les paquets type-0 (delta) de tous les chunks, tries par ts (le curseur de la
	// timeline exige des appels croissants).
	type rbDelta struct {
		ts  uint64
		pay []byte
	}
	var deltas []rbDelta
	for ci := range chunks {
		for _, pk := range chunks[ci].pks {
			if pk.Type == PacketTypeDelta {
				deltas = append(deltas, rbDelta{ts: pk.TimestampUS, pay: pk.Payload(chunks[ci].data)})
			}
		}
	}
	sort.Slice(deltas, func(i, j int) bool { return deltas[i].ts < deltas[j].ts })

	var kills []geoKill
	for _, d := range deltas {
		w := tl.advanceTo(d.ts)
		start := 2
		if rbHasEvents(d.pay) {
			s := rbLocate(d.pay, w, cfg)
			if s < 0 {
				continue // paquet a events non localise : la marche n'a pas de point de depart sur.
			}
			start = s
		}
		kills = rbHarvestPacket(d.pay, w, cfg, start, tl.bipLo, tl.bipHi, d.ts, kills)
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].ts < kills[j].ts })
	return kills
}

// rbHasEvents : le paquet porte-t-il une liste d'evenements ? Bit 1 du payload (killsource).
func rbHasEvents(pay []byte) bool { return kfBitAt(pay, 1) != 0 }

// rbHarvestPacket : snapshot, marche jusqu'a rbViews vues, restore, puis recolte les dead-states
// PROPRES et credibles. Reproduit killsource/walk.go:walkPacket + selectCredible.
func rbHarvestPacket(pay []byte, w *World, cfg FrameConfig, start, bipLo, bipHi int, ts uint64, kills []geoKill) []geoKill {
	snap := w.Snapshot()
	br := NewBitReader(pay)
	br.Skip(start)
	var recs []FrameRecord
	for v := 0; v < rbViews && len(pay)*8-br.BitPos() >= 8; v++ {
		r2, err := DecodeFrameRecords(br, w, cfg)
		recs = append(recs, r2...)
		if err != nil {
			break
		}
	}
	w.Restore(snap)
	for i := range recs {
		r := &recs[i]
		if r.DesyncAt != -1 || r.Trace.Dead == nil || !r.Trace.Dead.Mort {
			continue
		}
		d := r.Trace.Dead
		if int(r.Slot) < bipLo || int(r.Slot) > bipHi {
			continue
		}
		if d.EnumA < 0 || int(d.EnumA) >= rbRosterCeiling {
			continue
		}
		if d.EnumB < 0 || int(d.EnumB) >= rbRosterCeiling {
			continue
		}
		if d.Val0c > 9 {
			continue
		}
		kills = append(kills, geoKill{victSlot: r.Slot, victRost: d.EnumA, killer: d.EnumB, ts: ts})
	}
	return kills
}

// rbSignature123 : un delta sur le slot 123 decode-t-il en `s`, finit 35 bits plus loin, avec un
// unique composant ? (killsource/walk.go:signature123)
func rbSignature123(pay []byte, s int, w *World, cfg FrameConfig) bool {
	rec, end, ok := TryDeltaAt(pay, s, w, cfg)
	return ok && rec.Slot == 123 && end == s+35 && len(rec.Trace.Comps) == 1
}

// rbLocateStrict : premiere position S>=2, bit S-1 == 0, signature stricte decode. -1 si aucune.
func rbLocateStrict(pay []byte, w *World, cfg FrameConfig) int {
	nb := len(pay) * 8
	for s := 2; s+35 < nb; s++ {
		if kfBitAt(pay, s-1) != 0 {
			continue
		}
		if rbSignature123(pay, s, w, cfg) {
			return s
		}
	}
	return -1
}

// rbLocateFallback : meme condition, LARGEUR LIBRE, essaye seulement apres l'echec strict.
func rbLocateFallback(pay []byte, w *World, cfg FrameConfig) int {
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

// rbLocate : localisateur complet — signature stricte (avec controle de generation), puis repli.
func rbLocate(pay []byte, w *World, cfg FrameConfig) int {
	if s := rbLocateStrict(pay, w, cfg); s >= 0 {
		if rec, _, ok := TryDeltaAt(pay, s, w, cfg); ok && w.GenerationMatches(rec.ID) {
			return s
		}
	}
	return rbLocateFallback(pay, w, cfg)
}
