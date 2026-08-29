package analysis

// team_score_display_test.go — la règle points/manches, sur les témoins RÉELS du corpus
// mesuré le 2026-08-29 (`.ai/V7.5/RAPPORT_MANCHES_2026-08-29.md`).

import "testing"

func ip(v int) *int { return &v }

func TestReadTeamScore(t *testing.T) {
	cases := []struct {
		name       string
		in         TeamScoreInput
		wantOK     bool
		wantKind   ScoreKind
		wantMine   int
		wantTheirs int
		wantPoints *[2]int
	}{
		{
			// Témoin 293a763e : victoire 2 manches à 1 alors que les POINTS disent 181-186.
			// C'est le cas qui justifie tout le chantier.
			name: "Oddball : les manches priment sur des points qui mentent",
			in: TeamScoreInput{
				MyPoints: ip(181), EnemyPoints: ip(186),
				MyRoundsWon: ip(2), EnemyRoundsWon: ip(1), RoundsTotal: ip(3),
				RoundsDecide: true,
			},
			wantOK: true, wantKind: ScoreKindRounds, wantMine: 2, wantTheirs: 1,
			wantPoints: &[2]int{181, 186},
		},
		{
			// Le CTF d'arène a bien 2 « manches » (deux mi-temps) mais n'est PAS déclaré :
			// c'est exactement le faux positif que la mesure a écarté.
			name: "CTF d'arène : deux mi-temps, mais variante non déclarée -> points",
			in: TeamScoreInput{
				MyPoints: ip(2), EnemyPoints: ip(3),
				MyRoundsWon: ip(0), EnemyRoundsWon: ip(1), RoundsTotal: ip(2),
				RoundsDecide: false,
			},
			wantOK: true, wantKind: ScoreKindPoints, wantMine: 2, wantTheirs: 3,
			wantPoints: &[2]int{2, 3},
		},
		{
			name: "Slayer : une seule manche -> points",
			in: TeamScoreInput{
				MyPoints: ip(50), EnemyPoints: ip(43),
				MyRoundsWon: ip(1), EnemyRoundsWon: ip(0), RoundsTotal: ip(1),
				RoundsDecide: true,
			},
			wantOK: true, wantKind: ScoreKindPoints, wantMine: 50, wantTheirs: 43,
			wantPoints: &[2]int{50, 43},
		},
		{
			// Témoin adb93fb7 : 1 manche chacun + 1 nulle. Les manches ne désignent
			// personne, on retombe sur les points.
			name: "manches à égalité -> points",
			in: TeamScoreInput{
				MyPoints: ip(277), EnemyPoints: ip(234),
				MyRoundsWon: ip(1), EnemyRoundsWon: ip(1), RoundsTotal: ip(3),
				RoundsDecide: true,
			},
			wantOK: true, wantKind: ScoreKindPoints, wantMine: 277, wantTheirs: 234,
			wantPoints: &[2]int{277, 234},
		},
		{
			// Ligne antérieure au backfill : colonnes NULL. Aucune régression.
			name: "manches inconnues -> points",
			in: TeamScoreInput{
				MyPoints: ip(160), EnemyPoints: ip(118), RoundsDecide: true,
			},
			wantOK: true, wantKind: ScoreKindPoints, wantMine: 160, wantTheirs: 118,
			wantPoints: &[2]int{160, 118},
		},
		{
			// Manches connues et décisives, points absents : on affiche quand même le
			// résultat. Le secondaire manque, pas le principal.
			name: "points inconnus mais manches décisives -> manches sans secondaire",
			in: TeamScoreInput{
				MyRoundsWon: ip(2), EnemyRoundsWon: ip(0), RoundsTotal: ip(2),
				RoundsDecide: true,
			},
			wantOK: true, wantKind: ScoreKindRounds, wantMine: 2, wantTheirs: 0,
			wantPoints: nil,
		},
		{
			name:   "rien de connu -> rien à afficher",
			in:     TeamScoreInput{RoundsDecide: true},
			wantOK: false,
		},
		{
			name:   "score négatif -> rien à afficher",
			in:     TeamScoreInput{MyPoints: ip(-1), EnemyPoints: ip(3)},
			wantOK: false,
		},
		{
			name: "un seul camp connu en points -> rien à afficher",
			in: TeamScoreInput{
				MyPoints: ip(50), RoundsDecide: false,
			},
			wantOK: false,
		},
		{
			name: "un seul camp connu en manches -> repli sur les points",
			in: TeamScoreInput{
				MyPoints: ip(160), EnemyPoints: ip(95),
				MyRoundsWon: ip(2), RoundsTotal: ip(2), RoundsDecide: true,
			},
			wantOK: true, wantKind: ScoreKindPoints, wantMine: 160, wantTheirs: 95,
			wantPoints: &[2]int{160, 95},
		},
		{
			name: "zéro manche gagnée est une mesure, pas une absence",
			in: TeamScoreInput{
				MyPoints: ip(45), EnemyPoints: ip(160),
				MyRoundsWon: ip(0), EnemyRoundsWon: ip(2), RoundsTotal: ip(2),
				RoundsDecide: true,
			},
			wantOK: true, wantKind: ScoreKindRounds, wantMine: 0, wantTheirs: 2,
			wantPoints: &[2]int{45, 160},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ReadTeamScore(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if got.Kind != tc.wantKind {
				t.Errorf("Kind = %q, want %q", got.Kind, tc.wantKind)
			}
			if got.Mine != tc.wantMine || got.Theirs != tc.wantTheirs {
				t.Errorf("affichage = %d - %d, want %d - %d", got.Mine, got.Theirs, tc.wantMine, tc.wantTheirs)
			}
			switch {
			case tc.wantPoints == nil && got.Points != nil:
				t.Errorf("Points = %v, want nil", *got.Points)
			case tc.wantPoints != nil && got.Points == nil:
				t.Errorf("Points = nil, want %v", *tc.wantPoints)
			case tc.wantPoints != nil && *got.Points != *tc.wantPoints:
				t.Errorf("Points = %v, want %v", *got.Points, *tc.wantPoints)
			}
		})
	}
}
