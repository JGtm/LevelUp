package himap

// LA CLE D'UN FOND FORGE EST SON map_id — garde-rails de la decision du 2026-08-13
// (plan fonds par map_id). Un canevas (fo08_wetland, fo11_blank, ...) est partage par des
// dizaines de cartes Forge : publier un fond sous la cle du canevas, c'est servir la carte
// d'un autre match. Ces tests verifient la DECLARATION (CartesForge) et l'ASSET VERSIONNE
// (map_backgrounds/) — ils tournent partout, sans installation du jeu.

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
)

// regexpMapID : la forme d'un asset UGC (uuid v4 minuscule) — la cle de publication Forge.
var regexpMapID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// TestCartesForgeDeclarations — chaque declaration porte une cle map_id valide et UNIQUE,
// un nom, un `.mvar` de CARTE (jamais le rack du canevas) et un canevas connu.
func TestCartesForgeDeclarations(t *testing.T) {
	vus := map[string]string{}
	for _, c := range CartesForge {
		if !regexpMapID.MatchString(c.MapID) {
			t.Errorf("%s : MapID %q n'est pas un asset UGC — la cle de publication serait fausse", c.Nom, c.MapID)
		}
		if autre, deja := vus[c.MapID]; deja {
			t.Errorf("MapID %s declare deux fois (%s et %s)", c.MapID, autre, c.Nom)
		}
		vus[c.MapID] = c.Nom
		if strings.TrimSpace(c.Nom) == "" {
			t.Errorf("%s : Nom vide — rapports et logs illisibles", c.MapID)
		}
		// LE PIEGE DU RACK : le depot porte, pour chaque carte, `<carte>_map.mvar` (la
		// carte) et `<carte>_<canevas>.mvar` (~17 Ko, le rack d'objets du canevas — ses
		// objectifs tiennent dans <5 % de l'emprise). Cuire le rack rendrait un fond vide.
		if !strings.HasSuffix(c.FichierMvar, "_map.mvar") {
			t.Errorf("%s : FichierMvar %q ne finit pas par _map.mvar — c'est le rack du canevas, pas la carte",
				c.Nom, c.FichierMvar)
		}
		if strings.TrimSpace(c.ModuleCanevas) == "" {
			t.Errorf("%s : ModuleCanevas vide", c.Nom)
		}
		if !EstCanevasForge(c.ModuleCanevas) {
			t.Errorf("%s : EstCanevasForge(%q) devrait etre vrai", c.Nom, c.ModuleCanevas)
		}
	}
	// Un module NATIF n'est jamais un canevas Forge : la chaine native doit le cuire.
	for _, natif := range []string{"ridgeline", "catalyst", "btb_engine"} {
		if EstCanevasForge(natif) {
			t.Errorf("EstCanevasForge(%q) devrait etre faux — module natif", natif)
		}
	}
}

// TestFondForgeJamaisSousCleModule — LE GARDE-RAIL SUR L'ASSET VERSIONNE.
//
// Trois assertions, et chacune attrape une regression distincte :
//  1. aucun fond ne vit sous la cle d'un CANEVAS declare (l'ancienne cle, supprimee au
//     lot fonds par map_id — la re-voir apparaitre serait une re-collision) ;
//  2. chaque carte declaree a son fond publie sous sa cle map_id (une declaration sans
//     asset est du code mort, regle 7 projet) ;
//  3. chaque fond keye par un uuid correspond a une declaration (un asset orphelin n'a
//     aucun producteur pour le re-cuire).
func TestFondForgeJamaisSousCleModule(t *testing.T) {
	dir, err := cheminDepuisDepot("data/titles/halo_infinite/reference/map_backgrounds")
	if err != nil {
		t.Fatalf("dossier des fonds versionnes introuvable : %v", err)
	}
	declares := map[string]string{}
	for _, c := range CartesForge {
		declares[c.MapID] = c.Nom
		// L'IMAGE peut etre un `.png` ou un `.webp` : le format est une propriete de la
		// DONNEE, pas du code (plan fonds WebP, D3, etape 4). Le sidecar, lui, reste
		// toujours `.json` — seule son extension est verifiee en dur.
		for _, imgExt := range extensionsImageFond {
			p := filepath.Join(dir, c.ModuleCanevas+imgExt)
			if _, statErr := os.Stat(p); statErr == nil {
				t.Errorf("fond Forge sous cle MODULE refuse : %s existe — la cle de %s est son map_id %s",
					p, c.Nom, c.MapID)
			}
		}
		if p := filepath.Join(dir, c.ModuleCanevas+".json"); fichierExiste(p) {
			t.Errorf("fond Forge sous cle MODULE refuse : %s existe — la cle de %s est son map_id %s",
				p, c.Nom, c.MapID)
		}
		if !uneExtensionExiste(dir, c.MapID, extensionsImageFond) {
			t.Errorf("%s : aucune image de fond (.png ou .webp) pour %s — declaration sans asset publie",
				c.Nom, c.MapID)
		}
		if p := filepath.Join(dir, c.MapID+".json"); !fichierExiste(p) {
			t.Errorf("%s : fond %s.json absent — declaration sans asset publie", c.Nom, c.MapID)
		}
	}
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entrees {
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if regexpMapID.MatchString(base) {
			if _, ok := declares[base]; !ok {
				t.Errorf("fond %s keye par un map_id sans declaration CartesForge — orphelin, aucun producteur", e.Name())
			}
		}
	}
}

