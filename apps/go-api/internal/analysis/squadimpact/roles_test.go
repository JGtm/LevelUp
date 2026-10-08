package squadimpact

import (
	"testing"

	"levelup/go-api/internal/analysis"
)

// TestWeight_Bareme : chaque colonne de la matrice a son barème, et le barème est celui de la
// spécification (Finisseur et Premier sang +2, Héros silencieux +1,5, Bourreau +1 ; Boulet −2,
// Faux-frère −1,5, Première victime, Touriste, Kamikaze et Voleur −1).
func TestWeight_Bareme(t *testing.T) {
	want := map[string]float64{
		RoleClutchFinisher: 2, RoleFirstBlood: 2, RoleSilentHero: 1.5, RoleTopKiller: 1,
		RoleLastCasualty: -2, RoleFalseBrother: -1.5,
		RoleFirstGroupDeath: -1, RoleLastGroupKill: -1, RoleKamikaze: -1, RoleThief: -1,
	}
	for _, role := range RoleOrder() {
		w, ok := Weight(role)
		if !ok {
			t.Errorf("rôle %q de la matrice sans barème", role)
			continue
		}
		if w != want[role] {
			t.Errorf("%s : barème %v, attendu %v", role, w, want[role])
		}
	}
	if _, ok := Weight("top_gun"); ok {
		t.Error("top_gun est calculé mais hors barème")
	}
	if len(RoleOrder()) != len(want) || len(stackOrder) != len(want) {
		t.Errorf("colonnes %d, empilement %d, barème %d : les trois listes doivent couvrir les mêmes rôles",
			len(RoleOrder()), len(stackOrder), len(want))
	}
}

// TestStackOrder_DuPlusFortAuPlusFaible : les gains d'abord, du plus fort au plus faible, puis
// les pertes, de la plus forte à la plus faible — l'ordre d'empilement contre l'axe zéro.
func TestStackOrder_DuPlusFortAuPlusFaible(t *testing.T) {
	prev := 0.0
	gains := true
	for i, role := range stackOrder {
		w := weights[role]
		if i == 0 {
			prev = w
			continue
		}
		switch {
		case gains && w < 0:
			gains = false
		case gains && w > prev:
			t.Errorf("%s (%v) après un gain plus faible (%v)", role, w, prev)
		case !gains && w > 0:
			t.Errorf("%s : gain après une perte", role)
		case !gains && w < prev:
			t.Errorf("%s (%v) après une perte plus faible (%v)", role, w, prev)
		}
		prev = w
	}
}

// TestRolesOfMatch_EscouadeSeule : le calcul est team-wide, seuls les rôles des membres de
// l'escouade sont rendus, sous le nom de leur ligne ; le « Voleur » arrive en dernier.
func TestRolesOfMatch_EscouadeSeule(t *testing.T) {
	allies := []analysis.ParticipantSnap{
		{XUID: "x_main", Outcome: 2, Kills: 5, Deaths: 2, Assists: 3},
		{XUID: "x_a", Outcome: 2, Kills: 4, Deaths: 0, Assists: 8},
		{XUID: "x_ns", Outcome: 2, Kills: 12, Deaths: 1, Assists: 1},
	}
	squad := map[string]string{"x_main": "Main", "x_a": "A"}
	thief := &analysis.ImpactBadge{BadgeKey: RoleThief, PlayerXUID: "x_a"}
	got := RolesOfMatch(MatchInput{Allies: allies, Thief: thief}, squad)
	if len(got) == 0 || got[len(got)-1] != (Attribution{Player: "A", Role: RoleThief}) {
		t.Fatalf("Voleur attendu en dernier sur A, obtenu %+v", got)
	}
	silent := false
	for _, a := range got {
		if a.Role == RoleTopKiller {
			t.Errorf("Bourreau tombé sur l'allié hors escouade : ne doit pas apparaître (%+v)", a)
		}
		silent = silent || a == Attribution{Player: "A", Role: RoleSilentHero}
	}
	if !silent {
		t.Errorf("Héros silencieux attendu sur A, obtenu %+v", got)
	}
}

// TestRolesOfMatch_SansAlliesNiEvenements : un match sans équipe alliée ni événement n'attribue
// rien, pas même un « Voleur ».
func TestRolesOfMatch_SansAlliesNiEvenements(t *testing.T) {
	thief := &analysis.ImpactBadge{BadgeKey: RoleThief, PlayerXUID: "x_a"}
	if got := RolesOfMatch(MatchInput{Thief: thief}, map[string]string{"x_a": "A"}); got != nil {
		t.Errorf("attendu aucun rôle, obtenu %+v", got)
	}
}
