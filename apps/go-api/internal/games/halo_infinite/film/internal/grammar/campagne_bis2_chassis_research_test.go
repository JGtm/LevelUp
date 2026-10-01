//go:build research

package grammar

// campagne_bis2_chassis_research_test.go — MESURES BIS 2 DE LA CAMPAGNE DE GRAMMAIRE (2026-10-01),
// T6-C2 : L IDENTITE DES CHASSIS `ti=40` ABSENTS DES MODULES INSTALLES (`77ef810a`, `4118381d`,
// `d0b40d0a`), par les DONNEES DE CREATION des films.
//
// Le bloc `object-multiplayer-properties` d un record de creation (NEW ou image-cle) porte, a cote
// du mot d identite `MPPWord32` (le chassis), le `variant-name` et le nom de queue (deux
// identifiants de chaine de 32 bits). Un chassis d un ancien build, absent de l installation, qui
// porte le MEME nom de variante qu un chassis identifie est une redeclaration du meme vehicule
// (meme regle que la table des familles : un vehicule a un GlobalID par module). L instrument
// releve, par film et par chassis, ces deux noms, le nombre de creations et l altitude de creation.
// Lecture seule ; aucune sortie de production ne change.
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -run '^TestCampagneBis2IdentiteChassis$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// b2cCle : un chassis et ses deux noms.
type b2cCle struct {
	chassis, variante, queue uint64
	source                   string
}

// TestCampagneBis2IdentiteChassis releve les noms MPP des chassis `ti=40` de chaque film.
func TestCampagneBis2IdentiteChassis(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tsource\tchassis\tvariant_name\ttail_name\tcreations\tz_min\tz_max"}
	for _, id := range films {
		garde := filmproc.Arm("campagne/bis2-chassis", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		type cumul struct {
			n          int
			zmin, zmax float32
		}
		acc := map[b2cCle]*cumul{}
		ajouter := func(k b2cCle, z float32, aZ bool) {
			x := acc[k]
			if x == nil {
				x = &cumul{zmin: 1e9, zmax: -1e9}
				acc[k] = x
			}
			x.n++
			if aZ {
				if z < x.zmin {
					x.zmin = z
				}
				if z > x.zmax {
					x.zmax = z
				}
			}
		}
		wr := profile.QuantRangeCEBiped
		if cre, _, err := ScanVehicleCreations(f.fc, &wr); err == nil {
			for _, c := range cre {
				ajouter(b2cCle{chassis: c.MPPVal[MPPWord32], variante: c.MPPVal[MPPVariantName],
					queue: c.MPPVal[MPPTailName], source: "NEW"}, c.Z, true)
			}
		}
		restore, errMPP := InstallFilmFormatMPP(f.fc)
		ctx := f.fc.ContexteDeLecture()
		marche := f.fc.MarcheDImageCle()
		for _, num := range f.fc.ChunkNumbers() {
			data, pks, ok := f.fc.ChunkAt(num)
			if !ok {
				continue
			}
			for _, pk := range pks {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				pay := pk.Payload(data)
				for _, r := range marche.Records(pay) {
					if r.TI != 40 {
						continue
					}
					var vals [MPPFieldCount]uint64
					vus := 0
					ctx.Obs = &Observation{MppHook: func(fl MPPField, v uint64, present bool) {
						if present && int(fl) < len(vals) {
							vals[fl] = v
							vus++
						}
					}}
					WalkKeyframeFullState(pay, r.Bit, f.reg, ctx)
					if vus > 0 {
						ajouter(b2cCle{chassis: vals[MPPWord32], variante: vals[MPPVariantName],
							queue: vals[MPPTailName], source: "image-cle"}, 0, false)
					}
				}
			}
		}
		if errMPP == nil {
			restore()
		}
		for k, x := range acc {
			zs := "\t"
			if x.zmax >= x.zmin {
				zs = fmt.Sprintf("%.1f\t%.1f", x.zmin, x.zmax)
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%08x\t%08x\t%08x\t%d\t%s", id, f.build, k.source,
				k.chassis, k.variante, k.queue, x.n, zs))
		}
		t.Logf("%s %s : %d cles chassis/noms", id, f.build, len(acc))
		garde.Disarm()
	}
	corps := lignes[1:]
	sort.Strings(corps)
	b2Ecrire(t, sortie, "mb2_chassis_noms.tsv", append([]string{lignes[0]}, corps...))
}
