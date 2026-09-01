package replay

// bombe_b3_desaccords_test.go — LES QUATRE DÉSACCORDS DE V1, DÉPARTAGÉS PAR LA POSITION.
//
// # LA QUESTION
//
// B2-V1 a rendu 13/17 accords : sur quatre explosions, le canal des armes tenues désigne un
// poseur, le pont statborg par manche en crédite un AUTRE. Deux témoins s'opposent — lequel
// se trompe ? La POSITION départage : l'armement est une INTERACTION TENUE au site
// (`primitive_carriable_arming_base` : `Device_GetInteractionHoldTime`), donc le poseur est
// IMMOBILE à la pose, au même lieu que les poses authentifiées du même film.
//
// # PROTOCOLE, écrit avant la mesure
//
//	P1  CONTRÔLE POSITIF : sur chaque explosion à ACCORD, le poseur (slot du canal) doit
//	    être quasi immobile sur [tPose−2500, tPose+2000] — son amplitude en quanta calibre
//	    l'« immobilité d'armement ». Sa position au lâcher devient un SITE de référence.
//	P2  Pour chaque DÉSACCORD : amplitude du slot du canal sur la même fenêtre, ET
//	    amplitude de chaque vie du xuid crédité par le statborg vivante à tPose (pont
//	    inverse slot->xuid). Distances de chacun aux sites de référence du film.
//	P3  VERDICT par explosion : le candidat compatible avec P1 (amplitude du même ordre que
//	    le contrôle, distance à un site du même ordre que les sites entre eux) est le
//	    poseur. Si les deux le sont, ou aucun : INDÉCIS, publié tel quel.
//
// La mèche appliquée par film suit les mesures ti=12 : 4930 ms partout, sauf `1c01e34f`
// (Husky Raid : ~5100 ms, plancher variantes 0/1000).
//
// RÉGIME : garde `ASSAUT_CACHE`. Aucune base, aucun réseau, sentinelle mémoire armée, verrou
// process filmdec (un seul décodage à la fois).
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run BombeB3 -v -timeout 60m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// b3Films : les trois films porteurs des quatre désaccords de B2-V1.
var b3Films = []string{"1c01e34f", "3d58eb37", "69b16f5d"}

// b3MecheMS rend la mèche mesurée du film (Husky Raid a la sienne).
func b3MecheMS(id string) int {
	if id == "1c01e34f" {
		return 5100
	}
	return b2MecheMS
}

const (
	b3FenAvantMS = 2500 // fenêtre d'immobilité avant la pose
	b3FenApresMS = 2000 // et après
)

// b3Pos est l'index des positions QuantaOnly d'un film, par slot.
type b3Pos map[uint32][]filmdec.BipedPosition

// b3ChargerPositions indexe les positions du film par slot, datées ms match.
func b3ChargerPositions(t *testing.T, dir string) (b3Pos, uint64) {
	t.Helper()
	opt := filmdec.DefaultScanFilmOptions()
	opt.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("%s : positions illisibles : %v", dir, err)
	}
	originUS, err := ScanFilmClockOrigin(dir)
	if err != nil {
		t.Fatalf("%s : horloge illisible : %v", dir, err)
	}
	bySlot := b3Pos{}
	for _, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	return bySlot, originUS
}

// b3Fenetre rend les quanta du slot dans [t0, t1] (ms match).
func (bp b3Pos) b3Fenetre(slot uint32, originUS uint64, t0, t1 int) [][3]uint32 {
	var out [][3]uint32
	for _, p := range bp[slot] {
		tMS := int((p.TimestampUS - originUS) / 1000)
		if tMS >= t0 && tMS <= t1 {
			out = append(out, p.Q)
		}
	}
	return out
}

// b3Amplitude rend l'amplitude du nuage (distance max au premier point), en quanta.
func b3Amplitude(qs [][3]uint32) float64 {
	if len(qs) < 2 {
		return -1
	}
	ref := qs[0]
	worst := 0.0
	for _, q := range qs[1:] {
		if d := b3Dist(ref, q); d > worst {
			worst = d
		}
	}
	return worst
}

// b3Dist est l'ADAPTATEUR de types vers l'unique ecriture de la distance 3D du paquet
// (`dist3`, geometry.go — regle du garde-rail TestUneSeuleFormuleDeDistance3D). Les quanta
// d'axe tiennent sur 17-18 bits : la conversion en float32 (mantisse 24 bits) est exacte.
func b3Dist(a, b [3]uint32) float64 {
	return dist3(
		[3]float32{float32(a[0]), float32(a[1]), float32(a[2])},
		[3]float32{float32(b[0]), float32(b[1]), float32(b[2])},
	)
}

// b3DernierAvant rend le dernier quantum du slot à <= t (fenêtre 3 s), s'il existe.
func (bp b3Pos) b3DernierAvant(slot uint32, originUS uint64, t int) ([3]uint32, bool) {
	var best [3]uint32
	bestT, found := -1, false
	for _, p := range bp[slot] {
		tMS := int((p.TimestampUS - originUS) / 1000)
		if tMS <= t && tMS > t-3000 && tMS > bestT {
			best, bestT, found = p.Q, tMS, true
		}
	}
	return best, found
}

