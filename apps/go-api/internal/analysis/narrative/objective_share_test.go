package narrative

import "testing"

// ctfRows : une équipe 0 (porteur, escorte, tueur) et un adversaire 1 qui a aussi capturé
// (ses actions ne doivent JAMAIS entrer dans la part de l'équipe 0).
func ctfRows() []ObjectiveTeamRow {
	return []ObjectiveTeamRow{
		{XUID: "porteur", TeamID: 0, Family: FamilyCTF, TimePlayedSeconds: 600, Values: map[string]float64{
			"flag_captures": 2, "flag_steals": 2, "time_as_flag_carrier_seconds": 26,
		}}, // 5×2 + 2×2 = 14
		{XUID: "escorte", TeamID: 0, Family: FamilyCTF, TimePlayedSeconds: 600, Values: map[string]float64{
			"flag_capture_assists": 2, "flag_returns": 2, "flag_carriers_killed": 2,
		}}, // 2×2 + 1.5×2 + 1.5×2 = 10
		{XUID: "tueur", TeamID: 0, Family: FamilyCTF, TimePlayedSeconds: 600, Values: map[string]float64{}},
		{XUID: "adverse", TeamID: 1, Family: FamilyCTF, TimePlayedSeconds: 600, Values: map[string]float64{
			"flag_captures": 4,
		}},
	}
}

func TestComputeObjectiveTeamShare_CTFPorteurEscorteTueur(t *testing.T) {
	rows := ctfRows()
	cases := []struct {
		xuid   string
		points float64
		share  float64
	}{
		{"porteur", 14, 14.0 / 24.0},
		{"escorte", 10, 10.0 / 24.0},
		{"tueur", 0, 0},
	}
	var sum float64
	for _, c := range cases {
		got, ok := ComputeObjectiveTeamShare(rows, c.xuid)
		if !ok {
			t.Fatalf("%s : ok = false", c.xuid)
		}
		if got.Family != FamilyCTF || !almostEqual(got.Points, c.points) ||
			!almostEqual(got.TeamPoints, 24) || !almostEqual(got.Share, c.share) {
			t.Errorf("%s : %+v, want points=%v teamPoints=24 share=%v", c.xuid, got, c.points, c.share)
		}
		sum += got.Share
	}
	if !almostEqual(sum, 1) {
		t.Errorf("la somme des parts du camp = %v, want 1", sum)
	}
}

// L'indice du joueur est celui de ComputeObjectiveIndex sur le même agrégat d'un match :
// une seule définition de r_f.
func TestComputeObjectiveTeamShare_IndexEgalIndiceDuPackage(t *testing.T) {
	rows := ctfRows()
	got, _ := ComputeObjectiveTeamShare(rows, "porteur")
	st := objectiveStatsOfRow(FamilyCTF, &rows[0])
	want, n := ComputeObjectiveIndex(ObjectiveIndexInput{FamilyCTF: st})
	if n != 1 || !almostEqual(got.Index, want) {
		t.Fatalf("Index = %v, want %v (n=%d)", got.Index, want, n)
	}
	if got.Index <= 0 || got.Index > ObjectiveIndexClip {
		t.Fatalf("Index hors [0, %v] : %v", ObjectiveIndexClip, got.Index)
	}
}

// Un joueur sans action garde une part nulle ; son indice ne compte que sa tenue.
func TestComputeObjectiveTeamShare_PartNulle(t *testing.T) {
	got, ok := ComputeObjectiveTeamShare(ctfRows(), "tueur")
	if !ok || got.Share != 0 || got.Points != 0 || got.Index != 0 {
		t.Fatalf("tueur = %+v ok=%v, want part nulle et indice nul", got, ok)
	}
}

// Un camp sans aucune action : part 0, pas une absence.
func TestComputeObjectiveTeamShare_CampSansAction(t *testing.T) {
	rows := []ObjectiveTeamRow{
		{XUID: "a", TeamID: 0, Family: FamilyCTF, TimePlayedSeconds: 300, Values: map[string]float64{}},
		{XUID: "b", TeamID: 1, Family: FamilyCTF, TimePlayedSeconds: 300, Values: map[string]float64{"flag_captures": 1}},
	}
	got, ok := ComputeObjectiveTeamShare(rows, "a")
	if !ok || got.Share != 0 || got.TeamPoints != 0 {
		t.Fatalf("camp sans action : %+v ok=%v", got, ok)
	}
}

// KOTH l'emporte sur Strongholds dès qu'une ligne a des ticks : les poids KOTH (ticks
// compris) servent alors à TOUS les joueurs du match, y compris ceux classés Strongholds.
func TestMatchObjectiveFamily_KOTHPrime(t *testing.T) {
	rows := []ObjectiveTeamRow{
		{XUID: "a", Family: FamilyZonesKOTH},
		{XUID: "b", Family: FamilyZonesStrongholds},
		{XUID: "c", Family: FamilyZonesStrongholds},
	}
	if got := MatchObjectiveFamily(rows); got != FamilyZonesKOTH {
		t.Fatalf("famille = %q, want KOTH", got)
	}
	rows[0].Family = FamilyZonesStrongholds
	if got := MatchObjectiveFamily(rows); got != FamilyZonesStrongholds {
		t.Fatalf("famille = %q, want Strongholds", got)
	}
}

