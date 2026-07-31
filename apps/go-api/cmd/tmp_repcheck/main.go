// tmp_repcheck — THROWAWAY. Valide consumeBipedDefaultState (FUN_140f44c38) contre la
// GROUND TRUTH capturée en live (CE breakpoint) : un record biped rep de 198 bits.
// Reconstruit le flux depuis l'état du BitReader du jeu (acc + used + bytes) et mesure
// combien de bits mon port consomme. Cible = 198.
package main

import (
	"encoding/hex"
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
)

// Capture live (CE) : rep biped, start=137 end=335 len=198.
const (
	acc      = uint64(0x86C61DAF68040000)
	used     = 9
	bytesHex = "33A0240844D660256000400198D2FEB7AE3982197E0001F37123810A2A0F2AC3A97381B67B6000000000000000000001"
	target   = 198
)

func main() {
	bs, _ := hex.DecodeString(bytesHex)
	// Bits non-lus de l'accumulateur = acc positions [63..used], MSB (63) d'abord.
	var bits []byte
	for pos := 63; pos >= used; pos-- {
		bits = append(bits, byte((acc>>uint(pos))&1))
	}
	for _, b := range bs {
		for j := 7; j >= 0; j-- {
			bits = append(bits, (b>>uint(j))&1)
		}
	}
	buf := make([]byte, (len(bits)+7)/8)
	for i, bit := range bits {
		if bit == 1 {
			buf[i/8] |= 1 << uint(7-(i%8))
		}
	}

	// Sanity : premiers bits (grammaire attendue : g0 gate ; si 1 -> R(8) version ; gRep ...)
	fmt.Printf("flux reconstruit : %d bits dispo\n", len(bits))
	fmt.Printf("40 premiers bits : ")
	for i := 0; i < 40 && i < len(bits); i++ {
		fmt.Printf("%d", bits[i])
	}
	fmt.Println()

	filmdec.SetRecordStateParam(2)
	br := filmdec.NewBitReader(buf)
	filmdec.ConsumeBipedDefaultStateProbe(br)
	got := br.BitPos()
	fmt.Printf("\nconsumeBipedDefaultState consomme = %d bits (cible %d)\n", got, target)
	if got == target {
		fmt.Println("=> BIT-EXACT !")
	} else {
		fmt.Printf("=> ECART = %d bits (%s de %d)\n", target-got, map[bool]string{true: "court", false: "long"}[got < target], abs(target-got))
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
