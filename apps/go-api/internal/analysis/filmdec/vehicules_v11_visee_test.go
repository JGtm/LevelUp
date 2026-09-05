package filmdec

// vehicules_v11_visee_test.go — INSTRUMENT DU LOT V11 : LA VISEE **SANS i0** DE L'OCCUPANT.
//
// CE QUE LE BALAYAGE SANS i0 (vehicules_v11_scan_test.go) A TROUVE, ET QUI CHANGE LA QUESTION.
// Sur la bande BIPEDE, les records SANS `i0` ne sont pas du bruit : la forme de masque la plus
// frequente est `i21,i25` (17 411 touches sur `0d76e8f1`, 3 818 sur `fccc61cd`), suivie de
// `i5,i21,i25`. `i21` y est vu 228,8 fois par slot contre **2,1** sur la bande FANTOME — un
// facteur 109. Autrement dit : **le bipede replique son vecteur de visee dans des records qui
// ne portent AUCUNE position**, et ces records-la sont structurellement invisibles a
// `ScanBipedRecords` (qui exige un `i0` absolu ET un masque commencant par 0).
//
// POURQUOI CELA REPOND A LA QUESTION DU CHANTIER. Le modele V1a.4 dit que l'occupant d'un
// vehicule cesse de repliquer sa POSITION monde — c'est la primitive du « trou ». Il ne dit
// rien de sa VISEE. Si le flux `i21` continue pendant le trou, alors le conducteur,
// l'artilleur ET le passager ont une visee lisible pendant qu'ils sont a bord.
//
// LA GRAMMAIRE UTILISEE EST CELLE, DEJA VALIDEE, D'`i21` : `readAimingVectorComponent`
// (R(1) flag0 ; R(12) cap ; R(11) elevation). Elle n'est pas reecrite ici — seul le point
// d'entree change : quand `i21` est le PREMIER composant du masque, sa charge utile commence
// exactement au bit qui suit les index du masque.
//
// LECTURE SEULE : aucun fichier ecrit, aucune base ouverte.
//
//	CGO_ENABLED=0 V11_ROOT=<cache> V11_FILMS=0d76e8f1,fccc61cd \
//	  go test ./internal/analysis/filmdec/ -run TestV11Visee -v -timeout 120m

import (
	"math"
	"math/rand/v2"
	"sort"
	"testing"
)

// v11Visee est UNE lecture de visee issue d'un record SANS i0.
type v11Visee struct {
	Slot     uint32
	TS       uint64
	YawRaw   uint32
	PitchRaw uint32
}

// Cap rend le cap de visee en degres, avec la convention MESUREE d'`AimHeadingDeg`.
func (v v11Visee) Cap() float64 { return 360 * (float64(v.YawRaw) + 0.5) / (1 << aimYawBits) }

// v11TrouMinUS : duree minimale d'un TROU du flux de position, reprise du lot V1a.4 (3 s).
const v11TrouMinUS = 3_000_000

// v11AppariementMaxUS : ecart temporel maximal pour apparier une lecture sans i0 a une lecture
// avec i0 du meme slot. Un cap de visee bouge vite : 200 ms est deja large.
const v11AppariementMaxUS = 200_000

// TestV11ViseeSansI0 — LA MESURE EN TROIS TEMPS :
//
//  1. COMBIEN de lectures `i21` sans i0, et sur quelle bande (temoin FANTOME) ;
//  2. SONT-ELLES JUSTES : appariees aux lectures `i21` AVEC i0 du meme slot a moins de 200 ms,
//     l'ecart de cap doit etre petit ; temoin par permutation deterministe ;
//  3. SURVIVENT-ELLES AU TROU : la part de ces lectures qui tombe DANS un trou du flux de
//     position (>= 3 s) — c'est-a-dire pendant que le bipede est a bord d'un vehicule.
func TestV11ViseeSansI0(t *testing.T) {
	for _, dir := range v11Films(t) {
		v11ViseeUnFilm(t, dir)
	}
}

