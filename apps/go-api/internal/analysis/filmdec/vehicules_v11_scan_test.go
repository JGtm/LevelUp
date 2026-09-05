package filmdec

// vehicules_v11_scan_test.go — INSTRUMENT DU LOT V11 : LE BALAYAGE DES RECORDS **SANS i0**.
//
// LE POINT AVEUGLE QUE CE FICHIER OUVRE. `ScanBipedRecords` n'accepte un record que s'il porte
// un `i0` ABSOLU de la region attendue ET si le premier index de son masque vaut 0
// (`ascendingFromZero`). C'est la definition operationnelle de « vie muette » du lot V8 : une
// tourelle attachee ne replique pas sa position, donc aucun de ses records n'a d'`i0`, donc
// AUCUN n'est vu. Or un record qui ne replique QUE `i2 object-forward-and-up` (l'orientation)
// est exactement de cette forme. La marche sequentielle (`DecodeFrameViews`) ne repond pas non
// plus : elle ne recupere que 323 records `ti=40` sur les ~32 000 que le film contient, et la
// plupart de ses masques sont post-desynchronisation (ils portent des index > 47, impossibles
// pour cet archetype).
//
// CE QUE CE BALAYAGE FAIT. Il teste la seule grammaire d'EN-TETE, sans rien exiger d'`i0` :
//
//	[1 prefixe=1][13 slot][2 tag][2 zeros][3 maskCount][6 x maskCount index croissants]
//
// et il classe chaque touche par la CLASSE de son slot. Il n'affirme rien seul : c'est la
// comparaison de quatre classes qui parle.
//
//	CHASSIS   slot `ti=40` avec au moins une position acceptee — le CONTROLE POSITIF.
//	TOURELLE  slot `ti=40` recense, sans position, dont `slot+1` est un chassis recense a la
//	          MEME fenetre (le motif etabli par le lot V8).
//	MUET      les autres slots `ti=40` recenses sans position.
//	FANTOME   une bande de slots JAMAIS vus porter le moindre archetype, de meme cardinalite
//	          que TOURELLE — le PLANCHER DE BRUIT, mesure et non estime.
//
// LECTURE SEULE : aucun fichier ecrit, aucune base ouverte.
//
//	CGO_ENABLED=0 V11_ROOT=<cache> V11_FILMS=0d76e8f1,fccc61cd \
//	  go test ./internal/analysis/filmdec/ -run TestV11ScanSansI0 -v -timeout 120m

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"
)

// v11ClasseNoms indexe les classes de slots comparees.
var v11ClasseNoms = []string{"CHASSIS", "TOURELLE", "MUET", "BIPEDE", "FANTOME"}

// v11MaxComposants est le nombre de composants de l'archetype `ti=40` (0..47) : un masque qui
// declare un index au-dela ne peut pas appartenir a un record de vehicule.
const v11MaxComposants = 48

// v11Classement porte, par classe, le releve du balayage.
type v11Classement struct {
	Slots   int
	Touches int
	SansI0  int
	Formes  map[string]int
	ParSlot map[uint32]int
	// Index compte, sur les seules touches SANS i0, combien portent chaque index de composant.
	Index map[int]int
}

func newV11Classement() *v11Classement {
	return &v11Classement{Formes: map[string]int{}, ParSlot: map[uint32]int{}, Index: map[int]int{}}
}

