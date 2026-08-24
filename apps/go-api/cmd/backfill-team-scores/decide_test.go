package main

// decide_test.go — la règle de correction, sans réseau ni base.
//
// Ce que ces tests protègent : le seul endroit de l'outil dont une erreur écrirait une
// valeur FAUSSE en production. Chaque cas de la table correspond à une situation observée
// dans le corpus mesuré le 2026-08-24 ou à une garde explicitement demandée.

import (
	"fmt"
	"testing"
)

// teamsPayload fabrique un payload GetMatchStats minimal mais de la MÊME forme que celui
// de l'API : `Teams[].Stats.CoreStats.Score`, les nombres en float64 comme les rend
// encoding/json. Un helper qui simplifierait la forme ne testerait plus l'extraction réelle.
func teamsPayload(teams ...map[string]any) map[string]any {
	anyTeams := make([]any, 0, len(teams))
	for _, t := range teams {
		anyTeams = append(anyTeams, t)
	}
	return map[string]any{"Teams": anyTeams}
}

func team(id int, score float64) map[string]any {
	return map[string]any{
		"TeamId": float64(id),
		"Stats":  map[string]any{"CoreStats": map[string]any{"Score": score}},
	}
}

// teamWithZones ajoute le bloc voisin qui a causé le défaut. Sa présence ne doit JAMAIS
// changer la décision : c'est `CoreStats.Score` qui fait foi, pas les ticks.
func teamWithZones(id int, score, ticks float64) map[string]any {
	t := team(id, score)
	t["Stats"].(map[string]any)["ZonesStats"] = map[string]any{"StrongholdScoringTicks": ticks}
	return t
}

func ptr(v int) *int { return &v }

// Exemption de seuil assumée (CLAUDE.md, fonction <= 80 L) : le corps est une TABLE de
// cas, pas de la logique. La découper en plusieurs fonctions éclaterait la table sans rien
// simplifier et rendrait plus difficile de voir qu'un cas a son miroir sur l'autre camp.
func TestDecide(t *testing.T) {
	cases := []struct {
		name     string
		payload  map[string]any
		current  RegistryScores
		want     Verdict
		wantT0   int
		wantT1   int
		writable bool
	}{
		{
			// Cas réel `7344d24f` : la base porte les ticks de zone (193/112), l'API le
			// score affiché (200/126). C'est LE cas que l'outil existe pour réparer.
			name:     "ticks de zone en base, score affiche a l'API",
			payload:  teamsPayload(teamWithZones(0, 200, 193), teamWithZones(1, 126, 112)),
			current:  RegistryScores{Team0: ptr(193), Team1: ptr(112)},
			want:     VerdictFix,
			wantT0:   200,
			wantT1:   126,
			writable: true,
		},
		{
			// Cas réel `696a9d7c` : déjà juste. L'outil doit être idempotent et muet.
			name:    "base deja conforme",
			payload: teamsPayload(teamWithZones(0, 200, 189), teamWithZones(1, 94, 94)),
			current: RegistryScores{Team0: ptr(200), Team1: ptr(94)},
			want:    VerdictIdentical,
			wantT0:  200,
			wantT1:  94,
		},
		{
			// Un zéro conforme reste conforme : ne pas confondre 0 et « absent ».
			name:    "zero conforme des deux cotes",
			payload: teamsPayload(team(0, 0), team(1, 0)),
			current: RegistryScores{Team0: ptr(0), Team1: ptr(0)},
			want:    VerdictIdentical,
		},
		{
			// NULL n'est PAS zéro : une colonne NULL est une ligne à corriger.
			name:     "colonne NULL contre score nul a l'API",
			payload:  teamsPayload(team(0, 0), team(1, 3)),
			current:  RegistryScores{Team0: nil, Team1: nil},
			want:     VerdictFix,
			wantT0:   0,
			wantT1:   3,
			writable: true,
		},
		{
			// Un seul des deux NULL : toujours à corriger, jamais « à moitié conforme ».
			name:     "un seul camp NULL",
			payload:  teamsPayload(team(0, 50), team(1, 42)),
			current:  RegistryScores{Team0: ptr(50), Team1: nil},
			want:     VerdictFix,
			wantT0:   50,
			wantT1:   42,
			writable: true,
		},
		{
			// Cas réel des 6 Oddball inversés : la correction remet chaque score sur son camp.
			name:     "camps inverses en base",
			payload:  teamsPayload(team(0, 218), team(1, 155)),
			current:  RegistryScores{Team0: ptr(155), Team1: ptr(218)},
			want:     VerdictFix,
			wantT0:   218,
			wantT1:   155,
			writable: true,
		},
		{
			// FFA : l'API ne publie pas TeamId 0 et 1. On ne devine pas un camp.
			name:    "aucun camp 0 ni 1 (FFA)",
			payload: teamsPayload(team(2, 30), team(3, 25)),
			current: RegistryScores{Team0: ptr(30), Team1: ptr(25)},
			want:    VerdictSkipNoTeams,
		},
		{
			name:    "camp 1 absent du payload",
			payload: teamsPayload(team(0, 12)),
			current: RegistryScores{Team0: ptr(12), Team1: ptr(0)},
			want:    VerdictSkipNoTeams,
		},
		{
			name:    "payload sans bloc Teams",
			payload: map[string]any{},
			current: RegistryScores{Team0: ptr(1), Team1: ptr(2)},
			want:    VerdictSkipNoTeams,
		},
		{
			// Garde de vraisemblance : jamais de négatif en base.
			name:    "score negatif refuse (camp 0)",
			payload: teamsPayload(team(0, -1), team(1, 5)),
			current: RegistryScores{Team0: ptr(0), Team1: ptr(5)},
			want:    VerdictSkipImplausible,
		},
		{
			// MIROIR du précédent sur l'autre camp. Sans lui, réduire la boucle de
			// vraisemblance au seul t0 passerait tous les tests (constat P2-6 de la
			// revue adversariale du 2026-08-24).
			name:    "score negatif refuse (camp 1)",
			payload: teamsPayload(team(0, 5), team(1, -1)),
			current: RegistryScores{Team0: ptr(5), Team1: ptr(0)},
			want:    VerdictSkipImplausible,
		},
		{
			// Garde de vraisemblance : la colonne est un SMALLINT.
			name:    "score hors bornes SMALLINT refuse (camp 0)",
			payload: teamsPayload(team(0, 32768), team(1, 5)),
			current: RegistryScores{Team0: ptr(1), Team1: ptr(5)},
			want:    VerdictSkipImplausible,
		},
		{
			// MIROIR sur le camp 1.
			name:    "score hors bornes SMALLINT refuse (camp 1)",
			payload: teamsPayload(team(0, 5), team(1, 32768)),
			current: RegistryScores{Team0: ptr(5), Team1: ptr(1)},
			want:    VerdictSkipImplausible,
		},
		{
			name:    "borne haute SMALLINT exactement acceptee (camp 0)",
			payload: teamsPayload(team(0, 32767), team(1, 5)),
			current: RegistryScores{Team0: ptr(1), Team1: ptr(5)},
			want:    VerdictFix,
			wantT0:  32767, wantT1: 5,
			writable: true,
		},
		{
			// MIROIR : la borne haute doit être acceptée des DEUX côtés.
			name:    "borne haute SMALLINT exactement acceptee (camp 1)",
			payload: teamsPayload(team(0, 5), team(1, 32767)),
			current: RegistryScores{Team0: ptr(1), Team1: ptr(5)},
			want:    VerdictFix,
			wantT0:  5, wantT1: 32767,
			writable: true,
		},
		{
			// MIROIR de « camp 1 absent » : ici c'est le camp 0 qui manque.
			name:    "camp 0 absent du payload",
			payload: teamsPayload(team(1, 12)),
			current: RegistryScores{Team0: ptr(0), Team1: ptr(12)},
			want:    VerdictSkipNoTeams,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(tc.payload, tc.current)
			if got.Verdict != tc.want {
				t.Fatalf("verdict = %q, attendu %q (cause: %s)", got.Verdict, tc.want, got.Reason)
			}
			if got.Writable() != tc.writable {
				t.Errorf("Writable() = %v, attendu %v", got.Writable(), tc.writable)
			}
			if tc.want == VerdictFix || tc.want == VerdictIdentical {
				if got.NewTeam0 != tc.wantT0 || got.NewTeam1 != tc.wantT1 {
					t.Errorf("scores = %d/%d, attendu %d/%d", got.NewTeam0, got.NewTeam1, tc.wantT0, tc.wantT1)
				}
			}
			if got.Reason == "" {
				t.Error("Reason vide : tout verdict doit être justifiable au journal")
			}
		})
	}
}