func v11ViseeUnFilm(t *testing.T, dir string) {
	t.Helper()
	if CountFilmChunks(dir) == 0 {
		t.Logf("V11 %s : film absent — saute", dir)
		return
	}
	release := LockProcessDecode()
	defer release()
	chunks := make([]int, 0, CountFilmChunks(dir))
	for c := 1; c <= CountFilmChunks(dir); c++ {
		chunks = append(chunks, c)
	}
	bande := bipedSlotBandDir(dir, chunks)
	if bande.Count() == 0 {
		t.Logf("V11 %s : bande bipede vide", dir)
		return
	}
	fantomeSet := map[uint32]bool{}
	for _, s := range v11BandeFantome(dir, bande.Count()) {
		fantomeSet[s] = true
	}
	fantome := NewSlotBand(fantomeSet)
	vues := v11LitViseesSansI0(dir, bande)
	bruit := v11LitViseesSansI0(dir, fantome)
	t.Logf("V11 VISEE %s — lectures i21 SANS i0 : bande bipede %d (%.1f par slot, %d slots) · "+
		"TEMOIN FANTOME %d (%.1f par slot, %d slots)",
		dir, len(vues), float64(len(vues))/float64(bande.Count()), bande.Count(),
		len(bruit), float64(len(bruit))/float64(fantome.Count()), fantome.Count())

	opt := ScanFilmOptions{RequireTag1: true, DropSaturated: true, CaptureDirs: true, QuantaOnly: true}
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Logf("V11 %s : balayage bipede : %v", dir, err)
		return
	}
	v11PublieAccord(t, dir, vues, pos)
	v11PublieTrous(t, dir, vues, pos)
}

// v11LitViseesSansI0 balaie le film par le DECODEUR DE PRODUCTION (`ScanBipedAimRecords`,
// offline_aim_only.go) sur la bande demandee. Passer par le code livre — et non par un
// chainage d'instrument — est ce qui rend les chiffres opposables : le temoin FANTOME mesure
// alors le plancher de faux positifs DU CODE, pas d'une copie.
func v11LitViseesSansI0(dir string, bande SlotBand) []v11Visee {
	var out []v11Visee
	for c := 1; c <= CountFilmChunks(dir); c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			for _, a := range ScanBipedAimRecords(pk.Payload(data), bande) {
				out = append(out, v11Visee{Slot: a.Slot, TS: pk.TimestampUS,
					YawRaw: a.YawRaw, PitchRaw: a.PitchRaw})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TS < out[j].TS })
	return out
}

// v11PublieAccord confronte chaque lecture SANS i0 a la lecture AVEC i0 la plus proche dans le
// temps du MEME slot. Une grammaire juste rend un petit ecart de cap ; un decodage desaligne
// rend du bruit. TEMOIN : les memes caps, apparies par PERMUTATION deterministe.
func v11PublieAccord(t *testing.T, dir string, vues []v11Visee, pos []BipedPosition) {
	t.Helper()
	ref := map[uint32][]BipedPosition{}
	for _, p := range pos {
		if p.HasYaw {
			ref[p.Slot] = append(ref[p.Slot], p)
		}
	}
	for s := range ref {
		sort.SliceStable(ref[s], func(i, j int) bool { return ref[s][i].TimestampUS < ref[s][j].TimestampUS })
	}
	var ecarts, caps, refs []float64
	for _, v := range vues {
		r, ok := v11PlusProche(ref[v.Slot], v.TS)
		if !ok {
			continue
		}
		c, _ := r.AimHeadingDeg()
		caps, refs = append(caps, v.Cap()), append(refs, float64(c))
		ecarts = append(ecarts, v11Wrap180(v.Cap()-float64(c)))
	}
	if len(ecarts) == 0 {
		t.Logf("V11 ACCORD %s — aucune paire (sans i0) / (avec i0) a moins de %d us", dir, v11AppariementMaxUS)
		return
	}
	melange := v11Melange(len(caps))
	temoin := make([]float64, len(caps))
	for i := range caps {
		temoin[i] = v11Wrap180(caps[melange[i]] - refs[i])
	}
	m1, r1 := v11Stats(ecarts)
	m2, r2 := v11Stats(temoin)
	t.Logf("V11 ACCORD %s — %d paires · mediane |ecart de cap| %.1f deg · R %.3f\n"+
		"    TEMOIN (appariement detruit par melange) : mediane %.1f deg · R %.3f",
		dir, len(ecarts), m1, r1, m2, r2)
}

// v11PlusProche rend l'echantillon AVEC i0 le plus proche dans le temps, s'il est a moins de
// v11AppariementMaxUS.
func v11PlusProche(s []BipedPosition, ts uint64) (BipedPosition, bool) {
	if len(s) == 0 {
		return BipedPosition{}, false
	}
	i := sort.Search(len(s), func(k int) bool { return s[k].TimestampUS >= ts })
	best, ok := BipedPosition{}, false
	for _, k := range []int{i - 1, i} {
		if k < 0 || k >= len(s) {
			continue
		}
		d := v11Abs(s[k].TimestampUS, ts)
		if d > v11AppariementMaxUS {
			continue
		}
		if !ok || d < v11Abs(best.TimestampUS, ts) {
			best, ok = s[k], true
		}
	}
	return best, ok
}

