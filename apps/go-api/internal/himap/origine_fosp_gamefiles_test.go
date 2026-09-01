package himap

// origine_fosp_gamefiles_test.go — ELUCIDER `fosp`, le quatrieme cran.
//
// Le manifeste d'equipement du titre ne recoupe AUCUNE reference des `foki` (16 types sondes,
// 13 indetermines) : la chaine ne descend pas jusqu'a l'`eqip` en une etape. Sous chaque point
// d'apparition figure invariablement `fosp:4` — meme cardinalite pour les socles prouves et
// pour les candidats. Si `fosp` porte l'objet engendre, il type les 13 restants ; sinon on le
// consigne comme non elucide et le typage passe par une autre voie.
//
// LECTURE SEULE sur les fichiers du jeu installe.

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const origGroupeSpawn = "fosp"

// TestOrigineFospElucidation publie ce que les `fosp` d'un point atteignent.
func TestOrigineFospElucidation(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	idxForge := origIndexForge(t, racine)
	modCarte := moduleDuJeu(t, "pc", "catalyst")
	geo, _ := GeometrySearchPath(racine, modCarte)
	idxRef, err := NewModuleIndex(append(append([]string{}, geo...),
		filepath.Join(racine, "any", "globals", "forge", "forge_objects-rtx-new.module"))...)
	if err != nil {
		t.Skipf("index de reference indisponible : %v", err)
	}
	manif := origManifesteEquipement(t)
	types := origTypesRetenus(t)
	t.Logf("== `fosp` DES %d POINTS · index %d entrees ==", len(types), idxRef.Taille())
	for _, ty := range types {
		fokis := origFokiDe(idxForge, idxRef, ty)
		// Collecte des `fosp` sous les `foki`.
		var fosps []uint32
		vus := map[uint32]bool{}
		for _, f := range fokis {
			sous, err := idxRef.Extract(f)
			if err != nil {
				continue
			}
			RefsInline(sous, func(h uint32) bool {
				if g, _, ok := idxRef.Lookup(h); ok && g == origGroupeSpawn && !vus[h] {
					vus[h] = true
					fosps = append(fosps, h)
				}
				return false
			})
		}
		// Ce que les `fosp` atteignent a leur tour.
		groupes := map[string]int{}
		fams := map[string]bool{}
		for _, sp := range fosps {
			tag, err := idxRef.Extract(sp)
			if err != nil {
				continue
			}
			RefsInline(tag, func(h uint32) bool {
				if fam, ok := manif[h]; ok {
					fams[fam] = true
				}
				if g, _, ok := idxRef.Lookup(h); ok {
					groupes[g]++
				}
				return false
			})
		}
		gs := make([]string, 0, len(groupes))
		for g, n := range groupes {
			gs = append(gs, g+":"+itoaSimple(n))
		}
		sort.Strings(gs)
		if len(gs) > 8 {
			gs = gs[:8]
		}
		fl := make([]string, 0, len(fams))
		for f := range fams {
			fl = append(fl, f)
		}
		sort.Strings(fl)
		marque := ""
		if pf, ok := origSocleConnu(ty); ok {
			marque = " <<< socle prouve " + pf
		}
		t.Logf("0x%08X  %d fosp -> %s%s%s", ty, len(fosps), strings.Join(gs, " "),
			manifSuffixe(fl), marque)
	}
}

func manifSuffixe(fl []string) string {
	if len(fl) == 0 {
		return ""
	}
	return "  MANIFESTE:" + strings.Join(fl, ",")
}
