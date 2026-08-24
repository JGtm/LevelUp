package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// TestBuildAssistPairs_BlocAbsentSansFilm : aucune ligne de film pour ce match (ou titre
// sans décodeur) → AUCUN bloc. C'est la porte qui évite qu'un message « non mesuré »
// s'affiche sur tous les matchs d'un titre qui n'a jamais eu de décodeur.
func TestBuildAssistPairs_BlocAbsentSansFilm(t *testing.T) {
	got := buildAssistPairs(nil, domain.MatchAssistScopeRaw{}, nil)
	if got != nil {
		t.Fatalf("buildAssistPairs = %+v, attendu nil", got)
	}
	// Même verdict si la portée est vide alors que des paires seraient présentes :
	// c'est un état incohérent, et le bloc ne doit pas se fabriquer une portée.
	got = buildAssistPairs(
		[]domain.MatchAssistPairRaw{{AssistXUID: "A", KillerXUID: "K", AssistCount: 1}},
		domain.MatchAssistScopeRaw{},
		nil,
	)
	if got != nil {
		t.Fatalf("buildAssistPairs (portée vide) = %+v, attendu nil", got)
	}
}

// TestBuildAssistPairs_NonMesureVsZero : LES DEUX ÉTATS QUE LE BLOC EXISTE POUR
// DISTINGUER. Le film est là dans les deux cas ; ce qui change est MeasuredDeaths.
func TestBuildAssistPairs_NonMesureVsZero(t *testing.T) {
	// (a) « non mesuré » : des morts, aucune assistance mesurée.
	nonMesure := buildAssistPairs(nil, domain.MatchAssistScopeRaw{MatchDeaths: 40}, nil)
	if nonMesure == nil {
		t.Fatal("bloc attendu (le match a un film), obtenu nil")
	}
	if nonMesure.MeasuredDeaths != 0 {
		t.Errorf("MeasuredDeaths = %d, attendu 0", nonMesure.MeasuredDeaths)
	}
	if len(nonMesure.Pairs) != 0 {
		t.Errorf("Pairs = %+v, attendu vide", nonMesure.Pairs)
	}

	// (b) « mesuré, zéro assistance » : mêmes paires vides, portée DIFFÉRENTE.
	mesureZero := buildAssistPairs(nil, domain.MatchAssistScopeRaw{MatchDeaths: 40, MeasuredDeaths: 38}, nil)
	if mesureZero == nil {
		t.Fatal("bloc attendu, obtenu nil")
	}
	if mesureZero.MeasuredDeaths != 38 {
		t.Errorf("MeasuredDeaths = %d, attendu 38", mesureZero.MeasuredDeaths)
	}
	if len(mesureZero.Pairs) != 0 {
		t.Errorf("Pairs = %+v, attendu vide", mesureZero.Pairs)
	}
}

// TestBuildAssistPairs_GamertagTueurDepuisScoreboard : le nom du tueur vient du
// scoreboard et de lui seul. Un tueur qui n'y figure pas garde son xuid et un gamertag
// VIDE — jamais un nom inventé, jamais le xuid recopié dans un champ de nom.
func TestBuildAssistPairs_GamertagTueurDepuisScoreboard(t *testing.T) {
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "K1", Gamertag: "Kilo"},
		{XUID: "A1", Gamertag: "IgnoreMoi"}, // l'assistant NE prend PAS son nom d'ici
		{XUID: "K3", Gamertag: ""},          // présent mais anonyme : ne compte pas
	}
	raw := []domain.MatchAssistPairRaw{
		{AssistXUID: "A1", AssistGamertag: "Alpha", KillerXUID: "K1", AssistCount: 3, StolenCount: 2},
		{AssistXUID: "A1", AssistGamertag: "Alpha", KillerXUID: "K2", AssistCount: 1},
		{AssistXUID: "A1", AssistGamertag: "Alpha", KillerXUID: "K3", AssistCount: 1},
	}
	got := buildAssistPairs(raw, domain.MatchAssistScopeRaw{MatchDeaths: 50, MeasuredDeaths: 44}, scoreboard)
	if got == nil || len(got.Pairs) != 3 {
		t.Fatalf("bloc = %+v, attendu 3 paires", got)
	}
	if got.Pairs[0].KillerGamertag != "Kilo" {
		t.Errorf("tueur au scoreboard : gamertag = %q, attendu %q", got.Pairs[0].KillerGamertag, "Kilo")
	}
	if got.Pairs[1].KillerGamertag != "" {
		t.Errorf("tueur ABSENT du scoreboard : gamertag = %q, attendu vide", got.Pairs[1].KillerGamertag)
	}
	if got.Pairs[1].KillerXUID != "K2" {
		t.Errorf("le xuid du tueur doit survivre : %q", got.Pairs[1].KillerXUID)
	}
	if got.Pairs[2].KillerGamertag != "" {
		t.Errorf("tueur au scoreboard SANS nom : gamertag = %q, attendu vide", got.Pairs[2].KillerGamertag)
	}
	// L'assistant garde le nom que le FILM a écrit (le scoreboard ne le corrige pas :
	// c'est la mesure, et le graphe n'a pas à réécrire ce qui a été lu).
	if got.Pairs[0].AssistGamertag != "Alpha" {
		t.Errorf("assistant = %q, attendu %q", got.Pairs[0].AssistGamertag, "Alpha")
	}
	// Les compteurs traversent sans être recalculés — Q21d les a déjà comptés.
	if got.Pairs[0].AssistCount != 3 || got.Pairs[0].StolenCount != 2 {
		t.Errorf("compteurs = %d/%d, attendus 3/2", got.Pairs[0].AssistCount, got.Pairs[0].StolenCount)
	}
}
