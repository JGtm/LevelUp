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

// TestJoliDossier_IndependantDuSeparateur — regression CI du 2026-09-06 : joliDossier
// s'appuyait sur filepath.Base/Ext, dependants de l'OS DE COMPILATION (path/filepath de la
// stdlib Go ne coupe que sur "/" sous Linux). Les chemins de lot1.json/lot2.json sont
// TOUJOURS des chemins Windows (machine d'extraction du chantier) : la fonction doit
// reconnaitre "\" ET "/" quelle que soit la plateforme qui l'execute, comme le faisait
// os.path.basename de Python SOUS WINDOWS (le seul OS ou _outils/livraison.py a tourne).
func TestJoliDossier_IndependantDuSeparateur(t *testing.T) {
	cas := map[string]string{
		`C:\Steam\SFX\sb_010_wea_un_assaultrifle.pck`:  "UNSC_assaultrifle",
		"C:/Steam/SFX/sb_010_wea_un_assaultrifle.pck":  "UNSC_assaultrifle",
		`C:\Steam/SFX\sb_010_tur_bt_gatlingmortar.pck`: "Banished_Tourelle_gatlingmortar", // separateurs mixtes
		"sb_010_whizby_pl_generic.pck":                 "Divers_Whizby_generic",           // nom nu, sans repertoire
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

// TestLivraisonFormatNombrePy — constat C9 de la revue R1.
//
// La table entiere est la SORTIE REELLE de `"%s" %% json.loads(litteral)` sous CPython 3.12,
// relevee une fois par une sonde hors depot (480 litteraux au total, dont 420 tires au sort ;
// les 56 ci-dessous sont ceux qui portent une regle). L'ancienne version, qui rendait
// `strconv.FormatFloat(v, 'g', -1, 64)`, divergeait sur 144 d'entre eux ; son test ne probait
// que `-48`, `800`, `0` et `-3.5` — les quatre cas qui coincidaient.
//
// Les regles que la table fixe : un ENTIER JSON est un int Python (`-0` devient `0`, un
// entier hors int64 garde ses chiffres) ; un FLOTTANT sort en `repr`, toujours pointe
// (`2.0`, `1e3` -> `1000.0`), en exposant seulement hors de [1e-4, 1e16), avec un exposant
// signe sur deux chiffres au moins (`1e-05`, `1e+16`) — et le zero NEGATIF flottant garde son
// signe (`-0.0`) la ou l'entier `-0` le perd.
func TestLivraisonFormatNombrePy(t *testing.T) {
	cas := []struct{ litteral, veut string }{
		{"0", "0"},
		{"-0", "0"},
		{"1", "1"},
		{"-1", "-1"},
		{"48", "48"},
		{"-48", "-48"},
		{"800", "800"},
		{"1000000", "1000000"},
		{"-1000000", "-1000000"},
		{"9007199254740993", "9007199254740993"},
		{"123456789012345678901234567890", "123456789012345678901234567890"},
		{"-123456789012345678901234567890", "-123456789012345678901234567890"},
		{"0.0", "0.0"},
		{"-0.0", "-0.0"},
		{"1.0", "1.0"},
		{"-1.0", "-1.0"},
		{"2.0", "2.0"},
		{"-3.0", "-3.0"},
		{"-3.5", "-3.5"},
		{"0.5", "0.5"},
		{"-0.25", "-0.25"},
		{"4.25", "4.25"},
		{"-4.25", "-4.25"},
		{"1.5", "1.5"},
		{"1234567.5", "1234567.5"},
		{"-1234567.5", "-1234567.5"},
		{"85.0", "85.0"},
		{"-85.0", "-85.0"},
		{"1e3", "1000.0"},
		{"1e-3", "0.001"},
		{"1e-4", "0.0001"},
		{"1e-5", "1e-05"},
		{"1e15", "1000000000000000.0"},
		{"1e16", "1e+16"},
		{"1e17", "1e+17"},
		{"1e100", "1e+100"},
		{"1e-100", "1e-100"},
		{"-1e3", "-1000.0"},
		{"-1e-5", "-1e-05"},
		{"-1e16", "-1e+16"},
		{"9999999999999998.0", "9999999999999998.0"},
		{"10000000000000002.0", "1.0000000000000002e+16"},
		{"0.0001", "0.0001"},
		{"0.00001", "1e-05"},
		{"0.000123456", "0.000123456"},
		{"1e-323", "1e-323"},
		{"5e-324", "5e-324"},
		{"1.7976931348623157e308", "1.7976931348623157e+308"},
		{"0.1", "0.1"},
		{"0.2", "0.2"},
		{"0.3", "0.3"},
		{"0.30000000000000004", "0.30000000000000004"},
		{"2.675", "2.675"},
		{"1.0000000000000002", "1.0000000000000002"},
		{"123.456", "123.456"},
		{"-123.456", "-123.456"},
	}
	for _, c := range cas {
		if got := livraisonFormatNombrePy(json.Number(c.litteral)); got != c.veut {
			t.Errorf("livraisonFormatNombrePy(%q) = %q, veut %q (sortie de CPython 3.12)",
				c.litteral, got, c.veut)
		}
	}
}

// TestLivraisonChoixDossier_ExempleVideSauteLeVote — constat C7 de la revue R1.
//
// Python rend `exemples_retenus[0]` des que la liste est non vide, MEME si cet element vaut
// `""`, et ce sont ses appelants qui l'ecartent par leur `if f:` : le vote est saute et l'on
// passe au SUIVANT — jamais un repli sur `exemples_proposes` DU MEME VOTE. Rendre ("", true)
// construisait la source "<dossier>/", que os.Stat accepte (c'est un repertoire) et que la
// troncature refuse : le run entier avortait.
func TestLivraisonChoixDossier_ExempleVideSauteLeVote(t *testing.T) {
	votes := []livraisonVote{
		{Arme: "Arme", Groupe: "_coup_m0_1p", Vote: "garder",
			ExemplesRetenus: []string{""}, ExemplesProposes: []string{"jamais_lu.wav"}},
		{Arme: "Arme", Groupe: "_coup_m1_1p", Vote: "garder", ExemplesRetenus: []string{"suivant.wav"}},
	}
	choix, err := livraisonChoixDossier("Arme", "/racine", map[string]livraisonCoupsEntree{}, votes)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if choix.Source != "Arme/suivant.wav" {
		t.Fatalf("Source = %q, veut %q (l'exemple vide saute le vote, sans repli sur "+
			"exemples_proposes du meme vote)", choix.Source, "Arme/suivant.wav")
	}
}

// Meme regle sur le repli par evenement : `if f and not f.startswith("_")`.
func TestLivraisonChoixDossier_ExempleVideSauteLeVoteDEvenement(t *testing.T) {
	votes := []livraisonVote{
		{Arme: "Arme", Groupe: "ev_cafebabe", Vote: "garder", ExemplesRetenus: []string{""}},
		{Arme: "Arme", Groupe: "ev_00c0ffee", Vote: "garder", ExemplesRetenus: []string{"suivant.wav"}},
	}
	choix, err := livraisonChoixDossier("Arme", "/racine", map[string]livraisonCoupsEntree{}, votes)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if choix.Source != "Arme/suivant.wav" || choix.EvHex != "00c0ffee" {
		t.Fatalf("choix = %+v, veut la source du second vote (00c0ffee)", choix)
	}
}

func strPtr(s string) *string { return &s }
