package replay

// socles_de_drapeau_test.go — LA NEUTRALITE D'UN SOCLE VOYAGE DANS SON PROPRE CHAMP (D-B2,
// 2026-09-13), et l'equipe reste ce que le fichier de carte dit.
//
// DEPLACE depuis `replaybuild/flagspawns_test.go` le 2026-09-29 avec la projection qu'il teste
// (lot V2 du plan Emprise vies). Le second test appelait une COPIE de la projection ecrite dans
// le test ; il appelle desormais [MapObjectivesEntry.SoclesDeDrapeau], la projection elle-meme.
//
// POURQUOI DEUX CHAMPS PLUTOT QU'UN. `TeamNeutral` (-1) est SURCHARGE au catalogue : il vaut
// « socle neutre » sur 63 socles et « equipe inconnue » sur 8 autres (recensement dans
// `flag_spawn_neutral_label_test.go`). Une seule valeur ne peut donc pas porter les deux faits.
// `equipeDuSocle` rend l'equipe d'affichage ; `FlagSpawn.Neutral` rend la variante, et c'est lui
// que le tri du calque lit.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
)

func TestEquipeDuSocle_LeLabelNeutralisePasLeTeamIndex(t *testing.T) {
	for _, c := range []struct {
		nom  string
		p    PointObjective
		want int
	}{
		{"socle d'equipe 0", PointObjective{TeamIndex: 0}, 0},
		{"socle d'equipe 1", PointObjective{TeamIndex: 1}, 1},
		// Le socle central d'Illusion : neutre au label, equipe 0 au fichier (D9).
		{"socle neutre au label malgre team_index 0",
			PointObjective{TeamIndex: 0, Neutral: true}, TeamNeutral},
		// Les huit socles de D-B2 : equipe INCONNUE, aucun label — ils gardent -1.
		{"socle d'equipe a team_index -1 sans label",
			PointObjective{TeamIndex: TeamNeutral}, TeamNeutral},
	} {
		t.Run(c.nom, func(t *testing.T) {
			if got := equipeDuSocle(c.p); got != c.want {
				t.Fatalf("equipeDuSocle = %d, attendu %d", got, c.want)
			}
		})
	}
}

// TestSoclesDeDrapeauReportentLaNeutraliteDuLabel — le socle a `team_index = -1` SANS label
// sort avec `Neutral` FAUX, donc hors du panier neutre du calque, alors que son equipe reste -1.
//
// C'EST LA CORRECTION DE D-B2 EN UN SEUL ASSERT : avant, la seule chose qui voyageait etait
// l'equipe, et l'aval ne pouvait pas distinguer ce socle du socle central.
func TestSoclesDeDrapeauReportentLaNeutraliteDuLabel(t *testing.T) {
	spawn := func(idx int, instance int32, x float64, team int, labels ...string) mapvar.Objective {
		return mapvar.Objective{
			Role: mapvar.RoleFlagSpawn, InstanceID: instance, ObjectIdx: idx,
			TeamIndex: team, Pos: mapvar.Vec3{X: x}, Labels: labels,
		}
	}
	e := MapObjectivesEntry{Objectives: []mapvar.Objective{
		spawn(1, 11, -10, 0, "ctf_include", "flag_spawn"),
		spawn(2, 12, 10, 1, "ctf_include", "flag_spawn"),
		spawn(3, 13, 0, 0, "ctf_neutral_include", "flag_spawn"),
		// LE CAS D-B2 : equipe tue par le fichier, aucun label neutre.
		spawn(4, 14, 30, TeamNeutral, "ctf_include", "flag_spawn"),
	}}

	socles := e.SoclesDeDrapeau()
	if len(socles) != 4 {
		t.Fatalf("%d socles, attendu 4 (les deux d'equipe, le neutre, celui sans equipe)", len(socles))
	}
	var neutres, sansEquipe int
	for _, s := range socles {
		if s.Neutral {
			neutres++
			if s.X != 0 || s.Team != TeamNeutral {
				t.Errorf("socle neutre en x=%v equipe %d, attendu le centre (0) et %d", s.X, s.Team, TeamNeutral)
			}
		}
		if !s.Neutral && s.Team == TeamNeutral {
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
