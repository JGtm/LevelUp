// tmp_kfhex — THROWAWAY (outillage RE). Produit deux artefacts TEXTE pour les
// agents d'analyse qui ne lancent pas de Go :
//  1. kf_type2_hex.txt   : hexdump complet du payload KEYFRAME type-2 (chunk_02
//     décompressé), "OFFSET_HEX: 16 octets hex" par ligne.
//  2. expected_ids.txt   : pour chaque slot S du world_dump CE, id=0x40000000|S,
//     les 4 octets little-endian (tels qu'en mémoire) + typeIndex.
//
// L'extraction du payload type-2 réplique EXACTEMENT tmp_kfframe (inflate zlib +
// framing [Type u16@0][?u16@2][Size u32@4][Ts u64@8][payload@16], 1er paquet type=2)
// afin que les offsets du hexdump correspondent à ce que tmp_kfframe rapporte.
//
// Usage : tmp_kfhex <chunkdir> <world_dump.txt> <crack_outdir>
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

// extractType2 walks [Type u16@0][?u16@2][Size u32@4][Ts u64@8][payload@16] and
// returns the payload + timestamp of the first type-2 packet.
func extractType2(d []byte) (payload []byte, ts uint64, ok bool) {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		t := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 2 {
			return d[off+16 : off+16+sz], t, true
		}
		off += 16 + sz
	}
	return nil, 0, false
}

func writeHexDump(pay []byte, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	defer bw.Flush()
	fmt.Fprintf(bw, "# hexdump payload KEYFRAME type-2 de 000d5950 (chunk_02 inflate)\n")
	fmt.Fprintf(bw, "# taille payload = %d octets ; format : OFFSET_HEX: 16 octets hex\n", len(pay))
	for i := 0; i < len(pay); i += 16 {
		end := i + 16
		if end > len(pay) {
			end = len(pay)
		}
		fmt.Fprintf(bw, "%08x:", i)
		for j := i; j < end; j++ {
			fmt.Fprintf(bw, " %02x", pay[j])
		}
		fmt.Fprintln(bw)
	}
	return nil
}

type ent struct {
	slot int
	ti   int
}

// parseWorldDump reads "slot:typeIndex" tokens (space-separated), skipping '#' lines.
func parseWorldDump(path string) ([]ent, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []ent
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			c := strings.IndexByte(tok, ':')
			if c <= 0 {
				continue
			}
			slot, e1 := strconv.Atoi(tok[:c])
			ti, e2 := strconv.Atoi(tok[c+1:])
			if e1 != nil || e2 != nil {
				continue
			}
			out = append(out, ent{slot, ti})
		}
	}
	return out, nil
}

func writeExpectedIDs(ents []ent, outPath string) (int, error) {
	sort.Slice(ents, func(i, j int) bool { return ents[i].slot < ents[j].slot })
	f, err := os.Create(outPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	defer bw.Flush()
	fmt.Fprintf(bw, "# IDs attendus dans le payload type-2 de 000d5950 (source : world_dump CE).\n")
	fmt.Fprintf(bw, "# id = 0x40000000 | slot. idLE = les 4 octets little-endian tels qu'en memoire.\n")
	fmt.Fprintf(bw, "# colonnes : slot(dec)  id(hex)  idLE(bytes hex)  typeIndex\n")
	for _, e := range ents {
		id := uint32(0x40000000) | uint32(e.slot)
		var le [4]byte
		binary.LittleEndian.PutUint32(le[:], id)
		fmt.Fprintf(bw, "%-6d 0x%08x %02x %02x %02x %02x %d\n",
			e.slot, id, le[0], le[1], le[2], le[3], e.ti)
	}
	return len(ents), nil
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: tmp_kfhex <chunkdir> <world_dump.txt> <crack_outdir>")
		os.Exit(2)
	}
	chunkDir := os.Args[1]
	worldDump := os.Args[2]
	outDir := os.Args[3]

	d2 := inflate(filepath.Join(chunkDir, "chunk_02.bin"))
	pay, ts, ok := extractType2(d2)
	if !ok {
		fmt.Println("ERREUR : pas de paquet type-2 dans chunk_02")
		os.Exit(1)
	}
	fmt.Printf("chunk_02 inflated = %d octets ; payload type-2 = %d octets (ts=%d)\n", len(d2), len(pay), ts)

	hexPath := filepath.Join(outDir, "kf_type2_hex.txt")
	if err := writeHexDump(pay, hexPath); err != nil {
		panic(err)
	}
	fmt.Printf("ecrit : %s (%d lignes)\n", hexPath, (len(pay)+15)/16)

	ents, err := parseWorldDump(worldDump)
	if err != nil {
		panic(err)
	}
	idsPath := filepath.Join(outDir, "expected_ids.txt")
	n, err := writeExpectedIDs(ents, idsPath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("ecrit : %s (%d entites)\n", idsPath, n)

	// premieres observations : histogramme d'octets + tete du payload
	var freq [256]int
	for _, b := range pay {
		freq[b]++
	}
	fmt.Printf("octets : 0x00=%.1f%% 0xff=%.1f%% ; 16 premiers = ",
		100*float64(freq[0])/float64(len(pay)), 100*float64(freq[0xff])/float64(len(pay)))
	for i := 0; i < 16 && i < len(pay); i++ {
		fmt.Printf("%02x ", pay[i])
	}
	fmt.Println()
}
