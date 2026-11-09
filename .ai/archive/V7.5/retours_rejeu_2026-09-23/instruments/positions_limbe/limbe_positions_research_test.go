//go:build research

package replay

// limbe_positions_research_test.go — SONDE D'ENQUETE (lot positions_limbe, 2026-09-23).
//
// LECTURE SEULE de TROIS fichiers de faits persistes (aucun film, aucune ecriture). Question :
// les positions aberrantes publiees (premier point d'une vie de bipede, echantillon isole d'un
// vehicule) tombent-elles dans le MEME PAQUET qu'un record de CREATION du meme slot ?
//
//	go test -tags research -count=1 -run TestLimbePositions -v ./internal/games/halo_infinite/film/replay/

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

const limbeRepo = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`

func TestLimbePositions(t *testing.T) {
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(limbeRepo,
		"data", "titles", "halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fm := range [][2]string{{"81c02726", "Isolation"}, {"ab526724", "Starboard"}, {"879a4dba", "Fortitude"}} {
		entry, err := cat.Lookup(fm[1])
		if err != nil {
			t.Fatal(err)
		}
		blob, err := os.ReadFile(filepath.Join(limbeRepo, "data", "cache", "film_facts", "halo_infinite", fm[0]+".filmfacts.bin"))
		if err != nil {
			t.Fatal(err)
		}
		f, err := DecodeFilmFactsFile(blob, entry)
		blob = nil
		if err != nil {
			t.Fatalf("%s : %v", fm[0], err)
		}
		limbeRapport(t, fm[0], f.Facts.FilmInputs)
	}
}

func limbeD3(a, b grammar.BipedPosition) float64 {
	return math.Sqrt(float64((a.X-b.X)*(a.X-b.X) + (a.Y-b.Y)*(a.Y-b.Y) + (a.Z-b.Z)*(a.Z-b.Z)))
}

func limbeParSlot(pos []grammar.BipedPosition) map[uint32][]grammar.BipedPosition {
	out := map[uint32][]grammar.BipedPosition{}
	for _, p := range pos {
		if p.HasWorld {
			out[p.Slot] = append(out[p.Slot], p)
		}
	}
	for s := range out {
		v := out[s]
		sort.SliceStable(v, func(i, j int) bool { return v[i].TimestampUS < v[j].TimestampUS })
	}
	return out
}

func limbeRapport(t *testing.T, id string, in FilmInputs) {
	origin := uint64(math.MaxUint64)
	for _, p := range in.Positions {
		if p.TimestampUS < origin {
			origin = p.TimestampUS
		}
	}
	fr := func(ts uint64) float64 { return (float64(ts) - float64(origin)) / 1e5 }
	bip := limbeParSlot(in.Positions)
	veh := limbeParSlot(in.Vehicles.Positions)
	parTS := map[uint64][]string{}
	for _, p := range in.Positions {
		parTS[p.TimestampUS] = append(parTS[p.TimestampUS], "b"+limbeItoa(p.Slot))
	}
	for _, p := range in.Vehicles.Positions {
		parTS[p.TimestampUS] = append(parTS[p.TimestampUS], "v"+limbeItoa(p.Slot))
	}
	t.Logf("===== %s : origine=%d bipedes=%d creationsBipede=%d vehicules=%d creationsVehicule=%d",
		id, origin, len(in.Positions), len(in.BipedCreations), len(in.Vehicles.Positions), len(in.Vehicles.Creations))

	// --- BIPEDES : creation -> premieres positions du slot
	var memePaquet, memePaquetAberrant, premierAberrant, total int
	var delais []float64
	cre := append([]grammar.BipedCreation(nil), in.BipedCreations...)
	sort.Slice(cre, func(i, j int) bool { return cre[i].TimestampUS < cre[j].TimestampUS })
	for _, c := range cre {
		total++
		pts := bip[c.Slot]
		k := sort.Search(len(pts), func(i int) bool { return pts[i].TimestampUS+500_000 >= c.TimestampUS })
		if k >= len(pts) {
			t.Logf("  B cre slot=%d gen=%d f=%.1f : AUCUNE position apres", c.Slot, c.Generation, fr(c.TimestampUS))
			continue
		}
		p0 := pts[k]
		aberr := k+1 < len(pts) && limbeD3(p0, pts[k+1]) > 25
		same := p0.TimestampUS == c.TimestampUS
		if same {
			memePaquet++
			if aberr {
				memePaquetAberrant++
			}
		}
		if aberr {
			premierAberrant++
		}
		first := k
		if aberr {
			first = k + 1
		}
		if first < len(pts) {
			delais = append(delais, float64(int64(pts[first].TimestampUS)-int64(c.TimestampUS))/1e6)
		}
		if same || aberr || fr(pts[k].TimestampUS)-fr(c.TimestampUS) > 30 {
			line := ""
			for j := k; j < len(pts) && j < k+3; j++ {
				p := pts[j]
				line += " | f=" + limbeFtoa(fr(p.TimestampUS)) + " dt=" + limbeFtoa(float64(int64(p.TimestampUS)-int64(c.TimestampUS))/1e3) + "ms" +
					" q=" + limbeItoa(p.Q[0]) + "/" + limbeItoa(p.Q[1]) + "/" + limbeItoa(p.Q[2]) +
					" (" + limbeFtoa(float64(p.X)) + "," + limbeFtoa(float64(p.Y)) + "," + limbeFtoa(float64(p.Z)) + ")" +
					" yaw=" + limbeBtoa(p.HasYaw) + " body=" + limbeBtoa(p.HasBody) + " sh=" + limbeBtoa(p.HasShield)
			}
			t.Logf("  B cre slot=%d gen=%d idx=%d f=%.1f memePaquet=%v premierAberrant=%v%s", c.Slot, c.Generation,
				c.ParticipantIndex, fr(c.TimestampUS), same, aberr, line)
		}
	}
	sort.Float64s(delais)
	med := 0.0
	if len(delais) > 0 {
		med = delais[len(delais)/2]
	}
	t.Logf("  BILAN BIPEDES %s : creations=%d memePaquet=%d (dont aberrant=%d) premierAberrant=%d delaiCreation->1rePositionValide min=%.2fs med=%.2fs max=%.2fs",
		id, total, memePaquet, memePaquetAberrant, premierAberrant, limbeFirst(delais), med, limbeLast(delais))

	// --- BIPEDES : vies qui commencent (silence > 5 s) sans creation au meme instant
	for s, pts := range bip {
		for k := range pts {
			if k > 0 && pts[k].TimestampUS-pts[k-1].TimestampUS < 5_000_000 {
				continue
			}
			if k+1 >= len(pts) || limbeD3(pts[k], pts[k+1]) <= 25 {
				continue
			}
			var creAt string
			for _, c := range cre {
				if c.Slot == s && limbeAbsDiff(c.TimestampUS, pts[k].TimestampUS) <= 200_000 {
					creAt = "creation@" + limbeFtoa(float64(int64(pts[k].TimestampUS)-int64(c.TimestampUS))/1e3) + "ms"
				}
			}
			t.Logf("  B DEBUT-ABERRANT slot=%d f=%.1f q=%d/%d/%d (%.2f,%.2f,%.2f) suivant f=%.1f (%.2f,%.2f,%.2f) %s memeTS=%v",
				s, fr(pts[k].TimestampUS), pts[k].Q[0], pts[k].Q[1], pts[k].Q[2], pts[k].X, pts[k].Y, pts[k].Z,
				fr(pts[k+1].TimestampUS), pts[k+1].X, pts[k+1].Y, pts[k+1].Z, creAt, parTS[pts[k].TimestampUS])
		}
	}

	// --- VEHICULES : creations
	vc := append([]types.EquipmentCreation(nil), in.Vehicles.Creations...)
	sort.Slice(vc, func(i, j int) bool { return vc[i].TimestampUS < vc[j].TimestampUS })
	creParSlot := map[uint32][]types.EquipmentCreation{}
	for _, c := range vc {
		creParSlot[c.Slot] = append(creParSlot[c.Slot], c)
		t.Logf("  V cre slot=%d gen=%d f=%.1f bit=%d (%.2f,%.2f,%.2f) mpp32=%#x memeTS=%v", c.Slot, c.Gen, fr(c.TimestampUS),
			c.BitPos, c.X, c.Y, c.Z, c.MPPVal[grammar.MPPWord32], parTS[c.TimestampUS])
	}
	// --- VEHICULES : excursions (aller-retour > 25 m) et leurs champs
	for s, pts := range veh {
		for k := 1; k+1 < len(pts); k++ {
			a, b, c := pts[k-1], pts[k], pts[k+1]
			if limbeD3(a, b) <= 25 || limbeD3(b, c) <= 25 || limbeD3(a, c) > 0.2*math.Min(limbeD3(a, b), limbeD3(b, c)) {
				continue
			}
			var creAt string
			for _, cr := range creParSlot[s] {
				if limbeAbsDiff(cr.TimestampUS, b.TimestampUS) <= 200_000 {
					creAt += " creation@" + limbeFtoa(float64(int64(b.TimestampUS)-int64(cr.TimestampUS))/1e3) + "ms(" +
						limbeFtoa(float64(cr.X)) + "," + limbeFtoa(float64(cr.Y)) + "," + limbeFtoa(float64(cr.Z)) + ")"
				}
			}
			kfAv, kfAp := limbeKfAutour(in.Vehicles.Keyframes.TimesUS, b.TimestampUS)
			vu := ""
			for key, seen := range in.Vehicles.Keyframes.SeenUS {
				if key.Slot != s {
					continue
				}
				for _, ts := range seen {
					if ts == kfAv || ts == kfAp {
						vu += " recense@" + limbeFtoa(fr(ts)) + "(gen" + limbeItoa(key.Gen) + ")"
					}
				}
			}
			t.Logf("  V EXCURSION slot=%d f=%.1f q=%d/%d/%d (%.2f,%.2f,%.2f) avant f=%.1f apres f=%.1f champs=%s | avant=%s | apres=%s |%s kf=[%.1f,%.1f]%s memeTS=%v",
				s, fr(b.TimestampUS), b.Q[0], b.Q[1], b.Q[2], b.X, b.Y, b.Z, fr(a.TimestampUS), fr(c.TimestampUS),
				limbeChamps(b), limbeChamps(a), limbeChamps(c), creAt, fr(kfAv), fr(kfAp), vu, parTS[b.TimestampUS])
		}
	}
}

func limbeChamps(p grammar.BipedPosition) string {
	return "vel=" + limbeBtoa(p.HasVel) + ":" + limbeItoa(p.VelRaw) + "/" + limbeItoa(p.VelScale) +
		" aim=" + limbeBtoa(p.HasAim) + ":" + limbeItoa(p.AimRaw) + " roll=" + limbeBtoa(p.HasRoll) + ":" + limbeItoa(p.RollRaw) +
		" mode=" + limbeItoa(uint32(p.FwdMode)) + " def=" + limbeBtoa(p.AimDefault) + " body=" + limbeBtoa(p.HasBody) +
		" sh=" + limbeBtoa(p.HasShield) + " over=" + limbeBtoa(p.MaskOver)
}

func limbeKfAutour(times []uint64, ts uint64) (uint64, uint64) {
	i := sort.Search(len(times), func(k int) bool { return times[k] > ts })
	var av, ap uint64
	if i > 0 {
		av = times[i-1]
	}
	if i < len(times) {
		ap = times[i]
	}
	return av, ap
}

func limbeAbsDiff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

func limbeFirst(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	return v[0]
}

func limbeLast(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	return v[len(v)-1]
}

func limbeItoa(v uint32) string { return limbeFtoa(float64(v)) }

func limbeBtoa(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func limbeFtoa(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return limbeFmtInt(int64(v))
	}
	return limbeFmtFloat(v)
}

func limbeFmtInt(v int64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := ""
	if v == 0 {
		s = "0"
	}
	for v > 0 {
		s = string(rune('0'+v%10)) + s
		v /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func limbeFmtFloat(v float64) string {
	r := math.Round(v*100) / 100
	ip := math.Trunc(r)
	fp := math.Abs(math.Round((r - ip) * 100))
	s := limbeFmtInt(int64(ip))
	if r < 0 && ip == 0 {
		s = "-0"
	}
	d := limbeFmtInt(int64(fp))
	if len(d) < 2 {
		d = "0" + d
	}
	return s + "." + d
}

// TestLimbeCreationEtSignatures : (1) pour chaque point de bipede qui ouvre une vie apres un
// silence > 5 s, la creation du MEME slot la plus proche avant et apres ; (2) la part des points
// qui PRECEDENT la creation de leur vie ; (3) la signature des champs captures (vehicules).
func TestLimbeCreationEtSignatures(t *testing.T) {
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(limbeRepo,
		"data", "titles", "halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fm := range [][2]string{{"81c02726", "Isolation"}, {"ab526724", "Starboard"}, {"879a4dba", "Fortitude"}} {
		entry, err := cat.Lookup(fm[1])
		if err != nil {
			t.Fatal(err)
		}
		blob, err := os.ReadFile(filepath.Join(limbeRepo, "data", "cache", "film_facts", "halo_infinite", fm[0]+".filmfacts.bin"))
		if err != nil {
			t.Fatal(err)
		}
		f, err := DecodeFilmFactsFile(blob, entry)
		blob = nil
		if err != nil {
			t.Fatalf("%s : %v", fm[0], err)
		}
		limbeCreations(t, fm[0], f.Facts.FilmInputs)
		limbeSignatures(t, fm[0], f.Facts.FilmInputs)
	}
}

func limbeCreations(t *testing.T, id string, in FilmInputs) {
	origin := uint64(math.MaxUint64)
	for _, p := range in.Positions {
		if p.TimestampUS < origin {
			origin = p.TimestampUS
		}
	}
	fr := func(ts uint64) float64 { return (float64(ts) - float64(origin)) / 1e5 }
	bip := limbeParSlot(in.Positions)
	creParSlot := map[uint32][]uint64{}
	for _, c := range in.BipedCreations {
		creParSlot[c.Slot] = append(creParSlot[c.Slot], c.TimestampUS)
	}
	for s := range creParSlot {
		v := creParSlot[s]
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	}
	var ouvertures, avantCreation, avantCreationAberrant, apresCreationAberrant int
	for s, pts := range bip {
		for k := range pts {
			if k > 0 && pts[k].TimestampUS-pts[k-1].TimestampUS < 5_000_000 {
				continue
			}
			ouvertures++
			aberr := k+1 < len(pts) && limbeD3(pts[k], pts[k+1]) > 25
			// la creation du slot la plus proche APRES ce point, et AVANT
			cs := creParSlot[s]
			i := sort.Search(len(cs), func(j int) bool { return cs[j] >= pts[k].TimestampUS })
			apres, avant := -1.0, -1.0
			if i < len(cs) {
				apres = float64(cs[i]-pts[k].TimestampUS) / 1e6
			}
			if i > 0 {
				avant = float64(pts[k].TimestampUS-cs[i-1]) / 1e6
			}
			// « avant la creation » : une creation du slot tombe dans les 30 s qui SUIVENT ce point
			// et AUCUNE dans les 2 s qui le precedent.
			pre := apres >= 0 && apres < 30 && !(avant >= 0 && avant < 2)
			if pre {
				avantCreation++
				if aberr {
					avantCreationAberrant++
				}
			} else if aberr {
				apresCreationAberrant++
			}
			if pre || aberr {
				prevGap := -1.0
				if k > 0 {
					prevGap = float64(pts[k].TimestampUS-pts[k-1].TimestampUS) / 1e6
				}
				nxt := ""
				if k+1 < len(pts) {
					nxt = "suivant f=" + limbeFtoa(fr(pts[k+1].TimestampUS)) + " (" + limbeFtoa(float64(pts[k+1].X)) + "," +
						limbeFtoa(float64(pts[k+1].Y)) + "," + limbeFtoa(float64(pts[k+1].Z)) + ") d=" + limbeFtoa(limbeD3(pts[k], pts[k+1]))
				}
				t.Logf("  B OUVERTURE %s slot=%d f=%.1f (%.2f,%.2f,%.2f) q=%d/%d/%d aberrant=%v silenceAvant=%.1fs creationAvant=%.2fs creationApres=%.2fs yaw=%v body=%v sh=%v %s",
					id, s, fr(pts[k].TimestampUS), pts[k].X, pts[k].Y, pts[k].Z, pts[k].Q[0], pts[k].Q[1], pts[k].Q[2],
					aberr, prevGap, avant, apres, pts[k].HasYaw, pts[k].HasBody, pts[k].HasShield, nxt)
			}
		}
	}
	t.Logf("  BILAN OUVERTURES %s : ouvertures=%d precedentLaCreation=%d (dont aberrantes=%d) aberrantesHorsPreCreation=%d",
		id, ouvertures, avantCreation, avantCreationAberrant, apresCreationAberrant)
}

func limbeSignatures(t *testing.T, id string, in FilmInputs) {
	veh := limbeParSlot(in.Vehicles.Positions)
	sig := func(p grammar.BipedPosition) string {
		return "vel" + limbeBtoa(p.HasVel) + "aim" + limbeBtoa(p.HasAim) + "roll" + limbeBtoa(p.HasRoll) +
			"def" + limbeBtoa(p.AimDefault) + "body" + limbeBtoa(p.HasBody) + "sh" + limbeBtoa(p.HasShield)
	}
	tous := map[string]int{}
	aberr := map[string]int{}
	n := 0
	for _, pts := range veh {
		for k := range pts {
			n++
			tous[sig(pts[k])]++
			if k > 0 && k+1 < len(pts) {
				a, b, c := pts[k-1], pts[k], pts[k+1]
				if limbeD3(a, b) > 25 && limbeD3(b, c) > 25 && limbeD3(a, c) < 0.2*math.Min(limbeD3(a, b), limbeD3(b, c)) {
					aberr[sig(b)]++
				}
			}
		}
	}
	t.Logf("  SIGNATURES VEHICULES %s : echantillons=%d tous=%v aberrants=%v", id, n, tous, aberr)
	bip := limbeParSlot(in.Positions)
	btous := map[string]int{}
	for _, pts := range bip {
		for _, p := range pts {
			btous["yaw"+limbeBtoa(p.HasYaw)+"body"+limbeBtoa(p.HasBody)+"sh"+limbeBtoa(p.HasShield)]++
		}
	}
	t.Logf("  SIGNATURES BIPEDES %s : %v", id, btous)
}
