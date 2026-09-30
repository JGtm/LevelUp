package replayverite

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// aide_test.go — les artefacts SYNTHETIQUES des tests : un match minimal, juste, que chaque test
// abime d'une seule facon.

func ptr[T any](v T) *T { return &v }

// serie rend une courbe en escalier de valeurs croissantes datees.
func serie(pas ...Pas) Serie { return Serie{Total: pas} }

// piste rend une vie : un slot, un xuid, des points a une position fixe de l'image a a b.
func piste(slot int, xuid string, a, b int) Piste {
	return Piste{Slot: slot, XUID: xuid, StartFrame: ptr(a), EndFrame: b,
		Points: []Point{{T: a, X: 0, Y: 0, Z: ptr(0.0)}, {T: b, X: 1, Y: 0, Z: ptr(0.0)}}}
}

// documentJuste : deux joueurs, deux vies chacun (une mort chacun), tout concorde avec faitsJustes.
//
//	A (111, camp 0) : 3 kills, 1 mort, 1 assistance ; vies [0,100] et [150,300]
//	B (222, camp 1) : 1 kill,  1 mort, 0 assistance ; vies [0,200] et [250,300]
func documentJuste() *Document {
	return &Document{
		MatchID: "m", FrameCount: 302, FrameIntervalMs: 100,
		Bounds: &Bornes{MinX: 0, MinY: 0, MaxX: 100, MaxY: 100},
		Tracks: []Piste{piste(512, "111", 0, 100), piste(514, "111", 150, 300),
			piste(513, "222", 0, 200), piste(515, "222", 250, 300)},
		Shots:  []Action{{T: 50, Slot: ptr(512)}, {T: 210, Slot: ptr(514)}},
		Roster: []Inscrit{{XUID: "111", Presence: []Presence{{From: 0, To: 300}}}, {XUID: "222", Presence: []Presence{{From: 0, To: 300}}}},
		ScoreTimeline: &ChronoScore{
			Teams: []ScoreEquipe{{TeamID: ptr(0), Total: []Pas{{T: 50, V: 1}, {T: 90, V: 3}}}, {TeamID: ptr(1), Total: []Pas{{T: 190, V: 1}}}},
			Players: []ScoreJoueur{
				{XUID: "111", Kills: serie(Pas{0, 0}, Pas{50, 3}), Deaths: serie(Pas{0, 0}, Pas{100, 1}), Assists: serie(Pas{0, 0}, Pas{60, 1})},
				{XUID: "222", Kills: serie(Pas{0, 0}, Pas{190, 1}), Deaths: serie(Pas{0, 0}, Pas{201, 1}), Assists: serie(Pas{0, 0})},
			},
		},
		Identity: &Identite{StatborgSlots: []SlotStatborg{
			{Slot: 10, XUID: "111", Link: Lien{Source: "deduit", Method: "instants_de_mort"}},
			{Slot: 12, XUID: "222", Link: Lien{Source: "deduit", Method: "instants_de_mort"}},
		}},
		Coverage: Couverture{
			ContinuousFire: &CouvTirContinu{Packets: 100, Closed: 80},
			Keyframes:      &CouvImagesCles{},
			Verdict:        map[string]string{"shots": "nominal"},
			Bridge:         &CouvPont{},
			Seats:          &CouvSieges{},
			Teams:          &CouvEquipes{Accord: 2},
			Score:          &CouvScore{TeamIdentity: "b"},
		},
	}
}

func faitsJustes() domain.MatchFacts {
	return domain.MatchFacts{
		Players: []domain.MatchPlayerFact{
			{XUID: "111", Kills: 3, Deaths: 1, Assists: 1, TeamID: 0},
			{XUID: "222", Kills: 1, Deaths: 1, Assists: 0, TeamID: 1},
		},
		TeamScores: &[2]int{3, 1},
	}
}

// exigerScore verifie les compteurs d'un score.
func exigerScore(t *testing.T, b Bulletin, id string, vp, fp, fn int) {
	t.Helper()
	s, ok := b.Scores[id]
	if !ok {
		t.Fatalf("%s absent du bulletin", id)
	}
	if s.VP != vp || s.FP != fp || s.FN != fn {
		t.Errorf("%s = VP %d / FP %d / FN %d, veut %d / %d / %d (ecarts %v)", id, s.VP, s.FP, s.FN, vp, fp, fn, s.Ecarts)
	}
}

// exigerViolations verifie le nombre d'instances d'une classe.
func exigerViolations(t *testing.T, b Bulletin, id string, n int) {
	t.Helper()
	if got := b.Violations[id].Total(); got != n {
		t.Errorf("%s = %d instance(s) %v, veut %d", id, got, b.Violations[id].Instances, n)
	}
}
