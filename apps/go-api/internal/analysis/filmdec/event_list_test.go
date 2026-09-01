package filmdec

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// Instrument de recherche/garde pour le décodage de la LISTE D'ÉVÉNEMENTS (event_list.go).
// TOUS ces tests sont sous garde d'environnement : sans la variable, ils SKIP. Aucun effet de
// bord, aucune base ouverte — lecture seule des chunks du cache film.
//
//	EVT_CHUNK_DIR = chemin d'un dossier de chunks (data/cache/film_chunks/<short8>)
//	EVT_CACHE     = racine du cache (data/cache/film_chunks) pour un balayage corpus
//	EVT_FILMS     = short8 séparés par des virgules, restreint EVT_CACHE

func evtChunkDir(t *testing.T) string {
	t.Helper()
	d := os.Getenv("EVT_CHUNK_DIR")
	if d == "" {
		t.Skip("EVT_CHUNK_DIR non défini")
	}
	return d
}

func evtCacheRoot(t *testing.T) string {
	t.Helper()
	d := os.Getenv("EVT_CACHE")
	if d == "" {
		t.Skip("EVT_CACHE non défini")
	}
	return d
}

func splitComma(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func evtListFilmDirs(root string) []string {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, filepath.Join(root, e.Name()))
		}
	}
	return out
}

// TestEvtHeadHistogram : histogramme des types d'événement de TÊTE sur un film + comptes
// board/exit + recoupement avec fire_events (GARDE-FOU de cadrage : head type36 doit égaler
// exactement le compte fire_events, preuve que le cadrage config+continuation+R(7) est bit-exact).
func TestEvtHeadHistogram(t *testing.T) {
	dir := evtChunkDir(t)
	hist := map[int]int{}
	empty, total := 0, 0
	byteHist := map[byte]int{}
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeDelta || p.Size < 1 {
				continue
			}
			total++
			pay := p.Payload(chunk)
			byteHist[pay[0]]++
			typ, present := PacketHeadEventType(pay)
			if !present {
				empty++
				continue
			}
			hist[typ]++
		}
	}
	t.Logf("paquets delta=%d, liste vide=%d, liste non vide=%d", total, empty, total-empty)
	t.Logf("board(8)=%d  exit(22)=%d  enter(53)=%d",
		hist[EventBipedBoardVehicle], hist[EventUnitExitVehicle], hist[EventUnitEnterVehicle])
	if fire, err := ScanFilmFireEvents(dir); err == nil {
		t.Logf("GARDE cadrage: fire_events=%d == head type36=%d (octet 0xD2=%d)",
			len(fire), hist[36], byteHist[0xD2])
		if len(fire) != hist[36] {
			t.Errorf("cadrage ROMPU: fire_events=%d != head type36=%d", len(fire), hist[36])
		}
	}
	type kv struct{ k, v int }
	var top []kv
	for k, v := range hist {
		top = append(top, kv{k, v})
	}
	sort.Slice(top, func(i, j int) bool { return top[i].v > top[j].v })
	for i := 0; i < len(top) && i < 20; i++ {
		t.Logf("  type %3d : %d", top[i].k, top[i].v)
	}
}

// TestEvtCorpusCounts : compte board/exit (événements de TÊTE) sur tout le cache. Valide la
// forme des comptes corpus (référence : 374 board / 5 600 exit sur 1 367 films).
func TestEvtCorpusCounts(t *testing.T) {
	root := evtCacheRoot(t)
	dirs := evtListFilmDirs(root)
	board, exit, enter := 0, 0, 0
	filmsBoard, filmsExit, filmsScanned := 0, 0, 0
	for _, dir := range dirs {
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		filmsScanned++
		fb, fe := 0, 0
		for c := 1; c <= n; c++ {
			chunk, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, p := range WalkPackets(chunk) {
				if p.Type != PacketTypeDelta || p.Size < 1 {
					continue
				}
				typ, present := PacketHeadEventType(p.Payload(chunk))
				if !present {
					continue
				}
				switch typ {
				case EventBipedBoardVehicle:
					fb++
				case EventUnitExitVehicle:
					fe++
				case EventUnitEnterVehicle:
					enter++
				}
			}
		}
		board += fb
		exit += fe
		if fb > 0 {
			filmsBoard++
		}
		if fe > 0 {
			filmsExit++
		}
	}
	t.Logf("films balayés=%d (sur %d dossiers)", filmsScanned, len(dirs))
	t.Logf("board(8)=%d sur %d films | exit(22)=%d sur %d films | enter(53)=%d",
		board, filmsBoard, exit, filmsExit, enter)
	t.Logf("référence corpus 1367 films : board=374 (154 films), exit=5600 (279 films)")
}

