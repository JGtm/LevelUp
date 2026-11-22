package squadimpact

import (
	"testing"

	"levelup/go-api/internal/analysis"
)

// TestWeight_Bareme : chaque colonne de la matrice a son barème, et le barème est celui de la
// spécification (Finisseur et Premier sang +2, Héros silencieux +1,5, Bourreau +1 ; Boulet −2,
// Faux-frère −1,5, Voleur −1, Première victime, Touriste, Kamikaze et Os de verre −0,5).
func TestWeight_Bareme(t *testing.T) {
	want := map[string]float64{
		RoleClutchFinisher: 2, RoleFirstBlood: 2, RoleSilentHero: 1.5, RoleTopKiller: 1,
		RoleLastCasualty: -2, RoleFalseBrother: -1.5,
		RoleThief:           -1,
		RoleFirstGroupDeath: -0.5, RoleLastGroupKill: -0.5, RoleKamikaze: -0.5, RoleGlassJaw: -0.5,
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

// TestStackOrder_OsDeVerreApresSesPairs : à −0,5, « Os de verre » suit Première victime, Touriste
// et Kamikaze, et le Voleur (−1) les précède.
func TestStackOrder_OsDeVerreApresSesPairs(t *testing.T) {
	tail := stackOrder[len(stackOrder)-5:]
	want := []string{RoleThief, RoleFirstGroupDeath, RoleLastGroupKill, RoleKamikaze, RoleGlassJaw}
	for i := range want {
		if tail[i] != want[i] {
			t.Fatalf("fin de l'empilement %v, attendu %v", tail, want)
		}
	}
}

// TestRolesOfMatch_OsDeVerre : le rôle tombe sur l'allié qui a le plus de bris, y compris hors
// escouade (alors il n'est pas rendu), et il précède le « Voleur ».
func TestRolesOfMatch_OsDeVerre(t *testing.T) {
	allies := []analysis.ParticipantSnap{
		{XUID: "x_main", Outcome: 2, Kills: 5, Deaths: 2},
		{XUID: "x_a", Outcome: 2, Kills: 4, Deaths: 1},
		{XUID: "x_ns", Outcome: 2, Kills: 3, Deaths: 3},
	}
	squad := map[string]string{"x_main": "Main", "x_a": "A"}
	thief := &analysis.ImpactBadge{BadgeKey: RoleThief, PlayerXUID: "x_a"}
	shields := map[string]ShieldCount{"x_main": {Breaks: 8}, "x_a": {Breaks: 11, Recharges: 3}, "x_ns": {Breaks: 4}}
	got := RolesOfMatch(MatchInput{Allies: allies, Thief: thief, Shields: shields}, squad)
	n := len(got)
	if n < 2 || got[n-2] != (Attribution{Player: "A", Role: RoleGlassJaw}) || got[n-1].Role != RoleThief {
		t.Fatalf("Os de verre sur A puis Voleur attendus en fin de liste, obtenu %+v", got)
	}
	// Le plus de bris est un allié hors escouade : aucun rôle pour l'escouade.
	shields["x_ns"] = ShieldCount{Breaks: 30}
	for _, a := range RolesOfMatch(MatchInput{Allies: allies, Shields: shields}, squad) {
		if a.Role == RoleGlassJaw {
			t.Errorf("Os de verre tombé hors escouade rendu à %+v", a)
		}
	}
	// Match non mesuré : aucun rôle.
	for _, a := range RolesOfMatch(MatchInput{Allies: allies, Thief: thief}, squad) {
		if a.Role == RoleGlassJaw {
			t.Errorf("match non mesuré : Os de verre rendu à %+v", a)
		}
	}
}

// TestGlassJaw_Departage : le plus de bris, puis le moins de recharges, puis le xuid croissant ;
// aucun rôle sans mesure ni sans bris allié ; un ennemi ou un bot ne concourt pas.
func TestGlassJaw_Departage(t *testing.T) {
	allies := []analysis.ParticipantSnap{{XUID: "b"}, {XUID: "a"}, {XUID: "c"}, {XUID: ""}}
	cas := []struct {
		nom     string
		shields map[string]ShieldCount
		want    string
		ok      bool
	}{
		{"plus de bris", map[string]ShieldCount{"a": {Breaks: 5}, "b": {Breaks: 7}, "c": {Breaks: 6}}, "b", true},
		{"egalite : moins de recharges", map[string]ShieldCount{"a": {7, 4}, "b": {7, 2}, "c": {7, 3}}, "b", true},
		{"egalite totale : xuid croissant", map[string]ShieldCount{"b": {7, 2}, "a": {7, 2}, "c": {7, 2}}, "a", true},
		{"ennemi et bot ecartes", map[string]ShieldCount{"a": {Breaks: 3}, "ennemi": {Breaks: 40}, "": {Breaks: 50}}, "a", true},
		{"aucun bris allie", map[string]ShieldCount{"a": {}, "b": {Recharges: 2}, "ennemi": {Breaks: 9}}, "", false},
		{"non mesure", nil, "", false},
		{"mesure vide", map[string]ShieldCount{}, "", false},
	}
	for _, c := range cas {
		got, ok := GlassJaw(allies, c.shields)
		if got != c.want || ok != c.ok {
			t.Errorf("%s : (%q, %v), attendu (%q, %v)", c.nom, got, ok, c.want, c.ok)
		}
	}
	if got, ok := GlassJaw(nil, map[string]ShieldCount{"a": {Breaks: 3}}); ok {
		t.Errorf("sans allié : (%q, true), attendu aucun rôle", got)
	}
}
