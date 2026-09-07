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
	if got := codeSortie(lignes); got != 0 {
		t.Fatalf("code = %d, attendu 0", got)
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
	if got := codeSortie(lignes); got != 1 {
		t.Fatalf("code = %d, attendu 1 (b porte une perte)", got)
	}
}

// TestCodeSortieErreurDeCuissonEstUnEchec — un temoin qui n'a pas pu etre cuit ou compare doit
// faire echouer le gate : un rapport incomplet n'est pas un rapport vert.
func TestCodeSortieErreurDeCuissonEstUnEchec(t *testing.T) {
	lignes := []ligneRapport{{Temoin: Temoin{ID: "a"}, Erreur: errTest("cuisson cassee")}}
	if got := codeSortie(lignes); got != 1 {
		t.Fatalf("code = %d, attendu 1 (erreur de cuisson)", got)
	}
}

// TestCodeSortieAbsentNEstPasUnEchec — LE COMPORTEMENT DEMANDE explicitement : un temoin
// absent du parc local est un avertissement (deja emis en slog par traiterTemoin), jamais un
// echec — sinon le gate echouerait toujours sur un poste sans parc local.
func TestCodeSortieAbsentNEstPasUnEchec(t *testing.T) {
	lignes := []ligneRapport{
		{Temoin: Temoin{ID: "a"}, Absent: true, AbsentCause: "aucun chunk"},
	}
	if got := codeSortie(lignes); got != 0 {
		t.Fatalf("code = %d, attendu 0 (absent != echec)", got)
	}
}

// TestBilanDepuisRapportSommeLesAxes — gains et pertes s'additionnent sur TOUS les axes du
// rapport, pas seulement le premier.
func TestBilanDepuisRapportSommeLesAxes(t *testing.T) {
	rap := replaydiff.Rapport{
		SchemaAncien: 20, SchemaNouveau: 41,
		Bilans: map[string]replaydiff.BilanAxe{
			"objectifs":  {Gains: 3, Pertes: 1},
			"equipement": {Gains: 0, Pertes: 2},
		},
	}
	schemaParc, schemaHEAD, gains, pertes, _ := bilanDepuisRapport(rap)
	if schemaParc != 20 || schemaHEAD != 41 {
		t.Fatalf("schemas = %d -> %d, attendu 20 -> 41", schemaParc, schemaHEAD)
	}
	if gains != 3 || pertes != 3 {
		t.Fatalf("gains=%d pertes=%d, attendu gains=3 pertes=3 (somme des deux axes)", gains, pertes)
	}
}

// TestImprimerTableauNommeLeStatut — le tableau doit distinguer absent / erreur / perte / ok
// en toutes lettres, pour un operateur qui ne lit que la derniere colonne.
func TestImprimerTableauNommeLeStatut(t *testing.T) {
	var b strings.Builder
	imprimerTableau(&b, []ligneRapport{
		{Temoin: Temoin{ID: "aaaa1111", Famille: "ctf"}, Gains: 2, Pertes: 0},
		{Temoin: Temoin{ID: "bbbb2222", Famille: "oddball"}, Pertes: 1},
		{Temoin: Temoin{ID: "cccc3333", Famille: "slayer"}, Absent: true, AbsentCause: "aucun chunk"},
		{Temoin: Temoin{ID: "dddd4444", Famille: "assaut"}, Erreur: errTest("carte hors catalogue")},
	})
	out := b.String()
	for _, attendu := range []string{
		"aaaa1111", "ok",
		"bbbb2222", "PERTE",
		"cccc3333", "ABSENT DU PARC", "aucun chunk",
		"dddd4444", "ERREUR", "carte hors catalogue",
	} {
		if !strings.Contains(out, attendu) {
			t.Errorf("le tableau doit contenir %q :\n%s", attendu, out)
		}
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

// TestBilanDepuisRapportExtraitLeDetailDesPertesSeulement — LE COMPORTEMENT DEMANDE :
// "rapporter, pas masquer". Le detail ne doit contenir QUE les sens Perte/Disparu — jamais un
// gain ni un changement, qui noieraient le signal.
func TestBilanDepuisRapportExtraitLeDetailDesPertesSeulement(t *testing.T) {
	rap := replaydiff.Rapport{
		Differences: []replaydiff.Difference{
			{Axe: "objectifs", Metrique: "flag_captures", Sens: replaydiff.SensPerte, Ancien: "3", Nouveau: "1"},
			{Axe: "objectifs", Metrique: "flag_grabs", Sens: replaydiff.SensDisparu, Ancien: "5"},
			{Axe: "armes", Metrique: "pickups/n", Sens: replaydiff.SensGain, Nouveau: "12"},
			{Axe: "carte", Metrique: "bounds.maxX", Sens: replaydiff.SensChangement, Ancien: "1", Nouveau: "2"},
		},
	}
	_, _, _, _, detail := bilanDepuisRapport(rap)
	if len(detail) != 2 {
		t.Fatalf("%d entrees de detail, attendu 2 (perte + disparu seulement) : %+v", len(detail), detail)
	}
	for _, d := range detail {
		if d.Sens != replaydiff.SensPerte && d.Sens != replaydiff.SensDisparu {
			t.Errorf("le detail contient un sens %q — seuls perte/disparu sont attendus", d.Sens)
		}
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
