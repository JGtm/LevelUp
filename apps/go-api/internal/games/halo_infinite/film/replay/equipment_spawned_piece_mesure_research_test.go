package replay

// equipment_spawned_piece_mesure_research_test.go — LOT H.2 : LA MESURE AVANT / APRES DE LA
// REGLE DES PIECES ENGENDREES, sur le parc d'artefacts.
//
// CE QU'ELLE MESURE, ET POURQUOI ELLE N'A PAS BESOIN DE RECUIRE. La regle H.2 est une
// REECRITURE PURE du champ `origin` d'une pose publiee, decidee sur le seul `id` de l'objet
// (`equipmentIsSpawnedPiece`). Elle n'entre ni dans le decodage, ni dans le choix des poses, ni
// dans leurs bornes : appliquer le predicat aux poses DEJA PUBLIEES d'un artefact rend donc,
// pose par pose, exactement ce que le constructeur rendrait apres recuisson. La mesure se fait
// ainsi sans cuire un seul film — et sans toucher au cache, qui est lu en SEULE LECTURE.
//
// GARDE : `H2_ARTS` = racine des artefacts (`data/cache/replays/halo_infinite`). Sans elle le
// test se skippe : la CI n'a pas de parc.
//
//	CGO_ENABLED=0 H2_ARTS=<depot>/data/cache/replays/halo_infinite \
//	  go test ./internal/games/halo_infinite/film/replay/ \
//	  -run '^TestH2MesurePiecesEngendrees$' -count=1 -v

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// h2ArtsEnv — la racine des artefacts a lire.
const h2ArtsEnv = "H2_ARTS"

// h2Art est la part de l'artefact que cette mesure lit : les poses, et rien d'autre.
type h2Art struct {
	SchemaVersion int `json:"schemaVersion"`
	Placements    []struct {
		Family string `json:"family"`
		ID     string `json:"id"`
		Origin string `json:"origin"`
	} `json:"equipmentPlacements"`
}

// TestH2MesurePiecesEngendrees compte les poses par FAMILLE x ORIGINE avant et apres la regle,
// et isole ce qui bouge.
func TestH2MesurePiecesEngendrees(t *testing.T) {
	dir := os.Getenv(h2ArtsEnv)
	if dir == "" {
		t.Skipf("mesure H.2 : definir %s (racine des artefacts, lecture seule)", h2ArtsEnv)
	}
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("racine des artefacts illisible : %v", err)
	}

	avant, apres := map[string]int{}, map[string]int{}
	bascules := map[string]int{}
	films, avecPoses, poses := 0, 0, 0
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".json") || strings.Contains(nom, ".derived.") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, nom))
		if err != nil {
			continue
		}
		var a h2Art
		if err := json.Unmarshal(raw, &a); err != nil {
			continue
		}
		films++
		if len(a.Placements) > 0 {
			avecPoses++
		}
		for _, p := range a.Placements {
			poses++
			// ORIGINE ABSENTE = artefact anterieur au schema 10 : elle se lit `unknown` et ne
			// se devine pas (contrat d'EquipmentPlacement.Origin). La regle H.2 s'y applique
			// quand meme : une piece engendree l'est quel que soit le schema qui l'a publiee.
			o := p.Origin
			if o == "" {
				o = OriginUnknown
			}
			avant[p.Family+"/"+o]++
			n := o
			if equipmentIsSpawnedPiece(p.ID) {
				n = OriginDeployed
			}
			apres[p.Family+"/"+n]++
			if n != o {
				bascules[fmt.Sprintf("%s %s -> %s", p.ID, o, n)]++
			}
		}
	}

	t.Logf("parc lu : %d artefacts, %d avec des poses, %d poses", films, avecPoses, poses)
	t.Log("famille/origine : avant -> apres")
	cles := make([]string, 0, len(avant))
	for k := range avant {
		cles = append(cles, k)
	}
	for k := range apres {
		if _, ok := avant[k]; !ok {
			cles = append(cles, k)
		}
	}
	sort.Strings(cles)
	var bouges int
	for _, k := range cles {
		marque := ""
		if avant[k] != apres[k] {
			marque = "   <-- CHANGE"
			bouges++
		}
		t.Logf("  %-28s %6d -> %6d%s", k, avant[k], apres[k], marque)
	}
	t.Logf("%d croisement(s) famille x origine deplace(s)", bouges)

	t.Log("bascules, par objet :")
	bk := make([]string, 0, len(bascules))
	for k := range bascules {
		bk = append(bk, k)
	}
	sort.Strings(bk)
	total := 0
	for _, k := range bk {
		t.Logf("  %-40s %4d", k, bascules[k])
		total += bascules[k]
	}
	t.Logf("TOTAL bascule : %d pose(s)", total)

	// L'INVARIANT QUI BORNE LA PORTEE : rien ne bascule hors d'une piece engendree, et le
	// total des poses ne bouge pas. Ce n'est pas un reglage, c'est la definition du predicat —
	// mais c'est aussi ce qu'un refactor casserait sans bruit.
	var avantT, apresT int
	for _, v := range avant {
		avantT += v
	}
	for _, v := range apres {
		apresT += v
	}
	if avantT != apresT || avantT != poses {
		t.Errorf("total des poses : %d avant, %d apres, %d lues", avantT, apresT, poses)
	}
	for k := range bascules {
		if !strings.HasSuffix(k, "-> "+OriginDeployed) {
			t.Errorf("bascule inattendue : %q — la regle ne promeut QUE vers %q", k, OriginDeployed)
		}
	}
}