// v11HistoIndex rend l'histogramme des index de composant, normalise par slot.
func (r *v11Classement) v11HistoIndex() string {
	idx := make([]int, 0, len(r.Index))
	for i := range r.Index {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	var b strings.Builder
	for _, i := range idx {
		fmt.Fprintf(&b, "i%d=%.1f ", i, float64(r.Index[i])/float64(r.Slots))
	}
	return b.String()
}

// TestV11ScanSansI0 — LA MESURE : les slots de tourelle emettent-ils des records que le
// balayage de positions ne peut pas voir ?
func TestV11ScanSansI0(t *testing.T) {
	for _, dir := range v11Films(t) {
		v11ScanUnFilm(t, dir)
	}
}

func v11ScanUnFilm(t *testing.T, dir string) {
	t.Helper()
	if CountFilmChunks(dir) == 0 {
		t.Logf("V11 %s : film absent — saute", dir)
		return
	}
	release := LockProcessDecode()
	defer release()
	kf := ScanFilmWorldObjectKeyframes(dir, v11VehiculeTI)
	if len(kf.Band) == 0 {
		t.Logf("V11 %s : bande ti=40 vide", dir)
		return
	}
	avecPos := v11SlotsAvecPosition(t, dir, kf.Band)
	classe := v11ClasseParSlot(kf, avecPos, dir)
	res := v11Balaye(dir, classe)
	total := 0
	for _, r := range res {
		total += r.Touches
	}
	t.Logf("V11 SCAN %s — %d touches d'en-tete au total", dir, total)
	for _, nom := range v11ClasseNoms {
		r := res[nom]
		if r == nil || r.Slots == 0 {
			continue
		}
		t.Logf("V11 SCAN %s [%s] — %d slots · %d touches (%.1f par slot) · dont SANS i0 %d "+
			"(%.1f par slot)\n    formes sans i0 : %s",
			dir, nom, r.Slots, r.Touches, float64(r.Touches)/float64(r.Slots), r.SansI0,
			float64(r.SansI0)/float64(r.Slots), v11TopFormes(r.Formes, 8))
		t.Logf("    index sans i0 (touches par slot) : %s", r.v11HistoIndex())
		if nom == "TOURELLE" || nom == "FANTOME" {
			v11PublieParSlot(t, dir, nom, r)
		}
	}
}

// v11ClasseParSlot repartit les slots de la bande dans les quatre classes.
func v11ClasseParSlot(kf WorldObjectKeyframes, avecPos map[uint32]int, dir string) map[uint32]string {
	fenetre := map[uint32][]uint64{}
	for k, seen := range kf.SeenUS {
		if len(seen) > 0 {
			fenetre[k.Slot] = []uint64{seen[0], seen[len(seen)-1]}
		}
	}
	out := map[uint32]string{}
	var muets []uint32
	for s := range fenetre {
		switch {
		case avecPos[s] > 0:
			out[s] = "CHASSIS"
		case v11MemeFenetre(fenetre, s, s+1) && avecPos[s+1] > 0:
			out[s] = "TOURELLE"
			muets = append(muets, s)
		default:
			out[s] = "MUET"
			muets = append(muets, s)
		}
	}
	n := 0
	for _, s := range muets {
		if out[s] == "TOURELLE" {
			n++
		}
	}
	chunks := make([]int, 0, CountFilmChunks(dir))
	for c := 1; c <= CountFilmChunks(dir); c++ {
		chunks = append(chunks, c)
	}
	for _, s := range bipedSlotBandDir(dir, chunks).Slots() {
		if _, deja := out[s]; !deja {
			out[s] = "BIPEDE"
		}
	}
	for _, s := range v11BandeFantome(dir, n) {
		out[s] = "FANTOME"
	}
	return out
}

// v11MemeFenetre dit si deux slots sont recenses sur la MEME fenetre d'images-cles.
func v11MemeFenetre(f map[uint32][]uint64, a, b uint32) bool {
	fa, fb := f[a], f[b]
	return len(fa) == 2 && len(fb) == 2 && fa[0] == fb[0] && fa[1] == fb[1]
}

// v11BandeFantome rend n slots JAMAIS vus porter le moindre archetype aux images-cles : le
// planchers de bruit du balayage se mesure, il ne s'estime pas.
func v11BandeFantome(dir string, n int) []uint32 {
	utilises := map[uint32]bool{}
	max := uint32(0)
	for c := 1; c <= CountFilmChunks(dir); c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				if r.Slot >= 0 {
					utilises[uint32(r.Slot)] = true
					if uint32(r.Slot) > max {
						max = uint32(r.Slot)
					}
				}
			}
			break
		}
	}
	var out []uint32
	for s := max + 64; s < 1<<bipedSlotBits && len(out) < n; s++ {
		if !utilises[s] {
			out = append(out, s)
		}
	}
	return out
}

// v11Balaye parcourt TOUS les paquets delta du film, bit a bit, et classe chaque en-tete de
// record reconnue par la classe de son slot.
func v11Balaye(dir string, classe map[uint32]string) map[string]*v11Classement {
	res := map[string]*v11Classement{}
	for _, nom := range v11ClasseNoms {
		res[nom] = newV11Classement()
	}
	for s, nom := range classe {
		res[nom].Slots++
		res[nom].ParSlot[s] = 0
	}
	for c := 1; c <= CountFilmChunks(dir); c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			v11BalayePayload(pk.Payload(data), classe, res)
		}
	}
	return res
}

