package filmdec

// vehicules_v2_test.go — INSTRUMENT DE MESURE du lot V2 (vehicules : emplacements de spawn,
// confrontation .mvar, cooldowns, etat detruit). LECTURE SEULE, garde par V2_FILMS.
//
// CE QU'IL MESURE, ET SUR QUELLE MATIERE. Le record de CREATION d'un vehicule (ti=40) porte sa
// POSITION DE NAISSANCE (i0 dyn.-prec.) et l'IDENTITE du chassis (MPPWord32) — voie ouverte par
// V1b (vehicle_creation.go). Le RECENSEMENT des images-cles (world_object_census.go) borne la vie
// (slot, gen) : dernier instant vu, premier instant NON vu. De ces deux lectures viennent :
//   - Item 1 EMPLACEMENTS : agregation des naissances par chassis, regroupees en amas.
//   - Item 2 .MVAR : confrontation des amas au catalogue static map_weapon_pads.json.
//   - Item 3 COOLDOWNS : ecart entre la fin BORNEE d'une vie et la naissance suivante au meme amas.
//   - Item 4 ETAT DETRUIT : frequence de i14 object-dissolver dans le flux delta et sa correlation
//     avec les fins de vie bornees.
//
// ACCEPTATION IDENTIQUE A V1.5 (comparabilite). Les naissances passent par le MEME gate durci que
// l'instrument V1.5 (vehicle_creation_test.go) : default-state porte (consumeDefaultStateTI40) +
// gate i0 dyn.-prec. + NUAGE des positions reelles (vehProbe/vehicleBuildCloud) + calibration MPP
// via ti=37 (vehicleCalibrateMPP). Aucune seconde copie du gate n'est ecrite : les helpers V1b
// sont reutilises tels quels. La coordonnee est decodee en cube unite [0,1] (comme V1.5, pour ne
// PAS bouger le gate valide) puis PROJETEE en metres par les bornes de la carte (map_quant_bounds).
// La projection est lineaire et exacte : metres[ax] = min[ax] + unite[ax] * (max[ax] - min[ax]).
//
// LIMITE STRUCTURELLE (items 3 et 4). Les images-cles sont espacees de ~20 s (world_object_census.go)
// : une fin de vie est bornee a +/-20 s, JAMAIS datee. Un cooldown < ~20 s n'est pas mesurable a
// cette resolution. i14 dans le flux delta est le seul signal potentiellement DATE de disparition ;
// il est mesure, jamais suppose.
//
// UN SEUL decodage filmdec par process : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api, cache Go ISOLE) :
//
//	$env:GOCACHE='<scratch>\gocache_v2'
//	CGO_ENABLED=0 V2_FILM_ROOT=<repo>/data/cache \
//	  V2_FILMS="0d76e8f1:behemoth,fccc61cd:launch site" \
//	  V2_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  V2_PADS=<repo>/data/titles/halo_infinite/reference/map_weapon_pads.json \
//	  go test ./internal/analysis/filmdec/ -run '^TestV2SpawnsCooldowns$' -v -timeout 180m

import (
	"path/filepath"
	"sort"
	"testing"
)

// SEUILS, ecrits AVANT toute mesure.
const (
	v2ClusterThreshM = 2.0    // Item 1 : seuil de regroupement (rayon d'amas <= 2 m)
	v2MaxClusters    = 4      // Item 1 : <= 4 amas par chassis
	v2StabilityBand  = 1      // Item 1 : nombre d'amas stable d'un film a l'autre (+/- 1)
	v2PadNearM       = 1.0    // Item 2 : amas a < 1 m d'un emplacement declare
	v2KeyframeStepS  = 20.0   // resolution du recensement (limite items 3/4)
	v2FloorFrac      = 0.0017 // Item 4 : plancher de faux positifs (cadrage V0, 0,17 %)
	v2CooldownIQRMax = 0.25   // Item 3 : IQR <= 25 % de la mediane
	v2DissolverIdx   = 14     // i14 object-dissolver
)

