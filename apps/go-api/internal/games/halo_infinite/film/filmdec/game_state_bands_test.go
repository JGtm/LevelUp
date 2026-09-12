package filmdec

// game_state_bands_test.go — LE JOURNAL DES BANDES ET DE LEURS TEMOINS : le recensement des
// images-cles, la purete par classe, les deux bandes de controle et les rapports reel/fantome.
// `game_state_measure_test.go` porte l en-tete, le contrat et les seuils ; ce fichier en est
// scinde pour tenir le seuil de 500 lignes.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func gameShort8(dir string) string {
	return filepath.Base(strings.TrimRight(filepath.Clean(dir), string(filepath.Separator)))
}

func gameLogBands(t *testing.T, sc GameEntityScan) {
	t.Helper()
	t.Logf("FILM %s · chunks %d · paquets delta %d · horloge film %d us · delta [%d , %d] us "+
		"(%.1f s)", gameShort8(os.Getenv(gameFilmEnv)), sc.Chunks, sc.Packets, sc.FilmClockUS,
		sc.FirstPacketUS, sc.LastPacketUS, float64(sc.LastPacketUS-sc.FirstPacketUS)/1e6)
	t.Logf("BANDES · ti=0 %s · ti=5 %s · ti=4 %s · controle voisin %d · controle vide %d "+
		"· slots ambigus ecartes %d · comblement NON applique (aurait ajoute %d slots)",
		gameSlotList(sc.Bands[GameEngineTypeIndex]), gameSlotList(sc.Bands[PlayerEngineTypeIndex]),
		gameSlotList(sc.Bands[ProbeWitnessTypeIndex]), len(sc.Bands[GameEntityClassNeighbour]),
		len(sc.Bands[GameEntityClassVoid]), sc.Ambiguous, sc.Filled)
	t.Logf("SONDE ti=4 high-frequency : %d annonces recues par le hook pendant ce balayage",
		sc.ProbeWitness)
	gameLogKeyframeCensus(t, sc)
}

