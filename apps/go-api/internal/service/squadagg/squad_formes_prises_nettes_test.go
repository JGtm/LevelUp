package squadagg

// squad_formes_prises_nettes_test.go — LA JOINTURE DES PRISES NETTES aux colonnes d'objectif
// du bloc « formes retenues ».
//
// CE FICHIER EXISTE PARCE QUE LA REVUE DU 2026-09-13 A RENDU `joindrePrisesNettes` NO-OP ET
// QUE RIEN N'A ROUGI (mutation M7, constat C1). Le premier test rougit sous cette mutation.
//
// LES DEUX AUTRES TIENNENT LA PROPRIÉTÉ QUI COÛTE LE PLUS CHER SI ELLE CASSE : un (match,
// joueur) SANS mesure ne reçoit AUCUNE clé — ni zéro, ni valeur d'un autre match.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
)

// repoObjectifsFactice implémente port.SquadFormesObjectiveRepository.
type repoObjectifsFactice struct {
	colonnes  []squadformes.ObjectiveColumnRow
	prises    []sessionusage.FlagGrabsNetRow
	prisesErr error
}

func (r *repoObjectifsFactice) LoadObjectiveColumnRows(_ context.Context, _ []string) ([]squadformes.ObjectiveColumnRow, error) {
	return r.colonnes, nil
}

func (r *repoObjectifsFactice) LoadFlagGrabsNet(_ context.Context, _ []string) ([]sessionusage.FlagGrabsNetRow, error) {
	return r.prises, r.prisesErr
}

// colonnesCTF : deux matchs de CTF, deux joueurs sur le premier, un sur le second.
func colonnesCTF() []squadformes.ObjectiveColumnRow {
	return []squadformes.ObjectiveColumnRow{
		{MatchID: "m1", XUID: "P", Family: narrative.FamilyCTF, Values: map[string]float64{"flag_captures": 1}},
		{MatchID: "m1", XUID: "A", Family: narrative.FamilyCTF, Values: map[string]float64{"flag_captures": 0}},
		{MatchID: "m2", XUID: "P", Family: narrative.FamilyCTF, Values: map[string]float64{"flag_captures": 2}},
	}
}

func ligne(t *testing.T, rows []squadformes.ObjectiveColumnRow, matchID, xuid string) squadformes.ObjectiveColumnRow {
	t.Helper()
	for _, r := range rows {
		if r.MatchID == matchID && r.XUID == xuid {
			return r
		}
	}
	t.Fatalf("ligne (%s, %s) absente", matchID, xuid)
	return squadformes.ObjectiveColumnRow{}
}

// TestJoindrePrisesNettes_LaGrandeurArriveSurLesLignes — LE test qui rougit sous la mutation
// M7 (la jointure rendue no-op).
func TestJoindrePrisesNettes_LaGrandeurArriveSurLesLignes(t *testing.T) {
	repo := &repoObjectifsFactice{
		colonnes: colonnesCTF(),
		prises: []sessionusage.FlagGrabsNetRow{
			{MatchID: "m1", XUID: "P", Raw: 9, Net: 3, Openings: 25, WindowMS: 1500},
			{MatchID: "m1", XUID: "A", Raw: 4, Net: 4, Openings: 25, WindowMS: 1500},
		},
	}
	rows := loadFormesObjectives(context.Background(),
		SquadFormesQuery{Objectives: repo}, []string{"m1", "m2"})

	p := ligne(t, rows, "m1", "P")
	if v, ok := p.Values[narrative.GrandeurFlagGrabsNet]; !ok || v != 3 {
		t.Errorf("(m1, P) = (%v, %v), attendu (3, true) — la jointure est débranchée", v, ok)
	}
	// LA FENÊTRE VOYAGE AVEC LA MESURE : sans elle, l'infobulle ne peut pas dire la règle.
	if p.FlagJuggleWindowSeconds != 1.5 {
		t.Errorf("fenêtre de (m1, P) = %v, attendu 1.5", p.FlagJuggleWindowSeconds)
	}
	if v := ligne(t, rows, "m1", "A").Values[narrative.GrandeurFlagGrabsNet]; v != 4 {
		t.Errorf("(m1, A) = %v, attendu 4", v)
	}
}

// TestJoindrePrisesNettes_SansMesureAucuneCle — un match sans film lu ne reçoit PAS de zéro :
// c'est l'absence de clé qui porte le « non mesuré » jusqu'à l'écran.
func TestJoindrePrisesNettes_SansMesureAucuneCle(t *testing.T) {
	repo := &repoObjectifsFactice{
		colonnes: colonnesCTF(),
		prises: []sessionusage.FlagGrabsNetRow{
			{MatchID: "m1", XUID: "P", Raw: 9, Net: 3, Openings: 25, WindowMS: 1500},
		},
	}
	rows := loadFormesObjectives(context.Background(),
		SquadFormesQuery{Objectives: repo}, []string{"m1", "m2"})

	// m2 n'a pas de film lu.
	m2 := ligne(t, rows, "m2", "P")
	if _, ok := m2.Values[narrative.GrandeurFlagGrabsNet]; ok {
		t.Errorf("(m2, P) porte une valeur (%v) — ce serait un faux zéro",
			m2.Values[narrative.GrandeurFlagGrabsNet])
	}
	if m2.FlagJuggleWindowSeconds != 0 {
		t.Errorf("(m2, P) porte une fenêtre (%v) sans mesure", m2.FlagJuggleWindowSeconds)
	}
	// m1/A est mesuré comme match, mais ce joueur-là n'a pas de ligne de prises : même règle.
	if _, ok := ligne(t, rows, "m1", "A").Values[narrative.GrandeurFlagGrabsNet]; ok {
		t.Error("(m1, A) porte une valeur alors que la lecture ne le nomme pas")
	}
	// Les colonnes ORDINAIRES, elles, gardent leur zéro : il y est une mesure.
	if v, ok := ligne(t, rows, "m1", "A").Values["flag_captures"]; !ok || v != 0 {
		t.Errorf("flag_captures de (m1, A) = (%v, %v), attendu (0, true)", v, ok)
	}
}

// TestJoindrePrisesNettes_LectureEnEchecDegradeSeule — les colonnes d'objectif restent
// servies : deux lectures indépendantes dégradent indépendamment.
func TestJoindrePrisesNettes_LectureEnEchecDegradeSeule(t *testing.T) {
	repo := &repoObjectifsFactice{colonnes: colonnesCTF(), prisesErr: errors.New("vue absente")}
	rows := loadFormesObjectives(context.Background(),
		SquadFormesQuery{Objectives: repo}, []string{"m1", "m2"})

	if len(rows) != 3 {
		t.Fatalf("%d lignes, attendu 3 — l'échec des prises a emporté les colonnes", len(rows))
	}
	if _, ok := ligne(t, rows, "m1", "P").Values[narrative.GrandeurFlagGrabsNet]; ok {
		t.Error("une valeur de prises nettes a survécu à l'échec de lecture")
	}
}