// v2Birth est une naissance de vehicule projetee en metres.
type v2Birth struct {
	film       string
	slot, gen  uint32
	chassis    uint32
	hasChassis bool
	posM       [3]float64
	tsUS       uint64
}

// v2Life est la vie (slot, gen) bornee par le recensement.
type v2Life struct {
	firstSeen, lastSeen uint64
	goneBy              uint64 // premiere image-cle APRES lastSeen
	hasGoneBy           bool
}

// v2Ev est un evenement delta date (i14).
type v2Ev struct {
	slot, gen uint32
	tsUS      uint64
}

// v2FilmData porte la lecture d'UN film.
type v2FilmData struct {
	short8 string
	births []v2Birth // une par vie (slot, gen), representative = la plus precoce
	lives  map[[2]uint32]v2Life
	i14    []v2Ev
	i14rec int // records delta acceptes portant i14
	i14tot int // records delta acceptes au total
	lastKF uint64
}

// v2MapAgg agrege tous les films d'une carte.
type v2MapAgg struct {
	mapKey string
	bounds MapQuantEntry
	films  []*v2FilmData
}

func TestV2SpawnsCooldowns(t *testing.T) {
	films := v2ParseFilms(t)
	root := v2Root()
	boundsCat := v2LoadBounds(t)
	pads := v2LoadPads(t)

	release := LockProcessDecode()
	defer release()

	aggs := map[string]*v2MapAgg{}
	for _, f := range films {
		dir := filepath.Join(root, "film_chunks", f.short8)
		entry, err := boundsCat.Lookup(f.mapKey)
		if err != nil {
			t.Fatalf("%s : bornes de %q introuvables : %v", f.short8, f.mapKey, err)
		}
		ag := aggs[f.mapKey]
		if ag == nil {
			ag = &v2MapAgg{mapKey: f.mapKey, bounds: entry}
			aggs[f.mapKey] = ag
		}
		t.Run(f.short8, func(t *testing.T) {
			fd := v2ProcessFilm(t, dir, f.short8, entry)
			ag.films = append(ag.films, fd)
		})
	}

	keys := make([]string, 0, len(aggs))
	for k := range aggs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ag := aggs[k]
		t.Logf("\n############## CARTE %q — %d films ##############", k, len(ag.films))
		v2Item1Spawns(t, ag)
		v2Item2Mvar(t, ag, pads)
		v2Item3Cooldowns(t, ag)
		v2Item4Destroyed(t, ag)
	}
	v2BonusNote(t)
}

// v2ProcessFilm lit un film : naissances (metres + chassis), recensement, i14.
func v2ProcessFilm(t *testing.T, dir, short8 string, entry MapQuantEntry) *v2FilmData {
	lay, err := vehicleI0Layout(dir)
	if err != nil {
		t.Fatalf("decoupage i0 illisible : %v", err)
	}
	prevW := CurrentMPPWidths()
	t.Cleanup(func() { SetMPPWidths(prevW) })
	vehicleCalibrateMPP(t, dir, lay)

	n := CountFilmChunks(dir)
	band := worldObjectSlotBand(dir, n, VehicleTypeIndex)
	if len(band) == 0 {
		t.Fatalf("aucun slot ti=%d aux images-cles", VehicleTypeIndex)
	}
	arch, err := vehicleArchetype(dir)
	if err != nil {
		t.Fatalf("archetype ti=%d illisible : %v", VehicleTypeIndex, err)
	}
	cloud := vehicleBuildCloud(t, dir, band, lay)
	pr := vehProbe{dir: dir, lay: lay, cloud: cloud, comps: len(arch.Components)}
	cre, st, err := pr.scan(band, consumeDefaultStateTI40)
	if err != nil {
		t.Fatalf("balayage creations : %v", err)
	}

	rng := entry.Range()
	births := v2BirthsPerLife(cre, short8, rng)
	kf := ScanFilmWorldObjectKeyframes(dir, VehicleTypeIndex)
	lives := v2LivesFromCensus(kf)
	i14, i14rec, i14tot := v2ScanI14(dir, band)

	t.Logf("V2 %s — creations acceptees %d -> %d vies avec naissance · %d vies recensees · i14 %d/%d records (%.3f %%)",
		short8, st.Accepted, len(births), len(lives), i14rec, i14tot, 100*v2SafeFrac(i14rec, i14tot))
	return &v2FilmData{
		short8: short8, births: births, lives: lives,
		i14: i14, i14rec: i14rec, i14tot: i14tot, lastKF: kf.LastTimeUS(),
	}
}