// v11BalayePayload teste la grammaire d'en-tete a CHAQUE position bit d'un payload.
func v11BalayePayload(pay []byte, classe map[uint32]string, res map[string]*v11Classement) {
	total := len(pay) * 8
	for p := 0; p+64 <= total; p++ {
		if pay[p>>3]>>(7-uint(p&7))&1 != 1 { // prefixe delta
			continue
		}
		if readBitsAt(pay, p+16, 2) != 0 { // 14e bit d'id + selecteur de baseline
			continue
		}
		mc := int(readBitsAt(pay, p+18, 3))
		if mc < 1 || mc > bipedMaxMaskCnt {
			continue
		}
		slot := readBitsAt(pay, p+1, bipedSlotBits)
		nom, ok := classe[slot]
		if !ok {
			continue
		}
		if p+21+6*mc > total {
			continue
		}
		idx, ok := v11MasqueCroissant(pay, p+21, mc)
		if !ok {
			continue
		}
		r := res[nom]
		r.Touches++
		r.ParSlot[slot]++
		if idx[0] != 0 {
			r.SansI0++
			r.Formes[v11FormeMasque(idx)]++
			for _, i := range idx {
				r.Index[i]++
			}
		}
	}
}

// v11MasqueCroissant valide une liste d'index de composants STRICTEMENT croissants et tous
// dans l'archetype (< 48). Contrairement a `ascendingFromZero`, il n'exige PAS que le premier
// index vaille 0 : c'est tout l'objet de ce balayage.
func v11MasqueCroissant(pay []byte, at, count int) ([]int, bool) {
	out := make([]int, 0, count)
	prev := -1
	for k := 0; k < count; k++ {
		v := int(readBitsAt(pay, at+bipedIndexBits*k, bipedIndexBits))
		if v <= prev || v >= v11MaxComposants {
			return nil, false
		}
		prev = v
		out = append(out, v)
	}
	return out, true
}

// v11PublieParSlot imprime le detail slot par slot d'une classe (diagnostic).
func v11PublieParSlot(t *testing.T, dir, nom string, r *v11Classement) {
	t.Helper()
	slots := make([]uint32, 0, len(r.ParSlot))
	for s := range r.ParSlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var b strings.Builder
	for _, s := range slots {
		fmt.Fprintf(&b, "%d:%d ", s, r.ParSlot[s])
	}
	t.Logf("V11 SCAN %s [%s] par slot : %s", dir, nom, b.String())
}

// ---------------------------------------------------------------------------------------
// GATE (c) — LA VISEE DE L'OCCUPANT EST-ELLE DISTINCTE DU CAP DU CHASSIS ?
// ---------------------------------------------------------------------------------------

// TestV11ConeDistinct mesure, pendant l'occupation, l'ecart entre la visee de l'occupant
// (`ScanFilmBipedAimOnly`, offline_aim_only.go) et le cap du VEHICULE qu'il occupe.
//
// POURQUOI CE GATE. Le rejeu dessine aujourd'hui UN cone par vehicule, oriente par le cap du
// chassis. Une visee d'occupant qui vaudrait ce cap n'apporterait RIEN. Le gate est donc :
// l'ecart doit etre GRAND et etale, pas nul.
//
// LE CAP DU CHASSIS EST PRIS SUR `i1` (direction de velocite), la seule orientation de
// vehicule VALIDEE (lot V1a : 1,7 a 2,1 deg d'ecart median au deplacement, temoin 51 a
// 88 deg) — et elle se lit SANS bornes de carte, la direction etant un vecteur unitaire
// empaquete.
func TestV11ConeDistinct(t *testing.T) {
	for _, dir := range v11Films(t) {
		v11DistinctUnFilm(t, dir)
	}
}

