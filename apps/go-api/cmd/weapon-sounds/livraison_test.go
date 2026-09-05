package main

// livraison_test.go — garde-rails du mode `livrer`.
//
// MT19937 : la sequence de reference est celle de CPython, obtenue une fois par
// `python3 -c "import random; r=random.Random(20260816); ..."` (lot v2 G.3, 2026-09-05) et
// figee ici — une regression sur le generateur romprait le rendu de Covenant_provoker sans
// qu'aucun autre test ne le voie (aucune donnee reelle du jeu n'est versionnee pour ce mode).
//
// Le reste couvre les branches PURES (aucun fichier) : la cle de tri, le choix de source par
// vote, le filtrage de variation et le formatage des nombres pour le TS — la preuve OCTET A
// OCTET contre le vrai script Python (rendu, copie de role, troncature, melange par couches)
// vit hors depot (jeu synthetique, scratchpad de la session du lot) faute d'un jeu d'entrees
// reel sur ce poste : les dossiers d'armes avec leurs .wav sources ont disparu depuis la
// livraison du 2026-08-16, seuls `_donnees/*.json` et les scripts restent.

import (
	"encoding/json"
	"testing"
)

func TestMT19937_SequenceIdentiqueACPython(t *testing.T) {
	m := newMT19937FromSeed(20260816)
	raw := make([]uint32, 8)
	for i := range raw {
		raw[i] = m.next()
	}
	voulu := []uint32{
		3729100634, 2508985057, 2014205137, 1502121425,
		4055382398, 3244912307, 3681902585, 2002287815,
	}
	for i, v := range voulu {
		if raw[i] != v {
			t.Fatalf("next() #%d = %d, veut %d (CPython random.Random(20260816).getrandbits(32))", i, raw[i], v)
		}
	}
}

func TestMT19937_ChoiceIdentiqueACPython(t *testing.T) {
	// python3 -c "import random; r=random.Random(20260816); print([r.choice(range(n)) for n in (3,5,19,2,63,1)])"
	m := newMT19937FromSeed(20260816)
	longueurs := []int{3, 5, 19, 2, 63, 1}
	voulu := []int{2, 3, 11, 1, 18, 0}
	for i, n := range longueurs {
		if got := m.choice(n); got != voulu[i] {
			t.Fatalf("choice(%d) #%d = %d, veut %d", n, i, got, voulu[i])
		}
	}
}

func TestJoliDossier(t *testing.T) {
	cas := map[string]string{
		`C:\Steam\SFX\sb_010_wea_un_assaultrifle.pck`:  "UNSC_assaultrifle",
		`C:\Steam\SFX\sb_010_tur_bt_gatlingmortar.pck`: "Banished_Tourelle_gatlingmortar",
		`C:\Steam\SFX\sb_010_whizby_pl_generic.pck`:    "Divers_Whizby_generic",
		`C:\Steam\SFX\sb_010_bank09089e7e.pck`:         "bank09089e7e", // pas de match -> prefixe retire tel quel
	}
	for pck, veut := range cas {
		if got := joliDossier(pck); got != veut {
			t.Errorf("joliDossier(%q) = %q, veut %q", pck, got, veut)
		}
	}
}

func TestLivraisonClefCoup(t *testing.T) {
	cas := []struct {
		groupe           string
		mode, persp1p0v1 int
	}{
		{"_coup_m1_1p", 1, 0},
		{"_coup_m1_3p", 1, 1},
		{"_coup_m2_3p", 2, 1},
		{"ev_deadbeef", 99, 9}, // hors patron
	}
	for _, c := range cas {
		mode, persp := livraisonClefCoup(c.groupe)
		if mode != c.mode || persp != c.persp1p0v1 {
			t.Errorf("livraisonClefCoup(%q) = (%d,%d), veut (%d,%d)", c.groupe, mode, persp, c.mode, c.persp1p0v1)
		}
	}
}

func TestLivraisonChoixDossier_VoteCoupTriePar1pAvant3p(t *testing.T) {
	votes := []livraisonVote{
		{Arme: "Arme", Groupe: "_coup_m1_3p", Vote: "garder", ExemplesRetenus: []string{"_M1_3p_1.wav"}},
		{Arme: "Arme", Groupe: "_coup_m1_1p", Vote: "garder", ExemplesRetenus: []string{"_M1_1p_1.wav"}},
	}
	choix, err := livraisonChoixDossier("Arme", "/racine", map[string]livraisonCoupsEntree{}, votes)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if choix.Source != "Arme/_M1_1p_1.wav" {
		t.Fatalf("Source = %q, veut la variante 1p (tri (mode, persp) avant (1,3p))", choix.Source)
	}
}

