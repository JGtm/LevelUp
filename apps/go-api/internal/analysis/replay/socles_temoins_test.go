package replay

// socles_temoins_test.go — LA MESURE DU CROISEMENT SUR LES QUATRE TÉMOINS, et le test de
// la décision produit.
//
// CE QU'IL MESURE, artefact cuit par artefact cuit : combien d'emplacements la carte porte
// au fichier, combien le match en ALLUME, combien restent éteints, et ce que le calque pose
// À LA PREMIÈRE IMAGE avant et après le croisement.
//
// LE TÉMOIN QUI TRANCHE est `000d5950`, Cliffhanger en Super Fiesta : la carte porte
// dix-huit emplacements au fichier et le film n'en sert AUCUN. Rien ne doit partir au
// client. C'est le test de la décision du 2026-08-19 (« on ne les affiche que si allumés »),
// et il échoue si un jour quelqu'un décide de publier le catalogue brut.
//
// SOUS GARDE : les artefacts de rejeu ne sont pas versionnés (data/cache/replays/). Sans la
// variable, le test SKIPPE — un test qui se déclare vert sans avoir rien lu ne garde rien.
//
// USAGE (depuis apps/go-api) :
//
//	SOCLES_TEMOINS=<depot>/data/cache/replays/halo_infinite \
//	  go test ./internal/analysis/replay/ -run '^TestSoclesTemoins' -v

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// soclesTemoinsEnv porte le dossier des artefacts de rejeu cuits.
const soclesTemoinsEnv = "SOCLES_TEMOINS"

// temoinSocle est un film de mesure : son identifiant, le MODULE de sa carte (la jointure
// disponible hors ligne — le map_id vit en base) et ce qu'on attend de lui.
type temoinSocle struct {
	film   string
	module string
	mode   string
	// allumesAttendus : le nombre d'emplacements que le match doit allumer. -1 = non figé.
	allumesAttendus int
}

// lesTemoins — les quatre films du lot, deux cartes, quatre modes.
var lesTemoins = []temoinSocle{
	{"01e1f945", "catalyst", "KOTH", -1},
	{"64e8adfa", "catalyst", "CTF", -1},
	{"bcb6d393", "cliffhanger_ridgeline", "CTF", -1},
	// LE TÉMOIN DE LA DÉCISION : zéro socle au film, donc zéro à l'écran.
	{"000d5950", "cliffhanger_ridgeline", "Super Fiesta", 0},
}

// bilanTemoin est ce qu'un film rend à la mesure.
type bilanTemoin struct {
	filmPads   int
	catalogueN int
	allumes    int
	eteints    int
	orphelins  int // socles du film qu'aucun emplacement ne réclame
	// pointsImage0 / iconesImage0 : ce que le calque pose à l'image 0, AVANT et APRÈS.
	pointsAvant, pointsApres int
	iconesAvant, iconesApres int
	// deplaceMax : le plus grand écart entre la position du film et celle du fichier, en
	// mètres. C'est la seule chose que le croisement change à l'écran.
	deplaceMax float64
}

