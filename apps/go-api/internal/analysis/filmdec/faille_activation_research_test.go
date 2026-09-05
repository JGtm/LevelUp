package filmdec

// faille_activation_research_test.go — R1 : quand un joueur ACTIVE le translocateur (pose la
// faille dans le monde), le film enregistre-t-il une ENTITE ?
//
// INSTRUMENT DE RECHERCHE (plan PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03, lot R1). Trois
// canaux d'entités sont balayés dans les FENETRES des ancres de vérité terrain (une ancre =
// un saut de translocation mesuré, avec sa fenêtre de pose [prise, saut]) :
//
//  1. CREATIONS (records NEW) de TOUS les archétypes d'objet du monde dont le default-state
//     est porté ET dont i0 est `object-position-component` : ti 36, 37, 38, 39, 42, 43.
//     La marche est celle de la production (`equipCreationWalk`), l'archétype en paramètre.
//     ti=41 (projectile) est EXCLU des créations : son default-state n'est pas résolu
//     (defaultStateDeserByTI) — il reste couvert par le canal 2.
//  2. DELTAS d'objet du monde (bande UNION des archétypes ci-dessus + ti=41) : toute vie
//     répliquée dont une position tombe à <= failleProcheM (2D) d'une ancre pendant sa
//     fenêtre. Ce canal voit une entité même quand son record de création est illisible.
//  3. RECENSEMENT des images-clés (TOUS les ti 0..49) : toute vie recensée pour la PREMIERE
//     fois dans [t0, t1 + failleCensusUS]. Sans position (le recensement borne, il ne situe
//     pas), mais il attrape un archétype muet du canal delta.
//
// LECTURE SEULE, gardé par FAILLE_FILM (patron TRANSLOC_FILM). UN SEUL décodage par process.
// Balayage BORNE : seuls les paquets dont l'horodatage tombe dans une fenêtre d'ancre
// (+/- failleMargeUS) sont balayés bit à bit.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 FAILLE_FILM=<repo>/data/cache/film_chunks/1b2d9e08 \
//	  FAILLE_BOUNDS=-11.45,104.54,73.91,19.72,153.51,82.53 \
//	  FAILLE_ANCRES="A1:535:17.34,135.50:146862000-185262000:185262000;A2:560:18.34,120.19:328162000-351062000:351062000" \
//	  go test ./internal/analysis/filmdec/ -run '^TestFailleActivationEntites$' -timeout 30m -v
//
// Format d'une ancre : label:slotBipede:x,y:t0us-t1us:tSautUS — t0/t1 = fenêtre de pose
// (frames prise -> saut converties en US par (originMs + t*frameIntervalMs)*1000).

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	failleFilmEnv   = "FAILLE_FILM"
	failleBoundsEnv = "FAILLE_BOUNDS"
	failleAncresEnv = "FAILLE_ANCRES"
	// failleProcheM : distance 2D (m) en deçà de laquelle une position « est » l'ancre.
	// Seuil du plan R1.1, écrit avant la mesure.
	failleProcheM = 3.0
	// failleMargeUS élargit chaque fenêtre de pose : une pose datée à la frontière ne doit
	// pas sortir du balayage pour 2 s d'horloge.
	failleMargeUS = 2_000_000
	// failleCensusUS : marge du recensement d'images-clés (période mesurée ~20 s, cf.
	// world_object_census.go) — une vie née en fin de fenêtre n'est recensée qu'à l'image-clé
	// suivante.
	failleCensusUS = 25_000_000
)

// failleCreationTIs : archétypes dont le record NEW est lisible (default-state porté) et dont
// i0 est une position d'objet du monde (ecs_table.tsv, colonne i=0).
var failleCreationTIs = []uint32{36, 37, 38, 39, 42, 43}

// failleDeltaTIs : archétypes dont les records DELTA portent i0 = object-position-component.
var failleDeltaTIs = []int{36, 37, 38, 39, 41, 42, 43}

