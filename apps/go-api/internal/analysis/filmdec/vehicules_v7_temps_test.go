package filmdec

// vehicules_v7_temps_test.go — INSTRUMENT (lot V7) : LA COINCIDENCE TEMPORELLE entre un type
// d'evenement de tete et la FIN SERREE d'une vie de vehicule.
//
// POURQUOI CET INSTRUMENT EXISTE, ET CE QUE LE PRECEDENT A DIT. `TestV7Refs` cherche le vehicule
// DANS les bits de l'evenement. Il peut echouer meme si l'evenement existe : le lot V6 a montre
// que la ref de domaine 7 d'un embarquement ne prend que QUATRE valeurs sur 22 instances — les
// objets du monde y sont probablement designes par un identifiant de DEFINITION, pas par un slot.
// Si c'est le cas, AUCUN evenement ne « resoudra un vehicule en bande », et la seule prise
// restante est LE TEMPS.
//
// L'ORACLE, ECRIT AVANT LA MESURE. Un evenement qui date la destruction d'un vehicule tombe A
// L'INSTANT ou ce vehicule cesse d'emettre. La FIN SERREE d'une vie est son dernier echantillon
// de position (~0,5 s de precision, contre ~20 s pour le recensement seul) : c'est la primitive
// `v2dTightEnd` du lot V3, refaite ici sur la seule horloge (aucune coordonnee monde n'est
// necessaire, donc `QuantaOnly` — pas de catalogue de bornes).
//
// LES DEUX SENS DE LA MESURE, publies tous les deux :
//
//	SENS A — COUVERTURE DES FINS : parmi les vies a fin serree, quelle part a un evenement de
//	         type T a moins de `v7NearUS` ? Une destruction vraie couvre les vies DETRUITES ;
//	         un vehicule ABANDONNE (la majorite) ne doit PAS l'avoir. Le taux plafonne donc
//	         sous 100 % par construction — c'est l'EXCES SUR LE TEMOIN qui compte.
//	SENS B — PURETE DES EVENEMENTS : parmi les evenements de type T, quelle part tombe a moins
//	         de `v7NearUS` d'une fin serree ? Une destruction vraie est PURE (proche de 100 %) :
//	         chacune de ses occurrences tue un vehicule.
//
// LE TEMOIN, ecrit avant la mesure : la MEME fin serree DECALEE de +60 s (et -60 s), ce qui
// conserve la densite d'evenements du film et ne change QUE l'instant vise. Un type dont le
// sens A et le sens B s'ecroulent au temoin est un candidat ; un type dont le temoin tient est
// du bruit de densite.
//
// Garde d'environnement V7_ROOT / V7_FILMS : sans elle, tout SKIP.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// v7NearUS est la fenetre de coincidence : 1 s. Le flux de position d'un vehicule est echantillonne
// a ~0,5 s (V3 § 1.1), donc la fin serree est connue a une demi-seconde pres ; 1 s la couvre sans
// diluer. La table publie aussi 3 s, la fenetre du lot V3 (`v3dFenetreMS`).
const (
	v7NearUS  = uint64(1_000_000)
	v7Near3US = uint64(3_000_000)
)

// v7EndLife est une vie de vehicule dont la fin est SERREE par le flux de position.
type v7EndLife struct {
	slot    uint32
	lo, hi  uint64 // fenetre de la vie (tolerance de recensement comprise)
	endUS   uint64
	samples int
	bounded bool // la disparition est bornee par une image-cle (preuve d'absence)
}

