package mapvar

// vehicules_corpus_test.go — L'INVENTAIRE DES VEHICULES POSES PAR LES FICHIERS DE CARTE.
//
// # La question, telle que l'utilisateur la pose (2026-08-31)
//
// « lesquels sont sur quelle map, vehicule aleatoires ou non, points de spawns et cooldown ».
// Les quatre reponses vivent dans le MEME endroit : l'objet place par la variante de carte. Le
// `.mvar` NOMME son vehicule (contrairement aux socles d'arme, qui sont generiques et dont
// l'arme appartient au match — cf. socles.go), il DONNE sa position au centimetre, et son sac
// de gameplay porte un delai de reapparition PAR EMPLACEMENT.
//
// # Le corpus
//
// `cmd/mapobj-build --all --save-mvar <dir>` depose un dossier PAR CARTE (nomme par son
// `map_id`) contenant ses `.mvar`. Ce test balaie cette arborescence. Sans elle il se saute :
// aucune donnee n'est versionnee ici, le corpus se re-telecharge.
//
//	$env:VEHI_MVAR_DIR="C:/.../scratchpad/mvar"
//	go test ./internal/analysis/replay/mapvar/ -run VehiculesCorpus -v -timeout 30m
//
// # Ce que le fichier NE DIT PAS, et qu'on ne devine pas
//
// La MEME reserve que les socles d'arme, mot pour mot : le fichier POSE, le mode ALLUME. Un
// vehicule pose au `.mvar` n'est pas forcement present dans une partie donnee — les etiquettes
// de mode (`Labels`) filtrent, et une carte DEV peut poser ses vehicules dans le scenario de
// base du `.module` plutot que dans la variante. Ce balayage mesure ce que le FICHIER DE CARTE
// porte, ni plus ni moins.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// vehiTypes : les 21 `type_id` de groupe de tag `vehi` de la palette Forge
// (`.ai/V7.5/dumps/forge_zones/palette_complete_groupes.csv`), avec leur nom quand il est
// craque (`palette_noms.csv` + [TestVehiculesNomsCraques] du 2026-08-31).
var vehiTypes = map[int32]string{
	-1825803927: "shade_turret",
	-1751154772: "?1029649325",
	-1718495603: "banshee",
	-1362694062: "?-1773333388",
	-870843776:  "mongoose",
	-522135259:  "wasp",
	-269578988:  "unsc_turret",
	-262750720:  "warthog",
	-188587954:  "wraith",
	-105823600:  "warthog_razorback",
	60452899:    "brute_chopper",
	83469709:    "auto_turret",
	199265464:   "?-2002047233",
	666920711:   "ghost",
	1133144079:  "warthog_gauss",
	1304071901:  "plasma_turret",
	1503350133:  "scorpion",
	2128426546:  "mongoose_gungoose",
	-1430390016: "?1161655938",
	-411259918:  "phantom",
	223996207:   "falcon",
}

// vehiPlacement est un vehicule pose par un fichier de carte.
type vehiPlacement struct {
	mapID, fichier, nom string
	typeID              int32
	x, y, z             float64
	equipe, categorie   int
	labels              []int32
	champsBag           string
}

// TestVehiculesCorpus imprime l'inventaire complet, puis les agregats qui repondent aux
// quatre questions.
func TestVehiculesCorpus(t *testing.T) {
	dir := os.Getenv("VEHI_MVAR_DIR")
	if dir == "" {
		t.Skip("mesure non demandee : VEHI_MVAR_DIR requis (corpus mapobj-build --save-mvar)")
	}
	var poses []vehiPlacement
	fichiers, cartes := 0, map[string]bool{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(p), ".mvar") {
			return nil
		}
		buf, err := os.ReadFile(p)
		if err != nil {
			t.Logf("illisible : %s (%v)", p, err)
			return nil
		}
		v, err := Parse(buf)
		if err != nil {
			t.Logf("non decode : %s (%v)", p, err)
			return nil
		}
		fichiers++
		mapID := filepath.Base(filepath.Dir(p))
		cartes[mapID] = true
		root, _ := DecodeRoot(buf)
		objs, _ := root.Field(3)
		for _, o := range v.Objects {
			nom, ok := vehiTypes[o.TypeID]
			if !ok {
				continue
			}
			poses = append(poses, vehiPlacement{
				mapID: mapID, fichier: filepath.Base(p), nom: nom, typeID: o.TypeID,
				x: o.Pos.X, y: o.Pos.Y, z: o.Pos.Z,
				equipe: o.TeamIndex, categorie: o.Category, labels: o.Labels,
				champsBag: vehiBagFields(objs, o.Index),
			})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("balayage : %v", err)
	}

	sort.Slice(poses, func(i, j int) bool {
		if poses[i].nom != poses[j].nom {
			return poses[i].nom < poses[j].nom
		}
		return poses[i].mapID < poses[j].mapID
	})
	t.Logf("CORPUS : %d fichier(s) .mvar sur %d carte(s), %d vehicule(s) pose(s)",
		fichiers, len(cartes), len(poses))

	t.Log("=== INVENTAIRE (nom, carte, fichier, position, equipe, categorie, labels, champs du sac)")
	for _, p := range poses {
		t.Logf("VEHI\t%s\t%s\t%s\t%.2f\t%.2f\t%.2f\tequipe=%d\tcat=%d\tlabels=%v\t%s",
			p.nom, p.mapID, p.fichier, p.x, p.y, p.z, p.equipe, p.categorie, p.labels, p.champsBag)
	}

	t.Log("=== PAR VEHICULE : combien d'exemplaires, sur combien de cartes")
	parNom := map[string]map[string]int{}
	for _, p := range poses {
		if parNom[p.nom] == nil {
			parNom[p.nom] = map[string]int{}
		}
		parNom[p.nom][p.mapID]++
	}
	noms := make([]string, 0, len(parNom))
	for n := range parNom {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	for _, n := range noms {
		total := 0
		for _, c := range parNom[n] {
			total += c
		}
		t.Logf("  %-20s %3d exemplaire(s) sur %2d carte(s)", n, total, len(parNom[n]))
	}
}

// vehiBagFields imprime les champs SCALAIRES du sac de gameplay d'un objet, avec leur type —
// c'est la sonde qui cherche le delai de reapparition (`respawnTime`) et les eventuels
// reglages d'aleatoire. Le sac n'est pas expose par [Object] : on relit l'arbre brut.
func vehiBagFields(objs Value, index int) string {
	if index < 0 || index >= len(objs.Items) {
		return "sac=absent"
	}
	bag, ok := objs.Items[index].Field(8)
	if !ok {
		return "sac=absent"
	}
	var out []string
	for sousChamp := uint16(0); sousChamp <= 3; sousChamp++ {
		sub, ok := bag.Field(sousChamp)
		if !ok || len(sub.Items) == 0 {
			continue
		}
		gp := sub.Items[0]
		cles := make([]int, 0, len(gp.Fields))
		for k := range gp.Fields {
			cles = append(cles, int(k))
		}
		sort.Ints(cles)
		for _, k := range cles {
			f := gp.Fields[uint16(k)]
			if len(f.Fields) > 0 || len(f.Items) > 0 {
				continue // sous-arbre : forme, labels — deja lus par Object
			}
			out = append(out, fmt.Sprintf("#%d/%d[0]/#%d=t%d:i%d/u%d/f%.3f",
				8, sousChamp, k, f.Type, f.Int, f.Uint, f.Float))
		}
	}
	if len(out) == 0 {
		return "sac=vide"
	}
	return strings.Join(out, " ")
}
