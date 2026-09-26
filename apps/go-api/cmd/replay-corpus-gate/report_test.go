package main

import (
	"strings"
	"testing"

	"levelup/go-api/internal/replaydiff"
)

// TestCodeSortieZeroSansPerte — aucun temoin en perte ni en erreur : code 0.
func TestCodeSortieZeroSansPerte(t *testing.T) {
	lignes := []ligneRapport{
		{Temoin: Temoin{ID: "a"}, Gains: 3, Pertes: 0},
		{Temoin: Temoin{ID: "b"}, Gains: 0, Pertes: 0},
	}
	if got := codeSortie(lignes, true); got != codeOK {
		t.Fatalf("code = %d, attendu codeOK (0)", got)
	}
}

// TestCodeSortieUnDesQuUnAxePerteSuffit — LE COMPORTEMENT DEMANDE : un seul temoin en perte
// suffit a faire echouer tout le gate, meme si d'autres temoins sont propres.
func TestCodeSortieUnDesQuUnAxePerteSuffit(t *testing.T) {
	lignes := []ligneRapport{
		{Temoin: Temoin{ID: "a"}, Pertes: 0},
		{Temoin: Temoin{ID: "b"}, Pertes: 1},
		{Temoin: Temoin{ID: "c"}, Pertes: 0},
	}
	if got := codeSortie(lignes, true); got != codePerte {
		t.Fatalf("code = %d, attendu codePerte (1) (b porte une perte)", got)
	}
}

// TestCodeSortieErreurDeCuissonEstUnEchec — un temoin qui n'a pas pu etre cuit ou compare doit
// faire echouer le gate : un rapport incomplet n'est pas un rapport vert.
func TestCodeSortieErreurDeCuissonEstUnEchec(t *testing.T) {
	lignes := []ligneRapport{{Temoin: Temoin{ID: "a"}, Erreur: errTest("cuisson cassee")}}
	if got := codeSortie(lignes, true); got != codeErreurCuisson {
		t.Fatalf("code = %d, attendu codeErreurCuisson (3) — une erreur de cuisson n est PAS une perte", got)
	}
}

// TestCodeSortieAbsentNEstPasUnEchec — LE COMPORTEMENT DEMANDE explicitement : un temoin
// absent du parc local est un avertissement (deja emis en slog par traiterTemoin), jamais un
// echec — sinon le gate echouerait toujours sur un poste sans parc local.
func TestCodeSortieAbsentNEstPasUnEchec(t *testing.T) {
	lignes := []ligneRapport{
		{Temoin: Temoin{ID: "a"}, Absent: true, AbsentCause: "aucun chunk"},
	}
	if got := codeSortie(lignes, true); got != codeOK {
		t.Fatalf("code = %d, attendu codeOK (0) (absent != echec ICI, cf. verifierCouverture)", got)
	}
}

// TestBilanDepuisRapportSommeLesAxes — gains et pertes s'additionnent sur TOUS les axes du
// rapport, pas seulement le premier.
//
// IL PART DES ECARTS ET NON DES `Bilans` DEPUIS LE LOT 3.3.3, et ce n'est pas un affaiblissement :
// `remplirBilan` reclasse desormais chaque ecart (telemetrie, rejet rapporte a son denominateur —
// `verdict_metriques.go`), donc les bilans par axe ne sont plus la source du verdict. Les deux
// populations sont les memes (`replaydiff.Rapport.ajouter` alimente bilans ET ecarts d'un seul
// geste) ; ce que ce test garde est inchange : la somme porte sur TOUS les axes.
func TestBilanDepuisRapportSommeLesAxes(t *testing.T) {
	ecart := func(axe, metrique, sens string) replaydiff.Difference {
		return replaydiff.Difference{Axe: axe, Metrique: metrique, Sens: sens,
			Ancien: "1", Nouveau: "2"}
	}
	rap := replaydiff.Rapport{
		SchemaAncien: 20, SchemaNouveau: 41,
		Differences: []replaydiff.Difference{
			ecart("objectifs", "objectives.a", replaydiff.SensGain),
			ecart("objectifs", "objectives.b", replaydiff.SensGain),
			ecart("objectifs", "objectives.c", replaydiff.SensGain),
			ecart("objectifs", "objectives.d", replaydiff.SensPerte),
			ecart("equipement", "equipment.e", replaydiff.SensPerte),
			ecart("equipement", "equipment.f", replaydiff.SensPerte),
		},
	}
	var l ligneRapport
	l.remplirBilan(rap)
	if l.SchemaReference != 20 || l.SchemaHEAD != 41 {
		t.Fatalf("schemas = %d -> %d, attendu 20 -> 41", l.SchemaReference, l.SchemaHEAD)
	}
	if l.Gains != 3 || l.Pertes != 3 {
		t.Fatalf("gains=%d pertes=%d, attendu gains=3 pertes=3 (somme des deux axes)", l.Gains, l.Pertes)
	}
}