// v7TightEnds rend, pour un film, les vies de vehicule a fin serree. La fenetre de chaque vie
// reprend EXACTEMENT la regle du calque (`replay.vehicleLives` / `assignVehicleWindows`) :
// tolerance de recensement de part et d'autre, et frontiere partagee entre deux vies d'un meme
// slot — sans quoi le nuage d'une vie mordrait sur sa voisine.
func v7TightEnds(dir string, k WorldObjectKeyframes, pos []BipedPosition) []v7EndLife {
	type win struct {
		slot     uint32
		lo, hi   uint64
		firstUS  uint64
		hasProof bool
	}
	var wins []win
	for key, seen := range k.SeenUS {
		if len(seen) == 0 {
			continue
		}
		w := win{slot: key.Slot, firstUS: seen[0]}
		if seen[0] > v7CensusTolUS {
			w.lo = seen[0] - v7CensusTolUS
		}
		gone := v7FirstAfter(k.TimesUS, seen[len(seen)-1])
		w.hasProof = gone > 0
		if gone == 0 {
			gone = seen[len(seen)-1] + v7CensusTolUS
		}
		w.hi = gone
		wins = append(wins, w)
	}
	sort.Slice(wins, func(i, j int) bool {
		if wins[i].slot != wins[j].slot {
			return wins[i].slot < wins[j].slot
		}
		return wins[i].firstUS < wins[j].firstUS
	})
	for i := range wins {
		if i > 0 && wins[i-1].slot == wins[i].slot && wins[i-1].hi > wins[i].lo {
			wins[i].lo = wins[i-1].hi
		}
		if i+1 < len(wins) && wins[i+1].slot == wins[i].slot && wins[i].hi > wins[i+1].firstUS {
			wins[i].hi = wins[i+1].firstUS
		}
	}
	bySlot := map[uint32][]uint64{}
	for _, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], p.TimestampUS)
	}
	for s := range bySlot {
		v := bySlot[s]
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	}
	var out []v7EndLife
	for _, w := range wins {
		l := v7EndLife{slot: w.slot, lo: w.lo, hi: w.hi, bounded: w.hasProof}
		for _, ts := range bySlot[w.slot] {
			if ts < w.lo || ts > w.hi {
				continue
			}
			l.samples++
			l.endUS = ts
		}
		if l.samples > 0 {
			out = append(out, l)
		}
	}
	return out
}

// v7NearestUS rend l'ecart absolu au plus proche instant d'une liste TRIEE, et false si vide.
func v7NearestUS(times []uint64, at uint64) (uint64, bool) {
	if len(times) == 0 {
		return 0, false
	}
	i := sort.Search(len(times), func(k int) bool { return times[k] >= at })
	best := ^uint64(0)
	if i < len(times) {
		best = times[i] - at
	}
	if i > 0 && at-times[i-1] < best {
		best = at - times[i-1]
	}
	return best, true
}

// v7Compte accumule les deux sens de mesure d'un type, reel et temoins.
type v7Compte struct {
	// SENS A : vies couvertes.
	lives, near1, near3 int
	shiftP, shiftM      int
	// SENS B : evenements purs.
	events, pure1, pure3 int
	pureShiftP           int
}

// v7ScanTemps balaie un film et rend, par type de tete, les instants de ses evenements.
func v7EventTimes(dir string) map[int][]uint64 {
	out := map[int][]uint64{}
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta || p.Size < 1 {
				continue
			}
			ty, present := PacketHeadEventType(p.Payload(data))
			if !present {
				continue
			}
			out[ty] = append(out[ty], p.TimestampUS)
		}
	}
	for ty := range out {
		v := out[ty]
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	}
	return out
}

// TestV7Temps — LA COINCIDENCE. Une ligne par type de tete.
func TestV7Temps(t *testing.T) {
	dirs := v7FilmDirs(t)
	release := LockProcessDecode()
	defer release()
	acc := map[int]*v7Compte{}
	totalLives := 0
	for _, d := range dirs {
		k := ScanFilmWorldObjectKeyframes(d, v0VehiculeTI)
		if len(k.Band) == 0 {
			t.Logf("film %-10s bande ti=%d VIDE — saute", filepath.Base(filepath.Clean(d)), v0VehiculeTI)
			continue
		}
		opt := DefaultScanFilmOptions()
		opt.RequireTag1, opt.MaxSpeedMPS, opt.IsolationGapMS, opt.QuantaOnly = false, 0, 0, true
		pos, err := ScanFilmBipedPositionsForBand(d, NewSlotBand(k.Band), opt)
		if err != nil {
			t.Logf("film %-10s positions illisibles : %v", filepath.Base(filepath.Clean(d)), err)
			continue
		}
		ends := v7TightEnds(d, k, pos)
		evs := v7EventTimes(d)
		totalLives += len(ends)
		t.Logf("film %-10s bande %3d · %3d vies recensees · %3d a fin SERREE · %6d echantillons "+
			"· %2d types de tete", filepath.Base(filepath.Clean(d)), len(k.Band), len(k.SeenUS),
			len(ends), len(pos), len(evs))
		v7Accumule(acc, ends, evs)
		if os.Getenv("V7_DUMP") != "" {
			v7Dump(t, filepath.Base(filepath.Clean(d)), ends, evs)
		}
	}
	v7Publie(t, acc, totalLives, len(dirs))
}

