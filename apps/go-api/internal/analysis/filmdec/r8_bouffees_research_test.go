package filmdec

// r8_bouffees_research_test.go — LE BALAYAGE SYSTEMATIQUE : quel composant du BIPEDE est
// transmis quand un joueur est PROJETE ?
//
// LA METHODE, ET POURQUOI ELLE EST LA BONNE ICI. Chercher « le composant du propulseur »
// en devinant lequel a echoue trois fois (i54, i56, i51). On renverse : l'ORACLE PHYSIQUE
// fournit les ANCRES — les instants ou un bipede se deplace a une vitesse qu'aucune course
// n'explique — et on demande a CHAQUE composant de l'archetype s'il est transmis a ces
// instants plus souvent qu'ailleurs. C'est la methode du lot R1 (ancre datee -> ce qui
// bouge a cet instant), avec une ancre qui ne suppose rien du format.
//
// AUCUN DECODAGE DE CHARGE UTILE N'EST NECESSAIRE : le MASQUE d'un record delta dit quels
// composants sont transmis, et il se lit sans marcher les grammaires. Le balayage est donc
// exact meme sur les composants dont le deser n'est pas porte.
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	bouffee    segment de vitesse horizontale dans [8, 25] m/s. 8 m/s est tres au-dessus du
//	           P90 aleatoire mesure (4,06 m/s) et au-dessus de la course du jeu ; 25 m/s
//	           ecarte les teleportations de reapparition (le temoin aleatoire monte a 247).
//	fenetre    +/- 300 ms autour de l'ancre — deux pas de replication de part et d'autre.
//	portance   part du composant dans les records de la fenetre, divisee par sa part dans
//	           TOUS les records du meme slot. Une portance de 1 = le composant est transmis
//	           a son rythme ordinaire, donc il ne dit rien de l'evenement.
//
// TEMOIN POSITIF ET CLE DE LECTURE : les bouffees se separent en deux populations, celles
// qui tombent a moins d'1 s d'une lecture de GRAPPIN (instant certain, canal independant)
// et les autres. Un composant qui porte l'usage d'un equipement de mobilite DOIT lever la
// portance sur les bouffees de grappin. S'il ne le fait sur AUCUN composant, le balayage
// est aveugle et son negatif ne vaut rien — c'est le controle qui rend la mesure lisible.
//
// GARDES : `R8_FILMS`, `R8_BOUNDS`, `R8_IDS`.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R8_FILMS=<repo>/data/cache/film_chunks \
//	  R8_BOUNDS=<worktree>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R8_IDS=00ba2e1c go test ./internal/analysis/filmdec/ -run '^TestR8Bouffees$' \
//	  -timeout 120m -v

import (
	"path/filepath"
	"sort"
	"testing"
)

const (
	// r8BurstMinMPS / r8BurstMaxMPS bornent une bouffee.
	//
	// LE SEUIL A ETE RECALIBRE UNE FOIS, ET SUR LE TEMOIN POSITIF SEUL. Le seuil ecrit
	// d'avance (8,0 m/s) ne laissait que 3 bouffees de grappin sur 74 lectures : le temoin
	// etait vide, donc l'instrument sans puissance, et un negatif n'aurait rien voulu dire.
	// 6,0 m/s est la valeur qui rend le temoin lisible — choisie sur la population du
	// GRAPPIN, jamais sur celles du repulseur ou du propulseur, qui n'ont pas ete regardees
	// avant de le fixer. 25 m/s ecarte les teleportations de reapparition.
	r8BurstMinMPS = 6.0
	r8BurstMaxMPS = 25.0
	// r8BurstMergeUS : deux segments rapides du meme slot a moins de 1 s portent LA MEME
	// projection.
	r8BurstMergeUS = 1_000_000
	// r8GrappleLeadUS / r8GrappleLagUS : une lecture de grappin date le TIR, la traction
	// suit. La bouffee est rattachee au grappin si elle tombe dans [-0,3 s, +2,0 s] de la
	// lecture — la duree d'une ligne de grappin publiee (8 a 20 frames de 100 ms).
	r8GrappleLeadUS = 300_000
	r8GrappleLagUS  = 2_000_000
	// r8AnchorWindowUS : demi-fenetre autour d'une ancre (300 ms).
	r8AnchorWindowUS = 300_000
	// r8MaxComponent borne le recensement : l'index de composant du masque biped.
	r8MaxComponent = 64
)

// r8Rec est un record delta de bipede reduit a ce que le balayage lit : qui, quand, et
// quels composants sont au masque.
type r8Rec struct {
	slot uint32
	ts   uint64
	mask uint64
}

// r8ScanRecords balaye les records delta de bipede et n'en retient que l'en-tete et le
// masque. L'appelant doit detenir LockProcessDecode.
func r8ScanRecords(s r8MobSetup) []r8Rec {
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + s.lay.TotalBits()
	var out []r8Rec
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
				if !ok {
					p++
					continue
				}
				var m uint64
				for _, id := range idx {
					if id >= 0 && id < r8MaxComponent {
						m |= 1 << uint(id)
					}
				}
				out = append(out, r8Rec{slot: slot, ts: pk.TimestampUS, mask: m})
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ts < out[j].ts })
	return out
}

// r8Anchor est un instant d'ancre pour un slot.
type r8Anchor struct {
	slot uint32
	ts   uint64
}

