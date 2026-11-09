//go:build gamefiles

package himap

import (
	"os"
	"path/filepath"
	"testing"
)

// LE RELEVE — combien chacun des trois filtres de visibilite retirerait-il ?
//
// Trois declarations du format designent de la geometrie que le jeu ne veut pas voir sur une
// vue de dessus (cf. filtres_reclaimer.go). Aucune n'est branchee sur la cuisson. La question
// n'est PAS « faut-il les brancher » mais « combien cela retire-t-il » : un filtre qui touche
// 0,1 % des triangles ne vaut pas un lot, un filtre qui en touche 40 % doit etre regarde a
// l'image avant d'etre pose. Ces tests ne corrigent rien et ne cuisent rien — ils comptent.
//
// AUCUN MAILLAGE N'EST DECODE. Les tags sont ouverts avec un blob de ressources NIL et les
// triangles se lisent aux descripteurs (`TrianglesDuMaillage`) : c'est ce qui rend le releve
// tenable en memoire la ou une cuisson demande des giga-octets.

// ---------------------------------------------------------------------------
// 3. LE RELEVE
// ---------------------------------------------------------------------------

// nomsBitsFlagsInstance : les 16 drapeaux du champ `flags` d'une instance, dans l'ordre du
// plugin. Ils ne sont pas relus du XML ici — `TestPluginDrapeauxDeFiltre` verrouille deja
// l'accord — mais ecrits pour que le balayage se lise sans le plugin sous les yeux.
var nomsBitsFlagsInstance = [16]string{
	"render only", "does not block aoe damage", "remove from dynamic shadow geometry",
	"cinema only", "exclude from cinema", "disable play collision", "disable bullet collision",
	"ignore cubemap volume", "always generate floating shadow", "PVS always visible",
	"PVS always use LOD 0", "PVS don't use as an occluder", "exclude from intel map",
	"Exclude from broadphase calculation", "can generate decorators", "isHLOD",
}

// TestBalayageDrapeauxInstances balaye TOUS les modules installes et compte, BIT A BIT, le
// champ `flags` des instances de geometrie.
//
// POURQUOI L'HISTOGRAMME COMPLET ET PAS LE SEUL BIT 12. Un compte de zero ne dit pas la meme
// chose selon ce que valent les autres bits : si le champ entier etait toujours nul, ce
// serait l'OFFSET qui serait faux, pas le drapeau qui serait inutilise. Le releve des 16 bits
// separe les deux — et c'est la seule facon de conclure « inutilise » sans se tromper.
//
// Le balayage porte aussi sur les modules GLOBAUX : la geometrie de Live Fire vit dans
// `pc/globals/common-rtx-new.module`, pas dans le module de sa carte.
func TestBalayageDrapeauxInstances(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	dir, err := LevelsDir("pc")
	if err != nil {
		t.Skip(err)
	}
	dossiers, err := os.ReadDir(dir)
	if err != nil {
		t.Skip(err)
	}
	var chemins []string
	for _, d := range dossiers {
		if !d.IsDir() {
			continue
		}
		p := filepath.Join(dir, d.Name(), d.Name()+"-rtx-new.module")
		if _, err := os.Stat(p); err == nil {
			chemins = append(chemins, p)
		}
	}
	globs, _ := filepath.Glob(filepath.Join(racine, "pc", "globals", "*.module"))
	chemins = append(chemins, globs...)

	var total int
	var bits [16]int
	for _, p := range chemins {
		bsps, err := ReadModuleInstances(p)
		if err != nil {
			// ErrAucunTagSbsp est un fait connu (sgh_interlock), pas une panne : on le dit
			// et on continue, sans le confondre avec une chaine cassee.
			t.Logf("%-28s : %v", filepath.Base(p), err)
			continue
		}
		n, intel := 0, 0
		for _, b := range bsps {
			for _, in := range b.Instances {
				n++
				for k := 0; k < 16; k++ {
					if in.Flags&(1<<uint(k)) != 0 {
						bits[k]++
					}
				}
				if in.ExclueDeCarteIntel() {
					intel++
				}
			}
		}
		total += n
		t.Logf("%-28s : %6d instances · %d `exclude from intel map`", filepath.Base(p), n, intel)
	}
	t.Logf("TOTAL %d instances sur %d modules", total, len(chemins))
	for k, nom := range nomsBitsFlagsInstance {
		part := 0.0
		if total > 0 {
			part = 100 * float64(bits[k]) / float64(total)
		}
		t.Logf("   bit %2d %-38s %8d  (%5.2f%%)", k, nom, bits[k], part)
	}
	if total == 0 {
		t.Skip("aucune instance lue")
	}
	nonNuls := 0
	for _, n := range bits {
		if n > 0 {
			nonNuls++
		}
	}
	// Le champ doit VIVRE. S'il ne portait aucun bit sur des dizaines de milliers
	// d'instances, la conclusion a tirer serait « offset faux », pas « drapeau inutilise ».
	if nonNuls == 0 {
		t.Fatalf("aucun des 16 bits de `flags` n'est jamais pose sur %d instances : "+
			"l'offset %#x est suspect", total, insOffFlags)
	}
}

// ---------------------------------------------------------------------------
// 4. ISOLATION — la carte Forge du releve
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// outils communs
// ---------------------------------------------------------------------------

func indexDeCarte(t *testing.T, racine, modCarte string) *ModuleIndex {
	t.Helper()
	chemins, err := GeometrySearchPath(racine, modCarte)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

// assetsNatifs ouvre au plus `max` tags rtgo distincts references par une carte installee.
func assetsNatifs(t *testing.T, dossier string, max int) []*RuntimeGeoAsset {
	t.Helper()
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	modCarte := moduleDuJeu(t, "pc", dossier)
	idx := indexDeCarte(t, racine, modCarte)
	bsps, err := ReadModuleInstances(modCarte)
	if err != nil {
		t.Fatal(err)
	}
	var bsp BSPInstances
	for _, b := range bsps {
		if len(b.Instances) > len(bsp.Instances) {
			bsp = b
		}
	}
	vus := map[uint32]bool{}
	var out []*RuntimeGeoAsset
	for _, in := range bsp.Instances {
		id := in.RuntimeGeoID()
		if vus[id] || len(out) >= max {
			continue
		}
		vus[id] = true
		if g, _, ok := idx.Lookup(id); !ok || g != GroupeRtgo {
			continue
		}
		tag, err := idx.Extract(id)
		if err != nil {
			continue
		}
		if a, err := NewRuntimeGeoAsset(tag, nil); err == nil {
			out = append(out, a)
		}
	}
	return out
}
