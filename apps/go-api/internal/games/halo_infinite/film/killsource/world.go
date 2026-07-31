package killsource

// world.go — LE MONDE EST UNE TIMELINE, PAS UN ETAT UNIQUE.
//
// BUG MESURE, et sa correction vaut vingt morts : lier TOUS les keyframes d un coup fait ecraser
// les liaisons des premiers par le dernier. Un slot RECYCLE en cours de match (biped au debut,
// filtre de spawn a la fin) etait alors decode avec le MAUVAIS archetype sur toute la premiere
// moitie du film. Les keyframes s appliquent donc DANS L ORDRE DU TEMPS.
//
// TROIS MECANISMES, chacun mesure separement :
//
//  1. PRE-CHARGEMENT. Une entite apparue ENTRE deux keyframes n est declaree que par le keyframe
//     SUIVANT ; sans elle la marche meurt sur << DELTA slot NON LIE >>. Le monde est donc amorce
//     avec la PREMIERE declaration de chaque slot, quelle que soit sa date ; les keyframes
//     chronologiques l ecrasent des qu ils la redeclarent, donc un slot recycle reste correct.
//  2. BALAYAGE COMPLEMENTAIRE DES ANCRES. Le walker de keyframe SAUTE des ancres pourtant
//     parfaitement valides (scan a slot strictement croissant + raccourci de largeur). Le
//     balayage restreint a l archetype 35 en recupere 8 et 25 selon le film, avec ZERO
//     contradiction d archetype. Son controle de faux positif est la LOCALISATION : il cherche
//     dans tout l espace [0,8192] et ne rapporte que des slots de la plage bipede (~1.2 % de
//     l espace). Sans la restriction d archetype il est inexploitable (3 480 slots, 1 705
//     contradictions).
//  3. FENETRES DE VIE. Les slots de biped sont alloues dans l ORDRE CROISSANT au fil du match, et
//     un keyframe declare l ensemble COMPLET des entites vivantes a son instant. Un slot que
//     personne ne declare est donc ne APRES le dernier keyframe dont l allocation ne l avait pas
//     atteint, et mort AVANT le suivant. La fenetre se calcule sur les SEULS keyframes, alors que
//     l instant qu elle debloque vient du KILL-FEED : c est un test FALSIFIABLE, pas un reglage.
//     Verifie 7/7, p ~ 4e-11. La variante naive (<< combler la plage >>) a ete FALSIFIEE : +8
//     dechets, 79 -> 78 morts.

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// keyframeEvent : un paquet keyframe decode, avec son horodatage.
type keyframeEvent struct {
	ts   uint64
	recs []filmdec.KeyframeRec
}

// timeline : suite chronologique des keyframes du film.
type timeline struct {
	events   []keyframeEvent
	cursor   int
	w        *filmdec.World
	gapOpen  map[int][]int
	gapClose map[int][]int
	initSnap filmdec.WorldSnapshot
}

// newTimeline : registre depuis le chunk 0 + keyframes tries par horodatage.
func newTimeline(f *film) (*timeline, error) {
	if len(f.chunks) == 0 {
		return nil, ErrNoChunk
	}
	reg, err := filmdec.ParseRegistryChunk(f.chunks[0])
	if err != nil {
		return nil, errRegistry(err)
	}
	tl := &timeline{}
	for i := range f.packets {
		if f.packets[i].typ == packetTypeKeyframe {
			tl.events = append(tl.events, keyframeEvent{f.packets[i].ts, keyframeRecs(f.packets[i].payload)})
		}
	}
	sort.Slice(tl.events, func(i, j int) bool { return tl.events[i].ts < tl.events[j].ts })
	tl.w = filmdec.NewWorld(reg)
	tl.preload()
	tl.buildGapWindows()
	tl.initSnap = tl.w.Snapshot()
	return tl, nil
}

// preload : mecanisme 1 — la PREMIERE declaration de chaque slot, quelle que soit sa date.
func (tl *timeline) preload() {
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
}

// rewind : remet la timeline a son etat initial. La calibration la parcourt une premiere fois.
func (tl *timeline) rewind() {
	tl.w.Restore(tl.initSnap)
	tl.cursor = 0
}

// advanceTo : applique tous les keyframes d horodatage <= ts. LES APPELS DOIVENT ETRE
// STRICTEMENT CROISSANTS — c est un curseur, pas une recherche.
func (tl *timeline) advanceTo(ts uint64) *filmdec.World {
	for tl.cursor < len(tl.events) && tl.events[tl.cursor].ts <= ts {
		for _, r := range tl.events[tl.cursor].recs {
			tl.w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
		}
		// fermeture d abord (le slot est mort AVANT ce keyframe, qui ne le declare pas),
		// puis ouverture (il nait APRES ce keyframe).
		for _, s := range tl.gapClose[tl.cursor] {
			tl.w.Unbind(uint32(s))
		}
		for _, s := range tl.gapOpen[tl.cursor] {
			tl.w.BindWildcard(uint32(s), bipedArchetype)
		}
		tl.cursor++
	}
	return tl.w
}