// Halo 5 (ou tout titre sans match_objective_stats) : aucune ligne, ou des lignes sans bloc
// d'objectif. La part est ABSENTE (ok == false), sans panique.
func TestComputeObjectiveTeamShare_SansDonneesAbsent(t *testing.T) {
	cases := map[string][]ObjectiveTeamRow{
		"nil":                nil,
		"vide":               {},
		"sans famille":       {{XUID: "a", TeamID: 0}, {XUID: "b", TeamID: 1}},
		"valeurs nil":        {{XUID: "a", TeamID: 0, Family: FamilyCTF}},
		"joueur absent":      {{XUID: "z", TeamID: 0, Family: FamilyCTF, Values: map[string]float64{"flag_captures": 1}}},
		"camp inconnu":       {{XUID: "a", TeamID: NoObjectiveTeam, Family: FamilyCTF, Values: map[string]float64{"flag_captures": 1}}},
		"famille hors-table": {{XUID: "a", TeamID: 0, Family: "inconnue"}},
	}
	for name, rows := range cases {
		got, ok := ComputeObjectiveTeamShare(rows, "a")
		switch name {
		case "valeurs nil":
			// ligne présente, famille connue, aucune valeur : part 0, mesuré.
			if !ok || got.Share != 0 {
				t.Errorf("%s : %+v ok=%v, want part 0 mesurée", name, got, ok)
			}
		case "famille hors-table":
			// MatchObjectiveFamily ne retient que les familles de la table.
			if ok {
				t.Errorf("%s : ok = true, want absent", name)
			}
		default:
			if ok {
				t.Errorf("%s : ok = true (%+v), want absent", name, got)
			}
		}
	}
}

// VIP : kills_as_vip est rapporté aux sélections, comme dans l'indice.
func TestObjectiveActionPoints_VIPDenominateurSelections(t *testing.T) {
	st := ObjectiveFamilyStats{
		Matches: 1, TimesSelectedAsVIP: 2,
		ColumnSums: map[string]float64{"vip_kills": 1, ObjectiveColKillsAsVIP: 4},
	}
	// 3×1 + 2×4/2 = 7
	if got := ObjectiveActionPoints(FamilyVIP, st); !almostEqual(got, 7) {
		t.Fatalf("points VIP = %v, want 7", got)
	}
	if got := ObjectiveActionPoints(FamilyVIP, ObjectiveFamilyStats{}); got != 0 {
		t.Fatalf("Matches = 0 : %v, want 0", got)
	}
}

// Les colonnes de soutien sont des colonnes de la table de poids : aucune colonne neuve.
func TestObjectiveSupportColumns_SousEnsembleDesPoids(t *testing.T) {
	for fam, cols := range ObjectiveSupportColumns {
		for _, c := range cols {
			if _, ok := ObjectiveFamilyActionWeights[fam][c]; !ok {
				t.Errorf("famille %s : colonne de soutien %q absente de ObjectiveFamilyActionWeights", fam, c)
			}
		}
	}
	// Ni capture ni vol : le soutien ne reprend pas ce qui prend l'objectif.
	for _, c := range ObjectiveSupportColumns[FamilyCTF] {
		if c == objectiveColFlagCaptures || c == objectiveColFlagSteals {
			t.Errorf("%q n'est pas du soutien", c)
		}
	}
}

// L'indice brut dépasse le plafond que garde l'indice des radars ; le relèvement du soutien
// ne touche que les colonnes de soutien et jamais la table de poids.
func TestComputeObjectiveTeamShareWith_BrutRelevementEtPartDuSoutien(t *testing.T) {
	rows := []ObjectiveTeamRow{
		{XUID: "porteur", TeamID: 0, Family: FamilyCTF, TimePlayedSeconds: 600, Values: map[string]float64{
			"flag_captures": 6, "flag_steals": 4, "time_as_flag_carrier_seconds": 60,
		}},
		{XUID: "escorte", TeamID: 0, Family: FamilyCTF, TimePlayedSeconds: 600, Values: map[string]float64{
			"flag_returns": 4, "flag_carriers_killed": 4, "flag_capture_assists": 2,
		}},
	}
	before := ObjectiveFamilyActionWeights[FamilyCTF]["flag_returns"]
	p1, _ := ComputeObjectiveTeamShare(rows, "porteur")
	if p1.RawIndex <= ObjectiveIndexClip || p1.Index != ObjectiveIndexClip {
		t.Fatalf("porteur : RawIndex %v Index %v, want brut > plafond et Index écrêté", p1.RawIndex, p1.Index)
	}
	e1, _ := ComputeObjectiveTeamShare(rows, "escorte")
	e2, _ := ComputeObjectiveTeamShareWith(rows, "escorte", ObjectiveWeighting{SupportMult: 2})
	// escorte : 1,5x4 + 1,5x4 + 2x2 = 16 ; relevé x2 = 32.
	if !almostEqual(e1.Points, 16) || !almostEqual(e2.Points, 32) {
		t.Errorf("points d'escorte %v / %v, want 16 / 32", e1.Points, e2.Points)
	}
	if e2.Share <= e1.Share || e2.RawIndex <= e1.RawIndex {
		t.Errorf("le relèvement doit monter la part (%v -> %v) et l'indice (%v -> %v)", e1.Share, e2.Share, e1.RawIndex, e2.RawIndex)
	}
	if !almostEqual(e1.SupportShare, 1) || !almostEqual(e1.SupportPoints, 16) {
		t.Errorf("part du soutien de l'escorte = %v (%v points), want 1 (16)", e1.SupportShare, e1.SupportPoints)
	}
	if p1.SupportShare != 0 {
		t.Errorf("le porteur n'a aucune action de soutien : %v", p1.SupportShare)
	}
	if ObjectiveFamilyActionWeights[FamilyCTF]["flag_returns"] != before {
		t.Error("la table de poids a été modifiée par le relèvement")
	}
	// SupportMult nul = poids de la table.
	z, _ := ComputeObjectiveTeamShareWith(rows, "escorte", ObjectiveWeighting{})
	if !almostEqual(z.Points, e1.Points) {
		t.Errorf("SupportMult 0 : %v, want %v", z.Points, e1.Points)
	}
}