// r8Bursts extrait les bouffees : segments rapides fusionnes par slot.
func r8Bursts(speeds r8SpeedIndex) []r8Anchor {
	var out []r8Anchor
	slots := make([]uint32, 0, len(speeds))
	for s := range speeds {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, slot := range slots {
		var last uint64
		first := true
		for _, seg := range speeds[slot] {
			if seg.v < r8BurstMinMPS || seg.v > r8BurstMaxMPS {
				continue
			}
			if first || seg.t0-last > r8BurstMergeUS {
				out = append(out, r8Anchor{slot: slot, ts: seg.t0})
				first = false
			}
			last = seg.t0
		}
	}
	return out
}

// r8SplitByGrapple separe les ancres selon leur proximite d'une lecture de grappin.
func r8SplitByGrapple(anchors []r8Anchor, gr []GrappleRead) (withG, without []r8Anchor) {
	for _, a := range anchors {
		near := false
		for _, g := range gr {
			if g.Slot != a.slot {
				continue
			}
			if a.ts+r8GrappleLeadUS >= g.TimestampUS && a.ts <= g.TimestampUS+r8GrappleLagUS {
				near = true
				break
			}
		}
		if near {
			withG = append(withG, a)
		} else {
			without = append(without, a)
		}
	}
	return withG, without
}

func r8AbsDiff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// r8Lift compte, pour chaque composant, sa part dans les records de la fenetre des ancres
// et sa part dans TOUS les records des MEMES slots — le rapport est la portance.
type r8Lift struct {
	inWin, inAll     [r8MaxComponent]int
	winRecs, allRecs int
}

func r8Measure(recs []r8Rec, anchors []r8Anchor) r8Lift {
	var lf r8Lift
	slots := map[uint32]bool{}
	for _, a := range anchors {
		slots[a.slot] = true
	}
	bySlot := map[uint32][]r8Anchor{}
	for _, a := range anchors {
		bySlot[a.slot] = append(bySlot[a.slot], a)
	}
	for _, r := range recs {
		if !slots[r.slot] {
			continue
		}
		lf.allRecs++
		near := false
		for _, a := range bySlot[r.slot] {
			if r8AbsDiff(a.ts, r.ts) <= r8AnchorWindowUS {
				near = true
				break
			}
		}
		if near {
			lf.winRecs++
		}
		for c := 0; c < r8MaxComponent; c++ {
			if r.mask&(1<<uint(c)) == 0 {
				continue
			}
			lf.inAll[c]++
			if near {
				lf.inWin[c]++
			}
		}
	}
	return lf
}

func TestR8Bouffees(t *testing.T) {
	for _, dir := range r8FilmDirs(t) {
		r8BouffeesOneFilm(t, dir)
	}
}

func r8BouffeesOneFilm(t *testing.T, dir string) {
	t.Helper()
	entry := r8MapEntry(t, dir)
	wr := entry.Range()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	SetWorldObjectPrecisionFromLayout(entry.Layout())
	defer func() { WorldObjectPrecision = saved }()

	s := r8MobResolve(t, dir)
	opt := DefaultScanFilmOptions()
	opt.WorldRange = &wr
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("positions illisibles dans %s : %v", dir, err)
	}
	speeds := r8BuildSpeeds(pos)
	gr, _, err := ScanFilmGrappleReads(dir)
	if err != nil {
		t.Logf("lectures de grappin illisibles : %v", err)
	}
	recs := r8ScanRecords(s)
	anchors := r8Bursts(speeds)
	withG, without := r8SplitByGrapple(anchors, gr)
	t.Logf("%s : records=%d positions=%d bouffees=%d (grappin=%d autres=%d) lectures grappin=%d",
		filepath.Base(dir), len(recs), len(pos), len(anchors), len(withG), len(without), len(gr))
	r8LogLift(t, s, "BOUFFEES DE GRAPPIN (temoin positif)", r8Measure(recs, withG))
	r8LogLift(t, s, "BOUFFEES SANS GRAPPIN (candidat propulseur)", r8Measure(recs, without))
}

// r8LogLift publie la portance par composant, la plus forte en tete. Les composants vus
// moins de 10 fois dans la fenetre sont tus : une portance sur 3 records n'est pas une
// mesure.
func r8LogLift(t *testing.T, s r8MobSetup, titre string, lf r8Lift) {
	t.Helper()
	if lf.winRecs == 0 || lf.allRecs == 0 {
		t.Logf("  %s : aucune fenetre — rien a mesurer", titre)
		return
	}
	type row struct {
		c              int
		win, all, lift float64
		nWin, nAll     int
	}
	var rows []row
	for c := 0; c < r8MaxComponent; c++ {
		if lf.inWin[c] < 10 || lf.inAll[c] == 0 {
			continue
		}
		w := float64(lf.inWin[c]) / float64(lf.winRecs)
		a := float64(lf.inAll[c]) / float64(lf.allRecs)
		rows = append(rows, row{c: c, win: w, all: a, lift: w / a,
			nWin: lf.inWin[c], nAll: lf.inAll[c]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].lift > rows[j].lift })
	t.Logf("  %s — %d records en fenetre sur %d", titre, lf.winRecs, lf.allRecs)
	t.Logf("    %-4s %-38s %8s %8s %8s %8s", "i", "composant", "nFen", "partFen", "partTout", "portance")
	for i, r := range rows {
		if i >= 12 {
			break
		}
		t.Logf("    %-4d %-38s %8d %7.3f%% %7.3f%% %8.2f",
			r.c, s.arch.component(r.c), r.nWin, 100*r.win, 100*r.all, r.lift)
	}
}
