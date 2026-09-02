package filmdec

// respawn_coverage_research_test.go — LA COUVERTURE DU COMPTEUR DE RESPAWN, voie bande
// débruitée par la calibration.
//
// LA CALIBRATION EST ACQUISE (2026-09-02, respawn_calibration_research_test.go, ancres
// user) : T0 = secondes RESTANTES (entier, -1/s), T1 = durée TOTALE (8 ou 10 s). LA
// QUESTION D'ICI : peut-on lire ce compteur pour CHAQUE mort ? La chaîne séquentielle
// (records certains) ne couvre qu'un tiers des paquets — 1 à 7 lectures par film pour une
// centaine de morts. La voie BANDE voit tous les paquets mais son ancrage est probabiliste
// (301 en-têtes/slot contre 283 sur bande vide, cf. game_entities_chain). LE DÉBRUITAGE
// EST LA CALIBRATION MÊME : une lecture de bruit tire ses 21 bits au hasard —
// P(actif ET T1 ∈ {8,10} ET T0 <= T1) ~ 1e-5 — quand une vraie lecture y tombe TOUJOURS.
//
// LA JOINTURE AUX MORTS NE PASSE PAS PAR LE SLOT : le slot d'entité ti=5 n'a pas de pont
// établi vers le joueur. Chaque épisode date sa MORT D'ORIGINE par sa propre arithmétique
// — une lecture (t, T0=k, T1=D) place la mort à t − (D − k) s, à la latence d'affichage
// près — et se joint à l'unique mort du fil dans la fenêtre. Deux morts candidates =
// CONTESTÉ, on s'abstient (la règle des fermetures du pont).
//
// LES MORTS ET LEUR CALAGE D'HORLOGE SONT DES COPIES D'INSTRUMENT du canon de
// `analysis/replay` (deaths_source.go, lives.go) : ce paquet ne peut pas l'importer
// (cycle), et une sonde ne fait pas production — le canon reste là-bas, cette copie meurt
// avec la sonde au portage.
//
// LECTURE SEULE, UN SEUL FILM par processus (D17), gardé par GAME_FILM — sauté en CI.
//
// USAGE (depuis apps/go-api) :
//
//	GAME_FILM=C:/.../data/cache/film_chunks/00162144 \
//	  go test ./internal/analysis/filmdec/ -run '^TestRespawnCoveragePhase1$' -timeout 30m -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis"
)

const (
	// respawnJoinWindowMS : demi-fenêtre d'appariement épisode -> mort. La latence
	// d'affichage observée est ~1 s (première lecture à N-1) ; 1 500 ms couvre sa
	// dispersion sans engloutir la mort voisine (le respawn minimal est de 8 s).
	respawnJoinWindowMS = 1_500
	// respawnDeathWindowMS / respawnLifeGapUS : les constantes du canon (lives.go).
	respawnDeathWindowMS = 150
	respawnLifeGapUS     = 5_000_000
)

// respawnObs : une lecture plausible du compteur, datée sur l'horloge du film.
type respawnObs struct {
	slot   uint32
	tUS    uint64
	t0, t1 uint16
	chain  bool
}

