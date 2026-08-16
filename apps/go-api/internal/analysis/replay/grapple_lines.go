package replay

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// grapple_lines.go — LA TRACTION DE GRAPPIN, datée PAR VIE sur l'axe du rejeu, avec son
// POINT D'ACCROCHE en coordonnées monde.
//
// D'OÙ VIENT LA DONNÉE (2026-08-16, PLAN_GRAPPIN_LIGNE, gate 0 passé sur 3 films /
// 3 cartes) : le corps tag==3 d'i59 transmet, par usage de grappin, une PAIRE de lectures
// à 0,150 s — le tir (corps léger) puis l'accroche (corps lourd) — portant chacune la
// position absolue de l'ancre, quantifiée aux largeurs d'axe de la carte. L'ancre est un
// point monde FIXE (|P1-P2| médian 0,05-0,07 u entre membres d'une paire) vers lequel la
// trajectoire du porteur CONVERGE après l'accroche (contrôle (b) du plan, témoins
// mélangés effondrés).
//
// LA FENÊTRE EST MESURÉE, PAS CHOISIE (règle 1.1 du plan) : elle s'ouvre au TIR (ou à
// l'accroche quand le tir n'a pas été lu) et se ferme à l'ARRIVÉE — l'instant, propre à
// chaque traction, où la distance entre la trajectoire du porteur et l'ancre atteint son
// minimum. La mesure du gate 0 borne la recherche : la distance médiane atteint son
// minimum ~1 s après l'accroche et REMONTE à t+2 s (le joueur repart) — chercher au-delà
// de grapplePullCapUS daterait le passage suivant, pas la traction.
//
// UN TIR SANS ACCROCHE EST UN RATÉ : il est compté (UnpairedFires), jamais tracé — la
// traction n'existe que si l'accroche l'atteste. Une accroche sans tir lu est une
// traction dont la fenêtre commence à l'accroche : on ne recule pas un début non lu.

// grapplePairGapUS : au-delà de cet écart, un tir et une accroche du même slot ne forment
// plus une paire. L'écart mesuré est de 0,150 s CONSTANT (p10 = p50, 3 films) ; la borne
// large absorbe la retransmission sans risquer d'apparier deux usages distincts (les
// usages successifs d'une même vie sont à plusieurs secondes).
const grapplePairGapUS = 500_000

// grapplePullCapUS borne la recherche de l'arrivée après l'accroche (cf. en-tête).
const grapplePullCapUS = 2_500_000

// GrappleLine est UNE traction de grappin : la fenêtre datée et le point d'accroche.
type GrappleLine struct {
	// Slot désigne la Track concernée — donc une VIE, pas un joueur (même règle que les
	// autres calques : le slot migre aux réapparitions).
	Slot uint32 `json:"slot"`
	// T0 / T1 bornent la traction sur le même axe que Point.T : du tir (ou de l'accroche
	// si le tir n'a pas été lu) à l'ARRIVÉE mesurée sur la trajectoire de la vie.
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// AX / AY : le point d'accroche en coordonnées monde (mêmes axes que Point.X/Y).
	// C'est la position du corps LOURD (l'accroche) : la plus précise des deux lectures
	// de la paire — l'ancre y est fixée.
	AX float32 `json:"ax"`
	AY float32 `json:"ay"`
	// AZ : l'altitude de l'ancre. Gratuite (le même champ la porte), publiée pour la
	// cohérence d'étage — le rendu 2D ne l'exige pas. PIÈGE omitempty accepté : la
	// déquantification à mi-bucket (min + step*(q+0,5)) rend un zéro exact hors
	// d'atteinte en pratique (même argument que Bounds.MinZ).
	AZ float32 `json:"az,omitempty"`
}

// GrappleCoverage dit ce que le calque a lu et ce qu'il en a publié — le dénominateur
// sans lequel « N tractions » ne se juge pas.
type GrappleCoverage struct {
	// LightReads / HeavyReads : lectures de corps léger (tir) et lourd (accroche).
	LightReads int `json:"lightReads"`
	HeavyReads int `json:"heavyReads"`
	// Pulls : tractions publiées. PullLives : vies en portant au moins une.
	Pulls     int `json:"pulls"`
	PullLives int `json:"pullLives"`
	// UnpairedFires : tirs sans accroche (ratés) — comptés, jamais tracés.
	UnpairedFires int `json:"unpairedFires"`
	// BrokenBodies : corps tag==3 non décodables (grammaire non établie, cf.
	// components_biped_anchor.go) — comptés, jamais devinés.
	BrokenBodies int `json:"brokenBodies"`
}