// gameLogKeyframeCensus journalise QUELS archetypes les images-cles portent, et avec combien
// de slots. Sans ce recensement, une bande vide ne se distingue pas d un archetype absent.
func gameLogKeyframeCensus(t *testing.T, sc GameEntityScan) {
	t.Helper()
	tis := make([]int, 0, len(sc.KeyframeTICensus))
	for ti := range sc.KeyframeTICensus {
		tis = append(tis, ti)
	}
	sort.Ints(tis)
	var parts []string
	for _, ti := range tis {
		parts = append(parts, fmt.Sprintf("ti=%d:%d", ti, sc.KeyframeTICensus[ti]))
	}
	t.Logf("RECENSEMENT DES IMAGES-CLES (slots distincts par archetype) : %s",
		strings.Join(parts, " "))
	slots := make([]uint32, 0, len(sc.KeyframeSlotTI))
	for s := range sc.KeyframeSlotTI {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var low []string
	for i, s := range slots {
		if i >= 24 {
			break
		}
		low = append(low, fmt.Sprintf("%d->%v", s, sc.KeyframeSlotTI[s]))
	}
	t.Logf("PREMIERS SLOTS DES IMAGES-CLES : %s", strings.Join(low, " "))
}

// gameSlotList rend la liste triee des slots d'une bande (bornee, pour rester lisible).
func gameSlotList(band map[uint32]bool) string {
	s := make([]uint32, 0, len(band))
	for k := range band {
		s = append(s, k)
	}
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	if len(s) == 0 {
		return "(vide)"
	}
	if len(s) > 12 {
		return fmt.Sprintf("%d slots [%d .. %d]", len(s), s[0], s[len(s)-1])
	}
	return fmt.Sprintf("%d slots %v", len(s), s)
}

func gameLogClass(t *testing.T, sc GameEntityScan, class int, label string) {
	t.Helper()
	st := sc.Stats[class]
	if st == nil {
		t.Logf("%s : AUCUNE statistique (archetype absent du registre)", label)
		return
	}
	pur, ok := st.Purity()
	pure := "n/a"
	if ok {
		pure = fmt.Sprintf("%.2f %%", 100*pur)
	}
	t.Logf("%s · bande %d slots · peuples %d · records %d (%.0f/slot) · DANS LA GRAMMAIRE %d "+
		"· avec un composant vise %d · marche ABOUTIE %d · CASSEE %d · purete %s "+
		"(hors grammaire %d / %d composants) · plancher de bruit %.1f",
		label, st.BandSize, st.Slots, st.Records, st.RecordsPerSlot(), st.InGrammar,
		st.WithWanted, st.Walked, st.Broken, pure, st.OutOfGrammar, st.GrammarLen,
		st.NoiseFloor())
	gameLogCensus(t, sc, class, st)
}

// gameLogCensus journalise le recensement d'annonces au masque, avec le facteur au-dessus du
// plancher de bruit de la bande. Un composant qui ne se detache pas de ce plancher n'est pas
// mesure : il est indistinguable du bruit, et c'est ce qu'il faut ecrire.
func gameLogCensus(t *testing.T, sc GameEntityScan, class int, st *GameEntityStats) {
	t.Helper()
	floor := st.NoiseFloor()
	arch := gameArchOf(sc, class)
	type row struct {
		i, n int
	}
	var rows []row
	for i := 0; i < worldObjectMaxComponent; i++ {
		if st.MaskCensus[i] > 0 {
			rows = append(rows, row{i, st.MaskCensus[i]})
		}
	}
	sort.Slice(rows, func(a, b int) bool { return rows[a].n > rows[b].n })
	if len(rows) > 20 {
		rows = rows[:20]
	}
	for _, r := range rows {
		x := 0.0
		if floor > 0 {
			x = float64(r.n) / floor
		}
		t.Logf("    i%-2d %-52s annonces %7d  x%.1f du plancher", r.i, arch[r.i], r.n, x)
	}
}

// gameArchOf rend les noms de composants de l'archetype d'une classe (vide pour un controle).
func gameArchOf(sc GameEntityScan, class int) map[int]string {
	out := map[int]string{}
	if class < 0 {
		return out
	}
	dir := os.Getenv(gameFilmEnv)
	reg, err := gameEntityRegistry(dir)
	if err != nil {
		return out
	}
	arch, ok := reg.Archetype(class)
	if !ok {
		return out
	}
	for i, n := range arch.Components {
		out[i] = n
	}
	return out
}

func gameLogControls(t *testing.T, sc GameEntityScan) {
	t.Helper()
	for _, c := range []struct {
		class int
		label string
	}{
		{GameEntityClassNeighbour, "CONTROLE voisinage"},
		{GameEntityClassVoid, "CONTROLE vide (haut de l'espace)"},
	} {
		st := sc.Stats[c.class]
		if st == nil {
			continue
		}
		t.Logf("%s · bande %d slots · peuples %d · records %d (%.0f/slot) · DANS LA GRAMMAIRE "+
			"ti=5 %d (%.0f/slot) · avec un composant vise %d (%.0f/slot) · plancher %.1f",
			c.label, st.BandSize, st.Slots, st.Records, st.RecordsPerSlot(), st.InGrammar,
			gamePerSlot(st.InGrammar, st.Slots), st.WithWanted,
			gamePerSlot(st.WithWanted, st.Slots), st.NoiseFloor())
	}
	gameLogRatios(t, sc)
}

// gameLogRatios rend le RAPPORT REEL / FANTOME, la grandeur qui dit si une bande a mesure
// quelque chose. Un rapport de 1 signifie que l'archetype n'est pas distinguable du bruit.
func gameLogRatios(t *testing.T, sc GameEntityScan) {
	t.Helper()
	nb, vd := sc.Stats[GameEntityClassNeighbour], sc.Stats[GameEntityClassVoid]
	for _, ti := range []int{GameEngineTypeIndex, PlayerEngineTypeIndex, ProbeWitnessTypeIndex} {
		st := sc.Stats[ti]
		if st == nil || st.RecordsPerSlot() == 0 {
			continue
		}
		t.Logf("RAPPORT REEL/FANTOME ti=%d · EN-TETES BRUTS %.0f/slot contre %.0f (voisinage) "+
			"et %.0f (vide) -> x%.2f et x%.2f", ti, st.RecordsPerSlot(), gameRPS(nb),
			gameRPS(vd), gameRatio(st.RecordsPerSlot(), gameRPS(nb)),
			gameRatio(st.RecordsPerSlot(), gameRPS(vd)))
		t.Logf("RAPPORT REEL/FANTOME ti=%d · COMPOSANT VISE %.1f/slot contre %.1f (voisinage) "+
			"et %.1f (vide) -> x%.2f et x%.2f", ti, gameWantedPerSlot(st),
			gameWantedPerSlot(nb), gameWantedPerSlot(vd),
			gameRatio(gameWantedPerSlot(st), gameWantedPerSlot(nb)),
			gameRatio(gameWantedPerSlot(st), gameWantedPerSlot(vd)))
	}
}

func gameRPS(st *GameEntityStats) float64 {
	if st == nil {
		return 0
	}
	return st.RecordsPerSlot()
}

// gamePerSlot rend un compte ramene au slot peuple.
func gamePerSlot(n, slots int) float64 {
	if slots == 0 {
		return 0
	}
	return float64(n) / float64(slots)
}

// gameWantedPerSlot rend le debit de records PORTANT UN COMPOSANT VISE — la seule grandeur
// dont le rapport reel/fantome dit quelque chose sur la mesure reellement faite.
func gameWantedPerSlot(st *GameEntityStats) float64 {
	if st == nil {
		return 0
	}
	return gamePerSlot(st.WithWanted, st.Slots)
}

func gameRatio(a, b float64) float64 {
	if b <= 0 {
		return 0
	}
	return a / b
}