func TestRespawnCoveragePhase1(t *testing.T) {
	dir := os.Getenv(gameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", gameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	obs, rawBand, rawChain := respawnCollect(t, dir)
	nChain := 0
	for _, o := range obs {
		if o.chain {
			nChain++
		}
	}
	t.Logf("LECTURES : bande %d dont PLAUSIBLES %d · chaine %d dont PLAUSIBLES %d",
		rawBand, len(obs)-nChain, rawChain, nChain)

	episodes := respawnGroupObs(obs)
	t.Logf("EPISODES : %d", len(episodes))

	deaths, off, matched := respawnDeathsAndOffset(t, dir)
	t.Logf("MORTS DU FIL : %d (offset film %d ms, %d appariees au calage)", len(deaths), off, matched)

	respawnJoinReport(t, episodes, deaths, off)
}

// respawnCollect lit les deux voies et ne garde que les lectures CALIBRÉES-PLAUSIBLES :
// actives, T1 ∈ {8, 10}, T0 <= T1. Le taux de rejet de la bande est publié — c'est lui
// qui dit si le débruitage par valeur tient sa promesse.
func respawnCollect(t *testing.T, dir string) (obs []respawnObs, rawBand, rawChain int) {
	t.Helper()
	sc, err := ScanFilmGameEntities(dir)
	if err != nil {
		t.Fatalf("voie bande impossible : %v", err)
	}
	for _, r := range sc.Player {
		if !r.HasRespawn {
			continue
		}
		rawBand++
		if respawnPlausible(r.Respawn) {
			obs = append(obs, respawnObs{r.Slot, r.TimestampUS, r.Respawn.T0, r.Respawn.T1, false})
		}
	}
	recs, _, _, err := ScanFilmGameEntitiesChain(dir)
	if err != nil {
		t.Fatalf("voie chaine impossible : %v", err)
	}
	for _, r := range recs {
		if r.TI != PlayerEngineTypeIndex || !r.HasRespawn {
			continue
		}
		rawChain++
		if respawnPlausible(r.Respawn) {
			obs = append(obs, respawnObs{r.Slot, r.TimestampUS, r.Respawn.T0, r.Respawn.T1, true})
		}
	}
	sort.Slice(obs, func(i, j int) bool { return obs[i].tUS < obs[j].tUS })
	return obs, rawBand, rawChain
}

func respawnPlausible(rt RespawnTimer) bool {
	return rt.Active && (rt.T1 == 8 || rt.T1 == 10) && rt.T0 <= rt.T1
}

// respawnEpisodeObs : un épisode = des lectures d'un même slot, T1 constant, T0 qui ne
// remonte jamais, bornées par un trou. La COHÉRENCE -1/s se vérifie lecture à lecture
// (|ΔT0 + Δt| <= 1 s de tolérance d'échantillonnage) : un épisode incohérent est du bruit
// qui a passé le filtre de valeur — il se compte, il ne se joint pas.
type respawnEpisodeObs struct {
	slot     uint32
	readings []respawnObs
	coherent bool
}

func respawnGroupObs(obs []respawnObs) []respawnEpisodeObs {
	bySlot := map[uint32][]respawnObs{}
	for _, o := range obs {
		bySlot[o.slot] = append(bySlot[o.slot], o)
	}
	slots := make([]uint32, 0, len(bySlot))
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []respawnEpisodeObs
	for _, s := range slots {
		rs := bySlot[s]
		cur := respawnEpisodeObs{slot: s, coherent: true}
		flush := func() {
			if len(cur.readings) > 0 {
				out = append(out, cur)
			}
			cur = respawnEpisodeObs{slot: s, coherent: true}
		}
		for _, r := range rs {
			if n := len(cur.readings); n > 0 {
				prev := cur.readings[n-1]
				dt := float64(r.tUS-prev.tUS) / 1e6
				if dt > 2.5 || r.t1 != prev.t1 || r.t0 > prev.t0 {
					flush()
				} else if d := float64(prev.t0-r.t0) - dt; d > 1.0 || d < -1.0 {
					cur.coherent = false
				}
			}
			cur.readings = append(cur.readings, r)
		}
		flush()
	}
	return out
}

// respawnDeathsAndOffset — COPIE D'INSTRUMENT (canon : replay/deaths_source.go +
// replay/lives.go) : les morts du dernier chunk, les fins de vie des positions, et le
// plateau qui cale les deux horloges.
func respawnDeathsAndOffset(t *testing.T, dir string) (deaths []analysis.HighlightEvent, off int64, matched int) {
	t.Helper()
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film lisible dans %s", dir)
	}
	raw, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", n)))
	if err != nil {
		t.Fatalf("chunk highlight : %v", err)
	}
	evs, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		t.Fatalf("parse highlight : %v", err)
	}
	for _, e := range evs {
		if e.EventType == analysis.EventTypeDeath {
			deaths = append(deaths, e)
		}
	}
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].TimeMS < deaths[j].TimeMS })

	// QuantaOnly : seuls les HORODATAGES servent au calage — aucune borne de carte requise.
	scan := DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions : %v", err)
	}
	sort.SliceStable(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	ends := respawnLifeEndsMS(pos)
	off, matched = respawnBestOffset(ends, deaths)
	return deaths, off, matched
}