// v11PublieTrous mesure la part des lectures SANS i0 qui tombe DANS un trou du flux de
// position (>= 3 s) du meme slot : c'est la population « occupant a bord ».
func v11PublieTrous(t *testing.T, dir string, vues []v11Visee, pos []BipedPosition) {
	t.Helper()
	parSlot := map[uint32][]uint64{}
	for _, p := range pos {
		parSlot[p.Slot] = append(parSlot[p.Slot], p.TimestampUS)
	}
	for s := range parSlot {
		sort.Slice(parSlot[s], func(i, j int) bool { return parSlot[s][i] < parSlot[s][j] })
	}
	dans, hors, trous := 0, 0, 0
	for s, ts := range parSlot {
		for i := 0; i+1 < len(ts); i++ {
			if ts[i+1]-ts[i] >= v11TrouMinUS {
				trous++
			}
		}
		_ = s
	}
	for _, v := range vues {
		if v11DansTrou(parSlot[v.Slot], v.TS) {
			dans++
		} else {
			hors++
		}
	}
	t.Logf("V11 TROUS %s — %d trous de position (>= %d us) · lectures i21 sans i0 DANS un trou "+
		"%d (%.1f %%) · hors trou %d",
		dir, trous, v11TrouMinUS, dans, 100*float64(dans)/float64(dans+hors), hors)
}

// v11DansTrou dit si l'instant tombe dans un intervalle >= v11TrouMinUS du flux de position.
func v11DansTrou(ts []uint64, at uint64) bool {
	if len(ts) < 2 {
		return false
	}
	i := sort.Search(len(ts), func(k int) bool { return ts[k] >= at })
	if i == 0 || i >= len(ts) {
		return false
	}
	return ts[i]-ts[i-1] >= v11TrouMinUS
}

// ---------------------------------------------------------------------------------------
// LA VISEE PENDANT UN EPISODE D'OCCUPATION ATTESTE (embarquement -> sortie), PAR SIEGE.
// ---------------------------------------------------------------------------------------

// v11Episode est un episode d'occupation atteste : un embarquement et la sortie qui le ferme,
// pour le MEME slot d'occupant.
type v11Episode struct {
	Slot       uint32
	DebutUS    uint64
	FinUS      uint64
	Siege      uint32
	SiegeValid bool
}

// TestV11ViseeOccupation — LA QUESTION PRODUIT : pendant qu'un joueur est A BORD, sa visee
// est-elle lisible ? Et l'est-elle pour les TROIS roles — conducteur (siege 0), artilleur et
// passager (siege > 0) ?
//
// Les episodes sont ATTESTES par la liste d'evenements (`ScanFilmVehicleEvents`), pas
// devines : embarquement puis premiere sortie du meme occupant. Le siege est celui que
// l'evenement publie.
func TestV11ViseeOccupation(t *testing.T) {
	for _, dir := range v11Films(t) {
		v11OccupationUnFilm(t, dir)
	}
}

