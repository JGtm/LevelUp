package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/testutil"

	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// maillageCarre : un maillage d'une face carree de 1 m.
func maillageCarre() *hinavmesh.Maillage {
	return &hinavmesh.Maillage{
		Sommets: []hinavmesh.Point{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}},
		Faces:   []hinavmesh.Face{{Sommets: []int32{0, 1, 2, 3}}},
	}
}

const avisDeTest = `{
  "schemaVersion": 1,
  "defaut": "silhouette",
  "sans_avis": ["sans-avis"],
  "sans_maillage": ["sans-maillage"],
  "exceptions": {
    "publiee": {"carte": "P", "etat": "publie", "commentaire": ""},
    "faces": {"carte": "F", "etat": "faces", "commentaire": ""},
    "aucune": {"carte": "A", "etat": "aucune", "commentaire": "foireux"},
    "vide": {"carte": "V", "etat": "", "commentaire": "ratés"}
  }
}`

func avisChargeDeTest(t *testing.T, contenu string) *avisFond {
	t.Helper()
	chemin := filepath.Join(t.TempDir(), "avis.json")
	if err := os.WriteFile(chemin, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := chargeAvis(chemin)
	if err != nil {
		t.Fatalf("chargeAvis : %v", err)
	}
	return a
}

// La règle du fichier d'avis, carte par carte : défaut silhouette ; publie et sans_maillage ne
// touchent à rien ; aucune, vide et sans_avis sont recuites en silhouette et marquées à revoir.
func TestAvisDecide(t *testing.T) {
	a := avisChargeDeTest(t, avisDeTest)
	cas := []struct {
		cle     string
		saute   bool
		mode    himap.ModeNavmesh
		aRevoir bool
	}{
		{"inconnue", false, himap.ModeSilhouette, false},
		{"publiee", true, "", false},
		{"faces", false, himap.ModeFaces, false},
		{"aucune", false, himap.ModeSilhouette, true},
		{"vide", false, himap.ModeSilhouette, true},
		{"sans-avis", false, himap.ModeSilhouette, true},
		{"sans-maillage", true, "", false},
	}
	for _, c := range cas {
		d := a.decide(c.cle)
		if d.Saute != c.saute || d.Mode != c.mode || d.ARevoir != c.aRevoir {
			t.Errorf("%s : %+v, attendu saute=%v mode=%q aRevoir=%v", c.cle, d, c.saute, c.mode, c.aRevoir)
		}
		if !c.saute && !d.Rogne {
			t.Errorf("%s : une carte recuite doit etre rognee au maillage", c.cle)
		}
	}
}

// Un etat ou un defaut que l'outil ne sait pas traduire fait echouer le chargement.
func TestChargeAvisRefuseUnEtatInconnu(t *testing.T) {
	for nom, contenu := range map[string]string{
		"etat inconnu":   `{"defaut":"silhouette","exceptions":{"x":{"etat":"flou"}}}`,
		"defaut inconnu": `{"defaut":"flou","exceptions":{}}`,
		"defaut vide":    `{"exceptions":{}}`,
	} {
		chemin := filepath.Join(t.TempDir(), "avis.json")
		if err := os.WriteFile(chemin, []byte(contenu), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := chargeAvis(chemin); err == nil {
			t.Errorf("%s : accepte", nom)
		}
	}
}

// Le fichier d'avis publie est lisible et traduit chaque carte qu'il nomme.
func TestAvisPublieEstLisible(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	chemin := filepath.Join(racine, "data/titles/halo_infinite/reference/map_fond_avis_2026-11-13.json")
	a, err := chargeAvis(chemin)
	if err != nil {
		t.Fatalf("avis publie illisible : %v", err)
	}
	if d := a.decide("b6c7bdfa-0021-4ef7-b99f-332084f1060c"); !d.Saute {
		t.Errorf("Yuletide doit rester publiee : %+v", d)
	}
	if d := a.decide("sgh_blueprint"); !d.Saute {
		t.Errorf("Recharge doit rester publiee : %+v", d)
	}
	if d := a.decide("bb7b78ae-3468-46ce-b5ba-cca61c3a338a"); d.Saute || !d.ARevoir || d.Mode != himap.ModeSilhouette {
		t.Errorf("High Ground : %+v, attendu silhouette a revoir", d)
	}
}

// Le reglage `navmeshMode` prime sur l'avis, y compris sur « publie » ; sans avis il ne decide
// que du mode.
func TestReglageNavmeshModePrimeSurLAvis(t *testing.T) {
	a := avisChargeDeTest(t, avisDeTest)
	e := &environnement{avis: a, reglages: &reglagesFond{Cartes: map[string]reglageCarte{
		"publiee": {NavmeshMode: "faces", GateLe: "2026-11-13"},
	}}}
	if d := e.decisionFond("publiee"); d.Saute || d.Mode != himap.ModeFaces || !d.Rogne {
		t.Errorf("publiee + navmeshMode faces : %+v", d)
	}
	sans := &environnement{reglages: e.reglages}
	if d := sans.decisionFond("publiee"); d.Saute || d.Rogne || d.Mode != himap.ModeFaces {
		t.Errorf("sans avis : %+v, attendu le seul mode", d)
	}
	if d := sans.decisionFond("autre"); d.Saute || d.Rogne || d.Mode != "" {
		t.Errorf("sans avis ni reglage : %+v, attendu rien de decide", d)
	}
}

// Un navmeshMode inconnu fait echouer le chargement des reglages.
func TestChargeReglagesRefuseUnNavmeshModeInconnu(t *testing.T) {
	p := filepath.Join(t.TempDir(), "r.json")
	blob := `{"schemaVersion":1,"cartes":{"x":{"navmeshMode":"flou","raison":"...","gateLe":"2026-11-13"}}}`
	if err := os.WriteFile(p, []byte(blob), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := chargeReglages(p); err == nil || !strings.Contains(err.Error(), "navmeshMode") {
		t.Fatalf("navmeshMode inconnu accepte : %v", err)
	}
}

// L'avis arme le rognage et sa forme sur les deux chaines ; sans avis, une carte Forge n'est
// pas rognee et une carte native l'est.
func TestAvisArmeLeRognageSurLesDeuxChaines(t *testing.T) {
	a := avisChargeDeTest(t, avisDeTest)
	e := &environnement{avis: a, reglages: &reglagesFond{Cartes: map[string]reglageCarte{}}}
	m := maillageCarre()
	for nom, opts := range map[string]himap.OptionsNavmesh{
		"natif": e.optionsNavmeshNatif("x", m),
		"forge": e.optionsNavmeshForge("x", m),
	} {
		if !opts.Rogne || opts.Mode != himap.ModeSilhouette || opts.Marge != 0 {
			t.Errorf("%s : %+v, attendu rognage en silhouette a la marge par defaut de la brique", nom, opts)
		}
	}
	sans := &environnement{reglages: &reglagesFond{Cartes: map[string]reglageCarte{}}}
	if sans.optionsNavmeshForge("x", m).Rogne {
		t.Error("sans avis, une carte Forge ne doit pas etre rognee")
	}
	if !sans.optionsNavmeshNatif("x", m).Rogne {
		t.Error("sans avis, une carte native reste rognee")
	}
}

// Un avis complementaire apporte ignorer (rien n'est touche) et reprendre (recuite comme les
// autres, signalee a part) ; nouveau ne change rien ; tout autre etat est refuse.
func TestAvisComplementaire(t *testing.T) {
	regle := filepath.Join(t.TempDir(), "regle.json")
	compl := filepath.Join(filepath.Dir(regle), "compl.json")
	if err := os.WriteFile(regle, []byte(avisDeTest), 0o644); err != nil {
		t.Fatal(err)
	}
	contenu := `{"defaut":"nouveau","exceptions":{"ignoree":{"etat":"ignorer"},"aucune":{"etat":"reprendre"},"x":{"etat":"nouveau"}}}`
	if err := os.WriteFile(compl, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := chargeAvis(regle, compl)
	if err != nil {
		t.Fatal(err)
	}
	if d := a.decide("ignoree"); !d.Ignore || !d.Saute || d.Rogne {
		t.Errorf("ignorer : %+v", d)
	}
	if d := a.decide("aucune"); !d.Reprise || !d.Rogne || d.Saute || d.Mode != himap.ModeSilhouette {
		t.Errorf("reprendre : %+v", d)
	}
	if d := a.decide("x"); d.Etat != etatDefaut || d.Reprise || d.Ignore {
		t.Errorf("nouveau : %+v", d)
	}
	mauvais := filepath.Join(filepath.Dir(regle), "mauvais.json")
	if err := os.WriteFile(mauvais, []byte(`{"exceptions":{"y":{"etat":"flou"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := chargeAvis(regle, mauvais); err == nil {
		t.Error("etat complementaire inconnu accepte")
	}
}

// Les deux fichiers d'avis publies se chargent ensemble : Vallaheim Firefight est ignoree, Ardent
// Prayer et Urban Raid sont a reprendre.
func TestAvisPubliesEtComplementEnsemble(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	ref := filepath.Join(racine, "data/titles/halo_infinite/reference")
	a, err := chargeAvis(filepath.Join(ref, "map_fond_avis_2026-11-13.json"), filepath.Join(ref, "map_fond_avis_lot2_2026-11-14.json"))
	if err != nil {
		t.Fatalf("avis publies illisibles : %v", err)
	}
	if d := a.decide("e8268e75-6583-42ad-9e0c-2d2f043f5f0f"); !d.Ignore {
		t.Errorf("Vallaheim Firefight doit etre ignoree : %+v", d)
	}
	for _, cle := range []string{"8cf45707-ced1-4722-b736-c139cbf0d1e2", "be848f91-3d87-4b80-8eb9-df3b52cb8d10"} {
		if d := a.decide(cle); !d.Reprise || d.Saute {
			t.Errorf("%s doit etre a reprendre : %+v", cle, d)
		}
	}
}