// respawnLifeEndsMS : les fins de vie (ms, horloge film) — découpe au trou de 5 s.
func respawnLifeEndsMS(pos []BipedPosition) []int64 {
	lastBySlot := map[uint32]uint64{}
	var ends []int64
	for _, p := range pos {
		if last, ok := lastBySlot[p.Slot]; ok && p.TimestampUS-last > respawnLifeGapUS {
			ends = append(ends, int64(last/1000))
		}
		lastBySlot[p.Slot] = p.TimestampUS
	}
	for _, last := range lastBySlot {
		ends = append(ends, int64(last/1000))
	}
	sort.Slice(ends, func(i, j int) bool { return ends[i] < ends[j] })
	return ends
}

// respawnBestOffset : le plateau du calage (copie d'instrument de bestDeathOffset).
func respawnBestOffset(ends []int64, deaths []analysis.HighlightEvent) (int64, int) {
	if len(ends) == 0 || len(deaths) == 0 {
		return 0, 0
	}
	lo, hi := ends[0], ends[0]
	for _, e := range ends {
		if e < lo {
			lo = e
		}
		if e > hi {
			hi = e
		}
	}
	bestN := -1
	var plateau []int64
	for off := lo - 60_000; off <= hi; off += 10 {
		n := respawnCountMatches(ends, deaths, off)
		if n > bestN {
			bestN, plateau = n, []int64{off}
		} else if n == bestN {
			plateau = append(plateau, off)
		}
	}
	return plateau[len(plateau)/2], bestN
}

func respawnCountMatches(ends []int64, deaths []analysis.HighlightEvent, off int64) int {
	used := make([]bool, len(ends))
	n := 0
	for _, d := range deaths {
		target := int64(d.TimeMS) + off
		bi, bd := -1, int64(respawnDeathWindowMS+1)
		for i, e := range ends {
			if used[i] {
				continue
			}
			delta := e - target
			if delta < 0 {
				delta = -delta
			}
			if delta < bd {
				bd, bi = delta, i
			}
		}
		if bi >= 0 {
			used[bi] = true
			n++
		}
	}
	return n
}

// respawnJoinReport apparie chaque épisode cohérent à l'unique mort de sa fenêtre, et
// publie les dénominateurs : morts couvertes, contestations, orphelins, partage 8/10 s.
func respawnJoinReport(t *testing.T, episodes []respawnEpisodeObs, deaths []analysis.HighlightEvent, off int64) {
	t.Helper()
	matched, contested, orphan, incoherent := 0, 0, 0, 0
	coveredDeaths := map[int]bool{}
	dur8, dur10 := 0, 0
	for _, ep := range episodes {
		if !ep.coherent {
			incoherent++
			continue
		}
		// La mort d'origine, datée par CHAQUE lecture puis moyennée : (t, T0=k, T1=D)
		// place la mort à t − (D − k) secondes, à la latence d'affichage près.
		var sum float64
		for _, r := range ep.readings {
			sum += float64(r.tUS)/1e3 - float64(r.t1-r.t0)*1e3
		}
		estMS := int64(sum / float64(len(ep.readings)))
		var cands []int
		for i, d := range deaths {
			filmMS := int64(d.TimeMS) + off
			delta := estMS - filmMS
			if delta >= -respawnJoinWindowMS && delta <= 2*respawnJoinWindowMS {
				cands = append(cands, i)
			}
		}
		switch len(cands) {
		case 0:
			orphan++
		case 1:
			matched++
			coveredDeaths[cands[0]] = true
			if ep.readings[0].t1 == 8 {
				dur8++
			} else {
				dur10++
			}
		default:
			contested++
		}
	}
	total := len(deaths)
	if total == 0 {
		total = 1
	}
	t.Logf("JOINTURE : episodes %d · APPARIES %d · contestes %d · orphelins %d · incoherents %d",
		len(episodes), matched, contested, orphan, incoherent)
	t.Logf("COUVERTURE : %d morts couvertes sur %d (%.1f %%) · durees : 8 s x%d · 10 s x%d",
		len(coveredDeaths), len(deaths), 100*float64(len(coveredDeaths))/float64(total), dur8, dur10)
	t.Log("VERDICT attendu pour publier : couverture >= 70 %, contestes ~0, " +
		"incoherents faibles (sinon le filtre de valeur ne suffit pas)")
}
