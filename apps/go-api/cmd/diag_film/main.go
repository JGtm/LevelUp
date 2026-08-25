//go:build cgo

// diag_film — télécharge et analyse les chunks film d'un match pour diagnostiquer
// pourquoi le scanner weapon ne trouve aucun fire event.
//
// Usage (depuis apps/go-api/) :
//
//	go run -tags cgo ./cmd/diag_film/ -match <match_id>
//	go run -tags cgo ./cmd/diag_film/ -match b8c1b220-5ef4-4dee-9e92-77d3ff55d6d3
//
// Charge les tokens depuis .env.local + data/auth/watcher_tokens.json.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/platform/auth"
	gosync "levelup/go-api/internal/sync"
)

const (
	defaultMatchID = "b8c1b220-5ef4-4dee-9e92-77d3ff55d6d3"
)

func main() {
	matchID := flag.String("match", defaultMatchID, "Match ID à analyser")
	envFile := flag.String("env-file", "../../.env.local", "Chemin .env.local (depuis apps/go-api/)")
	authFile := flag.String("auth-file", "../../data/auth/watcher_tokens.json", "watcher_tokens.json")
	gamertag := flag.String("gamertag", "Chocoboflor", "Gamertag pour charger les tokens")
	verbose := flag.Bool("verbose", true, "Dump fire events détaillés + agrégat par arme/joueur")
	maxEventsPerChunk := flag.Int("max-events", 5, "Nombre max de fire events à afficher par chunk en mode verbose")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "diag_film start, match=%s\n", *matchID)
	loadEnvLocal(*envFile)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	ctx := context.Background()

	fmt.Fprintln(os.Stderr, "loading tokens...")
	tokens, err := loadTokens(ctx, *authFile, *gamertag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tokens error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "tokens OK, creating client...")

	client := gosync.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, 3)

	fmt.Printf("=== Téléchargement film pour %s ===\n", *matchID)
	rawChunks, found, err := client.GetMatchFilm(ctx, *matchID)
	if err != nil {
		log.Fatalf("GetMatchFilm: %v", err)
	}
	if !found {
		fmt.Println("Film non trouvé (404/410 - film expiré)")
		return
	}
	fmt.Printf("Chunks reçus : %d\n\n", len(rawChunks))

	// Agrégats cross-chunk
	type weaponPlayerKey struct {
		weaponName string
		playerIdx  int
	}
	weaponPlayerCounts := make(map[weaponPlayerKey]int)
	weaponTotals := make(map[string]int)
	playerEventTotals := make(map[int]int)
	totalEvents := 0
	totalSnapshots := 0

	for idx, fc := range rawChunks {
		data := fc.Data
		fmt.Printf("--- Chunk %d : start=%dms dur=%dms size=%d bytes ---\n",
			idx, fc.StartMS, fc.DurationMS, len(data))

		// Hex dump des premiers 64 bytes
		if len(data) >= 64 {
			fmt.Printf("  First 64 bytes: %s\n", hex.EncodeToString(data[:64]))
		} else {
			fmt.Printf("  Data: %s\n", hex.EncodeToString(data))
		}

		// Frame markers
		frames := analysis.FindFramePositions(data)
		fmt.Printf("  Frame markers [A0 7B 42]  : %d\n", len(frames))

		// Formula A patterns [20 00 02]
		faPattern := []byte{0x20, 0x00, 0x02}
		faCount := countPattern(data, faPattern)
		fmt.Printf("  FormulaA patterns [20 00 02] : %d occurrences\n", faCount)

		// Formula A results (parsed)
		faResults := analysis.ScanFormulaA(data)
		fmt.Printf("  ScanFormulaA results      : %d\n", len(faResults))

		// Formula A NS results
		faNS := analysis.ScanFormulaANS(data)
		fmt.Printf("  ScanFormulaANS results    : %d\n", len(faNS))

		// Fire events via ScanFireEventsAll
		estimateTS := analysis.TimestampEstimator(data, fc.StartMS, fc.DurationMS)
		fireEvents := analysis.ScanFireEventsB5(data, estimateTS)
		fmt.Printf("  ScanFireEventsB5 events   : %d\n", len(fireEvents))

		// Check if universal marker bits appear at all (raw search)
		markerBits := 0b10100100110 // universalMarkerBits = 11 bits
		markerCount := countMarkerBits(data, markerBits, 11)
		fmt.Printf("  Universal marker bits count: %d (should be >0 if fire events exist)\n", markerCount)

		// Check if CommonWeaponSuffix appears
		commonSuffix := []byte{0x42, 0xc9, 0x67, 0x9f}
		suffixCount := countPattern(data, commonSuffix)
		fmt.Printf("  CommonWeaponSuffix [42 c9 67 9f] count: %d\n", suffixCount)

		// Top weapon bytes found near the suffix
		if suffixCount > 0 {
			fmt.Printf("  Sample weapon bytes found near suffix:\n")
			pos := 0
			n := 0
			for n < 3 {
				idx2 := bytes.Index(data[pos:], commonSuffix)
				if idx2 < 0 {
					break
				}
				abs := pos + idx2
				if abs >= 4 {
					wb := make([]byte, 8)
					copy(wb, data[abs-4:abs+4])
					wid := binary.BigEndian.Uint64(wb)
					fmt.Printf("    @%d: wid=%d  hex=%s\n", abs-4, wid, hex.EncodeToString(wb))
				}
				pos = abs + 1
				n++
			}
		}

		totalSnapshots += len(faResults)
		totalEvents += len(fireEvents)

		if *verbose && len(faResults) > 0 {
			n := *maxEventsPerChunk
			if n > len(faResults) {
				n = len(faResults)
			}
			fmt.Printf("  Sample FormulaA snapshots (premières %d/%d):\n", n, len(faResults))
			for i := 0; i < n; i++ {
				r := faResults[i]
				wid := binary.BigEndian.Uint64(r.WeaponBytes[:])
				name := analysis.WeaponIDToName[wid]
				if name == "" {
					name = "INCONNU"
				}
				fmt.Printf("    @%d  pi=%d  weapon=%-22s  hex=%s\n",
					r.Offset, r.PlayerIndex, name, hex.EncodeToString(r.WeaponBytes[:]))
			}
		}

		if *verbose && len(fireEvents) > 0 {
			n := *maxEventsPerChunk
			if n > len(fireEvents) {
				n = len(fireEvents)
			}
			fmt.Printf("  Sample fire events (premiers %d/%d):\n", n, len(fireEvents))
			for i := 0; i < n; i++ {
				ev := fireEvents[i]
				fmt.Printf("    t=%.0fms  pi=%d  slot=%d  weapon=%-22s  fire_seq=%d  fire_counter=%d  hex=%s\n",
					ev.TimestampMS, ev.PlayerIndex, ev.Slot, ev.WeaponName,
					ev.FireSeq, ev.FireCounter, hex.EncodeToString(ev.WeaponBytes[:]))
			}
		}

		// Agrégats
		for _, ev := range fireEvents {
			weaponPlayerCounts[weaponPlayerKey{ev.WeaponName, ev.PlayerIndex}]++
			weaponTotals[ev.WeaponName]++
			playerEventTotals[ev.PlayerIndex]++
		}

		fmt.Println()
	}

	if *verbose {
		fmt.Println("════════════════════════════════════════════════════════════════")
		fmt.Println("  AGRÉGAT CROSS-CHUNK")
		fmt.Println("════════════════════════════════════════════════════════════════")
		fmt.Printf("Total chunks               : %d\n", len(rawChunks))
		fmt.Printf("Total FormulaA snapshots   : %d\n", totalSnapshots)
		fmt.Printf("Total fire events          : %d\n", totalEvents)

		fmt.Println("\nFire events par arme (toutes chunks confondues) :")
		// Tri par count desc
		type wc struct {
			name  string
			count int
		}
		var wlist []wc
		for n, c := range weaponTotals {
			wlist = append(wlist, wc{n, c})
		}
		for i := 0; i < len(wlist); i++ {
			for j := i + 1; j < len(wlist); j++ {
				if wlist[j].count > wlist[i].count {
					wlist[i], wlist[j] = wlist[j], wlist[i]
				}
			}
		}
		for _, w := range wlist {
			fmt.Printf("  %-22s : %d events\n", w.name, w.count)
		}

		fmt.Println("\nFire events par player_index (devrait être 0..N selon nb joueurs du match) :")
		// Trier par player_index croissant
		var pis []int
		for pi := range playerEventTotals {
			pis = append(pis, pi)
		}
		for i := 0; i < len(pis); i++ {
			for j := i + 1; j < len(pis); j++ {
				if pis[j] < pis[i] {
					pis[i], pis[j] = pis[j], pis[i]
				}
			}
		}
		for _, pi := range pis {
			fmt.Printf("  pi=%d : %d events\n", pi, playerEventTotals[pi])
		}

		fmt.Println("\nTop combinaisons (joueur, arme) — ce qui finirait dans weapon_kills :")
		type wpc struct {
			key   weaponPlayerKey
			count int
		}
		var combos []wpc
		for k, c := range weaponPlayerCounts {
			combos = append(combos, wpc{k, c})
		}
		for i := 0; i < len(combos); i++ {
			for j := i + 1; j < len(combos); j++ {
				if combos[j].count > combos[i].count {
					combos[i], combos[j] = combos[j], combos[i]
				}
			}
		}
		topN := 20
		if len(combos) < topN {
			topN = len(combos)
		}
		for i := 0; i < topN; i++ {
			c := combos[i]
			fmt.Printf("  pi=%-2d  weapon=%-22s : %d events\n",
				c.key.playerIdx, c.key.weaponName, c.count)
		}
	}
}