// b3ViesVivantes rend les slots du xuid dont une position tombe dans [t-2500, t+2000].
func b3ViesVivantes(bp b3Pos, slotXUID map[uint32]uint64, originUS uint64, xuid uint64, t int) []uint32 {
	var out []uint32
	for slot, x := range slotXUID {
		if x != xuid {
			continue
		}
		if len(bp.b3Fenetre(slot, originUS, t-b3FenAvantMS, t+b3FenApresMS)) > 0 {
			out = append(out, slot)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// TestBombeB3Desaccords applique P1-P3 aux trois films litigieux.
func TestBombeB3Desaccords(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandée : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestBombeB3Desaccords")()
	release := filmdec.LockProcessDecode()
	defer release()

	for _, id := range b3Films {
		dir := filepath.Join(cache, "film_chunks", id)
		evs, slotXUID, deaths := b2Timeline(t, cache, id, b2Bombe)
		periodes := b2Periodes(evs, slotXUID, deaths)
		detonateurs := b2Detonateurs(t, cache, id)
		bp, originUS := b3ChargerPositions(t, dir)

		var sites []b3Site
		meche := b3MecheMS(id)

		// P1 — les accords calibrent l'immobilité et posent les sites.
		for _, tE := range a5Explosions[id] {
			tPose := tE - meche
			xDet, okDet := detonateurs[tE]
			p, okP := b2PorteurA(periodes, tPose, b2DernierPorteurMaxMS)
			if !okDet || !okP || p.XUID == 0 {
				continue
			}
			if xuidStr(p.XUID) != xDet {
				continue // désaccord : traité en P2
			}
			ampl := b3Amplitude(bp.b3Fenetre(p.Slot, originUS, tPose-b3FenAvantMS, tPose+b3FenApresMS))
			fin := p.FinMS
			if fin > tE {
				fin = tE
			}
			if q, ok := bp.b3DernierAvant(p.Slot, originUS, fin); ok {
				sites = append(sites, b3Site{q: q, tE: tE, slot: p.Slot})
				t.Logf("%s ACCORD %d : slot %d, amplitude d'armement %.0f quanta — SITE pris au lâcher %d",
					id, tE, p.Slot, ampl, fin)
			}
		}
		for i := 0; i < len(sites); i++ {
			for j := i + 1; j < len(sites); j++ {
				t.Logf("%s : distance entre sites %d et %d : %.0f quanta",
					id, sites[i].tE, sites[j].tE, b3Dist(sites[i].q, sites[j].q))
			}
		}

		// P2 — les désaccords, mesurés des deux côtés.
		for _, tE := range a5Explosions[id] {
			tPose := tE - meche
			xDet, okDet := detonateurs[tE]
			p, okP := b2PorteurA(periodes, tPose, b2DernierPorteurMaxMS)
			if !okDet || !okP || p.XUID == 0 || xuidStr(p.XUID) == xDet {
				continue
			}
			t.Logf("%s DÉSACCORD %d (pose ~%d) : canal slot %d (xuid %d) contre statborg xuid %s",
				id, tE, tPose, p.Slot, p.XUID, xDet)
			b3MesureDesaccord(t, bp, sites, slotXUID, originUS, p, xDet, tPose)
		}
	}
}

// b3MesureDesaccord mesure amplitude et distances aux sites des DEUX candidats d'un désaccord.
func b3MesureDesaccord(t *testing.T, bp b3Pos, sites []b3Site, slotXUID map[uint32]uint64,
	originUS uint64, p HeldObjectPeriod, xDet string, tPose int) {
	t.Helper()
	amplCanal := b3Amplitude(bp.b3Fenetre(p.Slot, originUS, tPose-b3FenAvantMS, tPose+b3FenApresMS))
	t.Logf("    canal    : slot %d amplitude %.0f quanta%s", p.Slot, amplCanal, b3VersSites(bp, sites, p.Slot, originUS, tPose))
	var xDetU uint64
	for _, x := range slotXUID {
		if xuidStr(x) == xDet {
			xDetU = x
			break
		}
	}
	if xDetU == 0 {
		t.Logf("    statborg : xuid %s SANS vie pontée dans ce film — position invérifiable", xDet)
		return
	}
	vies := b3ViesVivantes(bp, slotXUID, originUS, xDetU, tPose)
	if len(vies) == 0 {
		t.Logf("    statborg : xuid %s sans vie vivante à la pose — position invérifiable", xDet)
	}
	for _, s := range vies {
		ampl := b3Amplitude(bp.b3Fenetre(s, originUS, tPose-b3FenAvantMS, tPose+b3FenApresMS))
		t.Logf("    statborg : vie slot %d amplitude %.0f quanta%s", s, ampl, b3VersSites(bp, sites, s, originUS, tPose))
	}
}

// b3Site est un lieu de pose authentifié par un accord.
type b3Site struct {
	q    [3]uint32
	tE   int
	slot uint32
}

// b3VersSites formate les distances du slot (à l'instant t) aux sites de référence.
func b3VersSites(bp b3Pos, sites []b3Site, slot uint32, originUS uint64, t int) string {
	q, ok := bp.b3DernierAvant(slot, originUS, t+b3FenApresMS)
	if !ok || len(sites) == 0 {
		return ""
	}
	out := ""
	for _, s := range sites {
		out += fmt.Sprintf(", site %d : %.0f quanta", s.tE, b3Dist(q, s.q))
	}
	return out
}

// xuidStr écrit un xuid en décimal, comme le pont statborg.
func xuidStr(x uint64) string { return strconv.FormatUint(x, 10) }
