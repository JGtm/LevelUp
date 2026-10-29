//go:build research && campagne_overlay

package grammar

// r_veh_ti40_bits_research_test.go — compagnon de `r_veh_ti40_research_test.go` (recherche R-L4,
// 2026-10-02), decoupe pour le seuil de 500 lignes : le releve des bits de l etat par defaut des
// records `ti=40` a porte bVar14. Deplacement pur. Exige la surcouche `r_veh_overlay/`.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestRVehTi40Bits : les bits de l etat par defaut des records `ti=40` a porte bVar14, de
// l en-tete jusqu au n2 VRAI, pour CAMPAGNE_BITS_N records par chassis.
func TestRVehTi40Bits(t *testing.T) {
	racine, sortie, films, physique, _ := rvEnv(t)
	defer func() { bis2Intercepteur, b2vPorte = nil, false }()
	n, _ := strconv.Atoi(os.Getenv("CAMPAGNE_BITS_N"))
	if n <= 0 {
		n = 3
	}
	lignes := []string{"film\tbuild\tchunk\tslot\tchassis\tquat\tn2_vrai_d\tbits_de_b14_a_n2_vrai"}
	for _, id := range films {
		f, fin := rvOuvrir(t, racine, id, "campagne/r-veh-bits")
		if f == nil {
			continue
		}
		vus := map[uint32]int{}
		rvRecords(f, physique, func(r *rvRecord) {
			if !r.pre.b14 || vus[r.mpp] >= n {
				return
			}
			vus[r.mpp]++
			ds := rvOuEstN2(r.pay, r.pre)
			fin := r.pre.dsFin + 32
			if len(ds) > 0 {
				fin = r.pre.dsFin + ds[0] + 32
			}
			// position du bit b14 : on la retrouve en relisant V et MPP.
			br := LecteurSur(r.pay)
			br.PoserContexte(f.fc.ContexteDeLecture())
			br.SetBitPos(r.pre.dsDebut)
			consumeVersionPrefix(br)
			consumeMultiplayerPropertiesBlock(br)
			var sb strings.Builder
			for q := br.BitPos(); q < fin && q < len(r.pay)*8; q++ {
				if q == r.pre.dsFin {
					sb.WriteByte('|')
				}
				sb.WriteByte(byte('0' + sourceBits32(r.pay, q)>>31))
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%04d\t%05d\t%08x\t%d\t%v\t%s", f.id, f.build, r.chunk,
				r.b.Slot, r.mpp, r.pre.quat, ds, sb.String()))
		})
		fin()
	}
	b2Ecrire(t, sortie, "r_veh_ti40_bits.tsv", lignes)
}