// failleAncre est une vérité terrain : un saut de translocation, sa fenêtre de pose, son slot.
type failleAncre struct {
	label  string
	slot   uint32 // slot du bipède qui saute (pour le canal des événements)
	x, y   float64
	t0, t1 uint64 // fenêtre de pose [prise, saut], horloge film en US
	tJump  uint64 // instant exact du saut
}

// failleSetup lit l'environnement de l'instrument ; skip sans FAILLE_FILM, fatal si le reste
// est illisible (un instrument qui devine rend des chiffres faux plutôt qu'une absence).
// Rend aussi l'origine d'horloge, pour que les rapports parlent en millisecondes de FILM.
func failleSetup(t *testing.T) (string, Vec3Range, []failleAncre, uint64) {
	t.Helper()
	dir := os.Getenv(failleFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", failleFilmEnv)
	}
	wr, err := failleBounds()
	if err != nil {
		t.Fatalf("%s : %v", failleBoundsEnv, err)
	}
	ancres, err := failleParseAncres(os.Getenv(failleAncresEnv))
	if err != nil {
		t.Fatalf("%s : %v", failleAncresEnv, err)
	}
	origine, err := failleOrigineHorloge(dir)
	if err != nil {
		t.Fatalf("origine d'horloge illisible : %v", err)
	}
	for i := range ancres {
		ancres[i].t0 += origine
		ancres[i].t1 += origine
		ancres[i].tJump += origine
	}
	t.Logf("ORIGINE D'HORLOGE (premier paquet du film) : %d us — ancres décalées sur l'horloge moteur", origine)
	return dir, wr, ancres, origine
}

// failleMS rend un horodatage moteur en millisecondes d'horloge FILM (lisible dans les logs).
func failleMS(ts, origine uint64) int64 { return (int64(ts) - int64(origine)) / 1000 }

// failleOrigineHorloge rend l'horodatage moteur du PREMIER paquet du film : le zéro de
// l'horloge du film. Les ancres de FAILLE_ANCRES sont données sur l'horloge du FILM
// ((originMs + frame*frameIntervalMs)*1000, cf. l'artefact du rejeu) ; les paquets, eux,
// sont datés sur l'horloge MOTEUR. Même lecture que replay.ScanFilmClockOrigin — recopiée
// ici parce que filmdec ne peut pas importer replay (sens de dépendance inverse).
func failleOrigineHorloge(dir string) (uint64, error) {
	raw, err := ReadFilmChunk(dir, 1)
	if err != nil {
		return 0, err
	}
	packets := WalkPackets(raw)
	if len(packets) == 0 {
		return 0, fmt.Errorf("aucun paquet lisible dans le chunk 1 de %s", dir)
	}
	return packets[0].TimestampUS, nil
}

// failleBounds lit les bornes de la carte (même format que TRANSLOC_BOUNDS : le champ
// `bounds` de l'artefact du rejeu, minX,minY,minZ,maxX,maxY,maxZ en mètres).
func failleBounds() (Vec3Range, error) {
	var wr Vec3Range
	parts := strings.Split(strings.TrimSpace(os.Getenv(failleBoundsEnv)), ",")
	if len(parts) != 6 {
		return wr, fmt.Errorf("6 nombres attendus, %d reçus", len(parts))
	}
	var v [6]float32
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return wr, fmt.Errorf("nombre %d illisible (%q) : %w", i, p, err)
		}
		v[i] = float32(f)
	}
	for a := 0; a < 3; a++ {
		wr[a].Min, wr[a].Max = v[a], v[a+3]
		if wr[a].Max <= wr[a].Min {
			return wr, fmt.Errorf("axe %d : borne haute (%g) sous la basse (%g)", a, wr[a].Max, wr[a].Min)
		}
	}
	return wr, nil
}

