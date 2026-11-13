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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/testutil"

	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// cheminReglagesPublies, relatif a la RACINE du depot — la racine se demande a
// testutil.RepoRoot(), jamais par une echelle de « .. » : une echelle se casse en silence des
// que le test change de dossier, et le garde-rail archlint l interdit.
const cheminReglagesPublies = "data/titles/halo_infinite/reference/map_fond_reglages.json"

// raisonMinimale : une raison d'une ligne n'explique rien. Le seuil est bas exprès — il
// attrape le « ok user » et laisse passer une justification honnête.
const raisonMinimale = 80

var formatDateGate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func TestReglagesFondJustifies(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	chemin := filepath.Join(racine, filepath.FromSlash(cheminReglagesPublies))
	if _, err := os.Stat(chemin); err != nil {
		t.Skipf("aucun reglage publie (%v) — c est le cas nominal tant que rien n a ete juge", err)
	}
	reg, err := chargeReglages(chemin)
	if err != nil {
		t.Fatalf("réglages publiés illisibles : %v", err)
	}
	if len(reg.Cartes) == 0 {
		t.Fatal("fichier de réglages présent mais vide : le supprimer plutôt que le garder")
	}
	for cle, c := range reg.Cartes {
		for _, probleme := range problemesReglage(cle, c) {
			t.Error(probleme)
		}
	}
}

// problemesReglage rend ce qui interdit de garder une entree de reglage : raison trop courte,
// date de gate mal formee, entree sans levier, habillage inconnu.
func problemesReglage(cle string, c reglageCarte) []string {
	var out []string
	if len(c.Raison) < raisonMinimale {
		out = append(out, fmt.Sprintf("réglage %q : raison de %d caractères (minimum %d).\n"+
			"Un réglage par carte sans justification écrite est un réglage orphelin.",
			cle, len(c.Raison), raisonMinimale))
	}
	if !formatDateGate.MatchString(c.GateLe) {
		out = append(out, fmt.Sprintf("réglage %q : gateLe = %q, attendu AAAA-MM-JJ — la date du gate qui l'a décidé",
			cle, c.GateLe))
	}
	// Le contrôle vit dans `reglageCarte.sansLevier` : la liste de champs qui était
	// énumérée ici avait cessé de suivre la structure et refusait une entrée qui ne
	// déclare que `moduleGeometrie`.
	if c.sansLevier() {
		out = append(out, fmt.Sprintf("réglage %q : ne déclare aucun levier — entrée sans effet, à retirer", cle))
	}
	if c.Style != "" && !himap.StyleFondValide(himap.StyleFond(c.Style)) {
		out = append(out, fmt.Sprintf("réglage %q : habillage inconnu %q", cle, c.Style))
	}
	return out
}

// `sansNavmesh` retire le maillage d'une carte native : comme tout levier, il exige sa raison
// et la date de son gate. Une entree qui ne porte que ce levier, sans raison, est refusee.
func TestReglageSansNavmeshExigeRaisonEtDate(t *testing.T) {
	nue := reglageCarte{SansNavmesh: true}
	if len(problemesReglage("ctf_x", nue)) < 2 {
		t.Errorf("une entree sansNavmesh sans raison ni date doit etre refusee : %v", problemesReglage("ctf_x", nue))
	}
	if nue.sansLevier() {
		t.Error("sansNavmesh est un levier : l'entree ne doit pas etre vue comme sans effet")
	}
	raison := strings.Repeat("Le maillage de cette carte rogne un pont jouable ; mesure et gate decrits ici. ", 2)
	ok := reglageCarte{SansNavmesh: true, Raison: raison, GateLe: "2026-11-13"}
	if p := problemesReglage("ctf_x", ok); len(p) != 0 {
		t.Errorf("entree complete refusee : %v", p)
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
		case reflect.Pointer:
			v.Set(reflect.New(champ.Type.Elem()))
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

// `rogneAuNavmesh` : defaut vrai pour une carte native, faux pour une carte Forge ; `false`
// explicite desarme le rognage natif, `true` explicite arme le Forge.
func TestRogneAuNavmeshDefautsNatifEtForge(t *testing.T) {
	vrai, faux := true, false
	e := &environnement{reglages: &reglagesFond{Cartes: map[string]reglageCarte{
		"vrai": {RogneAuNavmesh: &vrai, GateLe: "2026-11-13"},
		"faux": {RogneAuNavmesh: &faux, GateLe: "2026-11-13"},
		"nul":  {GateLe: "2026-11-13"},
	}}}
	cas := []struct {
		cle          string
		natif, forge bool
	}{
		{"vrai", true, true},
		{"faux", false, false},
		{"nul", true, false},
		{"absente", true, false},
	}
	for _, c := range cas {
		if got := e.rogneAuNavmeshNatifDe(c.cle); got != c.natif {
			t.Errorf("%s : natif = %v, attendu %v", c.cle, got, c.natif)
		}
		if got := e.rogneAuNavmeshDe(c.cle); got != c.forge {
			t.Errorf("%s : Forge = %v, attendu %v", c.cle, got, c.forge)
		}
	}
}

// `sansNavmesh` cuit sans le maillage mais publie quand meme le plan des murs : les options de
// cuisson sont nulles, le plan se tire du maillage charge.
func TestSansNavmeshGardeLePlanDesMurs(t *testing.T) {
	maillage := &hinavmesh.Maillage{
		Sommets: []hinavmesh.Point{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}},
		Faces:   []hinavmesh.Face{{Sommets: []int32{0, 1, 2, 3}}},
	}
	e := &environnement{reglages: &reglagesFond{Cartes: map[string]reglageCarte{
		"sans": {SansNavmesh: true, GateLe: "2026-11-13"},
	}}}

	if opts := e.optionsNavmeshNatif("sans", maillage); opts != (himap.OptionsNavmesh{}) {
		t.Errorf("sansNavmesh : options navmesh = %+v, attendu nulles", opts)
	}
	if opts := e.optionsNavmeshNatif("autre", maillage); opts.Maillage != maillage || !opts.Rogne {
		t.Errorf("une carte sans reglage doit etre rognee a son maillage : %+v", opts)
	}
	plan := planDeMaillage(context.Background(), "sans", sourcePlanNatif, maillage)
	if plan == nil || len(plan.Edges) != 4 {
		t.Errorf("plan = %+v, attendu les 4 aretes du carre", plan)
	}
}