// TestImprimerTableauNommeLeStatut — le tableau doit distinguer absent / erreur / perte /
// changement / ok en toutes lettres, pour un operateur qui ne lit que la derniere colonne.
func TestImprimerTableauNommeLeStatut(t *testing.T) {
	var b strings.Builder
	imprimerTableau(&b, []ligneRapport{
		{Temoin: Temoin{ID: "aaaa1111", Famille: "ctf"}, Gains: 2, Pertes: 0},
		{Temoin: Temoin{ID: "bbbb2222", Famille: "oddball"}, Pertes: 1},
		{Temoin: Temoin{ID: "cccc3333", Famille: "slayer"}, Absent: true, AbsentCause: "aucun chunk"},
		{Temoin: Temoin{ID: "dddd4444", Famille: "assaut"}, Erreur: errTest("carte hors catalogue")},
		{Temoin: Temoin{ID: "eeee5555", Famille: "vehicules"}, Gains: 4, Changements: 2},
	}, "base(origin/feat/v75)")
	out := b.String()
	for _, attendu := range []string{
		"aaaa1111", statutOK,
		"bbbb2222", statutPerte,
		"cccc3333", statutAbsent, "aucun chunk",
		"dddd4444", statutErreur, "carte hors catalogue",
		"eeee5555", statutChangement,
	} {
		if !strings.Contains(out, attendu) {
			t.Errorf("le tableau doit contenir %q :\n%s", attendu, out)
		}
	}
}

// TestLigneVersJSONPorteStatutEtCause — D5 (1.9.9) : un lecteur qui n'a QUE le JSON lisait
// `{"gains":0,"pertes":0,"changements":0}` pour un temoin en ERREUR comme pour un temoin
// propre, et comptait donc des temoins conclus qui ne l'etaient pas. Chaque ligne porte
// desormais son `statut`, TOUJOURS renseigne, et la cause de son absence.
func TestLigneVersJSONPorteStatutEtCause(t *testing.T) {
	cas := []struct {
		nom    string
		ligne  ligneRapport
		statut string
	}{
		{"propre", ligneRapport{Temoin: Temoin{ID: "a"}, Gains: 3}, statutOK},
		{"perte", ligneRapport{Temoin: Temoin{ID: "b"}, Pertes: 2}, statutPerte},
		{"changement", ligneRapport{Temoin: Temoin{ID: "c"}, Changements: 2}, statutChangement},
		{"erreur", ligneRapport{Temoin: Temoin{ID: "d"}, Erreur: errTest("plafond memoire")}, statutErreur},
		{"absent", ligneRapport{Temoin: Temoin{ID: "e"}, Absent: true, AbsentCause: "faits non exportes"}, statutAbsent},
	}
	for _, c := range cas {
		lj := ligneVersJSON(c.ligne)
		if lj.Statut != c.statut {
			t.Errorf("%s : statut JSON = %q, %q attendu", c.nom, lj.Statut, c.statut)
		}
	}
	if got := ligneVersJSON(cas[3].ligne).Erreur; got != "plafond memoire" {
		t.Errorf("la cause de l'erreur doit etre portee au JSON, obtenu %q", got)
	}
	if got := ligneVersJSON(cas[4].ligne).AbsentCause; got != "faits non exportes" {
		t.Errorf("la cause de l'absence doit etre portee au JSON, obtenu %q", got)
	}
}

