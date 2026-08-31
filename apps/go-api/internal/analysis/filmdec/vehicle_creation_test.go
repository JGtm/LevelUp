package filmdec

// vehicle_creation_test.go — INSTRUMENT DE MESURE du RECORD DE CREATION de ti=40 (vehicule).
//
// L'ORACLE, TRANSPOSE DE ti=42 EN PRECISION-DYNAMIQUE. Un record de creation `ti=40` est ACCEPTE
// quand, apres son default-state porte + la porte has-components + le masque, le composant i0 se
// decode en grammaire biped (porte 5 bits, quanta non satures) ET tombe sur une position REELLE de
// vehicule — le NUAGE des positions decodees par ScanFilmBipedPositionsForBand (livre par V1a). La
// coincidence i0 == position reelle ne s'obtient PAS par hasard : c'est le discriminant fort qui a
// valide ti=42, ici en dyn.-prec. Un default-state de mauvaise largeur (ou un faux deser) met le
// curseur ailleurs, i0 ne tombe pas sur le nuage, le record est rejete.
//
// DEUX MESURES, LES SEUILS ECRITS AVANT LE CODE.
//
// V1.4 — ATTERRISSAGE. Le VRAI deser accepte des records sur la vraie bande ; le TEMOIN FANTOME
// (slots jamais vus ti=40) et les TEMOINS FAUX (ti37/ti36/ti38) doivent accepter ~0. SEUILS :
//   - temoin fantome : Accepted < 5 % de la vraie bande (gate d'interpretabilite du superviseur) ;
//   - chaque temoin faux : Accepted < 10 % de la vraie bande.
//
// V1.5 — IDENTITE MPPWord32 sur les records ACCEPTES. SEUILS :
//   - constance 100 % par vie (slot, gen) ;
//   - <= 8 valeurs distinctes par film ;
//   - le nombre de valeurs ne suit PAS le nombre de vies.
//
// DECOUPAGE MPP : calibre film-wide sur `ti=37` (oracle objet du monde deja valide), applique a
// `ti=40` (le champ de tete du bloc MPP est une constante de REPLICATION du film). Override manuel :
// VEHICLE_MPP_LEAD.
//
// LECTURE SEULE, garde par VEHICLE_CREATION_FILM. UN SEUL decodage filmdec par process : le verrou
// est pris pour tout le test. Verdicts LOGGES (convention des instruments film).
//
// USAGE (depuis apps/go-api, cache Go ISOLE) :
//
//	$env:GOCACHE='<scratch>\gocache_v1b'
//	CGO_ENABLED=0 VEHICLE_CREATION_FILM=<repo>/data/cache/film_chunks/0d76e8f1 \
//	  go test ./internal/analysis/filmdec/ -run '^TestVehicleCreationIdentity$' -timeout 60m -v

import (
	"os"
	"sort"
	"strconv"
	"testing"
)

const vehicleCreationFilmEnv = "VEHICLE_CREATION_FILM"

// Seuils, ecrits AVANT le code.
const (
	vehMPPMaxDistinct      = 8    // V1.5 : <= 8 valeurs distinctes par film
	vehPhantomMaxFrac      = 0.05 // V1.4 : fantome accepte < 5 % de la vraie bande
	vehFalseWitnessMaxFrac = 0.10 // V1.4 : chaque temoin faux accepte < 10 % de la vraie bande
)

// vehCand est un deserialiseur candidat nomme.
type vehCand struct {
	name string
	fn   func(*BitReader)
}

// vehProbe porte l'etat d'une campagne de balayage : film, decoupage i0, nuage de positions reelles
// et nombre de composants. La methode scan applique le gate dyn.-prec. + nuage a une bande donnee.
type vehProbe struct {
	dir   string
	lay   I0Layout
	cloud map[[3]int32]bool
	comps int
}

