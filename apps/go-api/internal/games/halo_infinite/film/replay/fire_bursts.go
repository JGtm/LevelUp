package replay

// fire_bursts.go — LA PUBLICATION DU TIR CONTINU : une rafale lue dans la vue de controle devient
// une rafale posee sur l arme qui l a tiree (lot M4b de la campagne « retours rejeu », 2026-09-24).
//
// # CE QUE LA LECTURE DONNE, ET CE QUI EN FAIT UNE RAFALE PUBLIEE
//
// La grammaire rend, par joueur et par bit de tir, l intervalle ou la gachette est TENUE, ses trous
// et ses bornes (`internal/grammar/tir_continu.go`). L index de la rafale est la PLACE : le tireur
// est l occupant de la place a cet instant (`tirs_par_place.go`). L arme se decide ensuite, et
// rien n y est devine :
//
//	EMBARQUE   l episode d occupation du tireur a la premiere frame (meme porte que les tirs de
//	           vehicule, `vehicle_shots.go`) nomme la monture ; l arme est celle que son chassis —
//	           ou la piece montee dont l episode vient — declare (`tir_continu_armes.go`). La
//	           rafale se pose sur le PORTEUR et se borne a la fin de l episode ;
//	A PIED     l arme est celle de l EMPLACEMENT que le bloc d action designe (`Arme` de la main 0,
//	           l emplacement degaine), lu dans la derniere dotation du slot avant la rafale,
//	           corrigee des prises et lachers dates qui la suivent ; la rafale se borne a la vie.
//
// Seule la GACHETTE PRINCIPALE de la main 0 est publiee : c est le bit que les armes a tir continu
// posent (1 050 entrees sur 1 050 a 81c02726, sonde P1-S3). Une rafale sur un autre bit, une monture
// sans arme a tir continu (klaxon, mortier) ou une arme en main a CHARGE (pistolet a plasma,
// Ravageur : la gachette tenue sans tir, le numero de tir ne bouge pas) est comptee, jamais
// dessinee en tir.
//
// PUR : aucune I/O.

