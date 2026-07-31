package main

import (
	"encoding/json"
	"fmt"
	"os"

	"levelup/go-api/internal/analysis/weaponv3"
)

const out = "C:/Users/GUILLA~1/AppData/Local/Temp/claude/C--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration--claude-worktrees-filmdec-continuation/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad/weapon_families.json"

func main() {
	m := map[string]string{}
	for f, n := range weaponv3.KnownWeaponHigh32 {
		m[fmt.Sprintf("0x%08X", f)] = n
	}
	b, _ := json.Marshal(m)
	if err := os.WriteFile(out, b, 0o644); err != nil {
		panic(err)
	}
	fmt.Println(len(m), "familles ecrites")
}