// failleParseAncres décode `label:slot:x,y:t0-t1:tJump;...`.
func failleParseAncres(raw string) ([]failleAncre, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("aucune ancre (format label:slot:x,y:t0us-t1us:tSautUS;...)")
	}
	var out []failleAncre
	for _, ent := range strings.Split(raw, ";") {
		f := strings.Split(strings.TrimSpace(ent), ":")
		if len(f) != 5 {
			return nil, fmt.Errorf("ancre %q : 5 champs attendus, %d reçus", ent, len(f))
		}
		slot, err := strconv.ParseUint(f[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("ancre %q : slot illisible : %w", ent, err)
		}
		xy := strings.Split(f[2], ",")
		if len(xy) != 2 {
			return nil, fmt.Errorf("ancre %q : position x,y attendue", ent)
		}
		x, errX := strconv.ParseFloat(xy[0], 64)
		y, errY := strconv.ParseFloat(xy[1], 64)
		tt := strings.Split(f[3], "-")
		if errX != nil || errY != nil || len(tt) != 2 {
			return nil, fmt.Errorf("ancre %q : position ou fenêtre illisible", ent)
		}
		t0, err0 := strconv.ParseUint(tt[0], 10, 64)
		t1, err1 := strconv.ParseUint(tt[1], 10, 64)
		tj, errJ := strconv.ParseUint(f[4], 10, 64)
		if err0 != nil || err1 != nil || errJ != nil || t1 <= t0 {
			return nil, fmt.Errorf("ancre %q : fenêtre/saut illisibles ou inversés", ent)
		}
		out = append(out, failleAncre{label: f[0], slot: uint32(slot), x: x, y: y, t0: t0, t1: t1, tJump: tj})
	}
	return out, nil
}

// failleFenetres rend les indices des ancres dont la fenêtre élargie contient ts.
func failleFenetres(ts uint64, ancres []failleAncre) []int {
	var out []int
	for i, a := range ancres {
		lo := a.t0 - failleMargeUS
		if a.t0 < failleMargeUS {
			lo = 0
		}
		if ts >= lo && ts <= a.t1+failleMargeUS {
			out = append(out, i)
		}
	}
	return out
}

// failleDist2D : distance horizontale (m) entre une position décodée et une ancre.
func failleDist2D(x, y float32, a failleAncre) float64 {
	dx, dy := float64(x)-a.x, float64(y)-a.y
	return math.Sqrt(dx*dx + dy*dy)
}

// failleKF est le produit d'UNE marche des images-clés : instants, slots par archétype
// (pour les bandes), et recensement complet (ti, slot, gen) -> instants.
type failleKF struct {
	timesUS  []uint64
	seenByTI map[int]map[uint32]bool
	lives    map[int]map[EquipmentLifeKey][]uint64
}

// failleWalkKeyframes marche les images-clés du film UNE fois pour tous les archétypes.
func failleWalkKeyframes(dir string, n int) failleKF {
	kf := failleKF{seenByTI: map[int]map[uint32]bool{}, lives: map[int]map[EquipmentLifeKey][]uint64{}}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			kf.timesUS = append(kf.timesUS, pk.TimestampUS)
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				if kf.seenByTI[r.TI] == nil {
					kf.seenByTI[r.TI] = map[uint32]bool{}
					kf.lives[r.TI] = map[EquipmentLifeKey][]uint64{}
				}
				kf.seenByTI[r.TI][uint32(r.Slot)] = true
				key := EquipmentLifeKey{Slot: uint32(r.Slot), Gen: uint32(r.Gen)}
				if v := kf.lives[r.TI][key]; len(v) == 0 || v[len(v)-1] != pk.TimestampUS {
					kf.lives[r.TI][key] = append(kf.lives[r.TI][key], pk.TimestampUS)
				}
			}
		}
	}
	sort.Slice(kf.timesUS, func(i, j int) bool { return kf.timesUS[i] < kf.timesUS[j] })
	return kf
}