// buildGrappleLines assemble les tractions : appariement tir->accroche par vie, ancre
// déquantifiée aux bornes de la carte, arrivée mesurée sur la trajectoire PUBLIÉE.
func buildGrappleLines(reads []filmdec.GrappleRead, entry filmdec.MapQuantEntry,
	origin, step uint64, tracks []Track) ([]GrappleLine, *GrappleCoverage) {
	cov := &GrappleCoverage{}
	if len(reads) == 0 || step == 0 {
		return nil, cov
	}
	byTrack := make(map[uint32]*Track, len(tracks))
	for i := range tracks {
		byTrack[tracks[i].Slot] = &tracks[i]
	}
	bySlot := map[uint32][]filmdec.GrappleRead{}
	for _, r := range reads {
		if r.Heavy {
			cov.HeavyReads++
		} else {
			cov.LightReads++
		}
		bySlot[r.Slot] = append(bySlot[r.Slot], r)
	}
	slots := make([]uint32, 0, len(bySlot))
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []GrappleLine
	lives := map[uint32]struct{}{}
	for _, s := range slots {
		list := bySlot[s]
		sort.SliceStable(list, func(i, j int) bool { return list[i].TimestampUS < list[j].TimestampUS })
		for _, l := range grappleLinesOfLife(list, entry, origin, step, byTrack[s], cov) {
			out = append(out, l)
			lives[s] = struct{}{}
		}
	}
	cov.Pulls, cov.PullLives = len(out), len(lives)
	if len(out) == 0 {
		return nil, cov
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T0 != out[j].T0 {
			return out[i].T0 < out[j].T0
		}
		return out[i].Slot < out[j].Slot
	})
	return out, cov
}

// grappleLinesOfLife déroule les lectures TRIÉES d'une vie : chaque accroche clôt une
// traction (fenêtre ouverte au tir apparié, sinon à l'accroche), chaque tir resté sans
// accroche est un raté compté.
func grappleLinesOfLife(list []filmdec.GrappleRead, entry filmdec.MapQuantEntry,
	origin, step uint64, track *Track, cov *GrappleCoverage) []GrappleLine {
	var out []GrappleLine
	pendingFire := -1
	for i, r := range list {
		if !r.Heavy {
			if pendingFire >= 0 {
				cov.UnpairedFires++ // le tir précédent n'a jamais accroché
			}
			pendingFire = i
			continue
		}
		startUS := r.TimestampUS
		if pendingFire >= 0 {
			if r.TimestampUS-list[pendingFire].TimestampUS <= grapplePairGapUS {
				startUS = list[pendingFire].TimestampUS
			} else {
				cov.UnpairedFires++ // trop vieux pour être le tir de cette accroche
			}
			pendingFire = -1
		}
		if l, ok := grappleLine(r, startUS, entry, origin, step, track); ok {
			out = append(out, l)
		}
	}
	if pendingFire >= 0 {
		cov.UnpairedFires++
	}
	return out
}

// grappleLine construit UNE traction : ancre déquantifiée, fenêtre [tir, arrivée] bornée
// à la fenêtre de la vie publiée. ok=false si la vie n'est pas publiée ou si la fenêtre
// est vide (mort à l'accroche) : rien à tracer.
func grappleLine(r filmdec.GrappleRead, startUS uint64, entry filmdec.MapQuantEntry,
	origin, step uint64, track *Track) (GrappleLine, bool) {
	if track == nil {
		return GrappleLine{}, false // vie non publiée : aucune fiche où poser la traction
	}
	lay := filmdec.I0Layout{AxisW: entry.AxisWidths}
	wr := entry.Range()
	ax := filmdec.DequantBipedAxis(r.PosQ[0], 0, lay, wr)
	ay := filmdec.DequantBipedAxis(r.PosQ[1], 1, lay, wr)
	az := filmdec.DequantBipedAxis(r.PosQ[2], 2, lay, wr)
	t0 := frameOf(startUS, origin, step)
	tAttach := frameOf(r.TimestampUS, origin, step)
	t1 := grappleArrival(track, tAttach, int(grapplePullCapUS/step), ax, ay, az)
	if t0 < track.StartFrame {
		t0 = track.StartFrame
	}
	if t1 > track.EndFrame {
		t1 = track.EndFrame
	}
	if t1 <= t0 {
		return GrappleLine{}, false // mort à l'accroche ou fenêtre hors de la vie publiée
	}
	return GrappleLine{Slot: r.Slot, T0: t0, T1: t1, AX: round2(ax), AY: round2(ay), AZ: round2(az)}, true
}

// grappleArrival rend la frame d'ARRIVÉE : celle, entre l'accroche et la borne de
// recherche, où la trajectoire passe au plus près de l'ancre. Sans point dans la fenêtre
// (fin de vie), l'accroche elle-même est rendue — la fenêtre vide sera écartée en amont.
func grappleArrival(track *Track, tAttach, capFrames int, ax, ay, az float32) int {
	best, bestD := tAttach, float32(-1)
	for _, p := range track.Points {
		if p.T < tAttach || p.T > tAttach+capFrames {
			continue
		}
		dx, dy, dz := p.X-ax, p.Y-ay, p.Z-az
		d := dx*dx + dy*dy + dz*dz
		if bestD < 0 || d < bestD {
			best, bestD = p.T, d
		}
	}
	return best
}