func countPattern(data, pattern []byte) int {
	n := 0
	pos := 0
	for {
		idx := bytes.Index(data[pos:], pattern)
		if idx < 0 {
			break
		}
		n++
		pos += idx + 1
	}
	return n
}

// countMarkerBits counts bit-level occurrences of a marker in data.
func countMarkerBits(data []byte, marker, markerLen int) int {
	n := 0
	totalBits := len(data) * 8
	markerMask := (1 << markerLen) - 1
	for bitPos := 0; bitPos+markerLen <= totalBits; bitPos++ {
		byteIdx := bitPos / 8
		bitIdx := bitPos % 8
		var window uint32
		if byteIdx+3 < len(data) {
			window = uint32(data[byteIdx])<<24 | uint32(data[byteIdx+1])<<16 |
				uint32(data[byteIdx+2])<<8 | uint32(data[byteIdx+3])
		} else if byteIdx+2 < len(data) {
			window = uint32(data[byteIdx])<<24 | uint32(data[byteIdx+1])<<16 |
				uint32(data[byteIdx+2])<<8
		} else {
			window = uint32(data[byteIdx]) << 24
		}
		bits := int((window >> (32 - markerLen - bitIdx)) & uint32(markerMask))
		if bits == marker {
			n++
		}
	}
	return n
}

