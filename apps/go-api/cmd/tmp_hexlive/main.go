package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

func main() {
	d, _ := os.ReadFile(`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/killcapture.bin`)
	fmt.Printf("killcapture taille=%d ; %d records de 16\n", len(d), len(d)/16)
	for i := 0; i < 10 && i*16+16 <= len(d); i++ {
		o := i * 16
		fmt.Printf("rec%d:", i)
		for j := 0; j < 16; j += 4 {
			fmt.Printf(" [%02d]=%08x", j, binary.LittleEndian.Uint32(d[o+j:]))
		}
		fmt.Println()
	}
}
