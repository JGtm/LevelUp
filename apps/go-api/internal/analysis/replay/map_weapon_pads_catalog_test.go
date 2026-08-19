package replay

// Tests — LE CATALOGUE DES SOCLES : son lecteur, et le fichier RÉELLEMENT versionné.
//
// CE QUE CE FICHIER VERROUILLE :
//   - un catalogue d'une AUTRE version de schéma est refusé, jamais lu de travers ;
//   - une carte absente rend ErrUnknownMap, le cas nominal que l'appelant dégrade ;
//   - le fichier livré est exploitable : chaque emplacement porte une famille CONNUE et un
//     type_id brut en hexadécimal, et la famille est bien celle que ce type_id implique —
//     la publier à côté du brut ne sert à rien si les deux peuvent diverger ;
//   - les TROIS CARTES MESURÉES au plan portent le compte d'emplacements mesuré. C'est le
//     garde-fou de la régénération : le jour où un dump change, le chiffre bouge ici.

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ecrireCatalogueSocles pose un catalogue minimal sur disque et rend son chemin.
func ecrireCatalogueSocles(t *testing.T, corps string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "map_weapon_pads.json")
	if err := os.WriteFile(p, []byte(corps), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadMapWeaponPads_Version(t *testing.T) {
	vieux := ecrireCatalogueSocles(t,
		`{"schema_version":0,"title_slug":"halo_infinite","maps":{}}`)
	if _, err := LoadMapWeaponPads(vieux); err == nil {
		t.Fatal("un catalogue d'une autre version doit être refusé")
	}
	bon := ecrireCatalogueSocles(t,
		`{"schema_version":1,"title_slug":"halo_infinite","maps":{"m1":{"map_id":"m1",`+
			`"mvar_file":"m1.mvar","level_id":42,"objects_n":3,"pads":[`+
			`{"pos":{"x":1,"y":2,"z":3},"type_id":"0x5F379533","family":"power","objects":1}]}}}`)
	cat, err := LoadMapWeaponPads(bon)
	if err != nil {
		t.Fatalf("LoadMapWeaponPads: %v", err)
	}
	e, err := cat.Lookup("m1")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if len(e.Pads) != 1 || e.Pads[0].Family != "power" || e.Pads[0].Pos.X != 1 {
		t.Fatalf("entrée lue de travers : %+v", e)
	}
}

func TestLoadMapWeaponPads_CarteInconnue(t *testing.T) {
	cat := &MapWeaponPadsCatalog{Maps: map[string]MapWeaponPadsEntry{}}
	if _, err := cat.Lookup("absente"); !errors.Is(err, ErrUnknownMap) {
		t.Fatalf("err = %v, attendu ErrUnknownMap", err)
	}
	var nul *MapWeaponPadsCatalog
	if _, err := nul.Lookup("absente"); !errors.Is(err, ErrUnknownMap) {
		t.Fatalf("catalogue nil : err = %v, attendu ErrUnknownMap", err)
	}
}

func TestLoadMapWeaponPads_Illisible(t *testing.T) {
	if _, err := LoadMapWeaponPads(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("un fichier absent doit rendre une erreur")
	}
	if _, err := LoadMapWeaponPads(ecrireCatalogueSocles(t, "{pas du json")); err == nil {
		t.Fatal("un fichier illisible doit rendre une erreur")
	}
}

// famillesAttendues — la correspondance type_id -> famille, ÉCRITE ICI aussi. Le
// producteur la dérive de mapvar ; ce test la rejoue de mémoire, exprès : si les deux
// divergent, c'est que la table a bougé sans que le catalogue soit régénéré.
var famillesAttendues = map[string]string{
	"0x5F379533": "power",
	"0x6253CFC0": "rack",
	"0x5E86D110": "powerup",
}

// cartesMesurees — le compte d'emplacements des trois cartes du plan
// `.ai/V7.5/replay2d/PLAN_SOCLES_MVAR.md`, par map_id.
//
// CATALYST 11 est la mesure du plan au socle près (13 objets, 11 emplacements — deux
// déclarations doubles à 4,7 cm et 9 mm). CLIFFHANGER 18 = les 17 socles d'ARME du plan
// PLUS le socle de POWER-UP : l'instrument de mesure n'énumérait que les type_id que
// l'oracle de CETTE carte validait, et l'artefact de Cliffhanger, cuit avant la voie
// `ti=37`, ne porte aucun power-up à valider. SMALLHALLA 27 pour la même raison, sur une
// carte Forge où 39 objets se regroupent en 27 emplacements.
var cartesMesurees = map[string]struct {
	nom string
	n   int
}{
	"e859cf75-9b8a-429a-91be-2376681c8537": {"Catalyst", 11},
	"f7e8cde9-0c0a-487c-94a3-61bfa0f20465": {"Catalyst (2e map_id)", 11},
	"5324364b-39a8-4f93-96a6-b80a1f18ce8a": {"Cliffhanger", 18},
	"98783453-ce40-4020-9e87-62099a290b62": {"Smallhalla (Forge, le RACK)", 27},
}

// cheminCatalogueLivre localise le fichier versionné, ou rend "" (arbre partiel).
func cheminCatalogueLivre(nom string) string {
	for _, up := range []string{"../../../..", "../../../../.."} {
		p := filepath.Join(up, "data", "titles", "halo_infinite", "reference", nom)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// TestCatalogueSoclesLivre lit le catalogue RÉELLEMENT versionné — celui que le
// croisement lira au rejeu.
func TestCatalogueSoclesLivre(t *testing.T) {
	path := cheminCatalogueLivre("map_weapon_pads.json")
	if path == "" {
		t.Skip("catalogue des socles absent de cet arbre")
	}
	cat, err := LoadMapWeaponPads(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Maps) == 0 {
		t.Fatal("catalogue vide")
	}
	total := 0
	for id, e := range cat.Maps {
		if e.MapID != id {
			t.Errorf("%s : map_id interne %q — la clé et le champ doivent coïncider", id, e.MapID)
		}
		if e.MvarFile == "" {
			t.Errorf("%s : aucun mvar_file — la trace de production manque", id)
		}
		total += len(e.Pads)
		verifierPads(t, id, e)
	}
	t.Logf("catalogue livré : %d cartes, %d emplacements", len(cat.Maps), total)
	for id, attendu := range cartesMesurees {
		e, err := cat.Lookup(id)
		if err != nil {
			t.Errorf("%s (%s) absente du catalogue : %v", attendu.nom, id, err)
			continue
		}
		if len(e.Pads) != attendu.n {
			t.Errorf("%s : %d emplacements, attendu %d (mesure du plan)",
				attendu.nom, len(e.Pads), attendu.n)
		}
	}
}

// verifierPads contrôle chaque emplacement d'une carte : famille connue, type_id brut
// cohérent avec elle, position finie.
func verifierPads(t *testing.T, id string, e MapWeaponPadsEntry) {
	t.Helper()
	for i, p := range e.Pads {
		fam, connu := famillesAttendues[p.TypeID]
		if !connu {
			t.Errorf("%s pad %d : type_id %q hors des trois types mesurés", id, i, p.TypeID)
			continue
		}
		if p.Family != fam {
			t.Errorf("%s pad %d : famille %q pour le type %s, attendu %q — le brut et "+
				"l'inférence ont divergé", id, i, p.Family, p.TypeID, fam)
		}
		if !strings.HasPrefix(p.TypeID, "0x") {
			t.Errorf("%s pad %d : type_id %q n'est pas en hexadécimal", id, i, p.TypeID)
		}
		if _, err := strconv.ParseUint(strings.TrimPrefix(p.TypeID, "0x"), 16, 32); err != nil {
			t.Errorf("%s pad %d : type_id %q illisible : %v", id, i, p.TypeID, err)
		}
		if p.Objects < 1 {
			t.Errorf("%s pad %d : %d objets fusionnés — un emplacement en porte au moins un",
				id, i, p.Objects)
		}
		for _, v := range []float64{p.Pos.X, p.Pos.Y, p.Pos.Z} {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Errorf("%s pad %d : position non finie %+v", id, i, p.Pos)
				break
			}
		}
	}
}

// TestCatalogueSoclesLivreEstDeterministe relit le fichier livré et vérifie que ses
// emplacements sont dans l'ORDRE SPATIAL du producteur.
//
// POURQUOI C'EST UN TEST ET NON UN DÉTAIL : un catalogue versionné dont l'ordre dépend du
// fichier source produit un diff git illisible à chaque régénération, et personne ne relit
// alors ce qui a réellement changé.
func TestCatalogueSoclesLivreEstDeterministe(t *testing.T) {
	path := cheminCatalogueLivre("map_weapon_pads.json")
	if path == "" {
		t.Skip("catalogue des socles absent de cet arbre")
	}
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cat MapWeaponPadsCatalog
	if err := json.Unmarshal(blob, &cat); err != nil {
		t.Fatal(err)
	}
	for id, e := range cat.Maps {
		for i := 1; i < len(e.Pads); i++ {
			a, b := e.Pads[i-1].Pos, e.Pads[i].Pos
			if a.X > b.X || (a.X == b.X && a.Y > b.Y) {
				t.Errorf("%s : emplacements %d et %d hors ordre spatial (%v puis %v)",
					id, i-1, i, a, b)
				break
			}
		}
	}
}