func loadEnvLocal(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func loadTokens(ctx context.Context, authFile, gamertag string) (*struct {
	SpartanToken   string
	ClearanceToken string
}, error) {
	const margin = 5 * 60 * 1e9 // 5 minutes

	store := auth.NewTokenStore(authFile)
	stored, _ := store.Load()
	if stored != nil && stored.IsXSTSValid(0) {
		result, err := auth.ExchangeXSTSForHaloTokens(ctx, stored.XSTSToken)
		if err == nil {
			return &struct {
				SpartanToken   string
				ClearanceToken string
			}{result.SpartanToken, result.ClearanceToken}, nil
		}
	}

	_ = margin

	// ADR 0023 Phase 5 : refresh token depuis le MultiUserTokenStore, seule source.
	// data/auth/watcher_tokens.json → data/auth/watcher_tokens (répertoire du store).
	tokenStore := auth.NewMultiUserTokenStore(strings.TrimSuffix(authFile, ".json"))
	if user, lerr := tokenStore.LoadByGamertag(gamertag); lerr == nil && user != nil {
		res, rerr := auth.RefreshHaloTokensViaStoreFirst(ctx, tokenStore, auth.NewSISUProvider(), user.XUID, gamertag)
		if rerr == nil {
			if tokens := auth.HaloTokensFromExchange(res); tokens != nil {
				return &struct {
					SpartanToken   string
					ClearanceToken string
				}{tokens.SpartanToken, tokens.ClearanceToken}, nil
			}
		}
	}

	return nil, fmt.Errorf("impossible de charger les tokens pour %s (vérifier data/auth/watcher_tokens et %s)", gamertag, authFile)
}