// bande rend la bande de slots d'un archétype — la règle COMBLÉE de production
// (`worldObjectSlotBand` : combler la plage observée, puis retirer tout slot vu porter un AUTRE
// archétype), appliquée au relevé de l'unique marche d'images-clés de l'instrument plutôt qu'en
// relisant le film une fois par archétype. La convention d'exclusion n'est pas réécrite ici :
// elle est dans `slotBandFromCensus`, qui délègue lui-même la règle à `slotBandExcluding`.
func (k failleKF) bande(ti int) map[uint32]bool {
	return slotBandFromCensus(k.seenByTI, ti)
}

// TestFailleActivationEntites balaye les trois canaux d'entités dans les fenêtres des ancres.
func TestFailleActivationEntites(t *testing.T) {
	dir, wr, ancres, origine := failleSetup(t)
	release := LockProcessDecode()
	defer release()
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	prev := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prev })
	SetWorldObjectPrecisionFromLayout(lay)

	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	kf := failleWalkKeyframes(dir, n)
	t.Logf("FILM %s · %d chunks · %d images-clés · %d archétypes recensés",
		dir, n, len(kf.timesUS), len(kf.seenByTI))
	for _, a := range ancres {
		t.Logf("ANCRE %s : slot %d · (%.2f,%.2f) · fenêtre film [%d,%d] ms · saut %d ms",
			a.label, a.slot, a.x, a.y, failleMS(a.t0, origine), failleMS(a.t1, origine), failleMS(a.tJump, origine))
	}
	failleRecensement(t, kf, ancres, origine)
	failleCreations(t, dir, &wr, kf, ancres, n, origine)
	failleDeltas(t, dir, &wr, kf, ancres, n, origine)
}

// failleRecensement (canal 3) : vies recensées pour la première fois dans une fenêtre.
func failleRecensement(t *testing.T, kf failleKF, ancres []failleAncre, origine uint64) {
	t.Helper()
	for _, a := range ancres {
		total, nouvelles := 0, 0
		for ti, lives := range kf.lives {
			for key, times := range lives {
				total++
				first := times[0]
				if first < a.t0 || first > a.t1+failleCensusUS {
					continue
				}
				nouvelles++
				t.Logf("  [%s] RECENSEMENT ti=%d slot=%d gen=%d : 1re image-clé à %d ms (%d vues, dernière %d ms)",
					a.label, ti, key.Slot, key.Gen, failleMS(first, origine), len(times),
					failleMS(times[len(times)-1], origine))
			}
		}
		t.Logf("== [%s] RECENSEMENT : %d vies nouvelles dans [t0, t1+%ds] sur %d vies du film ==",
			a.label, nouvelles, failleCensusUS/1_000_000, total)
	}
}

// failleCreations (canal 1) : records NEW de tous les archétypes lisibles, fenêtres seules.
func failleCreations(t *testing.T, dir string, wr *Vec3Range, kf failleKF, ancres []failleAncre, n int, origine uint64) {
	t.Helper()
	raw0, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw0)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	var cur equipCreationRead
	defer installCreationHooks(&cur)()
	for _, ti := range failleCreationTIs {
		arch, ok := reg.Archetype(int(ti))
		if !ok {
			t.Logf("ti=%d absent du registre : sauté", ti)
			continue
		}
		deser := defaultStateDeserByTI[ti]
		band := kf.bande(int(ti))
		if deser == nil || len(band) == 0 {
			t.Logf("ti=%d : deser=%v bande=%d slots : sauté", ti, deser != nil, len(band))
			continue
		}
		w := equipCreationWalk{comps: len(arch.Components), wr: wr, band: band, cur: &cur, ti: ti, deser: deser}
		failleCreationsPourTI(t, dir, w, ancres, n, ti, origine)
	}
}

