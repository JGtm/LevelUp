//go:build research

package grammar

// tourelle_55_ecrivain_research_test.go — LOT 5.5, POINT 1 : L ECRIVAIN DE LA VISEE DE TOURELLE.
//
// # LA QUESTION, ET POURQUOI ELLE SE POSE CHEZ L ECRIVAIN
//
// Les tirs de TOURELLE du rejeu n ont pas de direction : `vehicleWeaponMounts.ts` classe le
// montage arriere du Warthog et le plateau du Scorpion en `tourelle` et rend `angle = null`,
// parce que « la visee du tourelleur est inconnue ». La table ECS porte deux candidats DECLARES
// et NON PORTES, sans meme une adresse de deserialiseur :
//
//	ti=40 i41  vehicle-seats-override-pitch-component   non_porte, deser_addr VIDE
//	ti=40 i42  vehicle-seats-override-yaw-component     non_porte, deser_addr VIDE
//
// Aucune conclusion (« le film ne le porte pas », « c est la visee du bipede qui fait foi ») ne
// vaut sans la lecture de l ECRIVAIN. Cet instrument la produit mecaniquement : la chaine du
// descripteur de `NOTE_3_6_METHODE_DESCRIPTEURS` (chaine -> accesseur -> slot `+0x18` ->
// ecrivain `+0x40`), CALIBREE sur les six temoins du lot 3.7 — une concordance qui rate et rien
// n est publie.
//
// # CE QUE LA PASSE COUVRE, ET POURQUOI CES SEPT CIBLES
//
// Les deux candidats du brief, plus le voisinage de tourelle du meme archetype : il coute le
// meme balayage, et il departage « override de contrainte de siege » de « rotation de tourelle ».
// `i31 vehicle-auto-turret-aiming-vector` est le NEGATIF utile — une tourelle AUTOMATIQUE (Wraith
// anti-infanterie, sentinelle) a, elle, un vecteur de visee nomme ; si les deux `seats-override`
// ressemblaient a celui-la, la question serait tranchee sur la forme.
//
// # CE QU IL NE FAIT PAS
//
// Il ne desassemble pas. La suite des `ADD [reg+0x2c], N` donne les largeurs LITTERALES, jamais
// les conditions ni les largeurs passees en argument. Elle DESIGNE la grammaire ; la lecture
// complete reste Ghidra (lecture seule) sur les bornes que cette passe publie.
//
// # REGIME
//
//	TOUR55_EXE="<chemin local de HaloInfinite.exe>" \
//	  go test -tags=research -count=1 -v -timeout 20m \
//	    -run '^TestTourelle55Ecrivain$' ./internal/games/halo_infinite/film/internal/grammar/
//
// Hors ligne, LECTURE SEULE sur l executable, AUCUN film ouvert, AUCUNE base DuckDB. Sans
// `TOUR55_EXE`, le test SKIP.

import (
	"os"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/research/reapparition"
)

// ciblesTourelle55 — les composants interroges, dans l ordre de la question.
var ciblesTourelle55 = []struct {
	Ref      string
	Nom      string
	Pourquoi string
}{
	{"ti=40 i41", "vehicle-seats-override-pitch-component",
		"CANDIDAT 1 : l elevation imposee aux sieges — rotation de tourelle, ou override de contrainte ?"},
	{"ti=40 i42", "vehicle-seats-override-yaw-component",
		"CANDIDAT 2 : l azimut impose aux sieges — c est lui qui orienterait la tourelle au sol"},
	{"ti=40 i31", "vehicle-auto-turret-aiming-vector-component",
		"TEMOIN DE FORME : une tourelle AUTOMATIQUE a un vecteur de visee nomme"},
	{"ti=40 i30", "vehicle-auto-turret-triggers-component",
		"voisinage : la detente de la tourelle automatique"},
	{"ti=40 i35", "vehicle-auto-turret-target-component",
		"voisinage : la cible de la tourelle automatique"},
	{"ti=40 i39", "vehicle-auto-turret-component",
		"voisinage : l etat de la tourelle automatique"},
	{"ti=40 i40", "vehicle-equipment-turret-parent-component",
		"voisinage : le lien tourelle -> porteur"},
	{"ti=35 i21", "unit-desired-aiming-vector-component",
		"LA CONCURRENTE : la visee du bipede occupant, deja lue par le depot (rides[].aim)"},
}

func TestTourelle55Ecrivain(t *testing.T) {
	exe := os.Getenv("TOUR55_EXE")
	if exe == "" {
		t.Skip("TOUR55_EXE absent : chemin de HaloInfinite.exe attendu")
	}
	ex, err := reapparition.Charger(exe)
	if err != nil {
		t.Fatalf("chargement de l image : %v", err)
	}
	if err := reapparition.Calibrer(ex, testWriter{t}); err != nil {
		t.Fatalf("CALIBRATION EN ECHEC — aucune adresse neuve n est publiee : %v", err)
	}
	for _, c := range ciblesTourelle55 {
		t.Logf("--- %s  %s", c.Ref, c.Nom)
		t.Logf("    POURQUOI : %s", c.Pourquoi)
		d := ex.Resoudre(c.Nom)
		if !d.Aboutie() {
			t.Logf("    ECHEC : %s", d.Echec)
			for _, m := range d.Multiples {
				t.Logf("      %s", m)
			}
			continue
		}
		t.Logf("    chaine %#x  thunk %#x  descripteur %#x  ECRIVAIN %#x  compagnon+0x28 %#x",
			d.ChaineVA, d.ThunkVA, d.DescripteurVA, d.EcrivainVA, d.Slots[0x28/8])
		if !d.BornesConnues {
			t.Logf("    bornes : ABSENTES de .pdata — grammaire non relevee")
			continue
		}
		r := ex.Relever(d.Bornes)
		t.Logf("    bornes %#x..%#x · %s", d.Bornes.Debut, d.Bornes.Fin, r.Resume())
		for _, l := range r.Largeurs {
			t.Logf("      largeur %#x : R(%d)", l.VA, l.Bits)
		}
		for _, cible := range r.CiblesUniques {
			n := 0
			for _, a := range r.Appels {
				if a.Cible == cible {
					n++
				}
			}
			t.Logf("      appel  -> %#x  (x%d)", cible, n)
		}
	}
}

// testWriter renvoie vers le journal du test ce que les rapports de `reapparition` ecrivent sur
// un `io.Writer` — la calibration s affiche donc dans la sortie `-v`, au meme endroit que le
// reste de la passe.
type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", p)
	return len(p), nil
}