func v11OccupationUnFilm(t *testing.T, dir string) {
	t.Helper()
	if CountFilmChunks(dir) == 0 {
		return
	}
	release := LockProcessDecode()
	defer release()
	evs, err := ScanFilmVehicleEvents(dir)
	if err != nil {
		t.Logf("V11 OCCUPATION %s : %v", dir, err)
		return
	}
	chunks := make([]int, 0, CountFilmChunks(dir))
	for c := 1; c <= CountFilmChunks(dir); c++ {
		chunks = append(chunks, c)
	}
	vues := v11LitViseesSansI0(dir, bipedSlotBandDir(dir, chunks))
	parSlot := map[uint32][]v11Visee{}
	for _, v := range vues {
		parSlot[v.Slot] = append(parSlot[v.Slot], v)
	}
	opt := ScanFilmOptions{RequireTag1: true, DropSaturated: true, CaptureDirs: true, QuantaOnly: true}
	pos, _ := ScanFilmBipedPositions(dir, opt)
	posSlot := map[uint32][]uint64{}
	posTous := map[uint32][]uint64{}
	for _, p := range pos {
		posTous[p.Slot] = append(posTous[p.Slot], p.TimestampUS)
		if p.HasYaw {
			posSlot[p.Slot] = append(posSlot[p.Slot], p.TimestampUS)
		}
	}
	for s := range posTous {
		sort.Slice(posTous[s], func(i, j int) bool { return posTous[s][i] < posTous[s][j] })
	}
	eps := v11Episodes(evs, posTous)
	t.Logf("V11 OCCUPATION %s — %d evenements, %d episodes attestes", dir, len(evs), len(eps))
	couverts, total := 0, 0
	for _, e := range eps {
		nSans := v11CompteFenetre(parSlot[e.Slot], e.DebutUS, e.FinUS)
		nAvec := 0
		for _, ts := range posSlot[e.Slot] {
			if ts >= e.DebutUS && ts <= e.FinUS {
				nAvec++
			}
		}
		total++
		if nSans > 0 {
			couverts++
		}
		t.Logf("    episode slot=%d siege=%d(%v) duree=%.1f s · visees SANS i0 = %d "+
			"(%.1f /s) · lectures i21 AVEC i0 = %d",
			e.Slot, e.Siege, e.SiegeValid, float64(e.FinUS-e.DebutUS)/1e6, nSans,
			float64(nSans)/math.Max(1, float64(e.FinUS-e.DebutUS)/1e6), nAvec)
	}
	if total > 0 {
		t.Logf("V11 OCCUPATION %s — episodes avec AU MOINS une visee a bord : %d / %d (%.1f %%)",
			dir, couverts, total, 100*float64(couverts)/float64(total))
	}
}

// v11Episodes construit les episodes d'occupation ATTESTES par la SORTIE.
//
// POURQUOI PAS « embarquement -> sortie ». L'EMBARQUEMENT est rarissime dans la liste
// d'evenements (1 pour 10 sorties sur `0d76e8f1`, mesure de ce lot ; 15 en tout sur 12 films
// au lot V8) : l'apparier ne rendrait presque aucun episode. La SORTIE, elle, est presente
// 105 / 105. Le DEBUT de l'episode est donc pris ou la production le prend depuis V1a.4 : au
// dernier point du flux de position qui precede la sortie, c'est-a-dire a l'ouverture du TROU
// que la sortie ferme.
func v11Episodes(evs []VehicleEvent, pos map[uint32][]uint64) []v11Episode {
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].TimestampUS < evs[j].TimestampUS })
	var out []v11Episode
	for _, e := range evs {
		if e.Kind != EventUnitExitVehicle || !e.OccupantPresent || !e.OccupantInBand {
			continue
		}
		ts := pos[e.OccupantSlot]
		i := sort.Search(len(ts), func(k int) bool { return ts[k] >= e.TimestampUS })
		if i == 0 {
			continue // aucune position avant la sortie : rien a borner
		}
		debut := ts[i-1]
		if e.TimestampUS-debut < v11TrouMinUS {
			continue // pas de trou : la sortie ne ferme pas un episode lisible
		}
		out = append(out, v11Episode{Slot: e.OccupantSlot, DebutUS: debut,
			FinUS: e.TimestampUS, Siege: e.Seat, SiegeValid: e.SeatValid})
	}
	return out
}

// v11CompteFenetre compte les visees d'un slot dans une fenetre temporelle.
func v11CompteFenetre(v []v11Visee, lo, hi uint64) int {
	n := 0
	for _, x := range v {
		if x.TS >= lo && x.TS <= hi {
			n++
		}
	}
	return n
}

// v11Wrap180 ramene un ecart d'angle dans ]-180, 180].
func v11Wrap180(d float64) float64 {
	for d <= -180 {
		d += 360
	}
	for d > 180 {
		d -= 360
	}
	return d
}

// v11Stats rend la mediane des ecarts ABSOLUS et la concentration circulaire R.
func v11Stats(e []float64) (mediane, r float64) {
	abs := make([]float64, len(e))
	var sc, ss float64
	for i, v := range e {
		abs[i] = math.Abs(v)
		sc += math.Cos(v * math.Pi / 180)
		ss += math.Sin(v * math.Pi / 180)
	}
	sort.Float64s(abs)
	n := float64(len(e))
	return abs[len(abs)/2], math.Hypot(sc, ss) / n
}

// v11Melange rend une permutation DETERMINISTE des indices 0..n-1 (graine figee).
func v11Melange(n int) []int {
	o := make([]int, n)
	for i := range o {
		o[i] = i
	}
	r := rand.New(rand.NewPCG(11, 7))
	r.Shuffle(n, func(i, j int) { o[i], o[j] = o[j], o[i] })
	return o
}

// v11Abs rend l'ecart absolu entre deux instants.
func v11Abs(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}