import (
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// fireBurstBoard est ce que la publication consulte pour UNE rafale.
type fireBurstBoard struct {
	vehicules vehicleShotBoard
	slotsOf   map[int][]uint32
	places    tireursParPlace
	vies      map[uint32][][2]int // fenetres [premiere, derniere frame] des pistes publiees, par slot
	dotations map[uint32][]Loadout
	prises    map[uint32][]WeaponChange
	cov       *ContinuousFireCoverage
}

// buildFireBursts publie les rafales et rend leur couverture. Nil sans aucune lecture.
func buildFireBursts(doc *ReplayDocument, in []types.ContinuousFireBurst, st types.ContinuousFireStats,
	owner map[uint32]int, clock replayClock) ([]FireBurst, *ContinuousFireCoverage) {
	if !st.Scanned {
		return nil, nil
	}
	cov := coverageFromStats(st)
	b := fireBurstBoard{
		vehicules: vehicleShotBoard{rides: vehicleRidesByOccupant(doc.Vehicles), tracks: doc.Vehicles,
			lives: vehicleLifeIndex(doc.Vehicles), clock: clock},
		slotsOf: vehicleSlotsByPlayer(owner), places: nouveauxTireursParPlace(doc.Roster),
		vies: fenetresDesPistes(doc.Tracks), dotations: dotationsParSlot(doc.Loadouts),
		prises: prisesParSlot(doc.WeaponChanges), cov: cov,
	}
	var out []FireBurst
	for _, r := range in {
		cov.BurstsRead++
		if f, ok := b.publier(r); ok {
			out = append(out, f)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T0 != out[j].T0 {
			return out[i].T0 < out[j].T0
		}
		return out[i].Slot < out[j].Slot
	})
	if !cov.balanced() {
		slog.Error("rejeu : couverture du tir continu desequilibree", "lues", cov.BurstsRead,
			"publiees", cov.Published)
	}
	return out, cov
}

// publier decide du sort d UNE rafale lue et la compte.
func (b fireBurstBoard) publier(r types.ContinuousFireBurst) (FireBurst, bool) {
	if r.Hand != 0 || r.Barrel || r.Input != 0 {
		b.cov.OtherInput++
		return FireBurst{}, false
	}
	clock := b.vehicules.clock
	f := FireBurst{T0: clock.frame(r.StartUS), T1: clock.frame(r.EndUS),
		StartBound: r.StartBound, EndBound: r.EndBound}
	idx, parPlace := b.places.tireur(r.FilmIndex, f.T0)
	slots := b.slotsOf[idx]
	if len(slots) == 0 {
		b.cov.NoPlayer++
		return FireBurst{}, false
	}
	cand := vehicleShotCandidates(b.vehicules.rides, slots, f.T0)
	for _, c := range cand[min(1, len(cand)):] {
		if c.track != cand[0].track {
			b.cov.Ambiguous++
			return FireBurst{}, false
		}
	}
	var ok bool
	if len(cand) > 0 {
		ok = b.surLaMonture(&f, r, cand[0])
	} else {
		ok = b.aPied(&f, r, slots)
	}
	if !ok {
		return FireBurst{}, false
	}
	b.poserLesTrous(&f, r.Holes)
	if parPlace {
		b.cov.ByPlace++
	}
	b.cov.Published++
	b.cov.Shots += coupsDeLaRafale(f, int(clock.step/1000)) //nolint:gosec // pas de la grille, en µs
	return f, true
}

// surLaMonture pose une rafale sur le vehicule de l episode `pick`, avec l arme de sa monture. Un
// PASSAGER (siege lu > 0, hors piece montee) tire son arme en main : la rafale suit la regle a pied,
// posee sur le vehicule.
func (b fireBurstBoard) surLaMonture(f *FireBurst, r types.ContinuousFireBurst, pick vehicleShotRide) bool {
	holder, _ := b.vehicules.shotHolder(pick)
	v := b.vehicules.tracks[holder].Slot
	if tag, ok := b.armeDeLaMonture(pick); ok {
		b.armer(f, VehicleWeaponKey(tag), continuousWeaponsByTag[tag])
		b.borner(f, pick.ride.T1)
		if f.T1 < f.T0 {
			b.cov.Empty++
			return false
		}
		f.Slot, f.Vehicle = pick.ride.Slot, &v
		b.cov.OnVehicle++
		return true
	}
	if pick.ride.Turret != nil || pick.ride.Seat == nil || *pick.ride.Seat == 0 {
		b.cov.VehicleNoWeapon++
		return false
	}
	if !b.armeEnMain(f, r, pick.ride.Slot) {
		return false
	}
	b.borner(f, pick.ride.T1)
	if f.T1 < f.T0 {
		b.cov.Empty++
		return false
	}
	f.Slot, f.Vehicle = pick.ride.Slot, &v
	b.cov.OnFoot++
	return true
}

// armeDeLaMonture rend l arme a tir continu de la monture : celle de la piece montee dont
// l episode vient (`ride.Turret`), sinon celle du chassis de la vie de l episode — que seul son
// pilote tire (siege non lu ou siege 0).
func (b fireBurstBoard) armeDeLaMonture(pick vehicleShotRide) (uint32, bool) {
	tr := b.vehicules.tracks[pick.track]
	if pick.ride.Turret != nil {
		i, ok := b.vehicules.lives[*pick.ride.Turret]
		if !ok {
			return 0, false
		}
		return continuousWeaponOfChassis(b.vehicules.tracks[i].Chassis)
	}
	if tr.Part == "" && pick.ride.Seat != nil && *pick.ride.Seat != 0 {
		return 0, false // un passager ne tient pas l arme du chassis
	}
	return continuousWeaponOfChassis(tr.Chassis)
}

// aPied pose une rafale sur la piste du tireur qui couvre sa premiere frame.
func (b fireBurstBoard) aPied(f *FireBurst, r types.ContinuousFireBurst, slots []uint32) bool {
	slot, fin, ok := b.pisteA(slots, f.T0)
	if !ok {
		b.cov.NoTrack++
		return false
	}
	if !b.armeEnMain(f, r, slot) {
		return false
	}
	b.borner(f, fin)
	if f.T1 < f.T0 {
		b.cov.Empty++
		return false
	}
	f.Slot = slot
	b.cov.OnFoot++
	return true
}

// armeEnMain arme la rafale de l arme en main du slot, ou la compte ecartee.
func (b fireBurstBoard) armeEnMain(f *FireBurst, r types.ContinuousFireBurst, slot uint32) bool {
	fam, ok := b.familleEnMain(slot, f.T0, r.Weapon)
	if !ok {
		b.cov.WeaponUnknown++
		return false
	}
	cw, ok := continuousWeaponsByTag[fam]
	if !ok {
		b.cov.NotContinuous++
		return false
	}
	b.armer(f, formatFamille(fam), cw)
	return true
}

// armer pose la cle d arme et sa cadence.
func (b fireBurstBoard) armer(f *FireBurst, cle string, cw continuousWeapon) {
	f.Weapon, f.Rate, f.Rate0, f.Ramp = cle, cw.rate, cw.rate0, cw.ramp
}

// borner ramene la fin de la rafale a `fin` (fin de l episode ou de la vie), et le compte.
func (b fireBurstBoard) borner(f *FireBurst, fin int) {
	if f.T1 > fin {
		f.T1 = fin
		b.cov.ClippedToMount++
	}
}

// pisteA rend le slot du joueur dont la piste publiee couvre la frame, et sa derniere frame.
func (b fireBurstBoard) pisteA(slots []uint32, fr int) (uint32, int, bool) {
	for _, s := range slots {
		for _, w := range b.vies[s] {
			if fr >= w[0] && fr <= w[1] {
				return s, w[1], true
			}
		}
	}
	return 0, 0, false
}

// familleEnMain rend la famille d arme de l emplacement `arme` du slot a la frame : la derniere
// dotation lue avant la frame, corrigee des prises datees qui la suivent. L emplacement est celui
// que le bloc d action ecrit (0 ou 1, l ordre de `Loadout.W`, la meme convention que
// `equippedWeapons` cote client) ; une sentinelle ou une absence n arme rien.
func (b fireBurstBoard) familleEnMain(slot uint32, fr, arme int) (uint32, bool) {
	if arme < 0 {
		return 0, false
	}
	var w []string
	depuis := -1
	for _, l := range b.dotations[slot] {
		if l.T > fr {
			break
		}
		w, depuis = l.W, l.T
	}
	if w == nil {
		return 0, false
	}
	cur := ""
	if arme < len(w) {
		cur = w[arme]
	}
	for _, c := range b.prises[slot] {
		if c.T <= depuis || c.T > fr || c.K == nil || *c.K != arme {
			continue
		}
		cur = c.W
	}
	return parseHex32(cur)
}

// poserLesTrous projette les trous interieurs sur l axe de frames, bornes a la rafale : une frame
// n est muette que si le trou la COUVRE ENTIERE (debut arrondi a la frame suivante, fin a la frame
// qui le contient). Un trou plus court qu une frame — deux paquets lus de part et d autre, dans la
// meme frame — ne rend rien muet : la frame porte une lecture. Les trous qui se touchent fusionnent.
func (b fireBurstBoard) poserLesTrous(f *FireBurst, holes []types.ContinuousFireHole) {
	clock := b.vehicules.clock
	for _, h := range holes {
		a, z := clock.frame(h.StartUS+clock.step-1), clock.frame(h.EndUS)
		a, z = max(a, f.T0), min(z, f.T1)
		if z <= a {
			continue
		}
		if n := len(f.Holes); n > 0 && a <= f.Holes[n-1].T1 {
			f.Holes[n-1].T1 = max(f.Holes[n-1].T1, z)
			continue
		}
		f.Holes = append(f.Holes, FireBurstHole{T0: a, T1: z})
	}
}

// coupsDeLaRafale compte les coups que la rafale porte hors de ses trous : le premier a la pose de
// la gachette, les suivants a la cadence (lineaire de rate0 a rate sur `ramp` secondes, puis
// constante). C est le compte que le client pose, frame de rejeu pour frame de rejeu.
func coupsDeLaRafale(f FireBurst, frameMS int) int {
	n := 0
	for _, t := range instantsDesCoups(f, frameMS) {
		if !dansUnTrou(f.Holes, t) {
			n++
		}
	}
	return n
}

// epsilonFrame : la tolerance de la borne de fin. Les coups s accumulent en flottants, et un coup qui
// tombe EXACTEMENT sur la derniere frame (60/s : six coups par frame) ne doit pas s en perdre par
// l arrondi. La meme que le client (`model/fireBursts.ts`, `EPSILON_FRAME`).
const epsilonFrame = 1e-6

// instantsDesCoups rend les instants des coups d une rafale, en frames (fractionnaires).
func instantsDesCoups(f FireBurst, frameMS int) []float64 {
	if f.Rate <= 0 || frameMS <= 0 {
		return nil
	}
	parFrame := float64(frameMS) / 1000
	var out []float64
	for t := float64(f.T0); t <= float64(f.T1)+epsilonFrame; {
		out = append(out, t)
		ecoule := (t - float64(f.T0)) * parFrame
		cadence := f.Rate
		if f.Rate0 > 0 && f.Ramp > 0 && ecoule < f.Ramp {
			cadence = f.Rate0 + (f.Rate-f.Rate0)*ecoule/f.Ramp
		}
		t += 1 / (cadence * parFrame)
	}
	return out
}

// dansUnTrou dit si l instant tombe dans un passage muet.
func dansUnTrou(holes []FireBurstHole, t float64) bool {
	for _, h := range holes {
		if t >= float64(h.T0) && t < float64(h.T1) {
			return true
		}
	}
	return false
}

// fenetresDesPistes rend, par slot, les fenetres [premiere, derniere frame] de ses pistes publiees.
func fenetresDesPistes(tracks []Track) map[uint32][][2]int {
	out := map[uint32][][2]int{}
	for _, t := range tracks {
		if len(t.Points) == 0 {
			continue
		}
		out[t.Slot] = append(out[t.Slot], [2]int{t.Points[0].T, t.Points[len(t.Points)-1].T})
	}
	return out
}

// dotationsParSlot indexe les dotations par slot, triees par frame.
func dotationsParSlot(ls []Loadout) map[uint32][]Loadout {
	out := map[uint32][]Loadout{}
	for _, l := range ls {
		out[l.Slot] = append(out[l.Slot], l)
	}
	for s := range out {
		sort.SliceStable(out[s], func(i, j int) bool { return out[s][i].T < out[s][j].T })
	}
	return out
}

// prisesParSlot indexe les prises et lachers dates par slot, tries par frame.
func prisesParSlot(ws []WeaponChange) map[uint32][]WeaponChange {
	out := map[uint32][]WeaponChange{}
	for _, w := range ws {
		out[w.Slot] = append(out[w.Slot], w)
	}
	for s := range out {
		sort.SliceStable(out[s], func(i, j int) bool { return out[s][i].T < out[s][j].T })
	}
	return out
}

// parseHex32 lit un identifiant de 32 bits en hexadecimal, prefixe `0x` ou non.
func parseHex32(s string) (uint32, bool) {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, false
	}
	return uint32(v), true
}