func TestVehicleCreationIdentity(t *testing.T) {
	dir := os.Getenv(vehicleCreationFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", vehicleCreationFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	lay, err := vehicleI0Layout(dir)
	if err != nil {
		t.Fatalf("decoupage i0 illisible : %v", err)
	}
	t.Logf("FILM %s · decoupage i0 %s", dir, lay)

	prevW := CurrentMPPWidths()
	t.Cleanup(func() { SetMPPWidths(prevW) })
	vehicleCalibrateMPP(t, dir, lay)
	t.Logf("decoupage MPP employe : %s", CurrentMPPWidths())

	n := CountFilmChunks(dir)
	band := worldObjectSlotBand(dir, n, VehicleTypeIndex)
	if len(band) == 0 {
		t.Fatalf("aucun slot ti=%d dans les images-cles de %s", VehicleTypeIndex, dir)
	}
	arch, err := vehicleArchetype(dir)
	if err != nil {
		t.Fatalf("archetype ti=%d illisible : %v", VehicleTypeIndex, err)
	}
	pr := vehProbe{dir: dir, lay: lay, cloud: vehicleBuildCloud(t, dir, band, lay), comps: len(arch.Components)}

	real, st, err := pr.scan(band, consumeDefaultStateTI40)
	if err != nil {
		t.Fatalf("balayage bande reelle : %v", err)
	}
	vehicleLogStats(t, "BANDE REELLE (vrai deser)", st)

	phantom := vehiclePhantomBand(dir, n, band)
	_, pst, err := pr.scan(phantom, consumeDefaultStateTI40)
	if err != nil {
		t.Fatalf("balayage fantome : %v", err)
	}
	vehicleLogStats(t, "TEMOIN FANTOME (vrai deser)", pst)

	vehiclePhantomVerdict(t, st.Accepted, pst.Accepted)
	vehicleFalseWitnesses(t, pr, band, st.Accepted)
	vehicleIdentityVerdict(t, real, st.Accepted)
}

// scan balaye `band` avec `deser` : gate i0 dyn.-prec. DURCI par le nuage des positions reelles.
func (pr vehProbe) scan(band map[uint32]bool, deser func(*BitReader)) (
	[]EquipmentCreation, EquipmentCreationStats, error) {
	var cur equipCreationRead
	w := equipCreationWalk{
		comps: pr.comps, wr: &equipCreationUnitRange, band: band, cur: &cur,
		ti: VehicleTypeIndex, deser: deser,
		posDecode: func(pay []byte, at int) ([3]float32, bool) {
			v, ok := decodeBipedI0Pos(pay, at, pr.lay, &equipCreationUnitRange)
			if !ok || !vehicleCloudNear(pr.cloud, v) {
				return v, false
			}
			return v, true
		},
		posBits: pr.lay.TotalBits(),
	}
	return runCreationWalk(pr.dir, w)
}

// vehicleBuildCloud decode toutes les positions reelles de la bande `ti=40` (grammaire biped, flux
// brut : filtres teleport/isolation desarmes) et rend le SET DE CELLULES qu'elles occupent. C'est
// le nuage de reference du gate : un i0 de creation doit y coincider.
func vehicleBuildCloud(t *testing.T, dir string, band map[uint32]bool, lay I0Layout) map[[3]int32]bool {
	t.Helper()
	opt := ScanFilmOptions{WorldRange: &equipCreationUnitRange, RequireTag1: false, Layout: &lay, DropSaturated: true}
	pos, err := ScanFilmBipedPositionsForBand(dir, band, opt)
	if err != nil {
		t.Fatalf("nuage de positions ti=40 illisible : %v", err)
	}
	cloud := map[[3]int32]bool{}
	for _, p := range pos {
		cloud[equipCell([3]float32{p.X, p.Y, p.Z})] = true
	}
	t.Logf("nuage : %d positions reelles ti=40 -> %d cellules occupees", len(pos), len(cloud))
	return cloud
}

// vehicleCloudNear dit si v tombe dans une cellule du nuage ou l'une de ses 26 voisines (absorbe
// l'ecart entre la position de creation et le premier point delta, comme equipOffsetProbe).
func vehicleCloudNear(cloud map[[3]int32]bool, v [3]float32) bool {
	c := equipCell(v)
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			for dz := int32(-1); dz <= 1; dz++ {
				if cloud[[3]int32{c[0] + dx, c[1] + dy, c[2] + dz}] {
					return true
				}
			}
		}
	}
	return false
}

// vehicleCalibrateMPP calibre le decoupage du bloc MPP (film-wide) sur `ti=37` et l'applique ; en
// cas d'echec, le defaut (9/5) est conserve. VEHICLE_MPP_LEAD force le champ de tete a la main.
func vehicleCalibrateMPP(t *testing.T, dir string, lay I0Layout) {
	t.Helper()
	if forced := os.Getenv("VEHICLE_MPP_LEAD"); forced != "" {
		w, err := strconv.Atoi(forced)
		if err != nil {
			t.Fatalf("VEHICLE_MPP_LEAD=%q illisible : %v", forced, err)
		}
		SetMPPWidths(MPPWidths{Lead: w, Index: CurrentMPPWidths().Index})
		t.Logf("champ de tete MPP FORCE : %d bits", w)
		return
	}
	prevPrec := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prevPrec })
	SetWorldObjectPrecisionFromLayout(lay)
	n := CountFilmChunks(dir)
	band37 := worldObjectSlotBand(dir, n, EquipmentTypeIndex)
	tracks, err := ScanFilmWorldObjects(dir, &equipCreationUnitRange, EquipmentTypeIndex)
	if err != nil || len(band37) == 0 {
		t.Logf("calibration MPP impossible (ti=37 : %v) — defaut %s conserve", err, CurrentMPPWidths())
		return
	}
	cal, ok := CalibrateMPPWidths(dir, &equipCreationUnitRange, band37, EquipmentLifeSpans(tracks))
	t.Logf("calibration MPP (via ti=37) : %s", cal)
	if ok {
		SetMPPWidths(cal.Widths)
	}
}

