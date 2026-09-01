package himap

// origine_typage_gamefiles_test.go — TYPER les points d'apparition rendus par la recette.
//
// La recette dit OU un objet ramassable nait. Elle ne dit pas LEQUEL. Cette sonde descend au
// `foki` de chaque type retenu et confronte ses references a DEUX tables deja en production :
//
//	le manifeste d'equipement du titre  `[[equipment_objects]]` de replay_labels.toml — 21
//	                                    GlobalID `eqip` avec leur famille (`grenade_frag`,
//	                                    `thruster`, `wall`, ...). C'est la table meme qui a
//	                                    servi a etablir les classes 2/3 du canal natif au
//	                                    schema 31 : le recoupement demande est donc DIRECT.
//	les trois socles d'armes prouves    `mapvar.PadFamilyOf` — power / rack / powerup.
//
// LECTURE SEULE sur les fichiers du jeu installe.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/testutil"
)

// origManifesteEquipement lit les `[[equipment_objects]]` du manifeste du titre.
// On parse le TOML a la main pour ne pas trainer une dependance de config dans `himap`.
func origManifesteEquipement(t *testing.T) map[uint32]string {
	t.Helper()
	// Le manifeste est VERSIONNE (config/titles/...) : sa racine se deduit de
	// l'emplacement du source, jamais d'une variable d'environnement. LEVELUP_REPO_ROOT
	// n'est jamais posee en CI — le test se serait skippe en silence sur la seule machine
	// ou il aurait pu servir (garde archlint TestNoProdRepoRootHelperInTests).
	racineDepot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	chemin := filepath.Join(racineDepot, "config", "titles", "halo_infinite", "mappings",
		"replay_labels.toml")
	b, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("manifeste versionne illisible (%s) : %v", chemin, err)
	}
	reID := regexp.MustCompile(`(?m)^id\s*=\s*"0x([0-9a-fA-F]{8})"`)
	reFam := regexp.MustCompile(`(?m)^family\s*=\s*"([^"]+)"`)
	out := map[uint32]string{}
	blocs := strings.Split(string(b), "[[equipment_objects]]")
	for _, bl := range blocs[1:] {
		mID := reID.FindStringSubmatch(bl)
		mFam := reFam.FindStringSubmatch(bl)
		if mID == nil || mFam == nil {
			continue
		}
		v, err := strconv.ParseUint(mID[1], 16, 32)
		if err != nil {
			continue
		}
		out[uint32(v)] = mFam[1]
	}
	return out
}

// TestOrigineTypageDesPoints publie, pour chaque type retenu, ce que son `foki` atteint.
func TestOrigineTypageDesPoints(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	manif := origManifesteEquipement(t)
	if len(manif) == 0 {
		t.Skip("manifeste d'equipement vide")
	}
	idxForge := origIndexForge(t, racine)
	modCarte := moduleDuJeu(t, "pc", "catalyst")
	geo, _ := GeometrySearchPath(racine, modCarte)
	idxRef, err := NewModuleIndex(append(append([]string{}, geo...),
		filepath.Join(racine, "any", "globals", "forge", "forge_objects-rtx-new.module"))...)
	if err != nil {
		t.Skipf("index de reference indisponible : %v", err)
	}
	types := origTypesRetenus(t)
	t.Logf("== TYPAGE DES %d POINTS · manifeste %d eqip · index %d entrees ==",
		len(types), len(manif), idxRef.Taille())
	compte := map[string]int{}
	for _, ty := range types {
		fokis := origFokiDe(idxForge, idxRef, ty)
		// Ce que le `foki` atteint : on regarde CHAQUE hash reference, resolu ou non, et on
		// le confronte au manifeste. Un `eqip` du manifeste peut ne figurer dans aucun module
		// indexe ici : c'est l'IDENTIFIANT qui compte, pas sa resolution.
		fams := map[string]bool{}
		groupes := map[string]int{}
		for _, f := range fokis {
			sous, err := idxRef.Extract(f)
			if err != nil {
				continue
			}
			RefsInline(sous, func(h uint32) bool {
				if fam, ok := manif[h]; ok {
					fams[fam] = true
				}
				if g, _, ok := idxRef.Lookup(h); ok {
					groupes[g]++
				}
				return false
			})
		}
		nature := "INDETERMINE"
		if pf, ok := origSocleConnu(ty); ok {
			nature = "ARME (socle prouve : " + pf + ")"
		} else if len(fams) > 0 {
			liste := make([]string, 0, len(fams))
			for f := range fams {
				liste = append(liste, f)
			}
			sort.Strings(liste)
			nature = "EQUIPEMENT/GRENADE -> " + strings.Join(liste, ",")
		}
		compte[strings.SplitN(nature, " ", 2)[0]]++
		gs := make([]string, 0, len(groupes))
		for g := range groupes {
			gs = append(gs, g+":"+itoaSimple(groupes[g]))
		}
		sort.Strings(gs)
		if len(gs) > 6 {
			gs = gs[:6]
		}
		t.Logf("0x%08X  %-46s  [%s]", ty, nature, strings.Join(gs, " "))
	}
	t.Logf("BILAN : %v", compte)
}

// origSocleConnu rend la famille de socle d'un type quand c'est l'un des trois PROUVES.
// La table est recopiee ici volontairement : `himap` ne doit pas importer `replay/mapvar`.
func origSocleConnu(ty uint32) (string, bool) {
	switch ty {
	case 0x5F379533:
		return "power", true
	case 0x6253CFC0:
		return "rack", true
	case 0x5E86D110:
		return "powerup", true
	}
	return "", false
}

// origTypesRetenus relit la liste produite par la recette (`ORIGINE_TYPES`). On la RELIT au
// lieu de la recalculer : c'est exactement l'artefact que le generateur consommera, et le lire
// est le seul moyen de tester la chaine plutot que la fonction.
func origTypesRetenus(t *testing.T) []uint32 {
	t.Helper()
	chemin := os.Getenv("ORIGINE_TYPES")
	if chemin == "" {
		t.Skip("ORIGINE_TYPES non defini : produire d'abord la liste via ORIGINE_TYPES_OUT")
	}
	b, err := os.ReadFile(chemin)
	if err != nil {
		t.Skipf("liste de types illisible : %v", err)
	}
	var ids []uint32
	if err := json.Unmarshal(b, &ids); err != nil {
		t.Fatalf("liste de types illisible : %v", err)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
