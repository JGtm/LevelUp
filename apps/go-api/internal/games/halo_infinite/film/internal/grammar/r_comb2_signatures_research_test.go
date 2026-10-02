//go:build research && campagne_overlay

package grammar

// r_comb2_signatures_research_test.go — R-COMB-2, point 3 (critique B7) : la signature du
// localisateur sous L8. La regle R-LS identifie l archetype « high-frequency » par NOM ; L8 route
// `ti=3 i1 high-frequency` vers sa table (26 bits). Cette sonde releve, pour chaque paquet a
// evenements, l ARCHETYPE du record de la signature trouvee, au moment de la localisation (monde
// d avant le paquet) : signature stricte du slot 123 (`marchLocateStrict`, production) et
// signature high-frequency de LS. Quatre marches : reference, L8, LS, L8+LS.
//
//	(memes variables que TestRComb2) -run '^TestRComb2Signatures$'

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rc2TiEn : l archetype du record DELTA lu a `s` (sans toucher l observation).
func rc2TiEn(pay []byte, s int, w *World, cfg FrameConfig) string {
	defer cfg.Obs.neutraliserEtatsDeMouvement()()
	rec, end, ok := TryDeltaAt(pay, s, w, cfg)
	if !ok {
		return "illisible"
	}
	return fmt.Sprintf("ti=%d slot=%d bits=%d comps=%d", rec.TypeIndex, rec.Slot, end-s, len(rec.Trace.Comps))
}

// TestRComb2Signatures ecrit `r_comb2_signatures_ti.tsv`.
func TestRComb2Signatures(t *testing.T) {
	racine, sortie, films := b2Env(t)
	cartes := map[string]string{}
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_CARTES"), ";") {
		if id, carte, ok := strings.Cut(x, "="); ok {
			cartes[strings.TrimSpace(id)] = strings.TrimSpace(carte)
		}
	}
	cat, _ := profile.LoadMapQuantCatalog(os.Getenv("CAMPAGNE_CATALOGUE"))
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tconfig\tsignature\trecord\tpaquets"}
	defer rc2Lever()
	for _, id := range films {
		var f *cmFilm
		var ok bool
		if c := cartes[id]; c != "" && cat != nil {
			entry, err := cat.Lookup(c)
			if err != nil {
				continue
			}
			f, ok = b2pOuvrirProduction(t, racine, id, entry, utiles)
		} else {
			f, ok = cmOuvrir(t, racine, id, utiles)
		}
		if !ok {
			continue
		}
		x := &rc2Film{f: f, b: cmLireBlocs(f), hf: rlocArchetypesHF(f.reg)}
		for _, k := range []rc2L{{}, rc2Avec("L8"), rc2Avec("LS"), rc2Avec("L8", "LS")} {
			n := map[string]int{}
			x.poser(k)
			tete := func(int) func([]byte, *World, FrameConfig) (int, bool) {
				return func(pay []byte, w *World, cfg FrameConfig) (int, bool) {
					if s := marchLocateStrict(pay, w, cfg); s >= 0 {
						n["123 strict\t"+rc2TiEn(pay, s, w, cfg)]++
					}
					if !k.ls {
						return debutDeLaListe(pay, w, cfg)
					}
					var tr rlocTrace
					d, okL := rlocLocaliser("ls2", x.hf, pay, w, cfg, &tr, nil)
					if strings.HasPrefix(tr.passe, "strict-hf") {
						_, sHF, _ := rlocSignatures(pay, w, cfg, x.hf)
						n["high-frequency (LS)\t"+rc2TiEn(pay, sHF, w, cfg)]++
					}
					return d, okL
				}
			}
			cmMarcher(f, cmVariante{tete: tete}, nil)
			rc2Lever()
			for cle, v := range n {
				lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%s\t%d", id, f.build, k.nom(), cle, v))
			}
		}
		t.Logf("%s %s : fait", id, f.build)
	}
	b2Ecrire(t, sortie, "r_comb2_signatures_ti.tsv", lignes)
}