// failleCreationsPourTI balaye les paquets delta des fenêtres pour UN archétype et rapporte
// chaque création avec sa distance 2D à l'ancre de sa fenêtre.
func failleCreationsPourTI(t *testing.T, dir string, w equipCreationWalk, ancres []failleAncre, n int, ti uint32, origine uint64) {
	t.Helper()
	var st EquipmentCreationStats
	paquets, creations, proches := 0, 0, 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || len(failleFenetres(pk.TimestampUS, ancres)) == 0 {
				continue
			}
			paquets++
			for _, cre := range w.scanPayload(pk.Payload(data), &st, pk, c) {
				creations++
				for _, ai := range failleFenetres(cre.TimestampUS, ancres) {
					a := ancres[ai]
					d := failleDist2D(cre.X, cre.Y, a)
					marque := ""
					if d <= failleProcheM {
						proches++
						marque = "  <== CANDIDAT (<= 3 m)"
					}
					t.Logf("  [%s] CREATION ti=%d slot=%d gen=%d @%d ms (%.2f,%.2f,%.2f) d=%.1f m MPP32=0x%08x (present=%v)%s",
						a.label, ti, cre.Slot, cre.Gen, failleMS(cre.TimestampUS, origine), cre.X, cre.Y, cre.Z,
						d, uint32(cre.MPPVal[MPPWord32]), cre.MPPPresent[MPPWord32], marque)
				}
			}
		}
	}
	t.Logf("== CREATIONS ti=%d : %d paquets fenêtrés · %d ancres NEW · %d acceptées · %d à <= %.0f m · rejets overflow=%d mask=%d pos=%d ==",
		ti, paquets, st.Anchors, creations, proches, failleProcheM, st.Overflow, st.MaskBad, st.PosBad)
}

// failleDeltas (canal 2) : positions delta des objets du monde (bande UNION), fenêtres seules.
func failleDeltas(t *testing.T, dir string, wr *Vec3Range, kf failleKF, ancres []failleAncre, n int, origine uint64) {
	t.Helper()
	union := map[uint32]bool{}
	bandes := map[int]map[uint32]bool{}
	for _, ti := range failleDeltaTIs {
		bandes[ti] = kf.bande(ti)
		for s := range bandes[ti] {
			union[s] = true
		}
	}
	type cle struct {
		ancre     int
		slot, gen uint32
	}
	type agg struct {
		nb         int
		dmin       float64
		tmin, tmax uint64
	}
	vies := map[cle]*agg{}
	paquets, echantillons := 0, 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || len(failleFenetres(pk.TimestampUS, ancres)) == 0 {
				continue
			}
			paquets++
			for _, s := range scanProjectileRecords(pk.Payload(data), union, wr) {
				echantillons++
				for _, ai := range failleFenetres(pk.TimestampUS, ancres) {
					d := failleDist2D(s.X, s.Y, ancres[ai])
					if d > failleProcheM {
						continue
					}
					k := cle{ancre: ai, slot: s.slot, gen: s.gen}
					v := vies[k]
					if v == nil {
						v = &agg{dmin: d, tmin: pk.TimestampUS, tmax: pk.TimestampUS}
						vies[k] = v
					}
					v.nb++
					if d < v.dmin {
						v.dmin = d
					}
					if pk.TimestampUS < v.tmin {
						v.tmin = pk.TimestampUS
					}
					if pk.TimestampUS > v.tmax {
						v.tmax = pk.TimestampUS
					}
				}
			}
		}
	}
	t.Logf("== DELTAS : bande union %d slots · %d paquets fenêtrés · %d échantillons · %d vies à <= %.0f m d'une ancre ==",
		len(union), paquets, echantillons, len(vies), failleProcheM)
	for k, v := range vies {
		a := ancres[k.ancre]
		var tis []int
		for _, ti := range failleDeltaTIs {
			if bandes[ti][k.slot] {
				tis = append(tis, ti)
			}
		}
		t.Logf("  [%s] VIE slot=%d gen=%d : %d pts · dmin=%.2f m · [%d,%d] ms · bandes ti %v",
			a.label, k.slot, k.gen, v.nb, v.dmin, failleMS(v.tmin, origine), failleMS(v.tmax, origine), tis)
	}
}
