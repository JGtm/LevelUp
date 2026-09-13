package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
)

// flag_spawn_neutral_label_test.go — LE LABEL TRANCHE LA NEUTRALITE D'UN SOCLE DE DRAPEAU,
// PAS LE `team_index` (D9, corrige le 2026-09-13).
//
// CE QUI ETAIT FAUX. Le socle CENTRAL d'Illusion (`9e821f5e`, `ctf_illusion.mvar`,
// object_index 201, au point (0, 0)) porte `ctf_neutral_include` ET `team_index = 0`. Tout
// l'aval triait la neutralite sur le seul `team_index` : ce socle devenait donc un TROISIEME
// drapeau d'equipe 0, fige au milieu de la carte, que `flag_neutral.go` ne pouvait pas
// ecarter — la plus grande zone aveugle du parc (rapport 6.11, decouverte D1, mesuree sur
// `bc60b4d9`).
//
// CE QUE CES TESTS FIGENT. (1) `IsCTFNeutral` lit le label et rien d'autre ; (2) la
// projection `PointsOfRole` le reporte dans `PointObjective.Neutral`, puisqu'elle laisse
// tomber `Labels` et que sans lui l'information n'existerait plus en aval ; (3) le calque
// statique publie ce socle en NEUTRE malgre son `team_index`, et ne touche PAS aux socles
// d'equipe voisins.
//
// LE CAS D'ILLUSION EST REPRODUIT A L'IDENTIQUE (memes positions, memes team_index, memes
// labels que le catalogue versionne) : ce test echoue si la regle repasse au `team_index`.

// illusionFlagSpawns reproduit les TROIS socles de drapeau d'Illusion tels que le catalogue
// versionne les porte (recense le 2026-09-13).
func illusionFlagSpawns() []mapvar.Objective {
	spawn := func(idx int, instance int32, y float64, team int, labels ...string) mapvar.Objective {
		return mapvar.Objective{
			Role:       mapvar.RoleFlagSpawn,
			TypeID:     -1430101324,
			InstanceID: instance,
			ObjectIdx:  idx,
			TeamIndex:  team,
			Pos:        mapvar.Vec3{X: 0, Y: y, Z: 0.8},
			Forward:    mapvar.Vec3{X: 1},
			Labels:     labels,
		}
	}
	return []mapvar.Objective{
		spawn(198, -1996868974, 14.8, 1, "ctf_include", "flag_spawn"),
		spawn(199, 1764626852, -14.8, 0, "ctf_include", "flag_spawn"),
		// LE SOCLE FAUTIF : central, neutre au label, equipe 0 au team_index.
		spawn(201, 1092073439, 0, 0, "ctf_neutral_include", "flag_spawn"),
	}
}

func TestIsCTFNeutral_LitLeLabelPasLEquipe(t *testing.T) {
	socles := illusionFlagSpawns()
	for _, c := range []struct {
		nom  string
		obj  mapvar.Objective
		want bool
	}{
		{"socle equipe 1", socles[0], false},
		{"socle equipe 0", socles[1], false},
		{"socle central (team_index 0, label neutre)", socles[2], true},
	} {
		t.Run(c.nom, func(t *testing.T) {
			if got := c.obj.IsCTFNeutral(); got != c.want {
				t.Fatalf("IsCTFNeutral() = %v, attendu %v (labels %v, team_index %d)",
					got, c.want, c.obj.Labels, c.obj.TeamIndex)
			}
		})
	}
}

func TestPointsOfRole_ReporteLaNeutraliteDuLabel(t *testing.T) {
	e := MapObjectivesEntry{Objectives: illusionFlagSpawns()}
	points := e.PointsOfRole(mapvar.RoleFlagSpawn)
	if len(points) != 3 {
		t.Fatalf("%d socle(s) ponctuel(s), attendu 3", len(points))
	}
	neutres := 0
	for _, p := range points {
		if !p.Neutral {
			continue
		}
		neutres++
		// Le seul socle neutre d'Illusion est le CENTRAL. Si la projection reportait la
		// neutralite depuis le team_index, ce serait l'un des deux socles d'equipe.
		if p.Center.X != 0 || p.Center.Y != 0 {
			t.Errorf("socle neutre au point (%v, %v), attendu le centre (0, 0)", p.Center.X, p.Center.Y)
		}
		if p.TeamIndex != 0 {
			t.Errorf("team_index du socle neutre = %d, attendu 0 — la fixture ne reproduit "+
				"plus le cas d'Illusion, qui est TOUT l'objet de ce test", p.TeamIndex)
		}
	}
	if neutres != 1 {
		t.Fatalf("%d socle(s) neutre(s), attendu exactement 1", neutres)
	}
}

func TestBuildMapObjectives_SocleCentralNeutreMalgreSonTeamIndex(t *testing.T) {
	e := MapObjectivesEntry{
		MapID:      "9e821f5e-042f-407c-97f3-de165b1cdb26",
		PublicName: "Illusion",
		Objectives: illusionFlagSpawns(),
	}
	// Spec du CTF : le rôle n'est PAS forcé neutre — c'est exactement le cas qui
	// découvre le défaut, puisque seul le label peut alors trancher.
	mo := BuildMapObjectives(e, []ObjectiveRoleSpec{{Role: mapvar.RoleFlagSpawn}})
	if mo == nil {
		t.Fatal("BuildMapObjectives rend nil — les socles de drapeau ne sont plus publies ?")
	}

	var central *ObjectiveMarkerDTO
	equipes := map[int]int{}
	for i := range mo.Markers {
		m := &mo.Markers[i]
		if m.Role != string(mapvar.RoleFlagSpawn) {
			continue
		}
		equipes[m.Team]++
		if m.X == 0 && m.Y == 0 {
			central = m
		}
	}
	if central == nil {
		t.Fatal("le socle central (0, 0) n'est pas publie")
	}
	if central.Team != TeamNeutral {
		t.Errorf("socle central publie en equipe %d, attendu %d (neutre) — la neutralite est "+
			"repassee au team_index du fichier de carte, qui vaut 0 sur ce socle",
			central.Team, TeamNeutral)
	}
	// Les deux socles d'equipe gardent leur camp : la correction ne neutralise pas tout.
	if equipes[0] != 1 || equipes[1] != 1 {
		t.Errorf("socles d'equipe publies : %d en equipe 0 et %d en equipe 1, attendu 1 et 1",
			equipes[0], equipes[1])
	}
}