func v11DistinctUnFilm(t *testing.T, dir string) {
	t.Helper()
	if CountFilmChunks(dir) == 0 {
		return
	}
	release := LockProcessDecode()
	defer release()
	evs, err := ScanFilmVehicleEvents(dir)
	if err != nil {
		t.Logf("V11 DISTINCT %s : %v", dir, err)
		return
	}
	visees, err := ScanFilmBipedAimOnly(dir)
	if err != nil {
		t.Logf("V11 DISTINCT %s : %v", dir, err)
		return
	}
	opt := ScanFilmOptions{DropSaturated: true, QuantaOnly: true, CaptureDirs: true,
		DynPrecOrientation: true}
	bandeV := ScanFilmWorldObjectKeyframes(dir, v11VehiculeTI).Band
	veh, err := ScanFilmBipedPositionsForBand(dir, NewSlotBand(bandeV), opt)
	if err != nil {
		t.Logf("V11 DISTINCT %s : bande vehicule : %v", dir, err)
		return
	}
	v11PublieDistinct(t, dir, evs, visees, v11CapsVehicule(veh))
}

// v11CapDate est un cap date.
type v11CapDate struct {
	TS  uint64
	Cap float64
}

// v11CapsVehicule rend, par slot de vehicule, la serie datee du cap de deplacement lu sur i1.
func v11CapsVehicule(veh []BipedPosition) map[uint32][]v11CapDate {
	out := map[uint32][]v11CapDate{}
	for _, p := range veh {
		v, ok := p.VelocityVector()
		if !ok || math.Hypot(float64(v[0]), float64(v[1])) < 3 {
			continue // a l'arret, la direction de velocite ne dit rien
		}
		out[p.Slot] = append(out[p.Slot], v11CapDate{
			TS: p.TimestampUS, Cap: v11Deg(math.Atan2(float64(v[1]), float64(v[0])))})
	}
	for s := range out {
		sort.Slice(out[s], func(i, j int) bool { return out[s][i].TS < out[s][j].TS })
	}
	return out
}

// v11Deg convertit des radians en degres dans [0, 360[.
func v11Deg(r float64) float64 {
	d := r * 180 / math.Pi
	for d < 0 {
		d += 360
	}
	return d
}

// v11PublieDistinct confronte, sur les 20 s qui precedent chaque sortie, la visee de
// l'occupant au cap du vehicule NOMME par cette sortie.
func v11PublieDistinct(t *testing.T, dir string, evs []VehicleEvent, visees []BipedAim,
	capVeh map[uint32][]v11CapDate) {
	t.Helper()
	parSlot := map[uint32][]BipedAim{}
	for _, a := range visees {
		parSlot[a.Slot] = append(parSlot[a.Slot], a)
	}
	var ecarts []float64
	n := 0
	for _, e := range evs {
		if e.Kind != EventUnitExitVehicle || !e.VehicleSlotValid || !e.OccupantInBand {
			continue
		}
		caps := capVeh[e.VehicleSlot]
		if len(caps) == 0 {
			continue
		}
		n++
		for _, a := range parSlot[e.OccupantSlot] {
			if a.TimestampUS > e.TimestampUS || e.TimestampUS-a.TimestampUS > 20_000_000 {
				continue
			}
			c, ok := v11CapProche(caps, a.TimestampUS)
			if !ok {
				continue
			}
			ecarts = append(ecarts, math.Abs(v11Wrap180(float64(a.AimHeadingDeg())-c)))
		}
	}
	if len(ecarts) == 0 {
		t.Logf("V11 DISTINCT %s — aucune paire (visee d'occupant / cap du vehicule nomme)", dir)
		return
	}
	sort.Float64s(ecarts)
	sous30 := 0
	for _, e := range ecarts {
		if e < 30 {
			sous30++
		}
	}
	t.Logf("V11 DISTINCT %s — %d sorties exploitables · %d paires · |ecart visee/cap chassis| "+
		"median %.1f deg · q1 %.1f · q3 %.1f · part sous 30 deg %.1f %%",
		dir, n, len(ecarts), ecarts[len(ecarts)/2], ecarts[len(ecarts)/4],
		ecarts[3*len(ecarts)/4], 100*float64(sous30)/float64(len(ecarts)))
}

// v11CapProche rend le cap de vehicule le plus proche dans le temps, a moins de 500 ms.
func v11CapProche(caps []v11CapDate, ts uint64) (float64, bool) {
	i := sort.Search(len(caps), func(k int) bool { return caps[k].TS >= ts })
	best, ok := 0.0, false
	for _, k := range []int{i - 1, i} {
		if k < 0 || k >= len(caps) {
			continue
		}
		if v11Abs(caps[k].TS, ts) <= 500_000 {
			best, ok = caps[k].Cap, true
		}
	}
	return best, ok
}