// TestEvtVehicleValidate : sur un ensemble de films, décode board/exit via la production
// (ScanFilmVehicleEvents) et rapporte occupant-en-bande, sonde, et histogrammes de siège.
// EVT_FILMS restreint la liste ; sinon tous les dossiers de EVT_CACHE.
func TestEvtVehicleValidate(t *testing.T) {
	root := evtCacheRoot(t)
	var dirs []string
	if fl := os.Getenv("EVT_FILMS"); fl != "" {
		for _, s := range splitComma(fl) {
			dirs = append(dirs, filepath.Join(root, s))
		}
	} else {
		dirs = evtListFilmDirs(root)
	}

	occTotal, occInBand := 0, 0
	sonde := map[int][2]int{}
	seatHist := map[int]map[uint32]int{
		EventBipedBoardVehicle: {},
		EventUnitExitVehicle:   {},
	}
	nBoard, nExit := 0, 0
	for _, dir := range dirs {
		evs, err := ScanFilmVehicleEvents(dir)
		if err != nil {
			continue
		}
		for _, ev := range evs {
			if ev.Kind == EventBipedBoardVehicle {
				nBoard++
			} else {
				nExit++
			}
			if ev.OccupantPresent {
				occTotal++
				s := sonde[ev.Kind]
				s[ev.OccupantSonde]++
				sonde[ev.Kind] = s
				if ev.OccupantInBand {
					occInBand++
				}
			}
			if ev.SeatValid {
				seatHist[ev.Kind][ev.Seat]++
			}
		}
	}
	t.Logf("board=%d exit=%d", nBoard, nExit)
	t.Logf("occupant présent=%d dont en-bande=%d (%.1f%%)",
		occTotal, occInBand, 100*float64(occInBand)/float64(maxi(occTotal, 1)))
	t.Logf("sonde board: s0=%d s1=%d | exit: s0=%d s1=%d",
		sonde[EventBipedBoardVehicle][0], sonde[EventBipedBoardVehicle][1],
		sonde[EventUnitExitVehicle][0], sonde[EventUnitExitVehicle][1])
	logSeat(t, "board", seatHist[EventBipedBoardVehicle])
	logSeat(t, "exit", seatHist[EventUnitExitVehicle])
}

// TestEvtVehicleDump0d76 : dump détaillé des embarquements/sorties d'un film (EVT_CHUNK_DIR),
// avec occupant + siège + instant, pour le livrable et le contrôle de cohérence board->exit.
func TestEvtVehicleDump0d76(t *testing.T) {
	dir := evtChunkDir(t)
	evs, err := ScanFilmVehicleEvents(dir)
	if err != nil {
		t.Fatalf("ScanFilmVehicleEvents: %v", err)
	}
	sort.Slice(evs, func(i, j int) bool { return evs[i].TimestampUS < evs[j].TimestampUS })
	for _, ev := range evs {
		kind := "EXIT "
		if ev.Kind == EventBipedBoardVehicle {
			kind = "BOARD"
		}
		seat := "-"
		if ev.SeatValid {
			seat = itoa(int(ev.Seat))
		}
		t.Logf("%s t=%8.2fs occ_slot=%d inBand=%v sonde=%d seat=%s (chunk=%d pkt=%d)",
			kind, float64(ev.TimestampUS)/1e6, ev.OccupantSlot, ev.OccupantInBand,
			ev.OccupantSonde, seat, ev.Chunk, ev.PacketIndex)
	}
	// Cohérence : pour chaque occupant de sortie, un embarquement précède-t-il ?
	firstBoard := map[uint32]uint64{}
	for _, ev := range evs {
		if ev.Kind == EventBipedBoardVehicle && ev.OccupantPresent {
			if _, ok := firstBoard[ev.OccupantSlot]; !ok {
				firstBoard[ev.OccupantSlot] = ev.TimestampUS
			}
		}
	}
	paired := 0
	for _, ev := range evs {
		if ev.Kind == EventUnitExitVehicle && ev.OccupantPresent {
			if tb, ok := firstBoard[ev.OccupantSlot]; ok && tb <= ev.TimestampUS {
				paired++
			}
		}
	}
	t.Logf("cohérence: %d sorties ont un embarquement antérieur du même occupant", paired)
}

func logSeat(t *testing.T, name string, h map[uint32]int) {
	type sv struct{ s, c int }
	var arr []sv
	tot := 0
	for s, c := range h {
		arr = append(arr, sv{int(s), c})
		tot += c
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].c > arr[j].c })
	line := ""
	for i := 0; i < len(arr) && i < 6; i++ {
		line += " seat=" + itoa(arr[i].s) + "×" + itoa(arr[i].c)
	}
	t.Logf("siège %s n=%d top:%s", name, tot, line)
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func itoa(x int) string {
	if x == 0 {
		return "0"
	}
	neg := x < 0
	if neg {
		x = -x
	}
	var b [12]byte
	i := len(b)
	for x > 0 {
		i--
		b[i] = byte('0' + x%10)
		x /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