// TestDecide_TicksNeverWin fige l'invariant central : quand le score affiché et le
// compteur de ticks coexistent dans le payload, c'est TOUJOURS le score affiché qui est
// écrit. Une régression ici rejouerait exactement le bug d'origine.
func TestDecide_TicksNeverWin(t *testing.T) {
	payload := teamsPayload(teamWithZones(0, 200, 193), teamWithZones(1, 126, 112))
	got := Decide(payload, RegistryScores{Team0: ptr(193), Team1: ptr(112)})
	if got.NewTeam0 == 193 || got.NewTeam1 == 112 {
		t.Fatalf("les ticks de zone ont gagné : %d/%d — CoreStats.Score doit faire foi", got.NewTeam0, got.NewTeam1)
	}
	if got.NewTeam0 != 200 || got.NewTeam1 != 126 {
		t.Fatalf("scores = %d/%d, attendu 200/126", got.NewTeam0, got.NewTeam1)
	}
}

// TestDecide_NeverWritesNilOrNegative balaie une plage de valeurs et vérifie qu'AUCUN
// verdict écrivable ne porte une valeur hors du domaine autorisé de la colonne.
// La plage est balayée sur LES DEUX camps : une garde qui ne contrôlerait que le camp 0
// laisserait passer une valeur hors domaine sur le camp 1.
func TestDecide_NeverWritesNilOrNegative(t *testing.T) {
	for _, v := range []float64{-100000, -32768, -1, 0, 1, 32767, 32768, 100000} {
		for _, camp := range []int{0, 1} {
			t.Run(fmt.Sprintf("camp%d_score_%.0f", camp, v), func(t *testing.T) {
				payload := teamsPayload(team(0, v), team(1, 0))
				if camp == 1 {
					payload = teamsPayload(team(0, 0), team(1, v))
				}
				got := Decide(payload, RegistryScores{})
				if !got.Writable() {
					return
				}
				if got.NewTeam0 < scoreMin || got.NewTeam0 > scoreMax ||
					got.NewTeam1 < scoreMin || got.NewTeam1 > scoreMax {
					t.Fatalf("verdict écrivable avec une valeur hors domaine : %d/%d", got.NewTeam0, got.NewTeam1)
				}
			})
		}
	}
}

func TestFormatScores_DistingueNullEtZero(t *testing.T) {
	if got := formatScores(RegistryScores{Team0: nil, Team1: ptr(0)}); got != "NULL/0" {
		t.Errorf("formatScores = %q, attendu \"NULL/0\" — confondre NULL et 0 masquerait une ligne à corriger", got)
	}
}