// extensionsImageFond : les deux formats qu'une image de fond peut prendre (D1-D6, plan
// fonds WebP) — jamais une extension supposee en dur.
var extensionsImageFond = []string{".png", ".webp"}

// fichierExiste dit si `p` existe (os.Stat sans erreur).
func fichierExiste(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// uneExtensionExiste dit si au moins un fichier `<dir>/<cle><ext>` existe pour une des
// extensions listees.
func uneExtensionExiste(dir, cle string, extensions []string) bool {
	for _, ext := range extensions {
		if fichierExiste(filepath.Join(dir, cle+ext)) {
			return true
		}
	}
	return false
}

// TestBoiteDesVolumesDeMort — L'EQUIVALENT FORGE DU MASQUE DE CALLOUTS.
//
// Une carte Forge n'a AUCUNE zone de callout : les 22 cartes qui en portent sont toutes
// natives. Les volumes de mort la bornent par l'autre bout — ils declarent ou l'on MEURT — et
// la cuisson les reconnait deja ; leur POSITION n'avait jamais servi.
//
// Le test verifie les trois proprietes dont depend le bornage : un objet sans forme ne borne
// rien, un objet qui n'est pas un volume de mort non plus, et l'emprise rendue enveloppe bien
// les volumes qui en ont une.
func TestBoiteDesVolumesDeMort(t *testing.T) {
	var typeMort int32
	for id := range TypesVolumesDeMort {
		typeMort = id
		break
	}
	demi := 5.0
	forme := &mapvar.ShapeRaw{Family: 3, S5: int64(2 * demi * 65536), S6: int64(2 * demi * 65536)}

	objets := []mapvar.Object{
		{TypeID: typeMort, Pos: mapvar.Vec3{X: 10, Y: 20}, ShapeRaw: forme},
		{TypeID: typeMort, Pos: mapvar.Vec3{X: -10, Y: -20}, ShapeRaw: forme},
		// Sans forme : ne borne rien.
		{TypeID: typeMort, Pos: mapvar.Vec3{X: 1000, Y: 1000}},
		// Pas un volume de mort : ignore, meme avec une forme.
		{TypeID: typeMort + 12345, Pos: mapvar.Vec3{X: -900, Y: -900}, ShapeRaw: forme},
	}
	boite, n := BoiteDesVolumesDeMort(objets)
	if n != 2 {
		t.Fatalf("volumes retenus = %d, attendu 2 (un sans forme et un non-mortel sont ecartes)", n)
	}
	// Demi-diagonale d'une boite 5x5 : le bornage prend le PIRE cas, jamais moins large.
	d := math.Hypot(demi, demi)
	if boite[0] > -10-d || boite[1] > -20-d || boite[2] < 10+d || boite[3] < 20+d {
		t.Fatalf("boite %v n'enveloppe pas les deux volumes (demi-diagonale %.2f)", boite, d)
	}
	if _, n := BoiteDesVolumesDeMort(nil); n != 0 {
		t.Fatal("aucun objet : aucun volume retenu")
	}
}
