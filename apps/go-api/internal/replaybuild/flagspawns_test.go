package replaybuild

// flagspawns_test.go — LA NEUTRALITE D'UN SOCLE VOYAGE DANS SON PROPRE CHAMP (D-B2,
// 2026-09-13), et l'equipe reste ce que le fichier de carte dit.
//
// POURQUOI DEUX CHAMPS PLUTOT QU'UN. `TeamNeutral` (-1) est SURCHARGE au catalogue : il vaut
// « socle neutre » sur 63 socles et « equipe inconnue » sur 8 autres (recensement dans
// `replay/flag_spawn_neutral_label_test.go`). Une seule valeur ne peut donc pas porter les
// deux faits. `flagSpawnTeam` rend l'equipe d'affichage ; `FlagSpawn.Neutral` rend la
// variante, et c'est lui que le tri du calque lit.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
)

func TestFlagSpawnTeam_LeLabelNeutralisePasLeTeamIndex(t *testing.T) {
	for _, c := range []struct {
		nom  string
		p    replay.PointObjective
		want int
	}{
		{"socle d'equipe 0", replay.PointObjective{TeamIndex: 0}, 0},
		{"socle d'equipe 1", replay.PointObjective{TeamIndex: 1}, 1},
		// Le socle central d'Illusion : neutre au label, equipe 0 au fichier (D9).
		{"socle neutre au label malgre team_index 0",
			replay.PointObjective{TeamIndex: 0, Neutral: true}, replay.TeamNeutral},
		// Les huit socles de D-B2 : equipe INCONNUE, aucun label — ils gardent -1.
		{"socle d'equipe a team_index -1 sans label",
			replay.PointObjective{TeamIndex: replay.TeamNeutral}, replay.TeamNeutral},
	} {
		t.Run(c.nom, func(t *testing.T) {
			if got := flagSpawnTeam(c.p); got != c.want {
				t.Fatalf("flagSpawnTeam = %d, attendu %d", got, c.want)
			}
		})
	}
}

// TestFlagSpawnsReporteLaNeutraliteDuLabel — le socle a `team_index = -1` SANS label sort
// avec `Neutral` FAUX, donc hors du panier neutre du calque, alors que son equipe reste -1.
//
// C'EST LA CORRECTION DE D-B2 EN UN SEUL ASSERT : avant, la seule chose qui voyageait etait
// l'equipe, et l'aval ne pouvait pas distinguer ce socle du socle central.
func TestFlagSpawnsReporteLaNeutraliteDuLabel(t *testing.T) {
	spawn := func(idx int, instance int32, x float64, team int, labels ...string) mapvar.Objective {
		return mapvar.Objective{
			Role: mapvar.RoleFlagSpawn, InstanceID: instance, ObjectIdx: idx,
			TeamIndex: team, Pos: mapvar.Vec3{X: x}, Labels: labels,
		}
	}
	e := replay.MapObjectivesEntry{Objectives: []mapvar.Objective{
		spawn(1, 11, -10, 0, "ctf_include", "flag_spawn"),
		spawn(2, 12, 10, 1, "ctf_include", "flag_spawn"),
		spawn(3, 13, 0, 0, "ctf_neutral_include", "flag_spawn"),
		// LE CAS D-B2 : equipe tue par le fichier, aucun label neutre.
		spawn(4, 14, 30, replay.TeamNeutral, "ctf_include", "flag_spawn"),
	}}

	var neutres, sansEquipe int
	for _, p := range e.PointsOfRole(mapvar.RoleFlagSpawn) {
		s := replay.FlagSpawn{
			Team: flagSpawnTeam(p), Neutral: p.Neutral,
			X: float32(p.Center.X), Y: float32(p.Center.Y),
		}
		if s.Neutral {
			neutres++
			if s.X != 0 {
				t.Errorf("socle neutre en x=%v, attendu le centre (0)", s.X)
			}
		}
		if !s.Neutral && s.Team == replay.TeamNeutral {
			sansEquipe++
			if s.X != 30 {
				t.Errorf("socle sans equipe en x=%v, attendu 30", s.X)
			}
		}
	}
	if neutres != 1 {
		t.Errorf("%d socle(s) neutre(s), attendu 1 (le central, par son label)", neutres)
	}
	if sansEquipe != 1 {
		t.Errorf("%d socle(s) d'equipe a -1 sans label, attendu 1 — ils ne doivent PAS "+
			"devenir neutres", sansEquipe)
	}
}