// bipedArchetype : l archetype du spartan. C est le SEUL a porter la forme lourde du dead-state
// (victime, tueur, categorie) ; les autres ne lisent qu un bit Mort.
const bipedArchetype = 35

// bipedRange : plage des slots que les keyframes declarent avec l archetype biped.
//
// ELLE SE DERIVE DU FILM, elle n est PAS codee en dur. La valeur historique [512,610] venait des
// quatre films a 8 joueurs et n a aucune justification structurelle ; derivee elle vaut
// [512,610] / [512,609] / [512,614] / [512,615] selon le film et rend +4 morts avec ZERO dechet
// (les quatre dead-states gagnes sont tous des couples justes). Elle devient OBLIGATOIRE des que
// le roster n est plus a 8.
func (tl *timeline) bipedRange() (lo, hi int) {
	lo, hi = 1<<30, -1
	for _, e := range tl.events {
		for _, r := range e.recs {
			if r.TI != bipedArchetype {
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

// buildGapWindows : mecanisme 3 — les fenetres de vie des slots de biped non declares.
func (tl *timeline) buildGapWindows() {
	tl.gapOpen, tl.gapClose = map[int][]int{}, map[int][]int{}
	for _, wn := range tl.gapWindows() {
		for i := range tl.events {
			if tl.events[i].ts == wn.from {
				tl.gapOpen[i] = append(tl.gapOpen[i], wn.slot)
			}
			if tl.events[i].ts == wn.to {
				tl.gapClose[i] = append(tl.gapClose[i], wn.slot)
			}
		}
	}
}

// gapWindow : intervalle d horodatage (exclusif) ou un slot non declare a pu vivre.
type gapWindow struct {
	slot     int
	from, to uint64
}

// gapWindows : fenetres de vie des slots de la plage bipede qu aucun keyframe ne declare.
func (tl *timeline) gapWindows() []gapWindow {
	declared, maxAt, lo, hi := tl.declaredBipeds()
	if hi < lo {
		return nil
	}
	var wins []gapWindow
	for s := lo; s <= hi; s++ {
		if declared[s] {
			continue
		}
		last := -1 // dernier keyframe dont l allocation n a pas encore atteint s
		for i := range tl.events {
			if maxAt[i] > 0 && maxAt[i] < s {
				last = i
			}
		}
		var from uint64
		if last >= 0 {
			from = tl.events[last].ts
		}
		to := ^uint64(0)
		if last+1 < len(tl.events) {
			to = tl.events[last+1].ts
		}
		wins = append(wins, gapWindow{s, from, to})
	}
	return wins
}

// declaredBipeds : slots de biped declares, plus grand slot declare par chaque keyframe, et la
// plage observee.
func (tl *timeline) declaredBipeds() (map[int]bool, []int, int, int) {
	declared := map[int]bool{}
	maxAt := make([]int, len(tl.events))
	lo, hi := 1<<30, -1
	for i, e := range tl.events {
		for _, r := range e.recs {
			if r.TI != bipedArchetype {
				continue
			}
			declared[r.Slot] = true
			if r.Slot < lo {
				lo = r.Slot
			}
			if r.Slot > hi {
				hi = r.Slot
			}
			if r.Slot > maxAt[i] {
				maxAt[i] = r.Slot
			}
		}
	}
	return declared, maxAt, lo, hi
}

// keyframeRecs : mecanisme 2 — sortie du walker de keyframe COMPLETEE par le balayage restreint
// aux ancres d archetype biped que le walker a sautees.
func keyframeRecs(pl []byte) []filmdec.KeyframeRec {
	recs := filmdec.WalkKeyframeWorld(pl)
	have := map[int]bool{}
	for _, r := range recs {
		have[r.Slot] = true
	}
	for _, a := range sweepKeyframe(pl, bipedArchetype) {
		if have[a.slot] {
			continue
		}
		have[a.slot] = true
		recs = append(recs, filmdec.KeyframeRec{Slot: a.slot, TI: a.ti, Gen: a.gen, Bit: a.bit})
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	return recs
}

// anchor : une ancre de table de keyframe reconnue par balayage exhaustif.
type anchor struct{ slot, ti, gen, bit int }

// sweepKeyframe : toutes les ancres valides d un payload keyframe, au sens exact du walker moins
// la contrainte de croissance des slots, restreintes a l archetype `onlyTI`.
//
//	id != 0xFFFFFFFF (sentinelle) et gen = id>>30 != 0
//	slot = id & 0x3FFFFFFF < 8192
//	le mot 32 bits a q+32 vaut < 50  (champ nul ET archetype < 50)
func sweepKeyframe(buf []byte, onlyTI int) []anchor {
	total := len(buf) * 8
	var out []anchor
	for q := 0; q+64 <= total; q++ {
		id := bits32(buf, q)
		if id == 0xFFFFFFFF || id>>30 == 0 {
			continue
		}
		slot := int(id & 0x3FFFFFFF)
		if slot >= 8192 || bits32(buf, q+32) >= 50 {
			continue
		}
		ti := bitsN(buf, q+58, 6)
		if onlyTI >= 0 && ti != onlyTI {
			continue
		}
		out = append(out, anchor{slot, ti, int(id >> 30), q})
	}
	return out
}