// TestLigneVersJSONPorteLesDeuxDetails — `changementsDetail` est le symetrique de
// `pertesDetail`, et les deux listes sont DISJOINTES.
func TestLigneVersJSONPorteLesDeuxDetails(t *testing.T) {
	lj := ligneVersJSON(ligneRapport{
		Temoin: Temoin{ID: "a"}, Pertes: 1, Changements: 1,
		PertesDetail: []replaydiff.Difference{
			{Axe: "couverture", Metrique: "coverage.bridge.livesTotal", Sens: replaydiff.SensPerte, Ancien: "58", Nouveau: "57"},
		},
		ChangementsDetail: []replaydiff.Difference{
			{Axe: "pistes", Metrique: "tracks/par-xuid/2535429985869093", Sens: replaydiff.SensChangement, Ancien: "6", Nouveau: "5"},
		},
	})
	if len(lj.PertesDetail) != 1 || lj.PertesDetail[0].Metrique != "coverage.bridge.livesTotal" {
		t.Errorf("pertesDetail mal porte : %+v", lj.PertesDetail)
	}
	if len(lj.ChangementsDetail) != 1 || lj.ChangementsDetail[0].Metrique != "tracks/par-xuid/2535429985869093" {
		t.Errorf("changementsDetail mal porte : %+v", lj.ChangementsDetail)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

// TestBilanDepuisRapportExtraitLeDetailDesPertesSeulement — LE COMPORTEMENT DEMANDE :
// "rapporter, pas masquer". Le detail des PERTES ne doit contenir QUE les sens Perte/Disparu —
// jamais un gain ni un changement, qui noieraient le signal — et celui des CHANGEMENTS que le
// sens Changement (2026-09-17, les deux listes sont DISJOINTES).
func TestBilanDepuisRapportExtraitLeDetailDesPertesSeulement(t *testing.T) {
	rap := replaydiff.Rapport{
		Differences: []replaydiff.Difference{
			{Axe: "objectifs", Metrique: "flag_captures", Sens: replaydiff.SensPerte, Ancien: "3", Nouveau: "1"},
			{Axe: "objectifs", Metrique: "flag_grabs", Sens: replaydiff.SensDisparu, Ancien: "5"},
			{Axe: "armes", Metrique: "pickups/n", Sens: replaydiff.SensGain, Nouveau: "12"},
			{Axe: "carte", Metrique: "bounds.maxX", Sens: replaydiff.SensChangement, Ancien: "1", Nouveau: "2"},
		},
	}
	var l ligneRapport
	l.remplirBilan(rap)
	if len(l.PertesDetail) != 2 {
		t.Fatalf("%d entrees de detail de perte, attendu 2 (perte + disparu seulement) : %+v",
			len(l.PertesDetail), l.PertesDetail)
	}
	for _, d := range l.PertesDetail {
		if d.Sens != replaydiff.SensPerte && d.Sens != replaydiff.SensDisparu {
			t.Errorf("le detail des pertes contient un sens %q — seuls perte/disparu sont attendus", d.Sens)
		}
	}
	if len(l.ChangementsDetail) != 1 {
		t.Fatalf("%d entrees de detail de changement, attendu 1 : %+v",
			len(l.ChangementsDetail), l.ChangementsDetail)
	}
	if l.ChangementsDetail[0].Sens != replaydiff.SensChangement {
		t.Errorf("le detail des changements contient un sens %q — seul changement est attendu",
			l.ChangementsDetail[0].Sens)
	}
}

// TestImprimerDetailPertesNommeAxeEtMetrique — un temoin en perte doit voir son detail
// imprime, avec l'axe ET la metrique nommes (pas juste un compte).
func TestImprimerDetailPertesNommeAxeEtMetrique(t *testing.T) {
	var b strings.Builder
	imprimerDetailPertes(&b, []ligneRapport{
		{
			Temoin: Temoin{ID: "aaaa1111", Famille: "ctf"}, Pertes: 1,
			PertesDetail: []replaydiff.Difference{
				{Axe: "objectifs", Metrique: "objectives/par-joueur/42/flag_captures", Sens: replaydiff.SensPerte, Ancien: "3", Nouveau: "1"},
			},
		},
		{Temoin: Temoin{ID: "bbbb2222", Famille: "slayer"}, Pertes: 0},
	})
	out := b.String()
	for _, attendu := range []string{"aaaa1111", "objectifs", "objectives/par-joueur/42/flag_captures", "3", "1"} {
		if !strings.Contains(out, attendu) {
			t.Errorf("le detail doit contenir %q :\n%s", attendu, out)
		}
	}
	if strings.Contains(out, "bbbb2222") {
		t.Errorf("un temoin sans perte ne doit pas apparaitre dans le detail :\n%s", out)
	}
}

// TestImprimerDetailPertesVideNEcritRien — aucun temoin en perte : pas de section vide qui
// laisserait croire a un rapport tronque.
func TestImprimerDetailPertesVideNEcritRien(t *testing.T) {
	var b strings.Builder
	imprimerDetailPertes(&b, []ligneRapport{{Temoin: Temoin{ID: "aaaa1111"}, Pertes: 0}})
	if b.String() != "" {
		t.Fatalf("aucun temoin en perte : sortie attendue vide, obtenu %q", b.String())
	}
}

// TestCodeSortiePertesNonBloquantesEnModeInformatif — LE COMPORTEMENT DEMANDE : en mode
// --reference=parc SANS --strict, une perte n'est qu'informative (code 0) — le parc n'est
// jamais a jour, un gate qui echouerait dessus a chaque fois ne gaterait rien.
func TestCodeSortiePertesNonBloquantesEnModeInformatif(t *testing.T) {
	lignes := []ligneRapport{{Temoin: Temoin{ID: "a"}, Pertes: 5}}
	if got := codeSortie(lignes, false); got != codeOK {
		t.Fatalf("code = %d, attendu codeOK (0) (pertesBloquent=false, mode parc informatif)", got)
	}
}

// TestCodeSortieErreurBloqueMemeEnModeInformatif — une ERREUR de cuisson/comparaison reste
// TOUJOURS bloquante, meme en mode informatif : le gate n'a alors pas pu faire son travail,
// ce qui est distinct d'une perte mesuree.
func TestCodeSortieErreurBloqueMemeEnModeInformatif(t *testing.T) {
	lignes := []ligneRapport{{Temoin: Temoin{ID: "a"}, Erreur: errTest("cuisson cassee")}}
	if got := codeSortie(lignes, false); got != codeErreurCuisson {
		t.Fatalf("code = %d, attendu codeErreurCuisson (3) (une erreur bloque toujours, meme pertesBloquent=false)", got)
	}
}

// TestVerifierCouvertureRefuseUnManifesteEntierementAbsent — CORPUS-R1 C3 (L6, P0) : LE
// SCENARIO qui rendait un gate vert SANS RIEN COMPARER (cache de film purge -> tous les temoins
// ABSENT -> codeSortie les sautait tous -> exit 0). verifierCouverture doit refuser AVANT que
// codeSortie ne soit meme appele.
func TestVerifierCouvertureRefuseUnManifesteEntierementAbsent(t *testing.T) {
	lignes := []ligneRapport{
		{Temoin: Temoin{ID: "a"}, Absent: true, AbsentCause: "cache purge"},
		{Temoin: Temoin{ID: "b"}, Absent: true, AbsentCause: "cache purge"},
	}
	err := verifierCouverture(lignes, false)
	if err == nil {
		t.Fatal("un manifeste entierement absent doit etre refuse (couverture incomplete), pas silencieux")
	}
	for _, attendu := range []string{"a", "b", "cache purge", "2/2"} {
		if !strings.Contains(err.Error(), attendu) {
			t.Errorf("le message doit nommer les temoins absents et leur cause, %q manquant : %v", attendu, err)
		}
	}
}

// TestVerifierCouvertureRefuseUnSeulAbsentParmiDAutres — meme un SEUL temoin absent (pas
// necessairement tous) est une couverture incomplete par defaut : il pourrait masquer une
// regression sur CE temoin precis.
func TestVerifierCouvertureRefuseUnSeulAbsentParmiDAutres(t *testing.T) {
	lignes := []ligneRapport{
		{Temoin: Temoin{ID: "a"}, Absent: true, AbsentCause: "cache purge"},
		{Temoin: Temoin{ID: "b"}, Pertes: 0},
	}
	if err := verifierCouverture(lignes, false); err == nil {
		t.Fatal("un seul temoin absent parmi d'autres doit aussi etre refuse par defaut")
	}
}

// TestVerifierCouvertureToleranteAvecAllowMissing — --allow-missing restaure explicitement
// l'ancien comportement (avertissement seul, jamais bloquant) pour un usage delibere.
func TestVerifierCouvertureToleranteAvecAllowMissing(t *testing.T) {
	lignes := []ligneRapport{{Temoin: Temoin{ID: "a"}, Absent: true, AbsentCause: "cache purge"}}
	if err := verifierCouverture(lignes, true); err != nil {
		t.Fatalf("--allow-missing doit tolerer l'absence, obtenu : %v", err)
	}
}

// TestVerifierCouvertureAucunAbsentToujoursOK — le cas nominal (rien d'absent) ne doit jamais
// rendre d'erreur, avec ou sans --allow-missing.
func TestVerifierCouvertureAucunAbsentToujoursOK(t *testing.T) {
	lignes := []ligneRapport{{Temoin: Temoin{ID: "a"}, Pertes: 0}, {Temoin: Temoin{ID: "b"}, Pertes: 3}}
	if err := verifierCouverture(lignes, false); err != nil {
		t.Fatalf("aucun temoin absent : attendu nil, obtenu %v", err)
	}
}