// v7Dump ecrit, vie par vie, LES TYPES D'EVENEMENTS QUI ENTOURENT LA FIN SERREE — et les memes
// autour du TEMOIN a +60 s. C'est la vue qualitative que la table agregee ne donne pas : un type
// systematiquement present a la fin d'une vie s'y voit a l'oeil, meme s'il est rare.
func v7Dump(t *testing.T, film string, ends []v7EndLife, evs map[int][]uint64) {
	t.Helper()
	for _, e := range ends {
		t.Logf("  %s slot %d fin serree %.2f s (%d echantillons, fin bornee %v) : reel [%s] · "+
			"temoin+60 [%s]", film, e.slot, float64(e.endUS)/1e6, e.samples, e.bounded,
			v7Autour(evs, e.endUS), v7Autour(evs, e.endUS+v7ShiftUS))
	}
}

// v7Autour rend la liste « type@ecart_ms » des evenements a moins de v7Near3US d'un instant.
func v7Autour(evs map[int][]uint64, at uint64) string {
	type hit struct {
		ty  int
		dms int64
	}
	var hits []hit
	for ty, times := range evs {
		for _, ts := range times {
			d := int64(ts) - int64(at)
			if d >= -int64(v7Near3US) && d <= int64(v7Near3US) {
				hits = append(hits, hit{ty, d / 1000})
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].dms < hits[j].dms })
	s := ""
	for _, h := range hits {
		s += " " + itoa(h.ty) + "@" + itoa(int(h.dms))
	}
	return s
}

// v7Accumule croise les fins serrees d'un film avec ses evenements.
func v7Accumule(acc map[int]*v7Compte, ends []v7EndLife, evs map[int][]uint64) {
	var endTimes []uint64
	for _, e := range ends {
		endTimes = append(endTimes, e.endUS)
	}
	sort.Slice(endTimes, func(i, j int) bool { return endTimes[i] < endTimes[j] })
	for ty, times := range evs {
		c := acc[ty]
		if c == nil {
			c = &v7Compte{}
			acc[ty] = c
		}
		for _, e := range ends {
			c.lives++
			if d, ok := v7NearestUS(times, e.endUS); ok {
				if d <= v7NearUS {
					c.near1++
				}
				if d <= v7Near3US {
					c.near3++
				}
			}
			if d, ok := v7NearestUS(times, e.endUS+v7ShiftUS); ok && d <= v7NearUS {
				c.shiftP++
			}
			if e.endUS > v7ShiftUS {
				if d, ok := v7NearestUS(times, e.endUS-v7ShiftUS); ok && d <= v7NearUS {
					c.shiftM++
				}
			}
		}
		for _, ts := range times {
			c.events++
			if d, ok := v7NearestUS(endTimes, ts); ok {
				if d <= v7NearUS {
					c.pure1++
				}
				if d <= v7Near3US {
					c.pure3++
				}
			}
			if d, ok := v7NearestUS(endTimes, ts+v7ShiftUS); ok && d <= v7NearUS {
				c.pureShiftP++
			}
		}
	}
}

// v7Publie ecrit la table des deux sens.
func v7Publie(t *testing.T, acc map[int]*v7Compte, lives, films int) {
	t.Helper()
	var tys []int
	for ty := range acc {
		tys = append(tys, ty)
	}
	sort.Ints(tys)
	t.Logf("== V7 TEMPS — %d films · %d vies a fin serree · fenetre %d ms (et %d ms) ==",
		films, lives, v7NearUS/1000, v7Near3US/1000)
	t.Logf("%-5s %-8s | SENS A couverture des fins            | SENS B purete des evenements",
		"type", "n evts")
	t.Logf("%-5s %-8s | %-8s %-8s %-9s %-9s | %-8s %-8s %-9s",
		"", "", "<=1s", "<=3s", "tem +60", "tem -60", "<=1s", "<=3s", "tem +60")
	for _, ty := range tys {
		c := acc[ty]
		if c.lives == 0 || c.events == 0 {
			continue
		}
		pa := func(n int) float64 { return 100 * float64(n) / float64(c.lives) }
		pb := func(n int) float64 { return 100 * float64(n) / float64(c.events) }
		t.Logf("%-5d %-8d | %6.1f %% %6.1f %% %7.1f %% %7.1f %% | %6.1f %% %6.1f %% %7.1f %%",
			ty, c.events, pa(c.near1), pa(c.near3), pa(c.shiftP), pa(c.shiftM),
			pb(c.pure1), pb(c.pure3), pb(c.pureShiftP))
	}
}