// TestSoclesTemoins mesure les quatre films et vérifie la décision produit.
func TestSoclesTemoins(t *testing.T) {
	dir := os.Getenv(soclesTemoinsEnv)
	if dir == "" {
		t.Skipf("%s absent — mesure des témoins ignorée", soclesTemoinsEnv)
	}
	path := cheminCatalogueLivre("map_weapon_pads.json")
	if path == "" {
		t.Skip("catalogue des socles absent de cet arbre")
	}
	cat, err := LoadMapWeaponPads(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, tm := range lesTemoins {
		t.Run(tm.film, func(t *testing.T) {
			mesurerTemoin(t, dir, cat, tm)
		})
	}
}

// mesurerTemoin joue le croisement sur un film et publie ses chiffres.
func mesurerTemoin(t *testing.T, dir string, cat *MapWeaponPadsCatalog, tm temoinSocle) {
	t.Helper()
	doc, ok := lireArtefact(t, dir, tm.film)
	if !ok {
		return
	}
	entry, ok := entreeParModule(cat, tm.module)
	if !ok {
		t.Fatalf("module %q absent du catalogue des socles", tm.module)
	}
	b := croiserTemoin(entry, doc.WeaponPads)
	t.Logf("%s (%s, %s) : film %d socles | catalogue %d emplacements | ALLUMÉS %d | éteints %d | orphelins %d",
		tm.film, tm.module, tm.mode, b.filmPads, b.catalogueN, b.allumes, b.eteints, b.orphelins)
	t.Logf("   image 0 — points AVANT %d / APRÈS %d, icônes AVANT %d / APRÈS %d, "+
		"déplacement max %.3f m", b.pointsAvant, b.pointsApres, b.iconesAvant, b.iconesApres, b.deplaceMax)

	if tm.allumesAttendus >= 0 && b.allumes != tm.allumesAttendus {
		t.Errorf("%s : %d emplacements allumés, attendu %d — LA DÉCISION PRODUIT EST ROMPUE",
			tm.film, b.allumes, tm.allumesAttendus)
	}
	// LE CALQUE NE PERD RIEN : autant de socles posés à l'image 0 après qu'avant. Le
	// croisement déplace, il n'ajoute ni ne retranche.
	if b.pointsApres != b.pointsAvant || b.iconesApres != b.iconesAvant {
		t.Errorf("%s : l'image 0 a changé de contenu (points %d->%d, icônes %d->%d)",
			tm.film, b.pointsAvant, b.pointsApres, b.iconesAvant, b.iconesApres)
	}
	// ET IL NE DÉPLACE PAS N'IMPORTE COMMENT : un socle allumé bouge de moins d'un mètre,
	// par construction du seuil de confirmation.
	if b.deplaceMax >= MapWeaponPadMatchM {
		t.Errorf("%s : un socle a été déplacé de %.3f m, seuil %.1f m",
			tm.film, b.deplaceMax, MapWeaponPadMatchM)
	}
}

// croiserTemoin applique le croisement et compte ce que le calque poserait à l'image 0.
//
// LA RÈGLE D'IMAGE 0 EST CELLE DU CALQUE WEB, transcrite : tous les socles posent leur
// POINT à chaque image ; seuls ceux dont l'état n'est pas « vide » posent une ICÔNE
// (weaponPadsLayer.ts, PAD_ALPHA.empty.icon = 0).
func croiserTemoin(e MapWeaponPadsEntry, pads []WeaponPad) bilanTemoin {
	b := bilanTemoin{filmPads: len(pads), catalogueN: len(e.Pads)}
	b.pointsAvant, b.iconesAvant = len(pads), iconesImage0(pads)
	cross := BuildMapWeaponPads(e, pads)
	if cross == nil {
		b.eteints = len(e.Pads)
		b.orphelins = len(pads)
		b.pointsApres, b.iconesApres = b.pointsAvant, b.iconesAvant
		return b
	}
	b.allumes = len(cross.Pads)
	b.eteints = cross.CatalogN - b.allumes
	b.orphelins = len(pads) - b.allumes
	// La liste effectivement dessinée : les socles du film, déplacés là où le catalogue les
	// pose (même règle que crossedWeaponPads côté web).
	dessines := make([]WeaponPad, len(pads))
	copy(dessines, pads)
	for _, spot := range cross.Pads {
		p := dessines[spot.Pad]
		d := mapvar.Dist3(
			mapvar.Vec3{X: float64(p.X), Y: float64(p.Y), Z: float64(p.Z)},
			mapvar.Vec3{X: float64(spot.X), Y: float64(spot.Y), Z: float64(spot.Z)})
		if d > b.deplaceMax {
			b.deplaceMax = d
		}
		p.X, p.Y, p.Z = spot.X, spot.Y, spot.Z
		dessines[spot.Pad] = p
	}
	b.pointsApres, b.iconesApres = len(dessines), iconesImage0(dessines)
	return b
}

// iconesImage0 compte les socles qui posent une icône à l'image 0 : ceux dont une
// occupation a commencé à l'image 0 ou avant, et dont l'absence n'est pas encore prouvée.
func iconesImage0(pads []WeaponPad) int {
	n := 0
	for _, p := range pads {
		if occupationImage0(p) {
			n++
		}
	}
	return n
}

// occupationImage0 dit si le socle porte quelque chose de PROUVÉ à l'image 0.
func occupationImage0(p WeaponPad) bool {
	for _, occ := range p.Presence {
		if occ.T0 > 0 {
			break
		}
		// Plein tant que la présence est prouvée, incertain tant que l'absence ne l'est
		// pas — les deux posent une icône ; vide n'en pose aucune.
		if occ.THigh <= occ.TLow || occ.THigh > 0 {
			return true
		}
	}
	return false
}

// entreeParModule retrouve l'entrée d'une carte par son module — la seule jointure
// disponible hors ligne (le map_id d'un match vit en base, et ce test n'ouvre rien).
func entreeParModule(cat *MapWeaponPadsCatalog, module string) (MapWeaponPadsEntry, bool) {
	ids := make([]string, 0, len(cat.Maps))
	for id := range cat.Maps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if cat.Maps[id].Module == module {
			return cat.Maps[id], true
		}
	}
	return MapWeaponPadsEntry{}, false
}

// lireArtefact charge un artefact de rejeu cuit. Absent = le film n'est pas sur ce poste,
// et la mesure le dit plutôt que d'échouer.
func lireArtefact(t *testing.T, dir, film string) (ReplayDocument, bool) {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("%s.json", film)))
	if err != nil {
		t.Skipf("artefact %s absent : %v", film, err)
		return ReplayDocument{}, false
	}
	var doc ReplayDocument
	if err := json.Unmarshal(blob, &doc); err != nil {
		t.Fatalf("artefact %s illisible : %v", film, err)
	}
	return doc, true
}