// v2BirthsPerLife dedup les creations a UNE naissance par vie (slot, gen) : la plus precoce.
func v2BirthsPerLife(cre []EquipmentCreation, film string, rng Vec3Range) []v2Birth {
	best := map[[2]uint32]v2Birth{}
	for _, c := range cre {
		key := [2]uint32{c.Slot, c.Gen}
		b := v2Birth{
			film: film, slot: c.Slot, gen: c.Gen, tsUS: c.TimestampUS,
			posM: v2ToMeters([3]float32{c.X, c.Y, c.Z}, rng),
		}
		if c.MPPPresent[MPPWord32] {
			b.hasChassis, b.chassis = true, uint32(c.MPPVal[MPPWord32])
		}
		if prev, ok := best[key]; !ok || b.tsUS < prev.tsUS {
			best[key] = b
		}
	}
	out := make([]v2Birth, 0, len(best))
	for _, b := range best {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tsUS < out[j].tsUS })
	return out
}

// v2ToMeters projette une coordonnee unite [0,1] en metres par les bornes de la carte.
func v2ToMeters(u [3]float32, rng Vec3Range) [3]float64 {
	var m [3]float64
	for ax := 0; ax < 3; ax++ {
		lo, hi := float64(rng[ax].Min), float64(rng[ax].Max)
		m[ax] = lo + float64(u[ax])*(hi-lo)
	}
	return m
}

// v2LivesFromCensus borne chaque vie : premier/dernier instant recense + premiere image-cle apres.
func v2LivesFromCensus(kf WorldObjectKeyframes) map[[2]uint32]v2Life {
	out := map[[2]uint32]v2Life{}
	for key, seen := range kf.SeenUS {
		if len(seen) == 0 {
			continue
		}
		l := v2Life{firstSeen: seen[0], lastSeen: seen[len(seen)-1]}
		for _, t := range kf.TimesUS {
			if t > l.lastSeen {
				l.goneBy, l.hasGoneBy = t, true
				break
			}
		}
		out[[2]uint32{key.Slot, key.Gen}] = l
	}
	return out
}

// v2ScanI14 balaye le flux delta ti=40 et releve les records portant i14, avec leur instant. Meme
// acceptation que le cadrage V0 (matchWorldObjectRecord + i0 present + position dequantifiee valide).
func v2ScanI14(dir string, band map[uint32]bool) (ev []v2Ev, withI14, total int) {
	n := CountFilmChunks(dir)
	posBits := projPosBits()
	wr := Vec3Range{{Min: 0, Max: 1}, {Min: 0, Max: 1}, {Min: 0, Max: 1}}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits + posBits)
			for p := 0; p <= limit; p++ {
				rec, ok := matchWorldObjectRecord(pay, p, band)
				if !ok || rec.Idx[0] != 0 {
					continue
				}
				if _, ok := decodeWorldObjectPos(pay, rec.After, &wr); !ok {
					continue
				}
				total++
				if v2HasIdx(rec.Idx, v2DissolverIdx) {
					withI14++
					ev = append(ev, v2Ev{slot: rec.Slot, gen: rec.Gen, tsUS: pk.TimestampUS})
				}
				p += posBits
			}
		}
	}
	return ev, withI14, total
}

func v2HasIdx(idx []int, want int) bool {
	for _, i := range idx {
		if i == want {
			return true
		}
	}
	return false
}
