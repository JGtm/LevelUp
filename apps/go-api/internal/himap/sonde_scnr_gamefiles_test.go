package himap

// SONDE (2026-08-10, non commitee) — ou vivent les volumes de mort des cartes
// INTEGREES ? scnr n'existe pas dans Infinite (verifie sur toute l'installation).
// Balayage generique des groupes suspects du module de carte : pour chaque bloc,
// part des enregistrements dont un triplet de floats tombe dans la boite monde
// du bsp, a chaque offset de la stride.
import (
	"testing"

	"levelup/go-api/internal/himodule"
)

func TestSondeScnr(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	for _, carte := range []string{"ridgeline", "catalyst"} {
		t.Run(carte, func(t *testing.T) {
			chemin := moduleAny(t, carte)
			m, err := himodule.Open(chemin)
			if err != nil {
				t.Fatal(err)
			}
			bsps, err := ReadModuleInstances(moduleDuJeu(t, "pc", carte))
			if err != nil {
				t.Fatal(err)
			}
			bsp := choisitBSP(bsps, nil)
			bb := bsp.Bounds
			for _, grp := range []string{"dwsg", "hlds", "cage", "stse", "trac", "sred", "snad"} {
				for _, f := range m.Files(grp) {
					t.Logf("%s : %s#%d — %d o", carte, grp, f.Index, f.UncompSize)
					tag, err := m.Extract(f)
					if err != nil {
						t.Logf("  extraction : %v", err)
						continue
					}
					ti, err := meilleurTagInfo(tag)
					if err != nil {
						t.Logf("  en-tete : %v", err)
						continue
					}
					t.Logf("  deps=%d dataBlocks=%d structs=%d", ti.deps, ti.dataBlocks, ti.structs)
					for _, l := range liensBlocs(ti) {
						abs, size := ti.blockAbs(l.target)
						c := compteChamp(ti, l)
						if c < 2 || size%c != 0 {
							continue
						}
						stride := size / c
						if stride < 12 || stride > 512 {
							continue
						}
						meilleurOff, meilleurFrac := -1, 0.0
						for off := 0; off+12 <= stride; off += 4 {
							dans := 0
							for i := 0; i < c; i++ {
								b := abs + i*stride + off
								x, y, z := f32(ti.tag, b), f32(ti.tag, b+4), f32(ti.tag, b+8)
								if x >= bb.Min[0] && x <= bb.Max[0] &&
									y >= bb.Min[1] && y <= bb.Max[1] &&
									z >= bb.Min[2] && z <= bb.Max[2] &&
									(x != 0 || y != 0 || z != 0) {
									dans++
								}
							}
							if f := float64(dans) / float64(c); f > meilleurFrac {
								meilleurFrac, meilleurOff = f, off
							}
						}
						if meilleurFrac >= 0.8 {
							t.Logf("  CANDIDAT %s bloc %d : %d records x %d o, triplet en boite a l'offset 0x%x (%.0f %%)",
								grp, l.target, c, stride, meilleurOff, 100*meilleurFrac)
						}
					}
				}
			}
		})
	}
}