func vehicleLogStats(t *testing.T, label string, st EquipmentCreationStats) {
	t.Helper()
	t.Logf("== %s — %d slots · ancres NEW %d · ACCEPTES %d ==", label, st.Slots, st.Anchors, st.Accepted)
	t.Logf("   rejets : default-state hors payload %d · masque invalide %d · position rejetee %d",
		st.Overflow, st.MaskBad, st.PosBad)
	t.Logf("   acceptes : masque eparse %d · masque PLEIN %d · sans i0 au masque %d",
		st.MaskSparse, st.MaskFull, st.NoI0)
}

// vehiclePhantomVerdict applique le gate d'interpretabilite : le fantome doit accepter ~0.
func vehiclePhantomVerdict(t *testing.T, realAccepted, phantomAccepted int) {
	t.Helper()
	frac := 1.0
	if realAccepted > 0 {
		frac = float64(phantomAccepted) / float64(realAccepted)
	}
	t.Logf("   V1.4 TEMOIN FANTOME : %d acceptes vs %d (vraie bande) = %.1f %% (seuil < %.0f %%) : %s",
		phantomAccepted, realAccepted, 100*frac, 100*vehPhantomMaxFrac, vehVerdict(frac < vehPhantomMaxFrac))
}

// vehicleFalseWitnesses rejoue le balayage de la VRAIE bande avec trois deserialiseurs FAUX.
func vehicleFalseWitnesses(t *testing.T, pr vehProbe, band map[uint32]bool, realAccepted int) {
	t.Helper()
	cands := []vehCand{
		{"ti37 (FUN_1407f105c)", consumeDefaultStateTI37},
		{"ti36 (FUN_1407f2224)", consumeDefaultStateTI36},
		{"ti38 (FUN_1408f0b48)", consumeDefaultStateTI38},
	}
	for _, c := range cands {
		_, st, err := pr.scan(band, c.fn)
		if err != nil {
			t.Logf("   temoin faux %s : erreur %v", c.name, err)
			continue
		}
		frac := 1.0
		if realAccepted > 0 {
			frac = float64(st.Accepted) / float64(realAccepted)
		}
		t.Logf("   V1.4 TEMOIN FAUX %-24s : %d acceptes = %.1f %% du vrai (seuil < %.0f %%) : %s",
			c.name, st.Accepted, 100*frac, 100*vehFalseWitnessMaxFrac, vehVerdict(frac < vehFalseWitnessMaxFrac))
	}
}

// vehicleIdentityVerdict applique les trois gates V1.5 sur MPPWord32 des records ACCEPTES.
func vehicleIdentityVerdict(t *testing.T, cre []EquipmentCreation, accepted int) {
	t.Helper()
	values := map[uint32]int{}
	lives := map[equipCreationLifeKey]map[uint32]bool{}
	for _, c := range cre {
		if !c.MPPPresent[MPPWord32] {
			continue
		}
		v := uint32(c.MPPVal[MPPWord32])
		values[v]++
		k := equipCreationLifeKey{c.Slot, c.Gen}
		if lives[k] == nil {
			lives[k] = map[uint32]bool{}
		}
		lives[k][v] = true
	}
	if len(lives) == 0 {
		t.Logf("V1.5 IDENTITE : aucun MPPWord32 transmis sur %d records acceptes — rien a juger", accepted)
		return
	}
	inconstant := 0
	for _, vs := range lives {
		if len(vs) > 1 {
			inconstant++
		}
	}
	t.Logf("== V1.5 IDENTITE MPPWord32 — %d vies · %d valeurs distinctes · %d vies inconstantes ==",
		len(lives), len(values), inconstant)
	for _, e := range equipCreationTop(values, vehMPPMaxDistinct+4) {
		t.Logf("   %#010x (%10d)  %6d records", e.key, e.key, e.count)
	}
	t.Logf("   V1.5 gate 1 (constance 100 %% par vie)   : %s (%d vies inconstantes, attendu 0)",
		vehVerdict(inconstant == 0), inconstant)
	t.Logf("   V1.5 gate 2 (cardinalite <= %d)           : %s (%d valeurs distinctes)",
		vehMPPMaxDistinct, vehVerdict(len(values) <= vehMPPMaxDistinct), len(values))
	t.Logf("   V1.5 gate 3 (valeurs decouplees des vies) : %s (%d valeurs pour %d vies)",
		vehVerdict(len(values) < len(lives)), len(values), len(lives))
}

// vehVerdict rend une etiquette de verdict lisible.
func vehVerdict(pass bool) string {
	if pass {
		return "PASS"
	}
	return "ECHEC"
}

// vehiclePhantomBand rend une bande FANTOME de meme cardinalite que `band`, tiree des slots vus
// porter un AUTRE archetype dans les images-cles. Le temoin passe par le meme code que la mesure.
func vehiclePhantomBand(dir string, n int, band map[uint32]bool) map[uint32]bool {
	others := map[uint32]bool{}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				if r.TI != VehicleTypeIndex && !band[uint32(r.Slot)] {
					others[uint32(r.Slot)] = true
				}
			}
		}
	}
	keys := make([]uint32, 0, len(others))
	for s := range others {
		keys = append(keys, s)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	phantom := map[uint32]bool{}
	for _, s := range keys {
		if len(phantom) >= len(band) {
			break
		}
		phantom[s] = true
	}
	return phantom
}
