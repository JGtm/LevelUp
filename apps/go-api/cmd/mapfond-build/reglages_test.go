package main

// GARDE-RAIL DES RÉGLAGES PAR CARTE.
//
// Un réglage par carte n'est défendable que parce qu'il porte sa RAISON et la DATE de son
// gate. Sans ces deux champs, dans six mois, personne ne sait si le réglage tient encore, ni
// sur quelles images il a été décidé — et on retombe exactement sur ce que la doctrine
// « aucun réglage par carte » cherchait à éviter, avec la dette en plus.
//
// Ce test lit l'asset PUBLIÉ, celui que la cuisson consomme.

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"levelup/go-api/internal/himap"
)

const cheminReglagesPublies = "../../../../data/titles/halo_infinite/reference/map_fond_reglages.json"

// raisonMinimale : une raison d'une ligne n'explique rien. Le seuil est bas exprès — il
// attrape le « ok user » et laisse passer une justification honnête.
const raisonMinimale = 80

var formatDateGate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func TestReglagesFondJustifies(t *testing.T) {
	chemin := filepath.FromSlash(cheminReglagesPublies)
	if _, err := os.Stat(chemin); err != nil {
		t.Skipf("aucun réglage publié (%v) — c'est le cas nominal tant que rien n'a été jugé", err)
	}
	reg, err := chargeReglages(chemin)
	if err != nil {
		t.Fatalf("réglages publiés illisibles : %v", err)
	}
	if len(reg.Cartes) == 0 {
		t.Fatal("fichier de réglages présent mais vide : le supprimer plutôt que le garder")
	}
	for cle, c := range reg.Cartes {
		if len(c.Raison) < raisonMinimale {
			t.Errorf("réglage %q : raison de %d caractères (minimum %d).\n"+
				"Un réglage par carte sans justification écrite est un réglage orphelin.",
				cle, len(c.Raison), raisonMinimale)
		}
		if !formatDateGate.MatchString(c.GateLe) {
			t.Errorf("réglage %q : gateLe = %q, attendu AAAA-MM-JJ — la date du gate qui l'a décidé",
				cle, c.GateLe)
		}
		// Le contrôle vit dans `reglageCarte.sansLevier` : la liste de champs qui était
		// énumérée ici avait cessé de suivre la structure et refusait une entrée qui ne
		// déclare que `moduleGeometrie`.
		if c.sansLevier() {
			t.Errorf("réglage %q : ne déclare aucun levier — entrée sans effet, à retirer", cle)
		}
		if c.Style != "" && !himap.StyleFondValide(himap.StyleFond(c.Style)) {
			t.Errorf("réglage %q : habillage inconnu %q", cle, c.Style)
		}
	}
}

// Un habillage inconnu doit FAIRE ÉCHOUER le chargement, pas être ignoré : une carte cuite
// dans l'habillage par défaut alors que la donnée en demandait un autre passerait le gate
// sous une fausse identité.
func TestChargeReglagesRefuseUnHabillageInconnu(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "r.json")
	blob := `{"schemaVersion":1,"cartes":{"x":{"style":"aquarelle","raison":"...","gateLe":"2026-08-26"}}}`
	if err := os.WriteFile(p, []byte(blob), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := chargeReglages(p); err == nil {
		t.Fatal("un habillage inconnu a été accepté")
	}
}

// Fichier absent = cas nominal, pas une erreur.
func TestChargeReglagesAbsentEstNominal(t *testing.T) {
	r, err := chargeReglages(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("un fichier absent doit être nominal, or : %v", err)
	}
	if len(r.Cartes) != 0 {
		t.Fatalf("réglages non vides sur fichier absent : %v", r.Cartes)
	}
}

// TestSansLevierReconnaitChaqueLevier : le garde-rail « entrée sans effet » ne doit refuser
// QUE les entrées vides. Une liste de champs écrite à la main s'est déjà désynchronisée de la
// structure — ce témoin balaie tous les champs par réflexion, donc il attrapera aussi le
// prochain levier ajouté.
func TestSansLevierReconnaitChaqueLevier(t *testing.T) {
	if !(reglageCarte{Carte: "X", Raison: "...", GateLe: "2026-08-27"}).sansLevier() {
		t.Fatal("une entrée qui ne porte que son identité doit être vue SANS levier")
	}
	typ := reflect.TypeOf(reglageCarte{})
	identite := map[string]bool{"Carte": true, "Raison": true, "GateLe": true}
	leviers := 0
	for i := 0; i < typ.NumField(); i++ {
		champ := typ.Field(i)
		if identite[champ.Name] {
			continue
		}
		leviers++
		c := reglageCarte{Carte: "X", Raison: "...", GateLe: "2026-08-27"}
		v := reflect.ValueOf(&c).Elem().Field(i)
		switch champ.Type.Kind() {
		case reflect.String:
			v.SetString("valeur")
		case reflect.Bool:
			v.SetBool(true)
		case reflect.Float64:
			v.SetFloat(1)
		case reflect.Slice:
			v.Set(reflect.MakeSlice(champ.Type, 1, 1))
		default:
			t.Fatalf("champ %s de type %s non couvert par ce témoin", champ.Name, champ.Type)
		}
		if c.sansLevier() {
			t.Errorf("le champ %s est un levier, il n'est pas reconnu comme tel", champ.Name)
		}
	}
	if leviers < 10 {
		t.Fatalf("%d leviers balayés : le témoin ne lit pas la structure", leviers)
	}
}

// TestModuleGeometrieExisteDansLInstallation : un `moduleGeometrie` mal orthographie ne se
// verrait qu a la cuisson, sous la forme d un « module introuvable » au milieu de 160 cartes.
// Le chemin est une DONNEE, il se verifie comme une donnee.
//
// Le test verifie AUSSI que la cle de l entree est bien un dossier installe : la cle de
// publication d une carte native est le nom du dossier du module (`filepath.Base` de son
// repertoire), et une cle qui ne correspond a rien serait un reglage silencieusement ignore.
func TestModuleGeometrieExisteDansLInstallation(t *testing.T) {
	racine, err := himap.DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	reg, err := chargeReglages(filepath.FromSlash(cheminReglagesPublies))
	if err != nil {
		t.Fatal(err)
	}
	vus := 0
	for cle, c := range reg.Cartes {
		if c.ModuleGeometrie == "" {
			continue
		}
		vus++
		chemin := filepath.Join(racine, filepath.FromSlash(c.ModuleGeometrie))
		if _, err := os.Stat(chemin); err != nil {
			t.Errorf("reglage %q : moduleGeometrie %q introuvable (%v)", cle, c.ModuleGeometrie, err)
		}
		propre, ok := himap.ChercheModuleInstalle(cle)
		if !ok {
			t.Errorf("reglage %q : aucun module installe ne porte cette cle — le reglage serait ignore", cle)
			continue
		}
		if filepath.Base(filepath.Dir(propre)) != cle {
			t.Errorf("reglage %q : la cle de publication serait %q", cle, filepath.Base(filepath.Dir(propre)))
		}
	}
	t.Logf("%d reglage(s) declarent un moduleGeometrie", vus)
}