func TestLivraisonChoixDossier_VoteRejeteExclu(t *testing.T) {
	votes := []livraisonVote{
		{Arme: "Arme", Groupe: "_coup_m1_3p", Vote: "rejeter", ExemplesRetenus: []string{"_M1_3p_1.wav"}},
	}
	choix, err := livraisonChoixDossier("Arme", "/racine", map[string]livraisonCoupsEntree{}, votes)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if choix.Source != "" {
		t.Fatalf("Source = %q, veut vide (un vote 'rejeter' n'est jamais retenu)", choix.Source)
	}
}

func TestLivraisonChoixDossier_EvenementEnRepli(t *testing.T) {
	votes := []livraisonVote{
		{Arme: "Arme", Groupe: "ev_cafebabe", Vote: "favori", ExemplesRetenus: []string{"fichier.wav"}},
	}
	choix, err := livraisonChoixDossier("Arme", "/racine", map[string]livraisonCoupsEntree{}, votes)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if choix.Source != "Arme/fichier.wav" || choix.EvHex != "cafebabe" {
		t.Fatalf("choix = %+v, veut Source=Arme/fichier.wav EvHex=cafebabe", choix)
	}
}

func TestLivraisonChoixDossier_EvenementSourceDejaPrefixeeIgnoreLeDossier(t *testing.T) {
	votes := []livraisonVote{
		{Arme: "Arme", Groupe: "ev_cafebabe", Vote: "favori", ExemplesRetenus: []string{"Autre_EMBARQUES/f.wav"}},
	}
	choix, err := livraisonChoixDossier("Arme", "/racine", map[string]livraisonCoupsEntree{}, votes)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if choix.Source != "Autre_EMBARQUES/f.wav" {
		t.Fatalf("Source = %q, veut le chemin tel quel (contient deja un \"/\")", choix.Source)
	}
}

func TestLivraisonChoixParRole_RendreEvent(t *testing.T) {
	choix, err := livraisonChoixDossier("Covenant_provoker", "/racine", nil, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if choix.Source != "__RENDRE__bb31841b" || choix.EvHex != "bb31841b" {
		t.Fatalf("choix = %+v", choix)
	}
}

func TestLivraisonEstVarianteEtOrdre(t *testing.T) {
	manifeste := map[string]livraisonManifesteArme{
		"UNSC_base":          {Dossier: "UNSC_base", Cle: strPtr("hinf_x"), NomFr: strPtr("Arme")},
		"UNSC_base_infectee": {Dossier: "UNSC_base_infectee", Cle: strPtr("hinf_x"), NomFr: strPtr("Arme (infectee)")},
		"Covenant_provoker":  {Dossier: "Covenant_provoker", Cle: strPtr("hinf_ravager"), NomFr: strPtr("Ravageur")},
	}
	if livraisonEstVariante("UNSC_base", manifeste) {
		t.Fatal("UNSC_base ne devrait pas etre une variante")
	}
	if !livraisonEstVariante("UNSC_base_infectee", manifeste) {
		t.Fatal("UNSC_base_infectee devrait etre une variante (parenthese dans nom_fr)")
	}
	ordre := livraisonOrdre([]string{"UNSC_base_infectee", "UNSC_base", "Covenant_provoker"}, manifeste)
	voulu := []string{"Covenant_provoker", "UNSC_base", "UNSC_base_infectee"}
	if len(ordre) != len(voulu) {
		t.Fatalf("ordre = %v, veut %v", ordre, voulu)
	}
	for i := range voulu {
		if ordre[i] != voulu[i] {
			t.Fatalf("ordre = %v, veut %v (role confirme d'abord, puis base, puis variante)", ordre, voulu)
		}
	}
}

func TestLivraisonFiltreVariation(t *testing.T) {
	nulle := livraisonFiltreVariation(&livraisonVariation{
		VolumeDB: &livraisonPlage{Bas: "0", Haut: "0"},
	})
	if nulle != nil {
		t.Fatalf("une fourchette (0,0) doit etre filtree, obtenu %+v", nulle)
	}
	asym := livraisonFiltreVariation(&livraisonVariation{
		PitchCents: &livraisonPlage{Bas: "0", Haut: "800"},
	})
	if asym == nil || asym.PitchCents == nil {
		t.Fatal("une fourchette (0, 800) NE doit PAS etre filtree (haut != 0)")
	}
}

func TestLivraisonFormatNombrePy(t *testing.T) {
	cas := map[string]string{"-48": "-48", "800": "800", "0": "0", "-3.5": "-3.5"}
	for entree, veut := range cas {
		if got := livraisonFormatNombrePy(json.Number(entree)); got != veut {
			t.Errorf("livraisonFormatNombrePy(%q) = %q, veut %q", entree, got, veut)
		}
	}
}

func strPtr(s string) *string { return &s }