// formatFamille ecrit une famille d arme comme `Loadout.W` : `0x` + 8 chiffres majuscules.
func formatFamille(fam uint32) string {
	return "0x" + strings.ToUpper(strconv.FormatUint(uint64(fam)|1<<32, 16)[1:])
}

// coverageFromStats reprend les compteurs de la lecture.
func coverageFromStats(st types.ContinuousFireStats) *ContinuousFireCoverage {
	return &ContinuousFireCoverage{Packets: st.Packets, Reached: st.Reached, Closed: st.Closed,
		Holes: st.Holes, HoleRuns: st.HoleRuns, HolesUnlocated: st.Unlocated, HolesOpenViewB: st.OpenViewB,
		HolesOverflow: st.StopOverflow, HolesKind: st.StopKind, HolesBlockBC: st.StopBlockBC,
		HolesCap: st.StopCap, HolesNotClosing: st.NotClosing, Entries: st.Entries,
		WithAction: st.WithAction, Firing: st.Firing, BurstsWithHole: st.BurstsWithHole,
		InnerHoles: st.InnerHoles, HeldHoleMS: st.HeldHoleMS}
}

// logFireBursts journalise la publication avec ses denominateurs.
func logFireBursts(cov *ContinuousFireCoverage) {
	if cov == nil {
		return
	}
	slog.Info("rejeu : tir continu", "paquets", cov.Packets, "vueCFermee", cov.Closed,
		"trous", cov.Holes, "lues", cov.BurstsRead, "publiees", cov.Published, "vehicule", cov.OnVehicle,
		"aPied", cov.OnFoot, "autreBit", cov.OtherInput, "sansJoueur", cov.NoPlayer,
		"ambigues", cov.Ambiguous, "montureSansArme", cov.VehicleNoWeapon, "sansPiste", cov.NoTrack,
		"armeInconnue", cov.WeaponUnknown, "armeNonContinue", cov.NotContinuous, "vides", cov.Empty,
		"coups", cov.Shots, "tenuTrouS", math.Round(float64(cov.HeldHoleMS)/100)/10)
}
